// Copyright 2020 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package starlarkjson defines utilities for converting Starlark values
// to/from JSON strings. The most recent IETF standard for JSON is
// https://www.ietf.org/rfc/rfc7159.txt.
package starlarkjson // import "go.starlark.net/starlarkjson"

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// Module json is a Starlark module of JSON-related functions.
//
//   json = module(
//      encode,
//      decode,
//      indent,
//   )
//
// def encode(x):
//
// The encode function accepts one required positional argument,
// which it converts to JSON by cases:
// - A Starlark value that implements Go's standard json.Marshal
//   interface defines its own JSON encoding.
// - None, True, and False are converted to null, true, and false, respectively.
// - Starlark int values, no matter how large, are encoded as decimal integers.
//   Some decoders may not be able to decode very large integers.
// - Starlark float values are encoded using decimal point notation,
//   even if the value is an integer.
//   It is an error to encode a non-finite floating-point value.
// - Starlark strings are encoded as JSON strings, using UTF-16 escapes.
// - a Starlark IterableMapping (e.g. dict) is encoded as a JSON object.
//   It is an error if any key is not a string.
// - any other Starlark Iterable (e.g. list, tuple) is encoded as a JSON array.
// - a Starlark HasAttrs (e.g. struct) is encoded as a JSON object.
// It an application-defined type matches more than one the cases describe above,
// (e.g. it implements both Iterable and HasFields), the first case takes precedence.
// Encoding any other value yields an error.
//
// def decode(x):
//
// The decode function accepts one positional parameter, a JSON string.
// It returns the Starlark value that the string denotes.
// - Numbers are parsed as int or float, depending on whether they
//   contain a decimal point.
// - JSON objects are parsed as new unfrozen Starlark dicts.
// - JSON arrays are parsed as new unfrozen Starlark lists.
// Decoding fails if x is not a valid JSON string.
//
// def indent(str, *, prefix="", indent="\t"):
//
// The indent function pretty-prints a valid JSON encoding,
// and returns a string containing the indented form.
// It accepts one required positional parameter, the JSON string,
// and two optional keyword-only string parameters, prefix and indent,
// that specify a prefix of each new line, and the unit of indentation.
//
var Module = &starlarkstruct.Module{
	Name: "json",
	Members: starlark.StringDict{
		"encode": starlark.NewBuiltin("json.encode", encode),
		"decode": starlark.NewBuiltin("json.decode", decode),
		"indent": starlark.NewBuiltin("json.indent", indent),
	},
}

