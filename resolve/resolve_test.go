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

func setOptions(src string) {
	resolve.AllowFloat = option(src, "float")
	resolve.AllowGlobalReassign = option(src, "globalreassign")
	resolve.AllowLambda = option(src, "lambda")
	resolve.AllowNestedDef = option(src, "nesteddef")
	resolve.AllowRecursion = option(src, "recursion")
	resolve.AllowSet = option(src, "set")
	resolve.LoadBindsGlobally = option(src, "loadbindsglobally")
}

func option(chunk, name string) bool {
	return strings.Contains(chunk, "option:"+name)
}

func TestResolve(t *testing.T) {
	defer setOptions("")
	filename := starlarktest.DataFile("resolve", "testdata/resolve.star")
	for _, chunk := range chunkedfile.Read(filename, t) {
		f, err := syntax.Parse(filename, chunk.Source, 0)
		if err != nil {
			t.Error(err)
			continue
		}

		// A chunk may set options by containing e.g. "option:float".
		setOptions(chunk.Source)

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
	resolve.AllowLambda = true
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
