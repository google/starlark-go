# Predeclared built-ins for this module:
#
# error(msg): report an error in Go's test framework without halting execution.
#  This is distinct from the built-in fail function, which halts execution.
# catch(f): evaluate f() and returns its evaluation error message, if any
# matches(str, pattern): report whether str matches regular expression pattern.
# module(**kwargs): a constructor for a module.
# _freeze(x): freeze the value x and everything reachable from it.
#
# Clients may use these functions to define their own testing abstractions.

def _eq(x, y):
    if x != y:
        error("%r != %r" % (x, y))

def _ne(x, y):
    if x == y:
        error("%r == %r" % (x, y))

def _true(cond, msg = "assertion failed"):
    if not cond:
        error(msg)

def _lt(x, y):
    if not (x < y):
        error("%s is not less than %s" % (x, y))

def _contains(x, y):
    if y not in x:
        error("%s does not contain %s" % (x, y))

def _fails(f, pattern):
    "assert_fails asserts that evaluation of f() fails with the specified error."
    msg = catch(f)
    if msg == None:
        error("evaluation succeeded unexpectedly (want error matching %r)" % pattern)
    elif not matches(pattern, msg):
        error("regular expression (%s) did not match error (%s)" % (pattern, msg))

freeze = _freeze  # an exported global whose value is the built-in freeze function

assert = module(
    "assert",
    fail = error,
    eq = _eq,
    ne = _ne,
    true = _true,
    lt = _lt,
    contains = _contains,
    fails = _fails,
)
