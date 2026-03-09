<!--
  lvlath - Repository Documentation

  Purpose:
    This document is the repository-level specification, tutorial, and contract map
    for lvlath/matrix. It explains the deterministic linear-algebra model, dense
    memory layout, numeric policy, graph-to-matrix adapters, spectral and statistical
    kernels, and performance-oriented usage patterns that downstream analytics and
    graph algorithms rely on.

  Contract status:
    - Public numeric and shape rules described here are part of the matrix contract.
    - Deterministic loop-order and reproducibility rules described here are part of the matrix contract.
    - Sentinel-error semantics described here are part of the matrix contract.
    - Adjacency and incidence adapter rules described here are part of the matrix contract.
    - Any incompatible change must be explicit, documented, and versioned.

  Scope:
    - Deterministic dense matrix operations and factorizations.
    - Graph-to-matrix and matrix-to-graph transformations.
    - Spectral, statistical, and reduction-oriented kernels.
    - Numeric sanitization and metric-closure workflows.
    - Allocation-aware patterns for high-load and reproducible computation.

  License:
    The lvlath repository is licensed under AGPL-3.0-only. See LICENSE.
-->

# 10. Matrices: Computational Engine & Linear Algebra

> **Package:** `lvlath/matrix` | **Focus:** Determinism, Hardware Sympathy, Pure Linear Algebra

The `lvlath/matrix` package provides a powerful algebraic and algorithmic perspective on graphs, enabling everything from constant-time edge queries to spectral clustering and connectivity analysis.

Unlike standard object-oriented math libraries (where you might see `A.Mul(B)` mutating state or hiding allocations), `lvlath/matrix` strictly enforces a **pure, functional-style, deterministic API** (e.g., `matrix.Product(A, B)`).

---

## 10.1. What & Why: The Architectural Paradigm

Why use `lvlath/matrix` instead of generic multidimensional slices or object-oriented math libraries? This architecture guarantees three core invariants:

### The Three Laws of Matrix:

1. **Pure Functional Determinism (Binary Reproducibility)**
    *   *The Problem:* Many libraries mutate internal state (`A.Mul(B)` modifies `A`) or rely on non-deterministic map iteration.
    *   *The lvlath Way:* Enforces a **pure, functional API**. Operations like `matrix.Product(A, B)` never mutate inputs. Fixed loop traversal ($i \rightarrow j$), stable vertex sorting, and strictly ordered edge processing guarantee **bit-for-bit reproducibility** across any machine or run.
2. **Hardware Sympathy (Cache Locality)**
    *   *The Problem:* Representing matrices as slice-of-slices (`[][]float64`) causes heap fragmentation and frequent CPU cache misses.
    *   *The lvlath Way:* The canonical `Dense` type maps 2D logical space into a **single 1D flat slice**. This ensures linear memory access, maximizing L1/L2 cache prefetching and keeping the hot data close to the execution units.
3. **Safe by Construction (Strict Error Policy)**
    *   *The Problem:* Other implementations silently propagate `NaN` or `Inf` until the pipeline collapses, or trigger runtime panics on out-of-bounds access.
    *   *The lvlath Way:* Numerical violations and out-of-bounds access yield strict **sentinel errors** (e.g., `matrix.ErrOutOfRange`, `matrix.ErrNaNInf`). The library operates with a "No Magic" policy: it **never panics**, forcing the caller to handle edge cases explicitly.

---

## 10.2. Architecture: Memory Model & Fast-Paths

To write high-performance matrix algorithms, you must understand how `lvlath/matrix` structures memory.

### 10.2.1. Row-Major Flattening
A matrix of size $R \times C$ is backed by a single `make([]float64, R*C)` allocation. Elements of a row are stored contiguously. The package uses the following **Linear Index Mapping** formula:

$$ \text{offset}(i, j) = i \times C + j $$

> [!NOTE]
> **Why this matters:**
> This mapping allows for "Fast-Path" iteration. When you use internal kernels, the library bypasses 2D bounds checks and iterates over the 1D slice directly, which is the most efficient way to utilize modern CPU pipelines.

