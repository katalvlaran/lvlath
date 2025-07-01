# 8. Dynamic Time Warping (DTW)

Dynamic Time Warping (DTW) is a powerful algorithm for measuring similarity between two sequences that may vary in speed or length. By “warping” the time axis, DTW aligns patterns optimally under configurable constraints.

---

## 8.1 What, Why, When?

### What
DTW computes an **alignment cost** and an **optimal warping path** between two sequences  
$$\(\mathbf{a} = [a_1, a_2, \dots, a_n]\)$$ and $$\(\mathbf{b} = [b_1, b_2, \dots, b_m]\)$$ ,  
allowing local expansions or contractions in time.

### Why
- **Robustness to non-linear time shifts**: Handles local accelerations/decelerations in signals.
- **Interoperability**: Works on any numeric sequence—audio, sensor readings, stock prices.
- **Optional path recovery**: Full-matrix mode lets you see exactly which elements align.

### When
Use DTW when your two sequences:
1. Share the same underlying pattern but differ in pacing (e.g. spoken words at different speeds, sensor data with jitter).
2. May have insertions/deletions (dropped or extra samples).
3. Require an interpretable alignment path for further analysis (e.g. gesture matching, signature verification).

---

## 8.2 Key Concepts & Mathematical Formulation

### 8.2.1 Cost Matrix
Define a matrix $$\(D\in\mathbb{R}^{(n+1)\times(m+1)}\)$$ where
- $$\(D_{i,j}\)$$ is the minimal cumulative cost to align prefixes $$\(a_{1..i}\)$$ and $$\(b_{1..j}\)$$.
- We index from 0 with boundary conditions at $$\(i=0\)$$ or $$\(j=0\)$$.

### 8.2.2 Recurrence Relation

$$\begin{aligned}
D_{0,0} &= 0, \quad D_{i,0} = +\infty\; (i>0), \quad D_{0,j} = +\infty\;(j>0), \\\
D_{i,j} &= \underbrace{\bigl|\,a_i - b_j\,\bigr|}_{\text{local cost}} + \min\Bigl\{ D_{i-1,j-1},\; D_{i-1,j} + p,\; D_{i,j-1} + p \Bigr\}, \\\
\end{aligned}$$

$$\begin{bmatrix} D_{0,0} = 0, & D_{i,0} = +\infty (i>0), & D_{0,j} = +\infty(j>0), \\\ D_{i,j} = \underbrace{\bigl|\,a_i - b_j\,\bigr|}_{\text{local cost}} + \min\Bigl\{ D_{i-1,j-1}, & D_{i-1,j} + p, & D_{i,j-1} + p \Bigr\}, \\\ \end{bmatrix}\;$$

where
- $$\(\lvert a_i - b_j\rvert\)$$ is the absolute difference (or any other distance metric from `core/`).
- $$\(p\ge0\)$$ is the **slope penalty** controlling the cost of insertions/deletions.

### 8.2.3 Window Constraint (Sakoe-Chiba)
To enforce locality and reduce computation, only compute $$\(D_{i,j}\)$$ when  
$$\[ \lvert i - j\rvert \le w, \]$$
for a user-supplied radius $$\(w\)$$. Outside the band, set $$\(D_{i,j}=+\infty\)$$ .

### 8.2.4 Memory Modes
- **FullMatrix**  
  Store the entire $$\((n+1)\times(m+1)\)$$ matrix in `matrix.Dense` for both distance and backtracking (path recovery).  
  $$\(\displaystyle\mathcal{O}(n\,m)\)$$ time and space.

- **TwoRows** (Rolling Array)  
  Keep only two rows in memory (`[]float64`): current and previous. Supports distance only.  
  $$\(\displaystyle\mathcal{O}(n\,m)\)$$ time, $$\(\mathcal{O}(\min(n,m))\)$$ space.

- **NoMemory**  
  Single value update (no backtracking), same time as TwoRows but constant extra memory.

---

## 8.3 Algorithms Overview

