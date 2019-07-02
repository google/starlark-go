// Copyright 2017 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package starlark_test

import (
	"fmt"
	"log"
	"reflect"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"unsafe"

	"go.starlark.net/starlark"
)

// ExampleExecFile demonstrates a simple embedding
// of the Starlark interpreter into a Go program.
func ExampleExecFile() {
	const data = `
print(greeting + ", world")
print(repeat("one"))
print(repeat("mur", 2))
squares = [x*x for x in range(10)]
`

	// repeat(str, n=1) is a Go function called from Starlark.
	// It behaves like the 'string * int' operation.
	repeat := func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var s string
		var n int = 1
		if err := starlark.UnpackArgs(b.Name(), args, kwargs, "s", &s, "n?", &n); err != nil {
			return nil, err
		}
		return starlark.String(strings.Repeat(s, n)), nil
	}

	// The Thread defines the behavior of the built-in 'print' function.
	thread := &starlark.Thread{
		Name:  "example",
		Print: func(_ *starlark.Thread, msg string) { fmt.Println(msg) },
	}

	// This dictionary defines the pre-declared environment.
	predeclared := starlark.StringDict{
		"greeting": starlark.String("hello"),
		"repeat":   starlark.NewBuiltin("repeat", repeat),
	}

	// Execute a program.
	globals, err := starlark.ExecFile(thread, "apparent/filename.star", data, predeclared)
	if err != nil {
		if evalErr, ok := err.(*starlark.EvalError); ok {
			log.Fatal(evalErr.Backtrace())
		}
		log.Fatal(err)
	}

	// Print the global environment.
	fmt.Println("\nGlobals:")
	for _, name := range globals.Keys() {
		v := globals[name]
		fmt.Printf("%s (%s) = %s\n", name, v.Type(), v.String())
	}

	// Output:
	// hello, world
	// one
	// murmur
	//
	// Globals:
	// squares (list) = [0, 1, 4, 9, 16, 25, 36, 49, 64, 81]
}

// ExampleThread_Load_sequential demonstrates a simple caching
// implementation of 'load' that works sequentially.
func ExampleThread_Load_sequential() {
	fakeFilesystem := map[string]string{
		"c.star": `load("b.star", "b"); c = b + "!"`,
		"b.star": `load("a.star", "a"); b = a + ", world"`,
		"a.star": `a = "Hello"`,
	}

	type entry struct {
		globals starlark.StringDict
		err     error
	}

	cache := make(map[string]*entry)

	var load func(_ *starlark.Thread, module string) (starlark.StringDict, error)
	load = func(_ *starlark.Thread, module string) (starlark.StringDict, error) {
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
			thread := &starlark.Thread{Name: "exec " + module, Load: load}
			globals, err := starlark.ExecFile(thread, module, data, nil)
			e = &entry{globals, err}

			// Update the cache.
			cache[module] = e
		}
		return e.globals, e.err
	}

	thread := &starlark.Thread{Name: "exec c.star", Load: load}
	globals, err := load(thread, "c.star")
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
			"c.star": `load("a.star", "a"); c = a * 2`,
			"b.star": `load("a.star", "a"); b = a * 3`,
			"a.star": `a = 1; print("loaded a")`,
		},
	}

	// We load modules b and c in parallel by concurrent calls to
	// cache.Load.  Both of them load module a, but a is executed
	// only once, as witnessed by the sole output of its print
	// statement.

	ch := make(chan string)
	for _, name := range []string{"b", "c"} {
		go func(name string) {
			globals, err := cache.Load(name + ".star")
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

// TestThread_Load_parallelCycle demonstrates detection
// of cycles during parallel loading.
func TestThreadLoad_ParallelCycle(t *testing.T) {
	cache := &cache{
		cache: make(map[string]*entry),
		fakeFilesystem: map[string]string{
			"c.star": `load("b.star", "b"); c = b * 2`,
			"b.star": `load("a.star", "a"); b = a * 3`,
			"a.star": `load("c.star", "c"); a = c * 5; print("loaded a")`,
		},
	}

	ch := make(chan string)
	for _, name := range "bc" {
		name := string(name)
		go func() {
			_, err := cache.Load(name + ".star")
			if err == nil {
				log.Fatalf("Load of %s.star succeeded unexpectedly", name)
			}
			ch <- err.Error()
		}()
	}
	got := []string{<-ch, <-ch}
	sort.Strings(got)

	// Typically, the c goroutine quickly blocks behind b;
	// b loads a, and a then fails to load c because it forms a cycle.
	// The errors observed by the two goroutines are:
	want1 := []string{
		"cannot load a.star: cannot load c.star: cycle in load graph",                     // from b
		"cannot load b.star: cannot load a.star: cannot load c.star: cycle in load graph", // from c
	}
	// But if the c goroutine is slow to start, b loads a,
	// and a loads c; then c fails to load b because it forms a cycle.
	// The errors this time are:
	want2 := []string{
		"cannot load a.star: cannot load c.star: cannot load b.star: cycle in load graph", // from b
		"cannot load b.star: cycle in load graph",                                         // from c
	}
	if !reflect.DeepEqual(got, want1) && !reflect.DeepEqual(got, want2) {
		t.Error(got)
	}
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
	globals starlark.StringDict
	err     error
	ready   chan struct{}
}

func (c *cache) Load(module string) (starlark.StringDict, error) {
	return c.get(new(cycleChecker), module)
}

// get loads and returns an entry (if not already loaded).
func (c *cache) get(cc *cycleChecker, module string) (starlark.StringDict, error) {
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

func (c *cache) doLoad(cc *cycleChecker, module string) (starlark.StringDict, error) {
	thread := &starlark.Thread{
		Name:  "exec " + module,
		Print: func(_ *starlark.Thread, msg string) { fmt.Println(msg) },
		Load: func(_ *starlark.Thread, module string) (starlark.StringDict, error) {
			// Tunnel the cycle-checker state for this "thread of loading".
			return c.get(cc, module)
		},
	}
	data := c.fakeFilesystem[module]
	return starlark.ExecFile(thread, module, data, nil)
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