#### **Visualization A: Memory Layout (2D Logical to 1D Physical)**
```text
      Logical 2D Matrix (3x3)                     Physical 1D Flat Slice (len=9)
      ┌────────┬────────┬────────┐
Row 0 │ A[0,0] │ A[0,1] │ A[0,2] │ ───────────────────────────────┐
      ├────────┼────────┼────────┤                                ▼          ┌──── Row 2 ───┐
Row 1 │ A[1,0] │ A[1,1] │ A[1,2] │ ───►[ 0,0 | 0,1 | 0,2 | 1,0 | 1,1 | 1,2 | 2,0 | 2,1 | 2,2 ]
      ├────────┼────────┼────────┤      └──── Row 0 ───┘   └──── Row 1 ───┘   ▲
Row 2 │ A[2,0] │ A[2,1] │ A[2,2] │ ───────────────────────────────────────────┘
      └────────┴────────┴────────┘      * Contiguous memory blocks fit neatly into 64-byte 
                                          L1/L2 CPU cache lines, maximizing hardware prefetching.
```

### 10.2.2. The "Why" (Hardware Sympathy)
1. **Cache Locality:** Modern CPUs fetch memory in cache lines. Scanning a row reads contiguous floats, minimizing cache misses.
2. **Allocation Efficiency:** `make([]float64, rows*cols)` requires exactly **one** heap allocation. Using `[][]float64` would require $R + 1$ allocations, scattering data and destroying locality.
3. **Fast-Paths (Interface Unwrapping):** Operations like `matrix.Sum` and `matrix.Scale` inspect the interface type. If it is a `*Dense` matrix, they execute a single flat loop `for idx := 0; idx < n; idx++`, bypassing all $i, j$ coordinate math and interface dispatch overhead.

---

## 10.3. Core Kernels & Math Formulation

The `lvlath/matrix` package provides thin API facades that delegate to highly optimized, strictly deterministic internal kernels.


| Facade API            | Operation             | Complexity             | Formulation & Notes                                                                                               |
|:----------------------|:----------------------|:-----------------------|:------------------------------------------------------------------------------------------------------------------|
| `Sum(A, B)`           | Matrix Sum            | $O(R \cdot C)$         | $C = A + B$. Simple element-wise addition.                                                                        |
| `T(A)`                | Matrix Transposition  | $O(R \cdot C)$         | $C_{i,j} = A_{j,i}$. Flips the matrix over its main diagonal.                                                     |
| `Product(A, B)`       | Matrix Multiplication | $O(R \cdot N \cdot C)$ | $C_{i,j} = \sum A_{i,k} B_{k,j}$. Skips $A_{i,k} = 0$ for optimized sparse-on-dense execution.                    |
| `InverseOf(A)`        | Matrix Inversion      | $O(N^3)$               | Uses **Doolittle LU factorization without pivoting** to guarantee bit-level determinism.                          |
| `EigenSym(A, tol, i)` | Eigen Decomposition   | $O(k \cdot N^3)$       | Finds $Q, \Lambda$ ($A = Q \Lambda Q^T$) via deterministic Jacobi sweeps for precision and stability.             |
| `Covariance(X)`       | Sample Covariance     | $O(R \cdot C^2)$       | $Cov = \frac{X_c^T X_c}{n-1}$ (where $X_c$ is the column-mean-centered matrix).                                   |
| `Correlation(X)`      | Pearson Correlation   | $O(R \cdot C^2)$       | $Corr = \frac{Z^T Z}{n-1}$ (Z-scored). Degenerate columns (std=0) yield zeroed results.                           |
| `APSPInPlace(M)`      | Floyd-Warshall        | $O(N^3)$               | $d_{ij} = \min(d_{ij}, d_{ik} + d_{kj})$. Requires `+Inf` for unreachable paths and $d_{ii}=0$.                   |

> [!NOTE]
> **On Determinants:** `lvlath` explicitly omits a `Det()` function. Naive determinant computations are numerically unstable for large matrices, and LU-based determinants can easily overflow.

---

## 10.4. Graph-to-Matrix Adapters

To run algebra on graphs, `lvlath` uses `AdjacencyMatrix` and `IncidenceMatrix` adapters. These structures encapsulate a `Dense` matrix alongside deterministic index mappings.

### The Conversion Pipeline
String-based Vertex IDs must be mapped to stable integer coordinates. To guarantee reproducibility, the adapter **lexicographically sorts** all vertices before mapping.

**ASCII Diagram: Graph to Matrix Mapping**
```text
 [core.Graph]           (Defensive Lex Sort)             [matrix.AdjacencyMatrix]
 Unordered IDs          Stable Integer Mapping           Dense Tensor Coordinates

  "Fiona"   ──────────►  0: "Alice" ───────────────────► Row 0 / Col 0
  "Bob"     ──────────►  1: "Bob"   ───────────────────► Row 1 / Col 1
  "Alice"   ──────────►  2: "Fiona" ───────────────────► Row 2 / Col 2
```

