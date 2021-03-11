package truth

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
)

type asWhat int

const (
	asFunc asWhat = iota
	asModule
)

const abc = `"abc"` // Please linter

func helper(t *testing.T, as asWhat, program string) (starlark.StringDict, error) {
	t.Helper() // TODO: make this work (for failed test reports)

	// Enabled so they can be tested
	resolve.AllowFloat = true
	resolve.AllowSet = true
	resolve.AllowLambda = true

	predeclared := starlark.StringDict{}
	if as == asModule {
		NewModule(predeclared)
	} else {
		starlark.Universe["that"] = starlark.NewBuiltin("that", That)
	}

	thread := &starlark.Thread{
		Name: t.Name(),
		Print: func(_ *starlark.Thread, msg string) {
			t.Logf("--> %s", msg)
		},
		Load: func(_ *starlark.Thread, module string) (starlark.StringDict, error) {
			return nil, errors.New("load() disabled")
		},
	}

	script := strings.Join([]string{
		`dfltCmp = ` + cmpSrc,
		`someCmp = lambda a, b: dfltCmp(b, a)`,
		program,
	}, "\n")

	d, err := starlark.ExecFile(thread, t.Name()+".star", script, predeclared)
	if err != nil {
		return nil, err
	}
	if err := Close(thread); err != nil {
		return nil, err
	}
	return d, nil
}

func testEach(t *testing.T, m map[string]error, asSlice ...asWhat) {
	as := asFunc
	for _, as = range asSlice {
	}
	for code, expectedErr := range m {
		t.Run(code, func(t *testing.T) {
			globals, err := helper(t, as, code)
			delete(globals, "dfltCmp")
			delete(globals, "someCmp")
			delete(globals, "fortytwo")
			require.Empty(t, globals)
			if expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.EqualError(t, err, expectedErr.Error())
				if _, ok := err.(UnhandledError); !ok {
					require.True(t, errors.As(err, &expectedErr))
					require.IsType(t, expectedErr, err)
				}
			}
		})
	}
}

func fail(value, expected string, suffixes ...string) error {
	var suffix string
	switch len(suffixes) {
	case 0:
	case 1:
		suffix = suffixes[0]
	default:
		panic(`There must be only one suffix`)
	}
	msg := "Not true that <" + value + "> " + expected + "." + suffix
	return newTruthAssertion(msg)
}

func TestClosedness(t *testing.T) {
	testEach(t, map[string]error{
		`
fortytwo = that(True)
that(False).is_false()
`: UnresolvedError("TestClosedness/_fortytwo_=_that(True)_that(False).is_false()_.star:4:16"),
		`
fortytwo = that(True)
fortytwo.is_true()
that(False).is_false()
`: nil,
	})
	testEach(t, map[string]error{
		`assert.that(True)`:           UnresolvedError("TestClosedness/assert.that(True).star:3:12"),
		`assert.that(True).is_true()`: nil,

		`assert.that(True).named("eh")`:           UnresolvedError(`TestClosedness/assert.that(True).named("eh").star:3:12`),
		`assert.that(True).named("eh").is_true()`: nil,

		`assert.that(10).is_within(0.1)`:            UnresolvedError("TestClosedness/assert.that(10).is_within(0.1).star:3:12"),
		`assert.that(10).is_within(0.1).of(10)`:     nil,
		`assert.that(10).is_not_within(0.1)`:        UnresolvedError("TestClosedness/assert.that(10).is_not_within(0.1).star:3:12"),
		`assert.that(10).is_not_within(0.1).of(42)`: nil,
	}, asModule)
}

func TestAsValue(t *testing.T) {
	testEach(t, map[string]error{
		`
fortytwo = that(42)
fortytwo.is_equal_to(42.0)
fortytwo.is_not_callable()
fortytwo.is_at_least(42)
`: nil,

		`
fortytwo = that([1,2,3])
fortytwo.contains(2)
fortytwo.contains_exactly(1,2,3)
fortytwo.contains_exactly(1,2,3).in_order()
fortytwo.contains_all_of(2,3).in_order()
`: nil,
	})
}

func TestImpossibleInOrder(t *testing.T) {
	testEach(t, map[string]error{
		`that([1,2,3]).is_ordered()`: nil,
		`that([1,2,3]).in_order()`:   fmt.Errorf(`Invalid assertion .in_order() on value of type list`),
	})
}
