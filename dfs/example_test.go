package dfs_test

import (
	"fmt"
	"strings"

	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/dfs"
)

// ExampleDFS demonstrates a depth-first traversal (post-order) on a diamond-shaped graph.
// Graph structure:
//
//	  A
//	 / \
//	B   C
//	 \ /
//	  D
//	 / \
//	E   F
//
// Starting at "A", expected post-order: E F D B C A
func ExampleDFS() {
	// Build a new directed graph
	g := core.NewGraph(core.WithDirected(true))

	// Add directed edges to form the diamond shape:
	// A -> B, A -> C, B -> D, C -> D, D -> E, D -> F
	for _, edge := range []struct{ U, V string }{
		{"A", "B"}, {"A", "C"},
		{"B", "D"}, {"C", "D"},
		{"D", "E"}, {"D", "F"},
	} {
		// We ignore errors here for brevity; AddEdge creates the vertices if needed.
		_, _ = g.AddEdge(edge.U, edge.V, 0)
	}

	// Perform DFS starting from vertex "A"
	res, err := dfs.DFS(g, "A")
	if err != nil {
		// If an error occurred (e.g., missing start vertex), print and exit
		fmt.Println("error:", err)
		return
	}

	// res.Order is the post-order traversal of the DFS.
	// We join the slice of vertex IDs with spaces for printing.
	fmt.Println(strings.Join(res.Order, " "))

	// Output (exact post-order for this structure):
	// E F D B C A
}

// ExampleTopologicalSort demonstrates computing a valid topological order
// on a DAG with a shared child D. Graph:
//
//	  A
//	 / \
//	B   C
//	 \ / \
//	  D   G
//	 / \   \
//	E   F   H
//
// One valid topological order is: A C G H B D F E
func ExampleTopologicalSort() {
	// Build a new directed graph
	g := core.NewGraph(core.WithDirected(true))

	// Add directed edges to form the DAG structure:
	// A -> B, A -> C, B -> D, C -> D, C -> G, D -> E, D -> F, G -> H
	for _, edge := range []struct{ U, V string }{
		{"A", "B"}, {"A", "C"},
		{"B", "D"}, {"C", "D"}, {"C", "G"},
		{"D", "E"}, {"D", "F"}, {"G", "H"},
	} {
		// AddEdge will create missing vertices automatically.
		_, _ = g.AddEdge(edge.U, edge.V, 0)
	}

	// Compute a topological sort of the entire graph
	order, err := dfs.TopologicalSort(g)
	if err != nil {
		// If an error occurred (e.g., cycle detected), print and exit
		fmt.Println("error:", err)
		return
	}

	// Print the topological order, joining vertex IDs with spaces
	fmt.Println(strings.Join(order, " "))

	// Output (one valid ordering; actual order may vary among valid permutations):
	// A C G H B D F E
}

// ExampleDetectCycles shows detecting cycles in a directed graph.
// Constructs a graph that contains a cycle involving vertices B, D, H, I, J, K, then prints the cycle.
func ExampleDetectCycles() {
	// Create a new directed graph
	g := core.NewGraph(core.WithDirected(true))

	// Add directed edges, deliberately creating a cycle:
	// A->B, B->C, B->D, C->E, E->F, F->G, D->H, H->I, I->J, J->K, K->B
	_, _ = g.AddEdge("A", "B", 0) // AddEdge creates vertices if they don’t exist yet
	_, _ = g.AddEdge("B", "C", 0)
	_, _ = g.AddEdge("B", "D", 0)
	_, _ = g.AddEdge("C", "E", 0)
	_, _ = g.AddEdge("E", "F", 0)
	_, _ = g.AddEdge("F", "G", 0)
	_, _ = g.AddEdge("D", "H", 0)
	_, _ = g.AddEdge("H", "I", 0)
	_, _ = g.AddEdge("I", "J", 0)
	_, _ = g.AddEdge("J", "K", 0)
	_, _ = g.AddEdge("K", "B", 0) // this edge closes the cycle back to B

	// Detect all simple cycles in the graph
	has, cycles, err := dfs.DetectCycles(g)
	if err != nil {
		// If an error occurred during neighbor lookup, print and exit
		fmt.Println("error:", err)
		return
	}

	// Print whether any cycle was found
	fmt.Println(has)

	// If cycles were found, print each cycle on its own line
	for _, cyc := range cycles {
		// Join the cycle’s vertices with " -> " for readability
		fmt.Println(strings.Join(cyc, " -> "))
	}

	// Output:
	// true
	// B -> D -> H -> I -> J -> K -> B
}
