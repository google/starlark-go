# Tests of math module.

load('math.star', 'math')
load('assert.star', 'assert')

def assert_near(actual, desired, threshold):
  val = desired - actual
  if val < 0:
    val = val * -1
  return val < threshold

assert.eq(math.ceil(0.5), 1.0)
assert.eq(math.fabs(-2.0), 2.0)
assert.eq(math.floor(0.5), 0)

assert.eq(math.exp(2.0), 7.38905609893065)
assert.eq(math.sqrt(4.0), 2.0)

assert.eq(math.acos(1.0), 0)
assert.eq(math.asin(1.0), 1.5707963267948966)
assert.eq(math.atan(1.0), 0.7853981633974483)
assert.eq(math.atan2(1.0,1.0), 0.7853981633974483)
assert.eq(math.cos(1.0), 0.5403023058681398)
assert.eq(math.hypot(1.0,1.0), 1.4142135623730951)
assert.eq(math.sin(1.0), 0.8414709848078965)
assert.eq(math.tan(1.0), 1.557407724654902)

assert.eq(math.degrees(1.0), 57.29577951308232)
assert.eq(math.radians(1.0), 0.017453292519943295)

assert.eq(math.acosh(1.0), 0)
assert.eq(assert_near(math.asinh(1.0), 0.8813735870195432, 0.00000001), True)
assert.eq(math.atanh(0.5), 0.5493061443340548)
assert.eq(math.cosh(1.0), 1.5430806348152437)
assert.eq(math.sinh(1.0), 1.1752011936438014)
assert.eq(math.tanh(1.0), 0.7615941559557649)

assert.eq(math.e, 2.7182818284590452)
assert.eq(math.pi, 3.1415926535897932)
assert.eq(math.tau, 6.2831853071795864)
assert.eq(math.phi, 1.6180339887498948)
assert.eq(math.inf, math.inf + 1)
assert.eq(math.nan == math.nan, False)
