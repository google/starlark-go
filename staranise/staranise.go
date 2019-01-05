//go:generate go run go.starlark.net/stargo/cmd/gen -o pkgs.go -p main fmt golang.org/x/tools/go/analysis golang.org/x/tools/go/analysis/passes/inspect go/token go/ast go/parser go/types

// The staranise command is a vet analysis that runs checkers written in Starlark.
package main

import (
	"fmt"
	"log"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"

	"go.starlark.net/resolve"
	"go.starlark.net/stargo"
	"go.starlark.net/starlark"
)

func main() {
	log.SetPrefix("staranise: ")
	log.SetFlags(0)

	resolve.AllowFloat = true
	resolve.AllowLambda = true
	resolve.AllowNestedDef = true
	resolve.AllowBitwise = true
	resolve.AllowRecursion = true
	resolve.AllowGlobalReassign = true
	resolve.AllowAddressing = true // TODO: crucial! set within stargo?

	predeclared := starlark.StringDict{
		"go": stargo.Builtins,
	}
	thread := &starlark.Thread{
		Name: "staranise",
		Load: func(thread *starlark.Thread, module string) (starlark.StringDict, error) {
			if module == "go" {
				return goPackages, nil
			}
			return nil, fmt.Errorf(`only load("go") is supported`)
		},
	}

	// Run one file.
	// TODO: either we should load many files,
	// or a single script should load and aggregate all the analyzers.
	globals, err := starlark.ExecFile(thread, "staranise/iterator_leak.star", nil, predeclared)
	if err != nil {
		if evalErr, ok := err.(*starlark.EvalError); ok {
			log.Fatal(evalErr.Backtrace())
		} else {
			log.Fatal(err)
		}
	}

	// Find the analyzers.
	var analyzers []*analysis.Analyzer
	for _, v := range globals {
		if v, ok := v.(stargo.Value); ok {
			if a, ok := v.Reflect().Interface().(*analysis.Analyzer); ok {
				analyzers = append(analyzers, a)
			}
		}
	}

	// Run them.
	multichecker.Main(analyzers...)
}
