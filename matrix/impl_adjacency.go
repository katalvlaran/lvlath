// SPDX-License-Identifier: MIT
// Package matrix - adjacency builders (dense) and metric-closure transform.
//
// Deliverables:
//   1) Directed + AllowMulti=false → first-edge-wins (ordered key (u,v)).
//   2) Undirected mirroring without loops (u==v is not mirrored).
//   3) Weighted=true but effectively-unweighted input → degrade to binary (1).
//   4) MetricClosure (Floyd–Warshall): diag=0, unreachable=+Inf (off-diagonal).
//   5) Deterministic iteration & stable vertex/edge order (no map order reliance).
//
// AI-Hints:
//   - For directed graphs, duplicate (u,v) edges are ignored when AllowMulti=false;
//     the *first* occurrence wins. For undirected graphs, the first of unordered
//     pair {min(u), max(v)} wins. This guarantees deterministic results.
//   - When input graph is effectively unweighted (all weights are 0 or graph flags
//     indicate unweighted) and options request Weighted=true, we intentionally build
//     a binary adjacency (1) to avoid an all-zero matrix.
//   - MetricClosure turns adjacency into pairwise shortest-path distances. It is
//     *not* an adjacency anymore, and ToGraph must return ErrMatrixNotImplemented

package matrix

import (
	"fmt"
	"math"
	"sort"

	"github.com/katalvlaran/lvlath/core"
)

// defaultReserve is the initial capacity for neighbor slices
const defaultReserve = 8

// AdjacencyMatrix wraps a Matrix as a graph adjacency representation.
// VertexIndex maps VertexID → row/col in Mat.
// vertexByIndex provides reverse lookup from column index to VertexID.
// Mat holds edge weights (float64), with unreachableWeight for no edge.
// opts preserves original build options for round‐trip fidelity.
type AdjacencyMatrix struct {
	Mat           Matrix         // underlying adjacency matrix
	VertexIndex   map[string]int // mapping of VertexID to index
	vertexByIndex []string       // reverse lookup by index
	opts          Options        // original construction options
}

// NewAdjacencyMatrix constructs an AdjacencyMatrix from g.
// Stage 1 (Validate): ensure g is non‐nil.
// Stage 2 (Prepare): extract vertex list and edge list.
// Stage 3 (Execute): call BuildDenseAdjacency.
// Stage 4 (Finalize): build reverse lookup and return.
// Returns ErrNilGraph or any BuildDenseAdjacency error.
func NewAdjacencyMatrix(g *core.Graph, opts Options) (*AdjacencyMatrix, error) {
	// Validate input graph
	if g == nil {
		return nil, ErrGraphNil
	}

	// Prepare vertex and edge slices
	vertices := g.Vertices() // get ordered vertices
	edges := g.Edges()       // get edges

	// Delegate to low‐level builder
	var (
		idx map[string]int // index map
		mat *Dense         // dense matrix result
		err error          // error placeholder
	)
	idx, mat, err = BuildDenseAdjacency(vertices, edges, opts)
	if err != nil {
		return nil, err
	}

	// Finalize reverse index
	rev := make([]string, len(vertices))
	for id, i := range idx {
		rev[i] = id
	}

	// Wrap and return
	return &AdjacencyMatrix{
		Mat:           mat,
		VertexIndex:   idx,
		vertexByIndex: rev,
		opts:          opts,
	}, nil
}

// buildGraphOptions prepares core.GraphOption slice from stored opts.
func (am *AdjacencyMatrix) buildGraphOptions() []core.GraphOption {
	var goOpts []core.GraphOption
	if am.opts.directed {
		goOpts = append(goOpts, core.WithDirected(true))
	}
	if am.opts.weighted {
		goOpts = append(goOpts, core.WithWeighted())
	}
	if am.opts.allowMulti {
		goOpts = append(goOpts, core.WithMultiEdges())
	}
	if am.opts.allowLoops {
		goOpts = append(goOpts, core.WithLoops())
	}

	return goOpts
}

