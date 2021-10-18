// Copyright 2021 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package regexp provides functions related to regular expressions.
package regexp // import "go.starlark.net/lib/regexp"

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// Module regexp is a Starlark module of functions related to regular expressions.
// The module defines the following functions:
//
//     compile(pattern) - Compiles a pattern in RE2 syntax (https://github.com/google/re2/wiki/Syntax) to a value of type 'regexp'.
//                        Each call to compile returns a distinct regexp value.
//                        A regexp value can be used for matching using its matches, find, find_all and other methods.
//
//     regexp.find(src) - Returns a string holding the first match in the given string for the regular expression regexp.
//                        Knowing that a first match is a match that starts from the lowest postion in the given string.
//                        In case several patterns in the regular expression match at the same position in the given string, the match
//                        of the first matching pattern starting from the left is returned.
//                        The result is None if there is no match or if the pattern successfully matches an empty string.
//
//     regexp.find_index(src) - Returns a new, mutable list holding the start and end index of the first match in the given string for the regular expression regexp.
//                              Knowing that a first match is a match that starts from the lowest postion in the given string.
//                              In case several patterns in the regular expression match at the same position in the given string, the match
//                              of the first matching pattern starting from the left is returned.
//                              The result is None if there is no match or if the pattern successfully matches an empty string.
//
//     regexp.find_all(src) - Returns a new, mutable list of all successive matches for the regular expression regexp.
//                            Knowing that successive matches are non-overlapping matches sorted according to their position in the given string.
//                            In case several patterns in the regular expression match at the same position in the given string, the match
//                            of the first matching pattern starting from the left is returned.
//                            An empty list indicates no match.
//
//     regexp.find_all_index(src) - Returns a new, mutable list of mutable lists holding the start and end index of each successive match for the regular expression regexp.
//                                  Knowing that successive matches are non-overlapping matches sorted according to their position in the given string.
//                                  In case several patterns in the regular expression match at the same position in the given string, the match
//                                  of the first matching pattern starting from the left is returned.
//                                  An empty list indicates no match.
//
//     regexp.find_submatch(src) - Returns a new, mutable list of strings holding the first match in the given string for the regular expression regexp
//                                 and the matches, if any, of its subexpressions also known as groups. None indicates no match.
//
//     regexp.find_submatch_index(src) - Returns a new, mutable list holding the start and end index of the first match in the given string for the regular expression regexp
//                                       and the start and end index of the matches, if any, of its subexpressions also known as groups. None indicates no match.
//
//     regexp.find_all_submatch(src) - Returns a new, mutable list of mutable lists holding each successive match in the given string for the regular expression regexp
//                                     and the matches, if any, of its subexpressions also known as groups.
//                                     Knowing that successive matches are non-overlapping matches sorted according to their position in the given string.
//                                     In case several patterns in the regular expression match at the same position in the given string, the match
//                                     of the first matching pattern starting from the left is returned.
//                                     An empty list indicates no match.
//
//     regexp.find_all_submatch_index(src) - Returns a new, mutable list of mutable lists holding the start and end index of each successive match in the given string for the regular expression regexp
//                                           and the start and end index of the matches, if any, of its subexpressions also known as groups.
//                                           Knowing that successive matches are non-overlapping matches sorted according to their position in the given string.
//                                           In case several patterns in the regular expression match at the same position in the given string, the match
//                                           of the first matching pattern starting from the left is returned.
//                                           An empty list indicates no match.
//
//     regexp.matches(src) - Reports whether the given string contains any match of the regular expression regexp.
//
//     regexp.replace_all(src, repl) - Returns a copy of the given string, replacing all successive matches of the regular expression regexp
//                                     with the replacement string repl.
//                                     Knowing that successive matches are non-overlapping matches sorted according to their position in the given string.
//                                     In case several patterns in the regular expression match at the same position in the given string, the match
//                                     of the first matching pattern starting from the left is returned.
//                                     Inside repl, $ signs can be used to insert text matching
//                                     the corresponding parenthesized group from the pattern.
//                                     $0 in repl refers to the entire matching text.
//
//     regexp.replace_all(src, replFunc) - Returns a copy of the given string in which all successive matches of the regular expression regexp
//                                         have been replaced by the result of the replacement function.
//                                         Knowing that successive matches are non-overlapping matches sorted according to their position in the given string.
//                                         In case several patterns in the regular expression match at the same position in the given string, the match
//                                         of the first matching pattern starting from the left is returned.
//
//     regexp.replace_all_literal(src, repl) - Returns a copy of the given string, replacing all successive matches of the regular expression regexp
//                                             with the replacement string repl.
//                                             Knowing that successive matches are non-overlapping matches sorted according to their position in the given string.
//                                             In case several patterns in the regular expression match at the same position in the given string, the match
//                                             of the first matching pattern starting from the left is returned.
//                                             The replacement repl is substituted directly.
//
//     regexp.split(src, max=-1) - Returns a new, mutable list of strings between all the matches of the regular expression regexp.
//                              If max > 0, at most max strings are returned knowing that the last string is
//                              the unsplit remainder. If max <= 0, all strings are returned.
//                              The parameter max is optional: by default no limit is applied.
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
		return nil, err
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return &Regexp{re: re}, nil
}

