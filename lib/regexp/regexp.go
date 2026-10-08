// Copyright 2025 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package regexp provides regular expression matching functions.
package regexp // import "go.starlark.net/lib/regexp"

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// Module regexp is a Starlark module of regular expression functions.
// The module defines the following functions:
//
//	compile(pattern) - Compiles the given pattern and returns a value of type
//	                   regexp.regexp. It is an error if the pattern is not a
//	                   valid regular expression.
//
//	quote(s) - Returns a string that quotes all regular expression metacharacters
//	           in s; the returned pattern matches the literal text s. For more
//	           details, refer to https://pkg.go.dev/regexp#QuoteMeta.
//
// Patterns use RE2 syntax, which is described at
// https://github.com/google/re2/wiki/Syntax, except for \C, which matches a
// single byte and is rejected by compile because it cannot be implemented by
// Starlark implementations whose strings are not byte-oriented.
//
// Matching is linear in the length of the subject string, but its cost is not
// counted against the thread's execution step budget (see
// Thread.SetMaxExecutionSteps), so applications that execute untrusted
// programs should bound their resources by other means; see the "Bounding
// time, memory, and other resources" section of the README.
var Module = &starlarkstruct.Module{
	Name: "regexp",
	Members: starlark.StringDict{
		"compile": starlark.NewBuiltin("compile", compile),
		"quote":   starlark.NewBuiltin("quote", quote),
	},
}

func compile(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var pattern string
	if err := starlark.UnpackPositionalArgs("compile", args, kwargs, 1, &pattern); err != nil {
		return nil, err
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return &Regexp{re: re}, nil
}

func quote(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var s string
	if err := starlark.UnpackPositionalArgs("quote", args, kwargs, 1, &s); err != nil {
		return nil, err
	}
	return starlark.String(regexp.QuoteMeta(s)), nil
}

// A Regexp is a Starlark value holding a compiled regular expression.
type Regexp struct{ re *regexp.Regexp }

var (
	_ starlark.Value    = (*Regexp)(nil)
	_ starlark.HasAttrs = (*Regexp)(nil)
)

// String returns a Starlark expression that evaluates to an equivalent regexp,
// such as regexp.compile("b(an)*a").
func (r *Regexp) String() string {
	return fmt.Sprintf("regexp.compile(%s)", strconv.Quote(r.re.String()))
}

// Type returns "regexp.regexp".
func (r *Regexp) Type() string { return "regexp.regexp" }

// Freeze renders the regexp immutable. required by starlark.Value interface
// because a compiled regexp is already immutable this is a no-op.
func (r *Regexp) Freeze() {}

// Truth reports whether the regexp is true. A compiled regexp is always true.
func (r *Regexp) Truth() starlark.Bool { return starlark.True }

// Hash returns a function of the pattern such that Equals(x, y) => Hash(x) ==
// Hash(y). required by starlark.Value interface.
func (r *Regexp) Hash() (uint32, error) { return starlark.String(r.re.String()).Hash() }

// Attr gets a value for a string attribute, implementing dot expression support
// in starlark. required by starlark.HasAttrs interface.
func (r *Regexp) Attr(name string) (starlark.Value, error) {
	if name == "pattern" {
		return starlark.String(r.re.String()), nil
	}
	return builtinAttr(r, name, regexpMethods)
}

// AttrNames lists available dot expression strings. required by
// starlark.HasAttrs interface.
func (r *Regexp) AttrNames() []string {
	return append(builtinAttrNames(regexpMethods), "pattern")
}

var regexpMethods = map[string]*starlark.Builtin{
	"find":                starlark.NewBuiltin("find", regexpFind),
	"find_all":            starlark.NewBuiltin("find_all", regexpFindAll),
	"find_all_submatches": starlark.NewBuiltin("find_all_submatches", regexpFindAllSubmatches),
	"find_submatches":     starlark.NewBuiltin("find_submatches", regexpFindSubmatches),
	"matches":             starlark.NewBuiltin("matches", regexpMatches),
	"replace_all":         starlark.NewBuiltin("replace_all", regexpReplaceAll),
	"split":               starlark.NewBuiltin("split", regexpSplit),
}

// matches(src) reports whether the regexp matches src anywhere.
func regexpMatches(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var src string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &src); err != nil {
		return nil, err
	}
	recv := b.Receiver().(*Regexp)
	return starlark.Bool(recv.re.MatchString(src)), nil
}

