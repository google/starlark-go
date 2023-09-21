// Copyright 2019 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package starlark_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"go.starlark.net/starlark"
)

// TestProfile is a simple integration test that the profiler
// emits minimally plausible pprof-compatible output.
func TestProfile(t *testing.T) {
	prof, err := os.CreateTemp(t.TempDir(), "profile_test")
	if err != nil {
		t.Fatal(err)
	}
	defer prof.Close()
	if err := starlark.StartProfile(prof); err != nil {
		t.Fatal(err)
	}

	const src = `
def fibonacci(n):
	x, y = 1, 1
	for i in range(n):
		x, y = y, x+y
	return y

fibonacci(100000)
`

	thread := new(starlark.Thread)
	if _, err := starlark.ExecFile(thread, "foo.star", src, nil); err != nil {
		_ = starlark.StopProfile()
		t.Fatal(err)
	}
	if err := starlark.StopProfile(); err != nil {
		t.Fatal(err)
	}
	prof.Sync()
	cmd := exec.Command("go", "tool", "pprof", "-top", prof.Name())
	cmd.Stderr = new(bytes.Buffer)
	cmd.Stdout = new(bytes.Buffer)
	if err := cmd.Run(); err != nil {
		t.Fatalf("pprof failed: %v; output=<<%s>>", err, cmd.Stderr)
	}

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
	got := fmt.Sprint(cmd.Stdout)
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
		t.Logf("stderr=%v", cmd.Stderr)
		t.Logf("stdout=%v", cmd.Stdout)
	}
}
