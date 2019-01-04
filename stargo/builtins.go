package stargo

import (
	"reflect"

	"go.starlark.net/starlark"
)

// Builtins is a module, typically predeclared under the name "go",
// that provides access to all the Go builtin functions and types.
// Because reflect.Type is effectively a built-in type in stargo,
// many of the methods of reflect.Type are included here too.
//
var Builtins = &starlark.Module{
	Name: "go",
	Members: starlark.StringDict{
		// We use an Arabic zero '٠' (U+0660) in these function names.

		// built-in functions
		"cap":     ValueOf(go٠cap),
		"close":   ValueOf(go٠close),
		"complex": ValueOf(go٠complex),
		"new":     ValueOf(go٠new),
		"panic":   ValueOf(go٠panic),
		"typeof":  ValueOf(reflect.TypeOf),

		// map
		"make_map": starlark.NewBuiltin("make_map", go٠make_map),
		"delete":   ValueOf(go٠delete),

		// slice
		"make_slice": starlark.NewBuiltin("make_slice", go٠make_slice),
		"append":     starlark.NewBuiltin("append", go٠append),
		"copy":       ValueOf(go٠copy),
		"slice":      starlark.NewBuiltin("slice", go٠slice),

		// complex
		"real": ValueOf(go٠real),
		"imag": ValueOf(go٠imag),

		// chan
		"make_chan":    starlark.NewBuiltin("make_chan", go٠make_chan),
		"make_chan_of": starlark.NewBuiltin("make_chan_of", go٠make_chan_of),
		"send":         ValueOf(go٠send),
		"recv":         ValueOf(go٠recv),
		"try_recv":     ValueOf(go٠try_recv),
		"try_send":     ValueOf(go٠try_send),
		"ChanDir":      TypeOf(reflect.TypeOf(reflect.ChanDir(0))),
		"BothDir":      ValueOf(reflect.BothDir),
		"RecvDir":      ValueOf(reflect.RecvDir),
		"SendDir":      ValueOf(reflect.SendDir),

		// type constructors
		"map_of":   ValueOf(reflect.MapOf),
		"array_of": ValueOf(reflect.ArrayOf),
		"slice_of": ValueOf(reflect.SliceOf),
		"ptr_to":   ValueOf(reflect.PtrTo),
		"chan_of":  ValueOf(reflect.ChanOf),
		"func_of":  ValueOf(reflect.FuncOf),

		// built-in types
		"bool":       TypeOf(reflect.TypeOf(bool(false))),
		"int":        TypeOf(reflect.TypeOf(int(0))),
		"int8":       TypeOf(reflect.TypeOf(int8(0))),
		"int16":      TypeOf(reflect.TypeOf(int16(0))),
		"int32":      TypeOf(reflect.TypeOf(int32(0))),
		"int64":      TypeOf(reflect.TypeOf(int64(0))),
		"uint":       TypeOf(reflect.TypeOf(uint(0))),
		"uint8":      TypeOf(reflect.TypeOf(uint8(0))),
		"uint16":     TypeOf(reflect.TypeOf(uint16(0))),
		"uint32":     TypeOf(reflect.TypeOf(uint32(0))),
		"uint64":     TypeOf(reflect.TypeOf(uint64(0))),
		"uintptr":    TypeOf(reflect.TypeOf(uintptr(0))),
		"rune":       TypeOf(reflect.TypeOf(rune(0))),
		"byte":       TypeOf(reflect.TypeOf(byte(0))),
		"float32":    TypeOf(reflect.TypeOf(float32(0))),
		"float64":    TypeOf(reflect.TypeOf(float64(0))),
		"complex64":  TypeOf(reflect.TypeOf(complex64(0))),
		"complex128": TypeOf(reflect.TypeOf(complex128(0))),
		"string":     TypeOf(reflect.TypeOf("")),
		"error":      TypeOf(reflect.TypeOf(new(error)).Elem()),
	},
}

// panic(any)
func go٠panic(x interface{}) { panic(x) }
