// Copyright 2017 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package skylark

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"math/big"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/skylark/resolve"
	"github.com/google/skylark/syntax"
)

const debug = false

// A Thread contains the state of a Skylark thread,
// such as its call stack and thread-local storage.
// The Thread is threaded throughout the evaluator.
type Thread struct {
	// frame is the current Skylark execution frame.
	frame *Frame

	// Print is the client-supplied implementation of the Skylark
	// 'print' function. If nil, fmt.Fprintln(os.Stderr, msg) is
	// used instead.
	Print func(thread *Thread, msg string)

	// Load is the client-supplied implementation of module loading.
	// Repeated calls with the same module name must return the same
	// module environment or error.
	//
	// See example_test.go for some example implementations of Load.
	Load func(thread *Thread, module string) (StringDict, error)

	// locals holds arbitrary "thread-local" values belonging to the client.
	locals map[string]interface{}
}

// SetLocal sets the thread-local value associated with the specified key.
// It must not be called after execution begins.
func (thread *Thread) SetLocal(key string, value interface{}) {
	if thread.locals == nil {
		thread.locals = make(map[string]interface{})
	}
	thread.locals[key] = value
}

// Local returns the thread-local value associated with the specified key.
func (thread *Thread) Local(key string) interface{} {
	return thread.locals[key]
}

// Caller returns the frame of the innermost enclosing Skylark function.
// It should only be used in built-ins called from Skylark code.
func (thread *Thread) Caller() *Frame {
	return thread.frame
}

// A StringDict is a mapping from names to values, and represents
// an environment such as the global variables of a module.
// It is not a true skylark.Value.
type StringDict map[string]Value

func (d StringDict) String() string {
	names := make([]string, 0, len(d))
	for name := range d {
		names = append(names, name)
	}
	sort.Strings(names)

	var buf bytes.Buffer
	path := make([]Value, 0, 4)
	buf.WriteByte('{')
	sep := ""
	for _, name := range names {
		buf.WriteString(sep)
		buf.WriteString(name)
		buf.WriteString(": ")
		writeValue(&buf, d[name], path)
		sep = ", "
	}
	buf.WriteByte('}')
	return buf.String()
}

func (d StringDict) Freeze() {
	for _, v := range d {
		v.Freeze()
	}
}

// Has reports whether the dictionary contains the specified key.
func (d StringDict) Has(key string) bool { _, ok := d[key]; return ok }

// A Frame holds the execution state of a single Skylark function call
// or module toplevel.
type Frame struct {
	thread  *Thread         // thread-associated state
	parent  *Frame          // caller's frame (or nil)
	posn    syntax.Position // source position of PC (set during call and error)
	fn      *Function       // current function (nil at toplevel)
	globals StringDict      // current global environment
	locals  []Value         // local variables, starting with parameters
	result  Value           // operand of current function's return statement
}

func (fr *Frame) errorf(posn syntax.Position, format string, args ...interface{}) *EvalError {
	fr.posn = posn
	msg := fmt.Sprintf(format, args...)
	return &EvalError{Msg: msg, Frame: fr}
}

// Position returns the source position of the current point of execution in this frame.
func (fr *Frame) Position() syntax.Position { return fr.posn }

// Function returns the frame's function, or nil for the top-level of a module.
func (fr *Frame) Function() *Function { return fr.fn }

// Parent returns the frame of the enclosing function call, if any.
func (fr *Frame) Parent() *Frame { return fr.parent }

// set updates the environment binding for name to value.
func (fr *Frame) set(id *syntax.Ident, v Value) {
	switch resolve.Scope(id.Scope) {
	case resolve.Local:
		fr.locals[id.Index] = v
	case resolve.Global:
		fr.globals[id.Name] = v
	default:
		log.Fatalf("%s: set(%s): neither global nor local (%d)", id.NamePos, id.Name, id.Scope)
	}
}

// lookup returns the value of name in the environment.
func (fr *Frame) lookup(id *syntax.Ident) (Value, error) {
	switch resolve.Scope(id.Scope) {
	case resolve.Local:
		if v := fr.locals[id.Index]; v != nil {
			return v, nil
		}
	case resolve.Free:
		return fr.fn.freevars[id.Index], nil
	case resolve.Global:
		if v := fr.globals[id.Name]; v != nil {
			return v, nil
		}
		if id.Name == "PACKAGE_NAME" {
			// Gross spec, gross hack.
			// Users should just call package_name() function.
			if v, ok := fr.globals["package_name"].(*Builtin); ok {
				return v.fn(fr.thread, v, nil, nil)
			}
		}
	case resolve.Universal:
		return Universe[id.Name], nil
	}
	return nil, fr.errorf(id.NamePos, "%s variable %s referenced before assignment",
		resolve.Scope(id.Scope), id.Name)
}

// An EvalError is a Skylark evaluation error and its associated call stack.
type EvalError struct {
	Msg   string
	Frame *Frame
}

func (e *EvalError) Error() string { return e.Msg }

// Backtrace returns a user-friendly error message describing the stack
// of calls that led to this error.
func (e *EvalError) Backtrace() string {
	var buf bytes.Buffer
	e.Frame.WriteBacktrace(&buf)
	fmt.Fprintf(&buf, "Error: %s", e.Msg)
	return buf.String()
}

// WriteBacktrace writes a user-friendly description of the stack to buf.
func (fr *Frame) WriteBacktrace(out *bytes.Buffer) {
	fmt.Fprintf(out, "Traceback (most recent call last):\n")
	var print func(fr *Frame)
	print = func(fr *Frame) {
		if fr != nil {
			print(fr.parent)

			name := "<toplevel>"
			if fr.fn != nil {
				name = fr.fn.Name()
			}
			fmt.Fprintf(out, "  %s:%d:%d: in %s\n",
				fr.posn.Filename(),
				fr.posn.Line,
				fr.posn.Col,
				name)
		}
	}
	print(fr)
}

// Stack returns the stack of frames, innermost first.
func (e *EvalError) Stack() []*Frame {
	var stack []*Frame
	for fr := e.Frame; fr != nil; fr = fr.parent {
		stack = append(stack, fr)
	}
	return stack
}

// ExecFile parses, resolves, and executes a Skylark file in the
// specified global environment, which may be modified during execution.
//
// The filename and src parameters are as for syntax.Parse.
//
// If ExecFile fails during evaluation, it returns an *EvalError
// containing a backtrace.
func ExecFile(thread *Thread, filename string, src interface{}, globals StringDict) error {
	return Exec(ExecOptions{Thread: thread, Filename: filename, Source: src, Globals: globals})
}

