// Package matrix_test provides benchmarks for core matrix package operations,
// using builder for graph generation and random fill for Dense matrices.
package matrix_test

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/katalvlaran/lvlath/builder"
	"github.com/katalvlaran/lvlath/matrix"
)

// benchSizes are the matrix sizes to benchmark.
var benchSizes = []int{50, 100, 200}

func BenchmarkBuildDenseAdjacency(b *testing.B) {
	// Report memory allocations per operation.
	b.ReportAllocs()
	for _, V := range benchSizes {
		V := V // capture for parallel execution
		b.Run(fmt.Sprintf("V=%d", V), func(b *testing.B) {
			// Stage 2 (Prepare): build a complete graph of V vertices
			g, err := builder.BuildGraph(nil, builder.Complete(V))
			if err != nil {
				b.Fatalf("failed to build graph: %v", err)
			}
			verts := g.Vertices() // vertex ID slice
			edges := g.Edges()    // edge slice
			opts := matrix.NewMatrixOptions(
				matrix.WithWeighted(true), // include weights
			)

			b.ResetTimer()
			// Stage 3 (Execute): build adjacency matrix repeatedly
			for i := 0; i < b.N; i++ {
				_, _, _ = matrix.BuildDenseAdjacency(verts, edges, opts)
			}
		})
	}
}

func BenchmarkBuildDenseAdjacencyWithClosure(b *testing.B) {
	b.ReportAllocs()
	for _, V := range benchSizes {
		V := V
		b.Run(fmt.Sprintf("V=%d+closure", V), func(b *testing.B) {
			// Stage 2 (Prepare): build graph and base matrix
			g, err := builder.BuildGraph(nil, builder.Complete(V))
			if err != nil {
				b.Fatalf("failed to build graph: %v", err)
			}
			verts := g.Vertices()
			edges := g.Edges()
			opts := matrix.NewMatrixOptions(
				matrix.WithWeighted(true),
				matrix.WithMetricClosure(true), // enable APSP closure
			)

			b.ResetTimer()
			// Stage 3 (Execute): build with metric closure
			for i := 0; i < b.N; i++ {
				_, _, _ = matrix.BuildDenseAdjacency(verts, edges, opts)
			}
		})
	}
}

func BenchmarkMulDense(b *testing.B) {
	b.ReportAllocs()
	for _, N := range benchSizes {
		N := N
		b.Run(fmt.Sprintf("Mul %dx%d", N, N), func(b *testing.B) {
			// Stage 2 (Prepare): create two N×N random Dense matrices
			a, _ := matrix.NewDense(N, N)
			c := rand.New(rand.NewSource(42))
			for i := 0; i < N; i++ {
				for j := 0; j < N; j++ {
					_ = a.Set(i, j, c.Float64())
				}
			}
			bm, _ := matrix.NewDense(N, N)
			for i := 0; i < N; i++ {
				for j := 0; j < N; j++ {
					_ = bm.Set(i, j, c.Float64())
				}
			}

			b.ResetTimer()
			// Stage 3 (Execute): multiply matrices
			for i := 0; i < b.N; i++ {
				_, _ = matrix.Mul(a, bm)
			}
		})
	}
}
