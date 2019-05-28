package compile

import (
	"bytes"
	"fmt"
	"testing"

	"go.starlark.net/resolve"
	"go.starlark.net/syntax"
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
			`"a" + "b" + "c" + "d"`,
			`constant "abcd"; return`,
		},
		{
			// string folding with variable:
			`"a" + "b" + x + "c" + "d"`,
			`constant "ab"; predeclared x; plus; constant "cd"; plus; return`,
		},
		{
			// list folding
			`[1] + [2] + [3]`,
			`constant 1; constant 2; constant 3; makelist<3>; return`,
		},
		{
			// list folding with variable
			`[1] + [2] + x + [3]`,
			`constant 1; constant 2; makelist<2>; ` +
				`predeclared x; plus; ` +
				`constant 3; makelist<1>; plus; ` +
				`return`,
		},
		{
			// tuple folding
			`() + (1,) + (2, 3)`,
			`constant 1; constant 2; constant 3; maketuple<3>; return`,
		},
		{
			// tuple folding with variable
			`() + (1,) + x + (2, 3)`,
			`constant 1; maketuple<1>; predeclared x; plus; ` +
				`constant 2; constant 3; maketuple<2>; plus; ` +
				`return`,
		},
	} {
		expr, err := syntax.ParseExpr("in.star", test.src, 0)
		if err != nil {
			t.Errorf("#%d: %v", i, err)
			continue
		}
		locals, err := resolve.Expr(expr, isPredeclared, isUniversal)
		if err != nil {
			t.Errorf("#%d: %v", i, err)
			continue
		}
		got := disassemble(Expr(expr, "<expr>", locals).Toplevel)
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
			case CONSTANT:
				switch x := f.Prog.Constants[arg].(type) {
				case string:
					fmt.Fprintf(out, " %q", x)
				default:
					fmt.Fprintf(out, " %v", x)
				}
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
