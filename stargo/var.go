package stargo

import (
	"fmt"
	"log"
	"reflect"
	"sort"

	"go.starlark.net/starlark"
)

// A goVar represents a reference to a Go variable.
//
// A goVar is transiently produced by l-mode ATTR and INDEX operations.
// It is the only kind of value that has a CanAddr (l-mode) reflect.Value.
// It is always consumed by a SETFIELD, SETINDEX, or ADDR operation,
// so it has a degenerate set of methods.
// It acts like a pointer for all operations but &x.
//
// goVar is split into three types:
// goIndexableVar, for indexable values that support a[i] where 0 <= i < Len;
// goMappingVar, for values that support m[k] for arbitrary values k; and
// goVar, for all other values.
// This separation is required because there is no way to dynamically
// support all three in a single type, as the getIndex operation in
// the Starlark interpreter dispatches to either the Mapping or the
// Indexable interface, but not both.
// By contrast, all vars may implement HasSetField because unlike the
// Index method, the SetField method is allowed to fail dynamically.
//
type goVar struct {
	v reflect.Value // CanAddr, thus kind may be Interface
}

var (
	_ Value                = goVar{}
	_ starlark.HasAttrs    = goVar{}
	_ starlark.HasSetField = goVar{}
	_ starlark.Variable    = goVar{}
)

// A goIndexableVar represents a Go variable whose Value() is indexable.
//
// Its reflect.Value.Kind may be String, Array, Slice, Ptr (if a
// non-nil pointer to an array), or an Interface containing one of
// those types, or any concrete or interface value that implements
// starlark.Indexable.
type goIndexableVar struct{ goVar }

var (
	_ Value                = goIndexableVar{}
	_ starlark.HasAttrs    = goIndexableVar{}
	_ starlark.HasSetField = goIndexableVar{}
	_ starlark.HasSetIndex = goIndexableVar{}
	_ starlark.Indexable   = goIndexableVar{}
	_ starlark.Variable    = goIndexableVar{}
)

// A goMappingVar represents a Go variable whose Value() is a mapping.
//
// Its reflect.Value.Kind may be Map, or an Interface containing a
// Map, or any concrete or interface value that implements starlark.Mapping.
type goMappingVar struct{ goVar }

var (
	_ Value             = goMappingVar{}
	_ starlark.HasAttrs = goMappingVar{}
	_ starlark.Mapping  = goMappingVar{}
	_ starlark.Variable = goMappingVar{}
)

// varOf returns a Starlark variable that represents the given Go variable.
func varOf(v reflect.Value) starlark.Variable {
	if !v.CanAddr() {
		log.Panicf("not a variable: %v", v)
	}
	if !v.CanInterface() {
		log.Panicf("unexported field: %v", v)
	}

	variable := goVar{v}

	// Go mapping or indexable value?
	switch conc := variable.concrete(); conc.Kind() {
	case reflect.Map:
		return goMappingVar{variable}
	case reflect.String, reflect.Array, reflect.Slice:
		return goIndexableVar{variable}
	case reflect.Ptr:
		if !conc.IsNil() && conc.Elem().Kind() == reflect.Array {
			// non-nil pointer-to-array
			return goIndexableVar{variable}
		}
	}

	// Starlark mapping or indexable value?
	switch v.Interface().(type) {
	case starlark.Mapping:
		return goMappingVar{variable}
	case starlark.Indexable:
		return goIndexableVar{variable}
	}

	return variable
}

// concrete returns the concrete value held by the variable.
// The result may be an rvalue or an lvalue.
func (v goVar) concrete() reflect.Value {
	if v.v.Kind() == reflect.Interface {
		return v.v.Elem()
	}
	return v.v
}

func (v goVar) Address() starlark.Value {
	addr := v.v.Addr()
	if v.v.Kind() == reflect.Array {
		// Invariant: addr is non-nil.
		return goArrayPtr{addr}
	} else {
		return goPtr{addr}
	}
}
func (v goVar) Value() starlark.Value { return toStarlark(copyVal(v.v)) }

func (v goVar) SetValue(val starlark.Value) error {
	if !v.v.CanSet() {
		return fmt.Errorf("cannot set %s", v.Type())
	}
	x, err := toGo(val, v.v.Type())
	if err != nil {
		return fmt.Errorf("cannot set %s: %v", v.Type(), err)
	}
	v.v.Set(x)
	return nil
}

