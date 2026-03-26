# Tests of Starlark 'fstring'
# option:set

# Tests of Starlark f-string literals
# syntax identical to Python 3.6+ (no '=', no '\{', no nested '{ }')
load("assert.star", "assert")

# --- basic interpolation ----------------------------------------------------

name = "Starlark"
version = 1
assert.eq(f"hello {name}", "hello Starlark")
assert.eq(f"{{}} {name}", "{} Starlark")
assert.eq(f"{{{{}}}} {name}", "{{}} Starlark")
assert.eq(f"{name} {version}", "Starlark 1")
assert.eq(f"{{ literal }}", "{ literal }")   # doubled braces → literal
assert.eq(f"start{ {"x":3} }end", "start{\"x\": 3}end")   # dict

# --- conversion flags -------------------------------------------------------
# todo: future plans
# pi = 3.14
# assert.eq(f"{pi!s}", "3.14")   # str()
# assert.eq(f"{pi!r}", "3.14")   # repr()  (same here because str/repr identical)
# big = 1000000
# assert.eq(f"{big:,}", "1,000,000")  # format-spec with ','

# --- field names -----------------------------------------------------------

d = {"x": 10, "y": 20}
assert.eq(f"{d['x']}", "10")
assert.eq(f"{d['y']}", "20")

# positional and keyword in .format() style already tested elsewhere;
# f-strings use **inline** expressions, so we just check they evaluate.
tpl = (4, 5)
assert.eq(f"{tpl[0]} and {tpl[1]}", "4 and 5")

# --- padding / alignment / precision ---------------------------------------
#todo: future plans
# n = 42
# assert.eq(f"{n:>5}", "   42")
# assert.eq(f"{n:<5}", "42   ")
# assert.eq(f"{n:^5}", " 42  ")
# assert.eq(f"{n:05}", "00042")

# f = 1.234567
# assert.eq(f"{f:.2f}", "1.23")

# --- escaping --------------------------------------------------------------
assert.eq(f"backslash \\ still one", "backslash \\ still one")
assert.eq(f"quotes ' and \" kept", 'quotes \' and " kept')

# --- empty f-string --------------------------------------------------------

assert.eq(f"", "")

# --- nested quotes (no problem) --------------------------------------------

assert.eq(f'He said "Hello {name}"', 'He said "Hello Starlark"')
assert.eq(f"it's {version} o'clock", "it's 1 o'clock")

# --- multiline f-string ----------------------------------------------------

msg = f"""
hello {name}
version {version}
""".strip()
assert.true(msg.startswith("hello Starlark"))
assert.true(msg.endswith("version 1"))

# --- unicode ---------------------------------------------------------------

α = 2
assert.eq(f"α = {α}", "α = 2")

# --- errors that must be caught at compile-time ----------------------------
#todo? more sound errors, now raises with text: `expect "}}" or "{expression}", got single"}"` on almost all errors
# (Un-comment each block to verify the parser rejects it.)

# 1. single '}' without '{'
# assert.fails(lambda: f"oops}", "single '}' in format")

# 2. unmatched '{' #now says "unexpected new line in string". is it ok?
# assert.fails(lambda: f"oops{", "unmatched '{' in format")

# 3. invalid expression inside braces # now fails with "got , want primary expression"
# assert.fails(lambda: f"{1+}", "invalid syntax")

# 4. unknown conversion #future plans?
# assert.fails(lambda: f"{pi!z}", "unknown conversion")

# 5. # now fails with "got , want primary expression"
# assert.fails(lambda: f"nothing{}", "nothing{}")          # todo: assert fails

# --- runtime errors --------------------------------------------------------

# expression raises → propagates
def raise_error():
    f"{1/0}"

assert.fails(raise_error, "division by zero")

# --- interaction with other string features -------------------------------

# f-string is still a plain string afterwards
s = f"{name}"
assert.eq(s.upper(), "STARLARK")
assert.eq(s * 2, "StarlarkStarlark")
assert.eq(len(s), 8)

# concatenation
assert.eq(f"A" + f"{name}" + f"B", "AStarlarkB")

# raw *and* f is illegal in Python; Starlark should follow
# (un-comment to check)
# assert.fails(lambda: rf"{name}", "cannot use both raw and f-string") #todo?

# --- edge cases ------------------------------------------------------------

# only doubled braces
assert.eq(f"{{}}", "{}")

# mixed
#todo: fix this or just look into this
# assert.eq(f"{{ {name} }}", "{ Starlark }")

# zero-width joiner emoji (4-byte UTF-8)
emoji = "😿"
assert.eq(f"{emoji}", "😿")
# assert.eq(f"{emoji!r}", '"😿"') # todo: future plans?

# ---------------------------------------------------------------------------
# end of f-string tests