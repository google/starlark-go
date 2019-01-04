package stargo_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"reflect"
	"testing"
	"unsafe"

	"go.starlark.net/internal/chunkedfile"
	"go.starlark.net/resolve"
	"go.starlark.net/stargo"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarktest"
)

// TODO: add test of mixed Starlark/Go backtrace.

func init() {
	// The tests make extensive use of these non-standard features.
	resolve.AllowLambda = true
	resolve.AllowNestedDef = true
	resolve.AllowFloat = true
	resolve.AllowBitwise = true
	resolve.AllowGlobalReassign = true
	resolve.AllowAddressing = true
}

func TestExecFile(t *testing.T) {
	testdata := starlarktest.DataFile("stargo", ".")
	thread := &starlark.Thread{Load: load}
	starlarktest.SetReporter(thread, t)
	for _, file := range []string{
		"testdata/addr.star",
		"testdata/array.star",
		"testdata/bytes.star",
		"testdata/chan.star",
		"testdata/complex.star",
		"testdata/func.star",
		"testdata/http.star",
		"testdata/int.star",
		"testdata/map.star",
		"testdata/parser.star",
		"testdata/ptr.star",
		"testdata/slice.star",
		"testdata/string.star",
		"testdata/unsafepointer.star",
	} {
		filename := filepath.Join(testdata, file)
		for _, chunk := range chunkedfile.Read(filename, t) {

			predeclared := starlark.StringDict{
				"go":              stargo.Builtins,
				"predeclared_var": stargo.VarOf(&V),
			}

			_, err := starlark.ExecFile(thread, filename, chunk.Source, predeclared)
			switch err := err.(type) {
			case *starlark.EvalError:
				found := false
				for _, fr := range err.Stack() {
					posn := fr.Position()
					if posn.Filename() == filename {
						chunk.GotError(int(posn.Line), err.Error())
						found = true
						break
					}
				}
				if !found {
					t.Error("\n", err.Backtrace())
				}
			case nil:
				// success
			default:
				t.Error("\n", err)
			}
			chunk.Done()
		}
	}
}

// load implements the 'load' operation as used in the evaluator tests.
func load(thread *starlark.Thread, module string) (starlark.StringDict, error) {
	switch module {
	case "assert.star":
		return starlarktest.LoadAssertModule()
	case "go":
		return goPackages, nil
	}
	return nil, fmt.Errorf("no such module")
}

// Some typical Go packages for testing.
var goPackages = starlark.StringDict{
	"fmt": &starlark.Module{
		Name: "fmt",
		Members: starlark.StringDict{
			"Errorf":   stargo.ValueOf(fmt.Errorf),
			"Fprintf":  stargo.ValueOf(fmt.Fprintf),
			"Sprintf":  stargo.ValueOf(fmt.Sprintf),
			"Stringer": stargo.TypeOf(reflect.TypeOf(new(fmt.Stringer)).Elem()),
		},
	},
	"bytes": &starlark.Module{
		Name: "bytes",
		Members: starlark.StringDict{
			"Buffer":      stargo.TypeOf(reflect.TypeOf(new(bytes.Buffer)).Elem()),
			"ErrTooLarge": stargo.VarOf(&bytes.ErrTooLarge),
			"Split":       stargo.ValueOf(bytes.Split),
		},
	},
	"io/ioutil": &starlark.Module{
		Name: "io/ioutil",
		Members: starlark.StringDict{
			"ReadAll": stargo.ValueOf(ioutil.ReadAll),
		},
	},
	"net/http": &starlark.Module{
		Name: "net/http",
		Members: starlark.StringDict{
			"Get":    stargo.ValueOf(http.Get),
			"Header": stargo.TypeOf(reflect.TypeOf(new(http.Header)).Elem()),
		},
	},
	"encoding/json": &starlark.Module{
		Name: "encoding/json",
		Members: starlark.StringDict{
			"MarshalIndent": stargo.ValueOf(json.MarshalIndent),
		},
	},
	"stargo_test": &starlark.Module{
		Name: "stargo_test",
		Members: starlark.StringDict{
			"myint16": stargo.TypeOf(reflect.TypeOf(myint16(0))),
			"A":       stargo.TypeOf(reflect.TypeOf(new(A)).Elem()),
			"B":       stargo.TypeOf(reflect.TypeOf(new(B)).Elem()),
			"V":       stargo.VarOf(&V),
			"U":       stargo.TypeOf(reflect.TypeOf(new(U)).Elem()),
		},
	},
	"go/token": &starlark.Module{
		Name: "go/token",
		Members: starlark.StringDict{
			"FileSet":    stargo.TypeOf(reflect.TypeOf(token.FileSet{})),
			"NewFileSet": stargo.ValueOf(token.NewFileSet),
			"Pos":        stargo.TypeOf(reflect.TypeOf(token.NoPos)),
		},
	},
	"go/parser": &starlark.Module{
		Name: "go/parser",
		Members: starlark.StringDict{
			"Mode":              stargo.TypeOf(reflect.TypeOf(parser.Mode(0))),
			"PackageClauseOnly": stargo.ValueOf(parser.PackageClauseOnly),
			"ParseFile":         stargo.ValueOf(parser.ParseFile),
		},
	},
	"unsafe": &starlark.Module{
		Name: "unsafe",
		Members: starlark.StringDict{
			"Pointer": stargo.TypeOf(reflect.TypeOf(new(unsafe.Pointer)).Elem()),
		},
	},
}

type myint16 int16

func (i myint16) Get() int { return int(i) }
func (i *myint16) Incr()   { *i++ }

type A struct {
	P *B
	V B
}

type B struct {
	A [1]int
}

func (B) V()  {}
func (*B) P() {}

var V bytes.Buffer

type U struct{ F float64 }
