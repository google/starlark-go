// Copyright 2017 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package starlark_test

import (
	"bytes"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"testing"

	"go.starlark.net/internal/chunkedfile"
	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarktest"
	"go.starlark.net/syntax"
)

func init() {
	// The tests make extensive use of these not-yet-standard features.
	resolve.AllowLambda = true
	resolve.AllowNestedDef = true
	resolve.AllowFloat = true
	resolve.AllowSet = true
	resolve.AllowBitwise = true
}

func TestEvalExpr(t *testing.T) {
	// This is mostly redundant with the new *.star tests.
	// TODO(adonovan): move checks into *.star files and
	// reduce this to a mere unit test of starlark.Eval.
	thread := new(starlark.Thread)
	for _, test := range []struct{ src, want string }{
		{`123`, `123`},
		{`-1`, `-1`},
		{`"a"+"b"`, `"ab"`},
		{`1+2`, `3`},

		// lists
		{`[]`, `[]`},
		{`[1]`, `[1]`},
		{`[1,]`, `[1]`},
		{`[1, 2]`, `[1, 2]`},
		{`[2 * x for x in [1, 2, 3]]`, `[2, 4, 6]`},
		{`[2 * x for x in [1, 2, 3] if x > 1]`, `[4, 6]`},
		{`[(x, y) for x in [1, 2] for y in [3, 4]]`,
			`[(1, 3), (1, 4), (2, 3), (2, 4)]`},
		{`[(x, y) for x in [1, 2] if x == 2 for y in [3, 4]]`,
			`[(2, 3), (2, 4)]`},
		// tuples
		{`()`, `()`},
		{`(1)`, `1`},
		{`(1,)`, `(1,)`},
		{`(1, 2)`, `(1, 2)`},
		{`(1, 2, 3, 4, 5)`, `(1, 2, 3, 4, 5)`},
		// dicts
		{`{}`, `{}`},
		{`{"a": 1}`, `{"a": 1}`},
		{`{"a": 1,}`, `{"a": 1}`},

		// conditional
		{`1 if 3 > 2 else 0`, `1`},
		{`1 if "foo" else 0`, `1`},
		{`1 if "" else 0`, `0`},

		// indexing
		{`["a", "b"][0]`, `"a"`},
		{`["a", "b"][1]`, `"b"`},
		{`("a", "b")[0]`, `"a"`},
		{`("a", "b")[1]`, `"b"`},
		{`"aΩb"[0]`, `"a"`},
		{`"aΩb"[1]`, `"\xce"`},
		{`"aΩb"[3]`, `"b"`},
		{`{"a": 1}["a"]`, `1`},
		{`{"a": 1}["b"]`, `key "b" not in dict`},
		{`{}[[]]`, `unhashable type: list`},
		{`{"a": 1}[[]]`, `unhashable type: list`},
		{`[x for x in range(3)]`, "[0, 1, 2]"},
	} {
		var got string
		if v, err := starlark.Eval(thread, "<expr>", test.src, nil); err != nil {
			got = err.Error()
		} else {
			got = v.String()
		}
		if got != test.want {
			t.Errorf("eval %s = %s, want %s", test.src, got, test.want)
		}
	}
}

func TestExecFile(t *testing.T) {
	testdata := starlarktest.DataFile("starlark", ".")
	thread := &starlark.Thread{Load: load}
	starlarktest.SetReporter(thread, t)
	for _, file := range []string{
		"testdata/assign.star",
		"testdata/bool.star",
		"testdata/builtins.star",
		"testdata/control.star",
		"testdata/dict.star",
		"testdata/float.star",
		"testdata/function.star",
		"testdata/int.star",
		"testdata/list.star",
		"testdata/misc.star",
		"testdata/set.star",
		"testdata/string.star",
		"testdata/tuple.star",
		"testdata/recursion.star",
	} {
		filename := filepath.Join(testdata, file)
		for _, chunk := range chunkedfile.Read(filename, t) {
			predeclared := starlark.StringDict{
				"hasfields": starlark.NewBuiltin("hasfields", newHasFields),
				"fibonacci": fib{},
			}

			resolve.AllowRecursion = option(chunk.Source, "recursion")

			_, err := starlark.ExecFile(thread, filename, chunk.Source, predeclared)
			switch err := err.(type) {
			case *starlark.EvalError:
				found := false
				for _, fr := range err.Stack() {
					posn := fr.Position()
					if posn.Filename() == filename {
						chunk.GotError(int(posn.Line), err.Error())
						found = true
						break
					}
				}
				if !found {
					t.Error(err.Backtrace())
				}
			case nil:
				// success
			default:
				t.Error(err)
			}
			resolve.AllowRecursion = false
			chunk.Done()
		}
	}
}

