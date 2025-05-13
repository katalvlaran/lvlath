package core_test

import (
	"fmt"
	"sort"

	"github.com/katalvlaran/lvlath/core"
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

// ExampleGraph_basic shows how to add and remove vertices.
func ExampleGraph_basic() {
	// Create an undirected, weighted graph
	g := core.NewGraph(false, true)

	// Add an edge with weight 5 (auto-adds vertices)
	g.AddEdge("A", "B", 5)
	// We now have 2 vertices and the mirror edge exists in undirected mode
	fmt.Println(len(g.Vertices()), g.HasEdge("B", "A"))

	// Remove vertex A and all its edges
	g.RemoveVertex("A")
	fmt.Println(len(g.Vertices()), g.HasVertex("A"))

	// Output:
	// 2 true
	// 1 false
}

// ExampleGraph_loops demonstrates self-loops and multiedges.
func ExampleGraph_loops() {
	// Undirected, unweighted graph
	g := core.NewGraph(false, false)

	// Add two self-loops with different weights
	g.AddEdge("X", "X", 1)
	g.AddEdge("X", "X", 2)

	// Count distinct logical loops (ignore mirror duplicates for self-loops)
	count := 0
	for _, e := range g.Edges() {
		if e.From.ID == "X" && e.To.ID == "X" {
			count++
		}
	}
	fmt.Println(count)

	// Output:
	// 2
}

// coreIDs extracts IDs from a slice of *core.Vertex.
func coreIDs(vs []*core.Vertex) []string {
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = v.ID
	}

	return out
}
