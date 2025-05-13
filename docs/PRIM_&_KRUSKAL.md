# 6. Minimum Spanning Trees: Prim & Kruskal

In this section, we explore **Minimum Spanning Trees (MST)**, fundamental structures in weighted undirected graphs. An MST connects all vertices with the minimum total edge weight, ensuring no cycles. MSTs have wide applications: network design, clustering, approximate TSP heuristics, and more.

---

## 6.1 Why MST? When & How

* **Why**: Minimizes cost to interconnect nodes (e.g., laying cables, road networks).
* **When**: Use on **connected**, **undirected**, **weighted** graphs.
* **How**: Greedy strategies that add cheapest eligible edges without forming cycles.

**Greedy patterns**:

* **Global** (Kruskal): Sort all edges, pick smallest if it joins two different components.
* **Local** (Prim): Grow tree from a start vertex, repeatedly add the smallest edge crossing the cut.

---

## 6.2 Core Algorithms

#### 6.2.0 Illustrative Example Graph

```text        
  A--2--B---3---C
  | \   |       |
  4  1  5       1
  |    \|       |
  D--2--E---4---F
```

### 6.2.1 Kruskal’s Algorithm: Detailed Walkthrough

1. **Sort all edges by weight** (ascending):

   ```
   1: AE, CF  2: AB, DE  3: BC  4: AD, EF  5: BE
   ```

2. **Initialize DSU**: each vertex in its own set

   ```
   {A}, {B}, {C}, {D}, {E}, {F}
   ```

3. **Scan edges**, union if endpoints are in different sets:

   | Step | Edge | Weight | DSU before       | Action        | DSU after                     | MST so far                   |
         |:----:|:----:|:------:|:-----------------|:--------------|:------------------------------|:-----------------------------|
   |  1   |  AE  |   1    | {A}, {E}, …      | union(A, E)   | {A,E}, {B}, {C}, {D}, {F}     | \[AE]                        |
   |  2   |  CF  |   1    | …,{C}, {F}       | union(C,   B) | {A,B,E}, {C,F}, {D}           | \[AE, CF, AB]                |
   |  4   |  DE  |   2    | {A,B,E}, {D}, …  | union(D, E)   | {A,B,D,E}, {C,F}              | \[AE, CF, AB, DE]            |
   |  5   |  BC  |   3    | {A,B,D,E}, {C,F} | union(B, C)   | {A,B,C,D,E,F} (all connected) | \[AE, CF, AB, DE, BC] — STOP |

4. **Result**:

* **MST edges**: AE, CF, AB, DE, BC
* **Total weight**: \$1+1+2+2+3=9\$

---

### 6.2.2 Prim’s Algorithm (start at A)

1. **Initialize**

   ```
   key[A]=0,    π[A]=nil   
   key[B,C,D,E,F]=∞,   π[…]=nil   
   PQ = {A(0), B(∞), C(∞), D(∞), E(∞), F(∞)}   
   MST = ∅
   ```

2. **Extract-min & relax**

* **Step 1**: extract A(0) → add A

  ```
  for each neighbor v of A:
    if w(A,v) < key[v]:
      key[v]=w(A,v), π[v]=A
  ```

  Updates:

  ```
  B: key=2, π=B→A   D: key=4, π=D→A   E: key=1, π=E→A   C,F unchanged
  ```

  PQ now {E(1), B(2), D(4), C(∞), F(∞)}

* **Step 2**: extract E(1) → add E

  ```
  neighbors: A(visited), B(5), D(2), F(4)
  ```

  Updates:

  ```
  D: 2 < 4 ⇒ key[D]=2, π[D]=E   F: 4 < ∞ ⇒ key[F]=4, π[F]=E
  ```

  PQ {B(2), D(2), C(∞), F(4)}

* **Step 3**: extract B(2) → add B
  neighbors (A,E visited; C via BC=3):

  ```
  C: 3 < ∞ ⇒ key[C]=3, π[C]=B
  ```

  PQ {D(2), C(3), F(4)}

* **Step 4**: extract D(2) → add D
  neighbors (A,E visited) ⇒ no change
  PQ {C(3), F(4)}

* **Step 5**: extract C(3) → add C
  neighbor F (via CF=1):

  ```
  1 < 4 ⇒ key[F]=1, π[F]=C
  ```

  PQ {F(1)}

* **Step 6**: extract F(1) → add F
  all neighbors already in MST

3. **Collect MST edges via π**:

   ```
   A–E (1), A–B (2), E–D (2), B–C (3), C–F (1)
   ```

   **Total weight**: \$1+2+2+3+1=9\$

---

#### 6.2.3 Complexity & When best to Use

| Algorithm | Strategy            | Time Complexity                  | Space    |
|-----------|---------------------|----------------------------------|----------|
| Kruskal   | Sort + Disjoint Set | O(E log E + α(V)·E) ≈ O(E log V) | O(E + V) |
| Prim      | Greedy frontier     | O(E log V)                       | O(E)     |
* **Kruskal**

    * Best when `E` is not much larger than `V`; easy to parallelize edge sorting
* **Prim** (with a binary heap)

    * Preferred when the graph is dense `E ≈ V²` or when you have a fast priority queue


*Implementation reference in `lvlath/algorithms/prim_kruskal.go`*

---

## 6.3 Go Example

```go
package main

import (
    "fmt"

    "github.com/katalvlaran/lvlath/algorithms"
    "github.com/katalvlaran/lvlath/core"
)

func main() {
    g := core.NewGraph(false, true)
    for _, e := range []struct {
        u, v string
        w    int64
    }{
        {"A", "B", 2}, {"B", "C", 3},
        {"A", "D", 4}, {"A", "E", 1},
        {"B", "E", 5}, {"C", "F", 1},
        {"D", "E", 2}, {"E", "F", 4},
    } {
        g.AddEdge(e.u, e.v, e.w)
    }

    kr, wK, _ := algorithms.Kruskal(g)
    fmt.Printf("Kruskal MST (%d):\n", wK)
    for _, e := range kr {
        fmt.Printf("  %s—%s : %d\n", e.From.ID, e.To.ID, e.Weight)
    }
    // Kruskal MST (9):
    //  C—F : 1
    //  E—A : 1
    //  B—A : 2
    //  D—E : 2
    //  B—C : 3

    pr, wP, _ := algorithms.Prim(g, "A")
    fmt.Printf("\nPrim MST (%d):\n", wP)
    for _, e := range pr {
        fmt.Printf("  %s—%s : %d\n", e.From.ID, e.To.ID, e.Weight)
    }
    // Prim MST (9):
    //  A—E : 1
    //  A—B : 2
    //  E—D : 2
    //  B—C : 3
    //  C—F : 1
}
```

[![Go Playground](https://img.shields.io/badge/Go_Playground-PrimKruskal-blue?logo=go)](https://go.dev/play/p/2DxXQoLAa_J)

---

## 6.4 Pitfalls & Best Practices

* **Connectivity**: Ensure graph is fully connected; `MST` covers only reachable vertices.
* **Disjoint Set Efficiency**: Path compression and union by rank are critical for large `E` in *Kruskal*.
* **Heap Choice**: *Prim* with a binary heap yields `O(E log V)`; use more advanced heaps (e.g., *Fibonacci*) for very dense graphs.
* **Equal Weights**: Decide tie‑breaking strategy for deterministic outputs.
* **Error Handling**: Both functions return an error if graph is directed or unweighted.

---


Next: [7. Max-Flow: Ford–Fulkerson / Edmonds-Karp / Dinic →](MAX_FLOW.md)
