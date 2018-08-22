// The http command runs a trivial web server whose Skylark configuration
// file defines a policy of whether to accept or reject each request.
// The server loads its configuration once at startup, so its policy may
// be changed by restarting it with a new configuration;
// no recompilation is necessary.
//
// This example program is explained by [TODO: URL].
package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/skylark"
)

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

// serveHTTP is a trivial HTTP request handler.
func serveHTTP(w http.ResponseWriter, req *http.Request) {
	// Log the time (not shown talk slides).
	t0 := time.Now()
	defer func() { fmt.Fprintln(w, time.Since(t0)) }()

	if err := validate(req); err != nil {
		fmt.Fprintln(w, "\n\n\n\nError: ", err)
		return
	}
	fmt.Fprintln(w, "\n\n\n\nOK")
}

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
	}
	case "url":
		return skylark.String(r.req.URL.Path), nil
	return nil, nil
}

func (r httpRequest) AttrNames() []string   { return []string{"query", "url"} }
func (r httpRequest) Freeze()               {}
func (r httpRequest) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: httpRequest") }
func (r httpRequest) String() string        { return fmt.Sprint(r.req) }
func (r httpRequest) Type() string          { return "http.Request" }
func (r httpRequest) Truth() skylark.Bool   { return true }
