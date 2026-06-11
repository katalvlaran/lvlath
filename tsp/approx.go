// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package tsp - Christofides 1.5-approximation.
//
// TSPApprox computes a 1.5-approximate Hamiltonian cycle for the symmetric,
// metric Travelling Salesman Problem using the Christofides pipeline:
//
//  1. Minimum Spanning Tree (MST) on the complete metric graph.
//  2. Minimum-weight perfect matching on odd-degree vertices of the MST.
//  3. Eulerian circuit on the resulting multigraph.
//  4. Shortcutting the Eulerian walk to a Hamiltonian cycle (skip revisits).
//
// Mathematical guarantee:
//   - For metric symmetric TSP (triangle inequality, non-negative, symmetric),
//     the returned tour length ≤ 1.5 · OPT.
//
// Contracts (validated by the dispatcher via validateAll):
//   - dist is square n×n, n ≥ 2,
//   - diagonal ≈ 0, no negative weights, no NaN,
//   - symmetric (opts.Symmetric==true / mustEnforceSymmetry(opts) == true),
//   - if opts.RunMetricClosure==false: no +Inf edges allowed.
//
// Options notes:
//   - opts.StartVertex fixes the start/closure of the cycle.
//   - opts.MatchingAlgo selects between BlossomMatch (preferred) and GreedyMatch,
//     with a strict fallback to Greedy when blossom returns ErrMatchingNotImplemented.
//   - No RNG is used here; determinism is intrinsic.
//   - Local-search post-passes (2-opt / 3-opt) orchestrates the dispatcher (SolveWithMatrix):
//   - if EnableLocalSearch && !BestImprovement → fast 2-opt
//   - if EnableLocalSearch &&  BestImprovement → hybrid [2-opt → 3-opt(best) → 2-opt polish]
//     This keeps Christofides pure and predictable; tuning lives in the dispatcher.
//
// Complexity (dense representation):
//   - MST (Prim O(n^2)) + odd collection O(n) +
//     matching (implementation-dependent; greedy O(k^2), blossom polytime) +
//     Eulerian (O(E)), shortcut O(n)  ⇒ typically O(n^2) for metric instances.
//
// Returned value:
//   - TSResult{Tour, Cost} with stable rounding (1e-9) applied to Cost.
//   - Tour invariants: len==n+1, Tour[0]==Tour[n]==opts.StartVertex, each vertex appears once.
//
// Errors:
//   - Only strict sentinels from types.go (e.g., ErrStartOutOfRange, ErrIncompleteGraph, …).
//
// Guarantee note:
//   - The 1.5·OPT bound relies on step (2) being a true minimum-weight perfect matching (MWPM).
//     When Blossom/MWPM is unavailable, the implementation explicitly falls back to a
//     deterministic greedy matching to keep the pipeline correct and reproducible.
//     In the greedy fallback the tour remains valid (Eulerian multigraph → shortcut),
//     but the formal 1.5 factor is not guaranteed. Set MatchingAlgo=GreedyMatch to opt in
//     explicitly; keep BlossomMatch to automatically benefit once MWPM is enabled.
package tsp

import (
	"errors"

	"github.com/katalvlaran/lvlath/matrix"
)

// approxMeta records approximation-specific facts observed inside the Christofides kernel.
// It is intentionally private because callers should consume this information through TSPResult.
//
// Implementation:
//   - Stage 1: Start with the guarantee implied by the requested matching policy.
//   - Stage 2: Downgrade the guarantee only when the kernel actually falls back.
//   - Stage 3: Preserve warning sentinels for TSPResult.Warnings.
//
// Behavior highlights:
//   - MatchingFallback is kernel-origin metadata, never facade inference.
//   - ProvenRatio is 1.5 only when a true MWPM path was used.
//   - Greedy matching is valid but has no formal Christofides bound.
//
// Inputs:
//   - Produced by tspApproxWithMeta.
//
// Returns:
//   - Internal metadata consumed by solvePreparedMatrix.
//
// Errors:
//   - None.
//
// Determinism:
//   - Metadata is derived from deterministic matching branch decisions.
//
// Complexity:
//   - Time O(1), Space O(len(Warnings)) only when warnings are copied later.
//
// Notes:
//   - NoApproximationRatio means “do not claim a formal approximation factor”.
//
// AI-Hints:
//   - Do not set MatchingFallback in SolveMatrix from Options alone.
//   - Do not claim 1.5 when BlossomMatch fell back to GreedyMatch.
type approxMeta struct {
	matchingFallback bool
	provenRatio      float64
	warnings         []error
}

