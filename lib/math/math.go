// Copyright 2021 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package math provides basic constants and mathematical functions.
package math // import "go.starlark.net/lib/math"

import (
	"math"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

const (
	tau    = math.Pi * 2
	oneRad = tau / 360
)

var (
	toDeg = func(x float64) float64 { return x / oneRad }
	toRad = func(x float64) float64 { return x * oneRad }
)

// Module math is a Starlark module of math-related functions and constants.
var Module = &starlarkstruct.Module{
	Name: "math",
	Members: starlark.StringDict{
		"abs":   starlark.NewBuiltin("abs", abs),
		"ceil":  starlark.NewBuiltin("ceil", ceil),
		"floor": starlark.NewBuiltin("floor", floor),
		"round": starlark.NewBuiltin("round", round),

		"exp":  starlark.NewBuiltin("exp", exp),
		"sqrt": starlark.NewBuiltin("sqrt", sqrt),

		"acos":  starlark.NewBuiltin("acos", acos),
		"asin":  starlark.NewBuiltin("asin", asin),
		"atan":  starlark.NewBuiltin("atan", atan),
		"atan2": starlark.NewBuiltin("atan2", atan2),
		"cos":   starlark.NewBuiltin("cos", cos),
		"hypot": starlark.NewBuiltin("hypot", hypot),
		"sin":   starlark.NewBuiltin("sin", sin),
		"tan":   starlark.NewBuiltin("tan", tan),

		"degrees": starlark.NewBuiltin("degrees", degrees),
		"radians": starlark.NewBuiltin("radians", radians),

		"acosh": starlark.NewBuiltin("acosh", acosh),
		"asinh": starlark.NewBuiltin("asinh", asinh),
		"atanh": starlark.NewBuiltin("atanh", atanh),
		"cosh":  starlark.NewBuiltin("cosh", cosh),
		"sinh":  starlark.NewBuiltin("sinh", sinh),
		"tanh":  starlark.NewBuiltin("tanh", tanh),

		"e":   starlark.Float(math.E),
		"phi": starlark.Float(math.Phi),
		"pi":  starlark.Float(math.Pi),
	},
}

// oneArgFunc unpacks a starlark function call with one argument, calls a passed in function taking one float64 as argument
// and returns the result as a starlark value.
func oneArgFunc(name string, args starlark.Tuple, kwargs []starlark.Tuple, fn func(float64) float64) (starlark.Value, error) {
	var x float64
	if err := starlark.UnpackPositionalArgs(name, args, kwargs, 1, &x); err != nil {
		var i int64
		if starlark.UnpackPositionalArgs(name, args, kwargs, 1, &i) != nil {
			return nil, err
		}
		x = float64(i)
	}
	return starlark.Float(fn(x)), nil
}

// twoArgFunc unpacks a starlark function call with two arguments, calls a passed in function taking two float64 as argument
// and returns the result as a starlark value.
func twoArgFunc(name string, args starlark.Tuple, kwargs []starlark.Tuple, fn func(float64, float64) float64) (starlark.Value, error) {
	var x, y float64
	if err := starlark.UnpackPositionalArgs(name, args, kwargs, 2, &x, &y); err != nil {
		return nil, err
	}
	return starlark.Float(fn(x, y)), nil
}

// floor(x) - Return the floor of x, the largest integer less than or equal to x.
func floor(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return oneArgFunc("floor", args, kwargs, math.Floor)
}

// ceil(x) - Return the ceiling of x, the smallest integer greater than or equal to x.
func ceil(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return oneArgFunc("ceil", args, kwargs, math.Ceil)
}

// abs(x) - Return the absolute value of x.
func abs(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return oneArgFunc("abs", args, kwargs, math.Abs)
}

// round(x) - Round returns the nearest integer, rounding half away from zero.
func round(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return oneArgFunc("round", args, kwargs, math.Round)
}

// exp(x) - Return e raised to the power x, where e = 2.718281… is the base of natural logarithms.
func exp(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return oneArgFunc("exp", args, kwargs, math.Exp)
}

// sqrt(x) - Return the square root of x.
func sqrt(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return oneArgFunc("sqrt", args, kwargs, math.Sqrt)
}

// acos(x) - Return the arc cosine of x, in radians.
func acos(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return oneArgFunc("acos", args, kwargs, math.Acos)
}

// asin(x) - Return the arc sine of x, in radians.
func asin(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return oneArgFunc("asin", args, kwargs, math.Asin)
}

// atan(x) - Return the arc tangent of x, in radians.
func atan(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return oneArgFunc("atan", args, kwargs, math.Atan)
}

// atan2(y, x) - Return atan(y / x), in radians.
// The result is between -pi and pi.
// The vector in the plane from the origin to point (x, y) makes this angle with the positive X axis.
// The point of atan2() is that the signs of both inputs are known to it, so it can compute the correct quadrant for the angle.
// For example, atan(1) and atan2(1, 1) are both pi/4, but atan2(-1, -1) is -3*pi/4.
func atan2(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return twoArgFunc("atan2", args, kwargs, math.Atan2)
}

// cos(x) - Return the cosine of x, in radians.
func cos(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return oneArgFunc("cos", args, kwargs, math.Cos)
}

// hypot(x, y) - Return the Euclidean norm, sqrt(x*x + y*y). This is the length of the vector from the origin to point (x, y).
func hypot(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return twoArgFunc("hypot", args, kwargs, math.Hypot)
}

// sin(x) - Return the sine of x, in radians.
func sin(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return oneArgFunc("sin", args, kwargs, math.Sin)
}

// tan(x) - Return the tangent of x, in radians.
func tan(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return oneArgFunc("tan", args, kwargs, math.Tan)
}

// degrees(x) - Convert angle x from radians to degrees.
func degrees(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return oneArgFunc("degrees", args, kwargs, toDeg)
}

// radians(x) - Convert angle x from degrees to radians.
func radians(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return oneArgFunc("radians", args, kwargs, toRad)
}

// acosh(x) - Return the inverse hyperbolic cosine of x.
func acosh(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return oneArgFunc("acosh", args, kwargs, math.Acosh)
}

// asinh(x) - Return the inverse hyperbolic sine of x.
func asinh(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return oneArgFunc("asinh", args, kwargs, math.Asinh)
}

// atanh(x) - Return the inverse hyperbolic tangent of x.
func atanh(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return oneArgFunc("atanh", args, kwargs, math.Atanh)
}

// cosh(x) - Return the hyperbolic cosine of x.
func cosh(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return oneArgFunc("cosh", args, kwargs, math.Cosh)
}

// sinh(x) - Return the hyperbolic sine of x.
func sinh(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return oneArgFunc("sinh", args, kwargs, math.Sinh)
}

// tanh(x) - Return the hyperbolic tangent of x.
func tanh(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return oneArgFunc("tanh", args, kwargs, math.Tanh)
}
