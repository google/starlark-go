package truth

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

func isNotEqualTo(t *T, args ...starlark.Value) (starlark.Value, error) {
	other := args[0]
	ok, err := starlark.Compare(syntax.EQL, t.actual, other)
	if err != nil {
		return nil, err
	}
	if ok {
		return nil, t.failComparingValues("is not equal to", other, "")
	}
	return starlark.None, nil
}

func isEqualTo(t *T, args ...starlark.Value) (starlark.Value, error) {
	arg1 := args[0]
	switch actual := t.actual.(type) {
	case starlark.String:
		if other, ok := arg1.(starlark.String); ok {
			a := actual.GoString()
			o := other.GoString()
			// Use unified diff strategy when comparing multiline strings.
			if strings.Contains(a, "\n") && strings.Contains(o, "\n") {
				diff := difflib.ContextDiff{
					A:        difflib.SplitLines(o),
					B:        difflib.SplitLines(a),
					FromFile: "Expected",
					ToFile:   "Actual",
					Context:  3,
					Eol:      "\n",
				}
				pretty, err := difflib.GetContextDiffString(diff)
				if err != nil {
					return nil, err
				}
				if pretty == "" {
					return starlark.None, nil
				}
				msg := "is equal to expected, found diff:\n" + pretty
				return nil, t.failWithProposition(msg, "")
			}
		}
	case starlark.Iterable:
		if t.actual.Type() == arg1.Type() {
			switch other := arg1.(type) {
			case starlark.IterableMapping: // e.g. dict
				if _, err := containsExactlyItemsIn(t, other); err != nil {
					return nil, err
				}
				return starlark.None, nil
			case starlark.Indexable: // e.g. tuple, list
				if _, err := containsExactlyElementsIn(t, other); err != nil {
					return nil, err
				}
				return inOrder(t)
			default: // e.g. set (any other Iterable)
				if _, err := containsExactlyElementsIn(t, other); err != nil {
					return nil, err
				}
				return starlark.None, nil
			}
		}
	default:
	}
	ok, err := starlark.Compare(syntax.NEQ, t.actual, arg1)
	if err != nil {
		return nil, err
	}
	if ok {
		suffix := ""
		if t.actual.String() == arg1.String() {
			suffix = " However, their str() representations are equal."
		}
		return nil, t.failComparingValues("is equal to", arg1, suffix)
	}
	return starlark.None, nil
}

func iterEmpty(able starlark.Iterable) bool {
	iter := able.Iterate()
	defer iter.Done()
	var v starlark.Value
	return !iter.Next(&v)
}

func containsExactly(t *T, args ...starlark.Value) (starlark.Value, error) {
	argc := len(args)
	switch actual := t.actual.(type) {
	case starlark.IterableMapping:
		if argc == 0 {
			if iterEmpty(actual) {
				if t.forOrdering == nil {
					t.forOrdering = &forOrdering{}
				}
				return t, nil
			}
			return nil, t.failWithProposition("is empty", "")
		}
		if argc%2 != 0 {
			return nil, errMustBeEqualNumberOfKVPairs(argc)
		}
		dic := starlark.NewDict(argc / 2)
		for i := 0; i < argc/2; i += 2 {
			if err := dic.SetKey(args[i], args[i+1]); err != nil {
				return nil, err
			}
		}
		return containsExactlyItemsIn(t, dic)
	case starlark.Iterable:
		if argc == 0 {
			if iterEmpty(actual) {
				if t.forOrdering == nil {
					t.forOrdering = &forOrdering{}
				}
				return t, nil
			}
			return nil, t.failWithProposition("is empty", "")
		}
		tup := starlark.Tuple(args)
		expectingSingleIterable := false
		if argc == 1 {
			_, isIterable := args[0].(starlark.Iterable)
			_, isString := args[0].(starlark.String)
			expectingSingleIterable = isIterable && !isString
		}
		return t.containsExactlyElementsIn(tup, expectingSingleIterable)
	case starlark.String:
		t.turnActualIntoIterableFromString()
		return containsExactly(t, args...)
	default:
		return nil, errUnhandled
	}
}

func inOrder(t *T, args ...starlark.Value) (starlark.Value, error) {
	if t.forOrdering == nil {
		return nil, errUnhandled
	}
	if err := t.forOrdering.inOrderError; err != nil {
		return nil, err
	}
	return starlark.None, nil
}

var errDictOrdering = newInvalidAssertion("values of type dict are not ordered")

func containsExactlyItemsIn(t *T, args ...starlark.Value) (starlark.Value, error) {
	arg1 := args[0] // TODO: what when passed **kwargs?
	if imActual, ok := t.actual.(starlark.IterableMapping); ok {
		if imExpected, ok := arg1.(starlark.IterableMapping); ok {
			tt := newT(newTupleSlice(imActual.Items()))
			tt.forOrdering = &forOrdering{inOrderError: errDictOrdering}
			return containsExactly(tt, newTupleSlice(imExpected.Items()).Values()...)
		}
	}
	return nil, errUnhandled
}

func containsExactlyElementsIn(t *T, args ...starlark.Value) (starlark.Value, error) {
	return t.containsExactlyElementsIn(args[0], false)
}

