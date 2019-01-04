package stargo

import (
	"fmt"
	"reflect"

	"go.starlark.net/starlark"
)

// A goSlice represents a Go value of kind slice.
//
// To avoid confusion, goSlice intentionally does not implement
// starlark.Sliceable, because s[i:j:stride] must follow the semantics
// of Starlark, not Go: the result would be a copy, not an alias, and
// the third integer operand would be a stride, not a capacity.
// Use the go.slice built-in for the Go slice operator.
type goSlice struct {
	v reflect.Value // kind=Slice; !CanAddr
}

var (
	_ Value                = goSlice{}
	_ starlark.HasAttrs    = goSlice{}
	_ starlark.HasSetIndex = goSlice{}
	_ starlark.Sequence    = goSlice{}
)

func (s goSlice) Attr(name string) (starlark.Value, error) { return method(s.v, name) }
func (s goSlice) AttrNames() []string                      { return methodNames(s.v) }
func (s goSlice) Freeze()                                  {} // unimplementable
func (s goSlice) Hash() (uint32, error)                    { return 0, fmt.Errorf("unhashable: %s", s.Type()) }
func (s goSlice) Index(i int) starlark.Value               { return varOf(s.v.Index(i)) }
func (s goSlice) Iterate() starlark.Iterator               { return &indexIterator{s.v, 0} }
func (s goSlice) Len() int                                 { return s.v.Len() }
func (s goSlice) Reflect() reflect.Value                   { return s.v }
func (s goSlice) SetIndex(i int, v starlark.Value) error   { return setIndex(s.v, i, v) }
func (s goSlice) String() string                           { return str(s.v) }
func (s goSlice) Truth() starlark.Bool                     { return s.v.IsNil() == false }
func (s goSlice) Type() string                             { return fmt.Sprintf("go.slice<%s>", s.v.Type()) }

// setIndex is the common implementation of slice/array element update.
func setIndex(seq reflect.Value, i int, v starlark.Value) error {
	elem := seq.Index(i)
	x, err := toGo(v, elem.Type())
	if err != nil {
		return err
	}
	if !elem.CanSet() {
		return fmt.Errorf("can't set element of %s", seq.Type()) // e.g. unexported
	}
	elem.Set(x)
	return nil
}

// sliceConvert returns a new slice of the specified type containing the elements of iterable.
func sliceConvert(iterable starlark.Iterable, sliceType reflect.Type) (reflect.Value, error) {
	n := starlark.Len(iterable)
	if n < 0 {
		n = 0
	}
	slice := reflect.MakeSlice(sliceType, 0, n)
	iter := iterable.Iterate()
	defer iter.Done()
	var elem starlark.Value
	for i := 0; iter.Next(&elem); i++ {
		y, err := toGo(elem, sliceType.Elem())
		if err != nil {
			return slice, fmt.Errorf("in element %d, %v", i+1, err)
		}
		slice = reflect.Append(slice, y)
	}
	return slice, nil
}

// -- builtins --

// make_slice(type, len, cap=len)
func go٠make_slice(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (res starlark.Value, err error) {
	var t Type
	var length int
	var cap int
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 2, &t, &length, &cap); err != nil {
		return nil, err
	}
	if len(args) == 2 {
		cap = length
	}
	err = protect(thread, b.Name(), func() {
		res = toStarlark(reflect.MakeSlice(t.t, length, cap))
	})
	return res, err
}

// slice(slice, start, end, cap?) -- like Go x[start:end:cap], not Starlark x[start:end:stride].
func go٠slice(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (res starlark.Value, err error) {
	var s goSlice
	var start, end, cap int
	cap = -1
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 3, &s, &start, &end, &cap); err != nil {
		return nil, err
	}
	err = protect(thread, b.Name(), func() {
		if cap < 0 {
			res = toStarlark(s.v.Slice(start, end))
		} else {
			res = toStarlark(s.v.Slice3(start, end, cap))
		}
	})
	return res, err
}

// copy(dest, src)
func go٠copy(dest, src interface{}) int {
	return reflect.Copy(reflect.ValueOf(dest), reflect.ValueOf(src))
}

// append(slice, *args)
func go٠append(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(kwargs) > 0 {
		return nil, fmt.Errorf("%s: unexpected keyword arguments", b.Name())
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("%s: got 0 arguments, want at least 1", b.Name())
	}
	slice, ok := args[0].(goSlice)
	if !ok {
		return nil, fmt.Errorf("%s: want slice, got %s for argument 1", b.Name(), args[0].Type())
	}
	vals := make([]reflect.Value, len(args)-1)
	for i, arg := range args[1:] {
		val, err := toGo(arg, slice.v.Type().Elem())
		if err != nil {
			return nil, fmt.Errorf("%s: in argument %d, %v", b.Name(), i+2, err)
		}
		vals[i] = val
	}
	return toStarlark(reflect.Append(slice.v, vals...)), nil
}

// cap(v)
func go٠cap(x interface{}) int { return reflect.ValueOf(x).Cap() }
