// Package matrix provides converters from core.Graph
// to simple matrix and edge-list representations.
package matrix

import "github.com/katalvlaran/lvlath/core"

// EdgeListItem is a flat representation of a single edge.
type EdgeListItem struct {
	FromID, ToID string
	Weight       int64
}

// ToEdgeList returns all edges in g as a slice of EdgeListItem.
// For undirected graphs, each edge appears twice (once per direction).
//
// Time Complexity: O(E)
func ToEdgeList(g *core.Graph) []EdgeListItem {
	var out []EdgeListItem
	for _, e := range g.Edges() {
		out = append(out, EdgeListItem{
			FromID: e.From.ID,
			ToID:   e.To.ID,
			Weight: e.Weight,
		})
	}

	return out
}

// Matrix is a lightweight adjacency-matrix representation.
//
// Index maps vertex ID → matrix row/column index.
// Data[i][j] holds the weight of the edge i→j or zero if absent.
type Matrix struct {
	Index map[string]int
	Data  [][]int64
}

// ToMatrix constructs a Matrix from g. If multiple edges exist between
// the same pair, the last one encountered sets the weight.
//
// Time Complexity: O(V + E)
// Memory: O(V²)
func ToMatrix(g *core.Graph) *Matrix {
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

	return &Matrix{Index: idx, Data: data}
}
