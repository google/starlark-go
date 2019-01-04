package stargo_test

import (
	"bytes"
	"fmt"
	"go/token"
	"log"

	"go.starlark.net/stargo"
	"go.starlark.net/starlark"
)

// ExampleValueOf shows how to pass a single Go value to a Starlark program,
// specifically token.NoPos, which is the zero constant of type token.Pos,
// a defined type whose underlying type is int, and which has a method, IsValid.
func ExampleValueOf() {
	predeclared := starlark.StringDict{
		"go":    stargo.Builtins,
		"NoPos": stargo.ValueOf(token.NoPos),
	}
	const src = `
print(NoPos)               # 0
print(type(NoPos))         # "go.int<token.Pos>"
print(NoPos.IsValid())     # false
print(go.int(NoPos))       # 0
print(type(go.int(NoPos))) # "int"
`
	thread := &starlark.Thread{
		Print: func(thread *starlark.Thread, msg string) { fmt.Println(msg) },
	}
	if _, err := starlark.ExecFile(thread, "valueof.star", src, predeclared); err != nil {
		if evalErr, ok := err.(*starlark.EvalError); ok {
			log.Fatal(evalErr.Backtrace())
		}
		log.Fatal(err)
	}

	// Output:
	//
	// 0
	// go.int<token.Pos>
	// False
	// 0
	// int
}

// ExampleVarOf demonstrates the use of stargo.VarOf to expose a Go
// variable (of type bytes.Buffer) to Starlark.
// The example calls this WriteString method,
// whose receiver type is *bytes.Buffer.
//
// A single variable is exposed twice, as both V and m.V.
// Though not strictly necessary, the latter form, using some kind of
// container such as a module or struct, is more typical.
// Also, the expression &m.V is legal, whereas &V is not.
func ExampleVarOf() {
	var V bytes.Buffer
	predeclared := starlark.StringDict{
		"go": stargo.Builtins,
		"m": &starlark.Module{
			Name: "m",
			Members: starlark.StringDict{
				"V": stargo.VarOf(&V),
			},
		},
		"V": stargo.VarOf(&V),
	}
	const src = `
print(type(m.V))                # "go.struct<bytes.Buffer>"
print(type(&m.V))               # "go.ptr<*bytes.Buffer>"
m.V.WriteString("hello")
print(m.V.String())             # "hello"
V.WriteString(", world")
print(V.String())               # "hello, world"
`
	thread := &starlark.Thread{
		Print: func(thread *starlark.Thread, msg string) { fmt.Println(msg) },
	}
	if _, err := starlark.ExecFile(thread, "varof.star", src, predeclared); err != nil {
		if evalErr, ok := err.(*starlark.EvalError); ok {
			log.Fatal(evalErr.Backtrace())
		}
		log.Fatal(err)
	}

	// Output:
	//
	// go.struct<bytes.Buffer>
	// go.ptr<*bytes.Buffer>
	// hello
	// hello, world
}
