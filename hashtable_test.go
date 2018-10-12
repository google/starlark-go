// Copyright 2017 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package skylark

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestHashtable(t *testing.T) {
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
	// TODO(adonovan): MakeInt probably dominates the cost of this benchmark.
	// Optimise or avoid it.
	for i := 0; i < b.N; i++ {
		testHashtable(b, nil)
	}
}

// testHashtable is both a test and a benchmark of hashtable.
// When sane != nil, it acts as a test against the semantics of Go's map.
func testHashtable(tb testing.TB, sane map[int]bool) {
	zipf := rand.NewZipf(rand.New(rand.NewSource(0)), 1.1, 1.0, 1000.0)
	var ht hashtable

	// Insert 10000 random ints into the map.
	for j := 0; j < 10000; j++ {
		k := int(zipf.Uint64())
		if err := ht.insert(MakeInt(k), None); err != nil {
			tb.Fatal(err)
		}
		if sane != nil {
			sane[k] = true
		}
	}

	// Do 10000 random lookups in the map.
	for j := 0; j < 10000; j++ {
		k := int(zipf.Uint64())
		_, found, err := ht.lookup(MakeInt(k))
		if err != nil {
			tb.Fatal(err)
		}
		if sane != nil {
			_, found2 := sane[k]
			if found != found2 {
				tb.Fatal("sanity check failed")
			}
		}
	}

	// Do 10000 random deletes from the map.
	for j := 0; j < 10000; j++ {
		k := int(zipf.Uint64())
		_, found, err := ht.delete(MakeInt(k))
		if err != nil {
			tb.Fatal(err)
		}
		if sane != nil {
			_, found2 := sane[k]
			if found != found2 {
				tb.Fatal("sanity check failed")
			}
			delete(sane, k)
		}
	}

	if sane != nil {
		if int(ht.len) != len(sane) {
			tb.Fatal("sanity check failed")
		}
	}
}