// ExecOptions specifies the arguments to Exec.
type ExecOptions struct {
	// Thread is the state associated with the Skylark thread.
	Thread *Thread

	// Filename is the name of the file to execute,
	// and the name that appears in error messages.
	Filename string

	// Source is an optional source of bytes to use
	// instead of Filename.  See syntax.Parse for details.
	Source interface{}

	// Globals is the environment of the module.
	// It may be modified during execution.
	Globals StringDict

	// BeforeExec is an optional function that is called after the
	// syntax tree has been resolved but before execution.  If it
	// returns an error, execution is not attempted.
	BeforeExec func(*Thread, syntax.Node) error
}

// Exec is a variant of ExecFile that gives the client greater control
// over optional features.
func Exec(opts ExecOptions) error {
	if debug {
		fmt.Printf("ExecFile %s\n", opts.Filename)
		defer fmt.Printf("ExecFile %s done\n", opts.Filename)
	}
	f, err := syntax.Parse(opts.Filename, opts.Source, 0)
	if err != nil {
		return err
	}

	globals := opts.Globals
	if err := resolve.File(f, globals.Has, Universe.Has); err != nil {
		return err
	}

	thread := opts.Thread

	if opts.BeforeExec != nil {
		if err := opts.BeforeExec(thread, f); err != nil {
			return err
		}
	}

	fr := thread.Push(globals, len(f.Locals))
	err = fr.ExecStmts(f.Stmts)
	thread.Pop()

	// Freeze the global environment.
	globals.Freeze()

	return err
}

// Push pushes a new Frame on the specified thread's stack, and returns it.
// It must be followed by a call to Pop when the frame is no longer needed.
func (thread *Thread) Push(globals StringDict, nlocals int) *Frame {
	fr := &Frame{
		thread:  thread,
		parent:  thread.frame,
		globals: globals,
		locals:  make([]Value, nlocals),
	}
	thread.frame = fr
	return fr
}

// Pop removes the topmost frame from the thread's stack.
func (thread *Thread) Pop() {
	thread.frame = thread.frame.parent
}

// Eval parses, resolves, and evaluates an expression within the
// specified global environment.
//
// Evaluation cannot mutate the globals dictionary itself, though it may
// modify variables reachable from the dictionary.
//
// The filename and src parameters are as for syntax.Parse.
//
// If Eval fails during evaluation, it returns an *EvalError
// containing a backtrace.
func Eval(thread *Thread, filename string, src interface{}, globals StringDict) (Value, error) {
	expr, err := syntax.ParseExpr(filename, src, 0)
	if err != nil {
		return nil, err
	}

	locals, err := resolve.Expr(expr, globals.Has, Universe.Has)
	if err != nil {
		return nil, err
	}

	fr := thread.Push(globals, len(locals))
	v, err := eval(fr, expr)
	thread.Pop()
	return v, err
}

// Sentinel values used for control flow.  Internal use only.
var (
	errContinue = fmt.Errorf("continue")
	errBreak    = fmt.Errorf("break")
	errReturn   = fmt.Errorf("return")
)

// ExecStmts executes the statements in the context of the specified
// frame, which must provide sufficient local slots.
//
// Most clients do not need this function; use Exec or Eval instead.
func (fr *Frame) ExecStmts(stmts []syntax.Stmt) error {
	for _, stmt := range stmts {
		if err := exec(fr, stmt); err != nil {
			return err
		}
	}
	return nil
}

func exec(fr *Frame, stmt syntax.Stmt) error {
	switch stmt := stmt.(type) {
	case *syntax.ExprStmt:
		_, err := eval(fr, stmt.X)
		return err

	case *syntax.BranchStmt:
		switch stmt.Token {
		case syntax.PASS:
			return nil // no-op
		case syntax.BREAK:
			return errBreak
		case syntax.CONTINUE:
			return errContinue
		}

	case *syntax.IfStmt:
		cond, err := eval(fr, stmt.Cond)
		if err != nil {
			return err
		}
		if cond.Truth() {
			return fr.ExecStmts(stmt.True)
		} else {
			return fr.ExecStmts(stmt.False)
		}

	case *syntax.AssignStmt:
		switch stmt.Op {
		case syntax.EQ:
			// simple assignment: x = y
			y, err := eval(fr, stmt.RHS)
			if err != nil {
				return err
			}
			return assign(fr, stmt.OpPos, stmt.LHS, y)

		case syntax.PLUS_EQ,
			syntax.MINUS_EQ,
			syntax.STAR_EQ,
			syntax.SLASH_EQ,
			syntax.SLASHSLASH_EQ,
			syntax.PERCENT_EQ:
			// augmented assignment: x += y

			var old Value // old value loaded from "address" x
			var set func(fr *Frame, new Value) error

			// Evaluate "address" of x exactly once to avoid duplicate side-effects.
			switch lhs := stmt.LHS.(type) {
			case *syntax.Ident:
				// x += ...
				x, err := fr.lookup(lhs)
				if err != nil {
					return err
				}
				old = x
				set = func(fr *Frame, new Value) error {
					fr.set(lhs, new)
					return nil
				}

			case *syntax.IndexExpr:
				// x[y] += ...
				x, err := eval(fr, lhs.X)
				if err != nil {
					return err
				}
				y, err := eval(fr, lhs.Y)
				if err != nil {
					return err
				}
				old, err = getIndex(fr, lhs.Lbrack, x, y)
				if err != nil {
					return err
				}
				set = func(fr *Frame, new Value) error {
					return setIndex(fr, lhs.Lbrack, x, y, new)
				}

			case *syntax.DotExpr:
				// x.f += ...
				x, err := eval(fr, lhs.X)
				if err != nil {
					return err
				}
				old, err = getAttr(fr, x, lhs)
				if err != nil {
					return err
				}
				set = func(fr *Frame, new Value) error {
					return setField(fr, x, lhs, new)
				}
			}

			y, err := eval(fr, stmt.RHS)
			if err != nil {
				return err
			}

			// Special case, following Python:
			// If x is a list, x += y is sugar for x.extend(y).
			if xlist, ok := old.(*List); ok && stmt.Op == syntax.PLUS_EQ {
				yiter, ok := y.(Iterable)
				if !ok {
					return fr.errorf(stmt.OpPos, "invalid operation: list += %s", y.Type())
				}
				if err := xlist.checkMutable("apply += to", true); err != nil {
					return fr.errorf(stmt.OpPos, "%v", err)
				}
				listExtend(xlist, yiter)
				return nil
			}

			new, err := Binary(stmt.Op-syntax.PLUS_EQ+syntax.PLUS, old, y)
			if err != nil {
				return fr.errorf(stmt.OpPos, "%v", err)
			}
			return set(fr, new)

		default:
			log.Fatalf("%s: unexpected assignment operator: %s", stmt.OpPos, stmt.Op)
		}

	case *syntax.DefStmt:
		f, err := evalFunction(fr, stmt.Def, stmt.Name.Name, &stmt.Function)
		if err != nil {
			return err
		}
		fr.set(stmt.Name, f)
		return nil

	case *syntax.ForStmt:
		x, err := eval(fr, stmt.X)
		if err != nil {
			return err
		}
		iter := Iterate(x)
		if iter == nil {
			return fr.errorf(stmt.For, "%s value is not iterable", x.Type())
		}
		defer iter.Done()
		var elem Value
		for iter.Next(&elem) {
			if err := assign(fr, stmt.For, stmt.Vars, elem); err != nil {
				return err
			}
			if err := fr.ExecStmts(stmt.Body); err != nil {
				if err == errBreak {
					break
				} else if err == errContinue {
					continue
				} else {
					return err
				}
			}
		}
		return nil

	case *syntax.ReturnStmt:
		if stmt.Result != nil {
			x, err := eval(fr, stmt.Result)
			if err != nil {
				return err
			}
			fr.result = x
		} else {
			fr.result = None
		}
		return errReturn

	case *syntax.LoadStmt:
		module := stmt.Module.Value.(string)
		if fr.thread.Load == nil {
			return fr.errorf(stmt.Load, "load not implemented by this application")
		}
		fr.posn = stmt.Load
		dict, err := fr.thread.Load(fr.thread, module)
		if err != nil {
			return fr.errorf(stmt.Load, "cannot load %s: %v", module, err)
		}
		for i, from := range stmt.From {
			v, ok := dict[from.Name]
			if !ok {
				return fr.errorf(stmt.From[i].NamePos, "load: name %s not found in module %s", from.Name, module)
			}
			fr.set(stmt.To[i], v)
		}
		return nil
	}

	start, _ := stmt.Span()
	log.Fatalf("%s: exec: unexpected statement %T", start, stmt)
	panic("unreachable")
}

