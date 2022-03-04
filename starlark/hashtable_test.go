// Copyright 2017 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package starlark

import (
	"fmt"
	"math/rand"
	"strconv"
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
	testStrings    [3 * testIters]string
	testStringDict StringDict = make(StringDict)
)

func makeTestInts() {
	zipf := rand.NewZipf(rand.New(rand.NewSource(0)), 1.1, 1.0, 1000.0)
	for i := range &testInts {
		r := int(zipf.Uint64())
		testInts[i].goInt = r
		testInts[i].Int = MakeInt(r)
		s := strconv.Itoa(r)
		testStrings[i] = s
		testStringDict[s] = None
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

func TestOrderedStringDict(t *testing.T) {
	makeTestIntsOnce.Do(makeTestInts)
	testOrderedStringDict(t, make(StringDict))
}

func testOrderedStringDict(tb testing.TB, sane StringDict) {
	var i int // index into testInts

	// Build the maps
	// Insert 10000 random ints into the map.
	d := OrderStringDict(testStringDict)
	for k, v := range testStringDict {
		sane[k] = v
	}

	// Do 10000 random lookups in the map.
	for j := 0; j < testIters; j++ {
		k := testStrings[i]
		i++
		_, found := d.Get(k)
		if sane != nil {
			_, found2 := sane[k]
			if found != found2 {
				tb.Fatal("sanity check failed")
			}
		}
	}

	// Do 10000 random sets from the map.
	for j := 0; j < testIters; j++ {
		k := testStrings[i]
		i++
		if !d.Set(k, None) {
			tb.Fatal("set failed")
		}
	}

	if sane != nil {
		if len(d.entries) != len(sane) {
			tb.Fatal("sanity check failed")
		}
	}
}

func benchmarkOrderedStringDict(b *testing.B, size int) {
	want := Bool(true)
	keys := testStringDict.Keys()
	sd := make(StringDict)
	for i := 0; i < size; i++ {
		key := keys[i]
		sd[key] = want
	}
	osd := OrderStringDict(sd)
	b.Run("map", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := keys[i%size]
			if sd[key] != want {
				b.Fatal("invalid value")
			}
		}
	})
	b.Run("order", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := keys[i%size]
			if val, _ := osd.Get(key); val != want {
				b.Fatal("invalid value")
			}
		}
	})
	b.Run("mapRange", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, v := range sd {
				if v != want {
					b.Fatal("invalid value")
				}
			}
		}
	})
	b.Run("orderRange", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			osd.Range(func(_ string, v Value) bool {
				if v != want {
					b.Fatal("invalid value")
				}
				return true
			})
		}
	})
	b.Run("orderIter", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for j := 0; j < osd.Len(); j++ {
				if v := osd.Index(j); v != want {
					b.Fatal("invalid value")
				}
			}
		}
	})
}

func BenchmarkOrderedStringDict_4(b *testing.B)   { benchmarkOrderedStringDict(b, 4) }
func BenchmarkOrderedStringDict_8(b *testing.B)   { benchmarkOrderedStringDict(b, 8) }
func BenchmarkOrderedStringDict_16(b *testing.B)  { benchmarkOrderedStringDict(b, 16) }
func BenchmarkOrderedStringDict_32(b *testing.B)  { benchmarkOrderedStringDict(b, 32) }
func BenchmarkOrderedStringDict_64(b *testing.B)  { benchmarkOrderedStringDict(b, 64) }
func BenchmarkOrderedStringDict_128(b *testing.B) { benchmarkOrderedStringDict(b, 128) }
