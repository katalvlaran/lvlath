![img.png](img.png)

---

```
┌─────────────────────────────────┐
│  o       o            o    o    │
│  ║ o   o ║   o───╖  o─╫─o  ║    │
│  ║  \ /  ║   o───╢    ║    ╟──╖ │
│  ╙o  o   ╙o  ╙───o    ╙o   o  o │
└─────────────────────────────────┘
```

[![pkg.go.dev](https://img.shields.io/badge/pkg.go.dev-reference-blue?logo=go)](https://pkg.go.dev/github.com/katalvlaran/lvlath)
[![Go Report Card](https://goreportcard.com/badge/github.com/katalvlaran/lvlath)](https://goreportcard.com/report/github.com/katalvlaran/lvlath)
[![Go version](https://img.shields.io/badge/go-%3E%3D1.23-blue)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![CI](https://github.com/katalvlaran/lvlath/actions/workflows/go.yml/badge.svg)](https://github.com/katalvlaran/lvlath/actions)

---

# lvlath

**lvlath** is a practical Go toolkit for **graphs**, **flows**, **TSP**, and **time‑series alignment**. It is **pure Go** (no cgo, no external deps), **deterministic by default**, and built around **strict error contracts**.

* Deterministic outcomes for the same inputs/options
* Early, explicit validation with shared sentinel errors
* Small, composable packages you can use independently or together
* Reproducible fixtures via the `builder` package

---

## What & Why (balanced overview)

* **`core`** — Thread‑safe graph primitives with the options most people actually need: directed/undirected, weighted/unweighted, multi‑edges and self‑loops, safe add/remove/clone, and predictable iteration order. Exposes shared sentinel errors (`ErrNonSquare`, `ErrAsymmetry`, `ErrTimeLimit`, …) so tests can assert with `errors.Is`.

  * **Bridge to `bfs`/`dfs`**: both operate directly on `core.Graph` and expose hookable events (visit/enqueue/edge) for tracing or early stopping—no global state.
* **`bfs`** — Breadth‑first search with a minimal API, cancellation & hooks, and stable layering (same input ⇒ same order). Foundation for reachability, components, and unweighted shortest paths.
* **`dfs`** — Depth‑first traversal plus optional cycle detection and topological sort. Explicit pre/post hooks make it ideal for classification and analysis.
* **`dijkstra`** — Single‑source shortest paths (non‑negative weights) with deterministic parent trees and clear unreachable semantics. A solid baseline for routing.
* **`prim_kruskal`** — Minimum Spanning Trees via Prim (O(E log V)) and Kruskal (≈O(E log V)). Deterministic tie‑breaking keeps MST weights stable.
* **`flow`** — Max‑flow / min‑cut: Edmonds–Karp (simple, robust) and Dinic (typically much faster). Clean separation of capacity graph, residual logic, and results.
* **`dtw`** — Classic O(N·M) Dynamic Time Warping with optional constraints (Sakoe–Chiba bands) and memory modes. Useful in DSP/ML pipelines.
* **`tsp`** — Practical toolbox: Christofides‑style approximation (symmetric metrics), 2‑opt/3‑opt local search, 1‑tree lower bound, and exact Branch‑and‑Bound for small N. Deterministic by default; seed‑controlled local search when needed.
* **`matrix`** — Minimal, bounds‑checked matrix types for dense algorithms (Prim, TSP), with helpers for degree vectors, multiplication, spectral analysis, and round‑trip conversion to/from graphs.
* **`gridgraph`** — 2D lattices with 4/8‑neighborhoods, obstacle masks, and weight helpers. A natural companion for BFS/Dijkstra.
* **`builder`** — Deterministic generators (rings, grids, rippled circles, seeded fixtures) to prototype quickly and write reproducible benchmarks/tests.

> Background “What/Why/When” lives in `docs/{ALGORITHM}.md`; formal contracts are in `{package}/doc.go`. A guided tour starts at `docs/TUTORIAL.md`.

---

## Installation

```bash
go get github.com/katalvlaran/lvlath@latest
```

*Requires Go ≥ 1.23. Pure Go. No external dependencies.*

---

## Package map & roles

```
core ─┬─ bfs, dfs ──► utilities (reachability, ordering)
      ├─ dijkstra ──► weighted shortest paths (non‑negative)
      ├─ prim_kruskal ──► MST (dense via matrix / sparse via edge list)
      ├─ flow ───────────► max‑flow / min‑cut
      ├─ tsp ────────────► symmetric approx + local search + exact small‑n
      ├─ matrix ─────────► dense views used by Prim/TSP
      ├─ gridgraph ──────► 2D lattices for BFS/Dijkstra teaching & demos
      └─ builder ────────► deterministic fixtures for examples & tests
```

* Use packages independently or compose them.
* Prefer `matrix` + Prim on dense graphs; use `core` + Kruskal on sparse ones.
* `builder` and `gridgraph` speed up learning, benchmarking, and test setup.

---

## Quick start (practical)

> Code below is API‑shaped to lvlath; see each `{package}/doc.go` for exact signatures.

### A) Graph **“lvlath”** — shapes, components & a tiny transformation

We create six disconnected letter‑shapes `l v l a t h` using the naming scheme you described. We’ll compute connected components, cyclomatic number, and patch three letters into a single `M`.

```go
package main

import (
    "fmt"
    "sort"

    "github.com/katalvlaran/lvlath/bfs"
    "github.com/katalvlaran/lvlath/core"
    "github.com/katalvlaran/lvlath/matrix"
)

// findComponents discovers connected components using BFS.Depth.
func findComponents(g *core.Graph) [][]string {
    visited := map[string]bool{}
    var comps [][]string

    // iterate in deterministic order
    verts := g.Vertices()
    sort.Strings(verts)
    for _, v := range verts {
        if visited[v] { continue }
        res, err := bfs.BFS(g, v) // unweighted graph required
        if err != nil { panic(err) }
        var comp []string
        for _, u := range res.Order {
            visited[u] = true
            comp = append(comp, u)
        }
        comps = append(comps, comp)
    }
    return comps
}

func main() {
    // Undirected, unweighted for traversal work.
    g := core.NewGraph() // defaults: undirected, unweighted (per core/doc.go)

    // l (first)
    g.AddEdge("l1_top", "l1_middle", 0)
    g.AddEdge("l1_middle", "l1_base", 0)
    g.AddEdge("l1_base", "l1_tail_base_right", 0)

    // v
    g.AddEdge("v_left", "v_center", 0)
    g.AddEdge("v_center", "v_right", 0)

    // l (second)
    g.AddEdge("l_top", "l_middle", 0)
    g.AddEdge("l_middle", "l_base", 0)
    g.AddEdge("l_base", "l_tail_base_right", 0)

    // a (one cycle inside the circle)
    g.AddEdge("a_tail_top_left", "a_top", 0)
    g.AddEdge("a_top", "a_circle_middle", 0)
    g.AddEdge("a_circle_middle", "a_circle_base", 0)
    g.AddEdge("a_circle_base", "a_circle_base_left", 0)
    g.AddEdge("a_circle_base_left", "a_circle_middle_left", 0)
    g.AddEdge("a_circle_middle_left", "a_circle_middle", 0)

    // t
    g.AddEdge("t_top", "t_cross", 0)
    g.AddEdge("t_left", "t_cross", 0)
    g.AddEdge("t_right", "t_cross", 0)
    g.AddEdge("t_base", "t_cross", 0)
    g.AddEdge("t_base", "t_tail_base_right", 0)

    // M
    g.AddEdge("M_base_left", "M_left", 0)
    g.AddEdge("M_left", "M_top_left", 0)
    g.AddEdge("M_top_left", "M_center", 0)
    g.AddEdge("M_center", "M_top_right", 0)
    g.AddEdge("M_top_right", "M_right", 0)
    g.AddEdge("M_right", "M_base_right", 0)

    // h
    g.AddEdge("h_top", "h_middle", 0)
    g.AddEdge("h_middle", "h_base", 0)
    g.AddEdge("h_middle", "h_right", 0)
    g.AddEdge("h_right", "h_base_right", 0)
  
    // 1) Components (expect 6)
    comps := findComponents(g)
    fmt.Println("components:", len(comps))

    // 2) Cyclomatic number = |E| - |V| + #components (expect 1 cycle in 'a')
    cycles := g.EdgeCount() - g.VertexCount() + len(comps)
    fmt.Println("cycles:", cycles)

    // 3) Tiny transformation: merge l-v-l into an M
    //    Remove the little tails and bridge the tops via v_center.
    _ = g.RemoveVertex("l1_tail_base_right")
    _ = g.RemoveVertex("l2_tail_base_right")
    _ = g.RemoveVertex("v_left")
    _ = g.RemoveVertex("v_right")
    g.AddEdge("l1_top", "v_center", 0)
    g.AddEdge("l2_top", "v_center", 0)

    // 4) Matrix view of the 'a' letter (induced subgraph):
    //    Construct an induced subgraph and build an adjacency matrix.
    aSet := map[string]bool{
        "a_tail_top_left": true,
        "a_top": true,
        "a_circle_middle": true,
        "a_circle_base": true,
        "a_circle_base_left": true,
        "a_circle_middle_left": true,
    }
    sub := g.CloneEmpty()
    for v := range aSet { _ = sub.AddVertex(v) }
    for _, e := range g.Edges() {
        if aSet[e.From] && aSet[e.To] { sub.AddEdge(e.From, e.To, 0) }
    }
    am := matrix.NewAdjacencyMatrix(sub) // see docs/matrix.md
    deg := matrix.DegreeVector(am)
    fmt.Println("a-degree-vector:", deg)
}
```

**What this proves**

* **Components**: six disconnected shapes are discovered deterministically.
* **Cycle accounting**: the single cycle in `'a'` is captured by `|E|-|V|+#comp`.
* **Transformations**: small edits to vertices/edges are safe and predictable.
* **Matrix integration**: you can take induced views and compute degree/eigen info.

> See `docs/BFS_&_DFS.md` and `docs/MATRICES.md` for deeper walkthroughs.

---

### B) Graph **“Hexagram”** — weighted example across packages

ASCII sketch:

```
                               [A]
                              / | \
                  (C─H:9)   3/  |  \4   (D─F:7)
        (B─G:7)         \   /   |   \   /         (E─G:9)
             [B]────3────[C]──5─┼────[D]────3────[E]
                \       / | \   |7  / | \       /
                6\    7/  | 5\  |  /4 |  \6    /7
                  \   /   |   \ | /   |   \   /
                   [F]──3─┼────[G]──5─┼────[H]
                  /   \   |9  / | \   |8  /   \
                2/    6\  |  /4 |  \6 |  /5    \6
                /       \ | /   |   \ | /       \
             [I]────5────[J]──8─┼────[K]────1────[L]
        (I─G:8)         /   \   |8  /   \          (L─G:7)
                  (J─H:7)   2\  |  /3   (K─F:8)
                              \ | /
                               [M]
```
We’ll build the weighted, undirected “Hexagram” graph and run BFS layers (on an unweighted view), Dijkstra, MST, and set up a symmetric TSP distance matrix.

```go
package main

import (
    "fmt"
    "sort"

    "github.com/katalvlaran/lvlath/bfs"
    "github.com/katalvlaran/lvlath/core"
    "github.com/katalvlaran/lvlath/dijkstra"
    "github.com/katalvlaran/lvlath/prim_kruskal"
    "github.com/katalvlaran/lvlath/tsp"
)

func main() {
    // Weighted, undirected
    g := core.NewGraph(core.WithWeighted())
    edges := []struct{ u, v string; w int64 }{
        {"A","C",3},{"A","G",7},{"A","D",4},
        {"B","C",3},{"B","G",7},{"B","F",6},
        {"C","F",7},{"C","J",9},{"C","G",5},{"C","H",9},{"C","D",5},
        {"D","F",7},{"D","G",4},{"D","K",8},{"D","H",6},{"D","E",3},
        {"E","G",9},{"E","H",7},
        {"F","G",3},{"F","K",8},{"F","J",6},{"F","I",2},
        {"H","G",5},{"H","L",6},{"H","K",4},{"H","J",7},
        {"I","G",8},{"I","J",5},
        {"J","G",4},{"J","K",8},{"J","M",2},
        {"K","G",6},{"K","L",1},{"K","M",3},
        {"L","G",7},
        {"M","G",8},
    }
    for _, e := range edges { g.AddEdge(e.u, e.v, e.w) }

    // 1) BFS neighbor layers from A (make an unweighted view)
    un := g.CloneEmpty() // same flags; we want unweighted, so use zero weights
    for _, v := range g.Vertices() { _ = un.AddVertex(v) }
    for _, e := range g.Edges() { un.AddEdge(e.From, e.To, 0) }

    res, err := bfs.BFS(un, "A")
    if err != nil { panic(err) }
    n1, n2 := []string{}, []string{}
    for v, d := range res.Depth {
        if d == 1 { n1 = append(n1, v) }
        if d == 2 { n2 = append(n2, v) }
    }
    sort.Strings(n1); sort.Strings(n2)
    fmt.Println("N1(A):", n1)
    fmt.Println("N2(A):", n2)

    // 2) Shortest path (Dijkstra) from I
    dist, prev, err := dijkstra.Dijkstra(
        g,
        dijkstra.Source("I"),
        dijkstra.WithReturnPath(),
    )
    if err != nil { panic(err) }
    fmt.Println("I→L distance:", dist["L"], " parent[L]=", prev["L"]) // rebuild as needed

    // 3) Minimum Spanning Tree (Kruskal)
    _, w, err := prim_kruskal.Kruskal(g)
    if err != nil { panic(err) }
    fmt.Println("MST total weight:", w)

    // 4) Symmetric TSP setup via metric closure (pairwise Dijkstra)
    //    Build a dense metric distance matrix on the vertex order below.
    order := []string{"A","B","C","D","E","F","G","H","I","J","K","L","M"}
    n := len(order)
    distMat := make([][]float64, n)
    for i := 0; i < n; i++ {
        // run Dijkstra from order[i]
        di, _, _ := dijkstra.Dijkstra(g, dijkstra.Source(order[i]))
        distMat[i] = make([]float64, n)
        for j := 0; j < n; j++ { distMat[i][j] = float64(di[order[j]]) }
    }

    // Solve a symmetric TSP using Christofides (tour + cost)
    opts := tsp.DefaultOptions()
    opts.Symmetric = true
    // Choose algorithm explicitly for clarity:
    ts, err := tsp.Christofides(distMat, opts)
    if err != nil { panic(err) }
    fmt.Println("TSP tour length:", ts.Cost, " nodes:", len(ts.Tour))
}
```

**Why this matters**

* **BFS**: shows how to derive exact neighbor shells via `Depth` without special helpers.
* **Dijkstra**: routes reliably on non‑negative weights; parent links are deterministic.
* **MST**: Kruskal provides a stable total weight; Prim is also available if you prefer a rooted growth.
* **TSP**: computing a metric closure yields a proper symmetric distance matrix for Christofides/LS.

---

## Design tenets (expectations you can rely on)

* **Deterministic by default** — no global state, no implicit RNG. If a randomized enumeration helps (e.g., local search), it’s seed‑controlled and optional.
* **Strict contracts** — inputs are validated early; packages share sentinel errors for testable `errors.Is` checks.
* **Composability** — packages stay small and focused; you can adopt one or many.
* **Bench‑friendly** — fixtures avoid timer pollution, and benchmarks reflect realistic workloads.
* **Numerical discipline** — explicit epsilon policies (per package) and stabilized comparisons to avoid cross‑platform drift.
* **Pragmatic scope** — dependable baselines over experimental breadth; clarity beats cleverness.

---

## Choosing algorithms (cheat‑sheet)

| Task                         | Start with     | Notes                                                                 |
| ---------------------------- | -------------- | --------------------------------------------------------------------- |
| Reachability, layers         | `bfs`          | Hooks for visit/enqueue; easy components via repeated BFS.            |
| Ordered traversals           | `dfs`          | Pre/post hooks for classification; cycle detection & topo sort.       |
| Shortest path (non‑negative) | `dijkstra`     | Deterministic distances & parents; for negatives, use another algo.   |
| Minimum spanning tree        | `prim_kruskal` | Prim O(E log V); Kruskal ≈ O(E log V). Deterministic tie‑breaking.    |
| Max‑flow / min‑cut           | `flow`         | Edmonds–Karp (clarity) or Dinic (speed).                              |
| Time‑series alignment        | `dtw`          | O(N·M); Sakoe–Chiba windows for long signals; memory modes available. |
| Symmetric TSP                | `tsp`          | Christofides + 2/3‑opt; exact BnB for small N with 1‑tree bounds.     |
| Asymmetric TSP               | `tsp`          | 2/3‑opt\* in directed mode; Christofides is symmetric‑only.           |

---

## Performance notes (with practical limits)

* **BFS/DFS**: O(V+E); hook overhead is O(1)/event and deterministic.
* **Dijkstra**: O((V+E) log V); requires non‑negative weights; supports mixed edges.
* **MST**: Prim O(E log V) (good with `matrix` on dense), Kruskal ≈ O(E log V) (great on sparse).
* **Flow**: Edmonds–Karp O(VE²) but simple; Dinic is usually much faster; capacities finite & non‑negative.
* **DTW**: O(N·M); memory modes (`None`, `TwoRows`, `FullMatrix`) trade memory for features.
* **TSP**: assumes triangle inequality when using Christofides; Blossom is **not** included (greedy matching fallback). Exact BnB is for small N.

---

## Documentation

* **Start here**: `docs/TUTORIAL.md` — end‑to‑end tour and selection matrix.
* **Per‑package contracts**: `{package}/doc.go` — formal API, options, edge cases.
* **Backgrounders**: `docs/{ALGORITHM}.md` — concise “What/Why/When” with formulas and diagrams.

API reference: **[pkg.go.dev › lvlath](https://pkg.go.dev/github.com/katalvlaran/lvlath)**

---

## FAQ

* **Does lvlath depend on cgo or external libraries?** — No. Pure Go.
* **Are results deterministic?** — Yes, unless you opt into seeded local search, in which case results are deterministic per seed.
* **Can I plug in my own matrix or graph types?** — Yes. `matrix` is deliberately minimal; `core` is designed for interop.

---

## Support & Contacts

* GitHub Issues: **katalvlaran/lvlath**
* Email: **[katalvlaran@gmail.com](mailto:katalvlaran@gmail.com)**

---

*© 2025 katalvlaran — MIT License*

*Happy graphing!*
