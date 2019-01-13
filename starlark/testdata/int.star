# Tests of Starlark 'int'
# option:bitwise option:float

load("assert.star", "assert")

# basic arithmetic
assert.eq(0 - 1, -1)
assert.eq(0 + 1, +1)
assert.eq(1 + 1, 2)
assert.eq(5 + 7, 12)
assert.eq(5 * 7, 35)
assert.eq(5 - 7, -2)

# truth
assert.true(123)
assert.true(-1)
assert.true(not 0)

# floored division
# (For real division, see float.star.)
assert.eq(100 // 7, 14)
assert.eq(100 // -7, -15)
assert.eq(-100 // 7, -15) # NB: different from Go/Java
assert.eq(-100 // -7, 14) # NB: different from Go/Java
assert.eq(98 // 7, 14)
assert.eq(98 // -7, -14)
assert.eq(-98 // 7, -14)
assert.eq(-98 // -7, 14)

# remainder
assert.eq(100 % 7, 2)
assert.eq(100 % -7, -5) # NB: different from Go/Java
assert.eq(-100 % 7, 5) # NB: different from Go/Java
assert.eq(-100 % -7, -2)
assert.eq(98 % 7, 0)
assert.eq(98 % -7, 0)
assert.eq(-98 % 7, 0)
assert.eq(-98 % -7, 0)

# compound assignment
def compound():
  x = 1
  x += 1
  assert.eq(x, 2)
  x -= 3
  assert.eq(x, -1)
  x *= 39
  assert.eq(x, -39)
  x //= 4
  assert.eq(x, -10)
  x /= -2
  assert.eq(x, 5)
  x %= 3
  assert.eq(x, 2)
  # use resolve.AllowBitwise to enable the ops:
  x = 2
  x &= 1
  assert.eq(x, 0)
  x |= 2
  assert.eq(x, 2)
  x ^= 3
  assert.eq(x, 1)
  x <<= 2
  assert.eq(x, 4)
  x >>=2
  assert.eq(x, 1)

compound()

# int conversion
# See float.star for float-to-int conversions.
# We follow Python 3 here, but I can't see the method in its madness.
# int from bool/int/float
assert.fails(int, 'missing argument')  # int()
assert.eq(int(False), 0)
assert.eq(int(True), 1)
assert.eq(int(3), 3)
assert.eq(int(3.1), 3)
assert.fails(lambda: int(3, base=10), "non-string with explicit base")
assert.fails(lambda: int(True, 10), "non-string with explicit base")
# int from string, base implicitly 10
assert.eq(int("100000000000000000000"), 10000000000 * 10000000000)
assert.eq(int("-100000000000000000000"), -10000000000 * 10000000000)
assert.eq(int("123"), 123)
assert.eq(int("-123"), -123)
assert.eq(int("0123"), 123) # not octal
assert.eq(int("-0123"), -123)
assert.fails(lambda: int("0x12"), "invalid literal with base 10")
assert.fails(lambda: int("-0x12"), "invalid literal with base 10")
assert.fails(lambda: int("0o123"), "invalid literal.*base 10")
assert.fails(lambda: int("-0o123"), "invalid literal.*base 10")
# int from string, explicit base
assert.eq(int("0"), 0)
assert.eq(int("00"), 0)
assert.eq(int("0", base=10), 0)
assert.eq(int("00", base=10), 0)
assert.eq(int("0", base=8), 0)
assert.eq(int("00", base=8), 0)
assert.eq(int("-0"), 0)
assert.eq(int("-00"), 0)
assert.eq(int("-0", base=10), 0)
assert.eq(int("-00", base=10), 0)
assert.eq(int("-0", base=8), 0)
assert.eq(int("-00", base=8), 0)
assert.eq(int("+0"), 0)
assert.eq(int("+00"), 0)
assert.eq(int("+0", base=10), 0)
assert.eq(int("+00", base=10), 0)
assert.eq(int("+0", base=8), 0)
assert.eq(int("+00", base=8), 0)
assert.eq(int("11", base=9), 10)
assert.eq(int("-11", base=9), -10)
assert.eq(int("10011", base=2), 19)
assert.eq(int("-10011", base=2), -19)
assert.eq(int("123", 8), 83)
assert.eq(int("-123", 8), -83)
assert.eq(int("0123", 8), 83) # redundant zeros permitted
assert.eq(int("-0123", 8), -83)
assert.eq(int("00123", 8), 83)
assert.eq(int("-00123", 8), -83)
assert.eq(int("0o123", 8), 83)
assert.eq(int("-0o123", 8), -83)
assert.eq(int("123", 7), 66) # 1*7*7 + 2*7 + 3
assert.eq(int("-123", 7), -66)
assert.eq(int("12", 16), 18)
assert.eq(int("-12", 16), -18)
assert.eq(int("0x12", 16), 18)
assert.eq(int("-0x12", 16), -18)
assert.eq(0x1000000000000001 * 0x1000000000000001, 0x1000000000000002000000000000001)
assert.eq(int("1010", 2), 10)
assert.eq(int("111111101", 2), 509)
assert.eq(int("0b0101", 0), 5)
assert.eq(int("0b00000", 0), 0)
assert.eq(1111111111111111 * 1111111111111111, 1234567901234567654320987654321)
assert.fails(lambda: int("0x123", 8), "invalid literal.*base 8")
assert.fails(lambda: int("-0x123", 8), "invalid literal.*base 8")
assert.fails(lambda: int("0o123", 16), "invalid literal.*base 16")
assert.fails(lambda: int("-0o123", 16), "invalid literal.*base 16")
assert.fails(lambda: int("0x110", 2), "invalid literal.*base 2")
# int from string, auto detect base
assert.eq(int("123", 0), 123)
assert.eq(int("+123", 0), +123)
assert.eq(int("-123", 0), -123)
assert.eq(int("0x12", 0), 18)
assert.eq(int("+0x12", 0), +18)
assert.eq(int("-0x12", 0), -18)
assert.eq(int("0o123", 0), 83)
assert.eq(int("+0o123", 0), +83)
assert.eq(int("-0o123", 0), -83)
assert.fails(lambda: int("0123", 0), "invalid literal.*base 0") # valid in Python 2.7
assert.fails(lambda: int("-0123", 0), "invalid literal.*base 0")
# github.com/google/starlark-go/issues/108
assert.fails(lambda: int("0Oxa", 8), "invalid literal with base 8: 0Oxa")
# follow-on bugs to issue 108
assert.fails(lambda: int("--4"), "invalid literal with base 10: --4")
assert.fails(lambda: int("++4"), "invalid literal with base 10: \+\+4")
assert.fails(lambda: int("+-4"), "invalid literal with base 10: \+-4")
assert.fails(lambda: int("0x-4", 16), "invalid literal with base 16: 0x-4")

# bitwise union (int|int), intersection (int&int), XOR (int^int), unary not (~int),
# left shift (int<<int), and right shift (int>>int).
# use resolve.AllowBitwise to enable the ops.
# TODO(adonovan): this is not yet in the Starlark spec,
# but there is consensus that it should be.
assert.eq(1|2, 3)
assert.eq(3|6, 7)
assert.eq((1|2) & (2|4), 2)
assert.eq(1 ^ 2, 3)
assert.eq(2 ^ 2, 0)
assert.eq(1 | 0 ^ 1, 1) # check | and ^ operators precedence
assert.eq(~1, -2)
assert.eq(~-2, 1)
assert.eq(~0, -1)
assert.eq(1 << 2, 4)
assert.eq(2 >> 1, 1)
assert.fails(lambda: 2 << -1, "negative shift count")
assert.fails(lambda: 1 << 512, "shift count too large")

# comparisons
# TODO(adonovan): test: < > == != etc
assert.lt(-2, -1)
assert.lt(-1, 0)
assert.lt(0, 1)
assert.lt(1, 2)
assert.true(2 >= 2)
assert.true(2 > 1)
assert.true(1 >= 1)
assert.true(1 > 0)
assert.true(0 >= 0)
assert.true(0 > -1)
assert.true(-1 >= -1)
assert.true(-1 > -2)

# precision
maxint64 = 9223372036854775807 # = 2^63
minint64 = -maxint64 - 1       # = -2^64
assert.eq(str(maxint64), "9223372036854775807")
assert.eq(str(maxint64+1), "9223372036854775808")
assert.eq(str(minint64), "-9223372036854775808")
assert.eq(str(minint64-1), "-9223372036854775809")
assert.eq(str(minint64 * minint64), "85070591730234615865843651857942052864")

# string formatting
assert.eq("%o %x %d" % (0o755, 0xDEADBEEF, 42), "755 deadbeef 42")
nums = [-95, -1, 0, +1, +95]
assert.eq(' '.join(["%o" % x for x in nums]), "-137 -1 0 1 137")
assert.eq(' '.join(["%d" % x for x in nums]), "-95 -1 0 1 95")
assert.eq(' '.join(["%i" % x for x in nums]), "-95 -1 0 1 95")
assert.eq(' '.join(["%x" % x for x in nums]), "-5f -1 0 1 5f")
assert.eq(' '.join(["%X" % x for x in nums]), "-5F -1 0 1 5F")
assert.eq("%o %x %d" % (123, 123, 123), "173 7b 123")
assert.eq("%o %x %d" % (123.1, 123.1, 123.1), "173 7b 123") # non-int operands are acceptable
assert.fails(lambda: "%d" % True, "cannot convert bool to int")
