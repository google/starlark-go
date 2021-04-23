load("regexp.star", "regexp")
load("assert.star", "assert")

match_pattern = r"(\w*)\s*(ADD|REM|DEL|EXT|TRF)\s*(.*)\s*(NAT|INT)\s*(.*)\s*(\(\w{2}\))\s*(.*)"
match_test = "EDM ADD FROM INJURED NAT Jordan BEAULIEU (DB) Western University"
match_r = regexp.compile(match_pattern)

assert.eq(regexp.match(match_pattern,match_test), [(match_test, "EDM", "ADD", "FROM INJURED ", "NAT", "Jordan BEAULIEU ", "(DB)", "Western University")])
assert.eq(match_r.match(match_test), [(match_test, "EDM", "ADD", "FROM INJURED ", "NAT", "Jordan BEAULIEU ", "(DB)", "Western University")])

assert.eq(regexp.sub(match_pattern, "", match_test), "")
assert.eq(match_r.sub("", match_test), "")

space_r = regexp.compile(" ")
assert.eq(regexp.split(" ", "foo bar baz bat"), ("foo", "bar", "baz", "bat"))
assert.eq(space_r.split("foo bar baz bat"), ("foo", "bar", "baz", "bat"))

foo_r = regexp.compile("foo")
assert.eq(regexp.findall("foo", "foo bar baz"), ("foo",))
assert.eq(foo_r.findall("foo bar baz"), ("foo",))
