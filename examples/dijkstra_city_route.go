// Package main demonstrates computing the fastest driving route
// between two city intersections using Dijkstra’s algorithm, taking
// into account varying travel times and a “closed” road represented
// by a very high weight.
//
// Playground: https://go.dev/play/p/Br0lZWD35sP
//
// Scenario:
//
//	We model six intersections (A–F).  Each road segment has a travel
//	time (minutes).  One road is effectively closed (C→D = ∞).
//
//	      [A]
//	     /   \
//	  4 /     \ 2
//	   /       \
//	 [B]---1---[C]    <-- C→D is closed (∞)
//	  | \        \10
//	5 |  \5      [E]
//	  |   \        \
//	 [D]   --------[F]
//	     6         3
//
// Goal: find the minimal-time path from A → F.
//
// Complexity: O(V·logV + E) average with heap, Memory: O(V + E).
package main

import (
	"fmt"
	"log"
	"math"

	"github.com/katalvlaran/lvlath/algorithms"
	"github.com/katalvlaran/lvlath/core"
)

func main4() {
	// 1) Construct an undirected, weighted city graph.
	g := core.NewGraph(false, true)

	// 2) Add road segments (u, v, time)
	roads := []struct {
		u, v string
		t    int64
	}{
		{"A", "B", 4},
		{"A", "C", 2},
		{"B", "C", 1},
		{"B", "D", 5},
		// C→D is closed: model as very high travel time
		{"C", "D", math.MaxInt32},
		{"C", "E", 10},
		{"D", "F", 6},
		{"E", "F", 3},
	}
	for _, r := range roads {
		g.AddEdge(r.u, r.v, r.t)
	}

	// 3) Run Dijkstra from source “A”
	dist, parent, err := algorithms.Dijkstra(g, "A")
	if err != nil {
		log.Fatalf("Dijkstra failed: %v", err)
	}

	// 4) Reconstruct path A → F
	dest := "F"
	path := []string{dest}
	for cur := dest; cur != "A"; {
		p, ok := parent[cur]
		if !ok {
			log.Fatalf("No path from A to %s", dest)
		}
		path = append([]string{p}, path...)
		cur = p
	}

	// 5) Print results
	fmt.Printf("Fastest route from A to %s:\n", dest)
	for i := 0; i < len(path)-1; i++ {
		u, v := path[i], path[i+1]
		fmt.Printf("  %s → %s : %d min\n", u, v, dist[v]-dist[u])
	}
	fmt.Printf("Total travel time: %d minutes\n", dist[dest])
}
