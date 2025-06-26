# Core Graph (What, Why, When)

## What & Why

**What**:

**`core`** is the foundational, in-memory adjacency-list graph implementation powering all lvlath graph algorithms. It provides:

* **Flexible configuration** via composable functional options:

  * `NewGraph(...)` for undirected or directed default
  * `core.WithWeighted()` to allow non-zero edge weights
  * `core.WithMultiEdges()` for parallel (multi-)edges
  * `core.WithLoops()` to permit self-loops
  * **New** `core.NewMixedGraph(...)` to enable per-edge directedness overrides (mixed-mode)

* **Thread-safe operations** with minimal lock contention:

  * Separate `sync.RWMutex` for vertices vs. edges/adjacency
  * Constant-time edge existence checks via nested maps

* **Deterministic iteration**:

  * `Vertices()` and `Edges()` return sorted results (O(V·log V), O(E·log E))
  * `Neighbors(v)` returns sorted edge list (O(d·log d))

**Why**:

* Acts as the universal foundation for all lvlath algorithms (BFS, Dijkstra, MST, Flow, etc.).
* Minimal overhead: optimized for Go’s sync.RWMutex.
* Deterministic APIs: sorted outputs aid reproducibility and testing.

---

## Math Formulation

  **Let**
  * `V` be a finite set of vertices
  * $$E \subseteq V \times V \times \{\mathrm{edgeIDs}\}$$ be a finite set of edges, each carrying an integer weight $$w_{uv} \in \mathbb{Z}$$ .

  **Directed vs. Undirected**
  * Directed:
    $$E \subseteq \{(u,v) \mid u \neq v\};\quad \mathrm{adj}[u][v] \subseteq E,\quad \mathrm{adj}[v][u] = \emptyset$$ .
  * Undirected (mirrored storage):
    $$E \subseteq \{(u,v) \mid u \neq v\};\quad \mathrm{adj}[u][v] \subseteq E,\quad \mathrm{adj}[v][u] \subseteq E$$ ,
  storing a single edgeID in both lists.
  * Loop `(v,v)` stored once in `adj[v][v]`.

  **Mixed-Mode**

  Default graph mode may be undirected, but edges added with *WithEdgeDirected(true)* become directed; those with false remain undirected.

---

## Constructor Patterns

```go
// Undirected, unweighted, no loops, no multi-edges:
g := core.NewGraph()

// Directed, weighted, loops allowed:
g2 := core.NewGraph(
  core.WithDirected(true),
  core.WithWeighted(),
  core.WithLoops(),
)

// Mixed-mode (default undirected) + weighted + multi-edges:
gMixed := core.NewMixedGraph(
  core.WithWeighted(),
  core.WithMultiEdges(),
  core.WithLoops(),
)
```

* **`NewGraph(opts...)`** sets default behavior for all edges.
* **`NewMixedGraph(opts...)`** is equivalent to `NewGraph(opts..., WithMixedEdges())` and signals that per-edge directedness overrides are permitted.

---

## Core API & Complexity

| Method                      | Description                                         | Complexity  |
| --------------------------- | --------------------------------------------------- | ----------- |
| `AddVertex(id string)`      | Add vertex (idempotent)                             | O(1)        |
| `HasVertex(id string)`      | Check vertex existence                              | O(1)        |
| `RemoveVertex(id string)`   | Remove vertex + all incident edges                  | O(deg(v)+M) |
| `AddEdge(u,v,w, opts…)`     | Add edge; weight & per-edge overrides               | O(1)        |
| `HasEdge(u,v)`              | Test existence                                      | O(1)        |
| `RemoveEdge(edgeID string)` | Remove by edge ID                                   | O(1)        |
| `Neighbors(v)`              | Sorted list of edges incident on v                  | O(d·log d)  |
| `NeighborIDs(v)`            | Sorted list of unique neighbor IDs                  | O(d·log d)  |
| `Vertices()`                | Sorted vertex IDs                                   | O(V·log V)  |
| `Edges()`                   | Sorted edges by ID                                  | O(E·log E)  |
| `CloneEmpty()`              | Copy mode & vertices only                           | O(V)        |
| `Clone()`                   | Deep copy (vertices+edges)                          | O(V+E)      |
| `FilterEdges(pred)`         | Remove edges failing `pred`, cleanup isolated verts | O(E+V)      |

---

## Pseudocode

```text
function AddEdge(G, u, v, w):
  // ensure endpoints exist
  if not G.hasVertex(u): G.addVertex(u)
  if not G.hasVertex(v): G.addVertex(v)
  
  directed := resolveDirected(G, opts)   // default vs. per-edge
  edgeID   := newEdgeID()

  // insert forward arc
  G.adj[u][v].append(edgeID)
  G.edgeData[edgeID] = {From:u, To:v, Weight:w, Directed:directed}

  // mirror if undirected and not self-loop
  if not directed and u != v:
    G.adj[v][u].append(edgeID)
```

---

## ASCII Diagram

**Undirected edge** (mirror stored once per direction):
```
      A──e1──B
     /        \
 adj[A][B]   adj[B][A]
```

**Directed edge** (single direction):
```
   A─e2─>B   (only adj[A][B])
```

**Mixed-mode graph** (coexisting types):
```
   A──e3──C   undirected
   |e4        undirected
   D──e5─>E   directed
```

**Simple visualization**
```
            Ⓐ
            │   (weight)
   ┌───5────┴────9───┐
   │                 │
   ▼                 ▼
   Ⓑ───4──→Ⓒ───7──→Ⓓ   
```

---

## Go Example

```go
package main

import (
  "fmt"
  "github.com/katalvlaran/lvlath/core"
)

func main() {
  g := core.NewGraph(
    core.WithWeighted(),
    core.WithMultiEdges(),
    core.WithMixedEdges(),
  )
  // Add undirected road
  id, _ := g.AddEdge("Kyiv", "Lviv", 537)
  fmt.Println("Added edge", id)
  _, _ = g.AddEdge("Kyiv", "Odesa", 508)
  _, _ = g.AddEdge("Kyiv", "Lutsk", 423)
  _, _ = g.AddEdge("Kyiv", "Vinnytsia", 230)
  _, _ = g.AddEdge("Lviv", "Vinnytsia", 377)
  _, _ = g.AddEdge("Odesa", "Yuzhnoukrainsk", 148)
  nbs, _ := g.Neighbors("Kyiv")
  fmt.Println("Neighbors of Kyiv:", nbs)
}

```
[![Playground - Core](https://img.shields.io/badge/Go_Playground-Core-blue?logo=go)](https://go.dev/play/p/r5MFWaecYsV)

---


## Pitfalls & Best Practices

* **ErrBadWeight**: check `g.Weighted()` before calling `AddEdge(..., weight)` on unweighted graphs.
* **Self-loops**: in undirected mode, loops appear **only once** in `Neighbors()` and `Edges()` (no mirror).
* **Parallel edges**: if `WithMultiEdges()` is disabled, adding a second edge between the same endpoints errors `ErrMultiEdgeNotAllowed`.
* **Mixed-mode**: only available via `NewMixedGraph()`; per-edge overrides (`WithEdgeDirected`) on a plain `NewGraph()` return `ErrMixedEdgesNotAllowed`.
* **Clone**: always use `c := g.Clone()` to get a safe deep copy (including `nextEdgeID`), so that further `AddEdge` IDs remain unique.
* **High-throughput**: batch multiple mutations under a single lock or consider sharding for extremely large data.

---