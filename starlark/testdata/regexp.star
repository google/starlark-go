# Tests of regexp module.

load('assert.star', 'assert')
load('regexp.star', 'regexp')

rx = regexp.compile("b(an)*a")

# type, str, pattern
assert.eq(type(rx), "regexp.regexp")
assert.eq(str(rx), 'regexp.compile("b(an)*a")')
assert.eq(rx.pattern, "b(an)*a")
assert.true(rx)

# a regexp is hashable, so it may be used as a dict key
patterns = {
    rx: "banana",
    regexp.compile("x"): "x",
}
assert.eq(patterns[rx], "banana")

# matches
assert.true(rx.matches("banana"))
assert.true(rx.matches("--banana--"))
assert.true(not rx.matches("apple"))

# find
assert.eq(rx.find("--banana--"), "banana")
assert.eq(rx.find("ba"), "ba")
assert.eq(rx.find("apple"), None)
# None distinguishes "no match" from a match of the empty string
empty = regexp.compile("x*")
assert.eq(empty.find("zzz"), "")
assert.eq(regexp.compile("x+").find("zzz"), None)

# find_all
a = regexp.compile("a+")
assert.eq(a.find_all("banana"), ["a", "a", "a"])
assert.eq(a.find_all("banana", 2), ["a", "a"])
assert.eq(a.find_all("banana", -1), ["a", "a", "a"])
assert.eq(a.find_all("banana", 0), [])
assert.eq(a.find_all("melon"), [])

# find_submatches
assert.eq(rx.find_submatches("banana"), ["banana", "an"])
assert.eq(rx.find_submatches("apple"), [])
# a subpattern that does not participate in the match yields ""
assert.eq(regexp.compile("a(x)?").find_submatches("a"), ["a", ""])
# named subpatterns are ordinary subpatterns here
assert.eq(regexp.compile("(?P<key>\\w+)=(?P<value>\\w+)").find_submatches("a=1"), ["a=1", "a", "1"])

# find_all_submatches
kv = regexp.compile("(\\w+)=(\\w+)")
assert.eq(kv.find_all_submatches("a=1,b=2"), [["a=1", "a", "1"], ["b=2", "b", "2"]])
assert.eq(kv.find_all_submatches("a=1,b=2", 1), [["a=1", "a", "1"]])
assert.eq(kv.find_all_submatches("nope"), [])

# replace_all with a string
assert.eq(a.replace_all("banana", "o"), "bonono")
assert.eq(a.replace_all("melon", "o"), "melon")
# $1 and ${name} expand to the corresponding submatch; $$ is a literal $
assert.eq(kv.replace_all("a=1,b=2", "$2=$1"), "1=a,2=b")
assert.eq(regexp.compile("(?P<x>\\w)").replace_all("ab", "<${x}>"), "<a><b>")
assert.eq(a.replace_all("banana", "$$"), "b$n$n$")

# replace_all with a function
assert.eq(a.replace_all("banana", lambda m: m.upper()), "bAnAnA")
assert.eq(kv.replace_all("a=1,b=2", lambda m: m.replace("=", ":")), "a:1,b:2")

def count_calls(src):
    calls = []
    def repl(m):
        calls.append(m)
        return "*"
    return (a.replace_all(src, repl), calls)

assert.eq(count_calls("banana"), ("b*n*n*", ["a", "a", "a"]))

# an error raised by the replacement function propagates
assert.fails(lambda: a.replace_all("banana", lambda m: fail("oops")), "oops")
# the replacement function must return a string
assert.fails(lambda: a.replace_all("banana", lambda m: 1), "replace_all: replacement function returned int, want string")
# repl must be a string or a callable
assert.fails(lambda: a.replace_all("banana", 1), "replace_all: for parameter repl: got int, want string or callable")

# split
comma = regexp.compile(",+")
assert.eq(comma.split("a,b,,c"), ["a", "b", "c"])
assert.eq(comma.split("a,b,,c", 2), ["a", "b,,c"])
assert.eq(comma.split("a,b,,c", 0), [])
assert.eq(comma.split("abc"), ["abc"])

# quote
assert.eq(regexp.quote("a.b"), "a\\.b")
assert.true(regexp.compile(regexp.quote("a.b")).matches("a.b"))
assert.true(not regexp.compile(regexp.quote("a.b")).matches("axb"))

# compile reports invalid patterns
assert.fails(lambda: regexp.compile("("), "error parsing regexp")
# \C is not supported, as it is byte-oriented
assert.fails(lambda: regexp.compile("\\C"), "invalid escape sequence")

# argument errors
assert.fails(lambda: regexp.compile(), "compile: got 0 arguments, want 1")
assert.fails(lambda: regexp.compile(1), "compile: for parameter 1: got int, want string")
assert.fails(lambda: rx.matches(), "matches: got 0 arguments, want 1")
assert.fails(lambda: rx.find("a", "b"), "find: got 2 arguments, want 1")
assert.fails(lambda: a.find_all("a", "b"), "find_all: for parameter 2: got string, want int")
assert.fails(lambda: rx.nonesuch, "regexp.regexp has no .nonesuch field or method")
