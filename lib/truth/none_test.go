package truth

import (
	"fmt"
	"testing"
)

func TestCannotCompareToNone(t *testing.T) {
	p := "It is illegal to compare using ."
	testEach(t, map[string]error{
		`that(5).is_at_least(None)`:     newInvalidAssertion(p + "is_at_least(None)"),
		`that(5).is_at_most(None)`:      newInvalidAssertion(p + "is_at_most(None)"),
		`that(5).is_greater_than(None)`: newInvalidAssertion(p + "is_greater_than(None)"),
		`that(5).is_less_than(None)`:    newInvalidAssertion(p + "is_less_than(None)"),
	})
}

func TestNone(t *testing.T) {
	testEach(t, map[string]error{
		`that(None).is_none()`:      nil,
		`that(None).is_not_none()`:  fail(`None`, `is not None`),
		`that("abc").is_not_none()`: nil,
		`that("abc").is_none()`:     fail(abc, `is None`),
	})
}

func TestNoneSuccess(t *testing.T) {
	testEach(t, map[string]error{
		`that(None).is_none()`:                 nil,
		`that(None).is_falsy()`:                nil,
		`that(None).is_equal_to(None)`:         nil,
		`that(None).is_not_equal_to(0)`:        nil,
		`that(None).is_not_equal_to(0.0)`:      nil,
		`that(None).is_not_equal_to(False)`:    nil,
		`that(None).is_not_equal_to("")`:       nil,
		`that(None).is_not_equal_to(())`:       nil,
		`that(None).is_not_equal_to([])`:       nil,
		`that(None).is_not_equal_to({})`:       nil,
		`that(None).is_not_equal_to(set([]))`:  nil,
		`that(None).is_in((5, None, "six"))`:   nil,
		`that(None).is_not_in((5, "six"))`:     nil,
		`that(None).is_any_of(5, None, "six")`: nil,
		`that(None).is_none_of()`:              nil,
		`that(None).is_none_of(5, "six")`:      nil,
		`that(None).is_of_type(type(None))`:    nil,
		`that(None).is_not_of_type("int")`:     nil,
		`that(None).is_not_callable()`:         nil,
	})
}

func TestNoneFailure(t *testing.T) {
	testEach(t, map[string]error{
		`that(None).is_not_none()`:                   fail(`None`, `is not None`),
		`that(None).is_truthy()`:                     fail(`None`, `is truthy`),
		`that(None).is_equal_to(0)`:                  fail(`None`, `is equal to <0>`),
		`that(None).is_not_equal_to(None)`:           fail(`None`, `is not equal to <None>`),
		`that(None).is_in((5, "six"))`:               fail(`None`, `is equal to any of <(5, "six")>`),
		`that(None).is_not_in((5, None))`:            fail(`None`, `is not in (5, None). It was found at index 1`),
		`that(None).is_any_of(5, "six")`:             fail(`None`, `is equal to any of <(5, "six")>`),
		`that(None).is_none_of(5, None)`:             fail(`None`, `is not in (5, None). It was found at index 1`),
		`that(None).is_of_type("int")`:               fail(`None`, `is of type <"int">`, ` However, it is of type <"NoneType">`),
		`that(None).is_not_of_type(type(None))`:      fail(`None`, `is not of type <"NoneType">`, ` However, it is of type <"NoneType">`),
		`that(None).has_attribute("test_attribute")`: fail(`None`, `has attribute <"test_attribute">`),
		`that(None).is_callable()`:                   fail(`None`, `is callable`),

		`that(None).is_true()`:  fail(`None`, `is True`),
		`that(None).is_false()`: fail(`None`, `is False`, ` However, it is falsy. Did you mean to call .is_falsy() instead?`),
	})
}

