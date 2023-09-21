// Copyright 2017 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syntax_test

import (
	"bufio"
	"bytes"
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.starlark.net/internal/chunkedfile"
	"go.starlark.net/starlarktest"
	"go.starlark.net/syntax"
)

func TestExprParseTrees(t *testing.T) {
	for _, test := range []struct {
		input, want string
	}{
		{`print(1)`,
			`(CallExpr Fn=print Args=(1))`},
		{"print(1)\n",
			`(CallExpr Fn=print Args=(1))`},
		{`x + 1`,
			`(BinaryExpr X=x Op=+ Y=1)`},
		{`[x for x in y]`,
			`(Comprehension Body=x Clauses=((ForClause Vars=x X=y)))`},
		{`[x for x in (a if b else c)]`,
			`(Comprehension Body=x Clauses=((ForClause Vars=x X=(ParenExpr X=(CondExpr Cond=b True=a False=c)))))`},
		{`x[i].f(42)`,
			`(CallExpr Fn=(DotExpr X=(IndexExpr X=x Y=i) Name=f) Args=(42))`},
		{`x.f()`,
			`(CallExpr Fn=(DotExpr X=x Name=f))`},
		{`x+y*z`,
			`(BinaryExpr X=x Op=+ Y=(BinaryExpr X=y Op=* Y=z))`},
		{`x%y-z`,
			`(BinaryExpr X=(BinaryExpr X=x Op=% Y=y) Op=- Y=z)`},
		{`a + b not in c`,
			`(BinaryExpr X=(BinaryExpr X=a Op=+ Y=b) Op=not in Y=c)`},
		{`lambda x, *args, **kwargs: None`,
			`(LambdaExpr Params=(x (UnaryExpr Op=* X=args) (UnaryExpr Op=** X=kwargs)) Body=None)`},
		{`{"one": 1}`,
			`(DictExpr List=((DictEntry Key="one" Value=1)))`},
		{`a[i]`,
			`(IndexExpr X=a Y=i)`},
		{`a[i:]`,
			`(SliceExpr X=a Lo=i)`},
		{`a[:j]`,
			`(SliceExpr X=a Hi=j)`},
		{`a[::]`,
			`(SliceExpr X=a)`},
		{`a[::k]`,
			`(SliceExpr X=a Step=k)`},
		{`[]`,
			`(ListExpr)`},
		{`[1]`,
			`(ListExpr List=(1))`},
		{`[1,]`,
			`(ListExpr List=(1))`},
		{`[1, 2]`,
			`(ListExpr List=(1 2))`},
		{`()`,
			`(TupleExpr)`},
		{`(4,)`,
			`(ParenExpr X=(TupleExpr List=(4)))`},
		{`(4)`,
			`(ParenExpr X=4)`},
		{`(4, 5)`,
			`(ParenExpr X=(TupleExpr List=(4 5)))`},
		{`1, 2, 3`,
			`(TupleExpr List=(1 2 3))`},
		{`1, 2,`,
			`unparenthesized tuple with trailing comma`},
		{`{}`,
			`(DictExpr)`},
		{`{"a": 1}`,
			`(DictExpr List=((DictEntry Key="a" Value=1)))`},
		{`{"a": 1,}`,
			`(DictExpr List=((DictEntry Key="a" Value=1)))`},
		{`{"a": 1, "b": 2}`,
			`(DictExpr List=((DictEntry Key="a" Value=1) (DictEntry Key="b" Value=2)))`},
		{`{x: y for (x, y) in z}`,
			`(Comprehension Curly Body=(DictEntry Key=x Value=y) Clauses=((ForClause Vars=(ParenExpr X=(TupleExpr List=(x y))) X=z)))`},
		{`{x: y for a in b if c}`,
			`(Comprehension Curly Body=(DictEntry Key=x Value=y) Clauses=((ForClause Vars=a X=b) (IfClause Cond=c)))`},
		{`-1 + +2`,
			`(BinaryExpr X=(UnaryExpr Op=- X=1) Op=+ Y=(UnaryExpr Op=+ X=2))`},
		{`"foo" + "bar"`,
			`(BinaryExpr X="foo" Op=+ Y="bar")`},
		{`-1 * 2`, // prec(unary -) > prec(binary *)
			`(BinaryExpr X=(UnaryExpr Op=- X=1) Op=* Y=2)`},
		{`-x[i]`, // prec(unary -) < prec(x[i])
			`(UnaryExpr Op=- X=(IndexExpr X=x Y=i))`},
		{`a | b & c | d`, // prec(|) < prec(&)
			`(BinaryExpr X=(BinaryExpr X=a Op=| Y=(BinaryExpr X=b Op=& Y=c)) Op=| Y=d)`},
		{`a or b and c or d`,
			`(BinaryExpr X=(BinaryExpr X=a Op=or Y=(BinaryExpr X=b Op=and Y=c)) Op=or Y=d)`},
		{`a and b or c and d`,
			`(BinaryExpr X=(BinaryExpr X=a Op=and Y=b) Op=or Y=(BinaryExpr X=c Op=and Y=d))`},
		{`f(1, x=y)`,
			`(CallExpr Fn=f Args=(1 (BinaryExpr X=x Op== Y=y)))`},
		{`f(*args, **kwargs)`,
			`(CallExpr Fn=f Args=((UnaryExpr Op=* X=args) (UnaryExpr Op=** X=kwargs)))`},
		{`lambda *args, *, x=1, **kwargs: 0`,
			`(LambdaExpr Params=((UnaryExpr Op=* X=args) (UnaryExpr Op=*) (BinaryExpr X=x Op== Y=1) (UnaryExpr Op=** X=kwargs)) Body=0)`},
		{`lambda *, a, *b: 0`,
			`(LambdaExpr Params=((UnaryExpr Op=*) a (UnaryExpr Op=* X=b)) Body=0)`},
		{`a if b else c`,
			`(CondExpr Cond=b True=a False=c)`},
		{`a and not b`,
			`(BinaryExpr X=a Op=and Y=(UnaryExpr Op=not X=b))`},
		{`[e for x in y if cond1 if cond2]`,
			`(Comprehension Body=e Clauses=((ForClause Vars=x X=y) (IfClause Cond=cond1) (IfClause Cond=cond2)))`}, // github.com/google/skylark/issues/53
	} {
		e, err := syntax.ParseExpr("foo.star", test.input, 0)
		var got string
		if err != nil {
			got = stripPos(err)
		} else {
			got = treeString(e)
		}
		if test.want != got {
			t.Errorf("parse `%s` = %s, want %s", test.input, got, test.want)
		}
	}
}