// Determines if the subject contains exactly the expected elements.
func (t *T) containsExactlyElementsIn(expected starlark.Value, warnElementsIn bool) (starlark.Value, error) {
	iterableActual, err := t.failIterable()
	if err != nil {
		return nil, err
	}
	iterableExpected, err := newT(expected).failIterable()
	if err != nil {
		return nil, err
	}

	missing := newDuplicateCounter()
	extra := newDuplicateCounter()
	iterActual := iterableActual.Iterate()
	defer iterActual.Done()
	iterExpected := iterableExpected.Iterate()
	defer iterExpected.Done()

	warning := ""
	if warnElementsIn {
		warning = warnContainsExactlySingleIterable
	}

	forOrderingSet := t.forOrdering != nil

	var elemActual, elemExpected starlark.Value
	iterations := 0
	for {
		// Step through both iterators comparing elements pairwise.
		if !iterActual.Next(&elemActual) {
			break
		}
		if !iterExpected.Next(&elemExpected) {
			extra.Increment(elemActual)
			break
		}
		iterations++

		// As soon as we encounter a pair of elements that differ, we know that
		// in_order cannot succeed, so we can check the rest of the elements
		// more normally. Since any previous pairs of elements we iterated
		// over were equal, they have no effect on the result now.
		ok, err := starlark.Compare(syntax.NEQ, elemActual, elemExpected)
		if err != nil {
			return nil, err
		}
		if ok {
			// Missing elements; elements that are not missing will be removed.
			missing.Increment(elemExpected)
			var m starlark.Value
			for iterExpected.Next(&m) {
				missing.Increment(m)
			}

			eStrActual := elemActual.String()
			// Remove all actual elements from missing, and add any that weren't
			// in missing to extra.
			if missing.contains(eStrActual) {
				missing.decrement(eStrActual)
			} else {
				extra.increment(eStrActual)
			}
			var e starlark.Value
			for iterActual.Next(&e) {
				eStr := e.String()
				if missing.contains(eStr) {
					missing.decrement(eStr)
				} else {
					extra.increment(eStr)
				}
			}

			// Fail if there are either missing or extra elements.

			if !missing.Empty() {
				if !extra.Empty() {
					// Subject is missing required elements and has extra elements.
					msg := fmt.Sprintf("contains exactly <%s>."+
						" It is missing <%s> and has unexpected items <%s>",
						expected.String(), missing.String(), extra.String())
					return nil, t.failWithProposition(msg, warning)
				}
				return nil, t.failWithBadResults("contains exactly", expected,
					"is missing", missing, warning)
			}

			if !extra.Empty() {
				return nil, t.failWithBadResults("contains exactly", expected,
					"has unexpected items", extra, warning)
			}

			// The iterables were not in the same order.
			if !forOrderingSet {
				forOrderingSet = true
				t.forOrdering = &forOrdering{
					inOrderError: t.failComparingValues(
						"contains exactly these elements in order", expected, ""),
				}
			}
		}
	}
	if iterations == 0 && missing.Empty() && !extra.Empty() {
		return nil, t.failWithProposition("is empty", "")
	}

	// We must have reached the end of one of the iterators without finding any
	// pairs of elements that differ. If the actual iterator still has elements,
	// they're extras. If the required iterator has elements, they're missing.
	var e starlark.Value
	for iterActual.Next(&e) {
		extra.Increment(e)
	}
	if !extra.Empty() {
		return nil, t.failWithBadResults("contains exactly", expected,
			"has unexpected items", extra, warning)
	}

	var m starlark.Value
	for iterExpected.Next(&m) {
		missing.Increment(m)
	}
	if !missing.Empty() {
		return nil, t.failWithBadResults("contains exactly", expected,
			"is missing", missing, warning)
	}

	if !forOrderingSet {
		t.forOrdering = &forOrdering{}
	}

	// If neither iterator has elements, we reached the end and the elements
	// were in order.
	return t, nil
}

// Adds a prefix to the subject, when it is displayed in error messages.
//
// This is especially useful in the context of types that have no helpful
// string representation (e.g., bool). Writing `assert.that(foo).named("foo").is_true()`
// then results in a more reasonable error.
func named(t *T, args ...starlark.Value) (starlark.Value, error) {
	str, ok := args[0].(starlark.String)
	if !ok || str.Len() == 0 {
		return nil, errors.New(".named() expects a (non empty) string")
	}
	t.name = str.GoString()
	return t, nil
}

func isNone(t *T, args ...starlark.Value) (starlark.Value, error) {
	if t.actual != starlark.None {
		return nil, t.failWithProposition("is None", "")
	}
	return starlark.None, nil
}

func isNotNone(t *T, args ...starlark.Value) (starlark.Value, error) {
	if t.actual == starlark.None {
		return nil, t.failWithProposition("is not None", "")
	}
	return starlark.None, nil
}

func isFalse(t *T, args ...starlark.Value) (starlark.Value, error) {
	if b, ok := t.actual.(starlark.Bool); ok && b == starlark.False {
		return starlark.None, nil
	}
	suffix := ""
	if !t.actual.Truth() {
		suffix = " However, it is falsy. Did you mean to call .is_falsy() instead?"
	}
	return nil, t.failWithProposition("is False", suffix)
}

func isFalsy(t *T, args ...starlark.Value) (starlark.Value, error) {
	if t.actual.Truth() {
		return nil, t.failWithProposition("is falsy", "")
	}
	return starlark.None, nil
}

func isTrue(t *T, args ...starlark.Value) (starlark.Value, error) {
	if b, ok := t.actual.(starlark.Bool); ok && b == starlark.True {
		return starlark.None, nil
	}
	suffix := ""
	if t.actual.Truth() {
		suffix = " However, it is truthy. Did you mean to call .is_truthy() instead?"
	}
	return nil, t.failWithProposition("is True", suffix)
}

