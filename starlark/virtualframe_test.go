// Copyright 2026 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package starlark_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime/pprof"
	"strings"
	"testing"

	"go.starlark.net/starlark"
)

func TestVirtualFrame(t *testing.T) {
	prof, err := os.CreateTemp(t.TempDir(), "virtualframe_test")
	if err != nil {
		t.Fatal(err)
	}
	defer prof.Close()

	if err := pprof.StartCPUProfile(prof); err != nil {
		t.Fatal(err)
	}

	const src = `
def busy_inner(n):
	acc = 0
	for i in range(n):
		acc = acc + i
	return acc

def caller_middle(n):
	return busy_inner(n)

def top_entry():
	for _ in range(20):
		caller_middle(200000)

top_entry()
`

	thread := new(starlark.Thread)
	if _, err := starlark.ExecFile(thread, "virtualframe_test.star", src, nil); err != nil {
		pprof.StopCPUProfile()
		t.Fatal(err)
	}
	pprof.StopCPUProfile()

	if err := prof.Sync(); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "tool", "pprof", "-traces", prof.Name())
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("go tool pprof failed: %v\nstderr:\n%s", err, &stderr)
	}

	got := stdout.String()
	for _, want := range []string{
		"busy_inner",
		"caller_middle",
		"top_entry",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("profile traces did not contain Starlark function %q\nOutput:\n%s", want, got)
		}
	}

	// Also verify file:line attribution using -top or -list
	cmdTop := exec.Command("go", "tool", "pprof", "-top", "-lines", prof.Name())
	var topOut bytes.Buffer
	cmdTop.Stdout = &topOut
	cmdTop.Stderr = &stderr
	if err := cmdTop.Run(); err != nil {
		t.Fatalf("go tool pprof -top -lines failed: %v\nstderr:\n%s", err, &stderr)
	}
	gotTop := fmt.Sprint(&topOut)
	for _, want := range []string{
		"busy_inner virtualframe_test.star:",
		"caller_middle virtualframe_test.star:9",
		"top_entry virtualframe_test.star:13",
	} {
		if !strings.Contains(gotTop, want) {
			t.Errorf("profile -top -lines did not contain %q\nOutput:\n%s", want, gotTop)
		}
	}
}
