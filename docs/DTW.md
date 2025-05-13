## 8. Dynamic Time Warping (DTW)

Dynamic Time Warping (DTW) measures similarity between two sequences—such as audio signals, sensor readings, or time series—that may vary in speed or length. It "warps" the time axis to align patterns optimally, handling local accelerations and decelerations.

### Why DTW?

* **Speech & Audio**: match spoken words at different speaking rates.
* **Gesture Recognition**: align motion capture data with varying pace.
* **Signature Verification**: compare handwriting strokes despite speed differences.

### Key Concepts

1. **Cost Matrix**: a grid where cell `(i,j)` holds the cumulative cost to align prefixes
   ![\Large a_{1..i}](https://latex.codecogs.com/svg.image?\large&space;a_{1..i}) and .
   ![\Large b_{1..i}](https://latex.codecogs.com/svg.image?\large&space;b_{1..i}).

2. **Recurrence**:

![\Large D_{i,j} = |a_i - b_j| + \min\bigl\{ D_{i-1,j} + p,\, D_{i,j-1} + p,\, D_{i-1,j-1} \bigr\}](https://latex.codecogs.com/svg.image?\large&space;D_{i,j}=|a_i-b_j|&plus;\min\{D_{i-1,j}&plus;p\,D_{i,j-1}&plus;p\,D_{i-1,j-1}\} )

where:

* ![\Large |a_i - b_j|](https://latex.codecogs.com/svg.image?\large&space;|a_i-b_j|) is the local distance (e.g. absolute difference)
* `p` is the **slope penalty** for insert/delete moves.

3. **Window Constraint** (Sakoe–Chiba): limit
   ![\Large |a_i - b_j|](https://latex.codecogs.com/svg.image?\large&space;|i-j|\le&space;w) to speed up and enforce locality.

4. **Memory Modes**:
    * **FullMatrix**: store entire
      ![\Large (n+1)\times(m+1)](https://latex.codecogs.com/svg.image?\large&space;(n&plus;1)\times(m&plus;1)) grid to backtrack optimal path.
    * **RollingArray**: only two rows at a time; saves memory
      ![\LargeO(\min(n,m))](https://latex.codecogs.com/svg.image?\large&space;O(\min(n,m)), but no path.

### Pseudocode (FullMatrix + Path)

```pseudo
function DTW(a[1..n], b[1..m], w, p):
  D ← matrix of size (n+1)x(m+1)
  for i in 1..n: D[i][0] ← +∞
  for j in 1..m: D[0][j] ← +∞
  D[0][0] ← 0

  for i in 1..n:
    for j in max(1, i-w)..min(m, i+w):
      cost ← |a[i] - b[j]|
      ins  ← D[i-1][j]   + p
      del  ← D[i][j-1]   + p
      match← D[i-1][j-1]
      D[i][j] ← cost + min(ins, del, match)

  // Backtrack if path needed
  path ← empty list
  i, j ← n, m
  while i>0 or j>0:
    append (i-1, j-1) to path
    choose predecessor yielding D[i][j] - |a[i]-b[j]|
    move i,j accordingly
  return D[n][m], reverse(path)
```

### Illustration

| a\b | 0 | 1   | 2   | 3   |
|-----|---|-----|-----|-----|
| 0   | 0 | inf | inf | inf |
| 1   |inf| 0   | 1   | 2   |
| 2   |inf| 1   | 0   | 0   |
| 3   |inf| 2   | 0   | 0   |

Here the main diagonal aligns matching values with zero cost.

### Go Example

```go
// Example: Align two audio amplitude sequences
package main

import (
  "fmt"
  "github.com/katalvlaran/lvlath/dtw"
)

func main() {
  a := []float64{0.1, 0.5, 0.9, 0.2, 0.0}
  b := []float64{0.0, 0.4, 0.8, 0.3}
  opts := &dtw.DTWOptions{
    Window:       1,
    SlopePenalty: 0.5,
    ReturnPath:   true,
    MemoryMode:   dtw.FullMatrix,
  }
  dist, path, err := dtw.DTW(a, b, opts)
  if err != nil {
    panic(err)
  }
  fmt.Printf("DTW distance = %.2f\n", dist)
  fmt.Println("Warp path:", path)
}
```
[![Run on Go Playground](https://img.shields.io/badge/Go%20Playground-DTW-blue?logo=go)](https://go.dev/play/p/CV0IqXvxLNa)

### Complexity & Tips

* **Time**: O(n*m) with window reduces constant factor.
* **Memory**: O(n*m) or O(min(n,m)) for RollingArray.
* **Pitfalls**:

    * RollingArray **cannot** recover path; only distance.
    * Narrow window may force inf cost if sequences drift.

### Best Practices

* Pre-normalize signals (zero-mean, unit-variance) for audio.
* Experiment with slope penalty to control warping flexibility.
* Use `Window` to speed up long sequences when minor shifts expected.

---

Next: [9. SGridGraph: Grid-based Graphs →](GRID_GRAPH.md)