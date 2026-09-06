// Copyright 2018 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package starlarkstruct_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"go.starlark.net/starlarktest"
)

func Test(t *testing.T) {
	testdata := starlarktest.DataFile("starlarkstruct", ".")
	thread := &starlark.Thread{Load: load}
	starlarktest.SetReporter(thread, t)
	filename := filepath.Join(testdata, "testdata/struct.star")
	predeclared := starlark.StringDict{
		"struct": starlark.NewBuiltin("struct", starlarkstruct.Make),
		"gensym": starlark.NewBuiltin("gensym", gensym),
	}
	if _, err := starlark.ExecFile(thread, filename, nil, predeclared); err != nil {
		if err, ok := err.(*starlark.EvalError); ok {
			t.Fatal(err.Backtrace())
		}
		t.Fatal(err)
	}
}

// load implements the 'load' operation as used in the evaluator tests.
func load(thread *starlark.Thread, module string) (starlark.StringDict, error) {
	if module == "assert.star" {
		return starlarktest.LoadAssertModule()
	}
	return nil, fmt.Errorf("load not implemented")
}

// gensym is a built-in function that generates a unique symbol.
func gensym(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var name string
	if err := starlark.UnpackArgs("gensym", args, kwargs, "name", &name); err != nil {
		return nil, err
	}
	return &symbol{name: name}, nil
}

// A symbol is a distinct value that acts as a constructor of "branded"
// struct instances, like a class symbol in Python or a "provider" in Bazel.
type symbol struct{ name string }

var _ starlark.Callable = (*symbol)(nil)

func (sym *symbol) Name() string          { return sym.name }
func (sym *symbol) String() string        { return sym.name }
func (sym *symbol) Type() string          { return "symbol" }
func (sym *symbol) Freeze()               {} // immutable
func (sym *symbol) Truth() starlark.Bool  { return starlark.True }
func (sym *symbol) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: %s", sym.Type()) }

func (sym *symbol) CallInternal(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) > 0 {
		return nil, fmt.Errorf("%s: unexpected positional arguments", sym)
	}
	return starlarkstruct.FromKeywords(sym, kwargs), nil
}

func TestAttrAt(t *testing.T) {
	s := starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
		"b": starlark.MakeInt(2),
		"a": starlark.MakeInt(1),
		"c": starlark.MakeInt(3),
	})

	// Positions are those of AttrNames: sorted by field name.
	names := s.AttrNames()
	if got, want := strings.Join(names, ","), "a,b,c"; got != want {
		t.Fatalf("AttrNames(): want %s, got %s", want, got)
	}
	if s.Len() != len(names) {
		t.Fatalf("Len(): want %d, got %d", len(names), s.Len())
	}
	for pos, name := range names {
		gotName, gotValue := s.AttrAt(pos)
		if gotName != name {
			t.Fatalf("AttrAt(%d): want %s, got %s", pos, name, gotName)
		}
		want, err := s.Attr(name)
		if err != nil {
			t.Fatalf("Attr(%s) failed: %v", name, err)
		}
		if eq, err := starlark.Equal(gotValue, want); err != nil {
			t.Errorf("comparing AttrAt(%d) and Attr(%s): %v", pos, name, err)
		} else if !eq {
			t.Errorf("AttrAt(%d): want %s (= Attr(%s)), got %s", pos, want, name, gotValue)
		}
	}
}

func TestAttrAtOutOfRange(t *testing.T) {
	sym := &symbol{name: "S"}
	for _, test := range []struct {
		desc string
		s    *starlarkstruct.Struct
		pos  int
	}{
		{
			desc: "negative",
			s:    starlarkstruct.FromKeywords(starlarkstruct.Default, []starlark.Tuple{{starlark.String("a"), starlark.MakeInt(1)}}),
			pos:  -1,
		},
		{
			desc: "past end",
			s:    starlarkstruct.FromKeywords(starlarkstruct.Default, []starlark.Tuple{{starlark.String("a"), starlark.MakeInt(1)}}),
			pos:  1,
		},
		{
			desc: "empty struct",
			s:    starlarkstruct.FromKeywords(starlarkstruct.Default, nil),
			pos:  0,
		},
		{
			desc: "branded struct names its constructor",
			s:    starlarkstruct.FromKeywords(sym, []starlark.Tuple{{starlark.String("a"), starlark.MakeInt(1)}}),
			pos:  2,
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
			name, val := test.s.AttrAt(test.pos)
			if name != "" {
				t.Fatalf("AttrAt(%d): want empty name, got %s", test.pos, name)
			}
			if val != nil {
				t.Fatalf("AttrAt(%d): want nil value, got %v", test.pos, val)
			}
		})
	}
}
