// Package tsp defines common types, constants, and errors used by the Traveling Salesman Problem solvers.
//
// It provides unified, reusable structures and clear error handling to facilitate accurate,
// mathematically sound, and consistent implementations of exact and approximate TSP algorithms.
package tsp

import "errors"

// Sentinel errors explicitly indicating specific failure reasons encountered by TSP algorithms.
var ()

// Sentinel errors for precise input validation failures.
var (
	// ErrNonSquare indicates the distance matrix is not square.
	ErrNonSquare = errors.New("tsp: matrix is not square")

	// ErrNegativeWeight indicates a negative distance was encountered.
	ErrNegativeWeight = errors.New("tsp: negative distance encountered")

	// ErrAsymmetry indicates dist[i][j] != dist[j][i].
	ErrAsymmetry = errors.New("tsp: asymmetric distance matrix")

	// ErrNonZeroDiagonal indicates some dist[i][i] ≠ 0.
	ErrNonZeroDiagonal = errors.New("tsp: non-zero self-distance")

	// ErrIncompleteGraph is returned when the provided distance matrix does not allow a Hamiltonian cycle
	// due to one or more missing edges (represented as math.Inf(1)).
	ErrIncompleteGraph = errors.New("tsp: incomplete distance matrix (no Hamiltonian cycle possible)")

	// ??
	ErrDimensionMismatch = errors.New("??")

	// deprecated
	// ErrBadInput indicates invalid input data such as asymmetric matrices, negative edge weights,
	// or any violation of algorithm prerequisites.
	ErrBadInput = errors.New("tsp: invalid input (negative distances or asymmetric matrix)")
)

//!

// MatchingAlgo selects the perfect matching strategy in Christofides.
type MatchingAlgo int

const (
	// GreedyMatch pairs odd-degree vertices by nearest-neighbor (fast, weaker bound).
	GreedyMatch MatchingAlgo = iota

	// BlossomMatch uses Edmonds’ blossom algorithm for true minimum-weight matching
	// (ensures the 1.5× approximation guarantee).
	BlossomMatch
)

// BoundAlgo reserved for future Branch & Bound strategy selection.
type BoundAlgo int

const (
	// NoBound currently unused.
	NoBound BoundAlgo = iota

	// SimpleBound placeholder for a basic bound heuristic.
	SimpleBound
)

// TSResult encapsulates the result from a TSP solver algorithm.
type TSResult struct {
	// Tour is an ordered sequence of vertex indices representing the Hamiltonian cycle.
	// Tour always starts and ends at vertex 0.
	// Thus, for n vertices, len(Tour) == n + 1 and Tour[0] == Tour[n] == 0.
	Tour []int

	// Cost represents the total distance traveled across the complete Hamiltonian cycle.
	// It reflects the sum of the distances of edges forming the cycle, adhering precisely
	// to the provided distance matrix.
	Cost float64
}

// Options defines configurable parameters for TSP solver algorithms.
type Options struct {
	// StartVertex allows the tour to start at an arbitrary index [0..n-1]
	// instead of always vertex 0.
	StartVertex int

	// MatchingAlgo chooses between GreedyMatch or BlossomMatch for TSPApprox.
	MatchingAlgo MatchingAlgo

	// BoundAlgo will select bounding strategy in future Branch & Bound solvers.
	BoundAlgo BoundAlgo
}

// DefaultOptions provides a convenient factory method that returns a default Options struct instance.
// start at vertex 0, use Blossom matching for Christofides.
func DefaultOptions() Options {
	return Options{
		StartVertex:  0,
		MatchingAlgo: BlossomMatch,
		BoundAlgo:    NoBound,
	}
}