func TestStmtParseTrees(t *testing.T) {
	for _, test := range []struct {
		input, want string
	}{
		{`print(1)`,
			`(ExprStmt X=(CallExpr Fn=print Args=(1)))`},
		{`return 1, 2`,
			`(ReturnStmt Result=(TupleExpr List=(1 2)))`},
		{`return`,
			`(ReturnStmt)`},
		{`for i in "abc": break`,
			`(ForStmt Vars=i X="abc" Body=((BranchStmt Token=break)))`},
		{`for i in "abc": continue`,
			`(ForStmt Vars=i X="abc" Body=((BranchStmt Token=continue)))`},
		{`for x, y in z: pass`,
			`(ForStmt Vars=(TupleExpr List=(x y)) X=z Body=((BranchStmt Token=pass)))`},
		{`if True: pass`,
			`(IfStmt Cond=True True=((BranchStmt Token=pass)))`},
		{`if True: break`,
			`(IfStmt Cond=True True=((BranchStmt Token=break)))`},
		{`if True: continue`,
			`(IfStmt Cond=True True=((BranchStmt Token=continue)))`},
		{`if True: pass
else:
	pass`,
			`(IfStmt Cond=True True=((BranchStmt Token=pass)) False=((BranchStmt Token=pass)))`},
		{"if a: pass\nelif b: pass\nelse: pass",
			`(IfStmt Cond=a True=((BranchStmt Token=pass)) False=((IfStmt Cond=b True=((BranchStmt Token=pass)) False=((BranchStmt Token=pass)))))`},
		{`x, y = 1, 2`,
			`(AssignStmt Op== LHS=(TupleExpr List=(x y)) RHS=(TupleExpr List=(1 2)))`},
		{`x[i] = 1`,
			`(AssignStmt Op== LHS=(IndexExpr X=x Y=i) RHS=1)`},
		{`x.f = 1`,
			`(AssignStmt Op== LHS=(DotExpr X=x Name=f) RHS=1)`},
		{`(x, y) = 1`,
			`(AssignStmt Op== LHS=(ParenExpr X=(TupleExpr List=(x y))) RHS=1)`},
		{`load("", "a", b="c")`,
			`(LoadStmt Module="" From=(a c) To=(a b))`},
		{`if True: load("", "a", b="c")`, // load needn't be at toplevel
			`(IfStmt Cond=True True=((LoadStmt Module="" From=(a c) To=(a b))))`},
		{`def f(x, *args, **kwargs):
	pass`,
			`(DefStmt Name=f Params=(x (UnaryExpr Op=* X=args) (UnaryExpr Op=** X=kwargs)) Body=((BranchStmt Token=pass)))`},
		{`def f(**kwargs, *args): pass`,
			`(DefStmt Name=f Params=((UnaryExpr Op=** X=kwargs) (UnaryExpr Op=* X=args)) Body=((BranchStmt Token=pass)))`},
		{`def f(a, b, c=d): pass`,
			`(DefStmt Name=f Params=(a b (BinaryExpr X=c Op== Y=d)) Body=((BranchStmt Token=pass)))`},
		{`def f(a, b=c, d): pass`,
			`(DefStmt Name=f Params=(a (BinaryExpr X=b Op== Y=c) d) Body=((BranchStmt Token=pass)))`}, // TODO(adonovan): fix this
		{`def f():
	def g():
		pass
	pass
def h():
	pass`,
			`(DefStmt Name=f Body=((DefStmt Name=g Body=((BranchStmt Token=pass))) (BranchStmt Token=pass)))`},
		{"f();g()",
			`(ExprStmt X=(CallExpr Fn=f))`},
		{"f();",
			`(ExprStmt X=(CallExpr Fn=f))`},
		{"f();g()\n",
			`(ExprStmt X=(CallExpr Fn=f))`},
		{"f();\n",
			`(ExprStmt X=(CallExpr Fn=f))`},
	} {
		f, err := syntax.Parse("foo.star", test.input, 0)
		if err != nil {
			t.Errorf("parse `%s` failed: %v", test.input, stripPos(err))
			continue
		}
		if got := treeString(f.Stmts[0]); test.want != got {
			t.Errorf("parse `%s` = %s, want %s", test.input, got, test.want)
		}
	}
}

