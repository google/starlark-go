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
m.repeated_field = ["d", "e"]
assert.eq(list(m.repeated_field), ["d", "e"])
m.repeated_field.append("f")
assert.eq(list(m.repeated_field), ["d", "e", "f"])

assert.eq(dict(m.map_field), {"a": "A", "b": "B"})
m.map_field["c"] = "C"
assert.eq(dict(m.map_field), {"a": "A", "b": "B", "c": "C"})
