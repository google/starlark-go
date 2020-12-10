# Tests of 'bytes' (immutable byte strings).

load("assert.star", "assert")

# bytes(string) -- UTF-k to UTF-8 transcoding with U+FFFD replacement
hello = bytes("hello, 世界")
goodbye = bytes("goodbye")
empty = bytes("")
nonprinting = bytes("\t\n\x7F\u200D")  # TAB, NEWLINE, DEL, ZERO_WIDTH_JOINER
assert.eq(bytes("hello, 世界"[:-1]), b"hello, 世��")

# bytes(iterable of int) -- construct from numeric byte values
assert.eq(bytes([65, 66, 67]), b"ABC")
assert.eq(bytes((65, 66, 67)), b"ABC")
assert.eq(bytes([0xf0, 0x9f, 0x98, 0xbf]), b"😿")
assert.fails(lambda: bytes([300]),
             "at index 0, 300 out of range .want value in unsigned 8-bit range")
assert.fails(lambda: bytes([b"a"]),
             "at index 0, got bytes, want int")
assert.fails(lambda: bytes(1), "want string, bytes, or iterable of ints")

# literals
assert.eq(b"hello, 世界", hello)
assert.eq(b"goodbye", goodbye)
assert.eq(b"", empty)
assert.eq(b"\t\n\x7F\u200D", nonprinting)
assert.ne("abc", b"abc")
assert.eq(b"\012\xff\u0400\U0001F63F", b"\n\xffЀ😿") # see scanner tests for more
assert.eq(rb"\r\n\t", b"\\r\\n\\t") # raw

# type
assert.eq(type(hello), "bytes")

# len
assert.eq(len(hello), 13)
assert.eq(len(goodbye), 7)
assert.eq(len(empty), 0)
assert.eq(len(b"A"), 1)
assert.eq(len(b"Ѐ"), 2)
assert.eq(len(b"世"), 3)
assert.eq(len(b"😿"), 4)

# truth
assert.true(hello)
assert.true(goodbye)
assert.true(not empty)

# str(bytes) does UTF-8 to UTF-k transcoding.
# TODO(adonovan): specify.
assert.eq(str(hello), "hello, 世界")
assert.eq(str(hello[:-1]), "hello, 世��")  # incomplete UTF-8 encoding => U+FFFD
assert.eq(str(goodbye), "goodbye")
assert.eq(str(empty), "")
assert.eq(str(nonprinting), "\t\n\x7f\u200d")
assert.eq(str(b"\xED\xB0\x80"), "���") # UTF-8 encoding of unpaired surrogate => U+FFFD x 3

# repr
assert.eq(repr(hello), r'b"hello, 世界"')
assert.eq(repr(hello[:-1]), r'b"hello, 世\xe7\x95"')  # (incomplete UTF-8 encoding )
assert.eq(repr(goodbye), 'b"goodbye"')
assert.eq(repr(empty), 'b""')
assert.eq(repr(nonprinting), 'b"\\t\\n\\x7f\\u200d"')

# equality
assert.eq(hello, hello)
assert.ne(hello, goodbye)
assert.eq(b"goodbye", goodbye)

# ordered comparison
assert.lt(b"abc", b"abd")
assert.lt(b"abc", b"abcd")
assert.lt(b"\x7f", b"\x80") # bytes compare as uint8, not int8

# bytes are dict-hashable
dict = {hello: 1, goodbye: 2}
dict[b"goodbye"] = 3
assert.eq(len(dict), 2)
assert.eq(dict[goodbye], 3)

# hash(bytes) is 32-bit FNV-1a.
assert.eq(hash(b""), 0x811c9dc5)
assert.eq(hash(b"a"), 0xe40c292c)
assert.eq(hash(b"ab"), 0x4d2505ca)
assert.eq(hash(b"abc"), 0x1a47e90b)

# indexing
assert.eq(goodbye[0], b"g")
assert.eq(goodbye[-1], b"e")
assert.fails(lambda: goodbye[100], "out of range")

# slicing
assert.eq(goodbye[:4], b"good")
assert.eq(goodbye[4:], b"bye")
assert.eq(goodbye[::2], b"gobe")
assert.eq(goodbye[3:4], b"d")  # special case: len=1
assert.eq(goodbye[4:4], b"")  # special case: len=0

# bytes in bytes
assert.eq(b"bc" in b"abcd", True)
assert.eq(b"bc" in b"dcab", False)
assert.fails(lambda: "bc" in b"dcab", "requires bytes or int as left operand, not string")

# int in bytes
assert.eq(97 in b"abc", True)  # 97='a'
assert.eq(100 in b"abc", False) # 100='d'
assert.fails(lambda: 256 in b"abc", "int in bytes: 256 out of range")
assert.fails(lambda: -1 in b"abc", "int in bytes: -1 out of range")

# ord   TODO(adonovan): specify
assert.eq(ord(b"a"), 97)
assert.fails(lambda: ord(b"ab"), "ord: bytes has length 2, want 1")
assert.fails(lambda: ord(b""), "ord: bytes has length 0, want 1")

# repeat (bytes * int)
assert.eq(goodbye * 3, b"goodbyegoodbyegoodbye")
assert.eq(3 * goodbye, b"goodbyegoodbyegoodbye")

# elems() returns an iterable value over 1-byte substrings.
assert.eq(type(hello.elems()), "bytes.elems")
assert.eq(str(hello.elems()), "b\"hello, 世界\".elems()")
assert.eq(list(hello.elems()), [104, 101, 108, 108, 111, 44, 32, 228, 184, 150, 231, 149, 140])
assert.eq(bytes([104, 101, 108, 108, 111, 44, 32, 228, 184, 150, 231, 149, 140]), hello)
assert.eq(list(goodbye.elems()), [103, 111, 111, 100, 98, 121, 101])
assert.eq(list(empty.elems()), [])
assert.eq(bytes(hello.elems()), hello) # bytes(iterable) is dual to bytes.elems()

# x[i] = ...
def f():
    b"abc"[1] = b"B"

assert.fails(f, "bytes.*does not support.*assignment")

# TODO(adonovan): the specification is not finalized in many areas:
# - chr, ord functions
# - encoding/decoding bytes to string.
# - methods: find, index, split, etc.
#
# Summary of string operations (put this in spec).
#
# string to number:
# - bytes[i]  returns numeric value of ith byte.
# - ord(string)  returns numeric value of sole code point in string.
# - ord(string[i])  is not a useful operation: fails on non-ASCII; see below.
#   Q. Perhaps ord should return the first (not sole) code point? Then it becomes a UTF-8 decoder.
#      Perhaps ord(string, index=int) should apply the index and relax the len=1 check.
# - string.codepoint()  iterates over 1-codepoint substrings.
# - string.codepoint_ords()  iterates over numeric values of code points in string.
# - string.elems()  iterates over 1-element (UTF-k code) substrings.
# - string.elem_ords()  iterates over numeric UTF-k code values.
# - string.elem_ords()[i]  returns numeric value of ith element (UTF-k code).
# - string.elems()[i]  returns substring of a single element (UTF-k code).
# - int(string)  parses string as decimal (or other) numeric literal.
#
# number to string:
# - chr(int) returns string, UTF-k encoding of Unicode code point (like Python).
#   Redundant with '%c' % int (which Python2 calls 'unichr'.)
# - bytes(chr(int)) returns byte string containing UTF-8 encoding of one code point.
# - bytes([int]) returns 1-byte string (with regrettable list allocation).
# - str(int) - format number as decimal.
