// Package main demonstrates using BFS to find the shortest-hop path
// between two routers in a network map.
// Playground: https://go.dev/play/p/-UExg4rKqoL
//
// Scenario:
//
//	┌───┐     ┌───┐     ┌───┐
//	│ R1├─────┤ R2├─────┤ R3│
//	└─┬─┘     └─┬─┘     └─┬─┘
//	  │         │         │
//	┌─┴─┐     ┌─┴─┐     ┌─┴─┐
//	│ R4├─────┤ R5├─────┤ R6│
//	└───┘     └───┘     └───┘
//
// We want the fewest hops from R1 → R6.
package main

import (
	"fmt"
	"log"

	"github.com/katalvlaran/lvlath/algorithms"
	"github.com/katalvlaran/lvlath/core"
)

func main1() {
	// 1) Build network graph: routers R1…R6
	g := core.NewGraph(false, false) // undirected, unweighted
	links := [][2]string{
		{"R1", "R2"}, {"R2", "R3"},
		{"R1", "R4"}, {"R4", "R5"},
		{"R2", "R5"}, {"R5", "R6"},
	}
	for _, e := range links {
		g.AddEdge(e[0], e[1], 0)
	}

	// 2) Run BFS from source R1
	source, target := "R1", "R6"
	res, err := algorithms.BFS(g, source, nil)
	if err != nil {
		log.Fatalf("BFS failed: %v", err)
	}

	// 3) Reconstruct shortest path via Parent map
	path := []string{target}
	for cur := target; cur != source; {
		p, ok := res.Parent[cur]
		if !ok {
			log.Fatalf("No path from %s to %s", source, target)
		}
		path = append([]string{p}, path...)
		cur = p
	}

	// 4) Print results
	fmt.Printf("Shortest path from %s to %s (%d hops):\n", source, target, len(path)-1)
	for i, r := range path {
		fmt.Printf("  %2d: %s\n", i, r)
	}
}
