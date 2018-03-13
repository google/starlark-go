// Copyright 2017 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package resolve_test

import (
	"strings"
	"testing"

	"github.com/google/skylark/internal/chunkedfile"
	"github.com/google/skylark/resolve"
	"github.com/google/skylark/skylarktest"
	"github.com/google/skylark/syntax"
)

func TestResolve(t *testing.T) {
	filename := skylarktest.DataFile("skylark/resolve", "testdata/resolve.sky")
	for _, chunk := range chunkedfile.Read(filename, t) {
		f, err := syntax.Parse(filename, chunk.Source, 0)
		if err != nil {
			t.Error(err)
			continue
		}

		// A chunk may set options by containing e.g. "option:float".
		resolve.AllowNestedDef = option(chunk.Source, "nesteddef")
		resolve.AllowLambda = option(chunk.Source, "lambda")
		resolve.AllowFloat = option(chunk.Source, "float")
		resolve.AllowSet = option(chunk.Source, "set")
		resolve.AllowGlobalReassign = option(chunk.Source, "global_reassign")

		if err := resolve.File(f, isPredeclared, isUniversal); err != nil {
			for _, err := range err.(resolve.ErrorList) {
				chunk.GotError(int(err.Pos.Line), err.Msg)
			}
		}
		chunk.Done()
	}
}

func option(chunk, name string) bool {
	return strings.Contains(chunk, "option:"+name)
}

func TestDefVarargsAndKwargsSet(t *testing.T) {
	source := "def f(*args, **kwargs): pass\n"
	file, err := syntax.Parse("foo.sky", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := resolve.File(file, isPredeclared, isUniversal); err != nil {
		t.Fatal(err)
	}
	fn := file.Stmts[0].(*syntax.DefStmt)
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
	file, err := syntax.Parse("foo.sky", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := resolve.File(file, isPredeclared, isUniversal); err != nil {
		t.Fatal(err)
	}
	lam := file.Stmts[0].(*syntax.AssignStmt).RHS.(*syntax.LambdaExpr)
	if !lam.HasVarargs {
		t.Error("HasVarargs not set")
	}
	if !lam.HasKwargs {
		t.Error("HasKwargs not set")
	}
}

func isPredeclared(name string) bool { return name == "M" }

func isUniversal(name string) bool { return name == "U" || name == "float" }
