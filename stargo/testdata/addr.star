# Tests of addressing and aliasing issues.

load("assert.star", "assert")
load("go", "stargo_test")

# TODO: more tests:
# - promotion of anon fields
# - go.deref

# type A struct {
# 	P *B		-- pointer
# 	V B		-- value
# }
# type B struct {
# 	A [1]int
# }
# func  (B) V() {}
# func (*B) P() {}
#
# a = go.new(A)

a = go.new(stargo_test.A)
a.P = go.new(stargo_test.B)

assert.eq(type(a), 'go.ptr<*stargo_test.A>')

assert.eq(type(a.P), 'go.ptr<*stargo_test.B>')
assert.eq(dir(a.P), ['A', 'P', 'V']) # pointer methods too
assert.eq(type(&a.P), 'go.ptr<**stargo_test.B>')

assert.eq(type(a.P.A), 'go.array<[1]int>')
assert.eq(type(a.P.A[0]), 'int')
assert.eq(type(&a.P.A[0]), 'go.ptr<*int>')

a.P.A[0] = 1
assert.eq(a.P.A[0], 1)

a.V.A[0] = 2
assert.eq(a.V.A[0], 2)

a.V.A[0] += 1
assert.eq(a.V.A[0], 3)

assert.eq(type(a.V), 'go.struct<stargo_test.B>')
assert.eq(dir(a.V), ['A', 'V'])       # a.V is a var, but dir(a.V) gets a copy of the rvalue, so no pointer methods
assert.eq(a.V.P(), None)              # and yet a.V.P may be called
assert.eq(dir(&a.V), ['A', 'P', 'V']) # use dir(&x) to see the methods on the variable
assert.eq(type(a.V.A), 'go.array<[1]int>')

assert.eq(type(&a.V), 'go.ptr<*stargo_test.B>')
assert.eq(type(a.V.A[0]), 'int')
assert.eq(type(&a.V.A[0]), 'go.ptr<*int>')
assert.eq(a.V.A[0], 3)
ptrAVA = &a.V.A
ptrAVA[0] = 4
assert.eq(a.V.A[0], 4)

# Same tests, but this time b is an A value not a *A pointer:

b = *a

assert.eq(type(b), 'go.struct<stargo_test.A>')

assert.eq(type(b.P), 'go.ptr<*stargo_test.B>')
assert.eq(dir(b.P), ['A', 'P', 'V']) # pointer methods too
assert.fails(lambda: &b.P, 'go.ptr.* value has no address')

assert.eq(type(b.P.A), 'go.array<[1]int>')
assert.eq(type(b.P.A[0]), 'int')
assert.eq(type(&b.P.A[0]), 'go.ptr<*int>')

b.P.A[0] = 1
assert.eq(b.P.A[0], 1)

def f(): b.V.A[0] = 2
assert.fails(f, 'go.array.* value does not support item assignment') # implement SetIndex and report a more specific error?

assert.eq(type(b.V), 'go.struct<stargo_test.B>')
assert.eq(dir(b.V), ['A', 'V']) # just the value methods
assert.eq(type(b.V.A), 'go.array<[1]int>')

assert.fails(lambda: &b.V, 'go.struct.* value has no address')
assert.eq(type(b.V.A[0]), 'int')
assert.fails(lambda: &b.V.A[0], 'int value has no address')
assert.eq(b.V.A[0], 4)
assert.fails(lambda: &b.V.A, 'go.array.* value has no address')


# VarOf: stargo_test.V is a variable of type bytes.Buffer.


assert.eq(type(stargo_test.V), 'go.struct<bytes.Buffer>')
assert.eq(str(stargo_test.V), '{[] 0 0}') # .V yields the value of the variable, which has no .String method.
assert.eq(dir(stargo_test.V), ["buf", "lastRead", "off"]) # no methods, because .V yields a Buffer value.

assert.eq(type(&stargo_test.V), 'go.ptr<*bytes.Buffer>') # .V is addressable
assert.eq(str(&stargo_test.V), '') # the *Buffer pointer has a .String method
assert.contains(dir(&stargo_test.V), "WriteString") # the *Buffer pointer has pointer methods

# The variable has pointer methods.
assert.eq(stargo_test.V.WriteString("hello"), (5, None))
assert.eq(str(&stargo_test.V), 'hello')


# predeclared_var is another name for stargo_test.V, but the
# expression '&predeclared_var' is statically disallowed, so exposing
# a go.var to a Starlark program in this way is not particularly
# useful and is discouraged. Instead, expose it within a Package (like
# stargo_test) or struct or array so that it is accessed with x.f or
# a[i], which causes the compiler to apply the VALUE or ADDRESS
# operation appropriate to the context.

assert.eq(type(predeclared_var), 'go.var<bytes.Buffer>') # the go.var type is visible to users
assert.eq(str(predeclared_var), '{[104 101 108 108 111] 0 0}') # mostly it acts like its contents (an rvalue)
assert.contains(dir(predeclared_var), "buf")         # it has fields fields
assert.contains(dir(predeclared_var), "WriteString") # but it also has pointer methods, unlike a Buffer rvalue
assert.eq(predeclared_var.WriteString(", world"), (7, None))
assert.eq(stargo_test.V.String(), 'hello, world')

# TODO: add tests of indexing (read and set), mapping (read and set), and attrs (read and set) of v.F in
#   var V struct { F interface{} }
# where F contains concrete values of these types:
#   Go array, Go pointer-to-array, Go struct, Go pointer-to-struct, Go pointer-to-whatever, Go slice, Go string, Go map.
#   Starlark list, Starlark string, Starlark tuple, Starlark struct, Starlark dict.
