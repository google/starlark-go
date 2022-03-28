// Copyright 2020 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package starlarkjson is an alias for go.starlark.net/lib/json to provide
// backwards compatibility
//
// Deprecated: use go.starlark.net/lib/json instead
package starlarkjson // import "go.starlark.net/stalarkjson"

import (
	"go.starlark.net/lib/json"
)

// Module is an alias of json.Module for backwards import compatibility
var Module = json.Module
