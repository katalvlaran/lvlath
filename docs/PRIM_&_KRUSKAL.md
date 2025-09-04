# 6. Minimum Spanning Trees: Prim & Kruskal

---

## 6.1. Why MST? When & How

### 6.1.1. What Is an MST?
A **Minimum Spanning Tree (MST)** of a connected, undirected, weighted graph **G = (V, E, w)** is a subset of edges

$$\[T \subseteq E\]$$

such that:

1. **Spanning**: The subgraph $$(V, T)$$ is connected (covers all vertices).
2. **Tree**: It contains no cycles, and thus has exactly $$(|V| - 1)$$ edges.
3. **Minimum Weight**: It minimizes the total weight

   $$\[\sum_{e \in T} w(e) \quad\text{is minimal}.\]$$

### 6.1.2. Why MST?
- **Network Design**: Lay cables, roads, or pipelines with minimal cost.
- **Clustering**: Build hierarchical clusters by removing heaviest edges from the MST.
- **Approximation Algorithms**: Subroutine in Steiner Tree, k‑center, and metric TSP heuristics.
- **Theoretical Insights**: MST algorithms illustrate key graph properties (cut property, cycle property).

### 6.1.3. When & How
- **When**:  
  • Graph must be **connected**, **undirected**, and **weighted**.
- **How**: Greedy strategies rely on two fundamental properties:  
  • **Cut Property**: The minimum-weight edge crossing any cut belongs to *some* MST.  
  • **Cycle Property**: The maximum-weight edge in any cycle does *not* belong to *any* MST.

Two canonical greedy patterns:

| Strategy   | Sketch                                                                                           |
|------------|--------------------------------------------------------------------------------------------------|
| **Global** | Kruskal: sort all edges, add smallest that connects two components (Union‑Find).                 |
| **Local**  | Prim: start at a root, grow a single tree by adding the cheapest edge across the cut (min‑heap). |

---

## 6.2. Mathematical Formulation

Let $$\(G = (V, E)\)$$ be a graph with weight function $$\(w: E \to \mathbb{R}_{+}\)$$ . A spanning tree `T` satisfies:

1. $$\(|T| = |V| - 1\)$$ .
2. $$(V, T)$$ is connected.

The MST problem is:

$$\[ \min_{T \subseteq E} \; \sum_{e \in T} w(e) \quad \text{s.t.} \quad T \text{ connects all vertices and } |T| = |V| - 1. \]$$

### Key Properties

- **Cut Property**: For any partition $$(S, V\setminus S)$$ of vertices, the minimum-weight edge with one endpoint in `S` and the other in $$\(V\setminus S\)$$ belongs to some MST.
- **Cycle Property**: For any cycle in `G`, the heaviest edge on that cycle does not belong to any MST.

These properties justify the correctness of both Kruskal’s and Prim’s algorithms.

---

## 6.3. Core Algorithms

### 6.3.0 Illustrative Example Graph

To build intuition for both Prim’s and Kruskal’s algorithms, consider the following **undirected**, **weighted** graph on eight vertices:

```
            [A]
           / | \
         3/  |  \4
         /   |8  \
      [B]──5─┼────[C]
     / | \   |   / | \
   4/  | 5\  | 4/  |  \6
   /   |   \ | /   |   \
[D]──3─┼────[E]──5─┼────[F]
   \   |9  /   \   |8  /
   6\  |  /4   6\  |  /5
     \ | /       \ | /
      [G]────8────[H]
```
- **Vertices**:  A, B, C, D, E, F, G, H
- **Edges (u–v : weight)**:
  | Edge  | Weight |  
  |:-----:|:------:|
  |  A–B  |   3    |
  |  D–E  |   3    |
  |  A–C  |   4    |
  |  B–D  |   4    |
  |  C–E  |   4    |
  |  E–G  |   4    |
  |  B–C  |   5    |
  |  B–E  |   5    |
  |  E–F  |   5    |
  |  F–H  |   5    |
  |  C–F  |   6    |
  |  D–G  |   6    |
  |  E–H  |   6    |
  |  A–E  |   8    |
  |  C–H  |   8    |
  |  B–G  |   9    |