func toList(slice interface{}) *starlark.List {
	values := reflect.ValueOf(slice)
	elems := make([]starlark.Value, values.Len())
	for i := 0; i < values.Len(); i++ {
		switch v := values.Index(i).Interface().(type) {
		case string:
			elems[i] = starlark.String(v)
		case int:
			elems[i] = starlark.MakeInt(v)
		case []int, []string:
			elems[i] = toList(v)
		}
	}
	return starlark.NewList(elems)
}

// A Regexp represents a compiled RE2 regular expression.
type Regexp struct {
	re *regexp.Regexp
}

// String implements the Stringer interface.
func (r *Regexp) String() string { return r.re.String() }

// Type returns a short string describing the value's type.
func (r *Regexp) Type() string { return "regexp" }

// Freeze renders r immutable. Required by starlark.Value interface.
// The interface regex presents to the Starlark runtime renders it immutable,
// making this a no-op.
func (r *Regexp) Freeze() {}

// Hash returns a function of x such that Equals(x, y) => Hash(x) == Hash(y)
// required by starlark.Value interface.
func (r *Regexp) Hash() (uint32, error) { return starlark.String(r.re.String()).Hash() }

// Truth always returns true for a Regexp.
func (r *Regexp) Truth() starlark.Bool { return true }

// Attr gets a value for a string attribute, implementing dot expression support
// in Starklark. required by starlark.HasAttrs interface.
func (r *Regexp) Attr(name string) (starlark.Value, error) {
	return builtinAttr(r, name, regexMethods)
}

// AttrNames lists available dot expression strings for time. Required by
// starlark.HasAttrs interface.
func (r *Regexp) AttrNames() []string { return builtinAttrNames(regexMethods) }

var regexMethods = map[string]*starlark.Builtin{
	"find":                    starlark.NewBuiltin("find", find),
	"find_all":                starlark.NewBuiltin("find_all", findAll),
	"find_submatch":           starlark.NewBuiltin("find_submatch", findSubmatch),
	"find_all_submatch":       starlark.NewBuiltin("find_all_submatch", findAllSubmatch),
	"find_index":              starlark.NewBuiltin("find_index", findIndex),
	"find_all_index":          starlark.NewBuiltin("find_all_index", findAllIndex),
	"find_submatch_index":     starlark.NewBuiltin("find_submatch_index", findSubmatchIndex),
	"find_all_submatch_index": starlark.NewBuiltin("find_all_submatch_index", findAllSubmatchIndex),
	"matches":                 starlark.NewBuiltin("matches", matches),
	"replace_all":             starlark.NewBuiltin("replace_all", replaceAll),
	"replace_all_literal":     starlark.NewBuiltin("replace_all_literal", replaceAllLiteral),
	"split":                   starlark.NewBuiltin("split", split),
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

func matches(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var src string

	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &src); err != nil {
		return nil, err
	}

	re := b.Receiver().(*Regexp).re
	return starlark.Bool(re.MatchString(src)), nil
}

