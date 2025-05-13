// Package main demonstrates migrating a large graph using Clone.
// Playground: https://go.dev/play/p/pL_cToMVypD
package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/katalvlaran/lvlath/core"
)

func main3() {
	// ───────────────────────────────────────────────
	// 1) Generate original graph: 5000 vertices, 12000 random edges
	// ───────────────────────────────────────────────
	const (
		V = 5000  // number of servers (vertices)
		E = 12000 // number of links (edges)
	)
	rand.Seed(time.Now().UnixNano())

	original := core.NewGraph(false, false) // undirected, unweighted
	// add vertices
	for i := 0; i < V; i++ {
		original.AddVertex(&core.Vertex{ID: fmt.Sprintf("%d", i)})
	}
	// add random edges
	for i := 0; i < E; i++ {
		u := rand.Intn(V)
		v := rand.Intn(V)
		if u == v {
			v = (u + 1) % V
		}
		original.AddEdge(fmt.Sprintf("%d", u), fmt.Sprintf("%d", v), 0)
	}

	fmt.Printf("Original graph: V=%d, E=%d\n",
		len(original.Vertices()), len(original.Edges())/2) // /2 because undirected edges are mirrored

	// ───────────────────────────────────────────────
	// 2) Clone the original graph (deep copy)
	// ───────────────────────────────────────────────
	migrated := original.Clone()
	fmt.Printf("Cloned graph:   V=%d, E=%d\n",
		len(migrated.Vertices()), len(migrated.Edges())/2)

	// ───────────────────────────────────────────────
	// 3) Modify the clone: add 100 new servers + connections
	// ───────────────────────────────────────────────
	for i := V; i < V+100; i++ {
		id := fmt.Sprintf("%d", i)
		migrated.AddVertex(&core.Vertex{ID: id})
		// connect each new server to 3 random existing servers
		for j := 0; j < 3; j++ {
			target := rand.Intn(V)
			migrated.AddEdge(id, fmt.Sprintf("%d", target), 0)
		}
	}

	// ───────────────────────────────────────────────
	// 4) Remove 500 random links from the clone
	// ───────────────────────────────────────────────
	removed := 0
	for _, e := range migrated.Edges() {
		if removed >= 500 {
			break
		}
		migrated.RemoveEdge(e.From.ID, e.To.ID)
		removed++
	}

	fmt.Printf("After migration:V=%d, E=%d\n",
		len(migrated.Vertices()), len(migrated.Edges())/2)
}
