// Copyright 2017 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file defines tests of the Value API.
package skylark

import (
	"testing"
)

func TestListAppend(t *testing.T) {
	l := NewList(nil)
	l.Append(String("hello"))
	res, ok := AsString(l.Index(0))
	if !ok {
		t.Errorf("failed list.Append() got: %s, want: skylark.String", l.Index(0).Type())
	}
	if res != "hello" {
		t.Errorf("failed list.Append() got: %+v, want: hello", res)
	}
}
