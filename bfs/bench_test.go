package bfs_test

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/katalvlaran/lvlath/bfs"
	"github.com/katalvlaran/lvlath/core"
)

// BenchmarkBFS_Chain measures BFS on a linear chain graph of size N.
func BenchmarkBFS_Chain(b *testing.B) {
	const N = 10000
	// build a chain of N+1 vertices, N edges
	g := core.NewGraph()
	for i := 0; i < N; i++ {
		u := fmt.Sprintf("v%d", i)
		v := fmt.Sprintf("v%d", i+1)
		_, _ = g.AddEdge(u, v, 0)
	}
	V := N + 1
	E := N

	b.ReportAllocs()
	b.SetBytes(int64(V + E))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = bfs.BFS(g, "v0")
	}
}

// BenchmarkBFS_BinaryTree runs BFS on a complete binary tree of depth D (~2^D−1 nodes).
func BenchmarkBFS_BinaryTree(b *testing.B) {
	const depth = 10 // 2^10 − 1 = 1023 vertices, 1022 edges
	nodeCount := (1 << depth) - 1
	edgeCount := nodeCount - 1

	g := core.NewGraph()
	// create vertices
	for i := 1; i <= nodeCount; i++ {
		_ = g.AddVertex(fmt.Sprintf("%d", i))
	}
	// connect parent → children
	for i := 1; i <= (nodeCount-1)/2; i++ {
		p := fmt.Sprintf("%d", i)
		_, _ = g.AddEdge(p, fmt.Sprintf("%d", 2*i), 0)
		_, _ = g.AddEdge(p, fmt.Sprintf("%d", 2*i+1), 0)
	}

	b.ReportAllocs()
	b.SetBytes(int64(nodeCount + edgeCount))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = bfs.BFS(g, "1")
	}
}

// BenchmarkBFS_Grid runs BFS on an M×M grid (M² nodes, ≈2*M*(M−1) edges).
func BenchmarkBFS_Grid(b *testing.B) {
	const M = 100
	V := M * M
	// each interior cell has 2 edges (right & down), last row/col fewer
	E := 2 * M * (M - 1)

	g := core.NewGraph()
	// build vertices
	for i := 0; i < M; i++ {
		for j := 0; j < M; j++ {
			_ = g.AddVertex(fmt.Sprintf("%d_%d", i, j))
		}
	}
	// add grid edges
	for i := 0; i < M; i++ {
		for j := 0; j < M; j++ {
			id := fmt.Sprintf("%d_%d", i, j)
			if i+1 < M {
				_, _ = g.AddEdge(id, fmt.Sprintf("%d_%d", i+1, j), 0)
			}
			if j+1 < M {
				_, _ = g.AddEdge(id, fmt.Sprintf("%d_%d", i, j+1), 0)
			}
		}
	}

	b.ReportAllocs()
	b.SetBytes(int64(V + E))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = bfs.BFS(g, "0_0")
	}
}

// BenchmarkBFS_RandomSparse measures BFS on a sparse random graph.
func BenchmarkBFS_RandomSparse(b *testing.B) {
	const V = 5000
	const E = 10000

	rnd := rand.New(rand.NewSource(42))
	g := core.NewGraph()
	// add vertices
	for i := 0; i < V; i++ {
		_ = g.AddVertex(fmt.Sprintf("n%d", i))
	}
	// random edges (may include duplicates, but BFS logic ignores repeats)
	for k := 0; k < E; k++ {
		u := fmt.Sprintf("n%d", rnd.Intn(V))
		v := fmt.Sprintf("n%d", rnd.Intn(V))
		_, _ = g.AddEdge(u, v, 0)
	}

	b.ReportAllocs()
	b.SetBytes(int64(V + E))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = bfs.BFS(g, "n0")
	}
}

// BenchmarkBFS_HookOverhead compares BFS with and without an expensive OnVisit hook.
func BenchmarkBFS_HookOverhead(b *testing.B) {
	const N = 1000
	V := N + 1
	E := N

	g := core.NewGraph()
	for i := 0; i < N; i++ {
		_, _ = g.AddEdge(fmt.Sprintf("v%d", i), fmt.Sprintf("v%d", i+1), 0)
	}

	// No-op hook variant
	b.Run("NoHook", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(V + E))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = bfs.BFS(g, "v0")
		}
	})

	// CPU-intensive OnVisit hook variant
	b.Run("HeavyVisitHook", func(b *testing.B) {
		heavy := func(_ string, _ int) error {
			sum := 0
			for i := 0; i < 100; i++ {
				sum += i
			}
			_ = sum

			return nil
		}

		b.ReportAllocs()
		b.SetBytes(int64(V + E))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = bfs.BFS(g, "v0", bfs.WithOnVisit(heavy))
		}
	})
}
