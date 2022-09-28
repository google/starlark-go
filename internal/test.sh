#!/bin/sh

# Copyright 2021 The Bazel Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

set -eu

# Confirm that go.mod and go.sum are tidy.
cp go.mod go.mod.orig
cp go.sum go.sum.orig
go mod tidy
# Use -w to ignore differences in OS newlines.
diff -w go.mod.orig go.mod || { echo "go.mod is not tidy"; exit 1; }
diff -w go.sum.orig go.sum || { echo "go.sum is not tidy"; exit 1; }
rm go.mod.orig go.sum.orig

# Run tests
go test ./...