This section walks through the architecture and step‑by‑step mechanics of the DTW algorithm, balancing precision and intuition.

### 8.3.1 Core Idea

At its heart, DTW computes the minimal cumulative cost to warp and align two sequences `A = [a_1, ..., a_n]` and `B = [b_1, ..., b_m]`.  Instead of enforcing a one‑to‑one index match, DTW:

1. Builds a **cost matrix** $$\(D\)$$ of size $$\((n+1) \times (m+1)\)$$ , where
   
   $$D_{i,j} = \min \begin{cases}
   D_{i-1, j-1} + \lvert a_i - b_j \rvert, & \text{match (diagonal)} \\\
   D_{i-1, j} + p, & \text{insertion (vertical)} \\\
   D_{i, j-1} + p, & \text{deletion  (horizontal)} \\\
   \end{cases}$$

   and $$\(p\)$$ is the **slope penalty** controlling the cost of skips.

2. Optionally applies the **Sakoe-Chiba window** $$\(w\)$$ to restrict $$\(|i-j| \le w\)$$, improving locality and reducing computation.

3. Fills the matrix by dynamic programming in $$\(O(n\,m)\)$$ time and either:
    - **Stores only two rows** (rolling array) for distance-only $$(\(O(\min(n,m))\)$$ memory).
    - **Stores full matrix** for path recovery $$(\(O(n\,m)\)$$ memory).

4. (If requested) **Backtracks** from $$\((n,m)\)$$ to $$\((0,0)\)$$, reversing moves that achieve the minimal cost to recover the optimal alignment path.

### 8.3.2 Features

- **Window Constraint** (Sakoe-Chiba): $$\(\lvert i - j \rvert \le w\)$$ accelerates computation and prevents pathological warps.
- **Slope Penalty**: weight $$\(p\)$$ penalizes excessive insertions/deletions, trading off flexibility vs. smoothness.
- **Memory Modes**:
    - **FullMatrix**: recover alignment path.
    - **TwoRows**:
    - **NoMemory**: compute distance only, minimal memory.
- **Flexible Cost**: any local distance metric (e.g. Euclidean, Manhattan) can replace $$\(\lvert a_i - b_j \rvert\)$$ .

### 8.3.3 Improvements and Advantages

- **Robust to Speed Variations**: aligns subsequences of different pacing without manual resampling.
- **Local Flexibility**: window and penalty parameters let you fine‑tune allowable warping.
- **Modular Design**: separation of distance computation and backtracking enables memory‑efficient distance only or full path retrieval.
- **Extensible**: integrates with our `core/matrix` and `core/window` packages for optimized low‑level operations.

### 8.3.4 Pseudocode
```text
function DTW(A[1..n], B[1..m], window w, penalty p):
  allocate D[0..n][0..m]
  D[0][0] ← 0
  for i in 1..n: D[i][0] ← +∞
  for j in 1..m: D[0][j] ← +∞

  for i in 1..n:
    for j in max(1, i-w)..min(m, i+w):
      cost  ← |A[i] - B[j]|
      ins   ← D[i-1][j]   + p
      del   ← D[i][j-1]   + p
      match ← D[i-1][j-1]
      D[i][j] ← cost + min(ins, del, match)

  distance ← D[n][m]

  if path needed:
    P ← []
    i, j ← n, m
    while i>0 or j>0:
      append (i,j) to P
      subtract cost = |A[i]-B[j]|
      if D[i][j] - cost == D[i-1][j-1]: (i,j) ← (i-1,j-1)
      else if D[i][j] - cost == D[i-1][j] + p: i ← i-1
      else: j ← j-1
    reverse(P)
    return distance, P

  return distance
```

> **Time Complexity**: $$\(O(n \times m)\)$$ — every cell is visited once.
>
> **Memory Complexity**: $$\(O(n \times m)\)$$ for full matrix, or $$\(O(\min(n,m))\)$$ for rolling array.

### 8.3.5 Highlights

