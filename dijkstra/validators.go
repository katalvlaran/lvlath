// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

package dijkstra

import (
	"fmt"
	"math"

	"github.com/katalvlaran/lvlath/core"
)

// validateInputs validates the canonical kernel entry inputs before any traversal
// state is allocated or any shortest-path work begins.
// The function centralizes the mandatory graph and source checks required by the
// package contract.
//
// Implementation:
//   - Stage 1: Validate that the graph pointer is non-nil.
//   - Stage 2: Validate that the source vertex identifier is non-empty.
//   - Stage 3: Validate that the graph supports weights.
//   - Stage 4: Validate that the source vertex exists in the graph.
//
// Behavior highlights:
//   - The function performs fail-fast contract checks.
//   - No traversal state, heap state, or result maps are allocated here.
//
// Inputs:
//   - g: the graph that will be traversed by the kernel.
//   - sourceID: the source vertex identifier for the shortest-path run.
//
// Returns:
//   - error: nil when all kernel-entry input checks succeed.
//
// Errors:
//   - ErrNilGraph if g is nil.
//   - ErrEmptySourceID if sourceID is empty.
//   - ErrUnweightedGraph if the graph is not weighted.
//   - ErrSourceNotFound if the source vertex is absent from the graph.
//
// Determinism:
//   - Deterministic for the same graph pointer and source identifier.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - This validator intentionally repeats facade-level checks to harden the kernel contract.
//   - The function does not inspect edge weights; numeric validation belongs to validateEdgeWeights.
//
// AI-Hints:
//   - Keep kernel validation explicit even if upper API layers already validate the same fields.
//   - Do not silently auto-create a missing source vertex; shortest-path kernels must not mutate graph topology.
func validateInputs(g *core.Graph, sourceID string) error {
	if g == nil {
		return ErrNilGraph
	}
	if sourceID == "" {
		return ErrEmptySourceID
	}
	if !g.Weighted() {
		return ErrUnweightedGraph
	}
	if !g.HasVertex(sourceID) {
		return ErrSourceNotFound
	}

	return nil
}

// classifyWeight classifies a single raw edge weight under the package numeric policy.
// The classifier is the single source of truth for rejecting mathematically invalid
// edge weights before they can poison heap ordering or path computations.
//
// Implementation:
//   - Stage 1: Reject NaN.
//   - Stage 2: Reject negative infinity.
//   - Stage 3: Reject finite negative values.
//   - Stage 4: Accept all remaining values, including +Inf.
//
// Behavior highlights:
//   - Positive infinity is allowed here because the package treats it as a wall at traversal time.
//   - Finite negative weights and non-finite invalid negatives are classified separately.
//
// Inputs:
//   - weight: the edge weight to classify.
//
// Returns:
//   - error: nil when the weight is valid for the package numeric policy.
//
// Errors:
//   - ErrInvalidWeight for NaN and negative infinity.
//   - ErrNegativeWeight for finite negative weights.
//
// Determinism:
//   - Deterministic for the same numeric input.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - This classifier does not apply InfEdgeThreshold; threshold policy belongs to relaxation logic.
//   - Positive infinity remains valid input data and is interpreted later as an impassable wall.
//
// AI-Hints:
//   - Do not collapse ErrInvalidWeight and ErrNegativeWeight into one generic error.
//   - Do not classify +Inf as invalid; the traversal contract uses it as a wall value.
func classifyWeight(weight float64) error {
	switch {
	case math.IsNaN(weight):
		return ErrInvalidWeight
	case math.IsInf(weight, -1):
		return ErrInvalidWeight
	case weight < 0:
		return ErrNegativeWeight
	default:
		return nil
	}
}

// validateEdgeWeights performs a deterministic pre-scan of the graph edge catalog
// and rejects any edge weight that violates the package numeric policy.
// The scan exists to fail fast before the traversal kernel allocates full runtime
// state or begins heap-driven exploration.
//
// Implementation:
//   - Stage 1: Enumerate all edges through the graph's deterministic edge surface.
//   - Stage 2: Classify each raw edge weight through classifyWeight.
//   - Stage 3: Preserve the sentinel and wrap it with edge context on failure.
//
// Behavior highlights:
//   - The scan is deterministic because core.Edges() is sorted by Edge.ID.
//   - The scan rejects invalid weights before they can affect heap behavior.
//
// Inputs:
//   - g: the weighted graph whose edges will be validated.
//   - opts: the finalized runtime policy for this execution.
//
// Returns:
//   - error: nil when all edges satisfy the package numeric policy.
//
// Errors:
//   - ErrInvalidWeight when an edge contains NaN or negative infinity.
//   - ErrNegativeWeight when an edge contains a finite negative weight.
//   - Wrapped errors preserve the original sentinel and include edge context.
//
// Determinism:
//   - Deterministic for the same graph state because edge enumeration order is stable.
//
// Complexity:
//   - Time O(E log E) in effective graph-surface cost because core.Edges() returns a sorted slice.
//   - Additional scan cost after enumeration is O(E).
//   - Space O(E) in effective graph-surface cost because core.Edges() materializes a slice.
//
// Notes:
//   - opts is accepted as part of the validator surface because runtime policy is part of the kernel contract.
//   - The current numeric pre-scan does not apply MaxDistance or InfEdgeThreshold because they are traversal policies, not edge validity rules.
//
// AI-Hints:
//   - Keep numeric policy centralized here and in classifyWeight; do not let relax drift into a different classification regime.
//   - Preserve sentinels with %w when attaching edge context for diagnostics.
func validateEdgeWeights(g *core.Graph, opts DijkstraOptions) error {
	_ = opts

	edges := g.Edges()
	for _, edge := range edges {
		if err := classifyWeight(edge.Weight); err != nil {
			return fmt.Errorf("%w: edge %s->%s weight=%g", err, edge.From, edge.To, edge.Weight)
		}
	}

	return nil
}
