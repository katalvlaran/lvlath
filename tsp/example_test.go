// Package main demonstrates a real-world logistics scenario using lvlath/core and lvlath/matrix
// to build a weighted graph of 10 locations, convert it to a distance matrix, and then solve
// the TSP with lvlath/tsp. We use TSPApprox (Christofides) to plan a near-optimal delivery route.
//
// Scenario:
//
//	A delivery company must dispatch a single vehicle from the “Hub” warehouse to  nine retail
//	outlets and return. We model the road network as an undirected, weighted graph where vertices
//	are locations and edges are the driving distances in kilometers. Converting to an adjacency
//	matrix and running TSPApprox (O(n³)) yields a practical route in milliseconds.
//
// Use case:
//
//	Daily route planning for last-mile deliveries across urban and suburban locations.
//
// Playground: [![Go Playground – TSP Logistics](https://img.shields.io/badge/Go_Playground-TSP_Logistics-blue?logo=go)](https://play.golang.org/p/your-snippet-id)
package tsp_test

import (
	"fmt"
	"log"

	"github.com/katalvlaran/lvlath/core"   // core graph types
	"github.com/katalvlaran/lvlath/matrix" // adjacency‐matrix builder
	"github.com/katalvlaran/lvlath/tsp"    // TSP solvers
)

const (
	Hub        = "Hub"
	NorthMall  = "NorthMall"
	EastPlaza  = "EastPlaza"
	SouthPark  = "SouthPark"
	WestSide   = "WestSide"
	Uptown     = "Uptown"
	Downtown   = "Downtown"
	Airport    = "Airport"
	University = "University"
	Stadium    = "Stadium"
)

func ExampleTSP() {
	// 1) Build the weighted road network graph (undirected, weighted distances in km)
	g := core.NewGraph(core.WithWeighted())
	locations := []string{
		Hub, NorthMall, EastPlaza, SouthPark, WestSide,
		Uptown, Downtown, Airport, University, Stadium,
	}
	for _, loc := range locations {
		if err := g.AddVertex(loc); err != nil {
			log.Fatalf("add vertex %s: %v", loc, err)
		}
	}
	// Add pairwise roads (symmetric distances)
	roads := []struct {
		u, v string
		d    int64
	}{
		{Hub, NorthMall, 12}, {Hub, EastPlaza, 18}, {Hub, SouthPark, 20}, {Hub, WestSide, 15},
		{NorthMall, EastPlaza, 7}, {EastPlaza, SouthPark, 10}, {SouthPark, WestSide, 8}, {WestSide, NorthMall, 9},
		{NorthMall, Uptown, 6}, {Uptown, Downtown, 5}, {Downtown, EastPlaza, 11},
		{SouthPark, Airport, 14}, {Airport, University, 13}, {University, Stadium, 9}, {Stadium, Downtown, 12},
	}
	for _, r := range roads {
		if _, err := g.AddEdge(r.u, r.v, r.d); err != nil {
			log.Fatalf("add edge %s-%s: %v", r.u, r.v, err)
		}
	}

	// 2) Convert graph to adjacency matrix
	optsMat := matrix.NewMatrixOptions(matrix.WithWeighted(true))
	am, err := matrix.NewAdjacencyMatrix(g, optsMat)
	if err != nil {
		log.Fatalf("matrix conversion: %v", err)
	}
	// 'am.Index' maps location name → matrix index
	// 'am.Data' is the [][]float64 distance matrix

	// 3) Solve TSP via 1.5-approximation (Christofides)
	tspOpts := tsp.DefaultOptions()
	//res, err := tsp.TSPApprox(am.Data, tspOpts)
	res, err := tsp.TSPApprox(am.Data, tspOpts)
	if err != nil {
		log.Fatalf("TSPApprox failed: %v", err)
	}

	// 4) Print route without extra indentation
	fmt.Println("Planned delivery route:")
	for i, idx := range res.Tour {
		fmt.Printf("%d: %s\n", i, locations[idx])
	}
	fmt.Printf("\nTotal distance: %.0f km\n", res.Cost)
	// Output:
	// Planned delivery route:
	// 0: Hub
	// 1: SouthPark
	// 2: Airport
	// 3: NorthMall
	// 4: Uptown
	// 5: WestSide
	// 6: EastPlaza
	// 7: Downtown
	// 8: University
	// 9: Stadium
	// 10: Hub
	//
	// Total distance: 7 km
}
