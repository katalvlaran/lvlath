package graph

import (
	"fmt"
)

// AdjacencyMatrix represents a graph in adjacency-matrix form.
type AdjacencyMatrix struct {
	Index    map[string]int // vertex ID → row/col index
	Data     [][]int64      // Data[i][j] = weight (0 if no edge)
	directed bool
}

// NewAdjacencyMatrix builds an AdjacencyMatrix from g.
func NewAdjacencyMatrix(g *Graph) *AdjacencyMatrix {
	verts := g.Vertices()
	n := len(verts)
	idx := make(map[string]int, n)
	for i, v := range verts {
		idx[v.ID] = i
	}
	mat := make([][]int64, n)
	for i := range mat {
		mat[i] = make([]int64, n)
	}
	for _, e := range g.Edges() {
		i, j := idx[e.From.ID], idx[e.To.ID]
		mat[i][j] = e.Weight
	}
	return &AdjacencyMatrix{Index: idx, Data: mat, directed: g.directed}
}

// AddEdge inserts or updates an edge in the matrix.
func (m *AdjacencyMatrix) AddEdge(fromID, toID string, weight int64) error {
	i, ok1 := m.Index[fromID]
	j, ok2 := m.Index[toID]
	if !ok1 || !ok2 {
		return fmt.Errorf("AdjacencyMatrix: unknown vertex %q or %q", fromID, toID)
	}
	m.Data[i][j] = weight
	if !m.directed {
		m.Data[j][i] = weight
	}
	return nil
}

// RemoveEdge deletes an edge (sets weight to zero).
func (m *AdjacencyMatrix) RemoveEdge(fromID, toID string) error {
	return m.AddEdge(fromID, toID, 0)
}

// Neighbors returns IDs of vertices adjacent to id.
func (m *AdjacencyMatrix) Neighbors(id string) ([]string, error) {
	idx, ok := m.Index[id]
	if !ok {
		return nil, fmt.Errorf("AdjacencyMatrix: unknown vertex %q", id)
	}
	out := make([]string, 0)
	for vid, vidx := range m.Index {
		if m.Data[idx][vidx] != 0 {
			out = append(out, vid)
		}
	}
	return out, nil
}

// ToGraph converts the matrix back into a Graph.
func (m *AdjacencyMatrix) ToGraph(weighted bool) *Graph {
	g := NewGraph(!m.directed, weighted)
	// add vertices
	for id := range m.Index {
		g.AddVertex(&Vertex{ID: id, Metadata: make(map[string]interface{})})
	}
	// add edges
	for u, i := range m.Index {
		for v, j := range m.Index {
			if w := m.Data[i][j]; w != 0 {
				g.AddEdge(u, v, w)
			}
		}
	}
	return g
}
