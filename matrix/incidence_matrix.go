// Package matrix provides graph-aware wrappers for incidence matrix operations.
package matrix

import (
	"sort"

	"github.com/katalvlaran/lvlath/core"
)

// IncidenceMatrix represents a graph as a V×E matrix mapping vertices to edges.
// VertexIndex maps vertex ID → row index in Data (immutable).
// Edges holds the ordered list of *core.Edge corresponding to columns.
// Data[i][j] holds:
//   - For directed: -1 if row i is source, +1 if target, 0 otherwise.
//   - For undirected: 1 for both endpoints, 0 otherwise.
//
// Construction options (loops, multi-edges) are stored in opts for fidelity.
// Use NewIncidenceMatrix to build with validation and reproducibility.
type IncidenceMatrix struct {
	VertexIndex map[string]int // vertex to row map
	Edges       []*core.Edge   // column-edge mapping
	Data        [][]int        // V×E incidence matrix
	opts        MatrixOptions  // original build options
}

// NewIncidenceMatrix builds an IncidenceMatrix from a core.Graph and options.
// It returns ErrNilGraph if g is nil, ErrUnknownVertex for missing vertices,
// and ErrNonBinaryIncidence if an unweighted matrix contains non-±1 entries.
// Time: O(V+E); Memory: O(V·E).
func NewIncidenceMatrix(g *core.Graph, opts MatrixOptions) (IncidenceMatrix, error) {
	if g == nil {
		return IncidenceMatrix{}, ErrNilGraph
	}
	verts := g.Vertices()
	rawEdges := g.Edges()
	// buildIncidenceData applies options and filters edges
	vIdx, cols, data, err := BuildIncidenceData(verts, rawEdges, opts)
	if err != nil {
		return IncidenceMatrix{}, err
	}
	// Validate binary entries for unweighted graphs
	if !opts.Weighted {
		for i := range data {
			for _, val := range data[i] {
				if val != -1 && val != 0 && val != 1 {
					return IncidenceMatrix{}, ErrNonBinaryIncidence
				}
			}
		}
	}
	// Sort edges by ID for reproducibility and reorder columns accordingly
	sort.SliceStable(cols, func(i, j int) bool {
		return cols[i].ID < cols[j].ID
	})
	// Rebuild data to match sorted order
	eCount := len(cols)
	newData := make([][]int, len(verts))
	for i := range newData {
		newData[i] = make([]int, eCount)
	}
	for j, e := range cols {
		iF := vIdx[e.From]
		iT := vIdx[e.To]
		if opts.Directed {
			newData[iF][j] = -1
			newData[iT][j] = +1
		} else {
			newData[iF][j] = +1
			newData[iT][j] = +1
		}
	}
	// Clone vertex index
	vIdxCopy := make(map[string]int, len(vIdx))
	for k, v := range vIdx {
		vIdxCopy[k] = v
	}
	// Clone edges slice to avoid external mutation
	edgesCopy := make([]*core.Edge, len(cols))
	for i, e := range cols {
		edgesCopy[i] = e
	}

	return IncidenceMatrix{
		VertexIndex: vIdxCopy,
		Edges:       edgesCopy,
		Data:        newData,
		opts:        opts,
	}, nil
}

// VertexCount returns the number of vertices (rows).
func (m IncidenceMatrix) VertexCount() int {
	return len(m.VertexIndex)
}

// EdgeCount returns the number of edges (columns).
func (m IncidenceMatrix) EdgeCount() int {
	return len(m.Edges)
}

// VertexIncidence returns the incidence row for vertexID.
// Returns ErrUnknownVertex if the vertex is not indexed.
// Time: O(1).
func (m IncidenceMatrix) VertexIncidence(vertexID string) ([]int, error) {
	i, ok := m.VertexIndex[vertexID]
	if !ok {
		return nil, ErrUnknownVertex
	}
	row := make([]int, len(m.Data[i]))
	copy(row, m.Data[i])

	return row, nil
}

// EdgeEndpoints returns the source and target IDs of the edge at column j.
// Returns ErrDimensionMismatch if j is out of range.
// Time: O(1).
func (m IncidenceMatrix) EdgeEndpoints(j int) (fromID, toID string, err error) {
	if j < 0 || j >= len(m.Edges) {
		return "", "", ErrDimensionMismatch
	}
	e := m.Edges[j]

	return e.From, e.To, nil
}
