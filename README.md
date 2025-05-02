# lvlath

![Go](https://img.shields.io/badge/Go-1.23-blue)
![License](https://img.shields.io/badge/License-MIT-green)

> A high-performance, concurrent-safe Go library providing core graph data structures, matrix representations, and classic graph algorithms: BFS, DFS, Dijkstra, Prim, and Kruskal.

---

## 🚀 Features

* **Core Graph** (`graph/core`)

    * Thread-safe adjacency-list implementation
    * Directed & undirected, weighted & unweighted support
    * Clone, clone-empty, multiedges, self-loops
* **Matrix Representations** (`graph/matrix`)

    * Adjacency matrix with O(1) edge lookup
    * Incidence matrix for algebraic operations
    * Converters: `ToMatrix`, `ToEdgeList`
* **Algorithms** (`graph/algorithms`)

    * **BFS**: breadth-first search with hooks & cancellation
    * **DFS**: depth-first search with pre- and post-visit hooks
    * **Dijkstra**: shortest paths in weighted graphs
    * **Prim & Kruskal**: minimum spanning tree algorithms

## 📦 Installation

```bash
go get github.com/katalvlaran/lvlath
```

## 🗂️ Package Structure

```
lvlath/
├── graph/
│   ├── core/         # Graph, Vertex, Edge, concurrent-safe primitives
│   ├── matrix/       # AdjacencyMatrix, IncidenceMatrix, converters
│   └── algorithms/   # BFS, DFS, Dijkstra, Prim & Kruskal
├── LICENSE
└── README.md         # This file
```

## 🧑‍💻 Quick Start

### 1. Create a Graph

```go
import "github.com/katalvlaran/lvlath/graph/core"

g := core.NewGraph(false, true) // undirected, weighted
```

### 2. Add Vertices & Edges

```go
g.AddEdge("A", "B", 5)
g.AddEdge("B", "C", 2)
```

### 3. Run BFS

```go
import "github.com/katalvlaran/lvlath/graph/algorithms"

res, err := algorithms.BFS(g, "A", nil)
if err != nil {
    log.Fatal(err)
}
fmt.Println("Order:", extractIDs(res.Order)) // [A B C]
```

### 4. Find Shortest Paths

```go
dist, parent, err := algorithms.Dijkstra(g, "A")
// dist["C"] == 7, parent["C"] == "B"
```

### 5. Compute MST

```go
mst, total, err := algorithms.Prim(g, "A")
// total == 7, mst edges: A-B, B-C
```

## 🔍 Documentation

Detailed docs, examples, and API reference are available at [pkg.go.dev](https://pkg.go.dev/github.com/katalvlaran/lvlath).

## 🧪 Testing & Benchmarks

Run unit tests:

```bash
go test ./graph/...
```

Run benchmarks:

```bash
go test -bench=. ./graph/algorithms
```

## 📄 License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.

---

*Happy graphing!*
