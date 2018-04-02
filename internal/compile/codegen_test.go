package compile

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/google/skylark/resolve"
	"github.com/google/skylark/syntax"
)

// TestPlusFolding ensures that the compiler generates optimized code for
// n-ary addition of strings, lists, and tuples.
func TestPlusFolding(t *testing.T) {
	isPredeclared := func(name string) bool { return name == "x" }
	isUniversal := func(name string) bool { return false }
	for i, test := range []struct {
		src  string // source expression
		want string // disassembled code
	}{
		{
			// string folding
			// const + const + const + const => const
			`"a" + "b" + "c" + "d"`,
			`string "abcd"; return`,
		},
		{
			// string folding with variable:
			// const + const + var + const + const => sum(const, var, const)
			`"a" + "b" + x + "c" + "d"`,
			`string "ab"; predeclared x; plus; string "cd"; plus; return`,
		},
		{
			// list folding
			`[1] + [2] + [3]`,
			`int 1; int 2; int 3; makelist<3>; return`,
		},
		{
			// list folding with variable
			`[1] + [2] + x + [3]`,
			`int 1; int 2; makelist<2>; ` +
				`predeclared x; plus; ` +
				`int 3; makelist<1>; plus; ` +
				`return`,
		},
		{
			// tuple folding
			`() + (1,) + (2, 3)`,
			`int 1; int 2; int 3; maketuple<3>; return`,
		},
		{
			// tuple folding with variable
			`() + (1,) + x + (2, 3)`,
			`int 1; maketuple<1>; predeclared x; plus; ` +
				`int 2; int 3; maketuple<2>; plus; ` +
				`return`,
		},
	} {
		expr, err := syntax.ParseExpr("in.sky", test.src, 0)
		if err != nil {
			t.Errorf("#%d: %v", i, err)
			continue
		}
		locals, err := resolve.Expr(expr, isPredeclared, isUniversal)
		if err != nil {
			t.Errorf("#%d: %v", i, err)
			continue
		}
		got := disassemble(Expr(expr, locals))
		if test.want != got {
			t.Errorf("expression <<%s>> generated <<%s>>, want <<%s>>",
				test.src, got, test.want)
		}
	}
}

// disassemble is a trivial disassembler tailored to the accumulator test.
func disassemble(f *Funcode) string {
	out := new(bytes.Buffer)
	code := f.Code
	for pc := 0; pc < len(code); {
		op := Opcode(code[pc])
		pc++
		// TODO(adonovan): factor in common with interpreter.
		var arg uint32
		if op >= OpcodeArgMin {
			for s := uint(0); ; s += 7 {
				b := code[pc]
				pc++
				arg |= uint32(b&0x7f) << s
				if b < 0x80 {
					break
				}
			}
		}

		if out.Len() > 0 {
			out.WriteString("; ")
		}
		fmt.Fprintf(out, "%s", op)
		if op >= OpcodeArgMin {
			switch op {
			case INT, FLOAT, BIGINT:
				fmt.Fprintf(out, " %v", f.Prog.Constants[arg])
			case STRING:
				fmt.Fprintf(out, " %q", f.Prog.Constants[arg])
			case LOCAL:
				fmt.Fprintf(out, " %s", f.Locals[arg].Name)
			case PREDECLARED:
				fmt.Fprintf(out, " %s", f.Prog.Names[arg])
			default:
				fmt.Fprintf(out, "<%d>", arg)
			}
		}
	}
	return out.String()
}
