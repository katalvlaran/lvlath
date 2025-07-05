package tsp

import (
	"fmt"  // formatted I/O for error messages
	"math" // math constants and functions

	"github.com/katalvlaran/lvlath/matrix" // matrix helpers: MST, matching, Eulerian circuit
)

// TSPApprox computes a 1.5-approximation to the Travelling Salesman Problem
// on a complete, symmetric, metric distance matrix using Christofides’ algorithm.
// It returns a Hamiltonian cycle starting and ending at opts.StartVertex.
//
// Input:
//
//	dist — an n×n [][]float64 where
//	       • dist[i][i] == 0
//	       • dist[i][j] ≥ 0
//	       • dist[i][j] == dist[j][i]
//	       • math.Inf(1) signals “no edge” (incomplete graph).
//
// Options:
//
//	opts.StartVertex    — index in [0..n-1] to start/end the tour.
//	opts.MatchingAlgo   — GreedyMatch or BlossomMatch for the odd-vertex matching.
//	opts.BoundAlgo      — reserved for future B&B selection.
//
// Returns:
//
//	TSResult{Tour, Cost} on success,
//	ErrNonSquare      if dist is not square,
//	ErrNegativeWeight if any dist[i][j] < 0,
//	ErrNonZeroDiagonal if any dist[i][i] != 0,
//	ErrAsymmetry      if dist[i][j] != dist[j][i],
//	ErrIncompleteGraph if no Hamiltonian cycle exists.
func TSPApprox(dist [][]float64, opts Options) (TSResult, error) {
	// --- 1. Dimension & symmetry validation ---
	n := len(dist) // number of vertices
	if n == 0 {    // empty matrix
		return TSResult{}, ErrNonSquare
	}
	for i := 0; i < n; i++ {
		if len(dist[i]) != n { // each row must have length n
			return TSResult{}, ErrNonSquare
		}
		if dist[i][i] != 0 { // self‐distance must be zero
			return TSResult{}, ErrNonZeroDiagonal
		}
		for j := i + 1; j < n; j++ {
			if dist[i][j] < 0 { // negative distances forbidden
				return TSResult{}, ErrNegativeWeight
			}
			if dist[i][j] != dist[j][i] { // symmetry requirement
				return TSResult{}, ErrAsymmetry
			}
		}
	}

	// --- 2. Validate StartVertex ---
	if opts.StartVertex < 0 || opts.StartVertex >= n {
		return TSResult{}, fmt.Errorf("tsp: StartVertex %d out of range [0,%d): %w",
			opts.StartVertex, n, ErrBadInput)
	}

	// --- 3. Build Minimum Spanning Tree (MST) ---
	// Prim’s algorithm via matrix.MinimumSpanningTree (O(n²))
	_, mstAdj, err := MinimumSpanningTree(dist)
	if err != nil {
		return TSResult{}, err // could be ErrIncompleteGraph
	}

	// --- 4. Find odd‐degree vertices in MST ---
	var odd []int
	for v := 0; v < n; v++ {
		if len(mstAdj[v])%2 == 1 {
			odd = append(odd, v) // collect odd-degree nodes
		}
	}

	// --- 5. Perfect matching on odd vertices ---
	switch opts.MatchingAlgo {
	case BlossomMatch:
		if matchErr := blossomMatch(odd, dist, mstAdj); matchErr != nil && matchErr != matrix.ErrNotImplemented {
			return TSResult{}, matchErr
		}
	case GreedyMatch:
		greedyMatch(odd, dist, mstAdj)
	default:
		// default to true blossom if available
		greedyMatch(odd, dist, mstAdj)
	}

	// --- 6. Compute Eulerian circuit on the multigraph ---
	// Hierholzer’s algorithm via matrix.EulerianCircuit (O(E))
	euler := EulerianCircuit(mstAdj, opts.StartVertex)

	// --- 7. Shortcut to Hamiltonian cycle ---
	visit := make([]bool, n)     // mark visited vertices
	cycle := make([]int, 0, n+1) // pre-allocate n+1 spots
	for _, v := range euler {
		if !visit[v] { // include only first visit
			cycle = append(cycle, v)
			visit[v] = true
		}
	}
	cycle = append(cycle, opts.StartVertex) // close the cycle

	// --- 8. Compute total tour cost ---
	var cost float64
	for k := 0; k < len(cycle)-1; k++ {
		u, v := cycle[k], cycle[k+1]
		d := dist[u][v]
		if math.IsInf(d, 1) { // missing edge ⇒ no Hamiltonian cycle
			return TSResult{}, ErrIncompleteGraph
		}
		cost += d
	}
	// Round cost to nanosecond precision to avoid floating-point noise
	cost = math.Round(cost*1e9) / 1e9

	return TSResult{Tour: cycle, Cost: cost}, nil
}
