# Tests of regexp module.

load('assert.star', 'assert')
load('regexp.star', 'regexp')

rx = regexp.compile("b(an)*a")

# type, str, pattern, groupnames
assert.eq(type(rx), "regexp.regexp")
assert.eq(str(rx), 'regexp.compile("b(an)*a")')
assert.eq(rx.pattern, "b(an)*a")
assert.eq(rx.groupnames, [None])
assert.eq(regexp.compile("(?P<key>\\w+)=(\\w+)").groupnames, ["key", None])
assert.eq(regexp.compile("x").groupnames, [])
assert.true(rx)

# a regexp is hashable, so it may be used as a dict key
patterns = {rx: "banana", regexp.compile("x"): "x"}
assert.eq(patterns[rx], "banana")

# escape
assert.eq(regexp.escape("a.b"), "a\\.b")
assert.true(regexp.compile(regexp.escape("a.b")).matches("a.b"))
assert.true(not regexp.compile(regexp.escape("a.b")).matches("axb"))
word = regexp.compile("\\b" + regexp.escape("a.b") + "\\b")
assert.true(word.matches("x a.b y"))

# matches
assert.true(rx.matches("banana"))
assert.true(rx.matches("--banana--"))
assert.true(not rx.matches("apple"))

# find returns a match, or None
m = rx.find("--banana--")
assert.eq(type(m), "regexp.match")
assert.eq(m.text, "banana")
assert.eq(m.start, 2)
assert.eq(m.end, 8)
assert.eq("--banana--"[m.start:m.end], m.text)
assert.eq(str(m), '<regexp.match "banana" at 2:8>')
assert.eq(rx.find("apple"), None)

# a match is always true, even when its text is empty, so that find may be
# tested for success
assert.true(rx.find("banana"))
assert.true(regexp.compile("x*").find("zzz"))
assert.eq(regexp.compile("x*").find("zzz").text, "")
assert.eq(regexp.compile("x+").find("zzz"), None)

# a match is hashable
assert.eq({m: "yes"}[m], "yes")

# start/end are byte offsets, so s[start:end] is the matched text
acc = regexp.compile("é").find("héllo")
assert.eq((acc.start, acc.end), (1, 3))
assert.eq("héllo"[acc.start:acc.end], "é")

# submatches
kv = regexp.compile("(\\w+)=(\\d+)")
kvm = kv.find("a=1")
assert.eq(len(kvm.submatches), 2)
assert.eq(kvm.submatches[0].text, "a")
assert.eq(kvm.submatches[1].text, "1")
assert.eq(type(kvm.submatches), "regexp.submatches")
assert.eq(type(kvm.submatches[0]), "regexp.submatch")
assert.eq(kvm.submatches[0].start, 0)
assert.eq(kvm.submatches[1].start, 2)
assert.eq(kvm.submatches[1].end, 3)
assert.eq(str(kvm.submatches[0]), '<regexp.submatch "a" at 0:1>')
assert.eq(str(kvm.submatches), '[<regexp.submatch "a" at 0:1>, <regexp.submatch "1" at 2:3>]')
# submatches are indexable, iterable and support negative indices
assert.eq([sm.text for sm in kvm.submatches], ["a", "1"])
assert.eq(kvm.submatches[-1].text, "1")
assert.eq(list(kvm.submatches), [kvm.submatches[0], kvm.submatches[1]])
# a group that did not take part in the match is None
opt = regexp.compile("a(x)?").find("a")
assert.eq(opt.submatches[0], None)
assert.eq(len(opt.submatches), 1)
# a regexp without groups has no submatches
assert.eq(len(rx.find("ba").submatches), 1)
assert.eq(len(regexp.compile("x").find("x").submatches), 0)
assert.true(not regexp.compile("x").find("x").submatches)

# submatch(index) is submatches[index], counted from zero
assert.eq(kvm.submatch(0).text, "a")
assert.eq(kvm.submatch(1).text, "1")
assert.eq(kvm.submatch(-1).text, "1")
assert.eq(opt.submatch(0), None)
# submatch(name) selects a named group
named = regexp.compile("(?P<key>\\w+)=(?P<value>\\d+)").find("a=1")
assert.eq(named.submatch("key").text, "a")
assert.eq(named.submatch("value").text, "1")
assert.eq(named.submatch("value").start, 2)
# an invalid group is an error, unlike a group that did not participate
assert.fails(lambda: kvm.submatch(2), "submatch: group index 2 out of range: regexp has 2 groups")
assert.fails(lambda: kvm.submatch(-3), "submatch: group index -3 out of range: regexp has 2 groups")
assert.fails(lambda: kvm.submatch("key"), 'submatch: regexp has no group named "key"')
assert.fails(lambda: named.submatch(""), 'submatch: regexp has no group named ""')
assert.fails(lambda: kvm.submatch(1.0), "submatch: for parameter 1: got float, want string or int")
# a submatch has no submatches of its own
assert.fails(lambda: kvm.submatches[0].submatch(0), "regexp.submatch has no .submatch field or method")
assert.fails(lambda: kvm.submatches[0].submatches, "regexp.submatch has no .submatches field or method")

