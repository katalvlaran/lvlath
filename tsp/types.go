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

const (
	// Auto selects an explicit size-aware dispatcher policy.
	//
	// AI-Hints:
	//   - Auto is opt-in. Do not silently change DefaultOptions to Auto in a patch release.
	//   - Existing Algorithm iota values must remain stable for compatibility.
	Auto Algorithm = -1
)

//–––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––
// Results
//–––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––

// TSPResult is the canonical result artifact for lvlath/tsp solvers.
// It owns the Hamiltonian cycle snapshot, cost, optional vertex IDs, solver
// status, approximation metadata, timeout state, and non-fatal warnings.
//
// TSPResult is the only result type used by canonical public facades. Legacy
// TSResult values are projection-only compatibility artifacts and intentionally
// discard metadata that is required to interpret exactness, partial results, and
// approximation guarantees.
//
// Implementation:
//   - Stage 1: Canonical facades prepare matrix/graph input and finalize solver policy.
//   - Stage 2: The dispatcher selects exactly one solver path and publishes TSPResult.
//   - Stage 3: Helper methods expose detached projections without recomputing tours.
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
//   - VertexTour maps Tour indices to IDs without sorting or route recomputation.
//
// Errors:
//   - Helper methods classify nil receivers with ErrNilResult.
//   - Helper methods classify nil receivers with ErrNilResult when they return error.
//   - VertexTour returns ErrInvalidIDs when no ID mapping is available.
//   - VertexTour returns ErrInvalidTour when Tour contains an out-of-range index.
//
// Determinism:
//   - Tour order follows the solver’s canonicalization law.
//   - IDs preserve matrix row order or core.Graph adapter order.
//
// Complexity:
//   - Clone and Minimal are O(len(Tour)+len(IDs)+len(Warnings)).
//   - VertexTour is O(len(Tour)).
//   - HasWarning is O(len(Warnings)).
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
//   - Do not expose IDs by map iteration; IDs are matrix-index ordered.
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

// TSResult encapsulates the legacy minimal output of a TSP solver.
//
// Deprecated: use TSPResult through SolveMatrix, SolveGraph, or canonical
// solver entrypoints. TSResult is a projection-only compatibility shape; it
// intentionally discards exactness, timeout, approximation, matching fallback,
// warning, and adapter metadata.
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

// VertexTour projects r.Tour from matrix row/column indices to caller-visible vertex IDs.
// It is a read-only convenience helper for consumers that supplied IDs to SolveMatrix or
// used SolveGraph with graph vertex IDs recovered through the matrix adapter.
//
// Implementation:
//   - Stage 1: Validate the receiver and ensure an ID mapping is present.
//   - Stage 2: Allocate one output string slice with len(Tour).
//   - Stage 3: Scan Tour in order and map each vertex index through IDs.
//
// Behavior highlights:
//   - Does not mutate the result.
//   - Does not sort IDs and does not canonicalize Tour.
//   - Preserves the exact route order already published by the solver.
//
// Inputs:
//   - r: canonical TSP result; r.IDs must map matrix indices to stable IDs.
//
// Returns:
//   - []string: detached vertex-ID tour in the same order as r.Tour.
//   - error: nil on success or a sentinel-classified failure.
//
// Errors:
//   - ErrNilResult when called on a nil receiver.
//   - ErrInvalidIDs when no ID mapping is attached to the result.
//   - ErrInvalidTour when a tour index is outside [0..len(IDs)-1].
//
// Determinism:
//   - Fixed left-to-right scan over r.Tour.
//   - Output order is the solver-published tour order.
//
// Complexity:
//   - Time O(len(Tour)), Space O(len(Tour)).
//
// Notes:
//   - Empty IDs are not revalidated here; SolveMatrix validates provided IDs before solving.
//   - Results built by direct matrix calls with ids==nil cannot be projected to vertex IDs.
//
// AI-Hints:
//   - Do not recover IDs by map iteration inside this method.
//   - Do not recompute or re-canonicalize the tour here; this is a pure projection helper.
func (r *TSPResult) VertexTour() ([]string, error) {
	if r == nil {
		return nil, ErrNilResult
	}
	if len(r.IDs) == 0 {
		return nil, ErrInvalidIDs
	}

	vertexTour := make([]string, len(r.Tour))

	var (
		position int
		vertex   int
	)
	for position, vertex = range r.Tour {
		if vertex < 0 || vertex >= len(r.IDs) {
			return nil, ErrInvalidTour
		}

		vertexTour[position] = r.IDs[vertex]
	}

	return vertexTour, nil
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

// HasWarning reports whether r.Warnings contains target under errors.Is classification.
// It provides a compact, sentinel-safe query for non-fatal degradations such as matching
// fallback warnings without requiring callers to inspect error strings.
//
// Implementation:
//   - Stage 1: Treat nil receiver or nil target as a false query.
//   - Stage 2: Scan warnings in stored order.
//   - Stage 3: Return true on the first errors.Is match.
//
// Behavior highlights:
//   - Does not allocate.
//   - Does not join warnings and does not mutate the result.
//   - Uses errors.Is, so wrapped and joined sentinels remain discoverable.
//
// Inputs:
//   - target: sentinel error to classify in r.Warnings.
//
// Returns:
//   - bool: true when any stored warning matches target.
//
// Errors:
//   - None. Invalid query shapes return false.
//
// Determinism:
//   - Fixed left-to-right scan over Warnings.
//
// Complexity:
//   - Time O(len(Warnings)), Space O(1).
//
// Notes:
//   - Use WarningsError when the caller needs an errors.Is-compatible aggregate.
//   - Use HasWarning for simple branch decisions in application code and tests.
//
// AI-Hints:
//   - Do not compare warning strings; warning classification must use errors.Is.
//   - Do not treat nil receiver as a panic condition in result helpers.
func (r *TSPResult) HasWarning(target error) bool {
	if r == nil || target == nil {
		return false
	}

	var warning error
	for _, warning = range r.Warnings {
		if errors.Is(warning, target) {
			return true
		}
	}

	return false
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
//   - Use errors.Is(result.WarningsError(), target) instead of comparing warning strings.
//   - Prefer HasWarning for single-sentinel queries that do not need an aggregate error.
func (r *TSPResult) WarningsError() error {
	if r == nil {
		return ErrNilResult
	}
	if len(r.Warnings) == 0 {
		return nil
	}

	return errors.Join(r.Warnings...)
}
