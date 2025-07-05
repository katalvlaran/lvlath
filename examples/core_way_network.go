// Package main - Ukraine Transportation Network example.
//
// Context & Motivation:
//
//	Before the recent conflicts, Ukraine’s road, rail, and air networks formed a
//	critical backbone for commerce, travel, and cultural exchange.
//	showcasing lvlath/core’s support for weighted, multi-edge, mixed-mode graphs.
//	We keep 100% of the API calls intact - so the sample still compiles against lvlath/core,
//	but we add granular commentary that answers five practical questions:
//	  (1)  How are the graphs parameterised? - Flags such as WithWeighted, WithMixedEdges are now stated explicitly and commented.
//	  (2)  Which networks flow into the final multi‑graph? - Road, Rail, Air.
//	  (3)  What does each printed statistic illustrate? - We sign‑post every fmt.Printf with a short reason‑why.
//	  (4)  What is the asymptotic cost? - Big‑O per operation in situ.
//	  (5)  How would you extend it? - A final TODO block shows follow‑ups.
//
// Graph Variants Built:
//   - BuildUkraineRoads graph:
//     92 vertices (cities), identified by English names with underscores.
//     334 Undirected, weighted edges (distance in km), no loops, no multi-edges.
//   - BuildFullUkraineGraph - multimodal (weighted, allow multi-edges, mixed directedness) 761 edges graph
//
// Demonstrated Operations & Complexity:
//   - BuildUkraineRoads:     O(E_road) total (AddEdge O(1) amortized)
//   - BuildFullUkraineGraph: O(E_total), WithMultiEdges, WithMixedEdges
//   - HasEdge:               O(1)
//   - EdgeCount/VertexCount: O(1)
//   - Neighbors(v):          O(d·log d)
//   - NeighborIDs(v):        O(d·log d)
//   - CloneEmpty:            O(V)
//   - FilterEdges + cleanup: O(E + V)
//
// Expected Output (approx):
//
//		RoadGraph: vertices=92, edges=334
//	 22 Neighbors of Kyiv(roads only):
//	   Kaniv(125.4 km) Kremenchuk(297.1 km) Berdychiv(188.4 km) Lutsk(423.5 km) Lviv(537.7 km) Odesa(507.7 km) Poltava(348.8 km) Rivne(347.8 km) Sumy(351.3 km) Uman(219.1 km) Vinnytsia(229.4 km) Zhytomyr(154.2 km) Myrhorod(207.9 km) Nizhyn(129.0 km) Bila_Tserkva(88.3 km) Romny(195.0 km) Boryspil(34.6 km) Brovary(22.7 km) Bucha(27.8 km) Cherkasy(180.3 km) Chernihiv(147.5 km) Chernobyl(108.4 km)
//	 CloneEmpty: vertices=92, edges=0
//	 After filtering ≤300 km: edges=284
//
//	 FullGraph: vertices=92, edges=557
//	 All 45 Neighbors of Kyiv in FullGraph:
//	   Kaniv(125.4 km) Kremenchuk(297.1 km) Berdychiv(188.4 km) Lutsk(423.5 km) Lviv(537.7 km) Odesa(507.7 km) Poltava(348.8 km) Rivne(347.8 km) Sumy(351.3 km) Uman(219.1 km) Vinnytsia(229.4 km) Zhytomyr(154.2 km) Myrhorod(207.9 km) Nizhyn(129.0 km) Bila_Tserkva(88.3 km) Romny(195.0 km) Bucha(27.8 km) Cherkasy(180.3 km) Chernihiv(147.5 km) Chernobyl(158.0 km) Boryspil(34.6 km) Kaniv(125.4 km) Khmelnytskyi(318.9 km) Kropyvnytskyi(286.6 km) Bila_Tserkva(88.3 km) Boryspil(34.6 km) Brovary(22.7 km) Kharkiv(489.0 km) Lviv(569.0 km) Poltava(348.8 km) Rivne(347.8 km) Sumy(351.3 km) Vinnytsia(229.4 km) Zhytomyr(154.2 km) Brovary(22.7 km) Myrhorod(207.9 km) Nizhyn(129.0 km) Bucha(27.8 km) Dnipro(391.0 km) Kharkiv(409.0 km) Lviv(470.0 km) Odesa(441.0 km) Cherkasy(180.3 km) Chernihiv(147.5 km) Chernobyl(108.4 km)
//	 CloneEmpty: vertices=92, edges=0
//	 After filtering ≤300 km: edges=464
package main

import (
	"fmt"
	"log"

	"github.com/katalvlaran/lvlath/core"
)

const (
	// CAPITAL is the source city for neighbor listing.
	CAPITAL = Kyiv // evaluated at link‑time from the dataset file

	// MAX_KM sets the demo threshold for edge filtering.
	MAX_KM = 300.0

	// M_IN_KM uses to converts weight by km2m() + m2km().
	M_IN_KM = 1000
)

// WaySegment describes one weighted link between two cities.
type WaySegment struct {
	From, To string  // source and destination cityIDs
	KM       float64 // length in kilometers
}