// TestFileParseTrees tests sequences of statements, and particularly
// handling of indentation, newlines, line continuations, and blank lines.
func TestFileParseTrees(t *testing.T) {
	for _, test := range []struct {
		input, want string
	}{
		{`x = 1
print(x)`,
			`(AssignStmt Op== LHS=x RHS=1)
(ExprStmt X=(CallExpr Fn=print Args=(x)))`},
		{"if cond:\n\tpass",
			`(IfStmt Cond=cond True=((BranchStmt Token=pass)))`},
		{"if cond:\n\tpass\nelse:\n\tpass",
			`(IfStmt Cond=cond True=((BranchStmt Token=pass)) False=((BranchStmt Token=pass)))`},
		{`def f():
	pass
pass

pass`,
			`(DefStmt Name=f Body=((BranchStmt Token=pass)))
(BranchStmt Token=pass)
(BranchStmt Token=pass)`},
		{`pass; pass`,
			`(BranchStmt Token=pass)
(BranchStmt Token=pass)`},
		{"pass\npass",
			`(BranchStmt Token=pass)
(BranchStmt Token=pass)`},
		{"pass\n\npass",
			`(BranchStmt Token=pass)
(BranchStmt Token=pass)`},
		{`x = (1 +
2)`,
			`(AssignStmt Op== LHS=x RHS=(ParenExpr X=(BinaryExpr X=1 Op=+ Y=2)))`},
		{`x = 1 \
+ 2`,
			`(AssignStmt Op== LHS=x RHS=(BinaryExpr X=1 Op=+ Y=2))`},
	} {
		f, err := syntax.Parse("foo.star", test.input, 0)
		if err != nil {
			t.Errorf("parse `%s` failed: %v", test.input, stripPos(err))
			continue
		}
		var buf bytes.Buffer
		for i, stmt := range f.Stmts {
			if i > 0 {
				buf.WriteByte('\n')
			}
			writeTree(&buf, reflect.ValueOf(stmt))
		}
		if got := buf.String(); test.want != got {
			t.Errorf("parse `%s` = %s, want %s", test.input, got, test.want)
		}
	}
}

