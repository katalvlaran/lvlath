# 10. Matrices: Adjacency & Incidence

Matrices provide a powerful algebraic view of graphs, enabling use of linear algebra techniques for connectivity, spectral analysis, and transformations. In this section we cover:

* **When & Why**: trade‑offs vs adjacency lists
* **Adjacency Matrix**: definition, examples, code
* **Incidence Matrix**: definition, examples, code
* **Go Example**: building both matrices from a `core.Graph`
* **Best Practices & Pitfalls**

---

## 10.1 When & Why Use Matrices

* **Adjacency List vs Matrix**:

    * List: store neighbors per vertex; space O(V+E), fast on sparse graphs
    * Matrix: store all pair weights; space O(V²), constant‑time edge checks
* **Use Matrices When**:

    * Graph is dense (E≈V²)
    * You need O(1) edge existence or weight queries
    * You perform algebraic operations (e.g., spectral clustering, power methods)

---

## 10.2 Adjacency Matrix

An **adjacency matrix** \$A\$ of a graph with \$V\$ vertices is a \$V\times V\$ matrix where:

![\large A\_{ij}=\begin{cases}w\_{ij}&\text{if }(i,j)\in E\0&\text{otherwise}\end{cases}](https://latex.codecogs.com/svg.image?%5Clarge%20A_%7Bij%7D%3D%5Cbegin%7Bcases%7Dw_%7Bij%7D%26%5Ctext%7Bif%20%7D%28i%2Cj%29%5Cin%20E%5C%5C0%26%5Ctext%7Botherwise%7D%5Cend%7Bcases%7D)

### 10.2.1 Example Graph

```text
   A-----B
   | \   |
   |  \  |
   |   \ |
   D-----C
```

Edges (undirected, unweighted): AB, AC, AD, BC, CD

### 10.2.2 Matrix Representation

|       |  A  |  B  |  C  |  D  |
| ----- | :-: | :-: | :-: | :-: |
| **A** |  0  |  1  |  1  |  1  |
| **B** |  1  |  0  |  1  |  0  |
| **C** |  1  |  1  |  0  |  1  |
| **D** |  1  |  0  |  1  |  0  |

Diagonal entries are zero (no self‑loops); symmetric for undirected graphs.

---

## 10.3 Incidence Matrix

An **incidence matrix** `M` of an undirected graph with `V` vertices and `E` edges is a `V*E` matrix where each column corresponds to an edge. For edge ![\Large e_k=(u,v)](https://latex.codecogs.com/svg.image?\large&space;e_k=(u,v))$:

![\Large M_{i,k}=\begin{cases}1&\text{if}(i=u)\text{or}(i=v),\\0&\text{otherwise.}\end{cases}](https://latex.codecogs.com/svg.image?%5Clarge%26space%3BM_%7Bi%2Ck%7D%3D%5Cbegin%7Bcases%7D1%26%5Ctext%7Bif%7D%28i%3Du%29%5Ctext%7Bor%7D%28i%3Dv%29%2C%5C%5C0%26%5Ctext%7Botherwise.%7D%5Cend%7Bcases%7D)

### 10.3.1 Example Graph Continued

Number edges as:

```
 e₁: A–B   e₂: A–C   e₃: A–D   e₄: B–C   e₅: C–D
```

|       |  e₁ |  e₂ |  e₃ |  e₄ |  e₅ |
| ----- | :-: | :-: | :-: | :-: | :-: |
| **A** |  1  |  1  |  1  |  0  |  0  |
| **B** |  1  |  0  |  0  |  1  |  0  |
| **C** |  0  |  1  |  0  |  1  |  1  |
| **D** |  0  |  0  |  1  |  0  |  1  |

Each column has exactly two ones for an undirected edge.

---

## 10.4 Go Example: Building Matrices from a `core.Graph`

Below is a self‑contained example showing how to compute both matrices for any `core.Graph`. No external dependencies beyond `lvlath`.

```go
package main

import (
    "fmt"
    "github.com/katalvlaran/lvlath/core"
)

func main() {
    // 1. Construct a simple undirected graph
    g := core.NewGraph(false, false)
    for _, e := range []struct{ u, v string }{
        {"A", "B"}, {"A", "C"}, {"A", "D"},
        {"B", "C"}, {"C", "D"},
    } {
        g.AddEdge(e.u, e.v, 1)
    }

    // 2. Build an index mapping from vertex ID → row
    idx := make(map[string]int)
    for i, v := range g.Vertices() {
        idx[v.ID] = i
    }
    n := len(idx)

    // 3. Initialize matrices
    adj := make([][]int, n)
    inc := make([][]int, n)
    for i := range adj {
        adj[i] = make([]int, n)
        inc[i]  = make([]int, len(g.Edges()))
    }

    // 4. Fill adjacency and incidence
    for k, e := range g.Edges() {
        u, v := idx[e.From.ID], idx[e.To.ID]
        // Adjacency (undirected)
        adj[u][v] = int(e.Weight)
        adj[v][u] = int(e.Weight)
        // Incidence
        inc[u][k] = 1
        inc[v][k] = 1
    }

    // 5. Display results
    fmt.Println("Adjacency Matrix:")
    for _, row := range adj {
        fmt.Println(row)
    }

    fmt.Println("\nIncidence Matrix:")
    for _, row := range inc {
        fmt.Println(row)
    }
}
```

[![Go Playground](https://img.shields.io/badge/Go_Playground-Matrices-blue?logo=go)](https://go.dev/play/p/VUPSsth2WQw)

---

### 10.5 Complexity & Best Practices

| Representation   | Space Complexity | Edge Query | Iterating Neighbors |
| ---------------- | ---------------- | ---------- | ------------------- |
| Adjacency Matrix | O(V²)            | O(1)       | O(V)                |
| Incidence Matrix | O(V·E)           | O(E)       | O(E)                |

* **Dense graphs** (E≈V²): use **Adjacency Matrix** for simplicity and speed.
* **Sparse graphs**: prefer **Adjacency List**; use matrices only when algebraic operations are critical.
* **Lazy construction**: defer matrix builds until needed in algorithms like spectral clustering.

> **Pro Tip:** Combine matrix forms with list-based structures. E.g., use lists for traversal, matrices for post‑processing spectral methods.

---

Next: [11. Traveling Salesman (TSP) →](TRAVELING_SALESMAN.md)
