# Tests of Starlark 'struct' extension.
# This is not a standard feature and the Go and Starlark APIs may yet change.

load("assert.star", "assert")

assert.eq(str(struct), "<built-in function struct>")

# struct is a constructor for "unbranded" structs.
s = struct(host = "localhost", port = 80)
assert.eq(s, s)
assert.eq(s, struct(host = "localhost", port = 80))
assert.ne(s, struct(host = "localhost", port = 81))
assert.eq(type(s), "struct")
assert.eq(str(s), 'struct(host = "localhost", port = 80)')
assert.eq(s.host, "localhost")
assert.eq(s.port, 80)
assert.fails(lambda : s.protocol, "struct has no .protocol attribute")
assert.eq(dir(s), ["host", "port"])

# Use gensym to create "branded" struct types.
hostport = gensym(name = "hostport")
assert.eq(type(hostport), "symbol")
assert.eq(str(hostport), "hostport")

# Call the symbol to instantiate a new type.
http = hostport(host = "localhost", port = 80)
assert.eq(type(http), "struct")
assert.eq(str(http), 'hostport(host = "localhost", port = 80)')  # includes name of constructor
assert.eq(http, http)
assert.eq(http, hostport(host = "localhost", port = 80))
assert.ne(http, hostport(host = "localhost", port = 443))
assert.eq(http.host, "localhost")
assert.eq(http.port, 80)
assert.fails(lambda : http.protocol, "hostport struct has no .protocol attribute")

person = gensym(name = "person")
bob = person(name = "bob", age = 50)
alice = person(name = "alice", city = "NYC")
assert.ne(http, bob)  # different constructor symbols
assert.ne(bob, alice)  # different fields

hostport2 = gensym(name = "hostport")
assert.eq(hostport, hostport)
assert.ne(hostport, hostport2)  # same name, different symbol
assert.ne(http, hostport2(host = "localhost", port = 80))  # equal fields but different ctor symbols

# dir
assert.eq(dir(alice), ["city", "name"])
assert.eq(dir(bob), ["age", "name"])
assert.eq(dir(http), ["host", "port"])

# hasattr, getattr
assert.true(hasattr(alice, "city"))
assert.eq(hasattr(alice, "ageaa"), False)
assert.eq(getattr(alice, "city"), "NYC")

# +
assert.eq(bob + bob, bob)
assert.eq(bob + alice, person(age = 50, city = "NYC", name = "alice"))
assert.eq(alice + bob, person(age = 50, city = "NYC", name = "bob"))  # not commutative! a misfeature
assert.fails(lambda : alice + 1, "struct \+ int")
assert.eq(http + http, http)
assert.fails(lambda : http + bob, "different constructors: hostport \+ person")
