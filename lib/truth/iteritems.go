package truth

import (
	"fmt"
	"strings"

	"go.starlark.net/starlark"
)

const tupleSliceType = "items"

var _ starlark.Iterable = (tupleSlice)(nil)

// tupleSlice is used to iterate on starlark.Dict's key-values, not its keys.
// From starlark-go docs:
// > If a type satisfies both Mapping and Iterable, the iterator yields
// > the keys of the mapping.
type tupleSlice []starlark.Tuple

func newTupleSlice(ts []starlark.Tuple) tupleSlice { return tupleSlice(ts) }
func (ts tupleSlice) String() string {
	var b strings.Builder
	b.WriteString(tupleSliceType)
	b.WriteString("([")
	for i, v := range ts {
		if i != 0 {
			b.WriteString(", ")
		}
		b.WriteString(v.String())
	}
	b.WriteString("])")
	return b.String()
}
func (ts tupleSlice) Type() string { return tupleSliceType }
func (ts tupleSlice) Freeze() {
	for _, v := range ts {
		v.Freeze()
	}
}
func (ts tupleSlice) Truth() starlark.Bool { return len(ts) > 0 }
func (ts tupleSlice) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable type: %s", tupleSliceType)
}

func (ts tupleSlice) Values() []starlark.Value {
	vs := make([]starlark.Value, 0, len(ts))
	for _, v := range ts {
		vs = append(vs, v)
	}
	return vs
}

func (ts tupleSlice) Iterate() starlark.Iterator { return newTupleSliceIterator(ts) }

var _ starlark.Iterator = (*tupleSliceIterator)(nil)

type tupleSliceIterator struct {
	s tupleSlice
	i int
}

func newTupleSliceIterator(ts tupleSlice) *tupleSliceIterator { return &tupleSliceIterator{s: ts} }
func (tsi *tupleSliceIterator) Done()                         {}
func (tsi *tupleSliceIterator) Next(v *starlark.Value) bool {
	if tsi.i < len(tsi.s) {
		*v = tsi.s[tsi.i]
		tsi.i++
		return true
	}
	return false
}
