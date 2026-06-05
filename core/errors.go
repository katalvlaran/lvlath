// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

package core

import "errors"

var (
	// ErrEmptyVertexID signals that the provided vertex identifier is empty.
	//
	// Contract:
	//   - Any API that accepts a vertex ID MUST reject "" with this sentinel.
	ErrEmptyVertexID = errors.New("core: vertex ID is empty")

	// ErrVertexNotFound indicates that a referenced vertex does not exist.
	//
	// Contract:
	//   - Returned by query/mutation APIs that require a pre-existing vertex.
	ErrVertexNotFound = errors.New("core: vertex not found")

	// ErrEdgeNotFound indicates that a referenced edge (by Edge.ID) was not found.
	//
	// Contract:
	//   - Returned by edge-removal or lookup routines.
	ErrEdgeNotFound = errors.New("core: edge not found")

	// ErrBadWeight reports a non-zero weight on an unweighted graph.
	//
	// Contract:
	//   - On graphs without WithWeighted(), only weight == 0 is allowed.
	ErrBadWeight = errors.New("core: bad weight for unweighted graph")
	ErrNaNInf    = errors.New("core: weight is NaN or Inf")

	// ErrLoopNotAllowed reports a self-loop attempt when loops are disabled.
	//
	// Contract:
	//   - WithLoops() must be set to allow edges (v -> v).
	ErrLoopNotAllowed = errors.New("core: self-loop not allowed")

	// ErrEmptyEdgeID signals that an explicit edge ID was required but empty.
	//
	// Contract:
	//   - AddEdge(..., WithID("")) MUST return ErrEmptyEdgeID.
	//   - SetEdgeID(old,"") MUST return ErrEmptyEdgeID.
	//
	// Notes:
	//   - This sentinel is about edge identifiers (Edge.ID), not vertex IDs.
	ErrEmptyEdgeID = errors.New("core: empty edge ID")

	// ErrEdgeIDConflict signals that an explicit edge ID collides with an existing edge.
	//
	// Contract:
	//   - AddEdge(..., WithID(id)) MUST return ErrEdgeIDConflict if id is already present.
	//   - SetEdgeID(old,id) MUST return ErrEdgeIDConflict if id is already present.
	//
	// Determinism:
	//   - Collision checks are pure map membership checks; no iteration order dependence.
	ErrEdgeIDConflict = errors.New("core: edge ID already exists")

	// ErrMultiEdgeNotAllowed reports a parallel edge attempt when multi-edges are disabled.
	//
	// Contract:
	//   - WithMultiEdges() must be set to allow (u,v) duplication (or directional duplicates).
	ErrMultiEdgeNotAllowed = errors.New("core: multi-edges not allowed")

	// ErrMixedEdgesNotAllowed reports a per-edge directedness override on a non-mixed graph.
	//
	// Contract:
	//   - WithMixedEdges() (or NewMixedGraph) must be set before any WithEdgeDirected(...) override.
	ErrMixedEdgesNotAllowed = errors.New("core: mixed-mode per-edge overrides not allowed")

	// ErrNilGraphOption reports a nil GraphOption passed to NewGraph or NewMixedGraph.
	//
	// Contract:
	//   - NewGraph(nil) MUST return ErrNilGraphOption.
	//   - NewGraph(..., nil, ...) MUST return ErrNilGraphOption.
	//   - NewMixedGraph(..., nil, ...) MUST preserve this sentinel via delegation to NewGraph.
	//
	// Notes:
	//   - Nil constructor options are invalid public inputs, not no-ops and not panics.
	ErrNilGraphOption = errors.New("core: nil graph option")

	// ErrNilEdgeOption reports a nil EdgeOption passed to AddEdge.
	//
	// Contract:
	//   - AddEdge(..., nil) MUST return ErrNilEdgeOption.
	//   - AddEdge(..., opt1, nil, opt2) MUST return ErrNilEdgeOption.
	//
	// Notes:
	//   - Nil per-edge options are rejected during fail-fast validation before any lock
	//     acquisition, vertex auto-creation, or edge publication.
	ErrNilEdgeOption = errors.New("core: nil edge option")

	// ErrInvalidEdgeOption reports that an EdgeOption attempted to mutate
	// topology-owned edge fields after AddEdge had already validated the public input.
	//
	// Contract:
	//   - AddEdge owns Edge.From, Edge.To, and Edge.Weight after initial validation.
	//   - EdgeOption values may set Edge.ID via WithID.
	//   - EdgeOption values may set Edge.Directed only through an explicitly allowed
	//     mixed-mode policy.
	//   - Endpoint or weight mutation is rejected before ID assignment, catalog insertion,
	//     or adjacency mutation.
	//
	// Notes:
	//   - Directedness mutation on a non-mixed graph is classified separately as
	//     ErrMixedEdgesNotAllowed because it is a policy violation, not an endpoint/weight
	//     corruption attempt.
	ErrInvalidEdgeOption = errors.New("core: invalid edge option mutation")

	// ErrNilEdgePredicate reports a nil predicate passed to an edge-filtering API.
	//
	// Contract:
	//   - FilterEdges(nil) MUST return ErrNilEdgePredicate.
	//   - RemoveEdgesWhere(nil) MUST return ErrNilEdgePredicate.
	//   - Nil predicates are invalid public input, not no-ops and not panics.
	ErrNilEdgePredicate = errors.New("core: nil edge predicate")
)
