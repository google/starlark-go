# This file defines the iterator_leak analyzer, which checks for
# calls to the starlark.Iterator function (or Iterator.Iterate method)
# without a corresponding call to Done, or where the Done call exists
# but is preceded by a return statement.
#
# It is written in Starlark using Stargo,
# as a proof-of-concept of dynamically loaded checkers.

load(
    "go",
    analysis = "golang.org/x/tools/go/analysis",
    ast = "go/ast",
    inspect = "golang.org/x/tools/go/analysis/passes/inspect",
)

def run(pass_):
    # Short cut: inspect only packages that directly import starlark.Value.
    if "go.starlark.net/starlark" not in [p.Path() for p in pass_.Pkg.Imports()]:
        return None, None

    inspector = pass_.ResultOf[inspect.Analyzer]

    # hot types
    assign_stmt = *ast.AssignStmt
    selector_expr = *ast.SelectorExpr
    call_expr = *ast.CallExpr
    return_stmt = *ast.ReturnStmt

    iterators = {}  # maps iterator *types.Var to Iterate *ast.CallExpr

    def visit(n, push, stack):
        t = go.typeof(n)
        if t == assign_stmt:
            if (len(n.Lhs) == 1 and
                len(n.Rhs) == 1 and
                go.typeof(n.Rhs[0]) == call_expr and
                go.typeof(n.Rhs[0].Fun) == selector_expr and
                n.Rhs[0].Fun.Sel.Name == "Iterate" and
                go.typeof(n.Lhs[0]) == *ast.Ident):
                # n is one of:
                #   iter = value.Iterate()
                #   iter = starlark.Iterate(...)
                # TODO: check that it's our Iterate method/func and not some other.
                var = pass_.TypesInfo.ObjectOf(n.Lhs[0])
                iterators[var] = n.Rhs[0]

        elif t == call_expr:
            if (go.typeof(n.Fun) == selector_expr and
                n.Fun.Sel.Name == "Done" and
                go.typeof(n.Fun.X) == *ast.Ident):
                # n is iter.Done().
                var = pass_.TypesInfo.ObjectOf(n.Fun.X)
                iterators.pop(var, None) # delete

        elif t == return_stmt:
            # Report iterators leaked by an early return
            if len(iterators) > 0:
              if (len(stack) > 3 and
                  go.typeof(stack[-3]) == *ast.IfStmt and
                  go.typeof(stack[-3].Cond) == *ast.BinaryExpr and
                  go.typeof(stack[-3].Cond.X) == *ast.Ident and
                  pass_.TypesInfo.ObjectOf(stack[-3].Cond.X) in iterators):
                  pass # Allow:  if iter == ... { return ... }
              else:
                  for var, call in iterators.items():
                      pass_.Reportf(n.Return, "iterator leak (missing defer %s.Done())", var.Name())
                      iterators.pop(var, None) # delete

        return True

    node_types = (
        assign_stmt(),
        call_expr(),
        return_stmt(),
    )
    inspector.WithStack(node_types, visit)

    # Report iterators for which there is no Done call.
    for var, call in iterators.items():
        pass_.Reportf(call.Lparen, "iterator leak (missing %s.Done())", var.Name())

    return None, None

# TODO: go/analysis doesn't know that our run may panic a starlark.EvalError.
# How can we make it display the stack?  Should we wrap run ourselves?

# iterator_leak analyzer
iterator_leak = go.new(analysis.Analyzer)
iterator_leak.Name = "iterator_leak"
iterator_leak.Doc = "report calls to starlark.Iterator without a matching Done"
iterator_leak.Run = run
iterator_leak.Requires = [inspect.Analyzer]
