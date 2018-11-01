//+build ignore

// The update command creates/updates the <html><head> elements of
// each subpackage beneath docs so that "go get" requests redirect
// to GitHub and other HTTP requests redirect to godoc.corp.
//
// Usage:
//
//   $ cd $GOPATH/src/go.starlark.net
//   $ go run docs/update.go
//
package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("update: ")

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	if filepath.Base(cwd) != "go.starlark.net" {
		log.Fatalf("must run from the go.starlark.net directory")
	}

	cmd := exec.Command("go", "list", "./...")
	cmd.Stdout = new(bytes.Buffer)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
	for _, pkg := range strings.Split(strings.TrimSpace(fmt.Sprint(cmd.Stdout)), "\n") {
		rel := strings.TrimPrefix(pkg, "go.starlark.net/") // e.g. "cmd/starlark"
		subdir := filepath.Join("docs", rel)
		if err := os.MkdirAll(subdir, 0777); err != nil {
			log.Fatal(err)
		}

		// Create missing docs/$rel/index.html files.
		html := filepath.Join(subdir, "index.html")
		if _, err := os.Stat(html); os.IsNotExist(err) {
			data := strings.Replace(defaultHTML, "$PKG", pkg, -1)
			if err := ioutil.WriteFile(html, []byte(data), 0666); err != nil {
				log.Fatal(err)
			}
			log.Printf("created %s", html)
		}
	}
}

const defaultHTML = `<html>
<head>
  <meta name="go-import" content="go.starlark.net git https://github.com/google/starlark-go"></meta>
  <meta http-equiv="refresh" content="0;URL='http://godoc.org/$PKG'" /></meta>
</head>
<body>
  Redirecting to godoc.org page for $PKG...
</body>
</html>
`
