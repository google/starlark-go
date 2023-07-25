// Copyright 2020 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package proto defines a module of utilities for constructing and
// accessing protocol messages within Starlark programs.
//
// THIS PACKAGE IS EXPERIMENTAL AND ITS INTERFACE MAY CHANGE.
//
// This package defines several types of Starlark value:
//
//      Message                 -- a protocol message
//      RepeatedField           -- a repeated field of a message, like a list
//
//      FileDescriptor          -- information about a .proto file
//      FieldDescriptor         -- information about a message field (or extension field)
//      MessageDescriptor       -- information about the type of a message
//      EnumDescriptor          -- information about an enumerated type
//      EnumValueDescriptor     -- a value of an enumerated type
//
// A Message value is a wrapper around a protocol message instance.
// Starlark programs may access and update Messages using dot notation:
//
//      x = msg.field
//      msg.field = x + 1
//      msg.field += 1
//
// Assignments to message fields perform dynamic checks on the type and
// range of the value to ensure that the message is at all times valid.
//
// The value of a repeated field of a message is represented by the
// list-like data type, RepeatedField.  Its elements may be accessed,
// iterated, and updated in the usual ways.  As with assignments to
// message fields, an assignment to an element of a RepeatedField
// performs a dynamic check to ensure that the RepeatedField holds
// only elements of the correct type.
//
//      type(msg.uint32s)       # "proto.repeated<uint32>"
//      msg.uint32s[0] = 1
//      msg.uint32s[0] = -1     # error: invalid uint32: -1
//
// Any iterable may be assigned to a repeated field of a message.  If
// the iterable is itself a value of type RepeatedField, the message
// field holds a reference to it.
//
//      msg2.uint32s = msg.uint32s      # both messages share one RepeatedField
//      msg.uint32s[0] = 123
//      print(msg2.uint32s[0])          # "123"
//
// The RepeatedFields' element types must match.
// It is not enough for the values to be merely valid:
//
//      msg.uint32s = [1, 2, 3]         # makes a copy
//      msg.uint64s = msg.uint32s       # error: repeated field has wrong type
//      msg.uint64s = list(msg.uint32s) # ok; makes a copy
//
// For all other iterables, a new RepeatedField is constructed from the
// elements of the iterable.
//
//      msg.uints32s = [1, 2, 3]
//      print(type(msg.uints32s))       # "proto.repeated<uint32>"
//
//
// To construct a Message from encoded binary or text data, call
// Unmarshal or UnmarshalText.  These two functions are exposed to
// Starlark programs as proto.unmarshal{,_text}.
//
// To construct a Message from an existing Go proto.Message instance,
// you must first encode the Go message to binary, then decode it using
// Unmarshal. This ensures that messages visible to Starlark are
// encapsulated and cannot be mutated once their Starlark wrapper values
// are frozen.
//
// TODO(adonovan): document descriptors, enums, message instantiation.
//
// See proto_test.go for an example of how to use the 'proto'
// module in an application that embeds Starlark.
//
package proto

// TODO(adonovan): Go and Starlark API improvements:
// - Make Message and RepeatedField comparable.
//   (NOTE: proto.Equal works only with generated message types.)
// - Support maps, oneof, any. But not messageset if we can avoid it.
// - Support "well-known types".
// - Defend against cycles in object graph.
// - Test missing required fields in marshalling.

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"unsafe"
	_ "unsafe" // for linkname hack

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"go.starlark.net/syntax"
)

// SetPool associates with the specified Starlark thread the
// descriptor pool used to find descriptors for .proto files and to
// instantiate messages from descriptors.  Clients must call SetPool
// for a Starlark thread to use this package.
//
// For example:
//	SetPool(thread, protoregistry.GlobalFiles)
//
func SetPool(thread *starlark.Thread, pool DescriptorPool) {
	thread.SetLocal(contextKey, pool)
}

// Pool returns the descriptor pool previously associated with this thread.
func Pool(thread *starlark.Thread) DescriptorPool {
	pool, _ := thread.Local(contextKey).(DescriptorPool)
	return pool
}

const contextKey = "proto.DescriptorPool"

// A DescriptorPool loads FileDescriptors by path name or package name,
// possibly on demand.
//
// It is a superinterface of protodesc.Resolver, so any Resolver
// implementation is a valid pool. For example.
// protoregistry.GlobalFiles, which loads FileDescriptors from the
// compressed binary information in all the *.pb.go files linked into
// the process; and protodesc.NewFiles, which holds a set of
// FileDescriptorSet messages. See star2proto for example usage.
type DescriptorPool interface {
	FindFileByPath(string) (protoreflect.FileDescriptor, error)
}

var Module = &starlarkstruct.Module{
	Name: "proto",
	Members: starlark.StringDict{
		"file":           starlark.NewBuiltin("proto.file", file),
		"has":            starlark.NewBuiltin("proto.has", has),
		"marshal":        starlark.NewBuiltin("proto.marshal", marshal),
		"marshal_text":   starlark.NewBuiltin("proto.marshal_text", marshal),
		"set_field":      starlark.NewBuiltin("proto.set_field", setFieldStarlark),
		"get_field":      starlark.NewBuiltin("proto.get_field", getFieldStarlark),
		"unmarshal":      starlark.NewBuiltin("proto.unmarshal", unmarshal),
		"unmarshal_text": starlark.NewBuiltin("proto.unmarshal_text", unmarshal_text),

		// TODO(adonovan):
		// - merge(msg, msg) -> msg
		// - equals(msg, msg) -> bool
		// - diff(msg, msg) -> string
		// - clone(msg) -> msg
	},
}

