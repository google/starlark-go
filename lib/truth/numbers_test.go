package truth

import (
	"fmt"
	"testing"
)

func TestZero(t *testing.T) {
	zeros := []string{`0`, `0.0`, `-0.0`}
	m := make(map[string]error, 42+3*2)

	for i := 0; i < len(zeros); i++ {
		m[fmt.Sprintf(`that(%s).is_zero()`, zeros[i])] = nil
		m[fmt.Sprintf(`that(%s).is_non_zero()`, zeros[i])] =
			fail(zeros[i], `is non-zero`)

		for j := len(zeros) - 1; j >= 0; j-- {
			m[fmt.Sprintf(`that(%s).is_equal_to(%s)`, zeros[i], zeros[j])] = nil
			m[fmt.Sprintf(`that(%s).is_not_equal_to(%s)`, zeros[i], zeros[j])] =
				fail(zeros[i], fmt.Sprintf(`is not equal to <%s>`, zeros[j]))
		}
	}

	for _, zero := range zeros {
		m[fmt.Sprintf(`that(%s).is_finite()`, zero)] = nil
		m[fmt.Sprintf(`that(%s).is_not_finite()`, zero)] =
			newTruthAssertion(fmt.Sprintf(`<%s> should not have been finite.`, zero))

		m[fmt.Sprintf(`that(%s).is_not_nan()`, zero)] = nil
		m[fmt.Sprintf(`that(%s).is_nan()`, zero)] =
			fail(zero, `is equal to <nan>`)

		m[fmt.Sprintf(`that(%s).is_not_positive_infinity()`, zero)] = nil
		m[fmt.Sprintf(`that(%s).is_positive_infinity()`, zero)] =
			fail(zero, fmt.Sprintf(`is equal to <+inf>`))

		m[fmt.Sprintf(`that(%s).is_not_negative_infinity()`, zero)] = nil
		m[fmt.Sprintf(`that(%s).is_negative_infinity()`, zero)] =
			fail(zero, fmt.Sprintf(`is equal to <-inf>`))
	}

	testEach(t, m)
}

func TestNumericEdges(t *testing.T) {
	testEach(t, map[string]error{
		`that(9).is_zero()`:                  fail(`9`, `is zero`),
		`that(9).is_non_zero()`:              nil,
		`that(9).is_finite()`:                nil,
		`that(9).is_not_finite()`:            newTruthAssertion(`<9> should not have been finite.`),
		`that(9).is_not_nan()`:               nil,
		`that(9).is_nan()`:                   fail(`9`, `is equal to <nan>`),
		`that(9).is_not_positive_infinity()`: nil,
		`that(9).is_positive_infinity()`:     fail(`9`, `is equal to <+inf>`),
		`that(9).is_not_negative_infinity()`: nil,
		`that(9).is_negative_infinity()`:     fail(`9`, `is equal to <-inf>`),

		`that(float("+inf")).is_zero()`:                  fail(`+inf`, `is zero`),
		`that(float("+inf")).is_non_zero()`:              nil,
		`that(float("+inf")).is_finite()`:                newTruthAssertion(`<+inf> should have been finite.`),
		`that(float("+inf")).is_not_finite()`:            nil,
		`that(float("+inf")).is_not_nan()`:               nil,
		`that(float("+inf")).is_nan()`:                   fail(`+inf`, `is equal to <nan>`),
		`that(float("+inf")).is_not_positive_infinity()`: fail(`+inf`, `is not equal to <+inf>`),
		`that(float("+inf")).is_positive_infinity()`:     nil,
		`that(float("+inf")).is_not_negative_infinity()`: nil,
		`that(float("+inf")).is_negative_infinity()`:     fail(`+inf`, `is equal to <-inf>`),

		`that(float("-inf")).is_zero()`:                  fail(`-inf`, `is zero`),
		`that(float("-inf")).is_non_zero()`:              nil,
		`that(float("-inf")).is_finite()`:                newTruthAssertion(`<-inf> should have been finite.`),
		`that(float("-inf")).is_not_finite()`:            nil,
		`that(float("-inf")).is_not_nan()`:               nil,
		`that(float("-inf")).is_nan()`:                   fail(`-inf`, `is equal to <nan>`),
		`that(float("-inf")).is_not_positive_infinity()`: nil,
		`that(float("-inf")).is_positive_infinity()`:     fail(`-inf`, `is equal to <+inf>`),
		`that(float("-inf")).is_not_negative_infinity()`: fail(`-inf`, `is not equal to <-inf>`),
		`that(float("-inf")).is_negative_infinity()`:     nil,

		`that(float("nan")).is_zero()`:                  fail(`nan`, `is zero`),
		`that(float("nan")).is_non_zero()`:              nil,
		`that(float("nan")).is_finite()`:                newTruthAssertion(`<nan> should have been finite.`),
		`that(float("nan")).is_not_finite()`:            nil,
		`that(float("nan")).is_not_nan()`:               newTruthAssertion(`<nan> should not have been <nan>.`),
		`that(float("nan")).is_nan()`:                   nil,
		`that(float("nan")).is_not_positive_infinity()`: nil,
		`that(float("nan")).is_positive_infinity()`:     fail(`nan`, `is equal to <+inf>`),
		`that(float("nan")).is_not_negative_infinity()`: nil,
		`that(float("nan")).is_negative_infinity()`:     fail(`nan`, `is equal to <-inf>`),
	})
}

