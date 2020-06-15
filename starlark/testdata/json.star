# Tests of json module.
# option:float

load("assert.star", "assert")
load("json.star", "json")

assert.eq(dir(json), ["decode", "encode", "indent"])

# Some of these cases were inspired by github.com/nst/JSONTestSuite.

## json.encode

assert.eq(json.encode(None), "null")
assert.eq(json.encode(True), "true")
assert.eq(json.encode(False), "false")
assert.eq(json.encode(-123), "-123")
assert.eq(json.encode(12345*12345*12345*12345*12345*12345), "3539537889086624823140625")
assert.eq(json.encode(float(12345*12345*12345*12345*12345*12345)), "3.539537889086625e+24")
assert.eq(json.encode(12.345e67), "1.2345e+68")
assert.eq(json.encode("hello"), '"hello"')
assert.eq(json.encode([1, 2, 3]), "[1,2,3]")
assert.eq(json.encode((1, 2, 3)), "[1,2,3]")
assert.eq(json.encode(range(3)), "[0,1,2]") # a built-in iterable
assert.eq(json.encode(dict(x = 1, y = "two")), '{"x":1,"y":"two"}')
assert.eq(json.encode(struct(x = 1, y = "two")), '{"x":1,"y":"two"}')  # a user-defined HasAttrs
assert.eq(json.encode("\x80"), '"\\ufffd"') # invalid UTF-8 -> replacement char

def encode_error(expr, error):
    assert.fails(lambda: json.encode(expr), error)

encode_error(float("NaN"), "json.encode: cannot encode non-finite float NaN")
encode_error({1: "two"}, "dict has int key, want string")
encode_error(len, "cannot encode builtin_function_or_method as JSON")
encode_error(struct(x=[1, {"x": len}]), # nested failure
             'in field .x: at list index 1: in dict key "x": cannot encode...')
encode_error(struct(x=[1, {"x": len}]), # nested failure
             'in field .x: at list index 1: in dict key "x": cannot encode...')
encode_error({1: 2}, 'dict has int key, want string')

## json.decode

assert.eq(json.decode("null"), None)
assert.eq(json.decode("true"), True)
assert.eq(json.decode("false"), False)
assert.eq(json.decode("-123"), -123)
assert.eq(json.decode("-0"), -0)
assert.eq(json.decode("3539537889086624823140625"), 3539537889086624823140625)
assert.eq(json.decode("3539537889086624823140625.0"), float(3539537889086624823140625))
assert.eq(json.decode("3.539537889086625e+24"), 3.539537889086625e+24)
assert.eq(json.decode("0e+1"), 0)
assert.eq(json.decode("-0.0"), -0.0)
assert.eq(json.decode(
    "-0.000000000000000000000000000000000000000000000000000000000000000000000000000001"),
    -0.000000000000000000000000000000000000000000000000000000000000000000000000000001)
assert.eq(json.decode('[]'), [])
assert.eq(json.decode('[1]'), [1])
assert.eq(json.decode('[1,2,3]'), [1, 2, 3])
assert.eq(json.decode('{"one": 1, "two": 2}'), dict(one=1, two=2))
assert.eq(json.decode('{"foo\\u0000bar": 42}'), {"foo\x00bar": 42})
assert.eq(json.decode('"\\ud83d\\ude39\\ud83d\\udc8d"'), "😹💍")
assert.eq(json.decode('"\\u0123"'), 'ģ')
assert.eq(json.decode('"\x7f"'), "\x7f")

def decode_error(expr, error):
    assert.fails(lambda: json.decode(expr), error)

decode_error('truefalse',
             "json.decode: at offset 4, unexpected character 'f' after value")

decode_error('"abc', "unclosed string literal")
decode_error('"ab\\gc"', "invalid character 'g' in string escape code")
decode_error("'abc'", "unexpected character '\\\\''")

decode_error("1.2.3", "invalid number: 1.2.3")
decode_error("+1", "unexpected character '\\+'")
decode_error("-abc", "invalid number: -")
decode_error("-", "invalid number: -")
decode_error("-00", "invalid number: -00")
decode_error("00", "invalid number: 00")
decode_error("--1", "invalid number: --1")
decode_error("-+1", "invalid number: -\\+1")
decode_error("1e1e1", "invalid number: 1e1e1")
decode_error("0123", "invalid number: 0123")
decode_error("000.123", "invalid number: 000.123")
decode_error("-0123", "invalid number: -0123")
decode_error("-000.123", "invalid number: -000.123")
decode_error("0x123", "unexpected character 'x' after value")

decode_error('[1, 2 ', "unexpected end of file")
decode_error('[1, 2, ', "unexpected end of file")
decode_error('[1, 2, ]', "unexpected character ']'")
decode_error('[1, 2, }', "unexpected character '}'")
decode_error('[1, 2}', "got '}', want ',' or ']'")

decode_error('{"one": 1', "unexpected end of file")
decode_error('{"one" 1', "after object key, got '1', want ':'")
decode_error('{"one": 1 "two": 2', "in object, got '\"', want ',' or '}'")
decode_error('{"one": 1,', "unexpected end of file")
decode_error('{"one": 1, }', "unexpected character '}'")
decode_error('{"one": 1]', "in object, got ']', want ',' or '}'")

def codec(x):
    return json.decode(json.encode(x))

# string round-tripping
strings = [
    "😿", # U+1F63F CRYING_CAT_FACE
    "🐱‍👤", # CAT FACE + ZERO WIDTH JOINER + BUST IN SILHOUETTE
]
assert.eq(codec(strings), strings)

# codepoints is a string with every 16-bit code point.
codepoints = ''.join(['%c' % c for c in range(65536)])
assert.eq(codec(codepoints), codepoints)

# number round-tripping
numbers = [
    0, 1, -1, +1, 1.23e45, -1.23e-45,
    3539537889086624823140625,
    float(3539537889086624823140625),
]
assert.eq(codec(numbers), numbers)

## json.indent

s = json.encode(dict(x = 1, y = ["one", "two"]))

assert.eq(json.indent(s), '''{
	"x": 1,
	"y": [
		"one",
		"two"
	]
}''')

assert.eq(json.decode(json.indent(s)), {"x": 1, "y": ["one", "two"]})

assert.eq(json.indent(s, prefix='¶', indent='–––'), '''{
¶–––"x": 1,
¶–––"y": [
¶––––––"one",
¶––––––"two"
¶–––]
¶}''')

assert.fails(lambda: json.indent("!@#$%^& this is not json"), 'invalid character')
---
