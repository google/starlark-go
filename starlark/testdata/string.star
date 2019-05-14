# Tests of Starlark 'string'
# option:float option:set

load("assert.star", "assert")

# raw string literals:
assert.eq(r'a\bc', "a\\bc")

# truth
assert.true("abc")
assert.true("\0")
assert.true(not "")

# str + str
assert.eq("a"+"b"+"c", "abc")

# str * int,  int * str
assert.eq("abc" * 0, "")
assert.eq("abc" * -1, "")
assert.eq("abc" * 1, "abc")
assert.eq("abc" * 5, "abcabcabcabcabc")
assert.eq(0 * "abc", "")
assert.eq(-1 * "abc", "")
assert.eq(1 * "abc", "abc")
assert.eq(5 * "abc", "abcabcabcabcabc")
assert.fails(lambda: 1.0 * "abc", "unknown.*float \\* str")
assert.fails(lambda : "abc" * (1000000 * 1000000), "repeat count 1000000000000 too large")
assert.fails(lambda : "abc" * 1000000 * 1000000, "excessive repeat .3000000000000 elements")

# len
assert.eq(len("Hello, 世界!"), 14)
assert.eq(len("𐐷"), 4) # U+10437 has a 4-byte UTF-8 encoding (and a 2-code UTF-16 encoding)

# chr & ord
assert.eq(chr(65), "A")       # 1-byte UTF-8 encoding
assert.eq(chr(1049), "Й")     # 2-byte UTF-8 encoding
assert.eq(chr(0x1F63F), "😿") # 4-byte UTF-8 encoding
assert.fails(lambda: chr(-1), "Unicode code point -1 out of range \\(<0\\)")
assert.fails(lambda: chr(0x110000), "Unicode code point U\\+110000 out of range \\(>0x10FFFF\\)")
assert.eq(ord("A"), 65)
assert.eq(ord("Й"), 1049)
assert.eq(ord("😿"), 0x1F63F)
assert.eq(ord("Й"[1:]), 0xFFFD) # = Unicode replacement character
assert.fails(lambda: ord("abc"), "string encodes 3 Unicode code points, want 1")
assert.fails(lambda: ord(""), "string encodes 0 Unicode code points, want 1")
assert.fails(lambda: ord("😿"[1:]), "string encodes 3 Unicode code points, want 1") # 3 x 0xFFFD

# string.codepoint_ords
assert.eq(type("abcЙ😿".codepoint_ords()), "codepoints")
assert.eq(str("abcЙ😿".codepoint_ords()), '"abcЙ😿".codepoint_ords()')
assert.eq(list("abcЙ😿".codepoint_ords()), [97, 98, 99, 1049, 128575])
assert.eq(list(("A" + "😿Z"[1:]).codepoint_ords()), [ord("A"), 0xFFFD, 0xFFFD, 0xFFFD, ord("Z")])
assert.eq(list("".codepoint_ords()), [])

# string.codepoints
assert.eq(type("abcЙ😿".codepoints()), "codepoints")
assert.eq(str("abcЙ😿".codepoints()), '"abcЙ😿".codepoints()')
assert.eq(list("abcЙ😿".codepoints()), ["a", "b", "c", "Й", "😿"])
assert.eq(list(("A" + "😿Z"[1:]).codepoints()), ["A", "\x9f", "\x98", "\xbf", "Z"])
assert.eq(list("".codepoints()), [])

# string.elem_ords
assert.eq(type("abcЙ😿".elem_ords()), "elems")
assert.eq(str("abcЙ😿".elem_ords()), '"abcЙ😿".elem_ords()')
assert.eq(list("abcЙ😿".elem_ords()), [97, 98, 99,  208, 153, 240, 159, 152, 191])
assert.eq(list(("A" + "😿Z"[1:]).elem_ords()),  [65, 159, 152, 191, 90])
assert.eq(list("".elem_ords()), [])

# string.elems
assert.eq(type("abcЙ😿".elems()), "elems")
assert.eq(str("abcЙ😿".elems()), '"abcЙ😿".elems()')
assert.eq(list("abcЙ😿".elems()),
          ["a", "b", "c", "\xd0", "\x99", "\xf0", "\x9f", "\x98", "\xbf"])
assert.eq(list(("A" + "😿Z"[1:]).elems()),
          ["A", "\x9f", "\x98", "\xbf", "Z"])
