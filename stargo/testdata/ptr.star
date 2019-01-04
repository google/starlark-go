# Tests of pointers.

load("assert.star", "assert")

tarray = go.array_of(5, go.int)
assert.eq(str(tarray), '[5]int')

p = go.new(tarray)
assert.eq(type(p), 'go.ptr<*[5]int>')
assert.true(p)
for i in range(len(p)):
  for j in range(i, len(p)):
     p[j] += j+1
assert.eq(list(p), [1, 4, 9, 16, 25])

# A non-nil *go.array is a sequence.
p = go.new(tarray)
assert.eq(type(p), 'go.ptr<*[5]int>')
assert.eq(str(p), '&[0 0 0 0 0]')
assert.true(p)
assert.eq(len(p), 5)
assert.eq(list(p), [0, 0, 0, 0, 0])
p[0] = 123
assert.eq(list(p), [123, 0, 0, 0, 0])
# Conversion to slice is not permitted. Should it be?
intslice = go.slice_of(go.int)
assert.fails(lambda: intslice(p), 'cannot convert go.ptr<\*\[5\]int> to Go \[\]int')

# A nil *go.array is an empty sequence.
p = go.ptr_to(tarray)() # creates zero (nil) value of type *[5]int
assert.eq(type(p), 'go.ptr<*[5]int>')
assert.eq(str(p), '<nil>')
assert.true(not p)
assert.ne(p, None) # nil pointer is not None
assert.fails(lambda: len(p), 'has no len')
assert.fails(lambda: list(p), 'got go.ptr<\*\[5\]int>, want iterable')
assert.fails(lambda: p[0], 'unhandled index operation')
def f(): p[0] = 1
assert.fails(f, 'does not support item assignment')

# *T is a shortcut for go.ptr_to(T).
tarrayptr = *tarray
assert.eq(tarrayptr, go.ptr_to(tarray))
assert.eq(str(tarray), "[5]int")
assert.eq(str(tarrayptr),"*[5]int")

# *ptr dereferences a pointer.
assert.fails(lambda: *p, 'nil pointer dereference')

# But beware: it's ambiguous w.r.t. a variadic f(*args) call,
# so the explicit parens are required in the calls below.
p = go.new(go.array_of(2, go.int))
q = go.new(go.array_of(2, go.int))
assert.eq(str((*p)), "[0 0]") # here
assert.eq((*p), (*q))         # and here
# Most pointers are not iterable so the call
# immediately fails:
pint = go.new(go.int)
assert.fails(lambda: str(*pint), 'argument after \* must be iterable, not go.ptr<\*int>')
# But array pointers are quite subtle:
# below, *p is is the sequence (0, 0),
# so the call is equivalent to assert.eq(0, 0)!
assert.eq(*p)

# *ptr may be used as an lvalue.
assert.eq((*pint), 0)
*pint = 1
assert.eq((*pint), 1)
def f(): *pint = "two"
assert.fails(f, 'cannot set go.var<int>: cannot convert string to Go int')
y = *pint # y is a copy of *pint, not an alias
y = 2
assert.eq((*pint), 1)

# Taking the address of an array element.
p = go.new(go.array_of(3, go.int))
elem = &p[1]
*elem = 123
assert.eq(list(p), [0, 123, 0])