// newApproxMeta initializes approximation metadata from the requested matching policy.
// Implementation:
//   - Stage 1: If GreedyMatch is explicitly selected, no formal ratio is claimed.
//   - Stage 2: If BlossomMatch is selected, the ratio is provisional until fallback is observed.
//
// Behavior highlights:
//   - The 1.5 ratio is only retained if Blossom/MWPM succeeds.
//   - GreedyMatch is an explicit weaker policy.
//
// Inputs:
//   - opts: solver options.
//
// Returns:
//   - approxMeta: initial metadata state.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// AI-Hints:
//   - Keep this helper conservative; false guarantees are worse than missing metadata.
func newApproxMeta(opts Options) approxMeta {
	if opts.MatchingAlgo == GreedyMatch {
		return approxMeta{provenRatio: NoApproximationRatio}
	}

	return approxMeta{provenRatio: ChristofidesApproximationRatio}
}

// recordGreedyFallback records BlossomMatch -> GreedyMatch degradation.
// Implementation:
//   - Stage 1: Set MatchingFallback.
//   - Stage 2: Clear the formal approximation ratio.
//   - Stage 3: Attach ErrMatchingNotImplemented as a non-fatal warning.
//
// Behavior highlights:
//   - The returned tour may still be valid and deterministic.
//   - The formal 1.5 Christofides guarantee is intentionally removed.
//
// Errors:
//   - None.
//
// Complexity:
//   - Time O(1), Space amortized O(1).
//
// AI-Hints:
//   - Do not return ErrMatchingNotImplemented as a fatal error when greedy fallback succeeds.
func (m *approxMeta) recordGreedyFallback() {
	m.matchingFallback = true
	m.provenRatio = NoApproximationRatio
	m.warnings = append(m.warnings, ErrMatchingNotImplemented)
}

// TSPApprox runs the Christofides pipeline and returns the legacy minimal result.
// It is kept as a compatibility/public kernel wrapper; canonical metadata is published
// by SolveMatrix through tspApproxWithMeta.
//
// Implementation:
//   - Stage 1: Delegate to tspApprox.
//   - Stage 2: Discard approximation metadata intentionally.
//
// Behavior highlights:
//   - Does not duplicate Christofides logic.
//   - Preserves the legacy TSResult surface.
//
// Inputs:
//   - dist: validated symmetric complete distance matrix.
//   - opts: solver options.
//
// Returns:
//   - TSResult: tour and cost.
//   - error: sentinel-classified failure.
//
// Determinism:
//   - Same as tspApproxWithMeta.
//
// Complexity:
//   - Same as tspApproxWithMeta.
//
// AI-Hints:
//   - Do not add matching metadata inference here; use tspApproxWithMeta.
func TSPApprox(dist matrix.Matrix, opts Options) (TSResult, error) {
	result, _, err := tspApprox(dist, opts)
	return result, err
}

