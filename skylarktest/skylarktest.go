// Copyright 2017 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package skylarktest defines utilities for testing Skylark programs.
//
// Clients can call LoadAssertModule to load a module that defines
// several functions useful for testing.  See assert.sky for its
// definition.
//
// The assert.error function, which reports errors to the current Go
// testing.T, requires that clients call SetTest(thread, t) before use.
package skylarktest

import (
	"bytes"
	"fmt"
	"go/build"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/google/skylark"
	"github.com/google/skylark/skylarkstruct"
)

const localKey = "Reporter"

// A Reporter is a value to which errors may be reported.
// It is satisfied by *testing.T.
type Reporter interface {
	Error(args ...interface{})
}

// SetReporter associates an error reporter (such as a testing.T in
// a Go test) with the Skylark thread so that Skylark programs may
// report errors to it.
func SetReporter(thread *skylark.Thread, r Reporter) {
	thread.SetLocal(localKey, r)
}

// GetReporter returns the Skylark thread's error reporter.
// It must be preceded by a call to SetReporter.
func GetReporter(thread *skylark.Thread) Reporter {
	r, ok := thread.Local(localKey).(Reporter)
	if !ok {
		panic("internal error: skylarktest.SetReporter was not called")
	}
	return r
}

var (
	once      sync.Once
	assert    skylark.StringDict
	assertErr error
)

// LoadAssertModule loads the assert module.
// It is concurrency-safe and idempotent.
func LoadAssertModule() (skylark.StringDict, error) {
	once.Do(func() {
		predeclared := skylark.StringDict{
			"error":   skylark.NewBuiltin("error", error_),
			"catch":   skylark.NewBuiltin("catch", catch),
			"matches": skylark.NewBuiltin("matches", matches),
			"struct":  skylark.NewBuiltin("struct", skylarkstruct.Make),
			"_freeze": skylark.NewBuiltin("freeze", freeze),
		}
		filename := DataFile("skylark/skylarktest", "assert.sky")
		thread := new(skylark.Thread)
		assert, assertErr = skylark.ExecFile(thread, filename, nil, predeclared)
	})
	return assert, assertErr
}

// catch(f) evaluates f() and returns its evaluation error message
// if it failed or None if it succeeded.
func catch(thread *skylark.Thread, _ *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
	var fn skylark.Callable
	if err := skylark.UnpackArgs("catch", args, kwargs, "fn", &fn); err != nil {
		return nil, err
	}
	if _, err := skylark.Call(thread, fn, nil, nil); err != nil {
		return skylark.String(err.Error()), nil
	}
	return skylark.None, nil
}

// matches(pattern, str) reports whether string str matches the regular expression pattern.
func matches(thread *skylark.Thread, _ *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
	var pattern, str string
	if err := skylark.UnpackArgs("matches", args, kwargs, "pattern", &pattern, "str", &str); err != nil {
		return nil, err
	}
	ok, err := regexp.MatchString(pattern, str)
	if err != nil {
		return nil, fmt.Errorf("matches: %s", err)
	}
	return skylark.Bool(ok), nil
}

// error(x) reports an error to the Go test framework.
func error_(thread *skylark.Thread, _ *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("error: got %d arguments, want 1", len(args))
	}
	var buf bytes.Buffer
	thread.Caller().WriteBacktrace(&buf)
	buf.WriteString("Error: ")
	if s, ok := skylark.AsString(args[0]); ok {
		buf.WriteString(s)
	} else {
		buf.WriteString(args[0].String())
	}
	GetReporter(thread).Error(buf.String())
	return skylark.None, nil
}

// freeze(x) freezes its operand.
func freeze(thread *skylark.Thread, _ *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
	if len(kwargs) > 0 {
		return nil, fmt.Errorf("freeze does not accept keyword arguments")
	}
	if len(args) != 1 {
		return nil, fmt.Errorf("freeze got %d arguments, wants 1", len(args))
	}
	args[0].Freeze()
	return args[0], nil
}

// DataFile returns the effective filename of the specified
// test data resource.  The function abstracts differences between
// 'go build', under which a test runs in its package directory,
// and Blaze, under which a test runs in the root of the tree.
var DataFile = func(pkgdir, filename string) string {
	return filepath.Join(build.Default.GOPATH, "src/github.com/google", pkgdir, filename)
}
