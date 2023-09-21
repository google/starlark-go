// Copyright 2018 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package starlark_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.starlark.net/lib/json"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarktest"
)

func BenchmarkStarlark(b *testing.B) {
	starlark.Universe["json"] = json.Module

	testdata := starlarktest.DataFile("starlark", ".")
	thread := new(starlark.Thread)
	for _, file := range []string{
		"testdata/benchmark.star",
		// ...
	} {

		filename := filepath.Join(testdata, file)

		src, err := os.ReadFile(filename)
		if err != nil {
			b.Error(err)
			continue
		}
		opts := getOptions(string(src))

		// Evaluate the file once.
		globals, err := starlark.ExecFileOptions(opts, thread, filename, src, nil)
		if err != nil {
			reportEvalError(b, err)
		}

		// Repeatedly call each global function named bench_* as a benchmark.
		for _, name := range globals.Keys() {
			value := globals[name]
			if fn, ok := value.(*starlark.Function); ok && strings.HasPrefix(name, "bench_") {
				b.Run(name, func(b *testing.B) {
					_, err := starlark.Call(thread, fn, starlark.Tuple{benchmark{b}}, nil)
					if err != nil {
						reportEvalError(b, err)
					}
				})
			}
		}
	}
}

// A benchmark is passed to each bench_xyz(b) function in a bench_*.star file.
// It provides b.n, the number of iterations that must be executed by the function,
// which is typically of the form:
//
//	def bench_foo(b):
//	   for _ in range(b.n):
//	      ...work...
//
// It also provides stop, start, and restart methods to stop the clock in case
// there is significant set-up work that should not count against the measured
// operation.
//
// (This interface is inspired by Go's testing.B, and is also implemented
// by the java.starlark.net implementation; see
// https://github.com/bazelbuild/starlark/pull/75#pullrequestreview-275604129.)
type benchmark struct {
	b *testing.B
}

func (benchmark) Freeze()               {}
func (benchmark) Truth() starlark.Bool  { return true }
func (benchmark) Type() string          { return "benchmark" }
func (benchmark) String() string        { return "<benchmark>" }
func (benchmark) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: benchmark") }
func (benchmark) AttrNames() []string   { return []string{"n", "restart", "start", "stop"} }
func (b benchmark) Attr(name string) (starlark.Value, error) {
	switch name {
	case "n":
		return starlark.MakeInt(b.b.N), nil
	case "restart":
		return benchmarkRestart.BindReceiver(b), nil
	case "start":
		return benchmarkStart.BindReceiver(b), nil
	case "stop":
		return benchmarkStop.BindReceiver(b), nil
	}
	return nil, nil
}

var (
	benchmarkRestart = starlark.NewBuiltin("restart", benchmarkRestartImpl)
	benchmarkStart   = starlark.NewBuiltin("start", benchmarkStartImpl)
	benchmarkStop    = starlark.NewBuiltin("stop", benchmarkStopImpl)
)

func benchmarkRestartImpl(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	b.Receiver().(benchmark).b.ResetTimer()
	return starlark.None, nil
}

func benchmarkStartImpl(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	b.Receiver().(benchmark).b.StartTimer()
	return starlark.None, nil
}

func benchmarkStopImpl(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	b.Receiver().(benchmark).b.StopTimer()
	return starlark.None, nil
}

// BenchmarkProgram measures operations relevant to compiled programs.
// TODO(adonovan): use a bigger testdata program.
func BenchmarkProgram(b *testing.B) {
	// Measure time to read a source file (approx 600us but depends on hardware and file system).
	filename := starlarktest.DataFile("starlark", "testdata/paths.star")
	var src []byte
	b.Run("read", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var err error
			src, err = os.ReadFile(filename)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	// Measure time to turn a source filename into a compiled program (approx 450us).
	var prog *starlark.Program
	b.Run("compile", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var err error
			_, prog, err = starlark.SourceProgram(filename, src, starlark.StringDict(nil).Has)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	// Measure time to encode a compiled program to a memory buffer
	// (approx 20us; was 75-120us with gob encoding).
	var out bytes.Buffer
	b.Run("encode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			out.Reset()
			if err := prog.Write(&out); err != nil {
				b.Fatal(err)
			}
		}
	})

	// Measure time to decode a compiled program from a memory buffer
	// (approx 20us; was 135-250us with gob encoding)
	b.Run("decode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			in := bytes.NewReader(out.Bytes())
			if _, err := starlark.CompiledProgram(in); err != nil {
				b.Fatal(err)
			}
		}
	})
}