func option(chunk, name string) bool {
	return strings.Contains(chunk, "option:"+name)
}

// A fib is an iterable value representing the infinite Fibonacci sequence.
type fib struct{}

func (t fib) Freeze()                    {}
func (t fib) String() string             { return "fib" }
func (t fib) Type() string               { return "fib" }
func (t fib) Truth() starlark.Bool       { return true }
func (t fib) Hash() (uint32, error)      { return 0, fmt.Errorf("fib is unhashable") }
func (t fib) Iterate() starlark.Iterator { return &fibIterator{0, 1} }

type fibIterator struct{ x, y int }

func (it *fibIterator) Next(p *starlark.Value) bool {
	*p = starlark.MakeInt(it.x)
	it.x, it.y = it.y, it.x+it.y
	return true
}
func (it *fibIterator) Done() {}

// load implements the 'load' operation as used in the evaluator tests.
func load(thread *starlark.Thread, module string) (starlark.StringDict, error) {
	if module == "assert.star" {
		return starlarktest.LoadAssertModule()
	}

	// TODO(adonovan): test load() using this execution path.
	filename := filepath.Join(filepath.Dir(thread.Caller().Position().Filename()), module)
	return starlark.ExecFile(thread, filename, nil, nil)
}

func newHasFields(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return &hasfields{attrs: make(map[string]starlark.Value)}, nil
}

// hasfields is a test-only implementation of HasAttrs.
// It permits any field to be set.
// Clients will likely want to provide their own implementation,
// so we don't have any public implementation.
type hasfields struct {
	attrs  starlark.StringDict
	frozen bool
}

var (
	_ starlark.HasAttrs  = (*hasfields)(nil)
	_ starlark.HasBinary = (*hasfields)(nil)
)

func (hf *hasfields) String() string        { return "hasfields" }
func (hf *hasfields) Type() string          { return "hasfields" }
func (hf *hasfields) Truth() starlark.Bool  { return true }
func (hf *hasfields) Hash() (uint32, error) { return 42, nil }

func (hf *hasfields) Freeze() {
	if !hf.frozen {
		hf.frozen = true
		for _, v := range hf.attrs {
			v.Freeze()
		}
	}
}

func (hf *hasfields) Attr(name string) (starlark.Value, error) { return hf.attrs[name], nil }

func (hf *hasfields) SetField(name string, val starlark.Value) error {
	if hf.frozen {
		return fmt.Errorf("cannot set field on a frozen hasfields")
	}
	hf.attrs[name] = val
	return nil
}

func (hf *hasfields) AttrNames() []string {
	names := make([]string, 0, len(hf.attrs))
	for key := range hf.attrs {
		names = append(names, key)
	}
	return names
}

func (hf *hasfields) Binary(op syntax.Token, y starlark.Value, side starlark.Side) (starlark.Value, error) {
	// This method exists so we can exercise 'list += x'
	// where x is not Iterable but defines list+x.
	if op == syntax.PLUS {
		if _, ok := y.(*starlark.List); ok {
			return starlark.MakeInt(42), nil // list+hasfields is 42
		}
	}
	return nil, nil
}

