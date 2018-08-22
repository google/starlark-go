// Copyright 2018 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package skylarktime_test

import (
	"log"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/skylark"
	"github.com/google/skylark/internal/chunkedfile"
	"github.com/google/skylark/resolve"
	"github.com/google/skylark/skylarktest"
	"github.com/google/skylark/skylarktime"
)

func init() {
	// The tests make extensive use of these not-yet-standard features.
	resolve.AllowLambda = true
	resolve.AllowNestedDef = true
	resolve.AllowFloat = true
	resolve.AllowSet = true

	// Fake the clock for test determinism.
	now, err := time.Parse(time.UnixDate, "Sat Mar  7 11:06:39 PST 2015")
	if err != nil {
		log.Fatal(err)
	}
	skylarktime.Now = func() time.Time { return now }
}

func TestExecFile(t *testing.T) {
	testdata := skylarktest.DataFile("skylark/skylarktime", ".")
	thread := &skylark.Thread{Load: load}
	skylarktest.SetReporter(thread, t)
	for _, file := range []string{
		"testdata/time.sky",
	} {
		filename := filepath.Join(testdata, file)
		for _, chunk := range chunkedfile.Read(filename, t) {
			_, err := skylark.ExecFile(thread, filename, chunk.Source, nil)
			switch err := err.(type) {
			case *skylark.EvalError:
				found := false
				for _, fr := range err.Stack() {
					posn := fr.Position()
					if posn.Filename() == filename {
						chunk.GotError(int(posn.Line), err.Error())
						found = true
						break
					}
				}
				if !found {
					t.Error(err.Backtrace())
				}
			case nil:
				// success
			default:
				t.Error(err)
			}
			chunk.Done()
		}
	}
}

// load implements the 'load' operation as used in the evaluator tests.
func load(thread *skylark.Thread, module string) (skylark.StringDict, error) {
	if module == "assert.sky" {
		return skylarktest.LoadAssertModule()
	}
	if module == "time.sky" {
		return skylarktime.LoadTimeModule()
	}

	// TODO(adonovan): test load() using this execution path.
	filename := filepath.Join(filepath.Dir(thread.Caller().Position().Filename()), module)
	return skylark.ExecFile(thread, filename, nil, nil)
}