func isTruthy(t *T, args ...starlark.Value) (starlark.Value, error) {
	if !t.actual.Truth() {
		return nil, t.failWithProposition("is truthy", "")
	}
	return starlark.None, nil
}

func (t *T) comparable(bName, verb string, op syntax.Token, other starlark.Value) (starlark.Value, error) {
	if err := t.failNone(bName, other); err != nil {
		return nil, err
	}
	ok, err := starlark.Compare(op, t.actual, other)
	if err != nil {
		return nil, err
	}
	if ok {
		return nil, t.failComparingValues(verb, other, "")
	}
	return starlark.None, nil
}

func isAtLeast(t *T, args ...starlark.Value) (starlark.Value, error) {
	return t.comparable("is_at_least", "is at least", syntax.LT, args[0])
}

func isAtMost(t *T, args ...starlark.Value) (starlark.Value, error) {
	return t.comparable("is_at_most", "is at most", syntax.GT, args[0])
}

func isGreaterThan(t *T, args ...starlark.Value) (starlark.Value, error) {
	return t.comparable("is_greater_than", "is greater than", syntax.LE, args[0])
}

func isLessThan(t *T, args ...starlark.Value) (starlark.Value, error) {
	return t.comparable("is_less_than", "is less than", syntax.GE, args[0])
}

func isIn(t *T, args ...starlark.Value) (starlark.Value, error) {
	switch iterable := args[0].(type) {
	case starlark.Iterable:
		// NOTE: operates on Dict keys.
		it := iterable.Iterate()
		defer it.Done()
		var e starlark.Value
		for it.Next(&e) {
			ok, err := starlark.Compare(syntax.EQL, t.actual, e)
			if err != nil {
				return nil, err
			}
			if ok {
				return starlark.None, nil
			}
		}
		return nil, t.failComparingValues("is equal to any of", iterable, "")
	case starlark.String:
		if actual, ok := t.actual.(starlark.String); ok {
			if strings.Contains(iterable.GoString(), actual.GoString()) {
				return starlark.None, nil
			}
			return nil, t.failComparingValues("is equal to any of", iterable, "")
		}
	default:
	}
	return nil, errUnhandled
}

// Asserts that this subject is not a member of the given iterable.
func isNotIn(t *T, args ...starlark.Value) (starlark.Value, error) {
	arg1 := args[0]
	switch iterable := arg1.(type) {
	case starlark.String:
		if actual, ok := t.actual.(starlark.String); ok {
			if ix := strings.Index(iterable.GoString(), actual.GoString()); ix != -1 {
				msg := fmt.Sprintf("is not in %s. It was found at index %d",
					arg1.String(), ix)
				return nil, t.failWithProposition(msg, "")
			}
			return starlark.None, nil
		}
	case starlark.Indexable:
		for ix := 0; ix < iterable.Len(); ix++ {
			e := iterable.Index(ix)
			ok, err := starlark.Compare(syntax.EQL, t.actual, e)
			if err != nil {
				return nil, err
			}
			if ok {
				msg := fmt.Sprintf("is not in %s. It was found at index %d",
					arg1.String(), ix)
				return nil, t.failWithProposition(msg, "")
			}
		}
		return starlark.None, nil
	case starlark.Iterable:
		// NOTE: operates on Dict keys.
		it := iterable.Iterate()
		defer it.Done()
		var e starlark.Value
		for it.Next(&e) {
			ok, err := starlark.Compare(syntax.EQL, t.actual, e)
			if err != nil {
				return nil, err
			}
			if ok {
				msg := fmt.Sprintf("is not in %s", arg1.String())
				return nil, t.failWithProposition(msg, "")
			}
		}
		return starlark.None, nil
	default:
	}
	return nil, errUnhandled
}

func isAnyOf(t *T, args ...starlark.Value) (starlark.Value, error) {
	return isIn(t, starlark.Tuple(args))
}

func isNoneOf(t *T, args ...starlark.Value) (starlark.Value, error) {
	return isNotIn(t, starlark.Tuple(args))
}

// From https://github.com/google/starlark-go/blob/6677ee5c7211380ec7e6a1b50dc45287e40ca9e1/starlark/library.go#L383
func (t *T) hasattr(name string) bool {
	if o, ok := t.actual.(starlark.HasAttrs); ok {
		v, err := o.Attr(name)
		if err == nil {
			return (v != nil)
		}
		for _, x := range o.AttrNames() {
			if x == name {
				return true
			}
		}
	}
	return false
}

func hasAttribute(t *T, args ...starlark.Value) (starlark.Value, error) {
	if arg1, ok := args[0].(starlark.String); ok {
		attr := arg1.GoString()
		if !t.hasattr(attr) {
			return nil, t.failComparingValues("has attribute", arg1, "")
		}
		return starlark.None, nil
	}
	return nil, errUnhandled
}

func doesNotHaveAttribute(t *T, args ...starlark.Value) (starlark.Value, error) {
	if arg1, ok := args[0].(starlark.String); ok {
		attr := arg1.GoString()
		if t.hasattr(attr) {
			return nil, t.failComparingValues("does not have attribute", arg1, "")
		}
		return starlark.None, nil
	}
	return nil, errUnhandled
}

func isCallable(t *T, args ...starlark.Value) (starlark.Value, error) {
	if _, ok := t.actual.(starlark.Callable); !ok {
		return nil, t.failWithProposition("is callable", "")
	}
	return starlark.None, nil
}

