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
	"strings"
	"sync"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"go.starlark.net/syntax"
)

// Module regexp is a Starlark module of regular expression functions.
// The module defines the following functions:
//
//	compile(pattern) - Compiles the given pattern and returns a value of type
//	                   regexp.regexp. It is an error if the pattern is not a
//	                   valid regular expression.
//
//	escape(lit) - Returns a string that escapes all regular expression
//	              metacharacters in lit; the returned pattern matches the
//	              literal text lit. For more details, refer to
//	              https://pkg.go.dev/regexp#QuoteMeta.
//
// A compiled pattern, of type regexp.regexp, has these fields and methods:
//
//	pattern - The source text of the regular expression.
//
//	groupnames - A list holding the name of each capture group, or None for
//	             each unnamed group.
//
//	find(s) - Returns a regexp.match for the leftmost match in s, or None if
//	          there is no match.
//
//	findall(s) - Returns a regexp.matches, an iterable of the regexp.match of
//	             each successive non-overlapping match in s. It may be
//	             iterated more than once, and is true if there is at least one
//	             match.
//
//	matches(s) - Reports whether the pattern matches anywhere in s.
//
//	replace(s, repl) - Returns a copy of s in which each match is replaced.
//	                   If repl is a string, $1 and ${name} within it expand to
//	                   the corresponding submatch; if repl is callable, it is
//	                   called with the regexp.match of each match and must
//	                   return a string. An empty match immediately following a
//	                   previous match is skipped.
//
//	split(s, max) - Returns the strings between the matches in s, at most max
//	                of them; a negative max, the default, means all of them.
//
// A match, of type regexp.match, has these fields and methods:
//
//	text - The matching substring.
//
//	start, end - The offsets of the match within s, so that s[start:end] is
//	             equal to text.
//
//	submatches - An indexable sequence holding the regexp.submatch of each
//	             capture group, counted from zero, or None for each group that
//	             did not take part in the match.
//
//	submatch(x) - Returns submatches[x] if x is an int, or the submatch of the
//	              group named x if x is a string. It is an error if there is no
//	              such group.
//
// A submatch, of type regexp.submatch, has the text, start and end fields of a
// match, but no submatch method or submatches field.
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
		"escape":  starlark.NewBuiltin("escape", escape),
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

func escape(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var lit string
	if err := starlark.UnpackPositionalArgs("escape", args, kwargs, 1, &lit); err != nil {
		return nil, err
	}
	return starlark.String(regexp.QuoteMeta(lit)), nil
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
	switch name {
	case "groupnames":
		// SubexpNames reports the name of the whole expression at index
		// zero, and the name of each group, which is empty if unnamed,
		// thereafter.
		names := r.re.SubexpNames()
		elems := make([]starlark.Value, 0, len(names)-1)
		for _, name := range names[1:] {
			if name == "" {
				elems = append(elems, starlark.None)
			} else {
				elems = append(elems, starlark.String(name))
			}
		}
		return starlark.NewList(elems), nil
	case "pattern":
		return starlark.String(r.re.String()), nil
	}
	return builtinAttr(r, name, regexpMethods)
}

// AttrNames lists available dot expression strings. required by
// starlark.HasAttrs interface.
func (r *Regexp) AttrNames() []string {
	return attrNames(regexpMethods, "groupnames", "pattern")
}

var regexpMethods = map[string]*starlark.Builtin{
	"find":    starlark.NewBuiltin("find", regexpFind),
	"findall": starlark.NewBuiltin("findall", regexpFindall),
	"matches": starlark.NewBuiltin("matches", regexpMatches),
	"replace": starlark.NewBuiltin("replace", regexpReplace),
	"split":   starlark.NewBuiltin("split", regexpSplit),
}

// matches(s) reports whether the regexp matches s anywhere.
func regexpMatches(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var s string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &s); err != nil {
		return nil, err
	}
	recv := b.Receiver().(*Regexp)
	return starlark.Bool(recv.re.MatchString(s)), nil
}

