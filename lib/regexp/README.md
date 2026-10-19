# `regexp` — regular expressions for Starlark

The `regexp` module adds regular expression matching to Starlark. A pattern is
compiled once into a `regexp.regexp` value, which is then used to test, search,
split and rewrite strings. Searching yields `regexp.match` values that carry the
matched text, its offsets, and the submatch of each capture group.

```python
rx = regexp.compile(r"(\w+)=(\d+)")

if rx.find(s):                                   # is there a match?
    m = rx.find(s)                               # the leftmost match
    print(m.text, m.start, m.end)                # "a=1" 0 3
    print(m.submatches[0].text)                  # "a"

pairs = {m.submatches[0].text: int(m.submatches[1].text) for m in rx.findall(s)}
```

## Contents

- [Making the module available](#making-the-module-available)
- [Pattern syntax](#pattern-syntax)
- [Module functions](#module-functions)
- [The `regexp.regexp` type](#the-regexpregexp-type)
- [The `regexp.match` and `regexp.submatch` types](#the-regexpmatch-and-regexpsubmatch-types)
- [Replacement templates](#replacement-templates)
- [Things worth knowing](#things-worth-knowing)

## Making the module available

The module is not predeclared by default; the host application decides whether
a program can see it. In Go, either add it to the universe, so that every
thread sees it:

```go
import (
	"go.starlark.net/lib/regexp"
	"go.starlark.net/starlark"
)

starlark.Universe["regexp"] = regexp.Module
```

or predeclare it for a single execution:

```go
globals, err := starlark.ExecFileOptions(opts, thread, filename, src, starlark.StringDict{
	"regexp": regexp.Module,
})
```

The `starlark` command does the former, so the module is available in the REPL
and with `starlark -c`:

```console
$ starlark -c 'print([m.text for m in regexp.compile("a+").findall("banana")])'
["a", "a", "a"]
```

Applications that resolve `load` statements may instead expose the module under
a module name of their choosing, for example `load("regexp.star", "regexp")`.

## Pattern syntax

Patterns use **RE2 syntax**, documented at
<https://github.com/google/re2/wiki/Syntax>. RE2 matches in time linear in the
length of the subject string; as a consequence it has no backreferences and no
lookahead or lookbehind assertions, unlike Perl or Python's `re`.

Two further points:

- `\C`, which matches a single byte, is **not supported** and is rejected by
  `compile`. It cannot be implemented by Starlark implementations whose strings
  are not byte-oriented.
- Patterns and subject strings are interpreted as UTF-8, so `.` matches one
  rune, not one byte:

  ```python
  [m.text for m in regexp.compile(".").findall("héllo")]  # ["h", "é", "l", "l", "o"]
  ```

Flags are written inside the pattern in the usual RE2 way, e.g. `(?i)` for
case-insensitive matching, `(?s)` to let `.` match a newline, and `(?m)` to make
`^` and `$` match at line boundaries.

Starlark's raw string literals, `r"..."`, save escaping each backslash:
`r"(\w+)"` is the same pattern as `"(\\w+)"`.

## Module functions

| Function | Returns | Description |
| --- | --- | --- |
| `regexp.compile(pattern)` | `regexp.regexp` | Compiles `pattern`; fails if it is not a valid regular expression. |
| `regexp.escape(lit)` | `string` | Returns a pattern that matches the literal text `lit`. |

### `regexp.compile(pattern)`

| Parameter | Type | Default | Description |
| --- | --- | --- | --- |
| `pattern` | `string` | — | The regular expression, in RE2 syntax. |

Returns a `regexp.regexp`. Compilation errors are reported immediately:

```python
regexp.compile("(")     # error: error parsing regexp: missing closing ): `(`
regexp.compile("\\C")   # error: error parsing regexp: invalid escape sequence: `\C`
```

### `regexp.escape(lit)`

| Parameter | Type | Default | Description |
| --- | --- | --- | --- |
| `lit` | `string` | — | Literal text to escape. |

Returns `lit` with every metacharacter escaped, so that the result is a pattern
matching exactly and only `lit`. Useful for building a pattern from a value:

```python
regexp.escape("a.b")                                  # "a\\.b"
regexp.compile(regexp.escape("a.b")).matches("a.b")   # True
regexp.compile(regexp.escape("a.b")).matches("axb")   # False

rx = regexp.compile(r"\b" + regexp.escape(word) + r"\b")   # match a literal word
```

## The `regexp.regexp` type

A compiled pattern. It is immutable, hashable, and always truthy.

```python
rx = regexp.compile("b(an)*a")
type(rx)    # "regexp.regexp"
str(rx)     # 'regexp.compile("b(an)*a")'
rx.pattern  # "b(an)*a"
```

### Fields

| Field | Type | Description |
| --- | --- | --- |
| `pattern` | `string` | The source text of the regular expression. |
| `groupnames` | `list` of (`string` or `None`) | The name of each capture group, counted from zero, or `None` for each unnamed group. |

```python
regexp.compile(r"(?P<key>\w+)=(\w+)").groupnames  # ["key", None]
regexp.compile("x").groupnames                    # []
```

### Methods

| Method | Returns | Description |
| --- | --- | --- |
| [`find(s)`](#finds) | `regexp.match` or `None` | The leftmost match, or `None`. |
| [`findall(s)`](#findalls) | `regexp.matches` | Iterable of every non-overlapping match. |
| [`matches(s)`](#matchess) | `bool` | Whether the pattern matches anywhere in `s`. |
| [`replace(s, repl)`](#replaces-repl) | `string` | `s` with every match replaced. |
| [`split(s, max=-1)`](#splits-max-1) | `list` of `string` | The text between the matches. |

All arguments are positional; keyword arguments are not accepted.

#### `find(s)`

| Parameter | Type | Default | Description |
| --- | --- | --- | --- |
| `s` | `string` | — | The string to search. |

Returns a [`regexp.match`](#the-regexpmatch-and-regexpsubmatch-types) for the
leftmost match, or `None` if there is none. A match is always true, even when
its text is empty, so `find` doubles as a match test:

```python
rx = regexp.compile(r"(\w+)=(\d+)")

if rx.find(s): print("has a key=value pair")   # match test
m = rx.find(s)
key = m.submatches[0].text if m else None      # extract a group
pos = rx.find(s).start                         # offset (errors if no match)

regexp.compile("x*").find("zzz").text          # ""   — matched the empty string
regexp.compile("x+").find("zzz")               # None — no match at all
```

#### `findall(s)`

| Parameter | Type | Default | Description |
| --- | --- | --- | --- |
| `s` | `string` | — | The string to search. |

Returns a `regexp.matches`: an iterable of the `regexp.match` of each successive
non-overlapping match. It may be iterated more than once, and it is true if
there is at least one match. An empty match immediately following a previous
match is skipped.

```python
rx = regexp.compile(r"(\w+)=(\d+)")

if rx.findall(s): print("nonempty")                       # truthiness
texts = [m.text for m in rx.findall(s)]                   # every match
spans = [(m.start, m.end) for m in rx.findall(s)]         # all offsets
n = len(list(rx.findall(s)))                              # count
pairs = {m.submatches[0].text: int(m.submatches[1].text)  # build a dict
         for m in rx.findall(s)}
hits = [m.text for m in rx.findall(s) if m.submatches[0].text == "foo"]
```

`regexp.matches` is an iterable, not a sequence: it has no `len`, and it is not
hashable. Wrap it in `list(...)` when you need either.

#### `matches(s)`

| Parameter | Type | Default | Description |
| --- | --- | --- | --- |
| `s` | `string` | — | The string to search. |

Reports whether the pattern matches anywhere in `s`, without building a match.

```python
rx = regexp.compile("b(an)*a")
rx.matches("--banana--")  # True
rx.matches("apple")       # False
```

#### `replace(s, repl)`

| Parameter | Type | Default | Description |
| --- | --- | --- | --- |
| `s` | `string` | — | The string to rewrite. |
| `repl` | `string` or callable | — | Replacement template, or a function of one `regexp.match` returning a `string`. |

Returns a copy of `s` in which every non-overlapping match has been replaced.
An empty match immediately following a previous match is skipped.

With a string, `$1` and `${name}` expand to the corresponding group — see
[Replacement templates](#replacement-templates):

```python
a = regexp.compile("a+")
a.replace("banana", "o")               # "bonono"
a.replace("melon", "o")                # "melon"

kv = regexp.compile(r"(\w+)=(\d+)")
kv.replace("a=1,b=2", "$2=$1")         # "1=a,2=b"
kv.replace("a=1,b=2", "<redacted>")    # "<redacted>,<redacted>"
```

With a function, it is called once per match with the `regexp.match` and must
return a string:

```python
kv.replace(s, lambda m: m.text.upper())                                   # transform the match
kv.replace(s, lambda m: m.submatches[1].text + "=" + m.submatches[0].text) # swap the groups
kv.replace(s, lambda m: "%d:%d" % (m.start, m.end))                       # use the offsets
```

The function runs on the calling thread, so it is subject to the thread's
cancellation and execution-step limits, and any error it raises aborts
`replace` and propagates to the caller.

#### `split(s, max=-1)`

| Parameter | Type | Default | Description |
| --- | --- | --- | --- |
| `s` | `string` | — | The string to split. |
| `max` | `int` | `-1` | Maximum number of substrings; negative means all. |

Returns the strings between the matches of the pattern. If `max` substrings
would be exceeded, the last one is the unsplit remainder; `max = 0` yields an
empty list.

```python
comma = regexp.compile(",+")
comma.split("a,b,,c")      # ["a", "b", "c"]
comma.split("a,b,,c", 2)   # ["a", "b,,c"]
comma.split("a,b,,c", 0)   # []
comma.split("abc")         # ["abc"]
```

## The `regexp.match` and `regexp.submatch` types

`find` returns a `regexp.match`, and iterating a `regexp.matches` yields one per
match. Both types are immutable and hashable.

| Field or method | Type | Description |
| --- | --- | --- |
| `text` | `string` | The matching substring. |
| `start`, `end` | `int` | Offsets of the match within `s`, so that `s[start:end] == text`. |
| `submatches` | `regexp.submatches` | Indexable sequence of the capture groups, counted from zero. |
| `submatch(x)` | `regexp.submatch` or `None` | The group selected by index or by name. |

A `regexp.submatch` has `text`, `start` and `end` with the same meaning, but no
`submatch` method and no `submatches` field.

```python
rx = regexp.compile(r"(\w+)=(\d+)")
m = rx.find("key=42")

m.text                  # "key=42"
m.start, m.end          # (0, 6)
len(m.submatches)       # 2
m.submatches[0].text    # "key"
m.submatches[1].start   # 4
[sm.text for sm in m.submatches]   # ["key", "42"]
```

Offsets are byte offsets into the subject string, which is exactly how Starlark
indexes strings, so `s[m.start:m.end]` is always `m.text`.

### `submatch(x)`

| Parameter | Type | Description |
| --- | --- | --- |
| `x` | `int` or `string` | The index of a capture group, counted from zero, or the name of a named group. |

`submatch(i)` is `submatches[i]`, and `submatch("name")` selects the group
`(?P<name>…)`. A negative index counts from the end, as in `s[-1]`. It is an
error if there is no such group — that is a mistake in the program, unlike a
group that merely did not participate in the match, which yields `None`:

```python
named = regexp.compile(r"(?P<year>\d{4})-(?P<month>\d{2})").find("on 2026-10-19")
named.submatch("year").text    # "2026"
named.submatch(1).text         # "10"  — same as submatches[1]
named.submatch("day")          # error: submatch: regexp has no group named "day"
named.submatch(2)              # error: submatch: group index 2 out of range: regexp has 2 groups
```

### Groups that did not participate

An optional group that took no part in the match is reported as `None`, both by
`submatch` and within `submatches`, so test it before reading `.text`:

```python
opt = regexp.compile("a(x)?")
m = opt.find("ax")
m.submatches[0]                                  # <regexp.submatch "x" at 1:2>

m = opt.find("a")
m.submatches[0]                                  # None
v = m.submatches[0].text if m.submatches[0] else ""
```

## Replacement templates

When `replace` is given a string, it is a template in which `$` has a special
meaning:

| Form | Meaning |
| --- | --- |
| `$1`, `$2`, … | The text of the numbered group. |
| `$name`, `${name}` | The text of the named group `(?P<name>…)`. |
| `$$` | A literal `$`. |

Note that `$1` is the **first** group, matching RE2's own numbering, while
`submatches[0]` is that same group — the template numbering is one-based, the
sequence is zero-based.

Use the braced form whenever the name is followed by other word characters:
`$1x` names the group `1x`, while `${1}x` is group 1 followed by `x`. A
reference to a group that does not exist expands to the empty string.

```python
kv = regexp.compile(r"(?P<key>\w+)=(?P<value>\d+)")
kv.replace("a=1", "${value}=${key}")  # "1=a"
kv.replace("a=1", "$nope")            # ""
```

To insert text that itself contains a `$` — text supplied by the user, say —
either double each dollar sign, or pass a function, whose return value is used
verbatim:

```python
rx = regexp.compile("x")
rx.replace("x", "costs $$5")           # "costs $5"
rx.replace("x", lambda m: "costs $5")  # "costs $5"
```

## Things worth knowing

- **No match versus empty match.** `find` returns `None` when nothing matched,
  and a match — which is true even if `text` is `""` — when something did.
- **Equality compares text and offsets.** Two matches, or two submatches, are
  equal if they have the same text at the same offsets, so repeated lookups of
  the same group are interchangeable and usable as dict keys. Compiled patterns,
  by contrast, compare by identity: `regexp.compile("a") == regexp.compile("a")`
  is `False`; compare `rx.pattern` instead. Matches are not *ordered*: `<` and
  `>` are errors.
- **No pattern cache.** Each `compile` call parses the pattern afresh, so hoist
  `compile` out of loops and, where possible, to the top level of a module.
- **`findall` is eager underneath.** The value it returns behaves lazily — it is
  only iterated on demand and can be iterated repeatedly — but the current
  implementation locates all matches on first use, because Go's `regexp` gained
  iterator methods only after the version this module targets. Code that breaks
  out of a loop early is therefore correct, but not yet cheaper.
- **Matching is not metered.** Match time is linear in the length of the subject
  string, but it is not counted against the thread's execution step budget
  (`Thread.SetMaxExecutionSteps`). Applications running untrusted programs
  should bound their resources by other means; see the "Bounding time, memory,
  and other resources" section of the [top-level README](../../README.md).
