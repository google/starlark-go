package stargo

// This file defines the Go/Starlark value conversions.

import (
	"fmt"
	"reflect"
	"sort"
	"unsafe"

	"go.starlark.net/starlark"
)

// Precondition: v.IsValid().
func copyVal(v reflect.Value) reflect.Value {
	// TODO: opt: optimize toStarlark(copyval(x)).
	if !v.CanAddr() {
		return v
	}
	if v.CanInterface() {
		return reflect.ValueOf(v.Interface()) // create an rvalue copy
	}
	ptr := reflect.New(v.Type())
	ptr.Elem().Set(v) // panics if !CanSet, e.g. leak of unexported field
	return ptr.Elem()

	// TODO: opt: there must be a more efficient way to shallow-copy an
	// lvalue reflect.Value and return a non-addressable rvalue.
	// One inefficient way: call func(x interface{}) interface{} { return x }.
	// Can we do better?
}

// wrap converts a Go value to a Starlark one.
//
// Precondition: !v.CanAddr() (almost)
func toStarlark(v reflect.Value) starlark.Value {
	if v.CanAddr() {
		// This assertion is almost right: we should never need
		// addressable values in here. We create explicit
		// pointers as needed.
		//
		// However, copyVal may create addressable values
		// unnecessarily. Can we defeat it and make this an
		// invariant?
		panic("addr")
	}

	switch v.Kind() {
	case reflect.Invalid:
		return starlark.None

	case reflect.Bool:
		if named(v.Type()) {
			return goNamedBasic{v}
		}
		return starlark.Bool(v.Bool())

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if named(v.Type()) {
			return goNamedBasic{v}
		}
		return starlark.MakeInt64(v.Int())

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if named(v.Type()) {
			return goNamedBasic{v}
		}
		return starlark.MakeUint64(v.Uint())

	case reflect.Float32, reflect.Float64:
		if named(v.Type()) {
			return goNamedBasic{v}
		}
		return starlark.Float(v.Float())

	case reflect.String:
		if named(v.Type()) {
			return goNamedBasic{v}
		}
		return starlark.String(v.String())

	case reflect.Complex64, reflect.Complex128:
		if named(v.Type()) {
			return goNamedBasic{v}
		}
		return Complex(v.Complex())

	case reflect.Array:
		return goArray{v}

	case reflect.Chan:
		return goChan{v}

	case reflect.Func:
		return goFunc{v}

	case reflect.Interface:
		// This means v is an lvalue.
		if v.IsNil() {
			return starlark.None
		}
		return toStarlark(v.Elem())

	case reflect.Map:
		return goMap{v}

	case reflect.Ptr:
		// special case:
		if v.Type() == rtype {
			// Invariant: *rtype pointer is non-nil.
			return Type{v.Interface().(reflect.Type)}
		}
		if !v.IsNil() && v.Elem().Kind() == reflect.Array {
			return goArrayPtr{v}
		} else {
			return goPtr{v}
		}

	case reflect.Slice:
		return goSlice{v}

	case reflect.Struct:
		return goStruct{v}

	case reflect.UnsafePointer:
		return goUnsafePointer{v}
	}
	panic("wrap: unknown kind: " + v.Kind().String())
}

var (
	rtype     = reflect.TypeOf(reflect.TypeOf(int(0)))   // *reflect.rtype
	eface     = reflect.TypeOf(new(interface{})).Elem()  // interface{}
	stringer  = reflect.TypeOf(new(fmt.Stringer)).Elem() // fmt.Stringer
	errorType = reflect.TypeOf(new(error)).Elem()        // error
)

// named reports whether t denotes a named type.
func named(t reflect.Type) bool { return t.PkgPath() != "" }

// str returns the printed form of the value in v.
func str(v reflect.Value) string { return fmt.Sprint(v.Interface()) }