// tspApprox runs Christofides and returns kernel-origin approximation metadata.
// Implementation:
//   - Stage 1: Validate start vertex against matrix order.
//   - Stage 2: Build MST.
//   - Stage 3: Collect odd-degree vertices.
//   - Stage 4: Execute selected matching policy and record real fallback metadata.
//   - Stage 5: Build Eulerian circuit and shortcut to Hamiltonian tour.
//   - Stage 6: Compute and validate final cost/tour.
//
// Behavior highlights:
//   - MatchingFallback is set only when BlossomMatch actually falls back to GreedyMatch.
//   - ApproximationRatio is 1.5 only when true MWPM is used.
//   - Greedy fallback is deterministic and valid but not formally 1.5-bounded.
//
// Inputs:
//   - dist: symmetric complete TSP distance matrix.
//   - opts: Christofides policy.
//
// Returns:
//   - TSResult: minimal tour/cost result.
//   - approxMeta: approximation/fallback metadata.
//   - error: sentinel-classified failure.
//
// Errors:
//   - ErrStartOutOfRange.
//   - ErrIncompleteGraph from MST/matching/tour cost.
//   - ErrInvalidOptions for unsupported matching enum.
//   - ErrNonEulerian from malformed Eulerian multigraph state.
//   - ErrInvalidTour / ErrNaNInf / ErrNegativeWeight propagated by cost/tour helpers.
//
// Determinism:
//   - MST starts at vertex 0 and uses stable index tie-breaks.
//   - Greedy matching uses stable cost then vertex-index tie-break.
//   - Eulerian walk follows deterministic adjacency order.
//   - Final tour orientation is canonicalized.
//
// Complexity:
//   - Time O(n^2) with greedy matching on dense instances.
//   - Space O(n^2) if upstream matrix storage dominates; local extra memory is O(n+E).
//
// AI-Hints:
//   - Do not claim ChristofidesApproximationRatio when meta.matchingFallback is true.
//   - Do not treat ErrMatchingNotImplemented as fatal when fallback succeeds.
func tspApprox(dist matrix.Matrix, opts Options) (TSResult, approxMeta, error) {
	meta := newApproxMeta(opts)

	// Lightweight start-range guard (n already known to be ≥ 2 in the dispatcher).
	n := dist.Rows()
	if err := validateStartVertex(n, opts.StartVertex); err != nil {
		return TSResult{}, meta, err
	}

	// 1) Minimum Spanning Tree on the metric graph.
	//    Returns total weight (unused here) and a simple-graph adjacency (no multi-edges).
	_, mstAD, err := MinimumSpanningTree(dist) // O(n^2) Prim (see mst.go)
	if err != nil {
		return TSResult{}, meta, err
	}

	// 2) Collect odd-degree vertices of the MST.
	//    V has odd degree iff degree(v) mod 2 == 1. Fast parity check via bit-test.
	//    len(mstAD[v])&1 == 1  ⇔ degree(v) is odd (LSB set).
	odd := make([]int, 0, n/2+1) // conservative capacity avoids reslices
	var vertex int               // loop iterator
	for vertex = 0; vertex < n; vertex++ {
		if (len(mstAD[vertex]) & 1) == 1 {
			odd = append(odd, vertex)
		}
	}

	// 3) Add a minimum-weight perfect matching among odd-degree vertices.
	//    We modify the adjacency in-place, effectively forming the Eulerian multigraph.
	switch opts.MatchingAlgo {
	case BlossomMatch:
		if matchErr := blossomMatch(odd, dist, mstAD); matchErr != nil {
			if errors.Is(matchErr, ErrMatchingNotImplemented) {
				// Deterministic and safe fallback; preserves pipeline validity.
				if err = greedyMatch(odd, dist, mstAD); err != nil {
					return TSResult{}, meta, err
				}

				meta.recordGreedyFallback()
			} else {
				return TSResult{}, meta, matchErr
			}
		}

	case GreedyMatch:
		if err = greedyMatch(odd, dist, mstAD); err != nil {
			return TSResult{}, meta, err
		}
		meta.provenRatio = NoApproximationRatio

	default:
		return TSResult{}, meta, ErrInvalidOptions
	}

	// 4) Eulerian circuit on the multigraph (Hierholzer).
	//    Returns a closed walk that starts at opts.StartVertex and finishes at it.
	//    The circuit cost is O(E), where E is the number of (multi)edges.
	euler, err := EulerianCircuit(mstAD, opts.StartVertex)
	if err != nil {
		return TSResult{}, meta, err
	}
	// 5) Shortcut revisits to obtain a Hamiltonian tour; then canonicalize direction.
	tour, err := ShortcutEulerianToHamiltonian(euler, n, opts.StartVertex)
	if err != nil {
		return TSResult{}, meta, err
	}

	_ = CanonicalizeOrientationInPlace(tour)

	// 6) Compute the stabilized tour cost with strict edge validation.
	//    tourCost checks Inf/NaN/negatives defensively and rounds to 1e-9.
	cost, err := TourCost(dist, tour)
	if err != nil {
		return TSResult{}, meta, err
	}

	// Final invariant check (O(n)) - inexpensive, helps catch wiring mistakes early.
	if err = ValidateTour(tour, n, opts.StartVertex); err != nil {
		return TSResult{}, meta, err
	}

	return TSResult{Tour: tour, Cost: cost}, meta, nil
}
