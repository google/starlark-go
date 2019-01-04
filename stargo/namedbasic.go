package stargo

import (
	"fmt"
	"reflect"
	"strings"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// A goNamedBasic represents a Go value of a named type whose
// underlying type is bool, number, or string.
//
// Values of the predeclared Go bool, string, and numeric
// types are represented by starlark.Bool, starlark.Int,
// starlark.Float, and stargo.Complex.
//
// goNamedBasic values do not currently participate in arithmetic.
// Values must be explictly converted to Starlark string, int, or float, or
// stargo.Complex, and the result explicitly converted back if, desired.
//
// goNamedBasic string values do not currently support indexing, nor do
// they possess the methods of starlark.String. (They could be made to,
// because there is no conflict between the names of Starlark string
// methods, which are lower case, and exported Go methods, which are
// capitalized.)
type goNamedBasic struct {
	v reflect.Value // Kind=Bool | String | Number; !CanAddr
}

// TODO:
// - support a broader equivalence relation across numbers of different types and kinds.
// - support hashing, consistent with equality.
// - support string indexing.
// - support methods of Starlark string. What type would they return?
// - support arithmetic:
//
//   +                          number x number, string x string
//   *                          number x number, string x number, number x string
//   -   /   //   %             number x number
//   in                         string x string
//   |   &   ^   <<   >>        int x int
//   +x  -x                     number
//
// Arithmetic raises many subtle questions.
// Should operators such as % and >> use the semantics of Starlark or
// of Go when both operands are Go values?
// Should we permit operands of different types, such as
// - Starlark + Go?
// - Go + Go, same kind but different types?
// - Go + Go, different types?
// If we support different types, what determines the type of the result?
// - The left operand? This enables common cases like x+1, x+".txt", x<<1, etc.
// - The Go operand in the case of Go+Starlark or Starlark+Go?
// - Should I / I where I is a Go integer type yield a Starlark float, as int / int does?
// - Should I * S and S * I return S, where I is a Go int and S a string? Or string?
// The number of cases to consider may explode.
// Should Starlark support x^y for bool x bool and int x int, like Python?

var (
	_ Value               = goNamedBasic{}
	_ starlark.Comparable = goNamedBasic{}
	_ starlark.HasAttrs   = goNamedBasic{}
)

func (b goNamedBasic) Attr(name string) (starlark.Value, error) { return method(b.v, name) }
func (b goNamedBasic) AttrNames() []string                      { return methodNames(b.v) }
func (b goNamedBasic) Hash() (uint32, error)                    { return 0, fmt.Errorf("unhashable: %s", b.Type()) }
func (b goNamedBasic) Freeze()                                  {} // immutable
func (b goNamedBasic) Reflect() reflect.Value                   { return b.v }
func (b goNamedBasic) String() string                           { return str(b.v) }
func (b goNamedBasic) Truth() starlark.Bool                     { return isZero(b.v) == false }
func (b goNamedBasic) Type() string {
	// e.g. "go.uint<parser.Mode>".
	return fmt.Sprintf("go.%s<%s>", strings.ToLower(b.v.Kind().String()), b.v.Type())
}

func (x goNamedBasic) CompareSameType(op syntax.Token, y_ starlark.Value, depth int) (bool, error) {
	y := y_.(goNamedBasic)

	// Reject comparisons where the Go types are not equal,
	// just as the Go compiler does statically for x==y.
	// This restriction may prove too onerous for an untyped language.
	// Also, it is not consistent with Starlark's (mathematical)
	// equivalence relation for numbers.
	// TODO: more design required.
	if x.v.Type() != y.v.Type() {
		return false, fmt.Errorf("unsupported comparison %s %s %s", x.Type(), op, y.Type())
	}

	return x.v == y.v, nil
}