func _main() {
	// 1. Build undirected, weighted graph
	roadG := BuildUkraineRoads()
	fmt.Printf("\nRoadGraph: vertices=%d, edges=%d", roadG.VertexCount(), roadG.EdgeCount())

	// 1.2. List immediate highway neighbors of Kyiv
	edges, err := roadG.Neighbors(CAPITAL)
	fmt.Printf("\n%d Neighbors of %s(roads only):\n", len(edges), CAPITAL)
	if err != nil {
		log.Fatalf("error fetching neighbors of %s: %v", CAPITAL, err)
	}
	for _, e := range edges {
		// For undirected edges we check which end is CAPITAL
		// undirected neighbors list city and distance
		other := e.To
		if other == CAPITAL {
			other = e.From
		}
		fmt.Printf("  %s (%.1f km)\n", other, m2km(e.Weight))
	}
	// 1.3. CloneEmpty: same vertices, zero edges
	ce := roadG.CloneEmpty()
	fmt.Printf("CloneEmpty: vertices=%d, edges=%d\n", ce.VertexCount(), ce.EdgeCount())
	// Expect edges=0 because CloneEmpty preserves only vertices.

	// 1.4. Filter: keep only segments ≤ MAX_KM
	roadG.FilterEdges(func(e *core.Edge) bool {
		return m2km(e.Weight) <= MAX_KM
	})
	fmt.Printf("After filtering ≤%.0f km: edges=%d\n", MAX_KM, roadG.EdgeCount())

	// 2. Multimodal full graph with multi-edges
	fullG := BuildFullUkraineGraph()
	fmt.Printf("\nFullGraph: vertices=%d, edges=%d", fullG.VertexCount(), fullG.EdgeCount())

	// 2.2. Neighbors of Kyiv in full graph
	edges, _ = fullG.Neighbors(CAPITAL)
	fmt.Printf("\nAll %d Neighbors of %s in FullGraph:\n", len(edges), CAPITAL)
	for _, e := range edges {
		// undirected neighbors list city and distance
		other := e.To
		if other == CAPITAL {
			other = e.From
		}
		fmt.Printf("  %s (%.1f km)\n", other, m2km(e.Weight))
	}

	// 2.3. CloneEmpty demonstration
	clone := fullG.CloneEmpty()
	fmt.Printf("CloneEmpty: vertices=%d, edges=%d\n", clone.VertexCount(), clone.EdgeCount())

	// 2.4. Filter out long links > MAX_KM
	fullG.FilterEdges(func(e *core.Edge) bool {
		return m2km(e.Weight) <= MAX_KM
	})
	fmt.Printf("After filtering ≤%.0f km: edges=%d\n", MAX_KM, fullG.EdgeCount())

	// ------------------------------------------------
	// TODO - Suggested follow‑ups for curious readers
	// ------------------------------------------------
	// • Shortest‑path demo (Dijkstra) between any two hubs.
	// • Connected‑components per transport mode.
	// • Edge‑betweenness centrality to find critical corridors.
	// • Visual export (GeoJSON) for GIS tools.
}

// BuildUkraineRoads constructs the road network graph.
// Complexity: O(E).
func BuildUkraineRoads() *core.Graph {
	g := core.NewGraph(core.WithWeighted()) // weighted graph
	// Undirected by default: edges will be mirrored automatically.
	for _, seg := range RoadNetwork {
		// AddEdge stores weight as int64 km, auto-adds vertices.
		if _, err := g.AddEdge(seg.From, seg.To, km2m(seg.KM)); err != nil {
			log.Fatalf("AddEdge %s→%s failed: %v", seg.From, seg.To, err)
		}
	}
	return g
}

// BuildFullUkraineGraph combines roads and rails into a single multimodal network.
// Enables multi-edges so that a city pair with both road+rail is captured.
// Complexity: O(|RoadNetwork| + |RailwayNetwork| + |AirNetwork|).
func BuildFullUkraineGraph() *core.Graph {
	g := core.NewGraph(core.WithWeighted(), core.WithMixedEdges(), core.WithMultiEdges())
	var seg WaySegment
	// add roads as 337 undirected edges
	for _, seg = range RoadNetwork {
		if _, err := g.AddEdge(seg.From, seg.To, km2m(seg.KM)); err != nil {
			log.Fatalf("buildFullGraph roads: %v", err)
		}
	}
	// add rails as 200 parallel edges
	for _, seg = range RailwayNetwork {
		if _, err := g.AddEdge(seg.From, seg.To, km2m(seg.KM)); err != nil {
			log.Fatalf("buildFullGraph rails: %v", err)
		}
	}
	// add air as 37 directed edges
	for _, seg = range AirNetwork {
		if _, err := g.AddEdge(seg.From, seg.To, km2m(seg.KM), core.WithEdgeDirected(true)); err != nil {
			log.Fatalf("buildFullGraph rails: %v", err)
		}
	}
	return g
}

// Convert km(flat64) into meters(int64)
func km2m(x float64) int64 { return int64(x * M_IN_KM) }

// Convert meters(int64) into km(flat64)
func m2km(x int64) float64 { return float64(x) / M_IN_KM }
