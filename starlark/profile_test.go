// Copyright 2019 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package starlark_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	"go.starlark.net/starlark"
)

// TestProfile is a simple integration test that the profiler
// emits minimally plausible pprof-compatible output.
func TestProfile(t *testing.T) {
	const src = `
def fibonacci(n):
	x, y = 1, 1
	for i in range(n):
		x, y = y, x+y
	return y

fibonacci(100000)
`

	got := profile(t, func() error {
		thread := new(starlark.Thread)
		_, err := starlark.ExecFile(thread, "foo.star", src, nil)
		return err
	})

	// Typical output (may vary by go release):
	//
	// Type: wall
	// Time: Apr 4, 2019 at 11:10am (EDT)
	// Duration: 251.62ms, Total samples = 250ms (99.36%)
	// Showing nodes accounting for 250ms, 100% of 250ms total
	//  flat  flat%   sum%        cum   cum%
	// 320ms   100%   100%      320ms   100%  fibonacci
	//     0     0%   100%      320ms   100%  foo.star
	//
	// We'll assert a few key substrings are present.
	for _, want := range []string{
		"flat%",
		"fibonacci",
		"foo.star",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output did not contain %q", want)
		}
	}
	if t.Failed() {
		t.Logf("stdout=%v", got)
	}
}

func TestProfileThreadLabel(t *testing.T) {
	predeclared := starlark.StringDict{"sleep": sleep}
	got := profile(t, func() error {
		// Reuse one thread under different names,
		// like a worker running several tasks.
		thread := new(starlark.Thread)
		for _, name := range []string{"a", "b", ""} {
			thread.Name = name
			if _, err := starlark.ExecFile(thread, "foo.star", "sleep()", predeclared); err != nil {
				return err
			}
		}
		return nil
	}, "-tags", "-unit=ms")

	// Typical output:
	//
	//  thread: Total 200ms of 300ms (66.67%)
	//          100ms (33.33%): a
	//          100ms (33.33%): b
	//
	// A percentage may be padded, as in "(  100%)".
	// Each run sleeps 100ms, so each name gets at least 100ms.
	values := make(map[string]time.Duration)
	for _, m := range regexp.MustCompile(`(?m)^\s+(\S+) \(\s*\S+\): (.*)$`).FindAllStringSubmatch(got, -1) {
		values[m[2]], _ = time.ParseDuration(m[1])
	}
	if len(values) != 2 || values["a"] < 100*time.Millisecond || values["b"] < 100*time.Millisecond {
		t.Errorf("thread label values = %v, want a and b with at least 100ms each; output=<<%s>>", values, got)
	}

	// The header shows labeled time out of the total;
	// the rest comes from the unnamed run.
	labeled := pprofDuration(t, got, `thread: Total (\S+)`)
	total := pprofDuration(t, got, `thread: Total \S+ of (\S+)`)
	if unlabeled := total - labeled; unlabeled < 100*time.Millisecond {
		t.Errorf("unlabeled time = %v, want at least 100ms; output=<<%s>>", unlabeled, got)
	}
}

// sleep is a built-in that takes 100ms.
var sleep = starlark.NewBuiltin("sleep", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	time.Sleep(100 * time.Millisecond)
	return starlark.None, nil
})

// TestProfileBuiltinCallback checks that time spent in a Starlark
// function called by a built-in is not also attributed to the built-in,
// and that the built-in's own time before and after the call is.
func TestProfileBuiltinCallback(t *testing.T) {
	call := starlark.NewBuiltin("call", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		time.Sleep(50 * time.Millisecond)
		v, err := starlark.Call(thread, args[0], nil, nil)
		time.Sleep(50 * time.Millisecond)
		return v, err
	})
	predeclared := starlark.StringDict{"sleep": sleep, "call": call}

	const src = `
def f():
	sleep()

call(f)
`

	var elapsed time.Duration
	got := profile(t, func() error {
		thread := new(starlark.Thread)
		start := time.Now()
		_, err := starlark.ExecFile(thread, "foo.star", src, predeclared)
		elapsed = time.Since(start)
		return err
	})

	// A thread's spans do not overlap, so its samples
	// cannot add up to more than its elapsed time.
	checkSamples(t, got, elapsed)

	// The built-in's span resumes after the callback returns.
	if flat := pprofDuration(t, got, `(?m)^\s*(\S+)\s.*\scall$`); flat < 100*time.Millisecond {
		t.Errorf("call flat time = %v, want at least 100ms; output=<<%s>>", flat, got)
	}
}