# matches and submatches compare equal if they have the same text and offsets,
# so that two lookups of the same group are interchangeable
assert.eq(kv.find("a=1"), kv.find("a=1"))
assert.ne(kv.find("a=1"), kv.find("b=22"))
assert.eq(kvm.submatches[0], kvm.submatches[0])
assert.ne(kvm.submatches[0], kvm.submatches[1])
assert.eq(kvm.submatches, kvm.submatches)
assert.eq(kvm.submatch(0), kvm.submatches[0])
assert.eq({kvm.submatches[0]: 1}[kvm.submatches[0]], 1)
# matches are not ordered
assert.fails(lambda: kvm < kvm, "regexp.match < regexp.match not implemented")
assert.fails(lambda: kvm.submatches[0] <= kvm.submatches[1], "regexp.submatch <= regexp.submatch not implemented")

# findall is an iterable of matches
a = regexp.compile("a+")
assert.eq(type(a.findall("banana")), "regexp.matches")
assert.eq(str(a.findall("banana")), '<regexp.matches "a+">')
assert.eq([x.text for x in a.findall("banana")], ["a", "a", "a"])
assert.eq([(x.start, x.end) for x in a.findall("banana")], [(1, 2), (3, 4), (5, 6)])
assert.eq(len(list(a.findall("banana"))), 3)
assert.eq([x.text for x in a.findall("melon")], [])
# findall is re-iterable
all = a.findall("banana")
assert.eq(len(list(all)), 3)
assert.eq(len(list(all)), 3)
# findall is true iff there is at least one match
assert.true(a.findall("banana"))
assert.true(not a.findall("melon"))
assert.true(not regexp.compile("x+").findall("zzz"))
# submatches of each match
pairs = {x.submatches[0].text: int(x.submatches[1].text) for x in kv.findall("a=1,b=2")}
assert.eq(pairs, {"a": 1, "b": 2})
assert.eq([x.text for x in kv.findall("a=1,b=2")], ["a=1", "b=2"])
# an empty match immediately after a previous match is skipped
assert.eq([x.text for x in regexp.compile("x*").findall("xaxb")], ["x", "x", ""])

# replace with a string: $1 and ${name} expand to the corresponding submatch
assert.eq(a.replace("banana", "o"), "bonono")
assert.eq(a.replace("melon", "o"), "melon")
assert.eq(kv.replace("a=1,b=2", "$2=$1"), "1=a,2=b")
assert.eq(regexp.compile("(?P<x>\\w)").replace("ab", "<${x}>"), "<a><b>")
assert.eq(a.replace("banana", "$$"), "b$n$n$")

# replace with a function, which is called with the match
assert.eq(a.replace("banana", lambda x: x.text.upper()), "bAnAnA")
assert.eq(kv.replace("a=1,b=2", lambda x: x.submatches[1].text + "=" + x.submatches[0].text), "1=a,2=b")
assert.eq(kv.replace("a=1", lambda x: str(x.start) + ":" + str(x.end)), "0:3")
assert.eq(regexp.compile("a(x)?").replace("ax a", lambda x: str(x.submatches[0])), '<regexp.submatch "x" at 1:2> None')

def count_calls(src):
    calls = []
    def repl(x):
        calls.append(x.text)
        return "*"
    return (a.replace(src, repl), calls)

assert.eq(count_calls("banana"), ("b*n*n*", ["a", "a", "a"]))

# an error raised by the replacement function propagates
assert.fails(lambda: a.replace("banana", lambda x: fail("oops")), "oops")
# the replacement function must return a string
assert.fails(lambda: a.replace("banana", lambda x: 1), "replace: replacement function returned int, want string")
# repl must be a string or a callable
assert.fails(lambda: a.replace("banana", 1), "replace: for parameter repl: got int, want string or callable")

# split
comma = regexp.compile(",+")
assert.eq(comma.split("a,b,,c"), ["a", "b", "c"])
assert.eq(comma.split("a,b,,c", 2), ["a", "b,,c"])
assert.eq(comma.split("a,b,,c", 0), [])
assert.eq(comma.split("abc"), ["abc"])

# matches and submatches are iterable containers, and so are not hashable
assert.fails(lambda: {a.findall("banana"): 1}, "unhashable type: regexp.matches")
assert.fails(lambda: {kvm.submatches: 1}, "unhashable type: regexp.submatches")

# compile reports invalid patterns
assert.fails(lambda: regexp.compile("("), "error parsing regexp")
# \C is not supported, as it is byte-oriented
assert.fails(lambda: regexp.compile("\\C"), "invalid escape sequence")

# argument errors
assert.fails(lambda: regexp.compile(), "compile: got 0 arguments, want 1")
assert.fails(lambda: regexp.compile(1), "compile: for parameter 1: got int, want string")
assert.fails(lambda: regexp.escape(1), "escape: for parameter 1: got int, want string")
assert.fails(lambda: rx.matches(), "matches: got 0 arguments, want 1")
assert.fails(lambda: rx.find("a", "b"), "find: got 2 arguments, want 1")
assert.fails(lambda: rx.findall("a", "b"), "findall: got 2 arguments, want 1")
assert.fails(lambda: rx.nonesuch, "regexp.regexp has no .nonesuch field or method")
assert.fails(lambda: kvm.nonesuch, "regexp.match has no .nonesuch field or method")
