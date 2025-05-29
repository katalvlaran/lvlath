// Package core_test provides benchmarks for core.Graph operations.
package core_test

import (
	"fmt"
	"testing"

	"github.com/katalvlaran/lvlath/core"
)

// BenchmarkAddEdge_Unweighted measures performance of adding edges
// in an unweighted, undirected graph (default configuration).
func BenchmarkAddEdge_Unweighted(b *testing.B) {
	// Create a new default Graph (undirected, unweighted)
	g := core.NewGraph()
	// Report memory allocations per operation
	b.ReportAllocs()
	// Reset timer to exclude setup cost
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// AddEdge uses weight=0 by default to satisfy unweighted constraint
		_, _ = g.AddEdge("Root", fmt.Sprintf("N%d", i), 0)
	}
}

// BenchmarkAddEdge_Weighted measures performance of adding edges
// in a weighted graph (non-zero weights allowed).
func BenchmarkAddEdge_Weighted(b *testing.B) {
	// Create a weighted Graph
	g := core.NewGraph(core.WithWeighted())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Using i as weight exercises the weighted path
		_, _ = g.AddEdge("Root", fmt.Sprintf("N%d", i), int64(i))
	}
}

// BenchmarkAddEdge_MultiEdges measures performance of adding parallel edges
// when multi-edges are permitted (with weighted enabled here).
func BenchmarkAddEdge_MultiEdges(b *testing.B) {
	// Create graph allowing multi-edges and weights
	g := core.NewGraph(core.WithWeighted(), core.WithMultiEdges())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Cycle through 100 target nodes to stress many parallel edges
		_, _ = g.AddEdge("Root", fmt.Sprintf("N%d", i%100), int64(i))
	}
}

// BenchmarkNeighbors measures performance of retrieving neighbors
// in a star topology on a multi-edge graph.
func BenchmarkNeighbors(b *testing.B) {
	// Create graph with multi-edge support
	g := core.NewGraph(core.WithMultiEdges())
	// Build a star with 1000 leaves: Center→Node{i}
	for i := 0; i < 1000; i++ {
		_, _ = g.AddEdge("Center", fmt.Sprintf("Node%d", i), 0)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Neighbors should return 1000 edges in O(d log d)
		_, _ = g.Neighbors("Center")
	}
}

// BenchmarkClone measures performance of cloning a graph
// containing loops and multi-edges under load.
func BenchmarkClone(b *testing.B) {
	// Create graph with loops, multi-edges, and weights
	g := core.NewGraph(core.WithWeighted(), core.WithMultiEdges(), core.WithLoops())
	// Populate with 1000 edges A→V{i}
	for i := 0; i < 1000; i++ {
		_, _ = g.AddEdge("A", fmt.Sprintf("V%d", i), int64(i))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Clone performs O(V+E) copy
		_ = g.Clone()
	}
}
