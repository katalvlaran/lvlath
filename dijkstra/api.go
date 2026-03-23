// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

package dijkstra

import "github.com/katalvlaran/lvlath/core"

// Dijkstra runs deterministic single-source shortest paths over a weighted graph
// and returns a detached result object that exposes distances and optional path
// reconstruction helpers.
//
// Implementation:
//   - Stage 1: Validate facade-level inputs.
//   - Stage 2: Assemble the finalized runtime policy through applyOptions.
//   - Stage 3: Delegate the actual shortest-path computation to the canonical kernel.
//
// Behavior highlights:
//   - sourceID is an explicit required input and is never taken from options.
//   - The returned result is detached from the graph and safe for post-run queries.
//   - Path reconstruction remains disabled unless explicitly requested by options or wrappers.
//
// Inputs:
//   - g: the weighted graph to traverse.
//   - sourceID: the source vertex identifier for the traversal.
//   - opts: zero or more functional runtime options.
//
// Returns:
//   - *DijkstraResult: the detached shortest-path result for the requested source.
//
// Errors:
//   - ErrNilGraph if g is nil.
//   - ErrEmptySourceID if sourceID is empty.
//   - ErrNilOption if a nil option is supplied.
//   - Any option-validation error returned by applyOptions.
//   - Any validation or runtime error returned by the canonical kernel.
//
// Determinism:
//   - The facade preserves the deterministic behavior of the canonical kernel and adds no hidden ordering.
//
// Complexity:
//   - Time O(n) for option assembly plus the delegated kernel cost.
//   - Space O(1) in the facade itself, excluding the delegated kernel result.
//
// Notes:
//   - The facade owns orchestration only and must not embed alternative shortest-path logic.
//
// AI-Hints:
//   - Do not move sourceID back into options.
//   - Do not place graph validation, heap logic, or relaxation logic into this facade.
func Dijkstra(g *core.Graph, sourceID string, opts ...Option) (*DijkstraResult, error) {
	if g == nil {
		return nil, ErrNilGraph
	}
	if sourceID == "" {
		return nil, ErrEmptySourceID
	}

	config, err := applyOptions(opts...)
	if err != nil {
		return nil, err
	}

	return runDijkstra(g, sourceID, config)
}

// Distances runs Dijkstra and returns a deep copy of the finalized distance map.
// This wrapper is intended for callers that need distances only and do not want
// to interact with the full result object.
//
// Implementation:
//   - Stage 1: Delegate to Dijkstra.
//   - Stage 2: Deep-copy the returned distance map.
//   - Stage 3: Return the isolated copy to the caller.
//
// Behavior highlights:
//   - The wrapper isolates ownership of the returned map from the underlying result.
//   - The wrapper does not require path tracking.
//
// Inputs:
//   - g: the weighted graph to traverse.
//   - sourceID: the source vertex identifier for the traversal.
//   - opts: zero or more functional runtime options.
//
// Returns:
//   - map[string]float64: a detached copy of the finalized distance map.
//
// Errors:
//   - Any error returned by Dijkstra.
//
// Determinism:
//   - The wrapper preserves the deterministic result produced by Dijkstra.
//
// Complexity:
//   - Time O(V) and Space O(V) for copying the distance map, in addition to the delegated kernel cost.
//
// Notes:
//   - Returning the internal result map directly is forbidden because it weakens ownership isolation.
//
// AI-Hints:
//   - Keep the map copy explicit; do not leak shared mutable ownership through convenience wrappers.
//   - Do not call Clone here just to obtain distances; that would also copy Prev unnecessarily.
func Distances(g *core.Graph, sourceID string, opts ...Option) (map[string]float64, error) {
	result, err := Dijkstra(g, sourceID, opts...)
	if err != nil {
		return nil, err
	}

	clonedDistances := make(map[string]float64, len(result.Distances))
	for vertexID, distance := range result.Distances {
		clonedDistances[vertexID] = distance
	}

	return clonedDistances, nil
}

