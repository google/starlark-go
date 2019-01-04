package stargo

import (
	"fmt"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// A Complex represents a complex number with
// double-precision floating-point real and imaginary components.
//
// Stargo uses this type to represent values of Go's predeclared
// complex types, complex64 and complex128.
//
// TODO: A Complex does compare equal to any other type of number
// because it would be asymmetric. We would have to move this type into
// the starlark package and add a case in CompareDepth, or give Starlark
// an extensible mechanism for comparing all kinds of numbers including
// non-core ones) symmetrically, similar to HasBinary; see what Python
// does. Any solution must ensure that Hash is consistent with equals.
type Complex complex128

var (
	_ starlark.Comparable = Complex(0)
	_ starlark.HasBinary  = Complex(0) // + - * /
	_ starlark.HasUnary   = Complex(0) // + -
)

func (c Complex) Freeze()               {}                                              // immutable
func (c Complex) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: complex") } // TODO
func (c Complex) String() string        { return fmt.Sprint(complex128(c)) }
func (c Complex) Truth() starlark.Bool  { return c != 0 }
func (c Complex) Type() string          { return "complex" }

func (x Complex) CompareSameType(op syntax.Token, y_ starlark.Value, depth int) (bool, error) {
	y := y_.(Complex)
	switch op {
	case syntax.EQL:
		return x == y, nil
	case syntax.NEQ:
		return x != y, nil
	}
	return false, fmt.Errorf("invalid comparison: complex %s complex", op)
}

func (x Complex) Binary(op syntax.Token, y_ starlark.Value, side starlark.Side) (starlark.Value, error) {
	var y Complex
	switch y_ := y_.(type) {
	case Complex:
		y = y_
	case starlark.Float:
		y = Complex(complex(y_, 0))
	case starlark.Int:
		y = Complex(complex(y_.Float(), 0))
	default:
		return nil, nil
	}

	if side == starlark.Right {
		x, y = y, x
	}

	switch op {
	case syntax.PLUS:
		return x + y, nil
	case syntax.MINUS:
		return x - y, nil
	case syntax.STAR:
		return x * y, nil
	case syntax.SLASH:
		if y == 0.0 {
			return nil, fmt.Errorf("complex division by zero")
		}
		return x / y, nil
	}
	return nil, nil
}

func (c Complex) Unary(op syntax.Token) (starlark.Value, error) {
	switch op {
	case syntax.PLUS:
		return +c, nil
	case syntax.MINUS:
		return -c, nil
	}
	return nil, nil
}

// -- builtins --

// complex(re, im)
func go٠complex(x, y float64) complex128 { return complex(x, y) }

// real(complex)
func go٠real(x complex128) float64 { return real(x) }

// imag(complex)
func go٠imag(x complex128) float64 { return imag(x) }
