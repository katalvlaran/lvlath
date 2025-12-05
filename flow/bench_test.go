package flow_test

import (
	"context"
	"math/rand"
	"strconv"
	"testing"

	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/flow"
)

// buildRandomGraph constructs a directed, weighted graph with V vertices and
// roughly p probability of an edge between any ordered pair u→v.
// Edge weights are uniform in [1, maxWeight].
func buildRandomGraph(V int, p float64, maxWeight float64, seed int64) *core.Graph {
	r := rand.New(rand.NewSource(seed)) // deterministic seed for reproducibility
	g := core.NewGraph(core.WithDirected(true), core.WithWeighted())
	// Add all vertices
	for i := 0; i < V; i++ {
		_ = g.AddVertex(strconv.Itoa(i))
	}
	// Add edges with probability p
	for u := 0; u < V; u++ {
		for v := 0; v < V; v++ {
			if u == v {
				continue // skip self-loops
			}
			if r.Float64() < p {
				w := r.Float64()*maxWeight + 1.0
				_, _ = g.AddEdge(strconv.Itoa(u), strconv.Itoa(v), w)
			}
		}
	}
	return g
}

// BenchmarkFlowAlgorithms measures the performance of Ford–Fulkerson,
// Edmonds–Karp, and Dinic on graphs of increasing size and density.
// We benchmark each algorithm separately as sub-benchmarks.
func BenchmarkFlowAlgorithms(b *testing.B) {
	// Define benchmark cases with varying graph sizes and edge probabilities.
	cases := []struct {
		name      string
		vertices  int
		edgeProb  float64
		maxWeight float64
		seed      int64
	}{
		{"Small", 200, 0.05, 10.0, 42},
		{"Medium", 500, 0.02, 20.0, 4242},
		{"Large", 1000, 0.01, 50.0, 424242},
	}

	for _, tc := range cases {
		// Capture range variable
		tc := tc
		b.Run(tc.name, func(b *testing.B) {
			// Build the test graph once per case to isolate algorithmic cost.
			g := buildRandomGraph(tc.vertices, tc.edgeProb, tc.maxWeight, tc.seed)
			src := "0"
			dst := strconv.Itoa(tc.vertices - 1)

			// Use default options with background context and no verbose logging.
			opts := flow.DefaultOptions()
			opts.Ctx = context.Background()

			// Sub-benchmark for Ford–Fulkerson (O(E*F), not suitable for large F).
			b.Run("FordFulkerson", func(b *testing.B) {
				// We reset the timer after graph construction.
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, _, _ = flow.FordFulkerson(g, src, dst, opts)
				}
			})

			// Sub-benchmark for Edmonds–Karp (O(V*E²) worst-case).
			b.Run("EdmondsKarp", func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, _, _ = flow.EdmondsKarp(g, src, dst, opts)
				}
			})

			// Sub-benchmark for Dinic (O(E*√V) on unit networks, high practical performance).
			b.Run("Dinic", func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, _, _ = flow.Dinic(g, src, dst, opts)
				}
			})
		})
	}
}
