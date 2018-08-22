# Example: a web server application with policies defined in Skylark

This article presents an example application that uses Skylark as its
configuration language. It is a web server that accepts or rejects
each incoming HTTP request based on a policy defined by a function
implemented in Skylark.

Many applications use a configuration file to set parameters, define
customizations, or enable optional features. For the designer of a
configuration language for an application, Skylark may be attractive
if for no other reason than that it is familiar, rational, and well
documented, but the example below illustrates a compelling benefit
of Skylark over alternative languages: Skylark functions, despite
being implemented in the familiar paradigm of imperative programming,
may be called concurrently in a highly parallel application without
the possibility of a data race, thanks to Skylark's "freeze"
mechanism.

Let's take a look at the program, which we'll present in parts.
Here's its main function:

```go
var hook *skylark.Function

func main() {
	// Load the configuration.
	thread := new(skylark.Thread)
	globals := make(skylark.StringDict)
	if err := skylark.ExecFile(thread, "server.conf", nil, globals); err != nil {
		log.Fatalf("error in config file: %v", err)
	}
	hook, _ = globals["hook"].(*skylark.Function)
	if hook == nil {
		log.Fatalf("config file doesn't define 'hook' function")
	}

	// Run web server.
	log.Fatal(http.ListenAndServe(":8000", http.HandlerFunc(serveHTTP)))
}
```

The main function loads the Skylark configuration file, `server.conf`.
To execute a Skylark file, we must create a Skylark thread and new
dictionary for the global variables of the module.
There is a certain amount of boilerplate,
but the important part is the call to `ExecFile`.
If execution of the configuration file was successful,
the application expects that it defines a global
variable named `hook`, a function; otherwise, it issues an error.
The application saves the hook function in a Go global variable, also named `hook`.
Finally, the main function starts a web server listening on port 8000.

Here's the web server's request handler function:

```go
// serveHTTP is a trivial HTTP request handler.
func serveHTTP(w http.ResponseWriter, req *http.Request) {
	if err := validate(req); err != nil {
		fmt.Fprintln(w, "Error: ", err)
		return
	}
	fmt.Fprintln(w, "OK")
}
```

As you can see, it is trivial: it simply prints "OK" for each request.
However, it first calls the `validate` function to decide whether to
proceed with the request or to reject it:

```go
// validate calls passes the HTTP request to the Skylark hook function.
func validate(req *http.Request) error {
	args := skylark.Tuple{httpRequest{req}}
	x, err := skylark.Call(new(skylark.Thread), hook, args, nil)
	if err != nil {
		return err // hook evaluation failed
	} else if msg, ok := skylark.AsString(x); ok {
		return errors.New(msg) // hook returned an error message
	} else if x != skylark.None {
		return fmt.Errorf("hook returned %s, want string or None", x.Type())
	}
	return nil // success
}
```

Again, there is more boilerplate to create a new Skylark thread and
package the sole argument as a one-element tuple, but the important
part here is `skylark.Call`, which calls the Skylark `hook` function.

The validate function takes a parameter of type `*http.Request`. We'd
like to make this value accessible to the Skylark hook function so
that it can make its policy decision based on attributes of the HTTP
request such as the request URL and query parameters.
So, we define a new type, `httpRequest`, whose values each wrap an
`*http.Request` and satisfy the `skylark.Value` interface, allowing
them to be passed to the Skylark program.

In addition, to the basic methods of `skylark.Value` (which for
brevity we have not shown, but they are each no longer than a single
line) the `httpRequest` type defines the `Attr` method, and thus
satisfies the `skylark.HasAttrs` interface. A value with an `Attr`
method has named attributes (fields or methods) accessible using dot
notation, such as `req.url`. Our `httpRequest` wrapper type provides
only the `url` and `query` attributes, but it would be easy to add
more:

```go
// httpRequest is a a Skylark value that wraps an http.Request.
type httpRequest struct{ req *http.Request }

func (r httpRequest) Attr(name string) (skylark.Value, error) {
	switch name {
	case "query":
		query := new(skylark.Dict)
		for k, v := range r.req.URL.Query() {
			query.Set(skylark.String(k), skylark.String(v[0]))
		}
		return query, nil
	case "url":
		return skylark.String(r.req.URL.Path), nil
	}
	return nil, nil
}
```

The `Attr` function switches on the name of the attribute.
The `query` case builds and returns a Skylark dictionary containing
the HTTP request parameters; the `url` case returns the path component
of the request URL.

Not shown are half a dozen other methods of `httpRequest`, each no
more than one line, required to fulfil the `skylark.Value` and
`skylark.HasAttrs` interfaces.

Finally, let's look at a simple server configuration file:

```python
# server.conf

def hook(req):
    print("url=%s, query=%s" % (req.url, req.query))

    if req.url == '/food' and req.query['name'] == 'soup':
       return "no soup for you!"

    return None # ok
```

The file defines a simple hook function that returns `None` (no error)
in the success case, or an error when it sees certain request parameters.

A real example would likely be much more complicated, and might
consult tables of data generated earlier within the `server.conf` file.

Let's see how this program behaves:

```shell
TODO: demo
running time measured in microseconds.
```

Explain significance
- familiar style
- easy to use
- concurrency safe
  Go web servers are concurrent
  no data race here
  not possible to have a Skylark data race here because ExecFile froze everything.
-
