// Copyright 2017 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The skylark command interprets a Skylark file.
// With no arguments, it starts a read-eval-print loop (REPL).
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"sort"
	"strings"

	"github.com/google/skylark"
	"github.com/google/skylark/resolve"
	"github.com/google/skylark/syntax"
	"io"
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
	thread := new(skylark.Thread)
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
	thread := new(skylark.Thread)
	globals := make(skylark.StringDict)

	sc := bufio.NewScanner(os.Stdin)
outer:
	for {
		io.WriteString(os.Stderr, ">>> ")
		if !sc.Scan() {
			break
		}
		line := sc.Text()
		if l := strings.TrimSpace(line); l == "" || l[0] == '#' {
			continue // blank or comment
		}

		// If the line contains a well-formed
		// expression, evaluate it.
		if _, err := syntax.ParseExpr("<stdin>", line); err == nil {
			if v, err := skylark.Eval(thread, "<stdin>", line, globals); err != nil {
				printError(err)
			} else if v != skylark.None {
				fmt.Println(v)
			}
			continue
		}

		// Otherwise assume it is the first of several
		// comprising a file, followed by a blank line.
		var buf bytes.Buffer
		fmt.Fprintln(&buf, line)
		for {
			io.WriteString(os.Stderr, "... ")
			if !sc.Scan() {
				break outer
			}
			line := sc.Text()
			if l := strings.TrimSpace(line); l == "" {
				break // blank
			}
			fmt.Fprintln(&buf, line)
		}
		if err := skylark.ExecFile(thread, "<stdin>", &buf, globals); err != nil {
			printError(err)
		}
	}
	fmt.Println()
}

func printError(err error) {
	if evalErr, ok := err.(*skylark.EvalError); ok {
		fmt.Fprintln(os.Stderr, evalErr.Backtrace())
	} else {
		fmt.Fprintln(os.Stderr, err)
	}
}
