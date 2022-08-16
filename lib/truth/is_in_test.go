package truth

import "testing"

func TestIsIn(t *testing.T) {
	s := func(x string) string {
		return `that(3).is_in(` + x + `)`
	}
	testEach(t, map[string]error{
		`that("a").is_in("abc")`: nil,
		`that("d").is_in("abc")`: fail(`"d"`, `is equal to any of <"abc">`),
		s(`(3,)`):                nil,
		s(`(3, 5)`):              nil,
		s(`(1, 3, 5)`):           nil,
		s(`{3: "three"}`):        nil,
		s(`set([3, 5])`):         nil,
		s(`()`):                  fail(`3`, `is equal to any of <()>`),
		s(`(2,)`):                fail(`3`, `is equal to any of <(2,)>`),
	})
}

func TestIsNotIn(t *testing.T) {
	s := func(x string) string {
		return `that(3).is_not_in(` + x + `)`
	}
	testEach(t, map[string]error{
		`that("a").is_not_in("abc")`: fail(`"a"`, `is not in "abc". It was found at index 0`),
		`that("d").is_not_in("abc")`: nil,
		s(`(5,)`):                    nil,
		s(`set([5])`):                nil,
		s(`("3",)`):                  nil,
		s(`(3,)`):                    fail(`3`, `is not in (3,). It was found at index 0`),
		s(`(1, 3)`):                  fail(`3`, `is not in (1, 3). It was found at index 1`),
		s(`set([3])`):                fail(`3`, `is not in set([3])`),
	})
}