func isNotCallable(t *T, args ...starlark.Value) (starlark.Value, error) {
	if _, ok := t.actual.(starlark.Callable); ok {
		return nil, t.failWithProposition("is not callable", "")
	}
	return starlark.None, nil
}

func sizeOf(v starlark.Value) int {
	switch v := v.(type) {
	case starlark.Indexable:
		return v.Len()
	case starlark.Sequence:
		return v.Len()
	default:
		return -1
	}
}

type int64Stringer int64

var _ fmt.Stringer = (int64Stringer)(0)

func (i int64Stringer) String() string { return fmt.Sprintf("%d", i) }

func hasSize(t *T, args ...starlark.Value) (starlark.Value, error) {
	switch arg1 := args[0].(type) {
	case starlark.Int:
		size, ok := arg1.Int64()
		if ok {
			if actualSize := sizeOf(t.actual); actualSize != -1 {
				if int64(actualSize) != size {
					x := int64Stringer(actualSize)
					return nil, t.failWithBadResults("has a size of", arg1, "is", x, "")
				}
				return starlark.None, nil
			}
		}
	default:
	}
	return nil, errUnhandled
}

func isEmpty(t *T, args ...starlark.Value) (starlark.Value, error) {
	switch iterable := t.actual.(type) {
	case starlark.Iterable:
		if !iterEmpty(iterable) {
			return nil, t.failWithProposition("is empty", "")
		}
		return starlark.None, nil
	case starlark.String:
		if iterable.Len() != 0 {
			return nil, t.failWithProposition("is empty", "")
		}
		return starlark.None, nil
	default:
		return nil, errUnhandled
	}
}

func isNotEmpty(t *T, args ...starlark.Value) (starlark.Value, error) {
	switch iterable := t.actual.(type) {
	case starlark.Iterable:
		if iterEmpty(iterable) {
			return nil, t.failWithProposition("is not empty", "")
		}
		return starlark.None, nil
	case starlark.String:
		if iterable.Len() == 0 {
			return nil, t.failWithProposition("is not empty", "")
		}
		return starlark.None, nil
	default:
		return nil, errUnhandled
	}
}

func contains(t *T, args ...starlark.Value) (starlark.Value, error) {
	arg1 := args[0]
	switch iterable := t.actual.(type) {
	case starlark.Iterable:
		// NOTE: operates on Dict keys.
		it := iterable.Iterate()
		defer it.Done()
		var e starlark.Value
		for it.Next(&e) {
			ok, err := starlark.Compare(syntax.EQL, e, arg1)
			if err != nil {
				return nil, err
			}
			if ok {
				return starlark.None, nil
			}
		}
		return nil, t.failWithSubject(fmt.Sprintf("should have contained %s", arg1))
	case starlark.String:
		if arg1, ok := arg1.(starlark.String); ok {
			if strings.Contains(iterable.GoString(), arg1.GoString()) {
				return starlark.None, nil
			}
			return nil, t.failWithSubject(fmt.Sprintf("should have contained %s", arg1))
		}
	default:
	}
	return nil, errUnhandled
}

func doesNotContain(t *T, args ...starlark.Value) (starlark.Value, error) {
	arg1 := args[0]
	switch iterable := t.actual.(type) {
	case starlark.Iterable:
		// NOTE: operates on Dict keys.
		it := iterable.Iterate()
		defer it.Done()
		var e starlark.Value
		for it.Next(&e) {
			ok, err := starlark.Compare(syntax.EQL, e, arg1)
			if err != nil {
				return nil, err
			}
			if ok {
				return nil, t.failWithSubject(fmt.Sprintf("should not have contained %s", arg1))
			}
		}
		return starlark.None, nil
	case starlark.String:
		if arg1, ok := arg1.(starlark.String); ok {
			if strings.Contains(iterable.GoString(), arg1.GoString()) {
				return nil, t.failWithSubject(fmt.Sprintf("should not have contained %s", arg1))
			}
			return starlark.None, nil
		}
	default:
	}
	return nil, errUnhandled
}

// Asserts that this subject contains no two elements that are the same.
func containsNoDuplicates(t *T, args ...starlark.Value) (starlark.Value, error) {
	counter := newDuplicateCounter()
	switch actual := t.actual.(type) {
	case starlark.IterableMapping, *starlark.Set:
		// Dictionaries and Sets have unique members by definition; avoid iterating.
		return starlark.None, nil
	case starlark.Iterable:
		it := actual.Iterate()
		defer it.Done()
		var e starlark.Value
		for it.Next(&e) {
			counter.Increment(e)
		}
	case starlark.String:
		for _, s := range actual.GoString() {
			counter.increment(fmt.Sprintf("%q", string(s)))
		}
	default:
		return nil, errUnhandled
	}
	if counter.HasDupes() {
		msg := fmt.Sprintf("has the following duplicates: <%s>", counter.Dupes())
		return nil, t.failWithSubject(msg)
	}
	return starlark.None, nil
}

func containsAllIn(t *T, args ...starlark.Value) (starlark.Value, error) {
	if arg1, ok := args[0].(starlark.Iterable); ok {
		return t.containsAll("contains all elements in", arg1)
	}
	return nil, errUnhandled
}

func containsAllOf(t *T, args ...starlark.Value) (starlark.Value, error) {
	return t.containsAll("contains all of", starlark.Tuple(args))
}

