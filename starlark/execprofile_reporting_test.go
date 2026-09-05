package starlark_test

// This file checks text reports, merged profiles, parallel runs, and use with
// the pprof profiler.

import (
	"errors"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"go.starlark.net/starlark"
)

func TestProfileWriteText(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		var p starlark.ExecProfile
		report := renderProfile(t, &p)
		lines := strings.Split(strings.TrimSuffix(report, "\n"), "\n")
		if len(lines) != 1 {
			t.Fatalf("empty profile wrote %d lines, want 1 (header only):\n%s", len(lines), report)
		}
		for _, col := range []string{"calls", "self", "cum", "max", "function"} {
			if !strings.Contains(lines[0], col) {
				t.Errorf("header %q does not include column %q", lines[0], col)
			}
		}
	})

	const src = `def outer():
	return inner()

def inner():
	return 1

outer()
`
	p := executeProfiledFile(t, "report.star", src, nil)
	got := renderProfile(t, p)

	t.Run("rows", func(t *testing.T) {
		lines := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
		if want := len(p.Records()) + 1; len(lines) != want {
			t.Fatalf("wrote %d lines, want %d (one header + one per record):\n%s", len(lines), want, got)
		}
		for _, want := range []string{"outer (report.star:1:1)", "inner (report.star:4:1)", "<toplevel> (report.star:1:1)"} {
			if !strings.Contains(got, want) {
				t.Errorf("report does not include %q:\n%s", want, got)
			}
		}
	})

	t.Run("order", func(t *testing.T) {
		recs := p.Records()
		for i := 1; i < len(recs); i++ {
			x, y := recs[i-1], recs[i]
			if x.Self < y.Self {
				t.Errorf("record %d (%v) sorts before %d (%v); want highest Self first", i-1, x.Self, i, y.Self)
			}
			if x.Self == y.Self && x.Name > y.Name {
				t.Errorf("equal Self: %q sorts before %q; want Name in increasing order", x.Name, y.Name)
			}
		}
		var offsets []int
		for _, rec := range recs {
			i := strings.Index(got, rec.Name+" (")
			if i < 0 {
				t.Fatalf("report does not include a row for %s:\n%s", rec.Name, got)
			}
			offsets = append(offsets, i)
		}
		for i := 1; i < len(offsets); i++ {
			if offsets[i-1] > offsets[i] {
				t.Errorf("report row %d appears after row %d; want Records order:\n%s", i-1, i, got)
			}
		}
	})

	t.Run("stable", func(t *testing.T) {
		for i := range 20 {
			if report := renderProfile(t, p); report != got {
				t.Fatalf("render %d differs from the first:\n%s\nwant:\n%s", i, report, got)
			}
		}
	})

	t.Run("writeError", func(t *testing.T) {
		want := errors.New("disk full")
		if err := p.WriteText(failingWriter{want}); !errors.Is(err, want) {
			t.Errorf("WriteText to a failing writer = %v, want %v", err, want)
		}
		if renderProfile(t, p) != got {
			t.Error("profile changed after a failed write")
		}
	})
}

type failingWriter struct{ err error }

func (w failingWriter) Write(p []byte) (int, error) { return 0, w.err }

