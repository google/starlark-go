package stargo

import (
	"fmt"
	"reflect"
	"sort"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// A goStruct represents a Go value of kind struct.
type goStruct struct {
	v reflect.Value // kind=Struct; !CanAddr
}

var (
	_ Value               = goStruct{}
	_ starlark.Comparable = goStruct{}
	_ starlark.HasAttrs   = goStruct{}
)

func (s goStruct) Freeze()                {} // unimplementable
func (s goStruct) Reflect() reflect.Value { return s.v }
func (s goStruct) String() string         { return str(s.v) }
func (s goStruct) Truth() starlark.Bool   { return isZero(s.v) == false }
func (s goStruct) Type() string           { return fmt.Sprintf("go.struct<%s>", s.v.Type()) }

func (s goStruct) Attr(name string) (starlark.Value, error) {
	if m := s.v.MethodByName(name); m.IsValid() {
		return goFunc{m}, nil
	}

	// struct field
	if f := s.v.FieldByName(name); f.IsValid() {
		if !f.CanInterface() {
			return nil, fmt.Errorf("access to unexported field .%s", name)
		}
		return toStarlark(f), nil
	}

	return nil, nil
}

func (s goStruct) AttrNames() []string {
	t := s.v.Type()
	recv := t
	if s.v.CanAddr() {
		// TODO: this isn't right but we may yet want to
		// report the attrnames of *T since if f is in dir(lvalue),
		// you can call lvalue.f(). Or maybe not.
		recv = reflect.PtrTo(t)
	}
	var names []string
	names = appendMethodNames(names, recv)
	names = appendFieldNames(names, t)
	sort.Strings(names)
	return names
}

func (x goStruct) CompareSameType(op syntax.Token, y_ starlark.Value, depth int) (bool, error) {
	y := y_.(goStruct)
	switch op {
	case syntax.EQL:
		return structsEqual(x, y)
	case syntax.NEQ:
		eq, err := structsEqual(x, y)
		return !eq, err
	}
	return false, fmt.Errorf("invalid comparison: %s %s %s", x.Type(), op, y.Type())
}

// TODO: add a depth parameter, as in goArray.
func structsEqual(x, y goStruct) (bool, error) {
	t := x.v.Type()

	// TODO: should Go values compare using the Go equivalence
	// relation (which may may panic dynamically, but we can catch that)?
	// If so, then x==x does not imply struct{x}==struct{x}.
	// I don't think there's a satisfactory answer.
	// How do we even do that? reflect.Value equivalence is not clearly specified.
	// Ask Russ.

	// Only structs of the same type compare equal.
	// This differs from Go in which the types may be unequal
	// so long as one operand is assignable to the other.
	if t != y.v.Type() {
		return false, nil
	}

	// Compare non-blank fields using Starlark equivalence.
	n := t.NumField()
	for i := 0; i < n; i++ {
		name := t.Field(i).Name
		if name == "_" {
			continue
		}

		xf := x.v.Field(i)
		yf := y.v.Field(i)

		// TODO: wrap may panic for non-exported fields.
		if eq, err := starlark.Equal(toStarlark(xf), toStarlark(yf)); err != nil {
			return false, fmt.Errorf("in struct field .%s: %v", name, err)
		} else if !eq {
			return false, nil
		}
	}

	return true, nil
}

func (s goStruct) Hash() (uint32, error) {
	// TODO: implement by analogy:
	// goStruct.Hash : goArray.Hash :: structsEqual : arraysEqual.
	return 0, fmt.Errorf("unhashable: %s", s.Type())
}