func (t *T) actualAsSlice() ([]starlark.Value, error) {
	switch actual := t.actual.(type) {
	case starlark.Iterable:
		return collect(actual), nil
	case starlark.String:
		t.turnActualIntoIterableFromString()
		return []starlark.Value(t.actual.(starlark.Tuple)), nil
	default:
		return nil, errUnhandled
	}
}

func collect(iterable starlark.Iterable) []starlark.Value {
	var xs []starlark.Value
	iter := iterable.Iterate()
	defer iter.Done()
	var x starlark.Value
	for iter.Next(&x) {
		xs = append(xs, x)
	}
	return xs
}

func indexOf(v starlark.Value, xs []starlark.Value) (int, error) {
	for i, x := range xs {
		ok, err := starlark.Compare(syntax.EQL, v, x)
		if err != nil {
			return -2, err
		}
		if ok {
			return i, nil
		}
	}
	return -1, nil
}

// Determines if the subject contains all the expected elements.
func (t *T) containsAll(verb string, expected starlark.Iterable) (starlark.Value, error) {
	actualSlice, err := t.actualAsSlice()
	if err != nil {
		return nil, err
	}
	missing := newDuplicateCounter()
	var actualNotInOrder []starlark.Value // = Tuple
	ordered := true

	iterExpected := expected.Iterate()
	defer iterExpected.Done()
	var i starlark.Value
	// Step through the expected elements.
	for iterExpected.Next(&i) {
		index, err := indexOf(i, actualSlice)
		if err != nil {
			return nil, err
		}
		if index != -1 {
			// Drain all the elements before that element into actualNotInOrder.
			actualNotInOrder = append(actualNotInOrder, actualSlice[0:index]...)
			// And remove the element from the actual_list.
			actualSlice = actualSlice[1:]
			continue
		}

		// The expected value was not in the actual list.
		if index, err = indexOf(i, actualNotInOrder); err != nil {
			return nil, err
		}
		if index != -1 {
			actualNotInOrder = append(actualNotInOrder[:index], actualNotInOrder[index+1:]...)
			// If it was in actualNotInOrder, we're not in order.
			ordered = false
		} else {
			// It is not in actualNotInOrder, we're missing an expected element.
			missing.Increment(i)
		}
	}

	// If we have any missing expected elements, fail.
	if !missing.Empty() {
		return nil, t.failWithBadResults(verb, expected, "is missing", missing, "")
	}

	t.forOrdering = &forOrdering{}
	if !ordered {
		t.forOrdering.inOrderError = t.failComparingValues("contains all elements in order", expected, "")
	}
	return t, nil
}

func containsAnyIn(t *T, args ...starlark.Value) (starlark.Value, error) {
	if arg1, ok := args[0].(starlark.Iterable); ok {
		return t.containsAny("contains any element in", arg1)
	}
	return nil, errUnhandled
}

func containsAnyOf(t *T, args ...starlark.Value) (starlark.Value, error) {
	return t.containsAny("contains any of", starlark.Tuple(args))
}

// Determines if the subject contains any of the expected elements.
func (t *T) containsAny(verb string, expected starlark.Iterable) (starlark.Value, error) {
	actualSlice, err := t.actualAsSlice()
	if err != nil {
		return nil, err
	}

	iterExpected := expected.Iterate()
	defer iterExpected.Done()
	var i starlark.Value
	for iterExpected.Next(&i) {
		index, err := indexOf(i, actualSlice)
		if err != nil {
			return nil, err
		}
		if index != -1 {
			return starlark.None, nil
		}
	}
	return nil, t.failComparingValues(verb, expected, "")
}

func containsNoneIn(t *T, args ...starlark.Value) (starlark.Value, error) {
	if arg1, ok := args[0].(starlark.Iterable); ok {
		return t.containsNone("contains no elements in", arg1)
	}
	return nil, errUnhandled
}

func containsNoneOf(t *T, args ...starlark.Value) (starlark.Value, error) {
	return t.containsNone("contains none of", starlark.Tuple(args))
}

// Determines if the subject contains none of the excluded elements.
func (t *T) containsNone(failVerb string, excluded starlark.Iterable) (starlark.Value, error) {
	actualSlice, err := t.actualAsSlice()
	if err != nil {
		return nil, err
	}

	iterExcluded := excluded.Iterate()
	defer iterExcluded.Done()
	var i starlark.Value
	present := newDuplicateCounter()

	for iterExcluded.Next(&i) {
		index, err := indexOf(i, actualSlice)
		if err != nil {
			return nil, err
		}
		if index != -1 {
			present.Increment(i)
		}
	}
	if !present.Empty() {
		return nil, t.failWithBadResults(failVerb, excluded, "contains", present, "")
	}
	return starlark.None, nil
}

func isOrdered(t *T, args ...starlark.Value) (starlark.Value, error) {
	return isOrderedAccordingTo(t, t.registered.Cmp)
}

func isOrderedAccordingTo(t *T, args ...starlark.Value) (starlark.Value, error) {
	if arg1, ok := args[0].(*starlark.Function); ok {
		return t.pairwiseCheck(arg1, false)
	}
	return nil, errUnhandled
}

func isStrictlyOrdered(t *T, args ...starlark.Value) (starlark.Value, error) {
	return isStrictlyOrderedAccordingTo(t, t.registered.Cmp)
}

func isStrictlyOrderedAccordingTo(t *T, args ...starlark.Value) (starlark.Value, error) {
	if arg1, ok := args[0].(*starlark.Function); ok {
		return t.pairwiseCheck(arg1, true)
	}
	return nil, errUnhandled
}