func find(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var src string

	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &src); err != nil {
		return nil, err
	}

	re := b.Receiver().(*Regexp).re
	if result := re.FindString(src); result != "" {
		return starlark.String(result), nil
	}
	return starlark.None, nil
}

func findIndex(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var src string

	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &src); err != nil {
		return nil, err
	}

	re := b.Receiver().(*Regexp).re
	if result := re.FindStringIndex(src); result != nil {
		return toList(result), nil
	}
	return starlark.None, nil
}

func findAll(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		src string
	)

	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &src); err != nil {
		return nil, err
	}

	re := b.Receiver().(*Regexp).re
	return toList(re.FindAllString(src, -1)), nil
}

func findAllIndex(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		src string
	)

	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &src); err != nil {
		return nil, err
	}

	re := b.Receiver().(*Regexp).re
	return toList(re.FindAllStringIndex(src, -1)), nil
}

func findSubmatch(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var src string

	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &src); err != nil {
		return nil, err
	}

	re := b.Receiver().(*Regexp).re
	if result := re.FindStringSubmatch(src); len(result) > 0 {
		return toList(result), nil
	}
	return starlark.None, nil
}

func findSubmatchIndex(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var src string

	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &src); err != nil {
		return nil, err
	}

	re := b.Receiver().(*Regexp).re
	if result := re.FindStringSubmatchIndex(src); len(result) > 0 {
		return toList(result), nil
	}
	return starlark.None, nil
}

func findAllSubmatch(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var src string

	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &src); err != nil {
		return nil, err
	}

	re := b.Receiver().(*Regexp).re
	return toList(re.FindAllStringSubmatch(src, -1)), nil
}

func findAllSubmatchIndex(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var src string

	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &src); err != nil {
		return nil, err
	}

	re := b.Receiver().(*Regexp).re
	return toList(re.FindAllStringSubmatchIndex(src, -1)), nil
}

func replaceAllLiteral(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		src, repl string
	)

	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 2, &src, &repl); err != nil {
		return nil, err
	}

	re := b.Receiver().(*Regexp).re
	return starlark.String(re.ReplaceAllLiteralString(src, repl)), nil
}

func replaceAll(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		src  string
		repl starlark.Value
	)

	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 2, &src, &repl); err != nil {
		return nil, err
	}

	re := b.Receiver().(*Regexp).re
	switch x := repl.(type) {
	case starlark.Callable:
		var fnErr error
		fn := func(matched string) string {
			res, err := starlark.Call(thread, repl, starlark.Tuple{starlark.String(matched)}, nil)
			if err != nil {
				// Save the error to be able to return it to the caller
				fnErr = err
				return ""
			}
			resp, ok := res.(starlark.String)
			if !ok {
				// Save the error to be able to return it to the caller
				fnErr = fmt.Errorf("%s returned %s, want string", x.Name(), res.Type())
				return ""
			}
			return string(resp)
		}
		result := re.ReplaceAllStringFunc(src, fn)
		if fnErr != nil {
			return nil, fnErr
		}
		return starlark.String(result), nil
	case starlark.String:
		return starlark.String(re.ReplaceAllString(src, string(x))), nil
	}
	return nil, fmt.Errorf("got %s, want a string or callable", repl.Type())
}

func split(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		src string
		max int = -1
	)

	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "src", &src, "max?", &max); err != nil {
		return nil, err
	}
	if max == 0 {
		max = -1
	}
	re := b.Receiver().(*Regexp).re
	return toList(re.Split(src, max)), nil
}
