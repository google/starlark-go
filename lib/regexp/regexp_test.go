// Copyright 2025 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package regexp

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// mustCompile calls regexp.compile and returns the resulting value.
func mustCompile(t *testing.T, thread *starlark.Thread, pattern string) starlark.Value {
	t.Helper()
	rx, err := starlark.Call(thread, Module.Members["compile"], starlark.Tuple{starlark.String(pattern)}, nil)
	if err != nil {
		t.Fatalf("compile(%q): %v", pattern, err)
	}
	return rx
}

// method returns the named method of rx, bound to its receiver.
func method(t *testing.T, rx starlark.Value, name string) starlark.Value {
	t.Helper()
	m, err := rx.(starlark.HasAttrs).Attr(name)
	if err != nil {
		t.Fatalf("Attr(%q): %v", name, err)
	}
	if m == nil {
		t.Fatalf("Attr(%q): no such method", name)
	}
	return m
}

func TestCompileErrors(t *testing.T) {
	thread := &starlark.Thread{}
	for _, test := range []struct{ pattern, want string }{
		{`(`, "error parsing regexp: missing closing ): `(`"},
		// \C matches a single byte; it is infeasible for implementations
		// whose strings are not byte-oriented, and Go rejects it too.
		{`\C`, "error parsing regexp: invalid escape sequence: `\\C`"},
		{`a**`, "error parsing regexp: invalid nested repetition operator: `**`"},
	} {
		_, err := starlark.Call(thread, Module.Members["compile"], starlark.Tuple{starlark.String(test.pattern)}, nil)
		if err == nil {
			t.Errorf("compile(%q) succeeded unexpectedly", test.pattern)
			continue
		}
		if got := err.(*starlark.EvalError).Unwrap().Error(); got != test.want {
			t.Errorf("compile(%q) error: got %q, want %q", test.pattern, got, test.want)
		}
	}
}

// TestReplaceAllFuncError checks that an error returned by the replacement
// function is reported to the caller, and that the function is not called
// again once it has failed.
func TestReplaceAllFuncError(t *testing.T) {
	thread := &starlark.Thread{}
	rx := mustCompile(t, thread, "a")

	oops := errors.New("oops")
	calls := 0
	repl := starlark.NewBuiltin("repl", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		calls++
		return nil, oops
	})

	_, err := starlark.Call(thread, method(t, rx, "replace_all"),
		starlark.Tuple{starlark.String("banana"), repl}, nil)
	if !errors.Is(err, oops) {
		t.Errorf("replace_all error: got %v, want %v", err, oops)
	}
	if calls != 1 {
		t.Errorf("replacement function was called %d times, want 1", calls)
	}
}

func TestReplaceAllFuncWrongType(t *testing.T) {
	thread := &starlark.Thread{}
	rx := mustCompile(t, thread, "a")

	repl := starlark.NewBuiltin("repl", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		return starlark.MakeInt(1), nil
	})

	_, err := starlark.Call(thread, method(t, rx, "replace_all"),
		starlark.Tuple{starlark.String("banana"), repl}, nil)
	const want = "replace_all: replacement function returned int, want string"
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("replace_all error: got %v, want error containing %q", err, want)
	}
}

// TestReplaceAllFuncCallsThread checks that the replacement function runs on
// the calling thread, so that cancellation and thread-locals apply to it.
func TestReplaceAllFuncCallsThread(t *testing.T) {
	thread := &starlark.Thread{}
	thread.SetLocal("key", "value")
	rx := mustCompile(t, thread, "a")

	repl := starlark.NewBuiltin("repl", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		local, _ := thread.Local("key").(string)
		return starlark.String(local), nil
	})

	res, err := starlark.Call(thread, method(t, rx, "replace_all"),
		starlark.Tuple{starlark.String("ba"), repl}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := res, starlark.String("bvalue"); got != want {
		t.Errorf("replace_all: got %v, want %v", got, want)
	}
}

func TestModuleMembers(t *testing.T) {
	want := []string{"compile", "quote"}
	if got := Module.Members.Keys(); !reflect.DeepEqual(got, want) {
		t.Errorf("Module.Members: got %q, want %q", got, want)
	}
}

func TestRegexpValue(t *testing.T) {
	thread := &starlark.Thread{}
	rx := mustCompile(t, thread, `b(an)*a`).(*Regexp)

	if got, want := rx.Type(), "regexp.regexp"; got != want {
		t.Errorf("Type: got %q, want %q", got, want)
	}
	if got, want := rx.String(), `regexp.compile("b(an)*a")`; got != want {
		t.Errorf("String: got %q, want %q", got, want)
	}
	if got, want := rx.Truth(), starlark.True; got != want {
		t.Errorf("Truth: got %v, want %v", got, want)
	}
	rx.Freeze() // no-op; a compiled regexp is immutable

	// Equal patterns have equal hashes.
	h1, err := rx.Hash()
	if err != nil {
		t.Fatal(err)
	}
	h2, err := mustCompile(t, thread, `b(an)*a`).(*Regexp).Hash()
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Errorf("Hash: got %d and %d for the same pattern, want equal values", h1, h2)
	}

	want := []string{
		"find",
		"find_all",
		"find_all_submatches",
		"find_submatches",
		"matches",
		"replace_all",
		"split",
		"pattern",
	}
	if got := rx.AttrNames(); !reflect.DeepEqual(got, want) {
		t.Errorf("AttrNames: got %q, want %q", got, want)
	}
	for _, name := range want {
		if _, err := rx.Attr(name); err != nil {
			t.Errorf("Attr(%q): %v", name, err)
		}
	}

	// An unknown attribute yields (nil, nil), which the interpreter
	// turns into a "no .x field or method" error.
	v, err := rx.Attr("nonesuch")
	if v != nil || err != nil {
		t.Errorf("Attr(%q): got (%v, %v), want (nil, nil)", "nonesuch", v, err)
	}
}

// TestExample exercises the API as a program would, using the example from
// the proposal that introduced this module.
func TestExample(t *testing.T) {
	const src = `
rx = regexp.compile("b(an)*a")
out = [
    rx.matches("banana"),
    rx.find("--banana--"),
    rx.find_all("banana bana", 1),
    rx.replace_all("banana", "X"),
    rx.find_submatches("banana"),
    regexp.quote("a.b"),
]
`
	thread := &starlark.Thread{}
	globals, err := starlark.ExecFileOptions(&syntax.FileOptions{}, thread, "example.star", src,
		starlark.StringDict{"regexp": Module})
	if err != nil {
		t.Fatal(err)
	}
	want := `[True, "banana", ["banana"], "X", ["banana", "an"], "a\\.b"]`
	if got := fmt.Sprint(globals["out"]); got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}
