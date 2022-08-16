package truth

import "testing"

func TestIsEqualTo(t *testing.T) {
	testEach(t, map[string]error{
		`that(5).is_equal_to(5)`: nil,
		`that(5).is_equal_to(3)`: fail("5", "is equal to <3>"),
		`that({1:2,3:4}).is_equal_to([1,2,3,4])`: fail(`{1: 2, 3: 4}`,
			"is equal to <[1, 2, 3, 4]>"),
	})
}

func TestIsEqualToFailsOnFloatsAsWellAsWithFormattedRepresentations(t *testing.T) {
	testEach(t, map[string]error{
		`that(0.3).is_equal_to(0.1+0.2)`: fail("0.3", "is equal to <0.30000000000000004>"),
		`that(0.1+0.2).is_equal_to(0.3)`: fail("0.30000000000000004", "is equal to <0.3>"),
	})
}

func TestIsNotEqualTo(t *testing.T) {
	testEach(t, map[string]error{
		`that(5).is_not_equal_to(3)`: nil,
		`that(5).is_not_equal_to(5)`: fail("5", "is not equal to <5>"),
	})
}

func TestSequenceIsEqualToUsesContainsExactlyElementsInPlusInOrder(t *testing.T) {
	testEach(t, map[string]error{
		`that((3,5,[])).is_equal_to((3, 5, []))`: nil,
		`that((3,5,[])).is_equal_to(([],3,5))`: fail("(3, 5, [])",
			"contains exactly these elements in order <([], 3, 5)>"),
		`that((3,5,[])).is_equal_to((3,5,[],9))`: fail("(3, 5, [])",
			"contains exactly <(3, 5, [], 9)>. It is missing <9>"),
		`that((3,5,[])).is_equal_to((9,3,5,[],10))`: fail("(3, 5, [])",
			"contains exactly <(9, 3, 5, [], 10)>. It is missing <9, 10>"),
		`that((3,5,[])).is_equal_to((3,5))`: fail("(3, 5, [])",
			"contains exactly <(3, 5)>. It has unexpected items <[]>"),
		`that((3,5,[])).is_equal_to(([],3))`: fail("(3, 5, [])",
			"contains exactly <([], 3)>. It has unexpected items <5>"),
		`that((3,5,[])).is_equal_to((3,))`: fail("(3, 5, [])",
			"contains exactly <(3,)>. It has unexpected items <5, []>"),
		`that((3,5,[])).is_equal_to((4,4,3,[],5))`: fail("(3, 5, [])",
			"contains exactly <(4, 4, 3, [], 5)>. It is missing <4 [2 copies]>"),
		`that((3,5,[])).is_equal_to((4,4))`: fail("(3, 5, [])",
			"contains exactly <(4, 4)>. It is missing <4 [2 copies]> and has unexpected items <3, 5, []>"),
		`that((3,5,[])).is_equal_to((3,5,9))`: fail("(3, 5, [])",
			"contains exactly <(3, 5, 9)>. It is missing <9> and has unexpected items <[]>"),
		`that((3,5,[])).is_equal_to(())`: fail("(3, 5, [])", "is empty"),
	})
}

func TestSetIsEqualToUsesContainsExactlyElementsIn(t *testing.T) {
	s := `that(set([3, 5, 8]))`
	testEach(t, map[string]error{
		s + `.is_equal_to(set([3, 5, 8]))`: nil,
		s + `.is_equal_to(set([8, 3, 5]))`: nil,
		s + `.is_equal_to(set([3, 5, 8, 9]))`: fail("set([3, 5, 8])",
			"contains exactly <set([3, 5, 8, 9])>. It is missing <9>"),
		s + `.is_equal_to(set([9, 3, 5, 8, 10]))`: fail("set([3, 5, 8])",
			"contains exactly <set([9, 3, 5, 8, 10])>. It is missing <9, 10>"),
		s + `.is_equal_to(set([3, 5]))`: fail("set([3, 5, 8])",
			"contains exactly <set([3, 5])>. It has unexpected items <8>"),
		s + `.is_equal_to(set([8, 3]))`: fail("set([3, 5, 8])",
			"contains exactly <set([8, 3])>. It has unexpected items <5>"),
		s + `.is_equal_to(set([3]))`: fail("set([3, 5, 8])",
			"contains exactly <set([3])>. It has unexpected items <5, 8>"),
		s + `.is_equal_to(set([4]))`: fail("set([3, 5, 8])",
			"contains exactly <set([4])>. It is missing <4> and has unexpected items <3, 5, 8>"),
		s + `.is_equal_to(set([3, 5, 9]))`: fail("set([3, 5, 8])",
			"contains exactly <set([3, 5, 9])>. It is missing <9> and has unexpected items <8>"),
		s + `.is_equal_to(set([]))`: fail("set([3, 5, 8])", "is empty"),
	})
}

