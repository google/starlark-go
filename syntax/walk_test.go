package syntax_test

import (
	"bytes"
	"fmt"
	"log"
	"reflect"
	"strings"
	"testing"

	"go.starlark.net/syntax"
)

func TestWalk(t *testing.T) {
	const src = `
for x in y:
  if x:
    pass
  else:
    f([2*x for x in "abc"])
`
	// TODO(adonovan): test that it finds all syntax.Nodes
	// (compare against a reflect-based implementation).
	// TODO(adonovan): test that the result of f is used to prune
	// the descent.
	f, err := syntax.Parse("hello.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	var depth int
	syntax.Walk(f, func(n syntax.Node) bool {
		if n == nil {
			depth--
			return true
		}
		fmt.Fprintf(&buf, "%s%s\n",
			strings.Repeat("  ", depth),
			strings.TrimPrefix(reflect.TypeOf(n).String(), "*syntax."))
		depth++
		return true
	})
	got := buf.String()
	want := `
File
  ForStmt
    Ident
    Ident
    IfStmt
      Ident
      BranchStmt
      ExprStmt
        CallExpr
          Ident
          Comprehension
            BinaryExpr
              Literal
              Ident
            ForClause
              Ident
              Literal`
	got = strings.TrimSpace(got)
	want = strings.TrimSpace(want)
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

// ExampleWalk demonstrates the use of Walk to
// enumerate the identifiers in a Starlark source file
// containing a nonsense program with varied grammar.
func ExampleWalk() {
	const src = `
load("library", "a")

def b(c, *, d=e):
    f += {g: h}
    i = -(j)
    return k.l[m + n]

for o in [p for q, r in s if t]:
    u(lambda: v, w[x:y:z])
`
	f, err := syntax.Parse("hello.star", src, 0)
	if err != nil {
		log.Fatal(err)
	}

	var idents []string
	syntax.Walk(f, func(n syntax.Node) bool {
		if id, ok := n.(*syntax.Ident); ok {
			idents = append(idents, id.Name)
		}
		return true
	})
	fmt.Println(strings.Join(idents, " "))

	// The identifier 'a' appears in both LoadStmt.From[0] and LoadStmt.To[0].

	// Output:
	// a a b c d e f g h i j k l m n o p q r s t u v w x y z
}