func encode(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var x starlark.Value
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &x); err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)

	var quoteSpace [128]byte
	quote := func(s string) {
		// Non-trivial escaping is handled by Go's encoding/json.
		if isPrintableASCII(s) {
			buf.Write(strconv.AppendQuote(quoteSpace[:0], s))
		} else {
			// TODO(adonovan): opt: RFC 8259 mandates UTF-8 for JSON.
			// Can we avoid this call?
			data, _ := json.Marshal(s)
			buf.Write(data)
		}
	}

	var emit func(x starlark.Value) error
	emit = func(x starlark.Value) error {
		switch x := x.(type) {
		case json.Marshaler:
			// Application-defined starlark.Value types
			// may define their own JSON encoding.
			data, err := x.MarshalJSON()
			if err != nil {
				return err
			}
			buf.Write(data)

		case starlark.NoneType:
			buf.WriteString("null")

		case starlark.Bool:
			if x {
				buf.WriteString("true")
			} else {
				buf.WriteString("false")
			}

		case starlark.Int:
			fmt.Fprint(buf, x)

		case starlark.Float:
			if !isFinite(float64(x)) {
				return fmt.Errorf("cannot encode non-finite float %v", x)
			}
			fmt.Fprintf(buf, "%g", x) // always contains a decimal point

		case starlark.String:
			quote(string(x))

		case starlark.IterableMapping:
			// e.g. dict (must have string keys)
			buf.WriteByte('{')
			iter := x.Iterate()
			defer iter.Done()
			var k starlark.Value
			for i := 0; iter.Next(&k); i++ {
				if i > 0 {
					buf.WriteByte(',')
				}
				s, ok := starlark.AsString(k)
				if !ok {
					return fmt.Errorf("%s has %s key, want string", x.Type(), k.Type())
				}
				v, found, err := x.Get(k)
				if err != nil || !found {
					log.Fatalf("internal error: mapping %s has %s among keys but value lookup fails", x.Type(), k)
				}

				quote(s)
				buf.WriteByte(':')
				if err := emit(v); err != nil {
					return fmt.Errorf("in %s key %s: %v", x.Type(), k, err)
				}
			}
			buf.WriteByte('}')

		case starlark.Iterable:
			// e.g. tuple, list
			buf.WriteByte('[')
			iter := x.Iterate()
			defer iter.Done()
			var elem starlark.Value
			for i := 0; iter.Next(&elem); i++ {
				if i > 0 {
					buf.WriteByte(',')
				}
				if err := emit(elem); err != nil {
					return fmt.Errorf("at %s index %d: %v", x.Type(), i, err)
				}
			}
			buf.WriteByte(']')

		case starlark.HasAttrs:
			// e.g. struct
			buf.WriteByte('{')
			var names []string
			names = append(names, x.AttrNames()...)
			sort.Strings(names)
			for i, name := range names {
				v, err := x.Attr(name)
				if err != nil || v == nil {
					log.Fatalf("internal error: dir(%s) includes %q but value has no .%s field", x.Type(), name, name)
				}
				if i > 0 {
					buf.WriteByte(',')
				}
				quote(name)
				buf.WriteByte(':')
				if err := emit(v); err != nil {
					return fmt.Errorf("in field .%s: %v", name, err)
				}
			}
			buf.WriteByte('}')

		default:
			return fmt.Errorf("cannot encode %s as JSON", x.Type())
		}
		return nil
	}

	if err := emit(x); err != nil {
		return nil, fmt.Errorf("%s: %v", b.Name(), err)
	}
	return starlark.String(buf.String()), nil
}

// isPrintableASCII reports whether s contains only printable ASCII.
func isPrintableASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b < 0x20 || b >= 0x80 {
			return false
		}
	}
	return true
}

// isFinite reports whether f represents a finite rational value.
// It is equivalent to !math.IsNan(f) && !math.IsInf(f, 0).
func isFinite(f float64) bool {
	return math.Abs(f) <= math.MaxFloat64
}

func indent(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	prefix, indent := "", "\t" // keyword-only
	if err := starlark.UnpackArgs(b.Name(), nil, kwargs,
		"prefix?", &prefix,
		"indent?", &indent,
	); err != nil {
		return nil, err
	}
	var str string // positional-only
	if err := starlark.UnpackPositionalArgs(b.Name(), args, nil, 1, &str); err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	if err := json.Indent(buf, []byte(str), prefix, indent); err != nil {
		return nil, fmt.Errorf("%s: %v", b.Name(), err)
	}
	return starlark.String(buf.String()), nil
}