// find(src) returns the text of the leftmost match, or None if there is none.
// The result is distinguishable from an empty match, which yields "".
func regexpFind(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var src string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &src); err != nil {
		return nil, err
	}
	recv := b.Receiver().(*Regexp)
	loc := recv.re.FindStringIndex(src)
	if loc == nil {
		return starlark.None, nil
	}
	return starlark.String(src[loc[0]:loc[1]]), nil
}

// find_all(src, max) returns the text of successive non-overlapping matches.
// At most max matches are returned; a negative max means all of them.
func regexpFindAll(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		src string
		max = -1
	)
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &src, &max); err != nil {
		return nil, err
	}
	recv := b.Receiver().(*Regexp)
	return stringList(recv.re.FindAllString(src, max)), nil
}

// find_submatches(src) returns the text of the leftmost match and of each of
// its subpattern matches, or an empty list if there is no match. A subpattern
// that did not participate in the match yields "".
func regexpFindSubmatches(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var src string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &src); err != nil {
		return nil, err
	}
	recv := b.Receiver().(*Regexp)
	return stringList(recv.re.FindStringSubmatch(src)), nil
}

// find_all_submatches(src, max) returns, for each of at most max successive
// non-overlapping matches, the list that find_submatches would return.
func regexpFindAllSubmatches(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		src string
		max = -1
	)
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &src, &max); err != nil {
		return nil, err
	}
	recv := b.Receiver().(*Regexp)
	matches := recv.re.FindAllStringSubmatch(src, max)
	elems := make([]starlark.Value, 0, len(matches))
	for _, match := range matches {
		elems = append(elems, stringList(match))
	}
	return starlark.NewList(elems), nil
}

// replace_all(src, repl) returns a copy of src in which every match has been
// replaced. If repl is a string, $1 and ${name} within it expand to the
// corresponding submatch, and $$ to a literal dollar sign; if repl is callable,
// it is called with the text of each match and must return a string.
func regexpReplaceAll(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		src  string
		repl starlark.Value
	)
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 2, &src, &repl); err != nil {
		return nil, err
	}
	recv := b.Receiver().(*Regexp)

	switch repl := repl.(type) {
	case starlark.String:
		return starlark.String(recv.re.ReplaceAllString(src, string(repl))), nil

	case starlark.Callable:
		// ReplaceAllStringFunc cannot report an error, so remember the first
		// one and stop calling repl once it has occurred.
		var firstErr error
		res := recv.re.ReplaceAllStringFunc(src, func(match string) string {
			if firstErr != nil {
				return ""
			}
			v, err := starlark.Call(thread, repl, starlark.Tuple{starlark.String(match)}, nil)
			if err != nil {
				firstErr = err
				return ""
			}
			s, ok := starlark.AsString(v)
			if !ok {
				firstErr = fmt.Errorf("%s: replacement function returned %s, want string", b.Name(), v.Type())
				return ""
			}
			return s
		})
		if firstErr != nil {
			return nil, firstErr
		}
		return starlark.String(res), nil
	}

	return nil, fmt.Errorf("%s: for parameter repl: got %s, want string or callable", b.Name(), repl.Type())
}

// split(src, max) returns the strings between the matches of the regexp,
// yielding at most max substrings; a negative max means all of them.
func regexpSplit(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		src string
		max = -1
	)
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &src, &max); err != nil {
		return nil, err
	}
	recv := b.Receiver().(*Regexp)
	return stringList(recv.re.Split(src, max)), nil
}

// stringList converts a slice of Go strings to a Starlark list of strings.
func stringList(strs []string) *starlark.List {
	elems := make([]starlark.Value, 0, len(strs))
	for _, s := range strs {
		elems = append(elems, starlark.String(s))
	}
	return starlark.NewList(elems)
}

func builtinAttr(recv starlark.Value, name string, methods map[string]*starlark.Builtin) (starlark.Value, error) {
	b := methods[name]
	if b == nil {
		return nil, nil // no such method
	}
	return b.BindReceiver(recv), nil
}

func builtinAttrNames(methods map[string]*starlark.Builtin) []string {
	names := make([]string, 0, len(methods))
	for name := range methods {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
