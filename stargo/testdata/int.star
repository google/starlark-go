# Tests of Go integer types
load("assert.star", "assert")

# int types
assert.eq('go.type', type(go.int))
assert.eq('go.type', type(go.int8))
assert.eq('go.type', type(go.int16))
assert.eq('go.type', type(go.int32))
assert.eq('go.type', type(go.int64))
assert.eq('int', str(go.int))
assert.eq('int8', str(go.int8))
assert.eq('int16', str(go.int16))
assert.eq('int32', str(go.int32))
assert.eq('int64', str(go.int64))
assert.ne(go.int64, go.int)
assert.eq(0, go.int(0)) # zero value
assert.eq(1, go.int(1))
assert.eq(go.typeof(1), go.typeof(go.int(1))) # fishy: when does it turn back into a Starlark int? wrap?
assert.eq(go.int, go.typeof(go.int(1.0)))

assert.eq(255, go.uint8(255))
assert.eq(1, go.uint8(257)) # truncation

# arithmetic, mixed types
# Q. how does this work??
assert.eq(9, go.int32(4) * go.int8(2) + go.float64(1.0))
assert.eq('int', type(go.int32(4) * go.int8(2)))
assert.eq('float', type(go.int32(4) * go.int8(2) + go.float64(1.0)))

# int type
assert.eq('go.type', type(go.int))
assert.eq('int', str(go.int))
assert.eq(0, go.int()) # zero value
assert.eq(123, go.int(123)) # conversion

# These values are converted to Starlark integers for the < comparison,
# which is why it permits even values of different numeric types.
assert.lt(go.int(4), go.int(5))
assert.lt(go.int32(4), go.int32(5))
assert.lt(go.int32(4), go.int64(5)) # note different types
assert.lt(go.int32(4), go.float64(5.0)) # note different types


# TODO: methods.
# TODO: comparisons. Truth value. Hashable.


load("go", "stargo_test")
assert.eq('stargo_test.myint16', str(stargo_test.myint16))
assert.eq('go.type', type(stargo_test.myint16))
x = stargo_test.myint16(123)
# "branded" integers don't behave like int.
assert.eq('123', str(x))
assert.ne(123, x) # not equal to any int!
assert.ne('stargo_test.myint16', type(x))
assert.eq('0', str(stargo_test.myint16())) # zero value
assert.fails(lambda: x+x, 'unknown binary op') # no arithmetic! TODO
assert.fails(lambda: int(x), 'cannot convert go.int16<stargo_test.myint16> to int') # no way to convert to int! TODO (how?)
# Branded primitives may have methods.
# A value has only the value methods:
assert.eq(['Get'], dir(x))
assert.eq(123, x.Get())
# But a pointer has all the methods:
y = go.new(stargo_test.myint16)
assert.eq('go.ptr<*stargo_test.myint16>', type(y))
assert.eq(['Get', 'Incr'], dir(y))
y.Incr()
y.Incr()
y.Incr()
assert.eq(3, y.Get())
# TODO: adding *expr into the grammar would create ambiguity with calls f(*args).
# assert.eq(3, (*y))



