// Copyright 2026 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file exposes internal declarations to tests.

package starlark

func UnpackArgNoEscape(v Value, ptr any) error {
	return unpackArgNoEscape(v, ptr)
}

func UnpackPositionalArgsNoEscape(fnname string, args Tuple, kwargs []Tuple, min int, vars ...any) error {
	return unpackPositionalArgsNoEscape(fnname, args, kwargs, min, vars...)
}
