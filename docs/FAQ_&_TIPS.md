<!--
  SPDX-License-Identifier: AGPL-3.0-only
  Copyright (C) 2025-2026 katalvlaran

  lvlath - FAQ & Tips

  Purpose:
    This document is the practical troubleshooting and best-practices guide for
    lvlath users and contributors. It explains common mistakes, package-selection
    decisions, deterministic behavior, error classification, graph/matrix semantics,
    testing practices, benchmark pitfalls, and documentation synchronization rules.
    It is aligned with the lvlath Universal Engineering Standard (UES).

  Contract status:
    This file is guidance, not a replacement for package doc.go contracts. When in
    doubt, package doc.go and current source code are authoritative. This document
    must not mention fictional APIs or stronger guarantees than the implemented
    packages provide.

  Scope:
    - Package selection and algorithm-choice guidance.
    - Common troubleshooting for core, bfs, dfs, dijkstra, matrix, flow, dtw, tsp,
      gridgraph, builder, and examples.
    - Testing, benchmarking, documentation, and contribution best practices.

  License:
    lvlath is licensed under AGPL-3.0-only. See ../LICENSE.
-->

# FAQ & Tips

> Practical engineering guidance for using and evolving **lvlath** without breaking contracts.

This file answers the questions that usually appear after reading `README.md` and `docs/TUTORIAL.md`: which package should I use, what does this result field mean, why does this error happen, why is `+Inf` intentional, why did a traversal skip something, and how should I test or benchmark a contract-sensitive algorithm?

For formal package contracts, read `{package}/doc.go`. For theory, diagrams, and algorithm walkthroughs, read `docs/{PACKAGE}.md`.

---

## Table of Contents

