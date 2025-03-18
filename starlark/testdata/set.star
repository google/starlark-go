# Tests of Starlark 'set'
# option:set option:globalreassign

# Sets are not a standard part of Starlark, so the features
# tested in this file must be enabled in the application by setting
# resolve.AllowSet.  (All sets are created by calls to the 'set'
# built-in or derived from operations on existing sets.)
# The semantics are subject to change as the spec evolves.

# TODO(adonovan): support set mutation:
# - del set[k]
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
assert.eq(x.union(), x)
assert.eq(type(x.union(y)), "set")
assert.eq(list(x.union()), [1, 2, 3])
assert.eq(list(x.union(y)), [1, 2, 3, 4, 5])
assert.eq(list(x.union(y, [6, 7])), [1, 2, 3, 4, 5, 6, 7])
assert.eq(list(x.union([5, 1])), [1, 2, 3, 5])
assert.eq(list(x.union((6, 5, 4))), [1, 2, 3, 6, 5, 4])
assert.fails(lambda : x.union([1, 2, {}]), "unhashable type: dict")
assert.fails(lambda : x.union(1, 2, 3), "union: for parameter 1: got int, want iterable")

# set.update (allows any iterable for the right operand)
# The update function will mutate the set so the tests below are
# scoped using a function.

def test_update_return_value():
    assert.eq(set(x).update(y), None)

test_update_return_value()

def test_update_elems_singular():
    s = set("a".elems())
    s.update("b".elems())
    assert.eq(list(s), ["a", "b"])

test_update_elems_singular()

def test_update_elems_multiple():
    s = set("a".elems())
    s.update("bc".elems())
    assert.eq(list(s), ["a", "b", "c"])

test_update_elems_multiple()

def test_update_empty():
    s = set()
    s.update([])
    assert.eq(s, set())

test_update_empty()

def test_update_set():
    s = set(x)
    s.update(y)
    assert.eq(list(s), [1, 2, 3, 4, 5])

test_update_set()

def test_update_set_multiple_args():
    s = set(x)
    s.update([11, 12], [11, 13, 14])
    assert.eq(list(s), [1, 2, 3, 11, 12, 13, 14])

test_update_set_multiple_args()

def test_update_list_intersecting():
    s = set(x)
    s.update([5, 1])
    assert.eq(list(s), [1, 2, 3, 5])

test_update_list_intersecting()

def test_update_list_non_intersecting():
    s = set(x)
    s.update([6, 5, 4])
    assert.eq(list(s), [1, 2, 3, 6, 5, 4])

test_update_list_non_intersecting()

def test_update_non_hashable():
    s = set(x)
    assert.fails(lambda: x.update([1, 2, {}]), "unhashable type: dict")

test_update_non_hashable()

def test_update_non_iterable():
    s = set(x)
    assert.fails(lambda: x.update(9), "update: for parameter 1: got int, want iterable")

test_update_non_iterable()

def test_update_kwargs():
    s = set(x)
    assert.fails(lambda: x.update(gee = [3, 4]), "update: unexpected keyword arguments")

test_update_kwargs()

def test_update_no_arg():
    s = set(x)
    s.update()
    assert.eq(list(s), [1, 2, 3])

test_update_no_arg()

# intersection, set & set or set.intersection(*iterables)
assert.eq(list(set("a".elems()) & set("b".elems())), [])
assert.eq(list(set("ab".elems()) & set("bc".elems())), ["b"])
assert.eq(list(set("a".elems()).intersection("b".elems())), [])
assert.eq(list(set("ab".elems()).intersection("bc".elems())), ["b"])
assert.eq(set([1, 2, 3]).intersection([2, 3, 4, 2, 3, 4]), set([2, 3]))
assert.eq(set([1, 2, 3]).intersection([2, 3], {3: "three", 4: "four"}), set([3]))
assert.fails(lambda: set([1, 2]).intersection([[3]]), "intersection: unhashable type: list")

