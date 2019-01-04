# Tests of strings.

load("go", "fmt")
load("assert.star", "assert")

assert.eq("[1, 2, 3]", fmt.Sprintf("%s", [1, 2, 3]))
assert.eq("*starlark.List", fmt.Sprintf("%T", [1, 2, 3]))
assert.eq("1 2 3", fmt.Sprintf("%d %d %d", *[1, 2, 3]))

# Conversions of Starlark values to Go interfaces generally fail,
# except for fmt.Stringer, since every value has a String method.
# TODO: reconsider this, as it leaks implementation details:
# fmt.Stringer(1) changes an int to a starlark.Int.
assert.eq(type(fmt.Stringer), "go.type")
assert.eq(str(fmt.Stringer(1)), "1")
assert.eq(type(fmt.Stringer(1)), "go.struct<starlark.Int>")
