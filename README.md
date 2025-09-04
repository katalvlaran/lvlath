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

* **`core`** — A focused, thread‑safe set of graph primitives with the options most people actually need: directed/undirected, weighted/unweighted, multi‑edges and self‑loops, safe add/remove/clone, and predictable iteration order. It exposes shared sentinel errors (`ErrNonSquare`, `ErrAsymmetry`, `ErrTimeLimit`, …) so tests can assert using `errors.Is`.

  * **Plays well with `bfs`/`dfs`**: those packages operate directly on `core` graphs and offer hookable events (visit/enqueue/edge) for tracing, early stopping, or instrumentation—all without global state.
* **`bfs`** — Breadth‑first search done right: minimal API, cancellation & hooks, stable layering (same input ⇒ same order). It’s a reliable foundation for reachability, components, and shortest unweighted paths.
* **`dfs`** — Depth‑first traversal with explicit pre/post hooks for ordering and classification. Simple, dependable, and great for learning or building lightweight analyses.
* **`dijkstra`** — Single‑source shortest paths with non‑negative weights, clear unreachable semantics, and deterministic parent trees. Perfect as a baseline for routing on `core` graphs.
* **`prim_kruskal`** — Minimum Spanning Trees with two pragmatic paths: Prim O(n²) for dense matrix inputs and Kruskal for sparse edge lists. Deterministic tie‑breaking keeps MST weights stable across runs.
* **`flow`** — Max‑flow / min‑cut via Edmonds–Karp (robust, simple) and Dinic (usually much faster). Clean separation of capacity graph, residual logic, and results; the API is predictable, not “magic”.
* **`dtw`** — Classic O(nm) Dynamic Time Warping with optional constraints (Sakoe–Chiba bands). Solid for aligning noisy sequences in DSP/ML pipelines.
* **`tsp`** — A practical toolbox: Christofides‑style approximation for symmetric metrics, 2‑opt/3‑opt local search, a 1‑tree lower bound, and an exact Branch‑and‑Bound for small n. Deterministic by default; seed‑controlled local search when you want it.
* **`matrix`** — A minimal, bounds‑checked matrix interface used by dense algorithms (Prim O(n²), TSP). Pluggable by design; convert from/to `core` easily and test algorithms in isolation.
* **`gridgraph`** — 2D lattices with 4/8‑neighborhoods, mask‑based obstacles, and weight helpers (L1/Euclidean). A natural companion for `bfs`, `dfs`, and `dijkstra`.
* **`builder`** — Deterministic generators (rings, grids, rippled circles, seeded random fixtures). Perfect for reproducible demos, benchmarks, and tests.

> Background “What/Why/When” lives in `docs/{ALGORITHM}.md`; formal contracts are in `{package}/doc.go`. A guided tour starts at `docs/TUTORIAL.md`.

---

## Installation

```bash
go get github.com/katalvlaran/lvlath@latest
```

*Requires Go ≥ 1.23. No CGO, no external dependencies. Works wherever Go works.*

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

* Use packages independently or combine them.
* `matrix` lets dense algorithms operate without depending on `core` internals.
* `builder` and `gridgraph` are ideal for quick prototypes and reproducible test fixtures.

---

## Quick Start (practical)

> Code below is API‑shaped to lvlath; see each `{package}/doc.go` for exact signatures.

### 1) Graph **“lvlath”** — shapes, components & a tiny transformation

We’ll build six disconnected letter‑shapes and run a few analyses.