# intersection_update(*iterables)
intersection_update_set = set([1, 2, 3])
assert.eq(intersection_update_set.intersection_update(), None)  # no-op
assert.eq(intersection_update_set, set([1, 2, 3]))  # unchanged
assert.eq(intersection_update_set.intersection_update([2, 3, 4]), None)
assert.eq(intersection_update_set, set([2, 3]))
assert.eq(intersection_update_set.intersection_update([2, 3, 4, 2, 3, 4]), None)
assert.eq(intersection_update_set, set([2, 3]))
assert.eq(intersection_update_set.intersection_update([2, 3], {3: "three", 4: "four"}), None)
assert.eq(intersection_update_set, set([3]))
assert.fails(lambda: intersection_update_set.intersection_update(3), "intersection_update: for parameter 1: got int, want iterable")
freeze(intersection_update_set)
assert.fails(lambda: intersection_update_set.intersection_update([1]), "intersection_update: cannot delete from frozen hash table")
assert.fails(lambda: intersection_update_set.intersection_update(), "intersection_update: cannot delete from frozen hash table")

# symmetric difference, set ^ set or set.symmetric_difference(*iterables)
assert.eq(set([1, 2, 3]) ^ set([4, 5, 3]), set([1, 2, 4, 5]))
assert.eq(set([1, 2, 3, 4]).symmetric_difference([3, 4, 5, 6]), set([1, 2, 5, 6]))
assert.eq(set([1, 2, 3, 4]).symmetric_difference(set([])), set([1, 2, 3, 4]))
assert.eq(set([1, 2, 3]).symmetric_difference([2, 3, 4]), set([1, 4]))
assert.eq(set([1, 2, 3]).symmetric_difference([2, 3, 4, 2, 3, 4]), set([1, 4]))
assert.eq(set([1, 2, 3]).symmetric_difference({0: "zero", 1: "one"}), set([2, 3, 0]))
assert.fails(lambda: set([1, 2]).symmetric_difference(2), "symmetric_difference: for parameter 1: got int, want iterable")
assert.fails(lambda: set([1, 2]).symmetric_difference([1], [2]), "symmetric_difference: got 2 arguments, want 1")
assert.fails(lambda: set([1, 2]).symmetric_difference(), "symmetric_difference: got 0 arguments, want 1")

# set.symmetric_difference_update(*iterables)
symmetric_difference_update_set = set([1, 2, 3, 4])
assert.eq(symmetric_difference_update_set.symmetric_difference_update([2]), None)
assert.eq(symmetric_difference_update_set, set([1, 3, 4]))
assert.eq(symmetric_difference_update_set.symmetric_difference_update([2, 3, 2, 3]), None)
assert.eq(symmetric_difference_update_set, set([1, 2, 4]))
assert.eq(symmetric_difference_update_set.symmetric_difference_update({0: "zero", 1: "one"}), None)
assert.eq(symmetric_difference_update_set, set([0, 2, 4]))
assert.fails(lambda: symmetric_difference_update_set.symmetric_difference_update(2), "symmetric_difference_update: for parameter 1: got int, want iterable")
assert.fails(lambda: symmetric_difference_update_set.symmetric_difference_update([1], [2]), "symmetric_difference_update: got 2 arguments, want 1")
assert.fails(lambda: symmetric_difference_update_set.symmetric_difference_update(), "symmetric_difference_update: got 0 arguments, want 1")
freeze(symmetric_difference_update_set)
assert.fails(lambda: symmetric_difference_update_set.symmetric_difference_update([1]), "symmetric_difference_update: cannot insert into frozen hash table")

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
add_set.add(1)
assert.eq(list(add_set), [1, 2, 3, 4]) # adding existing element is a no-op and doesn't change iteration order
assert.fails(lambda: add_set.add([5]), "add: unhashable type: list")
def add_during_iteration(s, v):
    for _ in s:
        s.add(v)