func (v goVar) Freeze()                {} // unimplementable
func (v goVar) Hash() (uint32, error)  { return 0, fmt.Errorf("unhashable: %s", v.Type()) }
func (v goVar) Len() int               { return starlark.Len(v.Value()) } // delegate to value
func (v goVar) Reflect() reflect.Value { return v.v }
func (v goVar) String() string         { return str(v.v) }
func (v goVar) Truth() starlark.Bool   { return true }
func (v goVar) Type() string           { return fmt.Sprintf("go.var<%s>", v.v.Type()) }

func (v goVar) AttrNames() []string {
	var names []string

	// If var's static type is interface, look inside it.
	t := v.v.Type()
	if t.Kind() == reflect.Interface {
		elem := v.v.Elem()
		if !elem.IsValid() {
			return nil // interface contains nothing
		}
		t = elem.Type()

		// methods of the value
		names = appendMethodNames(names, t)
	} else {
		// methods of the address
		names = appendMethodNames(names, reflect.PtrTo(t))
	}

	// fields of struct or *struct value
	switch t.Kind() {
	case reflect.Struct:
		names = appendFieldNames(names, t)
	case reflect.Ptr:
		names = appendFieldNames(names, t.Elem())
	}

	sort.Strings(names)
	return names
}

func (v goVar) SetField(name string, val starlark.Value) error {
	// If variable has type struct, delegate to its address (a *struct).
	if v.v.Kind() == reflect.Struct {
		return goPtr{v.v.Addr()}.SetField(name, val)
	}

	// If variable contains a HasSetField, delegate to value.
	if hasSetField, ok := v.v.Interface().(starlark.HasSetField); ok {
		return hasSetField.SetField(name, val)
	}

	return fmt.Errorf("cannot set .%s field of %s", name, v.Type())
}

func (gv goVar) Attr(name string) (starlark.Value, error) {
	v := gv.v

	// methods
	if m := v.MethodByName(name); m.IsValid() {
		return goFunc{m}, nil
	}

	// pointer methods
	if m := v.Addr().MethodByName(name); m.IsValid() {
		return goFunc{m}, nil
	}

	// struct fields (including promoted ones)
	if v.Type().Kind() == reflect.Struct {
		if f := v.FieldByName(name); f.IsValid() {
			if !f.CanInterface() {
				return nil, fmt.Errorf("access to unexported field .%s", name)
			}
			return varOf(f), nil // alias
		}
	}

	// indirect struct fields (if var holds a *struct)
	if v.Type().Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil, fmt.Errorf("nil dereference")
		}
		elem := v.Elem()
		if elem.Type().Kind() == reflect.Struct {
			if f := elem.FieldByName(name); f.IsValid() {
				if !f.CanInterface() {
					return nil, fmt.Errorf("access to unexported field .%s", name)
				}
				return varOf(f), nil // alias
			}
		}
	}

	// If value in variable has attributes (e.g. starlark struct), delegate to it.
	if hasAttrs, ok := gv.Value().(starlark.HasAttrs); ok {
		return hasAttrs.Attr(name)
	}

	return nil, nil
}

func (v goIndexableVar) Index(i int) starlark.Value {
	// Precondition: 0 <= i < Len()

	// Indexable Go variable?
	switch conc := v.concrete(); conc.Kind() {
	case reflect.Array, reflect.Slice:
		return varOf(v.v.Index(i))
	case reflect.Ptr:
		if !conc.IsNil() && conc.Elem().Kind() == reflect.Array {
			// non-nil pointer-to-array
			return varOf(conc.Elem().Index(i))
		}
	}

	// Delegate to indexable value.
	return v.Value().(starlark.Indexable).Index(i)
}

func (v goIndexableVar) SetIndex(i int, y starlark.Value) error {
	// Precondition: 0 <= i < Len()

	// Variable of type array?
	if v.v.Kind() == reflect.Array {
		return setIndex(v.v, i, y)
	}

	// Variable holds HasSetIndex value? Delegate to value.
	// This case covers slice and non-nil-ptr-to-array.
	if hasSetIndex, ok := v.Value().(starlark.HasSetIndex); ok {
		return hasSetIndex.SetIndex(i, y)
	}

	return fmt.Errorf("can't set index %d of %s", i, v.Type())
}

func (v goMappingVar) Get(k starlark.Value) (starlark.Value, bool, error) {
	return v.Value().(starlark.Mapping).Get(k) // delegate to value
}