// find(s) returns the leftmost match, or None if there is none.
func regexpFind(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var s string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &s); err != nil {
		return nil, err
	}
	recv := b.Receiver().(*Regexp)
	loc := recv.re.FindStringSubmatchIndex(s)
	if loc == nil {
		return starlark.None, nil
	}
	return &Match{re: recv.re, s: s, loc: loc}, nil
}

// findall(s) returns an iterable of the successive non-overlapping matches.
func regexpFindall(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var s string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &s); err != nil {
		return nil, err
	}
	recv := b.Receiver().(*Regexp)
	return &Matches{re: recv.re, s: s}, nil
}

// replace(s, repl) returns a copy of s in which each match has been replaced.
// If repl is a string, $1 and ${name} within it expand to the corresponding
// submatch, and $$ to a literal dollar sign; if repl is callable, it is called
// with the match and must return a string.
func regexpReplace(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		s    string
		repl starlark.Value
	)
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 2, &s, &repl); err != nil {
		return nil, err
	}
	recv := b.Receiver().(*Regexp)

	switch repl := repl.(type) {
	case starlark.String:
		return starlark.String(recv.re.ReplaceAllString(s, string(repl))), nil

	case starlark.Callable:
		var out strings.Builder
		prev := 0
		for _, loc := range recv.re.FindAllStringSubmatchIndex(s, -1) {
			v, err := starlark.Call(thread, repl, starlark.Tuple{&Match{re: recv.re, s: s, loc: loc}}, nil)
			if err != nil {
				return nil, err
			}
			text, ok := starlark.AsString(v)
			if !ok {
				return nil, fmt.Errorf("%s: replacement function returned %s, want string", b.Name(), v.Type())
			}
			out.WriteString(s[prev:loc[0]])
			out.WriteString(text)
			prev = loc[1]
		}
		out.WriteString(s[prev:])
		return starlark.String(out.String()), nil
	}

	return nil, fmt.Errorf("%s: for parameter repl: got %s, want string or callable", b.Name(), repl.Type())
}

// split(s, max) returns the strings between the matches of the regexp,
// yielding at most max substrings; a negative max means all of them.
func regexpSplit(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		s   string
		max = -1
	)
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &s, &max); err != nil {
		return nil, err
	}
	recv := b.Receiver().(*Regexp)
	elems := make([]starlark.Value, 0)
	for _, part := range recv.re.Split(s, max) {
		elems = append(elems, starlark.String(part))
	}
	return starlark.NewList(elems), nil
}

// Matches is the iterable sequence of matches returned by findall. It may be
// iterated more than once.
type Matches struct {
	re *regexp.Regexp
	s  string

	once sync.Once
	locs [][]int
}

var (
	_ starlark.Value    = (*Matches)(nil)
	_ starlark.Iterable = (*Matches)(nil)
)

// all returns the location of each match, computing them on first use.
func (m *Matches) all() [][]int {
	// TODO: enumerate the matches lazily, using the regexp
	// package's iterator methods, once the module's minimum Go version
	// provides them.
	m.once.Do(func() { m.locs = m.re.FindAllStringSubmatchIndex(m.s, -1) })
	return m.locs
}

func (m *Matches) String() string {
	return fmt.Sprintf("<regexp.matches %s>", strconv.Quote(m.re.String()))
}

// Type returns "regexp.matches".
func (m *Matches) Type() string { return "regexp.matches" }

// Freeze renders the matches immutable. required by starlark.Value interface
// because the sequence of matches is determined by the regexp and the subject
// string, both of which are immutable, this is a no-op.
func (m *Matches) Freeze() {}

// Truth reports whether there is at least one match.
func (m *Matches) Truth() starlark.Bool { return starlark.Bool(m.re.MatchString(m.s)) }

// Hash reports an error: like other iterable containers, matches are not
// hashable. required by starlark.Value interface.
func (m *Matches) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable type: %s", m.Type()) }

// Iterate returns an iterator over the matches. required by starlark.Iterable
// interface.
func (m *Matches) Iterate() starlark.Iterator {
	return &matchesIterator{m: m, locs: m.all()}
}

