package starlark_test

// This file checks what happens when calls end because of errors or panics. It
// also checks Load, profile changes while code is running, and panics from
// custom Callables.

import (
	"testing"
	"time"

	"go.starlark.net/starlark"
)

func TestProfileUnwinding(t *testing.T) {
	boom := starlark.NewBuiltin("boom", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		panic("boom")
	})
	spin := starlark.NewBuiltin("spin", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		thread.Cancel("test")
		return starlark.None, nil
	})

	for _, test := range []struct {
		name              string
		src               string
		unwindingFunction string
	}{
		{"error", "def f():\n\tfail('nope')\n\nf()\n", "fail"},
		{"cancel", "def f():\n\tspin()\n\tspin()\n\nf()\n", "spin"},
		{"panic", "def f():\n\tboom()\n\nf()\n", "boom"},
	} {
		t.Run(test.name, func(t *testing.T) {
			predeclared := starlark.StringDict{"boom": boom, "spin": spin}
			var p starlark.ExecProfile
			thread := &starlark.Thread{Profile: &p}

			start := time.Now()
			func() {
				defer func() { recover() }()
				if _, err := starlark.ExecFile(thread, "unwind.star", test.src, predeclared); err == nil {
					t.Error("execution succeeded, want failure")
				}
			}()
			wall := time.Since(start)

			recs := uniqueRecordsByName(t, &p)
			for _, name := range []string{"<toplevel>", "f", test.unwindingFunction} {
				if _, ok := recs[name]; !ok {
					t.Errorf("no record for %s; got %v", name, p.Records())
				}
			}
			if got := recs["<toplevel>"].Calls; got != 1 {
				t.Errorf("<toplevel>.Calls = %d, want 1 (all frames must be removed after failure)", got)
			}
			if got := recs["f"].Calls; got != 1 {
				t.Errorf("f.Calls = %d, want 1", got)
			}
			checkFullyRecordedProfileInvariants(t, &p, wall)

			thread.Uncancel()
			if _, err := starlark.ExecFile(thread, "after.star", "def f():\n\tpass\n\nf()\n", nil); err != nil {
				t.Fatalf("thread could not run code after failure: %v", err)
			}
			if got := totalCallsByName(&p, "<toplevel>"); got != 2 {
				t.Errorf("<toplevel>.Calls = %d after second execution, want 2", got)
			}
		})
	}
}

func TestProfileLoad(t *testing.T) {
	const nap = 20 * time.Millisecond
	const module = `
def helper():
	nap()

helper()
x = 1
`
	predeclared := starlark.StringDict{"nap": sleepingBuiltin("nap", nap)}

	var p starlark.ExecProfile
	parent := &starlark.Thread{Profile: &p}
	parent.Load = func(thread *starlark.Thread, name string) (starlark.StringDict, error) {
		child := &starlark.Thread{Load: thread.Load, Profile: thread.Profile}
		return starlark.ExecFile(child, name, module, predeclared)
	}

	start := time.Now()
	if _, err := starlark.ExecFile(parent, "main.star", `load("mod.star", "x")`, predeclared); err != nil {
		t.Fatal(err)
	}
	wall := time.Since(start)

	if _, ok := findRecordByNameAndFile(&p, "helper", "mod.star"); !ok {
		t.Errorf("no record for the loaded module's helper:\n%v", p.Records())
	}
	if _, ok := findRecordByNameAndFile(&p, "<toplevel>", "mod.star"); !ok {
		t.Errorf("no record for the loaded module's toplevel:\n%v", p.Records())
	}
	main, ok := findRecordByNameAndFile(&p, "<toplevel>", "main.star")
	if !ok {
		t.Fatalf("no record for the main toplevel:\n%v", p.Records())
	}
	if main.Cum < nap {
		t.Errorf("main toplevel Cum = %v, want >= %v (Cum includes the load)", main.Cum, nap)
	}
	if main.Self >= nap {
		t.Errorf("main toplevel Self = %v, want < %v (Self excludes the load)", main.Self, nap)
	}
	checkFullyRecordedProfileInvariants(t, &p, wall)
}

