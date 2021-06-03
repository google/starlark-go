package truth

import (
	"fmt"
	"strings"

	"go.starlark.net/starlark"
)

var _ fmt.Stringer = (*duplicateCounter)(nil)

// duplicateCounter is a synchronized collection of counters for tracking duplicates.
//
// The count values may be modified only through Increment() and Decrement(),
// which increment and decrement by 1 (only). If a count ever becomes 0, the item
// is immediately expunged from the dictionary. Counts can never be negative;
// attempting to Decrement an absent key has no effect.
//
// Order is preserved so that error messages containing expected values match.
//
// Supports counting values based on their (starlark.Value).String() representation.
// TODO: track hashable objects in a hashmap. Use a slice and equality for non-hashables.
type duplicateCounter struct {
	m map[string]uint
	s []string
	d uint
}

func newDuplicateCounter() *duplicateCounter {
	return &duplicateCounter{
		m: make(map[string]uint),
	}
}

// HasDupes indicates whether there are values that appears > 1 times.
func (dc *duplicateCounter) HasDupes() bool { return dc.d != 0 }

func (dc *duplicateCounter) Empty() bool { return len(dc.m) == 0 }

func (dc *duplicateCounter) Len() int { return len(dc.m) }

func (dc *duplicateCounter) Contains(v starlark.Value) bool {
	return dc.contains(v.String())
}

func (dc *duplicateCounter) contains(v string) bool {
	_, ok := dc.m[v]
	return ok
}

// Increment increments a count by 1. Inserts the item if not present.
func (dc *duplicateCounter) Increment(v starlark.Value) {
	dc.increment(v.String())
}

func (dc *duplicateCounter) increment(v string) {
	if _, ok := dc.m[v]; !ok {
		dc.m[v] = 0
		dc.s = append(dc.s, v)
	}
	dc.m[v]++
	if dc.m[v] == 2 {
		dc.d++
	}
}

// Decrement decrements a count by 1. Expunges the item if the count is 0.
// If the item is not present, has no effect.
func (dc *duplicateCounter) Decrement(v starlark.Value) {
	dc.decrement(v.String())
}

func (dc *duplicateCounter) decrement(v string) {
	if count, ok := dc.m[v]; ok {
		if count != 1 {
			dc.m[v]--
			if dc.m[v] == 1 {
				dc.d--
			}
			return
		}
		delete(dc.m, v)
		if sz := len(dc.s); sz != 1 {
			s := make([]string, 0, len(dc.s)-1)
			for _, vv := range dc.s {
				if vv != v {
					s = append(s, vv)
				}
			}
			dc.s = s
		} else {
			dc.s = nil
		}
	}
}

// Returns the string representation of the duplicate counts.
//
// Items occurring more than once are accompanied by their count.
// Otherwise the count is implied to be 1.
//
// For example, if the internal dict is `{2: 1, 3: 4, "abc": 1}`, this returns
// the string `2, 3 [4 copies], "abc"`.
func (dc *duplicateCounter) String() string {
	var b strings.Builder
	for i, vv := range dc.s {
		if i != 0 {
			b.WriteString(", ")
		}

		b.WriteString(vv)
		if count := dc.m[vv]; count != 1 {
			b.WriteString(" [")
			b.WriteString(fmt.Sprintf("%d", count))
			b.WriteString(" copies]")
		}
	}
	return b.String()
}

// Dupes shows only items whose count > 1.
func (dc *duplicateCounter) Dupes() string {
	var b strings.Builder
	first := true
	for _, vv := range dc.s {
		if count := dc.m[vv]; count != 1 {
			if !first {
				b.WriteString(", ")
			}
			first = false

			b.WriteString(vv)
			b.WriteString(" [")
			b.WriteString(fmt.Sprintf("%d", count))
			b.WriteString(" copies]")
		}
	}
	return b.String()
}
