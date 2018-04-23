// Copyright 2017 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package skylarkstruct defines the Skylark 'struct' type,
// an optional language extension.
package skylarkstruct

// It is tempting to introduce a variant of Struct that is a wrapper
// around a Go struct value, for stronger typing guarantees and more
// efficient and convenient field lookup. However:
// 1) all fields of Skylark structs are optional, so we cannot represent
//    them using more specific types such as String, Int, *Depset, and
//    *File, as such types give no way to represent missing fields.
// 2) the efficiency gain of direct struct field access is rather
//    marginal: finding the index of a field by binary searching on the
//    sorted list of field names is quite fast compared to the other
//    overheads.
// 3) the gains in compactness and spatial locality are also rather
//    marginal: the array behind the []entry slice is (due to field name
//    strings) only a factor of 2 larger than the corresponding Go struct
//    would be, and, like the Go struct, requires only a single allocation.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/google/skylark"
	"github.com/google/skylark/syntax"
)

// Make is the implementation of a built-in function that instantiates
// an immutable struct from the specified keyword arguments.
//
// An application can add 'struct' to the Skylark environment like so:
//
// 	globals := skylark.StringDict{
// 		"struct":  skylark.NewBuiltin("struct", skylarkstruct.Make),
// 	}
//
func Make(_ *skylark.Thread, _ *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
	if len(args) > 0 {
		return nil, fmt.Errorf("struct: unexpected positional arguments")
	}
	return FromKeywords(Default, kwargs), nil
}

// FromKeywords returns a new struct instance whose fields are specified by the
// key/value pairs in kwargs.  (Each kwargs[i][0] must be a skylark.String.)
func FromKeywords(constructor skylark.Value, kwargs []skylark.Tuple) *Struct {
	if constructor == nil {
		panic("nil constructor")
	}
	s := &Struct{
		constructor: constructor,
		entries:     make(entries, 0, len(kwargs)),
	}
	for _, kwarg := range kwargs {
		k := string(kwarg[0].(skylark.String))
		v := kwarg[1]
		s.entries = append(s.entries, entry{k, v})
	}
	sort.Sort(s.entries)
	return s
}

// FromStringDict returns a whose elements are those of d.
// The constructor parameter specifies the constructor; use Default for an ordinary struct.
func FromStringDict(constructor skylark.Value, d skylark.StringDict) *Struct {
	if constructor == nil {
		panic("nil constructor")
	}
	s := &Struct{
		constructor: constructor,
		entries:     make(entries, 0, len(d)),
	}
	for k, v := range d {
		s.entries = append(s.entries, entry{k, v})
	}
	sort.Sort(s.entries)
	return s
}

// Struct is an immutable Skylark type that maps field names to values.
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
	constructor skylark.Value
	entries     entries // sorted by name
}

// Default is the default constructor for structs.
// It is merely the string "struct".
const Default = skylark.String("struct")

type entries []entry

func (a entries) Len() int           { return len(a) }
func (a entries) Less(i, j int) bool { return a[i].name < a[j].name }
func (a entries) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

type entry struct {
	name  string // not to_{proto,json}
	value skylark.Value
}

var (
	_ skylark.HasAttrs  = (*Struct)(nil)
	_ skylark.HasBinary = (*Struct)(nil)
)

// ToStringDict adds a name/value entry to d for each field of the struct.
func (s *Struct) ToStringDict(d skylark.StringDict) {
	for _, e := range s.entries {
		d[e.name] = e.value
	}
}

func (s *Struct) String() string {
	var buf bytes.Buffer
	if s.constructor == Default {
		// NB: The Java implementation always prints struct
		// even for Bazel provider instances.
		buf.WriteString("struct") // avoid String()'s quotation
	} else {
		buf.WriteString(s.constructor.String())
	}
	buf.WriteByte('(')
	for i, e := range s.entries {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(e.name)
		buf.WriteString(" = ")
		buf.WriteString(e.value.String())
	}
	buf.WriteByte(')')
	return buf.String()
}

// Constructor returns the constructor used to create this struct.
func (s *Struct) Constructor() skylark.Value { return s.constructor }

