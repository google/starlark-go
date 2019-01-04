# An example of using the go/parser package.

load("go", token="go/token", parser="go/parser")
load("assert.star", "assert")

fset = token.NewFileSet()
f, err = parser.ParseFile(fset, "hello.go", 'package main; var x, y = 1, 2', parser.Mode(0))
assert.eq(err, None)

# parser.Mode: a named integer type
mode = parser.Mode(0)
assert.eq(str(mode), '0')
assert.eq(type(mode), 'go.uint<parser.Mode>')
assert.eq(go.typeof(mode), parser.Mode)
assert.ne(mode, 0) # different types
assert.eq(go.typeof(parser.PackageClauseOnly), parser.Mode)
assert.eq(go.int(parser.PackageClauseOnly), 1)

# *ast.File
assert.eq('go.ptr<*ast.File>', type(f))
assert.eq('go.slice<[]ast.Decl>', type(f.Decls))
assert.eq('go.ptr<*ast.GenDecl>', type(f.Decls[0]))
assert.eq('go.slice<[]ast.Spec>', type(f.Decls[0].Specs))
assert.eq('go.ptr<*ast.ValueSpec>', type(f.Decls[0].Specs[0]))
assert.eq('go.slice<[]*ast.Ident>', type(f.Decls[0].Specs[0].Names))
assert.eq('go.ptr<*ast.Ident>', type(f.Decls[0].Specs[0].Names[0]))
assert.eq('x', f.Decls[0].Specs[0].Names[0].Name)

# token.Pos: another named integer type, with methods.
pos = f.Decls[0].Specs[0].Names[0].Pos()
assert.true(pos.IsValid())
assert.eq(str(fset.Position(pos)), "hello.go:1:19")
assert.eq(fset.Position(pos).Filename, "hello.go")
assert.eq(fset.Position(pos).Column, 19)