// TestProfileLoadCallback checks that a Load function that calls back
// into Starlark on the importing thread does not resume the importer's
// span, which LOAD has ended. Otherwise the importer would be charged
// for the Load function's work, even work done by other threads.
func TestProfileLoadCallback(t *testing.T) {
	noop := starlark.NewBuiltin("noop", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		return starlark.None, nil
	})
	load := func(thread *starlark.Thread, module string) (starlark.StringDict, error) {
		// Call back into Starlark on the importing thread
		// before and after executing a file on another thread.
		if _, err := starlark.Call(thread, noop, nil, nil); err != nil {
			return nil, err
		}
		predeclared := starlark.StringDict{"sleep": sleep}
		if _, err := starlark.ExecFile(new(starlark.Thread), "other.star", "sleep()", predeclared); err != nil {
			return nil, err
		}
		if _, err := starlark.Call(thread, noop, nil, nil); err != nil {
			return nil, err
		}
		return starlark.StringDict{"x": starlark.None}, nil
	}

	var elapsed time.Duration
	got := profile(t, func() error {
		thread := &starlark.Thread{Load: load}
		start := time.Now()
		_, err := starlark.ExecFile(thread, "main.star", `load("m", "x")`, nil)
		elapsed = time.Since(start)
		return err
	})

	// The threads run one at a time, so their samples
	// cannot add up to more than the elapsed time.
	checkSamples(t, got, elapsed)
}

// checkSamples checks that the sleep built-in accounts for at
// least 100ms of the profile output got, and that the total
// samples do not exceed the elapsed time.
func checkSamples(t *testing.T, got string, elapsed time.Duration) {
	t.Helper()
	if flat := pprofDuration(t, got, `(?m)^\s*(\S+)\s.*\ssleep$`); flat < 100*time.Millisecond {
		t.Errorf("sleep flat time = %v, want at least 100ms; output=<<%s>>", flat, got)
	}
	if total := pprofDuration(t, got, `Total samples = (\S+)`); total > elapsed {
		t.Errorf("total samples = %v, exceeds elapsed time %v; output=<<%s>>", total, elapsed, got)
	}
}

// pprofDuration returns the duration matched by the
// sole group of pattern in the profile output got.
func pprofDuration(t *testing.T, got, pattern string) time.Duration {
	t.Helper()
	m := regexp.MustCompile(pattern).FindStringSubmatch(got)
	if m == nil {
		t.Fatalf("output does not match %q: %s", pattern, got)
	}
	d, err := time.ParseDuration(m[1])
	if err != nil {
		t.Fatalf("parsing %q: %v", m[1], err)
	}
	return d
}

// profile calls run with the profiler enabled, and returns the
// output of 'go tool pprof' with the given flags (by default,
// -top -unit=ns) for the resulting profile.
func profile(t *testing.T, run func() error, flags ...string) string {
	prof, err := os.CreateTemp(t.TempDir(), "profile_test")
	if err != nil {
		t.Fatal(err)
	}
	defer prof.Close()
	if err := starlark.StartProfile(prof); err != nil {
		t.Fatal(err)
	}
	stopped := false
	defer func() {
		if !stopped {
			starlark.StopProfile() // run called t.Fatal
		}
	}()
	err = run()
	stopped = true
	if err := starlark.StopProfile(); err != nil {
		t.Fatal(err)
	}
	if err != nil {
		t.Fatal(err)
	}
	if len(flags) == 0 {
		// Use a fixed unit; pprof's automatic units include
		// "hrs", which time.ParseDuration does not accept.
		flags = []string{"-top", "-unit=ns"}
	}
	cmd := exec.Command("go", append(append([]string{"tool", "pprof"}, flags...), prof.Name())...)
	cmd.Stderr = new(bytes.Buffer)
	cmd.Stdout = new(bytes.Buffer)
	if err := cmd.Run(); err != nil {
		t.Fatalf("pprof %v failed: %v; output=<<%s>>", flags, err, cmd.Stderr)
	}
	return fmt.Sprint(cmd.Stdout)
}
