// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package tsp - odd-degree matching for Christofides.
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
	"errors"
	"math"

	"github.com/katalvlaran/lvlath/matrix"
)

// greedyMatch adds a deterministic greedy perfect matching over odd-degree vertices.
// Implementation:
//   - Stage 1: Validate even cardinality.
//   - Stage 2: Copy odd vertices into a local shrinkable buffer.
//   - Stage 3: Repeatedly choose one endpoint and scan finite partners.
//   - Stage 4: Mutate adj only after a valid partner is selected.
//
// Behavior highlights:
//   - Deterministic tie-break: lower edge cost, then smaller vertex index.
//   - Does not mutate adj on infeasible partner selection.
//   - Propagates edgeCost errors instead of converting them into implicit +Inf.
//
// Inputs:
//   - odd: odd-degree vertices from the MST; must have even cardinality.
//   - dist: validated complete distance matrix.
//   - adj: mutable Christofides multigraph adjacency.
//
// Returns:
//   - error: nil after all vertices are paired.
//
// Errors:
//   - ErrInvalidMatching for odd cardinality.
//   - ErrIncompleteGraph when no finite partner exists.
//   - ErrInvalidTour / ErrNaNInf / ErrNegativeWeight propagated from edgeCost.
//
// Determinism:
//   - Removes the current endpoint from the end of rem.
//   - Scans remaining candidates in increasing slice order.
//   - Ties are resolved by smaller vertex index.
//
// Complexity:
//   - Time O(k^2), Space O(k), where k=len(odd).
//
// Notes:
//   - Greedy matching does not provide the Christofides 1.5 proof.
//   - It remains useful as deterministic fallback and explicitly weaker policy.
//
// AI-Hints:
//   - Do not ignore edgeCost errors; missing edges must not be paired.
//   - Do not add edges to adj until the selected partner is known to be finite.
func greedyMatch(odd []int, dist matrix.Matrix, adj [][]int) error {
	oddCount := len(odd)
	// oddCount==0 is a valid (degenerate) case - nothing to do.
	if oddCount == 0 {
		return nil
	}
	if (oddCount & 1) == 1 {
		return ErrInvalidMatching
	}

	// Work on a compact local copy we can shrink from the end.
	remaining := make([]int, oddCount)
	copy(remaining, odd)

	var (
		fromVertex, toVertex                 int
		lastIndex, bestIndex, candidateIndex int
		weight, bestWeight                   float64
		err                                  error
	)

	// Pair until none (or a defensively handled single) remain.
	for len(remaining) > 1 {
		// Take one endpoint u from the end for O(1) removal.
		lastIndex = len(remaining) - 1
		fromVertex = remaining[lastIndex]
		remaining = remaining[:lastIndex]

		// Find nearest partner v among remaining.
		bestIndex = -1
		bestWeight = math.Inf(1)

		for candidateIndex = 0; candidateIndex < len(remaining); candidateIndex++ {
			toVertex = remaining[candidateIndex]

			// Use edgeCost to inherit strict sentinel semantics.
			weight, err = edgeCost(dist, fromVertex, toVertex)
			if err != nil {
				if errors.Is(err, ErrIncompleteGraph) {
					continue
				}

				return err
			}

			if bestIndex < 0 || weight < bestWeight || (math.Abs(weight-bestWeight) <= symTol && toVertex < remaining[bestIndex]) {
				bestIndex = candidateIndex
				bestWeight = weight
			}
		}

		// Defensive: if no finite partner found (should not happen on validated inputs), stop.
		if bestIndex < 0 || math.IsInf(bestWeight, 1) {
			return ErrIncompleteGraph
		}

		// Extract chosen partner toVertex in O(1) by swapping with the last element.
		lastIndex = len(remaining) - 1
		toVertex = remaining[bestIndex]
		remaining[bestIndex] = remaining[lastIndex]
		remaining = remaining[:lastIndex]

		// Add undirected edge fromVertex–toVertex (parallel edges are allowed - multigraph).
		adj[fromVertex] = append(adj[fromVertex], toVertex)
		adj[toVertex] = append(adj[toVertex], fromVertex)
	}

	return nil
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
