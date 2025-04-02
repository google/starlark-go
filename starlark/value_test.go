// Copyright 2017 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package starlark_test

// This file defines tests of the Value API.

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.starlark.net/starlark"
)

func TestStringMethod(t *testing.T) {
	s := starlark.String("hello")
	for i, test := range [][2]string{
		// quoted string:
		{s.String(), `"hello"`},
		{fmt.Sprintf("%s", s), `"hello"`},
		{fmt.Sprintf("%+s", s), `"hello"`},
		{fmt.Sprintf("%v", s), `"hello"`},
		{fmt.Sprintf("%+v", s), `"hello"`},
		// unquoted:
		{s.GoString(), `hello`},
		{fmt.Sprintf("%#v", s), `hello`},
	} {
		got, want := test[0], test[1]
		if got != want {
			t.Errorf("#%d: got <<%s>>, want <<%s>>", i, got, want)
		}
	}
}

func TestListAppend(t *testing.T) {
	l := starlark.NewList(nil)
	l.Append(starlark.String("hello"))
	res, ok := starlark.AsString(l.Index(0))
	if !ok {
		t.Errorf("failed list.Append() got: %s, want: starlark.String", l.Index(0).Type())
	}
	if res != "hello" {
		t.Errorf("failed list.Append() got: %+v, want: hello", res)
	}
}

func TestParamDefault(t *testing.T) {
	tests := []struct {
		desc         string
		starFn       string
		wantDefaults []starlark.Value
	}{
		{
			desc:         "function with all required params",
			starFn:       "all_required",
			wantDefaults: []starlark.Value{nil, nil, nil},
		},
		{
			desc:   "function with all optional params",
			starFn: "all_opt",
			wantDefaults: []starlark.Value{
				starlark.String("a"),
				starlark.None,
				starlark.String(""),
			},
		},
		{
			desc:   "function with required and optional params",
			starFn: "mix_required_opt",
			wantDefaults: []starlark.Value{
				nil,
				nil,
				starlark.String("c"),
				starlark.String("d"),
			},
		},
		{
			desc:   "function with required, optional, and varargs params",
			starFn: "with_varargs",
			wantDefaults: []starlark.Value{
				nil,
				starlark.String("b"),
				nil,
			},
		},
		{
			desc:   "function with required, optional, varargs, and keyword-only params",
			starFn: "with_varargs_kwonly",
			wantDefaults: []starlark.Value{
				nil,
				starlark.String("b"),
				starlark.String("c"),
				nil,
				nil,
			},
		},
		{
			desc:   "function with required, optional, and keyword-only params",
			starFn: "with_kwonly",
			wantDefaults: []starlark.Value{
				nil,
				starlark.String("b"),
				starlark.String("c"),
				nil,
			},
		},
		{
			desc:   "function with required, optional, and kwargs params",
			starFn: "with_kwargs",
			wantDefaults: []starlark.Value{
				nil,
				starlark.String("b"),
				starlark.String("c"),
				nil,
			},
		},
		{
			desc:   "function with required, optional, varargs, kw-only, and kwargs params",
			starFn: "with_varargs_kwonly_kwargs",
			wantDefaults: []starlark.Value{
				nil,
				starlark.String("b"),
				starlark.String("c"),
				nil,
				starlark.String("e"),
				nil,
				nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			thread := &starlark.Thread{}
			filename := "testdata/function_param.star"
			globals, err := starlark.ExecFile(thread, filename, nil, nil)
			if err != nil {
				t.Fatalf("ExecFile(%v, %q) failed: %v", thread, filename, err)
			}

			fn, ok := globals[tt.starFn].(*starlark.Function)
			if !ok {
				t.Fatalf("value %v is not a Starlark function", globals[tt.starFn])
			}

			var paramDefaults []starlark.Value
			for i := 0; i < fn.NumParams(); i++ {
				paramDefaults = append(paramDefaults, fn.ParamDefault(i))
			}
			if diff := cmp.Diff(tt.wantDefaults, paramDefaults); diff != "" {
				t.Errorf("param defaults got diff (-want +got):\n%s", diff)
			}
		})
	}
}

// TestListSliceConcurrency tests that list slicing operations correctly prevent
// concurrent modifications, adhering to the "fail-fast" iterator principle.
// It simulates two goroutines: one repeatedly slicing the list and another
// attempting to modify it concurrently. The test verifies that modifications
// are blocked during slicing and can proceed once slicing is complete.
func TestListSliceConcurrency(t *testing.T) {
	// Setup: Create a large list
	const listSize = 10000
	elems := make([]starlark.Value, listSize)
	for i := range elems {
		elems[i] = starlark.MakeInt(i)
	}
	list := starlark.NewList(elems)

	var wg sync.WaitGroup
	wg.Add(2)

	// Channels to signal completion or errors from goroutines
	slicerDone := make(chan struct{})
	modifierErrChan := make(chan error, 1)

	// Goroutine A: Repeatedly performs slicing operations. It should not panic.
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Slicing goroutine panicked unexpectedly: %v", r)
			}
			close(slicerDone)
		}()

		for i := 0; i < 100; i++ {
			_ = list.Slice(0, list.Len()/2, 1)
			time.Sleep(1 * time.Millisecond)
		}
	}()

	// Goroutine B: Attempts to modify the list while Goroutine A is potentially slicing.
	// It expects to receive an error indicating the list is being iterated.
	go func() {
		defer wg.Done()
		var firstError error
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Modifying goroutine panicked unexpectedly: %v", r)
			}
			modifierErrChan <- firstError
			close(modifierErrChan)
		}()

		time.Sleep(5 * time.Millisecond)

		for i := 0; i < 50; i++ {
			// Attempt to append. This should fail if the slicer has incremented itercount.
			err := list.Append(starlark.MakeInt(listSize))
			if err != nil {
				if firstError == nil {
					firstError = err
				}
				break
			}
			time.Sleep(2 * time.Millisecond)
		}
	}()

	wg.Wait()
	<-slicerDone

	// 1. Check if the modifier goroutine encountered the expected error.
	modifyErr := <-modifierErrChan
	if modifyErr == nil {
		t.Errorf("Modifier goroutine did not encounter an error. Expected modification to be blocked by concurrent slicing. List len: %d", list.Len())
	} else {
		// Verify the error is the specific "iteration" error.
		// Using Errorf for specific error formatting from starlark package.
		expectedErrStr := "cannot append to list during iteration"
		if modifyErr.Error() != expectedErrStr {
			t.Errorf("Modifier goroutine encountered wrong error.\nWant: %q\nGot:  %q", expectedErrStr, modifyErr.Error())
		}
	}

	// 2. Verify the list can be modified *after* concurrent operations are finished.
	lengthBeforeFinalAppend := list.Len()
	finalValue := starlark.MakeInt(listSize + 1)
	finalAppendErr := list.Append(finalValue)
	if finalAppendErr != nil {
		t.Fatalf("Append after concurrent operations failed unexpectedly: %v", finalAppendErr)
	}

	// 3. Check the final list length.
	expectedFinalLength := lengthBeforeFinalAppend + 1
	if list.Len() != expectedFinalLength {
		t.Errorf("Final list length mismatch.\nWant: %d (length after concurrency %d + 1)\nGot:  %d",
			expectedFinalLength, lengthBeforeFinalAppend, list.Len())
	}
}