// file(filename) loads the FileDescriptor of the given name, or the
// first if the pool contains more than one.
//
// It's unfortunate that renaming a .proto file in effect breaks the
// interface it presents to Starlark. Ideally one would import
// descriptors by package name, but there may be many FileDescriptors
// for the same package name, and there is no "package descriptor".
// (Technically a pool may also have many FileDescriptors with the same
// file name, but this can't happen with a single consistent snapshot.)
func file(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var filename string
	if err := starlark.UnpackPositionalArgs(fn.Name(), args, kwargs, 1, &filename); err != nil {
		return nil, err
	}

	pool := Pool(thread)
	if pool == nil {
		return nil, fmt.Errorf("internal error: SetPool was not called")
	}

	desc, err := pool.FindFileByPath(filename)
	if err != nil {
		return nil, err
	}

	return FileDescriptor{Desc: desc}, nil
}

// has(msg, field) reports whether the specified field of the message is present.
// A field may be specified by name (string) or FieldDescriptor.
// has reports an error if the message type has no such field.
func has(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var x, field starlark.Value
	if err := starlark.UnpackPositionalArgs(fn.Name(), args, kwargs, 2, &x, &field); err != nil {
		return nil, err
	}
	msg, ok := x.(*Message)
	if !ok {
		return nil, fmt.Errorf("%s: got %s, want proto.Message", fn.Name(), x.Type())
	}

	var fdesc protoreflect.FieldDescriptor
	switch field := field.(type) {
	case starlark.String:
		var err error
		fdesc, err = fieldDesc(msg.desc(), string(field))
		if err != nil {
			return nil, err
		}

	case FieldDescriptor:
		if field.Desc.ContainingMessage() != msg.desc() {
			return nil, fmt.Errorf("%s: %v does not have field %v", fn.Name(), msg.desc().FullName(), field)
		}
		fdesc = field.Desc

	default:
		return nil, fmt.Errorf("%s: for field argument, got %s, want string or proto.FieldDescriptor", fn.Name(), field.Type())
	}

	return starlark.Bool(msg.msg.Has(fdesc)), nil
}

// marshal{,_text}(msg) encodes a Message value to binary or text form.
func marshal(_ *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var m *Message
	if err := starlark.UnpackPositionalArgs(fn.Name(), args, kwargs, 1, &m); err != nil {
		return nil, err
	}
	if fn.Name() == "proto.marshal" {
		data, err := proto.Marshal(m.Message())
		if err != nil {
			return nil, fmt.Errorf("%s: %v", fn.Name(), err)
		}
		return starlark.Bytes(data), nil
	} else {
		text, err := prototext.MarshalOptions{Indent: "  "}.Marshal(m.Message())
		if err != nil {
			return nil, fmt.Errorf("%s: %v", fn.Name(), err)
		}
		return starlark.String(text), nil
	}
}

// unmarshal(msg) decodes a binary protocol message to a Message.
func unmarshal(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var desc MessageDescriptor
	var data starlark.Bytes
	if err := starlark.UnpackPositionalArgs(fn.Name(), args, kwargs, 2, &desc, &data); err != nil {
		return nil, err
	}
	return unmarshalData(desc.Desc, []byte(data), true)
}

// unmarshal_text(msg) decodes a text protocol message to a Message.
func unmarshal_text(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var desc MessageDescriptor
	var data string
	if err := starlark.UnpackPositionalArgs(fn.Name(), args, kwargs, 2, &desc, &data); err != nil {
		return nil, err
	}
	return unmarshalData(desc.Desc, []byte(data), false)
}

// set_field(msg, field, value) updates the value of a field.
// It is typically used for extensions, which cannot be updated using msg.field = v notation.
func setFieldStarlark(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	// TODO(adonovan): allow field to be specified by name (for non-extension fields), like has?
	var m *Message
	var field FieldDescriptor
	var v starlark.Value
	if err := starlark.UnpackPositionalArgs(fn.Name(), args, kwargs, 3, &m, &field, &v); err != nil {
		return nil, err
	}

	if *m.frozen {
		return nil, fmt.Errorf("%s: cannot set %v field of frozen %v message", fn.Name(), field, m.desc().FullName())
	}

	if field.Desc.ContainingMessage() != m.desc() {
		return nil, fmt.Errorf("%s: %v does not have field %v", fn.Name(), m.desc().FullName(), field)
	}

	return starlark.None, setField(m.msg, field.Desc, v)
}

// get_field(msg, field) retrieves the value of a field.
// It is typically used for extension fields, which cannot be accessed using msg.field notation.
func getFieldStarlark(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	// TODO(adonovan): allow field to be specified by name (for non-extension fields), like has?
	var msg *Message
	var field FieldDescriptor
	if err := starlark.UnpackPositionalArgs(fn.Name(), args, kwargs, 2, &msg, &field); err != nil {
		return nil, err
	}

	if field.Desc.ContainingMessage() != msg.desc() {
		return nil, fmt.Errorf("%s: %v does not have field %v", fn.Name(), msg.desc().FullName(), field)
	}

	return msg.getField(field.Desc), nil
}

