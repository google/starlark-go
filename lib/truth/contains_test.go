package truth

import "testing"

func TestContainsExactly(t *testing.T) {
	ss := `(3, 5, [])`
	s := func(x string) string {
		return `that(` + ss + `).contains_exactly(` + x + `)`
	}
	testEach(t, map[string]error{
		`that(` + ss + `).contains_exactly(3, 5, []).in_order()`: nil,
		`that(` + ss + `).contains_exactly(3, 5, [])`:            nil,
		`that(` + ss + `).contains_exactly([], 3, 5)`:            nil,
		`that(` + ss + `).contains_exactly([], 3, 5).in_order()`: fail(ss,
			"contains exactly these elements in order <([], 3, 5)>"),

		s(`3, 5, [], 9`): fail(ss,
			`contains exactly <(3, 5, [], 9)>. It is missing <9>`),
		s(`9, 3, 5, [], 10`): fail(ss,
			`contains exactly <(9, 3, 5, [], 10)>. It is missing <9, 10>`),
		s(`3, 5`): fail(ss,
			`contains exactly <(3, 5)>. It has unexpected items <[]>`),
		s(`[], 3`): fail(ss,
			`contains exactly <([], 3)>. It has unexpected items <5>`),
		s(`3`): fail(ss,
			`contains exactly <(3,)>. It has unexpected items <5, []>`),
		s(`4, 4`): fail(ss,
			`contains exactly <(4, 4)>. It is missing <4 [2 copies]> and has unexpected items <3, 5, []>`),
		s(`3, 5, 9`): fail(ss,
			`contains exactly <(3, 5, 9)>. It is missing <9> and has unexpected items <[]>`),
		s(`(3, 5, [])`): fail(ss,
			`contains exactly <((3, 5, []),)>. It is missing <(3, 5, [])> and has unexpected items <3, 5, []>`,
			warnContainsExactlySingleIterable),
		s(``): fail(ss, "is empty"),
	})
}

func TestContainsExactlyDoesNotWarnIfSingleStringNotContained(t *testing.T) {
	s := `.contains_exactly("abc")`
	testEach(t, map[string]error{
		`that(())` + s:      fail(`()`, `contains exactly <("abc",)>. It is missing <"abc">`),
		`that([])` + s:      fail(`[]`, `contains exactly <("abc",)>. It is missing <"abc">`),
		`that({})` + s:      errMustBeEqualNumberOfKVPairs(1),
		`that("")` + s:      fail(`""`, `contains exactly <("abc",)>. It is missing <"abc">`),
		`that(set([]))` + s: fail(`set([])`, `contains exactly <("abc",)>. It is missing <"abc">`),
	})
}

func TestContainsExactlyEmptyContainer(t *testing.T) {
	s := func(x string) string {
		return `that(` + x + `).contains_exactly(3)`
	}
	testEach(t, map[string]error{
		s(`()`): fail(`()`, `contains exactly <(3,)>. It is missing <3>`),
		s(`[]`): fail(`[]`, `contains exactly <(3,)>. It is missing <3>`),
		s(`{}`): errMustBeEqualNumberOfKVPairs(1),
		s(`""`): fail(`""`, `contains exactly <(3,)>. It is missing <3>`),
		//FIXME: Not true that <''> contains exactly <(3,)>. It is missing <[3]>. warnContainsExactlySingleIterable
		s(`set([])`): fail(`set([])`, `contains exactly <(3,)>. It is missing <3>`),
	})
}

func TestContainsExactlyNothing(t *testing.T) {
	s := func(x string) string {
		return `that(` + x + `).contains_exactly()`
	}
	testEach(t, map[string]error{
		s(`()`):      nil,
		s(`[]`):      nil,
		s(`{}`):      nil,
		s(`""`):      nil,
		s(`set([])`): nil,
	})
}

func TestContainsExactlyElementsIn(t *testing.T) {
	ss := `(3, 5, [])`
	s := func(x string) string {
		return `that(` + ss + `).contains_exactly_elements_in(` + x + `)`
	}
	testEach(t, map[string]error{
		`that(` + ss + `).contains_exactly_elements_in((3, 5, [])).in_order()`: nil,
		`that(` + ss + `).contains_exactly_elements_in(([], 3, 5))`:            nil,
		`that(` + ss + `).contains_exactly_elements_in(([], 3, 5)).in_order()`: fail(ss,
			"contains exactly these elements in order <([], 3, 5)>"),

		s(`(3, 5, [], 9)`):     fail(ss, `contains exactly <(3, 5, [], 9)>. It is missing <9>`),
		s(`(9, 3, 5, [], 10)`): fail(ss, `contains exactly <(9, 3, 5, [], 10)>. It is missing <9, 10>`),
		s(`(3, 5)`):            fail(ss, `contains exactly <(3, 5)>. It has unexpected items <[]>`),
		s(`([], 3)`):           fail(ss, `contains exactly <([], 3)>. It has unexpected items <5>`),
		s(`(3,)`):              fail(ss, `contains exactly <(3,)>. It has unexpected items <5, []>`),
		s(`(4, 4)`):            fail(ss, `contains exactly <(4, 4)>. It is missing <4 [2 copies]> and has unexpected items <3, 5, []>`),
		s(`(3, 5, 9)`):         fail(ss, `contains exactly <(3, 5, 9)>. It is missing <9> and has unexpected items <[]>`),
		s(`()`):                fail(ss, `is empty`),
	})
}

