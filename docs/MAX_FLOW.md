## 7. Max-Flow: Ford-Fulkerson / Edmonds-Karp / Dinic
### 7.1 What, Why, When

#### What
A **maximum-flow** problem seeks the largest possible “flow” from a designated **source** vertex \(s\) to a **sink** vertex \(t\) in a **flow network** \(G = (V, E)\). Each directed edge \((u \to v)\) carries a **capacity** \(c(u,v)\) limiting the amount of flow \(f(u,v)\) it can carry. A valid flow must satisfy:

1. **Capacity constraints**

   $$0 \;\le\; f(u,v)\;\le\; c(u,v) \quad\forall\;(u,v)\in E$$ .

2. **Flow conservation**

   $$\sum_{u:(u\to v)\in E} f(u,v) \;=\; \sum_{w:(v\to w)\in E} f(v,w) \quad\forall\,v\in V\setminus\{s,t\}$$ .

The **value** of a flow is the total outflow from the source

$$|f| \;=\; \sum_{v:(s\to v)\in E} f(s,v) \;=\; \sum_{u:(u\to t)\in E} f(u,t)$$ .

#### Why
- **Network routing & traffic engineering**: maximize throughput under link-capacity limits.
- **Bipartite matching & scheduling**: model assignments or resource allocation as flows.
- **Cut-based analysis**: by the max-flow/min-cut theorem, the maximum flow equals the capacity of the minimum cut, giving insight into network bottlenecks.

#### When
Use a max-flow algorithm whenever you need to **optimize** the distribution of a **limited resource** through a directed network, subject to per-edge capacity constraints. Common scenarios include:

- Ensuring the greatest data traffic from a server to clients.
- Assigning tasks to workers under workload limits.
- Analyzing vulnerability points (min-cuts) in infrastructure.

---

### 7.2 Mathematical Formulation

Given a directed graph `(G=(V,E)` and capacities $$\(c: E \to \mathbb{R}_{\ge0}\)$$, we seek a flow

$$f: E \;\to\; \mathbb{R}_{\ge0}$$

satisfying:

1. **Capacity constraints**

   $$0 \;\le\; f(u,v)\;\le\; c(u,v), \quad\forall\;(u,v)\in E$$ .

2. **Conservation of flow**

   $$\sum_{u\colon(u,v)\in E} f(u,v)\;=\; \sum_{w\colon(v,w)\in E} f(v,w), \quad\forall\,v\in V\setminus\{s,t\}$$ .

3. **Maximize**

   $$|f| \;=\; \sum_{v\colon(s,v)\in E} f(s,v)\;=\; \sum_{u\colon(u,t)\in E} f(u,t)$$ .

We introduce the **residual capacity**

$$c_f(u,v) \;=\; c(u,v) \;-\; f(u,v), \quad c_f(v,u) \;=\; f(u,v)$$ ,

which yields a **residual graph** $$\(G_f\)$$ whose edges represent remaining forward capacity and potential to cancel previous flow.


### Algorithms Overview

---
## 7.3.1 Ford–Fulkerson (DFS)

### Core Idea
Use a **depth-first search (DFS)** to find **any** path from source `(s)` to sink `(t)` in the **residual graph** $$(G_f)$$ that has **positive residual capacity** on every edge. Once a path is found, **augment** (push) the maximum possible flow (the **bottleneck**) along that path, update the residual capacities, and repeat until no augmenting path remains.

### Features
- **Simplicity**: conceptually the easiest max-flow algorithm to implement.
- **Residual graph** built on the fly: forward edges $$\(c_f(u,v)=c(u,v)-f(u,v)\)$$ , reverse edges $$\(c_f(v,u)=f(u,v)\)$$ .
- **Augmentation** along arbitrary DFS-found paths.

### Improvements & Advantages
- Ideal for **small** or **integral** networks where total flow `(F)` is modest.
- Requires only **O(V+E)** extra memory for DFS stack and residual map.
- Serves as a foundation for more advanced algorithms (e.g., Dinic, Push–Relabel).

---

