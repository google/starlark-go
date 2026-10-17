# `regexp` — regular expressions for Starlark

The `regexp` module adds regular expression matching to Starlark. A pattern is
compiled once into a `regexp.regexp` value, which is then used to test, search,
split and rewrite strings.

```python
rx = regexp.compile("b(an)*a")

rx.matches("banana")            # True
rx.find("--banana--")           # "banana"
rx.find_submatches("banana")    # ["banana", "an"]
rx.replace_all("banana", "X")   # "X"
```

## Contents

- [Making the module available](#making-the-module-available)
- [Pattern syntax](#pattern-syntax)
- [Module functions](#module-functions)
- [The `regexp.regexp` type](#the-regexpregexp-type)
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
$ starlark -c 'print(regexp.compile("a+").find_all("banana"))'
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
  regexp.compile(".").find_all("héllo")   # ["h", "é", "l", "l", "o"]
  ```

Flags are written inside the pattern in the usual RE2 way, e.g. `(?i)` for
case-insensitive matching, `(?s)` to let `.` match a newline, and `(?m)` to make
`^` and `$` match at line boundaries.

## Module functions

| Function | Returns | Description |
| --- | --- | --- |
| `regexp.compile(pattern)` | `regexp.regexp` | Compiles `pattern`; fails if it is not a valid regular expression. |
| `regexp.quote(s)` | `string` | Returns a pattern that matches the literal text `s`. |

### `regexp.compile(pattern)`

| Parameter | Type | Default | Description |
| --- | --- | --- | --- |
| `pattern` | `string` | — | The regular expression, in RE2 syntax. |

Returns a `regexp.regexp`. Compilation errors are reported immediately:

```python
regexp.compile("(")     # error: error parsing regexp: missing closing ): `(`
regexp.compile("\\C")   # error: error parsing regexp: invalid escape sequence: `\C`
```

### `regexp.quote(s)`

| Parameter | Type | Default | Description |
| --- | --- | --- | --- |
| `s` | `string` | — | Literal text to escape. |

Returns `s` with every metacharacter escaped, so that the result is a pattern
matching exactly and only `s`. Useful for building a pattern from user input:

```python
regexp.quote("a.b")                                 # "a\\.b"
regexp.compile(regexp.quote("a.b")).matches("a.b")  # True
regexp.compile(regexp.quote("a.b")).matches("axb")  # False
```

## The `regexp.regexp` type

A compiled pattern. It is immutable, hashable, and always truthy.

```python
rx = regexp.compile("b(an)*a")
type(rx)    # "regexp.regexp"
str(rx)     # 'regexp.compile("b(an)*a")'
rx.pattern  # "b(an)*a"
```

### Field

| Field | Type | Description |
| --- | --- | --- |
| `pattern` | `string` | The source text of the regular expression. |

### Methods

| Method | Returns | Description |
| --- | --- | --- |
| [`matches(src)`](#matchessrc) | `bool` | Whether the pattern matches anywhere in `src`. |
| [`find(src)`](#findsrc) | `string` or `None` | Text of the leftmost match, or `None`. |
| [`find_all(src, max=-1)`](#find_allsrc-max-1) | `list` of `string` | Text of successive matches. |
| [`find_submatches(src)`](#find_submatchessrc) | `list` of `string` | Leftmost match and its groups. |
| [`find_all_submatches(src, max=-1)`](#find_all_submatchessrc-max-1) | `list` of `list` of `string` | Successive matches and their groups. |
| [`replace_all(src, repl)`](#replace_allsrc-repl) | `string` | `src` with every match replaced. |
| [`split(src, max=-1)`](#splitsrc-max-1) | `list` of `string` | The text between the matches. |

All arguments are positional; keyword arguments are not accepted.

Methods that take a `max` count follow a single convention: a negative `max`
(the default) means "no limit", and `max = 0` yields an empty list.

#### `matches(src)`

| Parameter | Type | Default | Description |
| --- | --- | --- | --- |
| `src` | `string` | — | The string to search. |

```python
rx = regexp.compile("b(an)*a")
rx.matches("banana")      # True
rx.matches("--banana--")  # True
rx.matches("apple")       # False
```

#### `find(src)`

| Parameter | Type | Default | Description |
| --- | --- | --- | --- |
| `src` | `string` | — | The string to search. |

Returns the text of the leftmost match, or `None` if there is none. The `None`
result is what distinguishes "no match" from a match of the empty string:

```python
regexp.compile("b(an)*a").find("--banana--")  # "banana"
regexp.compile("b(an)*a").find("apple")       # None

regexp.compile("x*").find("zzz")              # ""   — matched the empty string
regexp.compile("x+").find("zzz")              # None — no match at all
```

#### `find_all(src, max=-1)`

| Parameter | Type | Default | Description |
| --- | --- | --- | --- |
| `src` | `string` | — | The string to search. |
| `max` | `int` | `-1` | Maximum number of matches; negative means all. |

Returns the text of successive non-overlapping matches, or an empty list if
there is none.

```python
a = regexp.compile("a+")
a.find_all("banana")      # ["a", "a", "a"]
a.find_all("banana", 2)   # ["a", "a"]
a.find_all("banana", 0)   # []
a.find_all("melon")       # []
```

#### `find_submatches(src)`

| Parameter | Type | Default | Description |
| --- | --- | --- | --- |
| `src` | `string` | — | The string to search. |

Returns a list holding the text of the leftmost match followed by the text of
each subpattern (capture group), or an empty list if there is no match. A group
that did not participate in the match yields `""`.

```python
rx = regexp.compile("b(an)*a")
rx.find_submatches("banana")                    # ["banana", "an"]
rx.find_submatches("apple")                     # []

regexp.compile("a(x)?").find_submatches("a")    # ["a", ""]

kv = regexp.compile("(?P<key>\\w+)=(?P<value>\\w+)")
kv.find_submatches("a=1")                       # ["a=1", "a", "1"]
```

Named groups are ordinary numbered groups here; the names matter only in
replacement templates.

#### `find_all_submatches(src, max=-1)`

| Parameter | Type | Default | Description |
| --- | --- | --- | --- |
| `src` | `string` | — | The string to search. |
| `max` | `int` | `-1` | Maximum number of matches; negative means all. |

Returns, for each of at most `max` successive non-overlapping matches, the list
that `find_submatches` would return.

```python
kv = regexp.compile("(\\w+)=(\\w+)")
kv.find_all_submatches("a=1,b=2")     # [["a=1", "a", "1"], ["b=2", "b", "2"]]
kv.find_all_submatches("a=1,b=2", 1)  # [["a=1", "a", "1"]]
kv.find_all_submatches("nope")        # []
```

#### `replace_all(src, repl)`

| Parameter | Type | Default | Description |
| --- | --- | --- | --- |
| `src` | `string` | — | The string to rewrite. |
| `repl` | `string` or callable | — | Replacement template, or a function of one `string` argument returning a `string`. |

Returns a copy of `src` in which every non-overlapping match has been replaced.

With a string, `$1` and `${name}` expand to the corresponding group — see
[Replacement templates](#replacement-templates):

```python
a = regexp.compile("a+")
a.replace_all("banana", "o")            # "bonono"
a.replace_all("melon", "o")             # "melon"

kv = regexp.compile("(\\w+)=(\\w+)")
kv.replace_all("a=1,b=2", "$2=$1")      # "1=a,2=b"
```

With a function, it is called once per match with the matched text and must
return a string:

```python
a.replace_all("banana", lambda m: m.upper())            # "bAnAnA"
kv.replace_all("a=1,b=2", lambda m: m.replace("=", ":"))  # "a:1,b:2"
```

The function runs on the calling thread, so it is subject to the thread's
cancellation and execution-step limits, and any error it raises aborts
`replace_all` and propagates to the caller.

#### `split(src, max=-1)`

| Parameter | Type | Default | Description |
| --- | --- | --- | --- |
| `src` | `string` | — | The string to split. |
| `max` | `int` | `-1` | Maximum number of substrings; negative means all. |

Returns the strings between the matches of the pattern. If `max` substrings
would be exceeded, the last one is the unsplit remainder.

```python
comma = regexp.compile(",+")
comma.split("a,b,,c")      # ["a", "b", "c"]
comma.split("a,b,,c", 2)   # ["a", "b,,c"]
comma.split("a,b,,c", 0)   # []
comma.split("abc")         # ["abc"]
```

## Replacement templates

When `replace_all` is given a string, it is a template in which `$` has a
special meaning:

| Form | Meaning |
| --- | --- |
| `$1`, `$2`, … | The text of the numbered group. |
| `$name`, `${name}` | The text of the named group `(?P<name>…)`. |
| `$$` | A literal `$`. |

Use the braced form whenever the name is followed by other word characters:
`$1x` names the group `1x`, while `${1}x` is group 1 followed by `x`. A
reference to a group that does not exist expands to the empty string.

```python
kv = regexp.compile("(?P<key>\\w+)=(?P<value>\\w+)")
kv.replace_all("a=1", "${value}=${key}")  # "1=a"
kv.replace_all("a=1", "$nope")            # ""
regexp.compile("x").replace_all("x", "$$1")  # "$1"
```

To insert text that itself contains a `$` — text supplied by the user, say —
either double each dollar sign, or pass a function, whose return value is used
verbatim:

```python
rx = regexp.compile("x")
rx.replace_all("x", "costs $$5")           # "costs $5"
rx.replace_all("x", lambda m: "costs $5")  # "costs $5"
```

## Things worth knowing

- **No match versus empty match.** `find` returns `None` when nothing matched
  and a string (possibly `""`) when something did; `find_submatches` and
  `find_all*` return an empty list.
- **Equality is identity.** A `regexp.regexp` may be used as a dict key or set
  element, but two separate `compile` calls produce distinct values, even for
  the same pattern:

  ```python
  regexp.compile("a") == regexp.compile("a")  # False
  rx = regexp.compile("a")
  rx == rx                                    # True
  ```

  Compare `rx.pattern` instead if you need pattern equality.
- **No pattern cache.** Each `compile` call parses the pattern afresh, so hoist
  `compile` out of loops and, where possible, to the top level of a module.
- **Matching is not metered.** Match time is linear in the length of the subject
  string, but it is not counted against the thread's execution step budget
  (`Thread.SetMaxExecutionSteps`). Applications running untrusted programs
  should bound their resources by other means; see the "Bounding time, memory,
  and other resources" section of the [top-level README](../../README.md).