func (s *Struct) Type() string        { return "struct" }
func (s *Struct) Truth() skylark.Bool { return true } // even when empty
func (s *Struct) Hash() (uint32, error) {
	// Same algorithm as Tuple.hash, but with different primes.
	var x, m uint32 = 8731, 9839
	for _, e := range s.entries {
		namehash, _ := skylark.String(e.name).Hash()
		x = x ^ 3*namehash
		y, err := e.value.Hash()
		if err != nil {
			return 0, err
		}
		x = x ^ y*m
		m += 7349
	}
	return x, nil
}
func (s *Struct) Freeze() {
	for _, e := range s.entries {
		e.value.Freeze()
	}
}

func (x *Struct) Binary(op syntax.Token, y skylark.Value, side skylark.Side) (skylark.Value, error) {
	if y, ok := y.(*Struct); ok && op == syntax.PLUS {
		if side == skylark.Right {
			x, y = y, x
		}

		if eq, err := skylark.Equal(x.constructor, y.constructor); err != nil {
			return nil, fmt.Errorf("in %s + %s: error comparing constructors: %v",
				x.constructor, y.constructor, err)
		} else if !eq {
			return nil, fmt.Errorf("cannot add structs of different constructors: %s + %s",
				x.constructor, y.constructor)
		}

		z := make(skylark.StringDict, x.len()+y.len())
		for _, e := range x.entries {
			z[e.name] = e.value
		}
		for _, e := range y.entries {
			z[e.name] = e.value
		}

		return FromStringDict(x.constructor, z), nil
	}
	return nil, nil // unhandled
}

// Attr returns the value of the specified field,
// or deprecated method if the name is "to_json" or "to_proto"
// and the struct has no field of that name.
func (s *Struct) Attr(name string) (skylark.Value, error) {
	// Binary search the entries.
	// This implementation is a specialization of
	// sort.Search that avoids dynamic dispatch.
	n := len(s.entries)
	i, j := 0, n
	for i < j {
		h := int(uint(i+j) >> 1)
		if s.entries[h].name < name {
			i = h + 1
		} else {
			j = h
		}
	}
	if i < n && s.entries[i].name == name {
		return s.entries[i].value, nil
	}

	// TODO(adonovan): there may be a nice feature for core
	// skylark.Value here, especially for JSON, but the current
	// features are incomplete and underspecified.
	//
	// to_{json,proto} are deprecated, appropriately; see Google issue b/36412967.
	switch name {
	case "to_json", "to_proto":
		return skylark.NewBuiltin(name, func(thread *skylark.Thread, fn *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
			var buf bytes.Buffer
			var err error
			if name == "to_json" {
				err = writeJSON(&buf, s)
			} else {
				err = writeProtoStruct(&buf, 0, s)
			}
			if err != nil {
				// TODO(adonovan): improve error error messages
				// to show the path through the object graph.
				return nil, err
			}
			return skylark.String(buf.String()), nil
		}), nil
	}

	var ctor string
	if s.constructor != Default {
		ctor = s.constructor.String() + " "
	}
	return nil, fmt.Errorf("%sstruct has no .%s attribute", ctor, name)
}

func writeProtoStruct(out *bytes.Buffer, depth int, s *Struct) error {
	for _, e := range s.entries {
		if err := writeProtoField(out, depth, e.name, e.value); err != nil {
			return err
		}
	}
	return nil
}

