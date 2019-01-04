//go:generate go run go.starlark.net/stargo/cmd/gen -o pkgs.go fmt encoding/json net/http go/token go/ast go/parser go/types io/ioutil bytes

/*

The stargo command is a Starlark REPL in which you can load and
interact with Go packages.

Example:

    >>> load("go", http="net/http", ioutil="io/ioutil")
    >>> resp, err = http.Get("http://golang.org")
    >>> if err != None: ...
    >>> resp.Status
    "200 OK"
    >>> data, err = ioutil.ReadAll(resp.Body)
    >>> if err != None: ...
    >>> data = go.string(data)[:50]
    "<!DOCTYPE html>\n<html>\n<head>\n<meta http-equiv=\"Co"

Another example:

    >>> load("go", token="go/token", parser="go/parser")
    >>> fset = token.NewFileSet()
    >>> f, err = parser.ParseFile(fset, "hello.go", "package main; var x = 1", parser.Mode(0))
    >>> if err != None: ...
    >>> type(f)
    "go.ptr<*ast.File>"
    >>> f.Decls[0].Specs[0].Names
    [x]
    >>> type(f.Decls[0].Specs[0].Values[0])
    "go.ptr<*ast.BasicLit>"
    >>> type(f.Decls[0].Specs[0].Values[0].Name)
    >>> pos = f.Decls[0].Specs[0].Values[0].ValuePos
    >>> type(pos)
    "go.int<token.Pos>"
    >>> dir(pos)
    ["IsValid"]
    >>> fset.Position(pos)
    hello.go:1:23
    >>> fset.Position(pos).Filename
    "hello.go"
    >>> fset.Position(pos).Column
    23

Another, fmt:

    >>> load("go", fmt="fmt")
    >>> fmt.Printf("%s %q\n", "hello", "go")
    hello "go"
    (11, None)


Another, bytes:

    >>> load("go", "bytes")
    >>> b = go.new(bytes.Buffer)
    >>> type(b)
    "go.ptr<*bytes.Buffer>"
    >>> b.WriteString("hi")
    (2, None)
    >>> b.String()
    "hi"
    >>> b.WriteString("hi again")
    (8, None)
    >>> b.String()
    "hihi again"

    >>> b = bytes.Buffer() # note: no go.new(..)
    >>> type(b)
    "go.struct<bytes.Buffer>"
    >>> b
    {[] 0 0}               # no String method!
    >>> b.WriteString(b)
    Traceback (most recent call last):
      <stdin>:1: in <expr>
    Error: go.struct<bytes.Buffer> has no .WriteString field or method

*/
package main

import (
	"fmt"
	"log"

	"go.starlark.net/repl"
	"go.starlark.net/resolve"
	"go.starlark.net/stargo"
	"go.starlark.net/starlark"
)

func main() {
	log.SetPrefix("stargo: ")
	log.SetFlags(0)

	resolve.AllowFloat = true
	resolve.AllowLambda = true
	resolve.AllowNestedDef = true
	resolve.AllowBitwise = true
	resolve.AllowRecursion = true
	resolve.AllowGlobalReassign = true
	resolve.AllowAddressing = true // TODO: crucial! set within stargo?

	predeclared := starlark.StringDict{
		"go": stargo.Builtins,
	}
	thread := &starlark.Thread{
		Name: "REPL",
		Load: func(thread *starlark.Thread, module string) (starlark.StringDict, error) {
			if module != "go" {
				return nil, fmt.Errorf(`only load("go") is supported`)
			}
			return goPackages, nil
		},
	}

	repl.REPL(thread, predeclared)
}
