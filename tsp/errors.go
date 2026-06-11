// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package tsp defines sentinel errors for the Traveling Salesman Problem solvers.
// The file is the single extension point for P0 safety/correctness error classes.
package tsp

import "errors"

var (
	// ErrNilDistanceMatrix reports a nil or typed-nil distance matrix input.
	//
	// AI-Hints:
	//   - Use this sentinel for nil matrix inputs, not ErrDimensionMismatch.
	//   - Preserve matrix-level nil sentinels with errors.Join where possible.
	ErrNilDistanceMatrix = errors.New("tsp: distance matrix is nil")

	// ErrNilGraph reports a nil graph input for graph-to-matrix adapters.
	//
	// AI-Hints:
	//   - Use this sentinel at the tsp adapter layer.
	//   - Preserve matrix/core adapter errors with errors.Join where applicable.
	ErrNilGraph = errors.New("tsp: graph is nil")

	// ErrNilTour reports a nil tour slice.
	//
	// AI-Hints:
	//   - Nil and structurally invalid tours are different classes.
	//   - Use ErrInvalidTour for malformed non-nil tours.
	ErrNilTour = errors.New("tsp: tour is nil")

	// ErrNilResult reports method calls on a nil *TSPResult receiver.
	//
	// AI-Hints:
	//   - Result helper methods must classify nil access with this sentinel.
	//   - Do not panic on nil result receivers.
	ErrNilResult = errors.New("tsp: result is nil")

	// ErrInvalidOptions reports structurally invalid solver options.
	//
	// AI-Hints:
	//   - Negative durations, NaN/Inf epsilon, invalid iteration caps,
	//     and unknown enum values belong here.
	ErrInvalidOptions = errors.New("tsp: invalid options")

	// ErrInvalidIDs reports empty or duplicated vertex IDs.
	//
	// AI-Hints:
	//   - Length mismatch remains ErrDimensionMismatch.
	//   - Empty/duplicate IDs are semantic ID violations.
	ErrInvalidIDs = errors.New("tsp: invalid vertex ids")

	// ErrInvalidTour reports a malformed non-nil Hamiltonian cycle.
	//
	// AI-Hints:
	//   - Use for wrong length, missing closure, duplicate vertex, or out-of-range vertex.
	ErrInvalidTour = errors.New("tsp: invalid tour")

	// ErrInvalidMatching reports malformed matching input or infeasible matching state.
	//
	// AI-Hints:
	//   - Use when the odd-degree set cannot be paired safely.
	//   - Do not silently skip unmatched odd vertices in Christofides.
	ErrInvalidMatching = errors.New("tsp: invalid matching input")

	// ErrNaNInf reports forbidden NaN, -Inf, or contract-invalid Inf values.
	//
	// AI-Hints:
	//   - +Inf may be allowed only before metric closure.
	//   - Final solver input must not contain +Inf off the diagonal.
	ErrNaNInf = errors.New("tsp: NaN or Inf encountered")

	// ErrMixedGraphNotRepresentable reports a core.Graph with mixed edge directionality
	// that cannot be losslessly converted through the current plain adjacency adapter.
	//
	// AI-Hints:
	//   - Reject mixed graphs until matrix exposes a mixed-aware orientation adapter.
	//   - Do not flatten mixed graphs into a global directed/undirected matrix silently.
	ErrMixedGraphNotRepresentable = errors.New("tsp: mixed directedness graph cannot be converted to a TSP distance matrix safely")

	// ErrNonEulerian ..
	ErrNonEulerian = errors.New("tsp: graph is not Eulerian")
)
