<!--
  SPDX-License-Identifier: AGPL-3.0-only
  Copyright (C) 2025-2026 katalvlaran

  lvlath - Repository Tutorial

  Purpose:
    This tutorial is the first educational entry point for lvlath. It introduces
    fundamental graph and matrix ideas, explains how lvlath maps mathematical
    algorithms into deterministic Go packages, and helps readers move from basic
    graph concepts to package-specific documentation. It does not duplicate every
    package specification; instead it teaches the mental model, core entities,
    options, result surfaces, and engineering discipline needed before reading
    docs/{PACKAGE}.md and {package}/doc.go files.

  License:
    lvlath is licensed under AGPL-3.0-only. See ../LICENSE.
-->

# lvlath Tutorial

> **hard math on an easy level**

Welcome to **lvlath**, a deterministic Go library for graph algorithms, dense graph algebra, flows, routing, tours, fixtures, and time-series alignment.

This tutorial is the beginning of the learning path. It introduces the objects and ideas that most packages share: graphs, matrices, options, result artifacts, error contracts, and deterministic algorithm execution.

---

## Table of Contents

1. [Introduction](#1-introduction)
2. [Documentation Map](#2-documentation-map)
3. [Installation](#3-installation)
4. [Mental Model: How lvlath Is Built](#4-mental-model-how-lvlath-is-built)
5. [Graph Foundations](#5-graph-foundations)
6. [First Rich Graph: Build, Inspect, Traverse](#6-first-rich-graph-build-inspect-traverse)
7. [Graph Algorithms on the Same Topology](#7-graph-algorithms-on-the-same-topology)
8. [Matrix Foundations](#8-matrix-foundations)
9. [Graph-to-Matrix Example: Routing, Closure, and Spectral-Ready Data](#9-graph-to-matrix-example-routing-closure-and-spectral-ready-data)
10. [Algorithm Selection Guide](#10-algorithm-selection-guide)
11. [Professional Best Practices](#11-professional-best-practices)
12. [Where To Go Next](#12-where-to-go-next)

---

## 1. Introduction

### What is lvlath?

`lvlath` is a Go library for algorithmic graph and matrix workflows:

* graph modeling with deterministic topology;
* BFS and DFS traversal with result contracts;
* Dijkstra weighted routing;
* minimum spanning tree backbones;
* maximum-flow / residual-network analysis;
* dense matrix algebra and graph adapters;
* dynamic time warping;
* grid graphs;
* TSP-style tour optimization;
* deterministic builders for examples, tests, and benchmarks.

The library is intentionally small-package oriented. Each algorithm package has its own contract, but all of them depend on the same engineering philosophy.

~~~text
Mathematics       Go implementation         lvlath contract
────────────────────────────────────────────────────────────────
G = (V, E)   ->   *core.Graph          ->   deterministic topology
BFS layers   ->   *bfs.BFSResult       ->   Order/Depth/Parent/Visited
DFS forest   ->   *dfs.DFSResult       ->   post-order + safe partial metadata
shortest path->   *dijkstra.Result     ->   distance, witness, +Inf semantics
matrix A     ->   *matrix.Dense        ->   row-major storage + numeric policy
flow network ->   residual *core.Graph ->   capacity state after augmentation
~~~

### Why lvlath?

Because real algorithm engineering needs more than a correct formula.

A one-page BFS snippet can work in a blog post but still fail in production when:

* traversal order changes between runs;
* cancellation loses useful state;
* weighted and unweighted questions are mixed;
* graph mutation violates topology assumptions;
* errors cannot be matched safely;
* numeric `NaN` or `+Inf` destroys downstream matrix statistics;
* examples do not reveal the contract users must rely on.

`lvlath` treats those details as first-class API design.

### What makes lvlath different?

1. **Deterministic graph surface** — stable `Vertices`, `Edges`, and `Neighbors` are the root of reproducible algorithms.
2. **Explicit options** — directedness, weights, loops, multi-edges, matrix policies, path tracking, thresholds, and memory modes are not hidden defaults.
3. **Result artifacts** — algorithm outputs are meaningful objects, not temporary implementation debris.
4. **Mathematical semantics preserved in Go** — `+Inf` means unreachable/absence only under explicit policies; DFS order means finish order; BFS depth means hop distance.
5. **Documentation split by responsibility** — package `doc.go` explains what is implemented; `docs/{PACKAGE}.md` teaches the math, diagrams, proofs, pitfalls, and examples.

---

## 2. Documentation Map

Read these documents as a learning sequence, not as isolated pages.

| Step | Document                                         | What you learn                                                                             |
|:-----|:-------------------------------------------------|:-------------------------------------------------------------------------------------------|
| 1    | [`CORE.md`](CORE.md)                             | Graph model, deterministic enumeration, capabilities, lock hierarchy, topology operations. |
| 2    | [`BFS.md`](BFS.md)                               | Hop-distance traversal, BFS result fields, components, partial results.                    |
| 3    | [`DFS.md`](DFS.md)                               | DFS finish order, cycle witnesses, topological sorting, partial-result safety.             |
| 4    | [`DIJKSTRA.md`](DIJKSTRA.md)                     | Weighted routing, strict improvement, `+Inf`, path tracking, runtime policy gates.         |
| 5    | [`PRIM_&_KRUSKAL.md`](PRIM_%26_KRUSKAL.md)       | Minimum spanning trees, cut/cycle properties, Prim and Kruskal.                            |
| 6    | [`FLOW.md`](FLOW.md)                             | Flow networks, residual capacity, Ford-Fulkerson, Edmonds-Karp, Dinic.                     |
| 7    | [`DTW.md`](DTW.md)                               | Dynamic Time Warping, windows, slope penalties, memory modes, path recovery.               |
| 8    | [`GRID_GRAPH.md`](GRID_GRAPH.md)                 | Grid/lattice topology for pathfinding and demos.                                           |
| 9    | [`MATRICES.md`](MATRICES.md)                     | Dense matrix storage, graph adapters, metric closure, statistics, numeric policy.          |
| 10   | [`TRAVELING_SALESMAN.md`](TRAVELING_SALESMAN.md) | Tour optimization, exactness vs approximation, metric assumptions.                         |
| 11   | [`lvlath_UES.md`](lvlath_UES.md)                 | Repository engineering standard.                                                           |
| 12   | [`FAQ_&_TIPS.md`](FAQ_%26_TIPS.md)               | Pitfalls, troubleshooting, and usage patterns.                                             |
| 13   | [`../CONTRIBUTING.md`](../CONTRIBUTING.md)       | Tests, linting, coverage, branches, PR discipline.                                         |

---

## 3. Installation

~~~bash
go get github.com/katalvlaran/lvlath@latest
~~~

Requires Go 1.23 or newer. Pure Go. No cgo. No external runtime dependencies.

---

## 4. Mental Model: How lvlath Is Built

Think of `lvlath` as a layered system.

~~~text
                         ┌─────────────────────────────┐
                         │          core.Graph         │
                         │ deterministic graph storage │
                         └──────────────┬──────────────┘
                                        │
              ┌─────────────────────────┼──────────────────────────┐
              │                         │                          │
       traversal layer             weighted layer             algebra layer
              │                         │                          │
          bfs / dfs           dijkstra / mst / flow              matrix
              │                         │                          │
              │                         │                          │
              └────── examples, docs, tests, builder, gridgraph ───┘

            dtw and tsp live as algorithm packages with the same contract style:
            explicit options, deterministic behavior, documented complexity.
~~~

### Repository laws

1. **Graph capabilities are explicit**

   A graph does not accidentally become directed, weighted, multi-edge, or loop-enabled. You configure the capability before using it.

2. **Traversal result fields are not interchangeable**

   `Visited`, `Order`, `Parent`, `Depth`, and `Skipped` mean different things. The tutorial will keep those meanings visible.

3. **Weighted and unweighted questions are different math**

   BFS minimizes hop count. Dijkstra minimizes non-negative total cost. Flow maximizes feasible throughput. MST minimizes connected backbone cost.

4. **Matrix cells require context**

   A `0` in a normal adjacency matrix may mean “no edge.” A finite `0` in zero-preserving weighted mode may mean a real zero-cost edge. `+Inf` can mean absence or unreachable only under the appropriate policy.

5. **Policy should be run-local**

   Use filters, thresholds, cutoffs, and options instead of mutating graph topology to answer one operational question.

---

## 5. Graph Foundations

A graph is:

$$ G = (V, E) $$

where:

* `V` is a set of vertices;
* `E` is a set of edges between vertices.

In `lvlath/core`, vertices are identified by string IDs, and edges carry direction, weight, ID, and policy-dependent behavior.

### Core capabilities

| Capability           | What it controls                           | Typical use                                              |
|:---------------------|:-------------------------------------------|:---------------------------------------------------------|
| `WithDirected(true)` | Default orientation of edges.              | Dependency graphs, one-way roads, call graphs.           |
| `WithWeighted()`     | Non-zero `float64` edge values.            | Cost, distance, latency, capacity.                       |
| `WithLoops()`        | Self-edges such as `A -> A`.               | State machines, roundabouts, internal transitions.       |
| `WithMultiEdges()`   | Parallel edges between the same endpoints. | Road + rail, multiple network links, redundant channels. |
| `WithMixedEdges()`   | Per-edge direction overrides.              | City maps with one-way streets and two-way roads.        |

### Storage intuition

~~~text
+------------------------------------------------------------------+
| core.Graph                                                       |
|------------------------------------------------------------------|
| vertices:       map[string]*Vertex                               |
| edges:          map[string]*Edge                                 |
| adjacency:      map[fromID]map[toID]map[edgeID]struct{}          |
| config:         directed / weighted / loops / multi / mixed      |
| locks:          vertex/config lock + edge/adjacency lock         |
+------------------------------------------------------------------+
            │                         │
            │                         └── edge IDs preserve identity
            │
            └── vertex IDs are the stable public coordinate system
~~~

### Directed vs undirected

~~~text
Directed edge A -> B:

   [A] ──▶ [B]

Storage intent:
   A can reach B through this edge.
   B does not automatically reach A.

Undirected edge A -- B:

   [A] ◀──▶ [B]

Storage intent:
   Traversal can move both ways.
~~~

### Weight is not just decoration

~~~text
Unweighted graph:
  all edges have weight 0;
  BFS/DFS are natural tools.

Weighted graph:
  edge values carry cost/capacity/distance;
  Dijkstra, MST, flow, matrix closure, and TSP become meaningful.
~~~

---

## 6. First Rich Graph: Build, Inspect, Traverse

The first example should be visually useful, not a toy line. We use a 10-vertex directed service graph with a central loop, two branches, and a storage tail.

~~~text
                         ┌────────────┐
                         │  incident  │
                         └──────┬─────┘
                                │
                                ▼
                         ┌────────────┐
                         │  gateway   │
                         └───┬────┬───┘
                             │    │
                ┌────────────┘    └──────────┐
                ▼                            ▼
            ┌────────┐                  ┌────────┐
            │  auth  │                  │  api   │
            └───┬────┘                  └────┬───┘
                │                            │
                ▼                            ▼
          ┌──────────┐                 ┌──────────┐
          │ profile  │◀──────┐         │ search   │
          └─────┬────┘       │         └─────┬────┘
                │            │               │
                ▼            │               ▼
           ┌────────┐        │         ┌──────────┐
           │   db   │        └─────────│  index   │
           └────┬───┘                  └─────┬────┘
                │                            │
                ▼                            ▼
           ┌────────┐                  ┌──────────┐
           │ backup │                  │  cache   │
           └────────┘                  └──────────┘

Features:
  - main dependency chain: incident -> gateway -> auth -> profile -> db -> backup
  - parallel branch: gateway -> api -> search -> index -> cache
  - cross-loop: index -> profile
  - 10 vertices, visible depth layers, one meaningful feedback-style structural join
~~~

~~~go
package main

import (
	"context"
	"fmt"
	"sort"

	"github.com/katalvlaran/lvlath/bfs"
	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/dfs"
)

func main() {
	g, err := core.NewGraph(core.WithDirected(true))
	if err != nil {
		fmt.Println(err)
		return
	}

	// Static tutorial data. Ignoring AddEdge errors is acceptable only here because
	// the topology is predetermined and the graph is unweighted. Check errors in
	// production and tests.
	_, _ = g.AddEdge("incident", "gateway", 0)
	_, _ = g.AddEdge("gateway", "auth", 0)
	_, _ = g.AddEdge("gateway", "api", 0)
	_, _ = g.AddEdge("auth", "profile", 0)
	_, _ = g.AddEdge("profile", "db", 0)
	_, _ = g.AddEdge("db", "backup", 0)
	_, _ = g.AddEdge("api", "search", 0)
	_, _ = g.AddEdge("search", "index", 0)
	_, _ = g.AddEdge("index", "cache", 0)
	_, _ = g.AddEdge("index", "profile", 0)

	// BFS: hop-distance layers from incident.
	bfsResult, err := bfs.BFS(g, "incident")
	if err != nil {
		fmt.Println(err)
		return
	}

	layers := map[int][]string{}
	for id, depth := range bfsResult.Depth {
		layers[depth] = append(layers[depth], id)
	}
	for depth := 0; depth <= 5; depth++ {
		sort.Strings(layers[depth])
		fmt.Printf("depth %d: %v\n", depth, layers[depth])
	}

	path, err := bfsResult.PathTo("backup")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("path-to-backup:", path)

	// DFS: finish order, useful for dependency reasoning.
	dfsResult, err := dfs.DFS(g, "incident")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("dfs-finish-order:", dfsResult.Order)

	// Components: direction ignored for weak membership.
	components, err := bfs.Components(context.Background(), g)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("weak-components:", components.Count)
}
~~~

### What to notice

* `BFS.Depth` is a shortest-hop certificate.
* `BFS.PathTo` reconstructs one deterministic shortest path from `StartID`.
* `DFS.Order` is finish order, not discovery order.
* `Components` answers weak connectivity and should not be confused with strongly connected components.

### Layer visualization

~~~text
Depth 0:   incident
              │
Depth 1:   gateway
           /      \
Depth 2: auth     api
          │        │
Depth 3: profile  search
          │        │
Depth 4: db       index
          │       /    \
Depth 5: backup  cache  profile(already discovered earlier)
~~~

---

## 7. Graph Algorithms on the Same Topology

The fastest way to misuse graph libraries is to treat every algorithm as answering the same question. In `lvlath`, each package is explicit about the mathematical objective.

### BFS: hop-distance and frontier waves

Use BFS when every edge has equal cost and you care about layers.

~~~text
BFS objective:
  minimize number of edges

Formula:
  dist(s, v) = min path length in hops
~~~

Good questions:

* “Which services are at most 2 hops from the incident?”
* “What is the first wave of dependencies?”
* “Which weak components exist?”

### DFS: structure, finish order, cycles

Use DFS when depth structure matters.

~~~text
DFS mental model:
  enter vertex -> explore deeper -> finish vertex -> append to Order

Returned DFSResult.Order = finish order.
~~~

Good questions:

* “What is a deterministic post-order traversal?”
* “Does a dependency cycle exist?”
* “What execution order is valid for this DAG?”

### Dijkstra: weighted non-negative routing

Use Dijkstra when edge weights are costs.

~~~text
Dijkstra objective:
  minimize sum of non-negative edge weights

Relaxation:
  candidate = dist[u] + weight(u, v)
  update only if candidate < dist[v]
~~~

Good questions:

* “What is the cheapest route from ingress to storage?”
* “Which targets are known but unreachable under a wall policy?”
* “What is one deterministic shortest-path witness?”

### MST: cheapest connected backbone

Use minimum spanning trees when you need to connect all vertices with minimum total cost, not route from one source.

~~~text
MST objective:
  choose |V|-1 edges, keep graph connected, minimize total weight
~~~

Good questions:

* “What is the cheapest cable/road/backbone layout?”
* “How do I cluster by removing heavy backbone edges?”

### Flow: maximum feasible throughput

Use max-flow when edge weights are capacities.

~~~text
Flow objective:
  maximize total value from source to sink

Constraints:
  0 <= f(u,v) <= c(u,v)
  inflow(v) = outflow(v), for v not source/sink
~~~

Good questions:

* “How much traffic can reach storage?”
* “Where is the bottleneck cut?”
* “What residual capacity remains after routing maximum flow?”

---

## 8. Matrix Foundations

Graphs describe relationships. Matrices let you compute with those relationships.

`lvlath/matrix` is the dense algebra layer for:

* adjacency matrices;
* incidence matrices;
* metric closure / all-pairs shortest paths;
* row/column reductions;
* matrix products and transposes;
* LU, QR, inverse, symmetric eigen decomposition;
* covariance and correlation;
* numerical sanitation and tolerant comparison.

### Matrix core capabilities

| Capability                 | What it controls                                                  | Why it matters                                            |
|:---------------------------|:------------------------------------------------------------------|:----------------------------------------------------------|
| `Dense` row-major storage  | Matrix stored as one flat `[]float64`.                            | Cache locality and predictable loop behavior.             |
| `Matrix` interface         | `Rows`, `Cols`, `At`, `Set`, `Clone`.                             | Safe generic API for kernels and adapters.                |
| `NewPreparedDense` options | Numeric admissibility, including `+Inf` policy.                   | Prevents dirty floats unless intentionally allowed.       |
| `View`                     | Shared no-copy window into a dense matrix.                        | Efficient block assembly; mutation writes through.        |
| `Induced`                  | Copied submatrix.                                                 | Independent lifetime and safe downstream transformations. |
| Graph adapters             | `NewAdjacencyMatrix`, `NewIncidenceMatrix`, `BuildMetricClosure`. | Converts graph topology into numeric artifacts.           |
| Sanitizers                 | `ReplaceInfNaN`, `Clip`.                                          | Hardens telemetry/statistics pipelines.                   |
| Comparators                | `AllClose`.                                                       | Stable numeric tests without fragile equality.            |

### Storage intuition

A `Dense` matrix with `R` rows and `C` columns stores entry `(i,j)` at:

$$ offset(i,j) = i \times C + j $$

~~~text
Logical matrix 3×4:

      c0   c1   c2   c3
r0  [ a00  a01  a02  a03 ]
r1  [ a10  a11  a12  a13 ]
r2  [ a20  a21  a22  a23 ]

Physical row-major slice:

[ a00 a01 a02 a03 | a10 a11 a12 a13 | a20 a21 a22 a23 ]
  row 0 contiguous   row 1 contiguous   row 2 contiguous
~~~

### Matrix options and semantics

~~~text
Directed vs undirected adjacency:

A -> B, weight 5

Directed:                         Undirected:
      A   B                              A   B
A   [ 0   5 ]                      A   [ 0   5 ]
B   [ 0   0 ]                      B   [ 5   0 ]

Zero-weight preservation:

A -> C, weight 0 is a real edge.
Classic adjacency cannot distinguish it from absence.
Zero-preserving weighted mode uses +Inf for absence.

Metric closure:

Adjacency says: “which direct edges exist?”
Metric closure says: “what is the shortest distance?”
Those are not the same object.
~~~

### Adjacency vs incidence

~~~text
Adjacency: V × V
  Best for: vertex-pair lookup, dense algebra, degree-like row operations.

Incidence: V × E
  Best for: edge identity, structural edge columns, flow/cut reasoning.
~~~

---

## 9. Graph-to-Matrix Example: Routing, Closure, and Spectral-Ready Data

This example uses a realistic 9-vertex infrastructure graph. It preserves a zero-weight replication link and prepares a symmetric operator for spectral analysis.

~~~text

          ┌─ api ◀───3──── edge ────2────▶ auth ─┐                                      
          │                 │                    │
          │                 │                    │1
          ├─2──▶ cache ◀──6─┘                    ▼  
          │                                   profile
          │4                                     │
          ▼                                      │2   
        search ───3──▶ logs─┐                    ▼    
          │                 │                   db
          │1                │                    │
          ▼                 └──4──▶ backup ◀──0──┘     
        index                                   
          │             
          └─────5─────▶ archive                                      
Meaning:
  - `db -> backup = 0` is a real zero-cost replication edge.
  - routing uses weights as cost/latency.
  - matrix mode must preserve zero-weight edges.
~~~

~~~go
package main

import (
	"fmt"

	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/matrix"
)

func main() {
	g, err := core.NewGraph(core.WithDirected(true), core.WithWeighted())
	if err != nil {
		fmt.Println(err)
		return
	}

	// Static tutorial data. Check AddEdge errors in production.
	_, _ = g.AddEdge("edge", "auth", 2)
	_, _ = g.AddEdge("edge", "api", 3)
	_, _ = g.AddEdge("edge", "cache", 6)
	_, _ = g.AddEdge("auth", "profile", 1)
	_, _ = g.AddEdge("api", "search", 4)
	_, _ = g.AddEdge("api", "cache", 2)
	_, _ = g.AddEdge("search", "index", 1)
	_, _ = g.AddEdge("index", "archive", 5)
	_, _ = g.AddEdge("search", "logs", 3)
	_, _ = g.AddEdge("logs", "backup", 2)
	_, _ = g.AddEdge("profile", "db", 2)
	_, _ = g.AddEdge("db", "backup", 0)

	opts, err := matrix.NewMatrixOptions(
		matrix.WithDirected(),
		matrix.WithWeighted(),
		matrix.WithPreserveZeroWeights(),
	)
	if err != nil {
		fmt.Println(err)
		return
	}

	// NewAdjacencyMatrix maps string vertex IDs to deterministic integer coordinates.
	adj, err := matrix.NewAdjacencyMatrix(g, opts)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("vertices:", len(adj.VertexIndex))

	// DegreeVector summarizes outgoing weighted structure by row-like aggregation.
	degree, err := adj.DegreeVector()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("degree-vector-len:", len(degree))

	// Zero-weight preservation check.
	db := adj.VertexIndex["db"]
	backup := adj.VertexIndex["backup"]
	zeroReplication, err := adj.Mat.At(db, backup)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("zero-replication:", zeroReplication)

	// Metric closure converts direct edges to shortest-path distances.
	closure, err := matrix.BuildMetricClosure(g, opts)
	if err != nil {
		fmt.Println(err)
		return
	}

	edge := adj.VertexIndex["edge"]
	dist, err := closure.Mat.At(edge, backup)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("distance-edge-backup:", dist)

	// Symmetrize prepares an operator suitable for symmetric spectral routines.
	sym, err := matrix.Symmetrize(adj.Mat)
	if err != nil {
		fmt.Println(err)
		return
	}
	_, _, err = matrix.EigenSym(sym, 1e-10, 200)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("spectral-ready:", true)
}
~~~

### What to notice

* `NewMatrixOptions` is the policy boundary; build it once and reuse it.
* `WithPreserveZeroWeights` prevents a real zero-weight edge from being confused with absence.
* `BuildMetricClosure` produces distances, not original topology.
* `Symmetrize` is a deliberate preparation step before symmetric eigen routines.

---

## 10. Algorithm Selection Guide

### If the input is a graph

| Problem                            | Package        | Main output                                                  |
|:-----------------------------------|:---------------|:-------------------------------------------------------------|
| Reachability by hop count          | `bfs`          | `BFSResult` with `Order`, `Depth`, `Parent`, `Visited`.      |
| Weak components                    | `bfs`          | `ComponentsResult`.                                          |
| Deep structural traversal          | `dfs`          | `DFSResult` with finish order.                               |
| Cycle witness                      | `dfs`          | `CycleDetectionResult`.                                      |
| Topological execution order        | `dfs`          | `[]string` order or cycle error.                             |
| Weighted route from one source     | `dijkstra`     | `DijkstraResult` with distances and optional path witnesses. |
| Cheapest connected backbone        | `prim_kruskal` | MST edge set and total weight.                               |
| Max throughput from source to sink | `flow`         | max flow value and residual graph.                           |
| Grid/lattice topology              | `gridgraph`    | generated `core.Graph`-style structures.                     |
| Dense topology features            | `matrix`       | adjacency, incidence, degree, closure.                       |
| Tour visiting all vertices         | `tsp`          | tour and cost.                                               |

### If the input is numeric data

| Problem                 | Package  | Main output                          |
|:------------------------|:---------|:-------------------------------------|
| Matrix algebra          | `matrix` | dense matrix result.                 |
| Numeric cleaning        | `matrix` | sanitized matrix.                    |
| Feature statistics      | `matrix` | covariance/correlation and metadata. |
| Sequence alignment      | `dtw`    | distance and optional warping path.  |
| Distance matrix routing | `tsp`    | tour and cost.                       |

### If the goal is documentation or testing

| Goal                          | Package/file        |
|:------------------------------|:--------------------|
| Reproducible generated graphs | `builder`           |
| Grid examples                 | `gridgraph`         |
| Package contracts             | `{package}/doc.go`  |
| Theory and diagrams           | `docs/{PACKAGE}.md` |
| Contribution rules            | `CONTRIBUTING.md`   |

---

## 11. Professional Best Practices

### 11.1. Decide the mathematical objective before choosing the package

Do not begin with “which function should I call?” Begin with “what quantity am I minimizing/maximizing/classifying?”

~~~text
min hops           -> BFS
min weighted cost  -> Dijkstra
min backbone cost  -> MST
max throughput     -> Flow
all-pairs distance -> Matrix metric closure
sequence similarity with timing drift -> DTW
closed tour cost   -> TSP
~~~

### 11.2. Treat graph options as domain modeling, not syntax

A directed weighted graph is not a “more advanced” graph than an undirected unweighted graph. It is a different model. Use the simplest capability set that honestly represents the domain.

### 11.3. Keep topology stable during algorithm execution

`core` protects graph storage, but algorithm meaning still assumes the topology is not being changed underneath the run. If reproducibility matters, freeze or clone before traversal.

### 11.4. Use views and policy gates instead of destructive edits

* Use `core.UnweightedView` when an unweighted algorithm must inspect a weighted topology.
* Use BFS/DFS filters for traversal policy.
* Use Dijkstra wall thresholds for degraded links.
* Use matrix options for zero-weight and metric-closure semantics.

### 11.5. Never parse error strings

Use `errors.Is` with package sentinels. Error strings are for humans, not control flow.

### 11.6. Keep result ownership clear

Returned maps, slices, matrices, and residual graphs are caller-owned after return. If you need to mutate them for a new experiment, clone when appropriate.

### 11.7. Sanitize before statistics

Before covariance, correlation, regression-like features, or spectral experiments, use matrix sanitation tools such as `ReplaceInfNaN` and `Clip` when your raw data can contain dirty values.

### 11.8. Preserve zero-weight edges deliberately

A real zero-cost edge is common in replication, internal routing, or free transfer models. Use zero-preserving matrix policies when that distinction matters.

### 11.9. Prefer deterministic examples and tests

Avoid time-based cancellation in examples. Trigger cancellation from deterministic hooks. Print stable summaries rather than map iteration results.

### 11.10. Benchmark the right shape

Algorithm complexity depends heavily on graph density and matrix size. Benchmark sparse routing, dense matrix kernels, high-capacity flow, and generated fixtures separately.

---

## 12. Where To Go Next

Use this tutorial as the entry point, then move into package-specific material:

> If you are modeling topology first:
> 
>    [`CORE.md`](CORE.md) -> [`BFS.md`](BFS.md) -> [`DFS.md`](DFS.md) -> [`DIJKSTRA.md`](DIJKSTRA.md)

> If you are doing routing/optimization:
> 
>   [`DIJKSTRA.md`](DIJKSTRA.md) -> [`PRIM_&_KRUSKAL.md`](PRIM_&_KRUSKAL.md) -> [`FLOW.md`](FLOW.md) -> [`TRAVELING_SALESMAN.md`](TRAVELING_SALESMAN.md)

> If you are doing numeric graph analytics:
> 
>   [`CORE.md`](CORE.md) -> [`MATRICES.md`](MATRICES.md) -> examples/matrix_*.go

> If you are doing sequence comparison:
> 
>   [`DTW.md`](DTW.md) -> matrix sanitation/statistics sections

> If you are contributing:
> 
>   [`lvlath_UES.md`](lvlath_UES.md) -> [`CONTRIBUTING.md`](CONTRIBUTING.md) -> package doc.go -> tests/examples

---

**lvlath** teaches algorithms through contracts: every graph option, traversal field, numeric policy, and matrix cell should have a reason you can explain.

