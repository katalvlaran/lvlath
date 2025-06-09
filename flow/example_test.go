package flow_test

import (
	"context"
	"fmt"

	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/flow"
)

////////////////////////////////////////////////////////////////////////////////
// Complex network example (7 vertices, 9 edges):
//
//    S→A (5)        A→B (8)
//    S→C (15)       B→D (10)
//    C→D (5)        C→E (10)
//    E→D (10)       D→T (10)
//    E→T (5)
//
// Expected max‐flow: 15 (see path breakdown in comments).
////////////////////////////////////////////////////////////////////////////////

// ExampleFordFulkerson_complex demonstrates Ford–Fulkerson on the complex network.
// It constructs the graph, runs Ford–Fulkerson, and prints the max flow.
// Playground: [![Playground - FordFulkerson](https://img.shields.io/badge/Go_Playground-Prim-blue?logo=go)](https://go.dev/play/p/1EPGX8HQ4qC)
func ExampleFordFulkerson_complex() {
	// 1. Create a directed, weighted graph.
	g := core.NewGraph(core.WithDirected(true), core.WithWeighted())

	// 2. Add all edges with their capacities.
	g.AddEdge("S", "A", 5)  // from source to A
	g.AddEdge("S", "C", 15) // from source to C
	g.AddEdge("A", "B", 8)  // A to B
	g.AddEdge("B", "D", 10) // B to D
	g.AddEdge("C", "D", 5)  // C to D
	g.AddEdge("C", "E", 10) // C to E
	g.AddEdge("E", "D", 10) // E to D
	g.AddEdge("D", "T", 10) // D to sink
	g.AddEdge("E", "T", 5)  // E to sink

	// 3. Configure options: use default Epsilon, non-verbose, background context.
	opts := flow.DefaultOptions()
	opts.Ctx = context.Background()

	// 4. Run Ford–Fulkerson to compute max flow from "S" to "T".
	maxFlow, _, err := flow.FordFulkerson(g, "S", "T", opts)
	if err != nil {
		panic(err) // should not happen in this example
	}

	// 5. Print the resulting max flow.
	fmt.Println(maxFlow)
	// Output:
	// 15
}

// //////////////////////////////////////////////////////////////////////////////
// ExampleEdmondsKarp_complex demonstrates Edmonds–Karp on the same network.
// It uses BFS to find shortest augmenting paths, guaranteeing O(V·E²) worst‐case.
// //////////////////////////////////////////////////////////////////////////////
// Playground: [![Playground - EdmondsKarp](https://img.shields.io/badge/Go_Playground-Prim-blue?logo=go)](https://go.dev/play/p/hgGauPZZOcV)
func ExampleEdmondsKarp_complex() {
	// Build the identical graph as above.
	g := core.NewGraph(core.WithDirected(true), core.WithWeighted())
	g.AddEdge("S", "A", 5)
	g.AddEdge("S", "C", 15)
	g.AddEdge("A", "B", 8)
	g.AddEdge("B", "D", 10)
	g.AddEdge("C", "D", 5)
	g.AddEdge("C", "E", 10)
	g.AddEdge("E", "D", 10)
	g.AddEdge("D", "T", 10)
	g.AddEdge("E", "T", 5)

	// Default options: background context, Epsilon=1e-9, no logging.
	opts := flow.DefaultOptions()
	opts.Ctx = context.Background()

	// Compute max flow via Edmonds–Karp.
	maxFlow, _, err := flow.EdmondsKarp(g, "S", "T", opts)
	if err != nil {
		panic(err)
	}

	// Print result (should match Ford–Fulkerson).
	fmt.Println(maxFlow)
	// Output:
	// 15
}

// //////////////////////////////////////////////////////////////////////////////
// ExampleDinic_complex demonstrates Dinic on the same network.
// It builds a level graph and pushes blocking flows, achieving O(E·√V) on unit networks.
// //////////////////////////////////////////////////////////////////////////////
// Playground: [![Playground - Dinic](https://img.shields.io/badge/Go_Playground-Prim-blue?logo=go)](https://go.dev/play/p/lnq6XOgGUBn)
func ExampleDinic_complex() {
	// Construct the same directed, weighted graph.
	g := core.NewGraph(core.WithDirected(true), core.WithWeighted())
	g.AddEdge("S", "A", 5)
	g.AddEdge("S", "C", 15)
	g.AddEdge("A", "B", 8)
	g.AddEdge("B", "D", 10)
	g.AddEdge("C", "D", 5)
	g.AddEdge("C", "E", 10)
	g.AddEdge("E", "D", 10)
	g.AddEdge("D", "T", 10)
	g.AddEdge("E", "T", 5)

	// Prepare options: background context, use default Epsilon, no verbosity.
	opts := flow.DefaultOptions()
	opts.Ctx = context.Background()

	// Run Dinic’s algorithm.
	maxFlow, _, err := flow.Dinic(g, "S", "T", opts)
	if err != nil {
		panic(err)
	}

	// Output the computed flow (consistent across all algorithms).
	fmt.Println(maxFlow)
	// Output:
	// 15
}