---

## 10.5. Structural Representations

### Social Network Scenario + Example Graph

Consider a social platform where users send messages.

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
*Edges:* Alice→Bob (5), Alice→Dave (3), Bob→Carol (2), Bob→Eva (1), Carol→Dave (4), Carol→Fiona (6), Dave→Eva (3), Fiona→Alice (4), Fiona→Bob (2).

### 10.5.1. Adjacency Matrix ($V \times V$)
Provides $O(1)$ edge queries at the cost of $O(V^2)$ space.
$$A_{ij} = \begin{cases} w_{ij} & \text{if } (i,j) \in E \\ 0 & \text{otherwise} \end{cases}$$

#### **Example: Weighted Adjacency (6×6)**


|           | Alice  |  Bob  | Carol | Dave  |  Eva  | Fiona  |
|-----------|:------:|:-----:|:-----:|:-----:|:-----:|:------:|
| **Alice** |   0    |   5   |   0   |   3   |   0   |   0    |
| **Bob**   |   0    |   0   |   2   |   0   |   1   |   0    |
| **Carol** |   0    |   0   |   0   |   4   |   0   |   6    |
| **Dave**  |   0    |   0   |   0   |   0   |   3   |   0    |
| **Eva**   |   0    |   0   |   0   |   0   |   0   |   0    |
| **Fiona** |   4    |   2   |   0   |   0   |   0   |   0    |

**Complexities:**
- **Build:** $O(V \log V + E)$ time (due to Lexicographical sorting), $O(V^2)$ memory.
- **Edge lookup:** $O(1)$ via `adj.Mat.At(i, j)`.
- **Iterate neighbors:** $O(V)$ scan per row.

---

### 10.5.2. Incidence Matrix ($V \times E$)
Captures structural topology by sign, ignoring numeric weights. Ideal for flow conservation and cycle space calculations.

`lvlath` enforces strict algebraic signs:
*   **Directed Non-Loop:** $M_{u,k} = -1.0$ (source), $M_{v,k} = +1.0$ (target).
*   **Directed Self-Loop:** Algebraically sums to zero. *Builder skips these columns.*
*   **Undirected Non-Loop:** $M_{u,k} = 1.0, M_{v,k} = 1.0$.
*   **Undirected Self-Loop:** Yields $M_{u,k} = +2.0$.

#### **Example: Social Network Incidence (6×9)**
Labeling edges $e_1$ to $e_9$ for the directed scenario:


|         | e₁ | e₂ | e₃ | e₄ | e₅ | e₆ | e₇ | e₈ | e₉ |
|---------|:--:|:--:|:--:|:--:|:--:|:--:|:--:|:--:|:--:|
| Alice   | -1 |  0 | -1 |  0 |  0 |  0 |  0 |  0 | +1 |
| Bob     | +1 | -1 |  0 | -1 |  0 |  0 | +1 |  0 |  0 |
| Carol   |  0 | +1 |  0 |  0 | -1 | -1 |  0 |  0 |  0 |
| Dave    |  0 |  0 | +1 |  0 | +1 |  0 |  0 | -1 |  0 |
| Eva     |  0 |  0 |  0 | +1 |  0 |  0 |  0 | +1 |  0 |
| Fiona   |  0 |  0 |  0 |  0 |  0 | +1 | -1 |  0 | -1 |

**Complexities:**
- **Build:** $O(V + E)$ time, $O(V \cdot E)$ memory.
- **Algebraic ops:** $O(V \cdot E)$. Perfect for Kirchhoff's laws and network flow equations.

---

## 10.6. Configuration Options (ON vs OFF Visualized)

Because a dense matrix cell $A_{i,j}$ holds a single `float64`, specific configuration policies (`matrix.Options`) dictate how complex topological traits translate to matrices.

### Option: `WithDirected()` vs `WithUndirected()`
Controls symmetry. Undirected operations automatically mirror weights across the main diagonal.

```text
Graph: [A] --(5)--> [B]

[ ON: WithDirected ]             [ OFF: WithUndirected ]
    A   B                            A   B  
A [ 0   5 ]                      A [ 0   5 ]
B [ 0   0 ]                      B [ 5   0 ] (Mirrored)
```

