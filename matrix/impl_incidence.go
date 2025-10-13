// SPDX-License-Identifier: MIT
// Package matrix — incidence builders (dense) with strict invariants.
//
// Deliverables (per TA-MATRIX):
//   1) Nil-guards for "light" getters (panic on nil receiver/Mat with fixed message).
//   2) Clarified signs: directed uses −1 at source and +1 at target; undirected uses +1/+1;
//      self-loop in directed sums (−1 + +1) in the *same row* ⇒ 0 column; in undirected a loop
//      contributes +2 in the single incident row (both half-edges touch the same vertex).
//   3) AllowMulti=false ⇒ first-edge-wins policy (directed: ordered (u,v); undirected: unordered {min,max}).
//   4) Deterministic order: vertices follow provided order; edge columns follow stable core edge order.
//   5) Sentinel errors unified (ErrGraphNil, ErrUnknownVertex, ErrDimensionMismatch, ErrNilMatrix).
//
// AI-Hints:
//   - Use AllowMulti=false when you need a canonical incidence (no duplicate columns).
//   - Incidence ignores numeric weights by design; it captures topology only (sign/endpoint).
//   - For undirected graphs, a self-loop appears as +2 in the single row — this is conventional in
//     incidence algebra; downstream tools that expect strictly {−1,0,+1} should normalize if needed.
//   - Determinism is guaranteed if you pass a deterministic vertex order and core returns edges by ID.
//
// Complexity:
//   - BuildDenseIncidence: O(|V| + |E|) time, O(|V| + |E|) space (index map + column list + dense matrix).
//   - Accessors: O(1) except VertexIncidence (O(#edges) to copy the row).

package matrix

import (
	"fmt" // build contextual error messages with sentinel wrapping
	"sort"

	"github.com/katalvlaran/lvlath/core"
)

// --- Incidence marks (no magic numbers) -------------------------------------------------------------

// srcMark is placed at the source vertex row in a directed incidence column (outgoing end).
const srcMark = -1.0 // −1 at "from" for directed graphs

// dstMark is placed at the target vertex row in a directed incidence column (incoming end).
const dstMark = +1.0 // +1 at "to" for directed graphs

// undirectedMark is placed at each incident vertex row for undirected non-loop edges.
const undirectedMark = +1.0 // +1 / +1 for undirected (two distinct endpoints)

// loopUndirectedMark is placed at the incident vertex row for undirected self-loops.
// Rationale: both half-edges touch the same vertex ⇒ +1 + +1 = +2 in that row.
const loopUndirectedMark = 2.0

// --- Public wrapper type ---------------------------------------------------------------------------

// IncidenceMatrix wraps a Matrix as a graph incidence representation.
// VertexIndex maps VertexID → row index in Mat.
// Edges holds the ordered list of *core.Edge corresponding to columns.
// Mat holds −1/0/+1 (and +2 for undirected loops) entries indicating incidence.
// opts preserves original construction options for round-trip fidelity.
type IncidenceMatrix struct {
	Mat         Matrix         // underlying incidence matrix (square? no: rows=|V|, cols=|E_eff|)
	VertexIndex map[string]int // mapping of VertexID to row index
	Edges       []*core.Edge   // ordered edges aligned to columns [0..cols)
	opts        Options        // original build options snapshot (public, per your current API)
}

// --- Constructor (public) --------------------------------------------------------------------------

// NewIncidenceMatrix constructs an IncidenceMatrix from g using MatrixOptions.
// Stage 1 (Validate): ensure g is non-nil.
// Stage 2 (Prepare): extract stable vertex and edge lists.
// Stage 3 (Execute): BuildDenseIncidence (deterministic, policy-aware).
// Stage 4 (Finalize): wrap and return.
// Errors: ErrGraphNil, plus any BuildDenseIncidence sentinel.
func NewIncidenceMatrix(g *core.Graph, opts Options) (*IncidenceMatrix, error) {
	// Validate input graph (public sentinel for nil graph).
	if g == nil {
		return nil, fmt.Errorf("NewIncidenceMatrix: %w", ErrGraphNil) // guard early
	}

	// Pull vertices in the order defined by core; callers may already sort lexicographically.
	vertices := g.Vertices() // O(|V|); assumed stable per core contract

	// Pull edges in stable order (by Edge.ID asc per core); determinism depends on this.
	edges := g.Edges() // O(|E|)

	// Delegate to deterministic dense builder (validates inputs and options).
	idx, cols, mat, err := BuildDenseIncidence(vertices, edges, opts)
	if err != nil {
		return nil, fmt.Errorf("NewIncidenceMatrix: %w", err) // bubble with context
	}

	// Wrap high-level struct and return (Mat is already dense and bounds-checked).
	return &IncidenceMatrix{
		Mat:         mat,  // Matrix implementation (dense) returned by builder
		VertexIndex: idx,  // stable vertex→row mapping
		Edges:       cols, // column-aligned edges (post de-duplication if any)
		opts:        opts, // snapshot options for export fidelity
	}, nil
}

