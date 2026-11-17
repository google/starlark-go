// Copyright 2026 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto_test

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
	_ "google.golang.org/protobuf/types/known/durationpb" // google.protobuf.Duration in protoregistry.GlobalTypes

	starlarkproto "go.starlark.net/lib/proto"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// testFiles returns a pool of google/protobuf/any.proto and test/test.proto,
// which defines message test.M with an Any field and the extension test.ext.
// No generated Go type exists for test.M, so protoregistry.GlobalTypes knows
// neither it nor its extension. The pool does not hold google.protobuf.Duration,
// which protoregistry.GlobalTypes does.
func testFiles(t *testing.T) *protoregistry.Files {
	t.Helper()
	files := new(protoregistry.Files)
	if err := files.RegisterFile(anypb.File_google_protobuf_any_proto); err != nil {
		t.Fatal(err)
	}
	fd, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Name:       proto.String("test/test.proto"),
		Package:    proto.String("test"),
		Syntax:     proto.String("proto2"),
		Dependency: []string{"google/protobuf/any.proto"},
		MessageType: []*descriptorpb.DescriptorProto{{
			Name: proto.String("M"),
			Field: []*descriptorpb.FieldDescriptorProto{{
				Name:     proto.String("a"),
				Number:   proto.Int32(1),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				TypeName: proto.String(".google.protobuf.Any"),
			}},
			ExtensionRange: []*descriptorpb.DescriptorProto_ExtensionRange{{
				Start: proto.Int32(100),
				End:   proto.Int32(101),
			}},
		}},
		Extension: []*descriptorpb.FieldDescriptorProto{{
			Name:     proto.String("ext"),
			Number:   proto.Int32(100),
			Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
			Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
			Extendee: proto.String(".test.M"),
		}},
	}, files)
	if err != nil {
		t.Fatal(err)
	}
	if err := files.RegisterFile(fd); err != nil {
		t.Fatal(err)
	}
	return files
}

// prelude binds dur to an Any holding google.protobuf.Duration(seconds = 7),
// a type only protoregistry.GlobalTypes resolves.
const prelude = `
anyf = proto.file("google/protobuf/any.proto")
dur = anyf.Any(type_url = "type.googleapis.com/google.protobuf.Duration", value = b"\x08\x07")
`

// TestResolver tests that the thread's resolver replaces
// protoregistry.GlobalTypes for google.protobuf.Any values. The tests of
// extensions are in starlark/testdata/proto{,_resolver}.star.
func TestResolver(t *testing.T) {
	files := testFiles(t)
	poolTypes := dynamicpb.NewTypes(files)
	tests := []struct {
		name     string
		resolver starlarkproto.TypeResolver // nil: none set
		src      string                     // sets got
		want     starlark.Value
		wantErr  string
	}{
		{
			name: "marshal_text expands an Any the global types resolve",
			src:  `got = proto.marshal_text(dur)`,
			want: starlark.String("[type.googleapis.com/google.protobuf.Duration]: {\n  seconds: 7\n}\n"),
		},
		{
			name:     "marshal_text leaves an Any the resolver does not resolve",
			resolver: poolTypes,
			src:      `got = proto.marshal_text(dur)`,
			want:     starlark.String("type_url: \"type.googleapis.com/google.protobuf.Duration\"\nvalue: \"\\x08\\x07\"\n"),
		},
		{
			name: "unmarshal_text expands an Any the global types resolve",
			src:  `got = proto.unmarshal_text(anyf.Any, "[type.googleapis.com/google.protobuf.Duration] { seconds: 7 }").value`,
			want: starlark.Bytes("\x08\x07"),
		},
		{
			name:     "unmarshal_text refuses an Any the resolver does not resolve",
			resolver: poolTypes,
			src:      `got = proto.unmarshal_text(anyf.Any, "[type.googleapis.com/google.protobuf.Duration] { seconds: 7 }")`,
			wantErr:  "unable to resolve",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			thread := &starlark.Thread{}
			starlarkproto.SetPool(thread, files)
			if tc.resolver != nil {
				starlarkproto.SetResolver(thread, tc.resolver)
			}
			globals, err := starlark.ExecFileOptions(&syntax.FileOptions{}, thread, "test.star", prelude+tc.src,
				starlark.StringDict{"proto": starlarkproto.Module})
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("got error %v, want one containing %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if eq, err := starlark.Equal(globals["got"], tc.want); err != nil || !eq {
				t.Errorf("got %v, want %v", globals["got"], tc.want)
			}
		})
	}
}

func TestResolver_Unset(t *testing.T) {
	thread := &starlark.Thread{}
	if r := starlarkproto.Resolver(thread); r != nil {
		t.Errorf("Resolver of a new thread = %v, want nil", r)
	}
	starlarkproto.SetResolver(thread, protoregistry.GlobalTypes)
	starlarkproto.SetResolver(thread, nil)
	if r := starlarkproto.Resolver(thread); r != nil {
		t.Errorf("Resolver after SetResolver(nil) = %v, want nil", r)
	}
}

// The Go functions resolve with protoregistry.GlobalTypes, and their
// WithResolver variants with the resolver they are given.
func TestUnmarshalWithResolver(t *testing.T) {
	files := testFiles(t)
	poolTypes := dynamicpb.NewTypes(files)
	d, err := files.FindDescriptorByName("test.M")
	if err != nil {
		t.Fatal(err)
	}
	md := d.(protoreflect.MessageDescriptor)
	xt, err := poolTypes.FindExtensionByName("test.ext")
	if err != nil {
		t.Fatal(err)
	}
	src := dynamicpb.NewMessage(md)
	src.Set(xt.TypeDescriptor(), protoreflect.ValueOfString("x"))
	data, err := proto.Marshal(src)
	if err != nil {
		t.Fatal(err)
	}
	ext := func(m *starlarkproto.Message) string {
		return m.Message().ProtoReflect().Get(xt.TypeDescriptor()).String()
	}

	m, err := starlarkproto.Unmarshal(md, data)
	if err != nil {
		t.Fatal(err)
	}
	if got := ext(m); got != "" {
		t.Errorf("Unmarshal: extension = %q, want it left unknown", got)
	}
	m, err = starlarkproto.UnmarshalWithResolver(md, data, poolTypes)
	if err != nil {
		t.Fatal(err)
	}
	if got := ext(m); got != "x" {
		t.Errorf("UnmarshalWithResolver: extension = %q, want %q", got, "x")
	}

	const text = `[test.ext]: "x"`
	if _, err = starlarkproto.UnmarshalText(md, []byte(text)); err == nil {
		t.Error("UnmarshalText: resolved an extension the global types do not hold")
	}
	m, err = starlarkproto.UnmarshalTextWithResolver(md, []byte(text), poolTypes)
	if err != nil {
		t.Fatal(err)
	}
	if got := ext(m); got != "x" {
		t.Errorf("UnmarshalTextWithResolver: extension = %q, want %q", got, "x")
	}
}
