// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package tsp defines public types for Traveling Salesman Problem solvers.
// The package exposes one canonical result artifact, TSPResult, so callers can
// interpret exactness, optimality, timeout state, approximation metadata, and
// route ownership without relying on reduced compatibility projections.
package tsp

// MatchingAlgo selects the perfect matching strategy on odd-degree vertices in
// the Christofides pipeline.
type MatchingAlgo int

const (
	// GreedyMatch pairs odd-degree vertices deterministically by local cheapest
	// available partner. It is a valid heuristic mode but does not prove the
	// Christofides 1.5 approximation ratio.
	GreedyMatch MatchingAlgo = iota

	// BlossomMatch selects the exact minimum-weight perfect matching engine.
	// It must either append an exact minimum-weight perfect matching or return a
	// sentinel-classified mathematical/structural error without changing algorithms.
	BlossomMatch
)

// BoundAlgo selects the lower-bound strategy used by Branch-and-Bound.
type BoundAlgo int

const (
	// NoBound disables Branch-and-Bound lower bounds and is intended for tests
	// and controlled benchmarks.
	NoBound BoundAlgo = iota

	// SimpleBound uses a deterministic degree relaxation bound.
	SimpleBound

	// OneTreeBound enables the symmetric 1-tree lower-bound policy.
	OneTreeBound
)

// Algorithm enumerates top-level TSP solver strategies supported by the dispatcher.
type Algorithm int

// Auto selects a deterministic size-aware dispatcher policy.
//
// AI-Hints:
//   - Auto is opt-in. Do not silently make it the default in a patch release.
//   - Existing Algorithm iota values must remain stable for compatibility.
const Auto Algorithm = -1

const (
	// Christofides runs the symmetric metric-TSP approximation pipeline.
	Christofides Algorithm = iota

	// ExactHeldKarp runs the Held-Karp dynamic program.
	ExactHeldKarp

	// TwoOptOnly runs deterministic 2-opt local search from a canonical seed tour.
	TwoOptOnly

	// ThreeOptOnly runs deterministic 3-opt local search from a canonical seed tour.
	ThreeOptOnly

	// BranchAndBound runs exact deterministic Branch-and-Bound search.
	BranchAndBound
)

// TSPResult is the only public result artifact published by package tsp.
//
// Implementation:
//   - Stage 1: Solver facades validate matrix/graph input and finalize explicit options.
//   - Stage 2: Exactly one selected kernel computes or improves a Hamiltonian cycle.
//   - Stage 3: The facade publishes a detached result snapshot with solver metadata.
//
// Behavior highlights:
//   - Tour is a closed Hamiltonian cycle in matrix row/column indices.
//   - IDs is an optional index-to-vertex-ID mapping detached from caller input.
//   - Exact and Optimal are separate because exact searches may return timeout incumbents.
//   - ApproximationRatio==0 means no formal approximation ratio is claimed.
//
// Inputs:
//   - Produced by SolveMatrix, SolveGraph, or direct result-native solver wrappers.
//
// Returns:
//   - TSPResult values are immutable-by-convention snapshots.
//   - Clone returns detached slices owned by the caller.
//   - VertexTour maps Tour indices through IDs without sorting or route recomputation.
//
// Errors:
//   - Helper methods return ErrNilResult for nil receiver access.
//   - VertexTour returns ErrMissingVertexIDs when no ID mapping is attached.
//   - VertexTour returns ErrInvalidTour when Tour contains an invalid matrix index.
//
// Determinism:
//   - Tour order follows the selected solver canonicalization law.
//   - IDs preserve matrix row order or graph-adapter row order.
//   - Clone and VertexTour preserve input slice order exactly.
//
// Complexity:
//   - Clone: Time O(len(Tour)+len(IDs)), Space O(len(Tour)+len(IDs)).
//   - VertexTour: Time O(len(Tour)), Space O(len(Tour)).
//   - IsNil: Time O(1), Space O(1).
//
// Notes:
//   - The package intentionally does not expose a metadata-dropping minimal result type.
//   - Cost must not be interpreted as globally optimal unless Optimal is true.
//
// AI-Hints:
//   - Do not reintroduce a reduced public result projection.
//   - Do not add live matrix, graph, or mutable solver-engine references to this result.
//   - Do not classify timed-out Branch-and-Bound incumbents as Optimal.
type TSPResult struct {
	// Tour is the closed Hamiltonian cycle in matrix row/column indices.
	Tour []int

	// Cost is the total route cost under the final solver distance matrix.
	Cost float64

	// IDs maps matrix indices to optional caller-visible vertex IDs.
	IDs []string

	// Algorithm is the concrete top-level solver used to produce the result.
	Algorithm Algorithm

	// Exact reports whether the producing algorithm is an exact solver.
	Exact bool

	// Optimal reports whether the result is proven globally optimal.
	Optimal bool

	// TimedOut reports whether a time budget stopped the producing solver.
	TimedOut bool

	// MetricClosureApplied reports whether APSP metric closure was applied before solving.
	MetricClosureApplied bool

	// Symmetric records the final symmetry policy used by validation and dispatch.
	Symmetric bool

	// ApproximationRatio records a proven approximation factor when available.
	ApproximationRatio float64

	// Iterations records accepted local-search moves or iterative solver work.
	Iterations int

	// NodesExpanded records Branch-and-Bound search node expansions.
	NodesExpanded int
}

