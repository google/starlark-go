
# Starlark in Go: Implementation

This document (a work in progress) describes some of the design
choices of the Go implementation of Starlark.

  * [Scanner](#scanner)
  * [Parser](#parser)
  * [Resolver](#resolver)
  * [Evaluator](#evaluator)
    * [Data types](#data-types)
    * [Freezing](#freezing)
  * [Testing](#testing)


## Scanner

The scanner is derived from Russ Cox's
[buildifier](https://github.com/bazelbuild/buildtools/tree/master/buildifier)
tool, which pretty-prints Bazel BUILD files.

Most of the work happens in `(*scanner).nextToken`.

## Parser

The parser is hand-written recursive-descent parser. It uses the
technique of [precedence
climbing](http://www.engr.mun.ca/~theo/Misc/exp_parsing.htm#climbing)
to reduce the number of productions.

In some places the parser accepts a larger set of programs than are
strictly valid, leaving the task of rejecting them to the subsequent
resolver pass. For example, in the function call `f(a, b=c)` the
parser accepts any expression for `a` and `b`, even though `b` may
legally be only an identifier. For the parser to distinguish these
cases would require additional lookahead.

## Resolver

The resolver reports structural errors in the program, such as the use
of `break` and `continue` outside of a loop.

Starlark has stricter syntactic limitations than Python. For example,
it does not permit `for` loops or `if` statements at top level, nor
does it permit global variables to be bound more than once.
These limitations come from the Bazel project's desire to make it easy
to identify the sole statement that defines each global, permitting
accurate cross-reference documentation.

In addition, the resolver validates all variable names, classifying
them as references to universal, global, local, or free variables.
Local and free variables are mapped to a small integer, allowing the
evaluator to use an efficient (flat) representation for the
environment.

Not all features of the Go implementation are "standard" (that is,
supported by Bazel's Java implementation), at least for now, so
non-standard features such as `set`
are flag-controlled.  The resolver reports
any uses of dialect features that have not been enabled.


## Evaluator

### Data types

<b>Integers:</b> Integers are representing using `big.Int`, an
arbitrary precision integer. This representation was chosen because,
for many applications, Starlark must be able to handle without loss
protocol buffer values containing signed and unsigned 64-bit integers,
which requires 65 bits of precision.

Small integers (<256) are preallocated, but all other values require
memory allocation. Integer performance is relatively poor, but it
matters little for Bazel-like workloads which depend much
more on lists of strings than on integers. (Recall that a typical loop
over a list in Starlark does not materialize the loop index as an `int`.)

An optimization worth trying would be to represent integers using
either an `int32` or `big.Int`, with the `big.Int` used only when
`int32` does not suffice. Using `int32`, not `int64`, for "small"
numbers would make it easier to detect overflow from operations like
`int32 * int32`, which would trigger the use of `big.Int`.

<b>Floating point</b>:
Floating point numbers are represented using Go's `float64`.
Again, `float` support is required to support protocol buffers. The
existence of floating-point NaN and its infamous comparison behavior
(`NaN != NaN`) had many ramifications for the API, since we cannot
assume the result of an ordered comparison is either less than,
greater than, or equal: it may also fail.

<b>Strings</b>:

TODO: discuss UTF-8 and string.bytes method.

<b>Dictionaries and sets</b>:
Starlark dictionaries have predictable iteration order.
Furthermore, many Starlark values are hashable in Starlark even though
the Go values that represent them are not hashable in Go: big
integers, for example.
Consequently, we cannot use Go maps to implement Starlark's dictionary.

We use a simple hash table whose buckets are linked lists, each
element of which holds up to 8 key/value pairs. In a well-distributed
table the list should rarely exceed length 1. In addition, each
key/value item is part of doubly-linked list that maintains the
insertion order of the elements for iteration.

<b>Struct:</b>
The `starlarkstruct` Go package provides a non-standard Starlark
extension data type, `struct`, that maps field identifiers to
arbitrary values. Fields are accessed using dot notation: `y = s.f`.
This data type is extensively used in Bazel, but its specification is
currently evolving.

Starlark has no `class` mechanism, nor equivalent of Python's
`namedtuple`, though it is likely that future versions will support
some way to define a record data type of several fields, with a
representation more efficient than a hash table.


### Freezing

All mutable values created during module initialization are _frozen_
upon its completion. It is this property that permits a Starlark module
to be referenced by two Starlark threads running concurrently (such as
the initialization threads of two other modules) without the
possibility of a data race.

The Go implementation supports freezing by storing an additional
"frozen" Boolean variable in each mutable object. Once this flag is set,
all subsequent attempts at mutation fail. Every value defines a
Freeze method that sets its own frozen flag if not already set, and
calls Freeze for each value that it contains.
For example, when a list is frozen, it freezes each of its elements;
when a dictionary is frozen, it freezes each of its keys and values;
and when a function value is frozen, it freezes each of the free
variables and parameter default values implicitly referenced by its closure.
Application-defined types must also follow this discipline.

The freeze mechanism in the Go implementation is finer grained than in
the Java implementation: in effect, the latter has one "frozen" flag
per module, and every value holds a reference to the frozen flag of
its module. This makes setting the frozen flag more efficient---a
simple bit flip, no need to traverse the object graph---but coarser
grained. Also, it complicates the API slightly because to construct a
list, say, requires a reference to the frozen flag it should use.

The Go implementation would also permit the freeze operation to be
exposed to the program, for example as a built-in function.
This has proven valuable in writing tests of the freeze mechanism
itself, but is otherwise mostly a curiosity.


### Fail-fast iterators

In some languages (such as Go), a program may mutate a data structure
while iterating over it; for example, a range loop over a map may
delete map elements. In other languages (such as Java), iterators do
extra bookkeeping so that modification of the underlying collection
invalidates the iterator, and the next attempt to use it fails.
This often helps to detect subtle mistakes.

Starlark takes this a step further. Instead of mutation of the
collection invalidating the iterator, the act of iterating makes the
collection temporarily immutable, so that an attempt to, say, delete a
dict element while looping over the dict, will fail. The error is
reported against the delete operation, not the iteration.

This is implemented by having each mutable iterable value record a
counter of active iterators. Starting a loop increments this counter,
and completing a loop decrements it. A collection with a nonzero
counter behaves as if frozen. If the collection is actually frozen,
the counter bookkeeping is unnecessary. (Consequently, iterator
bookkeeping is needed only while objects are still mutable, before
they can have been published to another thread, and thus no
synchronization is necessary.)

A consequence of this design is that in the Go API, it is imperative
to call `Done` on each iterator once it is no longer needed.

```
TODO
starlark.Value interface and subinterfaces
argument passing to builtins: UnpackArgs, UnpackPositionalArgs.
```

<b>Evaluation strategy:</b>
The evaluator uses a simple recursive tree walk, returning a value or
an error for each expression. We have experimented with just-in-time
compilation of syntax trees to bytecode, but two limitations in the
current Go compiler prevent this strategy from outperforming the
tree-walking evaluator.

First, the Go compiler does not generate a "computed goto" for a
switch statement ([Go issue
5496](https://github.com/golang/go/issues/5496)). A bytecode
interpreter's main loop is a for-loop around a switch statement with
dozens or hundreds of cases, and the speed with which each case can be
dispatched strongly affects overall performance.
Currently, a switch statement generates a binary tree of ordered
comparisons, requiring several branches instead of one.

Second, the Go compiler's escape analysis assumes that the underlying
array from a `make([]Value, n)` allocation always escapes
([Go issue 20533](https://github.com/golang/go/issues/20533)).
Because the bytecode interpreter's operand stack has a non-constant
length, it must be allocated with `make`. The resulting allocation
adds to the cost of each Starlark function call; this can be tolerated
by amortizing one very large stack allocation across many calls.
More problematic appears to be the cost of the additional GC write
barriers incurred by every VM operation: every intermediate result is
saved to the VM's operand stack, which is on the heap.
By contrast, intermediate results in the tree-walking evaluator are
never stored to the heap.

```
TODO
frames, backtraces, errors.
threads
Print
Load
```

## Testing

```
TODO
starlarktest package
`assert` module
starlarkstruct
integration with Go testing.T
```


## TODO


```
Discuss practical separation of code and data.
```
