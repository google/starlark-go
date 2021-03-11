package truth

import "testing"

func TestIsAnyOf(t *testing.T) {
	s := func(x string) string {
		return `that(3).is_any_of(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`3`):       nil,
		s(`3, 5`):    nil,
		s(`1, 3, 5`): nil,
		s(``):        fail(`3`, `is equal to any of <()>`),
		s(`2`):       fail(`3`, `is equal to any of <(2,)>`),
	})
}

func TestIsNoneOf(t *testing.T) {
	s := func(x string) string {
		return `that(3).is_none_of(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`5`):    nil,
		s(`"3"`):  nil,
		s(`3`):    fail(`3`, `is not in (3,). It was found at index 0`),
		s(`1, 3`): fail(`3`, `is not in (1, 3). It was found at index 1`),
	})
}
