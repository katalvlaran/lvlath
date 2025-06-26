## 5. Shortest Paths: Dijkstra

### 5.1. What, Why, When (and where)

**What?**  
Dijkstra’s algorithm computes the minimum-cost paths from a single source vertex to all other vertices in a weighted graph with non-negative edge weights. It forms the backbone of routing, navigation, and resource-allocation systems.

**Why?**
- **Correctness:** Guarantees optimal paths when all weights ≥ 0.
- **Performance:** Achieves O((V + E) log V) time using a binary heap.
- **Completeness:** Produces a full distance map and optional predecessor map for path reconstruction.

**When?**
- All edge weights in your graph are non-negative.
- You need distances to _every_ reachable node (not just a single destination).
- You may combine with `lvlath/matrix` for matrix-based workflows.

**Where?**
- **Implementation:** `github.com/katalvlaran/lvlath/dijkstra.Dijkstra`
- **Graph integration:** Works seamlessly with `lvlath/core.Graph` and its `WithWeighted()`, `WithMixedEdges()` settings.

> **Goal:** Compute the minimum‐cost path from a single source to every other vertex in a weighted graph with non‑negative edge weights.
---

### 5.2. Mathematical Formulation
Dijkstra maintains a set of **visited** nodes whose shortest distance from the source is finalized, and a **min‑heap** (priority queue) of frontier nodes keyed by their tentative distance.

