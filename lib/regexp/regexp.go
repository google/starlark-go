package regexp

import (
	"regexp"
	"sort"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// ModuleName defines the expected name for this Module when used
// in starlark's load() function, eg: load('regexp.star', 'regexp')
const ModuleName = "regexp.star"

var Module = &starlarkstruct.Module{
	Name: "regexp",
	Members: starlark.StringDict{
		"compile": starlark.NewBuiltin("compile", compile),
		"search":  starlark.NewBuiltin("search", search),
		"match":   starlark.NewBuiltin("match", match),
		"findall": starlark.NewBuiltin("findall", findall),
		"sub":     starlark.NewBuiltin("sub", sub),
	},
}

// compile(pattern, flags=0)
// Compile a regular expression pattern into a regular expression object, which
// can be used for matching using its match(), search() and other methods.
//
// The expression’s behaviour can be modified by specifying a flags value.
// Values can be any of the following variables, combined using bitwise OR
// (the | operator).
func compile(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		pattern starlark.String
		flags   starlark.Int
	)

	if err := starlark.UnpackArgs("compile", args, kwargs, "pattern", &pattern, "flags?", &flags); err != nil {
		return starlark.None, err
	}

	return newRegex(pattern)
}

// search(pattern,string,flags=0)
// Scan through string looking for the first location where the regular expression pattern produces a match,
// and return a corresponding match object. Return None if no position in the string matches the pattern;
// note that this is different from finding a zero-length match at some point in the string.
func search(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		pattern, str starlark.String
		flags        starlark.Int
	)
	if err := starlark.UnpackArgs("search", args, kwargs, "pattern", &pattern, "string", &str, "flags?", &flags); err != nil {
		return starlark.None, err
	}
	re, err := newGoRegex(pattern)
	if err != nil {
		return starlark.None, err
	}

	return reSearch(re, str, flags)
}

func reSearch(re *regexp.Regexp, str starlark.String, flags starlark.Int) (starlark.Value, error) {
	loc := re.FindStringIndex(string(str))
	if len(loc) == 0 {
		return starlark.None, nil
	}

	return starlark.String(str[loc[0]:loc[1]]), nil
}

// match(pattern, string, flags=0)
// If zero or more characters at the beginning of string match the regular expression pattern,
// return a corresponding match object. Return None if the string does not match the pattern;
// note that this is different from a zero-length match.
// Note that even in MULTILINE mode, re.match() will only match at the beginning of the string and not at the beginning of each line.
// If you want to locate a match anywhere in string, use search() instead
func match(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		pattern, str starlark.String
		flags        starlark.Int
	)
	if err := starlark.UnpackArgs("match", args, kwargs, "pattern", &pattern, "string", &str, "flags?", &flags); err != nil {
		return starlark.None, err
	}

	re, err := newGoRegex(pattern)
	if err != nil {
		return starlark.None, err
	}

	return reMatch(re, str, flags)
}

func reMatch(re *regexp.Regexp, str starlark.String, flags starlark.Int) (starlark.Value, error) {
	vals := starlark.NewList(nil)
	for _, match := range re.FindAllStringSubmatch(string(str), -1) {
		if err := vals.Append(slStrSlice(match)); err != nil {
			return starlark.None, err
		}
	}
	return vals, nil
}

// fullmatch(pattern, string, flags=0)¶
// If the whole string matches the regular expression pattern, return a corresponding match object.
// Return None if the string does not match the pattern; note that this is different from a zero-length match.
// func fullmatch(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
// 	var pattern starlark.String
// 	if err := starlark.UnpackArgs("fullmatch", args, kwargs, "pattern", &pattern); err != nil {
// 		return starlark.None, err
// 	}

// 	return starlark.None, nil
// }

func reSplit(re *regexp.Regexp, str starlark.String, maxSplit, flags starlark.Int) (starlark.Value, error) {
	ms, _ := maxSplit.Int64()
	if ms == 0 {
		// -1 is the sentinel for "all" in go, not 0
		ms = -1
	}

	res := re.Split(string(str), int(ms))
	return slStrSlice(res), nil
}

// findall(pattern, string, flags=0)
// Returns all non-overlapping matches of pattern in string, as a list of strings.
// The string is scanned left-to-right, and matches are returned in the order found.
// If one or more groups are present in the pattern, return a list of groups;
// this will be a list of tuples if the pattern has more than one group.
// Empty matches are included in the result.
func findall(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		pattern starlark.String
		str     starlark.String
		flags   starlark.Int
	)
	if err := starlark.UnpackArgs("findall", args, kwargs, "pattern", &pattern, "string", &str, "flags?", &flags); err != nil {
		return starlark.None, err
	}

	re, err := newGoRegex(pattern)
	if err != nil {
		return starlark.None, err
	}
	return reFindall(re, str, flags)
}

func reFindall(re *regexp.Regexp, str starlark.String, flags starlark.Int) (starlark.Value, error) {
	res := re.FindAllString(string(str), -1)
	return slStrSlice(res), nil
}

// func finditer(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
// 	var pattern starlark.String
// 	if err := starlark.UnpackArgs("finditer", args, kwargs, "pattern", &pattern); err != nil {
// 		return starlark.None, err
// 	}

