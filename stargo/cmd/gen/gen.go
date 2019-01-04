// The gen command, given an list of package patterns, generates Go
// declarations for stargo bindings for those packages.
package main

import (
	"flag"
	"fmt"
	"go/types"
	"log"
	"os"

	"golang.org/x/tools/go/packages"
)

var p = flag.String("p", "main", "name in generated source file's package declaration")
var o = flag.String("o", "/dev/stdout", "output file")

func main() {
	log.SetPrefix("gen")
	log.SetFlags(0)

	flag.Parse()
	cfg := &packages.Config{Mode: packages.LoadTypes}
	pkgs, err := packages.Load(cfg, flag.Args()...)
	if err != nil {
		log.Fatal(err)
	}
	if packages.PrintErrors(pkgs) > 0 {
		os.Exit(1)
	}

	out, err := os.Create(*o)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Fprintf(out, "package %s\n\n", *p)

	fmt.Fprintf(out, "import (\n")
	fmt.Fprintf(out, "\t%q\n", "reflect")
	fmt.Fprintf(out, "\t%q\n", "go.starlark.net/stargo")
	fmt.Fprintf(out, "\t%q\n", "go.starlark.net/starlark")
	fmt.Fprintf(out, "\n")

	for i, pkg := range pkgs {
		fmt.Fprintf(out, "\tπ%d %q\n", i, pkg.PkgPath)
	}
	fmt.Fprintf(out, ")\n\n")

	fmt.Fprintf(out, "var goPackages = starlark.StringDict{\n")
	for i, pkg := range pkgs {
		fmt.Fprintf(out, "\t%q: &starlark.Module{\n", pkg.PkgPath)
		fmt.Fprintf(out, "\t\tName: %q,\n", pkg.PkgPath)

		fmt.Fprintf(out, "\t\tMembers: starlark.StringDict{\n")
		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			if obj := scope.Lookup(name); obj.Exported() {
				fmt.Fprintf(out, "\t\t\t%q: stargo.", name)

				switch obj.(type) {
				case *types.TypeName:
					fmt.Fprintf(out, "TypeOf(reflect.TypeOf(new(π%d.%s)).Elem()),\n", i, name)
				case *types.Var:
					fmt.Fprintf(out, "VarOf(&π%d.%s),\n", i, name)
				default: // func, const
					fmt.Fprintf(out, "ValueOf(π%d.%s),\n", i, name)
				}
			}
		}
		fmt.Fprintf(out, "\t\t},\n")

		fmt.Fprintf(out, "\t},\n")
	}
	fmt.Fprintf(out, "}\n")

	if err := out.Close(); err != nil {
		log.Fatal(err)
	}
}
