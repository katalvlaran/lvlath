# 9. GridGraph

`lvlath/gridgraph` models a 2D grid of integer cell values as a graph. Each cell becomes a vertex; edges connect neighboring cells based on **Connectivity** (4- or 8-directional). Cells with value `0` are “water” (impassable); cells with value ≥1 are “land” (passable). This enables common grid tasks like island counting, terrain bridging, and conversion to general graph algorithms.

---

## 9.1 Definitions

* **Conn4 (4-connectivity)**: each cell connects to its four orthogonal neighbors (N, E, S, W).
* **Conn8 (8-connectivity)**: also includes the four diagonal neighbors (NE, SE, SW, NW).

```ascii
 Conn4:           Conn8:
   . X .            X X X
   X X X            X X X
   . X .            X X X
```

*Use Conn4 for strict orthogonal movement (e.g., grid mazes); use Conn8 for natural terrain connectivity.*

---

## 9.2 Use Cases

1. **Island Counting**: identify clusters of adjacent “land” cells via `ConnectedComponents()`.
2. **Terrain Bridging**: find minimal-cost paths over water (cost=1) vs land (cost=0) using `ExpandIsland()`.
3. **Graph Conversion**: transform any grid into a `*core.Graph` for shortest-path or flow algorithms via `ToCoreGraph()`.

---

## 9.3 Core Functions

These methods are defined on `*gridgraph.GridGraph` in `gridgraph.go` and related files:

| Function                                | Signature                                                        | Ref               |
| --------------------------------------- | ---------------------------------------------------------------- | ----------------- |
| Construct from 2D slice                 | `From2D(values [][]int, conn Connectivity) (*GridGraph, error)`  | citeturn4file3 |
| Connected components (clusters of land) | `ConnectedComponents() [][]int`                                  | citeturn4file5 |
| Minimal-cost bridge between two islands | `ExpandIsland(srcComp, dstComp int) (path []int, cost int, err)` | citeturn4file6 |
| Convert to general graph                | `ToCoreGraph() *core.Graph`                                      | citeturn4file0 |

### 9.3.1 ConnectedComponents

Performs a BFS flood-fill over all `CellValues[y][x] >= 1`, grouping each contiguous region of land into a slice of row-major indices. Returns all components in arbitrary order.

### 9.3.2 ExpandIsland (0–1 BFS)

Implements a 0–1 BFS from every cell in source component:

* **Cost 0** for stepping on land (`CellValues>=1`).
* **Cost 1** for stepping on water (`CellValues<1`).

Stops when any target-component cell is reached; reconstructs the shortest conversion path and its total cost.

### 9.3.3 ToCoreGraph

Transforms the grid into a `*core.Graph`:

* Each cell at `(x,y)` becomes a vertex with ID `"x,y"` and metadata `{x, y, value}`.
* Edges of weight 1 connect every neighboring pair per the chosen connectivity.

Time: O(W·H·d), Memory: O(W·H + E), where d = 4 or 8, E≈W·H·d.

---

## 9.4 ASCII Visualization

Consider the following 5×5 height map (`0`=water, `1`=land):

```ascii
 0 1 1 0 0
 1 1 0 1 1
 0 0 1 1 0
 1 1 1 0 0
 0 0 0 1 1
```

* **Conn4** components (two islands A and B):

```ascii
   . # # . .    (# marks one island)
   # # . # #    (second island appears separately)
   . . . . .    
   # # # . .    
   . . . . .    
```

* **Conn8** merges diagonals into a single component:

```ascii
   . # # . .
   # # . # #
   . . # # .
   # # # . .
   . . . . .
```

*(Here all `#` form one contiguous island under Conn8.)*

---

## 9.5 Go Playground Example

```go
package main

import (
    "fmt"
  
    "github.com/katalvlaran/lvlath/gridgraph"
)

func main() {
    // 5×5 height map (with 2 islands)
    heights := [][]int{
        {0, 1, 1, 0, 0},
        {1, 1, 0, 1, 1},
        {0, 0, 1, 1, 0},
        {1, 1, 0, 0, 0},
        {0, 0, 0, 1, 1},
    }
  
    // Build GridGraph with 8-connectivity
    gg, err := gridgraph.From2D(heights, gridgraph.Conn8)
    if err != nil {
        panic(err)
    }
  
    // 1) Find islands
    comps := gg.ConnectedComponents()
    fmt.Printf("Found %d island(s) with Conn8:\n", len(comps))
    for i, comp := range comps {
        fmt.Printf(" Island %d cells: %v\n", i+1, comp)
    }
    // Found 2 island(s) with Conn8:
    // Island 1 cells: [1 2 6 5 8 12 9 13 16 15]
    // Island 2 cells: [23 24]
  
    // 2) Bridge from island 0 to island 1
    if len(comps) >= 2 {
        path, cost, err := gg.ExpandIsland(0, 1)
        if err != nil {
            panic(err)
        }
        fmt.Printf("Min-cost bridge path: %v (cost=%d)\n", path, cost)
    }
    // Min-cost bridge path: [16 17 23] (cost=1)
}

```

[![Go Playground](https://img.shields.io/badge/Go_Playground-GridGraph-blue?logo=go)](https://go.dev/play/p/FQ5oviAYt0I)

---

## 9.6 Pitfalls & Best Practices

* **Connectivity choice**: use **Conn4** for strict orthogonal grids (e.g., mazes), **Conn8** for realistic movements over terrain.
* **0–1 BFS** offers **O(V+E)** performance: ideal for binary-cost grids.
* **Consistent indexing**: row-major indexing (`index(x,y)=y*W+x`) ensures reproducible component ordering.
* **Memory**: avoid deep recursion; both `ConnectedComponents` and `ExpandIsland` are iterative.
* **Metadata**: leverage `ToCoreGraph()` to annotate vertices for use with other algorithms.

---

Next: [10. Matrices: Adjacency & Incidence →](MATRICES.md)
