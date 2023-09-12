# Tests of Starlark 'set'
# option:set option:globalreassign

# Sets are not a standard part of Starlark, so the features
# tested in this file must be enabled in the application by setting
# resolve.AllowSet.  (All sets are created by calls to the 'set'
# built-in or derived from operations on existing sets.)
# The semantics are subject to change as the spec evolves.

# TODO(adonovan): support set mutation:
# - del set[k]
# - set.update
# - set += iterable, perhaps?
# Test iterator invalidation.

load("assert.star", "assert", "freeze")

# literals
# Parser does not currently support {1, 2, 3}.
# TODO(adonovan): add test to syntax/testdata/errors.star.

# set comprehensions
# Parser does not currently support {x for x in y}.
# See syntax/testdata/errors.star.

# set constructor
assert.eq(type(set()), "set")
assert.eq(list(set()), [])
assert.eq(type(set([1, 3, 2, 3])), "set")
assert.eq(list(set([1, 3, 2, 3])), [1, 3, 2])
assert.eq(type(set("hello".elems())), "set")
assert.eq(list(set("hello".elems())), ["h", "e", "l", "o"])
assert.eq(list(set(range(3))), [0, 1, 2])
assert.fails(lambda : set(1), "got int, want iterable")
assert.fails(lambda : set(1, 2, 3), "got 3 arguments")
assert.fails(lambda : set([1, 2, {}]), "unhashable type: dict")

# truth
assert.true(not set())
assert.true(set([False]))
assert.true(set([1, 2, 3]))

x = set([1, 2, 3])
y = set([3, 4, 5])

# set + any is not defined
assert.fails(lambda : x + y, "unknown.*: set \\+ set")

# set | set
assert.eq(list(set("a".elems()) | set("b".elems())), ["a", "b"])
assert.eq(list(set("ab".elems()) | set("bc".elems())), ["a", "b", "c"])
assert.fails(lambda : set() | [], "unknown binary op: set | list")
assert.eq(type(x | y), "set")
assert.eq(list(x | y), [1, 2, 3, 4, 5])
assert.eq(list(x | set([5, 1])), [1, 2, 3, 5])
assert.eq(list(x | set((6, 5, 4))), [1, 2, 3, 6, 5, 4])

# set.union (allows any iterable for right operand)
assert.eq(list(set("a".elems()).union("b".elems())), ["a", "b"])
assert.eq(list(set("ab".elems()).union("bc".elems())), ["a", "b", "c"])
assert.eq(set().union([]), set())
assert.eq(type(x.union(y)), "set")
assert.eq(list(x.union(y)), [1, 2, 3, 4, 5])
assert.eq(list(x.union([5, 1])), [1, 2, 3, 5])
assert.eq(list(x.union((6, 5, 4))), [1, 2, 3, 6, 5, 4])
assert.fails(lambda : x.union([1, 2, {}]), "unhashable type: dict")

# intersection, set & set or set.intersection(iterable)
assert.eq(list(set("a".elems()) & set("b".elems())), [])
assert.eq(list(set("ab".elems()) & set("bc".elems())), ["b"])
assert.eq(list(set("a".elems()).intersection("b".elems())), [])
assert.eq(list(set("ab".elems()).intersection("bc".elems())), ["b"])

# symmetric difference, set ^ set or set.symmetric_difference(iterable)
assert.eq(set([1, 2, 3]) ^ set([4, 5, 3]), set([1, 2, 4, 5]))
assert.eq(set([1,2,3,4]).symmetric_difference([3,4,5,6]), set([1,2,5,6]))
assert.eq(set([1,2,3,4]).symmetric_difference(set([])), set([1,2,3,4]))

def test_set_augmented_assign():
    x = set([1, 2, 3])
    x &= set([2, 3])
    assert.eq(x, set([2, 3]))
    x |= set([1])
    assert.eq(x, set([1, 2, 3]))
    x ^= set([4, 5, 3])
    assert.eq(x, set([1, 2, 4, 5]))

test_set_augmented_assign()

# len
assert.eq(len(x), 3)
assert.eq(len(y), 3)
assert.eq(len(x | y), 5)