// Iterates over this subject and compares adjacent elements.
func (t *T) pairwiseCheck(pairComparator *starlark.Function, strict bool) (starlark.Value, error) {
	switch actual := t.actual.(type) {
	case starlark.IterableMapping:
		return nil, errDictOrdering
	case starlark.Iterable:
		return t.doPairwiseCheck(actual, pairComparator, strict)
	case starlark.String:
		t.turnActualIntoIterableFromString()
		return t.doPairwiseCheck(t.actual.(starlark.Iterable), pairComparator, strict)
	default:
		return nil, errUnhandled
	}
}

func (t *T) doPairwiseCheck(actual starlark.Iterable, pairComparator *starlark.Function, strict bool) (starlark.Value, error) {
	iterActual := actual.Iterate()
	defer iterActual.Done()

	var prev, current starlark.Value
	if iterActual.Next(&prev) {
		for {
			if !iterActual.Next(&current) {
				break
			}

			args := starlark.Tuple{prev, current}
			someInt, err := t.registered.Apply(pairComparator, args)
			if err != nil {
				return nil, err
			}
			r, err := starlark.AsInt32(someInt)
			if err != nil {
				return nil, err
			}
			switch {
			case r < 0:
			case r == 0 && !strict:
			default:
				msg := "is ordered"
				if strict {
					msg = "is strictly ordered"
				}
				return nil, t.failComparingValues(msg, args, "")
			}
			prev = current
		}
	}
	return starlark.None, nil
}

func containsKey(t *T, args ...starlark.Value) (starlark.Value, error) {
	key := args[0]
	if actual, ok := t.actual.(starlark.Mapping); ok {
		_, found, err := actual.Get(key)
		if err != nil {
			return nil, err
		}
		if !found {
			msg := fmt.Sprintf("contains key <%s>", key.String())
			return nil, t.failWithProposition(msg, "")
		}
		return starlark.None, nil
	}
	return nil, errUnhandled
}

func doesNotContainKey(t *T, args ...starlark.Value) (starlark.Value, error) {
	key := args[0]
	if actual, ok := t.actual.(starlark.Mapping); ok {
		_, found, err := actual.Get(key)
		if err != nil {
			return nil, err
		}
		if found {
			msg := fmt.Sprintf("does not contain key <%s>", key.String())
			return nil, t.failWithProposition(msg, "")
		}
		return starlark.None, nil
	}
	return nil, errUnhandled
}

// Assertion that the subject contains the key mapping to the value.
func containsItem(t *T, args ...starlark.Value) (starlark.Value, error) {
	key, value := args[0], args[1]
	if actual, ok := t.actual.(starlark.Mapping); ok {
		val, found, err := actual.Get(key)
		if err != nil {
			return nil, err
		}
		if found {
			ok, err := starlark.Compare(syntax.EQL, value, val)
			if err != nil {
				return nil, err
			}
			if ok {
				return starlark.None, nil
			}
			msg := fmt.Sprintf(
				"contains item <%s>. However, it has a mapping from <%s> to <%s>",
				starlark.Tuple{key, value}.String(),
				key.String(),
				val.String(),
			)
			return nil, t.failWithProposition(msg, "")
		}

		if actual, ok := t.actual.(starlark.IterableMapping); ok {
			var otherKeys []starlark.Value
			for _, kv := range actual.Items() {
				if len(kv) != 2 {
					break
				}
				ok, err := starlark.Compare(syntax.EQL, value, kv[1])
				if err != nil {
					return nil, err
				}
				if ok {
					otherKeys = append(otherKeys, kv[0])
				}
			}
			if len(otherKeys) != 0 {
				msg := fmt.Sprintf(
					"contains item <%s>. However, the following keys are mapped to <%s>: %s",
					starlark.Tuple{key, value}.String(),
					value.String(),
					starlark.NewList(otherKeys).String(),
				)
				return nil, t.failWithProposition(msg, "")
			}
		}

		msg := fmt.Sprintf("contains item <%s>", starlark.Tuple{key, value}.String())
		return nil, t.failWithProposition(msg, "")
	}
	return nil, errUnhandled
}

func doesNotContainItem(t *T, args ...starlark.Value) (starlark.Value, error) {
	key, value := args[0], args[1]
	if actual, ok := t.actual.(starlark.Mapping); ok {
		val, found, err := actual.Get(key)
		if err != nil {
			return nil, err
		}
		if found {
			ok, err := starlark.Compare(syntax.EQL, value, val)
			if err != nil {
				return nil, err
			}
			if ok {
				msg := fmt.Sprintf(
					"does not contain item <%s>",
					starlark.Tuple{key, value}.String(),
				)
				return nil, t.failWithProposition(msg, "")
			}
		}
		return starlark.None, nil
	}
	return nil, errUnhandled
}

func hasLength(t *T, args ...starlark.Value) (starlark.Value, error) {
	if actual, ok := t.actual.(starlark.String); ok {
		expected, err := starlark.AsInt32(args[0])
		if err != nil {
			return nil, errUnhandled
		}
		actualLength := actual.Len()
		if actualLength != expected {
			msg := fmt.Sprintf("has a length of %d. It is %d", expected, actualLength)
			return nil, t.failWithProposition(msg, "")
		}
		return starlark.None, nil
	}
	return nil, errUnhandled
}

