package algorithms_test

import (
	"fmt"

	"github.com/katalvlaran/lvlath/graph/algorithms"
	"github.com/katalvlaran/lvlath/graph/core"
)

// buildTriangle builds A–B(1), B–C(2), A–C(3).
func buildTriangle() *core.Graph {
	g := core.NewGraph(false, true)
	g.AddEdge("A", "B", 1)
	g.AddEdge("B", "C", 2)
	g.AddEdge("A", "C", 3)
	return g
}

// ExampleBFS shows breadth-first order.
func ExampleBFS() {
	g := core.NewGraph(false, false)
	g.AddEdge("A", "B", 0)
	g.AddEdge("A", "C", 0)
	g.AddEdge("B", "D", 0)

	res, _ := algorithms.BFS(g, "A", nil)
	for _, v := range res.Order {
		fmt.Print(v.ID)
	}
	// Output: ABCD
}

// ExampleDFS shows depth-first order.
func ExampleDFS() {
	g := core.NewGraph(false, false)
	g.AddEdge("A", "B", 0)
	g.AddEdge("A", "C", 0)

	res, _ := algorithms.DFS(g, "A", nil)
	for _, v := range res.Order {
		fmt.Print(v.ID)
	}
	// Output: ABC
}

// ExampleDijkstra demonstrates shortest-path distances.
func ExampleDijkstra() {
	g := buildTriangle()
	dist, parent, _ := algorithms.Dijkstra(g, "A")
	fmt.Println("dist[C] =", dist["C"])
	fmt.Println("parent[C] =", parent["C"])
	// Output:
	// dist[C] = 3
	// parent[C] = B
}

// ExamplePrim demonstrates Prim’s MST.
func ExamplePrim() {
	g := buildTriangle()
	edges, sum, _ := algorithms.Prim(g, "A")
	fmt.Println("total weight:", sum)
	for _, e := range edges {
		fmt.Printf("%s-%s ", e.From.ID, e.To.ID)
	}
	// Output:
	// total weight: 3
	// A-B B-C
}

// ExampleKruskal demonstrates Kruskal’s MST.
func ExampleKruskal() {
	g := buildTriangle()
	edges, sum, _ := algorithms.Kruskal(g)
	fmt.Println("total weight:", sum)
	for _, e := range edges {
		fmt.Printf("%s-%s ", e.From.ID, e.To.ID)
	}
	// Output:
	// total weight: 3
	// A-B B-C
}
