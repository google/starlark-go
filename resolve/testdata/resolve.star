# Tests of resolver errors.
#
# The initial environment contains the predeclared names "M"
# (module-specific) and "U" (universal). This distinction
# should be unobservable to the Starlark program.

# use of declared global
x = 1
_ = x

---
# premature use of global is not a static error;
# see github.com/google/skylark/issues/116.
_ = x
x = 1

---
# use of undefined global
_ = x ### "undefined: x"

---
# redeclaration of global
x = 1
x = 2 ### "cannot reassign global x declared at .*resolve.star:23:1"

---
# Redeclaration of predeclared names is allowed.
#
# This rule permits tool maintainers to add members to the predeclared
# environment without breaking existing programs.

# module-specific predeclared name
M = 1 # ok
M = 2 ### "cannot reassign global M declared at .*/resolve.star"

# universal predeclared name
U = 1 # ok
U = 1 ### "cannot reassign global U declared at .*/resolve.star"

---
# A global declaration shadows all references to a predeclared;
# see github.com/google/skylark/issues/116.

a = U # ok: U is a reference to the global defined on the next line.
U = 1

---
# reference to predeclared name
M()

---
# locals may be referenced before they are defined

def f():
   M(x) # dynamic error
   x = 1

---
# Various forms of assignment:

def f(x): # parameter
    M(x)
    M(y) ### "undefined: y"

(a, b) = 1, 2
M(a)
M(b)
M(c) ### "undefined: c"

[p, q] = 1, 2
M(p)
M(q)
M(r) ### "undefined: r"

---
# a comprehension introduces a separate lexical block

_ = [x for x in "abc"]
M(x) ### "undefined: x"

---
# Functions may have forward refs.   (option:lambda option:nesteddef)
def f():
   g()
   h() ### "undefined: h"
   def inner():
     i()
     i = lambda: 0

def g():
  f()

---
# It is not permitted to rebind a global using a += assignment.

x = [1]
x.extend([2]) # ok
x += [3] ### `cannot reassign global x`

def f():
   x += [4] # x is local to f

y = 1
y += 2 ### `cannot reassign global y`
z += 3 # ok (but fails dynamically because z is undefined)

---
def f(a):
  if 1==1:
    b = 1
  c = 1
  M(a) # ok: param
  M(b) # ok: maybe bound local
  M(c) # ok: bound local
  M(d) # NB: we don't do a use-before-def check on local vars!
  M(e) # ok: global
  M(f) # ok: global
  d = 1

e = 1

---
# This program should resolve successfully but fail dynamically.
x = 1

def f():
  M(x) # dynamic error: reference to undefined local
  x = 2

f()

---
load("module", "name") # ok

def f():
  load("foo", "bar") ### "load statement within a function"

load("foo",
     "",     ### "load: empty identifier"
     "_a",   ### "load: names with leading underscores are not exported: _a"
     b="",   ### "load: empty identifier"
     c="_d", ### "load: names with leading underscores are not exported: _d"
     _e="f") # ok

---
# return statements must be within a function

return ### "return statement not within a function"

---
# if-statements and for-loops at top-level are forbidden
# (without globalreassign option)

for x in "abc": ### "for loop not within a function"
  pass

if x: ### "if statement not within a function"
  pass

---
# option:globalreassign

for x in "abc": # ok
  pass

if x: # ok
  pass

---
# while loops are forbidden (without -recursion option)

def f():
  while U: ### "dialect does not support while loops"
    pass

---
# option:recursion

def f():
  while U: # ok
    pass

while U: ### "while loop not within a function"
  pass

---
# option:globalreassign option:recursion

while U: # ok
  pass

---
# The parser allows any expression on the LHS of an assignment.

1 = 0 ### "can't assign to literal"
1+2 = 0 ### "can't assign to binaryexpr"
f() = 0 ### "can't assign to callexpr"

[a, b] = 0
[c, d] += 0 ### "can't use list expression in augmented assignment"
(e, f) += 0 ### "can't use tuple expression in augmented assignment"
[] = 0 ### "can't assign to \\[\\]"
() = 0 ### "can't assign to ()"

---
# break and continue statements must appear within a loop

break ### "break not in a loop"

continue ### "continue not in a loop"

