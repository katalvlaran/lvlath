package flow_test

import (
	"context"
	"fmt"

	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/flow"
)

////////////////////////////////////////////////////////////////////////////////
// Ford–Fulkerson Examples
////////////////////////////////////////////////////////////////////////////////

// ExampleFordFulkerson_simple demonstrates max-flow on a single-edge network.
// Graph: s→t with capacity 5
func ExampleFordFulkerson_simple() {
	g := core.NewGraph(true, true)
	g.AddEdge("s", "t", 5)

	maxFlow, _, _ := flow.FordFulkerson(context.Background(), g, "s", "t", nil)
	fmt.Println(maxFlow)
	// Output:
	// 5
}

// ExampleFordFulkerson_medium shows Ford–Fulkerson on a two‐path network.
// Graph:
//
//	s→a(3)→t
//	s→b(2)→t
//
// Expected flow: max(s→a→t)=2 + max(s→b→t)=2 + remaining s→a→t=1 ⇒ 4
func ExampleFordFulkerson_medium() {
	g := core.NewGraph(true, true)
	g.AddEdge("s", "a", 3)
	g.AddEdge("a", "t", 2)
	g.AddEdge("s", "b", 2)
	g.AddEdge("b", "t", 3)

	maxFlow, _, _ := flow.FordFulkerson(context.Background(), g, "s", "t", nil)
	fmt.Println(maxFlow)
	// Output:
	// 4
}

////////////////////////////////////////////////////////////////////////////////
// Dinic Examples
////////////////////////////////////////////////////////////////////////////////

// ExampleDinic_simple demonstrates Dinic on a single-edge network.
// Graph: s→t with capacity 7
func ExampleDinic_simple() {
	g := core.NewGraph(true, true)
	g.AddEdge("s", "t", 7)

	maxFlow, _, _ := flow.Dinic(g, "s", "t", nil)
	fmt.Println(maxFlow)
	// Output:
	// 7
}

// ExampleDinic_medium demonstrates Dinic on a network with two augmenting paths.
// Graph:
//
//	s→a(5)→t
//	s→b(3)→t
//
// Expected max-flow = 5 + 3 = 8
func ExampleDinic_medium() {
	g := core.NewGraph(true, true)
	g.AddEdge("s", "a", 5)
	g.AddEdge("a", "t", 4)
	g.AddEdge("s", "b", 3)
	g.AddEdge("b", "t", 6)

	maxFlow, _, _ := flow.Dinic(g, "s", "t", nil)
	fmt.Println(maxFlow)
	// Output:
	// 9
}