// list += iterable
func listExtend(x *List, y Iterable) {
	if ylist, ok := y.(*List); ok {
		// fast path: list += list
		x.elems = append(x.elems, ylist.elems...)
	} else {
		iter := y.Iterate()
		defer iter.Done()
		var z Value
		for iter.Next(&z) {
			x.elems = append(x.elems, z)
		}
	}
}

// getAttr implements x.dot.
func getAttr(fr *Frame, x Value, dot *syntax.DotExpr) (Value, error) {
	name := dot.Name.Name

	// field or method?
	if x, ok := x.(HasAttrs); ok {
		if v, err := x.Attr(name); v != nil || err != nil {
			return v, wrapError(fr, dot.Dot, err)
		}
	}

	return nil, fr.errorf(dot.Dot, "%s has no .%s field or method", x.Type(), name)
}

// setField implements x.name = y.
func setField(fr *Frame, x Value, dot *syntax.DotExpr, y Value) error {
	if x, ok := x.(HasSetField); ok {
		err := x.SetField(dot.Name.Name, y)
		return wrapError(fr, dot.Dot, err)
	}
	return fr.errorf(dot.Dot, "can't assign to .%s field of %s", dot.Name.Name, x.Type())
}

// getIndex implements x[y].
func getIndex(fr *Frame, lbrack syntax.Position, x, y Value) (Value, error) {
	switch x := x.(type) {
	case Mapping: // dict
		z, found, err := x.Get(y)
		if err != nil {
			return nil, fr.errorf(lbrack, "%v", err)
		}
		if !found {
			return nil, fr.errorf(lbrack, "key %v not in %s", y, x.Type())
		}
		return z, nil

	case Indexable: // string, list, tuple
		n := x.Len()
		i, err := AsInt32(y)
		if err != nil {
			return nil, fr.errorf(lbrack, "%s index: %s", x.Type(), err)
		}
		if i < 0 {
			i += n
		}
		if i < 0 || i >= n {
			return nil, fr.errorf(lbrack, "%s index %d out of range [0:%d]",
				x.Type(), i, n)
		}
		return x.Index(i), nil
	}
	return nil, fr.errorf(lbrack, "unhandled index operation %s[%s]", x.Type(), y.Type())
}

// setIndex implements x[y] = z.
func setIndex(fr *Frame, lbrack syntax.Position, x, y, z Value) error {
	switch x := x.(type) {
	case *Dict:
		if err := x.Set(y, z); err != nil {
			return fr.errorf(lbrack, "%v", err)
		}

	case HasSetIndex:
		i, err := AsInt32(y)
		if err != nil {
			return wrapError(fr, lbrack, err)
		}
		if i < 0 {
			i += x.Len()
		}
		if i < 0 || i >= x.Len() {
			return fr.errorf(lbrack, "%s index %d out of range [0:%d]", x.Type(), i, x.Len())
		}
		return wrapError(fr, lbrack, x.SetIndex(i, z))

	default:
		return fr.errorf(lbrack, "%s value does not support item assignment", x.Type())
	}
	return nil
}

// assign implements lhs = rhs for arbitrary expressions lhs.
func assign(fr *Frame, pos syntax.Position, lhs syntax.Expr, rhs Value) error {
	switch lhs := lhs.(type) {
	case *syntax.Ident:
		// x = rhs
		fr.set(lhs, rhs)

	case *syntax.TupleExpr:
		// (x, y) = rhs
		return assignSequence(fr, pos, lhs.List, rhs)

	case *syntax.ListExpr:
		// [x, y] = rhs
		return assignSequence(fr, pos, lhs.List, rhs)

	case *syntax.IndexExpr:
		// x[y] = rhs
		x, err := eval(fr, lhs.X)
		if err != nil {
			return err
		}
		y, err := eval(fr, lhs.Y)
		if err != nil {
			return err
		}
		return setIndex(fr, lhs.Lbrack, x, y, rhs)

	case *syntax.DotExpr:
		// x.f = rhs
		x, err := eval(fr, lhs.X)
		if err != nil {
			return err
		}
		return setField(fr, x, lhs, rhs)

	case *syntax.ParenExpr:
		return assign(fr, pos, lhs.X, rhs)

	default:
		return fr.errorf(pos, "ill-formed assignment: %T", lhs)
	}
	return nil
}

func assignSequence(fr *Frame, pos syntax.Position, lhs []syntax.Expr, rhs Value) error {
	nlhs := len(lhs)
	n := Len(rhs)
	if n < 0 {
		return fr.errorf(pos, "got %s in sequence assignment", rhs.Type())
	} else if n > nlhs {
		return fr.errorf(pos, "too many values to unpack (got %d, want %d)", n, nlhs)
	} else if n < nlhs {
		return fr.errorf(pos, "too few values to unpack (got %d, want %d)", n, nlhs)
	}

	// If the rhs is not indexable, extract its elements into a
	// temporary tuple before doing the assignment.
	ix, ok := rhs.(Indexable)
	if !ok {
		tuple := make(Tuple, n)
		iter := Iterate(rhs)
		if iter == nil {
			return fr.errorf(pos, "non-iterable sequence: %s", rhs.Type())
		}
		for i := 0; i < n; i++ {
			iter.Next(&tuple[i])
		}
		iter.Done()
		ix = tuple
	}

	for i := 0; i < n; i++ {
		if err := assign(fr, pos, lhs[i], ix.Index(i)); err != nil {
			return err
		}
	}
	return nil
}

