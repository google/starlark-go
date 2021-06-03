package truth

import "testing"

func TestIsAtLeast(t *testing.T) {
	testEach(t, map[string]error{
		`that(5).is_at_least(3)`: nil,
		`that(5).is_at_least(5)`: nil,
		`that(5).is_at_least(8)`: fail("5", "is at least <8>"),
	})
}

func TestIsAtMost(t *testing.T) {
	testEach(t, map[string]error{
		`that(5).is_at_most(5)`: nil,
		`that(5).is_at_most(8)`: nil,
		`that(5).is_at_most(3)`: fail("5", "is at most <3>"),
	})
}

func TestIsGreaterThan(t *testing.T) {
	testEach(t, map[string]error{
		`that(5).is_greater_than(3)`: nil,
		`that(5).is_greater_than(5)`: fail("5", "is greater than <5>"),
		`that(5).is_greater_than(8)`: fail("5", "is greater than <8>"),
	})
}

func TestIsLessThan(t *testing.T) {
	testEach(t, map[string]error{
		`that(5).is_less_than(8)`: nil,
		`that(5).is_less_than(5)`: fail("5", "is less than <5>"),
		`that(5).is_less_than(3)`: fail("5", "is less than <3>"),
	})
}
