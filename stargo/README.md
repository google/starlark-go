
# Stargo: Starlark bindings for Go packages

## Contents

  * [Overview](#overview)
  * [The API](#api)
    * [Values](#values)
      * [Conversions](#conversions)
    * [Types](#types)
    * [Variables](#variables)
  * [Builtins](#builtins)
  * [Packages](#packages)
  * [The `gen` tool](#gen-tool)
  * [Design Questions](#design-questions)

## Overview

The Stargo package provides Starlark wrappers for Go types, functions,
values, and variables. These wrappers satisfy all the relevant
Starlark value interfaces, enabling the elements of Go programs to be
easily and naturally manipulated by Starlark scripts.

Depending on its needs, an application may use Stargo to expose a
single Go value to Starlark, or several complete Go packages, or
anything in between.
For convenience, Stargo comes with a static code generation tool,
[`go.starlark.net/stargo/cmd/gen`](#gen-tool), that creates Starlark modules that
provide access to all the functions, types, variables, and constants
in one or more Go packages.

The `go.starlark.net/stargo/cmd/stargo` command is an example
Stargo-based application that provides a Starlark read-eval-print
loop (REPL) in which one may load Go packages such as `fmt`,
`encoding/json`, `net/http`, `go/token`, `go/ast`, `go/parser`,
`go/types`, `io/ioutil`, and `bytes`.

The example session below demonstrates how a Starlark script can use
the Go `net/http` package to make an HTTP request and inspect the
header and body of the response.

```
$ go run go.starlark.net/stargo/cmd/stargo
>>> load("go", http="net/http")
>>> resp, err = http.Get("https://go.starlark.net")
>>> print(err)
None
>>> type(resp)
"go.ptr<*http.Response>"
>>> print(dir(resp)) # show fields/methods of *http.Response
["Body", "Close", "ContentLength", "Cookies", "Header", ...]
>>>
>>> type(resp.Header)	# resp.Header is a Go map
"go.map<http.Header>"
>>> list(resp.Header)   # enumerate map keys
["Content-Type", "Accept-Ranges", "Date", "Etag", "X-Github-Request-Id", ...]
>>> resp.Header["Date"]
[Mon, 31 Dec 2018 04:49:45 GMT]
>>> dir(resp.Header)    # resp.Header also has methods
["Add", "Del", "Get", "Set", "Write", "WriteSubset"]
>>> type(resp.Header.Get)
"go.func<func(string) string>"
>>> type(resp.Header.Get("Date"))
"string"
>>> resp.Header.Get("Date")
"Mon, 31 Dec 2018 04:49:45 GMT"
>>>
>>> load("go", ioutil="io/ioutil")
>>> data, err = ioutil.ReadAll(resp.Body)
>>> print(err)
None
>>> go.string(data)[:100]
"<html>\n  <!-- This file will be served at go.starlark.net by GitHub pages. -->\n  <head>\n    <!-- "
```

Stargo was inspired by Nate Finch's [Starlight-go](https://npf.io/2018/12/starlight) project,
and attempts to improve upon it by solving some subtle issues related to variables and
aliasing by adding support for addressing to the Starlark compiler.


## The API

The stargo package presents a small API, consisting of only three concepts: `Value`, `Type`, and `Variable`.

```
package stargo

type Value interface {
	starlark.Value
	Reflect() reflect.Value
}
func ValueOf(x interface{}) starlark.Value

type Type struct{...}
func TypeOf(t reflect.Type) Type

func VarOf(ptr interface{}) starlark.Variable
```

### Values

A `stargo.Value` is a wrapper around a Go value (represented
internally by a `reflect.Value`) that satisfies the `starlark.Value`
interface, allowing it to be used as a value in a Starlark program
and accessed by the Go application into which it is embedded.

```
package stargo

type Value interface {
	starlark.Value
	Reflect() reflect.Value
}

func ValueOf(x interface{}) starlark.Value
```

To obtain the Starlark value for a Go value, call `stargo.ValueOf`.

Depending on the kind of the Go value, the concrete representation of
the `stargo.Value` may additionally satisfy further Starlark
interfaces so that it can be used with various operators.
For example, the stargo wrapper for a Go `func` value satisfies
`starlark.Callable`, so it may be called in an expression such as
`f()`; the stargo wrapper for a Go array or slice value satisfies
`starlark.Sequence`, so it may be indexed using `a[i]` or iterated
using a `for` loop or list comprehension; and the stargo wrapper for a Go
struct satisfies `starlark.HasAttrs` so that its fields may be
selected using `x.f` notation.

Not all Go values are wrapped.
For values of many of Go's predeclared basic types, `ValueOf` returns
a value of the corresponding core Starlark type:
Starlark `bool` for Go `bool`,
Starlark `string` for Go `string`,
Starlark `int` for all of Go's integer types,
and Starlark `float` for Go's floating-point types.
Values of Go's complex types `complex64` and `complex128` are wrapped,
as core Starlark does not have a data type for complex numbers.
[TODO: perhaps it should]
`ValueOf(nil)` returns starlark's `None` value.

User-defined types whose underlying type is `bool`, `string`, an
integer, or a floating-point number are treated differently from the
predeclared basic types,
because they may have methods and these methods must be accessible
from the corresponding Starlark value.

Consider the `token.Pos` type, from package `go/token`.
It is a defined type whose underlying type is `int`:

```
package token // import "go/token"

type Pos int

const NoPos Pos = 0

func (p Pos) IsValid() bool
```

As shown in the `ExampleValueOf` example program, we can introduce a
value of type `token.Pos` to the Starlark environment like so:

```go
	predeclared := starlark.StringDict{
		"go":    stargo.Builtins,
		"NoPos": stargo.ValueOf(token.NoPos),
	}
	starlark.ExecFile(thread, "valueof.star", src, predeclared)
```

The Starlark program may call methods of the Go value, or convert it
to an ordinary `int`:

```python
print(NoPos)               # 0
print(type(NoPos))         # "go.int<token.Pos>"
print(NoPos.IsValid())     # False
print(go.int(NoPos))       # 0
print(type(go.int(NoPos))) # int
````

Observe that the type of `NoPos` is not `int` but `go.int<token.Pos>`.
The angle-bracket notation is used by all the Stargo wrapper types to indicate
both the kind of the Go value (`int`), which determines its operators,
and the type of the Go value (`token.Pos`), which determines its methods.
There are similar Stargo wrapper types for `go.int8`, `go.uint16`, `go.float64`,
and all of Go's other predeclared numeric types.

To convert `NoPos` to an ordinary Starlark `int`, we call
`go.int(NoPos)`, which converts it to the regular (unnamed) Go type
`int`, whose values are not wrapped by Starlark.
Beware: calling `int(NoPos)` will fail.

[TODO: make Starlark support `int(x)` for arbitrary types?]

Stargo defines wrapper types for the following kinds of Go values.
Each type is identified by the string it returns from the Starlark `type(x)` function.

* `go.bool<T>`, `go.string<T>`, `go.int<T>`, and so on for each of Go's integer and floating-point kinds.
  These kinds are used only for user-defined types `T`, as illustrated by `go.int<token.Pos>`.
  For Go's predeclared types, `ValueOf` returns a core Starlark `bool`, `int`, `float`, or `string` value.
* `go.complex<T>`, for complex numbers.
  Supports comparison (NYI), hashing (NYI), and arithmetic (NYI).
  Builtins: `real`, `imag`, `complex`.
* `go.array<T>`, for arrays.
  Supports comparison, indexing operations (`a[i]` and `len(a)`), and hashing.
* `go.struct<T>`, for structs.
  Supports comparison (NYI), hashing (NYI), add field and method access (`x.f`).
* `go.chan<T>`, for channels.
  Supports comparison, iteration but not `len`, and hashing.
  Builtins: `make_chan`, `try_recv`, `recv`, `send`, `try_send`, `close`.
* `go.func<T>`, for functions and methods.
  Supports calling (`f()`), and hashing (as a pointer).
  [TODO Should it hash? Explain calls, panic, recover, stack, conversion]
* `go.ptr<T>`, for pointers.
  Supports comparison, hashing, field access and update (for a pointer to a struct),
  and indexing operations (for a non-nil pointer to an array).
* `go.unsafepointer<T>`, for `unsafe.Pointer` values.
  Supports comparison and hashing.
* `go.map<T>`, for maps.
  Supports map lookup and update (`m[k]`), iteration, and `len(m)`.
* `go.slice<T>`, for slices.
  Supports index access and update (`s[i]`), iteration, and `len(s)`.
  Does not support Starlark `s[start:end:stride]` slice operation,
  to avoid confusion with the Go slice operation; see `go.slice` built-in function.
* `go.type`, for Go values of type `reflect.Type`.
  Supports calling (`T()` and `T(x)`), comparison, hashing,
  and has all the methods of `reflect.Type`.

Go interface values do not need wrappers. An interface is a static
abstraction of a set of methods, but Starlark is a dynamic language,
so it has no need for static abstractions of type. Whenever any
expression yields a value of interface kind, the Stargo wrapper wraps
the payload of the interface; the interface type information is
discarded. One consequence is that the Starlark program may be able to
call methods that the corresponding Go program may not. Consider this function:

```go
func stdin() io.Reader { return os.Stdin }
```

The expression `stdin().Close()` would be rejected by the Go compiler
because `stdin()` has the type `io.Reader`, which has no `Close`
method, but the same expression is valid in Stargo because
`stdin()` returns a value of type `go.ptr<*os.File>`, which does have
a `Close` method.


```
TODO
- explain in more detail each of these types and their Starlark interfaces.
  May need subsections.
- None is not the same as typed nil, though it can be converted to it.
- all values (except `func`) print as if by `reflect.Value.String`.
```

## Conversions

```
TODO
- explain conversions: when they happen and what happens
 arguments to call to go
 results of go call to starlark
 map get, set, convert(dict)
 *array, *struct set operations
 slice a[i]=jjj
 slice convert
 append
 explicit T(x) conversion
 - None to (*T)(nil)
```

## Types

Types are first-class values in Stargo, just as they are in Go.

The `stargo.TypeOf` function returns a Starlark value that represents
a Go type, that is, an instance of `reflect.Type`.
The Starlark type of this value is `"go.type"`.

```
package stargo

type Type struct{...}

var _ Value = Type{}

func TypeOf(t reflect.Type) Type
```

Type values are most often needed when creating Go variables, maps,
slices and so on, by various built-in functions in the `go` module.

If you use [the `gen` tool](#gen-tool) to generate Stargo bindings for
a complete package, the name of the type is the same as its Go name,
for example `bytes.Buffer`, as in this example:

```
$ go run go.starlark.net/stargo/cmd/stargo
>>> load("go", "bytes")
>>>
>>> bytes.Buffer
bytes.Buffer
>>> type(bytes.Buffer)
"go.type"
>>> v = bytes.Buffer()			# create a value, like bytes.Buffer{} in Go
>>> p = go.new(bytes.Buffer)		# create a variable, as in Go.
>>>
>>> dir(bytes.Buffer)			# all the methods of reflect.Type
["Align", "AssignableTo", "Bits", "ChanDir", "Comparable", ...]
>>> bytes.Buffer.Kind()
struct
>>> T = bytes.Buffer
>>> [T.Method(i).Name for i in range(T.NumMethod())]	# bytes.Buffer has no methods
[]							
>>> T = go.ptr_to(bytes.Buffer)
>>> [T.Method(i).Name for i in range(T.NumMethod())]	# but *bytes.Buffer has many
["Bytes", "Cap", "Grow", "Len", "Next", "Read", "ReadByte", ...]
#
>>> dir(T())						# a simpler way to enumerate methods (and fields)
["Bytes", "Cap", "Grow", "Len", "Next", "Read", "ReadByte", ...]
```

```
TODO

Comparable, hashable.
Methods.
T() instantiates a T value
T(x) converts x to T. 
*T means go.ptr_to(T).
```


## Variables

The `VarOf` function, applied to the address of a Go variable,
returns a Starlark value that represents it.
As in Go, the resulting value acts as both a value and a variable,
depending on the context.

```
package stargo

func VarOf(ptr interface{}) starlark.Variable
```

The `ExampleVarOf` example program demonstrates a variable `V`, of type
`bytes.Buffer`, being introduced to a Starlark program:

```
	var V bytes.Buffer
	predeclared := starlark.StringDict{
		"V": stargo.VarOf(&V),
	}
	starlark.ExecFile(thread, "varof.star", src, predeclared)
```

The Starlark program may obtain the value of the variable, but for a
`bytes.Buffer` this is not useful as all its fields are
unexported. More importantly, one may call methods of the
variable---in other words, those defined with a receiver type of
`*Buffer`, such as `WriteString`:

```python
V.WriteString("hello")
print(V.String())          # "hello"
```

### About `starlark.Variable`

Go and Starlark differ in one important respect.
In Go, an expression consisting of variable names, array indexes, and
struct fields, such as `a[i].b[j].c`, is compiled differently when it
appears in an ordinary expression such as `a[i].b[j].c + 1` from how it is
compiled when it appears on the left-hand side of an assignment, such
as `a[i].b[j].c = 2`. In the latter case, instead of computing the value of
the expression, the compiler computes its _address_, then stores a new
value to the variable at that address, as if the user had written `ptr
= &a[i].b[j].c; *ptr = 2`.

By contrast, all core Starlark values are references, so a Starlark
compiler treats the statement as if the user had written `ref =
a[i].b[j]; ref.c = 2`, evaluating all but the last of the index and
field operations in the usual way, then finally applying the update
operation to the reference it yields.
Rewriting a Go statement in this way would change its behavior.
Consequently, the Go semantics for expressions of this form cannot be
implemented simply by defining new instances of `starlark.Value`:
at the moment the `a[i]` operation is computed, there is no way to know
whether its result will be needed as a value or as an address.
A change to the Starlark compiler is needed.

To support Stargo, the Starlark compiler now has an option,
`resolve.AllowAddressing`. When this option is enabled, sequences of
operations such as `a[i].b[j].c` are permitted to generate a value
that satisfies the `starlark.Variable` interface:

```go
package starlark

type Variable interface {
    Value
    Address() Value
    Value() Value
}
```

At the end of the sequence, the generated code will call the
variable's Address method if the expression's address is required, or
its Value method if it was computed for its value.
A wrapper value produced by Stargo satisfies the `Variable` interface
if its contains a reference to a Go variable:
its Address method returns a pointer to the variable,
and its Value method returns the contents of the variable.

The `AllowAddressing` option additionally allows Starlark programs to
use the unary `&` operator to obtain the address of an variable of the
form `&e.f` or `&e[i]`, where `e` is any expression.
The `&` operator has no other use in Starlark.
It is a static error to apply the `&` operator to any other form of
expression; in particular, `&v` cannot be used to obtain the address
of a named variable, even one produced by `stargo.VarOf`.
Partly for this reason, it is advisable to situate variables inside
some kind of collection such as a module or struct so their addresses
can be obtained using accessed as `&m.V` if necessary.

TODO: explain unary *ptr and *type, and that it is ambiguous wrt f(*args).


## Builtins

The `stargo.Builtins` module provides access to the built-in types, functions, and operations of Go.

An application should ensure that Starlark scripts have access to this
module, either through a load statement, or as predeclared value in
the code below:

```go
	predeclared := starlark.StringDict{
		"go":    stargo.Builtins,
	}
	starlark.ExecFile(thread, "filename.star", src, predeclared)
```

[TODO: we should probably standardize the way in which that happens]

The module, whose name is `"go"`, contains all the following Go types:

```
int      uint      uintptr       bool       
int8     uint8     float32       byte
int16    uint16    float64       error
int32    uint32    complex64     rune
int64    uint64    complex128    string  
```

plus the following functions:

* `new(T)` creates a variable of Go type T, and returns its address, a pointer.
* `typeof(x)` returns the Go type of the value x.
* `deref(ptr)` returns the value of the variable pointed to by ptr, a non-nil `go.ptr`.
* `make_map(map_type, cap=0)` returns a `go.map` that refers to a new map instance of the specified Go type and optional initial capacity `cap`.
* TODO: add `make_map_of(k, v, cap=0)`
* `make_slice(slice_type, len, cap=len)` returns a `go.slice` of the specified Go type, length, and optional capacity.
* `make_chan(chan_type, cap=0)` returns a `go.chan` that refers to a new channel instance of the specified Go type and optional capacity.
* `make_chan_of(elem_type, cap=0)` returns a `go.chan` that refers to a new channel instance of the specified element Go type and optional capacity.
   This is equivalent to but more convenient than `go.make_chan(go.chan_of(go.BothDir, elem_type), cap)`.
* `slice(slice, start, end [, cap])` performs the Go slice operation `slice[start:end]` or `slice[start:end:cap]`.
  Starlark also has a slice operator using this notation, but its behavior is incompatible with that of Go.
* `cap(x)` returns the capacity of an array, channel, or slice.
* `close(ch)` closes the channel chan.
* `complex(re, im)` returns a complex number of the specified real and imaginary components.
* `panic(x)` causes the current goroutine to panic.
* `delete(m, k)` deletes the entry `m[k]` from map m.
* `append(slice, *args)` appends the specified arguments to the slice and returns the resulting slice.
* `copy(dest, src)` copies elements from one slice to another.
* `real(cplx)` returns the real component of a complex number.
* `imag(cplx)` returns the imaginary component of a complex number.
* `send(ch, v)` sends the value v on the specified channel.
* `recv(ch)` returns a value received from the specified channel.
* `try_send()` acts like `send` but does not block.
* `try_recv()` acts like `recv` but does not block.
* `map_of(K, V)` returns the Go map type `map[K]V`.
* `array_of(n, T)` returns the Go array type `[n]T`.
* `slice_of(T)` returns the Go slice type `[]T`.
* `ptr_to(T)` return the Go type of the pointer `*T`.
* `chan_of(dir, elem_type)` returns the Go type of a channel of the specified element type and direction,
   which must be one of the following values of type `go.int<reflect.ChanDir>`:
  `BothDir` (`chan`),
  `RecvDir` (`<-chan`),
  or `SendDir` (`chan<-`).
* `func_of(in, out, variadic)` returns the type of a function
   of the specified input and output types (both iterables of `go.type`),
   and variadicity (a Boolean).

Consult the Go documentation for more details on each operation.


## Packages


```
- beware: stargo undermines many Starlark properties:
  programs may be nondeterministic (e.g. due to map iteration)
  programs may block (e.g. due to channels);
  programs may panic (though we attempt to catch them);
  values are not frozen, so data races are possible;
  the lvalue/rvalue concept becomes important.

- lvalue and rvalue modes, &v, and *ptr and the
  changes in the Starlark compiler and runtime.
  TODO unary *x is ambiguous wrt f(*args).

- API clients should not expect Index and Attr to work as usual.
  They must be followed by an UAMP or VALUE op.

- anonymous fields are promoted. (This creates potential ambiguity
  because names are unqualified so a type can have two methods or
  fields called f.)

- there is no reasonable interpretation we can give to Freeze, so I'm
  completely ignoring it. Caveat usor.

- when an expression retrieves a primitive value of a named type
  (e.g. token.Pos, which is a uint), we must preserve the namedness
  because it provides all the methods. (We don't just want to conver
  it to a uint.)  However, the primitive wrappers do not (yet)
  participate in arithmetic.  We could support that, but we should
  require that both operands of binary ops (+, etc) have the same
  type. Otherwise what does celcius + fahrenheit return?
  Named string types will not have the methods of string.

- don't use "if err" (truth value) because err is
  the concrete type, which could be zero. Use err != None.

- there are many checks done by the reflect package that we cannot
  hope to replicate exactly, so you should assume that more obscure
  mistakes can crash the program.  goFunc.CallInternal handles
  them, but many of the builtin functions do not.
```

## The `gen` tool

TODO

## Design questions

- equivalence relation for Go values (array and struct in particular)
    should it follow Go semantics or starlark?
    Go is easy. Starlark is hard.

- How do we implement hashing of Go values?
  If we compare using Go equivalence, we need access to the Go hash function.
  If we compare using Starlark equivalence, we need to recursively hash.
    primitives are done by starlark
    named basics could use the same hash
    pointers (ptr unsafe func chan) hash the address
    structs and arrays are simple: recursion. But unexported fields are problematic.
    slice and map are unhashable
    var unhashable?  

- perhaps the hasher API we're adding to Go should support all the basic types.
     
   


