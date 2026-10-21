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

// method returns the named method of recv, bound to its receiver.
func method(t *testing.T, recv starlark.Value, name string) starlark.Value {
	t.Helper()
	m, err := recv.(starlark.HasAttrs).Attr(name)
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

// TestReplaceFuncError checks that an error returned by the replacement
// function is reported to the caller, and that the function is not called
// again once it has failed.
func TestReplaceFuncError(t *testing.T) {
	thread := &starlark.Thread{}
	rx := mustCompile(t, thread, "a")

	oops := errors.New("oops")
	calls := 0
	repl := starlark.NewBuiltin("repl", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		calls++
		return nil, oops
	})

	_, err := starlark.Call(thread, method(t, rx, "replace"),
		starlark.Tuple{starlark.String("banana"), repl}, nil)
	if !errors.Is(err, oops) {
		t.Errorf("replace error: got %v, want %v", err, oops)
	}
	if calls != 1 {
		t.Errorf("replacement function was called %d times, want 1", calls)
	}
}

func TestReplaceFuncWrongType(t *testing.T) {
	thread := &starlark.Thread{}
	rx := mustCompile(t, thread, "a")

	repl := starlark.NewBuiltin("repl", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		return starlark.MakeInt(1), nil
	})

	_, err := starlark.Call(thread, method(t, rx, "replace"),
		starlark.Tuple{starlark.String("banana"), repl}, nil)
	const want = "replace: replacement function returned int, want string"
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("replace error: got %v, want error containing %q", err, want)
	}
}

// TestReplaceFuncArgument checks that the replacement function is called with
// the match, on the calling thread, so that cancellation and thread-locals
// apply to it.
func TestReplaceFuncArgument(t *testing.T) {
	thread := &starlark.Thread{}
	thread.SetLocal("prefix", "@")
	rx := mustCompile(t, thread, `(\w)(\d)`)

	repl := starlark.NewBuiltin("repl", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		m, ok := args[0].(*Match)
		if !ok {
			return nil, fmt.Errorf("got %s, want regexp.match", args[0].Type())
		}
		prefix, _ := thread.Local("prefix").(string)
		return starlark.String(fmt.Sprintf("%s%s:%d-%d:%s", prefix, m.text(), m.loc[0], m.loc[1],
			m.submatch(1).(*Submatch).text())), nil
	})

	res, err := starlark.Call(thread, method(t, rx, "replace"),
		starlark.Tuple{starlark.String("-a1-"), repl}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := res, starlark.String("-@a1:1-3:1-"); got != want {
		t.Errorf("replace: got %v, want %v", got, want)
	}
}

