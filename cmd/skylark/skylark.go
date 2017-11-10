// Copyright 2017 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The skylark command interprets a Skylark file.
//
// With no arguments, it starts a read-eval-print loop (REPL).
// If an input line can be parsed as an expression,
// the REPL parses and evaluates it and prints its result.
// Otherwise the REPL reads lines until a blank line,
// then tries again to parse the multi-line input as an
// expression. If the input still cannot be parsed as an expression,
// the REPL parses and executes it as a file (a list of statements),
// for side effects.
package main

// TODO(adonovan):
//
// - Distinguish expressions from statements more precisely.
//   Otherwise e.g. 1 is parsed as an expression but
//   1000000000000000000000000000 is parsed as a file
//   because the scanner fails to convert it to an int64.
//   The spec should clarify limits on numeric literals.
//
// - Unparenthesized tuples are not parsed as a single expression:
//     >>> (1, 2)
//     (1, 2)
//     >>> 1, 2
//     ...
//     >>>
//   This is not necessarily a bug.

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"sort"
	"strings"

	"github.com/chzyer/readline"
	"github.com/google/skylark"
	"github.com/google/skylark/resolve"
	"github.com/google/skylark/syntax"
)

// flags
var (
	cpuprofile = flag.String("cpuprofile", "", "gather CPU profile in this file")
	showenv    = flag.Bool("showenv", false, "on success, print final global environment")
)

// non-standard dialect flags
func init() {
	flag.BoolVar(&resolve.AllowFloat, "fp", resolve.AllowFloat, "allow floating-point numbers")
	flag.BoolVar(&resolve.AllowFreeze, "freeze", resolve.AllowFreeze, "add freeze built-in function")
	flag.BoolVar(&resolve.AllowSet, "set", resolve.AllowSet, "allow set data type")
	flag.BoolVar(&resolve.AllowLambda, "lambda", resolve.AllowLambda, "allow lambda expressions")
	flag.BoolVar(&resolve.AllowNestedDef, "nesteddef", resolve.AllowNestedDef, "allow nested def statements")
}

func main() {
	log.SetPrefix("skylark: ")
	log.SetFlags(0)
	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal(err)
		}
		defer pprof.StopCPUProfile()
	}

	switch len(flag.Args()) {
	case 0:
		repl()
	case 1:
		execfile(flag.Args()[0])
	default:
		log.Fatal("want at most one Skylark file name")
	}
}

func execfile(filename string) {
	thread := &skylark.Thread{Load: load}
	globals := make(skylark.StringDict)
	if err := skylark.ExecFile(thread, filename, nil, globals); err != nil {
		printError(err)
		os.Exit(1)
	}

	// Print the global environment.
	if *showenv {
		var names []string
		for name := range globals {
			if !strings.HasPrefix(name, "_") {
				names = append(names, name)
			}
		}
		sort.Strings(names)
		for _, name := range names {
			fmt.Fprintf(os.Stderr, "%s = %s\n", name, globals[name])
		}
	}
}

func repl() {
	thread := &skylark.Thread{Load: load}
	globals := make(skylark.StringDict)

	rl, err := readline.New(">>> ")
	if err != nil {
		printError(err)
		return
	}
	defer rl.Close()
outer:
	for {
		rl.SetPrompt(">>> ")
		line, err := rl.Readline()
		if err != nil {
			break
		}

		if l := strings.TrimSpace(line); l == "" || l[0] == '#' {
			continue // blank or comment
		}

		// If the line contains a well-formed expression, evaluate it.
		if _, err := syntax.ParseExpr("<stdin>", line); err == nil {
			if v, err := skylark.Eval(thread, "<stdin>", line, globals); err != nil {
				printError(err)
			} else if v != skylark.None {
				fmt.Println(v)
			}
			continue
		}

		// If the input so far is a single load or assignment statement,
		// execute it without waiting for a blank line.
		if f, err := syntax.Parse("<stdin>", line); err == nil && len(f.Stmts) == 1 {
			switch f.Stmts[0].(type) {
			case *syntax.AssignStmt, *syntax.LoadStmt:
				// Execute it as a file.
				if err := execFileNoFreeze(thread, line, globals); err != nil {
					printError(err)
				}
				continue
			}
		}

		// Otherwise assume it is the first of several
		// comprising a file, followed by a blank line.
		var buf bytes.Buffer
		fmt.Fprintln(&buf, line)
		for {
			rl.SetPrompt("... ")
			line, err := rl.Readline()
			if err != nil {
				break outer
			}
			if l := strings.TrimSpace(line); l == "" {
				break // blank
			}
			fmt.Fprintln(&buf, line)
		}
		text := buf.Bytes()

		// Try parsing it once more as an expression,
		// such as a call spread over several lines:
		//   f(
		//     1,
		//     2
		//   )
		if _, err := syntax.ParseExpr("<stdin>", text); err == nil {
			if v, err := skylark.Eval(thread, "<stdin>", text, globals); err != nil {
				printError(err)
			} else if v != skylark.None {
				fmt.Println(v)
			}
			continue
		}

		// Execute it as a file.
		if err := execFileNoFreeze(thread, text, globals); err != nil {
			printError(err)
		}
	}
	fmt.Println()
}

// execFileNoFreeze is skylark.ExecFile without globals.Freeze().
func execFileNoFreeze(thread *skylark.Thread, src interface{}, globals skylark.StringDict) error {
	// parse
	f, err := syntax.Parse("<stdin>", src)
	if err != nil {
		return err
	}

	// resolve
	if err := resolve.File(f, globals.Has, skylark.Universe.Has); err != nil {
		return err

	}

	// execute
	fr := thread.Push(globals, len(f.Locals))
	defer thread.Pop()
	return fr.ExecStmts(f.Stmts)
}

type entry struct {
	globals skylark.StringDict
	err     error
}

var cache = make(map[string]*entry)

// load is a simple sequential implementation of module loading.
func load(thread *skylark.Thread, module string) (skylark.StringDict, error) {
	e, ok := cache[module]
	if e == nil {
		if ok {
			// request for package whose loading is in progress
			return nil, fmt.Errorf("cycle in load graph")
		}

		// Add a placeholder to indicate "load in progress".
		cache[module] = nil

		// Load it.
		thread := &skylark.Thread{Load: load}
		globals := make(skylark.StringDict)
		err := skylark.ExecFile(thread, module, nil, globals)
		e = &entry{globals, err}

		// Update the cache.
		cache[module] = e
	}
	return e.globals, e.err
}

func printError(err error) {
	if evalErr, ok := err.(*skylark.EvalError); ok {
		fmt.Fprintln(os.Stderr, evalErr.Backtrace())
	} else {
		fmt.Fprintln(os.Stderr, err)
	}
}
