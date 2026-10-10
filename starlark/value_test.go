// Copyright 2017 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package starlark_test

// This file defines tests of the Value API.

import (
	"fmt"
	"iter"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.starlark.net/starlark"
)

func TestStringMethod(t *testing.T) {
	s := starlark.String("hello")
	for i, test := range [][2]string{
		// quoted string:
		{s.String(), `"hello"`},
		{fmt.Sprintf("%s", s), `"hello"`},
		{fmt.Sprintf("%+s", s), `"hello"`},
		{fmt.Sprintf("%v", s), `"hello"`},
		{fmt.Sprintf("%+v", s), `"hello"`},
		// unquoted:
		{s.GoString(), `hello`},
		{fmt.Sprintf("%#v", s), `hello`},
	} {
		got, want := test[0], test[1]
		if got != want {
			t.Errorf("#%d: got <<%s>>, want <<%s>>", i, got, want)
		}
	}
}

func TestListAppend(t *testing.T) {
	l := starlark.NewList(nil)
	l.Append(starlark.String("hello"))
	res, ok := starlark.AsString(l.Index(0))
	if !ok {
		t.Errorf("failed list.Append() got: %s, want: starlark.String", l.Index(0).Type())
	}
	if res != "hello" {
		t.Errorf("failed list.Append() got: %+v, want: hello", res)
	}
}

func TestParamDefault(t *testing.T) {
	tests := []struct {
		desc         string
		starFn       string
		wantDefaults []starlark.Value
	}{
		{
			desc:         "function with all required params",
			starFn:       "all_required",
			wantDefaults: []starlark.Value{nil, nil, nil},
		},
		{
			desc:   "function with all optional params",
			starFn: "all_opt",
			wantDefaults: []starlark.Value{
				starlark.String("a"),
				starlark.None,
				starlark.String(""),
			},
		},
		{
			desc:   "function with required and optional params",
			starFn: "mix_required_opt",
			wantDefaults: []starlark.Value{
				nil,
				nil,
				starlark.String("c"),
				starlark.String("d"),
			},
		},
		{
			desc:   "function with required, optional, and varargs params",
			starFn: "with_varargs",
			wantDefaults: []starlark.Value{
				nil,
				starlark.String("b"),
				nil,
			},
		},
		{
			desc:   "function with required, optional, varargs, and keyword-only params",
			starFn: "with_varargs_kwonly",
			wantDefaults: []starlark.Value{
				nil,
				starlark.String("b"),
				starlark.String("c"),
				nil,
				nil,
			},
		},
		{
			desc:   "function with required, optional, and keyword-only params",
			starFn: "with_kwonly",
			wantDefaults: []starlark.Value{
				nil,
				starlark.String("b"),
				starlark.String("c"),
				nil,
			},
		},
		{
			desc:   "function with required, optional, and kwargs params",
			starFn: "with_kwargs",
			wantDefaults: []starlark.Value{
				nil,
				starlark.String("b"),
				starlark.String("c"),
				nil,
			},
		},
		{
			desc:   "function with required, optional, varargs, kw-only, and kwargs params",
			starFn: "with_varargs_kwonly_kwargs",
			wantDefaults: []starlark.Value{
				nil,
				starlark.String("b"),
				starlark.String("c"),
				nil,
				starlark.String("e"),
				nil,
				nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			thread := &starlark.Thread{}
			filename := "testdata/function_param.star"
			globals, err := starlark.ExecFile(thread, filename, nil, nil)
			if err != nil {
				t.Fatalf("ExecFile(%v, %q) failed: %v", thread, filename, err)
			}

			fn, ok := globals[tt.starFn].(*starlark.Function)
			if !ok {
				t.Fatalf("value %v is not a Starlark function", globals[tt.starFn])
			}

			var paramDefaults []starlark.Value
			for i := 0; i < fn.NumParams(); i++ {
				paramDefaults = append(paramDefaults, fn.ParamDefault(i))
			}
			if diff := cmp.Diff(tt.wantDefaults, paramDefaults); diff != "" {
				t.Errorf("param defaults got diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestElementsIteratorCount(t *testing.T) {
	d := starlark.NewDict(1)
	m := &fakeMapping{Dict: d}
	assertNoElementsFastPath(t, m)

	// Asking for the sequence must not start an iteration.
	_ = starlark.Elements(m)
	err := d.SetKey(starlark.String("one"), starlark.MakeInt(1))
	if err != nil {
		t.Error("Elements(m) started an iteration")
	}
}

func TestEntriesIteratorCount(t *testing.T) {
	d := starlark.NewDict(1)
	m := &fakeMapping{Dict: d}
	assertNoEntriesFastPath(t, m)

	// Asking for the sequence must not start an iteration.
	_ = starlark.Entries(m)
	err := d.SetKey(starlark.String("one"), starlark.MakeInt(1))
	if err != nil {
		t.Error("Entries(m) started an iteration")
	}
}

// A fakeMapping is a starlark.IterableMapping that deliberately does not
// implement the Elements or Entries fast paths, so that the standalone
// starlark.Elements and starlark.Entries functions must use the generic
// Iterate/Done code path.
type fakeMapping struct {
	*starlark.Dict
}

var _ starlark.IterableMapping = (*fakeMapping)(nil)

func (m *fakeMapping) Elements() {}

func (m *fakeMapping) Entries() {}

func assertNoElementsFastPath(t *testing.T, v any) {
	t.Helper()
	if _, ok := v.(interface {
		Elements() iter.Seq[starlark.Value]
	}); ok {
		t.Fatalf("%T has an Elements fast path, so this test would not exercise Elements' generic code path", v)
	}
}

func assertNoEntriesFastPath(t *testing.T, v any) {
	t.Helper()
	if _, ok := v.(interface {
		Entries() iter.Seq2[starlark.Value, starlark.Value]
	}); ok {
		t.Fatalf("%T has an Entries fast path, so this test would not exercise Entries' generic code path", v)
	}
}
