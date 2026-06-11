// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

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

	// ExactHeldKarp: Held–Karp DP, O(n²*2ⁿ) time, O(n*2ⁿ) memory.
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

// TSPResult is the canonical result artifact for lvlath/tsp solvers.
// It preserves the minimal tour/cost output while making algorithm status,
// ownership, approximation, timeout, and adapter metadata explicit.
//
// Implementation:
//   - Stage 1: Solvers compute a TSResult through the existing kernels.
//   - Stage 2: The canonical facade wraps the minimal result into TSPResult.
//   - Stage 3: Metadata fields record the selected policy and observable execution status.
//
// Behavior highlights:
//   - Tour and IDs are detached caller-owned slices.
//   - Warnings are sentinel-preserving errors; classify them with errors.Is.
//   - Optimal is true only for exact algorithms that completed successfully.
//   - ApproximationRatio is meaningful only when the selected algorithm proves a bound.
//   - MatchingFallback records BlossomMatch -> GreedyMatch degradation.
//
// Inputs:
//   - Produced by SolveMatrix or SolveGraph.
//
// Returns:
//   - TSPResult values are immutable-by-convention snapshots.
//   - Clone returns a deep copy of all exposed slices.
//
// Errors:
//   - Helper methods classify nil receivers with ErrNilResult.
//
// Determinism:
//   - Tour order follows the solver’s canonicalization law.
//   - IDs preserve matrix row order or core.Graph adapter order.
//
// Complexity:
//   - Clone and Minimal are O(len(Tour)+len(IDs)+len(Warnings)).
//   - IsNil is O(1).
//
// Notes:
//   - TSResult remains the compatibility/minimal projection.
//   - TSPResult is the primary contract surface for new code.
//
// AI-Hints:
//   - Do not add live matrix or graph references to this result.
//   - Do not interpret Cost as optimal unless Optimal is true.
//   - Do not claim a Christofides bound when MatchingFallback is true.
type TSPResult struct {
	// Tour is the closed Hamiltonian cycle in matrix row/column indices.
	// The slice is detached and caller-owned.
	Tour []int

	// Cost is the total cost of Tour under the final solver distance matrix.
	Cost float64

	// IDs maps matrix indices to optional caller-visible vertex IDs.
	// When present, len(IDs) equals the matrix order.
	IDs []string

	// Algorithm is the top-level solver selected by Options.Algo.
	Algorithm Algorithm

	// Optimal is true only when an exact solver completed without timeout.
	Optimal bool

	// Exact is true for exact solvers such as Held-Karp and Branch-and-Bound.
	Exact bool

	// TimedOut reports that a solver stopped due to Options.TimeLimit.
	TimedOut bool

	// MetricClosureApplied reports whether direct matrix or graph input was metric-closed.
	MetricClosureApplied bool

	// Symmetric records the final symmetry policy used for validation and algorithm dispatch.
	Symmetric bool

	// ApproximationRatio records a proven approximation factor when available.
	// A zero value means no formal ratio is claimed.
	ApproximationRatio float64

	// MatchingFallback reports that BlossomMatch was requested but GreedyMatch was used.
	MatchingFallback bool

	// Iterations records local-search or iterative-bound work when the kernel publishes it.
	Iterations int

	// NodesExpanded records branch-and-bound search nodes when available.
	NodesExpanded int

	// Warnings stores non-fatal sentinel-preserving degradations.
	Warnings []error
}

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

// IsNil reports whether r is nil.
// Implementation:
//   - Stage 1: Compare receiver to nil.
//
// Behavior highlights:
//   - Safe on nil receivers.
//   - Does not inspect result fields.
//
// Inputs:
//   - r: optional *TSResult receiver.
//
// Returns:
//   - bool: true when r==nil.
//
// Errors:
//   - None.
//
// Determinism:
//   - Deterministic O(1).
//
// Complexity:
//   - Time O(1), Space O(1).
//
// AI-Hints:
//   - Use this in tests and guards instead of reflect-based nil checks.
func (r *TSResult) IsNil() bool {
	return r == nil
}

