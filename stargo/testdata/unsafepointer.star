load("go", "unsafe", "stargo_test")
load("assert.star", "assert")

# package stargo_test
# type U struct { F float64 }

def float64bits(f):
    # create uptr, a *uint64 alias of &u.F, of type *float64.
    u = go.new(stargo_test.U)
    u.F = float(f)
    fptr = &u.F
    uptr = (*go.uint64)(unsafe.Pointer(fptr)) # ok
    return *uptr

assert.eq(float64bits(0), 0)
assert.eq(float64bits(1), 0x3ff0000000000000)
assert.eq(float64bits(1<<53), 0x4340000000000000)
