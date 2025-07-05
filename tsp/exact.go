package tsp

import (
	"fmt"
	"math"
)

// TSPExact solves the Travelling Salesman Problem exactly on a given
// distance matrix using the Held–Karp dynamic‐programming algorithm.
// It requires a complete, symmetric, non‐negative distance matrix.
//
// Input:
//
//	dist — an n×n matrix where dist[i][j] is the cost from vertex i to j.
//	        A value of math.Inf(1) signals “no direct edge” (missing).
//	        The diagonal dist[i][i] must be zero.
//
// Options:
//
//	opts — currently unused; provided for API consistency.
//
// Returns:
//
//	TSResult{Tour, Cost} on success, where
//	  Tour is a slice of length n+1 starting and ending at vertex 0,
//	  and Cost is the total cycle cost.
//	ErrBadInput if the matrix is invalid (empty, ragged, negative weights,
//	  self‐distance ≠0, asymmetry).
//	ErrIncompleteGraph if no Hamiltonian cycle exists (disconnected).
//
// Time complexity:  O(n² · 2ⁿ) — DP over all subsets × endpoints.
// Memory complexity: O(n · 2ⁿ) — storing cost for each subset & endpoint.
func TSPExact(dist [][]float64, _ Options) (TSResult, error) {
	// Validate: non‐empty matrix
	n := len(dist)
	if n == 0 {
		return TSResult{}, ErrBadInput
	}

	// Validate: square matrix, zero diagonal, non‐negative & symmetric entries
	for i := 0; i < n; i++ {
		// Check each row has length n
		if len(dist[i]) != n {
			return TSResult{}, fmt.Errorf(
				"tsp: row %d length %d, want %d: %w",
				i, len(dist[i]), n, ErrBadInput,
			)
		}
		// Diagonal must be zero
		if dist[i][i] != 0 {
			return TSResult{}, fmt.Errorf(
				"tsp: dist[%d][%d]=%v; self-distance must be 0: %w",
				i, i, dist[i][i], ErrBadInput,
			)
		}
		for j := 0; j < n; j++ {
			// No negative weights allowed
			if dist[i][j] < 0 {
				return TSResult{}, fmt.Errorf(
					"tsp: negative distance dist[%d][%d]=%v: %w",
					i, j, dist[i][j], ErrBadInput,
				)
			}
			// Matrix must be symmetric
			if dist[i][j] != dist[j][i] {
				return TSResult{}, fmt.Errorf(
					"tsp: asymmetry at dist[%d][%d]=%v vs dist[%d][%d]=%v: %w",
					i, j, dist[i][j], j, i, dist[j][i], ErrBadInput,
				)
			}
		}
	}

	// Precompute bitmask for starting vertex (0)
	const startVertex = 0
	startMask := 1 << startVertex // mask with only bit0 set
	allMask := (1 << uint(n)) - 1 // mask with first n bits set

	// Allocate DP table: dp[mask][j] = min cost to start at 0, visit mask, end at j
	dp := make([][]float64, allMask+1) // O(2ⁿ) rows
	parent := make([][]int, allMask+1) // to reconstruct path
	for mask := 0; mask <= allMask; mask++ {
		dp[mask] = make([]float64, n) // each has n endpoints
		parent[mask] = make([]int, n) // store predecessor
		for j := 0; j < n; j++ {
			dp[mask][j] = math.Inf(1) // initialize cost to +∞
			parent[mask][j] = -1      // no predecessor by default
		}
	}
	// Base case: only vertex 0 visited, cost to be at 0 is zero
	dp[startMask][startVertex] = 0

	// Build up DP for all subsets that include vertex 0
	for mask := 0; mask <= allMask; mask++ {
		// Skip subsets not containing start
		if mask&startMask == 0 {
			continue
		}
		// Try to end at each vertex j ≠ 0 that is in subset
		for j := 1; j < n; j++ {
			if mask&(1<<uint(j)) == 0 {
				continue // j not in this subset
			}
			// Remove j from subset to form prevMask
			prevMask := mask ^ (1 << uint(j))
			// Try all possible k in prevMask that transitions to j
			for k := 0; k < n; k++ {
				if prevMask&(1<<uint(k)) == 0 {
					continue // k not in prevMask
				}
				costKtoJ := dist[k][j]
				if math.IsInf(costKtoJ, 1) {
					continue // no edge k→j
				}
				// Candidate cost: cost to reach k + edge k→j
				cand := dp[prevMask][k] + costKtoJ
				// Update if better
				if cand < dp[mask][j] {
					dp[mask][j] = cand
					parent[mask][j] = k
				}
			}
		}
	}

	// Close the tour: return from last vertex j back to start
	bestCost := math.Inf(1)
	last := -1
	for j := 1; j < n; j++ {
		backCost := dist[j][startVertex]
		if math.IsInf(backCost, 1) {
			continue // no edge back to start
		}
		total := dp[allMask][j] + backCost
		if total < bestCost {
			bestCost = total
			last = j
		}
	}
	// If no valid tour found, graph is incomplete
	if last < 0 || math.IsInf(bestCost, 1) {
		return TSResult{}, ErrIncompleteGraph
	}

	// Reconstruct tour by walking back through parent table
	tour := make([]int, n+1) // n+1 to return to start
	tour[n] = startVertex    // end at start
	mask := allMask
	j := last
	for i := n - 1; i >= 1; i-- {
		tour[i] = j             // place j in tour
		prev := parent[mask][j] // predecessor of j
		mask ^= 1 << uint(j)    // remove j from mask
		j = prev                // move to predecessor
	}
	tour[0] = startVertex // start at 0

	// Return result with optimal tour and cost
	return TSResult{
		Tour: tour,
		Cost: bestCost,
	}, nil
}