func eval(fr *Frame, e syntax.Expr) (Value, error) {
	switch e := e.(type) {
	case *syntax.Ident:
		return fr.lookup(e)

	case *syntax.Literal:
		switch e.Token {
		case syntax.INT:
			switch e.Value.(type) {
			case int64:
				return MakeInt64(e.Value.(int64)), nil
			case *big.Int:
				return Int{e.Value.(*big.Int)}, nil
			}
		case syntax.FLOAT:
			return Float(e.Value.(float64)), nil
		case syntax.STRING:
			return String(e.Value.(string)), nil
		}

	case *syntax.ListExpr:
		vals := make([]Value, len(e.List))
		for i, x := range e.List {
			v, err := eval(fr, x)
			if err != nil {
				return nil, err
			}
			vals[i] = v
		}
		return NewList(vals), nil

	case *syntax.CondExpr:
		cond, err := eval(fr, e.Cond)
		if err != nil {
			return nil, err
		}
		if cond.Truth() {
			return eval(fr, e.True)
		} else {
			return eval(fr, e.False)
		}

	case *syntax.IndexExpr:
		x, err := eval(fr, e.X)
		if err != nil {
			return nil, err
		}
		y, err := eval(fr, e.Y)
		if err != nil {
			return nil, err
		}
		return getIndex(fr, e.Lbrack, x, y)

	case *syntax.SliceExpr:
		return evalSliceExpr(fr, e)

	case *syntax.Comprehension:
		var result Value
		if e.Curly {
			result = new(Dict)
		} else {
			result = new(List)
		}
		return result, evalComprehension(fr, e, result, 0)

	case *syntax.TupleExpr:
		n := len(e.List)
		tuple := make(Tuple, n)
		for i, x := range e.List {
			v, err := eval(fr, x)
			if err != nil {
				return nil, err
			}
			tuple[i] = v
		}
		return tuple, nil

	case *syntax.DictExpr:
		dict := new(Dict)
		for i, entry := range e.List {
			entry := entry.(*syntax.DictEntry)
			k, err := eval(fr, entry.Key)
			if err != nil {
				return nil, err
			}
			v, err := eval(fr, entry.Value)
			if err != nil {
				return nil, err
			}
			if err := dict.Set(k, v); err != nil {
				return nil, fr.errorf(e.Lbrace, "%v", err)
			}
			if dict.Len() != i+1 {
				return nil, fr.errorf(e.Lbrace, "duplicate key: %v", k)
			}
		}
		return dict, nil

	case *syntax.UnaryExpr:
		x, err := eval(fr, e.X)
		if err != nil {
			return nil, err
		}
		y, err := Unary(e.Op, x)
		if err != nil {
			return nil, fr.errorf(e.OpPos, "%s", err)
		}
		return y, nil

	case *syntax.BinaryExpr:
		x, err := eval(fr, e.X)
		if err != nil {
			return nil, err
		}

		// short-circuit operators
		switch e.Op {
		case syntax.OR:
			if x.Truth() {
				return x, nil
			}
			return eval(fr, e.Y)
		case syntax.AND:
			if !x.Truth() {
				return x, nil
			}
			return eval(fr, e.Y)
		}

		y, err := eval(fr, e.Y)
		if err != nil {
			return nil, err
		}

		// comparisons
		switch e.Op {
		case syntax.EQL, syntax.NEQ, syntax.GT, syntax.LT, syntax.LE, syntax.GE:
			ok, err := Compare(e.Op, x, y)
			if err != nil {
				return nil, fr.errorf(e.OpPos, "%s", err)
			}
			return Bool(ok), nil
		}

		// binary operators
		z, err := Binary(e.Op, x, y)
		if err != nil {
			return nil, fr.errorf(e.OpPos, "%s", err)
		}
		return z, nil

	case *syntax.DotExpr:
		x, err := eval(fr, e.X)
		if err != nil {
			return nil, err
		}
		return getAttr(fr, x, e)

	case *syntax.CallExpr:
		return evalCall(fr, e)

	case *syntax.LambdaExpr:
		return evalFunction(fr, e.Lambda, "lambda", &e.Function)

	case *syntax.ParenExpr:
		return eval(fr, e.X)
	}

	start, _ := e.Span()
	log.Fatalf("%s: unexpected expr %T", start, e)
	panic("unreachable")
}

// Unary applies a unary operator (+, -, not) to its operand.
func Unary(op syntax.Token, x Value) (Value, error) {
	switch op {
	case syntax.MINUS:
		switch x := x.(type) {
		case Int:
			return zero.Sub(x), nil
		case Float:
			return -x, nil
		}
	case syntax.PLUS:
		switch x.(type) {
		case Int, Float:
			return x, nil
		}
	case syntax.NOT:
		return !x.Truth(), nil
	}
	return nil, fmt.Errorf("unknown unary op: %s %s", op, x.Type())
}

