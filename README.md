
<!-- This file is the project homepage for go.starlark.net -->

# Starlark in Go

[![Go Tests](https://github.com/google/starlark-go/actions/workflows/tests.yml/badge.svg)](https://github.com/google/starlark-go/actions/workflows/tests.yml)
[![Go Reference](https://pkg.go.dev/badge/go.starlark.net/starlark.svg)](https://pkg.go.dev/go.starlark.net/starlark)

This is the home of the _Starlark in Go_ project.
Starlark in Go is an interpreter for Starlark, implemented in Go.
Starlark was formerly known as Skylark.
The import path for the Go package is `"go.starlark.net/starlark"`.

Starlark is a dialect of Python intended for use as a configuration language.
Like Python, it is an untyped dynamic language with high-level data
types, first-class functions with lexical scope, and garbage collection.
Unlike CPython, independent Starlark threads execute in parallel, so
Starlark workloads scale well on parallel machines.
Starlark is a small and simple language with a familiar and highly
readable syntax. You can use it as an expressive notation for
structured data, defining functions to eliminate repetition, or you
can use it to add scripting capabilities to an existing application.

A Starlark interpreter is typically embedded within a larger
application, and the application may define additional domain-specific
functions and data types beyond those provided by the core language.
For example, Starlark was originally developed for the
[Bazel build tool](https://bazel.build).
Bazel uses Starlark as the notation both for its BUILD files (like
Makefiles, these declare the executables, libraries, and tests in a
directory) and for [its macro
language](https://docs.bazel.build/versions/master/skylark/language.html),
through which Bazel is extended with custom logic to support new
languages and compilers.


## Documentation

* Language definition: [doc/spec.md](doc/spec.md)

* About the Go implementation: [doc/impl.md](doc/impl.md)

* API documentation: [pkg.go.dev/go.starlark.net/starlark](https://pkg.go.dev/go.starlark.net/starlark)

* Mailing list: [starlark-go](https://groups.google.com/forum/#!forum/starlark-go)

* Issue tracker: [https://github.com/google/starlark-go/issues](https://github.com/google/starlark-go/issues)

### Getting started

Build the code:

```shell
# check out the code and dependencies,
# and install interpreter in $GOPATH/bin
$ go install go.starlark.net/cmd/starlark@latest
```

Run the interpreter:

```console
$ cat coins.star
coins = {
  'dime': 10,
  'nickel': 5,
  'penny': 1,
  'quarter': 25,
}
print('By name:\t' + ', '.join(sorted(coins.keys())))
print('By value:\t' + ', '.join(sorted(coins.keys(), key=coins.get)))

$ starlark coins.star
By name:	dime, nickel, penny, quarter
By value:	penny, nickel, dime, quarter
```

Interact with the read-eval-print loop (REPL):

```pycon
$ starlark
>>> def fibonacci(n):
...    res = list(range(n))
...    for i in res[2:]:
...        res[i] = res[i-2] + res[i-1]
...    return res
...
>>> fibonacci(10)
[0, 1, 1, 2, 3, 5, 8, 13, 21, 34]
>>>
```

When you have finished, type `Ctrl-D` to close the REPL's input stream.

Embed the interpreter in your Go program:

```go
import "go.starlark.net/starlark"

// Execute Starlark program in a file.
thread := &starlark.Thread{Name: "my thread"}
globals, err := starlark.ExecFile(thread, "fibonacci.star", nil, nil)
if err != nil { ... }

// Retrieve a module global.
fibonacci := globals["fibonacci"]

// Call Starlark function from Go.
v, err := starlark.Call(thread, fibonacci, starlark.Tuple{starlark.MakeInt(10)}, nil)
if err != nil { ... }
fmt.Printf("fibonacci(10) = %v\n", v) // fibonacci(10) = [0, 1, 1, 2, 3, 5, 8, 13, 21, 34]
```

See [starlark/example_test.go](starlark/example_test.go) for more examples.

### Bounding time, memory, and other resources

It is trivial for a Starlark program to consume all available time and
memory. Even without recursion, a program can get the interpreter stuck
in an endless loop, for example by applying a recursive built-in operator
to a cyclic value such as a list that contains itself.
So if you evaluate Starlark code from an untrusted source in the same
address space as your application, you are trusting the health of your
application to that source.

The interpreter provides two mechanisms for bounding computation:

- `Thread.SetMaxExecutionSteps` limits the number of abstract computation
  steps a thread may take. The measure is deterministic and reproducible,
  but it does not correspond to CPU time: not all steps are equal, and a
  single step may be a call to a built-in function or operator that is
  very slow or allocates a lot of memory (e.g. `string * int`).
  Set `Thread.OnMaxSteps` to change what happens when the limit is reached.

- `Thread.Cancel` asynchronously interrupts a running thread. The
  interpreter polls an atomic variable rather than a channel or
  `context.Context`, as this is much faster. To cancel a thread when a
  context is done, use a goroutine:

  ```go
  ctx, cancel := context.WithCancel(ctx)
  defer cancel()
  go func() {
      <-ctx.Done()
      thread.Cancel("context cancelled")
  }()
  ... evaluate Starlark in thread ...
  ```

  Long-running built-in functions can honor cancellation too if you pass
  the context to them using `thread.SetLocal`.

There is no way to bound the memory used by a Starlark thread. The
interpreter rejects single allocations that are unreasonably large, but a
thread can allocate an unbounded amount of memory in smaller steps.
Accounting for the live memory held by a thread is not even well defined,
since frozen values may be shared among threads, and it would require the
mark phase of a garbage collector capable of tracing both the Starlark
heap and the Go heap. Similarly, the interpreter imposes a limit on stack
depth to prevent recursion from overflowing the Go stack, but it does not
promise what that limit is, other than "enough in most cases".

If you need to defend against denial of service, the only reliable
approach is to let the operating system help you: evaluate Starlark in a
separate process with a tight limit on memory (e.g. using ulimit,
setrlimit, or cgroups), terminate it if it takes too much wall or CPU
time, and handle OOM failures, timeouts, and crashes in the parent. The
interpreter starts very quickly, so the overhead is usually modest.

For past discussions of this topic, see
[these issues](https://github.com/google/starlark-go/issues?q=is:issue+(160+OR+236+OR+252+OR+410+OR+470+OR+498+OR+606+OR+617+OR+621+OR+661)+NOT+PEP+NOT+"byte+code").

### Contributing

We welcome submissions but please let us know what you're working on
if you want to change or add to the Starlark repository.

Before undertaking to write something new for the Starlark project,
please file an issue or claim an existing issue.
All significant changes to the language or to the interpreter's Go
API must be discussed before they can be accepted.
This gives all participants a chance to validate the design and to
avoid duplication of effort.

Despite some differences, the Go implementation of Starlark strives to
match the behavior of [the Java implementation](https://github.com/bazelbuild/bazel)
used by Bazel and maintained by the Bazel team.
For that reason, proposals to change the language itself should
generally be directed to [the Starlark site](
https://github.com/bazelbuild/starlark/), not to the maintainers of this
project.
Only once there is consensus that a language change is desirable may
its Go implementation proceed.

We use GitHub pull requests for contributions.

Please complete Google's contributor license agreement (CLA) before
sending your first change to the project.  If you are the copyright
holder, you will need to agree to the
[individual contributor license agreement](https://cla.developers.google.com/about/google-individual),
which can be completed online.
If your organization is the copyright holder, the organization will
need to agree to the [corporate contributor license agreement](https://cla.developers.google.com/about/google-corporate).
If the copyright holder for your contribution has already completed
the agreement in connection with another Google open source project,
it does not need to be completed again.

### Stability

We reserve the right to make breaking language and API changes at this
stage in the project, although we will endeavor to keep them to a minimum.
Once the Bazel team has finalized the version 1 language specification,
we will be more rigorous with interface stability.

We aim to support the most recent two (go1.x) releases of the Go
toolchain. For example, if the latest release is go1.26, we support it
along with go1.25, but not go1.24. This mirrors the support window of
the golang.org/x modules we depend upon. It is impractical to support
older versions.

### Credits

Starlark was designed and implemented in Java by
Jon Brandvein,
Alan Donovan,
Laurent Le Brun,
Dmitry Lomov,
Vladimir Moskva,
François-René Rideau,
Gergely Svigruha, and
Florian Weikert,
standing on the shoulders of the Python community.
The Go implementation was written by Alan Donovan and Jay Conrod;
its scanner was derived from one written by Russ Cox.

### Legal

Starlark in Go is Copyright (c) 2018 The Bazel Authors.
All rights reserved.

It is provided under a 3-clause BSD license:
[LICENSE](https://github.com/google/starlark-go/blob/master/LICENSE).

Starlark in Go is not an official Google product.