func TestContainsExactlyElementsInEmptyContainer(t *testing.T) {
	testEach(t, map[string]error{
		`that(()).contains_exactly_elements_in(())`: nil,
		`that(()).contains_exactly_elements_in((3,))`: fail(`()`,
			`contains exactly <(3,)>. It is missing <3>`),
	})
}

func TestContainsExactlyTargetingOrderedDict(t *testing.T) {
	ss := `((2, "two"), (4, "four"))`
	s := `that(` + ss + `).contains_exactly(`
	testEach(t, map[string]error{
		`that(` + ss + `).contains_exactly((2, "two"), (4, "four")).in_order()`: nil,
		`that(` + ss + `).contains_exactly((2, "two"), (4, "four"))`:            nil,
		`that(` + ss + `).contains_exactly((4, "four"), (2, "two"))`:            nil,
		`that(` + ss + `).contains_exactly((4, "four"), (2, "two")).in_order()`: fail(ss,
			`contains exactly these elements in order <((4, "four"), (2, "two"))>`),

		s + `2, "two")`: fail(ss,
			`contains exactly <(2, "two")>. It is missing <2, "two"> and has unexpected items <(2, "two"), (4, "four")>`),

		s + `2, "two", 4, "for")`: fail(ss,
			`contains exactly <(2, "two", 4, "for")>. It is missing <2, "two", 4, "for"> and has unexpected items <(2, "two"), (4, "four")>`),

		s + `2, "two", 4, "four", 5, "five")`: fail(ss,
			`contains exactly <(2, "two", 4, "four", 5, "five")>. It is missing <2, "two", 4, "four", 5, "five"> and has unexpected items <(2, "two"), (4, "four")>`),
	})
}

func TestContainsExactlyPassingOddNumberOfArgs(t *testing.T) {
	testEach(t, map[string]error{
		`that({}).contains_exactly("key1", "value1", "key2")`: errMustBeEqualNumberOfKVPairs(3),
	})
}

func TestContainsExactlyItemsIn(t *testing.T) {
	s := func(x string) string {
		return `that({2: "two", 4: "four"}).contains_exactly_items_in(` + x + `)`
	}
	ss := `items([(2, "two"), (4, "four")])`
	testEach(t, map[string]error{
		s(`{2: "two", 4: "four"}`): nil,

		s(`{}`) + `.in_order()`: fail(ss, `is empty`),

		s(`{4: "four", 2: "two"}`) + `.in_order()`: newInvalidAssertion("values of type dict are not ordered"),
		// Starlark's dict is not ordered (uses insertion order)
		s(`{2: "two", 4: "four"}`) + `.in_order()`: newInvalidAssertion("values of type dict are not ordered"),

		s(`{2: "two"}`): fail(ss,
			`contains exactly <((2, "two"),)>. It has unexpected items <(4, "four")>`,
			warnContainsExactlySingleIterable),

		s(`{2: "two", 4: "for"}`): fail(ss,
			`contains exactly <((2, "two"), (4, "for"))>. It is missing <(4, "for")> and has unexpected items <(4, "four")>`),

		s(`{2: "two", 4: "four", 5: "five"}`): fail(ss,
			`contains exactly <((2, "two"), (4, "four"), (5, "five"))>. It is missing <(5, "five")>`),
	})
}

func TestContains(t *testing.T) {
	s := func(x string) string {
		return `that((2, 5, [])).contains(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`2`):   nil,
		s(`5`):   nil,
		s(`[]`):  nil,
		s(`3`):   newTruthAssertion(`<(2, 5, [])> should have contained 3.`),
		s(`"2"`): newTruthAssertion(`<(2, 5, [])> should have contained "2".`),
		s(`{}`):  newTruthAssertion(`<(2, 5, [])> should have contained {}.`),
	})
}

func TestDoesNotContain(t *testing.T) {
	s := func(x string) string {
		return `that((2, 5, [])).does_not_contain(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`3`):   nil,
		s(`"2"`): nil,
		s(`{}`):  nil,
		s(`2`):   newTruthAssertion(`<(2, 5, [])> should not have contained 2.`),
		s(`5`):   newTruthAssertion(`<(2, 5, [])> should not have contained 5.`),
		s(`[]`):  newTruthAssertion(`<(2, 5, [])> should not have contained [].`),
	})
}

