package dfs_test

import (
	"fmt"
	"testing"

	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/dfs"
)

// BenchmarkDFS_Chain10000 measures the performance of DFS on a linear chain graph of 10,000 vertices.
// Graph structure: N0 → N1 → N2 → ... → N10000
// We construct the graph once per benchmark, then repeatedly call dfs.DFS on the same graph.
//
// Complexity: Building the graph is O(V) with V=10000. Each DFS traversal is O(V + E) i.e., ~O(2V) ≈ O(V).
func BenchmarkDFS_Chain10000(b *testing.B) {
	// 1. Create an empty directed graph.
	//    We specify WithDirected(true) to indicate that edges are one-way.
	g := core.NewGraph(core.WithDirected(true))

	// 2. Add vertices and edges to form a chain of length 10,001 (0 through 10,000).
	//    We iterate from i=0 to i<10000 so that the last edge is N9999 → N10000.
	for i := 0; i < 10000; i++ {
		// 2a. Define the current and next vertex IDs as "N<i>" and "N<i+1>".
		currentID := fmt.Sprintf("N%d", i)
		nextID := fmt.Sprintf("N%d", i+1)

		// 2b. Add both vertices to the graph.
		//     AddVertex returns an error if the ID is empty or invalid.
		//     Since fmt.Sprintf always produces a non-empty string, error is nil.
		_ = g.AddVertex(currentID)
		_ = g.AddVertex(nextID)

		// 2c. Add a directed edge from currentID to nextID with weight 0.
		//     In our graph, edges are unweighted (weight = 0) by default.
		_, _ = g.AddEdge(currentID, nextID, 0)
	}

	// 3. Reset the benchmark timer to exclude graph construction time.
	b.ResetTimer()

	// 4. Run DFS b.N times, starting from vertex "N0".
	//    We ignore the returned DFSResult and error for benchmarking purposes,
	//    since we assume the graph is valid and "N0" exists.
	for i := 0; i < b.N; i++ {
		_, _ = dfs.DFS(g, "N0")
	}
}
