// Package main demonstrates the difference between Clone and CloneEmpty.
// Playground: https://go.dev/play/p/INSURpHaNjX
package main

import (
	"fmt"

	"github.com/katalvlaran/lvlath/core"
)

func main2() {
	// ───────────────────────────────────────────────
	// Build a tiny graph: A→B→C
	// ───────────────────────────────────────────────
	g := core.NewGraph(false, false) // undirected, unweighted
	g.AddEdge("A", "B", 0)
	g.AddEdge("B", "C", 0)

	// ───────────────────────────────────────────────
	// Clone: deep copy of both vertices & edges
	// Empty: copy of vertices only, no edges
	// ───────────────────────────────────────────────
	fullClone := g.Clone()
	emptyClone := g.CloneEmpty()

	fmt.Printf("Original edges:     %v\n", listEdges(g))
	fmt.Printf("Full clone edges:   %v\n", listEdges(fullClone))
	fmt.Printf("Empty clone edges:  %v\n", listEdges(emptyClone))

	// ───────────────────────────────────────────────
	// Now mutate: remove A→B from original
	// ───────────────────────────────────────────────
	g.RemoveEdge("A", "B")
	fmt.Println("\nAfter removing A→B from original:")
	fmt.Printf("Original edges:     %v\n", listEdges(g))
	fmt.Printf("Full clone edges:   %v\n", listEdges(fullClone))

	// ───────────────────────────────────────────────
	// And mutate empty clone separately: add A→B
	// ───────────────────────────────────────────────
	emptyClone.AddEdge("A", "B", 0)
	fmt.Println("\nAfter adding A→B to empty clone:")
	fmt.Printf("Empty clone edges:  %v\n", listEdges(emptyClone))
}

// listEdges returns a sorted slice of unique edges (undirected: U-V).
func listEdges(g *core.Graph) []string {
	seen := map[string]bool{}
	var out []string

	for _, e := range g.Edges() {
		u, v := e.From.ID, e.To.ID
		// normalize for undirected: always "smaller-larger"
		if u > v {
			u, v = v, u
		}
		key := fmt.Sprintf("%s-%s", u, v)
		if !seen[key] {
			seen[key] = true
			out = append(out, key)
		}
	}
	return out
}
