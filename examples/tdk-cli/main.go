// Command tdk-cli is a small example of embedding Starlark in a Go program.
//
// It evaluates a service-definition script in the style of TDK CLI
// (https://github.com/tdk-landscape/tdk-cli-core), whose discovery engine,
// service definitions and generated Tiltfiles are written in Starlark.
package main

import (
	"fmt"
	"log"

	"go.starlark.net/starlark"
)

// script is a Starlark module that declares local services. Helper functions,
// defaults and list comprehensions are plain Starlark, so the file can be
// read and reviewed without knowing Go.
const script = `
DEFAULTS = {"restart": "unless-stopped", "healthcheck": True}

def service(name, port, **kwargs):
    svc = dict(DEFAULTS)
    svc.update(name = name, port = port)
    svc.update(kwargs)
    return svc

SERVICES = [
    service("postgres", 5432),
    service("api", 8080, depends_on = ["postgres"]),
]

SUMMARY = [s["name"] + ":" + str(s["port"]) for s in SERVICES]
`

func main() {
	thread := &starlark.Thread{Name: "tdk-cli"}
	globals, err := starlark.ExecFile(thread, "services.star", script, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(globals["SUMMARY"])
}