func TestContainsNoDuplicates(t *testing.T) {
	s := func(t string) string {
		return `that(` + t + `).contains_no_duplicates()`
	}
	testEach(t, map[string]error{
		s(`()`):           nil,
		s(abc):            nil,
		s(`(2,)`):         nil,
		s(`(2, 5)`):       nil,
		s(`{2: 2}`):       nil,
		s(`set([2])`):     nil,
		s(`"aaa"`):        newTruthAssertion(`<"aaa"> has the following duplicates: <"a" [3 copies]>.`),
		s(`(3, 2, 5, 2)`): newTruthAssertion(`<(3, 2, 5, 2)> has the following duplicates: <2 [2 copies]>.`),
		s(`"abcabc"`): newTruthAssertion(
			`<"abcabc"> has the following duplicates: <"a" [2 copies], "b" [2 copies], "c" [2 copies]>.`),
	})
}

func TestContainsAllIn(t *testing.T) {
	ss := `(3, 5, [])`
	s := func(x string) string {
		return `that(` + ss + `).contains_all_in(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`()`):                                nil,
		`that(` + ss + `).contains_all_in(())`: nil,
		`that(` + ss + `).contains_all_in(()).in_order()`:         nil,
		`that(` + ss + `).contains_all_in((3,))`:                  nil,
		`that(` + ss + `).contains_all_in((3,)).in_order()`:       nil,
		`that(` + ss + `).contains_all_in((3, []))`:               nil,
		`that(` + ss + `).contains_all_in((3, [])).in_order()`:    nil,
		`that(` + ss + `).contains_all_in((3, 5, []))`:            nil,
		`that(` + ss + `).contains_all_in((3, 5, [])).in_order()`: nil,
		`that(` + ss + `).contains_all_in(([], 5, 3))`:            nil,
		`that(` + ss + `).contains_all_in(([], 5, 3)).in_order()`: fail(ss,
			`contains all elements in order <([], 5, 3)>`),
		s(`(2, 3)`):    fail(ss, "contains all elements in <(2, 3)>. It is missing <2>"),
		s(`(2, 3, 6)`): fail(ss, "contains all elements in <(2, 3, 6)>. It is missing <2, 6>"),
	})
}

func TestContainsAllOf(t *testing.T) {
	ss := `(3, 5, [])`
	s := func(x string) string {
		return `that(` + ss + `).contains_all_of(` + x + `)`
	}
	testEach(t, map[string]error{
		`that(` + ss + `).contains_all_of()`:                    nil,
		`that(` + ss + `).contains_all_of().in_order()`:         nil,
		`that(` + ss + `).contains_all_of(3)`:                   nil,
		`that(` + ss + `).contains_all_of(3).in_order()`:        nil,
		`that(` + ss + `).contains_all_of(3, [])`:               nil,
		`that(` + ss + `).contains_all_of(3, []).in_order()`:    nil,
		`that(` + ss + `).contains_all_of(3, 5, [])`:            nil,
		`that(` + ss + `).contains_all_of(3, 5, []).in_order()`: nil,
		`that(` + ss + `).contains_all_of([], 3, 5)`:            nil,
		`that(` + ss + `).contains_all_of([], 3, 5).in_order()`: fail(ss,
			`contains all elements in order <([], 3, 5)>`),
		s(`2, 3`):    fail(ss, "contains all of <(2, 3)>. It is missing <2>"),
		s(`2, 3, 6`): fail(ss, "contains all of <(2, 3, 6)>. It is missing <2, 6>"),
	})
}

func TestContainsWithStrings(t *testing.T) {
	testEach(t, map[string]error{
		`that("abcdefg").contains_all_of("a", "c", "e")`:            nil,
		`that("abcdefg").contains_all_of("a", "c", "e").in_order()`: nil,
		`that("abcdefg").contains_any_in(("a", "c", "e"))`:          nil,
		`that("abcdefg").contains_none_of("x", "z", "y")`:           nil,
	})
}

func TestContainsAllMixedHashableElements(t *testing.T) {
	ss := `(3, [], 5, 8)`
	s := func(x string) string {
		return `that(` + ss + `).contains_all_of(` + x + `)`
	}
	testEach(t, map[string]error{
		`that(` + ss + `).contains_all_of(3, [], 5, 8)`:            nil,
		`that(` + ss + `).contains_all_of(3, [], 5, 8).in_order()`: nil,
		`that(` + ss + `).contains_all_of(5, 3, 8, [])`:            nil,
		`that(` + ss + `).contains_all_of(5, 3, 8, []).in_order()`: fail(ss,
			`contains all elements in order <(5, 3, 8, [])>`),
		s(`3, [], 8, 5, 9`):  fail(ss, "contains all of <(3, [], 8, 5, 9)>. It is missing <9>"),
		s(`3, [], 8, 5, {}`): fail(ss, "contains all of <(3, [], 8, 5, {})>. It is missing <{}>"),
		s(`8, 3, [], 9, 5`):  fail(ss, "contains all of <(8, 3, [], 9, 5)>. It is missing <9>"),
	})
}

func TestContainsAnyIn(t *testing.T) {
	ss := `(3, 5, [])`
	s := func(x string) string {
		return `that(` + ss + `).contains_any_in(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`(3,)`):   nil,
		s(`(7, 3)`): nil,
		s(`()`):     fail(ss, "contains any element in <()>"),
		s(`(2, 6)`): fail(ss, "contains any element in <(2, 6)>"),
	})
}