func startsWith(t *T, args ...starlark.Value) (starlark.Value, error) {
	if actual, ok := t.actual.(starlark.String); ok {
		if prefix, ok := args[0].(starlark.String); ok {
			if !strings.HasPrefix(actual.GoString(), prefix.GoString()) {
				return nil, t.failComparingValues("starts with", prefix, "")
			}
			return starlark.None, nil
		}
	}
	return nil, errUnhandled
}

func endsWith(t *T, args ...starlark.Value) (starlark.Value, error) {
	if actual, ok := t.actual.(starlark.String); ok {
		if suffix, ok := args[0].(starlark.String); ok {
			if !strings.HasSuffix(actual.GoString(), suffix.GoString()) {
				return nil, t.failComparingValues("ends with", suffix, "")
			}
			return starlark.None, nil
		}
	}
	return nil, errUnhandled
}

func newRegex(regex string) (*regexp.Regexp, error) {
	if strings.Contains(regex, "\\C") {
		// https://github.com/google/starlark-go/issues/241#issuecomment-529663462
		return nil, errors.New("unsupported regex class \\C")
	}
	return regexp.Compile(regex)
}

func matches(t *T, args ...starlark.Value) (starlark.Value, error) {
	if actual, ok := t.actual.(starlark.String); ok {
		if regex, ok := args[0].(starlark.String); ok {
			r, err := newRegex("^" + regex.GoString())
			if err != nil {
				return nil, err
			}
			if !r.MatchString(actual.GoString()) {
				msg := fmt.Sprintf("matches <%s>", regex)
				return nil, t.failWithProposition(msg, "")
			}
			return starlark.None, nil
		}
	}
	return nil, errUnhandled
}

func doesNotMatch(t *T, args ...starlark.Value) (starlark.Value, error) {
	if actual, ok := t.actual.(starlark.String); ok {
		if regex, ok := args[0].(starlark.String); ok {
			r, err := newRegex("^" + regex.GoString())
			if err != nil {
				return nil, err
			}
			if r.MatchString(actual.GoString()) {
				msg := fmt.Sprintf("fails to match <%s>", regex)
				return nil, t.failWithProposition(msg, "")
			}
			return starlark.None, nil
		}
	}
	return nil, errUnhandled
}

func containsMatch(t *T, args ...starlark.Value) (starlark.Value, error) {
	if actual, ok := t.actual.(starlark.String); ok {
		if regex, ok := args[0].(starlark.String); ok {
			r, err := newRegex(regex.GoString())
			if err != nil {
				return nil, err
			}
			if !r.MatchString(actual.GoString()) {
				msg := fmt.Sprintf("should have contained a match for <%s>", regex)
				return nil, t.failWithProposition(msg, "")
			}
			return starlark.None, nil
		}
	}
	return nil, errUnhandled
}

func doesNotContainMatch(t *T, args ...starlark.Value) (starlark.Value, error) {
	if actual, ok := t.actual.(starlark.String); ok {
		if regex, ok := args[0].(starlark.String); ok {
			r, err := newRegex(regex.GoString())
			if err != nil {
				return nil, err
			}
			if r.MatchString(actual.GoString()) {
				msg := fmt.Sprintf("should not have contained a match for <%s>", regex)
				return nil, t.failWithProposition(msg, "")
			}
			return starlark.None, nil
		}
	}
	return nil, errUnhandled
}

func isOfType(t *T, args ...starlark.Value) (starlark.Value, error) {
	if expected, ok := args[0].(starlark.String); ok {
		actualType := t.actual.Type()
		if expected.GoString() == actualType {
			return starlark.None, nil
		}
		msg := fmt.Sprintf("is of type <%s>", expected)
		suffix := fmt.Sprintf(" However, it is of type <%q>", actualType)
		return nil, t.failWithProposition(msg, suffix)
	}
	return nil, errUnhandled
}

func isNotOfType(t *T, args ...starlark.Value) (starlark.Value, error) {
	if expected, ok := args[0].(starlark.String); ok {
		actualType := t.actual.Type()
		if expected.GoString() != actualType {
			return starlark.None, nil
		}
		msg := fmt.Sprintf("is not of type <%s>", expected)
		suffix := fmt.Sprintf(" However, it is of type <%q>", actualType)
		return nil, t.failWithProposition(msg, suffix)
	}
	return nil, errUnhandled
}

func isZero(t *T, args ...starlark.Value) (starlark.Value, error) {
	switch actual := t.actual.(type) {
	case starlark.Float:
		if actual != 0 {
			return nil, t.failWithProposition("is zero", "")
		}
		return starlark.None, nil
	case starlark.Int:
		if actual.Truth() {
			return nil, t.failWithProposition("is zero", "")
		}
		return starlark.None, nil
	default:
		return nil, errUnhandled
	}
}

func isNonZero(t *T, args ...starlark.Value) (starlark.Value, error) {
	switch actual := t.actual.(type) {
	case starlark.Float:
		if actual == 0 {
			return nil, t.failWithProposition("is non-zero", "")
		}
		return starlark.None, nil
	case starlark.Int:
		if !actual.Truth() {
			return nil, t.failWithProposition("is non-zero", "")
		}
		return starlark.None, nil
	default:
		return nil, errUnhandled
	}
}

func isFloatNotFinite(f float64) bool { return math.IsInf(f, 0) || math.IsNaN(f) }

func isFinite(t *T, args ...starlark.Value) (starlark.Value, error) {
	switch actual := t.actual.(type) {
	case starlark.Float:
		if isFloatNotFinite(float64(actual)) {
			return nil, t.failWithSubject("should have been finite")
		}
		return starlark.None, nil
	case starlark.Int:
		return starlark.None, nil
	default:
		return nil, errUnhandled
	}
}

