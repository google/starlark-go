// Copyright 2017 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syntax

import (
	"bytes"
	"fmt"
	"go/build"
	"io/ioutil"
	"path/filepath"
	"testing"
)

func scan(src interface{}) (tokens string, err error) {
	sc, err := newScanner("foo.star", src, false)
	if err != nil {
		return "", err
	}

	defer sc.recover(&err)

	var buf bytes.Buffer
	var val tokenValue
	for {
		tok := sc.nextToken(&val)

		if buf.Len() > 0 {
			buf.WriteByte(' ')
		}
		switch tok {
		case EOF:
			buf.WriteString("EOF")
		case IDENT:
			buf.WriteString(val.raw)
		case INT:
			if val.bigInt != nil {
				fmt.Fprintf(&buf, "%d", val.bigInt)
			} else {
				fmt.Fprintf(&buf, "%d", val.int)
			}
		case FLOAT:
			fmt.Fprintf(&buf, "%e", val.float)
		case STRING:
			fmt.Fprintf(&buf, "%q", val.string)
		default:
			buf.WriteString(tok.String())
		}
		if tok == EOF {
			break
		}
	}
	return buf.String(), nil
}

func TestScanner(t *testing.T) {
	for _, test := range []struct {
		input, want string
	}{
		{``, "EOF"},
		{`123`, "123 EOF"},
		{`x.y`, "x . y EOF"},
		{`chocolate.éclair`, `chocolate . éclair EOF`},
		{`123 "foo" hello x.y`, `123 "foo" hello x . y EOF`},
		{`print(x)`, "print ( x ) EOF"},
		{`print(x); print(y)`, "print ( x ) ; print ( y ) EOF"},
		{"\nprint(\n1\n)\n", "print ( 1 ) newline EOF"}, // final \n is at toplevel on non-blank line => token
		{`/ // /= //= ///=`, "/ // /= //= // /= EOF"},
		{`# hello
print(x)`, "print ( x ) EOF"},
		{`# hello
print(1)
cc_binary(name="foo")
def f(x):
		return x+1
print(1)
`,
			`print ( 1 ) newline ` +
				`cc_binary ( name = "foo" ) newline ` +
				`def f ( x ) : newline ` +
				`indent return x + 1 newline ` +
				`outdent print ( 1 ) newline ` +
				`EOF`},
		// EOF should act line an implicit newline.
		{`def f(): pass`,
			"def f ( ) : pass EOF"},
		{`def f():
	pass`,
			"def f ( ) : newline indent pass newline outdent EOF"},
		{`def f():
	pass
# oops`,
			"def f ( ) : newline indent pass newline outdent EOF"},
		{`def f():
	pass \
`,
			"def f ( ) : newline indent pass newline outdent EOF"},
		{`def f():
	pass
`,
			"def f ( ) : newline indent pass newline outdent EOF"},
		{`pass


pass`, "pass newline pass EOF"}, // consecutive newlines are consolidated
		{`def f():
    pass
    `, "def f ( ) : newline indent pass newline outdent EOF"},
		{`def f():
    pass
    ` + "\n", "def f ( ) : newline indent pass newline outdent EOF"},
		{"pass", "pass EOF"},
		{"pass\n", "pass newline EOF"},
		{"pass\n ", "pass newline EOF"},
		{"pass\n \n", "pass newline EOF"},
		{"if x:\n  pass\n ", "if x : newline indent pass newline outdent EOF"},
		{`x = 1 + \
2`, `x = 1 + 2 EOF`},
		{`x = 'a\nb'`, `x = "a\nb" EOF`},
		{`x = r'a\nb'`, `x = "a\\nb" EOF`},
		{"x = 'a\\\nb'", `x = "ab" EOF`},
		{`x = '\''`, `x = "'" EOF`},
		{`x = "\""`, `x = "\"" EOF`},
		{`x = r'\''`, `x = "\\'" EOF`},
		{`x = '''\''''`, `x = "'" EOF`},
		{`x = r'''\''''`, `x = "\\'" EOF`},
		{`x = ''''a'b'c'''`, `x = "'a'b'c" EOF`},
		{"x = '''a\nb'''", `x = "a\nb" EOF`},
		{"x = '''a\rb'''", `x = "a\nb" EOF`},
		{"x = '''a\r\nb'''", `x = "a\nb" EOF`},
		{"x = '''a\n\rb'''", `x = "a\n\nb" EOF`},
		{"x = r'a\\\nb'", `x = "a\\\nb" EOF`},
		{"x = r'a\\\rb'", `x = "a\\\nb" EOF`},
		{"x = r'a\\\r\nb'", `x = "a\\\nb" EOF`},
		{"a\rb", `a newline b EOF`},
		{"a\nb", `a newline b EOF`},
		{"a\r\nb", `a newline b EOF`},
		{"a\n\nb", `a newline b EOF`},
		// numbers
		{"0", `0 EOF`},
		{"00", `0 EOF`},
		{"0.", `0.000000e+00 EOF`},
		{"0.e1", `0.000000e+00 EOF`},
		{".0", `0.000000e+00 EOF`},
		{"0.0", `0.000000e+00 EOF`},
		{".e1", `. e1 EOF`},
		{"1", `1 EOF`},
		{"1.", `1.000000e+00 EOF`},
		{".1", `1.000000e-01 EOF`},
		{".1e1", `1.000000e+00 EOF`},
		{".1e+1", `1.000000e+00 EOF`},
		{".1e-1", `1.000000e-02 EOF`},
		{"1e1", `1.000000e+01 EOF`},
		{"1e+1", `1.000000e+01 EOF`},
		{"1e-1", `1.000000e-01 EOF`},
		{"123", `123 EOF`},
		{"123e45", `1.230000e+47 EOF`},
		{"999999999999999999999999999999999999999999999999999", `999999999999999999999999999999999999999999999999999 EOF`},
		{"12345678901234567890", `12345678901234567890 EOF`},
		// hex
		{"0xA", `10 EOF`},
		{"0xAAG", `170 G EOF`},
		{"0xG", `foo.star:1:1: invalid hex literal`},
		{"0XA", `10 EOF`},
		{"0XG", `foo.star:1:1: invalid hex literal`},
		{"0xA.", `10 . EOF`},
		{"0xA.e1", `10 . e1 EOF`},
		{"0x12345678deadbeef12345678", `5634002672576678570168178296 EOF`},
		// binary
		{"0b1010", `10 EOF`},
		{"0B111101", `61 EOF`},
		{"0b3", `foo.star:1:3: invalid binary literal`},
		{"0b1010201", `10 201 EOF`},
		{"0b1010.01", `10 1.000000e-02 EOF`},
		{"0b0000", `0 EOF`},
		// octal
		{"0o123", `83 EOF`},
		{"0o12834", `10 834 EOF`},
		{"0o12934", `10 934 EOF`},
		{"0o12934.", `10 9.340000e+02 EOF`},
		{"0o12934.1", `10 9.341000e+02 EOF`},
		{"0o12934e1", `10 9.340000e+03 EOF`},
		{"0o123.", `83 . EOF`},
		{"0o123.1", `83 1.000000e-01 EOF`},
		{"0123", `foo.star:1:5: obsolete form of octal literal; use 0o123`},
		{"012834", `foo.star:1:1: invalid int literal`},
		{"012934", `foo.star:1:1: invalid int literal`},
		{"i = 012934", `foo.star:1:5: invalid int literal`},
		// octal escapes in string literals
		{`"\037"`, `"\x1f" EOF`},
		{`"\377"`, `"\xff" EOF`},
		{`"\378"`, `"\x1f8" EOF`},                                // = '\37' + '8'
		{`"\400"`, `foo.star:1:1: invalid escape sequence \400`}, // unlike Python 2 and 3

		// backslash escapes
		// As in Go, a backslash must escape something.
		// (Python started issuing a deprecation warning in 3.6.)
		{`"foo\(bar"`, `foo.star:1:1: invalid escape sequence \(`},
		{`"\+"`, `foo.star:1:1: invalid escape sequence \+`},
		{`"\w"`, `foo.star:1:1: invalid escape sequence \w`},
		{`"\""`, `"\"" EOF`},
		{`"\'"`, `"'" EOF`},
		{`'\w'`, `foo.star:1:1: invalid escape sequence \w`},
		{`'\''`, `"'" EOF`},
		{`'\"'`, `"\"" EOF`},
		{`"""\w"""`, `foo.star:1:1: invalid escape sequence \w`},
		{`"""\""""`, `"\"" EOF`},
		{`"""\'"""`, `"'" EOF`},
		{`'''\w'''`, `foo.star:1:1: invalid escape sequence \w`},
		{`'''\''''`, `"'" EOF`},
		{`'''\"'''`, `"\"" EOF`},
		{`r"\w"`, `"\\w" EOF`},
		{`r"\""`, `"\\\"" EOF`},
		{`r"\'"`, `"\\'" EOF`},
		{`r'\w'`, `"\\w" EOF`},
		{`r'\''`, `"\\'" EOF`},
		{`r'\"'`, `"\\\"" EOF`},
		{`'a\zb'`, `foo.star:1:1: invalid escape sequence \z`},
		{`"\o123"`, `foo.star:1:1: invalid escape sequence \o`},
		// floats starting with octal digits
		{"012934.", `1.293400e+04 EOF`},
		{"012934.1", `1.293410e+04 EOF`},
		{"012934e1", `1.293400e+05 EOF`},
		{"0123.", `1.230000e+02 EOF`},
		{"0123.1", `1.231000e+02 EOF`},
		// github.com/google/skylark/issues/16
		{"x ! 0", "foo.star:1:3: unexpected input character '!'"},
		// github.com/google/starlark-go/issues/80
		{"([{<>}])", "( [ { < > } ] ) EOF"},
		{"f();", "f ( ) ; EOF"},
		// github.com/google/starlark-go/issues/104
		{"def f():\n  if x:\n    pass\n  ", `def f ( ) : newline indent if x : newline indent pass newline outdent outdent EOF`},
		{`while cond: pass`, "while cond : pass EOF"},
		// github.com/google/starlark-go/issues/107
		{"~= ~= 5", "~ = ~ = 5 EOF"},
		{"0in", "0 in EOF"},
		{"0or", "foo.star:1:3: invalid octal literal"},
		{"6in", "6 in EOF"},
		{"6or", "6 or EOF"},
	} {
		got, err := scan(test.input)
		if err != nil {
			got = err.(Error).Error()
		}
		if test.want != got {
			t.Errorf("scan `%s` = [%s], want [%s]", test.input, got, test.want)
		}
	}
}

// dataFile is the same as starlarktest.DataFile.
// We make a copy to avoid a dependency cycle.
var dataFile = func(pkgdir, filename string) string {
	return filepath.Join(build.Default.GOPATH, "src/go.starlark.net", pkgdir, filename)
}

func BenchmarkScan(b *testing.B) {
	filename := dataFile("syntax", "testdata/scan.star")
	b.StopTimer()
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		b.Fatal(err)
	}
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		sc, err := newScanner(filename, data, false)
		if err != nil {
			b.Fatal(err)
		}
		var val tokenValue
		for sc.nextToken(&val) != EOF {
		}
	}
}
