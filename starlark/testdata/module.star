# Tests of Module.

load("assert.star", "assert")

assert.eq(type(assert), "module")
assert.eq(str(assert), '<module "assert">')
assert.eq(dir(assert), ["contains", "eq", "fail", "fails", "lt", "ne", "true"])
assert.fails(lambda : {assert: None}, "unhashable: module")
