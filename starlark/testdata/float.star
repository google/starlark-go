# Tests of Starlark 'float'
# option:float option:set

load("assert.star", "assert")

# TODO(adonovan): more tests:
# - precision
# - limits

# literals
assert.eq(type(1.234), "float")
assert.eq(type(1e10), "float")
assert.eq(type(1e+10), "float")
assert.eq(type(1e-10), "float")
assert.eq(type(1.234e10), "float")
assert.eq(type(1.234e+10), "float")
assert.eq(type(1.234e-10), "float")

# truth
assert.true(123.0)
assert.true(-1.0)
assert.true(not 0.0)

# addition
assert.eq(0.0 + 1.0, 1.0)
assert.eq(1.0 + 1.0, 2.0)
assert.eq(1.25 + 2.75, 4.0)
assert.eq(5.0 + 7.0, 12.0)
assert.eq(5.1 + 7, 12.1)  # float + int
assert.eq(7 + 5.1, 12.1)  # int + float

# subtraction
assert.eq(5.0 - 7.0, -2.0)
assert.eq(5.1 - 7.1, -2.0)
assert.eq(5.5 - 7, -1.5)
assert.eq(5 - 7.5, -2.5)
assert.eq(0.0 - 1.0, -1.0)

# multiplication
assert.eq(5.0 * 7.0, 35.0)
assert.eq(5.5 * 2.5, 13.75)
assert.eq(5.5 * 7, 38.5)
assert.eq(5 * 7.1, 35.5)

# real division (like Python 3)
# The / operator is available only when the 'fp' dialect option is enabled.
assert.eq(100.0 / 8.0, 12.5)
assert.eq(100.0 / -8.0, -12.5)
assert.eq(-100.0 / 8.0, -12.5)
assert.eq(-100.0 / -8.0, 12.5)
assert.eq(98.0 / 8.0, 12.25)
assert.eq(98.0 / -8.0, -12.25)
assert.eq(-98.0 / 8.0, -12.25)
assert.eq(-98.0 / -8.0, 12.25)
assert.eq(2.5 / 2.0, 1.25)
assert.eq(2.5 / 2, 1.25)
assert.eq(5 / 4.0, 1.25)
assert.eq(5 / 4, 1.25)
assert.fails(lambda: 1.0 / 0, "real division by zero")
assert.fails(lambda: 1.0 / 0.0, "real division by zero")
assert.fails(lambda: 1 / 0.0, "real division by zero")

