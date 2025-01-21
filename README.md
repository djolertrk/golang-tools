# golang-tools

Set of low-level golang tools.

## callgraph

Build on MacOS:

```
$ brew install graphviz
$ go build -o callgraph callgraph.go
```

Run (lets use `callgraph.go` itself for the example):

```
$ callgraph callgraph.go > tmp.dot
$ dot -Tpng tmp.dot -o graph.png
```

