// Package matrix provides converters from core.Graph
// to simpler matrix or edge‐list representations.

package matrix

import "lvl_algorithms/graph/core"

// EdgeListItem is a simple exportable edge record.
type EdgeListItem struct {
	FromID, ToID string
	Weight       int64
}

// ToEdgeList returns all edges in g as a flat slice.
func ToEdgeList(g *core.Graph) []EdgeListItem {
	out := []EdgeListItem{}
	for _, e := range g.Edges() {
		out = append(out, EdgeListItem{
			FromID: e.From.ID,
			ToID:   e.To.ID,
			Weight: e.Weight,
		})
	}
	return out
}

// Matrix is a lightweight adjacency‐matrix representation.
type Matrix struct {
	Index map[string]int // vertex ID → index
	Data  [][]int64      // Data[i][j] = weight or 0
}

// ToMatrix builds a Matrix from g.
// For multiedges, the first weight encountered is used.
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
