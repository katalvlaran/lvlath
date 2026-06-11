// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package tsp exposes canonical public facades for Traveling Salesman Problem solvers.
// The canonical surface returns TSPResult; legacy wrappers project it into TSResult.
package tsp

import (
	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/matrix"
)

// SolveMatrix solves a TSP/ATSP instance represented by a distance matrix.
// It is the canonical matrix facade and publishes the full TSPResult contract.
//
// Implementation:
//   - Stage 1: Prepare direct matrix input, including optional metric closure.
//   - Stage 2: Validate optional IDs against final matrix order.
//   - Stage 3: Delegate to the existing solver dispatcher without alternative mathematics.
//   - Stage 4: Attach metadata and detached result slices.
//   - Stage 5: Publish TSPResult.
//
// Behavior highlights:
//   - Does not mutate caller-owned direct matrices.
//   - Uses matrix.APSPInPlace for metric closure.
//   - Preserves SolveWithMatrix compatibility through Minimal projection.
//   - Does not silently change algorithms or fallback beyond the existing explicit options.
//
// Inputs:
//   - dist: square distance matrix; +Inf may appear only before metric closure.
//   - ids: optional row/column labels; when non-nil, len(ids)==n and all IDs are non-empty/unique.
//   - opts: explicit solver policy; use DefaultOptions and override fields.
//
// Returns:
//   - *TSPResult: detached canonical result on success.
//   - error: nil on success or sentinel-classified failure.
//
// Errors:
//   - ErrInvalidOptions, ErrUnsupportedAlgorithm, ErrATSPNotSupportedByAlgo.
//   - ErrNilDistanceMatrix, ErrNonSquare, ErrDimensionMismatch.
//   - ErrNaNInf, ErrNonZeroDiagonal, ErrNegativeWeight, ErrIncompleteGraph, ErrAsymmetry.
//   - ErrInvalidIDs.
//   - Algorithm-specific sentinels such as ErrSizeTooLarge or ErrTimeLimit.
//
// Determinism:
//   - Matrix row/column order is the vertex order.
//   - Kernel tie-breaks and canonicalization define Tour order.
//   - IDs are copied without sorting.
//
// Complexity:
//   - Validation O(n^2).
//   - Metric closure O(n^3) when enabled.
//   - Solver complexity depends on opts.Algo.
//
// Notes:
//   - TSResult wrappers intentionally discard metadata.
//   - This facade does not yet publish partial B&B timeout incumbents; that belongs to the next P1 substage.
//
// AI-Hints:
//   - Prefer SolveMatrix over SolveWithMatrix in new code.
//   - Do not bypass this facade in examples unless testing a specific kernel.
func SolveMatrix(dist matrix.Matrix, ids []string, opts Options) (*TSPResult, error) {
	prepared, metricClosureApplied, n, err := prepareSolverDistanceMatrix(dist, opts)
	if err != nil {
		return nil, err
	}

	finalOptions := opts
	finalOptions.RunMetricClosure = false

	finalOptions, err = finalizeSolverOptions(n, finalOptions)
	if err != nil {
		return nil, err
	}

	if ids != nil {
		if err = validateIDs(ids, n); err != nil {
			return nil, err
		}
	}

	minimal, meta, err := solvePreparedMatrix(prepared, finalOptions, n)
	if err != nil && len(minimal.Tour) == 0 {
		return nil, err
	}

	result := publishTSPResult(minimal, ids, finalOptions, meta, metricClosureApplied)

	return result, err
}

