// Package truth defines builtins and methods to express
// test assertions within Starlark programs in the fashion of https://truth.dev
//
// This package is a Starlark port of PyTruth (2c3717ddad 2021-03-10) https://github.com/google/pytruth
//
// The Starlark:
//
//      assert.that(a).is_equal_to(b)
//      assert.that(c).named("my value").is_true()
//      assert.that(d).contains(a)
//      assert.that(d).contains_all_of(a, b).in_order()
//      assert.that(d).contains_any_of(a, b, c)
//
// is equivalent to the following Python:
//
//      from truth.truth import AssertThat
//      AssertThat(a).IsEqualTo(b)
//      AssertThat(c).Named("my value").IsTrue()
//      AssertThat(d).Contains(a)
//      AssertThat(d).ContainsAllOf(a, b).InOrder()
//      AssertThat(d).ContainsAnyOf(a, b, c)
//
// Often, tests assert a relationship between a value produced by the test
// (the "actual" value) and some reference value (the "expected" value). It is
// strongly recommended that the actual value is made the subject of the assertion.
// For example:
//
//      assert.that(actual).is_equal_to(expected)    # Recommended.
//      assert.that(expected).is_equal_to(actual)    # Not recommended.
//      assert.that(actual).is_in(expected_possibilities)     # Recommended.
//      assert.that(expected_possibilities).contains(actual)  # Not recommended.
//
// Some assertions
//
//      assert.that(a).is_equal_to(b)
//      assert.that(a).is_not_equal_to(b)
//      assert.that(a).is_truthy()
//      assert.that(a).is_falsy()
//      assert.that(a).is_true()
//      assert.that(a).is_false()
//      assert.that(a).is_none()
//      assert.that(a).is_not_none()
//      assert.that(a).is_in(b)
//      assert.that(a).is_any_of(b, c, d)
//      assert.that(a).is_not_in(b)
//      assert.that(a).is_none_of(b, c, d)
//      assert.that(a).is_of_type(type(b))
//      assert.that(a).is_not_of_type(type(b))
//      assert.that(a).has_attribute(b)
//      assert.that(a).does_not_have_attribute(b)
//      assert.that(a).is_callable()
//      assert.that(a).is_not_callable()
//      assert.that(a).is_less_than(b)
//      assert.that(a).is_greater_than(b)
//      assert.that(a).is_at_most(b)
//      assert.that(a).is_at_least(b)
//
// Truthiness
//   Predicates `.is_true()` and `.is_false()` match *only* `True` and `False`.
//   For `.is_truthy()` and `.is_falsy()`, `(starlark.Value).Truth() bool` is used.
//
//      assert.that(True).is_true()
//      assert.that(False).is_false()
//      assert.that(1).is_true()      # fails
//      assert.that(0).is_false()     # fails
//      assert.that(None).is_true()   # fails
//      assert.that(None).is_false()  # fails
//      assert.that(True).is_truthy()
//      assert.that(False).is_falsy()
//      assert.that(1).is_truthy()
//      assert.that(0).is_falsy()
//      assert.that(None).is_truthy() # fails
//      assert.that(None).is_falsy()
//
// Strings
//
//      assert.that("abc").has_length(3)
//      assert.that("abc").starts_with("a")
//      assert.that("abc").ends_with("c")
//      assert.that("abc").matches("a.+")                # prepends "^" to regexp
//      assert.that("abc").does_not_match("b.+")         # prepends "^" to regexp
//      assert.that("abc").contains_match("b.+")
//      assert.that("abc").does_not_contain_match("c.+")
//
// Numbers
//
//      assert.that(a).is_zero()
//      assert.that(a).is_non_zero()
//      assert.that(a).is_positive_infinity()
//      assert.that(a).is_not_positive_infinity()
//      assert.that(a).is_negative_infinity()
//      assert.that(a).is_not_negative_infinity()
//      assert.that(a).is_finite()
//      assert.that(a).is_not_finite()
//      assert.that(a).is_nan()
//      assert.that(a).is_not_nan()
//      assert.that(a).is_within(delta).of(b)
//      assert.that(a).is_not_within(delta).of(b)
//
// Lists, strings, and other iterables
//   `cmp(x,y)` should return negative if x < y, zero if x == y and positive if x > y.
//   *Ordered* means that the iterable's elements must increase (or decrease,
//  depending on `cmp`) from beginning to end. Adjacent elements are allowed to be equal.
//   *Strictly ordered* means that in addition, the elements must be unique
//  (i.e. monotonically increasing or decreasing).
//
//      assert.that(a).has_size(n)
//      assert.that(a).is_empty()
//      assert.that(a).is_not_empty()
//      assert.that(a).contains(b)
//      assert.that(a).does_not_contain(b)
//      assert.that(a).contains_all_of(b, c)
//      assert.that(a).contains_all_in([b, c])
//      assert.that(a).contains_any_of(b, c)
//      assert.that(a).contains_any_in([b, c])
//      assert.that(a).contains_exactly(b, c)
//      assert.that(sorted(a)).contains_exactly_elements_in(sorted(b)).in_order()
//      assert.that(a).contains_none_of(b, c)
//      assert.that(a).contains_none_in([b, c])
//      assert.that(a).contains_no_duplicates()
//      assert.that(a).is_ordered()
//      assert.that(a).is_ordered_according_to(cmp)
//      assert.that(a).is_strictly_ordered()
//      assert.that(a).is_strictly_ordered_according_to(cmp)
//
// Asserting order
//   By default, `.contains_all...` and `.contains_exactly...` do not enforce that the
//  order of the elements in the subject under test matches that of the expected
//  value. To do that, append `.in_order()` to the returned predicate.
//
//      assert.that([2, 4, 6]).contains_all_of(6, 2)
//      assert.that([2, 4, 6]).contains_all_of(6, 2).in_order()     # fails
//      assert.that([2, 4, 6]).contains_all_of(2, 6).in_order()
//      assert.that((1, 2, 3)).contains_all_in((1, 3)).in_order()
//      assert.that([2, 4, 6]).contains_exactly(2, 6, 4)
//      assert.that([2, 4, 6]).contains_exactly(2, 6, 4).in_order() # fails
//      assert.that([2, 4, 6]).contains_exactly(2, 4, 6).in_order()
//      assert.that((1, 2, 3)).contains_exactly_elements_in([1, 2, 3]).in_order()
//
//   When using `.in_order()`, ensure that both the subject under test and the expected
//  value have a defined order, otherwise the result is undefined.
//  For example, `assert.that(aList).contains_exactly_elements_in(aSet).in_order()`
//  may or may not succeed, depending on how the `set` implements ordering.
//  The builtin set datatype does not implement ordering.
//  These assertions *may or may not* succeed:
//
//      assert.that((1, 2, 3)).contains_all_in(set([1, 3])).in_order()
//      assert.that(set([3, 2, 1])).contains_exactly_elements_in((1, 2, 3)).in_order()
//      assert.that({1:2, 3:4}).contains_all_in((1, 3)).in_order()
//
// Dictionaries, in addition to the table above
//
//      assert.that(d).contains_key(k)
//      assert.that(d).does_not_contain_key(k)
//      assert.that(d).contains_item(k, v)
//      assert.that(d).does_not_contain_item(k, v)
//      assert.that(d).contains_exactly(k1, v1, k2, v2)
//      assert.that(d1).contains_exactly_items_in(d2)
//      assert.that(d1.items()).contains_all_in(d2.items())
//
//
// Notes (in no particular order):
//
//   `None` is not comparable (as in Python 3), so assertions involving `<` `>`
//  `<=` `>=` on `None` such as `assert.that(a).is_greater_than(None)`
//  fail with `InvalidAssertion` error.
//  It is recommended to first check the `None`-ness of values with `.is_none()`
//  or `.is_not_none()` before performing inequility assertions.
//
//   As in Python, `0`, `0.0` and `-0.0` all compare equal. The assertions
//  `.is_zero()` and `.is_non_zero()` are provided for semantic convenience.
//
//   Starlark strings are not iterable (unlike Python's) but are iterated on as
//  slices of utf-8 runes when needed in this implementation. This works:
//      assert.that("abcdefg").contains_all_of("a", "c", "e").in_order()
//      assert.that("abcdefg").is_strictly_ordered()
//
//   In `.contains...()` assertions a "duplicate values counter" is used that
//  relies on the `(starlark.Value).String() string` reprensentation of values
//  instead of `(starlark.Value).Hash() (uint32, error)` as some basic types
//  are not hashable (list, dict, set).
//  For this reason **it is recommended to only pass values of the main data types
//  built in to the interpreter to functions of this package:**
//  * NoneType
//  * bool
//  * int
//  * float
//  * string
//  * list
//  * tuple
//  * dict
//  * set
//  * function
//  * builtin_function_or_method
//
//   It is possible to incorrectly express assertions (see all but the last line in
//  this block):
//      assert.that(x)
//      assert.that(x).named(z)
//      assert.that(x).is_within(y)
//      assert.that(x).is_not_within(y)
//      assert.that(x).is_not_within(y).of(z)
//   This is why each call to `.that(...)` first checks that no non-terminated
//  assertions were previously executed in the current thread.
//   A `Close(*starlark.Thread) error` function is also provided to ensure
//  this property holds after the interpreter returns.
//
//   This library is threadsafe; you may execute multiple assertions in parallel.
//
package truth // import "go.starlark.net/lib/truth"
