// SPDX-License-Identifier: MIT
// Package matrix — adjacency builders (dense) and metric-closure transform.
//
// Stage-2 deliverables (per TA-MATRIX):
//   1) Directed + AllowMulti=false → first-edge-wins (ordered key (u,v)).
//   2) Undirected mirroring without loops (u==v is not mirrored).
//   3) Weighted=true but effectively-unweighted input → degrade to binary (1).
//   4) MetricClosure (Floyd–Warshall): diag=0, unreachable=+Inf (off-diagonal).
//   5) Deterministic iteration & stable vertex/edge order (no map order reliance).
//
// AI-Hints:
//   • For directed graphs, duplicate (u,v) edges are ignored when AllowMulti=false;
//     the *first* occurrence wins. For undirected graphs, the first of unordered
//     pair {min(u), max(v)} wins. This guarantees deterministic results.
//   • When input graph is effectively unweighted (all weights are 0 or graph flags
//     indicate unweighted) and options request Weighted=true, we intentionally build
//     a binary adjacency (1) to avoid an all-zero matrix.
//   • MetricClosure turns adjacency into pairwise shortest-path distances. It is
//     *not* an adjacency anymore, and ToGraph must return ErrMatrixNotImplemented
//     (enforced in impl_adjacency_export.go).
//
// Complexity (worst case):
//   • BuildDenseAdjacency: O(|V| + |E|) for construction + O(1) index lookups.
//   • MetricClosure (optional): O(n^3) time, O(n^2) memory (in-place over dense).

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

// --- Minimal core.Edge shape expectations ---------------------------------------
// We assume the following exported fields exist on core.Edge, per core package
// conventions used elsewhere in lvlath (ToGraph, tests, etc.):
//
//   type Edge struct {
//       ID     int64   // monotonically increasing identifier (ordering guarantee)
//       From   string  // source vertex id
//       To     string  // target vertex id
//       Weight int64   // edge weight; 0 for unweighted graphs
//   }
//
// If core evolves, keep this adapter in sync with TS-CORE/TA-CORE contracts.
