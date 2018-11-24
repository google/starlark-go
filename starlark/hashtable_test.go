// Copyright 2017 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package starlark

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
)

func TestHashtable(t *testing.T) {
	makeTestIntsOnce.Do(makeTestInts)
	testHashtable(t, make(map[int]bool))
}

func BenchmarkStringHash(b *testing.B) {
	for len := 1; len <= 1024; len *= 2 {
		buf := make([]byte, len)
		rand.New(rand.NewSource(0)).Read(buf)
		s := string(buf)

		b.Run(fmt.Sprintf("hard-%d", len), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				hashString(s)
			}
		})
		b.Run(fmt.Sprintf("soft-%d", len), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				softHashString(s)
			}
		})
	}
}

func BenchmarkHashtable(b *testing.B) {
	makeTestIntsOnce.Do(makeTestInts)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testHashtable(b, nil)
	}
}

const testIters = 10000

var (
	// testInts is a zipf-distributed array of Ints and corresponding ints.
	// This removes the cost of generating them on the fly during benchmarking.
	// Without this, Zipf and MakeInt dominate CPU and memory costs, respectively.
	makeTestIntsOnce sync.Once
	testInts         [3 * testIters]struct {
		Int   Int
		goInt int
	}
)

func makeTestInts() {
	zipf := rand.NewZipf(rand.New(rand.NewSource(0)), 1.1, 1.0, 1000.0)
	for i := range &testInts {
		r := int(zipf.Uint64())
		testInts[i].goInt = r
		testInts[i].Int = MakeInt(r)
	}
}

// testHashtable is both a test and a benchmark of hashtable.
// When sane != nil, it acts as a test against the semantics of Go's map.
func testHashtable(tb testing.TB, sane map[int]bool) {
	var i int // index into testInts

	var ht hashtable

	// Insert 10000 random ints into the map.
	for j := 0; j < testIters; j++ {
		k := testInts[i]
		i++
		if err := ht.insert(k.Int, None); err != nil {
			tb.Fatal(err)
		}
		if sane != nil {
			sane[k.goInt] = true
		}
	}

	// Do 10000 random lookups in the map.
	for j := 0; j < testIters; j++ {
		k := testInts[i]
		i++
		_, found, err := ht.lookup(k.Int)
		if err != nil {
			tb.Fatal(err)
		}
		if sane != nil {
			_, found2 := sane[k.goInt]
			if found != found2 {
				tb.Fatal("sanity check failed")
			}
		}
	}

	// Do 10000 random deletes from the map.
	for j := 0; j < testIters; j++ {
		k := testInts[i]
		i++
		_, found, err := ht.delete(k.Int)
		if err != nil {
			tb.Fatal(err)
		}
		if sane != nil {
			_, found2 := sane[k.goInt]
			if found != found2 {
				tb.Fatal("sanity check failed")
			}
			delete(sane, k.goInt)
		}
	}

	if sane != nil {
		if int(ht.len) != len(sane) {
			tb.Fatal("sanity check failed")
		}
	}
}
