package stargo

import (
	"fmt"
	"reflect"
	"runtime"
	"runtime/debug"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// A goFunc represents a Go value of kind func.
type goFunc struct {
	v reflect.Value // kind=Func; !CanAddr
}

var (
	_ Value             = goFunc{}
	_ starlark.Callable = goFunc{}
	_ starlark.HasAttrs = goFunc{}
)

func (f goFunc) Attr(name string) (starlark.Value, error) { return method(f.v, name) }
func (f goFunc) AttrNames() []string                      { return methodNames(f.v) }
func (f goFunc) Freeze()                                  {} // unimplementable
func (f goFunc) Hash() (uint32, error)                    { return ptrHash(f.v), nil }
func (f goFunc) Reflect() reflect.Value                   { return f.v }
func (f goFunc) String() string                           { return f.Name() }
func (f goFunc) Truth() starlark.Bool                     { return f.v.IsNil() == false }
func (f goFunc) Type() string                             { return fmt.Sprintf("go.func<%s>", f.v.Type()) }

func (f goFunc) CallInternal(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (_ starlark.Value, err error) {
	if len(kwargs) > 0 {
		return nil, fmt.Errorf("Go function %s does not accept named arguments", f)
	}
	if f.v.IsNil() {
		return nil, fmt.Errorf("call of nil function")
	}

	ft := f.v.Type()
	arity := ft.NumIn()
	variadic := ft.IsVariadic()
	if variadic {
		if len(args) < arity-1 {
			return nil, fmt.Errorf("in call to %s, got %d arguments, want at least %d", f, len(args), arity-1)
		}
	} else if len(args) != arity {
		return nil, fmt.Errorf("in call to %s, got %d arguments, want %d", f, len(args), arity)
	}

	var in, out []reflect.Value
	for i, arg := range args {
		var t reflect.Type
		if variadic && i >= arity-1 {
			t = ft.In(arity - 1).Elem()
		} else {
			t = ft.In(i)
		}
		x, err := toGo(arg, t)
		if err != nil {
			return nil, fmt.Errorf("in argument %d of call to %s, %s", i+1, f, err)
		}
		in = append(in, x)
	}

	if err := protect(thread, f, func() { out = f.v.Call(in) }); err != nil {
		return nil, err
	}
	switch len(out) {
	case 0:
		return starlark.None, nil
	case 1:
		return toStarlark(out[0]), nil
	default:
		return toStarlarkTuple(out), nil
	}
}

func toStarlarkTuple(values []reflect.Value) starlark.Tuple {
	tuple := make(starlark.Tuple, len(values))
	for i, v := range values {
		tuple[i] = toStarlark(v)
	}
	return tuple
}

func (f goFunc) Name() string {
	name := runtime.FuncForPC(f.v.Pointer()).Name()
	if name == "reflect.methodValueCall" || name == "reflect.makeFuncStub" {
		name = fmt.Sprintf("%#v", f.v.Interface())
	}
	if name == "" {
		name = str(f.v)
	}
	return name
}

// A goFrame is a fake Callable used to report Go frames in a Starlark backtrace.
type goFrame struct{ fr *runtime.Frame }

func (f goFrame) Position() syntax.Position {
	return syntax.MakePosition(&f.fr.File, int32(f.fr.Line), 0)
}
func (f goFrame) Name() string          { return f.fr.Func.Name() }
func (f goFrame) Freeze()               {} // immutable
func (f goFrame) String() string        { return f.Name() }
func (f goFrame) Truth() starlark.Bool  { return starlark.True }
func (f goFrame) Type() string          { return "goFrame" }
func (f goFrame) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: %s", f.Type()) }
func (f goFrame) CallInternal(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return nil, fmt.Errorf("internal error: call of goFrame")
}

// protect invokes function f, converting a panic into a
// starlark.EvalErr with a stack of Go frames.
// The name appears (formatted using %v) in the error message.
func protect(thread *starlark.Thread, name interface{}, f func()) (err error) {
	// Handle panics in Go code.
	// Display only the stack below this point.
	thispc, _, _, _ := runtime.Caller(0)
	ok := false
	defer func() {
		if ok {
			return // success
		}

		// for debugging unexpected panics
		if false {
			debug.PrintStack()
		}

		// panic in Go call
		thisFunc := runtime.FuncForPC(thispc) // protect

		// Build list of Go frames (innermost first).
		pcs := make([]uintptr, 32)
		pcs = pcs[:runtime.Callers(2, pcs)]
		var stack []runtime.Frame
		frames := runtime.CallersFrames(pcs)
		for {
			frame, more := frames.Next()
			if !more {
				break
			}
			stack = append(stack, frame)
			if frame.Func == thisFunc {
				break
			}
			if frame.Function == "runtime.gopanic" {
				stack = stack[:0] // don't show gopanic or its children
			}
		}

		// Convert to starlark.Frame list (outermost first).
		fr := thread.Caller()
		for i := range stack {
			callable := goFrame{&stack[len(stack)-1-i]}
			fr = starlark.NewFrame(fr, callable)
		}
		err = &starlark.EvalError{
			Msg:   fmt.Sprintf("panic in %v: %v", name, recover()),
			Frame: fr,
		}
	}()

	f()

	ok = true
	return nil
}

func funcConvert(callable starlark.Callable, funcType reflect.Type) (reflect.Value, error) {
	// The conversion fails if the callable is a
	// starlark.Function with the wrong number of parameters.
	// There's nothing we can do about result types though.
	if fn, ok := callable.(*starlark.Function); ok {
		// This logic handles only non-variadic cases.
		// TODO: handle all four {variadic,non} x {variadic,non} cases.
		if !fn.HasVarargs() && !funcType.IsVariadic() {
			n := fn.NumParams()
			if fn.HasKwargs() {
				n--
			}
			if funcType.NumIn() != n {
				return reflect.Value{}, fmt.Errorf("cannot convert %d-ary Starlark function %s to %s", n, callable.Name(), funcType)
			}
		}
	}

	return reflect.MakeFunc(funcType, func(args []reflect.Value) []reflect.Value {
		// During a reflective call of a Go function,
		// there is no notion of the current starlark.Thread,
		// so each from Go call must create a new one.
		// TODO: Can we do better?
		// Perhaps by implementing goroutine-local storage?
		thread := new(starlark.Thread)

		res, err := starlark.Call(thread, callable, toStarlarkTuple(args), nil)
		outs, err := resultsToGo(funcType, res, err)
		if err != nil {
			// We have no choice but to panic.
			// TODO: should we wrap the error type and handle it in protect?
			// Typically it will be an *EvalError; perhaps wrapping is unnecessary.
			// TODO: add "in result of call to callable"?
			panic(err)
		}
		return outs
	}), nil
}

// resultsToGo converts the result of a Starlark call to Go.
//
// How should we map Starlark results to Go?
// Ignoring errors, a starlark result is a single value
// whereas a Go result may be a tuple. Presumably we should
// unpack the Starlark value if the Go result has >1 component,
// and discard the Starlark result if the Go result is void.
//
// What about Go functions that return an error?  Should we
// map starlark eval errors to this component, or unpack the
// starlark result in the usual way? Or both?  What if the Go
// function does not have an error and the Starlark function
// fails? We have no choice but to panic the *EvalError.  This
// will replaces helpful Starlark frames with unhelpful Go
// frames for the interpreter internals.  If the Go function
// has a place for errors, we can return an EvalErr (perhaps
// augmenting the stack).  Otherwise panic is our only choice.
//
func resultsToGo(funcType reflect.Type, res starlark.Value, err error) ([]reflect.Value, error) {
	n := funcType.NumOut()
	outs := make([]reflect.Value, n)
	if err != nil {
		// If the Go function has an error result, return the Starlark error.
		if n > 0 && funcType.Out(n-1) == errorType {
			for i := 0; i < n-1; i++ {
				outs[i] = reflect.Zero(funcType.Out(i))
			}
			outs[n-1] = reflect.ValueOf(err).Convert(funcType.Out(n - 1))
			return outs, nil
		}
		return nil, err
	}

	// TODO: should we handle errors below the way we do err above, if funcType permits??

	switch n {
	case 0:
		// Go function has void result: discard Starlark value.

	case 1:
		// Go and Starlark functions both have a single result.
		y, err := toGo(res, funcType.Out(0))
		if err != nil {
			return nil, err
		}
		outs[0] = y

	default:
		// Go function returns a tuple: unpack Starlark result.
		iter := starlark.Iterate(res)
		if iter == nil {
			return nil, fmt.Errorf("cannot unpack %s into result of %s", res.Type(), funcType)
		}
		defer iter.Done()
		var x starlark.Value
		for i := range outs {
			if !iter.Next(&x) {
				return nil, fmt.Errorf("too few results to unpack (got %d, want %d)", i, n)
			}
			y, err := toGo(x, funcType.Out(i))
			if err != nil {
				return nil, fmt.Errorf("in result %d, %v", i+1, err)
			}
			outs[i] = y
		}
		if iter.Next(&x) {
			return nil, fmt.Errorf("too many results to unpack (want %d)", n)
		}
	}
	return outs, nil
}