func TestParameterPassing(t *testing.T) {
	const filename = "parameters.go"
	const src = `
def a():
	return
def b(a, b):
	return a, b
def c(a, b=42):
	return a, b
def d(*args):
	return args
def e(**kwargs):
	return kwargs
def f(a, b=42, *args, **kwargs):
	return a, b, args, kwargs
`

	thread := new(starlark.Thread)
	globals, err := starlark.ExecFile(thread, filename, src, nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct{ src, want string }{
		{`a()`, `None`},
		{`a(1)`, `function a takes no arguments (1 given)`},
		{`b()`, `function b takes exactly 2 positional arguments (0 given)`},
		{`b(1)`, `function b takes exactly 2 positional arguments (1 given)`},
		{`b(1, 2)`, `(1, 2)`},
		{`b`, `<function b>`}, // asserts that b's parameter b was treated as a local variable
		{`b(1, 2, 3)`, `function b takes exactly 2 positional arguments (3 given)`},
		{`b(1, b=2)`, `(1, 2)`},
		{`b(1, a=2)`, `function b got multiple values for keyword argument "a"`},
		{`b(1, x=2)`, `function b got an unexpected keyword argument "x"`},
		{`b(a=1, b=2)`, `(1, 2)`},
		{`b(b=1, a=2)`, `(2, 1)`},
		{`b(b=1, a=2, x=1)`, `function b got an unexpected keyword argument "x"`},
		{`b(x=1, b=1, a=2)`, `function b got an unexpected keyword argument "x"`},
		{`c()`, `function c takes at least 1 positional argument (0 given)`},
		{`c(1)`, `(1, 42)`},
		{`c(1, 2)`, `(1, 2)`},
		{`c(1, 2, 3)`, `function c takes at most 2 positional arguments (3 given)`},
		{`c(1, b=2)`, `(1, 2)`},
		{`c(1, a=2)`, `function c got multiple values for keyword argument "a"`},
		{`c(a=1, b=2)`, `(1, 2)`},
		{`c(b=1, a=2)`, `(2, 1)`},
		{`d()`, `()`},
		{`d(1)`, `(1,)`},
		{`d(1, 2)`, `(1, 2)`},
		{`d(1, 2, k=3)`, `function d got an unexpected keyword argument "k"`},
		{`d(args=[])`, `function d got an unexpected keyword argument "args"`},
		{`e()`, `{}`},
		{`e(1)`, `function e takes exactly 0 positional arguments (1 given)`},
		{`e(k=1)`, `{"k": 1}`},
		{`e(kwargs={})`, `{"kwargs": {}}`},
		{`f()`, `function f takes at least 1 positional argument (0 given)`},
		{`f(0)`, `(0, 42, (), {})`},
		{`f(0)`, `(0, 42, (), {})`},
		{`f(0, 1)`, `(0, 1, (), {})`},
		{`f(0, 1, 2)`, `(0, 1, (2,), {})`},
		{`f(0, 1, 2, 3)`, `(0, 1, (2, 3), {})`},
		{`f(a=0)`, `(0, 42, (), {})`},
		{`f(0, b=1)`, `(0, 1, (), {})`},
		{`f(0, a=1)`, `function f got multiple values for keyword argument "a"`},
		{`f(0, b=1, c=2)`, `(0, 1, (), {"c": 2})`},
		{`f(0, 1, x=2, *[3, 4], y=5, **dict(z=6))`, // github.com/google/skylark/issues/135
			`(0, 1, (3, 4), {"x": 2, "y": 5, "z": 6})`},
	} {
		var got string
		if v, err := starlark.Eval(thread, "<expr>", test.src, globals); err != nil {
			got = err.Error()
		} else {
			got = v.String()
		}
		if got != test.want {
			t.Errorf("eval %s = %s, want %s", test.src, got, test.want)
		}
	}
}

// TestPrint ensures that the Starlark print function calls
// Thread.Print, if provided.
func TestPrint(t *testing.T) {
	const src = `
print("hello")
def f(): print("hello", "world", sep=", ")
f()
`
	buf := new(bytes.Buffer)
	print := func(thread *starlark.Thread, msg string) {
		caller := thread.Caller()
		fmt.Fprintf(buf, "%s: %s: %s\n",
			caller.Position(), caller.Callable().Name(), msg)
	}
	thread := &starlark.Thread{Print: print}
	if _, err := starlark.ExecFile(thread, "foo.star", src, nil); err != nil {
		t.Fatal(err)
	}
	want := "foo.star:2: <toplevel>: hello\n" +
		"foo.star:3: f: hello, world\n"
	if got := buf.String(); got != want {
		t.Errorf("output was %s, want %s", got, want)
	}
}

func reportEvalError(tb testing.TB, err error) {
	if err, ok := err.(*starlark.EvalError); ok {
		tb.Fatal(err.Backtrace())
	}
	tb.Fatal(err)
}

// TestInt exercises the Int.Int64 and Int.Uint64 methods.
// If we can move their logic into math/big, delete this test.
func TestInt(t *testing.T) {
	one := starlark.MakeInt(1)

	for _, test := range []struct {
		i          starlark.Int
		wantInt64  string
		wantUint64 string
	}{
		{starlark.MakeInt64(math.MinInt64).Sub(one), "error", "error"},
		{starlark.MakeInt64(math.MinInt64), "-9223372036854775808", "error"},
		{starlark.MakeInt64(-1), "-1", "error"},
		{starlark.MakeInt64(0), "0", "0"},
		{starlark.MakeInt64(1), "1", "1"},
		{starlark.MakeInt64(math.MaxInt64), "9223372036854775807", "9223372036854775807"},
		{starlark.MakeUint64(math.MaxUint64), "error", "18446744073709551615"},
		{starlark.MakeUint64(math.MaxUint64).Add(one), "error", "error"},
	} {
		gotInt64, gotUint64 := "error", "error"
		if i, ok := test.i.Int64(); ok {
			gotInt64 = fmt.Sprint(i)
		}
		if u, ok := test.i.Uint64(); ok {
			gotUint64 = fmt.Sprint(u)
		}
		if gotInt64 != test.wantInt64 {
			t.Errorf("(%s).Int64() = %s, want %s", test.i, gotInt64, test.wantInt64)
		}
		if gotUint64 != test.wantUint64 {
			t.Errorf("(%s).Uint64() = %s, want %s", test.i, gotUint64, test.wantUint64)
		}
	}
}

func TestBacktrace(t *testing.T) {
	// This test ensures continuity of the stack of active Starlark
	// functions, including propagation through built-ins such as 'min'
	// (though min does not itself appear in the stack).
	const src = `
def f(x): return 1//x
def g(x): f(x)
def h(): return min([1, 2, 0], key=g)
def i(): return h()
i()
`
	thread := new(starlark.Thread)
	_, err := starlark.ExecFile(thread, "crash.star", src, nil)
	switch err := err.(type) {
	case *starlark.EvalError:
		got := err.Backtrace()
		// Compiled code currently has no column information.
		const want = `Traceback (most recent call last):
  crash.star:6: in <toplevel>
  crash.star:5: in i
  crash.star:4: in h
  <builtin>:1: in min
  crash.star:3: in g
  crash.star:2: in f
Error: floored division by zero`
		if got != want {
			t.Errorf("error was %s, want %s", got, want)
		}
	case nil:
		t.Error("ExecFile succeeded unexpectedly")
	default:
		t.Errorf("ExecFile failed with %v, wanted *EvalError", err)
	}
}

// TestRepeatedExec parses and resolves a file syntax tree once then
// executes it repeatedly with different values of its predeclared variables.
func TestRepeatedExec(t *testing.T) {
	predeclared := starlark.StringDict{"x": starlark.None}
	_, prog, err := starlark.SourceProgram("repeat.star", "y = 2 * x", predeclared.Has)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		x, want starlark.Value
	}{
		{x: starlark.MakeInt(42), want: starlark.MakeInt(84)},
		{x: starlark.String("mur"), want: starlark.String("murmur")},
		{x: starlark.Tuple{starlark.None}, want: starlark.Tuple{starlark.None, starlark.None}},
	} {
		predeclared["x"] = test.x // update the values in dictionary
		thread := new(starlark.Thread)
		if globals, err := prog.Init(thread, predeclared); err != nil {
			t.Errorf("x=%v: %v", test.x, err) // exec error
		} else if eq, err := starlark.Equal(globals["y"], test.want); err != nil {
			t.Errorf("x=%v: %v", test.x, err) // comparison error
		} else if !eq {
			t.Errorf("x=%v: got y=%v, want %v", test.x, globals["y"], test.want)
		}
	}
}