// SolveGraph solves a TSP instance by adapting a core.Graph into a matrix distance model.
// It is the canonical graph facade and publishes the full TSPResult contract.
//
// Implementation:
//   - Stage 1: Reject nil and currently unsupported mixed-direction graphs.
//   - Stage 2: Build matrix adapter options from core.Graph flags.
//   - Stage 3: Build adjacency/distance matrix through matrix.NewAdjacencyMatrix.
//   - Stage 4: Recover stable row-index IDs from matrix.VertexIndex.
//   - Stage 5: Delegate to SolveMatrix without re-running graph metric closure.
//
// Behavior highlights:
//   - Adapter policy is explicit and deterministic.
//   - Mixed directedness is rejected until matrix exposes a lossless mixed-edge adapter.
//   - Vertex IDs are recovered by index, not map iteration order.
//
// Inputs:
//   - g: source graph.
//   - opts: solver policy.
//
// Returns:
//   - *TSPResult: detached canonical result.
//   - error: sentinel-classified failure.
//
// Errors:
//   - ErrNilGraph for nil graph.
//   - ErrMixedGraphNotRepresentable for unsupported mixed directedness.
//   - matrix adapter errors from matrix.NewMatrixOptions / matrix.NewAdjacencyMatrix.
//   - SolveMatrix errors after adaptation.
//
// Determinism:
//   - Core graph order is inherited through matrix adapter vertex indices.
//   - ID recovery writes ids[index]=id, so map iteration does not affect output.
//
// Complexity:
//   - Adapter build O(V^2+E).
//   - Optional metric closure O(V^3).
//   - Solver complexity depends on opts.Algo.
//
// AI-Hints:
//   - Do not flatten mixed directed/undirected graphs into a global directed matrix.
//   - Do not recover IDs by appending over map iteration.
func SolveGraph(g *core.Graph, opts Options) (*TSPResult, error) {
	// Nil graph => invalid shape for building matrices.
	if g == nil {
		return nil, ErrNilGraph
	}

	graphStats := g.Stats()
	if graphStats.MixedMode || (graphStats.DirectedEdgeCount > 0 && graphStats.UndirectedEdgeCount > 0) {
		return nil, ErrMixedGraphNotRepresentable
	}

	matrixOptions := make([]matrix.Option, 0, 6)

	// Map core → matrix options (explicit, no booleans into WithX)
	// Orientation: mixed OR directed-default ⇒ directed matrix (can encode both types of edges).
	if graphStats.DirectedDefault || graphStats.DirectedEdgeCount > 0 {
		matrixOptions = append(matrixOptions, matrix.WithDirected())
	} else {
		matrixOptions = append(matrixOptions, matrix.WithUndirected())
	}

	// Multi-edges: matrix default = allowMulti=true; respect core policy explicitly.
	if graphStats.AllowsMulti {
		matrixOptions = append(matrixOptions, matrix.WithAllowMulti())
	} else {
		matrixOptions = append(matrixOptions, matrix.WithDisallowMulti())
	}

	// Loops: matrix default = allowLoops=false.
	if graphStats.AllowsLoops {
		matrixOptions = append(matrixOptions, matrix.WithAllowLoops())
	} else {
		matrixOptions = append(matrixOptions, matrix.WithDisallowLoops())
	}

	// Weights: matrix default = unweighted(false).
	if graphStats.Weighted {
		matrixOptions = append(matrixOptions, matrix.WithWeighted())
	} else {
		matrixOptions = append(matrixOptions, matrix.WithUnweighted())
	}

	// Metric closure (APSP).
	if opts.RunMetricClosure {
		matrixOptions = append(matrixOptions, matrix.WithMetricClosure())
	}

	// Build matrix options from graph flags + dispatcher policy; «single source of truth».
	frozenMatrixOptions, err := matrix.NewMatrixOptions(matrixOptions...)
	if err != nil {
		return nil, err
	}

	adjacency, err := matrix.NewAdjacencyMatrix(g, frozenMatrixOptions)
	if err != nil {
		// NewAdjacencyMatrix returns matrix-level errors; forward them as-is.
		// Upstream validateAll will surface tsp sentinels when we dispatch via SolveWithMatrix.
		return nil, err
	}

	// Recover stable vertex ordering ids[idx] = id.
	// Map iteration order is irrelevant: we write by canonical index -> stable array.
	vertexCount := adjacency.Mat.Rows()
	ids := make([]string, vertexCount)

	// VertexIndex is id -> index, so invert it.
	var (
		id    string
		index int
	)
	for id, index = range adjacency.VertexIndex {
		ids[index] = id
	}

	delegateOptions := opts
	if opts.RunMetricClosure {
		delegateOptions.RunMetricClosure = false
	}

	// Delegate to matrix dispatcher (unified validation is done there).
	result, err := SolveMatrix(adjacency.Mat, ids, delegateOptions)
	if err != nil {
		return nil, err
	}

	if opts.RunMetricClosure {
		result.MetricClosureApplied = true
	}

	return result, nil
}

