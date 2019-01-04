// Package stargo provides Starlark bindings for Go values, variables,
// and types, allowing you to write Starlark scripts that interact with
// the full richness of Go functions and data types.
//
// See the cmd/stargo subdirectory for an example application.
//
// Stargo was inspired by Nate Finch's work on Starlight-go.
//
// THIS IS ALL EXPERIMENTAL AND MAY CHANGE.
// In particular, the way in which Go packages are loaded
// from Starlark needs more thought.
//
package stargo

import (
	"reflect"

	"go.starlark.net/starlark"
)

// A Value is a Starlark value that wraps a Go value using reflection.
type Value interface {
	starlark.Value
	Reflect() reflect.Value
}

// ValueOf returns a Starlark value corresponding to the Go value x.
func ValueOf(x interface{}) starlark.Value {
	return toStarlark(reflect.ValueOf(x))
}

// VarOf returns a Starlark value that wraps the Go variable *ptr.
// VarOf panics if ptr is not a pointer.
func VarOf(ptr interface{}) starlark.Variable {
	v := reflect.ValueOf(ptr)
	if v.Kind() != reflect.Ptr {
		panic("VarOf: not a pointer")
	}
	return varOf(v.Elem())
}

// TypeOf returns a Starlark value that represents the specified non-nil type.
//
// The expression TypeOf(reflect.TypeOf(new(T)).Elem()) works for any
// type T, including interface types.
func TypeOf(t reflect.Type) Type {
	// TODO: should we add a convenience helper TypeOfNew(new(T))?
	if t == nil {
		panic("TypeOf(nil)")
	}
	return Type{t}
}
