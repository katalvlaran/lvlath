// Package matrix provides matrix‐based representations of graphs
// and utilities to convert between core.Graph and matrix forms.

/*
AdjacencyMatrix

Description:
  An Adjacency Matrix represents a graph as a 2D array where each cell
  Data[i][j] holds the weight of the edge from vertex i to vertex j,
  or zero if no edge exists.

Use cases:
  - Constant‐time edge existence and weight lookups.
  - Best for dense or small graphs.

Time complexity:
  - AddEdge/RemoveEdge: O(1)
  - Neighbors: O(V)
  - ToGraph: O(V^2 + E)

Memory:
  - O(V^2)

Algorithm AdjacencyMatrix construction:
  1. Build Index map: vertex ID → row/col index.
  2. Allocate Data as an NxN zero‐filled slice.
  3. Iterate all edges in graph:
     Data[Index[from]][Index[to]] = weight.
  4. For undirected graphs, Data[j][i] is set too.
*/

package matrix

import (
	"fmt"

	"github.com/katalvlaran/lvlath/graph/core"
)

// AdjacencyMatrix holds a graph in matrix form.
type AdjacencyMatrix struct {
	Index    map[string]int // vertex ID → row/col index
	Data     [][]int64      // Data[i][j] = weight (0 if no edge)
	directed bool
}

// NewAdjacencyMatrix builds an AdjacencyMatrix from a core.Graph.
func NewAdjacencyMatrix(g *core.Graph) *AdjacencyMatrix {
	verts := g.Vertices()
	n := len(verts)
	idx := make(map[string]int, n)
	for i, v := range verts {
		idx[v.ID] = i
	}

	// allocate NxN zero matrix
	mat := make([][]int64, n)
	for i := range mat {
		mat[i] = make([]int64, n)
	}

	// fill in edges
	for _, e := range g.Edges() {
		i, j := idx[e.From.ID], idx[e.To.ID]
		mat[i][j] = e.Weight
	}

	return &AdjacencyMatrix{Index: idx, Data: mat, directed: g.Directed()}
}

// AddEdge inserts or updates an edge weight in O(1).
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

// RemoveEdge removes the edge by setting weight to zero.
func (m *AdjacencyMatrix) RemoveEdge(fromID, toID string) error {
	return m.AddEdge(fromID, toID, 0)
}

// Neighbors returns IDs of all vertices v with Data[id][v] != 0.
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

// ToGraph reconstructs a core.Graph from this matrix.
// weighted controls whether the resulting graph is marked weighted.
func (m *AdjacencyMatrix) ToGraph(weighted bool) *core.Graph {
	g := core.NewGraph(!m.directed, weighted)
	// add vertices
	for id := range m.Index {
		g.AddVertex(&core.Vertex{ID: id, Metadata: make(map[string]interface{})})
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

// checkVertices returns indices for fromID and toID or an error.
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