assert.fails(lambda: add_during_iteration(add_set, 4), "add: cannot insert into hash table during iteration")
freeze(add_set)
assert.fails(lambda: add_set.add(4), "add: cannot insert into frozen hash table") # even a no-op mutation on a frozen set is an error
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
assert.fails(lambda: discard_set.discard([5]), "discard: unhashable type: list")
def discard_during_iteration(s, v):
    for _ in s:
        s.discard(v)
assert.fails(lambda: discard_during_iteration(discard_set, 2), "discard: cannot delete from hash table during iteration")
freeze(discard_set)
assert.fails(lambda: discard_set.discard(3), "discard: cannot delete from frozen hash table") # even a no-op mutation on a frozen set is an error
assert.fails(lambda: discard_set.discard(1), "discard: cannot delete from frozen hash table")

# update
update_set = set([1, 2, 3])
assert.eq(update_set.update(), None) # no-op
assert.eq(update_set, set([1, 2, 3])) # unchanged
assert.eq(update_set.update([]), None) # no-op
assert.eq(update_set, set([1, 2, 3])) # unchanged
assert.eq(update_set.update([4]), None)
assert.eq(update_set.update([4, 5], [5, 6]), None)
assert.eq(update_set, set([1, 2, 3, 4, 5, 6]))
assert.fails(lambda: update_set.update(7), "update: for parameter 1: got int, want iterable")
freeze(update_set)
assert.fails(lambda: update_set.update([7]), "update: cannot insert into frozen hash table")
assert.fails(lambda: update_set.update(), "update: cannot insert into frozen hash table")

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

# difference: set - set or set.difference(*iterables)
assert.eq(set([1,2,3,4]).difference([1,2,3,4]), set([]))
assert.eq(set([1,2,3,4]).difference([0,1,2]), set([3,4]))
assert.eq(set([1,2,3,4]).difference([]), set([1,2,3,4]))
assert.eq(set([1,2,3,4]).difference(), set([1,2,3,4]))
assert.eq(set([1,2,3,4]).difference(set([1,2,3])), set([4]))
assert.eq(set([1,2,3,4]).difference(set([1,2]), [2,3]), set([4]))

assert.eq(set([1,2,3,4]) - set([1,2,3,4]), set())
assert.eq(set([1,2,3,4]) - set([1,2]), set([3,4]))

# difference_update(*iterables)
difference_update_set = set([1, 2, 3, 4])
assert.eq(difference_update_set.difference_update(), None)  # no-op
assert.eq(difference_update_set, set([1, 2, 3, 4]))  # unchanged
assert.eq(difference_update_set.difference_update([2]), None)
assert.eq(difference_update_set, set([1, 3, 4]))
assert.eq(difference_update_set.difference_update([2, 3, 2, 3]), None)
assert.eq(difference_update_set, set([1, 4]))
assert.eq(difference_update_set.difference_update([2], {3: "three", 4: "four"}), None)
assert.eq(difference_update_set, set([1]))
assert.fails(lambda: difference_update_set.difference_update(2), "difference_update: for parameter 1: got int, want iterable")
freeze(difference_update_set)
assert.fails(lambda: difference_update_set.difference_update([1]), "difference_update: cannot delete from frozen hash table")
assert.fails(lambda: difference_update_set.difference_update(), "difference_update: cannot delete from frozen hash table")

# isdisjoint
assert.eq(set([1, 2]).isdisjoint([3, 4]), True)
assert.eq(set([1, 2]).isdisjoint([2, 3]), False)
assert.eq(set([1, 2]).isdisjoint([1]), False)
assert.eq(set([1, 2]).isdisjoint({2: "a", 3: "b"}), False)
assert.eq(set([1, 2]).isdisjoint({}), True)
assert.eq(set().isdisjoint({}), True)
assert.eq(set().isdisjoint([2, 3]), True)
assert.eq(set().isdisjoint([]), True)
assert.fails(lambda: set([1, 2]).isdisjoint(2), "isdisjoint: for parameter 1: got int, want iterable")
assert.fails(lambda: set([1, 2]).isdisjoint([1, 2], [3]), "isdisjoint: got 2 arguments, want at most 1")

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
