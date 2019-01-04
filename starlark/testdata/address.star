# Minimal tests of addressing operators.
# See Stargo for more comprehensive tests,
# including variables of aggregate (struct/array) type.
# option:addressing

load("assert.star", "assert")

assert.eq(addr.v, None)
ptr = &addr.v
assert.eq(type(ptr), "pointer")
*ptr = 2
assert.eq(addr.v, 2)
assert.eq((*ptr), 2) # parens are required
