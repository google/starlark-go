package starlark_test

// This file checks the main profiling behavior. It covers callable identity,
// call counts, Self and Cum time, recursive calls, calls back into Starlark,
// and profiles used for more than one execution.

import (
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

func TestProfileCounts(t *testing.T) {
	const src = `
def f(n):
	return n + 1

sq = lambda x: x * x

def g():
	total = 0
	for i in range(3):
		total += f(sq(i))
	return "n".upper() + str(bump(total))

g()
`
	predeclared := starlark.StringDict{"bump": new(incrementingCallable)}
	p := executeProfiledFile(t, "prof.star", src, predeclared)
	want := map[string]int64{
		"<toplevel>": 1,
		"g":          1,
		"f":          3,
		"lambda":     3,
		"range":      1,
		"str":        1,
		"upper":      1,
		"bump":       1,
	}
	got := make(map[string]int64)
	for _, r := range p.Records() {
		got[r.Name] = r.Calls
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("call counts (-want +got):\n%s", diff)
	}
}

type incrementingCallable struct{ calls int }

func (b *incrementingCallable) Name() string          { return "bump" }
func (b *incrementingCallable) String() string        { return "bump" }
func (b *incrementingCallable) Type() string          { return "incrementingCallable" }
func (b *incrementingCallable) Freeze()               {}
func (b *incrementingCallable) Truth() starlark.Bool  { return true }
func (b *incrementingCallable) Hash() (uint32, error) { return 0, nil }
func (b *incrementingCallable) CallInternal(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	b.calls++
	n, err := starlark.AsInt32(args[0])
	if err != nil {
		return nil, err
	}
	return starlark.MakeInt(n + 1), nil
}

func executeProfiledFile(t *testing.T, filename, src string, predeclared starlark.StringDict) *starlark.ExecProfile {
	t.Helper()
	p := new(starlark.ExecProfile)
	thread := &starlark.Thread{Profile: p}
	if _, err := starlark.ExecFile(thread, filename, src, predeclared); err != nil {
		t.Fatal(err)
	}
	return p
}

func uniqueRecordsByName(t *testing.T, p *starlark.ExecProfile) map[string]starlark.ExecProfileRecord {
	t.Helper()
	byName := make(map[string]starlark.ExecProfileRecord)
	for _, rec := range p.Records() {
		if _, dup := byName[rec.Name]; dup {
			t.Fatalf("two records named %s; test needs distinct names", rec.Name)
		}
		byName[rec.Name] = rec
	}
	return byName
}

func totalCallsByName(p *starlark.ExecProfile, name string) int64 {
	var calls int64
	for _, rec := range p.Records() {
		if rec.Name == name {
			calls += rec.Calls
		}
	}
	return calls
}

func checkFullyRecordedProfileInvariants(t *testing.T, p *starlark.ExecProfile, wall time.Duration) {
	t.Helper()
	var totalSelf time.Duration
	for _, rec := range p.Records() {
		if rec.Self < 0 || rec.Cum < 0 || rec.Max < 0 || rec.Calls < 0 {
			t.Errorf("%s: negative field in %+v", rec.Name, rec)
		}
		if rec.Self > rec.Cum {
			t.Errorf("%s: Self (%v) > Cum (%v)", rec.Name, rec.Self, rec.Cum)
		}
		if rec.Max > rec.Cum {
			t.Errorf("%s: Max (%v) > Cum (%v)", rec.Name, rec.Max, rec.Cum)
		}
		if rec.Cum > wall {
			t.Errorf("%s: Cum (%v) > wall (%v)", rec.Name, rec.Cum, wall)
		}
		totalSelf += rec.Self
	}
	if totalSelf > wall {
		t.Errorf("sum of Self (%v) exceeds wall time (%v)", totalSelf, wall)
	}
}

func sleepingBuiltin(name string, d time.Duration) *starlark.Builtin {
	return starlark.NewBuiltin(name, func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		time.Sleep(d)
		return starlark.None, nil
	})
}

