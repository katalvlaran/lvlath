// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

package flow

import "errors"

var (
	// ErrNilGraph is returned when a public flow entry point receives a nil graph.
	ErrNilGraph = errors.New("flow: nil graph")

	// ErrNilResult is returned by nil-safe MaxFlowResult helper methods.
	ErrNilResult = errors.New("flow: nil result")

	// ErrNoResidual is returned when a result does not contain a residual graph.
	ErrNoResidual = errors.New("flow: residual graph is unavailable")

	// ErrEmptyTerminal is returned when source or sink vertex ID is empty.
	ErrEmptyTerminal = errors.New("flow: source or sink vertex ID is empty")

	// ErrSameTerminal is returned when source and sink refer to the same vertex.
	ErrSameTerminal = errors.New("flow: source and sink must be distinct")

	// ErrSourceNotFound is returned when the source vertex does not exist.
	ErrSourceNotFound = errors.New("flow: source vertex not found")

	// ErrSinkNotFound is returned when the sink vertex does not exist.
	ErrSinkNotFound = errors.New("flow: sink vertex not found")

	// ErrInvalidOptions is returned when an option violates the public contract.
	ErrInvalidOptions = errors.New("flow: invalid options")

	// ErrInvalidEpsilon is returned when epsilon is negative, NaN, or infinite.
	ErrInvalidEpsilon = errors.New("flow: invalid epsilon")

	// ErrAugmentationLimit is returned when an algorithm reaches the configured
	// maximum number of successful augmentations before proving optimality.
	ErrAugmentationLimit = errors.New("flow: augmentation limit exceeded")

	// ErrInvalidCapacity is returned when a capacity cannot participate in max-flow.
	ErrInvalidCapacity = errors.New("flow: invalid capacity")

	// ErrNegativeCapacity is returned when an edge capacity is below -epsilon.
	ErrNegativeCapacity = errors.New("flow: negative capacity")

	// ErrNaNInf is returned when a capacity or numeric policy value is NaN or Inf.
	ErrNaNInf = errors.New("flow: NaN or Inf capacity")

	// ErrUnweightedGraph is returned when the graph cannot carry non-zero capacities.
	ErrUnweightedGraph = errors.New("flow: weighted graph is required for capacities")

	// ErrObserverFailure is returned when an augmentation observer rejects an event.
	ErrObserverFailure = errors.New("flow: observer failure")
)