// The Call method implements the starlark.Callable interface.
// When a message descriptor is called, it returns a new instance of the
// protocol message it describes.
//
//      Message(msg)            -- return a shallow copy of an existing message
//      Message(k=v, ...)       -- return a new message with the specified fields
//      Message(dict(...))      -- return a new message with the specified fields
//
func (d MessageDescriptor) CallInternal(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	dest := &Message{
		msg:    newMessage(d.Desc),
		frozen: new(bool),
	}

	// Single positional argument?
	if len(args) > 0 {
		if len(kwargs) > 0 {
			return nil, fmt.Errorf("%s: got both positional and named arguments", d.Desc.Name())
		}
		if len(args) > 1 {
			return nil, fmt.Errorf("%s: got %d positional arguments, want at most 1", d.Desc.Name(), len(args))
		}

		// Keep consistent with MessageKind case of toProto.
		// (support the same argument types).
		switch src := args[0].(type) {
		case *Message:
			if dest.desc() != src.desc() {
				return nil, fmt.Errorf("%s: got message of type %s, want type %s", d.Desc.Name(), src.desc().FullName(), dest.desc().FullName())
			}

			// Make shallow copy of message.
			// TODO(adonovan): How does frozen work if we have shallow copy?
			src.msg.Range(func(fdesc protoreflect.FieldDescriptor, v protoreflect.Value) bool {
				dest.msg.Set(fdesc, v)
				return true
			})
			return dest, nil

		case *starlark.Dict:
			kwargs = src.Items()
			// fall through

		default:
			return nil, fmt.Errorf("%s: got %s, want dict or message", d.Desc.Name(), src.Type())
		}
	}

	// Convert named arguments to field values.
	err := setFields(dest.msg, kwargs)
	return dest, err
}

// setFields updates msg as if by msg.name=value for each (name, value) in items.
func setFields(msg protoreflect.Message, items []starlark.Tuple) error {
	for _, item := range items {
		name, ok := starlark.AsString(item[0])
		if !ok {
			return fmt.Errorf("got %s, want string", item[0].Type())
		}
		fdesc, err := fieldDesc(msg.Descriptor(), name)
		if err != nil {
			return err
		}
		if err := setField(msg, fdesc, item[1]); err != nil {
			return err
		}
	}
	return nil
}

// setField validates a Starlark field value, converts it to canonical form,
// and assigns to the field of msg.  If value is None, the field is unset.
func setField(msg protoreflect.Message, fdesc protoreflect.FieldDescriptor, value starlark.Value) error {
	// None unsets a field.
	if value == starlark.None {
		msg.Clear(fdesc)
		return nil
	}

	// Assigning to a repeated field must make a copy,
	// because the fields.Set doesn't specify whether
	// it aliases the list or not, so we cannot assume.
	//
	// This is potentially surprising as
	//  x = []; msg.x = x; y = msg.x
	// causes x and y not to alias.
	if fdesc.IsList() {
		iter := starlark.Iterate(value)
		if iter == nil {
			return fmt.Errorf("got %s for .%s field, want iterable", value.Type(), fdesc.Name())
		}
		defer iter.Done()

		list := msg.Mutable(fdesc).List()
		list.Truncate(0)
		var x starlark.Value
		for i := 0; iter.Next(&x); i++ {
			v, err := toProto(fdesc, x)
			if err != nil {
				return fmt.Errorf("index %d: %v", i, err)
			}
			list.Append(v)
		}
		return nil
	}

	if fdesc.IsMap() {
		mapping, ok := value.(starlark.IterableMapping)
		if !ok {
			return fmt.Errorf("in map field %s: expected mappable type, but got %s", fdesc.Name(), value.Type())
		}

		iter := mapping.Iterate()
		defer iter.Done()

		// Each value is converted using toProto as usual, passing the key/value
		// field descriptors to check their types.
		mutMap := msg.Mutable(fdesc).Map()
		var k starlark.Value
		for iter.Next(&k) {
			kproto, err := toProto(fdesc.MapKey(), k)
			if err != nil {
				return fmt.Errorf("in key of map field %s: %w", fdesc.Name(), err)
			}

			// `found` is discarded, as the presence of the key in the
			// iterator guarantees the presence of some value (even if it is
			// starlark.None). Mismatching values will be caught in toProto
			// below.
			v, _, err := mapping.Get(k)
			if err != nil {
				return fmt.Errorf("in map field %s, at key %s: %w", fdesc.Name(), k.String(), err)
			}

			vproto, err := toProto(fdesc.MapValue(), v)
			if err != nil {
				return fmt.Errorf("in map field %s, at key %s: %w", fdesc.Name(), k.String(), err)
			}

			mutMap.Set(kproto.MapKey(), vproto)
		}

		return nil
	}

	v, err := toProto(fdesc, value)
	if err != nil {
		return fmt.Errorf("in field %s: %v", fdesc.Name(), err)
	}

	if fdesc.IsExtension() {
		// The protoreflect.Message.NewField method must be able
		// to return a new instance of the field type. Without
		// having the Go type information available for extensions,
		// the implementation of NewField won't know what to do.
		//
		// Thus we must augment the FieldDescriptor to one that
		// additional holds Go representation type information
		// (based in this case on dynamicpb).
		fdesc = dynamicpb.NewExtensionType(fdesc).TypeDescriptor()
		_ = fdesc.(protoreflect.ExtensionTypeDescriptor)
	}

	msg.Set(fdesc, v)
	return nil
}

