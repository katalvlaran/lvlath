// Package tsp defines common types, configuration options, and sentinel errors used by
// exact and approximate Traveling Salesman Problem (TSP) solvers.
//
// Design goals:
//   - Mathematical rigor: precise, specialized errors; explicit invariants for tours.
//   - Extensibility: a single Options struct covers both exact and heuristic solvers.
//   - Determinism: all random-driven heuristics are controlled by a Seed.
//   - Zero surprises: sensible defaults (Christofides + optional 2-Opt post-pass).
package tsp

import (
	"errors"
	"time"
)

//–––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––
// Sentinel errors (validation, feasibility, algorithm governance)
//–––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––

// Validation / input-shape errors. Do not wrap with fmt.Errorf where a sentinel suffices.
var (
	// ErrNonSquare indicates the distance matrix is not square.
	ErrNonSquare = errors.New("tsp: matrix is not square")

	// ErrNegativeWeight indicates a negative distance was encountered.
	ErrNegativeWeight = errors.New("tsp: negative distance encountered")

	// ErrAsymmetry indicates dist[i][j] != dist[j][i] for a symmetric-TSP solver.
	ErrAsymmetry = errors.New("tsp: asymmetric distance matrix")

	// ErrNonZeroDiagonal indicates some dist[i][i] ≠ 0.
	ErrNonZeroDiagonal = errors.New("tsp: non-zero self-distance")

	// ErrIncompleteGraph is returned when no Hamiltonian cycle exists
	// (one or more edges missing, represented by math.Inf(1)).
	ErrIncompleteGraph = errors.New("tsp: incomplete distance matrix (no Hamiltonian cycle possible)")

	// ErrDimensionMismatch indicates an unexpected matrix/DP shape in exact algorithms.
	ErrDimensionMismatch = errors.New("tsp: dimension mismatch")

	// ErrStartOutOfRange indicates Options.StartVertex is outside [0..n-1].
	ErrStartOutOfRange = errors.New("tsp: start vertex out of range")

	// ErrMatchingNotImplemented is returned by BlossomMatch when a true minimum-weight
	// perfect matching is not available (fallbacks may be applied by the caller).
	ErrMatchingNotImplemented = errors.New("tsp: blossom matching not implemented")

	// Deprecated: ErrBadInput is kept for legacy callers; do not use in new code.
	ErrBadInput = errors.New("tsp: invalid input")
)

// Planner/engine governance sentinels.
var (
	// ErrUnsupportedAlgorithm is returned when Options.Algo selects an unavailable strategy.
	ErrUnsupportedAlgorithm = errors.New("tsp: unsupported algorithm")

	// ErrTimeLimit indicates a user-specified time budget was exhausted.
	ErrTimeLimit = errors.New("tsp: time limit exceeded")

	// ErrNodeLimit indicates a search-node budget (e.g., for Branch&Bound) was exhausted.
	ErrNodeLimit = errors.New("tsp: node limit exceeded")

	// ErrATSPNotSupportedByAlgo signals that the chosen algorithm handles only symmetric TSP.
	ErrATSPNotSupportedByAlgo = errors.New("tsp: algorithm does not support ATSP")
)

//–––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––
// Matching & bounding enums used by Christofides/BB
//–––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––

// MatchingAlgo selects the perfect matching strategy on odd-degree vertices in Christofides.
type MatchingAlgo int

const (
	// GreedyMatch pairs odd-degree vertices by nearest neighbor (fast; weaker bound).
	GreedyMatch MatchingAlgo = iota

	// BlossomMatch uses Edmonds’ blossom algorithm for true minimum-weight matching
	// (restores the 1.5× guarantee on metric TSP when implemented).
	BlossomMatch
)

// BoundAlgo selects bounding strategy in Branch & Bound solvers.
type BoundAlgo int

const (
	// NoBound disables lower bounds (intended for testing/benchmarking only).
	NoBound BoundAlgo = iota

	// SimpleBound uses the degree-1 relaxation (fast, admissible for TSP/ATSP).
	SimpleBound

	// OneTreeBound enables the Held–Karp 1-tree lower bound (symmetric only).
	// Current integration is root-only (pre-DFS) for a safe, deterministic boost.
	OneTreeBound
)

//–––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––
// High-level algorithm selector
//–––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––

// Algorithm enumerates top-level TSP strategies supported by the dispatcher.
type Algorithm int