### Option: `WithAllowMulti()`
Dense matrices cannot hold multiple parallel edges in one cell. How do we compress them?

```text
Graph: [A] -(w:2)-> [B] (Edge 1),  [A] -(w:8)-> [B] (Edge 2)

[ ON: WithAllowMulti ]           [ OFF: WithDisallowMulti ]
    A   B                            A   B  
A [ 0   8 ]                      A [ 0   2 ]
  (Last-write-wins based           (First-edge-wins. Strictly
   on stable Edge ID order)         ignores subsequent parallel edges)
```

### Option: `WithAllowLoops()`
Self-referential edges $v \rightarrow v$.

```text
Graph: [A] -(w:3)-> [A]

[ ON: WithAllowLoops ]           [ OFF: WithDisallowLoops ]
    A   B                            A   B  
A [ 3   0 ]                      A [ 0   0 ] (Ignored)
B [ 0   0 ]                      B [ 0   0 ]
```

### Option: `WithWeighted()` vs `WithUnweighted()`
Extracting purely topological structure vs magnitudes.

```text
Graph: [A] -(w:42.5)-> [B]

[ ON: WithWeighted ]             [ OFF: WithUnweighted ]
    A     B                          A   B  
A [ 0   42.5 ]                   A [ 0   1 ] (Degraded to binary 1)
B [ 0    0   ]                   B [ 0   0 ]
```

### Option: `WithMetricClosure()` (APSP Distances)
Changes the semantic meaning of the matrix from "Adjacency" to "Distances".

```text
Graph: [A] -(w:5)-> [B],  [C] is disconnected.

[ OFF (Normal Adjacency) ]       [ ON: WithMetricClosure ]
    A   B   C                        A      B      C
A [ 0   5   0 ]                  A [ 0      5    +Inf ]
B [ 0   0   0 ]                  B [ +Inf   0    +Inf ]
C [ 0   0   0 ]                  C [ +Inf  +Inf   0   ]
```
*Note: ToGraph() on a MetricClosure matrix returns `ErrMatrixNotImplemented` to prevent exporting transitive distances as literal edges.*

---

## 10.7. Go Usage Example

Below is a complete, production-grade example using `lvlath/matrix` to ingest the Social Network, resolve options, and perform spectral clustering safely.

```go
package main

import (
	"fmt"
	"log"

	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/matrix"
)

func main() {
	// 1. Define the directed, weighted Social Graph
	interactions :=[]struct {
		from, to string
		weight   float64
	}{
		{"Alice", "Bob", 5}, {"Alice", "Dave", 3},
		{"Bob", "Eva", 1}, {"Bob", "Carol", 2},
		{"Carol", "Dave", 4}, {"Carol", "Fiona", 6},
		{"Fiona", "Alice", 4}, {"Fiona", "Bob", 2},
		{"Dave", "Eva", 3},
	}

	g := core.NewGraph(core.WithDirected(true), core.WithWeighted())
	for _, e := range interactions {
		g.AddEdge(e.from, e.to, e.weight)
	}
	// Graph has V=6 vertices, E=9 directed edges.

	// 2. Build AdjacencyMatrix (O(V+E) time, O(V²) memory)
	opts := matrix.NewMatrixOptions(
		matrix.WithDirected(true),
		matrix.WithWeighted(true),
	)
	adj, err := matrix.NewAdjacencyMatrix(g, opts)
	if err != nil {
		log.Fatalf("Matrix build failed: %v", err)
	}

	// 3. Safe, Constant-time lookup of weight Alice→Bob
	// adj.VertexIndex resolves string IDs to matrix integer coordinates deterministically.
	idxAlice := adj.VertexIndex["Alice"]
	idxBob := adj.VertexIndex["Bob"]
	
	// At() never panics. Returns ErrOutOfRange on failure.
	val, err := adj.Mat.At(idxAlice, idxBob) 
	if err == nil {
		fmt.Printf("Alice→Bob messages: %.1f\n", val) // Output: 5.0
	}

	// 4. Spectral Analysis Preparation
	// Eigen decomposition requires a SYMMETRIC matrix. 
	// We use the facade Symmetrize: (A + A^T) / 2
	symMat, _ := matrix.Symmetrize(adj.Mat)

	// Validate Symmetry (Tolerance: 1e-9)
	if err := matrix.ValidateSymmetric(symMat, 1e-9); err != nil {
		log.Fatalf("Symmetry validation failed: %v", err)
	}

	// 5. Run Jacobi eigen decomposition: O(N³) time, O(N²) memory.
	// Returns sorted eigenvalues and orthogonal eigenvector columns.
	eigenvals, _, err := matrix.EigenSym(symMat, 1e-12, 200)
	if err != nil {
		log.Fatalf("Eigen decomposition failed: %v", err)
	}
	fmt.Printf("Leading Eigenvalue: %.4f\n", eigenvals[0])

	// 6. Round-trip export back to core.Graph
	// O(V² + E) scan. Threshold=0 filters out absent edges.
	rebuilt, _ := adj.ToGraph(matrix.WithEdgeThreshold(0.0))
	fmt.Printf("Round-trip edges: %d (expected 9)\n", rebuilt.EdgeCount())
}
```

