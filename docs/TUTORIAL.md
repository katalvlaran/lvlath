# lvlath Tutorial

> **hard math on an easy level**

Welcome to **lvlath**, the Go library that makes complex graph and algorithmic operations intuitive and accessible.

---

## 📑 Table of Contents

1. [Introduction](#1-introduction)
2. [Quick Start](#2-quick-start)
3. [Graph Concepts](#3-graph-concepts)
4. [Traversals: BFS & DFS](BFS_%26_DFS.md)
5. [Shortest Paths: Dijkstra](DIJKSTRA.md)
6. [Minimum Spanning Trees: Prim & Kruskal](PRIM_%26_KRUSKAL.md)
7. [Max-Flow: Ford–Fulkerson / Edmonds-Karp / Dinic](MAX_FLOW.md)
8. [Dynamic Time Warping (DTW)](DTW.md)
9. [GridGraph: Grid-based Graphs](GRID_GRAPH.md)
10. [Matrices: Adjacency & Incidence](MATRICES.md)
11. [Traveling Salesman (TSP)](TRAVELING_SALESMAN.md)
12. [Best Practices & Tips](FAQ_&_TIPS.md)
13. [References & Further Reading] - preparing..
14. [FAQ / Troubleshooting](FAQ_&_TIPS.md)

---

## 1. Introduction

**What is lvlath?**

A modern, thread-safe Go library that unifies:

* Core graph primitives (vertices, edges) with safe concurrency
* Matrix representations (adjacency, incidence)
* Classic algorithms (BFS, DFS, Dijkstra, MSTs, Flow)
* Advanced solvers (DTW, TSP)

**Why lvlath?**

* **Clarity**: concise APIs and hookable callbacks
* **Performance**: optimized for Go's concurrency model
* **Learning-Focused**: live examples, visualizations, and educational commentary

**Example ASCII Graph**

```ascii
  +---+     +---+
  | A |-----| B |
  +---+     +---+
     \          \
      \          v
      +---+     +---+
      | C |-----| D |
      +---+     +---+
```

**Audience**

Whether you're a Go developer, data scientist, or algorithm enthusiast, lvlath helps you:

* **Understand** algorithm behavior through clear code and visuals
* **Integrate** advanced routines into your projects
* **Experiment** with educational scenarios

---

## 2. Quick Start

### Installation

```bash
go get github.com/katalvlaran/lvlath
```

### Creating Your First Graph
```go
package main

import (
    "fmt"
    "github.com/katalvlaran/lvlath/core"
)

// vertexIDs extracts IDs from vertices
func vertexIDs(vs []*core.Vertex) []string {
    ids := make([]string, len(vs))
    for i, v := range vs {
        ids[i] = v.ID
    }
    return ids
}

func main() {
    // 1) Initialize an undirected, unweighted graph
    g := core.NewGraph(false, false)

    // 2) Add vertices and edges
    g.AddVertex(&core.Vertex{ID: "A"})
    g.AddEdge("A", "B", 0) // auto-adds "B"
    g.AddEdge("A", "C", 0) // auto-adds "C"

    // 3) Inspect vertices and edges
    fmt.Println("Vertices:", vertexIDs(g.Vertices()))
    fmt.Println("Edge A→B exists?", g.HasEdge("A", "B"))

    // 4) Remove a vertex
    g.RemoveVertex("B")
    fmt.Println("After removing B:", vertexIDs(g.Vertices()))
}
```
[![Go Playground](https://img.shields.io/badge/Go_Playground-Build_new_graph-blue?logo=go)](https://go.dev/play/p/wDe6448IHEv)

**Expected Output:**

```
Vertices: [A B C]
Edge A→B exists? true
After removing B: [A C]
```


## 3. Graph Concepts

Graphs model relationships via **vertices** (nodes) and **edges** (connections). lvlath supports:

* **Directed vs Undirected**
* **Weighted vs Unweighted**

### Adjacency List

Internally, lvlath uses a **map of maps**:

```
map[vertexID]map[neighborID][]*Edge
```

```ascii
List representation:
A → B, C
B → C
C → (none)
```

### Adjacency Matrix

For dense graphs or constant-time lookups, convert to a matrix:

|   | A | B | C |
| - | - | - | - |
| A | 0 | 5 | 0 |
| B | 5 | 0 | 7 |
| C | 0 | 7 | 0 |

```go
am := matrix.NewAdjacencyMatrix(g)
val := am.Data[am.Index["A"]][am.Index["B"]]
```

### Incidence Matrix

Useful in algebraic analyses:

* Rows: vertices
* Columns: edges
* Entries: -1/+1 (directed) or 1 (undirected)

```go
im := matrix.NewIncidenceMatrix(g)
row, _ := im.VertexIncidence("B")
```
---

*Please, visit provided below {algorithm}.md files to see the full intro, learn, and practice with it.*

4. [Traversals: BFS & DFS](BFS_%26_DFS.md)
5. [Shortest Paths: Dijkstra](DIJKSTRA.md)
6. [Minimum Spanning Trees: Prim & Kruskal](PRIM_%26_KRUSKAL.md)
7. [Max-Flow: Ford–Fulkerson / Edmonds-Karp / Dinic](MAX_FLOW.md)
8. [Dynamic Time Warping (DTW)](DTW.md)
9. [GridGraph: Grid-based Graphs](GRID_GRAPH.md)
10. [Matrices: Adjacency & Incidence](MATRICES.md)
11. [Traveling Salesman (TSP)](TRAVELING_SALESMAN.md)

12. [FAQ / Troubleshooting](FAQ_&_TIPS.md)
---

