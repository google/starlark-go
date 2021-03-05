package math

import (
	"math"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

const tau = math.Pi * 2
const oneRad = tau / 360

// Module math is a Starlark module of math-related functions.
var Module = &starlarkstruct.Module{
	Name: "math",
	Members: starlark.StringDict{
		"ceil":  starlark.NewBuiltin("ceil", ceil),
		"fabs":  starlark.NewBuiltin("fabs", fabs),
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
		"pi":  starlark.Float(math.Pi),
		"tau": starlark.Float(tau),
		"phi": starlark.Float(math.Phi),
		"inf": starlark.Float(math.Inf(1)),
		"nan": starlark.Float(math.NaN()),
	},
}

// floatFunc unpacks a starlark function call, calls a passed in float64 function
// and returns the result as a starlark value
func floatFunc(name string, args starlark.Tuple, kwargs []starlark.Tuple, fn func(float64) float64) (starlark.Value, error) {
	var x starlark.Float
	if err := starlark.UnpackPositionalArgs(name, args, kwargs, 1, &x); err != nil {
		return nil, err
	}
	return starlark.Float(fn(float64(x))), nil
}

// floatFunc2 is a 2-argument float func
func floatFunc2(name string, args starlark.Tuple, kwargs []starlark.Tuple, fn func(float64, float64) float64) (starlark.Value, error) {
	var x, y starlark.Float
	if err := starlark.UnpackPositionalArgs(name, args, kwargs, 2, &x, &y); err != nil {
		return nil, err
	}
	return starlark.Float(fn(float64(x), float64(y))), nil
}

// Return the floor of x, the largest integer less than or equal to x.
func floor(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return floatFunc("floor", args, kwargs, math.Floor)
}

// Return the ceiling of x, the smallest integer greater than or equal to x
func ceil(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return floatFunc("ceil", args, kwargs, math.Ceil)
}

// Return the absolute value of x
func fabs(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return floatFunc("fabs", args, kwargs, math.Abs)
}

// Round returns the nearest integer, rounding half away from zero.
func round(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return floatFunc("fabs", args, kwargs, math.Round)
}

// Return e raised to the power x, where e = 2.718281… is the base of natural logarithms.
func exp(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return floatFunc("exp", args, kwargs, math.Exp)
}

// Return the square root of x
func sqrt(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return floatFunc("sqrt", args, kwargs, math.Sqrt)
}