func TestProfileReentry(t *testing.T) {
	const nap = 20 * time.Millisecond

	var builtinWork, callbackWork time.Duration
	work := starlark.NewBuiltin("work", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		ownStart := time.Now()
		time.Sleep(nap)
		builtinWork = time.Since(ownStart)

		callbackStart := time.Now()
		for range 2 {
			if _, err := starlark.Call(thread, args[0], nil, nil); err != nil {
				return nil, err
			}
		}
		callbackWork = time.Since(callbackStart)
		return starlark.None, nil
	})

	const src = `
def slow():
	nap()

work(slow)
`
	predeclared := starlark.StringDict{"work": work, "nap": sleepingBuiltin("nap", nap)}

	start := time.Now()
	p := executeProfiledFile(t, "reentry.star", src, predeclared)
	wall := time.Since(start)

	recs := uniqueRecordsByName(t, p)
	if got := recs["slow"].Calls; got != 2 {
		t.Errorf("slow.Calls = %d, want 2", got)
	}
	if got := recs["nap"].Self; got < 2*nap {
		t.Errorf("nap.Self = %v, want >= %v", got, 2*nap)
	}
	if got := recs["work"].Self; got < builtinWork {
		t.Errorf("work.Self = %v, want >= the %v spent in the built-in", got, builtinWork)
	}
	if got, limit := recs["work"].Self, builtinWork+callbackWork/2; got >= limit {
		t.Errorf("work.Self = %v, want < %v: it slept %v and spent %v in callbacks",
			got, limit, builtinWork, callbackWork)
	}
	if got, want := recs["work"].Cum, builtinWork+callbackWork; got < want {
		t.Errorf("work.Cum = %v, want >= %v", got, want)
	}
	checkFullyRecordedProfileInvariants(t, p, wall)
}

func TestProfileRecursion(t *testing.T) {
	const nap = 10 * time.Millisecond
	recursive := &syntax.FileOptions{Recursion: true}

	t.Run("direct", func(t *testing.T) {
		const src = `
def countdown(n):
	nap()
	if n > 0:
		countdown(n - 1)

countdown(3)
`
		predeclared := starlark.StringDict{"nap": sleepingBuiltin("nap", nap)}
		var p starlark.ExecProfile
		thread := &starlark.Thread{Profile: &p}

		start := time.Now()
		if _, err := starlark.ExecFileOptions(recursive, thread, "recursion.star", src, predeclared); err != nil {
			t.Fatal(err)
		}
		wall := time.Since(start)

		recs := uniqueRecordsByName(t, &p)
		if got := recs["countdown"].Calls; got != 4 {
			t.Errorf("countdown.Calls = %d, want 4", got)
		}
		if got := recs["countdown"].Cum; got > wall {
			t.Errorf("countdown.Cum = %v, want <= wall %v", got, wall)
		}
		if got := recs["countdown"].Cum; got < 4*nap {
			t.Errorf("countdown.Cum = %v, want >= %v", got, 4*nap)
		}
		if got, cum := recs["countdown"].Max, recs["countdown"].Cum; got != cum {
			t.Errorf("countdown.Max = %v, want %v (the full time of the outermost call)", got, cum)
		}
		checkFullyRecordedProfileInvariants(t, &p, wall)
	})

	t.Run("mutual", func(t *testing.T) {
		const src = `
def even(n):
	nap()
	return True if n == 0 else odd(n - 1)

def odd(n):
	nap()
	return False if n == 0 else even(n - 1)

even(3)
`
		predeclared := starlark.StringDict{"nap": sleepingBuiltin("nap", nap)}
		var p starlark.ExecProfile
		thread := &starlark.Thread{Profile: &p}

		start := time.Now()
		if _, err := starlark.ExecFileOptions(recursive, thread, "mutual.star", src, predeclared); err != nil {
			t.Fatal(err)
		}
		wall := time.Since(start)

		recs := uniqueRecordsByName(t, &p)
		if got := recs["even"].Calls; got != 2 {
			t.Errorf("even.Calls = %d, want 2", got)
		}
		if got := recs["odd"].Calls; got != 2 {
			t.Errorf("odd.Calls = %d, want 2", got)
		}
		checkFullyRecordedProfileInvariants(t, &p, wall)
	})
}

