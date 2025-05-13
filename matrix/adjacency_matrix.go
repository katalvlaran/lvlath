package matrix

import (
	"fmt"

	"github.com/katalvlaran/lvlath/core"
)

// AdjacencyMatrix holds a fixed-size, 2D representation of a graph.
//
// Description:
//
//	An Adjacency Matrix represents a graph as a 2D array where each cell
//	Data[i][j] holds the weight of the edge from vertex i to vertex j,
//	or zero if no edge exists.
//
// Use AdjacencyMatrix for constant-time edge existence and weight queries in dense graphs.
//
// Algorithm AdjacencyMatrix construction:
//  1. Build Index map: vertex ID → row/col index.
//  2. Allocate Data as an NxN zero‐filled slice.
//  3. Iterate all edges in graph:
//     Data[Index[from]][Index[to]] = weight.
//  4. For undirected graphs, Data[j][i] is set too.
//
// Time complexity:
//   - AddEdge/RemoveEdge: O(1)
//   - Neighbors: O(V)
//   - ToGraph: O(V² + E)
//
// Memory:
//   - O(V²).
type AdjacencyMatrix struct {
	// Index maps vertex ID → row/column index in Data.
	Index map[string]int
	// Data[i][j] holds the weight of edge i→j, or zero if none.
	Data     [][]int64
	directed bool
}

// NewAdjacencyMatrix builds an AdjacencyMatrix from g.
// The directed flag is set to g.Directed().
//
// Time Complexity: O(V² + E)
// Memory: O(V²)
func NewAdjacencyMatrix(g *core.Graph) *AdjacencyMatrix {
	verts := g.Vertices()
	n := len(verts)
	idx := make(map[string]int, n)
	for i, v := range verts {
		idx[v.ID] = i
	}

	data := make([][]int64, n)
	for i := range data {
		data[i] = make([]int64, n)
	}
	for _, e := range g.Edges() {
		i, j := idx[e.From.ID], idx[e.To.ID]
		data[i][j] = e.Weight
	}

	return &AdjacencyMatrix{
		Index:    idx,
		Data:     data,
		directed: g.Directed(),
	}
}

// AddEdge sets the capacity (weight) for edge fromID→toID. Returns an
// error if either vertex is unknown. In undirected mode it also sets
// the mirror entry.
//
// Time Complexity: O(1)
func (m *AdjacencyMatrix) AddEdge(fromID, toID string, weight int64) error {
	i, j, err := m.checkVertices(fromID, toID)
	if err != nil {
		return err
	}
	m.Data[i][j] = weight
	if !m.directed {
		m.Data[j][i] = weight
	}

	return nil
}

// RemoveEdge deletes the edge by setting its weight to zero.
// Returns an error if vertices are unknown.
//
// Time Complexity: O(1)
func (m *AdjacencyMatrix) RemoveEdge(fromID, toID string) error {
	return m.AddEdge(fromID, toID, 0)
}

// Neighbors returns all vertex IDs v for which Data[id][v] != 0.
// Returns an error if id is unknown.
//
// Time Complexity: O(V)
func (m *AdjacencyMatrix) Neighbors(id string) ([]string, error) {
	i, ok := m.Index[id]
	if !ok {
		return nil, fmt.Errorf("AdjacencyMatrix: unknown vertex %q", id)
	}
	var out []string
	for vid, j := range m.Index {
		if m.Data[i][j] != 0 {
			out = append(out, vid)
		}
	}

	return out, nil
}

// ToGraph reconstructs a *core.Graph from the matrix. The result will
// be directed if m.directed is true, and its edges are restored for all
// non-zero entries.
//
// Time Complexity: O(V² + E)
func (m *AdjacencyMatrix) ToGraph(weighted bool) *core.Graph {
	g := core.NewGraph(!m.directed, weighted)
	for id := range m.Index {
		g.AddVertex(&core.Vertex{ID: id, Metadata: make(map[string]interface{})})
	}
	for u, i := range m.Index {
		for v, j := range m.Index {
			if w := m.Data[i][j]; w != 0 {
				g.AddEdge(u, v, w)
			}
		}
	}

	return g
}

// checkVertices returns the matrix indices for fromID and toID,
// or an error if either vertex is not in the matrix.
func (m *AdjacencyMatrix) checkVertices(fromID, toID string) (i, j int, err error) {
	var ok bool
	i, ok = m.Index[fromID]
	if !ok {
		return 0, 0, fmt.Errorf("AdjacencyMatrix: unknown vertex %q", fromID)
	}
	j, ok = m.Index[toID]
	if !ok {
		return 0, 0, fmt.Errorf("AdjacencyMatrix: unknown vertex %q", toID)
	}

	return i, j, nil
}