// Binary applies a strict binary operator (not AND or OR) to its operands.
// For equality tests or ordered comparisons, use Compare instead.
func Binary(op syntax.Token, x, y Value) (Value, error) {
	switch op {
	case syntax.PLUS:
		switch x := x.(type) {
		case String:
			if y, ok := y.(String); ok {
				return x + y, nil
			}
		case Int:
			switch y := y.(type) {
			case Int:
				return x.Add(y), nil
			case Float:
				return x.Float() + y, nil
			}
		case Float:
			switch y := y.(type) {
			case Float:
				return x + y, nil
			case Int:
				return x + y.Float(), nil
			}
		case *List:
			if y, ok := y.(*List); ok {
				z := make([]Value, 0, x.Len()+y.Len())
				z = append(z, x.elems...)
				z = append(z, y.elems...)
				return NewList(z), nil
			}
		case Tuple:
			if y, ok := y.(Tuple); ok {
				z := make(Tuple, 0, len(x)+len(y))
				z = append(z, x...)
				z = append(z, y...)
				return z, nil
			}
		}

	case syntax.MINUS:
		switch x := x.(type) {
		case Int:
			switch y := y.(type) {
			case Int:
				return x.Sub(y), nil
			case Float:
				return x.Float() - y, nil
			}
		case Float:
			switch y := y.(type) {
			case Float:
				return x - y, nil
			case Int:
				return x - y.Float(), nil
			}
		}

	case syntax.STAR:
		switch x := x.(type) {
		case Int:
			switch y := y.(type) {
			case Int:
				return x.Mul(y), nil
			case Float:
				return x.Float() * y, nil
			case String:
				if i, err := AsInt32(x); err == nil {
					if i < 1 {
						return String(""), nil
					}
					return String(strings.Repeat(string(y), i)), nil
				}
			case *List:
				if i, err := AsInt32(x); err == nil {
					return NewList(repeat(y.elems, i)), nil
				}
			case Tuple:
				if i, err := AsInt32(x); err == nil {
					return Tuple(repeat([]Value(y), i)), nil
				}
			}
		case Float:
			switch y := y.(type) {
			case Float:
				return x * y, nil
			case Int:
				return x * y.Float(), nil
			}
		case String:
			if y, ok := y.(Int); ok {
				if i, err := AsInt32(y); err == nil {
					if i < 1 {
						return String(""), nil
					}
					return String(strings.Repeat(string(x), i)), nil
				}
			}
		case *List:
			if y, ok := y.(Int); ok {
				if i, err := AsInt32(y); err == nil {
					return NewList(repeat(x.elems, i)), nil
				}
			}
		case Tuple:
			if y, ok := y.(Int); ok {
				if i, err := AsInt32(y); err == nil {
					return Tuple(repeat([]Value(x), i)), nil
				}
			}

		}

	case syntax.SLASH:
		switch x := x.(type) {
		case Int:
			switch y := y.(type) {
			case Int:
				yf := y.Float()
				if yf == 0.0 {
					return nil, fmt.Errorf("real division by zero")
				}
				return x.Float() / yf, nil
			case Float:
				if y == 0.0 {
					return nil, fmt.Errorf("real division by zero")
				}
				return x.Float() / y, nil
			}
		case Float:
			switch y := y.(type) {
			case Float:
				if y == 0.0 {
					return nil, fmt.Errorf("real division by zero")
				}
				return x / y, nil
			case Int:
				yf := y.Float()
				if yf == 0.0 {
					return nil, fmt.Errorf("real division by zero")
				}
				return x / yf, nil
			}
		}

	case syntax.SLASHSLASH:
		switch x := x.(type) {
		case Int:
			switch y := y.(type) {
			case Int:
				if y.Sign() == 0 {
					return nil, fmt.Errorf("floored division by zero")
				}
				return x.Div(y), nil
			case Float:
				if y == 0.0 {
					return nil, fmt.Errorf("floored division by zero")
				}
				return floor((x.Float() / y)), nil
			}
		case Float:
			switch y := y.(type) {
			case Float:
				if y == 0.0 {
					return nil, fmt.Errorf("floored division by zero")
				}
				return floor(x / y), nil
			case Int:
				yf := y.Float()
				if yf == 0.0 {
					return nil, fmt.Errorf("floored division by zero")
				}
				return floor(x / yf), nil
			}
		}

	case syntax.PERCENT:
		switch x := x.(type) {
		case Int:
			switch y := y.(type) {
			case Int:
				if y.Sign() == 0 {
					return nil, fmt.Errorf("integer modulo by zero")
				}
				return x.Mod(y), nil
			case Float:
				if y == 0 {
					return nil, fmt.Errorf("float modulo by zero")
				}
				return x.Float().Mod(y), nil
			}
		case Float:
			switch y := y.(type) {
			case Float:
				if y == 0.0 {
					return nil, fmt.Errorf("float modulo by zero")
				}
				return Float(math.Mod(float64(x), float64(y))), nil
			case Int:
				if y.Sign() == 0 {
					return nil, fmt.Errorf("float modulo by zero")
				}
				return x.Mod(y.Float()), nil
			}
		case String:
			return interpolate(string(x), y)
		}

	case syntax.NOT_IN:
		z, err := Binary(syntax.IN, x, y)
		if err != nil {
			return nil, err
		}
		return !z.Truth(), nil

	case syntax.IN:
		switch y := y.(type) {
		case *List:
			for _, elem := range y.elems {
				if eq, err := Equal(elem, x); err != nil {
					return nil, err
				} else if eq {
					return True, nil
				}
			}
			return False, nil
		case Tuple:
			for _, elem := range y {
				if eq, err := Equal(elem, x); err != nil {
					return nil, err
				} else if eq {
					return True, nil
				}
			}
			return False, nil
		case Mapping: // e.g. dict
			_, found, err := y.Get(x)
			return Bool(found), err
		case *Set:
			ok, err := y.Has(x)
			return Bool(ok), err
		case String:
			needle, ok := x.(String)
			if !ok {
				return nil, fmt.Errorf("'in <string>' requires string as left operand, not %s", x.Type())
			}
			return Bool(strings.Contains(string(y), string(needle))), nil
		case rangeValue:
			i, err := NumberToInt(x)
			if err != nil {
				return nil, fmt.Errorf("'in <range>' requires integer as left operand, not %s", x.Type())
			}
			return Bool(y.contains(i)), nil
		}

	case syntax.PIPE:
		switch x := x.(type) {
		case Int:
			if y, ok := y.(Int); ok {
				return x.Or(y), nil
			}
		case *Set: // union
			if y, ok := y.(*Set); ok {
				iter := Iterate(y)
				defer iter.Done()
				return x.Union(iter)
			}
		}

	case syntax.AMP:
		switch x := x.(type) {
		case Int:
			if y, ok := y.(Int); ok {
				return x.And(y), nil
			}
		case *Set: // intersection
			if y, ok := y.(*Set); ok {
				set := new(Set)
				if x.Len() > y.Len() {
					x, y = y, x // opt: range over smaller set
				}
				for _, xelem := range x.elems() {
					// Has, Insert cannot fail here.
					if found, _ := y.Has(xelem); found {
						set.Insert(xelem)
					}
				}
				return set, nil
			}
		}

	default:
		// unknown operator
		goto unknown
	}

	// user-defined types
	if x, ok := x.(HasBinary); ok {
		z, err := x.Binary(op, y, Left)
		if z != nil || err != nil {
			return z, err
		}
	}
	if y, ok := y.(HasBinary); ok {
		z, err := y.Binary(op, x, Right)
		if z != nil || err != nil {
			return z, err
		}
	}

	// unsupported operand types
unknown:
	return nil, fmt.Errorf("unknown binary op: %s %s %s", x.Type(), op, y.Type())
}

func repeat(elems []Value, n int) (res []Value) {
	if n > 0 {
		res = make([]Value, 0, len(elems)*n)
		for i := 0; i < n; i++ {
			res = append(res, elems...)
		}
	}
	return res
}

func evalCall(fr *Frame, call *syntax.CallExpr) (Value, error) {
	var fn Value

	// Use optimized path for calling methods of built-ins: x.f(...)
	if dot, ok := call.Fn.(*syntax.DotExpr); ok {
		recv, err := eval(fr, dot.X)
		if err != nil {
			return nil, err
		}

		name := dot.Name.Name
		if method := builtinMethodOf(recv, name); method != nil {
			args, kwargs, err := evalArgs(fr, call)
			if err != nil {
				return nil, err
			}

			// Make the call.
			res, err := method(name, recv, args, kwargs)
			return res, wrapError(fr, call.Lparen, err)
		}

		// Fall back to usual path.
		fn, err = getAttr(fr, recv, dot)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		fn, err = eval(fr, call.Fn)
		if err != nil {
			return nil, err
		}
	}

	args, kwargs, err := evalArgs(fr, call)
	if err != nil {
		return nil, err
	}

	// Make the call.
	fr.posn = call.Lparen
	res, err := Call(fr.thread, fn, args, kwargs)
	return res, wrapError(fr, call.Lparen, err)
}

