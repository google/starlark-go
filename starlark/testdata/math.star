# Tests of math module.

load('math.star', 'math')
load('assert.star', 'assert')

def near(got, want, threshold):
  return math.abs(got-want) < threshold 

inf, nan = float("inf"), float("nan")

# ceil
assert.eq(math.ceil(0.0), 0.0)
assert.eq(math.ceil(0.4), 1.0)
assert.eq(math.ceil(0.5), 1.0)
assert.eq(math.ceil(1.0), 1.0)
assert.eq(math.ceil(10.0), 10.0)
assert.eq(math.ceil(0), 0.0)
assert.eq(math.ceil(1), 1.0)
assert.eq(math.ceil(10), 10.0)
assert.eq(math.ceil(inf), inf)
assert.eq(math.ceil(nan), nan)
assert.eq(math.ceil(-0.0), 0.0)
assert.eq(math.ceil(-0.4), 0.0)
assert.eq(math.ceil(-0.5), 0.0)
assert.eq(math.ceil(-1.0), -1.0)
assert.eq(math.ceil(-10.0), -10.0)
assert.eq(math.ceil(-1), -1.0)
assert.eq(math.ceil(-10), -10.0)
assert.eq(math.ceil(-inf), -inf)
assert.fails(lambda: math.ceil("0"), "got string, want float or int")
# abs
assert.eq(math.abs(2.0), 2.0)
assert.eq(math.abs(0.0), 0.0)
assert.eq(math.abs(-2.0), 2.0)
assert.eq(math.abs(2), 2)
assert.eq(math.abs(0), 0)
assert.eq(math.abs(-2), 2)
assert.eq(math.abs(inf), inf)
assert.eq(math.abs(-inf), inf)
assert.eq(math.abs(nan), nan)
assert.fails(lambda: math.abs("0"), "got string, want float or int")
# floor
assert.eq(math.floor(0.0), 0.0)
assert.eq(math.floor(0.4), 0.0)
assert.eq(math.floor(0.5), 0.0)
assert.eq(math.floor(1.0), 1.0)
assert.eq(math.floor(10.0), 10.0)
assert.eq(math.floor(inf), inf)
assert.eq(math.floor(nan), nan)
assert.eq(math.floor(-0.0), 0.0)
assert.eq(math.floor(-0.4), -1.0)
assert.eq(math.floor(-0.5), -1.0)
assert.eq(math.floor(-1.0), -1.0)
assert.eq(math.floor(-10.0), -10.0)
assert.eq(math.floor(-inf), -inf)
assert.fails(lambda: math.floor("0"), "got string, want float or int")
# round
assert.eq(math.round(0.0), 0.0)
assert.eq(math.round(0.4), 0.0)
assert.eq(math.round(0.5), 1.0)
assert.eq(math.round(0.6), 1.0)
assert.eq(math.round(1.0), 1.0)
assert.eq(math.round(10.0), 10.0)
assert.eq(math.round(inf), inf)
assert.eq(math.round(nan), nan)
assert.eq(math.round(-0.4), 0.0)
assert.eq(math.round(-0.5), -1.0)
assert.eq(math.round(-0.6), -1.0)
assert.eq(math.round(-1.0), -1.0)
assert.eq(math.round(-10.0), -10.0)
assert.eq(math.round(-inf), -inf)
assert.fails(lambda: math.round("0"), "got string, want float or int")
# exp
assert.eq(math.exp(0.0), 1)
assert.eq(math.exp(1.0), math.e)
assert.true(near(math.exp(2.0), math.e * math.e, 0.00000000000001))
assert.eq(math.exp(-1.0), 1 / math.e)
assert.eq(math.exp(0), 1)
assert.eq(math.exp(1), math.e)
assert.true(near(math.exp(2), math.e * math.e, 0.00000000000001))
assert.eq(math.exp(-1), 1 / math.e)
assert.eq(math.exp(inf), inf)
assert.eq(math.exp(-inf), 0)
assert.eq(math.exp(nan), nan)
assert.fails(lambda: math.exp("0"), "got string, want float or int")
# sqrt
assert.eq(math.sqrt(0.0), 0.0)
assert.eq(math.sqrt(4.0), 2.0)
assert.eq(math.sqrt(-4.0), nan)
assert.eq(math.sqrt(0), 0)
assert.eq(math.sqrt(4), 2)
assert.eq(math.sqrt(-4), nan)
assert.eq(math.sqrt(nan), nan)
assert.eq(math.sqrt(inf), inf)
assert.eq(math.sqrt(-inf), nan)
assert.fails(lambda: math.sqrt("0"), "got string, want float or int")
# acos
assert.eq(math.acos(1.0), 0)
assert.eq(math.acos(1), 0)
assert.eq(math.acos(0.0), math.pi / 2)
assert.eq(math.acos(0), math.pi / 2)
assert.eq(math.acos(-1.0), math.pi)
assert.eq(math.acos(-1), math.pi)
assert.eq(math.acos(1.01), nan)
assert.eq(math.acos(-1.01), nan)
assert.eq(math.acos(inf), nan)
assert.eq(math.acos(-inf), nan)
assert.eq(math.acos(nan), nan)
assert.fails(lambda: math.acos("0"), "got string, want float or int")
# asin
assert.eq(math.asin(0.0), 0)
assert.eq(math.asin(1.0), math.pi / 2)
assert.eq(math.asin(-1.0), -math.pi / 2)
assert.eq(math.asin(0), 0)
assert.eq(math.asin(1), math.pi / 2)
assert.eq(math.asin(-1), -math.pi / 2)
assert.eq(math.asin(1.01), nan)
assert.eq(math.asin(-1.01), nan)
assert.eq(math.asin(inf), nan)
assert.eq(math.asin(-inf), nan)
assert.eq(math.asin(nan), nan)
assert.fails(lambda: math.asin("0"), "got string, want float or int")
# atan
assert.eq(math.atan(0.0), 0)
assert.eq(math.atan(1.0), math.pi / 4)
assert.eq(math.atan(-1.0), -math.pi / 4)
assert.eq(math.atan(1), math.pi / 4)
assert.eq(math.atan(-1), -math.pi / 4)
assert.eq(math.atan(inf), math.pi / 2)
assert.eq(math.atan(-inf), -math.pi / 2)
assert.eq(math.atan(nan), nan)
assert.fails(lambda: math.atan("0"), "got string, want float or int")
# atan2
assert.eq(math.atan2(1.0, 1.0), math.pi / 4)
assert.eq(math.atan2(-1.0, 1.0), -math.pi / 4)
assert.eq(math.atan2(0.0, 10.0), 0)
assert.eq(math.atan2(0.0, -10.0), math.pi)
assert.eq(math.atan2(-0.0, -10.0), -math.pi)
assert.eq(math.atan2(10.0, 0.0), math.pi / 2)
assert.eq(math.atan2(-10.0, 0.0), -math.pi / 2)
assert.eq(math.atan2(1, 1), math.pi / 4)
assert.eq(math.atan2(-1, 1), -math.pi / 4)
assert.eq(math.atan2(0, 10.0), 0)
assert.eq(math.atan2(0.0, -10), math.pi)
assert.eq(math.atan2(-0.0, -10), -math.pi)
assert.eq(math.atan2(10.0, 0), math.pi / 2)
assert.eq(math.atan2(-10.0, 0), -math.pi / 2)
assert.eq(math.atan2(1.0, nan), nan)
assert.eq(math.atan2(nan, 1.0), nan)
assert.eq(math.atan2(10.0, inf), 0)
assert.eq(math.atan2(-10.0, inf), 0)
assert.eq(math.atan2(10.0, -inf), math.pi)
assert.eq(math.atan2(-10.0, -inf), -math.pi)
assert.eq(math.atan2(inf, 10.0), math.pi / 2)
assert.eq(math.atan2(inf, -10.0), math.pi / 2)
assert.eq(math.atan2(-inf, 10.0), -math.pi / 2)
assert.eq(math.atan2(-inf, -10.0), -math.pi / 2)
assert.eq(math.atan2(inf, inf), math.pi / 4)
assert.eq(math.atan2(-inf, inf), -math.pi / 4)
assert.eq(math.atan2(inf, -inf), 3 * math.pi / 4)
assert.eq(math.atan2(-inf, -inf), -3 * math.pi / 4)
assert.fails(lambda: math.atan2("0", 1.0), "got string, want float or int")
assert.fails(lambda: math.atan2(1.0, "0"), "got string, want float or int")
# cos
assert.eq(math.cos(0.0), 1)
assert.true(near(math.cos(math.pi / 2), 0, 0.00000000000001))
assert.eq(math.cos(math.pi), -1)
assert.true(near(math.cos(-math.pi / 2), 0, 0.00000000000001))
assert.eq(math.cos(-math.pi), -1)
assert.eq(math.cos(inf), nan)
assert.eq(math.cos(-inf), nan)
assert.eq(math.cos(nan), nan)
assert.fails(lambda: math.cos("0"), "got string, want float or int")
# hypot
assert.eq(math.hypot(4.0, 3.0), 5.0)
assert.eq(math.hypot(4, 3), 5.0)
assert.eq(math.hypot(inf, 3.0), inf)
assert.eq(math.hypot(-inf, 3.0), inf)
assert.eq(math.hypot(3.0, inf), inf)
assert.eq(math.hypot(3.0, -inf), inf)
assert.eq(math.hypot(nan, 3.0), nan)
assert.eq(math.hypot(3.0, nan), nan)
assert.fails(lambda: math.hypot("0", 1.0), "got string, want float or int")
assert.fails(lambda: math.hypot(1.0, "0"), "got string, want float or int")
# sin
assert.eq(math.sin(0.0), 0)
assert.eq(math.sin(0), 0)
assert.eq(math.sin(math.pi / 2), 1)
assert.eq(math.sin(-math.pi / 2), -1)
assert.eq(math.sin(inf), nan)
assert.eq(math.sin(-inf), nan)
assert.eq(math.sin(nan), nan)
assert.fails(lambda: math.sin("0"), "got string, want float or int")
# tan
assert.eq(math.tan(0.0), 0)
assert.eq(math.tan(0), 0)
assert.eq(math.tan(math.pi / 4), 1)
assert.eq(math.tan(-math.pi / 4), -1)
assert.eq(math.tan(inf), nan)
assert.eq(math.tan(-inf), nan)
assert.eq(math.tan(nan), nan)
assert.fails(lambda: math.tan("0"), "got string, want float or int")
# degrees
oneDeg = 57.29577951308232
assert.eq(math.degrees(1.0), oneDeg)
assert.eq(math.degrees(1), oneDeg)
assert.eq(math.degrees(-1.0), -oneDeg)
assert.eq(math.degrees(-1), -oneDeg)
assert.eq(math.degrees(inf), inf)
assert.eq(math.degrees(-inf), -inf)
assert.eq(math.degrees(nan), nan)
assert.fails(lambda: math.degrees("0"), "got string, want float or int")
# radians
oneRad = 0.017453292519943295
assert.eq(math.radians(1.0), oneRad)
assert.eq(math.radians(-1.0), -oneRad)
assert.eq(math.radians(1), oneRad)
assert.eq(math.radians(-1), -oneRad)
assert.eq(math.radians(inf), inf)
assert.eq(math.radians(-inf), -inf)
assert.eq(math.radians(nan), nan)
assert.fails(lambda: math.radians("0"), "got string, want float or int")
# acosh
assert.eq(math.acosh(1.0), 0)
assert.eq(math.acosh(1), 0)
assert.eq(math.acosh(0.99), nan)
assert.eq(math.acosh(0), nan)
assert.eq(math.acosh(-0.99), nan)
assert.eq(math.acosh(-inf), nan)
assert.eq(math.acosh(inf), inf)
assert.eq(math.acosh(nan), nan)
assert.fails(lambda: math.acosh("0"), "got string, want float or int")
# asinh
asinhOne = 0.8813735870195432
assert.eq(math.asinh(0.0), 0)
assert.eq(math.asinh(0), 0)
assert.true(near(math.asinh(1.0), asinhOne, 0.00000001))
assert.true(near(math.asinh(1), asinhOne, 0.00000001))
assert.true(near(math.asinh(-1.0), -asinhOne, 0.00000001))
assert.true(near(math.asinh(-1), -asinhOne, 0.00000001))
assert.eq(math.asinh(inf), inf)
assert.eq(math.asinh(-inf), -inf)
assert.eq(math.asinh(nan), nan)
assert.fails(lambda: math.asinh("0"), "got string, want float or int")
# atanh
atanhHalf = 0.5493061443340548
assert.eq(math.atanh(0.0), 0)
assert.eq(math.atanh(0), 0)
assert.eq(math.atanh(0.5), atanhHalf)
assert.eq(math.atanh(-0.5), -atanhHalf)
assert.eq(math.atanh(1), inf)
assert.eq(math.atanh(-1), -inf)
assert.eq(math.atanh(1.1), nan)
assert.eq(math.atanh(-1.1), nan)
assert.eq(math.atanh(inf), nan)
assert.eq(math.atanh(-inf), nan)
assert.eq(math.atanh(nan), nan)
assert.fails(lambda: math.atanh("0"), "got string, want float or int")
# cosh
coshOne = 1.5430806348152437
assert.eq(math.cosh(1.0), coshOne)
assert.eq(math.cosh(1), coshOne)
assert.eq(math.cosh(0.0), 1)
assert.eq(math.cosh(0), 1)
assert.eq(math.cosh(-inf), inf)
assert.eq(math.cosh(inf), inf)
assert.eq(math.cosh(nan), nan)
assert.fails(lambda: math.cosh("0"), "got string, want float or int")
# sinh
sinhOne = 1.1752011936438014
assert.eq(math.sinh(0.0), 0)
assert.eq(math.sinh(0), 0)
assert.eq(math.sinh(1.0), sinhOne)
assert.eq(math.sinh(1), sinhOne)
assert.eq(math.sinh(-1.0), -sinhOne)
assert.eq(math.sinh(-1), -sinhOne)
assert.eq(math.sinh(-inf), -inf)
assert.eq(math.sinh(inf), inf)
assert.eq(math.sinh(nan), nan)
assert.fails(lambda: math.sinh("0"), "got string, want float or int")
# tanh
tanhOne = 0.7615941559557649
assert.eq(math.tanh(0.0), 0)
assert.eq(math.tanh(0), 0)
assert.eq(math.tanh(1.0), tanhOne)
assert.eq(math.tanh(1), tanhOne)
assert.eq(math.tanh(-1.0), -tanhOne)
assert.eq(math.tanh(-1), -tanhOne)
assert.eq(math.tanh(-inf), -1)
assert.eq(math.tanh(inf), 1)
assert.eq(math.tanh(nan), nan)
assert.fails(lambda: math.tanh("0"), "got string, want float or int")
# log
assert.eq(math.log(math.e), 1)
assert.eq(math.log(10, base=10), 1)
assert.eq(math.log(10.0, 10.0), 1)
assert.eq(math.log(2, base=2.0), 1)
assert.eq(math.log(2, base=1), inf)
assert.eq(math.log(0.99, base=1.0), -inf)
assert.eq(math.log(0.0), -inf)
assert.eq(math.log(0), -inf)
assert.eq(math.log(-1.0), nan)
assert.eq(math.log(-1), nan)
assert.eq(math.log(nan), nan)
assert.fails(lambda: math.log("0"), "got string, want float or int")
assert.fails(lambda: math.log(10, base="10"), "got string, want float or int")
assert.fails(lambda: math.log(10, "10"), "got string, want float or int")
# Constants
assert.eq(math.e, 2.7182818284590452)
assert.eq(math.pi, 3.1415926535897932)
