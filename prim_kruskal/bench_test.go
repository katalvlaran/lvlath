package prim_kruskal_test

import (
	"testing"

	"github.com/katalvlaran/lvlath/prim_kruskal"
)

// BenchmarkKruskal measures performance on a random dense graph with 500 vertices and 2000 edges.
func BenchmarkKruskal(b *testing.B) {
	g := buildMediumGraph(500, 2000) // pre‐build graph once
	b.ResetTimer()                   // reset timer to exclude graph construction
	for i := 0; i < b.N; i++ {
		_, _, _ = prim_kruskal.Kruskal(g)
	}
}

// BenchmarkPrim measures performance on a random dense graph with 500 vertices and 2000 edges,
// always starting Prim from vertex "V0".
func BenchmarkPrim(b *testing.B) {
	g := buildMediumGraph(500, 2000) // pre‐build graph once
	b.ResetTimer()                   // reset timer to exclude graph construction
	for i := 0; i < b.N; i++ {
		_, _, _ = prim_kruskal.Prim(g, "V0")
	}
}
