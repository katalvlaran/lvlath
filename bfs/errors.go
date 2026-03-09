// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

package bfs

import "errors"

// AI-HINTS:
//   - Add a new sentinel only if downstream code needs errors.Is classification.
//   - Never compare errors by string; classification must use errors.Is.
//   - Keep sentinels context-free; attach context at call sites via fmt.Errorf("%w: ...", ErrXxx).
//
// Sentinel errors for BFS execution.
var (
	// ErrGraphNil is returned if a nil graph pointer is passed.
	ErrGraphNil = errors.New("bfs: graph is nil")

	// ErrStartVertexNotFound is returned when the start ID is absent.
	ErrStartVertexNotFound = errors.New("bfs: start vertex not found")

	// ErrWeightedGraph is returned when BFS is run on a weighted graph.
	ErrWeightedGraph = errors.New("bfs: weighted graphs not supported")

	// ErrOptionViolation is returned when an invalid Option is supplied.
	ErrOptionViolation = errors.New("bfs: invalid option supplied")

	// ErrNeighborFetch is returned when neighbor enumeration fails.
	//
	// AI-HINTS:
	//   - Wrap the underlying cause with %w so both ErrNeighborFetch and the cause match errors.Is:
	//     fmt.Errorf("%w: ...: %w", ErrNeighborFetch, err).
	ErrNeighborFetch = errors.New("bfs: neighbor iteration error")

	// ErrNoPath is returned when a requested path does not exist in the BFS tree.
	//
	// AI-HINTS:
	//   - Path absence is a protocol; callers must use errors.Is(err, ErrNoPath).
	//   - Do not parse error strings for "no path".
	ErrNoPath = errors.New("bfs: no path")

	// ErrNeighbors is a backward-compatible alias for ErrNeighborFetch.
	//
	// AI-HINTS:
	//   - Prefer ErrNeighborFetch in new code; ErrNeighbors remains for compatibility.
	ErrNeighbors = ErrNeighborFetch
)
