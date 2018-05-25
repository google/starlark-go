// Copyright 2017 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package skylark_test

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/google/skylark"
)

// ExampleExecFile demonstrates a simple embedding
// of the Skylark interpreter into a Go program.
func ExampleExecFile() {
	const data = `
print(greeting + ", world")

squares = [x*x for x in range(10)]
`

	thread := &skylark.Thread{
		Print: func(_ *skylark.Thread, msg string) { fmt.Println(msg) },
	}
	predeclared := skylark.StringDict{
		"greeting": skylark.String("hello"),
	}
	globals, err := skylark.ExecFile(thread, "apparent/filename.sky", data, predeclared)
	if err != nil {
		if evalErr, ok := err.(*skylark.EvalError); ok {
			log.Fatal(evalErr.Backtrace())
		}
		log.Fatal(err)
	}

	// Print the global environment.
	var names []string
	for name := range globals {
		names = append(names, name)
	}
	sort.Strings(names)
	fmt.Println("\nGlobals:")
	for _, name := range names {
		v := globals[name]
		fmt.Printf("%s (%s) = %s\n", name, v.Type(), v.String())
	}

	// Output:
	// hello, world
	//
	// Globals:
	// squares (list) = [0, 1, 4, 9, 16, 25, 36, 49, 64, 81]
}

// ExampleThread_Load_sequential demonstrates a simple caching
// implementation of 'load' that works sequentially.
func ExampleThread_Load_sequential() {
	fakeFilesystem := map[string]string{
		"c.sky": `load("b.sky", "b"); c = b + "!"`,
		"b.sky": `load("a.sky", "a"); b = a + ", world"`,
		"a.sky": `a = "Hello"`,
	}

	type entry struct {
		globals skylark.StringDict
		err     error
	}

	cache := make(map[string]*entry)

	var load func(_ *skylark.Thread, module string) (skylark.StringDict, error)
	load = func(_ *skylark.Thread, module string) (skylark.StringDict, error) {
		e, ok := cache[module]
		if e == nil {
			if ok {
				// request for package whose loading is in progress
				return nil, fmt.Errorf("cycle in load graph")
			}

			// Add a placeholder to indicate "load in progress".
			cache[module] = nil

			// Load and initialize the module in a new thread.
			data := fakeFilesystem[module]
			thread := &skylark.Thread{Load: load}
			globals, err := skylark.ExecFile(thread, module, data, nil)
			e = &entry{globals, err}

			// Update the cache.
			cache[module] = e
		}
		return e.globals, e.err
	}

	thread := &skylark.Thread{Load: load}
	globals, err := load(thread, "c.sky")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(globals["c"])

	// Output:
	// "Hello, world!"
}

// ExampleThread_Load_parallel demonstrates a parallel implementation
// of 'load' with caching, duplicate suppression, and cycle detection.
func ExampleThread_Load_parallel() {
	cache := &cache{
		cache: make(map[string]*entry),
		fakeFilesystem: map[string]string{
			"c.sky": `load("a.sky", "a"); c = a * 2`,
			"b.sky": `load("a.sky", "a"); b = a * 3`,
			"a.sky": `a = 1; print("loaded a")`,
		},
	}

	// We load modules b and c in parallel by concurrent calls to
	// cache.Load.  Both of them load module a, but a is executed
	// only once, as witnessed by the sole output of its print
	// statement.

	ch := make(chan string)
	for _, name := range []string{"b", "c"} {
		go func(name string) {
			globals, err := cache.Load(name + ".sky")
			if err != nil {
				log.Fatal(err)
			}
			ch <- fmt.Sprintf("%s = %s", name, globals[name])
		}(name)
	}
	got := []string{<-ch, <-ch}
	sort.Strings(got)
	fmt.Println(strings.Join(got, "\n"))

	// Output:
	// loaded a
	// b = 3
	// c = 2
}

