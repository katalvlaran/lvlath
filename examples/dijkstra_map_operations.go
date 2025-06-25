// Package main demonstrates running Dijkstra’s shortest-path algorithm on the full Ukraine transportation network.
// It builds a multimodal pre-war Ukraine graph (roads, rail, air),
// runs Dijkstra between arbitrary hubs, and illustrates filtering and path reconstruction.
//
// Context & Motivation:
//   (1) How are graphs parameterized? We use core.NewGraph with Weighted, MixedEdges, MultiEdges flags.
//   (2) Which networks flow into the final graph? Road, Rail, and Air datasets are combined by BuildFullUkraineGraph().
//   (3) What does each printed statistic illustrate? We show graph size, neighborhood listings, and shortest-path results.
//   (4) What is the asymptotic cost? Building is O(E), Dijkstra is O((V+E) log V).
//   (5) How to extend it? TODO comments indicate next steps (e.g., route queries between arbitrary hubs).
//
// Demonstrates:
//   • Building the multimodal network: O(|Road|+|Rail|+|Air|).
//   • Running Dijkstra from Kyiv: O((V+E) log V).
//   • Extracting distances and reconstructing a path.
//   • Filtering edges and re-running to show MaxDistance.

package main

import (
	"fmt"
	"log"
	"math"

	"github.com/katalvlaran/lvlath/dijkstra"
)

const Distance = 900_000

// main executes the full pipeline: build, analyze, run Dijkstra, and display results.
func main() {
	// 1) Build the roadways Ukraine graph.
	fullG := BuildUkraineRoads()
	vCount, eCount := fullG.VertexCount(), fullG.EdgeCount()
	fmt.Printf("FullGraph built: V=%d, E=%d\n", vCount, eCount)

	// 2) List immediate neighbors of Kyiv to illustrate network density.
	neighbors, err := fullG.Neighbors(CAPITAL)
	if err != nil {
		log.Fatalf("Neighbors query failed: %v", err)
	}
	fmt.Printf("%d total neighbors of %s (all modes):\n", len(neighbors), CAPITAL)
	for i, edge := range neighbors {
		// Determine the other endpoint for undirected edges
		other := edge.To
		if other == CAPITAL {
			other = edge.From
		}
		fmt.Printf("  %2d: %s (%.1f km)\n", i+1, other, m2km(edge.Weight))
	}

	// 3) Run Dijkstra from Kyiv, requesting full predecessor map.
	dist, prev, err := dijkstra.Dijkstra(
		fullG,
		dijkstra.Source(CAPITAL),
		dijkstra.WithReturnPath(),
		// Optionally bound search to 500 km for demonstration:
		dijkstra.WithMaxDistance(Distance), // distances stored in meters
	)
	if err != nil {
		log.Fatalf("Dijkstra failed: %v", err)
	}

	// 4) Print distances to a selection of major hubs.
	hubs := []string{Lviv, Odesa, Kharkiv, Dnipro, Simferopol}
	fmt.Printf("\nShortest distances from %s:\n", CAPITAL)
	for _, city := range hubs {
		d := dist[city]
		if d == math.MaxInt64 {
			fmt.Printf("  %-12s unreachable\n", city)
		} else {
			fmt.Printf("  %-12s %6.1f km\n", city, m2km(d))
		}
	}

	// 5) Reconstruct a sample path: Kyiv → Lviv
	target := Sevastopol
	path := []string{target}
	for u := prev[target]; u != ""; u = prev[u] {
		path = append([]string{u}, path...)
	}
	fmt.Printf("\nSample path %s→%s: %v\n", path, CAPITAL, target)
}