[![Go Playground](https://img.shields.io/badge/Go_Playground-Matrices-blue?logo=go)](https://go.dev/play/p/XerCKvJ-YEX)

---

## 10.8. Pitfalls & Best Practices (Architectural Mastery)

To utilize `lvlath/matrix` in high-load, production environments, you must respect its strict policies derived directly from its deterministic core.

> [!IMPORTANT]
> **1. Strict Error Policy (No Panics)**
> **Check Sentinels, Don't Recover.**
> Unlike other libraries that trigger panics on out-of-bounds access, `lvlath` **never panics**.
> *   All public methods return wrapped sentinels: `matrix.ErrOutOfRange`, `matrix.ErrSingular`, or `matrix.ErrDimensionMismatch`.
> *   **Practice:** Always use `errors.Is(err, matrix.ErrOutOfRange)` instead of `defer recover()`. This ensures your system remains predictable under edge-case inputs.

> [!NOTE]
> **2. Performance: Interface Unwrapping**
> **Unlock Cache-Friendly Kernels.**
> The public API uses the `Matrix` interface, which introduces a method-dispatch cost per element.
> *   **Internal Optimization:** Kernels like `matrix.Sum` automatically type-assert to `*Dense` to run optimized flat 1D loops.
> *   **Custom Loops:** If building hot-loops outside the package, assert your objects to `*Dense`. Working directly with `d.RawMatrix().Data` bypasses interface overhead and is orders of magnitude faster.

> [!IMPORTANT]
> **3. Numeric Policy & The Sentinel Trap**
> **Deterministic Handling of NaN/Inf.**
> By default, `lvlath` rejects "dirty" floats to prevent silent algorithmic corruption.
> *   **Policy:** Inserting `math.NaN()` or `math.Inf()` yields `ErrNaNInf`.
> *   **Pathfinding:** If computing distances (APSP), you *must* pass `WithAllowInfDistances()`. This permits `+Inf` (unreachable) but strictly rejects `NaN`.
> *   **Sanitization:** For untrusted data, use `WithNoValidateNaNInf()` during ingestion, then immediately sanitize via `matrix.ReplaceInfNaN(M, 0.0)` or `matrix.Clip(M, lo, hi)`.
> *   **Warning:** Never use `0.0` as a "no path" marker; algorithms will treat it as a zero-cost teleportation.

> [!NOTE]
> **4. Memory Management: Allocation Strategy**
> **Avoid Eager Materialization in Loops.**
> *   **The Risk:** Methods like `matrix.Sum(A, B)` allocate a **new** matrix every call, causing massive GC spikes in tight simulation loops.
> *   **The Fix:** Preallocate staging buffers with `matrix.ZerosLike(A)` outside the loop. For in-place transformations without allocation, use the `A.Apply(...)` write-barrier.

> [!IMPORTANT]
> **5. Concurrency: Safe Reads, Unsafe Writes**
> **Lock-Free for Speed.**
> *   **Safe:** Concurrent calls to `At()`, `Do()`, `Rows()`, and `View()` are thread-safe.
> *   **Unsafe:** Concurrent calls to `Set()`, `Apply()`, or `Fill()`. The underlying slice is not protected by mutexes.
> *   **Strategy:** If multiple goroutines must write to the same matrix, synchronize them externally or use discrete sub-views via `View(i, j, rows, cols)`.

---
**lvlath/matrix**: Pure linear algebra. Zero magic. Absolute determinism.
> Next: [11. Traveling Salesman (TSP) →](TRAVELING_SALESMAN.md)
> 