// toGo converts a Starlark value to a Go value of the specified type.
func toGo(v starlark.Value, to reflect.Type) (reflect.Value, error) {
	// This function needs a lot more rigor (and optimization).

	// If the Starlark value is already a wrapper
	// around a Go value or Type, try the usual Go conversions.
	if g, ok := v.(Value); ok {
		rv := g.Reflect()
		if rv.Type() == to {
			return rv, nil
		}
		if rv.Type().ConvertibleTo(to) {
			return rv.Convert(to), nil
		}

		// Other conversions not implicitly allowed in Go:

		// unsafe.Pointer <=> *T requires explicit conversion in Go.
		if rv.Kind() == reflect.UnsafePointer && to.Kind() == reflect.Ptr {
			return reflect.NewAt(to.Elem(), unsafe.Pointer(rv.Pointer())), nil
		}
		if rv.Kind() == reflect.Ptr && to.Kind() == reflect.UnsafePointer {
			return reflect.ValueOf(unsafe.Pointer(rv.Pointer())), nil
		}
		// TODO: we should allow *array to map/slice implicit conversions.

		return reflect.Value{}, fmt.Errorf("cannot convert %s to Go %s", v.Type(), to)
	}

	// Allow None to convert to any pointer type.
	// A nil pointer is not equal to None, though.
	if v == starlark.None {
		switch to.Kind() {
		case reflect.Chan,
			reflect.Func,
			reflect.Interface,
			reflect.Map,
			reflect.Ptr,
			reflect.Slice,
			reflect.UnsafePointer:
			return reflect.Zero(to), nil
		}
		return reflect.Value{}, fmt.Errorf("cannot convert %s to Go %s", v.Type(), to)
	}

	// starlark.Value -> interface{}
	if to == eface {
		// Treat bools, numbers, and strings like untyped constants.
		// All other starlark.Values are passed through unchanged.
		var r interface{}
		switch v := v.(type) {
		case starlark.String:
			r = string(v)
		case starlark.Bool:
			r = bool(v)
		case starlark.Float:
			r = float64(v)
		case Complex:
			r = complex128(v)
		case starlark.Int:
			i, ok := v.Int64()
			if !ok || int64(int(i)) != i {
				return reflect.Value{}, fmt.Errorf("can't convert %s to interface{}", v)
			}
			r = int(i)
		default:
			r = v
		}
		return reflect.ValueOf(r), nil
	}

	// Conversions from Starlark to Go act as if the Starlark
	// value was a Go untyped constant: one can assign 1 to any
	// numeric type, or "foo" to any string type, even if named.

	switch to.Kind() {
	case reflect.Bool:
		// We don't use v.Truth at the interface with Go booleans.
		// Callers must explicitly say bool(x).
		if v, ok := v.(starlark.Bool); ok {
			return reflect.ValueOf(bool(v)).Convert(to), nil
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		// TODO: truncation check
		switch v := v.(type) {
		case starlark.Float:
			return reflect.ValueOf(float64(v)).Convert(to), nil // TODO safe?

		case starlark.Int:
			i, ok := v.Int64()
			if !ok || int64(int(i)) != i {
				return reflect.Value{}, fmt.Errorf("can't convert %s to interface{}", v)
			}
			return reflect.ValueOf(int(i)).Convert(to), nil // TODO safe?
		}

	case reflect.Float32, reflect.Float64:
		// TODO: allow ints
		if f, ok := v.(starlark.Float); ok {
			return reflect.ValueOf(f).Convert(to), nil
		}

	case reflect.Complex64, reflect.Complex128:
		// TODO: allow ints and floats
		if c, ok := v.(Complex); ok {
			return reflect.ValueOf(c).Convert(to), nil
		}

	case reflect.String:
		if s, ok := starlark.AsString(v); ok {
			return reflect.ValueOf(s).Convert(to), nil
		}

	case reflect.Interface:
		// We handled Go conversions and empty interfaces already,
		// so this case means a Starlark value is being converted
		// to a non-empty Go interface, which should in general fail.
		//
		// There is one special case though: every Value has a String method.
		if to.AssignableTo(stringer) {
			return reflect.ValueOf(v), nil
		}

	case reflect.Map:
		// go.map(iterable mapping)
		// go.map(iterable of pairs)
		if iterable, ok := v.(starlark.Iterable); ok {
			return mapConvert(iterable, to)
		}

	case reflect.Slice:
		// []byte(string)
		if s, ok := v.(starlark.String); ok && to == reflect.TypeOf([]byte(nil)) {
			return reflect.ValueOf([]byte(s)), nil
		}

		// go.slice(iterable)
		if iterable, ok := v.(starlark.Iterable); ok {
			return sliceConvert(iterable, to)
		}

	case reflect.Ptr,
		reflect.Struct,
		reflect.UnsafePointer,
		reflect.Array,
		reflect.Chan:
		// No implicit conversion to Go chan, array, pointer, struct, unsafe.Pointer.

	case reflect.Func:
		if callable, ok := v.(starlark.Callable); ok {
			return funcConvert(callable, to)
		}
	}

	return reflect.Value{}, fmt.Errorf("cannot convert %s to Go %s", v.Type(), to)
}

// ---- helpers ----

// method implements the Attr method of all types except struct and
// pointer (which may have fields too).
func method(v reflect.Value, name string) (starlark.Value, error) {
	if m := v.MethodByName(name); m.IsValid() {
		return goFunc{m}, nil
	}
	return nil, nil
}

// methodNames implements the AttrNames method of all types except struct
// and pointer (which may have fields too).
func methodNames(v reflect.Value) []string {
	var names []string
	names = appendMethodNames(names, v.Type())
	sort.Strings(names)
	return names
}

func appendMethodNames(names []string, t reflect.Type) []string {
	for i := 0; i < t.NumMethod(); i++ {
		names = append(names, t.Method(i).Name)
	}
	return names
}

func appendFieldNames(names []string, t reflect.Type) []string {
	if t.Kind() == reflect.Struct {
		// includes promoted fields
		for i := 0; i < t.NumField(); i++ {
			names = append(names, t.Field(i).Name)
		}
	}
	return names
}

func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Bool:
		return v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Complex64, reflect.Complex128:
		return v.Complex() == 0
	case reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice, reflect.Chan, reflect.Func:
		return !v.IsNil()
	case reflect.UnsafePointer:
		return v.Pointer() != 0 // avoid IsNil due to golang.org/issues/29381 (fixed in go1.13)
	case reflect.String:
		return v.Len() == 0
	case reflect.Array, reflect.Struct:
		return v == reflect.Zero(v.Type())
	}
	panic(v.Type())
}
