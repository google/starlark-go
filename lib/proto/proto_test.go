package proto

import (
	"testing"

	"google.golang.org/protobuf/types/descriptorpb"

	"go.starlark.net/starlark"
)

func TestRepeatedFieldMembership(t *testing.T) {
	predeclared := starlark.StringDict{
		"FileDescriptorProto": MessageDescriptor{Desc: (&descriptorpb.FileDescriptorProto{}).ProtoReflect().Descriptor()},
		"FileDescriptorSet":   MessageDescriptor{Desc: (&descriptorpb.FileDescriptorSet{}).ProtoReflect().Descriptor()},
	}
	const src = `
def test_membership():
    msg = FileDescriptorProto(
        dependency = ["alpha", "beta"],
        public_dependency = [3, 8],
    )

    if "alpha" not in msg.dependency:
        fail("present string is missing")
    if "missing" in msg.dependency:
        fail("absent string is present")
    if "anything" in FileDescriptorProto().dependency:
        fail("empty repeated field contains a value")
    if 8 not in msg.public_dependency:
        fail("present integer is missing")
    if 5 in msg.public_dependency:
        fail("absent integer is present")

    messages = FileDescriptorSet(file = [msg])
    item = messages.file[0]
    if item != messages.file[0]:
        fail("wrappers of the same message do not compare equal")
    if item not in messages.file:
        fail("repeated field does not contain its message")
    by_message = {item: "found"}
    if by_message[messages.file[0]] != "found":
        fail("wrappers of the same message have inconsistent hashes")

    copy = FileDescriptorProto(
        dependency = ["alpha", "beta"],
        public_dependency = [3, 8],
    )
    if copy == item:
        fail("separately constructed messages compare equal")
    if copy in messages.file:
        fail("repeated field contains a distinct message")

test_membership()
`

	if _, err := starlark.ExecFile(&starlark.Thread{Name: t.Name()}, "membership.star", src, predeclared); err != nil {
		t.Fatal(err)
	}
}