# floored division
assert.eq(100.0 // 8.0, 12.0)
assert.eq(100.0 // -8.0, -13.0)
assert.eq(-100.0 // 8.0, -13.0)
assert.eq(-100.0 // -8.0, 12.0)
assert.eq(98.0 // 8.0, 12.0)
assert.eq(98.0 // -8.0, -13.0)
assert.eq(-98.0 // 8.0, -13.0)
assert.eq(-98.0 // -8.0, 12.0)
assert.eq(2.5 // 2.0, 1.0)
assert.eq(2.5 // 2, 1.0)
assert.eq(5 // 4.0, 1.0)
assert.eq(5 // 4, 1)
assert.eq(type(5 // 4), "int")
assert.fails(lambda: 1.0 // 0, "floored division by zero")
assert.fails(lambda: 1.0 // 0.0, "floored division by zero")
assert.fails(lambda: 1 // 0.0, "floored division by zero")

# remainder
assert.eq(100.0 % 8.0, 4.0)
assert.eq(100.0 % -8.0, 4.0)
assert.eq(-100.0 % 8.0, -4.0)
assert.eq(-100.0 % -8.0, -4.0)
assert.eq(98.0 % 8.0, 2.0)
assert.eq(98.0 % -8.0, 2.0)
assert.eq(-98.0 % 8.0, -2.0)
assert.eq(-98.0 % -8.0, -2.0)
assert.eq(2.5 % 2.0, 0.5)
assert.eq(2.5 % 2, 0.5)
assert.eq(5 % 4.0, 1.0)
assert.fails(lambda: 1.0 % 0, "float modulo by zero")
assert.fails(lambda: 1.0 % 0.0, "float modulo by zero")
assert.fails(lambda: 1 % 0.0, "float modulo by zero")

# floats cannot be used as indices, even if integral
assert.fails(lambda: "abc"[1.0], "want int")
assert.fails(lambda: ["A", "B", "C"].insert(1.0, "D"), "want int")

# nan
nan = float("NaN")
def isnan(x): return x != x
assert.true(nan != nan)
assert.true(not (nan == nan))

# ordered comparisons with NaN
assert.true(not nan < nan)
assert.true(not nan > nan)
assert.true(not nan <= nan)
assert.true(not nan >= nan)
assert.true(not nan == nan) # use explicit operator, not assert.ne
assert.true(nan != nan)
assert.true(not nan < 0)
assert.true(not nan > 0)
assert.true(not [nan] < [nan])
assert.true(not [nan] > [nan])

# Even a value containing NaN is not equal to itself.
nanlist = [nan]
assert.true(not nanlist < nanlist)
assert.true(not nanlist > nanlist)
assert.ne(nanlist, nanlist)

# Since NaN values never compare equal,
# a dict may have any number of NaN keys.
nandict = {nan: 1, nan: 2, nan: 3}
assert.eq(len(nandict), 3)
assert.eq(str(nandict), "{NaN: 1, NaN: 2, NaN: 3}")
assert.true(nan not in nandict)
assert.eq(nandict.get(nan, None), None)

# inf
inf = float("Inf")
neginf = float("-Inf")
assert.true(isnan(+inf / +inf))
assert.true(isnan(+inf / -inf))
assert.true(isnan(-inf / +inf))
assert.eq(0.0 / +inf, 0.0)
assert.eq(0.0 / -inf, 0.0)
assert.true(inf > -inf)
assert.eq(inf, -neginf)
assert.eq(float(int("2" + "0" * 308)), inf) # 2e308 is too large to represent as a float
assert.eq(float(int("-2" + "0" * 308)), -inf)
# TODO(adonovan): assert inf > any finite number, etc.

# negative zero
negz = -0
assert.eq(negz, 0)

# float/float comparisons
fltmax = 1.7976931348623157e+308 # approx
fltmin = 4.9406564584124654e-324 # approx
assert.lt(-inf, -fltmax)
assert.lt(-fltmax, -1.0)
assert.lt(-1.0, -fltmin)
assert.lt(-fltmin, 0.0)
assert.lt(0, fltmin)
assert.lt(fltmin, 1.0)
assert.lt(1.0, fltmax)
assert.lt(fltmax, inf)

# int/float comparisons
assert.eq(0, 0.0)
assert.eq(1, 1.0)
assert.eq(-1, -1.0)
assert.ne(-1, -1.0 + 1e-7)
assert.lt(-2, -2 + 1e-15)

# int conversion (rounds towards zero)
assert.eq(int(100.1), 100)
assert.eq(int(100.0), 100)
assert.eq(int(99.9), 99)
assert.eq(int(-99.9), -99)
assert.eq(int(-100.0), -100)
assert.eq(int(-100.1), -100)
assert.eq(int(1e100), int("10000000000000000159028911097599180468360808563945281389781327557747838772170381060813469985856815104"))
assert.fails(lambda: int(inf), "cannot convert.*infinity")
assert.fails(lambda: int(nan), "cannot convert.*NaN")

# float conversion
assert.eq(float(), 0.0)
assert.eq(float(False), 0.0)
assert.eq(float(True), 1.0)
assert.eq(float(0), 0.0)
assert.eq(float(1), 1.0)
assert.eq(float(1.1), 1.1)
assert.eq(float("1.1"), 1.1)
assert.fails(lambda: float("1.1abc"), "invalid syntax")
assert.fails(lambda: float("1e100.0"), "invalid syntax")
assert.fails(lambda: float("1e1000"), "out of range")
assert.fails(lambda: float(None), "want number or string")
assert.eq(float("-1.1"), -1.1)
assert.eq(float("+1.1"), +1.1)
assert.eq(float("+Inf"), inf)
assert.eq(float("-Inf"), neginf)
assert.true(isnan(float("NaN")))
assert.fails(lambda: float("+NaN"), "invalid syntax")
assert.fails(lambda: float("-NaN"), "invalid syntax")

# hash
# Check that equal float and int values have the same internal hash.
def checkhash():
  for a in [1.23e100, 1.23e10, 1.23e1, 1.23,
            1, 4294967295, 8589934591, 9223372036854775807]:
    for b in [a, -a, 1/a, -1/a]:
      f = float(b)
      i = int(b)
      if f == i:
        fh = {f: None}
        ih = {i: None}
        if fh != ih:
          assert.true(False, "{%v: None} != {%v: None}: hashes vary" % fh, ih)
checkhash()

# string formatting
assert.eq("%s" % 123.45e67, "1.2345e+69")
assert.eq("%r" % 123.45e67, "1.2345e+69")
assert.eq("%e" % 123.45e67, "1.234500e+69")
assert.eq("%f" % 123.45e67, "1234500000000000033987094856609369647752433474509923447907937257783296.000000")
assert.eq("%g" % 123.45e67, "1.2345e+69")
assert.eq("%e" % 123, "1.230000e+02")
assert.eq("%f" % 123, "123.000000")
assert.eq("%g" % 123, "123")
assert.fails(lambda: "%e" % "123", "requires float, not str")
assert.fails(lambda: "%f" % "123", "requires float, not str")
assert.fails(lambda: "%g" % "123", "requires float, not str")

i0 = 1
f0 = 1.0
assert.eq(type(i0), "int")
assert.eq(type(f0), "float")

ops = {
    '+': lambda x, y: x + y,
    '-': lambda x, y: x - y,
    '*': lambda x, y: x * y,
    '/': lambda x, y: x / y,
    '//': lambda x, y: x // y,
    '%': lambda x, y: x % y,
}

# Check that if either argument is a float, so too is the result.
def checktypes():
  want = set("""
int + int = int
int + float = float
float + int = float
float + float = float
int - int = int
int - float = float
float - int = float
float - float = float
int * int = int
int * float = float
float * int = float
float * float = float
int / int = float
int / float = float
float / int = float
float / float = float
int // int = int
int // float = float
float // int = float
float // float = float
int % int = int
int % float = float
float % int = float
float % float = float
"""[1:].splitlines())
  for opname in ("+", "-", "*", "/", "%"):
    for x in [i0, f0]:
      for y in [i0, f0]:
        op = ops[opname]
        got = "%s %s %s = %s" % (type(x), opname, type(y), type(op(x, y)))
        assert.contains(want, got)
checktypes()