func TestContainsAnyOf(t *testing.T) {
	ss := `(3, 5, [])`
	s := func(x string) string {
		return `that(` + ss + `).contains_any_of(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`3`):    nil,
		s(`7, 3`): nil,
		s(``):     fail(ss, "contains any of <()>"),
		s(`2, 6`): fail(ss, "contains any of <(2, 6)>"),
	})
}

func TestContainsNoneIn(t *testing.T) {
	ss := `(3, 5, [])`
	s := func(x string) string {
		return `that(` + ss + `).contains_none_in(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`()`):     nil,
		s(`(2,)`):   nil,
		s(`(2, 6)`): nil,
		s(`(5,)`):   fail(ss, "contains no elements in <(5,)>. It contains <5>"),
		s(`(2, 5)`): fail(ss, "contains no elements in <(2, 5)>. It contains <5>"),
	})
}

func TestContainsNoneOf(t *testing.T) {
	ss := `(3, 5, [])`
	s := func(x string) string {
		return `that(` + ss + `).contains_none_of(` + x + `)`
	}
	testEach(t, map[string]error{
		s(``):     nil,
		s(`2`):    nil,
		s(`2, 6`): nil,
		s(`5`):    fail(ss, "contains none of <(5,)>. It contains <5>"),
		s(`2, 5`): fail(ss, "contains none of <(2, 5)>. It contains <5>"),
	})
}

func TestContainsKey(t *testing.T) {
	ss := `{2: "two", None: "None"}`
	s := func(x string) string {
		return `that(` + ss + `).contains_key(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`2`):     nil,
		s(`None`):  nil,
		s(`3`):     fail(ss, `contains key <3>`),
		s(`"two"`): fail(ss, `contains key <"two">`),
	})
}

func TestDoesNotContainKey(t *testing.T) {
	ss := `{2: "two", None: "None"}`
	s := func(x string) string {
		return `that(` + ss + `).does_not_contain_key(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`3`):     nil,
		s(`"two"`): nil,
		s(`2`):     fail(ss, `does not contain key <2>`),
		s(`None`):  fail(ss, `does not contain key <None>`),
	})
}

func TestContainsItem(t *testing.T) {
	ss := `{2: "two", 4: "four", "too": "two"}`
	s := func(x string) string {
		return `that(` + ss + `).contains_item(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`2, "two"`):     nil,
		s(`4, "four"`):    nil,
		s(`"too", "two"`): nil,
		s(`2, "to"`):      fail(ss, `contains item <(2, "to")>. However, it has a mapping from <2> to <"two">`),
		s(`7, "two"`): fail(ss, `contains item <(7, "two")>.`+
			` However, the following keys are mapped to <"two">: [2, "too"]`),
		s(`7, "seven"`): fail(ss, `contains item <(7, "seven")>`),
	})
}

func TestDoesNotContainItem(t *testing.T) {
	ss := `{2: "two", 4: "four", "too": "two"}`
	s := func(x string) string {
		return `that(` + ss + `).does_not_contain_item(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`2, "to"`):    nil,
		s(`7, "two"`):   nil,
		s(`7, "seven"`): nil,
		s(`2, "two"`):   fail(ss, `does not contain item <(2, "two")>`),
		s(`4, "four"`):  fail(ss, `does not contain item <(4, "four")>`),
	})
}
