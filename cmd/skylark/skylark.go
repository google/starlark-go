// Copyright 2017 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The skylark command interprets a Skylark file.
// With no arguments, it starts a read-eval-print loop (REPL).
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"sort"
	"strings"

	"github.com/google/skylark"
	"github.com/google/skylark/repl"
	"github.com/google/skylark/resolve"
)

// flags
var (
	cpuprofile = flag.String("cpuprofile", "", "gather CPU profile in this file")
	showenv    = flag.Bool("showenv", false, "on success, print final global environment")
)

// non-standard dialect flags
func init() {
	flag.BoolVar(&resolve.AllowFloat, "fp", resolve.AllowFloat, "allow floating-point numbers")
	flag.BoolVar(&resolve.AllowSet, "set", resolve.AllowSet, "allow set data type")
	flag.BoolVar(&resolve.AllowLambda, "lambda", resolve.AllowLambda, "allow lambda expressions")
	flag.BoolVar(&resolve.AllowNestedDef, "nesteddef", resolve.AllowNestedDef, "allow nested def statements")
	flag.BoolVar(&resolve.AllowBitwise, "bitwise", resolve.AllowBitwise, "allow bitwise operations (&, |, ^, ~, <<, and >>)")
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

	thread := &skylark.Thread{Load: repl.MakeLoad()}
	globals := make(skylark.StringDict)

	switch len(flag.Args()) {
	case 0:
		fmt.Println("Welcome to Skylark (github.com/google/skylark)")
		repl.REPL(thread, globals)
	case 1:
		// Execute specified file.
		filename := flag.Args()[0]
		var err error
		globals, err = skylark.ExecFile(thread, filename, nil, nil)
		if err != nil {
			repl.PrintError(err)
			os.Exit(1)
		}
	default:
		log.Fatal("want at most one Skylark file name")
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