```go
package main

import (
    "fmt"
    "github.com/katalvlaran/lvlath/bfs"
    "github.com/katalvlaran/lvlath/core"
    "github.com/katalvlaran/lvlath/matrix"
    "github.com/katalvlaran/lvlath/dtw"
)

func main() {
  // Weighted, undirected graph.
  g := core.NewGraph(core.WithWeighted())

  // l (left)
  g.AddEdge("l1_top", "l1_middle", 2)
  g.AddEdge("l1_middle", "l1_base", 2)
  g.AddEdge("l1_base", "l1_tail_base_right", 1)

  // v
  g.AddEdge("v_top_left", "v_centre", 1)
  g.AddEdge("v_centre", "v_top_right", 1)

  // l (right)
  g.AddEdge("l2_top", "l2_middle", 2)
  g.AddEdge("l2_middle", "l2_base", 2)
  g.AddEdge("l2_base", "l2_tail_base_right", 1)

  // a = ??
  g.AddEdge("a_top_left", "a_top", 1)
  g.AddEdge("a_top", "a_circle_middle", 2)
  g.AddEdge("a_circle_middle", "a_circle_base", 2)
  g.AddEdge("a_circle_base", "a_circle_base_left", 2)
  g.AddEdge("a_circle_base_left", "a_circle_middle_left", 2)
  g.AddEdge("a_circle_middle_left", "a_circle_middle", 2)
  g.AddEdge("a_circle_base", "a_tail_base_right", 1)

  // t = ??
  g.AddEdge("t_tail_top", "t_cross", 1)
  g.AddEdge("t_tail_left", "t_cross", 1)
  g.AddEdge("t_tail_right", "t_cross", 1)
  g.AddEdge("t_base", "t_cross", 2)
  g.AddEdge("t_base", "t_tail_base_right", 1)

  // h = ??
  g.AddEdge("h_top", "h_middle", 2)
  g.AddEdge("h_middle", "h_base", 2)
  g.AddEdge("h_middle", "h_right", 2)
  g.AddEdge("h_right", "h_base_right", 2)

  // 1) Components (should be six: l, v, l, a, t, h)
  //comps := bfs.Components(g)
  //fmt.Println("components:", len(comps))

  // 2) Cyclomatic number = |E| - |V| + #components (one cycle in 'a').
  //cycles := g.EdgeCount() - g.VertexCount() + len(comps)
  //fmt.Println("cycles:", cycles)

  // 3) Small transformation: turn "l","v","l" into a single "M"
  // Bridge l1_top ↔ v_centre ↔ l2_top, and add the inner peak.
  g.RemoveVertex("l1_tail_base_right")
  g.RemoveVertex("v_top_left")
  g.AddEdge("l1_top", "v_centre", 2)

  g.RemoveVertex("l2_tail_base_right")
  g.RemoveVertex("v_top_right")
  g.AddEdge("l2_top", "v_centre", 2)

  // 4) Matrix view of the subgraph induced by the letter 'a'
  //aNodes := []string{"a_tail", "a_circle1", "a_circle2", "a_circle3"}
  //A := matrix.FromGraphInduced(g, aNodes)
  //fmt.Println("a-subgraph size:", A.N())

  // 5) (Optional) DTW on a simple structural signature:
  // degree sequence along a BFS from a_tail vs canonical [1,2,2,2]
  //deg := []float64{1, 2, 2, 2}
  //cost := dtw.Distance(deg, deg) // identical here; replace with measured signature
  //fmt.Println("dtw(cost for 'a' vs canonical):", cost)
}
```

**What this demonstrates**

* Components: the six disconnected letter‑shapes are discovered reliably.
* Cycles: the sole cycle in `'a'` is captured by the cyclomatic number.
* Transformations: you can patch shapes (here, merge `l‑v‑l` into an `M`) by adding/removing edges deterministically.
* Matrix/DTW: dense views and simple signatures allow structural comparisons.

---

### 2) Graph **“Hexagram”** — weighted example used across packages

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

Build it (weighted, undirected), then exercise algorithms:

```go
package main

import (
    "fmt"
    "github.com/katalvlaran/lvlath/bfs"
    "github.com/katalvlaran/lvlath/core"
    "github.com/katalvlaran/lvlath/dijkstra"
    "github.com/katalvlaran/lvlath/prim_kruskal"
    "github.com/katalvlaran/lvlath/tsp"
)

func main() {
    g := core.NewGraph(core.WithWeighted())
    edges := []struct{u, v string; w int64}{
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

    // 1) Neighbor layers (BFS) from A
    //layers, _ := bfs.Layers(g, "A", 2) // radius 2
    //fmt.Println("N1:", layers[1])
    //fmt.Println("N2:", layers[2])

    // 2) Shortest path (Dijkstra) from I to L
    //dist, parent := dijkstra.Run(g, "I")
    //fmt.Println("I→L distance:", dist["L"], " parent[L]=", parent["L"])

    // 3) Minimum Spanning Tree (Prim for dense – via implicit matrix)
    //w := prim_kruskal.MSTWeight(g) // or prim_kruskal.Prim(g).Weight
    //fmt.Println("MST total weight:", w)

    // 4) Symmetric TSP tour (Christofides + 2‑opt polish)
    //tour, cost := tsp.Auto(g, tsp.Options{Symmetric:true, Polish2Opt:true})
    //fmt.Println("TSP(|V|=", len(tour), ") cost:", cost)
}
```

**What this demonstrates**

* **BFS layers** make “neighbors at depth r” trivial.
* **Dijkstra** produces predictable distances/parents (no negative weights).
* **MST** uses deterministic tie‑breaking; weights are stable across runs.
* **TSP** auto‑selects a symmetric pipeline and returns a reproducible tour.

