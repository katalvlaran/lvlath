package tsp

import "errors"

// ErrTSPIncompleteGraph is returned when the distance matrix does not
// admit any Hamiltonian cycle (i.e., some required edge is missing).
var ErrTSPIncompleteGraph = errors.New("tsp: incomplete distance matrix")

// TSResult holds the outcome of a TSP solver.
type TSResult struct {
	// Tour is the sequence of vertex indices, starting and ending at 0.
	// For n vertices, len(Tour) == n+1 and Tour[0]==Tour[n]==0.
	Tour []int

	// Cost is the total distance of the cycle.
	Cost float64
}