This graph exhibits a mix of **sparse** and **dense** regions, several **alternative paths**, and potential **cycles**-ideal for showcasing:

- **Prim’s behavior** when growing a tree from a chosen root (e.g. A), repeatedly pulling the cheapest frontier edge.
- **Kruskal’s global selection**, sorting all edges and merging components via union-find.

Both algorithms must handle:

- **Multiple minimal edges** of equal weight (e.g. G–H vs. D–G).
- **Branches** with varying densities (B–E–G–H vs. C–F–H).
- **Connectivity** across the entire vertex set.

In the subsequent sections, we will trace through this graph step by step, observing how each algorithm constructs the MST of **total weight = 28**.

---

### 6.3.1 Kruskal’s Algorithm

**Core Idea**

Kruskal’s method builds an MST by **globally** considering edges in ascending order of weight and merging components only when they lie in different sets. It relies on a Disjoint‑Set (Union-Find) structure to track connected components efficiently, ensuring no cycles are formed.

**Key Features**

- **Global Greedy Selection**: Sorts all `(E)` edges once $$(O(E log E))$$.
- **Cycle Avoidance**: Uses Union-Find with **path compression** and **union by rank** to test/component merge in $$O(\alpha(V))$$ per operation.
- **Deterministic**: Stable sort on edges (by ID tie‑breaker) → reproducible MST even when weights tie.

**When to Use**

- Graphs where `(E)` is not excessively larger than `(V)`.
- Scenarios where you can preprocess/sort edges in parallel.

**Time & Space Complexity**

- **Time**: $$(O(E \times \log E + E \cdot \alpha(V)) \approx O(E \log V))$$.
- **Space**: $$O(V + E)$$.

---

#### Pseudocode
```text
function KruskalMST(graph):
    edges = graph.Edges()                          // list of (u,v,w)
    sort(edges by w ascending, stable)             // O(E log E)

    // Initialize disjoint sets: each vertex in its own set
    for each vertex v in graph.Vertices():
        MakeSet(v)

    MST = empty list
    totalWeight = 0

    // Process edges in ascending order
    for each (u, v, w) in edges:
        if Find(u) ≠ Find(v):                      // different components?
            Union(u, v)                            // merge sets
            MST.append((u,v))
            totalWeight += w
            if |MST| == |V| - 1:
                break                             // early exit
    end for

    if |MST| < |V|-1:
        error "disconnected graph"
    return MST, totalWeight
```

---

#### Step-by-Step on the 8‑Vertex Graph

Recall the example graph from **6.3.0** with vertices `{A…H}` and edge weights:
1. **Sort edges by weight**.
2. Initialize DSU: `{A},{B},…,{H}`.
3. Iterate:

| Step | Edge | Weight | Components before    | Action     | Components after       | MST edges              |
|:----:|:----:|:------:|:---------------------|:-----------|:-----------------------|:-----------------------|
| 1    | A–B  | 3      | {A},{B},…            | Union(A,B) | {A,B},…                | [A–B]                  |
| 2    | D–E  | 3      | {D},{E},…            | Union(D,E) | {D,E},…                | [A–B, D–E]             |
| 3    | A–C  | 4      | {A,B},{C},…          | Union(A,C) | {A,B,C},…              | [A–B, D–E, A–C]        |
| 4    | B–D  | 4      | {A,B,C},{D,E},…      | Union(B,D) | {A,B,C,D,E},…          | […, B–D]               |
| 5    | C–E  | 4      | same set             | Skip       | —                      | —                      |
| 6    | E–G  | 4      | {…E},{G},…           | Union(E,G) | {A,B,C,D,E,G},…        | […, E–G]               |
| 7    | E–F  | 5      | {…E,G},{F},…         | Union(E,F) | {A,B,C,D,E,F,G},…      | […, E–F]               |
| 8    | F–H  | 5      | {…F},{H}             | Union(F,H) | {A,B,C,D,E,F,G,H}      | […, F–H]               |

- **MST complete** at step 8 (7 edges for 8 vertices).
- **Total weight** = 3+3+4+4+4+5+5 = **28**.

---

#### Highlights & Insights