### Pseudocode
```text
procedure FordFulkerson(G, s, t):
  for each edge (u,v) in G:
    resid[u][v] ← capacity(u,v)
    resid[v][u] ← 0  // initialize reverse edges
  maxFlow ← 0

  repeat:
    visited ← empty set
    (path, bottleneck) ← DFS_FindPath(s, t, ∞)
    if bottleneck = 0 then
      break  // no more augmenting paths
    for each consecutive (u→v) in path:
      resid[u][v] ← resid[u][v] - bottleneck
      resid[v][u] ← resid[v][u] + bottleneck
    maxFlow ← maxFlow + bottleneck
  until false

  return maxFlow

function DFS_FindPath(u, t, flow):
  if u = t then return ( [t], flow )
  mark u as visited
  for each v adjacent to u where resid[u][v] > 0:
    if v not visited:
      f ← min(flow, resid[u][v])
      (subpath, bottleneck) ← DFS_FindPath(v, t, f)
      if bottleneck > 0:
        return ( [u] ++ subpath, bottleneck )
  return ( [], 0 )
```

### Highlights
- **Residual network**: dynamically updated map of forward/reverse capacities.
- **DFS stop condition**: visits each reachable vertex at most once per augment.
- **Augment**: path length \(\le V\); update cost proportional to path length.

### ASCII Diagram
```
   (s)
   /  \ 4         3
 5/    \        \/
(A)     (B) —6→ (t)
 \       \
  2       1
   \       \
   (C) —4→ (t)
   
   
   
   
   
```
1. DFS might choose \(s→A→t\): bottleneck = min(5,3)=3.
2. Update residual: forward(s,A)=2, reverse(A,s)=3; forward(A,t)=0, reverse(t,A)=3.
3. Next DFS: \(s→C→t\) bottleneck = min(15,4)=4.
4. Repeat until no path; total flow = 3 + 4 + ... = \(F_{max}\).

### Go Playground Example
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
  g := core.NewGraph(
    core.WithDirected(true),
    core.WithWeighted(),
    core.WithMultiEdges(),
  )
  // Complex 8‑vertex example
  g.AddEdge("s", "a", 5)
  g.AddEdge("s", "c", 15)
  g.AddEdge("a", "b", 8)
  g.AddEdge("b", "d", 10)
  g.AddEdge("c", "d", 5)
  g.AddEdge("c", "e", 10)
  g.AddEdge("e", "d", 10)
  g.AddEdge("d", "t", 10)
  g.AddEdge("e", "t", 5)

  opts := flow.DefaultOptions()
  opts.Ctx = ctx

  maxFlow, _, err := flow.FordFulkerson(g, "s", "t", opts)
  if err != nil {
    panic(err)
  }
  fmt.Printf("Maximum flow = %d\n", maxFlow)
  // Output: Maximum flow = 15
}
```

[![Go Playground](https://img.shields.io/badge/Go_Playground-MaxFlow_FordFulkerson-blue?logo=go)](https://go.dev/play/p/wCBSnZRQbCE)

### Complexity
- **Time**: \(O(E \times F)\), where \(F\) is the integer maximum flow.
- **Memory**: \(O(V + E)\) for the residual capacity map and DFS stack.

---

## 7.3.2. Edmonds–Karp (Breadth‑First Search)

### Core Idea
Always choose the shortest (fewest‑edge) augmenting path from source `S` to sink `T` by performing a BFS on the **residual network**.  This strategy bounds the number of augmentations and guarantees a worst‑case **polynomial** time complexity.

### Features & Advantages

- **Shortest‑path selection**: `BFS` ensures each augmenting path increases the distance (in edges) between  `S` and `T` by at least `1` before being reused.
- **Polynomial bound**: Guarantees `O(V · E²)` runtime on integer capacities versus `O(E · F)` for naive `DFS`.
- **Deterministic performance**: More predictable than Ford–Fulkerson, especially on adversarial networks.
- **Simple to implement**: Leverages familiar `BFS`, minimal extra data structures.

### Complexity

- **Time**: `O(V · E²)` in the worst case (each of up to `O(E · F)` augmentations uses `O(E)` `BFS`, but path lengths strictly increase).
- **Memory**: `O(V + E)` for residual map and `BFS` queue.

### Pseudocode
```text
procedure EdmondsKarp(G, S, T):
  initialize flow f(u,v) = 0 for all (u,v)
  build residual capacities r(u,v) = c(u,v)
  maxFlow ← 0
  while true:
    # 1. BFS to find shortest augmenting path
    parent[] ← empty, bottleneck[] ← 0
    queue ← [S], bottleneck[S] ← ∞
    while queue not empty and bottleneck[T] = 0:
      u ← dequeue(queue)
      for each neighbor v of u with r(u,v) > 0 and bottleneck[v] = 0:
        parent[v] ← u
        bottleneck[v] ← min(bottleneck[u], r(u,v))
        enqueue(queue, v)
    if bottleneck[T] = 0:
      break   # no more augmenting paths
    # 2. Augment along path
    flowAmount ← bottleneck[T]
    v ← T
    while v ≠ S:
      u ← parent[v]
      r(u,v) ← r(u,v) - flowAmount
      r(v,u) ← r(v,u) + flowAmount
      v ← u
    maxFlow ← maxFlow + flowAmount
  return maxFlow
