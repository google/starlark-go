// Copyright 2017 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package chunkedfile provides utilities for testing that source code
// errors are reported in the appropriate places.
//
// A chunked file consists of several chunks of input text separated by
// "---" lines.  Each chunk is an input to the program under test, such
// as an evaluator.  Lines containing "###" are interpreted as
// expectations of failure: the following text is a Go string literal
// denoting a regular expression that should match the failure message.
//
// Example:
//
//      x = 1 / 0 ### "division by zero"
//      ---
//      x = 1
//      print(x + "") ### "int + string not supported"
//
// A client test feeds each chunk of text into the program under test,
// then calls chunk.GotError for each error that actually occurred.  Any
// discrepancy between the actual and expected errors is reported using
// the client's reporter, which is typically a testing.T.
package chunkedfile // import "go.starlark.net/internal/chunkedfile"

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

const debug = false

// A Chunk is a portion of a source file.
// It contains a set of expected errors.
type Chunk struct {
	Source   string
	filename string
	report   Reporter
	wantErrs map[int]*regexp.Regexp
}

// Reporter is implemented by *testing.T.
type Reporter interface {
	Errorf(format string, args ...interface{})
}

// Read parses a chunked file and returns its chunks.
// It reports failures using the reporter.
//
// Error messages of the form "file.star:line:col: ..." are prefixed
// by a newline so that the Go source position added by (*testing.T).Errorf
// appears on a separate line so as not to confused editors.
func Read(filename string, report Reporter) (chunks []Chunk) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		report.Errorf("%s", err)
		return
	}
	linenum := 1

	eol := "\n"
	if runtime.GOOS == "windows" {
		eol = "\r\n"
	}

	for i, chunk := range strings.Split(string(data), eol+"---"+eol) {
		if debug {
			fmt.Printf("chunk %d at line %d: %s\n", i, linenum, chunk)
		}
		// Pad with newlines so the line numbers match the original file.
		src := strings.Repeat("\n", linenum-1) + chunk

		wantErrs := make(map[int]*regexp.Regexp)

		// Parse comments of the form:
		// ### "expected error".
		lines := strings.Split(chunk, "\n")
		for j := 0; j < len(lines); j, linenum = j+1, linenum+1 {
			line := lines[j]
			hashes := strings.Index(line, "###")
			if hashes < 0 {
				continue
			}
			rest := strings.TrimSpace(line[hashes+len("###"):])
			pattern, err := strconv.Unquote(rest)
			if err != nil {
				report.Errorf("\n%s:%d: not a quoted regexp: %s", filename, linenum, rest)
				continue
			}
			rx, err := regexp.Compile(pattern)
			if err != nil {
				report.Errorf("\n%s:%d: %v", filename, linenum, err)
				continue
			}
			wantErrs[linenum] = rx
			if debug {
				fmt.Printf("\t%d\t%s\n", linenum, rx)
			}
		}
		linenum++

		chunks = append(chunks, Chunk{src, filename, report, wantErrs})
	}
	return chunks
}

// GotError should be called by the client to report an error at a particular line.
// GotError reports unexpected errors to the chunk's reporter.
func (chunk *Chunk) GotError(linenum int, msg string) {
	if rx, ok := chunk.wantErrs[linenum]; ok {
		delete(chunk.wantErrs, linenum)
		if !rx.MatchString(msg) {
			chunk.report.Errorf("\n%s:%d: error %q does not match pattern %q", chunk.filename, linenum, msg, rx)
		}
	} else {
		chunk.report.Errorf("\n%s:%d: unexpected error: %v", chunk.filename, linenum, msg)
	}
}

// Done should be called by the client to indicate that the chunk has no more errors.
// Done reports expected errors that did not occur to the chunk's reporter.
func (chunk *Chunk) Done() {
	for linenum, rx := range chunk.wantErrs {
		chunk.report.Errorf("\n%s:%d: expected error matching %q", chunk.filename, linenum, rx)
	}
}
