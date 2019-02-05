package starlark

// This file defines a simple spell checker for use in attribute errors
// ("no such field .foo; did you mean .food?")

import (
	"strings"
	"unicode"
)

// nearest returns the element of candidates
// nearest to x using the Levenshtein metric.
func nearest(x string, candidates []string) string {
	// Ignore underscores and case when matching.
	fold := func(s string) string {
		return strings.Map(func(r rune) rune {
			if r == '_' {
				return -1
			}
			return unicode.ToLower(r)
		}, s)
	}

	x = fold(x)

	var best string
	bestD := (len(x) + 1) / 2 // allow up to 50% typos
	for _, c := range candidates {
		d := levenshtein(x, fold(c), bestD)
		if d < bestD {
			bestD = d
			best = c
		}
	}
	return best
}

// levenshtein returns the non-negative Levenshtein edit distance
// between the byte strings x and y.
//
// If the computed distance exceeds max,
// the function may return early with an approximate value > max.
func levenshtein(x, y string, max int) int {
	// This implementation is derived from one by Laurent Le Brun in
	// Bazel that uses the single-row space efficiency trick
	// described at bitbucket.org/clearer/iosifovich.

	// Let x be the shorter string.
	if len(x) > len(y) {
		x, y = y, x
	}

	// Remove common prefix.
	for i := 0; i < len(x); i++ {
		if x[i] != y[i] {
			x = x[i:]
			y = y[i:]
			break
		}
	}
	if x == "" {
		return len(y)
	}

	row := make([]int, len(y)+1)
	for i := range row {
		row[i] = i
	}

	for i := 1; i <= len(x); i++ {
		row[0] = i
		best := i
		prev := i - 1
		for j := 1; j <= len(y); j++ {
			a := prev + b2i(x[i-1] != y[j-1]) // substitution
			b := 1 + row[j-1]                 // deletion
			c := 1 + row[j]                   // insertion
			k := min(a, min(b, c))
			prev, row[j] = row[j], k
			best = min(best, k)
		}
		if best > max {
			return best
		}
	}
	return row[len(y)]
}

func min(x, y int) int {
	if x < y {
		return x
	} else {
		return y
	}
}