const (
	// Christofides: 1.5-approx for metric symmetric TSP (MST + perfect matching + Euler + shortcut).
	Christofides Algorithm = iota

	// ExactHeldKarp: Held–Karp DP, O(n²·2ⁿ) time, O(n·2ⁿ) memory.
	ExactHeldKarp

	// TwoOptOnly: local improvement on a seed tour (internal seed tour generator will be used).
	TwoOptOnly

	// ThreeOptOnly: stronger local improvement (reserved; disabled by default).
	ThreeOptOnly

	// BranchAndBound: exact search with lower/upper bounds (reserved in first iteration).
	BranchAndBound
)

//–––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––
// Results
//–––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––

// TSResult encapsulates the output of a TSP solver.
type TSResult struct {
	// Tour is an ordered sequence of vertex indices representing the Hamiltonian cycle.
	// Invariants:
	//   len(Tour) == n + 1
	//   Tour[0] == Tour[n] == StartVertex
	//   each vertex in [0..n-1] appears exactly once in Tour[0:n]
	Tour []int

	// Cost is the total distance along the cycle, computed from the provided distance matrix.
	Cost float64
}

//–––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––
// Options & defaults
//–––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––

// Default knobs
const (
	// DefaultEps is the minimal strictly-better improvement for local search steps.
	DefaultEps = 1e-12

	// DefaultTwoOptMaxIters caps the number of 2-opt swap attempts across all iterations.
	DefaultTwoOptMaxIters = 10_000
)

// Options defines configurable parameters for TSP solvers.
// Zero value is not meaningful; use DefaultOptions() and override fields as needed.
type Options struct {
	// StartVertex selects the start/end vertex index [0..n-1]. Default: 0.
	StartVertex int

	// Algo selects the top-level algorithm (dispatcher). Default: Christofides.
	Algo Algorithm

	// Symmetric controls matrix validation:
	//   true  → require dist[i][j] == dist[j][i] (TSP),
	//   false → allow asymmetry (ATSP) for algorithms that support it.
	// Default: true.
	Symmetric bool

	// MatchingAlgo chooses between GreedyMatch or BlossomMatch in Christofides.
	MatchingAlgo MatchingAlgo

	// BoundAlgo controls lower-bound strategy in Branch & Bound (reserved).
	BoundAlgo BoundAlgo

	// RunMetricClosure, if true, runs Floyd–Warshall to replace +Inf with shortest paths
	// before solving, enabling partially connected graphs to become metric-closed.
	RunMetricClosure bool

	// EnableLocalSearch applies a post-pass 2-opt (and later 3-opt) when supported.
	// Default: true (for Christofides and seed tours).
	EnableLocalSearch bool

	// TwoOptMaxIters bounds the total number of accepted moves in local search
	// (applies to both 2-opt and 3-opt). Zero ⇒ unlimited. Default: 10_000.
	TwoOptMaxIters int

	// BestImprovement, if true: use best-improvement policy (3-opt/2-opt); else first-improvement
	BestImprovement bool

	// ShuffleNeighborhood, if true: randomize candidate order using Seed; if false: canonical order
	ShuffleNeighborhood bool

	// Eps is the minimal improvement considered significant in local search comparisons.
	// Default: 1e-12.
	Eps float64

	// TimeLimit optionally bounds wall-clock time for long-running heuristics/search.
	// Zero means “no limit”.
	TimeLimit time.Duration

	// Seed controls deterministic behavior of randomized components (seeded RNG).
	// Default: 0 (fixed seed → deterministic).
	Seed int64
}

// DefaultOptions returns a fully populated Options struct with safe, production-ready defaults:
//   - Start at vertex 0
//   - Christofides (metric symmetric), Blossom matching (fallback allowed), no B&B
//   - No metric closure by default
//   - Local search enabled (2-opt) with conservative iteration cap
//   - Symmetric matrix required
//   - Deterministic RNG (Seed=0), no time limit
func DefaultOptions() Options {
	return Options{
		StartVertex:       0,
		Algo:              Christofides,
		Symmetric:         true,
		MatchingAlgo:      BlossomMatch,
		BoundAlgo:         NoBound,
		RunMetricClosure:  false,
		EnableLocalSearch: true,
		TwoOptMaxIters:    DefaultTwoOptMaxIters,
		Eps:               DefaultEps,
		TimeLimit:         0,
		Seed:              0,
	}
}