// TestCompoundStmt tests handling of REPL-style compound statements.
func TestCompoundStmt(t *testing.T) {
	for _, test := range []struct {
		input, want string
	}{
		// blank lines
		{"\n",
			``},
		{"   \n",
			``},
		{"# comment\n",
			``},
		// simple statement
		{"1\n",
			`(ExprStmt X=1)`},
		{"print(1)\n",
			`(ExprStmt X=(CallExpr Fn=print Args=(1)))`},
		{"1;2;3;\n",
			`(ExprStmt X=1)(ExprStmt X=2)(ExprStmt X=3)`},
		{"f();g()\n",
			`(ExprStmt X=(CallExpr Fn=f))(ExprStmt X=(CallExpr Fn=g))`},
		{"f();\n",
			`(ExprStmt X=(CallExpr Fn=f))`},
		{"f(\n\n\n\n\n\n\n)\n",
			`(ExprStmt X=(CallExpr Fn=f))`},
		// complex statements
		{"def f():\n  pass\n\n",
			`(DefStmt Name=f Body=((BranchStmt Token=pass)))`},
		{"if cond:\n  pass\n\n",
			`(IfStmt Cond=cond True=((BranchStmt Token=pass)))`},
		// Even as a 1-liner, the following blank line is required.
		{"if cond: pass\n\n",
			`(IfStmt Cond=cond True=((BranchStmt Token=pass)))`},
		// github.com/google/starlark-go/issues/121
		{"a; b; c\n",
			`(ExprStmt X=a)(ExprStmt X=b)(ExprStmt X=c)`},
		{"a; b c\n",
			`invalid syntax`},
	} {

		// Fake readline input from string.
		// The ! suffix, which would cause a parse error,
		// tests that the parser doesn't read more than necessary.
		sc := bufio.NewScanner(strings.NewReader(test.input + "!"))
		readline := func() ([]byte, error) {
			if sc.Scan() {
				return []byte(sc.Text() + "\n"), nil
			}
			return nil, sc.Err()
		}

		var got string
		f, err := syntax.ParseCompoundStmt("foo.star", readline)
		if err != nil {
			got = stripPos(err)
		} else {
			for _, stmt := range f.Stmts {
				got += treeString(stmt)
			}
		}
		if test.want != got {
			t.Errorf("parse `%s` = %s, want %s", test.input, got, test.want)
		}
	}
}

func stripPos(err error) string {
	s := err.Error()
	if i := strings.Index(s, ": "); i >= 0 {
		s = s[i+len(": "):] // strip file:line:col
	}
	return s
}

// treeString prints a syntax node as a parenthesized tree.
// Idents are printed as foo and Literals as "foo" or 42.
// Structs are printed as (type name=value ...).
// Only non-empty fields are shown.
func treeString(n syntax.Node) string {
	var buf bytes.Buffer
	writeTree(&buf, reflect.ValueOf(n))
	return buf.String()
}