```

### Highlights
- **BFS distance labels** prevent cycling on long paths.
- **Bottleneck tracking** in `BFS` avoids separate scan to compute minimum capacity.
- **Reverse edges** in residual graph allow cancellation of previous flows.

### ASCII Example (9 vertices, 14 edges)
```
          [S]
         / | \
       5/ 7|  \15
       /   |   \
    [A]   [B]   [C]
     | \   |   / |
    8| 3\ 6|  /5 |10
     |   \ | /   |
    [D]─7─[E]─8─[F]
     | \     \   |
    7|  \2    \4 |6
     |   \     \ |
    [G]─9─[H]─8─[T]
        
```
- **Vertices**: S,A,B,C,D,E,F,G,H,T
- **Edges** labeled with capacities
- BFS finds S→B→E→H→T (capacity = min(15, 3, 9, 11) = 3)
- Next BFS picks S→C→F→T, etc., until no path remains.

### Go Playground Example
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
  g := core.NewGraph(
    core.WithDirected(true),
    core.WithWeighted(),
  )
  // Build 10-edge network
  edges := []struct{U,V string;C int64}{
    {"S","A",5}, {"S","B",15}, {"S","C",10},
    {"A","D",8}, {"B","E",3}, {"C","F",5},
    {"D","G",7}, {"E","H",9}, {"F","T",11},
    {"G","T",2}, {"H","T",4},
  }
  for _, e := range edges {
    g.AddEdge(e.U, e.V, e.C)
  }
  opts := flow.DefaultOptions()
  opts.Ctx = ctx
  maxFlow, _, err := flow.EdmondsKarp(g, "S", "T", opts)
  if err != nil {
    panic(err)
  }
  fmt.Println("MaxFlow =", maxFlow)
}
```

