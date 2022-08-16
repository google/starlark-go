package truth

import (
	"fmt"
	"strings"

	"go.starlark.net/starlark"
)

// InvalidAssertion signifies an invalid assertion was attempted
// such as comparing with None.
type InvalidAssertion string

var _ error = InvalidAssertion("")

func newInvalidAssertion(prop string) InvalidAssertion { return InvalidAssertion(prop) }
func (e InvalidAssertion) Error() string               { return string(e) }

// TruthAssertion signifies an assertion predicate was invalidated.
type TruthAssertion string

var _ error = TruthAssertion("")

func newTruthAssertion(msg string) TruthAssertion { return TruthAssertion(msg) }
func (e TruthAssertion) Error() string            { return string(e) }

// unhandled internal & public errors

const errUnhandled = unhandledError(0)

type unhandledError int

var _ error = errUnhandled

func (e unhandledError) Error() string { return "unhandled" }

// UnhandledError appears when an operation on an incompatible type is attempted.
type UnhandledError struct {
	name   string
	actual starlark.Value
	args   starlark.Tuple
}

var _ error = (*UnhandledError)(nil)

func (t *T) unhandled(name string, args ...starlark.Value) *UnhandledError {
	return &UnhandledError{
		name:   name,
		actual: t.actual,
		args:   args,
	}
}

func (e UnhandledError) Error() string {
	var b strings.Builder
	b.WriteString("Invalid assertion .")
	b.WriteString(e.name)
	b.WriteByte('(')
	for i, arg := range e.args {
		if i != 0 {
			b.WriteString(", ")
		}
		b.WriteString(arg.String())
	}
	b.WriteString(") on value of type ")
	b.WriteString(e.actual.Type())
	return b.String()
}

// UnresolvedError describes that an `assert.that(actual)` was called but never any of its `.truth_methods(subject)`.
// At the exception of (as each by themselves this still require an assertion):
// * `.named(name)`
// * `.is_within(tolerance)`
// * `.is_not_within(tolerance)`
type UnresolvedError string

var _ error = UnresolvedError("")

func (e UnresolvedError) Error() string {
	return fmt.Sprintf("%s: %s.that(...) is missing an assertion", string(e), Default)
}
