# Tests of functions.

load("assert.star", "assert")

tfunc = go.func_of([go.string, go.int], [go.string, go.error], False)
assert.eq(str(tfunc), "func(string, int) (string, error)")
assert.eq(type(tfunc), "go.type")
assert.fails(tfunc(), "call of nil function")

# func type methods
assert.eq(tfunc.NumIn(), 2)
assert.eq(tfunc.In(0), go.string)
assert.eq(tfunc.In(1), go.int)
assert.eq(tfunc.NumOut(), 2)
assert.eq(tfunc.Out(0), go.string)
assert.eq(tfunc.Out(1), go.error)
assert.eq(tfunc.IsVariadic(), False)

# Skylark-to-Go func conversion
assert.fails(lambda : tfunc(lambda a, b, c: None), "cannot convert 3-ary Starlark function lambda to func...")
assert.fails(lambda : tfunc(lambda a: None), "cannot convert 1-ary Starlark function lambda to func...")
f = tfunc(lambda a, b: res)
res = None
assert.fails(lambda : f("hi", 123), "cannot unpack NoneType into result of func...*string, error")
res = (1, 2)
assert.fails(lambda : f("hi", 123), "in result 1, cannot convert int to Go string")
res = ("hi", 1)
assert.fails(lambda : f("hi", 123), "in result 2, cannot convert int to Go error")
res = ("hi", None)
assert.eq(f("hi", 123), res)  # ok
res = ("hi", None, 1)
assert.fails(lambda : f("hi", 123), "too many results to unpack \(want 2\)")
res = ()
assert.fails(lambda : f("hi", 123), "too few results to unpack \(got 0, want 2\)")

# TODO: test all combinatons of n-ary results in m-ary contexts, with/without errors, and variadic.

load("go", "bytes", http = "net/http")

# Calling a Go function with implicit conversions.
h = go.make_map(http.Header)
h.Set("Content-Type", "text/html")
h.Set("Foo", "bar")
out = go.new(bytes.Buffer)
h.WriteSubset(out, {"Foo": True})  # implicit conversion of dict to map[string]bool
assert.eq(out.String(), "Content-Type: text/html\r\n")

load("go", "fmt")

# call of variadic Go function
assert.fails(fmt.Sprintf, "in call to fmt.Sprintf, got 0 arguments, want at least 1")
assert.eq("hello", fmt.Sprintf("hello"))
assert.eq("1", fmt.Sprintf("%d", 1))
assert.eq("1 2", fmt.Sprintf("%d %d", 1, 2))
assert.eq("1 2", fmt.Sprintf("%d %d", *[1, 2]))
assert.eq("1 2", fmt.Sprintf("%d %d", 1, *[2]))