func writeProtoField(out *bytes.Buffer, depth int, field string, v skylark.Value) error {
	if depth > 16 {
		return fmt.Errorf("to_proto: depth limit exceeded")
	}

	switch v := v.(type) {
	case *Struct:
		fmt.Fprintf(out, "%*s%s {\n", 2*depth, "", field)
		if err := writeProtoStruct(out, depth+1, v); err != nil {
			return err
		}
		fmt.Fprintf(out, "%*s}\n", 2*depth, "")
		return nil

	case *skylark.List, skylark.Tuple:
		iter := skylark.Iterate(v)
		defer iter.Done()
		var elem skylark.Value
		for iter.Next(&elem) {
			if err := writeProtoField(out, depth, field, elem); err != nil {
				return err
			}
		}
		return nil
	}

	// scalars
	fmt.Fprintf(out, "%*s%s: ", 2*depth, "", field)
	switch v := v.(type) {
	case skylark.Bool:
		fmt.Fprintf(out, "%t", v)

	case skylark.Int:
		// TODO(adonovan): limits?
		out.WriteString(v.String())

	case skylark.Float:
		fmt.Fprintf(out, "%g", v)

	case skylark.String:
		fmt.Fprintf(out, "%q", string(v))

	default:
		return fmt.Errorf("cannot convert %s to proto", v.Type())
	}
	out.WriteByte('\n')
	return nil
}

// writeJSON writes the JSON representation of a Skylark value to out.
// TODO(adonovan): there may be a nice feature for core skylark.Value here,
// but the current feature is incomplete and underspecified.
func writeJSON(out *bytes.Buffer, v skylark.Value) error {
	switch v := v.(type) {
	case skylark.NoneType:
		out.WriteString("null")
	case skylark.Bool:
		fmt.Fprintf(out, "%t", v)
	case skylark.Int:
		// TODO(adonovan): test large numbers.
		out.WriteString(v.String())
	case skylark.Float:
		// TODO(adonovan): test.
		fmt.Fprintf(out, "%g", v)
	case skylark.String:
		s := string(v)
		if goQuoteIsSafe(s) {
			fmt.Fprintf(out, "%q", s)
		} else {
			// vanishingly rare for text strings
			data, _ := json.Marshal(s)
			out.Write(data)
		}
	case skylark.Indexable: // Tuple, List
		out.WriteByte('[')
		for i, n := 0, skylark.Len(v); i < n; i++ {
			if i > 0 {
				out.WriteString(", ")
			}
			if err := writeJSON(out, v.Index(i)); err != nil {
				return err
			}
		}
		out.WriteByte(']')
	case *Struct:
		out.WriteByte('{')
		for i, e := range v.entries {
			if i > 0 {
				out.WriteString(", ")
			}
			if err := writeJSON(out, skylark.String(e.name)); err != nil {
				return err
			}
			out.WriteString(": ")
			if err := writeJSON(out, e.value); err != nil {
				return err
			}
		}
		out.WriteByte('}')
	default:
		return fmt.Errorf("cannot convert %s to JSON", v.Type())
	}
	return nil
}

func goQuoteIsSafe(s string) bool {
	for _, r := range s {
		// JSON doesn't like Go's \xHH escapes for ASCII control codes,
		// nor its \UHHHHHHHH escapes for runes >16 bits.
		if r < 0x20 || r >= 0x10000 {
			return false
		}
	}
	return true
}

func (s *Struct) len() int { return len(s.entries) }

// AttrNames returns a new sorted list of the struct fields.
//
// Unlike in the Java implementation,
// the deprecated struct methods "to_json" and "to_proto" do not
// appear in AttrNames, and hence dir(struct), since that would force
// the majority to have to ignore them, but they may nonetheless be
// called if the struct does not have fields of these names. Ideally
// these will go away soon. See Google issue b/36412967.
func (s *Struct) AttrNames() []string {
	names := make([]string, len(s.entries))
	for i, e := range s.entries {
		names[i] = e.name
	}
	return names
}

func (x *Struct) CompareSameType(op syntax.Token, y_ skylark.Value, depth int) (bool, error) {
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

	if eq, err := skylark.Equal(x.constructor, y.constructor); err != nil {
		return false, fmt.Errorf("error comparing struct constructors %v and %v: %v",
			x.constructor, y.constructor, err)
	} else if !eq {
		return false, nil
	}

	for i, n := 0, x.len(); i < n; i++ {
		if x.entries[i].name != y.entries[i].name {
			return false, nil
		} else if eq, err := skylark.EqualDepth(x.entries[i].value, y.entries[i].value, depth-1); err != nil {
			return false, err
		} else if !eq {
			return false, nil
		}
	}
	return true, nil
}
