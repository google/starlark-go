// Copyright 2021 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package math provides basic constants and mathematical functions.
package math // import "go.starlark.net/lib/math"

import (
	"errors"
	"fmt"
	"math"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// Module math is a Starlark module of math-related functions and constants.
// The module defines the following functions:
//
//     ceil(x) - Returns the ceiling of x, the smallest integer greater than or equal to x.
//     copysign(x, y) - Returns a value with the magnitude of x and the sign of y.
//     fabs(x) - Returns the absolute value of x as float.
//     floor(x) - Returns the floor of x, the largest integer less than or equal to x.
//     mod(x, y) - Returns the floating-point remainder of x/y. The magnitude of the result is less than y and its sign agrees with that of x.
//     pow(x, y) - Returns x**y, the base-x exponential of y.
//     remainder(x, y) - Returns the IEEE 754 floating-point remainder of x/y.
//     round(x) - Returns the nearest integer, rounding half away from zero.
//
//     exp(x) - Returns e raised to the power x, where e = 2.718281… is the base of natural logarithms.
//     sqrt(x) - Returns the square root of x.
//
//     acos(x) - Returns the arc cosine of x, in radians.
//     asin(x) - Returns the arc sine of x, in radians.
//     atan(x) - Returns the arc tangent of x, in radians.
//     atan2(y, x) - Returns atan(y / x), in radians.
//                   The result is between -pi and pi.
//                   The vector in the plane from the origin to point (x, y) makes this angle with the positive X axis.
//                   The point of atan2() is that the signs of both inputs are known to it, so it can compute the correct
//                   quadrant for the angle.
//                   For example, atan(1) and atan2(1, 1) are both pi/4, but atan2(-1, -1) is -3*pi/4.
//     cos(x) - Returns the cosine of x, in radians.
//     hypot(x, y) - Returns the Euclidean norm, sqrt(x*x + y*y). This is the length of the vector from the origin to point (x, y).
//     sin(x) - Returns the sine of x, in radians.
//     tan(x) - Returns the tangent of x, in radians.
//
//     degrees(x) - Converts angle x from radians to degrees.
//     radians(x) - Converts angle x from degrees to radians.
//
//     acosh(x) - Returns the inverse hyperbolic cosine of x.
//     asinh(x) - Returns the inverse hyperbolic sine of x.
//     atanh(x) - Returns the inverse hyperbolic tangent of x.
//     cosh(x) - Returns the hyperbolic cosine of x.
//     sinh(x) - Returns the hyperbolic sine of x.
//     tanh(x) - Returns the hyperbolic tangent of x.
//
//     log(x, base) - Returns the logarithm of x in the given base, or natural logarithm by default.
//
//     gamma(x) - Returns the Gamma function of x.
//
// All functions accept both int and float values as arguments.
//
// The module also defines approximations of the following constants:
//
//     e - The base of natural logarithms, approximately 2.71828.
//     pi - The ratio of a circle's circumference to its diameter, approximately 3.14159.
//
var Module = &starlarkstruct.Module{
	Name: "math",
	Members: starlark.StringDict{
		"ceil":      starlark.NewBuiltin("ceil", ceil),
		"copysign":  newBinaryBuiltin("copysign", math.Copysign),
		"fabs":      newUnaryBuiltin("fabs", math.Abs),
		"floor":     starlark.NewBuiltin("floor", floor),
		"mod":       newBinaryBuiltin("mod", math.Mod),
		"pow":       newBinaryBuiltin("pow", math.Pow),
		"remainder": newBinaryBuiltin("remainder", math.Remainder),
		"round":     newUnaryBuiltin("round", math.Round),

		"exp":  newUnaryBuiltin("exp", math.Exp),
		"sqrt": newUnaryBuiltin("sqrt", math.Sqrt),

		"acos":  newUnaryBuiltin("acos", math.Acos),
		"asin":  newUnaryBuiltin("asin", math.Asin),
		"atan":  newUnaryBuiltin("atan", math.Atan),
		"atan2": newBinaryBuiltin("atan2", math.Atan2),
		"cos":   newUnaryBuiltin("cos", math.Cos),
		"hypot": newBinaryBuiltin("hypot", math.Hypot),
		"sin":   newUnaryBuiltin("sin", math.Sin),
		"tan":   newUnaryBuiltin("tan", math.Tan),

		"degrees": newUnaryBuiltin("degrees", degrees),
		"radians": newUnaryBuiltin("radians", radians),

		"acosh": newUnaryBuiltin("acosh", math.Acosh),
		"asinh": newUnaryBuiltin("asinh", math.Asinh),
		"atanh": newUnaryBuiltin("atanh", math.Atanh),
		"cosh":  newUnaryBuiltin("cosh", math.Cosh),
		"sinh":  newUnaryBuiltin("sinh", math.Sinh),
		"tanh":  newUnaryBuiltin("tanh", math.Tanh),

		"log": starlark.NewBuiltin("log", log),

		"gamma": newUnaryBuiltin("gamma", math.Gamma),

		"e":  starlark.Float(math.E),
		"pi": starlark.Float(math.Pi),
	},
}

// floatOrInt is an Unpacker that converts a Starlark int or float to Go's float64.
type floatOrInt float64

func (p *floatOrInt) Unpack(v starlark.Value) error {
	switch v := v.(type) {
	case starlark.Int:
		*p = floatOrInt(v.Float())
		return nil
	case starlark.Float:
		*p = floatOrInt(v)
		return nil
	}
	return fmt.Errorf("got %s, want float or int", v.Type())
}

// newUnaryBuiltin wraps a unary floating-point Go function
// as a Starlark built-in that accepts int or float arguments.
func newUnaryBuiltin(name string, fn func(float64) float64) *starlark.Builtin {
	return starlark.NewBuiltin(name, func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var x floatOrInt
		if err := starlark.UnpackPositionalArgs(name, args, kwargs, 1, &x); err != nil {
			return nil, err
		}
		return starlark.Float(fn(float64(x))), nil
	})
}