func TestSequenceIsEqualToComparedWithNonIterables(t *testing.T) {
	testEach(t, map[string]error{
		`that((3, 5, [])).is_equal_to(3)`: fail("(3, 5, [])", "is equal to <3>"),
	})
}

func TestSetIsEqualToComparedWithNonIterables(t *testing.T) {
	testEach(t, map[string]error{
		`that(set([3, 5, 8])).is_equal_to(3)`: fail("set([3, 5, 8])", "is equal to <3>"),
	})
}

func TestOrderedDictIsEqualToUsesContainsExactlyItemsInPlusInOrder(t *testing.T) {
	d1 := `((2, "two"), (4, "four"))`
	d2 := `((2, "two"), (4, "four"))`
	d3 := `((4, "four"), (2, "two"))`
	d4 := `((2, "two"), (4, "for"))`
	d5 := `((2, "two"), (4, "four"), (5, "five"))`
	s := `that(` + d1 + `).is_equal_to(`
	testEach(t, map[string]error{
		s + d2 + `)`: nil,
		s + d3 + `)`: fail(d1, "contains exactly these elements in order <"+d3+">"),

		s + `((2, "two"),))`: fail(d1,
			`contains exactly <((2, "two"),)>. It has unexpected items <(4, "four")>`),

		s + d4 + `)`: fail(d1,
			"contains exactly <"+d4+`>. It is missing <(4, "for")> and has unexpected items <(4, "four")>`),
		s + d5 + `)`: fail(d1,
			"contains exactly <"+d5+`>. It is missing <(5, "five")>`),
	})
}

func TestDictIsEqualToUsesContainsExactlyItemsIn(t *testing.T) {
	d := `{2: "two", 4: "four"}`
	dd := `{2: "two", 4: "for"}`
	ddd := `{2: "two", 4: "four", 5: "five"}`
	dBis := `items([(2, "two"), (4, "four")])`
	s := `that(` + d + `).is_equal_to(`
	testEach(t, map[string]error{
		s + d + `)`: nil,

		s + `{2: "two"})`: fail(dBis,
			`contains exactly <((2, "two"),)>. It has unexpected items <(4, "four")>`,
			warnContainsExactlySingleIterable),

		s + dd + `)`: fail(dBis,
			`contains exactly <((2, "two"), (4, "for"))>. It is missing <(4, "for")> and has unexpected items <(4, "four")>`),
		s + ddd + `)`: fail(dBis,
			`contains exactly <((2, "two"), (4, "four"), (5, "five"))>. It is missing <(5, "five")>`),
		s + `{})`: fail(`items([(2, "two"), (4, "four")])`, "is empty"),
	})
}

func TestIsEqualToComparedWithNonDictionary(t *testing.T) {
	ss := `{2: "two", 4: "four"}`
	testEach(t, map[string]error{
		`that(` + ss + `).is_equal_to(3)`: fail(ss, `is equal to <3>`),
	})
}

func TestNamedMultilineString(t *testing.T) {
	s := `that("line1\nline2").named("some-name")`
	testEach(t, map[string]error{
		s + `.is_equal_to("line1\nline2")`: nil,
		s + `.is_equal_to("")`: newTruthAssertion(
			`Not true that actual some-name is equal to <"">.`),
		s + `.is_equal_to("line1\nline2\n")`: newTruthAssertion(
			`Not true that actual some-name is equal to expected, found diff:
*** Expected
--- Actual
***************
*** 1,3 ****
  line1
  line2
- 
--- 1,2 ----
.`),
	})
}

func TestIsEqualToRaisesErrorWithVerboseDiff(t *testing.T) {
	testEach(t, map[string]error{
		`that("line1\nline2\nline3\nline4\nline5\n") \
         .is_equal_to("line1\nline2\nline4\nline6\n")`: newTruthAssertion(
			`Not true that actual is equal to expected, found diff:
*** Expected
--- Actual
***************
*** 1,5 ****
  line1
  line2
  line4
! line6
  
--- 1,6 ----
  line1
  line2
+ line3
  line4
! line5
  
.`),
	})
}
