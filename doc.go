// Package lvlath is a practical toolkit for graphs, flows, TSP, and
// time-series alignment-built for predictability, strict error contracts,
// and deterministic outcomes. It is pure Go (no cgo, no external deps) and
// organized as small, composable subpackages.
//
// ─────────────────────────────────────────────────────────────────────────────
// What lvlath is
// ─────────────────────────────────────────────────────────────────────────────
//
// lvlath is a set of focused packages you can use independently or together:
//
//   - core         - common contracts and sentinel errors shared across packages
//   - matrix       - minimal distance/weight matrix interface and helpers
//   - builder      - reproducible generators/fixtures for graphs and matrices
//   - gridgraph    - 2D grids, neighborhood generators, and weight helpers
//   - bfs, dfs     - fundamental traversals
//   - dijkstra     - single-source shortest paths with non-negative weights
//   - prim_kruskal - minimum spanning trees (Prim O(n²) for dense, Kruskal for sparse)
//   - flow         - max-flow / min-cut (Edmonds–Karp, Dinic)
//   - dtw          - Dynamic Time Warping for time-series alignment
//   - tsp          - Christofides-style approximation (symmetric metrics),
//     2-opt/3-opt local search, 1-tree lower bound, exact BnB,
//     and an integration dispatcher
//
// Each package documents its own API and edge cases in {package}/doc.go.
// Background “What/Why/When” material with formulas and visual intuition lives
// in docs/{ALGORITHM}.md. A practical, end-to-end learning path starts at
// docs/TUTORIAL.md.
//
// ─────────────────────────────────────────────────────────────────────────────
// Error model and invariants
// ─────────────────────────────────────────────────────────────────────────────
//
// All packages use a shared set of sentinel errors (test-friendly via errors.Is):
//
//	ErrNonSquare, ErrNonZeroDiagonal, ErrAsymmetry,
//	ErrNegativeWeight, ErrIncompleteGraph, ErrDimensionMismatch,
//	ErrTimeLimit, ErrATSPNotSupportedByAlgo, ErrMatchingNotImplemented.
//
// Policy:
//   - Validate inputs early and explicitly (shape, bounds, NaN, ±Inf).
//   - Make ambiguous conditions explicit through options (e.g., symmetric vs ATSP).
//   - Disallow unsupported combinations in dispatchers (e.g., Christofides with ATSP).
//
// ─────────────────────────────────────────────────────────────────────────────
// Determinism and numerics
// ─────────────────────────────────────────────────────────────────────────────
//
//   - No global state, no implicit RNG. Algorithms are deterministic for the
//     same inputs and options.
//   - Where randomized neighborhood order is useful (local search), it is fully
//     controlled by Options.Seed and can be disabled; results remain stable under
//     the same seed.
//   - Floating-point comparisons honor a clear epsilon policy (documented per
//     package). Tests stabilize comparisons via consistent rounding to avoid
//     platform drift.
//
// ─────────────────────────────────────────────────────────────────────────────
// Package overview (balanced highlights)
// ─────────────────────────────────────────────────────────────────────────────
//
// core
//
//	Shared contracts and sentinel errors. Read this first to understand how
//	lvlath surfaces invalid inputs, time budgets, and unsupported modes.
//
// matrix
//
//	A minimal, bounds-checked matrix interface used by dense algorithms
//	(e.g., Prim O(n²), TSP). Keeps implementations pluggable and testable.
//
// builder
//
//	Deterministic generators for graphs/matrices (rings, grids, rippled circles,
//	random-but-seeded fixtures). Useful for prototyping, teaching, benchmarks,
//	and reproducible tests.
//
// gridgraph
//
//	2D lattice graphs with 4/8-neighborhoods, mask-based obstacles,
//	and convenience weights (e.g., Euclidean/L1). Pairs well with bfs/dfs/dijkstra.
//
// bfs, dfs
//
//	Straightforward traversals with hook points (visit/enqueue/edge events).
//	Used as learning primitives and as building blocks.
//
// dijkstra
//
//	Single-source shortest paths with non-negative edges. Clear behavior on
//	unreachable vertices; predictable parent trees.
//
// prim_kruskal
//
//	MST via Prim O(n²) on dense matrix inputs and Kruskal for sparse edge lists.
//	Deterministic tie-breaking by indices; consistent total weight across runs.
//
// flow
//
//	Max-flow/min-cut via Edmonds–Karp (simple, robust) and Dinic (faster on
//	medium/large instances). Clean separation of capacity graph, source/sink,
//	and residual logic. Results are deterministic for equal inputs.
//
// dtw
//
//	Classic O(nm) Dynamic Time Warping with optional constraints (e.g., Sakoe–Chiba
//	bands). Stable costs and clear boundary conditions. Suitable for alignment
//	tasks in signal processing and ML pipelines.
//
// tsp
//
//	Practical toolbox around the traveling-salesperson problem:
//	  – Symmetric metrics: Christofides-style pipeline (MST → matching → Euler →
//	    shortcut) with optional 2-opt/3-opt polish. Greedy matching by default;
//	    exact Blossom is not included and is surfaced as ErrMatchingNotImplemented.
//	  – Lower bounds: 1-tree (Held–Karp style) for pruning/validation.
//	  – Exact search: Branch-and-Bound for small n, with pluggable bounds.
//	  – ATSP: handled via local search paths (2-opt/3-opt) in directed mode.
//	A top-level dispatcher wires validation, algorithm choice, and post-processing.
//
// ─────────────────────────────────────────────────────────────────────────────
// Performance and trade-offs (concise, per area)
// ─────────────────────────────────────────────────────────────────────────────
//
//   - MST: Prim O(n²) favors dense matrix inputs; Kruskal favors sparse lists.
//   - Shortest paths: Dijkstra requires non-negative weights; for negatives,
//     use a different algorithm (not provided here).
//   - Flow: Edmonds–Karp is simpler (O(VE²)), Dinic is typically much faster.
//   - DTW: O(nm) time/space; use windowing constraints to control complexity.
//   - TSP: Auto paths for symmetric metrics deliver strong practical tours;
//     exact BnB is intended for small instances; ATSP uses local search.
//   - All implementations are designed to be testable and reproducible rather
//     than micro-optimized at the expense of clarity. Benchmarks in *_test.go
//     cover realistic sizes and avoid timer pollution.
//
// ─────────────────────────────────────────────────────────────────────────────
// Scope boundaries and known limitations
// ─────────────────────────────────────────────────────────────────────────────
//
//   - Dijkstra: negative edges are not supported by design.
//   - TSP: Blossom (exact minimum-weight matching) is not implemented; the API
//     surfaces this as ErrMatchingNotImplemented. Christofides uses a greedy
//     matching heuristic with clear, documented behavior.
//   - DTW: classic dynamic programming formulation; memory-optimized variants
//     are not included.
//   - Flow: capacities are assumed finite and non-negative; generalized flows
//     are out of scope.
//
// ─────────────────────────────────────────────────────────────────────────────
// Where to go next
// ─────────────────────────────────────────────────────────────────────────────
//
//   - docs/TUTORIAL.md - start here for an end-to-end tour, selection matrix,
//     and practical guidance on determinism and numeric stability.
//   - {package}/doc.go - the formal API contracts, options, and edge cases.
//   - docs/{ALGORITHM}.md - compact backgrounders with formulas, diagrams,
//     and runnable examples for each algorithm.
//
// lvlath targets modern Go, is entirely in Go, and has no external dependencies.
package lvlath
