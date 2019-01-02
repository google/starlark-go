// Copyright 2017 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The starlark command interprets a Starlark file.
// With no arguments, it starts a read-eval-print loop (REPL).
package main // import "go.starlark.net/cmd/starlark"

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"sort"
	"strings"

	"go.starlark.net/repl"
	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
)

// flags
var (
	cpuprofile = flag.String("cpuprofile", "", "gather CPU profile in this file")
	showenv    = flag.Bool("showenv", false, "on success, print final global environment")
	execprog   = flag.String("c", "", "execute program `prog`")
)

// non-standard dialect flags
func init() {
	flag.BoolVar(&resolve.AllowFloat, "float", resolve.AllowFloat, "allow floating-point numbers")
	flag.BoolVar(&resolve.AllowSet, "set", resolve.AllowSet, "allow set data type")
	flag.BoolVar(&resolve.AllowLambda, "lambda", resolve.AllowLambda, "allow lambda expressions")
	flag.BoolVar(&resolve.AllowNestedDef, "nesteddef", resolve.AllowNestedDef, "allow nested def statements")
	flag.BoolVar(&resolve.AllowBitwise, "bitwise", resolve.AllowBitwise, "allow bitwise operations (&, |, ^, ~, <<, and >>)")
	flag.BoolVar(&resolve.AllowRecursion, "recursion", resolve.AllowRecursion, "allow while statements and recursive functions")
	flag.BoolVar(&resolve.AllowGlobalReassign, "globalreassign", resolve.AllowGlobalReassign, "allow reassignment of globals, and if/for/while statements at top level")
}

func main() {
	log.SetPrefix("starlark: ")
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

	thread := &starlark.Thread{Load: repl.MakeLoad()}
	globals := make(starlark.StringDict)

	switch {
	case flag.NArg() == 1 || *execprog != "":
		var (
			filename string
			src      interface{}
			err      error
		)
		if *execprog != "" {
			// Execute provided program.
			filename = "cmdline"
			src = *execprog
		} else {
			// Execute specified file.
			filename = flag.Arg(0)
		}
		thread.Name = "exec " + filename
		globals, err = starlark.ExecFile(thread, filename, src, nil)
		if err != nil {
			repl.PrintError(err)
			os.Exit(1)
		}
	case flag.NArg() == 0:
		fmt.Println("Welcome to Starlark (go.starlark.net)")
		thread.Name = "REPL"
		repl.REPL(thread, globals)
	default:
		log.Fatal("want at most one Starlark file name")
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
