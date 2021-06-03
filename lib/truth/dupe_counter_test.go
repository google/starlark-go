package truth

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

var (
	a = starlark.String("a")
	b = starlark.String("b")

	alist     = starlark.NewList([]starlark.Value{a})
	emptylist = starlark.NewList([]starlark.Value{})
)

func maker(t *testing.T) starlark.Value {
	t.Helper()
	d := starlark.NewDict(1)
	err := d.SetKey(starlark.String("a"), starlark.NewList([]starlark.Value{
		starlark.String("b"),
		starlark.String("c"),
	}))
	require.NoError(t, err)
	return d
}

func TestDupeCounterContains(t *testing.T) {
	d := newDuplicateCounter() // {}
	require.False(t, d.Contains(a))
	require.False(t, d.Contains(b))

	d.Increment(a) // {'a': 1}
	require.True(t, d.Contains(a))
	require.False(t, d.Contains(b))

	d.Decrement(a) // {}
	require.False(t, d.Contains(a))
	require.False(t, d.Contains(b))
}

func TestDupeCounterLen(t *testing.T) {
	d := newDuplicateCounter()
	require.True(t, d.Empty())
	d.Increment(a)
	require.Equal(t, 1, d.Len())
	d.Increment(alist)
	require.Equal(t, 2, d.Len())
}

func TestDupeCounterEverything(t *testing.T) {
	d := newDuplicateCounter() // {}
	require.True(t, d.Empty())
	require.Equal(t, ``, d.String())

	d.Increment(a) // {'a': 1}
	require.Equal(t, 1, d.Len())
	require.Equal(t, `"a"`, d.String())

	d.Increment(a) // {'a': 2}
	require.Equal(t, 1, d.Len())
	require.Equal(t, `"a" [2 copies]`, d.String())

	d.Increment(b) // {'a': 2, 'b': 1}
	require.Equal(t, 2, d.Len())
	require.Equal(t, `"a" [2 copies], "b"`, d.String())

	d.Decrement(a) // {'a': 1, 'b': 1}
	require.Equal(t, 2, d.Len())
	require.Equal(t, `"a", "b"`, d.String())

	d.Decrement(a) // {'b': 1}
	require.Equal(t, 1, d.Len())
	require.Equal(t, `"b"`, d.String())

	d.Increment(a) // {'b': 1, 'a': 1}
	require.Equal(t, 2, d.Len())
	require.Equal(t, `"b", "a"`, d.String())

	d.Decrement(a) // {'b': 1}
	require.Equal(t, 1, d.Len())
	require.Equal(t, `"b"`, d.String())

	d.Decrement(b) // {}
	require.True(t, d.Empty())
	require.Equal(t, ``, d.String())

	d.Decrement(a) // {}
	require.True(t, d.Empty())
	require.Equal(t, ``, d.String())
}

func TestDupeCounterUnhashableKeys(t *testing.T) {
	d := newDuplicateCounter() // {}
	require.False(t, d.Contains(emptylist))

	d.Increment(alist) // {['a']: 1}
	require.True(t, d.Contains(alist))
	require.Equal(t, 1, d.Len())
	require.Equal(t, `["a"]`, d.String())

	d.Decrement(alist) // {}
	require.False(t, d.Contains(alist))
	require.True(t, d.Empty())
	require.Equal(t, ``, d.String())
}

func TestDupeCounterIncrementEquivalentDictionaries(t *testing.T) {
	d := newDuplicateCounter()
	d.Increment(maker(t))
	d.Increment(maker(t))
	d.Increment(maker(t))
	d.Increment(maker(t))
	require.Equal(t, 1, d.Len())
	require.Equal(t, `{"a": ["b", "c"]} [4 copies]`, d.String())
}

func TestDupeCounterDecrementEquivalentDictionaries(t *testing.T) {
	d := newDuplicateCounter()
	d.Increment(maker(t))
	d.Increment(maker(t))
	require.Equal(t, 1, d.Len())
	d.Decrement(maker(t))
	require.Equal(t, 1, d.Len())
	d.Decrement(maker(t))
	require.True(t, d.Empty())
	d.Decrement(maker(t))
	require.True(t, d.Empty())
}
