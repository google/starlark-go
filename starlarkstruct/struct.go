// Copyright 2017 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package starlarkstruct defines the Starlark types 'struct' and
// 'module', both optional language extensions.
//
package starlarkstruct // import "go.starlark.net/starlarkstruct"

// It is tempting to introduce a variant of Struct that is a wrapper
// around a Go struct value, for stronger typing guarantees and more
// efficient and convenient field lookup. However:
// 1) all fields of Starlark structs are optional, so we cannot represent
//    them using more specific types such as String, Int, *Depset, and
//    *File, as such types give no way to represent missing fields.
// 2) the efficiency gain of direct struct field access is rather
//    marginal: finding the index of a field by map access is O(1)
//    and is quite fast compared to the other overheads.
// Such an implementation is likely to be more compact than
// the current map-based representation, though.

import (
	"fmt"
	"strings"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// Make is the implementation of a built-in function that instantiates
// an immutable struct from the specified keyword arguments.
//
// An application can add 'struct' to the Starlark environment like so:
//
// 	globals := starlark.StringDict{
// 		"struct":  starlark.NewBuiltin("struct", starlarkstruct.Make),
// 	}
//
func Make(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) > 0 {
		return nil, fmt.Errorf("struct: unexpected positional arguments")
	}
	return FromKeywords(Default, kwargs), nil
}

// FromKeywords returns a new struct instance whose fields are specified by the
// key/value pairs in kwargs.  (Each kwargs[i][0] must be a starlark.String.)
func FromKeywords(constructor starlark.Value, kwargs []starlark.Tuple) *Struct {
	if constructor == nil {
		panic("nil constructor")
	}

	osd := starlark.NewOrderedStringDict(len(kwargs))
	for _, kwarg := range kwargs {
		k := string(kwarg[0].(starlark.String))
		v := kwarg[1]
		osd.Insert(k, v)
	}
	osd.Sort() // sort by key
	return &Struct{
		constructor: constructor,
		osd:         *osd,
	}
}

// FromStringDict returns a new struct instance whose elements are those of d.
// The constructor parameter specifies the constructor; use Default for an ordinary struct.
func FromStringDict(constructor starlark.Value, d starlark.StringDict) *Struct {
	if constructor == nil {
		panic("nil constructor")
	}
	osd := starlark.NewOrderedStringDict(len(d))
	for key, val := range d {
		osd.Insert(key, val)
	}
	osd.Sort() // sort by key
	return &Struct{
		constructor: constructor,
		osd:         *osd,
	}
}

// Struct is an immutable Starlark type that maps field names to values.
// It is not iterable and does not support len.
//
// A struct has a constructor, a distinct value that identifies a class
// of structs, and which appears in the struct's string representation.
//
// Operations such as x+y fail if the constructors of the two operands
// are not equal.
//
// The default constructor, Default, is the string "struct", but
// clients may wish to 'brand' structs for their own purposes.
// The constructor value appears in the printed form of the value,
// and is accessible using the Constructor method.
//
// Use Attr to access its fields and AttrNames to enumerate them.
type Struct struct {
	constructor starlark.Value
	osd         starlark.OrderedStringDict
}

// Default is the default constructor for structs.
// It is merely the string "struct".
const Default = starlark.String("struct")

var (
	_ starlark.HasAttrs  = (*Struct)(nil)
	_ starlark.HasBinary = (*Struct)(nil)
)

// ToStringDict adds a name/value entry to d for each field of the struct.
func (s *Struct) ToStringDict(d starlark.StringDict) {
	for i := 0; i < s.osd.Len(); i++ {
		k, v := s.osd.KeyIndex(i)
		d[k] = v
	}
}

func (s *Struct) String() string {
	buf := new(strings.Builder)
	if s.constructor == Default {
		// NB: The Java implementation always prints struct
		// even for Bazel provider instances.
		buf.WriteString("struct") // avoid String()'s quotation
	} else {
		buf.WriteString(s.constructor.String())
	}
	buf.WriteByte('(')
	for i := 0; i < s.osd.Len(); i++ {
		k, v := s.osd.KeyIndex(i)
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(k)
		buf.WriteString(" = ")
		buf.WriteString(v.String())
	}
	buf.WriteByte(')')
	return buf.String()
}