assert.eq(list("".elems()), [])

# indexing, x[i]
assert.eq("Hello, 世界!"[0], "H")
assert.eq("Hello, 世界!"[7], "\xe4")
assert.eq("Hello, 世界!"[13], "!")
assert.fails(lambda: "abc"[-4], "out of range")
assert.eq("abc"[-3], "a")
assert.eq("abc"[-2], "b")
assert.eq("abc"[-1], "c")
assert.eq("abc"[0], "a")
assert.eq("abc"[1], "b")
assert.eq("abc"[2], "c")
assert.fails(lambda: "abc"[4], "out of range")

# x[i] = ...
x2 = "abc"
def f(): x2[1] = 'B'
assert.fails(f, "string.*does not support.*assignment")

# slicing, x[i:j]
assert.eq("abc"[:], "abc")
assert.eq("abc"[-4:], "abc")
assert.eq("abc"[-3:], "abc")
assert.eq("abc"[-2:], "bc")
assert.eq("abc"[-1:], "c")
assert.eq("abc"[0:], "abc")
assert.eq("abc"[1:], "bc")
assert.eq("abc"[2:], "c")
assert.eq("abc"[3:], "")
assert.eq("abc"[4:], "")
assert.eq("abc"[:-4], "")
assert.eq("abc"[:-3], "")
assert.eq("abc"[:-2], "a")
assert.eq("abc"[:-1], "ab")
assert.eq("abc"[:0], "")
assert.eq("abc"[:1], "a")
assert.eq("abc"[:2], "ab")
assert.eq("abc"[:3], "abc")
assert.eq("abc"[:4], "abc")
assert.eq("abc"[1:2], "b")
assert.eq("abc"[2:1], "")
# non-unit strides
assert.eq("abcd"[0:4:1], "abcd")
assert.eq("abcd"[::2], "ac")
assert.eq("abcd"[1::2], "bd")
assert.eq("abcd"[4:0:-1], "dcb")
assert.eq("banana"[7::-2], "aaa")
assert.eq("banana"[6::-2], "aaa")
assert.eq("banana"[5::-2], "aaa")
assert.eq("banana"[4::-2], "nnb")
assert.eq("banana"[::-1], "ananab")
assert.eq("banana"[None:None:-2], "aaa")
assert.fails(lambda: "banana"[1.0::], "invalid start index: got float, want int")
assert.fails(lambda: "banana"[:"":], "invalid end index: got string, want int")
assert.fails(lambda: "banana"[:"":True], "got bool for slice step, want int")

# in, not in
assert.true("oo" in "food")
assert.true("ox" not in "food")
assert.true("" in "food")
assert.true("" in "")
assert.fails(lambda: 1 in "", "requires string as left operand")
assert.fails(lambda: "" in 1, "unknown binary op: string in int")

# ==, !=
assert.eq("hello", "he"+"llo")
assert.ne("hello", "Hello")

# hash must follow java.lang.String.hashCode.
wanthash = {
    "": 0,
    "\0" * 100: 0,
    "hello": 99162322,
    "world": 113318802,
    "Hello, 世界!": 417292677,
}
gothash = {s: hash(s) for s in wanthash}
assert.eq(gothash, wanthash)

# TODO(adonovan): ordered comparisons

# string % tuple formatting
assert.eq("A %d %x Z" % (123, 456), "A 123 1c8 Z")
assert.eq("A %(foo)d %(bar)s Z" % {"foo": 123, "bar":"hi"}, "A 123 hi Z")
assert.eq("%s %r" % ("hi", "hi"), 'hi "hi"') # TODO(adonovan): use ''-quotation
assert.eq("%%d %d" % 1, "%d 1")
assert.fails(lambda: "%d %d" % 1, "not enough arguments for format string")
assert.fails(lambda: "%d %d" % (1, 2, 3), "too many arguments for format string")
assert.fails(lambda: "" % 1, "too many arguments for format string")
# %c
assert.eq("%c" % 65, "A")
assert.eq("%c" % 0x3b1, "α")
assert.eq("%c" % "A", "A")
assert.eq("%c" % "α", "α")
assert.fails(lambda: "%c" % "abc", "requires a single-character string")
assert.fails(lambda: "%c" % "", "requires a single-character string")
assert.fails(lambda: "%c" % 65.0, "requires int or single-character string")
assert.fails(lambda: "%c" % 10000000, "requires a valid Unicode code point")
assert.fails(lambda: "%c" % -1, "requires a valid Unicode code point")
# TODO(adonovan): more tests

