package stargo

import (
	"fmt"
	"reflect"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// A goArray represents a Go value of kind array.
//
// To avoid confusion, goArray intentionally does not implement
// starlark.Sliceable, because s[i:j:stride] must follow the semantics
// of Starlark, not Go: the result would be a copy, not an alias, and
// the third integer operand would be a stride, not a capacity.
// Use the go.slice built-in for the Go slice operator.
type goArray struct {
	v reflect.Value // kind=Array; !CanAddr
}

var (
	_ Value               = goArray{}
	_ starlark.Comparable = goArray{}
	_ starlark.Sequence   = goArray{}
	_ starlark.HasAttrs   = goArray{}
)

func (a goArray) Attr(name string) (starlark.Value, error) { return method(a.v, name) }
func (a goArray) AttrNames() []string                      { return methodNames(a.v) }
func (a goArray) Freeze()                                  {} // unimplementable
func (a goArray) Index(i int) starlark.Value               { return toStarlark(a.v.Index(i)) }
func (a goArray) Iterate() starlark.Iterator               { return &indexIterator{a.v, 0} }
func (a goArray) Len() int                                 { return a.v.Len() }
func (a goArray) Reflect() reflect.Value                   { return a.v }
func (a goArray) String() string                           { return str(a.v) }
func (a goArray) Truth() starlark.Bool                     { return isZero(a.v) == false }
func (a goArray) Type() string                             { return fmt.Sprintf("go.array<%s>", a.v.Type()) }

func (x goArray) CompareSameType(op syntax.Token, y_ starlark.Value, depth int) (bool, error) {
	y := y_.(goArray)
	switch op {
	case syntax.EQL:
		return arraysEqual(x, y)
	case syntax.NEQ:
		eq, err := arraysEqual(x, y)
		return !eq, err
	}
	return false, fmt.Errorf("invalid comparison: %s %s %s", x.Type(), op, y.Type())
}

// TODO: combine all equality predicates and thread a depth parameter.
func arraysEqual(x, y goArray) (bool, error) {
	// Only arrays of the same type compare equal.
	// This differs from Go in which the types may be unequal
	// so long as one operand is assignable to the other.
	if x.v.Type() != y.v.Type() {
		return false, nil
	}
	if x.Len() != y.Len() {
		return false, nil
	}
	for i, n := 0, x.Len(); i < n; i++ {
		if eq, err := starlark.Equal(x.Index(i), y.Index(i)); err != nil {
			return false, fmt.Errorf("at array index %d: %v", i, err)
		} else if !eq {
			return false, nil
		}
	}
	return true, nil
}

func (a goArray) Hash() (uint32, error) {
	n := a.Len()
	h := 7 * (uint32(n) + 1)
	for i := 0; i < n; i++ {
		x, err := a.Index(i).Hash()
		if err != nil {
			return 0, err
		}
		h ^= x
		h *= 16777619
	}
	return h, nil
}

type indexIterator struct {
	v reflect.Value // Kind=Array or Slice
	i int
}

func (it *indexIterator) Next(x *starlark.Value) bool {
	if it.i < it.v.Len() {
		*x = toStarlark(copyVal(it.v.Index(it.i)))
		it.i++
		return true
	}
	return false
}

func (it *indexIterator) Done() {} // mutation check is unimplementable
