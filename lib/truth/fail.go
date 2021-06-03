package truth

import (
	"fmt"
	"strings"

	"go.starlark.net/starlark"
)

const warnContainsExactlySingleIterable = "" +
	" Passing a single iterable to .contains_exactly(*expected) is often" +
	" not the correct thing to do. Did you mean to call" +
	" .contains_exactly_elements_in(some_iterable) instead?"

func errMustBeEqualNumberOfKVPairs(count int) error {
	return newInvalidAssertion(
		fmt.Sprintf("There must be an equal number of key/value pairs"+
			" (i.e., the number of key/value parameters (%d) must be even).", count))
}

func (t *T) failNone(check string, other starlark.Value) error {
	if other == starlark.None {
		msg := fmt.Sprintf("It is illegal to compare using .%s(None)", check)
		return newInvalidAssertion(msg)
	}
	return nil
}

func (t *T) failIterable() (starlark.Iterable, error) {
	itermap, ok := t.actual.(starlark.IterableMapping)
	if ok {
		iter := newTupleSlice(itermap.Items())
		return iter, nil
	}

	iter, ok := t.actual.(starlark.Iterable)
	if !ok {
		msg := fmt.Sprintf("Cannot use %s as Iterable.", t.subject())
		return nil, newInvalidAssertion(msg)
	}
	return iter, nil
}

func (t *T) failComparingValues(verb string, other starlark.Value, suffix string) error {
	proposition := fmt.Sprintf("%s <%s>", verb, other.String())
	return t.failWithProposition(proposition, suffix)
}

func (t *T) failWithProposition(proposition, suffix string) error {
	msg := fmt.Sprintf("Not true that %s %s.%s", t.subject(), proposition, suffix)
	return newTruthAssertion(msg)
}

func (t *T) failWithBadResults(
	verb string, other starlark.Value,
	failVerb string, actual fmt.Stringer,
	suffix string,
) error {
	msg := fmt.Sprintf("%s <%s>. It %s <%s>",
		verb, other.String(),
		failVerb, actual.String())
	return t.failWithProposition(msg, suffix)
}

func (t *T) failWithSubject(verb string) error {
	msg := fmt.Sprintf("%s %s.", t.subject(), verb)
	return newTruthAssertion(msg)
}

func (t *T) subject() string {
	str := ""
	switch actual := t.actual.(type) {
	case starlark.String:
		if strings.Contains(actual.GoString(), "\n") {
			if t.name == "" {
				return "actual"
			}
			return fmt.Sprintf("actual %s", t.name)
		}
	case starlark.Callable:
		str = t.actual.String()
	case starlark.Tuple:
		if t.actualIsIterableFromString {
			var b strings.Builder
			b.WriteString(`<"`)
			for _, v := range actual {
				b.WriteString(v.(starlark.String).GoString())
			}
			b.WriteString(`">`)
			str = b.String()
		}
	default:
	}
	if str == "" {
		str = fmt.Sprintf("<%s>", t.actual.String())
	}
	if t.name == "" {
		return str
	}
	return fmt.Sprintf("%s(%s)", t.name, str)
}