[![Go Playground](https://img.shields.io/badge/Go_Playground-MaxFlow_EdmondsKarp-blue?logo=go)](https://go.dev/play/p/5ALa9IQF9A5)

---

## 7.3.3 Dinic (Level Graph + Blocking Flow)

### Core Idea
Dinic’s algorithm accelerates the classic augmenting‐path approach by layering the network into a _level graph_ and then finding _blocking flows_ that saturate all shortest paths in one `BFS`–`DFS` round. This dramatically reduces the number of augmentations compared to naive `DFS` or `BFS` alone.

### Key Features
- **Level Graph Construction**: A `BFS` from the source partitions vertices by their distance (in edges) from `s`. Only edges **u→v** with remaining capacity and `level[v] = level[u] + 1` are retained.
- **Blocking Flow**: On the level graph, a `DFS` repeatedly pushes flow along disjoint shortest paths until no more augmenting path exists within this level structure.
- **Repeat**: Once the blocking flow is exhausted, rebuild the level graph and repeat until the sink `t` is unreachable.
- **Optional Level Rebuild Interval**: For large networks, you may choose to rebuild the level graph after a fixed number of `DFS` pushes to maintain performance balance (controlled by `opts.LevelRebuildInterval`).

###Complexity & Advantages
- **Time**:  $$O(E\sqrt V)$$ on unit‐capacity networks; in practice near $$O(E\sqrt V)$$ for many graphs.
- **Memory**: $$O(V + E)$$ for residual capacities, level map, and iterators.
- Compared to Edmonds–Karp $$O(VE^2)$$ , Dinic typically offers an order‐of‐magnitude speedup on dense or high‐capacity graphs.

### Pseudocode
```text
procedure Dinic(G, s, t):
  build residual capacities cap[u][v]
  maxFlow ← 0
  while true:
    # 1. Level Graph via BFS
    for each v in V: level[v] ← -1
    level[s] ← 0
    enqueue(s)
    while queue not empty:
      u ← dequeue()
      for each edge u→v with cap[u][v] > 0:
        if level[v] < 0:
          level[v] ← level[u] + 1
          enqueue(v)
    if level[t] < 0: break  # no more paths

    # 2. Build adjacency lists for level graph
    for each u in V:
      next[u] ← [v for v in neighbors(u) if cap[u][v] > 0 and level[v] = level[u] + 1]
      iter[u] ← 0

    # 3. DFS Blocking Flow
    while pushed ← dfs(s, t, ∞) > 0:
      maxFlow += pushed
    # optional: if opts.LevelRebuildInterval > 0 and reached count, break
  return maxFlow

function dfs(u, t, flow):
  if u = t: return flow
  for i from iter[u] to len(next[u])-1:
    v ← next[u][i]; iter[u] ← i + 1
    if cap[u][v] ≤ 0: continue
    send ← min(flow, cap[u][v])
    pushed ← dfs(v, t, send)
    if pushed > 0:
      cap[u][v] -= pushed
      cap[v][u] += pushed
      return pushed
  return 0
```

### Highlights
- **Level Graph** filters out long detours, focusing search on shortest‐distance edges.
- **Blocking Flow** pushes multiple units of flow per `BFS`, reducing total rounds.
- **Optional Rebuild** allows tuning between overhead of BFS and cost of many `DFS` calls.

### ASCII Example
```text
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
1. **Level 0**: s
2. **Level 1**: u1 (10), u2 (8)
3. **Level 2**: u2 via u1 (5), u3 via u1 (2)
4. **First blocking flow**: s→u1→u2→t carries **5**; also s→u1→u3→u2→t carries **2**; update residual.
5. **Rebuild** and repeat until no path.

### Go Playground Example
```go
package main

import (
  "fmt"

  "github.com/katalvlaran/lvlath/core"
  "github.com/katalvlaran/lvlath/flow"
)

func main() {
  g := core.NewGraph(core.WithDirected(true), core.WithWeighted())
  edges := []struct{ u, v string; c int64 }{
    {"s","u1",10}, {"s","u2",8},
    {"u1","u2",5}, {"u1","u3",2},
    {"u2","t",10},{"u3","u2",3}, {"u2","u3",4},
  }
  for _, e := range edges {
    g.AddEdge(e.u, e.v, e.c)
  }

  opts := flow.DefaultOptions()
  // tune rebuild: opts.LevelRebuildInterval = 2

  maxFlow, _, err := flow.Dinic(g, "s", "t", opts)
  if err != nil {
    panic(err)
  }
  fmt.Printf("Maximum flow s→t = %d\n", maxFlow)
}
// Output:
// Maximum flow s→t = 10
```
[![Go Playground](https://img.shields.io/badge/Go_Playground-MaxFlow_Dinic-blue?logo=go)](https://go.dev/play/p/v8SuFpLlFSQ)

---

### 7.4 Pitfalls & Best Practices

1. **Integer overflow**  
   When capacities or numbers of augmentations are large, sums of flows can exceed 32-bit limits. Always use **int64** (or larger) for capacities and accumulators.

2. **Zero-capacity edges**  
   Edges with `(c(u,v)=0)` may clutter your residual graph—either filter them out early or ensure your algorithm skips them to avoid needless work.

3. **Choice of algorithm**
    - **Ford–Fulkerson (DFS)**: simple but worst-case $$\(O(E\cdot F)\)$$ may be prohibitive if `(F)` is large.
    - **Edmonds–Karp (BFS)**: polynomial $$\(O(V\,E^2)\)$$ guarantees, but can be slow on dense graphs.
    - **Dinic**: $$\(O(E\sqrt V)\)$$ on unit networks and often very fast in practice; preferred for large or dense graphs.

4. **Parallel edges and loops**
    - **Multi-edges**: `lvlath/core` by default **aggregates** parallel capacities—ensure this matches your model semantics.
    - **Loops** $$\((v\to v)\)$$: typically ignored in augmentation since they cannot contribute to source–sink throughput.

5. **Residual graph size**  
   Residual graph may have up to twice as many edges as the original (forward + backward). For huge networks, consider **streaming** or **out-of-core** techniques to limit memory.

6. **Precision and thresholds**  
   When capacities are floating-point, use an **epsilon** threshold to treat very small capacities as zero, avoiding infinite augmentation loops.

7. **Algorithm tuning**
    - **Verbose logging**: helpful for debugging small examples, but **disable** in benchmarks to avoid I/O overhead.
    - **LevelRebuildInterval** `Dinic`: in some variants, rebuilding the level graph after a fixed number of push operations can yield practical speedups—tune this parameter for your workloads.

---

Next: [8. Dynamic Time Warping (DTW) →](DTW.md)
