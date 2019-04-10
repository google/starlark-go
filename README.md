
<!-- This file is the project homepage for go.starlark.net -->

# Starlark in Go

This is the home of the _Starlark in Go_ project.
Starlark in Go is an interpreter for Starlark, implemented in Go.
Starlark was formerly known as Skylark.
The new import path for Go packages is `"go.starlark.net/starlark"`.

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

* API documentation: [godoc.org/go.starlark.net/starlark](https://godoc.org/go.starlark.net/starlark)

* Mailing list: [starlark-go](https://groups.google.com/forum/#!forum/starlark-go)

* Issue tracker: [https://github.com/google/starlark-go/issues](https://github.com/google/starlark-go/issues)

* Travis CI: [![Travis CI](https://travis-ci.org/google/starlark-go.svg) https://travis-ci.org/google/starlark-go](https://travis-ci.org/google/starlark-go)

### Getting started

Build the code:

```shell
# check out the code and dependencies,
# and install interpreter in $GOPATH/bin
$ go get -u go.starlark.net/cmd/starlark
```

Run the interpreter:

```
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

```
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

```
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

### Credits

Starlark was designed and implemented in Java by
Ulf Adams,
Lukács Berki,
Jon Brandvein,
John Field,
Laurent Le Brun,
Dmitry Lomov,
Damien Martin-Guillerez,
Vladimir Moskva, and
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
