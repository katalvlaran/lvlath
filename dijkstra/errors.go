// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

package dijkstra

import "errors"

var (
	// ErrNilGraph reports that the caller passed a nil graph pointer.
	// This error originates during the earliest input-validation stage,
	// before any option application, allocation, or graph inspection occurs.
	//
	// AI-Hints:
	//   - This is an input-contract failure, not a runtime traversal failure.
	//   - Classify it with errors.Is instead of matching error text.
	ErrNilGraph = errors.New("dijkstra: graph is nil")

	// ErrEmptySourceID reports that the caller passed an empty source vertex ID.
	// This error originates during the input-validation stage before the algorithm
	// attempts to inspect graph topology or initialize traversal state.
	//
	// AI-Hints:
	//   - The canonical API requires sourceID as an explicit argument.
	//   - Do not reintroduce Source(...) as a replacement for this contract.
	ErrEmptySourceID = errors.New("dijkstra: source vertex id is empty")

	// ErrSourceNotFound reports that the requested source vertex does not exist
	// in the provided graph. This error originates after basic input validation
	// and before the shortest-path kernel allocates or starts traversal.
	//
	// AI-Hints:
	//   - This error classifies a missing source vertex only.
	//   - Missing target vertices must use ErrTargetNotFound instead.
	ErrSourceNotFound = errors.New("dijkstra: source vertex not found")

	// ErrTargetNotFound reports that a result-level query referenced a target
	// vertex that does not exist in the computed result domain. This error is
	// intended for result helpers such as DistanceTo, HasPathTo, and PathTo.
	//
	// AI-Hints:
	//   - Keep source validation and target lookup classification separate.
	//   - Do not collapse ErrSourceNotFound and ErrTargetNotFound into one sentinel.
	ErrTargetNotFound = errors.New("dijkstra: target vertex not found")

	// ErrUnweightedGraph reports that the graph does not support weights.
	// Dijkstra requires a weighted graph contract even when all runtime weights
	// are non-negative, because the algorithm consumes edge weights directly.
	//
	// AI-Hints:
	//   - This is a graph-policy violation detected before traversal begins.
	//   - Do not silently coerce unweighted graphs into an implicit unit-weight mode.
	ErrUnweightedGraph = errors.New("dijkstra: graph must be weighted")

	// ErrNilOption reports that a nil functional option was supplied to the API.
	// This error originates during option assembly, before the algorithm allocates
	// traversal state or inspects the full graph.
	//
	// AI-Hints:
	//   - Functional options are part of the public contract and must fail explicitly.
	//   - Never replace this with panic-based configuration handling.
	ErrNilOption = errors.New("dijkstra: nil option")

	// ErrBadMaxDistance reports that MaxDistance violates the public option contract.
	// Valid values are finite non-negative numbers or positive infinity.
	// This error originates during option application.
	//
	// AI-Hints:
	//   - NaN and negative infinity are invalid here.
	//   - Positive infinity is allowed and means "no distance cutoff".
	ErrBadMaxDistance = errors.New("dijkstra: max distance must be >= 0 and not NaN")

	// ErrBadInfEdgeThreshold reports that InfEdgeThreshold violates the public option
	// contract. Valid values are strictly positive numbers or positive infinity.
	// This error originates during option application.
	//
	// AI-Hints:
	//   - A zero threshold would incorrectly classify zero-weight edges as walls.
	//   - Positive infinity is allowed and preserves the default "no wall threshold" mode.
	ErrBadInfEdgeThreshold = errors.New("dijkstra: inf edge threshold must be > 0 and not NaN")

	// ErrNegativeWeight reports that a finite negative edge weight was encountered.
	// This error originates during numeric validation because Dijkstra's algorithm
	// is defined only for graphs with non-negative edge weights.
	//
	// AI-Hints:
	//   - Preserve this sentinel when wrapping with edge context.
	//   - Do not merge negative finite weights with NaN or -Inf into one vague error class.
	ErrNegativeWeight = errors.New("dijkstra: negative edge weight encountered")

	// ErrInvalidWeight reports that an edge weight is numerically invalid for this
	// package, such as NaN or negative infinity. This error originates during
	// numeric validation before the invalid value can poison heap ordering or distances.
	//
	// AI-Hints:
	//   - Preserve this sentinel when wrapping with edge context.
	//   - Positive infinity is not classified here when the package treats it as a wall.
	ErrInvalidWeight = errors.New("dijkstra: invalid non-finite edge weight")

	// ErrPathTrackingDisabled reports that the caller requested path reconstruction
	// from a result that was produced without predecessor tracking enabled.
	// This error originates from result-surface methods rather than from the kernel.
	//
	// AI-Hints:
	//   - Prev == nil means tracking was disabled, not that the graph has no paths.
	//   - Keep this distinct from ErrNoPath.
	ErrPathTrackingDisabled = errors.New("dijkstra: path tracking disabled")

	// ErrNoPath reports that the requested target vertex is known to the result,
	// but no path exists from the source under the effective traversal policy.
	// This error originates from result-surface methods such as PathTo.
	//
	// AI-Hints:
	//   - Unreachable is a normal shortest-path outcome, not a validation failure.
	//   - Keep this distinct from ErrTargetNotFound and ErrPathTrackingDisabled.
	ErrNoPath = errors.New("dijkstra: no path")

	// ErrEmptyTargetID reports that the caller passed an empty target vertex ID.
	// This error originates during result-surface queries and API wrappers that
	// require a concrete target vertex identifier.
	//
	// AI-Hints:
	//   - Keep this distinct from ErrTargetNotFound.
	//   - An empty identifier is an input-contract failure, not a missing-vertex lookup result.
	ErrEmptyTargetID = errors.New("dijkstra: target vertex id is empty")

	// ErrNilResult reports that a result-surface method was called on a nil
	// *DijkstraResult receiver. This error originates from result helper methods
	// and prevents nil-pointer panics from leaking into the public API.
	//
	// AI-Hints:
	//   - DijkstraResult implements core.Nilable, but methods must still remain safe on nil receivers.
	//   - Do not replace this with a panic or with ErrTargetNotFound.
	ErrNilResult = errors.New("dijkstra: result is nil")
)
