// Package main shows how to navigate a rugged terrain grid where moving
// between adjacent cells costs energy proportional to terrain difficulty.
// Dijkstra finds the least-energy path.
//
// Playground: https://go.dev/play/p/oZ9mtrVm2_8
//
// Scenario:
//
//	We have six waypoints (P1…P6) connected by trails.  Each trail’s
//	energy cost depends on slope and surface:
//
//	   P1───1───P2───3───P3
//	    │       │
//	    4       2
//	    │       │
//	   P4───1───P5───5───P6
//
// Goal: minimize total energy from P1 → P6.
//
// Complexity: O(V·logV + E), Memory: O(V + E).
package main

//
//import (
//	"fmt"
//	"log"
//
//	"github.com/katalvlaran/lvlath/algorithms"
//	"github.com/katalvlaran/lvlath/core"
//)
//
//func main5() {
//	// 1) Build undirected, weighted terrain graph
//	g := core.NewGraph(false, true)
//	trails := []struct {
//		u, v string
//		c    int64
//	}{
//		{"P1", "P2", 1},
//		{"P2", "P3", 3},
//		{"P1", "P4", 4},
//		{"P4", "P5", 1},
//		{"P2", "P5", 2},
//		{"P5", "P6", 5},
//		{"P3", "P6", 5}, // steep uphill
//	}
//	for _, t := range trails {
//		g.AddEdge(t.u, t.v, t.c)
//	}
//
//	// 2) Compute minimal-energy paths from P1
//	dist, parent, err := algorithms.Dijkstra(g, "P1")
//	if err != nil {
//		log.Fatalf("Dijkstra error: %v", err)
//	}
//
//	// 3) Reconstruct the P1 → P6 path
//	dest := "P6"
//	path := []string{dest}
//	for cur := dest; cur != "P1"; {
//		p, ok := parent[cur]
//		if !ok {
//			log.Fatalf("No route from P1 to %s", dest)
//		}
//		path = append([]string{p}, path...)
//		cur = p
//	}
//
//	// 4) Display the navigation plan
//	fmt.Printf("Least-energy path P1 → %s:\n", dest)
//	for i := 0; i < len(path)-1; i++ {
//		u, v := path[i], path[i+1]
//		fmt.Printf("  Trail %s→%s costs %d energy\n", u, v, dist[v]-dist[u])
//	}
//	fmt.Printf("Total energy cost: %d\n", dist[dest])
//}
