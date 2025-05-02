package graph

// EdgeListItem is a simple struct for exporting graph edges.
type EdgeListItem struct {
	FromID, ToID string
	Weight       int64
}

// ToEdgeList returns a slice of EdgeListItem for easy export.
// Note: EdgeListItem includes weight regardless of the Graph.weighted flag.
func (g *Graph) ToEdgeList() []EdgeListItem {
	g.mu.RLock()
	defer g.mu.RUnlock()

	list := make([]EdgeListItem, 0)
	for _, e := range g.Edges() {
		list = append(list, EdgeListItem{FromID: e.From.ID, ToID: e.To.ID, Weight: e.Weight})
	}
	return list
}

// Matrix is the adjacency matrix representation of the graph.
// Index maps vertex ID to row/column index in Data.
type Matrix struct {
	Index map[string]int // maps vertex ID to matrix index
	Data  [][]int64      // Data[i][j] = weight or 0
}

// ToMatrix constructs and returns a full adjacency matrix.
// For multiple edges, the first weight encountered is used.
func (g *Graph) ToMatrix() *Matrix {
	g.mu.RLock()
	defer g.mu.RUnlock()

	verts := g.Vertices()
	n := len(verts)
	idx := make(map[string]int, n)
	for i, v := range verts {
		idx[v.ID] = i
	}
	m := make([][]int64, n)
	for i := range m {
		m[i] = make([]int64, n)
	}
	for _, e := range g.Edges() {
		i, j := idx[e.From.ID], idx[e.To.ID]
		m[i][j] = e.Weight
	}
	return &Matrix{Index: idx, Data: m}
}
