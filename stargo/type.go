package stargo

import (
	"fmt"
	"reflect"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// A Type is a Starlark value that wraps a non-nil Go reflect.Type.
//
// The unary * operator yields the pointer type, e.g. *bytes.Buffer.
type Type struct {
	t reflect.Type // non-nil
}

var (
	_ Value               = Type{}
	_ starlark.Callable   = Type{}
	_ starlark.Comparable = Type{}
	_ starlark.HasAttrs   = Type{}
	_ starlark.HasUnary   = Type{}
)

func (t Type) Hash() (uint32, error)                    { return uint32(t.Reflect().Pointer()), nil }
func (t Type) String() string                           { return t.t.String() }
func (t Type) Type() string                             { return "go.type" }
func (t Type) Truth() starlark.Bool                     { return true }
func (t Type) Reflect() reflect.Value                   { return reflect.ValueOf(t.t) }
func (t Type) Freeze()                                  {} // immutable
func (t Type) Name() string                             { return t.t.Name() }
func (t Type) Attr(name string) (starlark.Value, error) { return method(t.Reflect(), name) }
func (t Type) AttrNames() []string                      { return methodNames(t.Reflect()) }

// Calling a type "T()" returns the zero value of T.
// Calling a type with one argument "T(x)" is a conversion;
// it applies the same set of conversions that would occur implicitly.
// Use new(T) to create a new variable of type T.
func (t Type) CallInternal(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(kwargs) > 0 {
		return nil, fmt.Errorf("%s: unexpected keyword arguments", t)
	}
	if len(args) > 1 {
		return nil, fmt.Errorf("%s: got %d arguments, want zero or one", t, len(args))
	}
	var r reflect.Value
	if len(args) == 0 {
		// T(): return zero value
		r = reflect.Zero(t.t)
	} else {
		// T(x): conversion
		var err error
		r, err = toGo(args[0], t.t)
		if err != nil {
			return nil, err
		}
	}
	return toStarlark(r), nil
}

func (x Type) CompareSameType(op syntax.Token, y starlark.Value, depth int) (bool, error) {
	switch op {
	case syntax.EQL:
		return x == y, nil
	case syntax.NEQ:
		return x != y, nil
	}
	return false, fmt.Errorf("invalid comparison: %s %s %s", x.Type(), op, y.Type())
}

func (t Type) Unary(op syntax.Token) (starlark.Value, error) {
	if op == syntax.STAR {
		return Type{reflect.PtrTo(t.t)}, nil
	}
	return nil, nil
}
