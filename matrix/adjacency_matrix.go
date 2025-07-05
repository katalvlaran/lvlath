// Package matrix provides graph-aware wrappers for adjacency matrix operations.
package matrix

import (
	"fmt"
	"math"

	"github.com/katalvlaran/lvlath/core"
)

// AdjacencyMatrix represents a graph as an N×N dense matrix.
// Index maps vertex ID → row/column index in Data (immutable).
// Data[i][j] holds the edge weight (float64) from vertex i to j, or zero if none.
// Original construction options are stored in opts for round-trip fidelity.
// See: NewAdjacencyMatrix for usage example.
type AdjacencyMatrix struct {
	Index map[string]int // vertex to index map (internal copy)
	Data  [][]float64    // dense weight matrix
	opts  MatrixOptions  // original build options
}

// NewAdjacencyMatrix constructs an AdjacencyMatrix from a core.Graph.
// It extracts vertices (g.Vertices()) and edges (g.Edges()), then delegates to buildAdjacencyData
// and returns ErrNilGraph if g is nil.
// Round-trip: am, _ := NewAdjacencyMatrix(g, opts); g2, _ := am.ToGraph(); g.Isomorphic(g2)
// Time: O(V+E); Memory: O(V²).
func NewAdjacencyMatrix(g *core.Graph, opts MatrixOptions) (AdjacencyMatrix, error) {
	if g == nil {
		return AdjacencyMatrix{}, ErrNilGraph
	}
	verts := g.Vertices()
	edges := g.Edges()
	// buildAdjacencyData handles unknown vertices and options
	idx, data, err := BuildAdjacencyData(verts, edges, opts)
	if err != nil {
		return AdjacencyMatrix{}, err
	}
	// store a copy of idx to prevent external mutation
	idxCopy := make(map[string]int, len(idx))
	for k, v := range idx {
		idxCopy[k] = v
	}

	return AdjacencyMatrix{Index: idxCopy, Data: data, opts: opts}, nil
}

// VertexCount returns the number of vertices (matrix dimension).
func (m AdjacencyMatrix) VertexCount() int {
	return len(m.Index)
}

// EdgeCount returns the number of non-zero edges in the matrix.
func (m AdjacencyMatrix) EdgeCount() int {
	count := 0
	n := len(m.Data)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if m.Data[i][j] != 0 {
				count++
			}
		}
	}

	return count
}

// Transpose returns a new AdjacencyMatrix that is the transpose of m.
// Note: Index map remains identical, but Data rows/cols are swapped.
// Time: O(V²); Memory: O(V²).
func (m AdjacencyMatrix) Transpose() AdjacencyMatrix {
	t := TransposeData(m.Data)

	return AdjacencyMatrix{Index: m.cloneIndex(), Data: t, opts: m.opts}
}

// Multiply returns the matrix product of m and other, both N×N.
// Returns ErrDimensionMismatch if matrices differ in size.
// Time: O(V³); Memory: O(V²).
func (m AdjacencyMatrix) Multiply(other AdjacencyMatrix) (AdjacencyMatrix, error) {
	// validate dimensions
	n := len(m.Data)
	if n != len(other.Data) || n > 0 && len(other.Data[0]) != n {
		return AdjacencyMatrix{}, ErrDimensionMismatch
	}
	data, err := MultiplyData(m.Data, other.Data)
	if err != nil {
		return AdjacencyMatrix{}, err
	}

	return AdjacencyMatrix{Index: m.cloneIndex(), Data: data, opts: m.opts}, nil
}

// DegreeVector computes the degree (sum of weights) of each vertex.
// For unweighted graphs, all non-zero entries count as 1.
// Time: O(V²).
func (m AdjacencyMatrix) DegreeVector() []float64 {
	return DegreeFromData(m.Data)
}

// SpectralAnalysis performs eigen decomposition with Jacobi rotations.
// tol is convergence threshold; maxIter caps rotations.
// See EigenDecompose for details; Time: O(V³).
func (m AdjacencyMatrix) SpectralAnalysis(tol float64, maxIter int) ([]float64, [][]float64, error) {
	return EigenDecompose(m.Data, tol, maxIter)
}

// ToGraph reconstructs a core.Graph using stored MatrixOptions.
// It returns ErrMultiEdgeNotAllowed if core rejects duplicates when AllowMulti=false.
// Time: O(V²+E); Memory: O(V+E).
func (m AdjacencyMatrix) ToGraph() (*core.Graph, error) {
	// configure graph options
	var optsG []core.GraphOption
	if m.opts.Directed {
		optsG = append(optsG, core.WithDirected(true))
	}
	if m.opts.Weighted {
		optsG = append(optsG, core.WithWeighted())
	}
	if m.opts.AllowMulti {
		optsG = append(optsG, core.WithMultiEdges())
	}
	if m.opts.AllowLoops {
		optsG = append(optsG, core.WithLoops())
	}
	g := core.NewGraph(optsG...)
	// add vertices
	for v := range m.Index {
		if err := g.AddVertex(v); err != nil {
			return nil, fmt.Errorf("matrix: ToGraph add vertex %s: %w", v, err)
		}
	}
	// add edges
	for u, i := range m.Index {
		for v, j := range m.Index {
			w := m.Data[i][j]
			if w == 0 {
				continue
			}
			if math.IsInf(w, 1) {
				continue
			}

			var weight int64
			if m.opts.Weighted {
				weight = int64(w)
			}
			if _, err := g.AddEdge(u, v, weight); err != nil {
				if err == core.ErrMultiEdgeNotAllowed {
					return nil, fmt.Errorf("matrix: duplicate edge %s->%s: %w", u, v, err)
				}

				return nil, fmt.Errorf("matrix: ToGraph add edge %s->%s: %w", u, v, err)
			}
		}
	}

	return g, nil
}

// cloneIndex returns a deep copy of the index map.
func (m AdjacencyMatrix) cloneIndex() map[string]int {
	out := make(map[string]int, len(m.Index))
	for k, v := range m.Index {
		out[k] = v
	}

	return out
}
