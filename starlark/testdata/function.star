# Tests of Starlark 'function'
# option:nesteddef option:set

# TODO(adonovan):
# - add some introspection functions for looking at function values
#   and test that functions have correct position, free vars, names of locals, etc.
# - move the hard-coded tests of parameter passing from eval_test.go to here.

load("assert.star", "assert", "freeze")

# Test lexical scope and closures:
def outer(x):
   def inner(y):
     return x + x + y # multiple occurrences of x should create only 1 freevar
   return inner

z = outer(3)
assert.eq(z(5), 11)
assert.eq(z(7), 13)
z2 = outer(4)
assert.eq(z2(5), 13)
assert.eq(z2(7), 15)
assert.eq(z(5), 11)
assert.eq(z(7), 13)

# Function name
assert.eq(str(outer), '<function outer>')
assert.eq(str(z), '<function inner>')
assert.eq(str(str), '<built-in function str>')
assert.eq(str("".startswith), '<built-in method startswith of string value>')

# Stateful closure
def squares():
    x = [0]
    def f():
      x[0] += 1
      return x[0] * x[0]
    return f

sq = squares()
assert.eq(sq(), 1)
assert.eq(sq(), 4)
assert.eq(sq(), 9)
assert.eq(sq(), 16)

# Freezing a closure
sq2 = freeze(sq)
assert.fails(sq2, "frozen list")

# recursion detection, simple
def fib(x):
  if x < 2:
    return x
  return fib(x-2) + fib(x-1)
assert.fails(lambda: fib(10), "function fib called recursively")

# recursion detection, advanced
#
# A simplistic recursion check that looks for repeated calls to the
# same function value will not detect recursion using the Y
# combinator, which creates a new closure at each step of the
# recursion.  To truly prohibit recursion, the dynamic check must look
# for repeated calls of the same syntactic function body.
Y = lambda f: (lambda x: x(x))(lambda y: f(lambda *args: y(y)(*args)))
fibgen = lambda fib: lambda x: (x if x<2 else fib(x-1)+fib(x-2))
fib2 = Y(fibgen)
assert.fails(lambda: [fib2(x) for x in range(10)], "function lambda called recursively")

# However, this stricter check outlaws many useful programs
# that are still bounded, and creates a hazard because
# helper functions such as map below cannot be used to
# call functions that themselves use map:
def map(f, seq): return [f(x) for x in seq]
def double(x): return x+x
assert.eq(map(double, [1, 2, 3]), [2, 4, 6])
assert.eq(map(double, ["a", "b", "c"]), ["aa", "bb", "cc"])
def mapdouble(x): return map(double, x)
assert.fails(lambda: map(mapdouble, ([1, 2, 3], ["a", "b", "c"])),
             'function map called recursively')
# With the -recursion option it would yield [[2, 4, 6], ["aa", "bb", "cc"]].

# call of function not through its name
# (regression test for parsing suffixes of primary expressions)
hf = hasfields()
hf.x = [len]
assert.eq(hf.x[0]("abc"), 3)
def f():
   return lambda: 1
assert.eq(f()(), 1)
assert.eq(["abc"][0][0].upper(), "A")

# functions may be recursively defined,
# so long as they don't dynamically recur.
calls = []
def yin(x):
  calls.append("yin")
  if x:
    yang(False)

def yang(x):
  calls.append("yang")
  if x:
    yin(False)

yin(True)
assert.eq(calls, ["yin", "yang"])

calls.clear()
yang(True)
assert.eq(calls, ["yang", "yin"])


# builtin_function_or_method use identity equivalence.
closures = set(["".count for _ in range(10)])
assert.eq(len(closures), 10)

---
# Default values of function parameters are mutable.
load("assert.star", "assert", "freeze")

def f(x=[0]):
  return x

assert.eq(f(), [0])

f().append(1)
assert.eq(f(), [0, 1])

# Freezing a function value freezes its parameter defaults.
freeze(f)
assert.fails(lambda: f().append(2), "cannot append to frozen list")

---
# This is a well known corner case of parsing in Python.
load("assert.star", "assert")

f = lambda x: 1 if x else 0
assert.eq(f(True), 1)
assert.eq(f(False), 0)

x = True
f2 = (lambda x: 1) if x else 0
assert.eq(f2(123), 1)

