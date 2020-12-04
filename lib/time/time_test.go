/*
	TODO:
	- unix timet conversions
	- timezone stuff
	- strftime formatting
	- constructor from 6 components + location
*/
package time

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarktest"
)

func TestFile(t *testing.T) {
	resolve.AllowFloat = true
	thread := &starlark.Thread{Load: newLoader(LoadModule, "time")}
	starlarktest.SetReporter(thread, t)

	// Execute test file
	_, err := starlark.ExecFile(thread, "testdata/test.star", nil, nil)
	if err != nil {
		t.Error(err)
	}
}

// Newloader implements the 'load' operation as used in the evaluator tests.
// takes a LoadModule function
// a ModuleName
// and the relative path to the testdata
func newLoader(loader func() (starlark.StringDict, error), moduleName string) func(thread *starlark.Thread, module string) (starlark.StringDict, error) {
	return func(thread *starlark.Thread, module string) (starlark.StringDict, error) {
		switch module {
		case moduleName:
			return loader()
		case "assert.star":
			starlarktest.DataFile = func(pkgdir, filename string) string {
				_, currFileName, _, ok := runtime.Caller(1)
				if !ok {
					return ""
				}
				return filepath.Join(filepath.Dir(currFileName), filename)
			}
			return starlarktest.LoadAssertModule()
		}

		return nil, fmt.Errorf("invalid module")
	}
}