// --- Lightweight accessors with nil-guards ----------------------------------------------------------

// VertexCount returns the number of vertices (rows) in the incidence matrix.
// Panics on nil receiver/Mat with a fixed message (developer error, not user error).
// Complexity: O(1).
func (im *IncidenceMatrix) VertexCount() int {
	// Guard nil receiver and underlying Mat — consistent panic message (golden expectation).
	if im == nil || im.Mat == nil {
		panic("IncidenceMatrix: nil receiver or Mat") // intentional panic for light getters
	}
	// Return number of rows (|V|).
	return im.Mat.Rows()
}

// EdgeCount returns the number of edges (columns) in the incidence matrix.
// Panics on nil receiver/Mat with a fixed message.
// Complexity: O(1).
func (im *IncidenceMatrix) EdgeCount() int {
	// Same nil-guard semantics as VertexCount for consistency.
	if im == nil || im.Mat == nil {
		panic("IncidenceMatrix: nil receiver or Mat")
	}
	// Return number of columns (|E_eff|).
	return im.Mat.Cols()
}

// VertexIncidence returns the incidence row for vertexID as a newly allocated slice.
// Errors: ErrNilMatrix (when im or Mat is nil), ErrUnknownVertex (lookup fails), and
// matrix index errors from Mat.At are wrapped with context.
// Complexity: O(#edges).
func (im *IncidenceMatrix) VertexIncidence(vertexID string) ([]float64, error) {
	// Soft-fail (error) on nil receiver for external-facing method (prefer error over panic).
	if im == nil || im.Mat == nil {
		return nil, fmt.Errorf("VertexIncidence: %w", ErrNilMatrix)
	}
	// Lookup row index for the given vertex ID.
	row, ok := im.VertexIndex[vertexID]
	if !ok {
		return nil, fmt.Errorf("VertexIncidence: unknown vertex %q: %w", vertexID, ErrUnknownVertex)
	}
	// Allocate output slice sized to number of columns (|E_eff|).
	cols := im.Mat.Cols()        // total columns
	out := make([]float64, cols) // one entry per column

	// Copy row entries via safe At; bubble any index error.
	var val float64
	var err error
	for j := 0; j < cols; j++ {
		val, err = im.Mat.At(row, j)
		if err != nil {
			return nil, fmt.Errorf("VertexIncidence: At(%d,%d): %w", row, j, err)
		}
		out[j] = val // assign to output
	}

	// Return the copied row.
	return out, nil
}

// EdgeEndpoints returns (fromID, toID) of the edge corresponding to column j.
// For undirected graphs, the (fromID,toID) ordering matches core.Edge endpoints.
// Errors: ErrNilMatrix, ErrDimensionMismatch on out-of-range j.
// Complexity: O(1).
func (im *IncidenceMatrix) EdgeEndpoints(j int) (fromID, toID string, err error) {
	// Soft-fail on nil receiver.
	if im == nil || im.Mat == nil {
		return "", "", fmt.Errorf("EdgeEndpoints: %w", ErrNilMatrix)
	}
	// Bounds check on column index; use public sentinel for dimension issues.
	if j < 0 || j >= im.Mat.Cols() {
		return "", "", fmt.Errorf("EdgeEndpoints: column %d out of range [0,%d): %w",
			j, im.Mat.Cols(), ErrDimensionMismatch)
	}
	// Fetch the aligned *core.Edge and return its endpoints (IDs).
	e := im.Edges[j]

	return e.From, e.To, nil
}

// --- Dense incidence builder -----------------------------------------------------------------------

// buildDenseIncidenceFromGraph is a convenience wrapper for tests/internal callers that
// possess only *core.Graph*. It extracts vertex IDs and edges deterministically and
// delegates to BuildDenseIncidence.
//
// NOTE: We defensively ensure lexicographic vertex order here to facilitate golden tests,
//
//	while the main constructor trusts the caller’s order.
func buildDenseIncidenceFromGraph(g *core.Graph, opts Options) (map[string]int, []*core.Edge, *Dense, error) {
	// Validate graph argument.
	if g == nil {
		return nil, nil, nil, fmt.Errorf("buildDenseIncidenceFromGraph: %w", ErrGraphNil)
	}

	// Pull vertex IDs and enforce lexicographic order for canonical layouts in tests.
	ids := g.Vertices()
	if !isLexSorted(ids) {
		cp := make([]string, len(ids))
		copy(cp, ids)
		sort.Strings(cp)
		ids = cp
	}

	// Pull edges in stable order and delegate to the main builder.
	return BuildDenseIncidence(ids, g.Edges(), opts)
}