type matchesIterator struct {
	m    *Matches
	locs [][]int
	i    int
}

func (it *matchesIterator) Next(p *starlark.Value) bool {
	if it.i >= len(it.locs) {
		return false
	}
	*p = &Match{re: it.m.re, s: it.m.s, loc: it.locs[it.i]}
	it.i++
	return true
}

func (it *matchesIterator) Done() {}

// A Match is a Starlark value holding one match of a regexp, and the submatch
// of each of its capture groups.
type Match struct {
	re  *regexp.Regexp
	s   string
	loc []int // pairs of offsets, as returned by FindStringSubmatchIndex
}

var (
	_ starlark.Value      = (*Match)(nil)
	_ starlark.HasAttrs   = (*Match)(nil)
	_ starlark.Comparable = (*Match)(nil)
)

func (m *Match) String() string { return spanString(m.Type(), m.text(), m.loc[0], m.loc[1]) }

// Type returns "regexp.match".
func (m *Match) Type() string { return "regexp.match" }

// Freeze renders the match immutable. required by starlark.Value interface
// because a match is already immutable this is a no-op.
func (m *Match) Freeze() {}

// Truth reports whether the match is true. A match is always true, even if its
// text is empty, so that the result of find may be tested for success.
func (m *Match) Truth() starlark.Bool { return starlark.True }

// Hash returns a function of the text and offsets of the match. required by
// starlark.Value interface.
func (m *Match) Hash() (uint32, error) { return spanHash(m.text(), m.loc[0], m.loc[1]) }

// CompareSameType compares two matches, which are equal if they have the same
// offsets and the same text, for the match itself and for each of its
// submatches. Matches are not ordered. required by starlark.Comparable
// interface.
func (m *Match) CompareSameType(op syntax.Token, y starlark.Value, depth int) (bool, error) {
	switch op {
	case syntax.EQL:
		return m.equal(y.(*Match)), nil
	case syntax.NEQ:
		return !m.equal(y.(*Match)), nil
	}
	return false, fmt.Errorf("%s %s %s not implemented", m.Type(), op, y.Type())
}

func (m *Match) equal(y *Match) bool {
	if len(m.loc) != len(y.loc) {
		return false
	}
	for i, x := range m.loc {
		if x != y.loc[i] {
			return false
		}
	}
	// The spans are equal; compare the text they denote.
	for i := 0; i < len(m.loc); i += 2 {
		if m.loc[i] < 0 {
			continue // group did not take part in either match
		}
		if m.s[m.loc[i]:m.loc[i+1]] != y.s[y.loc[i]:y.loc[i+1]] {
			return false
		}
	}
	return true
}

// Attr gets a value for a string attribute, implementing dot expression support
// in starlark. required by starlark.HasAttrs interface.
func (m *Match) Attr(name string) (starlark.Value, error) {
	switch name {
	case "text":
		return starlark.String(m.text()), nil
	case "start":
		return starlark.MakeInt(m.loc[0]), nil
	case "end":
		return starlark.MakeInt(m.loc[1]), nil
	case "submatches":
		return &Submatches{m: m}, nil
	}
	return builtinAttr(m, name, matchMethods)
}

// AttrNames lists available dot expression strings. required by
// starlark.HasAttrs interface.
func (m *Match) AttrNames() []string {
	return attrNames(matchMethods, "end", "start", "submatches", "text")
}

func (m *Match) text() string { return m.s[m.loc[0]:m.loc[1]] }

// numGroups reports the number of capture groups of the regexp.
func (m *Match) numGroups() int { return len(m.loc)/2 - 1 }

// submatch returns the submatch of the i'th capture group, counted from zero,
// or None if the group did not take part in the match.
func (m *Match) submatch(i int) starlark.Value {
	start, end := m.loc[2*i+2], m.loc[2*i+3]
	if start < 0 {
		return starlark.None
	}
	return &Submatch{s: m.s, start: start, end: end}
}

