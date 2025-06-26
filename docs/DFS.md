# Depth‑First Search (DFS) Tutorial

# Depth‑First Search (DFS) Tutorial

This document is a comprehensive guide to Depth‑First Search (DFS) on a `core.Graph`, covering algorithmic details, complexity analysis, and practical Go code examples using the `lvlath/dfs` package. It also includes cycle detection and topological sort implementations, tailored for directed graphs (and undirected where noted).

---

## 1. Introduction

Depth‑First Search (DFS) explores as far as possible along each branch before backtracking. It is a cornerstone of graph algorithms, powering:

- **Cycle Detection** in directed or undirected graphs via back‑edge detection.
- **Topological Sorting** of Directed Acyclic Graphs (DAGs).
- **Connectivity & Component Analysis**, **Path Finding**, and as a base for **SCC** or **articulation point** algorithms.

The `lvlath/dfs` package offers:

- **DFS** with pre-order (`OnVisit`) and post-order (`OnExit`) hooks.
- **Cycle Detection** (`DetectCycles`) using three-color marking and canonical cycle normalization (Booth’s algorithm).
- **Topological Sort** (`TopologicalSort`) on directed graphs, with cycle detection.
- **Cancellation**, **Depth Limits**, and **Neighbor Filters** via functional options.

Supported Go version: **1.21+**.

---

## 2. Algorithmic Concepts

### 2.1 Vertex Coloring & States

We maintain three visitation states per vertex:

| State | Value | Meaning                           |
|-------|-------|-----------------------------------|
| White | 0     | Unvisited                         |
| Gray  | 1     | Discovered, in recursion stack    |
| Black | 2     | Fully explored (post‑order done)  |

- A **back‑edge** Gray→Gray indicates a cycle.
- Post‑order (when marking Black) generates finish ordering used for topological sort.

### 2.2 Complexity

