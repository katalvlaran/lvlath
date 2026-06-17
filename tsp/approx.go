// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package tsp - Christofides approximation pipeline.
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
//   - The returned tour has the Christofides 1.5 bound only when the matching
//     stage computes exact minimum-weight perfect matching.
//
// Contracts (validated by the dispatcher via validateAll):
//   - dist is square n×n, n ≥ 2,
//   - diagonal ≈ 0, no negative weights, no NaN,
//   - symmetric (opts.Symmetric==true / mustEnforceSymmetry(opts) == true),
//   - if opts.RunMetricClosure==false: no +Inf edges allowed.
//
// Options notes:
//   - opts.StartVertex fixes the start/closure of the cycle.
//   - opts.MatchingAlgo selects between exact matching mode and GreedyMatch.
//   - Greedy fallback is allowed only when MatchingFallbackPolicy explicitly enables it.
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
//   - Exact-small MWPM may justify the 1.5 metadata for small odd sets.
//   - Large odd sets return ErrMatchingUnavailable until a true Blossom engine is implemented,
//     unless explicit greedy fallback is selected.
package tsp

import (
	"errors"

	"github.com/katalvlaran/lvlath/matrix"
)

// approxMeta records approximation-specific facts observed inside the Christofides kernel.
// It is intentionally private because callers should consume this information through TSPResult.
//
// Implementation:
//   - Stage 1: Start without a formal approximation ratio.
//   - Stage 2: Record the 1.5 ratio only after exact MWPM succeeds.
//   - Stage 3: Preserve fallback warning sentinels for TSPResult.Warnings.
//
// Behavior highlights:
//   - MatchingFallback is kernel-origin metadata, never facade inference.
//   - ProvenRatio is 1.5 only after exact MWPM success.
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

// newApproxMeta initializes conservative approximation metadata.
// Implementation:
//   - Stage 1: Start with NoApproximationRatio.
//   - Stage 2: Let the matching branch record exact MWPM success explicitly.
//
// Behavior highlights:
//   - No branch receives a provisional ratio.
//   - GreedyMatch and fallback remain ratio-free.
//
// Inputs:
//   - opts: solver options; currently unused but kept for signature stability.
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

// recordExactMWPM records a successful exact minimum-weight perfect matching.
// This is the only metadata transition that enables the formal Christofides ratio.
//
// Implementation:
//   - Stage 1: Clear matching fallback.
//   - Stage 2: Store ChristofidesApproximationRatio.
//
// Behavior highlights:
//   - Does not modify warnings.
//   - Does not infer exactness from Options.
//
// Inputs:
//   - None.
//
// Returns:
//   - None.
//
// Errors:
//   - None.
//
// Determinism:
//   - Pure metadata mutation.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Use only after blossomMatch succeeds.
//
// AI-Hints:
//   - Do not call this after GreedyMatch or greedy fallback.
func (m *approxMeta) recordExactMWPM() {
	m.matchingFallback = false
	m.provenRatio = ChristofidesApproximationRatio
}

// recordGreedyFallback records BlossomMatch -> GreedyMatch degradation.
// Implementation:
//   - Stage 1: Set MatchingFallback.
//   - Stage 2: Clear the formal approximation ratio.
//   - Stage 3: Attach ErrMatchingFallback as a non-fatal warning.
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
//   - Do not use ErrMatchingNotImplemented as the public warning for fallback.
//   - Do not keep the 1.5 ratio after any greedy fallback.
func (m *approxMeta) recordGreedyFallback() {
	m.matchingFallback = true
	m.provenRatio = NoApproximationRatio
	m.warnings = append(m.warnings, ErrMatchingFallback)
}

// ChristofidesSolve runs the Christofides pipeline and publishes canonical result metadata.
// It is the direct solver entrypoint for symmetric metric TSP instances when callers do
// not need SolveMatrix adapter features such as ID attachment or metric closure.
//
// Implementation:
//   - Stage 1: Run the existing Christofides pipeline and collect approximation metadata.
//   - Stage 2: Publish a detached TSPResult with matching and approximation facts.
//   - Stage 3: Preserve warning sentinels for caller classification through errors.Is.
//
// Behavior highlights:
//   - Claims ApproximationRatio=1.5 only when the matching metadata proves it.
//   - Clears the formal approximation ratio when greedy matching is used.
//   - Does not run graph or matrix adapter logic.
//
// Inputs:
//   - dist: final symmetric complete distance matrix.
//   - opts: Christofides policy.
//
// Returns:
//   - *TSPResult: canonical approximation result.
//   - error: nil on success or a sentinel-classified failure.
//
// Errors:
//   - ErrStartOutOfRange.
//   - ErrIncompleteGraph from MST, matching, or tour-cost checks.
//   - ErrInvalidOptions for invalid matching policy.
//   - ErrNonEulerian from malformed Eulerian multigraph state.
//   - ErrInvalidTour, ErrNaNInf, ErrNegativeWeight from cost/tour helpers.
//
// Determinism:
//   - MST, matching fallback, Eulerian walk, shortcutting, and canonicalization use fixed orders.
//
// Complexity:
//   - Time O(n^2) with greedy matching on dense instances.
//   - Space O(n+E) beyond the input matrix.
//
// Notes:
//   - Use SolveMatrix when IDs or direct matrix metric closure are required.
//   - Matching fallback policy is repaired in the matching correctness phase.
//
// AI-Hints:
//   - Do not claim 1.5 approximation when MatchingFallback is true.
//   - Do not infer matching fallback from Options outside the matching kernel.
func ChristofidesSolve(dist matrix.Matrix, opts Options) (*TSPResult, error) {
	minimal, approx, err := tspApprox(dist, opts)
	if err != nil {
		return nil, err
	}

	meta := newSolveMeta(Christofides)
	meta.applyApprox(approx)

	result := publishTSPResult(minimal, nil, opts, meta, false)
	result.Algorithm = Christofides

	return result, nil
}

// TSPApprox runs the Christofides pipeline and returns the legacy minimal result.
// It is kept as a compatibility/public kernel wrapper; canonical metadata is published
// by ChristofidesSolve and SolveMatrix.
//
// Implementation:
//   - Stage 1: Delegate to ChristofidesSolve.
//   - Stage 2: Project TSPResult to TSResult through Minimal.
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
//   - Do not add matching metadata inference here; use TSPResult from ChristofidesSolve.
//
// Deprecated: use ChristofidesSolve or SolveMatrix.
func TSPApprox(dist matrix.Matrix, opts Options) (TSResult, error) {
	result, err := ChristofidesSolve(dist, opts)
	if result == nil {
		return TSResult{}, err
	}

	return result.Minimal(), err
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
//   - Do not treat ErrMatchingUnavailable as non-fatal unless fallback policy allows it.
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
			if !errors.Is(matchErr, ErrMatchingNotImplemented) && !errors.Is(matchErr, ErrMatchingUnavailable) {
				return TSResult{}, meta, matchErr
			}

			if opts.MatchingFallbackPolicy != MatchingFallbackGreedy {
				return TSResult{}, meta, ErrMatchingUnavailable
			}
			// Deterministic and explicitly requested degradation. The resulting tour
			// remains valid when all later stages succeed, but no 1.5 ratio is claimed.
			if err = greedyMatchAtomic(odd, dist, mstAD); err != nil {
				return TSResult{}, meta, err
			}
			meta.recordGreedyFallback()
		} else {
			meta.recordExactMWPM()
		}

	case GreedyMatch:
		if err = greedyMatchAtomic(odd, dist, mstAD); err != nil {
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
