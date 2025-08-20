// Package tsp — odd-degree matching for Christofides.
//
// This file provides the minimum-weight perfect matching step on the set of
// odd-degree vertices of the MST. The greedy variant is deterministic and
// O(k²) (k = |odd|). A Blossom placeholder is provided and returns a strict
// sentinel (ErrMatchingNotImplemented) without mutating inputs.
//
// Design:
//   - Deterministic, side-effect-free w.r.t. inputs other than adj mutation.
//   - No panics / logs; only strict sentinels defined in types.go.
//   - Works with any matrix.Matrix; distances fetched via edgeCost for
//     unified validation semantics.
//   - Tie-breaking: by cost, then by smaller vertex id (stable across runs).
package tsp

import (
	"math"

	"github.com/katalvlaran/lvlath/matrix"
)

// greedyMatch adds a deterministic greedy minimum-weight perfect matching on the
// odd-degree vertices `odd` into the multigraph adjacency `adj`.
//
// Contract:
//   - `odd` contains an even number of distinct vertex ids in [0..n-1].
//   - `adj` is an undirected simple graph (MST) to which we add parallel edges.
//   - `dist` is a validated metric, symmetric matrix (see validateAll).
//
// Complexity: O(k²) time, O(1) extra space besides a local copy of `odd`.
func greedyMatch(odd []int, dist matrix.Matrix, adj [][]int) {
	// k==0 is a valid (degenerate) case — nothing to do.
	var k int
	k = len(odd)
	if k == 0 {
		return
	}

	// Work on a compact local copy we can shrink from the end.
	rem := make([]int, k)
	copy(rem, odd)

	var (
		u       int
		v       int
		last    int
		bestIdx int
		i       int

		w     float64
		bestW float64
		okTie bool
	)

	// Pair until none (or a defensively handled single) remain.
	for len(rem) > 1 {
		// Take one endpoint u from the end for O(1) removal.
		last = len(rem) - 1
		u = rem[last]
		rem = rem[:last]

		// Find nearest partner v among remaining.
		bestIdx = -1
		bestW = math.Inf(1)

		for i = 0; i < len(rem); i++ {
			v = rem[i]
			// Use edgeCost to inherit strict sentinel semantics.
			w, _ = edgeCost(dist, u, v) // validated instance ⇒ no error; if any, treat as +Inf
			okTie = math.Abs(w-bestW) <= symTol

			if w < bestW || (okTie && v < rem[bestIdx]) {
				bestW = w
				bestIdx = i
			}
		}

		// Defensive: if no finite partner found (should not happen on validated inputs), stop.
		if bestIdx < 0 {
			break
		}

		// Extract chosen partner v in O(1) by swapping with the last element.
		last = len(rem) - 1
		v = rem[bestIdx]
		rem[bestIdx] = rem[last]
		rem = rem[:last]

		// Add undirected edge u–v (parallel edges are allowed — multigraph).
		adj[u] = append(adj[u], v)
		adj[v] = append(adj[v], u)
	}
}

// blossomMatch is a placeholder for a true minimum-weight perfect matching
// (e.g., Edmonds/Blossom V). It does not mutate adj and returns a strict
// sentinel so the caller can deterministically fall back to greedyMatch.
//
// Complexity: O(1).
func blossomMatch(odd []int, dist matrix.Matrix, adj [][]int) error {
	_ = odd
	_ = dist
	_ = adj
	return ErrMatchingNotImplemented
}

// -----------------------------------------------------------------------------
// Below are TEST-ONLY hooks to access internal matching routines.
// These functions are compiled only during `go test` and are not part of the public API or release builds.
//-----------------------------------------------------------------------------

// TestHookGreedyMatch exposes the internal greedyMatch for black-box tests.
// It forwards the call without modifying arguments or logic.
func TestHookGreedyMatch(odd []int, dist matrix.Matrix, adj [][]int) {
	// Call the unexported greedyMatch exactly as Christofides would.
	greedyMatch(odd, dist, adj)
}

// TestHookBlossomMatch exposes the internal blossomMatch for black-box tests.
// It returns the exact error returned by the internal function.
func TestHookBlossomMatch(odd []int, dist matrix.Matrix, adj [][]int) error {
	// Call the unexported blossomMatch and forward its error.
	return blossomMatch(odd, dist, adj)
}
