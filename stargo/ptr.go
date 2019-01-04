package stargo

import (
	"fmt"
	"reflect"
	"sort"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// A goPtr represents a Go value of kind pointer.
//
// If the variable it points to contains a struct, the struct's fields
// may be accessed or updated: p.f returns the value of that field, and
// p.f = x updates it.
//
// goPtr supports the unary * operator, but there is an ambiguity with
// the syntax for variadic function calls, f(*args). You must explicitly
// parenthesize the *ptr expression when using it as an argument in a
// call: f((*ptr))
//
// goPtrs are comparable, but they do not compare equal to None as that
// would violate symmetry: None does not compare equal to any other
// value. To test whether a pointer is valid, use its truth-value.
//
// See also: goArrayPtr, goUnsafePointer.
//
type goPtr struct {
	v reflect.Value // Kind=Ptr; !CanAddr
}

var (
	_ Value                = goPtr{}
	_ starlark.Comparable  = goPtr{}
	_ starlark.HasAttrs    = goPtr{}
	_ starlark.HasSetField = goPtr{}
	_ starlark.HasUnary    = goPtr{} // *ptr
)

func (p goPtr) Freeze()                {} // unimplementable
func (p goPtr) Hash() (uint32, error)  { return ptrHash(p.v), nil }
func (p goPtr) Reflect() reflect.Value { return p.v }
func (p goPtr) String() string         { return str(p.v) }
func (p goPtr) Truth() starlark.Bool   { return p.v.IsNil() == false }
func (p goPtr) Type() string           { return fmt.Sprintf("go.ptr<%s>", p.v.Type()) }

func (p goPtr) Attr(name string) (starlark.Value, error) {
	v := p.v

	// methods
	if m := v.MethodByName(name); m.IsValid() {
		return goFunc{m}, nil
	}

	// struct fields (including promoted ones)
	if v.Type().Elem().Kind() == reflect.Struct {
		if v.IsNil() {
			return nil, fmt.Errorf("nil dereference")
		}
		if f := v.Elem().FieldByName(name); f.IsValid() {
			if !f.CanInterface() {
				return nil, fmt.Errorf("access to unexported field .%s", name)
			}
			return varOf(f), nil // alias
		}
	}
	return nil, nil
}

func (p goPtr) AttrNames() []string {
	t := p.v.Type()

	var names []string
	names = appendMethodNames(names, t)
	names = appendFieldNames(names, t.Elem())
	sort.Strings(names)
	return names
}

func (p goPtr) SetField(name string, val starlark.Value) error {
	if p.v.IsNil() {
		return fmt.Errorf("nil dereference")
	}
	if elem := p.v.Elem(); elem.Kind() == reflect.Struct {
		if f := elem.FieldByName(name); f.CanSet() {
			x, err := toGo(val, f.Type())
			if err != nil {
				return err
			}
			f.Set(x)
			return nil
		}
	}
	return fmt.Errorf("can't set .%s field of %s", name, p.Type())
}

func (x goPtr) CompareSameType(op syntax.Token, y starlark.Value, depth int) (bool, error) {
	return comparePtrs(op, x, y.(goPtr))
}

func (p goPtr) Unary(op syntax.Token) (starlark.Value, error) {
	if op == syntax.STAR {
		if p.v.IsNil() {
			return nil, fmt.Errorf("nil pointer dereference")
		}
		return varOf(p.v.Elem()), nil
	}
	return nil, nil
}

// A goArrayPtr is a specialization of goPtr for non-nil pointers to
// values of kind array. Such pointers are iterable.
//
// A pointer to an array supports element index and update:
//
//   pa[i] = pa[j] + 1
//
// Nil array pointers are not iterable; they are handled by goPtr.
//
// There is no way for a single Go type to handle all pointers, because
// the Iterable.Index method, unlike its struct counterpart
// HasAttrs.Attr, does not return an error and cannot feasibly be
// changed to do so.
type goArrayPtr struct {
	v reflect.Value // Kind=Ptr, Elem.Kind=Array; !CanAddr; !IsNil
}

var (
	_ Value                = goArrayPtr{}
	_ starlark.Comparable  = goArrayPtr{}
	_ starlark.HasAttrs    = goArrayPtr{}
	_ starlark.HasSetIndex = goArrayPtr{}
	_ starlark.Indexable   = goArrayPtr{}
	_ starlark.Sequence    = goArrayPtr{}
	_ starlark.HasUnary    = goArrayPtr{} // *ptr
)

func (p goArrayPtr) Attr(name string) (starlark.Value, error) { return method(p.v, name) }
func (p goArrayPtr) AttrNames() []string                      { return methodNames(p.v) }
func (p goArrayPtr) Freeze()                                  {} // unimplementable
func (p goArrayPtr) Hash() (uint32, error)                    { return ptrHash(p.v), nil }
func (p goArrayPtr) Index(i int) starlark.Value               { return varOf(p.v.Elem().Index(i)) }
func (p goArrayPtr) Iterate() starlark.Iterator               { return &indexIterator{p.v.Elem(), 0} }
func (p goArrayPtr) Len() int                                 { return p.v.Type().Elem().Len() }
func (p goArrayPtr) Reflect() reflect.Value                   { return p.v }
func (p goArrayPtr) SetIndex(i int, y starlark.Value) error   { return setIndex(p.v.Elem(), i, y) }
func (p goArrayPtr) String() string                           { return str(p.v) }
func (p goArrayPtr) Truth() starlark.Bool                     { return true } // always non-nil
func (p goArrayPtr) Type() string                             { return fmt.Sprintf("go.ptr<%s>", p.v.Type()) }

func (x goArrayPtr) CompareSameType(op syntax.Token, y starlark.Value, depth int) (bool, error) {
	return comparePtrs(op, x, y.(goArrayPtr))
}

func (p goArrayPtr) Unary(op syntax.Token) (starlark.Value, error) {
	if op == syntax.STAR {
		return varOf(p.v.Elem()), nil
	}
	return nil, nil
}

// A goUnsafePointer represents a Go unsafe.Pointer value.
//
// goUnsafePointers are comparable. However, they do not compare equal to
// None (since that would violate symmetry). To test whether a pointer
// is valid, use bool(ptr).
type goUnsafePointer struct {
	v reflect.Value // Kind=UnsafePointer; !CanAddr
}

var (
	_ Value               = goUnsafePointer{}
	_ starlark.Comparable = goUnsafePointer{}
)

func (p goUnsafePointer) Freeze()                {} // immutable
func (p goUnsafePointer) Hash() (uint32, error)  { return ptrHash(p.v), nil }
func (p goUnsafePointer) Reflect() reflect.Value { return p.v }
func (p goUnsafePointer) String() string         { return str(p.v) }
func (p goUnsafePointer) Truth() starlark.Bool   { return isZero(p.v) == false }
func (p goUnsafePointer) Type() string           { return fmt.Sprintf("go.unsafepointer<%s>", p.v.Type()) }

func (x goUnsafePointer) CompareSameType(op syntax.Token, y starlark.Value, depth int) (bool, error) {
	return comparePtrs(op, x, y.(goUnsafePointer))
}

// Hash and Equal functions for chan, func, ptr, unsafepointer.
// Pointers, even of different types, compare equal if they point to the same object;
// reflect.Value equality does not have this property.
func ptrHash(x reflect.Value) uint32    { return uint32(x.Pointer()) }
func ptrsEqual(x, y reflect.Value) bool { return x.Pointer() == y.Pointer() }

func comparePtrs(op syntax.Token, x, y Value) (bool, error) {
	switch op {
	case syntax.EQL:
		return ptrsEqual(x.Reflect(), y.Reflect()), nil
	case syntax.NEQ:
		return !ptrsEqual(x.Reflect(), y.Reflect()), nil
	}
	return false, fmt.Errorf("invalid comparison: %s %s %s", x.Type(), op, y.Type())
}

// -- builtins --

// new(type)
func go٠new(t reflect.Type) interface{} { return reflect.New(t).Interface() }
