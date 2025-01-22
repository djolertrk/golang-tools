// A simple tool that produces a call graph in GraphViz (.dot) format,
// handling external imports via go/packages.
//
// author: djolertrk

package main

import (
    "fmt"
    "log"
    "os"

    "golang.org/x/tools/go/packages"
    "go/ast"
    "go/types"
)

// callDetail holds information about one call site.
type callDetail struct {
    callee   string
    filename string
    line     int
    column   int
}

// callGraphMap: caller -> list of calls
type callGraphMap map[string][]callDetail

func main() {
    if len(os.Args) < 2 {
        fmt.Fprintf(os.Stderr, "usage: %s <package-pattern>\n", os.Args[0])
        os.Exit(1)
    }

    // 1) Load package(s) using go/packages for module awareness.
    //    Example usage:
    //      callgraph . 
    //    or:
    //      callgraph github.com/gnolang/gno/gnovm/pkg/gnolang
    //    or:
    //      callgraph ./...  (to load all sub-packages)
    pkgPattern := os.Args[1]

    // We request enough info to see syntax (AST), types, and imports/deps.
    cfg := &packages.Config{
        Mode: packages.NeedName | packages.NeedImports | packages.NeedDeps |
            packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo,
    }
    pkgs, err := packages.Load(cfg, pkgPattern)
    if err != nil {
        log.Fatal(err)
    }
    // Check for any loading errors
    if packages.PrintErrors(pkgs) > 0 {
        log.Fatal("failed to load packages due to the above errors")
    }

    // 2) We'll build a global call graph across all loaded packages
    callGraph := make(callGraphMap)

    // 3) For each loaded package, gather call edges by walking its AST
    for _, pkg := range pkgs {
        fset := pkg.Fset        // token.FileSet for this package
        info := pkg.TypesInfo   // type info for this package

        // Possibly give the user a note about which package is being processed
        // fmt.Println("Analyzing package:", pkg.PkgPath)

        // Iterate all files in this package
        for _, fileAST := range pkg.Syntax {
            // We'll track the "currentFunc" as we enter each FuncDecl
            var currentFunc string

            ast.Inspect(fileAST, func(n ast.Node) bool {
                switch node := n.(type) {
                case *ast.FuncDecl:
                    // The function's "full name" can be pkgPath + "." + funcName 
                    // to distinguish identical func names in different packages.
                    currentFunc = pkg.PkgPath + "." + node.Name.Name

                    // Ensure the caller is in the map
                    if _, ok := callGraph[currentFunc]; !ok {
                        callGraph[currentFunc] = []callDetail{}
                    }

                case *ast.CallExpr:
                    // We have a function call; fetch the source position.
                    pos := fset.Position(node.Pos())

                    // Identify the callee via the TypesInfo in this package
                    switch fun := node.Fun.(type) {
                    case *ast.Ident:
                        // Simple call: foo()
                        if fnObj := info.Uses[fun]; fnObj != nil {
                            calleeName := fnObj.Name()
                            if currentFunc != "" && calleeName != "" {
                                callGraph[currentFunc] = append(
                                    callGraph[currentFunc],
                                    callDetail{
                                        callee:   calleeString(fnObj),
                                        filename: pos.Filename,
                                        line:     pos.Line,
                                        column:   pos.Column,
                                    },
                                )
                            }
                        }

                    case *ast.SelectorExpr:
                        // A call like pkg.Func() or receiver.Method()
                        if fnObj := info.Uses[fun.Sel]; fnObj != nil {
                            if currentFunc != "" && fnObj.Name() != "" {
                                callGraph[currentFunc] = append(
                                    callGraph[currentFunc],
                                    callDetail{
                                        callee:   calleeString(fnObj),
                                        filename: pos.Filename,
                                        line:     pos.Line,
                                        column:   pos.Column,
                                    },
                                )
                            }
                        }
                    }
                }
                return true
            })
        }
    }

    // 4) Output the final .dot graph
    // We'll label each edge with "filename:line:column".
    fmt.Println("digraph G {")
    for caller, calls := range callGraph {
        for _, c := range calls {
            label := fmt.Sprintf("%s:%d:%d", c.filename, c.line, c.column)
            fmt.Printf(`    %q -> %q [label="%s"];`+"\n",
                caller,
                c.callee,
                label)
        }
    }
    fmt.Println("}")
}

// calleeString returns a more descriptive name for the callee.
// By default, we use pkgpath + "." + name if possible.
func calleeString(obj types.Object) string {
    // If the callee is from the standard library or your own package,
    // obj.Pkg() might be nil (for builtins) or might have PkgPath.
    // We'll handle nil carefully.
    if obj.Pkg() == nil {
        return obj.Name()
    }
    return obj.Pkg().Path() + "." + obj.Name()
}
