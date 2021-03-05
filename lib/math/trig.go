package math

import (
	"math"

	"go.starlark.net/starlark"
)

// Return the arc cosine of x, in radians.
func acos(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return floatFunc("acos", args, kwargs, math.Acos)
}

// asin(x) - Return the arc sine of x, in radians.
func asin(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return floatFunc("asin", args, kwargs, math.Asin)
}

// atan(x) - Return the arc tangent of x, in radians.
func atan(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return floatFunc("atan", args, kwargs, math.Atan)
}

// atan2(y, x) - Return atan(y / x), in radians. The result is between -pi and pi.
// The vector in the plane from the origin to point (x, y) makes this angle with the positive X axis.
// The point of atan2() is that the signs of both inputs are known to it, so it can compute the correct quadrant for the angle.
// For example, atan(1) and atan2(1, 1) are both pi/4, but atan2(-1, -1) is -3*pi/4.
func atan2(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return floatFunc2("atan2", args, kwargs, math.Atan2)
}

// cos(x) - Return the cosine of x radians.
func cos(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return floatFunc("cos", args, kwargs, math.Cos)
}

// hypot(x, y) - Return the Euclidean norm, sqrt(x*x + y*y). This is the length of the vector from the origin to point (x, y).
func hypot(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return floatFunc2("hypot", args, kwargs, math.Hypot)
}

// sin(x) - Return the sine of x radians.
func sin(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return floatFunc("sin", args, kwargs, math.Sin)
}

// tan(x) - Return the tangent of x radians.
func tan(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return floatFunc("tan", args, kwargs, math.Tan)
}

// degrees(x) - Convert angle x from radians to degrees.
func degrees(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	toDeg := func(x float64) float64 { return x / oneRad }
	return floatFunc("degrees", args, kwargs, toDeg)
}

// radians(x) - Convert angle x from degrees to radians.
func radians(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	toRad := func(x float64) float64 { return x * oneRad }
	return floatFunc("radians", args, kwargs, toRad)
}

// acosh(x) - Return the inverse hyperbolic cosine of x.
func acosh(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return floatFunc("acosh", args, kwargs, math.Acosh)
}

// asinh(x) - Return the inverse hyperbolic sine of x.
func asinh(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return floatFunc("asinh", args, kwargs, math.Asinh)
}

// atanh(x) - Return the inverse hyperbolic tangent of x.
func atanh(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return floatFunc("atanh", args, kwargs, math.Atanh)
}

// cosh(x) - Return the hyperbolic cosine of x.
func cosh(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return floatFunc("cosh", args, kwargs, math.Cosh)
}

// sinh(x) - Return the hyperbolic sine of x.
func sinh(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return floatFunc("sinh", args, kwargs, math.Sinh)
}

// tanh(x) - Return the hyperbolic tangent of x.
func tanh(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return floatFunc("tanh", args, kwargs, math.Tanh)
}
