package matrix

import (
	"errors"

	"github.com/katalvlaran/lvlath/core"
)

// IncidenceMatrix represents a graph as a vertex-by-edge matrix.
// Rows correspond to vertices in core.Graph.Vertices() order;
// columns correspond to a filtered, unique edge list.
//
// For directed graphs:
//
//	Data[i][j] = -1 if vertex i is the source of edge j,
//	             +1 if vertex i is the target,
//	              0 otherwise.
//
// For undirected graphs:
//
//	Data[i][j] = 1 for both endpoints.
//
// Use cases: incidence queries, algebraic graph operations.
//
// Time Complexity: O(V + E)
// Memory: O(V·E)
type IncidenceMatrix struct {
	// VertexIndex maps vertex ID → row index in Data.
	VertexIndex map[string]int
	// Edges is the ordered slice of *core.Edge columns.
	Edges []*core.Edge
	// Data[row][col] holds the incidence value as above.
	Data [][]int
}

// NewIncidenceMatrix builds an incidence matrix from g.
// In undirected mode, only one representative per edge is used;
// self-loops are skipped automatically.
//
// Time Complexity: O(V + E)
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

	return &IncidenceMatrix{
		VertexIndex: vIdx,
		Edges:       edges,
		Data:        data,
	}
}

// VertexIncidence returns the incidence row for vertexID,
// or an error if the vertex is unknown.
//
// Time Complexity: O(1)
func (m *IncidenceMatrix) VertexIncidence(vertexID string) ([]int, error) {
	i, ok := m.VertexIndex[vertexID]
	if !ok {
		return nil, errors.New("matrix: unknown vertex")
	}

	return m.Data[i], nil
}

// EdgeEndpoints returns the (fromID, toID) of the edge at column j,
// or an error if j is out of bounds.
//
// Time Complexity: O(1)
func (m *IncidenceMatrix) EdgeEndpoints(j int) (fromID, toID string, err error) {
	if j < 0 || j >= len(m.Edges) {
		return "", "", errors.New("matrix: edge index out of range")
	}
	e := m.Edges[j]
	return e.From.ID, e.To.ID, nil
}

// filterUniqueEdges returns one representative per undirected edge.
// Self-loops are skipped. In directed graphs, all edges are returned.
//
// Time Complexity: O(E)
func filterUniqueEdges(g *core.Graph) []*core.Edge {
	all := g.Edges()
	if g.Directed() {
		return all
	}
	seen := make(map[string]map[string]bool, len(all))
	var unique []*core.Edge

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