func TestProfileAttachMidRun(t *testing.T) {
	var p starlark.ExecProfile
	attach := starlark.NewBuiltin("attach", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		thread.Profile = &p
		return starlark.None, nil
	})

	const src = `
def before():
	pass

def after():
	pass

before()
attach()
after()
after()
`
	thread := new(starlark.Thread)
	if _, err := starlark.ExecFile(thread, "attach.star", src, starlark.StringDict{"attach": attach}); err != nil {
		t.Fatal(err)
	}

	if got := totalCallsByName(&p, "after"); got != 2 {
		t.Errorf("after.Calls = %d, want 2 (calls that start after the profile is attached are recorded)", got)
	}
	for _, name := range []string{"before", "attach", "<toplevel>"} {
		if got := totalCallsByName(&p, name); got != 0 {
			t.Errorf("%s.Calls = %d, want 0 (the call started before the profile was attached)", name, got)
		}
	}
}

func TestProfileDetachMidRun(t *testing.T) {
	detach := starlark.NewBuiltin("detach", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		thread.Profile = nil
		return starlark.None, nil
	})

	const src = `
def before():
	pass

def after():
	pass

before()
detach()
after()
`
	p := executeProfiledFile(t, "detach.star", src, starlark.StringDict{"detach": detach})

	if got := totalCallsByName(p, "before"); got != 1 {
		t.Errorf("before.Calls = %d, want 1", got)
	}
	if got := totalCallsByName(p, "after"); got != 0 {
		t.Errorf("after.Calls = %d, want 0 (calls after the profile is removed are not recorded)", got)
	}
	// The top-level call started before the profile was removed and ended after
	// it. The call is not counted. Self still includes span parts that ended
	// while the profile was attached.
	top, ok := findRecordByNameAndFile(p, "<toplevel>", "detach.star")
	if !ok {
		t.Fatalf("no record for the toplevel:\n%v", p.Records())
	}
	if top.Calls != 0 {
		t.Errorf("<toplevel>.Calls = %d, want 0 (the call ended after the profile was removed)", top.Calls)
	}
	if top.Self <= 0 {
		t.Errorf("<toplevel>.Self = %v, want > 0 (spans closed while attached are kept)", top.Self)
	}
}

func TestProfileLoadOnSameThread(t *testing.T) {
	const embedderWorkDuration = 40 * time.Millisecond
	const module = `
def helper():
	pass

helper()
x = 1
`
	var p starlark.ExecProfile
	thread := &starlark.Thread{Profile: &p}
	thread.Load = func(th *starlark.Thread, name string) (starlark.StringDict, error) {
		globals, err := starlark.ExecFile(th, name, module, nil)
		time.Sleep(embedderWorkDuration)
		return globals, err
	}

	start := time.Now()
	if _, err := starlark.ExecFile(thread, "main.star", `load("mod.star", "x")`, nil); err != nil {
		t.Fatal(err)
	}
	wall := time.Since(start)

	if _, ok := findRecordByNameAndFile(&p, "helper", "mod.star"); !ok {
		t.Errorf("the module run by Load was not recorded:\n%v", p.Records())
	}
	main, ok := findRecordByNameAndFile(&p, "<toplevel>", "main.star")
	if !ok {
		t.Fatalf("no record for the main toplevel:\n%v", p.Records())
	}
	if main.Self >= embedderWorkDuration {
		t.Errorf("main toplevel Self = %v, want < %v: Self excludes time blocked in Load", main.Self, embedderWorkDuration)
	}
	if main.Cum < embedderWorkDuration {
		t.Errorf("main toplevel Cum = %v, want >= %v", main.Cum, embedderWorkDuration)
	}
	checkFullyRecordedProfileInvariants(t, &p, wall)
}

