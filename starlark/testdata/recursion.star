# Tests of Starlark recursion and while statement.

# This is a "chunked" file: each "---" effectively starts a new file.

# option:recursion

load("assert.star", "assert")

def fib(n):
	if n <= 1:
		return 1
	return fib(n-1) + fib(n-2)

assert.eq(fib(5), 8)

def runaway():
	return runaway()

# Runaway recursion should not overflow the Go stack (#617).
# The interpreter imposes a limit long before then.
assert.fails(runaway, "Starlark stack overflow")