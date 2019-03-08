package syntax

// This file defines resolver data types referenced by the syntax tree.
// We cannot guarantee API stability for these types
// as they are closely tied to the implementation.

// A Binding ties together all identifiers that denote the same variable.
// The resolver computes a binding for every Ident.
type Binding struct {
	Scope Scope

	// Index records the index into the enclosing
	// - {DefStmt,File}.Locals, if Scope==Local
	// - DefStmt.FreeVars,      if Scope==Free
	// - File.Globals,          if Scope==Global.
	// It is zero if Scope is Predeclared, Universal, or Undefined.
	Index int

	First *Ident // first binding use (iff Scope==Local/Free/Global)
}

// The Scope of Binding indicates what kind of scope it has.
type Scope uint8

const (
	UndefinedScope   Scope = iota // name is not defined
	LocalScope                    // name is local to its function
	CellScope                     // name is local but shared with a nested function
	FreeScope                     // name is cell of some enclosing function
	GlobalScope                   // name is global to module
	PredeclaredScope              // name is predeclared for this module (e.g. glob)
	UniversalScope                // name is universal (e.g. len)
)

var scopeNames = [...]string{
	UndefinedScope:   "undefined",
	LocalScope:       "local",
	FreeScope:        "free",
	CellScope:        "cell",
	GlobalScope:      "global",
	PredeclaredScope: "predeclared",
	UniversalScope:   "universal",
}

func (scope Scope) String() string { return scopeNames[scope] }
