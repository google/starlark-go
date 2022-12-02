# Tests of the experimental 'lib/proto' module.

load("assert.star", "assert")
load("proto.star", "proto")

schema = proto.file("google/protobuf/descriptor.proto")

m = schema.FileDescriptorProto(name = "somename.proto", dependency = ["a", "b", "c"])
assert.eq(type(m), "proto.Message")
assert.eq(m.name, "somename.proto")
assert.eq(list(m.dependency), ["a", "b", "c"])
m.dependency = ["d", "e"]
assert.eq(list(m.dependency), ["d", "e"])

