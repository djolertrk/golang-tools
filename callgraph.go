// A simple AST parser that produces callgraph in form of graphviz.
// author: djolertrk

package main

import (
    "fmt"
    "go/ast"
    "go/importer"
    "go/parser"
    "go/token"
    "go/types"
    "os"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Fprintf(os.Stderr, "usage: %s <go-file>\n", os.Args[0])
        os.Exit(1)
    }
    goFile := os.Args[1]

    // Parse the file into an AST
    fset := token.NewFileSet()
    file, err := parser.ParseFile(fset, goFile, nil, 0)
    if err != nil {
        panic(err)
    }

    // Type-check
    conf := types.Config{Importer: importer.Default()}
    info := &types.Info{
        Types:      make(map[ast.Expr]types.TypeAndValue),
        Defs:       make(map[*ast.Ident]types.Object),
        Uses:       make(map[*ast.Ident]types.Object),
        Implicits:  make(map[ast.Node]types.Object),
        Scopes:     make(map[ast.Node]*types.Scope),
        Selections: make(map[*ast.SelectorExpr]*types.Selection),
    }
    _, err = conf.Check("cmd/test", fset, []*ast.File{file}, info)
    if err != nil {
        panic(err)
    }

    // We'll store call info in a struct:
    type callDetail struct {
        callee   string
        filename string
        line     int
        column   int
    }

    // Map: callerName -> slice of callDetail
    callGraph := make(map[string][]callDetail)

    // Helper: turn the function's types.Object into a name
    getFuncName := func(obj types.Object) string {
        return obj.Name()
    }

    // Track the current function name while traversing
    var currentFunc string

    // Walk the AST to find function calls
    ast.Inspect(file, func(n ast.Node) bool {
        switch node := n.(type) {
        case *ast.FuncDecl:
            // Entering a function; record its name
            currentFunc = node.Name.Name
            if _, ok := callGraph[currentFunc]; !ok {
                callGraph[currentFunc] = []callDetail{}
            }

        case *ast.CallExpr:
            // We found a call expression
            // We'll get the position for labeling
            pos := fset.Position(node.Pos())

            // Identify the callee
            switch fun := node.Fun.(type) {
            case *ast.Ident:
                // A simple call like foo()
                if fnObj := info.Uses[fun]; fnObj != nil {
                    callee := getFuncName(fnObj)
                    if currentFunc != "" && callee != "" {
                        callGraph[currentFunc] = append(callGraph[currentFunc], callDetail{
                            callee:   callee,
                            filename: pos.Filename,
                            line:     pos.Line,
                            column:   pos.Column,
                        })
                    }
                }
            case *ast.SelectorExpr:
                // A call like pkg.Func() or recv.Method()
                if fnObj := info.Uses[fun.Sel]; fnObj != nil {
                    callee := getFuncName(fnObj)
                    if currentFunc != "" && callee != "" {
                        callGraph[currentFunc] = append(callGraph[currentFunc], callDetail{
                            callee:   callee,
                            filename: pos.Filename,
                            line:     pos.Line,
                            column:   pos.Column,
                        })
                    }
                }
            }
        }
        return true
    })

    // Output .dot with location info
    // We'll label each edge with "filename:line:column".
    fmt.Println("digraph G {")
    for caller, calls := range callGraph {
        for _, detail := range calls {
            label := fmt.Sprintf("%s:%d:%d", detail.filename, detail.line, detail.column)
            fmt.Printf(`    %q -> %q [label="%s"];`+"\n", caller, detail.callee, label)
        }
    }
    fmt.Println("}")
}