// 	return starlark.None, nil
// }

// sub(pattern, repl, string, count=0, flags=0)
// Return the string obtained by replacing the leftmost non-overlapping occurrences of pattern
// in string by the replacement repl. If the pattern isn’t found, string is returned unchanged.
// repl can be a string or a function; if it is a string, any backslash escapes in it are processed.
// That is, \n is converted to a single newline character, \r is converted to a carriage return, and so forth.
// Unknown escapes such as \& are left alone. Backreferences, such as \6, are replaced with the substring matched by group 6 in the pattern.
func sub(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		pattern, repl, str starlark.String
		count, flags       starlark.Int
	)
	if err := starlark.UnpackArgs("sub", args, kwargs, "pattern", &pattern, "repl", &repl, "string", &str, "count?", &count, "flags", &flags); err != nil {
		return starlark.None, err
	}

	re, err := newGoRegex(pattern)
	if err != nil {
		return starlark.None, nil
	}

	return reSub(re, repl, str, count, flags)
}

func reSub(re *regexp.Regexp, repl, str starlark.String, count, flags starlark.Int) (starlark.Value, error) {
	res := re.ReplaceAllString(string(str), string(repl))
	return starlark.String(res), nil
}

// subn(pattern, repl, string, count=0, flags=0)
// Perform the same operation as sub(), but return a tuple (new_string, number_of_subs_made)
// func subn(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
// 	var pattern starlark.String
// 	if err := starlark.UnpackArgs("subn", args, kwargs, "pattern", &pattern); err != nil {
// 		return starlark.None, err
// 	}

// 	return starlark.None, nil
// }

// func escape(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
// 	var pattern starlark.String
// 	if err := starlark.UnpackArgs("escape", args, kwargs, "pattern", &pattern); err != nil {
// 		return starlark.None, err
// 	}

// 	return starlark.None, nil
// }

func newGoRegex(pattern starlark.String) (*regexp.Regexp, error) {
	return regexp.Compile(string(pattern))
}

func slStrSlice(strs []string) starlark.Tuple {
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

func newRegex(pattern starlark.String) (*Regex, error) {
	re, err := newGoRegex(pattern)
	if err != nil {
		return nil, err
	}
	return &Regex{re: re}, nil
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
	return builtinMethods(r, name, regexMethods)
}

var regexMethods = map[string]builtinMethod{
	"search":  compiledSearch,
	"match":   compiledMatch,
	"split":   compiledSplit,
	"findall": compiledFindall,
	"sub":     compiledSub,
}

// AttrNames lists available dot expression strings for time. required by
// starlark.HasAttrs interface
func (r *Regex) AttrNames() []string { return builtinAttrNames(regexMethods) }

func builtinAttrNames(methods map[string]builtinMethod) []string {
	names := make([]string, 0, len(methods))
	for name := range methods {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func compiledSearch(fnname string, recV starlark.Value, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		str   starlark.String
		flags starlark.Int
	)
	if err := starlark.UnpackArgs("search", args, kwargs, "string", &str, "flags?", &flags); err != nil {
		return starlark.None, err
	}

	r := recV.(*Regex)
	return reSearch(r.re, str, flags)
}

func compiledMatch(fnname string, recV starlark.Value, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		str   starlark.String
		flags starlark.Int
	)
	if err := starlark.UnpackArgs("match", args, kwargs, "string", &str, "flags?", &flags); err != nil {
		return starlark.None, err
	}

	r := recV.(*Regex)
	return reMatch(r.re, str, flags)
}

func compiledSplit(fnname string, recV starlark.Value, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		str             starlark.String
		maxSplit, flags starlark.Int
	)
	if err := starlark.UnpackArgs("split", args, kwargs, "string", &str, "maxsplit?", &maxSplit, "flags", &flags); err != nil {
		return starlark.None, err
	}

	r := recV.(*Regex)
	return reSplit(r.re, str, maxSplit, flags)
}

func compiledFindall(fnname string, recV starlark.Value, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		str   starlark.String
		flags starlark.Int
	)
	if err := starlark.UnpackArgs("findall", args, kwargs, "string", &str, "flags?", &flags); err != nil {
		return starlark.None, err
	}

	r := recV.(*Regex)
	return reFindall(r.re, str, flags)
}

func compiledSub(fnname string, recV starlark.Value, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		repl, str    starlark.String
		count, flags starlark.Int
	)
	if err := starlark.UnpackArgs("sub", args, kwargs, "repl", &repl, "string", &str, "count?", &count, "flags", &flags); err != nil {
		return starlark.None, err
	}

	r := recV.(*Regex)
	return reSub(r.re, repl, str, count, flags)
}

type builtinMethod func(fnname string, recv starlark.Value, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error)

func builtinMethods(recv starlark.Value, name string, methods map[string]builtinMethod) (starlark.Value, error) {
	method := methods[name]
	if method == nil {
		return nil, nil // no such method
	}

	// Allocate a closure over 'method'.
	impl := func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		return method(b.Name(), b.Receiver(), args, kwargs)
	}
	return starlark.NewBuiltin(name, impl).BindReceiver(recv), nil
}