# str
assert.eq(str(set([1])), "set([1])")
assert.eq(str(set([2, 3])), "set([2, 3])")
assert.eq(str(set([3, 2])), "set([3, 2])")

# comparison
assert.eq(x, x)
assert.eq(y, y)
assert.true(x != y)
assert.eq(set([1, 2, 3]), set([3, 2, 1]))

# iteration
assert.true(type([elem for elem in x]), "list")
assert.true(list([elem for elem in x]), [1, 2, 3])

def iter():
    list = []
    for elem in x:
        list.append(elem)
    return list

assert.eq(iter(), [1, 2, 3])

# sets are not indexable
assert.fails(lambda : x[0], "unhandled.*operation")

# adding and removing
add_set = set([1,2,3])
add_set.add(4)
assert.true(4 in add_set)
freeze(add_set) # no mutation of frozen set because key already present
add_set.add(4)
assert.fails(lambda: add_set.add(5), "add: cannot insert into frozen hash table")

# remove
remove_set = set([1,2,3])
remove_set.remove(3)
assert.true(3 not in remove_set)
assert.fails(lambda: remove_set.remove(3), "remove: missing key")
freeze(remove_set)
assert.fails(lambda: remove_set.remove(3), "remove: cannot delete from frozen hash table")

# discard
discard_set = set([1,2,3])
discard_set.discard(3)
assert.true(3 not in discard_set)
assert.eq(discard_set.discard(3), None)
freeze(discard_set)
assert.eq(discard_set.discard(3), None)  # no mutation of frozen set because key doesn't exist
assert.fails(lambda: discard_set.discard(1), "discard: cannot delete from frozen hash table")


# pop
pop_set = set([1,2,3])
assert.eq(pop_set.pop(), 1)
assert.eq(pop_set.pop(), 2)
assert.eq(pop_set.pop(), 3)
assert.fails(lambda: pop_set.pop(), "pop: empty set")
pop_set.add(1)
pop_set.add(2)
freeze(pop_set)
assert.fails(lambda: pop_set.pop(), "pop: cannot delete from frozen hash table")

# clear
clear_set = set([1,2,3])
clear_set.clear()
assert.eq(len(clear_set), 0)
freeze(clear_set) # no mutation of frozen set because its already empty
assert.eq(clear_set.clear(), None) 

other_clear_set = set([1,2,3])
freeze(other_clear_set)
assert.fails(lambda: other_clear_set.clear(), "clear: cannot clear frozen hash table")

# difference: set - set or set.difference(iterable)
assert.eq(set([1,2,3,4]).difference([1,2,3,4]), set([]))
assert.eq(set([1,2,3,4]).difference([1,2]), set([3,4]))
assert.eq(set([1,2,3,4]).difference([]), set([1,2,3,4]))
assert.eq(set([1,2,3,4]).difference(set([1,2,3])), set([4]))

assert.eq(set([1,2,3,4]) - set([1,2,3,4]), set())
assert.eq(set([1,2,3,4]) - set([1,2]), set([3,4]))

# issuperset: set >= set or set.issuperset(iterable)
assert.true(set([1,2,3]).issuperset([1,2]))
assert.true(not set([1,2,3]).issuperset(set([1,2,4])))
assert.true(set([1,2,3]) >= set([1,2,3]))
assert.true(set([1,2,3]) >= set([1,2]))
assert.true(not set([1,2,3]) >= set([1,2,4]))

# proper superset: set > set
assert.true(set([1, 2, 3]) > set([1, 2]))
assert.true(not set([1,2, 3]) > set([1, 2, 3]))

# issubset: set <= set or set.issubset(iterable)
assert.true(set([1,2]).issubset([1,2,3]))
assert.true(not set([1,2,3]).issubset(set([1,2,4])))
assert.true(set([1,2,3]) <= set([1,2,3]))
assert.true(set([1,2]) <= set([1,2,3]))
assert.true(not set([1,2,3]) <= set([1,2,4]))

# proper subset: set < set
assert.true(set([1,2]) < set([1,2,3]))
assert.true(not set([1,2,3]) < set([1,2,3]))
