# Tests of the experimental 'lib/proto' module.

load("assert.star", "assert")
load("proto.star", "proto")

schema = proto.file("test.proto")

m = schema.Test(
    string_field="I'm a string!",
    int32_field=42,
    repeated_field=["a", "b", "c"],
    map_field={"a": "A", "b": "B"}
)
assert.eq(type(m), "proto.Message")
assert.eq(type(m.repeated_field), "proto.repeated<string>")
assert.eq(type(m.map_field), "proto.map<string, string>")

assert.eq(m.string_field, "I'm a string!")
assert.eq(m.int32_field, 42)

assert.eq(list(m.repeated_field), ["a", "b", "c"])
assert.eq(len(m.repeated_field), 3)
m.repeated_field = ["d", "e"]
assert.eq(len(m.repeated_field), 2)
assert.eq(list(m.repeated_field), ["d", "e"])
m.repeated_field.append("f")
assert.eq(list(m.repeated_field), ["d", "e", "f"])
assert.eq(len(m.repeated_field), 3)

assert.eq(dict(m.map_field), {"a": "A", "b": "B"})
assert.eq(len(m.map_field), 2)
m.map_field["c"] = "C"
assert.eq(dict(m.map_field), {"a": "A", "b": "B", "c": "C"})
assert.eq(len(m.map_field), 3)

m.map_field = {"d": "D", "e": "E"}
assert.eq(dict(m.map_field), {"d": "D", "e": "E"})

m.map_field = None
assert.eq(dict(m.map_field), {})

# list ordering of keys
m.map_field = {"a": "A", "b": "B", "c": "C"}
assert.eq(list(m.map_field), ["a", "b", "c"])

# str
assert.eq(str(m.map_field), '{"a": "A", "b": "B", "c": "C"}')

# bool
assert.eq(bool(m.map_field), True)
assert.eq(bool(schema.Test().map_field), False)

# presence checks
found = "a" in m.map_field
assert.eq(found, True)
not_found = "z" in m.map_field
assert.eq(not_found, False)

# type checking
def _assign_bad_key():
  m.map_field[1] = "X"
assert.fails(_assign_bad_key, "converting map key: got int, want string")
def _assign_bad_value():
  m.map_field["a"] = 1
assert.fails(_assign_bad_value, "converting map value: got int, want string")

# not hashable
assert.fails(lambda: {m.map_field: 1}, "unhashable")

# Extensions

assert.eq(str(schema.ext_string_field), "go.starlark.net.testdata.ext_string_field")
assert.eq(type(schema.ext_string_field), "proto.FieldDescriptor")
assert.eq(dir(schema.ext_string_field), [])

m2 = schema.Test(string_field="A")
assert.eq(proto.has(m2, schema.ext_string_field), False)
proto.set_field(m2, schema.ext_string_field, "B")
assert.eq(proto.has(m2, schema.ext_string_field), True)
assert.eq(proto.get_field(m2, schema.ext_string_field), "B")
