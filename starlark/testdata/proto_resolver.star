# Tests of the experimental 'lib/proto' module with a type resolver set
# on the thread. TestExecFileProtoResolver sets one built from the pool,
# so it knows the extensions of test.proto, unlike
# protoregistry.GlobalTypes, which proto.star tests.

load("assert.star", "assert")
load("proto.star", "proto")

schema = proto.file("test.proto")

m = schema.Test(string_field="A")
proto.set_field(m, schema.ext_string_field, "B")

# unmarshal resolves an extension with the resolver
m2 = proto.unmarshal(schema.Test, proto.marshal(m))
assert.eq(proto.has(m2, schema.ext_string_field), True)
assert.eq(proto.get_field(m2, schema.ext_string_field), "B")
assert.eq(m2.string_field, "A")

# unmarshal_text resolves an extension with the resolver
m3 = proto.unmarshal_text(schema.Test, '[go.starlark.net.testdata.ext_string_field]: "B"')
assert.eq(proto.has(m3, schema.ext_string_field), True)
assert.eq(proto.get_field(m3, schema.ext_string_field), "B")