// IsNil reports whether r is nil.
//
// Implementation:
//   - Stage 1: Compare the receiver against nil.
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
//   - Pure receiver check.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - This helper is intended for tests and defensive user code.
//
// AI-Hints:
//   - Prefer this over reflection-based nil checks in package tests.
func (r *TSPResult) IsNil() bool {
	return r == nil
}

// Clone returns a detached copy of r.
//
// Implementation:
//   - Stage 1: Return nil for a nil receiver.
//   - Stage 2: Shallow-copy scalar metadata.
//   - Stage 3: Deep-copy Tour and IDs.
//
// Behavior highlights:
//   - Caller owns all slices in the returned result.
//   - No solver state, matrix, or graph object is retained.
//
// Inputs:
//   - r: source result snapshot.
//
// Returns:
//   - *TSPResult: detached result copy, or nil when r is nil.
//
// Errors:
//   - None.
//
// Determinism:
//   - Preserves Tour and IDs order exactly.
//
// Complexity:
//   - Time O(len(Tour)+len(IDs)), Space O(len(Tour)+len(IDs)).
//
// Notes:
//   - Clone preserves the snapshot as-is; it does not revalidate Tour or Cost.
//
// AI-Hints:
//   - Do not return aliases for Tour or IDs; caller mutation must not corrupt the source.
func (r *TSPResult) Clone() *TSPResult {
	if r == nil {
		return nil
	}

	clone := *r
	clone.Tour = append([]int(nil), r.Tour...)
	clone.IDs = append([]string(nil), r.IDs...)

	return &clone
}

// VertexTour projects r.Tour from matrix indices to caller-visible vertex IDs.
//
// Implementation:
//   - Stage 1: Validate the receiver and ensure an ID mapping is present.
//   - Stage 2: Allocate one output slice with len(Tour).
//   - Stage 3: Scan Tour left-to-right and map each matrix index through IDs.
//
// Behavior highlights:
//   - Does not mutate the result.
//   - Does not sort IDs.
//   - Does not canonicalize, reverse, or recompute the route.
//
// Inputs:
//   - r: canonical TSP result with IDs attached.
//
// Returns:
//   - []string: detached vertex-ID tour in the same order as r.Tour.
//   - error: nil on success or sentinel-classified failure.
//
// Errors:
//   - ErrNilResult when called on a nil receiver.
//   - ErrMissingVertexIDs when no ID mapping is attached.
//   - ErrInvalidTour when a tour index is outside [0, len(IDs)-1].
//
// Determinism:
//   - Fixed left-to-right scan over r.Tour.
//   - Output order is exactly solver-published Tour order.
//
// Complexity:
//   - Time O(len(Tour)), Space O(len(Tour)).
//
// Notes:
//   - Empty string IDs are validated by SolveMatrix before solving.
//   - Direct matrix calls without IDs cannot be projected to vertex labels.
//
// AI-Hints:
//   - Do not recover IDs by map iteration inside this method.
//   - Do not recompute costs or route orientation here.
func (r *TSPResult) VertexTour() ([]string, error) {
	if r == nil {
		return nil, ErrNilResult
	}
	if len(r.IDs) == 0 {
		return nil, ErrMissingVertexIDs
	}
	if len(r.Tour) == 0 {
		return nil, ErrInvalidTour
	}

	vertexTour := make([]string, len(r.Tour))

	for position, vertex := range r.Tour {
		if vertex < 0 || vertex >= len(r.IDs) {
			return nil, ErrInvalidTour
		}
		vertexTour[position] = r.IDs[vertex]
	}

	return vertexTour, nil
}
