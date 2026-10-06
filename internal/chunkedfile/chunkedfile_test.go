// Copyright 2017 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package chunkedfile

import (
	"fmt"
	"testing"
)

type testReporter struct {
	reported []string
}

func (r *testReporter) Errorf(format string, args ...interface{}) {
	formatted := fmt.Sprintf(format, args...)
	r.reported = append(r.reported, formatted)
}

func (r *testReporter) assertNone(t *testing.T) {
	if len(r.reported) > 0 {
		t.Errorf("reporter expected no errors, got %d", len(r.reported))
	}
}

func (r *testReporter) assertOne(t *testing.T, exp string) {
	if len(r.reported) != 1 {
		t.Fatalf("reporter expected 1 error, got %d", len(r.reported))
	}
	if r.reported[0] != exp {
		t.Fatalf("reporter expected %q, got %q", exp, r.reported[0])
	}
}

func (r *testReporter) reset() {
	r.reported = nil
}

func TestChunkedFile(t *testing.T) {
	data := []byte(`x = 1 / 0 ### "division by zero"
---
x = 1
print(x)
`)

	reporter := &testReporter{}
	chunks := readBytes("test_file", data, reporter, "\n")

	reporter.assertNone(t) // should not have reported any errors

	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}

	// Check the first chunk
	exp := "x = 1 / 0 ### \"division by zero\""
	chunk := chunks[0]
	if chunk.Source != exp {
		t.Fatalf("expected %q, got %q", exp, chunk.Source)
	}

	// First chunk has an expected error

	if len(chunk.wantErrs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(chunk.wantErrs))
	}

	exp = "division by zero"
	for _, re := range chunk.wantErrs {
		if re.String() != exp {
			t.Fatalf("expected %q, got %q", exp, re.String())
		}
	}

	reporter.assertNone(t) // still should not have reported any errors

	// Send an error that is expected.

	chunk.GotError(1, "division by zero")

	reporter.assertNone(t) // should not have reported any errors because the error was expected

	if len(chunk.wantErrs) != 0 {
		// We should have gobbled up th expected error from the chunk
		t.Fatalf("expected 0 errors, got %d", len(chunk.wantErrs))
	}

	// Send an error that is not expected (the same error as before).
	// Now the reporter should report it as an unexpected error.

	chunk.GotError(1, "division by zero")

	exp = "\ntest_file:1: unexpected error: division by zero"
	reporter.assertOne(t, exp)

	// Check the second chunk

	exp = "\n\nx = 1\nprint(x)\n"
	chunk = chunks[1]
	if chunk.Source != exp {
		t.Fatalf("expected %q, got %q", exp, chunk.Source)
	}

	// Second chunk does not have any expected errors

	if len(chunk.wantErrs) != 0 {
		t.Fatalf("expected 0 errors, got %d", len(chunk.wantErrs))
	}

	// Send an error that is not expected.
	// The reporter should make it an unexpected error.

	reporter.reset()
	chunk.GotError(123, "foobar")

	exp = "\ntest_file:123: unexpected error: foobar"
	reporter.assertOne(t, exp)

}
