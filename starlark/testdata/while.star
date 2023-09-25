# Tests of Starlark while statement.

# This is a "chunked" file: each "---" effectively starts a new file.

# option:while

load("assert.star", "assert")

def sum(n):
	r = 0
	while n > 0:
		r += n
		n -= 1
	return r

def while_break(n):
	r = 0
	while n > 0:
		if n == 5:
			break
		r += n
		n -= 1
	return r

def while_continue(n):
	r = 0
	while n > 0:
		if n % 2 == 0:
			n -= 1
			continue
		r += n
		n -= 1
	return r

assert.eq(sum(5), 5+4+3+2+1)
assert.eq(while_break(10), 40)
assert.eq(while_continue(10), 25)
