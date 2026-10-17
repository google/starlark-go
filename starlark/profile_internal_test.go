// Copyright 2026 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package starlark

import "testing"

func TestProfEventName(t *testing.T) {
	// Enable profiling without starting the profiler goroutine,
	// so the event from Call waits in the channel until we read it.
	profiler.events = make(chan *profEvent, 1)
	defer func() { profiler.events = nil }()

	thread := &Thread{Name: "a"}
	thread.proftime = quantum // force a sample even for a short call
	f := NewBuiltin("f", func(*Thread, *Builtin, Tuple, []Tuple) (Value, error) { return None, nil })
	if _, err := Call(thread, f, nil, nil); err != nil {
		t.Fatal(err)
	}
	thread.Name = "b" // rename after the event was recorded

	select {
	case ev := <-profiler.events:
		if ev.name != "a" {
			t.Errorf("event name = %q, want %q", ev.name, "a")
		}
	default:
		t.Fatal("Call recorded no profile event")
	}
}