// Constructor returns the constructor used to create this struct.
func (s *Struct) Constructor() starlark.Value { return s.constructor }

func (s *Struct) Type() string         { return "struct" }
func (s *Struct) Truth() starlark.Bool { return true } // even when empty
func (s *Struct) Hash() (uint32, error) {
	// Same algorithm as Tuple.hash, but with different primes.
	var x, m uint32 = 8731, 9839
	for i := 0; i < s.osd.Len(); i++ {
		k, v := s.osd.KeyIndex(i)
		namehash, _ := starlark.String(k).Hash()
		x = x ^ 3*namehash
		y, err := v.Hash()
		if err != nil {
			return 0, err
		}
		x = x ^ y*m
		m += 7349
	}
	return x, nil
}
func (s *Struct) Freeze() {
	for i := 0; i < s.osd.Len(); i++ {
		s.osd.Index(i).Freeze()
	}
}

func (x *Struct) Binary(op syntax.Token, y starlark.Value, side starlark.Side) (starlark.Value, error) {
	if y, ok := y.(*Struct); ok && op == syntax.PLUS {
		if side == starlark.Right {
			x, y = y, x
		}

		if eq, err := starlark.Equal(x.constructor, y.constructor); err != nil {
			return nil, fmt.Errorf("in %s + %s: error comparing constructors: %v",
				x.constructor, y.constructor, err)
		} else if !eq {
			return nil, fmt.Errorf("cannot add structs of different constructors: %s + %s",
				x.constructor, y.constructor)
		}

		z := make(starlark.StringDict, x.len()+y.len())
		for i := 0; i < x.osd.Len(); i++ {
			k, v := x.osd.KeyIndex(i)
			z[k] = v
		}
		for i := 0; i < y.osd.Len(); i++ {
			k, v := y.osd.KeyIndex(i)
			z[k] = v
		}

		return FromStringDict(x.constructor, z), nil
	}
	return nil, nil // unhandled
}

// Attr returns the value of the specified field.
func (s *Struct) Attr(name string) (starlark.Value, error) {
	if v, ok := s.osd.Get(name); ok {
		return v, nil
	}
	var ctor string
	if s.constructor != Default {
		ctor = s.constructor.String() + " "
	}
	return nil, starlark.NoSuchAttrError(
		fmt.Sprintf("%sstruct has no .%s attribute", ctor, name))
}

func (s *Struct) len() int { return s.osd.Len() }

// AttrNames returns a new sorted list of the struct fields.
func (s *Struct) AttrNames() []string { return s.osd.Keys() }

func (x *Struct) CompareSameType(op syntax.Token, y_ starlark.Value, depth int) (bool, error) {
	y := y_.(*Struct)
	switch op {
	case syntax.EQL:
		return structsEqual(x, y, depth)
	case syntax.NEQ:
		eq, err := structsEqual(x, y, depth)
		return !eq, err
	default:
		return false, fmt.Errorf("%s %s %s not implemented", x.Type(), op, y.Type())
	}
}

func structsEqual(x, y *Struct, depth int) (bool, error) {
	if x.len() != y.len() {
		return false, nil
	}

	if eq, err := starlark.Equal(x.constructor, y.constructor); err != nil {
		return false, fmt.Errorf("error comparing struct constructors %v and %v: %v",
			x.constructor, y.constructor, err)
	} else if !eq {
		return false, nil
	}

	for i := 0; i < x.len(); i++ {
		xkey, xval := x.osd.KeyIndex(i)
		ykey, yval := y.osd.KeyIndex(i)

		if xkey != ykey {
			return false, nil
		} else if eq, err := starlark.EqualDepth(xval, yval, depth-1); err != nil {
			return false, err
		} else if !eq {
			return false, nil
		}
	}
	return true, nil

}
