// Copyright 2017 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package resolve_test

import (
	"strings"
	"testing"

	"go.starlark.net/internal/chunkedfile"
	"go.starlark.net/resolve"
	"go.starlark.net/starlarktest"
	"go.starlark.net/syntax"
)

// A test may enable non-standard options by containing (e.g.) "option:recursion".
func getOptions(src string) *syntax.FileOptions {
	return &syntax.FileOptions{
		Set:               option(src, "set"),
		While:             option(src, "while"),
		TopLevelControl:   option(src, "toplevelcontrol"),
		GlobalReassign:    option(src, "globalreassign"),
		LoadBindsGlobally: option(src, "loadbindsglobally"),
		Recursion:         option(src, "recursion"),
	}
}

func option(chunk, name string) bool {
	return strings.Contains(chunk, "option:"+name)
}

func TestResolve(t *testing.T) {
	filename := starlarktest.DataFile("resolve", "testdata/resolve.star")
	for _, chunk := range chunkedfile.Read(filename, t) {
		// A chunk may set options by containing e.g. "option:recursion".
		opts := getOptions(chunk.Source)

		f, err := opts.Parse(filename, chunk.Source, 0)
		if err != nil {
			t.Error(err)
			continue
		}

		if err := resolve.File(f, isPredeclared, isUniversal); err != nil {
			for _, err := range err.(resolve.ErrorList) {
				chunk.GotError(int(err.Pos.Line), err.Msg)
			}
		}
		chunk.Done()
	}
}

func TestDefVarargsAndKwargsSet(t *testing.T) {
	source := "def f(*args, **kwargs): pass\n"
	file, err := syntax.Parse("foo.star", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := resolve.File(file, isPredeclared, isUniversal); err != nil {
		t.Fatal(err)
	}
	fn := file.Stmts[0].(*syntax.DefStmt).Function.(*resolve.Function)
	if !fn.HasVarargs {
		t.Error("HasVarargs not set")
	}
	if !fn.HasKwargs {
		t.Error("HasKwargs not set")
	}
}

func TestLambdaVarargsAndKwargsSet(t *testing.T) {
	source := "f = lambda *args, **kwargs: 0\n"
	file, err := syntax.Parse("foo.star", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := resolve.File(file, isPredeclared, isUniversal); err != nil {
		t.Fatal(err)
	}
	lam := file.Stmts[0].(*syntax.AssignStmt).RHS.(*syntax.LambdaExpr).Function.(*resolve.Function)
	if !lam.HasVarargs {
		t.Error("HasVarargs not set")
	}
	if !lam.HasKwargs {
		t.Error("HasKwargs not set")
	}
}

func isPredeclared(name string) bool { return name == "M" }

func isUniversal(name string) bool { return name == "U" || name == "float" }
