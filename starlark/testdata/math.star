# Tests of math module.

load('math.star', 'math')
load('assert.star', 'assert')

def near(got, want, threshold):
  return math.abs(got-want) < threshold 

# ceil
assert.eq(math.ceil(0.0), 0.0)
assert.eq(math.ceil(0.4), 1.0)
assert.eq(math.ceil(0.5), 1.0)
assert.eq(math.ceil(1.0), 1.0)
assert.eq(math.ceil(10.0), 10.0)
assert.eq(math.ceil(0), 0.0)
assert.eq(math.ceil(1), 1.0)
assert.eq(math.ceil(10), 10.0)
assert.eq(math.ceil(float("inf")), float("inf"))
assert.eq(math.ceil(float("nan")), float("nan"))
assert.eq(math.ceil(-0.0), 0.0)
assert.eq(math.ceil(-0.4), 0.0)
assert.eq(math.ceil(-0.5), 0.0)
assert.eq(math.ceil(-1.0), -1.0)
assert.eq(math.ceil(-10.0), -10.0)
assert.eq(math.ceil(-1), -1.0)
assert.eq(math.ceil(-10), -10.0)
assert.eq(math.ceil(float("-inf")), float("-inf"))
assert.fails(lambda: math.ceil("0"), "got string, want float")
# abs
assert.eq(math.abs(2.0), 2.0)
assert.eq(math.abs(0.0), 0.0)
assert.eq(math.abs(-2.0), 2.0)
assert.eq(math.abs(2), 2)
assert.eq(math.abs(0), 0)
assert.eq(math.abs(-2), 2)
assert.eq(math.abs(float("inf")), float("inf"))
assert.eq(math.abs(float("-inf")), float("inf"))
assert.eq(math.abs(float("nan")), float("nan"))
assert.fails(lambda: math.abs("0"), "got string, want float")
# floor
assert.eq(math.floor(0.0), 0.0)
assert.eq(math.floor(0.4), 0.0)
assert.eq(math.floor(0.5), 0.0)
assert.eq(math.floor(1.0), 1.0)
assert.eq(math.floor(10.0), 10.0)
assert.eq(math.floor(float("inf")), float("inf"))
assert.eq(math.floor(float("nan")), float("nan"))
assert.eq(math.floor(-0.0), 0.0)
assert.eq(math.floor(-0.4), -1.0)
assert.eq(math.floor(-0.5), -1.0)
assert.eq(math.floor(-1.0), -1.0)
assert.eq(math.floor(-10.0), -10.0)
assert.eq(math.floor(float("-inf")), float("-inf"))
assert.fails(lambda: math.floor("0"), "got string, want float")
# round
assert.eq(math.round(0.0), 0.0)
assert.eq(math.round(0.4), 0.0)
assert.eq(math.round(0.5), 1.0)
assert.eq(math.round(0.6), 1.0)
assert.eq(math.round(1.0), 1.0)
assert.eq(math.round(10.0), 10.0)
assert.eq(math.round(float("inf")), float("inf"))
assert.eq(math.round(float("nan")), float("nan"))
assert.eq(math.round(-0.4), 0.0)
assert.eq(math.round(-0.5), -1.0)
assert.eq(math.round(-0.6), -1.0)
assert.eq(math.round(-1.0), -1.0)
assert.eq(math.round(-10.0), -10.0)
assert.eq(math.round(float("-inf")), float("-inf"))
assert.fails(lambda: math.round("0"), "got string, want float")
# exp
assert.eq(math.exp(0.0), 1)
assert.eq(math.exp(1.0), math.e)
assert.true(near(math.exp(2.0), math.e * math.e, 0.00000000000001))
assert.eq(math.exp(-1.0), 1 / math.e)
assert.eq(math.exp(0), 1)
assert.eq(math.exp(1), math.e)
assert.true(near(math.exp(2), math.e * math.e, 0.00000000000001))
assert.eq(math.exp(-1), 1 / math.e)
assert.eq(math.exp(float("inf")), float("inf"))
assert.eq(math.exp(float("-inf")), 0)
assert.eq(math.exp(float("nan")), float("nan"))
assert.fails(lambda: math.exp("0"), "got string, want float")
# sqrt
assert.eq(math.sqrt(0.0), 0.0)
assert.eq(math.sqrt(4.0), 2.0)
assert.eq(math.sqrt(-4.0), float("nan"))
assert.eq(math.sqrt(0), 0)
assert.eq(math.sqrt(4), 2)
assert.eq(math.sqrt(-4), float("nan"))
assert.eq(math.sqrt(float("nan")), float("nan"))
assert.eq(math.sqrt(float("inf")), float("inf"))
assert.eq(math.sqrt(float("-inf")), float("nan"))
assert.fails(lambda: math.sqrt("0"), "got string, want float")
# acos
assert.eq(math.acos(1.0), 0)
assert.eq(math.acos(1), 0)
assert.eq(math.acos(0.0), math.pi / 2)
assert.eq(math.acos(0), math.pi / 2)
assert.eq(math.acos(-1.0), math.pi)
assert.eq(math.acos(-1), math.pi)
assert.eq(math.acos(1.01), float("nan"))
assert.eq(math.acos(-1.01), float("nan"))
assert.eq(math.acos(float("inf")), float("nan"))
assert.eq(math.acos(float("-inf")), float("nan"))
assert.eq(math.acos(float("nan")), float("nan"))
assert.fails(lambda: math.acos("0"), "got string, want float")
# asin
assert.eq(math.asin(0.0), 0)
assert.eq(math.asin(1.0), math.pi / 2)
assert.eq(math.asin(-1.0), -math.pi / 2)
assert.eq(math.asin(0), 0)
assert.eq(math.asin(1), math.pi / 2)
assert.eq(math.asin(-1), -math.pi / 2)
assert.eq(math.asin(1.01), float("nan"))
assert.eq(math.asin(-1.01), float("nan"))
assert.eq(math.asin(float("inf")), float("nan"))
assert.eq(math.asin(float("-inf")), float("nan"))
assert.eq(math.asin(float("nan")), float("nan"))
assert.fails(lambda: math.asin("0"), "got string, want float")
# atan
assert.eq(math.atan(0.0), 0)
assert.eq(math.atan(1.0), math.pi / 4)
assert.eq(math.atan(-1.0), -math.pi / 4)
assert.eq(math.atan(1), math.pi / 4)
assert.eq(math.atan(-1), -math.pi / 4)
assert.eq(math.atan(float("inf")), math.pi / 2)
assert.eq(math.atan(float("-inf")), -math.pi / 2)
assert.eq(math.atan(float("nan")), float("nan"))
assert.fails(lambda: math.atan("0"), "got string, want float")
# atan2
assert.eq(math.atan2(1.0, 1.0), math.pi / 4)
assert.eq(math.atan2(-1.0, 1.0), -math.pi / 4)
assert.eq(math.atan2(0.0, 10.0), 0)
assert.eq(math.atan2(0.0, -10.0), math.pi)
assert.eq(math.atan2(-0.0, -10.0), -math.pi)
assert.eq(math.atan2(10.0, 0.0), math.pi / 2)
assert.eq(math.atan2(-10.0, 0.0), -math.pi / 2)
assert.eq(math.atan2(1.0, float("nan")), float("nan"))
assert.eq(math.atan2(float("nan"), 1.0), float("nan"))
assert.eq(math.atan2(10.0, float("inf")), 0)
assert.eq(math.atan2(-10.0, float("inf")), 0)
assert.eq(math.atan2(10.0, float("-inf")), math.pi)
assert.eq(math.atan2(-10.0, float("-inf")), -math.pi)
assert.eq(math.atan2(float("inf"), 10.0), math.pi / 2)
assert.eq(math.atan2(float("inf"), -10.0), math.pi / 2)
assert.eq(math.atan2(float("-inf"), 10.0), -math.pi / 2)
assert.eq(math.atan2(float("-inf"), -10.0), -math.pi / 2)
assert.eq(math.atan2(float("inf"), float("inf")), math.pi / 4)
assert.eq(math.atan2(float("-inf"), float("inf")), -math.pi / 4)
assert.eq(math.atan2(float("inf"), float("-inf")), 3 * math.pi / 4)
assert.eq(math.atan2(float("-inf"), float("-inf")), -3 * math.pi / 4)
assert.fails(lambda: math.atan2("0", 1.0), "got string, want float")
assert.fails(lambda: math.atan2(1.0, "0"), "got string, want float")
# cos
assert.eq(math.cos(0.0), 1)
assert.true(near(math.cos(math.pi / 2), 0, 0.00000000000001))
assert.eq(math.cos(math.pi), -1)
assert.true(near(math.cos(-math.pi / 2), 0, 0.00000000000001))
assert.eq(math.cos(-math.pi), -1)
assert.eq(math.cos(float("inf")), float("nan"))
assert.eq(math.cos(float("-inf")), float("nan"))
assert.eq(math.cos(float("nan")), float("nan"))
assert.fails(lambda: math.cos("0"), "got string, want float")
# hypot
assert.eq(math.hypot(4.0, 3.0), 5.0)
assert.eq(math.hypot(float("inf"), 3.0), float("inf"))
assert.eq(math.hypot(float("-inf"), 3.0), float("inf"))
assert.eq(math.hypot(3.0, float("inf")), float("inf"))
assert.eq(math.hypot(3.0, float("-inf")), float("inf"))
assert.eq(math.hypot(float("nan"), 3.0), float("nan"))
assert.eq(math.hypot(3.0, float("nan")), float("nan"))
assert.fails(lambda: math.hypot("0", 1.0), "got string, want float")
assert.fails(lambda: math.hypot(1.0, "0"), "got string, want float")
# sin
assert.eq(math.sin(0.0), 0)
assert.eq(math.sin(0), 0)
assert.eq(math.sin(math.pi / 2), 1)
assert.eq(math.sin(-math.pi / 2), -1)
assert.eq(math.sin(float("inf")), float("nan"))
assert.eq(math.sin(float("-inf")), float("nan"))
assert.eq(math.sin(float("nan")), float("nan"))
assert.fails(lambda: math.sin("0"), "got string, want float")
# tan
assert.eq(math.tan(0.0), 0)
assert.eq(math.tan(0), 0)
assert.eq(math.tan(math.pi / 4), 1)
assert.eq(math.tan(-math.pi / 4), -1)
assert.eq(math.tan(float("inf")), float("nan"))
assert.eq(math.tan(float("-inf")), float("nan"))
assert.eq(math.tan(float("nan")), float("nan"))
assert.fails(lambda: math.tan("0"), "got string, want float")
# degrees
oneDeg = 57.29577951308232
assert.eq(math.degrees(1.0), oneDeg)
assert.eq(math.degrees(1), oneDeg)
assert.eq(math.degrees(-1.0), -oneDeg)
assert.eq(math.degrees(-1), -oneDeg)
assert.eq(math.degrees(float("inf")), float("inf"))
assert.eq(math.degrees(float("-inf")), float("-inf"))
assert.eq(math.degrees(float("nan")), float("nan"))
assert.fails(lambda: math.degrees("0"), "got string, want float")
# radians
oneRad = 0.017453292519943295
assert.eq(math.radians(1.0), oneRad)
assert.eq(math.radians(-1.0), -oneRad)
assert.eq(math.radians(1), oneRad)
assert.eq(math.radians(-1), -oneRad)
assert.eq(math.radians(float("inf")), float("inf"))
assert.eq(math.radians(float("-inf")), float("-inf"))
assert.eq(math.radians(float("nan")), float("nan"))
assert.fails(lambda: math.radians("0"), "got string, want float")
# acosh
assert.eq(math.acosh(1.0), 0)
assert.eq(math.acosh(1), 0)
assert.eq(math.acosh(0.99), float("nan"))
assert.eq(math.acosh(0), float("nan"))
assert.eq(math.acosh(-0.99), float("nan"))
assert.eq(math.acosh(float("-inf")), float("nan"))
assert.eq(math.acosh(float("inf")), float("inf"))
assert.eq(math.acosh(float("nan")), float("nan"))
assert.fails(lambda: math.acosh("0"), "got string, want float")
# asinh
asinhOne = 0.8813735870195432
assert.eq(math.asinh(0.0), 0)
assert.eq(math.asinh(0), 0)
assert.true(near(math.asinh(1.0), asinhOne, 0.00000001))
assert.true(near(math.asinh(1), asinhOne, 0.00000001))
assert.true(near(math.asinh(-1.0), -asinhOne, 0.00000001))
assert.true(near(math.asinh(-1), -asinhOne, 0.00000001))
assert.eq(math.asinh(float("inf")), float("inf"))
assert.eq(math.asinh(float("-inf")), float("-inf"))
assert.eq(math.asinh(float("nan")), float("nan"))
assert.fails(lambda: math.asinh("0"), "got string, want float")
# atanh
atanhHalf = 0.5493061443340548
assert.eq(math.atanh(0.0), 0)
assert.eq(math.atanh(0), 0)
assert.eq(math.atanh(0.5), atanhHalf)
assert.eq(math.atanh(-0.5), -atanhHalf)
assert.eq(math.atanh(1), float("inf"))
assert.eq(math.atanh(-1), float("-inf"))
assert.eq(math.atanh(1.1), float("nan"))
assert.eq(math.atanh(-1.1), float("nan"))
assert.eq(math.atanh(float("inf")), float("nan"))
assert.eq(math.atanh(float("-inf")), float("nan"))
assert.eq(math.atanh(float("nan")), float("nan"))
assert.fails(lambda: math.atanh("0"), "got string, want float")
# cosh
coshOne = 1.5430806348152437
assert.eq(math.cosh(1.0), coshOne)
assert.eq(math.cosh(1), coshOne)
assert.eq(math.cosh(0.0), 1)
assert.eq(math.cosh(0), 1)
assert.eq(math.cosh(float("-inf")), float("inf"))
assert.eq(math.cosh(float("inf")), float("inf"))
assert.eq(math.cosh(float("nan")), float("nan"))
assert.fails(lambda: math.cosh("0"), "got string, want float")
# sinh
sinhOne = 1.1752011936438014
assert.eq(math.sinh(0.0), 0)
assert.eq(math.sinh(0), 0)
assert.eq(math.sinh(1.0), sinhOne)
assert.eq(math.sinh(1), sinhOne)
assert.eq(math.sinh(-1.0), -sinhOne)
assert.eq(math.sinh(-1), -sinhOne)
assert.eq(math.sinh(float("-inf")), float("-inf"))
assert.eq(math.sinh(float("inf")), float("inf"))
assert.eq(math.sinh(float("nan")), float("nan"))
assert.fails(lambda: math.sinh("0"), "got string, want float")
# tanh
tanhOne = 0.7615941559557649
assert.eq(math.tanh(0.0), 0)
assert.eq(math.tanh(0), 0)
assert.eq(math.tanh(1.0), tanhOne)
assert.eq(math.tanh(1), tanhOne)
assert.eq(math.tanh(-1.0), -tanhOne)
assert.eq(math.tanh(-1), -tanhOne)
assert.eq(math.tanh(float("-inf")), -1)
assert.eq(math.tanh(float("inf")), 1)
assert.eq(math.tanh(float("nan")), float("nan"))
assert.fails(lambda: math.tanh("0"), "got string, want float")

assert.eq(math.e, 2.7182818284590452)
assert.eq(math.pi, 3.1415926535897932)
assert.eq(math.phi, 1.6180339887498948)