---

## Results & sanity checks (things worth asserting)

* **Determinism**: identical inputs/options ⇒ identical outputs, across runs and platforms.
* **BFS/DFS**: layer/order stability with the same graph, hooks fire in a defined order.
* **Dijkstra**: non‑negative weight validation; unreachable vertices remain explicit.
* **MST**: total weight matches across Prim/Kruskal on equivalent inputs; tie‑breaks are deterministic.
* **Flow**: value == capacity of min‑cut; residual graph invariants hold.
* **DTW**: boundary conditions (start/end costs) and window constraints behave as documented.
* **TSP**: for symmetric inputs, Christofides never violates triangle inequality; Branch‑and‑Bound prunes using the same 1‑tree lower bound for a given seed.

---

## Design tenets (what to expect from every package)

* **Deterministic by default** — no global state, no implicit RNG. If randomness helps (e.g., local search), it’s seeded and optional.
* **Strict contracts** — early validation + shared sentinel errors you can match with `errors.Is`.
* **Composable** — packages are small and focused; use one or many.
* **Bench‑friendly** — builders/fixtures avoid timer pollution; examples use realistic scales.
* **Numerical discipline** — explicit epsilon policy (documented per package) and stabilized comparisons in tests.
* **Pragmatic scope** — dependable baselines over experimental breadth; clarity beats cleverness.

---

## Choosing algorithms (cheat‑sheet)

| Task                         | Start with     | Notes                                                                  |
| ---------------------------- | -------------- | ---------------------------------------------------------------------- |
| Reachability, layers         | `bfs`          | Hooks for visit/enqueue; easy components and frontier capture.         |
| Ordered traversals           | `dfs`          | Pre/post hooks for classification; good for teaching & utilities.      |
| Shortest path (non‑negative) | `dijkstra`     | Deterministic parents; for negatives, use another algo (not provided). |
| Minimum spanning tree        | `prim_kruskal` | Prim O(n²) for dense matrix views; Kruskal for sparse lists.           |
| Max‑flow / min‑cut           | `flow`         | Edmonds–Karp (clarity); Dinic (speed).                                 |
| Time‑series alignment        | `dtw`          | O(nm); constrain with Sakoe–Chiba windows for long signals.            |
| Symmetric TSP                | `tsp`          | Christofides + 2/3‑opt; exact BnB for small n with 1‑tree bounds.      |
| Asymmetric TSP               | `tsp`          | Use 2/3‑opt in directed mode; Christofides is symmetric‑only.          |

---

## Performance notes & practical limits

* **BFS/DFS**: linear in edges+vertices; hook overhead is O(1) per event and deterministic.
* **Dijkstra**: requires non‑negative weights; parent trees and distances are stable; choose adjacency‑list vs matrix based on density.
* **MST**: Prim O(n²) pairs naturally with `matrix` for dense graphs; Kruskal is preferable for sparse edge lists; tie‑breaks are deterministic.
* **Flow**: Edmonds–Karp is O(VE²) but simple; Dinic is typically far faster on medium/large instances; capacities are finite and non‑negative.
* **DTW**: O(nm) time/space; windowing greatly reduces cost; memory‑optimized variants are not included.
* **TSP**: symmetric pipeline assumes triangle inequality; exact minimum‑weight matching (Blossom) is not included—Christofides uses greedy matching with documented behavior. Exact BnB is intended for small n.

---

## Documentation

* **Start here**: `docs/TUTORIAL.md` — end‑to‑end tour, selection matrix, determinism & numeric guidance.
* **Per‑package contracts**: `{package}/doc.go` — formal API, options, edge cases.
* **Backgrounders**: `docs/{ALGORITHM}.md` — concise “What/Why/When” with formulas and diagrams.

API reference: **[pkg.go.dev › lvlath](https://pkg.go.dev/github.com/katalvlaran/lvlath)**

---

## FAQ

* *Does lvlath depend on cgo or external libraries?* — No. It’s pure Go.
* *Are results deterministic?* — Yes, unless you opt into seeded local search; then they’re deterministic per seed.
* *Can I plug in my own matrix or graph types?* — Yes; `matrix` is deliberately minimal and `core` aims to be interoperable.

---

## Support & Contacts

* GitHub Issues: **katalvlaran/lvlath**
* Email: **[katalvlaran@gmail.com](mailto:katalvlaran@gmail.com)**

---

*© 2025 katalvlaran — MIT License*

*Happy graphing!*
