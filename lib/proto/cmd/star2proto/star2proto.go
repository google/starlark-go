// The star2proto command executes a Starlark file and prints a protocol
// message, which it expects to find in a module-level variable named 'result'.
//
// THIS COMMAND IS EXPERIMENTAL AND ITS INTERFACE MAY CHANGE.
package main

// TODO(adonovan): add features to make this a useful tool for querying,
// converting, and building messages in proto, JSON, and YAML.
// - define operations for reading and writing files.
// - support (e.g.) querying a proto file given a '-e expr' flag.
//   This will need a convenient way to put the relevant descriptors in scope.

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"go.starlark.net/lib/json"
	starlarkproto "go.starlark.net/lib/proto"
	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

// flags
var (
	outputFlag  = flag.String("output", "text", "output format (text, wire, json)")
	varFlag     = flag.String("var", "result", "the variable to output")
	descriptors = flag.String("descriptors", "", "comma-separated list of names of files containing proto.FileDescriptorProto messages")
)

// Starlark dialect flags
func init() {
	flag.BoolVar(&resolve.AllowSet, "set", resolve.AllowSet, "allow set data type")

	// obsolete, no effect:
	flag.BoolVar(&resolve.AllowFloat, "fp", true, "allow floating-point numbers")
	flag.BoolVar(&resolve.AllowLambda, "lambda", resolve.AllowLambda, "allow lambda expressions")
	flag.BoolVar(&resolve.AllowNestedDef, "nesteddef", resolve.AllowNestedDef, "allow nested def statements")
}

func main() {
	log.SetPrefix("star2proto: ")
	log.SetFlags(0)
	flag.Parse()
	if len(flag.Args()) != 1 {
		fatalf("requires a single Starlark file name")
	}
	filename := flag.Args()[0]

	// By default, use the linked-in descriptors
	// (very few in star2proto, e.g. descriptorpb itself).
	pool := protoregistry.GlobalFiles

	// Load a user-provided FileDescriptorSet produced by a command such as:
	// $ protoc --descriptor_set_out=foo.fds foo.proto
	if *descriptors != "" {
		var fdset descriptorpb.FileDescriptorSet
		for i, filename := range strings.Split(*descriptors, ",") {
			data, err := os.ReadFile(filename)
			if err != nil {
				log.Fatalf("--descriptors[%d]: %s", i, err)
			}
			// Accumulate into the repeated field of FileDescriptors.
			if err := (proto.UnmarshalOptions{Merge: true}).Unmarshal(data, &fdset); err != nil {
				log.Fatalf("%s does not contain a proto2.FileDescriptorSet: %v", filename, err)
			}
		}

		files, err := protodesc.NewFiles(&fdset)
		if err != nil {
			log.Fatalf("protodesc.NewFiles: could not build FileDescriptor index: %v", err)
		}
		pool = files
	}

	// Execute the Starlark file.
	thread := &starlark.Thread{
		Print: func(_ *starlark.Thread, msg string) { fmt.Println(msg) },
	}
	starlarkproto.SetPool(thread, pool)
	predeclared := starlark.StringDict{
		"proto": starlarkproto.Module,
		"json":  json.Module,
	}
	globals, err := starlark.ExecFile(thread, filename, nil, predeclared)
	if err != nil {
		if evalErr, ok := err.(*starlark.EvalError); ok {
			fatalf("%s", evalErr.Backtrace())
		} else {
			fatalf("%s", err)
		}
	}

	// Print the output variable as a message.
	// TODO(adonovan): this is clumsy.
	// Let the user call print(), or provide an expression on the command line.
	result, ok := globals[*varFlag]
	if !ok {
		fatalf("%s must define a module-level variable named %q", filename, *varFlag)
	}
	msgwrap, ok := result.(*starlarkproto.Message)
	if !ok {
		fatalf("got %s, want proto.Message, for %q", result.Type(), *varFlag)
	}
	msg := msgwrap.Message()

	// -output
	var marshal func(protoreflect.ProtoMessage) ([]byte, error)
	switch *outputFlag {
	case "wire":
		marshal = proto.Marshal

	case "text":
		marshal = prototext.MarshalOptions{Multiline: true, Indent: "\t"}.Marshal

	case "json":
		marshal = protojson.MarshalOptions{Multiline: true, Indent: "\t"}.Marshal

	default:
		fatalf("unsupported -output format: %s", *outputFlag)
	}
	data, err := marshal(msg)
	if err != nil {
		fatalf("%s", err)
	}
	os.Stdout.Write(data)
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "star2proto: ")
	fmt.Fprintf(os.Stderr, format, args...)
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
}
