// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package tsp provides Traveling Salesman Problem (TSP) and Asymmetric TSP (ATSP)
// solvers over matrix.Matrix distance models and core.Graph adapters.
//
// The package exposes exact search, classical approximation, and local-search
// improvements behind a single deterministic dispatcher with a consistent API,
// strict sentinel errors, and stable cost rounding (1e-9).
//
// # Domain Scope & Goals
//
//   - Complete finite TSP/ATSP distance matrices as final solver input.
//   - Direct matrix inputs with optional metric closure from +Inf missing edges.
//   - core.Graph inputs through an explicit graph-to-matrix adapter.
//   - Exact solvers: Held-Karp dynamic programming and Branch-and-Bound.
//   - Approximation: Christofides 1.5-approx for symmetric metric TSP.
//   - Local search: Deterministic 2-opt and 3-opt post-passes (standalone or via dispatcher).
//
// # Non-Goals
//
//   - No branch-and-cut / cutting-plane Concorde-style solvers.
//   - No hidden fallback from exact to heuristic unless Algo == Auto is explicitly selected.
//   - No final TSP solving over matrices that still contain +Inf off-diagonal edges.
//   - No formal Christofides 1.5 guarantee when Blossom/MWPM is unavailable and
//     a deterministic greedy matching fallback is used.
//   - No lossless support for mixed directed/undirected core.Graph inputs until the
//     matrix adapter exposes a mixed-aware representation.
//
// # Matrix Law
//
//   - Public matrix input is matrix.Matrix.
//   - The matrix package validates nil, typed-nil, square shape, and APSP distance semantics.
//   - The tsp package adds TSP-domain constraints: n >= 2, diagonal ≈ 0, no negative weights,
//     optional symmetry, and finite complete final solver input.
//   - Direct RunMetricClosure copies input into a detached Dense distance matrix and
//     delegates APSP to matrix.APSPInPlace.
//   - Final solver kernels must never receive +Inf as a missing-edge sentinel.
//
// # Numeric Law
//
//   - NaN and -Inf trigger ErrNaNInf.
//   - Negative finite distances trigger ErrNegativeWeight.
//   - Diagonal must be zero within symTol (|a_ii| <= 1e-12).
//   - +Inf denotes a "missing edge" only before metric closure.
//   - Remaining +Inf after closure triggers ErrIncompleteGraph.
//   - Costs are rounded to 1e-9 (round1e9) to avoid floating-point drift.
//
// # Determinism & Stability Law
//
//   - No time-based randomness is used. Matrix row/column order dictates vertex order.
//   - core.Graph order is inherited through the matrix adapter and recovered by index.
//   - Tie-breaks use vertex indices unless a specific algorithm states a stricter rule.
//   - Branch-and-Bound branches by ascending edge weight and then vertex index.
//   - Tours are canonicalized by a fixed start vertex and orientation where applicable
//     (via CanonicalizeOrientationInPlace).
//   - Randomized local-search scans are strictly deterministic under Options.Seed.
//     Seed == 0 provides a fixed default stream.
//
// # Algorithms & Complexity
//
//	ExactHeldKarp (Held–Karp DP) - Supports TSP and ATSP
//	  Time:       O(n² * 2ⁿ)
//	  Memory:     O(n * 2ⁿ)
//	  Guards:     MaxExactN (=16) to bound resource allocation.
//
//	BranchAndBound (Exact DFS with pruning) - Supports TSP and ATSP
//	  Bound:      Degree-1 relaxation (admissible) and optional root-only 1-tree LB (TSP only).
//	  Branch:     Neighbors sorted by weight, then index (deterministic).
//	  Time:       Exponential worst case.
//	  Memory:     O(n) path state + O(n²) precomputes.
//
//	Christofides (1.5-approx) - Symmetric metric TSP only
//	  Pipeline:   MST → minimum perfect matching → Eulerian circuit → shortcut to tour.
//	  Matching:   BlossomMatch (true MWPM) with GreedyMatch fallback.
//	  Time:       Typically O(n²) on dense metric instances.
//
//	TwoOptOnly / ThreeOptOnly (Local search) - Supports TSP and ATSP
//	  2-opt (TSP):  Segment reversal; Δ = (a→c)+(b→d)−(a→b)−(c→d).
//	  2-opt* (ATSP):Tail swap without reversals.
//	  3-opt:        7 reconnections (TSP) / 3-opt* (ATSP).
//	  Strategy:     Deterministic first/best-improvement, optional shuffled enumeration via Seed.
//
//	Metric Closure (Floyd–Warshall)
//	  Time:       O(n³)
//	  Memory:     O(n²) detached Dense storage.
//
// # Input Symmetry Requirements
//
// Strict matrix symmetry (dist[i][j] == dist[j][i]) is enforced when:
//   - opts.Algo == Christofides
//   - opts.BoundAlgo == OneTreeBound
//   - opts.Symmetric == true (explicit user validation request)
//
// If opts.RunMetricClosure == false, the validator rejects +Inf off-diagonal entries.
// Otherwise, matrix-level metric closure is applied upstream.
//
// # Result & Partial-Result Law
//
//   - SolveMatrix and SolveGraph are canonical facades and return *TSPResult.
//   - TSPResult owns detached Tour, IDs, and Warnings slices.
//   - TSResult is a minimal compatibility projection used by SolveWithMatrix
//     and SolveWithGraph.
//   - Held-Karp, Christofides, and local-search wrappers return no partial results on failure.
//   - Branch-and-Bound may return a non-nil TSPResult with TimedOut=true and
//     Optimal=false when a feasible incumbent exists at timeout.
//   - Legacy TSResult wrappers discard partial metadata and return zero TSResult on error.
//   - Callers must treat any result returned with ErrTimeLimit as partial.
//
// # Matching Law
//
//   - BlossomMatch attempts true MWPM.
//   - If BlossomMatch returns ErrMatchingNotImplemented, the Christofides kernel may
//     fall back to deterministic GreedyMatch.
//   - Greedy fallback keeps the pipeline deterministic and feasible when finite edges exist,
//     but clears the formal 1.5 approximation guarantee.
//   - MatchingFallback, ApproximationRatio, and fallback Warnings are populated from
//     tspApproxWithMeta kernel metadata, never inferred in the facade from Options alone.
//
// # Error Law
//
//   - Errors must be classified exclusively through errors.Is. Do not compare error strings.
//   - Matrix and core adapter errors preserve underlying sentinels where applicable.
//   - ErrDimensionMismatch is strictly a structural error; it is not a catch-all for
//     nil matrices, invalid IDs, invalid options, invalid tours, or NaN/Inf.
//
// # AI-Hints
//
//   - Do not use SolveWithMatrix in new code when metadata matters.
//   - Do not infer MatchingFallback outside tspApproxWithMeta.
//   - Do not allow +Inf into final solver kernels.
//   - Do not compare error strings; use errors.Is.
//   - Do not change DefaultOptions to Auto in a patch release without an explicit
//     compatibility decision.
package tsp
