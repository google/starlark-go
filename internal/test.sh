#!/bin/sh

# Copyright 2021 The Bazel Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

set -eu

# Confirm that go.mod and go.sum are tidy
cp go.mod go.mod.orig
cp go.sum go.sum.orig
go mod tidy
# Use -w to ignore differences in OS newlines.
diff -w  go.mod.orig go.mod
diff -w go.sum.orig go.sum
rm go.mod.orig go.sum.orig

# Run tests
go test ./...
