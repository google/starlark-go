package truth

import "testing"

func TestHasAttribute(t *testing.T) {
	s := func(x string) string {
		return `that("my str").has_attribute(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`"elems"`):    nil,
		s(`"index"`):    nil,
		s(`"isdigit"`):  nil,
		s(`""`):         fail(`"my str"`, `has attribute <"">`),
		s(`"ermagerd"`): fail(`"my str"`, `has attribute <"ermagerd">`),
	})
}

func TestDoesNotHaveAttribute(t *testing.T) {
	s := func(x string) string {
		return `that({1: ()}).does_not_have_attribute(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`"other_attribute"`): nil,
		s(`""`):                nil,
		s(`"keys"`):            fail(`{1: ()}`, `does not have attribute <"keys">`),
		s(`"values"`):          fail(`{1: ()}`, `does not have attribute <"values">`),
		s(`"setdefault"`):      fail(`{1: ()}`, `does not have attribute <"setdefault">`),
	})
}

func TestIsCallable(t *testing.T) {
	s := func(t string) string {
		return `that(` + t + `).is_callable()`
	}
	testEach(t, map[string]error{
		s(`lambda x: x`):    nil,
		s(`"str".endswith`): nil,
		s(`that`):           nil,
		s(`None`):           fail(`None`, `is callable`),
		s(abc):              fail(abc, `is callable`),
	})
}

func TestIsNotCallable(t *testing.T) {
	testEach(t, map[string]error{
		`assert.that(assert.that).is_not_callable()`: fail(`built-in method assert of assert value`, `is not callable`),
	}, asModule)
	s := func(t string) string {
		return `that(` + t + `).is_not_callable()`
	}
	testEach(t, map[string]error{
		s(`None`):           nil,
		s(abc):              nil,
		s(`lambda x: x`):    fail(`function lambda`, `is not callable`),
		s(`"str".endswith`): fail(`built-in method endswith of string value`, `is not callable`),
		s(`that`):           fail(`built-in function that`, `is not callable`),
	})
}
