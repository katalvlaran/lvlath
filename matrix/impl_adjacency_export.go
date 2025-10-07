// SPDX-License-Identifier: MIT
// Package matrix — adjacency export (matrix → core.Graph).
//
// This file provides a *thin, deterministic* export surface from an already
// built AdjacencyMatrix back to core.Graph, honoring the unified options
// contract:
//
//   - Guard: if the source adjacency was built with MetricClosure=true,
//     ToGraph MUST be unsupported and return ErrMatrixNotImplemented.
//   - Threshold: add edge iff a[i,j] > edgeThreshold (strict).
//   - Weight policy: KeepWeights ⇒ weight = a[i,j]; Binary ⇒ weight = 1.
//   - Undirected export: emit each unordered pair {i,j} ONCE (upper triangle),
//     loops (i==j) are emitted once as well; no mirroring on export.
//   - Directed export: emit every ordered pair (i,j).
//   - Determinism: iterate vertices in the stored stable order (vertexByIndex);
//     nested loops are fixed; no map iteration ordering.
//
// AI-Hints:
//   • Set a low EdgeThreshold (e.g., 0 or 0.5) to export all non-zero edges reliably.
//   • Use Binary weights to get a clean unweighted graph for structural analytics.
//   • KeepWeights only makes sense if the adjacency was built as weighted.
//   • Export direction always mirrors the source adjacency’s Directed policy, ensuring
//     round-trip fidelity. Override via adapters before building if you need to flip.
//
// Complexity:
//   - O(n^2) reads of the matrix + O(n + m) vertex/edge insertions into core.
//   - No hidden allocations beyond necessary slices for vertex IDs.

package matrix

import (
	"fmt"
	"math"

	"github.com/katalvlaran/lvlath/core"
)