// toProto converts a Starlark value for a message field into protoreflect form.
func toProto(fdesc protoreflect.FieldDescriptor, v starlark.Value) (protoreflect.Value, error) {
	switch fdesc.Kind() {
	case protoreflect.BoolKind:
		// To avoid mistakes, we require v be exactly a bool.
		if v, ok := v.(starlark.Bool); ok {
			return protoreflect.ValueOfBool(bool(v)), nil
		}

	case protoreflect.Fixed32Kind,
		protoreflect.Uint32Kind:
		// uint32
		if i, ok := v.(starlark.Int); ok {
			if u, ok := i.Uint64(); ok && uint64(uint32(u)) == u {
				return protoreflect.ValueOfUint32(uint32(u)), nil
			}
			return noValue, fmt.Errorf("invalid %s: %v", typeString(fdesc), i)
		}

	case protoreflect.Int32Kind,
		protoreflect.Sfixed32Kind,
		protoreflect.Sint32Kind:
		// int32
		if i, ok := v.(starlark.Int); ok {
			if i, ok := i.Int64(); ok && int64(int32(i)) == i {
				return protoreflect.ValueOfInt32(int32(i)), nil
			}
			return noValue, fmt.Errorf("invalid %s: %v", typeString(fdesc), i)
		}

	case protoreflect.Uint64Kind,
		protoreflect.Fixed64Kind:
		// uint64
		if i, ok := v.(starlark.Int); ok {
			if u, ok := i.Uint64(); ok {
				return protoreflect.ValueOfUint64(u), nil
			}
			return noValue, fmt.Errorf("invalid %s: %v", typeString(fdesc), i)
		}

	case protoreflect.Int64Kind,
		protoreflect.Sfixed64Kind,
		protoreflect.Sint64Kind:
		// int64
		if i, ok := v.(starlark.Int); ok {
			if i, ok := i.Int64(); ok {
				return protoreflect.ValueOfInt64(i), nil
			}
			return noValue, fmt.Errorf("invalid %s: %v", typeString(fdesc), i)
		}

	case protoreflect.StringKind:
		if s, ok := starlark.AsString(v); ok {
			return protoreflect.ValueOfString(s), nil
		} else if b, ok := v.(starlark.Bytes); ok {
			// TODO(adonovan): allow bytes for string? Not friendly to a Java port.
			return protoreflect.ValueOfBytes([]byte(b)), nil
		}

	case protoreflect.BytesKind:
		if s, ok := starlark.AsString(v); ok {
			// TODO(adonovan): don't allow string for bytes: it's hostile to a Java port.
			// Instead provide b"..." literals in the core
			// and a bytes(str) conversion.
			return protoreflect.ValueOfBytes([]byte(s)), nil
		} else if b, ok := v.(starlark.Bytes); ok {
			return protoreflect.ValueOfBytes([]byte(b)), nil
		}

	case protoreflect.DoubleKind:
		switch v := v.(type) {
		case starlark.Float:
			return protoreflect.ValueOfFloat64(float64(v)), nil
		case starlark.Int:
			return protoreflect.ValueOfFloat64(float64(v.Float())), nil
		}

	case protoreflect.FloatKind:
		switch v := v.(type) {
		case starlark.Float:
			return protoreflect.ValueOfFloat32(float32(v)), nil
		case starlark.Int:
			return protoreflect.ValueOfFloat32(float32(v.Float())), nil
		}

	case protoreflect.GroupKind,
		protoreflect.MessageKind:
		// Keep consistent with MessageDescriptor.CallInternal!
		desc := fdesc.Message()
		switch v := v.(type) {
		case *Message:
			if desc != v.desc() {
				return noValue, fmt.Errorf("got %s, want %s", v.desc().FullName(), desc.FullName())
			}
			return protoreflect.ValueOfMessage(v.msg), nil // alias it directly

		case *starlark.Dict:
			dest := newMessage(desc)
			err := setFields(dest, v.Items())
			return protoreflect.ValueOfMessage(dest), err
		}

	case protoreflect.EnumKind:
		enumval, err := enumValueOf(fdesc.Enum(), v)
		if err != nil {
			return noValue, err
		}
		return protoreflect.ValueOfEnum(enumval.Number()), nil
	}

	return noValue, fmt.Errorf("got %s, want %s", v.Type(), typeString(fdesc))
}

var noValue protoreflect.Value

// toStarlark returns a Starlark value for the value x of a message field.
// If the result is a repeated field or message,
// the result aliases the original and has the specified "frozenness" flag.
//
// fdesc is only used for the type, not other properties of the field.
func toStarlark(typ protoreflect.FieldDescriptor, x protoreflect.Value, frozen *bool) starlark.Value {
	if list, ok := x.Interface().(protoreflect.List); ok {
		return &RepeatedField{
			typ:    typ,
			list:   list,
			frozen: frozen,
		}
	}
	return toStarlark1(typ, x, frozen)
}

// toStarlark1, for scalar (non-repeated) values only.
func toStarlark1(typ protoreflect.FieldDescriptor, x protoreflect.Value, frozen *bool) starlark.Value {

	switch typ.Kind() {
	case protoreflect.BoolKind:
		return starlark.Bool(x.Bool())

	case protoreflect.Fixed32Kind,
		protoreflect.Uint32Kind,
		protoreflect.Uint64Kind,
		protoreflect.Fixed64Kind:
		return starlark.MakeUint64(x.Uint())

	case protoreflect.Int32Kind,
		protoreflect.Sfixed32Kind,
		protoreflect.Sint32Kind,
		protoreflect.Int64Kind,
		protoreflect.Sfixed64Kind,
		protoreflect.Sint64Kind:
		return starlark.MakeInt64(x.Int())

	case protoreflect.StringKind:
		return starlark.String(x.String())

	case protoreflect.BytesKind:
		return starlark.Bytes(x.Bytes())

	case protoreflect.DoubleKind, protoreflect.FloatKind:
		return starlark.Float(x.Float())

	case protoreflect.GroupKind, protoreflect.MessageKind:
		return &Message{
			msg:    x.Message(),
			frozen: frozen,
		}

	case protoreflect.EnumKind:
		// Invariant: only EnumValueDescriptor may appear here.
		enumval := typ.Enum().Values().ByNumber(x.Enum())
		return EnumValueDescriptor{Desc: enumval}
	}

	panic(fmt.Sprintf("got %T, want %s", x, typeString(typ)))
}

// A Message is a Starlark value that wraps a protocol message.
//
// Two Messages are equivalent if and only if they are identical.
//
// When a Message value becomes frozen, a Starlark program may
// not modify the underlying protocol message, nor any Message
// or RepeatedField wrapper values derived from it.
type Message struct {
	msg    protoreflect.Message // any concrete type is allowed
	frozen *bool                // shared by a group of related Message/RepeatedField wrappers
}

