# Tests of map operations.

load("assert.star", "assert")

# map types
tm = go.map_of(go.string, go.int)
assert.eq(str(tm), 'map[string]int')

# make_map
m = go.make_map(tm)
m["one"] = 1
m["two"] = 2
assert.eq(str(m), 'map[one:1 two:2]')
def f(): m[1] = "one"
assert.fails(f, 'invalid map key: cannot convert int to Go string')
def f(): m["one"] = "one"
assert.fails(f, 'invalid map element: cannot convert string to Go int')
m["one"] = 1.1 # TODO: dubious value truncation
assert.eq(str(m), 'map[one:1 two:2]')

# A go.map is an IterableMapping,
# so may be used in many places a dict is expected.
assert.eq(dict(m),  {"one": 1, "two": 2})
d = dict()
d.update(m)
assert.eq(d, {"one": 1, "two": 2})
def f(**kwargs): return kwargs
assert.eq(f(**m), d)

# m[k]
assert.eq(m["one"], 1) # found
assert.eq(m["two"], 2) # found
assert.fails(lambda: m["three"], 'key "three" not in go.map<map\[string\]int>')
assert.fails(lambda: tm()["three"], 'key "three" not in go.map<map\[string\]int>') # nil map
assert.fails(lambda: tm()[4], 'invalid map key: cannot convert int to Go string')

# iteration (order is nondeterministic hence sort)
assert.eq(sorted(m), ["one", "two"])
assert.eq(sorted(list(m)), ["one", "two"])
assert.eq(sorted([x for x in m]), ["one", "two"])

# delete
assert.fails(lambda: go.delete(m, 1), 'delete:.* int is not assignable to type string')
go.delete(m, "three") # missing key: no effect
go.delete(m, "two")
assert.eq([x for x in m], ["one"])
go.delete(tm(), "foo") # deletion from nil map => no effect

tmm = go.map_of(go.string, tm) # a map of maps
assert.eq(str(tmm), 'map[string]map[string]int')
assert.fails(lambda: go.map_of(tm, go.string), 'invalid key type')

# T(x) conversion
assert.eq(str(tm({"one": 1, "two": 2})), "map[one:1 two:2]") # from iterable mapping (e.g. dict)
assert.fails(lambda: tm({"one": 1, "two": "2"}), 'in map value, cannot convert string to Go int')
assert.eq(str(tm([("one", 1), ("two", 2)])), "map[one:1 two:2]") # from iterable of k/v-pairs
assert.fails(lambda: tm([("one", 1), ("two", "2")]), 'in map value, cannot convert string to Go int')