// TestEmptyFilePosition ensures that even Programs
// from empty files have a valid position.
func TestEmptyPosition(t *testing.T) {
	var predeclared starlark.StringDict
	for _, content := range []string{"", "empty = False"} {
		_, prog, err := starlark.SourceProgram("hello.star", content, predeclared.Has)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := prog.Filename(), "hello.star"; got != want {
			t.Errorf("Program.Filename() = %q, want %q", got, want)
		}
	}
}

// TestUnpackUserDefined tests that user-defined
// implementations of starlark.Value may be unpacked.
func TestUnpackUserDefined(t *testing.T) {
	// success
	want := new(hasfields)
	var x *hasfields
	if err := starlark.UnpackArgs("unpack", starlark.Tuple{want}, nil, "x", &x); err != nil {
		t.Errorf("UnpackArgs failed: %v", err)
	}
	if x != want {
		t.Errorf("for x, got %v, want %v", x, want)
	}

	// failure
	err := starlark.UnpackArgs("unpack", starlark.Tuple{starlark.MakeInt(42)}, nil, "x", &x)
	if want := "unpack: for parameter 1: got int, want hasfields"; fmt.Sprint(err) != want {
		t.Errorf("unpack args error = %q, want %q", err, want)
	}
}

func TestDocstring(t *testing.T) {
	globals, _ := starlark.ExecFile(&starlark.Thread{}, "doc.star", `
def somefunc():
	"somefunc doc"
	return 0
`, nil)

	if globals["somefunc"].(*starlark.Function).Doc() != "somefunc doc" {
		t.Fatal("docstring not found")
	}
}

