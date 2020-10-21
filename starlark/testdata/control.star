# Tests of Starlark control flow

load(lib="assert.star")
assert = lib.assert

def controlflow():
  # elif
  x = 0
  if True:
    x=1
  elif False:
    assert.fail("else of true")
  else:
    assert.fail("else of else of true")
  assert.true(x)

  x = 0
  if False:
    assert.fail("then of false")
  elif True:
    x = 1
  else:
    assert.fail("else of true")
  assert.true(x)

  x = 0
  if False:
    assert.fail("then of false")
  elif False:
    assert.fail("then of false")
  else:
    x = 1
  assert.true(x)
controlflow()

def loops():
  y = ""
  for x in [1, 2, 3, 4, 5]:
    if x == 2:
      continue
    if x == 4:
      break
    y = y + str(x)
  return y
assert.eq(loops(), "13")

# return
g = 123
def f(x):
  for g in (1, 2, 3):
    if g == x:
      return g
assert.eq(f(2), 2)
assert.eq(f(4), None) # falling off end => return None
assert.eq(g, 123) # unchanged by local use of g in function

# infinite sequences
def fib(n):
  seq = []
  for x in fibonacci: # fibonacci is an infinite iterable defined in eval_test.go
    if len(seq) == n:
      break
    seq.append(x)
  return seq
assert.eq(fib(10),  [0, 1, 1, 2, 3, 5, 8, 13, 21, 34])
