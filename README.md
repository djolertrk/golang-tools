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
The example:
![graph](https://github.com/user-attachments/assets/56b632bc-4eef-4316-8c97-1095d1dd7324)


