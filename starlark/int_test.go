// Copyright 2017 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package starlark

import (
	"flag"
	"fmt"
	"log"
	"math"
	"math/big"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

// TestIntOpts exercises integer arithmetic, especially at the boundaries.
func TestIntOpts(t *testing.T) {
	f := MakeInt64
	left, right := big.NewInt(math.MinInt32), big.NewInt(math.MaxInt32)

	for i, test := range []struct {
		val  Int
		want string
	}{
		// Add
		{f(math.MaxInt32).Add(f(1)), "80000000"},
		{f(math.MinInt32).Add(f(-1)), "-80000001"},
		// Mul
		{f(math.MaxInt32).Mul(f(math.MaxInt32)), "3fffffff00000001"},
		{f(math.MinInt32).Mul(f(math.MinInt32)), "4000000000000000"},
		{f(math.MaxUint32).Mul(f(math.MaxUint32)), "fffffffe00000001"},
		{f(math.MinInt32).Mul(f(-1)), "80000000"},
		// Div
		{f(math.MinInt32).Div(f(-1)), "80000000"},
		{f(1 << 31).Div(f(2)), "40000000"},
		// And
		{f(math.MaxInt32).And(f(math.MaxInt32)), "7fffffff"},
		{f(math.MinInt32).And(f(math.MinInt32)), "-80000000"},
		{f(1 << 33).And(f(1 << 32)), "0"},
		// Mod
		{f(1 << 32).Mod(f(2)), "0"},
		// Or
		{f(1 << 32).Or(f(0)), "100000000"},
		{f(math.MaxInt32).Or(f(0)), "7fffffff"},
		{f(math.MaxUint32).Or(f(0)), "ffffffff"},
		{f(math.MinInt32).Or(f(math.MinInt32)), "-80000000"},
		// Xor
		{f(math.MinInt32).Xor(f(-1)), "7fffffff"},
		// Not
		{f(math.MinInt32).Not(), "7fffffff"},
		{f(math.MaxInt32).Not(), "-80000000"},
		// Shift
		{f(1).Lsh(31), "80000000"},
		{f(1).Lsh(32), "100000000"},
		{f(math.MaxInt32 + 1).Rsh(1), "40000000"},
		{f(math.MinInt32 * 2).Rsh(1), "-80000000"},
	} {
		if got := fmt.Sprintf("%x", test.val); got != test.want {
			t.Errorf("%d equals %s, want %s", i, got, test.want)
		}
		small, big := test.val.get()
		if small < math.MinInt32 || math.MaxInt32 < small {
			t.Errorf("expected big, %d %s", i, test.val)
		}
		if big == nil {
			continue
		}
		if small != 0 {
			t.Errorf("expected 0 small, %d %s with %d", i, test.val, small)
		}
		if big.Cmp(left) >= 0 && big.Cmp(right) <= 0 {
			t.Errorf("expected small, %d %s", i, test.val)
		}
	}
}

func TestImmutabilityMakeBigInt(t *testing.T) {
	// use max int64 for the test
	expect := int64(^uint64(0) >> 1)

	mutint := big.NewInt(expect)
	value := MakeBigInt(mutint)
	mutint.Set(big.NewInt(1))

	got, _ := value.Int64()
	if got != expect {
		t.Errorf("expected %d, got %d", expect, got)
	}
}

func TestImmutabilityBigInt(t *testing.T) {
	// use 1 and max int64 for the test
	for _, expect := range []int64{1, int64(^uint64(0) >> 1)} {
		value := MakeBigInt(big.NewInt(expect))

		bigint := value.BigInt()
		bigint.Set(big.NewInt(2))

		got, _ := value.Int64()
		if got != expect {
			t.Errorf("expected %d, got %d", expect, got)
		}
	}
}

// TestIntFallback creates a small Int value in a child process with
// limited address space to ensure that it still works, but prints a warning.
func TestIntFallback(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skipf("test disabled on this platform (requires ulimit -v)")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("can't find file name of executable: %v", err)
	}
	// ulimit -v limits the address space in KB. Not portable.
	// 1GB is enough for the Go runtime but not for the optimization.
	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("ulimit -v 1000000 && %q --entry=intfallback", exe))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("intfallback subcommand failed: %v\n%s", err, out)
	}

	// Check the warning was printed.
	if !strings.Contains(string(out), "Integer performance may suffer") {
		t.Errorf("expected warning was not printed. Output=<<%s>>", out)
	}
}

// intfallback is called in a child process with limited address space.
func intfallback() {
	const want = 123
	if got, _ := MakeBigInt(big.NewInt(want)).Int64(); got != want {
		log.Fatalf("intfallback: got %d, want %d", got, want)
	}
}

// The --entry flag invokes an alternate entry point, for use in subprocess tests.
func TestMain(m *testing.M) {
	var entry string
	flag.StringVar(&entry, "entry", "", "child process entry-point")
	flag.Parse()
	switch entry {
	case "":
		os.Exit(m.Run()) // normal case
	case "intfallback":
		intfallback()
	default:
		log.Fatalf("unknown entry point: %s", entry)
	}
}
