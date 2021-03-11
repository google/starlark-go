package truth

import "testing"

func TestIsOrdered(t *testing.T) {
	s := func(t string) string {
		return `that(` + t + `).is_ordered()`
	}
	testEach(t, map[string]error{
		s(`()`):        nil,
		s(`(3,)`):      nil,
		s(`(3, 5, 8)`): nil,
		s(`(3, 5, 5)`): nil,
		s(`"abcdef"`):  nil,
		s(`"fedcba"`):  newTruthAssertion(`Not true that <"fedcba"> is ordered <("f", "e")>.`),
		s(`{5: 4}`):    newInvalidAssertion("values of type dict are not ordered"),
		s(`(5, 4)`):    newTruthAssertion(`Not true that <(5, 4)> is ordered <(5, 4)>.`),
		s(`(3, 5, 4)`): newTruthAssertion(`Not true that <(3, 5, 4)> is ordered <(5, 4)>.`),
	})
}

func TestIsOrderedAccordingTo(t *testing.T) {
	s := func(t string) string {
		return `that(` + t + `).is_ordered_according_to(someCmp)`
	}
	testEach(t, map[string]error{
		s(`()`):        nil,
		s(`(3,)`):      nil,
		s(`(8, 5, 3)`): nil,
		s(`(5, 5, 3)`): nil,
		s(`"fedcba"`):  nil,
		s(`"abcdef"`):  newTruthAssertion(`Not true that <"abcdef"> is ordered <("a", "b")>.`),
		s(`{5: 4}`):    newInvalidAssertion("values of type dict are not ordered"),
		s(`(4, 5)`):    newTruthAssertion(`Not true that <(4, 5)> is ordered <(4, 5)>.`),
		s(`(3, 5, 4)`): newTruthAssertion(`Not true that <(3, 5, 4)> is ordered <(3, 5)>.`),
	})
}

func TestIsStrictlyOrdered(t *testing.T) {
	s := func(t string) string {
		return `that(` + t + `).is_strictly_ordered()`
	}
	testEach(t, map[string]error{
		s(`()`):        nil,
		s(`(3,)`):      nil,
		s(`(3, 5, 8)`): nil,
		s(`"abcdef"`):  nil,
		s(`"abcdee"`):  newTruthAssertion(`Not true that <"abcdee"> is strictly ordered <("e", "e")>.`),
		s(`"fedcba"`):  newTruthAssertion(`Not true that <"fedcba"> is strictly ordered <("f", "e")>.`),
		s(`{5: 4}`):    newInvalidAssertion("values of type dict are not ordered"),
		s(`(5, 4)`):    newTruthAssertion(`Not true that <(5, 4)> is strictly ordered <(5, 4)>.`),
		s(`(3, 5, 5)`): newTruthAssertion(`Not true that <(3, 5, 5)> is strictly ordered <(5, 5)>.`),
	})
}

func TestIsStrictlyOrderedAccordingTo(t *testing.T) {
	s := func(t string) string {
		return `that(` + t + `).is_strictly_ordered_according_to(someCmp)`
	}
	testEach(t, map[string]error{
		s(`()`):        nil,
		s(`(3,)`):      nil,
		s(`(8, 5, 3)`): nil,
		s(`"fedcba"`):  nil,
		s(`"fedcbb"`):  newTruthAssertion(`Not true that <"fedcbb"> is strictly ordered <("b", "b")>.`),
		s(`"abcdef"`):  newTruthAssertion(`Not true that <"abcdef"> is strictly ordered <("a", "b")>.`),
		s(`{5: 4}`):    newInvalidAssertion("values of type dict are not ordered"),
		s(`(4, 5)`):    newTruthAssertion(`Not true that <(4, 5)> is strictly ordered <(4, 5)>.`),
		s(`(5, 5, 3)`): newTruthAssertion(`Not true that <(5, 5, 3)> is strictly ordered <(5, 5)>.`),
	})
}
