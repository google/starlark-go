//go:build go1.23

package proto

import (
	"iter"

	"go.starlark.net/starlark"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Elements returns a go1.23 iterator over the values of a repeated field.
// For example:
//
//   for val := range repeatedField.Elements() { ... }
func (rf *RepeatedField) Elements() iter.Seq[starlark.Value] {
	return func(yield func(starlark.Value) bool) {
		for i := range rf.list.Len() {
			if !yield(rf.Index(i)) {
				return
			}
		}
	}
}

// Entries returns a go1.23 iterator over the values of a map field. For
// example:
//
//   for k, v := range mapField.Entries() { ... }
func (mf *MapField) Entries() iter.Seq2[starlark.Value, starlark.Value] {
	return func(yield func(k, v starlark.Value) bool) {
		mf.mp.Range(func(mk protoreflect.MapKey, v protoreflect.Value) bool {
			return yield(
				toStarlark1(mf.typ.MapKey(), mk.Value(), mf.frozen),
				toStarlark1(mf.typ.MapValue(), v, mf.frozen),
			)
		})
	}
}