# str.format
assert.eq("a{}b".format(123), "a123b")
assert.eq("a{}b{}c{}d{}".format(1, 2, 3, 4), "a1b2c3d4")
assert.eq("a{{b".format(), "a{b")
assert.eq("a}}b".format(), "a}b")
assert.eq("a{{b}}c".format(), "a{b}c")
assert.eq("a{x}b{y}c{}".format(1, x=2, y=3), "a2b3c1")
assert.fails(lambda: "a{z}b".format(x=1), "keyword z not found")
assert.fails(lambda: "{-1}".format(1), "keyword -1 not found")
assert.fails(lambda: "{-0}".format(1), "keyword -0 not found")
assert.fails(lambda: "{+0}".format(1), "keyword \\+0 not found")
assert.fails(lambda: "{+1}".format(1), "keyword \\+1 not found") # starlark-go/issues/114
assert.eq("{0000000000001}".format(0, 1), "1")
assert.eq("{012}".format(*range(100)), "12") # decimal, despite leading zeros
assert.fails(lambda: '{0,1} and {1}'.format(1, 2), "keyword 0,1 not found")
assert.fails(lambda: "a{123}b".format(), "tuple index out of range")
assert.fails(lambda: "a{}b{}c".format(1), "tuple index out of range")
assert.eq("a{010}b".format(0,1,2,3,4,5,6,7,8,9,10), "a10b") # index is decimal
assert.fails(lambda: "a{}b{1}c".format(1, 2), "cannot switch from automatic field numbering to manual")
assert.eq("a{!s}c".format("b"), "abc")
assert.eq("a{!r}c".format("b"), r'a"b"c')
assert.eq("a{x!r}c".format(x='b'), r'a"b"c')
assert.fails(lambda: "{x!}".format(x=1), "unknown conversion")
assert.fails(lambda: "{x!:}".format(x=1), "unknown conversion")
assert.fails(lambda: '{a.b}'.format(1), "syntax x.y is not supported")
assert.fails(lambda: '{a[0]}'.format(1), "syntax a\[i\] is not supported")
assert.fails(lambda: '{ {} }'.format(1), "nested replacement fields not supported")
assert.fails(lambda: '{{}'.format(1), "single '}' in format")
assert.fails(lambda: '{}}'.format(1), "single '}' in format")
assert.fails(lambda: '}}{'.format(1), "unmatched '{' in format")
assert.fails(lambda: '}{{'.format(1), "single '}' in format")

# str.split, str.rsplit
assert.eq("a.b.c.d".split("."), ["a", "b", "c", "d"])
assert.eq("a.b.c.d".rsplit("."), ["a", "b", "c", "d"])
assert.eq("a.b.c.d".split(".", -1), ["a", "b", "c", "d"])
assert.eq("a.b.c.d".rsplit(".", -1), ["a", "b", "c", "d"])
assert.eq("a.b.c.d".split(".", 0), ["a.b.c.d"])
assert.eq("a.b.c.d".rsplit(".", 0), ["a.b.c.d"])
assert.eq("a.b.c.d".split(".", 1), ["a", "b.c.d"])
assert.eq("a.b.c.d".rsplit(".", 1), ["a.b.c", "d"])
assert.eq("a.b.c.d".split(".", 2), ["a", "b", "c.d"])
assert.eq("a.b.c.d".rsplit(".", 2), ["a.b", "c", "d"])
assert.eq("  ".split("."), ["  "])
assert.eq("  ".rsplit("."), ["  "])