// Message returns the wrapped message.
func (m *Message) Message() protoreflect.ProtoMessage { return m.msg.Interface() }

func (m *Message) desc() protoreflect.MessageDescriptor { return m.msg.Descriptor() }

var _ starlark.HasSetField = (*Message)(nil)

// Unmarshal parses the data as a binary protocol message of the specified type,
// and returns it as a new Starlark message value.
func Unmarshal(desc protoreflect.MessageDescriptor, data []byte) (*Message, error) {
	return unmarshalData(desc, data, true)
}

// UnmarshalText parses the data as a text protocol message of the specified type,
// and returns it as a new Starlark message value.
func UnmarshalText(desc protoreflect.MessageDescriptor, data []byte) (*Message, error) {
	return unmarshalData(desc, data, false)
}

// unmarshalData constructs a Starlark proto.Message by decoding binary or text data.
func unmarshalData(desc protoreflect.MessageDescriptor, data []byte, binary bool) (*Message, error) {
	m := &Message{
		msg:    newMessage(desc),
		frozen: new(bool),
	}
	var err error
	if binary {
		err = proto.Unmarshal(data, m.Message())
	} else {
		err = prototext.Unmarshal(data, m.Message())
	}
	if err != nil {
		return nil, fmt.Errorf("unmarshalling %s failed: %v", desc.FullName(), err)
	}
	return m, nil
}

func (m *Message) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(string(m.desc().FullName()))
	buf.WriteByte('(')

	// Sort fields (including extensions) by number.
	var fields []protoreflect.FieldDescriptor
	m.msg.Range(func(fdesc protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		// TODO(adonovan): opt: save v in table too.
		fields = append(fields, fdesc)
		return true
	})
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Number() < fields[j].Number()
	})

	for i, fdesc := range fields {
		if i > 0 {
			buf.WriteString(", ")
		}
		if fdesc.IsExtension() {
			// extension field: "[pkg.Msg.field]"
			buf.WriteString(string(fdesc.FullName()))
		} else if fdesc.Kind() != protoreflect.GroupKind {
			// ordinary field: "field"
			buf.WriteString(string(fdesc.Name()))
		} else {
			// group field: "MyGroup"
			//
			// The name of a group is the mangled version,
			// while the true name of a group is the message itself.
			// For example, for a group called "MyGroup",
			// the inlined message will be called "MyGroup",
			// but the field will be named "mygroup".
			// This rule complicates name logic everywhere.
			buf.WriteString(string(fdesc.Message().Name()))
		}
		buf.WriteString("=")
		writeString(buf, fdesc, m.msg.Get(fdesc))
	}
	buf.WriteByte(')')
	return buf.String()
}

func (m *Message) Type() string                { return "proto.Message" }
func (m *Message) Truth() starlark.Bool        { return true }
func (m *Message) Freeze()                     { *m.frozen = true }
func (m *Message) Hash() (h uint32, err error) { return uint32(uintptr(unsafe.Pointer(m))), nil } // identity hash

// Attr returns the value of this message's field of the specified name.
// Extension fields are not accessible this way as their names are not unique.
func (m *Message) Attr(name string) (starlark.Value, error) {
	// The name 'descriptor' is already effectively reserved
	// by the Go API for generated message types.
	if name == "descriptor" {
		return MessageDescriptor{Desc: m.desc()}, nil
	}

	fdesc, err := fieldDesc(m.desc(), name)
	if err != nil {
		return nil, err
	}
	return m.getField(fdesc), nil
}

func (m *Message) getField(fdesc protoreflect.FieldDescriptor) starlark.Value {
	if fdesc.IsExtension() {
		// See explanation in setField.
		fdesc = dynamicpb.NewExtensionType(fdesc).TypeDescriptor()
	}

	if m.msg.Has(fdesc) {
		return toStarlark(fdesc, m.msg.Get(fdesc), m.frozen)
	}
	return defaultValue(fdesc)
}

//go:linkname detrandDisable google.golang.org/protobuf/internal/detrand.Disable
func detrandDisable()

func init() {
	// Nasty hack to disable the randomization of output that occurs in textproto.
	// TODO(adonovan): once go/proto-proposals/canonical-serialization
	// is resolved the need for the hack should go away. See also go/go-proto-stability.
	// If the proposal is rejected, we will need our own text-mode formatter.
	detrandDisable()
}

// defaultValue returns the (frozen) default Starlark value for a given message field.
func defaultValue(fdesc protoreflect.FieldDescriptor) starlark.Value {
	frozen := true

	// The default value of a repeated field is an empty list.
	if fdesc.IsList() {
		return &RepeatedField{typ: fdesc, list: emptyList{}, frozen: &frozen}
	}

	// The zero value for a message type is an empty instance of that message.
	if desc := fdesc.Message(); desc != nil {
		return &Message{msg: newMessage(desc), frozen: &frozen}
	}

	// Convert the default value, which is not necessarily zero, to Starlark.
	// The frozenness isn't used as the remaining types are all immutable.
	return toStarlark1(fdesc, fdesc.Default(), &frozen)
}

// A frozen empty implementation of protoreflect.List.
type emptyList struct{ protoreflect.List }

func (emptyList) Len() int { return 0 }

// newMessage returns a new empty instance of the message type described by desc.
func newMessage(desc protoreflect.MessageDescriptor) protoreflect.Message {
	// If desc refers to a built-in message,
	// use the more efficient generated type descriptor (a Go struct).
	mt, err := protoregistry.GlobalTypes.FindMessageByName(desc.FullName())
	if err == nil && mt.Descriptor() == desc {
		return mt.New()
	}

	// For all others, use the generic dynamicpb representation.
	return dynamicpb.NewMessage(desc).ProtoReflect()
}