// ToGraph reconstructs a core.Graph from this adjacency matrix.
// Export behavior is controlled by runtime options (threshold / weight policy).
// Direction/loops/multi policy is inherited from the source adjacency.
// Errors: ErrNilMatrix, ErrDimensionMismatch, ErrMatrixNotImplemented, plus bubbled core errors.
func (am *AdjacencyMatrix) ToGraph(optFns ...Option) (*core.Graph, error) {
	// Validate receiver: both the wrapper and the underlying matrix must be non-nil.
	if am == nil || am.Mat == nil {
		return nil, fmt.Errorf("ToGraph: %w", ErrNilMatrix) // unified sentinel for nil receiver
	}

	// Validate shape consistency: square matrix and index table aligned.
	n := am.Mat.Rows()                                    // number of rows
	if n != am.Mat.Cols() || n != len(am.vertexByIndex) { // square + index length
		return nil, fmt.Errorf("ToGraph: rows=%d cols=%d idx=%d: %w",
			am.Mat.Rows(), am.Mat.Cols(), len(am.vertexByIndex), ErrDimensionMismatch)
	}

	// Guard Metric-Closure: distance matrices are not exportable as simple edges.
	// NOTE: opts field is part of AdjacencyMatrix; metricClose is set by builders.
	if am.opts.metricClose { // single, explicit flag — no reflective tricks
		return nil, fmt.Errorf("ToGraph: metric-closure adjacency cannot be converted: %w", ErrMatrixNotImplemented)
	}

	// Gather export options (threshold/weights). Direction comes from source options.
	exp := gatherOptions(optFns...)  // apply user overrides on documented defaults
	thr := exp.edgeThreshold         // a[i,j] must be strictly greater to emit an edge
	keepWeights := exp.keepWeights   // true ⇒ weight=a[i,j]; false ⇒ weight=1
	directed := am.opts.directed     // inherit orientation of the built adjacency
	allowLoops := am.opts.allowLoops // snapshot loop policy for core construction
	allowMulti := am.opts.allowMulti // snapshot multi-edge policy for core construction
	weightedSrc := am.opts.weighted  // whether adjacency originally preserved weights

	// Prepare the target graph with deterministic, policy-accurate flags.
	gOpts := make([]core.GraphOption, 0, 4) // preallocate small, fixed set
	// Direction: undirected export ⇒ core.WithDirected(false); else true.
	gOpts = append(gOpts, core.WithDirected(directed)) // pass through directedness as is
	// Weights: keepWeights ⇒ weighted graph; binary export ⇒ unweighted.
	if keepWeights && weightedSrc { // only mark weighted if it matters
		gOpts = append(gOpts, core.WithWeighted())
	}
	// Loops / multi-edges: preserve build-time policy snapshot where sensible.
	// (While export won’t generate duplicates itself, we keep flags for fidelity.)
	if allowLoops {
		gOpts = append(gOpts, core.WithLoops())
	}
	if allowMulti {
		gOpts = append(gOpts, core.WithMultiEdges())
	}
	g := core.NewGraph(gOpts...) // construction is O(1); core owns its internals

	// Vertex IDs are already in deterministic order within am.vertexByIndex.
	var err error
	for _, vid := range am.vertexByIndex {
		// AddVertex is idempotent in core (by contract); ignore returned id if any.
		if err = g.AddVertex(vid); err != nil {
			// Surface core error verbatim; callers will handle via errors.Is for core sentinels.
			return nil, fmt.Errorf("ToGraph: AddVertex %q: %w", vid, err)
		}
	}

	// Deterministic nested loops over matrix entries with a single write site.
	// Directed: all ordered pairs (i,j). Undirected: upper triangle i..n-1 (incl. diag).
	var i, j int
	var fromID, toID string
	var val float64
	if directed {
		for i = 0; i < n; i++ { // iterate rows
			fromID = am.vertexByIndex[i] // resolve source id once per row
			for j = 0; j < n; j++ {      // iterate columns
				toID = am.vertexByIndex[j] // resolve target id
				val, err = am.Mat.At(i, j) // O(1) bounds-checked read
				if err != nil {
					return nil, fmt.Errorf("ToGraph: At(%d,%d): %w", i, j, err) // surface matrix read error
				}
				if err = returnEdge(g, fromID, toID, val, thr, keepWeights); err != nil {
					return nil, err // already wrapped with context
				}
			}
		}
	} else {
		for i = 0; i < n; i++ { // upper triangle only to avoid duplicates
			fromID = am.vertexByIndex[i] // source id for this row
			for j = i; j < n; j++ {      // j starts at i ⇒ (i,i) loop once, (i,j) once
				toID = am.vertexByIndex[j] // target id
				val, err = am.Mat.At(i, j) // read once; no mirror read
				if err != nil {
					return nil, fmt.Errorf("ToGraph: At(%d,%d): %w", i, j, err)
				}
				if err = returnEdge(g, fromID, toID, val, thr, keepWeights); err != nil {
					return nil, err
				}
			}
		}
	}

	// Successful, deterministic export complete.
	return g, nil
}

// returnEdge is egde emission helper (non-anonymous, no captures) to keep loop body minimal.
// Applies threshold/weight policy and inserts a single edge when eligible.
// Returns a wrapped error with context or nil.
func returnEdge(g *core.Graph, fromID, toID string, aij float64, threshold float64, keep bool) error {
	// Skip distances/+Inf (metric-closure never reaches here) and sub-threshold values.
	if math.IsInf(aij, +1) || !(aij > threshold) {
		return nil // not an edge per strict policy
	}
	// Derive integer weight for core: keep ⇒ cast a[i,j]; binary ⇒ 1.
	var w int64
	if keep {
		w = int64(aij) // adjacency values originate from int64 edge weights
	} else {
		w = 1
	}
	// Insert the edge; core enforces multi/loop constraints and ordering.
	if _, err := g.AddEdge(fromID, toID, w); err != nil {
		return fmt.Errorf("ToGraph: AddEdge %q->%q: %w", fromID, toID, err)
	}

	return nil
}
