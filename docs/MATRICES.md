# 10. Matrices: Adjacency & Incidence

Matrices provide a powerful algebraic and algorithmic perspective on graphs, enabling everything from constant‑time edge queries to spectral clustering and connectivity analysis. This guide covers:

- **When & Why**: trade‑offs between adjacency lists and matrices
- **Adjacency Matrix**: definition, complexity, examples
- **Incidence Matrix**: definition, complexity, examples
- **Go Usage**: building and converting matrices with `lvlath/matrix`
- **Best Practices & Pitfalls**

---

## 10.1 When & Why Use Matrices

**Adjacency List vs. Matrix**

| Representation       | Space Complexity | Edge Query | Iterating Neighbors |
|----------------------| ---------------- | ---------- | ------------------- |
| **Adjacency List**   | O(V+E)           | O(deg(v))  | O(deg(v))           |
| **Adjacency Matrix** | O(V²)            | O(1)       | O(V)                |
| **Incidence Matrix** | O(V·E)           | O(E)       | O(E)                |

Use **Adjacency Matrix** when dealing with dense graphs (E≈V²) or requiring constant‑time edge lookups and algebraic operations (e.g., spectral clustering). Use **Incidence Matrix** for algebraic transformations (cycle space, cuts) and Eulerian path detection.

Use **matrices** when:

1. **Dense graphs** (E ∼ V²), where O(V²) storage is acceptable.
2. You require **constant‑time** edge existence or weight lookup.
3. You perform **algebraic** or **spectral** operations (e.g., eigenvalues, clustering).

---

## 10.2 Adjacency Matrix

An **adjacency matrix** `A` of a graph with `V` vertices is a `V*V` matrix where:

![\large A\_{ij}=\begin{cases}w\_{ij}&\text{if }(i,j)\in E\0&\text{otherwise}\end{cases}](https://latex.codecogs.com/svg.image?%5Clarge%20A_%7Bij%7D%3D%5Cbegin%7Bcases%7Dw_%7Bij%7D%26%5Ctext%7Bif%20%7D%28i%2Cj%29%5Cin%20E%5C%5C0%26%5Ctext%7Botherwise%7D%5Cend%7Bcases%7D)



- **Directed vs. Undirected:** For undirected graphs, A is symmetric.
- **Weighted vs. Unweighted:** When unweighted, ![\large w_{ij}=1](https://latex.codecogs.com/svg.image?w_{ij}=1)  for existing edges.

### 10.2.1 Social Network Scenario + Example Graph

```text
  [User Network Example]

           Fiona
       ↙︎     ↓     ↖         
      ↓   ↗ Bob ↘︎   ↑
    Alice    ↓    Carol
      ↓     Eva     ↓  
       ↘︎     ↑    ↙︎
           Dave   
```

Consider a small social platform where users interact by sending messages:

```
Users: Alice, Bob, Carol, Dave, Eva, Fiona
Edges:
  Alice→Bob (5 messages)
  Bob→Carol (2)
  Carol→Dave (4)
  Alice→Dave (3)
  Bob→Eva (1)
  Carol→Fiona (6)
  Fiona→Bob (2)
  Dave→Eva (3)
  Fiona→Alice (4)
```

This directed, weighted graph captures messaging frequency among six users.

### 10.2.2 Matrix Representation (6×6)

|           | Alice  |  Bob  | Carol | Dave  |  Eva  | Fiona  |
|-----------|:------:|:-----:|:-----:|:-----:|:-----:|:------:|
| **Alice** |   0    |   5   |   0   |   3   |   0   |   0    |
| **Bob**   |   0    |   0   |   2   |   0   |   1   |   0    |
| **Carol** |   0    |   0   |   0   |   4   |   0   |   6    |
| **Dave**  |   0    |   0   |   0   |   0   |   3   |   0    |
| **Eva**   |   0    |   0   |   0   |   0   |   0   |   0    |
| **Fiona** |   4    |   2   |   0   |   0   |   0   |   0    |

**Complexities:**
- Build: \(O(V+E)\) time, \(O(V^2)\) memory.
- Edge lookup: \(O(1)\).
- Iterate neighbors: \(O(V)\).

---

## 10.3 Incidence Matrix

An **incidence matrix** `M` of a graph with `V` vertices and `E` edges is a `V*E` matrix where each column corresponds to an edge ![e_k=(u,v)](https://latex.codecogs.com/svg.image?e_k=(u,v))

![\Large M_{i,k}=\begin{cases}1&\text{if}(i=u)\text{or}(i=v),\\0&\text{otherwise.}\end{cases}](https://latex.codecogs.com/svg.image?%5Clarge%26space%3BM_%7Bi%2Ck%7D%3D%5Cbegin%7Bcases%7D1%26%5Ctext%7Bif%7D%28i%3Du%29%5Ctext%7Bor%7D%28i%3Dv%29%2C%5C%5C0%26%5Ctext%7Botherwise.%7D%5Cend%7Bcases%7D)

$$
M_{i,k}=\begin{cases}1&\text{if}(i=u)\text{or}(i=v),\\0&\text{otherwise.}\end{cases}
$$

- **Directed**: ![! Large M_{u,k}=-1, M_{v,k}=+1](https://latex.codecogs.com/svg.image?$M_{u,k}=-1$,$M_{v,k}=&plus;1$)
- **Undirected**: ![! Large M_{u,k}=M_{v,k}=1](https://latex.codecogs.com/svg.image?$M_{u,k}=M_{v,k}=1$)
- Other entries are 0.

**Complexity:**
- Build: O(V + E) time, O(V·E) memory
- Lookup endpoints: O(1)
- Algebraic ops: often O(V·E)

### 10.3.1 Social Network Incidence (6×9)

Label edges 1–9 in the scenario above. The \(M\) matrix has 6 rows (users) and 9 columns (messages links). For example:

|         | e₁ | e₂ | e₃ | e₄ | e₅ | e₆ | e₇ | e₈ | e₉ |
|---------|:--:|:--:|:--:|:--:|:--:|:--:|:--:|:--:|:--:|
| Alice   | -1 |  0 |  0 | -1 |  0 |  0 | +1 |  0 |  0 |
| Bob     | +1 | -1 |  0 |  0 | +1 |  0 | -1 |  0 |  0 |
| Carol   |  0 | +1 | -1 |  0 |  0 | +1 |  0 |  0 |  0 |
| Dave    |  0 |  0 | +1 |  0 |  0 |  0 |  0 | +1 |  0 |
| Eva     |  0 |  0 |  0 |  0 | -1 |  0 |  0 | -1 |  0 |
| Fiona   |  0 |  0 |  0 |  0 |  0 | -1 |  0 |  0 | +1 |

Columns:
1. Alice→Bob, 2. Bob→Carol, 3. Carol→Dave, 4. Alice→Dave,
5. Bob→Eva, 6. Carol→Fiona, 7. Fiona→Bob, 8. Dave→Eva, 9. Fiona→Alice.

---

## 10.4 Go Usage Example

Below is a condensed Go example using `lvlath/core` and `lvlath/matrix` to build both matrices and perform a simple lookup and spectral analysis.

```go
package main

import (
	"fmt"

	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/matrix"
)

func main() {
	// (1) Define a small “social graph” of directed, weighted interactions.
	//    Each tuple is (sender, receiver, messageCount).
	interactions := []struct {
		from, to string
		weight   int64
	}{
		{"Alice", "Bob", 5}, {"Alice", "Dave", 3},
		{"Bob", "Eva", 1}, {"Bob", "Carol", 2},
		{"Carol", "Dave", 4}, {"Carol", "Fiona", 6},
		{"Fiona", "Alice", 4}, {"Fiona", "Bob", 2},
		{"Dave", "Eva", 3},
	}

	// Build the core.Graph with directed edges and integer weights.
	// Complexity: building the graph is O(E) for E insertions.
	g := core.NewGraph(
		core.WithDirected(true), // edges have orientation: u→v ≠ v→u
		core.WithWeighted(),     // preserve the weight field on each edge
	)
	for _, e := range interactions {
		g.AddEdge(e.from, e.to, e.weight)
	}
	// At this point: Graph has V=6 vertices, E=9 directed edges.

	// (2) Create options for adjacency-matrix build:
	//     directed + weighted (all other flags default to allow loops/multi).
	// NewMatrixOptions applies defaults internally in O(1).
	directedOpts := matrix.NewMatrixOptions(
		matrix.WithDirected(true),
		matrix.WithWeighted(true),
	)

	// Build the AdjacencyMatrix: O(V+E) time, O(V²) memory.
	adjMat, _ := matrix.NewAdjacencyMatrix(g, directedOpts)
	// Now adjMat.Data is an N×N float64 slice;
	// adjMat.Index maps each vertex ID → row/col index.

	// (3) Constant-time lookup of weight Alice→Bob.
	//    Time: O(1) array access.
	iAlice := adjMat.Index["Alice"]
	iBob := adjMat.Index["Bob"]
	fmt.Printf("Alice→Bob messages: %d\n", int(adjMat.Data[iAlice][iBob]))
	// Output: Alice→Bob messages: 5

	// (4) To perform spectral analysis, we need a **symmetric** matrix.
	//     Re-build the adjacency matrix as undirected (A↔B for any A→B).
	undirectedOpts := matrix.NewMatrixOptions(
		matrix.WithDirected(false), // flip to undirected
		matrix.WithWeighted(true),  // keep the same weights
	)
	symMat, _ := matrix.NewAdjacencyMatrix(g, undirectedOpts)
	// symMat.Data is now symmetric: Data[i][j] == Data[j][i].

	// (5) Run Jacobi eigen decomposition: O(N³) time, O(N²) memory.
	//     Returns sorted eigenvalues and corresponding eigenvector columns.
	eigenvals, _, _ := symMat.SpectralAnalysis(1e-9, 200)
	fmt.Printf("Eigenvalues of undirected adjacency: %v\n", eigenvals)
	// Example output:
	// Eigenvalues of undirected adjacency: [-9.5683, -3.8172, -2.0599, 1.8795, 2.6547, 10.9112]
	// • Large +ve eigenvalues point to strongly connected clusters.
	// • Negative eigenvalues hint at “bipartite-like” structure or cuts.
	// • These λ are the roots of det(A − λI) = 0 for this graph’s matrix A.

	// (6) Finally, round-trip back to core.Graph to ensure no data loss:
	//     O(V² + E) to rebuild edges.
	rebuilt, _ := adjMat.ToGraph()
	rebuiltEdges := rebuilt.EdgeCount()
	fmt.Printf("Round-trip edges: %d (expected %d)\n", rebuiltEdges, len(interactions))
	// Should print: Round-trip edges: 9 (expected 9)
}
```

[![Go Playground](https://img.shields.io/badge/Go_Playground-Matrices-blue?logo=go)](https://go.dev/play/p/XerCKvJ-YEX)

---

## 10.5 Best Practices & Pitfalls

- **Avoid In‑Place Mutation**: build on copies to preserve original data.
- **Choose Correct Representation**: use matrices only when graph is sufficiently dense or when algebraic operations are required.
- **Watch Memory**: adjacency matrix uses O(V²) space—avoid on large sparse graphs.
- **Set Tolerance & Iterations** for spectral: too strict tol or too few iterations may fail convergence (ErrEigenFailed).
- **Validate Options**: `opts := matrix.NewMatrixOptions(...)` ensures consistent behavior.

> **Pro Tip:** Combine matrix forms with list-based structures. E.g., use lists for traversal, matrices for post‑processing spectral methods.

---

Next: [11. Traveling Salesman (TSP) →](TRAVELING_SALESMAN.md)
