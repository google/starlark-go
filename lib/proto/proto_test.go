package proto

import (
	"strings"
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

func TestRepeatedFieldComparison(t *testing.T) {
	predeclared := starlark.StringDict{
		"FileDescriptorProto": MessageDescriptor{Desc: (&descriptorpb.FileDescriptorProto{}).ProtoReflect().Descriptor()},
		"FileDescriptorSet":   MessageDescriptor{Desc: (&descriptorpb.FileDescriptorSet{}).ProtoReflect().Descriptor()},
	}
	const src = `
strings_a = FileDescriptorProto(
    dependency = ["alpha", "beta"],
    public_dependency = [3, 8],
    weak_dependency = [3, 8],
)
strings_b = FileDescriptorProto(dependency = ["alpha", "beta"])
strings_c = FileDescriptorProto(dependency = ["alpha", "gamma"])
strings_prefix = FileDescriptorProto(dependency = ["alpha"])
empty = FileDescriptorProto()

def test_scalar_comparison():
    if strings_a.dependency != strings_b.dependency:
        fail("equal repeated strings compare unequal")
    if strings_a.dependency == strings_c.dependency:
        fail("unequal repeated strings compare equal")
    if not strings_a.dependency < strings_c.dependency:
        fail("lexicographic < failed")
    if not strings_a.dependency <= strings_c.dependency:
        fail("lexicographic <= failed")
    if not strings_c.dependency > strings_a.dependency:
        fail("lexicographic > failed")
    if not strings_c.dependency >= strings_a.dependency:
        fail("lexicographic >= failed")
    if not strings_a.dependency <= strings_b.dependency:
        fail("equal fields do not satisfy <=")
    if not strings_a.dependency >= strings_b.dependency:
        fail("equal fields do not satisfy >=")
    if not strings_prefix.dependency < strings_a.dependency:
        fail("prefix ordering failed")
    if not strings_a.dependency > strings_prefix.dependency:
        fail("reverse prefix ordering failed")
    if empty.dependency != FileDescriptorProto().dependency:
        fail("empty repeated fields compare unequal")
    if strings_a.public_dependency != strings_a.weak_dependency:
        fail("same element type from different fields compares unequal")
    if empty.dependency == empty.public_dependency:
        fail("different repeated types compare equal")
    if strings_a.dependency == ["alpha", "beta"]:
        fail("repeated field compares equal to list")

shared = FileDescriptorProto(name = "shared.proto")
messages_a = FileDescriptorSet(file = [shared])
messages_b = FileDescriptorSet(file = [shared])
messages_copy = FileDescriptorSet(file = [FileDescriptorProto(name = "shared.proto")])

def test_message_comparison():
    if messages_a.file != messages_b.file:
        fail("fields containing the same message compare unequal")
    if messages_a.file == messages_copy.file:
        fail("fields containing distinct messages compare equal")
    if not messages_a.file <= messages_b.file:
        fail("equal message fields do not satisfy <=")
    if not messages_a.file >= messages_b.file:
        fail("equal message fields do not satisfy >=")

test_scalar_comparison()
test_message_comparison()
`

	thread := &starlark.Thread{Name: t.Name()}
	globals, err := starlark.ExecFile(thread, "comparison.star", src, predeclared)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name string
		expr string
		want string
	}{
		{
			name: "order different repeated types",
			expr: "empty.dependency < empty.public_dependency",
			want: "not implemented",
		},
		{
			name: "order distinct messages",
			expr: "messages_a.file < messages_copy.file",
			want: "proto.Message < proto.Message not implemented",
		},
		{
			name: "unhashable",
			expr: "{strings_a.dependency: None}",
			want: "unhashable: proto.repeated<string>",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := starlark.Eval(thread, "comparison.star", test.expr, globals); err == nil {
				t.Fatalf("%s succeeded, want error containing %q", test.expr, test.want)
			} else if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("%s: got error %q, want error containing %q", test.expr, err, test.want)
			}
		})
	}
}