// IsNil reports whether r is nil.
// Implementation:
//   - Stage 1: Compare receiver to nil.
//
// Behavior highlights:
//   - Safe on nil receivers.
//   - Does not inspect result fields.
//
// Inputs:
//   - r: optional *TSPResult receiver.
//
// Returns:
//   - bool: true when r==nil.
//
// Errors:
//   - None.
//
// Determinism:
//   - Deterministic O(1).
//
// Complexity:
//   - Time O(1), Space O(1).
//
// AI-Hints:
//   - Use this in tests and guards instead of reflect-based nil checks.
func (r *TSPResult) IsNil() bool {
	return r == nil
}

// Clone returns a detached copy of r.
// Implementation:
//   - Stage 1: Return nil for nil receiver.
//   - Stage 2: Shallow-copy scalar fields.
//   - Stage 3: Deep-copy Tour, IDs, and Warnings slices.
//
// Behavior highlights:
//   - Caller owns all slices in the returned result.
//   - Warning error values are copied as interface values, not deep-cloned.
//
// Inputs:
//   - r: source result.
//
// Returns:
//   - *TSPResult: detached result copy or nil.
//
// Errors:
//   - None.
//
// Determinism:
//   - Preserves slice order exactly.
//
// Complexity:
//   - Time O(len(Tour)+len(IDs)+len(Warnings)).
//   - Space O(len(Tour)+len(IDs)+len(Warnings)).
//
// Notes:
//   - Clone does not validate the result; it preserves the snapshot as-is.
//
// AI-Hints:
//   - Do not return aliases for Tour or IDs; external mutation would corrupt result ownership.
func (r *TSPResult) Clone() *TSPResult {
	if r == nil {
		return nil
	}

	out := *r

	if r.Tour != nil {
		out.Tour = append([]int(nil), r.Tour...)
	}
	if r.IDs != nil {
		out.IDs = append([]string(nil), r.IDs...)
	}
	if r.Warnings != nil {
		out.Warnings = append([]error(nil), r.Warnings...)
	}

	return &out
}

// Minimal projects r into the legacy TSResult shape.
// Implementation:
//   - Stage 1: Return zero TSResult for nil receiver.
//   - Stage 2: Copy Tour into a caller-owned slice.
//   - Stage 3: Preserve Cost exactly.
//
// Behavior highlights:
//   - Compatibility wrapper for old SolveWithMatrix/SolveWithGraph users.
//   - Discards metadata intentionally.
//
// Inputs:
//   - r: canonical result.
//
// Returns:
//   - TSResult: minimal tour/cost projection.
//
// Errors:
//   - None.
//
// Determinism:
//   - Preserves Tour order exactly.
//
// Complexity:
//   - Time O(len(Tour)), Space O(len(Tour)).
//
// Notes:
//   - Use SolveMatrix/SolveGraph in new code when metadata matters.
//
// AI-Hints:
//   - Do not make wrappers recompute the tour; wrappers must project the canonical result only.
func (r *TSPResult) Minimal() TSResult {
	if r == nil {
		return TSResult{}
	}

	return TSResult{
		Tour: append([]int(nil), r.Tour...),
		Cost: r.Cost,
	}
}

// WarningsError joins non-fatal warnings into one errors.Is-compatible value.
// Implementation:
//   - Stage 1: Return ErrNilResult for nil receiver.
//   - Stage 2: Return nil when no warnings exist.
//   - Stage 3: Return errors.Join over the warning list.
//
// Behavior highlights:
//   - Preserves sentinel classification for each warning.
//   - Does not mutate the warning slice.
//
// Inputs:
//   - r: canonical result.
//
// Returns:
//   - error: nil, ErrNilResult, or joined warning error.
//
// Errors:
//   - ErrNilResult for nil receiver.
//
// Determinism:
//   - Warnings are joined in stored slice order.
//
// Complexity:
//   - Time O(len(Warnings)), Space O(len(Warnings)) inside errors.Join.
//
// AI-Hints:
//   - Use errors.Is(result.WarningsError(), ErrMatchingNotImplemented) for fallback checks.
func (r *TSPResult) WarningsError() error {
	if r == nil {
		return ErrNilResult
	}
	if len(r.Warnings) == 0 {
		return nil
	}

	return errors.Join(r.Warnings...)
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
