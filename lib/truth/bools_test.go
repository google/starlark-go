package truth

import "testing"

func TestTrue(t *testing.T) {
	testEach(t, map[string]error{
		`that(True).is_true()`:  nil,
		`that(True).is_false()`: fail("True", "is False"),
	})
}

func TestFalse(t *testing.T) {
	testEach(t, map[string]error{
		`that(False).is_false()`: nil,
		`that(False).is_true()`:  fail("False", "is True"),
	})
}

func TestTruthyThings(t *testing.T) {
	values := []string{
		`1`,
		`True`,
		`2.5`,
		`"Hi"`,
		`[3]`,
		`{4: "four"}`,
		`("my", "tuple")`,
		`set([5])`,
		`-1`,
	}
	m := make(map[string]error, 4*len(values))
	for _, v := range values {
		m[`that(`+v+`).is_truthy()`] = nil
		m[`that(`+v+`).is_falsy()`] = fail(v, "is falsy")
		m[`that(`+v+`).is_false()`] = fail(v, "is False")
		if v != `True` {
			m[`that(`+v+`).is_true()`] = fail(v, "is True",
				" However, it is truthy. Did you mean to call .is_truthy() instead?")
		}
	}
	testEach(t, m)
}

func TestFalsyThings(t *testing.T) {
	values := []string{
		`None`,
		`False`,
		`0`,
		`0.0`,
		`""`,
		`()`, // tuple
		`[]`,
		`{}`,
		`set()`,
	}
	m := make(map[string]error, 4*len(values))
	for _, v := range values {
		vv := v
		if v == `set()` {
			vv = `set([])`
		}
		m[`that(`+v+`).is_falsy()`] = nil
		m[`that(`+v+`).is_truthy()`] = fail(vv, "is truthy")
		m[`that(`+v+`).is_true()`] = fail(vv, "is True")
		if v != `False` {
			m[`that(`+v+`).is_false()`] = fail(vv, "is False",
				" However, it is falsy. Did you mean to call .is_falsy() instead?")
		}
	}
	testEach(t, m)
}
