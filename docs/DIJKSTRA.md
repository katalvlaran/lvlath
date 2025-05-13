## 5. Shortest Paths: Dijkstra

### What, Why, When (and where)

* **What is Dijkstra?**
  An efficient algorithm to compute the minimum-cost paths from a single source to all other vertices in a graph with non-negative edge weights. Widely used in routing, navigation, network optimization, and resource allocation.
* **Why** use Dijkstra? When you need fastest routes in road networks, optimal routing in logistics, or cost‑minimizing paths in resource allocation.
* **When** to choose it: your graph is *weighted* and *non‑negative*, and you need distances (and optionally predecessors) for *all* reachable nodes.

> **Goal:** Compute the minimum‐cost path from a single source to every other vertex in a weighted graph with non‑negative edge weights.

* *Where:* `algorithms.Dijkstra` in `lvlath/algorithms/dijkstra.go`

---

### ✏️ Mathematical Formulation
Dijkstra maintains a set of **visited** nodes whose shortest distance from the source is finalized, and a **min‑heap** (priority queue) of frontier nodes keyed by their tentative distance.

Let `G = (V, E)` be a graph with weight function
![\Large w: E \to \mathbb{R}_{\ge0}](https://latex.codecogs.com/svg.image?\large&space;w:E\to\mathbb{R}_{\ge0}) . We maintain:

* A **distance** map
  ![\Large d: V \to [0, \infty]](https://latex.codecogs.com/svg.image?\large&space;d:V\to[0,\infty])

* A **predecessor** map
  ![\Large \pi: V \to V \cup \{\text{nil}\]](https://latex.codecogs.com/svg.image?\large&space;\pi:V\to&space;V\cup\{\text{nil}\})

1. **Initialization**:
   ![\Large d[s] = 0, \quad d[v] = +\infty \quad \forall v \neq s, \quad \pi[v] = \text{nil}](https://latex.codecogs.com/svg.image?\large&space;d[s]=0,\quad&space;d[v]=&plus;\infty\quad\forall&space;v\neq&space;s,\quad\pi[v]=\text{nil})


2. **Relaxation**: For each edge
   ![\Large (u, v) \in E](https://latex.codecogs.com/svg.image?\large&space;(u,v)\in&space;E) , update:

   ![\Large \text{if}d[u]&plus;w(u,v)<d[v]\text{then}d[v]:=d[u]&plus;w(u,v),\;\pi[v]:=u](https://latex.codecogs.com/svg.image?\large&space;\text{if}d[u]&plus;w(u,v)<d[v]\text{then}d[v]:=d[u]&plus;w(u,v),\;\pi[v]:=u&space;)

3. **Main Loop** (using a min‑priority queue):

    * Extract the unsettled vertex `u` with minimal `d[u]`.
    * For each outgoing edge `(u,v)`: perform **Relaxation**.
    * Repeat until the queue is empty.

**Time Complexity:** `O((V + E) \log V)` with a binary heap.
**Space Complexity:** `O(V + E)`.

---

### Step-by-Step Pseudocode

```plaintext
function Dijkstra(G, source):
  for each vertex v in G:
    d[v] ← ∞
    π[v] ← nil
  d[source] ← 0

  Q ← min‑heap containing all vertices keyed by d[v]

  while Q is not empty:
    u ← ExtractMin(Q)
    for each neighbor v of u:
      let w = weight(u,v)
      if d[u] + w < d[v]:
        d[v] ← d[u] + w
        π[v] ← u
        DecreaseKey(Q, v, d[v])
  return d, π
```

**Example_#1:**
### 🎨 ASCII Illustration of a 4-node Graph
```
      [A]──2──[B]
     /         | 
   5/          |3
   /           |
  [C]────3────[D]
```
* Edges labeled with weights in parentheses.
* Start at **A** (distance 0).
* Explore frontier: B (2), C (5).
* Next extract B (2): relax B→D (2+1=3), B→C (2+3=5) (ties keep C at 5).
* Continue extracting D (3), relax D→C (3+3=6) (no update).
* Finally C (5).

Final distances: `d[A]=0`, `d[B]=2`, `d[D]=3`, `d[C]=5`.

### 🚀 Go Playground Example
```go
package main

import (
  "fmt"
  "github.com/katalvlaran/lvlath/algorithms"
  "github.com/katalvlaran/lvlath/core"
)

func main() {
  // Build a small weighted graph
  g := core.NewGraph(false, true)
  g.AddEdge("A", "B", 2)
  g.AddEdge("A", "C", 5)
  g.AddEdge("B", "D", 1)
  g.AddEdge("C", "D", 3)

  dist, parent, err := algorithms.Dijkstra(g, "A")
  if err != nil {
    panic(err)
  }

  fmt.Println("Distances from A:")
  for _, v := range []string{"A", "B", "C", "D"} {
    fmt.Printf(" %s: %d\n", v, dist[v])
  }

  // Reconstruct path A→D
  path := []string{"D"}
  for u := parent["D"]; u != ""; u = parent[u] {
    path = append([]string{u}, path...)
  }
  fmt.Println("Path A→D:", path)
}
```
[![Go Playground](https://img.shields.io/badge/Go_Playground-Dijkstra_1-blue?logo=go)](https://go.dev/play/p/jHfG9cqil6-)

### Pitfalls & Best Practices

* **Non‑negative weights only**: Dijkstra fails with negative edges—use Bellman‑Ford instead.
* **Graph size**: For very large graphs, consider A\* with a heuristic, or multi‑level techniques.
* **Max heap operations**: `DecreaseKey` is essential; the `nodePQ` in `lvlath` pushes duplicates but skips visited nodes safely.


---
**Example_#2:**
### 🎨 ASCII Illustration of a 5-node Graph
```text
      (A)
      / \
    5/   \2
   (B)─1─(C)
    |   / |
   2|  3  |4
    |/    |
   (D)─1─(E)
       
```

* **Weights** labeled on edges.
* **Source**: A (distance 0).
* **First extraction**: A → relax B (5), C (2).
* Continue until all settled.


### 🚀 Go Playground Example
```go
package main

import (
    "fmt"

    "github.com/katalvlaran/lvlath/algorithms"
    "github.com/katalvlaran/lvlath/core"
)

func main() {
    g := core.NewGraph(false, true)
    // build graph
    g.AddEdge("A","B",5)
    g.AddEdge("A","C",2)
    g.AddEdge("B","C",1)
    g.AddEdge("B","D",2)
    g.AddEdge("C","D",3)
    g.AddEdge("C","E",4)
    g.AddEdge("D","E",1)

    dist, _, err := algorithms.Dijkstra(g, "A")
    if err != nil {
        fmt.Println("Error:", err)
        return
    }
    fmt.Println("Distances from A:")
    for v, d := range dist {
        fmt.Printf("  %s -> %s = %d\n", "A", v, d)
    }
}
```
[![Playground – Dijkstra](https://img.shields.io/badge/Go_Playground-Dijkstra_2-blue?logo=go)](https://go.dev/play/p/hchsjSxesxS)

---
**Example_#3:**
### 🎨 ASCII Illustration of a 5-node Graph

```text
     (A)
   5/   \2 
   /     \
 (B)──1──(C)
  |       |
 2|       |4
  |       |
 (D)──1──(E)
```
* **Vertices:** A–E
* **Edge-weights** labeled
* **Source:** A

**Walkthrough:**

* **Step 1:** Extract **A** (0), relax neighbors B (5), C (2).
* **Step 2:** Extract **C** (2), relax E (2+?), D, F…
* Continue until all distances settled.

### 🚀 Go Playground Example
```go
package main

import (
    "fmt"
    "github.com/katalvlaran/lvlath/algorithms"
    "github.com/katalvlaran/lvlath/core"
)

func main() {
    // Build weighted, undirected graph
    g := core.NewGraph(false, true)
    edges := []struct {u, v string; w int64}{
        {"A","B",5}, {"A","C",2}, {"B","C",1},
        {"B","D",2}, {"C","E",4}, {"D","E",1},
    }
    for _, e := range edges {
        g.AddEdge(e.u, e.v, e.w)
    }

    dist, prev, err := algorithms.Dijkstra(g, "A")
    if err != nil {
        fmt.Println("Error:", err)
        return
    }

    fmt.Println("Shortest distances from A:")
    for _, v := range []string{"A","B","C","D","E"} {
        fmt.Printf("  A → %s = %d (via %s)\n", v, dist[v], prev[v])
    }
}
```
[![Playground – Dijkstra](https://img.shields.io/badge/Go_Playground-Dijkstra_3-blue?logo=go)](https://go.dev/play/p/BIh9sFghSBq)

---

### ⚙️ Best Practices & Pitfalls

* **Ensure non-negative weights**: Negative edges invalidate Dijkstra (use Bellman–Ford).
* **Skip stale heap entries**: Always compare popped priority with current `dist[u]`.
* **Guard against overflow**: For large weights, check `du + w` before assignment.
* **Performance tuning**: Consider Fibonacci or radix heaps for faster decrease-key.

---

Next: [6. Minimum Spanning Trees: Prim & Kruskal →](PRIM_%26_KRUSKAL.md)