tf = lambda: True, lambda: False
assert.true(tf[0]())
assert.true(not tf[1]())

---
# Missing parameters are correctly reported
# in functions of more than 64 parameters.
# (This tests a corner case of the implementation:
# we avoid a map allocation for <64 parameters)

load("assert.star", "assert")

def f(a, b, c, d, e, f, g, h,
      i, j, k, l, m, n, o, p,
      q, r, s, t, u, v, w, x,
      y, z, A, B, C, D, E, F,
      G, H, I, J, K, L, M, N,
      O, P, Q, R, S, T, U, V,
      W, X, Y, Z, aa, bb, cc, dd,
      ee, ff, gg, hh, ii, jj, kk, ll,
      mm):
  pass

assert.fails(lambda: f(
    1, 2, 3, 4, 5, 6, 7, 8,
    9, 10, 11, 12, 13, 14, 15, 16,
    17, 18, 19, 20, 21, 22, 23, 24,
    25, 26, 27, 28, 29, 30, 31, 32,
    33, 34, 35, 36, 37, 38, 39, 40,
    41, 42, 43, 44, 45, 46, 47, 48,
    49, 50, 51, 52, 53, 54, 55, 56,
    57, 58, 59, 60, 61, 62, 63, 64), "missing 1 argument \(mm\)")

assert.fails(lambda: f(
    1, 2, 3, 4, 5, 6, 7, 8,
    9, 10, 11, 12, 13, 14, 15, 16,
    17, 18, 19, 20, 21, 22, 23, 24,
    25, 26, 27, 28, 29, 30, 31, 32,
    33, 34, 35, 36, 37, 38, 39, 40,
    41, 42, 43, 44, 45, 46, 47, 48,
    49, 50, 51, 52, 53, 54, 55, 56,
    57, 58, 59, 60, 61, 62, 63, 64, 65,
    mm = 100), 'multiple values for parameter "mm"')

---
# Regression test for github.com/google/starlark-go/issues/21,
# which concerns dynamic checks.
# Related: https://github.com/bazelbuild/starlark/issues/21,
# which concerns static checks.

load("assert.star", "assert")

def f(*args, **kwargs):
  return args, kwargs

assert.eq(f(x=1, y=2), ((), {"x": 1, "y": 2}))
assert.fails(lambda: f(x=1, **dict(x=2)), 'multiple values for parameter "x"')

def g(x, y):
  return x, y

assert.eq(g(1, y=2), (1, 2))
assert.fails(lambda: g(1, y=2, **{'y': 3}), 'multiple values for parameter "y"')

---
# Regression test for a bug in CALL_VAR_KW.

load("assert.star", "assert")

def f(a, b, x, y):
  return a+b+x+y

assert.eq(f(*("a", "b"), **dict(y="y", x="x")) + ".", 'abxy.')
---
# Order of evaluation of function arguments.
# Regression test for github.com/google/skylark/issues/135.
load("assert.star", "assert")

r = []

def id(x):
       r.append(x)
       return x

def f(*args, **kwargs):
  return (args, kwargs)

y = f(id(1), id(2), x=id(3), *[id(4)], y=id(5), **dict(z=id(6)))
assert.eq(y, ((1, 2, 4), dict(x=3, y=5, z=6)))

# This matches Python2, but not Starlark-in-Java:
# *args and *kwargs are evaluated last.
# See github.com/bazelbuild/starlark#13 for pending spec change.
assert.eq(r, [1, 2, 3, 5, 4, 6])


---
# option:nesteddef option:recursion
# See github.com/bazelbuild/starlark#170
load("assert.star", "assert")

def a():
    list = []
    def b(n):
        list.append(n)
        if n > 0:
            b(n - 1) # recursive reference to b

    b(3)
    return list

assert.eq(a(), [3, 2, 1, 0])

def c():
    list = []
    x = 1
    def d():
      list.append(x) # this use of x observes both assignments
    d()
    x = 2
    d()
    return list

assert.eq(c(), [1, 2])

def e():
    def f():
      return x # forward reference ok: x is a closure cell
    x = 1
    return f()

assert.eq(e(), 1)

---
# option:nesteddef
load("assert.star", "assert")

def e():
    x = 1
    def f():
      print(x) # this reference to x fails
      x = 3    # because this assignment makes x local to f
    f()

assert.fails(e, "local variable x referenced before assignment")
