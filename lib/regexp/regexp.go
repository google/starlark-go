// Copyright 2021 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package regexp provides regular expression related functions.
package regexp // import "go.starlark.net/lib/regexp"

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// Module regexp is a Starlark module of regular expression related functions.
// The module defines the following functions:
//
//     compile(pattern) - Compile a regular expression pattern into a regular expression object, which
//                        can be used for matching using its matches(), find(), find_all and other methods.
//
//     re.find(src) - Returns a string holding the text of the leftmost match in the given string of the regular expression re.
//                    If there is no match, the return value is an empty string, but it will also be empty if the
//                    regular expression successfully matches an empty string.
//
//     re.find_all(src, max) - Returns a tuple of all successive matches of the regular expression re. An empty tuple indicates
//                             no match. If max > 0, at most max strings will be returned. If max == 0,
//                             an empty tuple will be returned. If max < 0, all strings will be returned.
//                             The parameter max is optional, by default no limit will be applied.
//
//     re.find_submatches(src) - Returns a tuple of strings holding the text of the leftmost match of the regular expression re
//                               in the given string and the matches, if any, of its subexpressions. An empty tuple indicates
//                               no match.
//
//     re.matches(src) - Indicates whether the given string contains any match of the regular expression re.
//
//     re.replace_all(src , repl) - Returns a copy of the given string, replacing matches of the regular expression re
//                                  with the replacement string repl. Inside repl, $ signs are interpreted as in Expand,
//                                  so for instance $1 represents the text of the first submatch.
//
//     re.replace_all(src , replFunc) - Returns a copy of the given string in which all matches of the regular expression re
//                                      have been replaced by the return value of the replacement function applied to the
//                                      matched substring. The replacement returned by replacement function is substituted directly,
//                                      without using Expand.
//
//     re.split(src, max) - Returns a tuple of strings between all the matches of the regular expression re.
//                          If max > 0, at most max strings will be returned knowing that the last string will
//                          be the unsplit remainder. If max == 0, an empty tuple will be returned. If max < 0,
//                          all strings will be returned. The parameter max is optional, by default no limit
//                          will be applied.
//
var Module = &starlarkstruct.Module{
	Name: "regexp",
	Members: starlark.StringDict{
		"compile": starlark.NewBuiltin("compile", compile),
	},
}

func compile(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		pattern string
	)

	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &pattern); err != nil {
		return starlark.None, err
	}

	if strings.Contains(pattern, `\C`) {
		return nil, fmt.Errorf(`The byte-oriented pattern \C is not supported`)
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return &Regex{re: re}, nil
}

func toTuple(strs []string) starlark.Tuple {
	var vals starlark.Tuple
	for _, s := range strs {
		vals = append(vals, starlark.String(s))
	}
	return vals
}

// hashString computes the FNV hash of s.
func hashString(s string) uint32 {
	var h uint32
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}

// Regex is a starlark representation of a compiled regular expression
type Regex struct {
	re *regexp.Regexp
}

// String implements the Stringer interface
func (r *Regex) String() string { return r.re.String() }

// Type returns a short string describing the value's type.
func (r *Regex) Type() string { return "regexp" }

// Freeze renders time immutable. required by starlark.Value interface.
// The interface regex presents to the starlark runtime renders it immutable,
// making this a no-op
func (r *Regex) Freeze() {}

// Hash returns a function of x such that Equals(x, y) => Hash(x) == Hash(y)
// required by starlark.Value interface
func (r *Regex) Hash() (uint32, error) { return hashString(r.re.String()), nil }

// Truth returns the truth value of an object required by starlark.Value
// interface. Any non-empty regexp is considered truthy
func (r *Regex) Truth() starlark.Bool { return r.String() != "" }

// Attr gets a value for a string attribute, implementing dot expression support
// in starklark. required by starlark.HasAttrs interface
func (r *Regex) Attr(name string) (starlark.Value, error) {
	return builtinAttr(r, name, regexMethods)
}

var regexMethods = map[string]*starlark.Builtin{
	"find":            starlark.NewBuiltin("find", find),
	"find_all":        starlark.NewBuiltin("find_all", findAll),
	"find_submatches": starlark.NewBuiltin("find_submatches", findSubmatches),
	"matches":         starlark.NewBuiltin("matches", matches),
	"replace_all":     starlark.NewBuiltin("replace_all", replaceAll),
	"split":           starlark.NewBuiltin("split", split),
}

// AttrNames lists available dot expression strings for time. required by
// starlark.HasAttrs interface
func (r *Regex) AttrNames() []string { return builtinAttrNames(regexMethods) }

func matches(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		src string
	)

	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &src); err != nil {
		return starlark.None, err
	}

	re := b.Receiver().(*Regex).re
	return starlark.Bool(re.MatchString(src)), nil
}

func find(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		src string
	)

	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &src); err != nil {
		return starlark.None, err
	}

	re := b.Receiver().(*Regex).re
	return starlark.String(re.FindString(src)), nil
}

func findAll(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		src string
		max int = -1
	)

	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &src, &max); err != nil {
		return starlark.None, err
	}

	re := b.Receiver().(*Regex).re
	return toTuple(re.FindAllString(src, max)), nil
}

func findSubmatches(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		src string
	)

	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &src); err != nil {
		return starlark.None, err
	}

	re := b.Receiver().(*Regex).re
	return toTuple(re.FindStringSubmatch(src)), nil
}

func replaceAll(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		src  string
		repl starlark.Value
	)

	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 2, &src, &repl); err != nil {
		return starlark.None, err
	}

	re := b.Receiver().(*Regex).re
	switch x := repl.(type) {
	case starlark.Callable:
		fn := func(matched string) string {
			res, err := starlark.Call(thread, repl, starlark.Tuple{starlark.String(matched)}, nil)
			if err != nil {
				log.Printf("An error occured while executing the function: %s", err.Error())
				return matched
			}
			resp, ok := res.(starlark.String)
			if !ok {
				log.Printf("A string is expected as return type of the function but was %s", res.Type())
				return matched
			}
			return string(resp)
		}
		return starlark.String(re.ReplaceAllStringFunc(src, fn)), nil
	case starlark.String:
		return starlark.String(re.ReplaceAllString(src, string(x))), nil
	}
	return nil, fmt.Errorf("got %s, want a function or string", repl.Type())
}

func split(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		src string
		max int = -1
	)

	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &src, &max); err != nil {
		return starlark.None, err
	}

	re := b.Receiver().(*Regex).re
	return toTuple(re.Split(src, max)), nil
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
