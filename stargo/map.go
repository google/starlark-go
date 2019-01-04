package stargo

import (
	"fmt"
	"reflect"

	"go.starlark.net/starlark"
)

// A goMap represents a Go value of kind map.
type goMap struct {
	v reflect.Value // kind=Map; !CanAddr
}

var (
	_ starlark.HasAttrs        = goMap{}
	_ starlark.Sequence        = goMap{}
	_ starlark.IterableMapping = goMap{}
	_ starlark.HasSetKey       = goMap{}
	_ Value                    = goMap{}
)

func (m goMap) Attr(name string) (starlark.Value, error) { return method(m.v, name) }
func (m goMap) AttrNames() []string                      { return methodNames(m.v) }
func (m goMap) Reflect() reflect.Value                   { return m.v }
func (m goMap) Freeze()                                  {} // unimplementable
func (m goMap) Hash() (uint32, error)                    { return 0, fmt.Errorf("unhashable: %s", m.Type()) }
func (m goMap) String() string                           { return str(m.v) }
func (m goMap) Truth() starlark.Bool                     { return m.v.IsNil() == false }
func (m goMap) Type() string                             { return fmt.Sprintf("go.map<%s>", m.v.Type()) }

func (m goMap) Iterate() starlark.Iterator { return mapIterator{m.v.MapRange()} }
func (m goMap) Len() int                   { return m.v.Len() }

type mapIterator struct {
	it *reflect.MapIter // Kind=Map; !CanAddr
}

func (it mapIterator) Done() {}

func (it mapIterator) Next(x *starlark.Value) bool {
	if it.it.Next() {
		*x = toStarlark(it.it.Key())
		return true
	}
	return false
}

func (m goMap) Items() []starlark.Tuple {
	n := m.v.Len()
	elems := make(starlark.Tuple, 2*n)
	items := make([]starlark.Tuple, 0, n)
	iter := m.v.MapRange()
	for iter.Next() {
		elems[0] = toStarlark(iter.Key())
		elems[1] = toStarlark(iter.Value())
		items = append(items, elems[0:2:2])
		elems = elems[2:]
	}
	return items
}

func (m goMap) Get(k starlark.Value) (v starlark.Value, found bool, err error) {
	t := m.v.Type()
	kv, err := toGo(k, t.Key())
	if err != nil {
		return nil, false, fmt.Errorf("invalid map key: %v", err)
	}
	if vv := m.v.MapIndex(kv); vv.IsValid() {
		found = true
		v = toStarlark(vv)
	}
	return
}

func (m goMap) SetKey(k, v starlark.Value) error {
	t := m.v.Type()
	kv, err := toGo(k, t.Key())
	if err != nil {
		return fmt.Errorf("invalid map key: %v", err)
	}
	vv, err := toGo(v, t.Elem())
	if err != nil {
		return fmt.Errorf("invalid map element: %v", err)
	}
	if m.v.IsNil() {
		return fmt.Errorf("assignment to element of nil map")
	}
	m.v.SetMapIndex(kv, vv)
	return nil
}

func mapConvert(iterable starlark.Iterable, mapType reflect.Type) (res reflect.Value, err error) {
	n := starlark.Len(iterable)
	if n < 0 {
		n = 0
	}
	m := reflect.MakeMapWithSize(mapType, n)

	set := func(k, v starlark.Value) error {
		// Convert Starlark key and value to Go.
		key, err := toGo(k, mapType.Key())
		if err != nil {
			return fmt.Errorf("in map key, %s", err)
		}
		value, err := toGo(v, mapType.Elem())
		if err != nil {
			return fmt.Errorf("in map value, %s", err)
		}
		m.SetMapIndex(key, value)
		return nil
	}

	iter := iterable.Iterate()
	defer iter.Done()
	if mapping, ok := iterable.(starlark.IterableMapping); ok {
		// iterable mapping (e.g. dict)

		var key starlark.Value
		for iter.Next(&key) {
			value, found, err := mapping.Get(key)
			if err == nil && !found {
				err = fmt.Errorf("internal error: %s does not contain key %s returned by its own iterator",
					mapping.Type(), key.Type())
			}
			if err != nil {
				return res, fmt.Errorf("in map key, %s", err)
			}
			if err := set(key, value); err != nil {
				return res, err
			}
		}
	} else {
		// iterable non-mapping: treat as a sequence of pairs, like dict.update.
		var pair starlark.Value
		for i := 0; iter.Next(&pair); i++ {
			iter2 := starlark.Iterate(pair)
			if iter2 == nil {
				return res, fmt.Errorf("dictionary update sequence element #%d is not iterable (%s)", i, pair.Type())
			}
			defer iter2.Done()
			len := starlark.Len(pair)
			if len < 0 {
				return res, fmt.Errorf("dictionary update sequence element #%d has unknown length (%s)", i, pair.Type())
			} else if len != 2 {
				return res, fmt.Errorf("dictionary update sequence element #%d has length %d, want 2", i, len)
			}
			var key, value starlark.Value
			iter2.Next(&key)
			iter2.Next(&value)
			if err := set(key, value); err != nil {
				return res, err
			}
		}
	}
	return m, nil
}

// -- builtins --

// make_map(type, cap=0)
func go٠make_map(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (res starlark.Value, err error) {
	var t Type
	var cap int
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &t, &cap); err != nil {
		return nil, err
	}
	if t.t.Kind() != reflect.Map {
		return nil, fmt.Errorf("%s: not a map type: %s", b.Name(), t)
	}
	err = protect(thread, b.Name(), func() {
		res = toStarlark(reflect.MakeMapWithSize(t.t, cap))
	})
	return res, err
}

// delete(map, key)
func go٠delete(m, key interface{}) {
	reflect.ValueOf(m).SetMapIndex(reflect.ValueOf(key), reflect.Value{})
}
