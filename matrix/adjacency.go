// Package matrix provides graph‐aware wrappers over the core Matrix API,
// exposing high‐level methods for adjacency‐matrix representations of graphs.
package matrix

import (
	"errors"
	"fmt"
	"math"

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
	opts          MatrixOptions  // original construction options
}

// NewAdjacencyMatrix constructs an AdjacencyMatrix from g.
// Stage 1 (Validate): ensure g is non‐nil.
// Stage 2 (Prepare): extract vertex list and edge list.
// Stage 3 (Execute): call BuildDenseAdjacency.
// Stage 4 (Finalize): build reverse lookup and return.
// Returns ErrNilGraph or any BuildDenseAdjacency error.
func NewAdjacencyMatrix(g *core.Graph, opts MatrixOptions) (*AdjacencyMatrix, error) {
	// Validate input graph
	if g == nil {
		return nil, ErrMatrixNilGraph
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
	if am.opts.Directed {
		goOpts = append(goOpts, core.WithDirected(true))
	}
	if am.opts.Weighted {
		goOpts = append(goOpts, core.WithWeighted())
	}
	if am.opts.AllowMulti {
		goOpts = append(goOpts, core.WithMultiEdges())
	}
	if am.opts.AllowLoops {
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
		return nil, fmt.Errorf("Neighbors: nil AdjacencyMatrix or Mat: %w", ErrMatrixNilGraph)
	}

	// Validate index exists
	srcIdx, ok := am.VertexIndex[u]
	if !ok {
		return nil, fmt.Errorf("Neighbors: unknown vertex %q: %w", u, ErrMatrixUnknownVertex)
	}

	// Validate shape
	cols := am.Mat.Cols()
	if cols != len(am.vertexByIndex) {
		return nil, fmt.Errorf(
			"Neighbors: dimension mismatch, cols=%d vs index=%d: %w",
			cols, len(am.vertexByIndex), ErrMatrixDimensionMismatch,
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
		return "", fmt.Errorf("indexToVertex: index %d out of range: %w", idx, ErrMatrixDimensionMismatch)
	}
	return am.vertexByIndex[idx], nil
}

// ToGraph reconstructs a core.Graph from this adjacency matrix.
// Stage 1 (Validate): receiver and shape.
// Stage 2 (Prepare): graph options.
// Stage 3 (Execute): add vertices and edges.
// Stage 4 (Finalize): return.
// Errors are wrapped with context.
func (am *AdjacencyMatrix) ToGraph() (*core.Graph, error) {
	// Validate receiver
	if am == nil || am.Mat == nil {
		return nil, fmt.Errorf("ToGraph: nil AdjacencyMatrix or Mat: %w", ErrMatrixNilGraph)
	}

	// Validate consistent dimensions
	n := am.Mat.Rows()
	if n != am.Mat.Cols() || n != len(am.vertexByIndex) {
		return nil, fmt.Errorf(
			"ToGraph: dimension mismatch rows=%d, cols=%d, idx=%d: %w",
			n, am.Mat.Cols(), len(am.vertexByIndex), ErrMatrixDimensionMismatch,
		)
	}

	// Prepare graph
	g := core.NewGraph(am.buildGraphOptions()...)

	// Add edges
	var (
		fromID, toID   string  // vertex IDs
		fromIdx, toIdx int     // indices
		w              float64 // weight
		err            error
	)
	for fromIdx, fromID = range am.vertexByIndex {
		for toIdx, toID = range am.vertexByIndex {
			w, err = am.Mat.At(fromIdx, toIdx)
			if err != nil {
				return nil, fmt.Errorf("ToGraph: At(%d,%d): %w", fromIdx, toIdx, err)
			}
			if w == 0 || w == math.Inf(1) {
				continue // skip absent or infinite edges
			}
			// convert weight if necessary
			var weight int64
			if am.opts.Weighted {
				weight = int64(w)
			}
			if _, err = g.AddEdge(fromID, toID, weight); err != nil {
				if errors.Is(err, core.ErrMultiEdgeNotAllowed) {
					return nil, fmt.Errorf("ToGraph: duplicate edge %q->%q: %w", fromID, toID, err)
				}

				return nil, fmt.Errorf("ToGraph: AddEdge %q->%q: %w", fromID, toID, err)
			}
		}
	}

	// Finalize
	return g, nil
}
