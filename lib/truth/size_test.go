package truth

import "testing"

func TestHasSize(t *testing.T) {
	s := func(x string) string {
		return `that((2, 5, 8)).has_size(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`3`):  nil,
		s(`-1`): fail(`(2, 5, 8)`, `has a size of <-1>. It is <3>`),
		s(`2`):  fail(`(2, 5, 8)`, `has a size of <2>. It is <3>`),
	})
}

func TestIsEmpty(t *testing.T) {
	s := func(t string) string {
		return `that(` + t + `).is_empty()`
	}
	testEach(t, map[string]error{
		s(`()`):       nil,
		s(`[]`):       nil,
		s(`{}`):       nil,
		s(`set([])`):  nil,
		s(`""`):       nil,
		s(`(3,)`):     fail(`(3,)`, `is empty`),
		s(`[4]`):      fail(`[4]`, `is empty`),
		s(`{5: 6}`):   fail(`{5: 6}`, `is empty`),
		s(`set([7])`): fail(`set([7])`, `is empty`),
		s(`"height"`): fail(`"height"`, `is empty`),
	})
}

func TestIsNotEmpty(t *testing.T) {
	s := func(t string) string {
		return `that(` + t + `).is_not_empty()`
	}
	testEach(t, map[string]error{
		s(`(3,)`):     nil,
		s(`[4]`):      nil,
		s(`{5: 6}`):   nil,
		s(`set([7])`): nil,
		s(`"height"`): nil,
		s(`()`):       fail(`()`, `is not empty`),
		s(`[]`):       fail(`[]`, `is not empty`),
		s(`{}`):       fail(`{}`, `is not empty`),
		s(`set([])`):  fail(`set([])`, `is not empty`),
		s(`""`):       fail(`""`, `is not empty`),
	})
}