func TestFrameLocals(t *testing.T) {
	// trace prints a nice stack trace including argument
	// values of calls to Starlark functions.
	trace := func(thread *starlark.Thread) string {
		buf := new(bytes.Buffer)
		for fr := thread.TopFrame(); fr != nil; fr = fr.Parent() {
			fmt.Fprintf(buf, "%s(", fr.Callable().Name())
			if fn, ok := fr.Callable().(*starlark.Function); ok {
				for i := 0; i < fn.NumParams(); i++ {
					if i > 0 {
						buf.WriteString(", ")
					}
					name, _ := fn.Param(i)
					fmt.Fprintf(buf, "%s=%s", name, fr.Local(i))
				}
			} else {
				buf.WriteString("...") // a built-in function
			}
			buf.WriteString(")\n")
		}
		return buf.String()
	}

	var got string
	builtin := func(thread *starlark.Thread, _ *starlark.Builtin, _ starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
		got = trace(thread)
		return starlark.None, nil
	}
	predeclared := starlark.StringDict{
		"builtin": starlark.NewBuiltin("builtin", builtin),
	}
	_, err := starlark.ExecFile(&starlark.Thread{}, "foo.star", `
def f(x, y): builtin()
def g(z): f(z, z*z)
g(7)
`, predeclared)
	if err != nil {
		t.Errorf("ExecFile failed: %v", err)
	}

	var want = `
builtin(...)
f(x=7, y=49)
g(z=7)
<toplevel>()
`[1:]
	if got != want {
		t.Errorf("got <<%s>>, want <<%s>>", got, want)
	}
}

type badType string

func (b *badType) String() string        { return "badType" }
func (b *badType) Type() string          { return "badType:" + string(*b) } // panics if b==nil
func (b *badType) Truth() starlark.Bool  { return true }
func (b *badType) Hash() (uint32, error) { return 0, nil }
func (b *badType) Freeze()               {}

var _ starlark.Value = new(badType)

// TestUnpackErrorBadType verifies that the Unpack functions fail
// gracefully when a parameter's default value's Type method panics.
func TestUnpackErrorBadType(t *testing.T) {
	for _, test := range []struct {
		x    *badType
		want string
	}{
		{new(badType), "got NoneType, want badType"},       // Starlark type name
		{nil, "got NoneType, want *starlark_test.badType"}, // Go type name
	} {
		err := starlark.UnpackArgs("f", starlark.Tuple{starlark.None}, nil, "x", &test.x)
		if err == nil {
			t.Errorf("UnpackArgs succeeded unexpectedly")
			continue
		}
		if !strings.Contains(err.Error(), test.want) {
			t.Errorf("UnpackArgs error %q does not contain %q", err, test.want)
		}
	}
}
