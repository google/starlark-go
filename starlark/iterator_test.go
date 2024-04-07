// Copyright 2024 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build go1.23

package starlark_test

// This file defines tests of the starlark.Value Go API's go1.23 iterators:
//
//  ({Tuple,*List,(Set}).Elements
//  Elements
//  (*Dict).Entries
//  Entries

import (
	"fmt"
	"reflect"
	"testing"

	. "go.starlark.net/starlark"
)

func TestTupleElements(t *testing.T) {
	tuple := Tuple{MakeInt(1), MakeInt(2), MakeInt(3)}

	var got []string
	for elem := range tuple.Elements() {
		got = append(got, fmt.Sprint(elem))
		if len(got) == 2 {
			break // skip 3
		}
	}
	for elem := range Elements(tuple) {
		got = append(got, fmt.Sprint(elem))
		if len(got) == 4 {
			break // skip 3
		}
	}
	want := []string{"1", "2", "1", "2"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestListElements(t *testing.T) {
	list := NewList([]Value{MakeInt(1), MakeInt(2), MakeInt(3)})

	var got []string
	for elem := range list.Elements() {
		got = append(got, fmt.Sprint(elem))
		if len(got) == 2 {
			break // skip 3
		}
	}
	for elem := range Elements(list) {
		got = append(got, fmt.Sprint(elem))
		if len(got) == 4 {
			break // skip 3
		}
	}
	want := []string{"1", "2", "1", "2"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSetElements(t *testing.T) {
	set := NewSet(3)
	set.Insert(MakeInt(1))
	set.Insert(MakeInt(2))
	set.Insert(MakeInt(3))

	var got []string
	for elem := range set.Elements() {
		got = append(got, fmt.Sprint(elem))
		if len(got) == 2 {
			break // skip 3
		}
	}
	for elem := range Elements(set) {
		got = append(got, fmt.Sprint(elem))
		if len(got) == 4 {
			break // skip 3
		}
	}
	want := []string{"1", "2", "1", "2"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDictEntries(t *testing.T) {
	dict := NewDict(2)
	dict.SetKey(String("one"), MakeInt(1))
	dict.SetKey(String("two"), MakeInt(2))
	dict.SetKey(String("three"), MakeInt(3))

	var got []string
	for k, v := range dict.Entries() {
		got = append(got, fmt.Sprintf("%v %v", k, v))
		if len(got) == 2 {
			break // skip 3
		}
	}
	for k, v := range Entries(dict) {
		got = append(got, fmt.Sprintf("%v %v", k, v))
		if len(got) == 4 {
			break // skip 3
		}
	}
	want := []string{`"one" 1`, `"two" 2`, `"one" 1`, `"two" 2`}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