var matchMethods = map[string]*starlark.Builtin{
	"submatch": starlark.NewBuiltin("submatch", matchSubmatch),
}

// submatch(x) returns the submatch of the group named x, if x is a string, or
// of the x'th group counted from zero, if x is an int. It is an error if there
// is no such group; the result is None if the group did not take part in the
// match.
func matchSubmatch(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var x starlark.Value
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &x); err != nil {
		return nil, err
	}
	recv := b.Receiver().(*Match)

	switch x := x.(type) {
	case starlark.String:
		// SubexpNames reports the name of the whole expression at index
		// zero, so the i'th group, counted from zero, is at index i+1.
		for i, name := range recv.re.SubexpNames()[1:] {
			if name == string(x) && name != "" {
				return recv.submatch(i), nil
			}
		}
		return nil, fmt.Errorf("%s: regexp has no group named %s", b.Name(), strconv.Quote(string(x)))

	case starlark.Int:
		i, err := starlark.AsInt32(x)
		if err != nil {
			return nil, fmt.Errorf("%s: %s", b.Name(), err)
		}
		n := recv.numGroups()
		if i < 0 {
			i += n // negative index counts from the end, as in s[-1]
		}
		if i < 0 || i >= n {
			return nil, fmt.Errorf("%s: group index %s out of range: regexp has %d groups", b.Name(), x, n)
		}
		return recv.submatch(i), nil
	}

	return nil, fmt.Errorf("%s: for parameter 1: got %s, want string or int", b.Name(), x.Type())
}

// Submatches is the indexable sequence of the submatches of a match, one for
// each capture group of its regexp, counted from zero. An element is None if
// its group did not take part in the match.
type Submatches struct{ m *Match }

var (
	_ starlark.Value      = (*Submatches)(nil)
	_ starlark.Indexable  = (*Submatches)(nil)
	_ starlark.Sequence   = (*Submatches)(nil)
	_ starlark.Comparable = (*Submatches)(nil)
)

func (s *Submatches) String() string {
	var out strings.Builder
	out.WriteByte('[')
	for i := 0; i < s.Len(); i++ {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(s.Index(i).String())
	}
	out.WriteByte(']')
	return out.String()
}

// Type returns "regexp.submatches".
func (s *Submatches) Type() string { return "regexp.submatches" }

// Freeze renders the submatches immutable. required by starlark.Value
// interface because the submatches of a match are already immutable this is a
// no-op.
func (s *Submatches) Freeze() {}

// Truth reports whether the regexp has at least one capture group.
func (s *Submatches) Truth() starlark.Bool { return s.Len() > 0 }

// Hash reports an error: like other indexable containers, submatches are not
// hashable. required by starlark.Value interface.
func (s *Submatches) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable type: %s", s.Type())
}

// CompareSameType compares two sequences of submatches elementwise, as lists
// are compared. They are not ordered. required by starlark.Comparable
// interface.
func (s *Submatches) CompareSameType(op syntax.Token, y starlark.Value, depth int) (bool, error) {
	switch op {
	case syntax.EQL:
		return s.equal(y.(*Submatches), depth)
	case syntax.NEQ:
		eq, err := s.equal(y.(*Submatches), depth)
		return !eq, err
	}
	return false, fmt.Errorf("%s %s %s not implemented", s.Type(), op, y.Type())
}

func (s *Submatches) equal(y *Submatches, depth int) (bool, error) {
	if s.Len() != y.Len() {
		return false, nil
	}
	for i := 0; i < s.Len(); i++ {
		eq, err := starlark.EqualDepth(s.Index(i), y.Index(i), depth-1)
		if err != nil || !eq {
			return eq, err
		}
	}
	return true, nil
}

// Len returns the number of capture groups of the regexp. required by
// starlark.Indexable interface.
func (s *Submatches) Len() int { return s.m.numGroups() }

// Index returns the submatch of the i'th capture group, counted from zero, or
// None if the group did not take part in the match. required by
// starlark.Indexable interface.
func (s *Submatches) Index(i int) starlark.Value { return s.m.submatch(i) }