func findRecordByNameAndFile(p *starlark.ExecProfile, name, filename string) (starlark.ExecProfileRecord, bool) {
	for _, rec := range p.Records() {
		if rec.Name == name && rec.Position.Filename() == filename {
			return rec, true
		}
	}
	return starlark.ExecProfileRecord{}, false
}

func TestProfileAccumulates(t *testing.T) {
	const src = `
def f():
	pass

f()
f()
`
	var p starlark.ExecProfile
	thread := &starlark.Thread{Profile: &p}
	for range 3 {
		if _, err := starlark.ExecFile(thread, "accum.star", src, nil); err != nil {
			t.Fatal(err)
		}
	}
	if got := totalCallsByName(&p, "f"); got != 6 {
		t.Errorf("f.Calls = %d after 3 executions, want 6", got)
	}
	if got := totalCallsByName(&p, "<toplevel>"); got != 3 {
		t.Errorf("<toplevel>.Calls = %d after 3 executions, want 3", got)
	}
}

type addresslessCallable string

func (c addresslessCallable) Name() string          { return string(c) }
func (c addresslessCallable) String() string        { return string(c) }
func (c addresslessCallable) Type() string          { return "addresslessCallable" }
func (c addresslessCallable) Freeze()               {}
func (c addresslessCallable) Truth() starlark.Bool  { return true }
func (c addresslessCallable) Hash() (uint32, error) { return 0, nil }
func (c addresslessCallable) CallInternal(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return starlark.None, nil
}

func TestProfileAddresslessCallable(t *testing.T) {
	predeclared := starlark.StringDict{"one": addresslessCallable("one"), "two": addresslessCallable("two")}
	p := executeProfiledFile(t, "tags.star", "one()\none()\ntwo()\n", predeclared)

	if got := totalCallsByName(p, "one"); got != 2 {
		t.Errorf("one.Calls = %d, want 2", got)
	}
	if got := totalCallsByName(p, "two"); got != 1 {
		t.Errorf("two.Calls = %d, want 1 (the same address must still use separate rows for each name)", got)
	}
	rec, ok := findRecordByNameAndFile(p, "one", "<builtin>")
	if !ok {
		t.Fatalf("one has no record positioned in <builtin>:\n%v", p.Records())
	}
	if rec.Position.Line != 0 {
		t.Errorf("one.Position = %v, want line 0 in <builtin>", rec.Position)
	}
}

func totalProfileSelf(p *starlark.ExecProfile) time.Duration {
	var total time.Duration
	for _, rec := range p.Records() {
		total += rec.Self
	}
	return total
}

func TestProfileIdentityAcrossCompilations(t *testing.T) {
	const files = 200
	var p starlark.ExecProfile
	for i := range files {
		src := strings.Repeat("\n", i) + "def f():\n\tpass\n\nf()\n"
		thread := &starlark.Thread{Profile: &p}
		if _, err := starlark.ExecFile(thread, "gen.star", src, nil); err != nil {
			t.Fatal(err)
		}
		// Run garbage collection after each execution. This frees its funcode
		// addresses so the next compilation can reuse them.
		runtime.GC()
	}

	var rows int
	for _, rec := range p.Records() {
		if rec.Name == "f" {
			rows++
		}
	}
	if rows != files {
		t.Errorf("%d rows for f after %d separately compiled files, want %d: "+
			"distinct functions must not share a row", rows, files, files)
	}
	if got := totalCallsByName(&p, "f"); got != files {
		t.Errorf("total f.Calls = %d, want %d", got, files)
	}
}