Let `G = (V, E)` be a graph with weight function
![\Large w: E \to \mathbb{R}_{\ge0}](https://latex.codecogs.com/svg.image?\large&space;w:E\to\mathbb{R}_{\ge0}) . We maintain:

* A **distance** map
  ![\Large d:_V_\to_\[0,_\infty\]](https://latex.codecogs.com/svg.image?\large&space;d:V\to[0,\infty])


* A **predecessor** map
  ![\Large \pi: V \to V \cup \{\text{nil}\]](https://latex.codecogs.com/svg.image?\large&space;\pi:V\to&space;V\cup\{\text{nil}\})


1. **Initialization**: $$d[s] = 0, \quad d[v] = +\infty \quad \forall v \neq s, \quad \pi[v] = \mathrm{nil}$$
   
   $$d[s] = 0, \quad d[v] = +\infty \quad \forall v \neq s, \quad \pi[v] = \mathrm{nil}$$

    ![\Large d\[s\] = 0, \quad d\[v\] = +\infty \quad \forall v \neq s, \quad \pi\[v\] = \text{nil}](https://latex.codecogs.com/svg.image?\large&space;d[s]=0,\quad&space;d[v]=&plus;\infty\quad\forall&space;v\neq&space;s,\quad\pi[v]=\text{nil})


2. **Relaxation**: For each edge ![\Large (u, v) \in E](https://latex.codecogs.com/svg.image?\large&space;(u,v)\in&space;E) , update:


   ![\Large \text{if}d\[u\]&plus;w(u,v)<d\[v\]\text{then}d\[v\]:=d\[u\]&plus;w(u,v),\;\pi\[v\]:=u](https://latex.codecogs.com/svg.image?\large&space;\text{if}d[u]&plus;w(u,v)<d[v]\text{then}d[v]:=d[u]&plus;w(u,v),\;\pi[v]:=u&space;)

3. **Main Loop** (using a min‑priority queue):

    * Extract the unsettled vertex `u` with minimal `d[u]`.
    * For each outgoing edge `(u,v)`: perform **Relaxation**.
    * Repeat until the queue is empty.

**Time Complexity:** `O((V + E) \log V)` with a binary heap.
**Space Complexity:** `O(V + E)`.

---

### 5.3. Step-by-Step Pseudocode

```plaintext
function Dijkstra(Graph G, source s):
    // 1) Initialize distance and predecessor
    for each vertex v in G.Vertices():
        d[v] ← +∞              // unknown distance
        π[v] ← nil             // no predecessor
    d[s] ← 0                  // distance to source is zero

    // 2) Build priority queue Q of all vertices keyed by d[v]
    Q ← MinHeap()
    Q.Insert(s, 0)

    // 3) Main loop: extract-min and relax
    while Q is not empty:
        (u, distU) ← Q.ExtractMin()    // vertex with smallest tentative distance
        if distU > MaxDistance:
            break                      // distances beyond threshold are ignored

        for each edge e in G.Neighbors(u):
            let v = e.To
            let w = e.Weight
            // 3a) Mixed-edge and direction filter
            if e.Directed and e.From != u:
                continue               // skip reverse of directed edge

            // 3b) Precondition checks
            if w < 0:
                error "negative weight"
            if w ≥ InfEdgeThreshold:
                continue               // treat heavy edge as impassable

            // 3c) Relaxation step
            alt ← distU + w
            if alt < d[v]:
                d[v] ← alt
                π[v] ← u
                // 3d) Update priority queue (lazy decrease-key)
                Q.Insert(v, alt)

    return d, π
```

*Complexity:* O((V + E) log V) time, O(V + E) space.
### 5.4. Examples
### 5.4.1 Medium Example: 4‑Node Graph Walkthrough

This example demonstrates Dijkstra’s algorithm on a small undirected graph of four vertices. We trace each step, show the evolving priority queue, and reconstruct the shortest path.

#### Graph Structure

```
      [A]──2──[B]
     /         |
   5/          |1
   /           |
  [C]────3────[D]
```

- **Vertices**: A, B, C, D
- **Edges & Weights**:
    - A–B (weight 2)
    - A–C (weight 5)
    - B–D (weight 1)
    - C–D (weight 3)

We start from source **A** (distance 0) and compute distances to all vertices.

| Step | PQ Contents               | Action                                | Resulting Distances         |
|:----:|:--------------------------|:--------------------------------------|:----------------------------|
|  0   | [(A,0)]                   | Initialize: d[A]=0, others=∞         | A=0, B=∞, C=∞, D=∞          |
|  1   | [(B,2),(C,5)]             | Relax A’s neighbors: B←2, C←5         | B=2, C=5, D=∞              |
|  2   | [(C,5),(D,3)]             | Pop B(2), relax B→D(3), B→C(2+3=5)    | D=3 (better), C stays 5    |
|  3   | [(D,3),(C,5)]             | Pop D(3), relax D→C(3+3=6 no change)  | distances unchanged        |
|  4   | [(C,5)]                   | Pop C(5) - all neighbors processed    | final: A=0, B=2, D=3, C=5   |

#### Go Code Example

Below is a fully commented Go program using `lvlath/core` and `lvlath/dijkstra`. It builds the graph, runs Dijkstra, prints distances, and reconstructs the shortest path from A to D.

```go
package main

import (
  "fmt"

  "github.com/katalvlaran/lvlath/core"
  "github.com/katalvlaran/lvlath/dijkstra"
)

func main() {
  // 1) Create an undirected, weighted graph.
  g := core.NewGraph(core.WithWeighted()) // core.WithWeighted() enables non-negative edge weights.

  // 2) Define the edges of the 4-node graph.
  //    AddEdge(u, v, w) creates both u→v and v→u in an undirected graph.
  g.AddEdge("A", "B", 2)
  g.AddEdge("A", "C", 5)
  g.AddEdge("B", "D", 1)
  g.AddEdge("C", "D", 3)

  // 3) Run Dijkstra from source "A", requesting predecessor map.
  //    WithReturnPath() returns a map of each node's predecessor on the shortest path.
  dist, prev, _ := dijkstra.Dijkstra(g, dijkstra.Source("A"), dijkstra.WithReturnPath())

  // 4) Print the computed distances from A to each vertex.
  fmt.Println("Distances from A:")
  for _, v := range []string{"A", "B", "C", "D"} {
    fmt.Printf("  %s: %d\n", v, dist[v])
  }

  // 5) Reconstruct and print the shortest path from A to D.
  path := []string{"D"} // Start at target D and walk backwards via prev[] until A.
  for u := prev["D"]; u != ""; u = prev[u] {
    path = append([]string{u}, path...)
  }
  fmt.Println("Path A→D:", path)
}
```
*Complexity:* Time O((V+E) log V), Space O(V+E). This code showcases clear separation of graph construction, algorithm execution, and result interpretation.

**Expected Output:**
```
Distances from A:
  A: 0
  B: 2
  C: 5
  D: 3
Path A→D: [A B D]
```
[![Go Playground](https://img.shields.io/badge/Go_Playground-Dijkstra_1-blue?logo=go)](https://go.dev/play/p/81N4Vr_gEXt)

---

### 5.4.2. Example #2: Advanced 10-Node Graph Demonstration

#### ASCII Illustration of the Graph
```
      (A)                     (J)
      / \                     / \
    5/   \2                 5/   \2
   (B)─1─(C)               (I)─1─(H)
    |   / |                 |   / |
   2|  3  |4               2|  3  |4
    |/    |                 |/    |
   (D)─1─(E)───────17──────(F)─1─(G)
    └──────────19───────────┘       
```
- **Vertices:** A, B, C, D, E, F, G, H, I, J (10 total)
- **Edges (undirected)** with weights: AB=5, AC=2, BC=1, BD=2, CD=3, CE=4, DE=1, DF=19, EF=17, FG=1, FH=3, FI=2, HI=1, HJ=2, IJ=5
- **Purpose:** Illustrate algorithm traversing two clusters (A–E and F–J) connected by high-weight edges, ensuring Dijkstra picks low-cost paths first and correctly handles "long bridge" edges.

#### Step-by-Step Execution Overview

| Step | Frontier (min-heap)                | Extracted | Updated Distances                                                               |
|------|------------------------------------|-----------|---------------------------------------------------------------------------------|
| 0    | {A:0}                              | —         | d[A]=0                                                                          |
| 1    | {B:5, C:2}                         | C (2)     | d[C]=2; relax C→B (2+1=3) ⇒ d[B]=3; C→E (2+4=6) ⇒ d[E]=6; C→D (2+3=5) ⇒ d[D]=5  |
| 2    | {B:3, D:5, E:6}                    | B (3)     | d[B]=3; relax B→D (3+2=5) ties d[D]=5; B→A, B→C already settled                 |
| 3    | {D:5, E:6, A:∞?*, ...}             | D (5)     | relax D→E (5+1=6) tie d[E]=6; D→F (5+19=24) ⇒ d[F]=24                           |
| 4    | {E:6, F:24, ...}                   | E (6)     | relax E→F (6+17=23) ⇒ d[F]=23                                                   |
| 5    | {F:23, ...}                        | F (23)    | relax F→G (23+1=24) ⇒ d[G]=24; F→H (23+3=26) ⇒ d[H]=26; F→I (23+2=25) ⇒ d[I]=25 |
| 6    | {G:24, H:26, I:25, J:∞?*, ...}     | G (24)    | G has no new relaxations                                                        |
| 7    | {I:25, H:26, J:∞?*, ...}           | I (25)    | relax I→J (25+5=30) ⇒ d[J]=30; I→H (25+1=26) tie; I→F skip                      |
| 8    | {H:26, J:30}                       | H (26)    | H→J (26+2=28) improves d[J]=28; H→F skip                                        |
| 9    | {J:28}                             | J (28)    | no further relaxations                                                          |

> *Vertices with no direct path from the source until necessary remain ∞ (omitted for clarity).

**Final distances** from **A**:
```
A:0, B:3, C:2, D:5, E:6, F:23, G:24, H:26, I:25, J:28
```

**Complexity for this run:**
- V=10, E=16 ⇒ heap ops ∼ O((V+E) log V) ≃ O(26·log10).


#### Go Playground Example (Annotated)
```go
package main

import (
	"fmt"
	"log"

	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/dijkstra"
)

func main() {
	// 1) Construct an undirected, weighted graph.
	g := core.NewGraph(core.WithWeighted())
	// 1.1) Add edges in the first cluster A–E.
	g.AddEdge("A", "B", 5)
	g.AddEdge("A", "C", 2)
	g.AddEdge("B", "C", 1)
	g.AddEdge("B", "D", 2)
	g.AddEdge("C", "D", 3)
	g.AddEdge("C", "E", 4)
	g.AddEdge("D", "E", 1)
	// 1.2) Connect clusters via high-weight edges D–F and E–F.
	g.AddEdge("D", "F", 19)
	g.AddEdge("E", "F", 17)
	// 1.3) Build second cluster F–J.
	g.AddEdge("F", "G", 1)
	g.AddEdge("F", "H", 3)
	g.AddEdge("F", "I", 2)
	g.AddEdge("G", "H", 4)
	g.AddEdge("H", "I", 1)
	g.AddEdge("H", "J", 2)
	g.AddEdge("I", "J", 5)

	// 2) Execute Dijkstra from source "A", requesting predecessors.
	dist, prev, err := dijkstra.Dijkstra(
		g,
		dijkstra.Source("A"),
		dijkstra.WithReturnPath(),
	)
	if err != nil {
		log.Fatalf("Dijkstra failed: %v", err)
	}

	// 3) Print final distances and a sample shortest path to J.
	fmt.Println("Final distances from A:")
	for _, v := range []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"} {
		fmt.Printf("  %s → %d\n", v, dist[v])
	}

	// 4) Reconstruct and print the path from A to J using the predecessor map.
	path := []string{"J"}
	for u := prev["J"]; u != ""; u = prev[u] {
		path = append([]string{u}, path...)
	}
	fmt.Printf("Shortest path A→J: %v\n", path)
}
```

**Explanation of key steps:**
1. **Graph construction** uses `core.NewGraph(core.WithWeighted())` to enable weighted edges.
2. **Edge additions** are grouped by cluster, highlighting the low-weight core and high-weight "bridge" design.
3. **Dijkstra invocation**: we pass `dijkstra.Source("A")` to set the source and `WithReturnPath()` to capture the predecessor chain.
4. **Distance printing** in sorted order shows the algorithm's prioritization of cheap routes before exploring costly bridges.
5. **Path reconstruction** iterates `prev[v]` until the empty string, yielding the route.

This advanced example showcases how Dijkstra seamlessly handles tightly connected subgraphs and defer exploration of high-cost edges until necessary.

[![Playground - Dijkstra](https://img.shields.io/badge/Go_Playground-Dijkstra_2-blue?logo=go)](https://go.dev/play/p/qCvQDTO9Mgm)

---

### 5.5. Best Practices & Pitfalls

* **Validate Input Early:** Ensure your graph is non-nil, weighted, and source vertex exists before running Dijkstra to avoid wasted work and panics.
* **Enforce Non-Negative Weights:** Dijkstra assumes \(w(u,v) \ge 0\). For graphs with negative edges use Bellman–Ford or Johnson's algorithm.
* **Skip Stale Queue Entries:** When popping from the min-heap, always compare the extracted distance with the current `dist[u]` and skip if they differ — this implements the lazy decrease-key safely.
* **Check for Overflow:** For graphs with very large weights, verify that `dist[u] + w` does not overflow the integer type before assignment, or use a saturated arithmetic strategy.
* **Tune Priority Queue Strategy:** Go’s `container/heap` is an implicit binary heap (O(log n) per op). For heavy workloads consider Fibonacci or pairing heaps for amortized O(1) decrease-key, or radix heaps for integer weights.
* **Limit Search Space:** Use `WithMaxDistance()` to bound exploration when you only need distances up to a threshold, reducing work on large graphs.
* **Leverage Mixed-Edge Support:** With `WithMixedEdges()`, Dijkstra can handle directed and undirected edges in the same graph — ensure you use the direction filter in relaxation to avoid reverse traversal.
* **Profile Real Data:** Benchmark on realistic graph models (e.g., road networks, social graphs) rather than grids to capture actual performance characteristics and hotspot edges.
> * Secret Tip — Early Exit for Single-Target: If you only need the shortest path to one destination, break the loop when that vertex is finalized, saving the cost of processing the rest of the graph.


---

Next: [6. Minimum Spanning Trees: Prim & Kruskal →](PRIM_%26_KRUSKAL.md)