// newBinaryBuiltin wraps a binary floating-point Go function
// as a Starlark built-in that accepts int or float arguments.
func newBinaryBuiltin(name string, fn func(float64, float64) float64) *starlark.Builtin {
	return starlark.NewBuiltin(name, func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var x, y floatOrInt
		if err := starlark.UnpackPositionalArgs(name, args, kwargs, 2, &x, &y); err != nil {
			return nil, err
		}
		return starlark.Float(fn(float64(x), float64(y))), nil
	})
}

//  log wraps the Log function
// as a Starlark built-in that accepts int or float arguments.
func log(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		x    floatOrInt
		base floatOrInt = math.E
	)
	if err := starlark.UnpackPositionalArgs("log", args, kwargs, 1, &x, &base); err != nil {
		return nil, err
	}
	if base == 1 {
		return nil, errors.New("division by zero")
	}
	return starlark.Float(math.Log(float64(x)) / math.Log(float64(base))), nil
}

func ceil(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var x starlark.Value

	if err := starlark.UnpackPositionalArgs("ceil", args, kwargs, 1, &x); err != nil {
		return nil, err
	}

	switch t := x.(type) {
	case starlark.Int:
		return t, nil
	case starlark.Float:
		return starlark.NumberToInt(starlark.Float(math.Ceil(float64(t))))
	}

	return nil, fmt.Errorf("got %s, want float or int", x.Type())
}

func floor(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var x starlark.Value

	if err := starlark.UnpackPositionalArgs("floor", args, kwargs, 1, &x); err != nil {
		return nil, err
	}

	switch t := x.(type) {
	case starlark.Int:
		return t, nil
	case starlark.Float:
		return starlark.NumberToInt(starlark.Float(math.Floor(float64(t))))
	}

	return nil, fmt.Errorf("got %s, want float or int", x.Type())
}

func degrees(x float64) float64 {
	return 360 * x / (2 * math.Pi)
}

func radians(x float64) float64 {
	return 2 * math.Pi * x / 360
}