# {,r}split on white space:
assert.eq(" a bc\n  def \t  ghi".split(), ["a", "bc", "def", "ghi"])
assert.eq(" a bc\n  def \t  ghi".split(None), ["a", "bc", "def", "ghi"])
assert.eq(" a bc\n  def \t  ghi".split(None, 0), ["a bc\n  def \t  ghi"])
assert.eq(" a bc\n  def \t  ghi".rsplit(None, 0), [" a bc\n  def \t  ghi"])
assert.eq(" a bc\n  def \t  ghi".split(None, 1), ["a", "bc\n  def \t  ghi"])
assert.eq(" a bc\n  def \t  ghi".rsplit(None, 1), [" a bc\n  def", "ghi"])
assert.eq(" a bc\n  def \t  ghi".split(None, 2), ["a", "bc", "def \t  ghi"])
assert.eq(" a bc\n  def \t  ghi".rsplit(None, 2), [" a bc", "def", "ghi"])
assert.eq(" a bc\n  def \t  ghi".split(None, 3), ["a", "bc", "def", "ghi"])
assert.eq(" a bc\n  def \t  ghi".rsplit(None, 3), [" a", "bc", "def", "ghi"])
assert.eq(" a bc\n  def \t  ghi".split(None, 4), ["a", "bc", "def", "ghi"])
assert.eq(" a bc\n  def \t  ghi".rsplit(None, 4), ["a", "bc", "def", "ghi"])
assert.eq(" a bc\n  def \t  ghi".rsplit(None, 5), ["a", "bc", "def", "ghi"])

assert.eq(" a bc\n  def \t  ghi ".split(None, 0), ["a bc\n  def \t  ghi "])
assert.eq(" a bc\n  def \t  ghi ".rsplit(None, 0), [" a bc\n  def \t  ghi"])
assert.eq(" a bc\n  def \t  ghi ".split(None, 1), ["a", "bc\n  def \t  ghi "])
assert.eq(" a bc\n  def \t  ghi ".rsplit(None, 1), [" a bc\n  def", "ghi"])

# Observe the algorithmic difference when splitting on spaces versus other delimiters.
assert.eq('--aa--bb--cc--'.split('-', 0), ['--aa--bb--cc--'])  # contrast this
assert.eq('  aa  bb  cc  '.split(None, 0), ['aa  bb  cc  '])   #  with this
assert.eq('--aa--bb--cc--'.rsplit('-', 0), ['--aa--bb--cc--']) # ditto this
assert.eq('  aa  bb  cc  '.rsplit(None, 0), ['  aa  bb  cc'])  #  and this
#
assert.eq('--aa--bb--cc--'.split('-', 1), ['', '-aa--bb--cc--'])
assert.eq('--aa--bb--cc--'.rsplit('-', 1), ['--aa--bb--cc-', ''])
assert.eq('  aa  bb  cc  '.split(None, 1), ['aa', 'bb  cc  '])
assert.eq('  aa  bb  cc  '.rsplit(None, 1), ['  aa  bb',  'cc'])
#
assert.eq('--aa--bb--cc--'.split('-', -1), ['', '', 'aa', '', 'bb', '', 'cc', '', ''])
assert.eq('--aa--bb--cc--'.rsplit('-', -1), ['', '', 'aa', '', 'bb', '', 'cc', '', ''])
assert.eq('  aa  bb  cc  '.split(None, -1), ['aa', 'bb', 'cc'])
assert.eq('  aa  bb  cc  '.rsplit(None, -1), ['aa', 'bb', 'cc'])
assert.eq('  '.split(None), [])
assert.eq('  '.rsplit(None), [])

assert.eq("localhost:80".rsplit(":", 1)[-1], "80")

# str.splitlines
assert.eq('\nabc\ndef'.splitlines(), ['', 'abc', 'def'])
assert.eq('\nabc\ndef'.splitlines(True), ['\n', 'abc\n', 'def'])
assert.eq('\nabc\ndef\n'.splitlines(), ['', 'abc', 'def'])
assert.eq('\nabc\ndef\n'.splitlines(True), ['\n', 'abc\n', 'def\n'])
assert.eq(''.splitlines(), []) #
assert.eq(''.splitlines(True), []) #
assert.eq('a'.splitlines(), ['a'])
assert.eq('a'.splitlines(True), ['a'])
assert.eq('\n'.splitlines(), [''])
assert.eq('\n'.splitlines(True), ['\n'])
assert.eq('a\n'.splitlines(), ['a'])
assert.eq('a\n'.splitlines(True), ['a\n'])
assert.eq('a\n\nb'.splitlines(), ['a', '', 'b'])
assert.eq('a\n\nb'.splitlines(True), ['a\n', '\n', 'b'])
assert.eq('a\nb\nc'.splitlines(), ['a', 'b', 'c'])
assert.eq('a\nb\nc'.splitlines(True), ['a\n', 'b\n', 'c'])
assert.eq('a\nb\nc\n'.splitlines(), ['a', 'b', 'c'])
assert.eq('a\nb\nc\n'.splitlines(True), ['a\n', 'b\n', 'c\n'])