func renderProfile(t *testing.T, p *starlark.ExecProfile) string {
	t.Helper()
	var buf strings.Builder
	if err := p.WriteText(&buf); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestProfileMerge(t *testing.T) {
	_, shared, err := starlark.SourceProgram("shared.star", "def f():\n\tpass\n\nf()\nf()\n", func(string) bool { return false })
	if err != nil {
		t.Fatal(err)
	}
	_, only, err := starlark.SourceProgram("only.star", "def g():\n\tpass\n\ng()\n", func(string) bool { return false })
	if err != nil {
		t.Fatal(err)
	}

	run := func(p *starlark.ExecProfile, prog *starlark.Program) {
		t.Helper()
		thread := &starlark.Thread{Profile: p}
		if _, err := prog.Init(thread, nil); err != nil {
			t.Fatal(err)
		}
	}

	var a, b starlark.ExecProfile
	run(&a, shared)
	run(&b, shared)
	run(&b, only)

	before, ok := findRecordByNameAndFile(&b, "f", "shared.star")
	if !ok {
		t.Fatal("b has no record for f")
	}
	aBefore, ok := findRecordByNameAndFile(&a, "f", "shared.star")
	if !ok {
		t.Fatal("a has no record for f")
	}

	a.Merge(&b)

	merged, ok := findRecordByNameAndFile(&a, "f", "shared.star")
	if !ok {
		t.Fatal("merged profile has no record for f")
	}
	if want := aBefore.Calls + before.Calls; merged.Calls != want {
		t.Errorf("merged f.Calls = %d, want %d", merged.Calls, want)
	}
	if want := aBefore.Self + before.Self; merged.Self != want {
		t.Errorf("merged f.Self = %v, want %v", merged.Self, want)
	}
	if want := aBefore.Cum + before.Cum; merged.Cum != want {
		t.Errorf("merged f.Cum = %v, want %v", merged.Cum, want)
	}
	if want := max(aBefore.Max, before.Max); merged.Max != want {
		t.Errorf("merged f.Max = %v, want %v (the larger, not the sum)", merged.Max, want)
	}
	if _, ok := findRecordByNameAndFile(&a, "g", "only.star"); !ok {
		t.Errorf("merge did not keep the record for g:\n%v", a.Records())
	}

	after, _ := findRecordByNameAndFile(&b, "f", "shared.star")
	if after != before {
		t.Errorf("Merge modified its argument: %+v, was %+v", after, before)
	}

	beforeDegenerate := a.Records()
	a.Merge(&a)
	if got := a.Records(); !slices.Equal(got, beforeDegenerate) {
		t.Errorf("Merge(self) changed the profile:\n got %v\nwant %v", got, beforeDegenerate)
	}
	a.Merge(nil)
	if got := a.Records(); !slices.Equal(got, beforeDegenerate) {
		t.Errorf("Merge(nil) changed the profile:\n got %v\nwant %v", got, beforeDegenerate)
	}
}

func TestProfileParallel(t *testing.T) {
	_, prog, err := starlark.SourceProgram("par.star", "def f():\n\tpass\n\n[f() for _ in range(20)]\n", func(string) bool { return false })
	if err != nil {
		t.Fatal(err)
	}

	const threads = 8
	profiles := make([]starlark.ExecProfile, threads)
	var wg sync.WaitGroup
	for i := range profiles {
		wg.Add(1)
		go func() {
			defer wg.Done()
			thread := &starlark.Thread{Profile: &profiles[i]}
			if _, err := prog.Init(thread, nil); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()

	var total starlark.ExecProfile
	for i := range profiles {
		total.Merge(&profiles[i])
	}
	if got, want := totalCallsByName(&total, "f"), int64(threads*20); got != want {
		t.Errorf("merged f.Calls = %d, want %d", got, want)
	}
}

func TestProfileWithPprof(t *testing.T) {
	pprof, err := os.CreateTemp(t.TempDir(), "profile")
	if err != nil {
		t.Fatal(err)
	}
	defer pprof.Close()
	if err := starlark.StartProfile(pprof); err != nil {
		t.Fatal(err)
	}

	const src = `
def f(n):
	x, y = 1, 1
	for i in range(n):
		x, y = y, x + y
	return y

f(200000)
`
	var p starlark.ExecProfile
	thread := &starlark.Thread{Profile: &p}

	start := time.Now()
	_, execErr := starlark.ExecFile(thread, "both.star", src, nil)
	wall := time.Since(start)

	if err := starlark.StopProfile(); err != nil {
		t.Fatal(err)
	}
	if execErr != nil {
		t.Fatal(execErr)
	}

	if got := totalCallsByName(&p, "f"); got != 1 {
		t.Errorf("f.Calls = %d, want 1", got)
	}
	checkFullyRecordedProfileInvariants(t, &p, wall)

	if fi, err := pprof.Stat(); err != nil {
		t.Fatal(err)
	} else if fi.Size() == 0 {
		t.Error("the pprof profiler wrote nothing")
	}
}