// fieldDesc returns the descriptor for the named non-extension field.
func fieldDesc(desc protoreflect.MessageDescriptor, name string) (protoreflect.FieldDescriptor, error) {
	if fdesc := desc.Fields().ByName(protoreflect.Name(name)); fdesc != nil {
		return fdesc, nil
	}
	return nil, starlark.NoSuchAttrError(fmt.Sprintf("%s has no .%s field", desc.FullName(), name))
}

// SetField updates a non-extension field of this message.
// It implements the HasSetField interface.
func (m *Message) SetField(name string, v starlark.Value) error {
	fdesc, err := fieldDesc(m.desc(), name)
	if err != nil {
		return err
	}
	if *m.frozen {
		return fmt.Errorf("cannot set .%s field of frozen %s message",
			name, m.desc().FullName())
	}
	return setField(m.msg, fdesc, v)
}

// AttrNames returns the set of field names defined for this message.
// It satisfies the starlark.HasAttrs interface.
func (m *Message) AttrNames() []string {
	seen := make(map[string]bool)

	// standard fields
	seen["descriptor"] = true

	// non-extension fields
	fields := m.desc().Fields()
	for i := 0; i < fields.Len(); i++ {
		fdesc := fields.Get(i)
		if !fdesc.IsExtension() {
			seen[string(fdesc.Name())] = true
		}
	}

	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// typeString returns a user-friendly description of the type of a
// protocol message field (or element of a repeated field).
func typeString(fdesc protoreflect.FieldDescriptor) string {
	switch fdesc.Kind() {
	case protoreflect.GroupKind,
		protoreflect.MessageKind:
		return string(fdesc.Message().FullName())

	case protoreflect.EnumKind:
		return string(fdesc.Enum().FullName())

	default:
		return strings.ToLower(strings.TrimPrefix(fdesc.Kind().String(), "TYPE_"))
	}
}

// A RepeatedField is a Starlark value that wraps a repeated field of a protocol message.
//
// An assignment to an element of a repeated field incurs a dynamic
// check that the new value has (or can be converted to) the correct
// type using conversions similar to those done when calling a
// MessageDescriptor to construct a message.
//
// TODO(adonovan): make RepeatedField implement starlark.Comparable.
// Should the comparison include type, or be defined on the elements alone?
type RepeatedField struct {
	typ       protoreflect.FieldDescriptor // only for type information, not field name
	list      protoreflect.List
	frozen    *bool
	itercount int
}

var _ starlark.HasSetIndex = (*RepeatedField)(nil)

func (rf *RepeatedField) Type() string {
	return fmt.Sprintf("proto.repeated<%s>", typeString(rf.typ))
}

func (rf *RepeatedField) SetIndex(i int, v starlark.Value) error {
	if *rf.frozen {
		return fmt.Errorf("cannot insert value in frozen repeated field")
	}
	if rf.itercount > 0 {
		return fmt.Errorf("cannot insert value in repeated field with active iterators")
	}
	x, err := toProto(rf.typ, v)
	if err != nil {
		// The repeated field value cannot know which field it
		// belongs to---it might be shared by several of the
		// same type---so the error message is suboptimal.
		return fmt.Errorf("setting element of repeated field: %v", err)
	}
	rf.list.Set(i, x)
	return nil
}

func (rf *RepeatedField) Freeze()               { *rf.frozen = true }
func (rf *RepeatedField) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: %s", rf.Type()) }
func (rf *RepeatedField) Index(i int) starlark.Value {
	return toStarlark1(rf.typ, rf.list.Get(i), rf.frozen)
}
func (rf *RepeatedField) Iterate() starlark.Iterator {
	if !*rf.frozen {
		rf.itercount++
	}
	return &repeatedFieldIterator{rf, 0}
}
func (rf *RepeatedField) Len() int { return rf.list.Len() }
func (rf *RepeatedField) String() string {
	// We use list [...] notation even though it not exactly a list.
	buf := new(bytes.Buffer)
	buf.WriteByte('[')
	for i := 0; i < rf.list.Len(); i++ {
		if i > 0 {
			buf.WriteString(", ")
		}
		writeString(buf, rf.typ, rf.list.Get(i))
	}
	buf.WriteByte(']')
	return buf.String()
}
func (rf *RepeatedField) Truth() starlark.Bool { return rf.list.Len() > 0 }

type repeatedFieldIterator struct {
	rf *RepeatedField
	i  int
}

func (it *repeatedFieldIterator) Next(p *starlark.Value) bool {
	if it.i < it.rf.Len() {
		*p = it.rf.Index(it.i)
		it.i++
		return true
	}
	return false
}

func (it *repeatedFieldIterator) Done() {
	if !*it.rf.frozen {
		it.rf.itercount--
	}
}

func writeString(buf *bytes.Buffer, fdesc protoreflect.FieldDescriptor, v protoreflect.Value) {
	// TODO(adonovan): opt: don't materialize the Starlark value.
	// TODO(adonovan): skip message type when printing submessages? {...}?
	var frozen bool // ignored
	x := toStarlark(fdesc, v, &frozen)
	buf.WriteString(x.String())
}

// -------- descriptor values --------

// A FileDescriptor is an immutable Starlark value that describes a
// .proto file.  It is a reference to a protoreflect.FileDescriptor.
// Two FileDescriptor values compare equal if and only if they refer to
// the same protoreflect.FileDescriptor.
//
// Its fields are the names of the message types (MessageDescriptor) and enum
// types (EnumDescriptor).
type FileDescriptor struct {
	Desc protoreflect.FileDescriptor // TODO(adonovan): hide field, expose method?
}

var _ starlark.HasAttrs = FileDescriptor{}

func (f FileDescriptor) String() string              { return string(f.Desc.Path()) }
func (f FileDescriptor) Type() string                { return "proto.FileDescriptor" }
func (f FileDescriptor) Truth() starlark.Bool        { return true }
func (f FileDescriptor) Freeze()                     {} // immutable
func (f FileDescriptor) Hash() (h uint32, err error) { return starlark.String(f.Desc.Path()).Hash() }
func (f FileDescriptor) Attr(name string) (starlark.Value, error) {
	if desc := f.Desc.Messages().ByName(protoreflect.Name(name)); desc != nil {
		return MessageDescriptor{Desc: desc}, nil
	}
	if desc := f.Desc.Extensions().ByName(protoreflect.Name(name)); desc != nil {
		return FieldDescriptor{desc}, nil
	}
	if enum := f.Desc.Enums().ByName(protoreflect.Name(name)); enum != nil {
		return EnumDescriptor{Desc: enum}, nil
	}
	return nil, nil
}
func (f FileDescriptor) AttrNames() []string {
	var names []string
	messages := f.Desc.Messages()
	for i, n := 0, messages.Len(); i < n; i++ {
		names = append(names, string(messages.Get(i).Name()))
	}
	extensions := f.Desc.Extensions()
	for i, n := 0, extensions.Len(); i < n; i++ {
		names = append(names, string(extensions.Get(i).Name()))
	}
	enums := f.Desc.Enums()
	for i, n := 0, enums.Len(); i < n; i++ {
		names = append(names, string(enums.Get(i).Name()))
	}
	sort.Strings(names)
	return names
}

// A MessageDescriptor is an immutable Starlark value that describes a protocol
// message type.
//
// A MessageDescriptor value contains a reference to a protoreflect.MessageDescriptor.
// Two MessageDescriptor values compare equal if and only if they refer to the
// same protoreflect.MessageDescriptor.
//
// The fields of a MessageDescriptor value are the names of any message types
// (MessageDescriptor), fields or extension fields (FieldDescriptor),
// and enum types (EnumDescriptor) nested within the declaration of this message type.
type MessageDescriptor struct {
	Desc protoreflect.MessageDescriptor
}

var (
	_ starlark.Callable = MessageDescriptor{}
	_ starlark.HasAttrs = MessageDescriptor{}
)

func (d MessageDescriptor) String() string       { return string(d.Desc.FullName()) }
func (d MessageDescriptor) Type() string         { return "proto.MessageDescriptor" }
func (d MessageDescriptor) Truth() starlark.Bool { return true }
func (d MessageDescriptor) Freeze()              {} // immutable
func (d MessageDescriptor) Hash() (h uint32, err error) {
	return starlark.String(d.Desc.FullName()).Hash()
}
func (d MessageDescriptor) Attr(name string) (starlark.Value, error) {
	if desc := d.Desc.Messages().ByName(protoreflect.Name(name)); desc != nil {
		return MessageDescriptor{desc}, nil
	}
	if desc := d.Desc.Extensions().ByName(protoreflect.Name(name)); desc != nil {
		return FieldDescriptor{desc}, nil
	}
	if desc := d.Desc.Fields().ByName(protoreflect.Name(name)); desc != nil {
		return FieldDescriptor{desc}, nil
	}
	if desc := d.Desc.Enums().ByName(protoreflect.Name(name)); desc != nil {
		return EnumDescriptor{desc}, nil
	}
	return nil, nil
}
func (d MessageDescriptor) AttrNames() []string {
	var names []string
	messages := d.Desc.Messages()
	for i, n := 0, messages.Len(); i < n; i++ {
		names = append(names, string(messages.Get(i).Name()))
	}
	enums := d.Desc.Enums()
	for i, n := 0, enums.Len(); i < n; i++ {
		names = append(names, string(enums.Get(i).Name()))
	}
	sort.Strings(names)
	return names
}
func (d MessageDescriptor) Name() string { return string(d.Desc.Name()) } // for Callable

// A FieldDescriptor is an immutable Starlark value that describes
// a field (possibly an extension field) of protocol message.
//
// A FieldDescriptor value contains a reference to a protoreflect.FieldDescriptor.
// Two FieldDescriptor values compare equal if and only if they refer to the
// same protoreflect.FieldDescriptor.
//
// The primary use for FieldDescriptors is to access extension fields of a message.
//
// A FieldDescriptor value has not attributes.
// TODO(adonovan): expose metadata fields (e.g. name, type).
type FieldDescriptor struct {
	Desc protoreflect.FieldDescriptor
}

var (
	_ starlark.HasAttrs = FieldDescriptor{}
)

func (d FieldDescriptor) String() string       { return string(d.Desc.FullName()) }
func (d FieldDescriptor) Type() string         { return "proto.FieldDescriptor" }
func (d FieldDescriptor) Truth() starlark.Bool { return true }
func (d FieldDescriptor) Freeze()              {} // immutable
func (d FieldDescriptor) Hash() (h uint32, err error) {
	return starlark.String(d.Desc.FullName()).Hash()
}
func (d FieldDescriptor) Attr(name string) (starlark.Value, error) {
	// TODO(adonovan): expose metadata fields of Desc?
	return nil, nil
}
func (d FieldDescriptor) AttrNames() []string {
	var names []string
	// TODO(adonovan): expose metadata fields of Desc?
	sort.Strings(names)
	return names
}

// An EnumDescriptor is an immutable Starlark value that describes an
// protocol enum type.
//
// An EnumDescriptor contains a reference to a protoreflect.EnumDescriptor.
// Two EnumDescriptor values compare equal if and only if they
// refer to the same protoreflect.EnumDescriptor.
//
// An EnumDescriptor may be called like a function.  It converts its
// sole argument, which must be an int, string, or EnumValueDescriptor,
// to an EnumValueDescriptor.
//
// The fields of an EnumDescriptor value are the values of the
// enumeration, each of type EnumValueDescriptor.
type EnumDescriptor struct {
	Desc protoreflect.EnumDescriptor
}

var (
	_ starlark.HasAttrs = EnumDescriptor{}
	_ starlark.Callable = EnumDescriptor{}
)

func (e EnumDescriptor) String() string              { return string(e.Desc.FullName()) }
func (e EnumDescriptor) Type() string                { return "proto.EnumDescriptor" }
func (e EnumDescriptor) Truth() starlark.Bool        { return true }
func (e EnumDescriptor) Freeze()                     {}                // immutable
func (e EnumDescriptor) Hash() (h uint32, err error) { return 0, nil } // TODO(adonovan): number?
func (e EnumDescriptor) Attr(name string) (starlark.Value, error) {
	if v := e.Desc.Values().ByName(protoreflect.Name(name)); v != nil {
		return EnumValueDescriptor{v}, nil
	}
	return nil, nil
}
func (e EnumDescriptor) AttrNames() []string {
	var names []string
	values := e.Desc.Values()
	for i, n := 0, values.Len(); i < n; i++ {
		names = append(names, string(values.Get(i).Name()))
	}
	sort.Strings(names)
	return names
}
func (e EnumDescriptor) Name() string { return string(e.Desc.Name()) } // for Callable

// The Call method implements the starlark.Callable interface.
// A call to an enum descriptor converts its argument to a value of that enum type.
func (e EnumDescriptor) CallInternal(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var x starlark.Value
	if err := starlark.UnpackPositionalArgs(string(e.Desc.Name()), args, kwargs, 1, &x); err != nil {
		return nil, err
	}
	v, err := enumValueOf(e.Desc, x)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", e.Desc.Name(), err)
	}
	return EnumValueDescriptor{Desc: v}, nil
}