# str.{,l,r}strip
assert.eq(" \tfoo\n ".strip(), "foo")
assert.eq(" \tfoo\n ".lstrip(), "foo\n ")
assert.eq(" \tfoo\n ".rstrip(), " \tfoo")
assert.eq(" \tfoo\n ".strip(""), "foo")
assert.eq(" \tfoo\n ".lstrip(""), "foo\n ")
assert.eq(" \tfoo\n ".rstrip(""), " \tfoo")
assert.eq("blah.h".strip("b.h"), "la")
assert.eq("blah.h".lstrip("b.h"), "lah.h")
assert.eq("blah.h".rstrip("b.h"), "bla")

# str.count
assert.eq("banana".count("a"), 3)
assert.eq("banana".count("a", 2), 2)
assert.eq("banana".count("a", -4, -2), 1)
assert.eq("banana".count("a", 1, 4), 2)
assert.eq("banana".count("a", 0, -100), 0)

# str.{starts,ends}with
assert.true("foo".endswith("oo"))
assert.true(not "foo".endswith("x"))
assert.true("foo".startswith("fo"))
assert.true(not "foo".startswith("x"))
assert.fails(lambda: "foo".startswith(1), "got int.*want string")
#
assert.true('abc'.startswith(('a', 'A')))
assert.true('ABC'.startswith(('a', 'A')))
assert.true(not 'ABC'.startswith(('b', 'B')))
assert.fails(lambda: '123'.startswith((1, 2)), 'got int, for element 0')
assert.fails(lambda: '123'.startswith(['3']), 'got list')
#
assert.true('abc'.endswith(('c', 'C')))
assert.true('ABC'.endswith(('c', 'C')))
assert.true(not 'ABC'.endswith(('b', 'B')))
assert.fails(lambda: '123'.endswith((1, 2)), 'got int, for element 0')
assert.fails(lambda: '123'.endswith(['3']), 'got list')
# start/end
assert.true('abc'.startswith('bc', 1))
assert.true(not 'abc'.startswith('b', 999))
assert.true('abc'.endswith('ab', None, -1))
assert.true(not 'abc'.endswith('b', None, -999))

# str.replace
assert.eq("banana".replace("a", "o", 1), "bonana")
assert.eq("banana".replace("a", "o"), "bonono")
# TODO(adonovan): more tests

# str.{,r}find
assert.eq("foofoo".find("oo"), 1)
assert.eq("foofoo".find("ox"), -1)
assert.eq("foofoo".find("oo", 2), 4)
assert.eq("foofoo".rfind("oo"), 4)
assert.eq("foofoo".rfind("ox"), -1)
assert.eq("foofoo".rfind("oo", 1, 4), 1)
assert.eq("foofoo".find(""), 0)
assert.eq("foofoo".rfind(""), 6)

# str.{,r}partition
assert.eq("foo/bar/wiz".partition("/"), ("foo", "/", "bar/wiz"))
assert.eq("foo/bar/wiz".rpartition("/"), ("foo/bar", "/", "wiz"))
assert.eq("foo/bar/wiz".partition("."), ("foo/bar/wiz", "", ""))
assert.eq("foo/bar/wiz".rpartition("."), ("", "", "foo/bar/wiz"))
assert.fails(lambda: "foo/bar/wiz".partition(""), "empty separator")
assert.fails(lambda: "foo/bar/wiz".rpartition(""), "empty separator")

assert.eq('?'.join(["foo", "a/b/c.go".rpartition("/")[0]]), 'foo?a/b')

# str.is{alpha,...}
def test_predicates():
  predicates = ["alnum", "alpha", "digit", "lower", "space", "title", "upper"]
  table = {
      "Hello, World!": "title",
      "hello, world!": "lower",
      "base64": "alnum lower",
      "HAL-9000": "upper",
      "Catch-22": "title",
      "": "",
      "\n\t\r": "space",
      "abc": "alnum alpha lower",
      "ABC": "alnum alpha upper",
      "123": "alnum digit",
      "ǄǇ": "alnum alpha upper",
      "ǅǈ": "alnum alpha",
      "ǅ ǈ": "title",
      "ǆǉ": "alnum alpha lower",
  }
  for str, want in table.items():
    got = ' '.join([name for name in predicates if getattr(str, "is"+name)()])
    if got != want:
      assert.fail("%r matched [%s], want [%s]" % (str, got, want))
