package truth

import "testing"

func TestHasLength(t *testing.T) {
	ss := abc
	s := func(x string) string {
		return `that(` + ss + `).has_length(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`3`): nil,
		s(`4`): fail(ss, `has a length of 4. It is 3`),
		s(`2`): fail(ss, `has a length of 2. It is 3`),
	})
}

func TestStartsWith(t *testing.T) {
	ss := abc
	s := func(x string) string {
		return `that(` + ss + `).starts_with(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`""`):   nil,
		s(`"a"`):  nil,
		s(`"ab"`): nil,
		s(abc):    nil,
		s(`"b"`):  fail(ss, `starts with <"b">`),
	})
}

func TestEndsWith(t *testing.T) {
	ss := abc
	s := func(x string) string {
		return `that(` + ss + `).ends_with(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`""`):   nil,
		s(`"c"`):  nil,
		s(`"bc"`): nil,
		s(abc):    nil,
		s(`"b"`):  fail(ss, `ends with <"b">`),
	})
}

func TestMatches(t *testing.T) {
	ss := abc
	s := func(x string) string {
		return `that(` + ss + `).matches(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`"a"`):     nil,
		s(`r".b"`):   nil,
		s(`r"[Aa]"`): nil,
		s(`"d"`):     fail(ss, `matches <"d">`),
		s(`"b"`):     fail(ss, `matches <"b">`),
	})
}

func TestDoesNotMatch(t *testing.T) {
	ss := abc
	s := func(x string) string {
		return `that(` + ss + `).does_not_match(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`"b"`):     nil,
		s(`"d"`):     nil,
		s(`"a"`):     fail(ss, `fails to match <"a">`),
		s(`r".b"`):   fail(ss, `fails to match <".b">`),
		s(`r"[Aa]"`): fail(ss, `fails to match <"[Aa]">`),
	})
}

func TestContainsMatch(t *testing.T) {
	ss := abc
	s := func(x string) string {
		return `that(` + ss + `).contains_match(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`"a"`):     nil,
		s(`r".b"`):   nil,
		s(`r"[Aa]"`): nil,
		s(`"b"`):     nil,
		s(`"d"`):     fail(ss, `should have contained a match for <"d">`),
	})
}

func TestDoesNotContainMatch(t *testing.T) {
	ss := abc
	s := func(x string) string {
		return `that(` + ss + `).does_not_contain_match(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`"d"`):     nil,
		s(`"a"`):     fail(ss, `should not have contained a match for <"a">`),
		s(`"b"`):     fail(ss, `should not have contained a match for <"b">`),
		s(`r".b"`):   fail(ss, `should not have contained a match for <".b">`),
		s(`r"[Aa]"`): fail(ss, `should not have contained a match for <"[Aa]">`),
	})
}
