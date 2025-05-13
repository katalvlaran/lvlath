## **11. Traveling Salesman (TSP)**

**Slogan:** “From exponential exactness to practical approximations.”

---

### 11.1 Problem Statement

The Traveling Salesman Problem (TSP) asks: given a complete weighted graph `G=(V,E)`, find the shortest possible tour that visits each vertex exactly once and returns to the start.

* **Input:** `n` cities, pairwise distances `d(i,j)≥0`.
* **Output:** a permutation ![\Large \pi](https://latex.codecogs.com/svg.image?\large&space;\pi) of ![\Large \{1..n\}](https://latex.codecogs.com/svg.image?\large&space;\{1..n\}) minimizing ![\Large \sum_{k=1}^{n} d(\pi_k,\pi_{k+1})](https://latex.codecogs.com/svg.image?\large&space;\sum_{k=1}^{n}d(\pi_k,\pi_{k&plus;1})), with ![\Large \pi_{n+1}=\pi_1](https://latex.codecogs.com/svg.image?\large&space;\pi_{n&plus;1}=\pi_1).


> **Complexity:** NP-hard. Exact solutions are ![\Large O(n^2 2^n)](https://latex.codecogs.com/svg.image?\large&space;O(n^2&space;2^n)); approximation algorithms trade optimality for speed.

---

### 11.2 Exact Solution: Held–Karp Dynamic Programming

**Time & Space:** ![\Large O(n^2 2^n)](https://latex.codecogs.com/svg.image?\large&space;O(n^2&space;2^n)) time, ![\Large O(n2^n)](https://latex.codecogs.com/svg.image?\large&space;O(n2^n)) memory.


**Pseudocode (bitmask DP):**

```text
let DP[mask][i] = minimal cost to reach subset mask ⊆ {1..n} ending at city i
initialize DP[1<<0][0] = 0
for mask from 1 to (1<<n)-1:
  for i in V where mask contains i:
    for j in V where j ∉ mask:
      newMask = mask | (1<<j)
      DP[newMask][j] = min(DP[newMask][j], DP[mask][i] + d(i,j))
answer = min_{i>0}(DP[(1<<n)-1][i] + d(i,0))
```

**Key recurrence:**
![\Large DP\[mask\]\[j\]=\min\_{i\in mask\setminus{j}}\bigl(DP\[mask\setminus{j}\]\[i\]+d(i,j)\bigr)](https://latex.codecogs.com/svg.image?%5Clarge%20DP%5Bmask%5D%5Bj%5D%3D%5Cmin_%7Bi%5Cin%20mask%5Csetminus%5C%7Bj%5C%7D%7D%5Cbigl%28DP%5Bmask%5Csetminus%5C%7Bj%5C%7D%5D%5Bi%5D%2Bd%28i%2Cj%29%5Cbigr%29)

---

### 11.3 1.5‑Approximation: Christofides’ Algorithm

**Guarantee:** Tour cost ≤ 1.5·OPT in metric graphs (triangle inequality).

**Steps:**

1. **Minimum Spanning Tree (MST):** compute `T` via Prim/Kruskal.
2. **Odd-degree Matching:** find minimum-weight perfect matching `M` on odd-degree vertices of `T`.
3. **Eulerian Multigraph:** combine ![\Large T\cup M](https://latex.codecogs.com/svg.image?\large&space;T\cup&space;M), find Eulerian circuit.
4. **Shortcutting:** traverse circuit, skipping visited vertices to form Hamiltonian tour.

**Pseudocode Sketch:**

```text
T = MST(G)
O = vertices of T with odd degree
M = minimum perfect matching on O
double_edges = edges(T) ∪ edges(M)
eulerian = findEulerianTour(double_edges)
tour = shortcut(eulerian)
return tour
```

---

### 11.4 Example: 4‑City Cycle

Consider cities A, B, C, D with distances:

```
    A
   / \
  5   6
 /     \
B — 2 — C
 \     /
  7   4
   \ /
    D
```

* **Exact (Held–Karp):** computes optimal cost = 2+6+5+7 = **20**.
* **Christofides:**

    * MST edges: (B–C:2), (A–B:5), (C–D:4)
    * Odd vertices: {A, D}, match A–D (cost 7).
    * Eulerian tour shortcuts to same optimal **20** (on small graphs usually exact).

```go
package main

import (
  "fmt"

  "github.com/katalvlaran/lvlath/tsp"
)

func main() {
  dist := [][]float64{
    {0, 5, 6, 0}, // A
    {5, 0, 2, 7}, // B
    {6, 2, 0, 4}, // C
    {0, 7, 4, 0}, // D
  }
  cost, _ := tsp.TSPApprox(dist)
  fmt.Printf("Optimal cost: %f, Tour: %v\n", cost.Cost, cost.Tour)
}

```
[![Run on Go Playground](https://img.shields.io/badge/Go%20Playground-TSPApprox-blue?logo=go)](https://go.dev/play/p/b6hZKMMJiL4)

---

### 11.5 Best Practices & Pitfalls

* **Use Held–Karp** only for n ≤ 20; memory explodes otherwise.
* **Precompute** full distance matrix for O(1) lookups.
* **Christofides requires** graph to satisfy the triangle inequality. Otherwise, approximation guarantee breaks.
* **Edge cases:** identical cities, zero-weight loops—filter or handle gracefully.
* **Parallelize** independent DP subsets or matching for performance boost.

---