// Iterate returns an iterator over the submatches. required by
// starlark.Iterable interface.
func (s *Submatches) Iterate() starlark.Iterator { return &submatchesIterator{s: s} }

type submatchesIterator struct {
	s *Submatches
	i int
}

func (it *submatchesIterator) Next(p *starlark.Value) bool {
	if it.i >= it.s.Len() {
		return false
	}
	*p = it.s.Index(it.i)
	it.i++
	return true
}

func (it *submatchesIterator) Done() {}

// A Submatch is a Starlark value holding the match of one capture group.
type Submatch struct {
	s          string
	start, end int
}

var (
	_ starlark.Value      = (*Submatch)(nil)
	_ starlark.HasAttrs   = (*Submatch)(nil)
	_ starlark.Comparable = (*Submatch)(nil)
)

func (s *Submatch) String() string { return spanString(s.Type(), s.text(), s.start, s.end) }

// Type returns "regexp.submatch".
func (s *Submatch) Type() string { return "regexp.submatch" }

// Freeze renders the submatch immutable. required by starlark.Value interface
// because a submatch is already immutable this is a no-op.
func (s *Submatch) Freeze() {}

// Truth reports whether the submatch is true. A submatch is always true, even
// if its text is empty; a group that did not take part in the match is
// reported as None, not as a false submatch.
func (s *Submatch) Truth() starlark.Bool { return starlark.True }

// Hash returns a function of the text and offsets of the submatch. required by
// starlark.Value interface.
func (s *Submatch) Hash() (uint32, error) { return spanHash(s.text(), s.start, s.end) }

// CompareSameType compares two submatches, which are equal if they have the
// same offsets and the same text. Submatches are not ordered. required by
// starlark.Comparable interface.
func (s *Submatch) CompareSameType(op syntax.Token, y starlark.Value, depth int) (bool, error) {
	switch op {
	case syntax.EQL:
		return s.equal(y.(*Submatch)), nil
	case syntax.NEQ:
		return !s.equal(y.(*Submatch)), nil
	}
	return false, fmt.Errorf("%s %s %s not implemented", s.Type(), op, y.Type())
}

func (s *Submatch) equal(y *Submatch) bool {
	return s.start == y.start && s.end == y.end && s.text() == y.text()
}

// Attr gets a value for a string attribute, implementing dot expression support
// in starlark. required by starlark.HasAttrs interface.
func (s *Submatch) Attr(name string) (starlark.Value, error) {
	switch name {
	case "text":
		return starlark.String(s.text()), nil
	case "start":
		return starlark.MakeInt(s.start), nil
	case "end":
		return starlark.MakeInt(s.end), nil
	}
	return nil, nil // no such field
}

// AttrNames lists available dot expression strings. required by
// starlark.HasAttrs interface.
func (s *Submatch) AttrNames() []string { return []string{"end", "start", "text"} }

func (s *Submatch) text() string { return s.s[s.start:s.end] }

// spanString returns the string form of a match or submatch.
func spanString(typ, text string, start, end int) string {
	return fmt.Sprintf("<%s %s at %d:%d>", typ, strconv.Quote(text), start, end)
}

// spanHash returns a hash of the text and offsets of a match or submatch.
func spanHash(text string, start, end int) (uint32, error) {
	h, err := starlark.String(text).Hash()
	if err != nil {
		return 0, err
	}
	return h ^ uint32(start)*0x9e3779b9 ^ uint32(end)*0x85ebca6b, nil
}

func builtinAttr(recv starlark.Value, name string, methods map[string]*starlark.Builtin) (starlark.Value, error) {
	b := methods[name]
	if b == nil {
		return nil, nil // no such method
	}
	return b.BindReceiver(recv), nil
}

// attrNames returns the sorted names of the given methods and fields.
func attrNames(methods map[string]*starlark.Builtin, fields ...string) []string {
	names := make([]string, 0, len(methods)+len(fields))
	for name := range methods {
		names = append(names, name)
	}
	names = append(names, fields...)
	sort.Strings(names)
	return names
}