// SolveWithMatrix is the compatibility wrapper for the legacy TSResult surface.
// New code should prefer SolveMatrix because it preserves metadata and result semantics.
//
// Implementation:
//   - Stage 1: Call SolveMatrix.
//   - Stage 2: Project TSPResult into TSResult through Minimal.
//
// Behavior highlights:
//   - Does not contain alternative solver logic.
//   - Discards metadata intentionally.
//
// Inputs:
//   - dist: square distance matrix.
//   - ids: optional row/column labels.
//   - opts: solver policy.
//
// Returns:
//   - TSResult: legacy tour/cost projection.
//   - error: same error as SolveMatrix.
//
// Determinism:
//   - Same as SolveMatrix.
//
// Complexity:
//   - Same as SolveMatrix plus O(len(Tour)) projection.
//
// AI-Hints:
//   - Do not add algorithm logic here; wrappers must remain honest projections.
func SolveWithMatrix(dist matrix.Matrix, ids []string, opts Options) (TSResult, error) {
	result, err := SolveMatrix(dist, ids, opts)
	if result == nil {
		return TSResult{}, err
	}

	return result.Minimal(), err
}

// SolveWithGraph is the compatibility wrapper for the legacy graph TSResult surface.
// New code should prefer SolveGraph because it preserves metadata and result semantics.
func SolveWithGraph(g *core.Graph, opts Options) (TSResult, error) {
	result, err := SolveGraph(g, opts)
	if result == nil {
		return TSResult{}, err
	}

	return result.Minimal(), err
}

// publishTSPResult builds the canonical detached result artifact.
// Implementation:
//   - Stage 1: Copy minimal tour/cost.
//   - Stage 2: Copy optional IDs.
//   - Stage 3: Attach final policy and kernel-origin metadata.
//   - Stage 4: Copy warning sentinels into caller-owned storage.
//
// Behavior highlights:
//   - No live references to matrices, graphs, or mutable solver state.
//   - Metadata comes from solveMeta, not facade inference.
//
// Inputs:
//   - minimal: successful or partial tour/cost payload.
//   - ids: optional stable row/column IDs.
//   - opts: finalized options after Auto selection.
//   - meta: kernel-origin metadata.
//   - metricClosureApplied: adapter/facade policy fact.
//
// Returns:
//   - *TSPResult: detached canonical result.
//
// Errors:
//   - None.
//
// Determinism:
//   - Tour, IDs, and warnings preserve source order exactly.
//
// Complexity:
//   - Time O(len(Tour)+len(IDs)+len(Warnings)).
//   - Space O(len(Tour)+len(IDs)+len(Warnings)).
//
// AI-Hints:
//   - Do not recompute costs or routes here; this is a publishing stage only.
func publishTSPResult(
	minimal TSResult,
	ids []string,
	opts Options,
	meta solveMeta,
	metricClosureApplied bool,
) *TSPResult {
	result := &TSPResult{
		Tour:                 append([]int(nil), minimal.Tour...),
		Cost:                 minimal.Cost,
		Algorithm:            opts.Algo,
		Exact:                meta.exact,
		Optimal:              meta.optimal,
		TimedOut:             meta.timedOut,
		MetricClosureApplied: metricClosureApplied,
		Symmetric:            opts.Symmetric,
		ApproximationRatio:   meta.approximationRatio,
		MatchingFallback:     meta.matchingFallback,
		Iterations:           meta.iterations,
		NodesExpanded:        meta.nodesExpanded,
	}

	if ids != nil {
		result.IDs = append([]string(nil), ids...)
	}
	if meta.warnings != nil {
		result.Warnings = append([]error(nil), meta.warnings...)
	}

	return result
}