// enumValueOf converts an int, string, or enum value to a value of the specified enum type.
func enumValueOf(enum protoreflect.EnumDescriptor, x starlark.Value) (protoreflect.EnumValueDescriptor, error) {
	switch x := x.(type) {
	case starlark.Int:
		i, err := starlark.AsInt32(x)
		if err != nil {
			return nil, fmt.Errorf("invalid number %s for %s enum", x, enum.Name())
		}
		desc := enum.Values().ByNumber(protoreflect.EnumNumber(i))
		if desc == nil {
			return nil, fmt.Errorf("invalid number %d for %s enum", i, enum.Name())
		}
		return desc, nil

	case starlark.String:
		name := protoreflect.Name(x)
		desc := enum.Values().ByName(name)
		if desc == nil {
			return nil, fmt.Errorf("invalid name %q for %s enum", name, enum.Name())
		}
		return desc, nil

	case EnumValueDescriptor:
		if parent := x.Desc.Parent(); parent != enum {
			return nil, fmt.Errorf("invalid value %s.%s for %s enum",
				parent.Name(), x.Desc.Name(), enum.Name())
		}
		return x.Desc, nil
	}

	return nil, fmt.Errorf("cannot convert %s to %s enum", x.Type(), enum.Name())
}

// An EnumValueDescriptor is an immutable Starlark value that represents one value of an enumeration.
//
// An EnumValueDescriptor contains a reference to a protoreflect.EnumValueDescriptor.
// Two EnumValueDescriptor values compare equal if and only if they
// refer to the same protoreflect.EnumValueDescriptor.
//
// An EnumValueDescriptor has the following fields:
//
//      index   -- int, index of this value within the enum sequence
//      name    -- string, name of this enum value
//      number  -- int, numeric value of this enum value
//      type    -- EnumDescriptor, the enum type to which this value belongs
//
type EnumValueDescriptor struct {
	Desc protoreflect.EnumValueDescriptor
}