func TestProfileDetachStopsCharging(t *testing.T) {
	var start time.Time
	var detachedAt time.Duration
	detach := starlark.NewBuiltin("detach", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		thread.Profile = nil
		detachedAt = time.Since(start)
		return starlark.None, nil
	})

	// The code after detach() must not make calls. It runs in f's frame, so only
	// f's span could include its time. A call would end the span and hide the
	// bug that this test checks.
	const src = `
def f():
	detach()
	x = "y" * 10000000
	x = "y" * 10000000
	x = "y" * 10000000
	x = "y" * 10000000
	x = "y" * 10000000
	return x

f()
`
	var p starlark.ExecProfile
	thread := &starlark.Thread{Profile: &p}

	start = time.Now()
	if _, err := starlark.ExecFile(thread, "detach.star", src, starlark.StringDict{"detach": detach}); err != nil {
		t.Fatal(err)
	}
	wall := time.Since(start)

	postDetachWork := wall - detachedAt
	if postDetachWork < time.Millisecond {
		t.Fatalf("only %v of work ran after detach; the test needs more work to check this case", postDetachWork)
	}
	if got, limit := totalProfileSelf(&p), detachedAt+postDetachWork/2; got > limit {
		t.Errorf("recorded Self totals %v, want <= %v: Self must not include work after detach at %v",
			got, limit, detachedAt)
	}
}

func TestProfileSwapMidRun(t *testing.T) {
	const nap = 30 * time.Millisecond
	var q starlark.ExecProfile
	swap := starlark.NewBuiltin("swap", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		thread.Profile = &q
		return starlark.None, nil
	})

	const src = `
def after():
	nap()

swap()
after()
a = "y" * 10000000
b = "y" * 10000000
c = "y" * 10000000
d = "y" * 10000000
e = "y" * 10000000
`
	predeclared := starlark.StringDict{"swap": swap, "nap": sleepingBuiltin("nap", nap)}
	p := executeProfiledFile(t, "swap.star", src, predeclared)

	if got := totalCallsByName(&q, "after"); got != 1 {
		t.Errorf("q recorded after.Calls = %d, want 1", got)
	}
	if got := totalCallsByName(p, "after"); got != 0 {
		t.Errorf("p recorded after.Calls = %d, want 0 (p had been removed)", got)
	}
	if got, limit := totalProfileSelf(p), 5*time.Millisecond; got > limit {
		t.Errorf("p recorded %v of Self, want <= %v: p was removed at swap()", got, limit)
	}
	if got := totalProfileSelf(&q); got < nap {
		t.Errorf("q recorded %v of Self, want >= %v", got, nap)
	}
}

type panickingNameCallable struct{}

func (panickingNameCallable) Name() string          { panic("panickingNameCallable.Name") }
func (panickingNameCallable) String() string        { return "panickingNameCallable" }
func (panickingNameCallable) Type() string          { return "panickingNameCallable" }
func (panickingNameCallable) Freeze()               {}
func (panickingNameCallable) Truth() starlark.Bool  { return true }
func (panickingNameCallable) Hash() (uint32, error) { return 0, nil }
func (panickingNameCallable) CallInternal(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return starlark.None, nil
}

func TestProfilePanicInCallableMethod(t *testing.T) {
	var p starlark.ExecProfile
	thread := &starlark.Thread{Profile: &p}

	func() {
		defer func() {
			if recover() == nil {
				t.Error("no panic from callable with a panicking Name method")
			}
		}()
		starlark.Call(thread, panickingNameCallable{}, nil, nil)
	}()

	if got := thread.CallStackDepth(); got != 0 {
		t.Errorf("CallStackDepth = %d after the panic, want 0: the frame must be removed while handling the panic", got)
	}
	if _, err := starlark.ExecFile(thread, "after.star", "def f():\n\tpass\n\nf()\n", nil); err != nil {
		t.Fatalf("thread unusable after the panic: %v", err)
	}
	if got := totalCallsByName(&p, "f"); got != 1 {
		t.Errorf("f.Calls = %d, want 1", got)
	}
}