test_predicates()

# Strings are not iterable.
# ok
assert.eq(len("abc"), 3)                       # len
assert.true("a" in "abc")                      # str in str
assert.eq("abc"[1], "b")                       # indexing
# not ok
def for_string():
  for x in "abc":
    pass
def args(*args): return args
assert.fails(lambda: args(*"abc"), "must be iterable, not string") # varargs
assert.fails(lambda: list("abc"), "got string, want iterable") # list(str)
assert.fails(lambda: tuple("abc"), "got string, want iterable") # tuple(str)
assert.fails(lambda: set("abc"), "got string, want iterable") # set(str)
assert.fails(lambda: set() | "abc", "unknown binary op: set | string")  # set union
assert.fails(lambda: enumerate("ab"), "got string, want iterable") # enumerate
assert.fails(lambda: sorted("abc"), "got string, want iterable") # sorted
assert.fails(lambda: [].extend("bc"), "got string, want iterable") # list.extend
assert.fails(lambda: ",".join("abc"), "got string, want iterable") # string.join
assert.fails(lambda: dict(["ab"]), "not iterable .*string") # dict
# The Java implementation does not correctly reject the following cases:
# (See Google Issue b/34385336)
assert.fails(for_string, "string value is not iterable") # for loop
assert.fails(lambda: [x for x in "abc"], "string value is not iterable") # comprehension
assert.fails(lambda: all("abc"), "got string, want iterable") # all
assert.fails(lambda: any("abc"), "got string, want iterable") # any
assert.fails(lambda: reversed("abc"), "got string, want iterable") # reversed
assert.fails(lambda: zip("ab", "cd"), "not iterable: string") # zip

# str.join
assert.eq(','.join([]), '')
assert.eq(','.join(["a"]), 'a')
assert.eq(','.join(["a", "b"]), 'a,b')
assert.eq(','.join(["a", "b", "c"]), 'a,b,c')
assert.eq(','.join(("a", "b", "c")), 'a,b,c')
assert.eq(''.join(("a", "b", "c")), 'abc')
assert.fails(lambda: ''.join(None), 'got NoneType, want iterable')
assert.fails(lambda: ''.join(["one", 2]), 'join: in list, want string, got int')

# TODO(adonovan): tests for: {,r}index

# str.capitalize
assert.eq("hElLo, WoRlD!".capitalize(), "Hello, world!")
assert.eq("por qué".capitalize(), "Por qué")
assert.eq("¿Por qué?".capitalize(), "¿por qué?")

# str.lower
assert.eq("hElLo, WoRlD!".lower(), "hello, world!")
assert.eq("por qué".lower(), "por qué")
assert.eq("¿Por qué?".lower(), "¿por qué?")
assert.eq("ǇUBOVIĆ".lower(), "ǉubović")
assert.true("ǆenan ǉubović".islower())

# str.upper
assert.eq("hElLo, WoRlD!".upper(), "HELLO, WORLD!")
assert.eq("por qué".upper(), "POR QUÉ")
assert.eq("¿Por qué?".upper(), "¿POR QUÉ?")
assert.eq("ǉubović".upper(), "ǇUBOVIĆ")
assert.true("ǄENAN ǇUBOVIĆ".isupper())

# str.title
assert.eq("hElLo, WoRlD!".title(), "Hello, World!")
assert.eq("por qué".title(), "Por Qué")
assert.eq("¿Por qué?".title(), "¿Por Qué?")
assert.eq("ǉubović".title(), "ǈubović")
assert.true("ǅenan ǈubović".istitle())
assert.true(not "Ǆenan Ǉubović".istitle())

# method spell check
assert.fails(lambda: "".starts_with, "no .starts_with field.*did you mean .startswith")
assert.fails(lambda: "".StartsWith, "no .StartsWith field.*did you mean .startswith")
assert.fails(lambda: "".fin, "no .fin field.*.did you mean .find")