- **Guarantees** globally optimal alignment under given cost model.
- **Parameterizable**: adjust `w` and `p` to match application requirements.
- **Backtracking** recovers the warping path for visualization or further analysis.

### 8.3.6 Illustration & ASCII Example
```text
Aligning A = [1, 2, 3] with B = [1, 2, 2, 3], w = 1, p = 0:

    B →  0   1    2    2    3
A ↓
  0    0  +∞   +∞   +∞   +∞
  1   +∞   0    0    1    2
  2   +∞   1    0    0    1
  3   +∞   2    0    0    0
```
- The **diagonal band** (|i-j|≤1) is enforced.
- Optimal path: (1,1)→(2,2)→(2,3)→(3,4).

### 8.3.7 Go Playground Example
```go
package main

import (
  "fmt"
  "github.com/katalvlaran/lvlath/dtw"
)

func main() {
  a := []float64{0, 0.8, 1, 2, 1, 0}
  b := []float64{0, 1.3, 1.9, 1.6, 0}

  opts := dtw.DefaultOptions()
  opts.Window = -1          // no window, free warping
  opts.SlopePenalty = 0.0   // zero cost for insert/delete
  opts.ReturnPath = true    // retrieve path
  opts.MemoryMode = dtw.FullMatrix

  dist, path, err := dtw.DTW(a, b, &opts)
  if err != nil {
    panic(err)
  }

  fmt.Printf("distance=%.1f\n", dist, path)
  fmt.Println("path=", path)
  // Output:
  // distance=1.5
  // path= [{0 0} {1 1} {2 1} {3 2} {4 3} {5 4}]
}
```
[![Run on Go Playground](https://img.shields.io/badge/Go%20Playground-DTW-blue?logo=go)](https://go.dev/play/p/Q6v-pzWghyi)

---

## 8.4 Pitfalls & Best Practices

### 8.4.1 Pitfalls

1. **RollingArray ≠ Path**  
   Rolling (TwoRows/NoMemory) modes **cannot** recover a warping path. If you call `DTW(..., ReturnPath=true)` without `MemoryMode=FullMatrix`, you will get `ErrPathNeedsMatrix`.

2. **Inf Cost if Window Too Narrow**  
   A small window $$\(w\)$$ may exclude any valid alignment, resulting in $$\(D_{n,m}=+\infty\)$$ . Always ensure $$\(w\ge|n-m|\)$$ if full alignment is required.

3. **Improper Penalty Tuning**
    - A zero penalty $$\(p=0\)$$ allows free skipping but may over-warp (collapse long segments).
    - A large penalty discourages skips but may force poor matches along the diagonal.

4. **Unnormalized Data**  
   Raw amplitudes or measurements may have different scales. Always **pre-normalize** (e.g. zero-mean, unit-variance) if comparing heterogeneous signals.

### 8.4.2 Best Practices

- **Choose the Right Mode**
    - Use **FullMatrix** + `ReturnPath=true` when you need the explicit alignment.
    - Use **TwoRows** for large sequences when only the distance matters.

- **Window Recommendation**
    - Start with a generous $$\(w=\max(n,m)\)$$ (i.e. no constraint) to verify basic alignment.
    - Then tighten $$\(w\)$$ to reduce noise and runtime.

- **Penalty Strategy**
    - A small fractional penalty (e.g. $$\(p=0.1\))$$ often balances flexibility and stability.
    - Experiment with your domain data.

- **Leverage Subpackages**
    - Use `core.Distance` implementations to swap out $$\(\lvert a_i - b_j\rvert\)$$ for Euclidean, squared, or custom metrics.
    - Use `matrix.Dense` for heavy-duty full-matrix operations and backtracking.

- **Batch & Parallelize**  
  For multiple sequence pairs, consider concurrent invocations of `DTW` in separate goroutines. Each call is safe as it does not share mutable state.

---

> Next: [9. SGridGraph: Grid-based Graphs →](GRID_GRAPH.md)
