package truth

import (
	"math/big"

	"go.starlark.net/starlark"
)

// T wraps an assert target
type T struct {
	// Target in assert.that(target)
	actual starlark.Value

	// Readable optional prefix with .named(name)
	name string

	// True when actual was a String and was made into an iterable.
	// Helps when pretty printing.
	actualIsIterableFromString bool

	// forOrdering is relevant to .in_order() assertions
	forOrdering *forOrdering

	// registered holds the compiled default compare function
	registered *registeredValues

	// withinTolerance is used to delta-compare numbers
	withinTolerance *withinTolerance
}

type forOrdering struct {
	inOrderError error
}

type withinTolerance struct {
	within           bool
	actual           *big.Rat
	tolerance        *big.Rat
	toleranceAsValue starlark.Value
}

func (t *T) turnActualIntoIterableFromString() {
	s := t.actual.(starlark.String).GoString()
	vs := make([]starlark.Value, 0, len(s))
	for _, c := range s {
		vs = append(vs, starlark.String(c))
	}
	t.actual = starlark.Tuple(vs)
	t.actualIsIterableFromString = true
}

type registeredValues struct {
	Cmp   starlark.Value
	Apply func(f *starlark.Function, args starlark.Tuple) (starlark.Value, error)
}

const cmpSrc = `lambda a, b: int(a > b) - int(a < b)`

func (t *T) registerValues(thread *starlark.Thread) error {
	if t.registered == nil {
		cmp, err := starlark.Eval(thread, "", cmpSrc, starlark.StringDict{})
		if err != nil {
			return err
		}

		apply := func(f *starlark.Function, args starlark.Tuple) (starlark.Value, error) {
			return starlark.Call(thread, f, args, nil)
		}

		t.registered = &registeredValues{
			Cmp:   cmp,
			Apply: apply,
		}
	}
	return nil
}