var (
	_ starlark.HasAttrs   = EnumValueDescriptor{}
	_ starlark.Comparable = EnumValueDescriptor{}
)

func (e EnumValueDescriptor) String() string {
	enum := e.Desc.Parent()
	return string(enum.Name() + "." + e.Desc.Name()) // "Enum.EnumValue"
}
func (e EnumValueDescriptor) Type() string                { return "proto.EnumValueDescriptor" }
func (e EnumValueDescriptor) Truth() starlark.Bool        { return true }
func (e EnumValueDescriptor) Freeze()                     {} // immutable
func (e EnumValueDescriptor) Hash() (h uint32, err error) { return uint32(e.Desc.Number()), nil }
func (e EnumValueDescriptor) AttrNames() []string {
	return []string{"index", "name", "number", "type"}
}
func (e EnumValueDescriptor) Attr(name string) (starlark.Value, error) {
	switch name {
	case "index":
		return starlark.MakeInt(e.Desc.Index()), nil
	case "name":
		return starlark.String(e.Desc.Name()), nil
	case "number":
		return starlark.MakeInt(int(e.Desc.Number())), nil
	case "type":
		enum := e.Desc.Parent()
		return EnumDescriptor{Desc: enum.(protoreflect.EnumDescriptor)}, nil
	}
	return nil, nil
}
func (x EnumValueDescriptor) CompareSameType(op syntax.Token, y_ starlark.Value, depth int) (bool, error) {
	y := y_.(EnumValueDescriptor)
	switch op {
	case syntax.EQL:
		return x.Desc == y.Desc, nil
	case syntax.NEQ:
		return x.Desc != y.Desc, nil
	default:
		return false, fmt.Errorf("%s %s %s not implemented", x.Type(), op, y_.Type())
	}
}
