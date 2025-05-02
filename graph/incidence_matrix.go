package graph

import (
	"errors"
)

// IncidenceMatrix represents a graph in vertex-by-edge form.
// Rows ⇢ vertices, Columns ⇢ edges.
type IncidenceMatrix struct {
	VertexIndex map[string]int // ID → row
	Edges       []*Edge        // list of edges (column order)
	Data        [][]int        // Data[row][col]: for directed: -1=from, +1=to; undirected: 1/1
}

// NewIncidenceMatrix builds an incidence matrix from g.
func NewIncidenceMatrix(g *Graph) *IncidenceMatrix {
	verts := g.Vertices()
	n := len(verts)
	edges := filterUniqueEdges(g)
	m := len(edges)

	vIdx := make(map[string]int, n)
	for i, v := range verts {
		vIdx[v.ID] = i
	}

	data := make([][]int, n)
	for i := range data {
		data[i] = make([]int, m)
	}

	for j, e := range edges {
		iFrom := vIdx[e.From.ID]
		iTo := vIdx[e.To.ID]
		if g.directed {
			data[iFrom][j] = -1
			data[iTo][j] = 1
		} else {
			data[iFrom][j] = 1
			data[iTo][j] = 1
		}
	}

	return &IncidenceMatrix{VertexIndex: vIdx, Edges: edges, Data: data}
}

// VertexIncidence returns the row for a given vertex ID.
func (m *IncidenceMatrix) VertexIncidence(vertexID string) ([]int, error) {
	i, ok := m.VertexIndex[vertexID]
	if !ok {
		return nil, errors.New("incidence: unknown vertex")
	}
	return m.Data[i], nil
}

// EdgeEndpoints returns (fromID, toID) for the j-th column.
func (m *IncidenceMatrix) EdgeEndpoints(j int) (fromID, toID string, err error) {
	if j < 0 || j >= len(m.Edges) {
		return "", "", errors.New("incidence: edge index out of range")
	}
	e := m.Edges[j]
	return e.From.ID, e.To.ID, nil
}

// filterUniqueEdges returns one representative per undirected edge.
func filterUniqueEdges(g *Graph) []*Edge {
	all := g.Edges()
	if g.directed {
		return all
	}
	seen := make(map[string]map[string]bool)
	unique := make([]*Edge, 0, len(all))
	for _, e := range all {
		u, v := e.From.ID, e.To.ID
		if u == v {
			continue // skip self-loops
		}
		if u > v {
			u, v = v, u
		}
		if seen[u] == nil {
			seen[u] = make(map[string]bool)
		}
		if !seen[u][v] {
			seen[u][v] = true
			unique = append(unique, e)
		}
	}
	return unique
}
