package starlark_test

// This file measures allocations and call overhead with execution
// profiling both disabled and enabled.

import (
	"testing"

	"go.starlark.net/starlark"
)

func prepareWarmedCallBenchmark(tb testing.TB, profile *starlark.ExecProfile) (*starlark.Thread, starlark.Value) {
	tb.Helper()
	_, prog, err := starlark.SourceProgram("bench.star", "def f():\n\tpass\n", func(string) bool { return false })
	if err != nil {
		tb.Fatal(err)
	}
	thread := new(starlark.Thread)
	globals, err := prog.Init(thread, nil)
	if err != nil {
		tb.Fatal(err)
	}
	f := globals["f"]
	thread.Profile = profile
	if _, err := starlark.Call(thread, f, nil, nil); err != nil {
		tb.Fatal(err)
	}
	return thread, f
}

func TestProfileNoAlloc(t *testing.T) {
	allocs := func(profile *starlark.ExecProfile) float64 {
		thread, f := prepareWarmedCallBenchmark(t, profile)
		return testing.AllocsPerRun(1000, func() {
			if _, err := starlark.Call(thread, f, nil, nil); err != nil {
				t.Fatal(err)
			}
		})
	}

	off := allocs(nil)
	on := allocs(new(starlark.ExecProfile))

	if on > off {
		t.Errorf("%v allocs/call with a profile attached, %v without: the profiler must not allocate per call", on, off)
	}
}

func benchmarkCall(b *testing.B, profile *starlark.ExecProfile) {
	thread, f := prepareWarmedCallBenchmark(b, profile)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := starlark.Call(thread, f, nil, nil); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkProfileOff(b *testing.B) { benchmarkCall(b, nil) }

func BenchmarkProfileOn(b *testing.B) { benchmarkCall(b, new(starlark.ExecProfile)) }

func benchmarkScriptCalls(b *testing.B, profile *starlark.ExecProfile) {
	const calls = 1000
	_, prog, err := starlark.SourceProgram("bench.star",
		"def f():\n\tpass\n\ndef loop(r):\n\tfor _ in r:\n\t\tf()\n", func(string) bool { return false })
	if err != nil {
		b.Fatal(err)
	}
	thread := new(starlark.Thread)
	globals, err := prog.Init(thread, nil)
	if err != nil {
		b.Fatal(err)
	}
	r := make(starlark.Tuple, calls)
	for i := range r {
		r[i] = starlark.MakeInt(i)
	}
	loop := globals["loop"]
	args := starlark.Tuple{r}
	thread.Profile = profile
	if _, err := starlark.Call(thread, loop, args, nil); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	for b.Loop() {
		if _, err := starlark.Call(thread, loop, args, nil); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*calls), "ns/call")
}

func BenchmarkProfileScriptOff(b *testing.B) { benchmarkScriptCalls(b, nil) }
func BenchmarkProfileScriptOn(b *testing.B) {
	benchmarkScriptCalls(b, new(starlark.ExecProfile))
}
