package tsp

import (
	"errors"
	"fmt"
	"math"
)

// TSPExact solves the Travelling Salesman Problem exactly on a given
// distance matrix using the Held–Karp dynamic‐programming algorithm.
//
// The input is an n×n matrix dist, where dist[i][j] is the cost to go
// from vertex i to j.  A value of math.Inf(1) represents “no edge.”
// The diagonal dist[i][i] must be zero.
//
// It returns a TSResult containing:
//   - Tour: a slice of length n+1 of vertex indices, starting and ending at 0.
//   - Cost: total cycle cost.
//
// Or ErrTSPIncompleteGraph if no Hamiltonian cycle exists.
//
// Time complexity:  O(n² · 2ⁿ)
// Memory complexity: O(n · 2ⁿ)
//
// This implementation indexes subsets by bitmasks from 0…(1<<n)-1.
// dp[mask][j] = minimum cost to start at 0, visit exactly the vertices in
//
//	mask (mask&1<<0 != 0), and end at j.
//
// After filling dp, we “close” the tour by returning from j back to 0.
func TSPExact(dist [][]float64) (TSResult, error) {
	n := len(dist)
	if n == 0 {
		return TSResult{}, errors.New("tsp: empty matrix")
	}
	// --- 1. Validate input matrix ---
	for i := 0; i < n; i++ {
		if len(dist[i]) != n {
			return TSResult{}, fmt.Errorf("tsp: row %d length %d, want %d", i, len(dist[i]), n)
		}
		if dist[i][i] != 0 {
			return TSResult{}, fmt.Errorf("tsp: dist[%d][%d]=%v; self-distance must be 0", i, i, dist[i][i])
		}
	}

	// Maximum subset mask: all n bits set.
	allMask := (1 << n) - 1

	// --- 2. Allocate DP and parent tables ---
	dp := make([][]float64, 1<<n)
	parent := make([][]int, 1<<n)
	for mask := 0; mask <= allMask; mask++ {
		dp[mask] = make([]float64, n)
		parent[mask] = make([]int, n)
		for j := 0; j < n; j++ {
			dp[mask][j] = math.Inf(1) // initialize to +∞
			parent[mask][j] = -1      // “no predecessor”
		}
	}
	// Base case: mask with only bit 0 set, cost to be at 0 is 0.
	startMask := 1 << 0
	dp[startMask][0] = 0

	// --- 3. Fill DP for all masks that include vertex 0 ---
	for mask := 0; mask <= allMask; mask++ {
		// skip subsets that don't include the start vertex 0
		if mask&startMask == 0 {
			continue
		}
		// for each possible endpoint j ≠ 0 in this subset
		for j := 1; j < n; j++ {
			if mask&(1<<j) == 0 {
				continue // j not in subset
			}
			// previous subset without j
			prevMask := mask ^ (1 << j)
			// try all possible k that led to j
			for k := 0; k < n; k++ {
				if prevMask&(1<<k) == 0 {
					continue // k not in prevMask
				}
				c := dist[k][j]
				if math.IsInf(c, 1) {
					continue // no edge k→j
				}
				cand := dp[prevMask][k] + c
				if cand < dp[mask][j] {
					dp[mask][j] = cand
					parent[mask][j] = k
				}
			}
		}
	}

	// --- 4. Close the tour by returning to 0 ---
	bestCost := math.Inf(1)
	last := -1
	for j := 1; j < n; j++ {
		c := dist[j][0]
		if math.IsInf(c, 1) {
			continue // no edge back to start
		}
		total := dp[allMask][j] + c
		if total < bestCost {
			bestCost = total
			last = j
		}
	}
	if last < 0 || math.IsInf(bestCost, 1) {
		return TSResult{}, ErrTSPIncompleteGraph
	}

	// --- 5. Reconstruct tour from parent table ---
	tour := make([]int, n+1)
	tour[n] = 0 // return to start
	mask := allMask
	j := last
	for i := n - 1; i >= 1; i-- {
		tour[i] = j
		p := parent[mask][j]
		mask ^= 1 << j
		j = p
	}
	tour[0] = 0

	return TSResult{Tour: tour, Cost: bestCost}, nil
}