| Operation           | Time Complexity                                                                                                                             | Memory Complexity                                                                                                 |
|---------------------|---------------------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------|
| **DFS (traversal)** | ![\Large O(V + E)](https://latex.codecogs.com/svg.image?\large&space;O(V&plus;E))                                                           | ![\Large O(V)](https://latex.codecogs.com/svg.image?\large&space;O(V))                                            |
| **DetectCycles**    | ![\Large O\bigl(V + E + C \times L^2\bigr)](https://latex.codecogs.com/svg.image?\large&space;O\bigl(V&plus;E&plus;C\times&space;L^2\bigr)) | ![\Large O\bigl(V + E + C \times L^2\bigr)](https://latex.codecogs.com/svg.image?\large&space;O(V&plus;L_{\max})) |
| **TopologicalSort** | ![\Large O(V + E)](https://latex.codecogs.com/svg.image?\large&space;O(V&plus;E))                                                           | ![\Large O(V)](https://latex.codecogs.com/svg.image?\large&space;O(V))                                            |


| Operation           | Time Complexity                        | Memory Complexity                     |
|---------------------|----------------------------------------|---------------------------------------|
| **DFS (traversal)** | $$O(V + E)$$                           | $$O(V)$$                              |
| **DetectCycles**    |  $$O\bigl(V + E + C \times L^2\bigr)$$ | $$O\bigl(V + E + C \times L^2\bigr)$$ |
| **TopologicalSort** |  $$O(V + E)$$                          | $$O(V)$$                              |

- V = number of vertices, E = number of edges.
- C = number of simple cycles, L = average cycle length.
- Booth’s algorithm for minimal rotation runs in O(L), but canonical normalization considers forward and reverse, overall O(L).

---

## 3. DFS Pseudocode

```text
procedure DFS(u, depth):
  state[u] = Gray                  # mark in progress
  if OnVisit defined: OnVisit(u)
  for each v in Neighbors(u):
    if state[v] == White and FilterNeighbor(v):
      parent[v] = u
      DFS(v, depth+1)
    else if state[v] == Gray:
      record back-edge (u → v) # cycle detected
  if OnExit defined: OnExit(u)
  state[u] = Black                 # mark finished
  append u to post-order list
```

---

## 4. Go Implementation: DFS


```go
package main

import (
	"fmt"
	"strings"

	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/dfs"
)

// ExampleDFS demonstrates a depth-first traversal (post-order) on a diamond-shaped graph.
// Graph structure:
//
//	  A
//	 / \
//	B   C
//	 \ /
//	  D
//	 / \
//	E   F
//
// Starting at "A", expected post-order: E F D B C A
func main() {
	g := core.NewGraph(core.WithDirected(true))
	g.AddEdge("A", "B", 0)
	g.AddEdge("A", "C", 0)
	g.AddEdge("B", "D", 0)
	g.AddEdge("C", "D", 0)
	g.AddEdge("D", "E", 0)
	g.AddEdge("D", "F", 0)

	res, err := dfs.DFS(g, "A")
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(strings.Join(res.Order, " "))
	// Output: E F D B C A
}
```

[![Playground - DFS_SimpleDiamond](https://img.shields.io/badge/Go_Playground-DFS_SimpleDiamond-blue?logo=go)](https://go.dev/play/p/Q8rB9YCA7lN)

**`DFSResult` fields:**

- `Order []string` - vertices in post-order finish.
- `Depth map[string]int` - discovery depth from `startID`.
- `Parent map[string]string` - DFS tree edges.
- `Visited map[string]bool` - reachable vertices.
- `SkippedNeighbors int` - count filtered out.

**Error Cases:**

- `ErrGraphNil` if graph pointer is nil.
- `ErrStartVertexNotFound` if `startID` absent.
- `context.Canceled` on cancellation.
- Hook errors propagate and clear `Order`.

---

## 5. Cycle Detection (`DetectCycles`)

Detects all **simple cycles** in a (directed or undirected) graph by:

1. Running DFS with three-color marking.
2. On each Gray→Gray back-edge, extracting the cycle from the current path stack.
3. Canonicalizing via minimal rotation (Booth’s algorithm) on both forward and reversed cycle to avoid duplicates.
4. Sorting final cycle list by signature for determinism.


```go
package main

import (
	"fmt"
	"strings"

	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/dfs"
)

// ExampleDetectCycles shows detecting cycles in a directed graph.
func main() {
	g := core.NewGraph(core.WithDirected(true))
	// Create cycle
	g.AddEdge("A", "B", 0)
	g.AddEdge("B", "C", 0)
	g.AddEdge("B", "D", 0)
	g.AddEdge("C", "E", 0)
	g.AddEdge("E", "F", 0)
	g.AddEdge("F", "G", 0)
	g.AddEdge("G", "A", 0)
	g.AddEdge("D", "H", 0)
	g.AddEdge("H", "I", 0)
	g.AddEdge("I", "J", 0)
	g.AddEdge("J", "K", 0)
	g.AddEdge("K", "B", 0)

	has, cycles, err := dfs.DetectCycles(g)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(has)
	for _, c := range cycles {
		fmt.Println(strings.Join(c, " -> "))
	}
	// Output:
	// true
	// A -> B -> C -> E -> F -> G -> A
	// B -> D -> H -> I -> J -> K -> B
}
```

[![Playground - DFS_DetectCycles](https://img.shields.io/badge/Go_Playground-DFS_SimpleDiamond-blue?logo=go)](https://go.dev/play/p/J6XSI3sBwVb)

**Directed vs. Undirected:**

- In **undirected** graphs, skip trivial backtracks to parent and 2-cycles.
- Requires `g.Looped()` to allow self‑loops.

---

## 6. Topological Sort (`TopologicalSort`)

Performs DFS-based topological sorting, reversing post‑order of a directed acyclic graph:

```text
function TopologicalSort(G):
  for each u in G: state[u] = White
  order = empty list

  for each u in G:
    if state[u] == White:
      visit(u)

  reverse(order)
  return order

function visit(u):
  if state[u] == Gray: error: cycle detected
  if state[u] == Black: return

  state[u] = Gray
  for each v in Neighbors(u): visit(v)
  state[u] = Black
  append u to order
```

* On detecting a Gray neighbor during recursion, **ErrCycleDetected** is returned.
* Requires **directed** graph: undirected behavior is undefined and rejected.

**Complexity**: O(V + E)

```go
package main

import (
   "fmt"
   "strings"

   "github.com/katalvlaran/lvlath/core"
   "github.com/katalvlaran/lvlath/dfs"
)

// ExampleTopologicalSort demonstrates computing a valid topological order
// on a DAG with shared child D. Graph:
//
//	   A
//	  / \
//	 B   C
//	  \ / \
//	   D   G
//	  / \   \
//	 E   F   H
//
// Expected topological order: A C G H B D F E
package main

import (
"fmt"
"strings"

"github.com/katalvlaran/lvlath/core"
"github.com/katalvlaran/lvlath/dfs"
)

func main() {
	g := core.NewGraph(core.WithDirected(true))
	g.AddEdge("A", "B", 0)
	g.AddEdge("A", "C", 0)
	g.AddEdge("B", "D", 0)
	g.AddEdge("C", "D", 0)
	g.AddEdge("C", "G", 0)
	g.AddEdge("D", "E", 0)
	g.AddEdge("D", "F", 0)
	g.AddEdge("G", "H", 0)

	order, err := dfs.TopologicalSort(g)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(strings.Join(order, " "))
	// Output: A C G H B D F E

}
```

[![Playground - DFS_TopologicalSort](https://img.shields.io/badge/Go_Playground-DFS_TopologicalSort-blue?logo=go)](https://go.dev/play/p/bqy8uEu1RXS)

**Key Points:**

- Returns `ErrCycleDetected` upon finding a back-edge Gray→Gray.
- Only supports **directed** graphs (rejects undirected).
- Cancellation via `WithCancelContext`.

---

## 7. Practical Tips & Pitfalls

- **Stack depth**: For very deep or skewed graphs, consider iterative DFS or increase Go stack limits.
- **Hooks overhead**: Keep `OnVisit`/`OnExit` callbacks lightweight to avoid performance regressions.
- **FilterNeighbor**: Prune large subgraphs early (e.g., skip expensive branches).
- **Context cancellation**: Use `WithContext` to abort long traversals cleanly.
- **FullTraversal**: Enable to cover disconnected components in one call.

---

## 8. Further Reading

- [Shortest Paths (Dijkstra)](DIJKSTRA.md)


*End of DFS Tutorial*
