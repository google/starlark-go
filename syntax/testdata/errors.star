# Tests of parse errors.
# This is a "chunked" file; each "---" line demarcates a new parser input.
#
# TODO(adonovan): lots more tests.

x = 1 +
2 ### "got newline, want primary expression"

---

_ = *x ### `got '\*', want primary`

---
# trailing comma is ok

def f(a, ): pass
def f(*args, ): pass
def f(**kwargs, ): pass

---

# Parameters are validated later.
def f(**kwargs, *args, *, b=1, a, **kwargs, *args, *, b=1, a):
  pass

---

def f(a, *-b, c): # ### `got '-', want ','`
  pass

---

def f(**kwargs, *args, b=1, a, **kwargs, *args, b=1, a):
  pass

---

def pass(): ### "not an identifier"
  pass

---

def f : ### `got ':', want '\('`

---
# trailing comma is ok

f(a, )
f(*args, )
f(**kwargs, )

---

f(a=1, *, b=2) ### `got ',', want primary`

---

_ = {x:y for y in z} # ok
_ = {x for y in z}   ### `got for, want ':'`

---

def f():
  pass
 pass ### `unindent does not match any outer indentation level`

---
def f(): pass
---
# Blank line after pass => outdent.
def f():
	pass

---
# No blank line after pass; EOF acts like a newline.
def f():
	pass
---
# This is a well known parsing ambiguity in Python.
# Python 2.7 accepts it but Python3 and Starlark reject it.
_ = [x for x in lambda: True, lambda: False if x()] ### "got lambda, want primary"

_ = [x for x in (lambda: True, lambda: False) if x()] # ok in all dialects

---
# Starlark, following Python 3, allows an unparenthesized
# tuple after 'in' only in a for statement but not in a comprehension.
# (Python 2.7 allows both.)
for x in 1, 2, 3:
      print(x)

_ = [x for x in 1, 2, 3] ### `got ',', want ']', for, or if`
---
# Unparenthesized tuple is not allowed as operand of 'if' in comprehension.
_ = [a for b in c if 1, 2] ### `got ',', want ']', for, or if`

---
# Lambda is ok though.
_ = [a for b in c if lambda: d] # ok

# But the body of such a lambda may not be a conditional:
_ = [a for b in c if (lambda: d if e else f)] # ok
_ = [a for b in c if lambda: d if e else f]   ### "got else, want ']'"

---
# A lambda is not allowed as the operand of a 'for' clause.
_ = [a for b in lambda: c] ### `got lambda, want primary`

---
# Comparison operations are not associative.

_ = (0 == 1) == 2 # ok
_ = 0 == (1 == 2) # ok
_ = 0 == 1 == 2 ### "== does not associate with =="

---

_ = (0 <= i) < n   # ok
_ = 0 <= (i < n) # ok
_ = 0 <= i < n ### "<= does not associate with <"

---

_ = (a in b) not in c  # ok
_ = a in (b not in c)  # ok
_ = a in b not in c    ### "in does not associate with not in"

---
# shift/reduce ambiguity is reduced
_ = [x for x in a if b else c] ### `got else, want ']', for, or if`
---
[a for b in c else d] ### `got else, want ']', for, or if`
---
_ = a + b not c ### "got identifier, want in"
---
f(1+2 = 3) ### "keyword argument must have form name=expr"
---
print(1, 2, 3
### `got end of file, want '\)'`
---
_ = a if b ### "conditional expression without else clause"
---
load("") ### "load statement must import at least 1 symbol"
---
load("", 1) ### `load operand must be "name" or localname="name" \(got int literal\)`
---
load("a", "x") # ok
---
load(1, 2) ### "first operand of load statement must be a string literal"
---
load("a", x) ### `load operand must be "x" or x="originalname"`
---
load("a", x2=x) ### `original name of loaded symbol must be quoted: x2="originalname"`
---
# All of these parse.
load("a", "x")
load("a", "x", y2="y")
load("a", x2="x", "y") # => positional-before-named arg check happens later (!)
---
# 'load' is not an identifier
load = 1 ### `got '=', want '\('`
---
# 'load' is not an identifier
f(load()) ### `got load, want primary`
---
# 'load' is not an identifier
def load(): ### `not an identifier`
  pass
---
# 'load' is not an identifier
def f(load): ### `not an identifier`
  pass
---
# A load statement allows a trailing comma.
load("module", "x",)
---
x = 1 +
2 ### "got newline, want primary expression"
---
def f():
    pass
# this used to cause a spurious indentation error
---
print 1 2 ### `got int literal, want newline`

---
# newlines are not allowed in raw string literals
raw = r'a ### `unexpected newline in string`
b'

---
# The parser permits an unparenthesized tuple expression for the first index.
x[1, 2:] # ok
---
# But not if it has a trailing comma.
x[1, 2,:] ### `got ':', want primary`
---
# Trailing tuple commas are permitted only within parens; see b/28867036.
(a, b,) = 1, 2 # ok
c, d = 1, 2 # ok
---
a, b, = 1, 2 ### `unparenthesized tuple with trailing comma`
---
a, b = 1, 2, ### `unparenthesized tuple with trailing comma`

---
# See github.com/google/starlark-go/issues/48
a = max(range(10))) ### `unexpected '\)'`

---
# github.com/google/starlark-go/issues/85
s = "\x-0" ### `invalid escape sequence`