pass

---
# Positional arguments (and required parameters)
# must appear before named arguments (and optional parameters).

M(x=1, 2) ### `positional argument may not follow named`

def f(x=1, y): pass ### `required parameter may not follow optional`
---
# No parameters may follow **kwargs in a declaration.

def f(**kwargs, x): ### `parameter may not follow \*\*kwargs`
  pass

def g(**kwargs, *args): ### `\* parameter may not follow \*\*kwargs`
  pass

def h(**kwargs1, **kwargs2): ### `multiple \*\* parameters not allowed`
  pass

---
# Only keyword-only params and **kwargs may follow *args in a declaration.

def f(*args, x): # ok
  pass

def g(*args1, *args2): ### `multiple \* parameters not allowed`
  pass

def h(*, ### `bare \* must be followed by keyword-only parameters`
      *): ### `multiple \* parameters not allowed`
  pass

def i(*args, *): ### `multiple \* parameters not allowed`
  pass

def j(*,      ### `bare \* must be followed by keyword-only parameters`
      *args): ### `multiple \* parameters not allowed`
  pass

def k(*, **kwargs): ### `bare \* must be followed by keyword-only parameters`
  pass

def l(*): ### `bare \* must be followed by keyword-only parameters`
  pass

def m(*args, a=1, **kwargs): # ok
  pass

def n(*, a=1, **kwargs): # ok
  pass

---
# No arguments may follow **kwargs in a call.
def f(*args, **kwargs):
  pass

f(**{}, 1) ### `argument may not follow \*\*kwargs`
f(**{}, x=1) ### `argument may not follow \*\*kwargs`
f(**{}, *[]) ### `\*args may not follow \*\*kwargs`
f(**{}, **{}) ### `multiple \*\*kwargs not allowed`

---
# Only keyword arguments may follow *args in a call.
def f(*args, **kwargs):
  pass

f(*[], 1) ### `argument may not follow \*args`
f(*[], a=1) # ok
f(*[], *[]) ### `multiple \*args not allowed`
f(*[], **{}) # ok

---
# Parameter names must be unique.

def f(a, b, a): pass ### "duplicate parameter: a"
def g(args, b, *args): pass ### "duplicate parameter: args"
def h(kwargs, a, **kwargs): pass ### "duplicate parameter: kwargs"
def i(*x, **x): pass ### "duplicate parameter: x"

---
# No floating point
a = float("3.141") ### `dialect does not support floating point`
b = 1 / 2          ### `dialect does not support floating point \(use //\)`
c = 3.141          ### `dialect does not support floating point`
---
# Floating point support (option:float)
a = float("3.141")
b = 1 / 2
c = 3.141

---
# option:globalreassign
# Legacy Bazel (and Python) semantics: def must precede use even for globals.

_ = x ### `undefined: x`
x = 1

---
# option:globalreassign
# Legacy Bazel (and Python) semantics: reassignment of globals is allowed.
x = 1
x = 2 # ok

---
# option:globalreassign
# Redeclaration of predeclared names is allowed.

# module-specific predeclared name
M = 1 # ok
M = 2 # ok (legacy)

# universal predeclared name
U = 1 # ok
U = 1 # ok (legacy)

---
# https://github.com/bazelbuild/starlark/starlark/issues/21
def f(**kwargs): pass
f(a=1, a=1) ### `keyword argument a repeated`


---
# spelling

print = U

hello = 1
print(hollo) ### `undefined: hollo \(did you mean hello\?\)`

def f(abc):
   print(abd) ### `undefined: abd \(did you mean abc\?\)`
   print(goodbye) ### `undefined: goodbye$`

---
load("module", "x") # ok
x = 1 ### `cannot reassign local x`
load("module", "x") ### `cannot reassign top-level x`

---
# option:loadbindsglobally
load("module", "x") # ok
x = 1 ### `cannot reassign global x`
load("module", "x") ### `cannot reassign global x`

---
# option:globalreassign
load("module", "x") # ok
x = 1 # ok
load("module", "x") # ok

---
# option:globalreassign option:loadbindsglobally
load("module", "x") # ok
x = 1
load("module", "x") # ok

---
_ = x # forward ref to file-local
load("module", "x") # ok