// DistanceTo runs Dijkstra and returns the finalized shortest-path distance
// to a single requested target vertex.
//
// Implementation:
//   - Stage 1: Validate the target identifier.
//   - Stage 2: Delegate to Dijkstra.
//   - Stage 3: Resolve the target distance through the result surface.
//
// Behavior highlights:
//   - The wrapper does not require path tracking.
//   - Unknown targets remain explicit errors.
//   - Reachable and unreachable known targets are distinguished by the returned distance.
//
// Inputs:
//   - g: the weighted graph to traverse.
//   - sourceID: the source vertex identifier for the traversal.
//   - targetID: the target vertex identifier to query.
//   - opts: zero or more functional runtime options.
//
// Returns:
//   - float64: the finalized shortest-path distance to the requested target.
//
// Errors:
//   - ErrEmptyTargetID if targetID is empty.
//   - Any error returned by Dijkstra.
//   - Any result-surface error returned by (*DijkstraResult).DistanceTo.
//
// Determinism:
//   - The wrapper preserves the deterministic result produced by Dijkstra.
//
// Complexity:
//   - Time O(1) after the delegated kernel cost.
//   - Space O(1) in the wrapper itself.
//
// Notes:
//   - A returned +Inf value is a valid canonical outcome for a known but unreachable target.
//
// AI-Hints:
//   - Do not bypass the result method and read the distance map directly in the wrapper.
//   - Keep target validation explicit instead of silently treating an empty ID as missing.
func DistanceTo(g *core.Graph, sourceID, targetID string, opts ...Option) (float64, error) {
	if targetID == "" {
		return 0, ErrEmptyTargetID
	}

	result, err := Dijkstra(g, sourceID, opts...)
	if err != nil {
		return 0, err
	}

	return result.DistanceTo(targetID)
}

// ShortestPathTo runs Dijkstra with path tracking enabled and returns one
// deterministic shortest-path witness together with its finalized distance.
//
// Implementation:
//   - Stage 1: Validate the target identifier.
//   - Stage 2: Build an isolated option slice and append WithPathTracking.
//   - Stage 3: Delegate to Dijkstra.
//   - Stage 4: Resolve the path witness and distance through the result surface.
//
// Behavior highlights:
//   - The wrapper enforces path tracking explicitly.
//   - Caller-owned option slices are not mutated.
//   - The wrapper returns one deterministic shortest-path witness, not all shortest paths.
//
// Inputs:
//   - g: the weighted graph to traverse.
//   - sourceID: the source vertex identifier for the traversal.
//   - targetID: the target vertex identifier to reconstruct.
//   - opts: zero or more functional runtime options.
//
// Returns:
//   - []string: one deterministic shortest-path witness from source to target.
//   - float64: the finalized shortest-path distance to the requested target.
//
// Errors:
//   - ErrEmptyTargetID if targetID is empty.
//   - Any error returned by Dijkstra.
//   - Any result-surface error returned by PathTo or DistanceTo.
//
// Determinism:
//   - The wrapper preserves the deterministic predecessor and path reconstruction laws of the kernel and result surface.
//
// Complexity:
//   - Time O(m + k) after the delegated kernel cost, where m is the number of options copied and k is the path length.
//   - Space O(m + k) in the wrapper and returned path, excluding the delegated kernel result.
//
// Notes:
//   - Wrapper policy wins by explicitly appending WithPathTracking to the local option slice copy.
//
// AI-Hints:
//   - Copy the option slice before appending wrapper-enforced options to avoid aliasing caller-owned storage.
//   - Do not reconstruct paths manually in this wrapper; the result surface is the canonical path API.
func ShortestPathTo(g *core.Graph, sourceID, targetID string, opts ...Option) ([]string, float64, error) {
	if targetID == "" {
		return nil, 0, ErrEmptyTargetID
	}

	wrapperOptions := make([]Option, 0, len(opts)+1)
	wrapperOptions = append(wrapperOptions, opts...)
	wrapperOptions = append(wrapperOptions, WithPathTracking())

	result, err := Dijkstra(g, sourceID, wrapperOptions...)
	if err != nil {
		return nil, 0, err
	}

	path, err := result.PathTo(targetID)
	if err != nil {
		return nil, 0, err
	}

	distance, err := result.DistanceTo(targetID)
	if err != nil {
		return nil, 0, err
	}

	return path, distance, nil
}