func decode(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (_ starlark.Value, err error) {
	var s string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &s); err != nil {
		return nil, err
	}

	// The decoder necessarily makes certain representation choices
	// such as list vs tuple, struct vs dict, int vs float.
	// In principle, we could parameterize it to allow the caller to
	// control the returned types, but there's no compelling need yet.

	// Use panic/recover with a distinguished type (failure) for error handling.
	type failure string
	fail := func(format string, args ...interface{}) {
		panic(failure(fmt.Sprintf(format, args...)))
	}

	i := 0

	// skipSpace consumes leading spaces, and reports whether there is more input.
	skipSpace := func() bool {
		for ; i < len(s); i++ {
			b := s[i]
			if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
				return true
			}
		}
		return false
	}

	// next consumes leading spaces and returns the first non-space.
	// It panics if at EOF.
	next := func() byte {
		if skipSpace() {
			return s[i]
		}
		fail("unexpected end of file")
		panic("unreachable")
	}

	// parse returns the next JSON value from the input.
	// It consumes leading but not trailing whitespace.
	// It panics on error.
	var parse func() starlark.Value
	parse = func() starlark.Value {
		b := next()
		switch b {
		case '"':
			// string

			// Find end of quotation.
			// Also, record whether trivial unquoting is safe.
			// Non-trivial unquoting is handled by Go's encoding/json.
			safe := true
			closed := false
			j := i + 1
			for ; j < len(s); j++ {
				b := s[j]
				if b == '\\' {
					safe = false
					j++ // skip x in \x
				} else if b == '"' {
					closed = true
					j++ // skip '"'
					break
				} else if b >= utf8.RuneSelf {
					safe = false
				}
			}
			if !closed {
				fail("unclosed string literal")
			}

			r := s[i:j]
			i = j

			// unquote
			if safe {
				r = r[1 : len(r)-1]
			} else if err := json.Unmarshal([]byte(r), &r); err != nil {
				fail("%s", err)
			}
			return starlark.String(r)

		case 'n':
			if strings.HasPrefix(s[i:], "null") {
				i += len("null")
				return starlark.None
			}

		case 't':
			if strings.HasPrefix(s[i:], "true") {
				i += len("true")
				return starlark.True
			}

		case 'f':
			if strings.HasPrefix(s[i:], "false") {
				i += len("false")
				return starlark.False
			}

		case '[':
			// array
			var elems []starlark.Value

			i++ // '['
			b = next()
			if b != ']' {
				for {
					elem := parse()
					elems = append(elems, elem)
					b = next()
					if b != ',' {
						if b != ']' {
							fail("got %q, want ',' or ']'", b)
						}
						break
					}
					i++ // ','
				}
			}
			i++ // ']'
			return starlark.NewList(elems)

		case '{':
			// object
			dict := new(starlark.Dict)

			i++ // '{'
			b = next()
			if b != '}' {
				for {
					key := parse()
					if _, ok := key.(starlark.String); !ok {
						fail("got %s for object key, want string", key.Type())
					}
					b = next()
					if b != ':' {
						fail("after object key, got %q, want ':' ", b)
					}
					i++ // ':'
					value := parse()
					dict.SetKey(key, value) // can't fail
					b = next()
					if b != ',' {
						if b != '}' {
							fail("in object, got %q, want ',' or '}'", b)
						}
						break
					}
					i++ // ','
				}
			}
			i++ // '}'
			return dict

		default:
			// number?
			if isdigit(b) || b == '-' {
				// scan literal. Allow [0-9+-eE.] for now.
				float := false
				var j int
				for j = i + 1; j < len(s); j++ {
					b = s[j]
					if isdigit(b) {
						// ok
					} else if b == '.' ||
						b == 'e' ||
						b == 'E' ||
						b == '+' ||
						b == '-' {
						float = true
					} else {
						break
					}
				}
				num := s[i:j]
				i = j

				// Unlike most C-like languages,
				// JSON disallows a leading zero before a digit.
				digits := num
				if num[0] == '-' {
					digits = num[1:]
				}
				if digits == "" || digits[0] == '0' && len(digits) > 1 && isdigit(digits[1]) {
					fail("invalid number: %s", num)
				}

				// parse literal
				if float {
					x, err := strconv.ParseFloat(num, 64)
					if err != nil {
						fail("invalid number: %s", num)
					}
					return starlark.Float(x)
				} else {
					x, ok := new(big.Int).SetString(num, 10)
					if !ok {
						fail("invalid number: %s", num)
					}
					return starlark.MakeBigInt(x)
				}
			}
		}
		fail("unexpected character %q", b)
		panic("unreachable")
	}
	defer func() {
		x := recover()
		switch x := x.(type) {
		case failure:
			err = fmt.Errorf("json.decode: at offset %d, %s", i, x)
		case nil:
			// nop
		default:
			panic(x) // unexpected panic
		}
	}()
	x := parse()
	if skipSpace() {
		fail("unexpected character %q after value", s[i])
	}
	return x, nil
}

func isdigit(b byte) bool {
	return b >= '0' && b <= '9'
}
