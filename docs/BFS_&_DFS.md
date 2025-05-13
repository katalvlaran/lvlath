## 4. Traversals (BFS & DFS)

Graph traversals explore all vertices in a controlled order. lvlath provides two fundamental methods:

* **Breadth‑First Search (BFS)** discovers nodes in increasing distance from the start.
* **Depth‑First Search (DFS)** dives deep along one branch before backtracking.

---

### 4.1 When and Why

* **BFS** is ideal for shortest paths in unweighted graphs, level‑order layering, and connectivity checks.
* **DFS** is used for cycle detection, topological sorting, and exploring all possible paths.

Both support **hooks** to inject custom logic during traversal:

```go
// BFSOptions allows you to observe enqueue, dequeue, and visit events:
type BFSOptions struct {
    OnEnqueue func(v *core.Vertex, depth int)
    OnDequeue func(v *core.Vertex, depth int)
    OnVisit   func(v *core.Vertex, depth int) error
}

// DFSOptions lets you run code on entry and exit of each node:
type DFSOptions struct {
    OnVisit func(v *core.Vertex, depth int) error
    OnExit  func(v *core.Vertex, depth int)
}
```

Use these hooks to, for example, record a tree, abort early, or visualize progress.

---

### 4.2 ASCII Walkthrough

Consider the graph below (7 nodes):

```
      [A]
      / \
    [B] [C]
     |     \
    [D]     [E]---[G]
     |
    [F]
```

* BFS starting at **A** visits: A → B,C → D,E → F,G
* DFS (preorder) may visit: A → B → D → F → C → E → G

---

### 4.3 Pseudocode

#### BFS

```text
function BFS(G, start):
  for each u in G:
    visited[u] := false
  queue := empty queue
  visited[start] := true
  enqueue(queue, start, depth=0)
  while not isEmpty(queue):
    (u, d) := dequeue(queue)
    for each v in Neighbors(u):
      if not visited[v]:
        visited[v] := true
        enqueue(queue, v, depth=d+1)
```

#### DFS (recursive)

```text
function DFS(u, depth):
  visited[u] := true
  for each v in Neighbors(u):
    if not visited[v]:
      DFS(v, depth+1)
```

---

### 4.4 Go Example

Self‑contained example: BFS with hooks to print depth.
```go
package main

import (
  "fmt"
  "github.com/katalvlaran/lvlath/core"
  "github.com/katalvlaran/lvlath/algorithms"
)

func main() {
  // Build a simple tree
  g := core.NewGraph(false, false)
  for _, e := range [][2]string{{"A","B"},{"A","C"},{"B","D"},{"D","F"},{"C","E"},{"E","G"}} {
    g.AddEdge(e[0], e[1], 0)
  }

  fmt.Println("BFS Order with depth:")
  algorithms.BFS(g, "A", &algorithms.BFSOptions{
    OnVisit: func(v *core.Vertex, d int) error {
      fmt.Printf("%s(depth=%d) ", v.ID, d)
      return nil
    },
  })
  // Output: A(depth=0) B(depth=1) C(depth=1) D(depth=2) E(depth=2) F(depth=3) G(depth=3)
}
```
[![Go Playground](https://img.shields.io/badge/Go_Playground-BFS-blue?logo=go)](https://go.dev/play/p/sn7NhFsA2-M)

---

Next: [5. Shortest Paths: Dijkstra →](DIJKSTRA.md)