1. [Choosing the right package](#1-choosing-the-right-package)
2. [Core graph modeling](#2-core-graph-modeling)
3. [BFS FAQ](#3-bfs-faq)
4. [DFS FAQ](#4-dfs-faq)
5. [Dijkstra FAQ](#5-dijkstra-faq)
6. [Matrix FAQ](#6-matrix-faq)
7. [Flow FAQ](#7-flow-faq)
8. [DTW FAQ](#8-dtw-faq)
9. [MST and TSP FAQ](#9-mst-and-tsp-faq)
10. [Builder and gridgraph tips](#10-builder-and-gridgraph-tips)
11. [Errors and diagnostics](#11-errors-and-diagnostics)
12. [Determinism and concurrency](#12-determinism-and-concurrency)
13. [Testing tips](#13-testing-tips)
14. [Benchmark tips](#14-benchmark-tips)
15. [Documentation tips](#15-documentation-tips)
16. [Quick recipes](#16-quick-recipes)

---

## 1. Choosing the right package

### Q1.1. I have a graph. Which package should I start with?

Start with the question, not the algorithm name.

| Question | Use | Reason |
|:--|:--|:--|
| “How many hops away is this vertex?” | `bfs` | BFS minimizes edge count. |
| “Which vertices are in the same weak island?” | `bfs.Components` | Weak membership ignores direction for component grouping. |
| “What is a deterministic deep traversal?” | `dfs` | DFS gives post-order and forest-style structure. |
| “Does this graph contain a cycle?” | `dfs.DetectCycles` | Returns deterministic witness cycles. |
| “What order can I execute a DAG?” | `dfs.TopologicalSort` | Topological order is a directed acyclic graph contract. |
| “What is the cheapest route by weight?” | `dijkstra` | Dijkstra minimizes non-negative total cost. |
| “What is the cheapest connected backbone?” | `prim_kruskal` | MST solves minimum connected spanning structure. |
| “What is the maximum throughput?” | `flow` | Flow solves capacity-constrained source-to-sink movement. |
| “How do I turn graph topology into numeric features?” | `matrix` | Adjacency, incidence, degree vector, metric closure, algebra. |
| “How do I model a grid map?” | `gridgraph` | Generates lattice topology instead of manual wiring. |
| “How do I build repeatable fixtures?” | `builder` | Deterministic graph/data generation. |
| “How do I align two time series?” | `dtw` | Dynamic Time Warping handles local timing drift. |
| “How do I visit all cities in a route?” | `tsp` | Tour construction/improvement/exact-small-instance routing. |

### Q1.2. How do I avoid using the wrong algorithm?

Keep this mental table:

```text
minimize hops             -> BFS
inspect depth structure   -> DFS
prove/locate cycle        -> DFS cycle witness
order DAG tasks           -> DFS topological sort
minimize weighted cost    -> Dijkstra
connect all cheaply       -> MST
maximize throughput       -> Flow
compute all-pairs costs   -> Matrix metric closure
align time-warped signals -> DTW
build a closed tour       -> TSP
```

The most common mistake is using BFS on a weighted routing problem. BFS answers “fewest edges,” not “cheapest total weight.”

---

## 2. Core graph modeling

### Q2.1. Why does graph construction use options?

Graph options define the mathematical universe of the graph.

```go
g, err := core.NewGraph(
	core.WithDirected(true),
	core.WithWeighted(),
	core.WithLoops(),
)
```

This is not syntax decoration. It means:

* edges are directed by default;
* non-zero weights are legal;
* self-loops are legal;
* the graph will reject operations that violate other capability rules.

### Q2.2. Should I call `AddVertex` before `AddEdge`?

Usually no. `AddEdge` can create endpoints as part of graph construction. For examples and fixtures, prefer direct edge construction when vertices exist only because edges mention them.

Good for static examples:

```go
_, _ = g.AddEdge("gateway", "auth", 0)
_, _ = g.AddEdge("auth", "profile", 0)
```

Use explicit `AddVertex` when an isolated vertex matters:

```go
_ = g.AddVertex("isolated-zone")
```

### Q2.3. When is it acceptable to ignore `AddEdge` errors?

Only in tightly controlled examples with predetermined data and a comment explaining the shortcut.

```go
// Static tutorial data only. In production, check every AddEdge error.
_, _ = g.AddEdge("A", "B", 1)
```

In tests and production code, check errors unless the test is not about graph construction and the impossibility of error is obvious and already guarded by the graph options.

### Q2.4. Why are deterministic graph surfaces important?

Many valid graph answers are not unique. Without stable ordering, the same graph can produce different but still valid BFS paths, DFS orders, equal-cost Dijkstra witnesses, or MST edge orders.

`lvlath` treats order as a contract. If a package returns ordered output, the source of that order should be documented.

### Q2.5. Should I mutate a graph while an algorithm is running?

No, not if you need reproducible results. `core` protects its internal storage, but algorithm packages generally read progressively rather than snapshotting the full graph. Concurrent topology mutation can change what the algorithm sees.

Use a clone/view or run algorithms against stable topology.

---

## 3. BFS FAQ

### Q3.1. Why does BFS reject weighted graphs?

Because BFS computes hop distance:

```text
dist(s, v) = minimum number of edges from s to v
```

If weights represent latency, distance, cost, or risk, BFS is the wrong algorithm. Use `dijkstra` for non-negative weighted costs.

When you intentionally want hop distance over a weighted topology, use an unweighted view rather than rewriting the original graph.

### Q3.2. What is the difference between `Visited` and `Order`?

In BFS:

```text
Visited = discovered and enqueued
Order   = dequeued and processed
```

On a successful full traversal they are often close, but on cancellation or hook failure they can differ. A vertex may be discovered and queued but not yet processed.

This distinction is intentional and useful for diagnostics.

### Q3.3. Why does `PathTo` return one path, not all shortest paths?

BFS records one deterministic parent tree. `PathTo(dst)` reconstructs one shortest-hop witness from that tree. It does not enumerate all shortest paths.

If your domain needs all shortest paths, that is a different algorithmic surface and should not be inferred from `BFSResult.Parent`.

### Q3.4. Why did BFS skip some nodes?

Common causes:

1. The graph is directed and the relation does not point outward from the current vertex.
2. `WithMaxDepth` stopped expansion at the boundary.
3. `WithFilterNeighbor` rejected the relation.
4. Traversal was canceled or a hook returned an error.
5. The target is in a different weak component.

Check `Depth`, `Visited`, `Order`, and `Skipped` according to the package contract.

### Q3.5. How should I test BFS cancellation?

Use deterministic cancellation from a hook. Avoid timeouts in tests and examples.

Good pattern:

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

res, err := bfs.BFS(
	g,
	"gateway",
	bfs.WithContext(ctx),
	bfs.WithOnVisit(func(id string, depth int) error {
		if id == "ledger" {
			cancel()
		}
		return nil
	}),
)

_ = res
_ = err
```

Then assert partial-result fields exactly as the BFS contract defines.

---

## 4. DFS FAQ

### Q4.1. Why is `DFSResult.Order` not the order I expected?

Because returned DFS order is finish order, not discovery order.

```text
Discovery: vertex is entered.
Finish:    all reachable descendants from that vertex are completed.
Order:     appended on finish.
```

For a tree:

```text
       A
      / \
     B   C
    / \
   D   E
```

Discovery might be:

```text
A, B, D, E, C
```

Finish order is:

```text
D, E, B, C, A
```

Use an `OnVisit` hook if you need discovery order.

### Q4.2. Why does DFS clear or invalidate order on partial failure?

A partial finish order can be mathematically misleading. DFS may have entered vertices whose descendants were not fully explored. Package docs define which fields remain safe on partial results.

Do not treat partial DFS `Order` as a valid topological or post-order artifact.

### Q4.3. Does cycle detection return all cycles?

No, unless a package explicitly documents exhaustive enumeration. Cycle detection generally returns deterministic witness cycles: enough to prove cyclicity and locate a loop.

This avoids pretending that expensive exhaustive simple-cycle enumeration is a cheap default.

### Q4.4. Why does topological sort require a directed graph?

Topological order is defined over directed acyclic graphs. For an edge `u -> v`, `u` must appear before `v`. Undirected relations do not define this dependency direction.

If your graph contains undirected edges, decide whether they represent missing direction, bidirectional dependency, or a different problem.

---

## 5. Dijkstra FAQ

### Q5.1. When should I use Dijkstra?

Use Dijkstra for single-source shortest paths when all edge weights are non-negative.

Good domains:

* route cost;
* latency;
* distance;
* non-negative risk;
* service radius.

Do not use Dijkstra for finite negative weights. That is mathematically invalid for the algorithm.

### Q5.2. What is the difference between unknown and unreachable?

They are different states.

| State | Meaning | Typical result |
|:--|:--|:--|
| Unknown target | Target is not in the result domain. | `ErrTargetNotFound`. |
| Known unreachable | Target exists but no admissible path reaches it. | `+Inf`, nil error. |
| Tracking disabled | Distance may exist, but path reconstruction was not recorded. | `ErrPathTrackingDisabled` from path query. |
| No path | Known target but no path under policy. | `ErrNoPath` from path query when tracking is enabled. |

Do not collapse these into one “failed route” state.

### Q5.3. Why does Dijkstra publish `+Inf`?

`+Inf` is the canonical distance for known but unreachable vertices. It is not a dirty numeric value in this context. It has a precise meaning.

Use:

```go
if math.IsInf(cost, 1) {
	// known target, unreachable under active policy
}
```

### Q5.4. Why did `PathTo` fail even though `DistanceTo` worked?

Path reconstruction requires path tracking. If the result was computed without path tracking, distance queries can still work while path queries fail with a tracking-disabled error.

Use the package’s path-tracking option when you need witnesses.

### Q5.5. How do wall thresholds and max-distance cutoffs differ?

They represent different runtime policies.

```text
InfEdgeThreshold:
  skip edges whose weight is too high.
  Think: blocked/degraded links.

MaxDistance:
  do not publish/explore paths beyond a total distance.
  Think: service radius or budget limit.
```

Neither should mutate the graph topology.

### Q5.6. Why is equal-cost path selection stable?

Dijkstra should use deterministic frontier ordering and strict improvement. If a candidate distance equals the current best distance, predecessor state should not be overwritten.

This keeps one witness stable instead of oscillating between equal-cost alternatives.

---

## 6. Matrix FAQ

### Q6.1. Why not just use `[][]float64`?

Because `[][]float64` scatters row allocations and carries no contract for shape, numeric policy, ownership, views, graph semantics, or error classification.

`matrix.Dense` uses row-major storage:

```text
offset(i, j) = i * cols + j
```

That gives predictable memory layout and lets kernels implement fast paths while preserving the safe `Matrix` interface.

### Q6.2. Why does `NewDense(0, n)` succeed?

Zero-shape matrices are valid structural results.

```text
0×0: empty matrix
0×N: no rows, N columns
N×0: N rows, no columns
```

They are useful in feature extraction, empty graph transforms, and degenerate pipeline stages. `At(0,0)` should still fail with an out-of-range error because there is no addressable cell.

### Q6.3. Why does matrix code reject `NaN` or `Inf`?

By default, dirty floats are rejected because they poison downstream algebra and statistics. `+Inf` is allowed only under explicit distance/absence policy, such as metric closure or allow-inf distance modes.

For telemetry or external data, sanitize first:

```go
clean, err := matrix.ReplaceInfNaN(raw, 0)
if err != nil {
	return err
}
clean, err = matrix.Clip(clean, 0, 100)
if err != nil {
	return err
}
```

### Q6.4. Why did my zero-weight edge disappear?

Classic adjacency often uses `0` as “no edge.” That conflicts with real zero-weight edges.

If a finite zero is a real edge, use a matrix policy that preserves zero weights. In zero-preserving weighted adjacency, absence is represented by `+Inf`.

```go
opts, err := matrix.NewMatrixOptions(
	matrix.WithDirected(),
	matrix.WithWeighted(),
	matrix.WithPreserveZeroWeights(),
)
```

### Q6.5. What is the difference between adjacency and incidence?

```text
Adjacency: V × V
  cell means relation from vertex i to vertex j.
  Good for dense lookup, degree-like row scans, matrix algebra.

Incidence: V × E
  column means one edge identity.
  Good when edge identity, signs, loops, or structural edge participation matter.
```

Do not use dense adjacency as an edge ledger when parallel-edge identity matters.

### Q6.6. Why can’t I export metric closure back to a graph?

Metric closure is not original topology. It stores shortest-path distances.

```text
Original edge A -> C may not exist.
Metric closure D[A,C] may be finite through A -> B -> C.
```

Exporting that as an edge would invent topology. Refusal is correct.

### Q6.7. When should I use `View` vs `Induced`?

Use `View` when you intentionally want shared storage:

```text
MatrixView.Set writes into the base Dense.
```

Use `Induced` when the submatrix needs independent lifetime or safe mutation.

### Q6.8. Why does covariance require at least two rows for positive features?

Sample covariance divides by `rows - 1`. With positive feature count and one row, sample covariance is not defined under that formula.

For zero features (`cols == 0`), a valid `0×0` covariance is a structural result.

---

## 7. Flow FAQ

### Q7.1. Which max-flow algorithm should I choose?

| Algorithm | Use when | Notes |
|:--|:--|:--|
| Ford-Fulkerson | Small or educational integral-capacity networks. | Simple but can be slow depending on augmenting paths. |
| Edmonds-Karp | You want a simple polynomial guarantee. | Uses BFS shortest augmenting paths. |
| Dinic | Larger or denser networks. | Builds level graphs and blocking flows; usually faster in practice. |

### Q7.2. Why did repeated flow runs give different-looking residual states?

Max-flow algorithms modify residual capacity state. If you need independent runs, rebuild or clone the input network according to the package’s documented ownership/residual behavior.

Do not reuse a mutated residual graph as if it were the original capacity graph.

### Q7.3. Are edge weights capacities or costs?

In `flow`, weights represent capacities. In `dijkstra`, weights represent costs. In MST, weights represent connection cost. The same numeric field can mean different domain quantities depending on the algorithm.

Name variables accordingly in examples:

```go
capacityGraph, _ := core.NewGraph(core.WithDirected(true), core.WithWeighted())
```

### Q7.4. What is a residual graph?

A residual graph represents what capacity remains after current flow and where flow could be canceled or rerouted.

```text
Original capacity: u -> v = 10
Current flow:      u -> v = 6
Residual forward:  u -> v = 4
Residual reverse:  v -> u = 6
```

Residual state is part of the algorithm result, not just debug output.

---

## 8. DTW FAQ

### Q8.1. Why is DTW slow on long sequences?

Classic Dynamic Time Warping is dynamic programming over an `n × m` grid.

```text
Time:  O(n*m)
Space: O(n*m) for full matrix path recovery
```

Use a window constraint when domain knowledge says alignment should stay near the diagonal. Use a lower-memory mode when you need only distance and not the full path.

### Q8.2. Why can’t I recover a path in rolling-memory mode?

Path recovery needs enough history to backtrack. Rolling-row modes intentionally discard most of the matrix to save memory. Use the full matrix mode when you need the warping path.

### Q8.3. Why did DTW return infinite or impossible alignment?

Common causes:

* window too narrow;
* sequences too different under the chosen local cost;
* invalid numeric input;
* penalty/window combination excludes all valid routes.

A safe starting point is to first run without a tight window, verify expected behavior, then tighten the window.

### Q8.4. Should I normalize time-series data before DTW?

Usually yes. DTW aligns shapes in time, but raw amplitude scale still matters. Normalize or standardize when comparing sensors or signals with different units or magnitudes.

---

## 9. MST and TSP FAQ

### Q9.1. Why does MST reject my graph?

MST requires an undirected, connected, weighted graph under the package’s contract. Common problems:

* graph is directed;
* graph is unweighted;
* graph is disconnected;
* loops or mixed edges are present in a way the MST package refuses;
* weights are invalid.

MST is about connecting all vertices with minimum total weight. It is not a shortest-path algorithm.

### Q9.2. Kruskal or Prim?

Use either when the package supports your graph shape, but understand the pattern:

```text
Kruskal:
  global greedy over sorted edges + union-find.
  Natural for sparse edge-list thinking.

Prim:
  grows one tree from a root using cheapest frontier edges.
  Natural for connected dense/frontier thinking.
```

Both rely on the cut property and should be deterministic when tie-breaks are fixed.

### Q9.3. Why does TSP need a complete distance matrix?

TSP asks for a tour that can move between every pair of cities. If your original graph is sparse, first convert it into an all-pairs distance matrix through metric closure, then run a TSP algorithm on that distance matrix if the package’s assumptions are satisfied.

### Q9.4. Why do approximation guarantees require metric assumptions?

Some TSP approximations rely on triangle inequality. If your distances do not satisfy metric properties, the algorithm may still produce a tour, but the theoretical guarantee may not apply.

Do not document or claim a guarantee outside its assumptions.

---

## 10. Builder and gridgraph tips

### Q10.1. When should I use `builder`?

Use `builder` when you need reproducible graph shapes for:

* examples;
* tests;
* benchmarks;
* algorithm demonstrations;
* golden outputs.

Builder-generated fixtures should encode meaningful topology, not hide arbitrary setup.

### Q10.2. When should I use `gridgraph`?

Use `gridgraph` when your domain is naturally a lattice:

* grid maps;
* pathfinding demos;
* obstacle experiments;
* 4-neighbor/8-neighbor movement;
* teaching BFS/Dijkstra visually.

Do not manually wire large grids unless the example is specifically about low-level graph construction.

### Q10.3. How should random fixtures be handled?

Use fixed seeds and document the shape.

Bad:

```go
rand.Seed(time.Now().UnixNano())
```

Good:

```go
const seed = 42
```

Reproducibility is part of the benchmark/test contract.

---

## 11. Errors and diagnostics

### Q11.1. Why are error sentinels so important?

They let callers branch safely while still preserving helpful context.

```go
if errors.Is(err, matrix.ErrDimensionMismatch) {
	// stable protocol branch
}
```

Error strings are for humans. They may change without changing the protocol.

### Q11.2. How should wrapped errors behave?

Wrapping should preserve sentinel identity.

Good:

```go
return fmt.Errorf("build adjacency: %w", matrix.ErrUnknownVertex)
```

If an adapter needs to expose both an algorithm-level error and an underlying root cause, it should preserve both through the package’s chosen wrapping/join strategy.

### Q11.3. How do I debug a failing algorithm call?

Ask in this order:

1. Is the graph/matrix nil or wrong shape?
2. Are options valid and finalized once?
3. Does the graph capability match the algorithm?
4. Are weights/numeric values legal?
5. Is the target known but unreachable, or unknown?
6. Is path tracking enabled when asking for a path?
7. Is the result partial by contract?
8. Are docs/examples synchronized with current API?

---

## 12. Determinism and concurrency

### Q12.1. What does “deterministic” mean in lvlath?

It means the observable result is stable for the same input state, options, and package version.

Examples:

* BFS order follows the documented neighbor order.
* DFS result order means documented finish order.
* Dijkstra equal-cost witnesses follow documented tie/strict-improvement rules.
* Matrix adapters use stable vertex/edge order and fixed loop order.
* Components are sorted according to their contract.

### Q12.2. Does determinism mean every mathematically valid answer is identical across packages?

No. It means each package publishes its own deterministic choice among valid answers. For example, one shortest path may be selected as a witness, not all possible shortest paths.

### Q12.3. Is `core.Graph` safe for concurrency?

`core` is designed with internal locking for graph storage and controlled mutations. Algorithm reproducibility still requires stable topology while an algorithm runs unless the package explicitly documents snapshot isolation.

### Q12.4. Is `matrix.Dense` safe for concurrency?

Concurrent read-only access is safe when no goroutine mutates the matrix. Concurrent writes or read/write races require external synchronization. Views share storage with their base matrix.

---

## 13. Testing tips

### Q13.1. What should a good package test suite contain?

At minimum:

```text
validation tests:
  nil input, bad options, invalid shape, invalid numeric values

medium tests:
  real non-trivial graph/matrix/math scenarios

special tests:
  zero-shape, loops, mixed edges, zero-weight edges, cancellation,
  equal-cost ties, partial-result behavior

determinism tests:
  stable order, stable witnesses, stable component sorting

error tests:
  errors.Is checks for every expected sentinel
```

### Q13.2. Should tests use `testify`?

No. Use the standard library and package-specific helpers. This keeps failure messages precise and avoids dependency weight.

### Q13.3. When is `reflect.DeepEqual` acceptable?

Use explicit structural helpers when the domain matters. For example, prefer path comparison helpers, matrix cell checks, sorted domain checks, and numeric tolerance helpers.

`reflect.DeepEqual` is acceptable only when it genuinely expresses the whole contract and produces understandable failure output.

### Q13.4. How should numeric tests compare floats?

Use tolerances for computations that can accumulate floating-point error.

```go
if math.Abs(got-want) > tol {
	t.Fatalf("got %.12g, want %.12g", got, want)
}
```

For matrix-wide comparison, use the package’s tolerant comparison surface where appropriate.

### Q13.5. How should tests handle map output?

Do not print or compare maps through unstable iteration. Extract keys, sort them, then compare or print stable output.

### Q13.6. How should cancellation be tested?

Use deterministic hooks or contexts. Avoid timeouts as the primary mechanism in examples or unit tests.

### Q13.7. What is a regression anchor?

A regression anchor is a test named after the bug or contract it protects.

Example:

```go
func TestAdjacencyMatrix_ToGraphPreservesZeroWeightEdgeByDefaultInInfEncoding(t *testing.T) {
	// ...
}
```

A bug fix without a regression test is incomplete unless there is a clear reason no test can be written.

---

## 14. Benchmark tips

### Q14.1. What should be inside the benchmark loop?

Only the measured operation.

Good:

```go
g := buildGraph(b)
b.ReportAllocs()
b.ResetTimer()
for i := 0; i < b.N; i++ {
	_, err := runAlgorithm(g)
	if err != nil {
		b.Fatal(err)
	}
}
```

Bad:

```go
for i := 0; i < b.N; i++ {
	g := buildGraph(b) // measures setup, not just algorithm
	_, _ = runAlgorithm(g)
}
```

### Q14.2. What is a benchmark regime?

A regime is the real shape or pressure you are measuring.

Examples:

* sparse chain;
* dense graph;
* mixed-edge endpoint resolution;
* path tracking on/off;
* cutoff policy overhead;
* flow level graph rebuild pressure;
* matrix fast path vs generic fallback;
* DTW full matrix vs rolling memory.

A benchmark name should reveal the regime.

### Q14.3. Why must setup errors be checked in benchmarks?

Because graph policy can reject edges. If setup silently fails, the benchmark may measure a smaller or different graph than intended.

This is one of the easiest ways to produce meaningless performance numbers.

### Q14.4. Should benchmarks use random data?

Only with fixed seed and documented shape. Randomness without reproducibility is not acceptable for regression-sensitive performance work.

---

## 15. Documentation tips

### Q15.1. Which file should explain what?

| File | Responsibility |
|:--|:--|
| `README.md` | Public face, package map, strengths, key examples. |
| `docs/TUTORIAL.md` | First learning path and fundamental concepts. |
| `{package}/doc.go` | Package charter and implemented API contract. |
| Exported GoDoc | Per-symbol behavior, errors, determinism, complexity. |
| `docs/{PACKAGE}.md` | Theory, diagrams, pseudocode, pitfalls, examples. |
| `examples/*.go` | Larger scenario programs. |
| `example_test.go` | Executable package examples with deterministic output. |
| `CONTRIBUTING.md` | Contribution workflow and quality gates. |

### Q15.2. What is documentation drift?

Documentation drift happens when docs, examples, tests, and code describe different behavior.

Common drift patterns:

* docs mention removed APIs;
* examples use old constructors;
* README claims Go version or license incorrectly;
* package docs promise stronger guarantees than tests protect;
* tests lock behavior that contradicts mathematical correctness.

Fix drift immediately. Do not leave “almost correct” docs in place.

### Q15.3. Should docs include formulas?

Yes, when the formula explains real implemented semantics.

Good formulas:

* BFS hop distance;
* Dijkstra relaxation;
* Floyd-Warshall update;
* covariance/correlation;
* max-flow constraints;
* DTW recurrence;
* TSP objective.

Bad formulas are decorative and should be removed.

### Q15.4. Should docs include ASCII diagrams?

Yes, when they clarify topology, memory layout, residual flow, traversal order, or matrix encoding.

A good ASCII diagram should make the next paragraph easier to understand.

---

## 16. Quick recipes

### Recipe A. Hop blast radius

Use BFS with a depth limit. Group by `Depth`.

```text
Use when:
  incident response, service dependency waves, local exploration.
Avoid when:
  edge weights matter.
```

### Recipe B. Weighted failover route

Use Dijkstra with path tracking and, if needed, wall thresholds.

```text
Use when:
  links have latency/cost/risk.
Remember:
  known unreachable = +Inf, not missing target.
```

### Recipe C. Detect dependency cycles

Use DFS cycle detection.

```text
Use when:
  release pipeline, lock graph, task dependency validation.
Remember:
  witness cycles are not necessarily exhaustive cycle enumeration.
```

### Recipe D. Build a topology feature matrix

Use `matrix.NewAdjacencyMatrix`, `DegreeVector`, incidence, or metric closure.

```text
Use when:
  analytics, spectral methods, route matrices, graph features.
Remember:
  choose zero-weight policy deliberately.
```

### Recipe E. Model throughput

Use flow algorithms on a directed weighted capacity graph.

```text
Use when:
  capacity planning, assignment, bottleneck cuts.
Remember:
  residual graph is algorithm state, not the original graph.
```

### Recipe F. Compare time-warped signals

Use DTW with explicit memory mode and window policy.

```text
Use when:
  same pattern, different speed or local timing.
Remember:
  path recovery requires enough stored matrix state.
```

### Recipe G. Build reproducible tests

Use `builder` or manually fixed topology. Sort output before printing. Check errors with `errors.Is`. Avoid time-based cancellation.

---

## Final rule

When in doubt, ask:

```text
What is the mathematical object?
What is the package contract?
What is the deterministic order source?
What sentinel class should failure expose?
Who owns the result?
What does this value mean under this policy?
```

If you cannot answer those questions, the code, test, example, or documentation is not ready.

