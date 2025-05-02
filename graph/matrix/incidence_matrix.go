// Package matrix: incidence matrix representation and conversion.

package matrix

import (
	"errors"

	"lvl_algorithms/graph/core"
)

/*
IncidenceMatrix

Description:
  Represents a graph as a vertex‐by‐edge matrix.
  Rows correspond to vertices, columns to edges.

  For directed graphs:
    Data[i][j] = -1 if vertex i is the source ("from"),
                 +1 if vertex i is the target ("to"),
                 0 otherwise.
  For undirected:
    Data[i][j] = 1 for both endpoints.

Use cases:
  - Quick incidence queries.
  - Graph‐theoretic matrix operations.
*/

type IncidenceMatrix struct {
	VertexIndex map[string]int // vertex ID → row index
	Edges       []*core.Edge   // column order
	Data        [][]int        // Data[row][col]
}

// NewIncidenceMatrix builds an IncidenceMatrix from a core.Graph.
func NewIncidenceMatrix(g *core.Graph) *IncidenceMatrix {
	verts := g.Vertices()
	n := len(verts)
	edges := filterUniqueEdges(g)
	m := len(edges)

	// map vertex IDs to rows
	vIdx := make(map[string]int, n)
	for i, v := range verts {
		vIdx[v.ID] = i
	}

	// allocate data
	data := make([][]int, n)
	for i := range data {
		data[i] = make([]int, m)
	}

	// fill in incidence
	for j, e := range edges {
		iFrom := vIdx[e.From.ID]
		iTo := vIdx[e.To.ID]
		if g.Directed() {
			data[iFrom][j] = -1
			data[iTo][j] = 1
		} else {
			data[iFrom][j] = 1
			data[iTo][j] = 1
		}
	}

	return &IncidenceMatrix{VertexIndex: vIdx, Edges: edges, Data: data}
}

// VertexIncidence returns the incidence row for a given vertex ID.
func (m *IncidenceMatrix) VertexIncidence(vertexID string) ([]int, error) {
	i, ok := m.VertexIndex[vertexID]
	if !ok {
		return nil, errors.New("incidence: unknown vertex")
	}
	return m.Data[i], nil
}

// EdgeEndpoints returns the endpoints (fromID, toID) of column j.
func (m *IncidenceMatrix) EdgeEndpoints(j int) (fromID, toID string, err error) {
	if j < 0 || j >= len(m.Edges) {
		return "", "", errors.New("incidence: edge index out of range")
	}
	e := m.Edges[j]
	return e.From.ID, e.To.ID, nil
}

// filterUniqueEdges returns one representative per undirected edge.
func filterUniqueEdges(g *core.Graph) []*core.Edge {
	all := g.Edges()
	if g.Directed() {
		return all
	}
	seen := make(map[string]map[string]bool)
	unique := make([]*core.Edge, 0, len(all))
	for _, e := range all {
		u, v := e.From.ID, e.To.ID
		if u == v {
			continue // skip self-loops
		}
		// normalize order
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
