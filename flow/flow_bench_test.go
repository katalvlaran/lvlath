package flow_test

import (
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/flow"
)

func BenchmarkDinicLarge(b *testing.B) {
	// Build a random directed, weighted graph with V=1000, ~E=10000
	const V = 1000
	rand.Seed(time.Now().UnixNano())

	// 1) Create vertices
	g := core.NewGraph(true, true)
	for i := 0; i < V; i++ {
		g.AddVertex(&core.Vertex{ID: strconv.Itoa(i)})
	}

	// 2) Add random edges with p≈0.02 → ~10000 edges
	for u := 0; u < V; u++ {
		for v := u + 1; v < V; v++ {
			if rand.Float64() < 0.02 {
				w := int64(rand.Intn(10) + 1)
				g.AddEdge(strconv.Itoa(u), strconv.Itoa(v), w)
			}
		}
	}

	src, dst := "0", strconv.Itoa(V-1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = flow.Dinic(g, src, dst, nil)
	}
}
