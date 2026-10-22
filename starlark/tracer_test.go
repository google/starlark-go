package starlark_test

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"

	"go.starlark.net/starlark"
)

type callTracerFuncs struct {
	onCall   func(thread *starlark.Thread, fn starlark.Callable, args starlark.Tuple, kwargs []starlark.Tuple)
	onReturn func(thread *starlark.Thread, fn starlark.Callable, result starlark.Value, err error)
}

func (t callTracerFuncs) TraceCall(thread *starlark.Thread, fn starlark.Callable, args starlark.Tuple, kwargs []starlark.Tuple) {
	if t.onCall != nil {
		t.onCall(thread, fn, args, kwargs)
	}
}

func (t callTracerFuncs) TraceReturn(thread *starlark.Thread, fn starlark.Callable, result starlark.Value, err error) {
	if t.onReturn != nil {
		t.onReturn(thread, fn, result, err)
	}
}

func loggingTracer(events *[]string, name string) callTracerFuncs {
	return callTracerFuncs{
		onCall: func(thread *starlark.Thread, fn starlark.Callable, args starlark.Tuple, kwargs []starlark.Tuple) {
			*events = append(*events, name+" call "+fn.Name())
		},
		onReturn: func(thread *starlark.Thread, fn starlark.Callable, result starlark.Value, err error) {
			*events = append(*events, name+" return "+fn.Name())
		},
	}
}

func nopBuiltin(name string) *starlark.Builtin {
	return starlark.NewBuiltin(name, func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		return starlark.None, nil
	})
}

func TestCallTracer(t *testing.T) {
	const src = `
def add(x, y = 0): return x + y
def run():
    add(1, y = 2)
    return add(3)
value = run()
length = len("abc")
`
	var events []string
	var addFunction starlark.Callable
	var firstCallArgs, copiedFirstCallArgs starlark.Tuple
	var firstCallArgsSeenLater string
	var errorSeenByTracer error
	thread := new(starlark.Thread)
	thread.AddCallTracer(callTracerFuncs{
		onCall: func(thread *starlark.Thread, fn starlark.Callable, args starlark.Tuple, kwargs []starlark.Tuple) {
			if got := thread.CallFrame(0).Name; got != fn.Name() {
				t.Errorf("TraceCall frame = %q, want %q", got, fn.Name())
			}
			events = append(events, fmt.Sprintf("call %s args=%v kwargs=%v", fn.Name(), args, kwargs))
			if fn.Name() == "add" {
				if addFunction == nil {
					addFunction = fn
					firstCallArgs = args
					copiedFirstCallArgs = slices.Clone(args)
				} else {
					firstCallArgsSeenLater = fmt.Sprint(firstCallArgs)
				}
			}
		},
		onReturn: func(thread *starlark.Thread, fn starlark.Callable, result starlark.Value, err error) {
			if got := thread.CallFrame(0).Name; got != fn.Name() {
				t.Errorf("TraceReturn frame = %q, want %q", got, fn.Name())
			}
			if err != nil {
				events = append(events, "return "+fn.Name()+" error")
				errorSeenByTracer = err
			} else {
				events = append(events, fmt.Sprintf("return %s = %v", fn.Name(), result))
			}
		},
	})
	globals, err := starlark.ExecFile(thread, "trace.star", src, nil)
	if err != nil {
		t.Fatal(err)
	}
	if addFunction != globals["add"] {
		t.Errorf("TraceCall function = %v, want %v", addFunction, globals["add"])
	}
	if got := fmt.Sprint(globals["value"]); got != "3" {
		t.Errorf("value = %s, want 3", got)
	}
	if got := fmt.Sprint(globals["length"]); got != "3" {
		t.Errorf("length = %s, want 3", got)
	}
	if firstCallArgsSeenLater != "(3,)" {
		t.Errorf("first call args during second call = %s, want (3,)", firstCallArgsSeenLater)
	}
	if got := fmt.Sprint(copiedFirstCallArgs); got != "(1,)" {
		t.Errorf("copied first call args = %s, want (1,)", got)
	}

	fail := starlark.NewBuiltin("fail", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		return nil, errors.New("failed")
	})
	_, callError := starlark.Call(thread, fail, nil, nil)
	if callError == nil {
		t.Fatal("Call returned no error")
	}
	if errorSeenByTracer != callError {
		t.Errorf("TraceReturn error = %v, want caller error %v", errorSeenByTracer, callError)
	}
	if _, ok := callError.(*starlark.EvalError); !ok {
		t.Errorf("Call error has type %T, want *starlark.EvalError", callError)
	}

	want := []string{
		"call <toplevel> args=() kwargs=[]",
		"call run args=() kwargs=[]",
		`call add args=(1,) kwargs=[("y", 2)]`,
		"return add = 3",
		"call add args=(3,) kwargs=[]",
		"return add = 3",
		"return run = 3",
		`call len args=("abc",) kwargs=[]`,
		"return len = 3",
		"return <toplevel> = None",
		"call fail args=() kwargs=[]",
		"return fail error",
	}
	if !reflect.DeepEqual(events, want) {
		t.Errorf("events = %q, want %q", events, want)
	}
}

