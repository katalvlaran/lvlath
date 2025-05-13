## 7. Max-Flow: Ford–Fulkerson / Edmonds-Karp / Dinic

### What, Why, When

**Maximum flow** finds the greatest possible throughput from a **source** `s` to a **sink** `t` in a **flow network** `G = (V, E)` where each edge `(u,v)` has a **capacity** `c(u,v)`. The goal is to assign a **flow** `f(u,v)` to each edge such that:

1. **Capacity constraint**: Flow on each edge does not exceed its capacity and is non-negative.
2. **Conservation of flow**: Except for `s` and `t`, the total inflow equals total outflow at every vertex.
3. **Objective**: Maximize the total flow leaving the source (equivalently entering the sink).

**Use cases**: network routing, traffic engineering, bipartite matching, project scheduling, and more.

**When to choose max-flow**: whenever you need to optimize resource distribution through a constrained network.

---

### Mathematical Formulation

Let `f : E \to \mathbb{R}` be a flow. We require:

Capacity constraints:

![\Large 0 \le f(u,v) \le c(u,v)](https://latex.codecogs.com/svg.image?%5Clarge%200%20%5Cleq%20f%28u%2Cv%29%20%5Cleq%20c%28u%2Cv%29)

Flow conservation for all vertices except `s` and `t`:

![\Large \sum\_{u\in V}f(u,v)=\sum\_{w\in V}f(v,w)\quad\text{for }v\neq s,t](https://latex.codecogs.com/svg.image?%5Clarge%20%5Csum_%7Bu%5Cin%20V%7Df%28u%2Cv%29%3D%5Csum_%7Bw%5Cin%20V%7Df%28v%2Cw%29%5Cquad%5Ctext%7Bfor%20%7Dv%5Cneq%20s%2Ct)

The value of a flow is the net outflow from the source:

![\Large |f|=\sum\_{v\in V}f(s,v)](https://latex.codecogs.com/svg.image?%5Clarge%20%7Cf%7C%3D%5Csum_%7Bv%5Cin%20V%7Df%28s%2Cv%29)

Define the **residual capacity**:

![\Large c\_f(u,v)=c(u,v)-f(u,v)](https://latex.codecogs.com/svg.image?%5Clarge%20c_f%28u%2Cv%29%3Dc%28u%2Cv%29-f%28u%2Cv%29)

and allow a **reverse edge** with capacity `f(u,v)` to enable flow cancellation.

---

### Algorithms Overview

---

## 7.2 Ford–Fulkerson (DFS)

**Core idea:** use a simple depth‑first search to find _any_ augmenting path, then augment. Repeat until no path remains.

### Pseudocode
```text
procedure FordFulkerson(G, s, t):
  for each edge (u,v) in G:
    flow[u][v] = 0
    flow[v][u] = 0  # reverse
  maxFlow = 0

  # Build residual capacities
  repeat:
    visited = {};
    path, bottleneck = DFS_FindPath(s, t, ∞)
    if bottleneck == 0: break
    # Augment flow along path
    for each (u→v) in path:
      flow[u][v] += bottleneck
      flow[v][u] -= bottleneck
    maxFlow += bottleneck
  return maxFlow
````

### ASCII Example

```
Network:
  
  [S]───4───[A]
   |         | 
   2         3
   |         |
  [B]───5───[T]

# First DFS might find: s→A→t: bottleneck = min(4,3)=3
# New residual:
#   s→A cap=1, A→s cap=3; A→t=0, t→A=3

# Next DFS might find: s→B→t: bottleneck = min(2,5)=2
# Residual: s→B=0, B→s=2; B→t=3, t→B=2

# No more paths => maxFlow=5
```

### Go Example

```go
package main

import (
  "context"
  "fmt"
  "github.com/katalvlaran/lvlath/core"
  "github.com/katalvlaran/lvlath/flow"
)

func main() {
          ctx := context.Background()
      g := core.NewGraph(true, true)
      g.AddEdge("s", "A", 4)
      g.AddEdge("A", "t", 3)
      g.AddEdge("s", "B", 2)
      g.AddEdge("B", "t", 5)
      maxFlow, _, _ := flow.FordFulkerson(ctx, g, "s", "t", nil)
      fmt.Println("MaxFlow =", maxFlow)
    // Output: MaxFlow = 5
}
```

[![Go Playground](https://img.shields.io/badge/Go_Playground-MaxFlow_FordFulkerson-blue?logo=go)](https://go.dev/play/p/wCBSnZRQbCE)

**Time complexity:** O(E·F) in worst case (F = maxFlow).
**Pitfall:** DFS can pick poor paths, causing many small augmentations.

---

## 7.3 Edmonds–Karp (BFS)

**Enhancement:** always pick the *shortest* augmenting path (fewest edges) via BFS.  Ensures at most O(V·E²) time.

### Pseudocode

```text
procedure EdmondsKarp(G, s, t):
  build residual graph
  maxFlow = 0
  while (path := BFS(s→t)) exists:
    bottleneck = min residual cap along path
    augment flow
    maxFlow += bottleneck
  return maxFlow
```

### Highlights

* BFS finds the shallowest path in O(E).
* Guarantees O(V·E²) total.

### Go Snippet

```go
package main

import (
  "context"
  "fmt"

  "github.com/katalvlaran/lvlath/core"
  "github.com/katalvlaran/lvlath/flow"
)

func main() {
    ctx := context.Background()
    g := core.NewGraph(true, true)
    g.AddEdge("s", "A", 4)
    g.AddEdge("A", "t", 3)
    g.AddEdge("s", "B", 2)
    g.AddEdge("B", "t", 5)
    maxFlow, _, _ := flow.EdmondsKarp(ctx, g, "s", "t", nil)
    // Same network as above => mf = 5
    fmt.Println("MaxFlow =", maxFlow)
}
```
[![Go Playground](https://img.shields.io/badge/Go_Playground-MaxFlow_EdmondsKarp-blue?logo=go)](https://go.dev/play/p/5ALa9IQF9A5)

---

## 7.4 Dinic (Level Graph + Blocking Flow)

**Further refinement:** build a **level graph** with BFS (edges only to next level), then send multiple flows via DFS blocking until exhausted, then rebuild. Runs in O(E · √V) on unit networks.

### Steps

1. **Level graph:** BFS from `s` assigns level\[v] = distance from `s`.
2. **Blocking flow:** DFS only along edges to nodes at level+1, until no augmenting remains.
3. **Repeat** until no level graph path to `t`.


```
      (s)
     /  \  
  10/    \8
   v      v
 (u1)-5->(u2)---10->(t)
  |     ^/ 
 2|   3//4 
  v   /v   
  ( u3 )   
```

1. Initially zero flow. Residual capacities equal original capacities.
2. BFS finds path `s→u1→u2→t` with bottleneck `min(10,5,10)=5`.
3. Augment flow by 5 along that path, update residual graph.

---

### 🚀 Go Playground Example

```go
package main

import (
	"fmt"

	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/flow"
)

func main() {
	// Construct directed graph with capacities
	g := core.NewGraph(true, false)
	// edges: u, v, capacity
	edges := []struct {
		u, v string
		c    int64
	}{
		{"s", "u1", 10}, {"s", "u2", 8},
		{"u1", "u2", 5}, {"u1", "u3", 2},
		{"u2", "t", 10}, {"u3", "u2", 3}, {"u2", "u3", 4},
	}
	for _, e := range edges {
		g.AddEdge(e.u, e.v, e.c)
	}

	// Compute max flow from s to t
	maxFlow, _, err := flow.Dinic(g, "s", "t", nil)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Maximum flow from s to t = %d\n", maxFlow)
}
```
[![Go Playground](https://img.shields.io/badge/Go_Playground-MaxFlow_Dinic-blue?logo=go)](https://go.dev/play/p/v8SuFpLlFSQ)

---

### 7.5 Pitfalls & Best Practices

* **Integer overflow**: watch sums of flows on high-capacity networks; use wide types.
* **Zero-capacity edges**: ensure no zero cycles that stall augmentation.
* **Algorithm choice**: for dense graphs, Dinic outperforms Edmonds–Karp; for unit networks, scaling algorithms or push–relabel may be superior.
* **Parallel edges**: lvlath merges capacities; split if distinct semantics required.
* **Memory**: residual graph doubles edges; large networks may need streaming or out-of-core solutions.

---

Next: [8. Dynamic Time Warping (DTW) →](DTW.md)