- **Early Exit**: Stops once $$(|V|-1)$$ edges are chosen.
- **Tie‑Breaking**: Stable sort + edge ID order → predictable when weights tie.
- **Union-Find Efficiency**: Path compression + union by rank keeps each `Find` ≈ amortized $$O(1)$$.

---

#### Go Playground Example

```go
package main

import (
	"fmt"

	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/prim_kruskal"
)

func main() {
	// Construct the graph from section 6.3.0
	g := core.NewGraph(core.WithWeighted())
	for _, e := range []struct {
		u, v string
		w    int64
	}{
		{"A", "B", 3}, {"D", "E", 3}, {"A", "C", 4}, {"B", "D", 4},
		{"C", "E", 4}, {"E", "G", 4}, {"B", "C", 5}, {"B", "E", 5},
		{"E", "F", 5}, {"F", "H", 5}, {"C", "F", 6}, {"D", "G", 6},
		{"E", "H", 6}, {"A", "E", 8}, {"C", "H", 8}, {"B", "G", 9},
	} {
		g.AddEdge(e.u, e.v, e.w)
	}

	// 2. Compute Kruskal
	mst, total, err := prim_kruskal.Kruskal(g)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	
	// 3. Display result
	fmt.Printf("Total weight = %d\nEdges: ", total)
	for i, e := range mst {
		if i > 0 {
			fmt.Print(" ")
		}
		fmt.Printf("%s-%s", e.From, e.To)
	}
	// Output:
	// Total weight = 28
	// Edges: A-B D-E A-C B-D E-G F-H E-F
}
```

[![Go Playground](https://img.shields.io/badge/Go_Playground-PrimKruskal-blue?logo=go)](https://go.dev/play/p/gWp4J0Gb2U4)

---

## 6.4. Pitfalls & Best Practices

### 6.4.1. Common Pitfalls

- **Disconnected Graphs**: Calling MST on disconnected graphs yields no spanning tree; algorithms must return an explicit error (e.g., `ErrDisconnected`).
- **Directed or Unweighted Graphs**: MST requires undirected, weighted graphs. Always validate:
```go
if graph == nil || graph.Directed() || !graph.Weighted() || graph.HasDirectedEdges() {
	return ErrInvalidGraph
}
 ```
- **Self‑Loops & Multi‑Edges**:  
  • Self‑loops never contribute to MST; skip them.  
  • Parallel edges: choose the lightest; multi‑edge support must be explicit (`WithMultiEdges`).
- **Non‑Determinism**: If edges have equal weights, use *stable* sorting or fixed insertion order to achieve repeatable MSTs.

### 6.4.2. Performance Considerations

- **Union‑Find Tuning** (_Kruskal_): Always implement **path compression** and **union by rank** to achieve near-linear time \(O(\alpha(V)E)\).
- **Heap Efficiency** (_Prim_): Use a binary heap for $$O(E \log V)$$. For dense graphs, consider Fibonacci heaps or pairing heaps to reduce amortized decrease-key cost.
- **Memory Allocation**: Pre‑allocate slices and maps based on $$|V|$$ and $$|E|$$ to avoid unnecessary GC pressure in large graphs.

### 6.4.3. Best Practices

1. **Validate Early**: Check graph properties (*nil*, *directed*, *weighted*, *mixed edges*) before heavy computation.
2. **Pre‑Sort Vertices/Edges**: Rely on `core.Graph` methods that return sorted lists for determinism.
3. **Edge Filtering**: Remove self‑loops and skip edges connecting already‑united components immediately.
4. **Error Semantics**: Use sentinel errors (`ErrInvalidGraph`, `ErrEmptyRoot`, `ErrDisconnected`) to clearly communicate failure modes.
5. **Testing**: Cover boundary cases $$(\(|V|=0\)$$, $$\(|V|=1\)$$, disconnected components, multi‑edges, mixed‑mode).
6. **Documentation**: Embed complexity, proof sketches (cut/cycle properties), and usage examples in GoDoc for end‑users.

---

> Next: [7. Max-Flow: Ford-Fulkerson / Edmonds-Karp / Dinic →](MAX_FLOW.md)
