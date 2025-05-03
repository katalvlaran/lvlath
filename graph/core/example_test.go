package core_test

import (
	"fmt"
	"sort"

	"github.com/katalvlaran/lvlath/graph/core"
)

// sortIDs is a tiny helper for predictable output.
func sortIDs(ids []string) []string {
	sort.Strings(ids)
	return ids
}

// ExampleGraph demonstrates basic creation, mutation, and queries.
func ExampleGraph() {
	// 1) Create an undirected, unweighted graph:
	g := core.NewGraph(false, false)

	// 2) Add edges (auto-adds vertices A, B, C):
	g.AddEdge("A", "B", 0)
	g.AddEdge("B", "C", 0)
	g.AddEdge("C", "A", 0)

	// 3) Inspect vertices and edges:
	vlist := g.Vertices()
	fmt.Println("Vertices:", sortIDs(coreIDs(vlist)))
	fmt.Println("Edge B→A exists?", g.HasEdge("B", "A"))

	// 4) Remove a vertex and its edges:
	g.RemoveVertex("B")
	fmt.Println("After removing B, vertices:", sortIDs(coreIDs(g.Vertices())))
	fmt.Println("Edge A→B exists?", g.HasEdge("A", "B"))

	// Output:
	// Vertices: [A B C]
	// Edge B→A exists? true
	// After removing B, vertices: [A C]
	// Edge A→B exists? false
}

// coreIDs extracts IDs from a slice of *core.Vertex.
func coreIDs(vs []*core.Vertex) []string {
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = v.ID
	}
	return out
}