// VertexCount returns the number of vertices in the graph (matrix dimension).
// Validate: receiver non‐nil, matrix shape vs index consistency.
func (am *AdjacencyMatrix) VertexCount() int {
	if am == nil || am.Mat == nil {
		panic("VertexCount: nil AdjacencyMatrix or underlying Mat")
	}
	// deep structural check
	if am.Mat.Rows() != len(am.vertexByIndex) {
		panic(fmt.Sprintf(
			"VertexCount: inconsistent dimensions %d vs %d",
			am.Mat.Rows(), len(am.vertexByIndex),
		))
	}

	return am.Mat.Rows()
}

// Neighbors returns all adjacent vertex IDs reachable from u.
// Stage 1 (Validate): receiver, lookup, shape check.
// Stage 2 (Prepare): allocate result slice.
// Stage 3 (Execute): single scan over columns.
// Stage 4 (Finalize): return.
func (am *AdjacencyMatrix) Neighbors(u string) ([]string, error) {
	// Validate receiver
	if am == nil || am.Mat == nil {
		return nil, fmt.Errorf("Neighbors: nil AdjacencyMatrix or Mat: %w", ErrGraphNil)
	}

	// Validate index exists
	srcIdx, ok := am.VertexIndex[u]
	if !ok {
		return nil, fmt.Errorf("Neighbors: unknown vertex %q: %w", u, ErrUnknownVertex)
	}

	// Validate shape
	cols := am.Mat.Cols()
	if cols != len(am.vertexByIndex) {
		return nil, fmt.Errorf(
			"Neighbors: dimension mismatch, cols=%d vs index=%d: %w",
			cols, len(am.vertexByIndex), ErrDimensionMismatch,
		)
	}

	// Prepare neighbor list and additional vars
	var (
		colIdx    int     // column index
		w         float64 // weight placeholder
		neighbors = make([]string, 0, defaultReserve)
		err       error  // error placeholder
		vid       string //
	)

	// Execute scan
	for colIdx = 0; colIdx < cols; colIdx++ {
		w, err = am.Mat.At(srcIdx, colIdx)
		if err != nil {
			return nil, fmt.Errorf("Neighbors: At(%d,%d): %w", srcIdx, colIdx, err)
		}
		// skip missing or infinite edges
		if w == 0 || w == math.Inf(1) {
			continue
		}
		// map index → vertex
		vid = am.vertexByIndex[colIdx]
		neighbors = append(neighbors, vid)
	}

	// Finalize
	return neighbors, nil
}

// indexToVertex returns the VertexID for a given matrix column index.
// Returns an error if index is out of range.
func (am *AdjacencyMatrix) indexToVertex(idx int) (string, error) {
	if idx < 0 || idx >= len(am.vertexByIndex) {
		return "", fmt.Errorf("indexToVertex: index %d out of range: %w", idx, ErrDimensionMismatch)
	}

	return am.vertexByIndex[idx], nil
}

// buildDenseAdjacencyFromGraph is a convenience wrapper used by tests
// and potential internal callers that have only *core.Graph*.
// It extracts the stable vertex-id list and edge list in a deterministic order
// and delegates to BuildDenseAdjacency.
//
// NOTE: we sort vertex IDs lexicographically here to be absolutely explicit,
// even if core.Vertices() is already sorted. This guarantees that callers that
// rely on this wrapper receive the canonical order.
// func buildDenseAdjacencyFromGraph(g *core.Graph, opts Options) (map[string]int, *Matrix, error) {
func buildDenseAdjacencyFromGraph(g *core.Graph, opts Options) (map[string]int, *Dense, error) {
	// Validate graph (public contract sentinel).
	if g == nil {
		return nil, nil, fmt.Errorf("buildDenseAdjacencyFromGraph: %w", ErrGraphNil)
	}

	// Pull vertex IDs from core; ensure deterministic lex order.
	ids := g.Vertices() // expected stable & sorted by core contract
	if !isLexSorted(ids) {
		// If not lex-sorted, sort defensively to meet our matrix determinism.
		cp := make([]string, len(ids))
		copy(cp, ids)
		sort.Strings(cp)
		ids = cp
	}

	// Pull edges in the order defined by core (Edge.ID asc).
	edges := g.Edges()

	// Delegate to main builder.
	return BuildDenseAdjacency(ids, edges, opts)
}

