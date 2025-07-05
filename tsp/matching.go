package tsp

import (
	"math"

	"github.com/katalvlaran/lvlath/matrix"
)

// GreedyMatch performs a simple minimum‐weight perfect matching on the odd‐degree
// vertex set. It repeatedly pairs each remaining odd vertex with its nearest
// neighbor, adding that edge to the multigraph adjacency.
//
// Complexity: O(k²), where k = len(odd).
func greedyMatch(odd []int, dist [][]float64, adj [][]int) {
	// work on a local copy of the odd slice
	remaining := append([]int(nil), odd...)
	for len(remaining) > 1 {
		// pick the first vertex
		u := remaining[0]
		remaining = remaining[1:]
		// find its closest partner v
		bestIdx, bestD := -1, math.Inf(1)
		for i, v := range remaining {
			if d := dist[u][v]; d < bestD {
				bestD, bestIdx = d, i
			}
		}
		// partner is remaining[bestIdx]
		v := remaining[bestIdx]
		// record the matching edge in adj (multigraph)
		adj[u] = append(adj[u], v)
		adj[v] = append(adj[v], u)
		// remove v from remaining
		remaining = append(remaining[:bestIdx], remaining[bestIdx+1:]...)
	}
}

// BlossomMatch is a placeholder for a true minimum‐weight perfect matching
// (e.g. Edmonds’ Blossom V algorithm). Currently it falls back to GreedyMatch
// to maintain correctness, then returns ErrNotImplemented to flag that a
// higher‐quality implementation is pending.
func blossomMatch(odd []int, dist [][]float64, adj [][]int) error {
	greedyMatch(odd, dist, adj)
	return matrix.ErrNotImplemented
}
