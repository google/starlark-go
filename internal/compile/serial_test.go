package compile

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"go.starlark.net/resolve"
	"go.starlark.net/syntax"
)

// compileFile parses, resolves, and compiles a file.
func compileFile(t *testing.T, src string) (*Program, error) {
	t.Helper()
	opts := syntax.LegacyFileOptions()
	f, err := opts.Parse("in.star", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	isPredeclared := func(string) bool { return false }
	if err := resolve.File(f, isPredeclared, isPredeclared); err != nil {
		t.Fatal(err)
	}
	module := f.Module.(*resolve.Module)
	return File(opts, f.Stmts, syntax.Start(f.Stmts[0]), "<toplevel>", module.Locals, module.Globals)
}

// bigFunction returns the source of a function f whose code contains
// n distinct constants and many jumps.
func bigFunction(n int) string {
	var src strings.Builder
	src.WriteString("def f(x):\n\tfor i in x:\n")
	for i := range n {
		fmt.Fprintf(&src, "\t\tif i: x = %d\n", i)
	}
	src.WriteString("\treturn x\n")
	return src.String()
}

// TestCodeEncoding checks that instruction words survive a round
// trip through the serialized form, including operands (constant
// indices and jump targets) that need multi-byte varints.
func TestCodeEncoding(t *testing.T) {
	prog, err := compileFile(t, bigFunction(300))
	if err != nil {
		t.Fatal(err)
	}

	prog2, err := DecodeProgram(prog.Encode())
	if err != nil {
		t.Fatal(err)
	}

	want := prog.Functions[0].Code()
	if max := slices.Max(want) >> 8; max < 1<<7 {
		t.Fatalf("test is ineffective: max operand %d fits in one byte", max)
	}
	if got := prog2.Functions[0].Code(); !slices.Equal(got, want) {
		t.Errorf("decoded code differs from original")
	}
}

// TestOperandLimit checks that an operand too large for an
// instruction word is reported as a compilation error.
func TestOperandLimit(t *testing.T) {
	defer func(prev uint32) { maxArg = prev }(maxArg)
	maxArg = 100

	_, err := compileFile(t, bigFunction(300))
	const want = "in.star:1:1: function f exceeds an implementation limit"
	if err == nil {
		t.Fatalf("compilation succeeded, want error")
	} else if !strings.HasPrefix(err.Error(), want) {
		t.Fatalf("got error %q, want prefix %q", err, want)
	}
}