// TestFindallIsReiterable checks that the value returned by findall may be
// iterated more than once, and that its truth does not consume it.
func TestFindallIsReiterable(t *testing.T) {
	thread := &starlark.Thread{}
	rx := mustCompile(t, thread, "a")

	res, err := starlark.Call(thread, method(t, rx, "findall"), starlark.Tuple{starlark.String("banana")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	matches := res.(*Matches)
	if !matches.Truth() {
		t.Error("Truth: got False, want True")
	}
	for round := 1; round <= 2; round++ {
		var texts []string
		iter := matches.Iterate()
		var v starlark.Value
		for iter.Next(&v) {
			texts = append(texts, v.(*Match).text())
		}
		iter.Done()
		if want := []string{"a", "a", "a"}; !reflect.DeepEqual(texts, want) {
			t.Errorf("iteration %d: got %q, want %q", round, texts, want)
		}
	}

	// An empty result is false, and yields nothing.
	res, err = starlark.Call(thread, method(t, rx, "findall"), starlark.Tuple{starlark.String("melon")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if matches := res.(*Matches); matches.Truth() {
		t.Error("Truth of empty matches: got True, want False")
	}
}

func TestModuleMembers(t *testing.T) {
	want := []string{"compile", "escape"}
	if got := Module.Members.Keys(); !reflect.DeepEqual(got, want) {
		t.Errorf("Module.Members: got %q, want %q", got, want)
	}
}

func TestAttrNames(t *testing.T) {
	thread := &starlark.Thread{}
	rx := mustCompile(t, thread, `b(an)*a`).(*Regexp)
	m := rx.find(t, thread, "banana")

	for _, test := range []struct {
		recv starlark.Value
		want []string
	}{
		{rx, []string{"find", "findall", "groupnames", "matches", "pattern", "replace", "split"}},
		{m, []string{"end", "start", "submatch", "submatches", "text"}},
		{m.submatch(0), []string{"end", "start", "text"}},
	} {
		got := test.recv.(starlark.HasAttrs).AttrNames()
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("%s.AttrNames: got %q, want %q", test.recv.Type(), got, test.want)
		}
		for _, name := range test.want {
			v, err := test.recv.(starlark.HasAttrs).Attr(name)
			if err != nil {
				t.Errorf("%s.Attr(%q): %v", test.recv.Type(), name, err)
			} else if v == nil {
				t.Errorf("%s.Attr(%q): got nil, want a value", test.recv.Type(), name)
			}
		}

		// An unknown attribute yields (nil, nil), which the interpreter
		// turns into a "no .x field or method" error.
		v, err := test.recv.(starlark.HasAttrs).Attr("nonesuch")
		if v != nil || err != nil {
			t.Errorf("%s.Attr(%q): got (%v, %v), want (nil, nil)", test.recv.Type(), "nonesuch", v, err)
		}
	}
}

// find is a test helper that calls the find method of r.
func (r *Regexp) find(t *testing.T, thread *starlark.Thread, s string) *Match {
	t.Helper()
	res, err := starlark.Call(thread, method(t, r, "find"), starlark.Tuple{starlark.String(s)}, nil)
	if err != nil {
		t.Fatalf("find(%q): %v", s, err)
	}
	m, ok := res.(*Match)
	if !ok {
		t.Fatalf("find(%q): got %v, want a match", s, res)
	}
	return m
}

func TestValueSemantics(t *testing.T) {
	thread := &starlark.Thread{}
	rx := mustCompile(t, thread, `b(an)*a`).(*Regexp)

	if got, want := rx.Type(), "regexp.regexp"; got != want {
		t.Errorf("Type: got %q, want %q", got, want)
	}
	if got, want := rx.String(), `regexp.compile("b(an)*a")`; got != want {
		t.Errorf("String: got %q, want %q", got, want)
	}
	rx.Freeze() // no-op; a compiled regexp is immutable

	m := rx.find(t, thread, "--banana--")
	if got, want := m.String(), `<regexp.match "banana" at 2:8>`; got != want {
		t.Errorf("match String: got %q, want %q", got, want)
	}
	if got, want := m.submatch(0).(*Submatch).String(), `<regexp.submatch "an" at 5:7>`; got != want {
		t.Errorf("submatch String: got %q, want %q", got, want)
	}
	// A match is true even when its text is empty, so that the result of
	// find may be tested for success.
	empty := mustCompile(t, thread, `x*`).(*Regexp).find(t, thread, "zzz")
	if !empty.Truth() {
		t.Error("Truth of empty match: got False, want True")
	}

	// Equal matches have equal hashes, and are equal under ==.
	m2 := rx.find(t, thread, "--banana--")
	h1, err := m.Hash()
	if err != nil {
		t.Fatal(err)
	}
	h2, err := m2.Hash()
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Errorf("Hash: got %d and %d for equal matches, want equal values", h1, h2)
	}
	for _, test := range []struct {
		x, y starlark.Value
		want bool
	}{
		{m, m2, true},
		{m, rx.find(t, thread, "banana"), false}, // different offsets
		{m.submatch(0), m2.submatch(0), true},
		{m.submatch(0), rx.find(t, thread, "bana").submatch(0), false},
	} {
		got, err := starlark.Equal(test.x, test.y)
		if err != nil {
			t.Errorf("Equal(%v, %v): %v", test.x, test.y, err)
		} else if got != test.want {
			t.Errorf("Equal(%v, %v): got %t, want %t", test.x, test.y, got, test.want)
		}
	}

	// Matches are not ordered.
	if _, err := m.CompareSameType(syntax.LT, m2, 1); err == nil {
		t.Error("match < match: got no error, want one")
	}

	// Iterable containers are not hashable.
	for _, v := range []starlark.Value{
		&Matches{re: rx.re, s: "banana"},
		&Submatches{m: m},
	} {
		if _, err := v.Hash(); err == nil {
			t.Errorf("%s.Hash: got no error, want one", v.Type())
		}
	}
}

// TestExample exercises the API as a program would, using the examples from
// the proposal that introduced this module.
func TestExample(t *testing.T) {
	const src = `
rx = regexp.compile(r"(\w+)=(\d+)")

m = rx.find(s)
out = [
    bool(rx.find(s)),                                   # match test
    m.submatches[0].text,                               # extract a group
    m.start,                                            # offset of the match
    bool(rx.findall(s)),                                # findall truthiness
    {x.submatches[0].text: int(x.submatches[1].text) for x in rx.findall(s)},
    len(list(rx.findall(s))),                           # count
    [(x.start, x.end) for x in rx.findall(s)],          # all offsets
    rx.replace(s, "<redacted>"),                        # literal replacement
    rx.replace(s, lambda x: x.submatches[1].text + "=" + x.submatches[0].text),
    rx.replace(s, lambda x: x.text.upper()),
    regexp.compile(r"\b" + regexp.escape("a.b") + r"\b").matches("x a.b y"),
    rx.groupnames,
]
named = regexp.compile(r"(?P<year>\d{4})-(?P<month>\d{2})").find("on 2026-10-19")
out.extend([named.submatch("year").text, named.submatch("month").text])
opt = regexp.compile(r"a(x)?").find("a")
out.append(opt.submatches[0].text if opt.submatches[0] else "")
`
	thread := &starlark.Thread{}
	globals, err := starlark.ExecFileOptions(&syntax.FileOptions{}, thread, "example.star", src,
		starlark.StringDict{
			"regexp": Module,
			"s":      starlark.String("a=1, b=22"),
		})
	if err != nil {
		t.Fatal(err)
	}
	want := `[True, "a", 0, True, {"a": 1, "b": 22}, 2, [(0, 3), (5, 9)], "<redacted>, <redacted>", "1=a, 22=b", "A=1, B=22", True, [None, None], "2026", "10", ""]`
	if got := fmt.Sprint(globals["out"]); got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}
