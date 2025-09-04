## 12. Best Practices & Tips

In this section, we share pragmatic advice to help you extend, test, and maintain **lvlath** at a professional level.

---

### 12.1 Extending lvlath with New Algorithms

1. **Create a new subpackage** under `pkg/`, e.g. `pkg/myalgo`:

   ```text
   pkg/myalgo/
     myalgo.go
     myalgo_test.go
     doc.go         # high-level overview
     types.go       # definitions for input/output types
     example_test.go
   ```

2. **doc.go** should answer:

    * What the algorithm does
    * When to use it
    * Time & space complexity

3. **Pseudo‑template** for `myalgo.go`:

   ```go
   package myalgo

   // SolveMyAlgo computes ...
   // Complexity: O(N log N)
   func SolveMyAlgo(input InputType, opts ...Option) (OutputType, error) {
     // 1. Validate input
     // 2. Preprocess (e.g., sort, index)
     // 3. Core algorithm
     // 4. Return result
   }
   ```

4. **Types & Options**: define clear, immutable structs for inputs and functional `Option` patterns.

---

### 12.2 Testing Patterns & Templates

1. **Table‑Driven Tests**:

   ```go
   func TestSolveMyAlgo(t *testing.T) {
     cases := []struct {
       name  string
       input InputType
       want  OutputType
       err   error
     }{
       {"simple case", ..., ..., nil},
       {"edge case", ..., ..., ErrBadInput},
     }
     for _, tc := range cases {
       t.Run(tc.name, func(t *testing.T) {
         got, err := SolveMyAlgo(tc.input)
         require.ErrorIs(t, err, tc.err)
         if err == nil {
           require.Equal(t, tc.want, got)
         }
       })
     }
   }
   ```

2. **Integration & Example Tests**:

    * Place in `example_test.go` to show end‑to‑end usage.

3. **Benchmarks**:

   ```go
   func BenchmarkSolveMyAlgo(b *testing.B) {
     data := generateLargeInput()
     b.ResetTimer()
     for i := 0; i < b.N; i++ {
       _, _ = SolveMyAlgo(data)
     }
   }
   ```

---

### 12.3 Hook Mechanism for Traversals

Leverage **OnVisit/OnEdge** hooks to inject custom logic without modifying core code:

```go
type WalkOption func(*walker)

// OnVisit is called for each node.
func OnVisit(fn func(node NodeID)) WalkOption
```

**ASCII Illustration** of BFS with hook logging:

```text
  1──2──3
  │  ╱
  4

Visit sequence with OnVisit: 1 → 2 → 4 → 3
```

---

### 12.4 Performance Pitfalls & Optimization

* **Avoid global state**: use pure functions.
* **Minimize allocations**: reuse buffers (e.g., slice pools).
* **Choose appropriate data structures**: adjacency list vs matrix based on density.

---

### 12.5 Coding Conventions

* **Consistent naming**: `SolveXxx`, `NewXxx`, `Option` suffix.
* **Documentation**: every public type/function must have GoDoc.
* **Error handling**: wrap with `%w` for clarity.

---

> **Keep your code clean, well‑documented, and benchmarked-hard math on an easy level!**

## 14. FAQ & Troubleshooting

Below are the most common questions, pitfalls, and solutions you may encounter when using **lvlath**.

---

### Q1: Why is **DTW** slow on long time series?

**Problem:** When aligning two sequences of length *n* and *m*, the full DTW matrix requires `O(n*m)` time and memory.

**Solutions:**

1. **Sakoe-Chiba Band:** Restrict comparisons to a window of width *w* around the diagonal:

```go
opts := dtw.SakoeChiba(windowSize)
res := dtw.Align(a, b, opts)
```

This reduces complexity to O(n*w).

2. **Rolling Array (MemoryMode):** Use `RollingArray` to only keep two rows in memory:

```go
dtw.SetMemoryMode(dtw.RollingArray)
```

3. **Downsample or Dimensionality Reduction:** Preprocess signals with averaging or PCA.

---

### Q2: How to choose **MemoryMode** for DTW?

```go
// FullMatrix stores all distances for path retrieval
dtw.SetMemoryMode(dtw.FullMatrix)

// RollingArray uses constant memory but cannot reconstruct full path
dtw.SetMemoryMode(dtw.RollingArray)
```

* Use **FullMatrix** if you need the optimal warp path.
* Use **RollingArray** when only the distance is required and memory is at a premium.

---

### Q3: What to do when **TSP** returns infinite distances?

* **Cause:** Graph might be disconnected or contains edges with no defined weight.
* **Check:** Ensure every vertex is reachable, or define a large fallback weight:

```go
g := core.NewGraph()
// ... add vertices/edges
if !g.HasEdge(u, v) {
    g.AddEdge(u, v, defaultWeight)
}
```

* **Tip:** Run a connectivity check before TSP:

```go
if len(algorithms.ConnectedComponents(g)) > 1 {
    log.Fatal("Graph is not fully connected")
}
```

---

### Q4: How to interpret **Max-Flow** discrepancies?

* **Common pitfall:** Forgetting to reset residual capacities between runs.
* **Fix:** Clone your network:

```go
copy := flow.CloneNetwork(orig)
ff := flow.EdmondsKarp(copy, s, t)
```

* **Tip:** Use `flow.Dinic` for larger networks-it builds level graphs to avoid redundant searches.

---

### Q5: Why does **BFS/DFS** sometimes skip nodes?

* **Cause:** Missing `visited` initialization or improper hook usage.

```go
visited := make(map[int]bool)

graph.BFS(start, graph.OnVisit(func(u int) {
    visited[u] = true
}))
```

* **Ensure:** You attach the `OnVisit` hook before starting the traversal.

---

### Q6: Memory & Performance Tips

* **Pre-allocate slices:** When adding many edges:

```go
edges := make([]core.Edge, 0, expectedEdges)
```

* **Avoid reflection:** Use concrete types in your algorithms, not `interface{}`.
* **Profiling:** Use `pprof` and `benchstat` for benchmarks.

---

*Still have questions? Feel free to open an issue on our [GitHub repo](https://github.com/katalvlaran/lvlath/issues).*


