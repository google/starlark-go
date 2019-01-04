package stargo

import (
	"fmt"
	"reflect"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// A goChan represents a Go value of kind chan.
//
// It implements Iterable (but not Sequence).
type goChan struct {
	v reflect.Value // kind=Chan; !CanAddr
}

var (
	_ Value               = goChan{}
	_ starlark.Comparable = goChan{}
	_ starlark.HasAttrs   = goChan{}
	_ starlark.Iterable   = goChan{}
)

func (c goChan) Attr(name string) (starlark.Value, error) { return method(c.v, name) }
func (c goChan) AttrNames() []string                      { return methodNames(c.v) }
func (c goChan) Freeze()                                  {} // unimplementable
func (c goChan) Hash() (uint32, error)                    { return ptrHash(c.v), nil }
func (c goChan) Iterate() starlark.Iterator               { return chanIterator{c.v} }
func (c goChan) Reflect() reflect.Value                   { return c.v }
func (c goChan) String() string                           { return str(c.v) }
func (c goChan) Truth() starlark.Bool                     { return c.v.IsNil() == false }
func (c goChan) Type() string                             { return fmt.Sprintf("go.chan<%s>", c.v.Type()) }

func (x goChan) CompareSameType(op syntax.Token, y starlark.Value, depth int) (bool, error) {
	return comparePtrs(op, x, y.(goChan))
}

type chanIterator struct{ v reflect.Value }

func (it chanIterator) Next(x *starlark.Value) bool {
	v, ok := it.v.TryRecv()
	if ok {
		*x = toStarlark(v)
	}
	return ok
}

func (it chanIterator) Done() {}

// -- builtins --

// make_chan(type, cap=0)
func go٠make_chan(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (res starlark.Value, err error) {
	var t Type
	cap := 0
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &t, &cap); err != nil {
		return nil, err
	}
	err = protect(thread, b.Name(), func() {
		res = toStarlark(reflect.MakeChan(t.t, cap))
	})
	return res, err
}

// make_chan_of(elem_type, cap=0)
func go٠make_chan_of(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (res starlark.Value, err error) {
	var elemType Type
	cap := 0
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &elemType, &cap); err != nil {
		return nil, err
	}
	err = protect(thread, b.Name(), func() {
		res = toStarlark(reflect.MakeChan(reflect.ChanOf(reflect.BothDir, elemType.t), cap))
	})
	return res, err
}

// recv(ch)
func go٠recv(ch interface{}) (interface{}, bool) {
	v, ok := reflect.ValueOf(ch).Recv()
	return v.Interface(), ok
}

// try_recv(ch)
func go٠try_recv(ch interface{}) (interface{}, bool) {
	v, ok := reflect.ValueOf(ch).TryRecv()
	if !v.IsValid() {
		return nil, ok
	}
	return v.Interface(), ok
}

// send(ch, v)
func go٠send(ch, v interface{}) { reflect.ValueOf(ch).Send(reflect.ValueOf(v)) }

// try_send(ch, v)
func go٠try_send(ch, v interface{}) bool { return reflect.ValueOf(ch).TrySend(reflect.ValueOf(v)) }

// close(chan)
func go٠close(ch interface{}) { reflect.ValueOf(ch).Close() }