func TestCallTracerAddAndRemove(t *testing.T) {
	var events []string
	nop := starlark.NewBuiltin("nop", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		events = append(events, "nop")
		return starlark.None, nil
	})

	thread := new(starlark.Thread)
	checkCallEvents := func(want []string) {
		t.Helper()
		events = nil
		if _, err := starlark.Call(thread, nop, nil, nil); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(events, want) {
			t.Errorf("events = %q, want %q", events, want)
		}
	}

	a := loggingTracer(&events, "A")
	b := loggingTracer(&events, "B")
	removeA := thread.AddCallTracer(a)
	removeB1 := thread.AddCallTracer(b)
	removeB2 := thread.AddCallTracer(b)
	checkCallEvents([]string{
		"A call nop", "B call nop", "B call nop", "nop",
		"B return nop", "B return nop", "A return nop",
	})

	removeB1()
	removeB1()
	checkCallEvents([]string{"A call nop", "B call nop", "nop", "B return nop", "A return nop"})

	removeA()
	removeB2()
	checkCallEvents([]string{"nop"})
}

func TestCallTracerChangesDuringCall(t *testing.T) {
	var events []string
	thread := new(starlark.Thread)
	added := loggingTracer(&events, "added")
	var removeInitial func()
	initial := loggingTracer(&events, "initial")
	recordCall := initial.onCall
	initial.onCall = func(thread *starlark.Thread, fn starlark.Callable, args starlark.Tuple, kwargs []starlark.Tuple) {
		recordCall(thread, fn, args, kwargs)
		if fn.Name() == "f" {
			thread.AddCallTracer(added)
			removeInitial()
		}
	}
	removeInitial = thread.AddCallTracer(initial)

	if _, err := starlark.ExecFile(thread, "tracer_changes.star", `
def g(): return 1
def f(): return g()
f()
`, nil); err != nil {
		t.Fatal(err)
	}

	want := []string{
		"initial call <toplevel>",
		"initial call f",
		"added call g",
		"added return g",
		"initial return f",
		"initial return <toplevel>",
	}
	if !reflect.DeepEqual(events, want) {
		t.Errorf("events = %q, want %q", events, want)
	}
}

func TestCallTracerPanic(t *testing.T) {
	mustPanic := func(want string, f func()) {
		defer func() {
			if got := fmt.Sprint(recover()); got != want {
				t.Errorf("recover() = %v, want %v", got, want)
			}
		}()
		f()
		t.Error("function returned without a panic")
	}

	var traceReturnRan bool
	thread := new(starlark.Thread)
	thread.AddCallTracer(callTracerFuncs{onReturn: func(thread *starlark.Thread, fn starlark.Callable, result starlark.Value, err error) {
		traceReturnRan = true
		if result != nil || err != nil {
			t.Errorf("TraceReturn(%s) = (%v, %v), want (<nil>, <nil>)", fn.Name(), result, err)
		}
	}})
	panicBuiltin := starlark.NewBuiltin("panic", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		panic(args[0])
	})
	mustPanic("1", func() {
		starlark.Call(thread, panicBuiltin, starlark.Tuple{starlark.MakeInt(1)}, nil)
	})
	if !traceReturnRan {
		t.Error("TraceReturn did not run")
	}
	if got := thread.CallStackDepth(); got != 0 {
		t.Errorf("CallStackDepth = %d after panic, want 0", got)
	}
}