func TestInvalidOperationOnNone(t *testing.T) {
	testEach(t, map[string]error{
		// Iterable subject
		`that(None).has_size(1)`:                               fmt.Errorf(`Invalid assertion .has_size(1) on value of type NoneType`),
		`that(None).is_empty()`:                                fmt.Errorf(`Invalid assertion .is_empty() on value of type NoneType`),
		`that(None).is_not_empty()`:                            fmt.Errorf(`Invalid assertion .is_not_empty() on value of type NoneType`),
		`that(None).contains(None)`:                            fmt.Errorf(`Invalid assertion .contains(None) on value of type NoneType`),
		`that(None).does_not_contain(5)`:                       fmt.Errorf(`Invalid assertion .does_not_contain(5) on value of type NoneType`),
		`that(None).contains_no_duplicates()`:                  fmt.Errorf(`Invalid assertion .contains_no_duplicates() on value of type NoneType`),
		`that(None).contains_all_in((None,))`:                  fmt.Errorf(`Invalid assertion .contains_all_in((None,)) on value of type NoneType`),
		`that(None).contains_all_of(None)`:                     fmt.Errorf(`Invalid assertion .contains_all_of(None) on value of type NoneType`),
		`that(None).contains_any_in((None,))`:                  fmt.Errorf(`Invalid assertion .contains_any_in((None,)) on value of type NoneType`),
		`that(None).contains_any_of(None)`:                     fmt.Errorf(`Invalid assertion .contains_any_of(None) on value of type NoneType`),
		`that(None).contains_exactly_elements_in([None])`:      newInvalidAssertion(`Cannot use <None> as Iterable.`),
		`that(None).contains_none_in((5,))`:                    fmt.Errorf(`Invalid assertion .contains_none_in((5,)) on value of type NoneType`),
		`that(None).contains_none_of(5)`:                       fmt.Errorf(`Invalid assertion .contains_none_of(5) on value of type NoneType`),
		`that(None).is_ordered()`:                              fmt.Errorf(`Invalid assertion .is_ordered() on value of type NoneType`),
		`that(None).is_ordered_according_to(someCmp)`:          fmt.Errorf(`Invalid assertion .is_ordered_according_to(<function lambda>) on value of type NoneType`),
		`that(None).is_strictly_ordered()`:                     fmt.Errorf(`Invalid assertion .is_strictly_ordered() on value of type NoneType`),
		`that(None).is_strictly_ordered_according_to(someCmp)`: fmt.Errorf(`Invalid assertion .is_strictly_ordered_according_to(<function lambda>) on value of type NoneType`),

		// Dictionary subject
		`that(None).contains_key("key")`:                         fmt.Errorf(`Invalid assertion .contains_key("key") on value of type NoneType`),
		`that(None).does_not_contain_key("key")`:                 fmt.Errorf(`Invalid assertion .does_not_contain_key("key") on value of type NoneType`),
		`that(None).contains_item("key", "value")`:               fmt.Errorf(`Invalid assertion .contains_item("key", "value") on value of type NoneType`),
		`that(None).does_not_contain_item("key", "value")`:       fmt.Errorf(`Invalid assertion .does_not_contain_item("key", "value") on value of type NoneType`),
		`that(None).contains_exactly("key", "value")`:            fmt.Errorf(`Invalid assertion .contains_exactly("key", "value") on value of type NoneType`),
		`that(None).contains_exactly_items_in({"key": "value"})`: fmt.Errorf(`Invalid assertion .contains_exactly_items_in({"key": "value"}) on value of type NoneType`),

		// String subject
		`that(None).has_length(0)`:              fmt.Errorf(`Invalid assertion .has_length(0) on value of type NoneType`),
		`that(None).starts_with("")`:            fmt.Errorf(`Invalid assertion .starts_with("") on value of type NoneType`),
		`that(None).ends_with("")`:              fmt.Errorf(`Invalid assertion .ends_with("") on value of type NoneType`),
		`that(None).matches("")`:                fmt.Errorf(`Invalid assertion .matches("") on value of type NoneType`),
		`that(None).does_not_match("")`:         fmt.Errorf(`Invalid assertion .does_not_match("") on value of type NoneType`),
		`that(None).contains_match("")`:         fmt.Errorf(`Invalid assertion .contains_match("") on value of type NoneType`),
		`that(None).does_not_contain_match("")`: fmt.Errorf(`Invalid assertion .does_not_contain_match("") on value of type NoneType`),
	})
}
