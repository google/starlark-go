package compile_test

import (
	"strings"
	"testing"

	"go.starlark.net/internal/compile"
)

// TestOpcodeNames checks that every opcode has a name, including the
// highest-numbered one (a regression test for an off-by-one in
// Opcode.String), and that out-of-range opcodes are reported as
// illegal rather than out of bounds.
func TestOpcodeNames(t *testing.T) {
	for op := compile.Opcode(0); op <= compile.OpcodeMax; op++ {
		if name := op.String(); strings.HasPrefix(name, "illegal op") {
			t.Errorf("opcode %d has no name", op)
		}
	}
	if name := (compile.OpcodeMax + 1).String(); !strings.HasPrefix(name, "illegal op") {
		t.Errorf("Opcode(OpcodeMax+1).String() = %q, want illegal op", name)
	}
}
