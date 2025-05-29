package matrix_test

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/matrix"
)

// BenchmarkBuildAdjacencyData measures performance of constructing an adjacency
// matrix (BuildAdjacencyData) for a graph with V vertices and E edges.
// Time complexity: O(V + E); Memory: O(V²).
func BenchmarkBuildAdjacencyData(b *testing.B) {
	const V = 1000
	const E = 5000
	// Prepare graph
	opts := matrix.NewMatrixOptions(matrix.WithWeighted(true))
	verts := make([]string, V)
	for i := 0; i < V; i++ {
		verts[i] = fmt.Sprintf("v%d", i)
	}
	g := core.NewGraph(core.WithWeighted())
	for _, v := range verts {
		_ = g.AddVertex(v)
	}
	// Add random edges
	//rand.Seed(42)
	rand.New(rand.NewSource(42))
	for i := 0; i < E; i++ {
		u := verts[rand.Intn(V)]
		v := verts[rand.Intn(V)]
		_, _ = g.AddEdge(u, v, 1)
	}
	edges := g.Edges()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = matrix.BuildAdjacencyData(verts, edges, opts)
	}
}

// BenchmarkEigenDecompose measures performance of eigen decomposition
// (EigenDecompose) on a random symmetric N×N matrix.
// Time complexity: O(N³); Memory: O(N²).
func BenchmarkEigenDecompose(b *testing.B) {
	const N = 200
	// Prepare symmetric matrix
	A := make([][]float64, N)
	//rand.Seed(123)
	rand.New(rand.NewSource(123))
	for i := 0; i < N; i++ {
		A[i] = make([]float64, N)
	}
	for i := 0; i < N; i++ {
		for j := i; j < N; j++ {
			v := rand.Float64()
			A[i][j] = v
			A[j][i] = v
		}
	}

	tol := 1e-6
	maxIter := 1000

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = matrix.EigenDecompose(A, tol, maxIter)
	}
}