func TestWithin(t *testing.T) {
	testEach(t, map[string]error{
		`
fortytwo = that(5.0).is_within(0.1)
fortytwo.of(4.9)
fortytwo.of(5.0)
fortytwo.of(5.1)
`: nil,

		`that(5.0).is_within(0.1).of(float("-inf"))`: newTruthAssertion(`<5.0> and <-inf> should have been within <0.1> of each other.`),
		`that(5.0).is_within(0.1).of(4.8)`:           newTruthAssertion(`<5.0> and <4.8> should have been within <0.1> of each other.`),
		`that(5.0).is_within(0.1).of(5.2)`:           newTruthAssertion(`<5.0> and <5.2> should have been within <0.1> of each other.`),
		`that(5.0).is_within(0.1).of(float("+inf"))`: newTruthAssertion(`<5.0> and <+inf> should have been within <0.1> of each other.`),
	})
}

func TestNotWithin(t *testing.T) {
	testEach(t, map[string]error{
		`
fortytwo = that(5.0).is_not_within(0.1)
fortytwo.of(float("-inf"))
fortytwo.of(4.8)
fortytwo.of(5.2)
fortytwo.of(float("+inf"))
`: nil,

		`that(5.0).is_not_within(0.1).of(4.9)`: newTruthAssertion(`<5.0> and <4.9> should not have been within <0.1> of each other.`),
		`that(5.0).is_not_within(0.1).of(5.0)`: newTruthAssertion(`<5.0> and <5.0> should not have been within <0.1> of each other.`),
		`that(5.0).is_not_within(0.1).of(5.1)`: newTruthAssertion(`<5.0> and <5.1> should not have been within <0.1> of each other.`),
	})
}

func TestWithinTolerance(t *testing.T) {
	testEach(t, map[string]error{
		`that(0).is_within(-1).of(42)`:            newInvalidAssertion(`tolerance cannot be negative`),
		`that(0).is_within(float("+inf")).of(42)`: newInvalidAssertion(`tolerance cannot be positive infinity`),
		`that(0).is_within(float("-inf")).of(42)`: newInvalidAssertion(`tolerance cannot be negative`),
		`that(0).is_within(float("nan")).of(42)`:  newInvalidAssertion(`tolerance cannot be <nan>`),

		`that(0).is_not_within(-1).of(42)`:            newInvalidAssertion(`tolerance cannot be negative`),
		`that(0).is_not_within(float("+inf")).of(42)`: newInvalidAssertion(`tolerance cannot be positive infinity`),
		`that(0).is_not_within(float("-inf")).of(42)`: newInvalidAssertion(`tolerance cannot be negative`),
		`that(0).is_not_within(float("nan")).of(42)`:  newInvalidAssertion(`tolerance cannot be <nan>`),
	})
}

func TestTheOtherFloats(t *testing.T) {
	others := []string{`2`, `float("+inf")`, `float("-inf")`, `float("nan")`}
	reprs := []string{`2`, `+inf`, `-inf`, `nan`}
	m := make(map[string]error)

	for i := 0; i < len(others); i++ {
		for j := len(others) - 1; j >= 0; j-- {
			x, y := others[i], others[j]
			rx, ry := reprs[i], reprs[j]

			var expectA, expectB error
			expectA = newTruthAssertion(fmt.Sprintf(`<%s> and <%s> should have been within <1> of each other.`, rx, ry))
			if i == 0 && j == i {
				expectA = nil
				expectB = newTruthAssertion(`<2> and <2> should not have been within <1> of each other.`)
			}

			m[fmt.Sprintf(`that(%s).is_within(1).of(%s)`, x, y)] = expectA
			m[fmt.Sprintf(`that(%s).is_not_within(1).of(%s)`, x, y)] = expectB
		}
	}

	testEach(t, m)
}
