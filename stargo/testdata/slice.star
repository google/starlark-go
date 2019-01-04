# Tests of slices.

load("assert.star", "assert")

intslice = go.slice_of(go.int)
assert.eq(str(intslice), '[]int')

s = go.make_slice(intslice, 0, 5)
assert.eq(len(s), 0)
assert.eq(go.cap(s), 5)

s = go.make_slice(intslice, 5)
assert.eq(len(s), 5)
assert.eq(go.cap(s), 5)
assert.eq(list(s), [0, 0, 0, 0, 0])

for i in range(len(s)):
  s[i] = i+1
assert.eq(list(s), [1, 2, 3, 4, 5])

# slice elements are addressable.
ptr = &s[0]
assert.eq((*ptr), 1)
# TODO: go.ptr needs a way to express *ptr=x. Perhaps we should permit *x in the grammar, as an lvalue.

# go.slice
assert.fails(lambda: go.slice(s, 1), 'got 2 arguments, want at least 3') # both start and end are required
s2 = go.slice(s, 0, 5)
assert.eq(&s2[0], &s[0]) # the elements are aliased
assert.eq(len(s), 5)
assert.eq(go.cap(s), 5)

s2 = go.slice(s, 1, 5)
assert.eq(list(s2), [2, 3, 4, 5])
assert.eq(len(s2), 4)
assert.eq(go.cap(s2), 4)

s2 = go.slice(s, 0, 0) # empty slice
assert.eq(len(s2), 0)
assert.eq(go.cap(s2), 5)
s2 = go.append(s2, 6)
assert.eq(list(s2), [6])
assert.eq(list(s), [6, 2, 3, 4, 5])
s[0] = 1

# 3-index slice
s2 = go.slice(s, 0, 1, 2) # s[0:1:2]
assert.eq(len(s2), 1)
assert.eq(go.cap(s2), 2)
assert.eq(list(s2), [1])
assert.eq(list(s),  [1, 2, 3, 4, 5])
s2 = go.append(s2, 7)
assert.eq(list(s2), [1, 7])
assert.eq(list(s),  [1, 7, 3, 4, 5])
s2 = go.append(s2, 8)
assert.eq(list(s2), [1, 7, 8])
assert.eq(list(s),  [1, 7, 3, 4, 5]) # no longer aliases s2

# go.slice(iterable) conversion.
assert.eq(str(intslice([1, 2, 3])), '[1 2 3]')
assert.fails(lambda: intslice([1, 2, "3"]), 'cannot convert string to Go int')

# Slices are not sliceable using m[start:end:stride].
# This is intentional, to avoid confusion over the
# differences between Go's (alias) and Starlark's (copy) semantics.
assert.fails(lambda: s[:1], 'invalid slice operand')

# cap
assert.fails(lambda: go.cap(1), 'call of reflect.Value.Cap on int Value')

# hash
assert.fails(lambda: {s: 1}, 'unhashable: go.slice<\[\]int>')

# append
s = go.slice(s, 0, 0)
s2 = go.append(s, 1, 2, 3)
assert.eq(len(s2), 3)
assert.eq(len(s), 0) # original is unchanged
assert.eq(str(s2), '[1 2 3]')

s3 = go.append(s, -1)
assert.eq(len(s3), 1)
assert.eq(str(s3), '[-1]')
assert.eq(str(s2), '[-1 2 3]') # s3 aliases s2

# Go functions with a slice parameter accept a Starlark iterable and implicitly convert.
# But paradoxically, go.append is a Starlark function, and it requires a go.slice, as
# without one it cannot tell what type is slice to return.
assert.fails(lambda: go.append([], 1, 2, 3), 'append: want slice, got list')
assert.eq(list(go.append(intslice(), 1, 2, 3)), [1, 2, 3])
assert.eq(list(go.append(intslice([1, 2, 3]), 4, 5, 6)), [1, 2, 3, 4, 5, 6])

# go.slice type is iterable.
assert.eq([x for x in s2], [-1, 2, 3])

# variadic append: Starlark allows both positional and variadic arguments.
# The elements of *s2 are read prior to mutating the destination (also s2). [TODO: test].
s2 = go.append(s2, 4, *s2)
assert.eq(str(s2), '[-1 2 3 4 -1 2 3]')

# []byte
byteslice = go.slice_of(go.byte)
assert.eq('[]uint8', str(byteslice))
assert.eq(str(byteslice("hello")), '[104 101 108 108 111]') # analogue of []byte(string) conversion

assert.fails(lambda: go.append(byteslice(), "hello"), 'cannot convert string to Go uint8') # no analogue of append([]byte, string...)
assert.eq(str(go.append(byteslice(), *byteslice("hello"))), '[104 101 108 108 111]') # analogue of append([]byte, string...)

# copy
stringslice = go.slice_of(go.string) # []string
src = go.append(stringslice(), "X", "Y")
dst = go.append(stringslice(), "one", "two", "three", "four")
assert.eq(go.copy(dst, src), 2)
assert.eq(list(dst), ["X", "Y", "three", "four"])
