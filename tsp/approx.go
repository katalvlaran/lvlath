// Package tsp — Christofides 1.5-approximation.
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

// TSPApprox runs Christofides on a symmetric, metric instance.
//
// Note: SolveWithMatrix already validated Options + Matrix. Here we keep only
// lightweight guards that do not duplicate the full O(n^2) validation.
func TSPApprox(dist matrix.Matrix, opts Options) (TSResult, error) {
	// Lightweight start-range guard (n already known to be ≥ 2 in the dispatcher).
	n := dist.Rows()
	if err := validateStartVertex(n, opts.StartVertex); err != nil {
		return TSResult{}, err
	}

	// 1) Minimum Spanning Tree on the metric graph.
	//    Returns total weight (unused here) and a simple-graph adjacency (no multi-edges).
	mstW, mstAD, err := MinimumSpanningTree(dist) // O(n^2) Prim (see mst.go)
	if err != nil {
		return TSResult{}, err
	}
	_ = mstW // MST weight is not required by Christofides beyond building the multigraph.

	// 2) Collect odd-degree vertices of the MST.
	//    V has odd degree iff degree(v) mod 2 == 1. Fast parity check via bit-test.
	//    len(mstAD[v])&1 == 1  ⇔ degree(v) is odd (LSB set).
	odd := make([]int, 0, n/2+1) // conservative capacity avoids reslices
	var v int                    // loop iterator
	for v = 0; v < n; v++ {
		if (len(mstAD[v]) & 1) == 1 {
			odd = append(odd, v)
		}
	}

	// 3) Add a minimum-weight perfect matching among odd-degree vertices.
	//    We modify the adjacency in-place, effectively forming the Eulerian multigraph.
	switch opts.MatchingAlgo {
	case BlossomMatch:
		if mErr := blossomMatch(odd, dist, mstAD); mErr != nil {
			if errors.Is(mErr, ErrMatchingNotImplemented) {
				// Deterministic and safe fallback; preserves pipeline validity.
				greedyMatch(odd, dist, mstAD)
			} else {
				return TSResult{}, mErr
			}
		}
	case GreedyMatch:
		greedyMatch(odd, dist, mstAD)
	default:
		// Strict but user-friendly: unknown enum ⇒ deterministic greedy.
		greedyMatch(odd, dist, mstAD)
	}

	// 4) Eulerian circuit on the multigraph (Hierholzer).
	//    Returns a closed walk that starts at opts.StartVertex and finishes at it.
	//    The circuit cost is O(E), where E is the number of (multi)edges.
	euler := EulerianCircuit(mstAD, opts.StartVertex)

	// 5) Shortcut revisits to obtain a Hamiltonian tour; then canonicalize direction.
	tour, err := ShortcutEulerianToHamiltonian(euler, n, opts.StartVertex)
	if err != nil {
		return TSResult{}, err
	}
	_ = CanonicalizeOrientationInPlace(tour)

	// 6) Compute the stabilized tour cost with strict edge validation.
	//    tourCost checks Inf/NaN/negatives defensively and rounds to 1e-9.
	cost, err := TourCost(dist, tour)
	if err != nil {
		return TSResult{}, err
	}

	// Final invariant check (O(n)) — inexpensive, helps catch wiring mistakes early.
	if verr := ValidateTour(tour, n, opts.StartVertex); verr != nil {
		return TSResult{}, verr
	}

	return TSResult{Tour: tour, Cost: cost}, nil
}
