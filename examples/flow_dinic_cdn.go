// Package main demonstrates modeling the maximum throughput of a simple
// Content Delivery Network (CDN) using Dinic’s max‐flow algorithm.
//
// Playground: https://go.dev/play/p/5rSh2H5F4GY
//
// Scenario:
//
//	We have a single client source “Client” connecting into two PoP (Point-of-Presence)
//	edge servers “PoP1” and “PoP2”.  Each PoP has limited upload capacity back to
//	the origin tier (“Origin1”, “Origin2”), which in turn forward to the “Sink”
//	(the Internet backbone).  We want to compute the maximum concurrent throughput.
//
//	       Client
//	       /    \
//	 cap10/      \cap15
//	     /        \
//	    PoP1     PoP2
//	    |   \  /   |
//	5min|  5 X 10  |3
//	    |   / \    |
//	 Origin1   Origin2
//	     \        /
//	 cap20\      /cap20
//	       \   /
//	 	    Sink
//
// Nodes: Client → {PoP1, PoP2} → {Origin1, Origin2} → Sink
// Capacities (in Gbps):
//
//	Client→PoP1: 10
//	Client→PoP2: 15
//	PoP1→Origin1: 5
//	PoP1→Origin2: 5
//	PoP2→Origin1: 10
//	PoP2→Origin2: 3
//	Origin1→Sink: 20
//	Origin2→Sink: 20
//
// We use Dinic to compute the maximum flow from “Client” to “Sink”.
//
// Expected max throughput = bottleneck across PoP→Origin links = 5+3=8 Gbps from PoP1 and PoP2 to Origin2 plus
// PoP2→Origin1=10 and PoP1→Origin1=5, but client uplinks are 10+15=25, and origin uplinks sum 40.
// The true max flow is limited by PoP→Origin edges: (5+5)+(10+3) = 23, but balanced across origins and sinks,
// Dinic will find the actual optimum (in this case 23 Gbps).
//
// Complexity: O(E·√V), Memory: O(V + E).
package main

//
//import (
//	"fmt"
//	"log"
//
//	"github.com/katalvlaran/lvlath/core"
//	"github.com/katalvlaran/lvlath/flow"
//)
//
//func main7() {
//	// 1) Build a directed, weighted graph for CDN capacities
//	g := core.NewGraph(true, true)
//
//	// 2) Add capacity edges (from → to, capacity Gbps)
//	edges := []struct {
//		from, to string
//		cap      int64
//	}{
//		{"Client", "PoP1", 10},
//		{"Client", "PoP2", 15},
//
//		{"PoP1", "Origin1", 5},
//		{"PoP1", "Origin2", 5},
//		{"PoP2", "Origin1", 10},
//		{"PoP2", "Origin2", 3},
//
//		{"Origin1", "Sink", 20},
//		{"Origin2", "Sink", 20},
//	}
//
//	for _, e := range edges {
//		g.AddEdge(e.from, e.to, e.cap)
//	}
//
//	// 3) Compute max-flow from "Client" → "Sink" using Dinic
//	maxFlow, residual, err := flow.Dinic(g, "Client", "Sink", nil)
//	if err != nil {
//		log.Fatalf("Dinic error: %v", err)
//	}
//
//	// 4) Display the result
//	fmt.Printf("✅ CDN maximum throughput: %.0f Gbps\n\n", maxFlow)
//
//	// 5) Show residual capacities on key edges to see bottlenecks
//	fmt.Println("🔍 Residual capacities after max-flow:")
//	for _, e := range []struct{ u, v string }{
//		{"Client", "PoP1"},
//		{"Client", "PoP2"},
//		{"PoP1", "Origin1"},
//		{"PoP1", "Origin2"},
//		{"PoP2", "Origin1"},
//		{"PoP2", "Origin2"},
//	} {
//		// If forward capacity exhausted, HasEdge will be false
//		rem := "0"
//		if residual.HasEdge(e.u, e.v) {
//			// residual edge weight = remaining capacity
//			for _, edgeList := range residual.AdjacencyList()[e.u][e.v] {
//				rem = fmt.Sprintf("%d", edgeList.Weight)
//				break
//			}
//		}
//		fmt.Printf("  %s → %s: %s Gbps remaining\n", e.u, e.v, rem)
//	}
//}