// ToGraph deterministic* export surface from an already
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
//   - Errors: ErrNilMatrix, ErrDimensionMismatch, ErrMatrixNotImplemented, plus bubbled core errors.
//
// AI-Hints:
//   - Set a low EdgeThreshold (e.g., 0 or 0.5) to export all non-zero edges reliably.
//   - Use Binary weights to get a clean unweighted graph for structural analytics.
//   - KeepWeights only makes sense if the adjacency was built as weighted.
//   - Export direction always mirrors the source adjacency’s Directed policy, ensuring
//     round-trip fidelity. Override via adapters before building if you need to flip.
//
// Complexity:
//   - O(n^2) reads of the matrix + O(n + m) vertex/edge insertions into core.
//   - No hidden allocations beyond necessary slices for vertex IDs.
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
	if am.opts.metricClose { // single, explicit flag - no reflective tricks
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

// DegreeVector computes per-vertex "degree" according to adjacency semantics:
//
//	– Directed: out-degree as row sum of outgoing entries.
//	– Undirected: degree as row sum; symmetric A implies sum equals degree
//	  for unweighted/binary adjacency.
//	– Loops: counted as exactly 1 (if present), regardless of stored weight.
//
// Determinism & Policy:
//   - +Inf denotes “no edge” and must NOT contribute to sums.
//   - NaN is ignored (treated as no edge) for robustness.
//   - Loop order is fixed (i → j) for stable accumulation.
//
// Complexity: O(n^2) time, O(n) extra space for the output.
//
// AI-Hints:
//   - For unweighted graphs, build a binary adjacency (1 for edges) to get pure degrees.
//   - For weighted directed graphs, this function returns row-sums (strength/out-degree).
//   - Prefer *Dense to avoid interface dispatch inside the double loop.
func (am *AdjacencyMatrix) DegreeVector() ([]float64, error) {
	// Validate container and matrix presence.
	if am == nil || am.Mat == nil {
		return nil, fmt.Errorf("DegreeVector: %w", ErrNilMatrix) // unified sentinel
	}
	// Validate square matrix (rows == cols).
	if err := ValidateSquare(am.Mat); err != nil {
		return nil, fmt.Errorf("DegreeVector: %w", err)
	}

	n := am.Mat.Rows()        // dimension of the matrix
	out := make([]float64, n) // allocate exactly one result vector
	// Fast-path: direct flat access on *Dense (row-major).
	if d, ok := am.Mat.(*Dense); ok {
		var i, j, base int      // loop indices and row base offset
		var s, v float64        // accumulator and current value
		for i = 0; i < n; i++ { // fixed outer loop (row)
			s = 0                   // reset accumulator for row i
			base = i * n            // compute base offset once per row
			for j = 0; j < n; j++ { // fixed inner loop (col)
				v = d.data[base+j] // read A[i,j]
				// Ignore invalid/unreachable (policy).
				if math.IsNaN(v) || math.IsInf(v, +1) {
					continue
				}
				// Early filter: only positive entries contribute.
				if v > 0 {
					if i == j {
						// Loop contributes exactly 1 if present.
						s += 1.0
					} else {
						// Off-diagonal contributes raw positive weight.
						s += v
					}
				}
			}
			out[i] = s // store degree/strength of vertex i
		}
		return out, nil // return fast-path result
	}

	// Fallback: interface path via At (bounds-safe; deterministic).
	var i, j int            // loop indices
	var s, v float64        // accumulator and current value
	var err error           // bubbled error
	for i = 0; i < n; i++ { // iterate rows deterministically
		s = 0                   // reset accumulator
		for j = 0; j < n; j++ { // iterate cols
			v, err = am.Mat.At(i, j) // read A[i,j]
			if err != nil {
				return nil, fmt.Errorf("DegreeVector: At(%d,%d): %w", i, j, err)
			}
			// Skip invalid/unreachable.
			if math.IsNaN(v) || math.IsInf(v, +1) {
				continue
			}
			// Only positive entries contribute.
			if v > 0 {
				if i == j {
					s += 1.0 // loop → exactly one
				} else {
					s += v // off-diagonal weight
				}
			}
		}
		out[i] = s // assign row sum
	}
	return out, nil
}