// ExampleThread_Load_parallelCycle demonstrates detection
// of cycles during parallel loading.
func ExampleThread_Load_parallelCycle() {
	cache := &cache{
		cache: make(map[string]*entry),
		fakeFilesystem: map[string]string{
			"c.sky": `load("b.sky", "b"); c = b * 2`,
			"b.sky": `load("a.sky", "a"); b = a * 3`,
			"a.sky": `load("c.sky", "c"); a = c * 5; print("loaded a")`,
		},
	}

	ch := make(chan string)
	for _, name := range "bc" {
		name := string(name)
		go func() {
			_, err := cache.Load(name + ".sky")
			if err == nil {
				log.Fatalf("Load of %s.sky succeeded unexpectedly", name)
			}
			ch <- err.Error()
		}()
	}
	got := []string{<-ch, <-ch}
	sort.Strings(got)
	fmt.Println(strings.Join(got, "\n"))

	// Output:
	// cannot load a.sky: cannot load c.sky: cycle in load graph
	// cannot load b.sky: cannot load a.sky: cannot load c.sky: cycle in load graph
}

// cache is a concurrency-safe, duplicate-suppressing,
// non-blocking cache of the doLoad function.
// See Section 9.7 of gopl.io for an explanation of this structure.
// It also features online deadlock (load cycle) detection.
type cache struct {
	cacheMu sync.Mutex
	cache   map[string]*entry

	fakeFilesystem map[string]string
}

type entry struct {
	owner   unsafe.Pointer // a *cycleChecker; see cycleCheck
	globals skylark.StringDict
	err     error
	ready   chan struct{}
}

func (c *cache) Load(module string) (skylark.StringDict, error) {
	return c.get(new(cycleChecker), module)
}

// get loads and returns an entry (if not already loaded).
func (c *cache) get(cc *cycleChecker, module string) (skylark.StringDict, error) {
	c.cacheMu.Lock()
	e := c.cache[module]
	if e != nil {
		c.cacheMu.Unlock()
		// Some other goroutine is getting this module.
		// Wait for it to become ready.

		// Detect load cycles to avoid deadlocks.
		if err := cycleCheck(e, cc); err != nil {
			return nil, err
		}

		cc.setWaitsFor(e)
		<-e.ready
		cc.setWaitsFor(nil)
	} else {
		// First request for this module.
		e = &entry{ready: make(chan struct{})}
		c.cache[module] = e
		c.cacheMu.Unlock()

		e.setOwner(cc)
		e.globals, e.err = c.doLoad(cc, module)
		e.setOwner(nil)

		// Broadcast that the entry is now ready.
		close(e.ready)
	}
	return e.globals, e.err
}

func (c *cache) doLoad(cc *cycleChecker, module string) (skylark.StringDict, error) {
	thread := &skylark.Thread{
		Print: func(_ *skylark.Thread, msg string) { fmt.Println(msg) },
		Load: func(_ *skylark.Thread, module string) (skylark.StringDict, error) {
			// Tunnel the cycle-checker state for this "thread of loading".
			return c.get(cc, module)
		},
	}
	data := c.fakeFilesystem[module]
	return skylark.ExecFile(thread, module, data, nil)
}

// -- concurrent cycle checking --

// A cycleChecker is used for concurrent deadlock detection.
// Each top-level call to Load creates its own cycleChecker,
// which is passed to all recursive calls it makes.
// It corresponds to a logical thread in the deadlock detection literature.
type cycleChecker struct {
	waitsFor unsafe.Pointer // an *entry; see cycleCheck
}

func (cc *cycleChecker) setWaitsFor(e *entry) {
	atomic.StorePointer(&cc.waitsFor, unsafe.Pointer(e))
}

func (e *entry) setOwner(cc *cycleChecker) {
	atomic.StorePointer(&e.owner, unsafe.Pointer(cc))
}

// cycleCheck reports whether there is a path in the waits-for graph
// from resource 'e' to thread 'me'.
//
// The waits-for graph (WFG) is a bipartite graph whose nodes are
// alternately of type entry and cycleChecker.  Each node has at most
// one outgoing edge.  An entry has an "owner" edge to a cycleChecker
// while it is being readied by that cycleChecker, and a cycleChecker
// has a "waits-for" edge to an entry while it is waiting for that entry
// to become ready.
//
// Before adding a waits-for edge, the cache checks whether the new edge
// would form a cycle.  If so, this indicates that the load graph is
// cyclic and that the following wait operation would deadlock.
func cycleCheck(e *entry, me *cycleChecker) error {
	for e != nil {
		cc := (*cycleChecker)(atomic.LoadPointer(&e.owner))
		if cc == nil {
			break
		}
		if cc == me {
			return fmt.Errorf("cycle in load graph")
		}
		e = (*entry)(atomic.LoadPointer(&cc.waitsFor))
	}
	return nil
}