// wrapError wraps the error in a skylark.EvalError only if needed.
func wrapError(fr *Frame, posn syntax.Position, err error) error {
	switch err := err.(type) {
	case nil, *EvalError:
		return err
	}
	return fr.errorf(posn, "%s", err.Error())
}

func evalArgs(fr *Frame, call *syntax.CallExpr) (args Tuple, kwargs []Tuple, err error) {
	// evaluate arguments.
	var kwargsAlloc Tuple // allocate a single backing array
	for i, arg := range call.Args {
		// keyword argument, k=v
		if binop, ok := arg.(*syntax.BinaryExpr); ok && binop.Op == syntax.EQ {
			k := binop.X.(*syntax.Ident).Name
			v, err := eval(fr, binop.Y)
			if err != nil {
				return nil, nil, err
			}
			if kwargs == nil {
				nkwargs := len(call.Args) - i // more than enough
				kwargsAlloc = make(Tuple, 2*nkwargs)
				kwargs = make([]Tuple, 0, nkwargs)
			}
			pair := kwargsAlloc[:2:2]
			kwargsAlloc = kwargsAlloc[2:]
			pair[0], pair[1] = String(k), v
			kwargs = append(kwargs, pair)
			continue
		}

		// *args and **kwargs arguments
		if unop, ok := arg.(*syntax.UnaryExpr); ok {
			if unop.Op == syntax.STAR {
				// *args
				x, err := eval(fr, unop.X)
				if err != nil {
					return nil, nil, err
				}
				iter := Iterate(x)
				if iter == nil {
					return nil, nil, fr.errorf(unop.OpPos, "argument after * must be iterable, not %s", x.Type())
				}
				defer iter.Done()
				var elem Value
				for iter.Next(&elem) {
					args = append(args, elem)
				}
				continue
			}

			if unop.Op == syntax.STARSTAR {
				// **kwargs
				x, err := eval(fr, unop.X)
				if err != nil {
					return nil, nil, err
				}
				xdict, ok := x.(*Dict)
				if !ok {
					return nil, nil, fr.errorf(unop.OpPos, "argument after ** must be a mapping, not %s", x.Type())
				}
				items := xdict.Items()
				for _, item := range items {
					if _, ok := item[0].(String); !ok {
						return nil, nil, fr.errorf(unop.OpPos, "keywords must be strings, not %s", item[0].Type())
					}
				}
				if kwargs == nil {
					kwargs = items
				} else {
					kwargs = append(kwargs, items...)
				}
				continue
			}
		}

		// ordinary argument
		v, err := eval(fr, arg)
		if err != nil {
			return nil, nil, err
		}
		args = append(args, v)
	}
	return args, kwargs, err
}

// Call calls the function fn with the specified positional and keyword arguments.
func Call(thread *Thread, fn Value, args Tuple, kwargs []Tuple) (Value, error) {
	c, ok := fn.(Callable)
	if !ok {
		return nil, fmt.Errorf("invalid call of non-function (%s)", fn.Type())
	}
	res, err := c.Call(thread, args, kwargs)
	// Sanity check: nil is not a valid Skylark value.
	if err == nil && res == nil {
		return nil, fmt.Errorf("internal error: nil (not None) returned from %s", fn)
	}
	return res, err
}

func evalSliceExpr(fr *Frame, e *syntax.SliceExpr) (Value, error) {
	// Unlike Python, Skylark does not allow a slice on the LHS of
	// an assignment statement.

	x, err := eval(fr, e.X)
	if err != nil {
		return nil, err
	}

	var lo, hi, step Value = None, None, None
	if e.Lo != nil {
		lo, err = eval(fr, e.Lo)
		if err != nil {
			return nil, err
		}
	}
	if e.Hi != nil {
		hi, err = eval(fr, e.Hi)
		if err != nil {
			return nil, err
		}
	}
	if e.Step != nil {
		step, err = eval(fr, e.Step)
		if err != nil {
			return nil, err
		}
	}
	res, err := slice(x, lo, hi, step)
	if err != nil {
		return nil, fr.errorf(e.Lbrack, "%s", err)
	}
	return res, nil
}

func slice(x, lo, hi, step_ Value) (Value, error) {
	n := Len(x)
	if n < 0 {
		n = 0 // n < 0 => invalid operand; will be rejected by type switch
	}

	step := 1
	if step_ != None {
		var err error
		step, err = AsInt32(step_)
		if err != nil {
			return nil, fmt.Errorf("got %s for slice step, want int", step_.Type())
		}
		if step == 0 {
			return nil, fmt.Errorf("zero is not a valid slice step")
		}
	}

	// TODO(adonovan): opt: preallocate result array.

	var start, end int
	if step > 0 {
		// positive stride
		// default indices are [0:n].
		var err error
		start, end, err = indices(lo, hi, n)
		if err != nil {
			return nil, err
		}

		if end < start {
			end = start // => empty result
		}

		if step == 1 {
			// common case: simple subsequence
			switch x := x.(type) {
			case String:
				return String(x[start:end]), nil
			case *List:
				elems := append([]Value{}, x.elems[start:end]...)
				return NewList(elems), nil
			case Tuple:
				return x[start:end], nil
			}
		}
	} else {
		// negative stride
		// default indices are effectively [n-1:-1], though to
		// get this effect using explicit indices requires
		// [n-1:-1-n:-1] because of the treatment of -ve values.
		start = n - 1
		if err := asIndex(lo, n, &start); err != nil {
			return nil, fmt.Errorf("invalid start index: %s", err)
		}
		if start >= n {
			start = n - 1
		}

		end = -1
		if err := asIndex(hi, n, &end); err != nil {
			return nil, fmt.Errorf("invalid end index: %s", err)
		}
		if end < -1 {
			end = -1
		}

		if start < end {
			start = end // => empty result
		}
	}

	// For positive strides, the loop condition is i < end.
	// For negative strides, the loop condition is i > end.
	sign := signum(step)
	switch x := x.(type) {
	case String:
		var str []byte
		for i := start; signum(end-i) == sign; i += step {
			str = append(str, x[i])
		}
		return String(str), nil
	case *List:
		var list []Value
		for i := start; signum(end-i) == sign; i += step {
			list = append(list, x.elems[i])
		}
		return NewList(list), nil
	case Tuple:
		var tuple Tuple
		for i := start; signum(end-i) == sign; i += step {
			tuple = append(tuple, x[i])
		}
		return tuple, nil
	}

	return nil, fmt.Errorf("invalid slice operand %s", x.Type())
}

// From Hacker's Delight, section 2.8.
func signum(x int) int { return int(uint64(int64(x)>>63) | (uint64(-x) >> 63)) }

