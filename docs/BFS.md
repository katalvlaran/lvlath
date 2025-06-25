<!--
  This document explains the BFS algorithm at a production-grade level,
  covering theory, implementation details, Go API usage, and best practices.
-->

# BFS (Breadth-First Search)

## 1 What & Why
- **What:**  
  Breadth-First Search (BFS) explores an unweighted graph level by level from a given start vertex, producing:
    - **Order** of visitation (by non-decreasing distance)
    - **Depth** map: distance (number of edges) from start to each reached vertex
    - **Parent** pointers for path reconstruction

- **Why:**
    - **Unweighted shortest paths:** finds minimum-edge paths in O(V+E) time
    - **Layered traversal:** useful for connectivity checks, level-order processing
    - **Foundation:** forms building block for flow algorithms, bipartiteness tests, social-network layers, etc.

## 2 Math Formulation
Shortest‐path distance in an unweighted graph is the minimum number of edges between two vertices:

![\displaystyle \mathrm{dist}(s,v)=\min_{\text{paths }s\to v}\sum_{e\in\text{path}}1](https://latex.codecogs.com/svg.image?\displaystyle%20\mathrm{dist}(s,v)%3D\min_{\text{paths}%20s\to%20v}\sum_{e\in\text{path}}1)

Complexity:
- **Time:** ![\mathcal{O}(V+E)](https://latex.codecogs.com/svg.image?\mathcal{O}(V+E))
- **Memory:** ![\mathcal{O}(V)](https://latex.codecogs.com/svg.image?\mathcal{O}(V))

## 3. High-Level Pseudocode

```text
# Initialize
Q ← empty queue                     // O(1)
for each u in V do                  // O(|V|)
  visited[u] ← false                // O(1)

# Seed
visited[start] ← true               // O(1)
depth[start] ← 0                    // O(1)
parent[start] ← nil                 // O(1)
enqueue(Q, start)                   // O(1)

# Main loop
while Q not empty do                // up to |V| iterations
  u ← dequeue(Q)                    // O(1)
  for each edge (u→v) in E(u) do    // total O(|E|) over all iterations
    if not visited[v] then          // O(1)
      visited[v] ← true             // O(1)
      depth[v] ← depth[u] + 1       // O(1)
      parent[v] ← u                 // O(1)
      enqueue(Q, v)                 // O(1)
    end if
  end for
end while

return (Order = dequeue sequence, Depth, Parent)
```
<!-- Comments: Each step is constant time; outer loop runs |V| times, inner total over all vertices examines each edge once. -->

## 4. Directed, Undirected & Mixed Modes

- **Undirected Graphs**: Each edge is traversed both ways. BFS sees each undirected neighbor once.
- **Directed Graphs**: Only follow edges in their forward direction (`From → To`).
- **Mixed-Edges Mode** (`core.WithMixedEdges`): Graph-level default may be undirected, but individual edges can override direction:
  - If `Edge.Directed == true`, treat edge as one-way.
  - Else treat as undirected (bidirectional).

Implementation detail: When iterating neighbors, determine the "true neighbor" based on `edge.From` and `edge.To`, respecting `Directed` flag.



## 5. Go API & Functional Options

```go
func BFS(
  g *core.Graph,
  startID string,
  opts ...Option,
) (*BFSResult, error)
```

**Error Cases**:
- `ErrGraphNil` if `g == nil`.
- `ErrStartVertexNotFound` if `startID` is absent.
- `ErrWeightedGraph` if `g.Weighted()` is `true` (BFS is unweighted).
- `ErrOptionViolation` for invalid options (e.g., negative `MaxDepth`).
- `ErrNeighbors` if neighbor retrieval fails.
- Wrapped errors from user-supplied `OnVisit`.

**Key Options**:
- `WithContext(ctx)` — Cancel or timeout BFS via `context.Context`.
- `WithMaxDepth(d)` — Stop exploring beyond depth \(d\).
  - \(d>0\): limit to depth \(d\).
  - \(d=0\): explicit "no depth limit."
  - \(d<0\): invalid (violates `ErrOptionViolation`).
- `WithFilterNeighbor(fn(curr, nbr string) bool)` — Skip edges for which `fn` returns `false` (prune traversal dynamically).
- `WithOnEnqueue(fn(id, depth))`, `WithOnDequeue(fn(id, depth))`, `WithOnVisit(fn(id, depth) error)` — Hooks at each stage.

## 6. Determinism & Reproducibility

Because `core.Neighbors(u)` returns edges sorted by `Edge.ID`, and BFS enqueues in that order, the visit sequence is _fully reproducible_ across runs on the same graph, barring context cancellation.

## 7. Code Example

```go
  // Build a simple diamond graph
  //    A
  //   / \
  //  B   C
  //   \ /
  //    D
  
  g := core.NewGraph()
  g.AddEdge("A", "B", 0)
  g.AddEdge("A", "C", 0)
  g.AddEdge("B", "D", 0)
  g.AddEdge("C", "D", 0)
  g.AddEdge("D", "E", 0)
  g.AddEdge("D", "F", 0)
  
  res, err := bfs.BFS(
    g, "A",
    bfs.WithOnVisit(func(id string, d int) error {
      fmt.Printf("Visited %s@%d\n", id, d)
      return nil
    }),
  )
  if err != nil {
    log.Fatal(err)
  }
  fmt.Println("Order:", res.Order)
  // Output:
  // Visited A@0
  // Visited B@1
  // Visited C@1
  // Visited D@2
  // Visited E@3
  // Visited F@3
  // Order: [A B C D E F]
```

[![Playground - BFS_SimpleDiamond](https://img.shields.io/badge/Go_Playground-BFS_SimpleDiamond-blue?logo=go)](https://go.dev/play/p/t2lxkt-unci)

## 8. Pitfalls & Best Practices

| Pitfall                                      | Recommendation                                                                          |
|-----------------------------------------------|-----------------------------------------------------------------------------------------|
| Calling `VerticesMap()` inside a hook         | Use the local `visited` map for O(1) checks; avoid O(V) scans in hooks (preloaded by BFS). |
| Omitting nil-graph or missing start checks    | Always handle `ErrGraphNil` and `ErrStartVertexNotFound` before processing.              |
| Unbounded exploration on deep or infinite graphs | Use `WithMaxDepth(n)` or `WithFilterNeighbor` to constrain memory/time.                  |
| Ignoring context cancellation                 | Combine `WithContext` and respect `Canceled`/`DeadlineExceeded` to abort promptly.      |

---

<!-- End of BFS.md: this serves as a single source of truth for theory, implementation, and usage. -->