func isNotFinite(t *T, args ...starlark.Value) (starlark.Value, error) {
	switch actual := t.actual.(type) {
	case starlark.Float:
		if !isFloatNotFinite(float64(actual)) {
			return nil, t.failWithSubject("should not have been finite")
		}
		return starlark.None, nil
	case starlark.Int:
		return nil, t.failWithSubject("should not have been finite")
	default:
		return nil, errUnhandled
	}
}

var (
	pInf = starlark.Float(math.Inf(+1))
	nInf = starlark.Float(math.Inf(-1))
)

func isPositiveInfinity(t *T, args ...starlark.Value) (starlark.Value, error) {
	return isEqualTo(t, pInf)
}

func isNegativeInfinity(t *T, args ...starlark.Value) (starlark.Value, error) {
	return isEqualTo(t, nInf)
}

func isNotPositiveInfinity(t *T, args ...starlark.Value) (starlark.Value, error) {
	return isNotEqualTo(t, pInf)
}

func isNotNegativeInfinity(t *T, args ...starlark.Value) (starlark.Value, error) {
	return isNotEqualTo(t, nInf)
}

var nan = starlark.Float(math.NaN())

func isNaN(t *T, args ...starlark.Value) (starlark.Value, error) {
	switch actual := t.actual.(type) {
	case starlark.Float:
		if !math.IsNaN(float64(actual)) {
			return nil, t.failComparingValues("is equal to", nan, "")
		}
		return starlark.None, nil
	case starlark.Int:
		return nil, t.failComparingValues("is equal to", nan, "")
	default:
		return nil, errUnhandled
	}
}

func isNotNaN(t *T, args ...starlark.Value) (starlark.Value, error) {
	switch actual := t.actual.(type) {
	case starlark.Float:
		if math.IsNaN(float64(actual)) {
			return nil, t.failWithSubject("should not have been <nan>")
		}
		return starlark.None, nil
	case starlark.Int:
		return starlark.None, nil
	default:
		return nil, errUnhandled
	}
}

func (t *T) setWithinTolerance(tolerance starlark.Value, within bool) (starlark.Value, error) {
	wt := withinTolerance{
		within:           within,
		toleranceAsValue: tolerance,
	}

	switch delta := tolerance.(type) {
	case starlark.Float:
		f := float64(delta)
		if math.IsNaN(f) {
			return nil, newInvalidAssertion("tolerance cannot be <nan>")
		}
		if f < 0 {
			return nil, newInvalidAssertion("tolerance cannot be negative")
		}
		if math.IsInf(f, +1) {
			return nil, newInvalidAssertion("tolerance cannot be positive infinity")
		}
		wt.tolerance = new(big.Rat).SetFloat64(f)
	case starlark.Int:
		if delta.Sign() < 0 {
			return nil, newInvalidAssertion("tolerance cannot be negative")
		}
		wt.tolerance = new(big.Rat).SetInt(delta.BigInt())
	default:
		return nil, errUnhandled
	}

	switch actual := t.actual.(type) {
	case starlark.Float:
		wt.actual = new(big.Rat).SetFloat64(float64(actual))
	case starlark.Int:
		wt.actual = new(big.Rat).SetInt(actual.BigInt())
	default:
		return nil, errUnhandled
	}

	if t.withinTolerance != nil {
		return nil, newInvalidAssertion("tolerance cannot be overwritten")
	}
	t.withinTolerance = &wt
	return t, nil
}

func isWithin(t *T, args ...starlark.Value) (starlark.Value, error) {
	return t.setWithinTolerance(args[0], true)
}

func isNotWithin(t *T, args ...starlark.Value) (starlark.Value, error) {
	return t.setWithinTolerance(args[0], false)
}

func (t *T) withinToleranceOf(expected *big.Rat, expectedAsValue starlark.Value) (starlark.Value, error) {
	if t.withinTolerance == nil {
		// .of() called, .is_within()/.is_not_within() not called
		return nil, errUnhandled
	}

	tolerablyEqual := false
	if expected != nil && t.withinTolerance.actual != nil {
		// tolerably_equal = abs(self._actual - expected) <= self._tolerance
		diff := new(big.Rat).Sub(t.withinTolerance.actual, expected)
		tolerablyEqual = new(big.Rat).Abs(diff).Cmp(t.withinTolerance.tolerance) < 1
	}
	// Otherwise, (*big.Rat).SetFloat64(float64) was given non-finite
	// in which case Cmp must fail (i.e tolerablyEqual=false)

	notWithin := ""
	if !t.withinTolerance.within {
		notWithin = "not "
	}
	if t.withinTolerance.within != tolerablyEqual {
		msg := fmt.Sprintf("and <%s> should %shave been within <%s> of each other",
			expectedAsValue, notWithin, t.withinTolerance.toleranceAsValue)
		return nil, t.failWithSubject(msg)
	}
	return starlark.None, nil
}

func of(t *T, args ...starlark.Value) (starlark.Value, error) {
	expected := args[0]
	switch x := expected.(type) {
	case starlark.Float:
		r := new(big.Rat).SetFloat64(float64(x))
		return t.withinToleranceOf(r, expected)
	case starlark.Int:
		r := new(big.Rat).SetInt(x.BigInt())
		return t.withinToleranceOf(r, expected)
	default:
		return nil, errUnhandled
	}
}