// indices converts start_ and end_ to indices in the range [0:len].
// The start index defaults to 0 and the end index defaults to len.
// An index -len < i < 0 is treated like i+len.
// All other indices outside the range are clamped to the nearest value in the range.
// Beware: start may be greater than end.
// This function is suitable only for slices with positive strides.
func indices(start_, end_ Value, len int) (start, end int, err error) {
	start = 0
	if err := asIndex(start_, len, &start); err != nil {
		return 0, 0, fmt.Errorf("invalid start index: %s", err)
	}
	// Clamp to [0:len].
	if start < 0 {
		start = 0
	} else if start > len {
		start = len
	}

	end = len
	if err := asIndex(end_, len, &end); err != nil {
		return 0, 0, fmt.Errorf("invalid end index: %s", err)
	}
	// Clamp to [0:len].
	if end < 0 {
		end = 0
	} else if end > len {
		end = len
	}

	return start, end, nil
}

// asIndex sets *result to the integer value of v, adding len to it
// if it is negative.  If v is nil or None, *result is unchanged.
func asIndex(v Value, len int, result *int) error {
	if v != nil && v != None {
		var err error
		*result, err = AsInt32(v)
		if err != nil {
			return fmt.Errorf("got %s, want int", v.Type())
		}
		if *result < 0 {
			*result += len
		}
	}
	return nil
}

func evalComprehension(fr *Frame, comp *syntax.Comprehension, result Value, clauseIndex int) error {
	if clauseIndex == len(comp.Clauses) {
		if comp.Curly {
			// dict: {k:v for ...}
			// Parser ensures that body is of form k:v.
			// Python-style set comprehensions {body for vars in x}
			// are not supported.
			entry := comp.Body.(*syntax.DictEntry)
			k, err := eval(fr, entry.Key)
			if err != nil {
				return err
			}
			v, err := eval(fr, entry.Value)
			if err != nil {
				return err
			}
			if err := result.(*Dict).Set(k, v); err != nil {
				return fr.errorf(entry.Colon, "%v", err)
			}
		} else {
			// list: [body for vars in x]
			x, err := eval(fr, comp.Body)
			if err != nil {
				return err
			}
			list := result.(*List)
			list.elems = append(list.elems, x)
		}
		return nil
	}

	clause := comp.Clauses[clauseIndex]
	switch clause := clause.(type) {
	case *syntax.IfClause:
		cond, err := eval(fr, clause.Cond)
		if err != nil {
			return err
		}
		if cond.Truth() {
			return evalComprehension(fr, comp, result, clauseIndex+1)
		}
		return nil

	case *syntax.ForClause:
		x, err := eval(fr, clause.X)
		if err != nil {
			return err
		}
		iter := Iterate(x)
		if iter == nil {
			return fr.errorf(clause.For, "%s value is not iterable", x.Type())
		}
		defer iter.Done()
		var elem Value
		for iter.Next(&elem) {
			if err := assign(fr, clause.For, clause.Vars, elem); err != nil {
				return err
			}

			if err := evalComprehension(fr, comp, result, clauseIndex+1); err != nil {
				return err
			}
		}
		return nil
	}

	start, _ := clause.Span()
	log.Fatalf("%s: unexpected comprehension clause %T", start, clause)
	panic("unreachable")
}

func evalFunction(fr *Frame, pos syntax.Position, name string, function *syntax.Function) (Value, error) {
	// Example: f(x, y=dflt, *args, **kwargs)

	// Evaluate parameter defaults.
	var defaults Tuple // parameter default values
	for _, param := range function.Params {
		if binary, ok := param.(*syntax.BinaryExpr); ok {
			// e.g. y=dflt
			dflt, err := eval(fr, binary.Y)
			if err != nil {
				return nil, err
			}
			defaults = append(defaults, dflt)
		}
	}

	// Capture the values of the function's
	// free variables from the lexical environment.
	freevars := make([]Value, len(function.FreeVars))
	for i, freevar := range function.FreeVars {
		v, err := fr.lookup(freevar)
		if err != nil {
			return nil, fr.errorf(pos, "%s", err)
		}
		freevars[i] = v
	}

	return &Function{
		name:     name,
		position: pos,
		syntax:   function,
		globals:  fr.globals,
		defaults: defaults,
		freevars: freevars,
	}, nil
}

func (fn *Function) Call(thread *Thread, args Tuple, kwargs []Tuple) (Value, error) {
	if debug {
		fmt.Printf("call of %s %v %v\n", fn.Name(), args, kwargs)
	}

	// detect recursion
	for fr := thread.frame; fr != nil; fr = fr.parent {
		// We look for the same syntactic function,
		// not function value, otherwise the user could
		// defeat the check by writing the Y combinator.
		if fr.fn != nil && fr.fn.syntax == fn.syntax {
			return nil, fmt.Errorf("function %s called recursively", fn.Name())
		}
	}

	fr := thread.Push(fn.globals, len(fn.syntax.Locals))
	fr.fn = fn
	err := fn.setArgs(fr, args, kwargs)
	if err == nil {
		err = fr.ExecStmts(fn.syntax.Body)
	}
	thread.Pop()

	if err != nil {
		if err == errReturn {
			return fr.result, nil
		}
		return nil, err
	}
	return None, nil
}