func writeTree(out *bytes.Buffer, x reflect.Value) {
	switch x.Kind() {
	case reflect.String, reflect.Int, reflect.Bool:
		fmt.Fprintf(out, "%v", x.Interface())
	case reflect.Ptr, reflect.Interface:
		if elem := x.Elem(); elem.Kind() == 0 {
			out.WriteString("nil")
		} else {
			writeTree(out, elem)
		}
	case reflect.Struct:
		switch v := x.Interface().(type) {
		case syntax.Literal:
			switch v.Token {
			case syntax.STRING:
				fmt.Fprintf(out, "%q", v.Value)
			case syntax.BYTES:
				fmt.Fprintf(out, "b%q", v.Value)
			case syntax.INT:
				fmt.Fprintf(out, "%d", v.Value)
			}
			return
		case syntax.Ident:
			out.WriteString(v.Name)
			return
		}
		fmt.Fprintf(out, "(%s", strings.TrimPrefix(x.Type().String(), "syntax."))
		for i, n := 0, x.NumField(); i < n; i++ {
			f := x.Field(i)
			if f.Type() == reflect.TypeOf(syntax.Position{}) {
				continue // skip positions
			}
			name := x.Type().Field(i).Name
			if name == "commentsRef" {
				continue // skip comments fields
			}
			if f.Type() == reflect.TypeOf(syntax.Token(0)) {
				fmt.Fprintf(out, " %s=%s", name, f.Interface())
				continue
			}

			switch f.Kind() {
			case reflect.Slice:
				if n := f.Len(); n > 0 {
					fmt.Fprintf(out, " %s=(", name)
					for i := 0; i < n; i++ {
						if i > 0 {
							out.WriteByte(' ')
						}
						writeTree(out, f.Index(i))
					}
					out.WriteByte(')')
				}
				continue
			case reflect.Ptr, reflect.Interface:
				if f.IsNil() {
					continue
				}
			case reflect.Int:
				if f.Int() != 0 {
					fmt.Fprintf(out, " %s=%d", name, f.Int())
				}
				continue
			case reflect.Bool:
				if f.Bool() {
					fmt.Fprintf(out, " %s", name)
				}
				continue
			}
			fmt.Fprintf(out, " %s=", name)
			writeTree(out, f)
		}
		fmt.Fprintf(out, ")")
	default:
		fmt.Fprintf(out, "%T", x.Interface())
	}
}

func TestParseErrors(t *testing.T) {
	filename := starlarktest.DataFile("syntax", "testdata/errors.star")
	for _, chunk := range chunkedfile.Read(filename, t) {
		_, err := syntax.Parse(filename, chunk.Source, 0)
		switch err := err.(type) {
		case nil:
			// ok
		case syntax.Error:
			chunk.GotError(int(err.Pos.Line), err.Msg)
		default:
			t.Error(err)
		}
		chunk.Done()
	}
}

func TestFilePortion(t *testing.T) {
	// Imagine that the Starlark file or expression print(x.f) is extracted
	// from the middle of a file in some hypothetical template language;
	// see https://github.com/google/starlark-go/issues/346. For example:
	// --
	// {{loop x seq}}
	//   {{print(x.f)}}
	// {{end}}
	// --
	fp := syntax.FilePortion{Content: []byte("print(x.f)"), FirstLine: 2, FirstCol: 4}
	file, err := syntax.Parse("foo.template", fp, 0)
	if err != nil {
		t.Fatal(err)
	}
	span := fmt.Sprint(file.Stmts[0].Span())
	want := "foo.template:2:4 foo.template:2:14"
	if span != want {
		t.Errorf("wrong span: got %q, want %q", span, want)
	}
}

// dataFile is the same as starlarktest.DataFile.
// We make a copy to avoid a dependency cycle.
var dataFile = func(pkgdir, filename string) string {
	return filepath.Join(build.Default.GOPATH, "src/go.starlark.net", pkgdir, filename)
}

func BenchmarkParse(b *testing.B) {
	filename := dataFile("syntax", "testdata/scan.star")
	b.StopTimer()
	data, err := os.ReadFile(filename)
	if err != nil {
		b.Fatal(err)
	}
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		_, err := syntax.Parse(filename, data, 0)
		if err != nil {
			b.Fatal(err)
		}
	}
}
