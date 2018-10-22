// Copyright 2018 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package skylarkstruct_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/google/skylark"
	"github.com/google/skylark/resolve"
	"github.com/google/skylark/skylarkstruct"
	"github.com/google/skylark/skylarktest"
)

func init() {
	// The tests make extensive use of these not-yet-standard features.
	resolve.AllowLambda = true
	resolve.AllowNestedDef = true
	resolve.AllowFloat = true
	resolve.AllowSet = true
}

func Test(t *testing.T) {
	testdata := skylarktest.DataFile("skylark/skylarkstruct", ".")
	thread := &skylark.Thread{Load: load}
	skylarktest.SetReporter(thread, t)
	filename := filepath.Join(testdata, "testdata/struct.sky")
	predeclared := skylark.StringDict{
		"struct": skylark.NewBuiltin("struct", skylarkstruct.Make),
		"gensym": skylark.NewBuiltin("gensym", gensym),
	}
	if _, err := skylark.ExecFile(thread, filename, nil, predeclared); err != nil {
		if err, ok := err.(*skylark.EvalError); ok {
			t.Fatal(err.Backtrace())
		}
		t.Fatal(err)
	}
}

// load implements the 'load' operation as used in the evaluator tests.
func load(thread *skylark.Thread, module string) (skylark.StringDict, error) {
	if module == "assert.sky" {
		return skylarktest.LoadAssertModule()
	}
	return nil, fmt.Errorf("load not implemented")
}

// gensym is a built-in function that generates a unique symbol.
func gensym(thread *skylark.Thread, _ *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
	var name string
	if err := skylark.UnpackArgs("gensym", args, kwargs, "name", &name); err != nil {
		return nil, err
	}
	return &symbol{name: name}, nil
}

// A symbol is a distinct value that acts as a constructor of "branded"
// struct instances, like a class symbol in Python or a "provider" in Bazel.
type symbol struct{ name string }

var _ skylark.Callable = (*symbol)(nil)

func (sym *symbol) Name() string          { return sym.name }
func (sym *symbol) String() string        { return sym.name }
func (sym *symbol) Type() string          { return "symbol" }
func (sym *symbol) Freeze()               {} // immutable
func (sym *symbol) Truth() skylark.Bool   { return skylark.True }
func (sym *symbol) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: %s", sym.Type()) }

func (sym *symbol) CallInternal(thread *skylark.Thread, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
	if len(args) > 0 {
		return nil, fmt.Errorf("%s: unexpected positional arguments", sym)
	}
	return skylarkstruct.FromKeywords(sym, kwargs), nil
}