// setArgs sets the values of the formal parameters of function fn in
// frame fr based on the actual parameter values in args and kwargs.
func (fn *Function) setArgs(fr *Frame, args Tuple, kwargs []Tuple) error {
	cond := func(x bool, y, z interface{}) interface{} {
		if x {
			return y
		}
		return z
	}

	// nparams is the number of ordinary parameters (sans * or **).
	nparams := len(fn.syntax.Params)
	if fn.syntax.HasVarargs {
		nparams--
	}
	if fn.syntax.HasKwargs {
		nparams--
	}

	// This is the algorithm from PyEval_EvalCodeEx.
	var kwdict *Dict
	n := len(args)
	if nparams > 0 || fn.syntax.HasVarargs || fn.syntax.HasKwargs {
		if fn.syntax.HasKwargs {
			kwdict = new(Dict)
			fr.locals[len(fn.syntax.Params)-1] = kwdict
		}

		// too many args?
		if len(args) > nparams {
			if !fn.syntax.HasVarargs {
				return fr.errorf(fn.position, "function %s takes %s %d argument%s (%d given)",
					fn.Name(),
					cond(len(fn.defaults) > 0, "at most", "exactly"),
					nparams,
					cond(nparams == 1, "", "s"),
					len(args)+len(kwargs))
			}
			n = nparams
		}

		// set of defined (regular) parameters
		var defined intset
		defined.init(nparams)

		// ordinary parameters
		for i := 0; i < n; i++ {
			fr.locals[i] = args[i]
			defined.set(i)
		}

		// variadic arguments
		if fn.syntax.HasVarargs {
			tuple := make(Tuple, len(args)-n)
			for i := n; i < len(args); i++ {
				tuple[i-n] = args[i]
			}
			fr.locals[nparams] = tuple
		}

		// keyword arguments
		paramIdents := fn.syntax.Locals[:nparams]
		for _, pair := range kwargs {
			k, v := pair[0].(String), pair[1]
			if i := findParam(paramIdents, string(k)); i >= 0 {
				if defined.set(i) {
					return fr.errorf(fn.position, "function %s got multiple values for keyword argument %s", fn.Name(), k)
				}
				fr.locals[i] = v
				continue
			}
			if kwdict == nil {
				return fr.errorf(fn.position, "function %s got an unexpected keyword argument %s", fn.Name(), k)
			}
			kwdict.Set(k, v)
		}

		// default values
		if len(args) < nparams {
			m := nparams - len(fn.defaults) // first default

			// report errors for missing non-optional arguments
			i := len(args)
			for ; i < m; i++ {
				if !defined.get(i) {
					return fr.errorf(fn.position, "function %s takes %s %d argument%s (%d given)",
						fn.Name(),
						cond(fn.syntax.HasVarargs || len(fn.defaults) > 0, "at least", "exactly"),
						m,
						cond(m == 1, "", "s"),
						defined.len())
				}
			}

			// set default values
			for ; i < nparams; i++ {
				if !defined.get(i) {
					fr.locals[i] = fn.defaults[i-m]
				}
			}
		}
	} else if nactual := len(args) + len(kwargs); nactual > 0 {
		return fr.errorf(fn.position, "function %s takes no arguments (%d given)", fn.Name(), nactual)
	}
	return nil
}

func findParam(params []*syntax.Ident, name string) int {
	for i, param := range params {
		if param.Name == name {
			return i
		}
	}
	return -1
}

type intset struct {
	small uint64       // bitset, used if n < 64
	large map[int]bool //    set, used if n >= 64
}

func (is *intset) init(n int) {
	if n >= 64 {
		is.large = make(map[int]bool)
	}
}

func (is *intset) set(i int) (prev bool) {
	if is.large == nil {
		prev = is.small&(1<<uint(i)) != 0
		is.small |= 1 << uint(i)
	} else {
		prev = is.large[i]
		is.large[i] = true
	}
	return
}

func (is *intset) get(i int) bool {
	if is.large == nil {
		return is.small&(1<<uint(i)) != 0
	}
	return is.large[i]
}

func (is *intset) len() int {
	if is.large == nil {
		// Suboptimal, but used only for error reporting.
		len := 0
		for i := 0; i < 64; i++ {
			if is.small&(1<<uint(i)) != 0 {
				len++
			}
		}
		return len
	}
	return len(is.large)
}

// https://github.com/google/skylark/blob/master/doc/spec.md#string-interpolation
func interpolate(format string, x Value) (Value, error) {
	var buf bytes.Buffer
	path := make([]Value, 0, 4)
	index := 0
	for {
		i := strings.IndexByte(format, '%')
		if i < 0 {
			buf.WriteString(format)
			break
		}
		buf.WriteString(format[:i])
		format = format[i+1:]

		if format != "" && format[0] == '%' {
			buf.WriteByte('%')
			format = format[1:]
			continue
		}

		var arg Value
		if format != "" && format[0] == '(' {
			// keyword argument: %(name)s.
			format = format[1:]
			j := strings.IndexByte(format, ')')
			if j < 0 {
				return nil, fmt.Errorf("incomplete format key")
			}
			key := format[:j]
			if dict, ok := x.(Mapping); !ok {
				return nil, fmt.Errorf("format requires a mapping")
			} else if v, found, _ := dict.Get(String(key)); found {
				arg = v
			} else {
				return nil, fmt.Errorf("key not found: %s", key)
			}
			format = format[j+1:]
		} else {
			// positional argument: %s.
			if tuple, ok := x.(Tuple); ok {
				if index >= len(tuple) {
					return nil, fmt.Errorf("not enough arguments for format string")
				}
				arg = tuple[index]
			} else if index > 0 {
				return nil, fmt.Errorf("not enough arguments for format string")
			} else {
				arg = x
			}
		}

		// NOTE: Skylark does not support any of these optional Python features:
		// - optional conversion flags: [#0- +], etc.
		// - optional minimum field width (number or *).
		// - optional precision (.123 or *)
		// - optional length modifier

		// conversion type
		if format == "" {
			return nil, fmt.Errorf("incomplete format")
		}
		switch c := format[0]; c {
		case 's', 'r':
			if str, ok := AsString(arg); ok && c == 's' {
				buf.WriteString(str)
			} else {
				writeValue(&buf, arg, path)
			}
		case 'd', 'i', 'o', 'x', 'X':
			i, err := NumberToInt(arg)
			if err != nil {
				return nil, fmt.Errorf("%%%c format requires integer: %v", c, err)
			}
			switch c {
			case 'd', 'i':
				buf.WriteString(i.bigint.Text(10))
			case 'o':
				buf.WriteString(i.bigint.Text(8))
			case 'x':
				buf.WriteString(i.bigint.Text(16))
			case 'X':
				buf.WriteString(strings.ToUpper(i.bigint.Text(16)))
			}
		case 'e', 'f', 'g', 'E', 'F', 'G':
			f, ok := AsFloat(arg)
			if !ok {
				return nil, fmt.Errorf("%%%c format requires float, not %s", c, arg.Type())
			}
			switch c {
			case 'e':
				fmt.Fprintf(&buf, "%e", f)
			case 'f':
				fmt.Fprintf(&buf, "%f", f)
			case 'g':
				fmt.Fprintf(&buf, "%g", f)
			case 'E':
				fmt.Fprintf(&buf, "%E", f)
			case 'F':
				fmt.Fprintf(&buf, "%F", f)
			case 'G':
				fmt.Fprintf(&buf, "%G", f)
			}
		case 'c':
			switch arg := arg.(type) {
			case Int:
				// chr(int)
				r, err := AsInt32(arg)
				if err != nil || r < 0 || r > unicode.MaxRune {
					return nil, fmt.Errorf("%%c format requires a valid Unicode code point, got %s", arg)
				}
				buf.WriteRune(rune(r))
			case String:
				r, size := utf8.DecodeRuneInString(string(arg))
				if size != len(arg) {
					return nil, fmt.Errorf("%%c format requires a single-character string")
				}
				buf.WriteRune(r)
			default:
				return nil, fmt.Errorf("%%c format requires int or single-character string, not %s", arg.Type())
			}
		case '%':
			buf.WriteByte('%')
		default:
			return nil, fmt.Errorf("unknown conversion %%%c", c)
		}
		format = format[1:]
		index++
	}

	if tuple, ok := x.(Tuple); ok && index < len(tuple) {
		return nil, fmt.Errorf("too many arguments for format string")
	}

	return String(buf.String()), nil
}
