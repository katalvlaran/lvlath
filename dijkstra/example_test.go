// Package dijkstra_test provides examples demonstrating how to use the Dijkstra algorithm.
// Each example is runnable via “go test -run Example”, showing both code and expected output.
package dijkstra_test

import (
	"fmt" // fmt is used to print results in examples
	// Import core to build Graphs
	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/matrix"

	"github.com/katalvlaran/lvlath/dijkstra"
)

// ExampleDijkstra_Triangle demonstrates computing shortest paths on a simple triangle graph.
// Complexity: O((V+E) log V) because we push/pop up to E entries and extract each vertex once.
func ExampleDijkstra_Triangle() {
	// 1) Create a new weighted graph. By passing core.WithWeighted(), we enable non-negative weights.
	g := core.NewGraph(core.WithWeighted())
	// 2) Add an undirected edge A—B with weight=1.
	g.AddEdge("A", "B", 1)
	// 3) Add an undirected edge B—C with weight=2.
	g.AddEdge("B", "C", 2)
	// 4) Add an undirected edge A—C with weight=5.
	g.AddEdge("A", "C", 5)

	// 5) Compute Dijkstra from source "A" without requesting the predecessor map.
	//    We call dijkstra.Source("A") to set the Source field; no WithReturnPath() means prev==nil.
	dist, _, err := dijkstra.Dijkstra(
		g,
		dijkstra.Source("A"),
	)
	// 6) Handle any potential error (e.g., empty source or unweighted graph).
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// 7) Print distances to A, B, and C.
	//    dist["A"] should be 0, dist["B"] should be 1, dist["C"] should be 3 (via A→B→C).
	fmt.Printf("dist[A]=%d, dist[B]=%d, dist[C]=%d\n", dist["A"], dist["B"], dist["C"])
	// Output: dist[A]=0, dist[B]=1, dist[C]=3
}

// ExampleDijkstra_MediumGraph demonstrates path reconstruction on a slightly larger graph.
// We show how to use WithReturnPath() to obtain the predecessor (prev) map.
// Complexity: O((V+E) log V).
func ExampleDijkstra_MediumGraph() {
	// 1) Create a new directed, weighted graph.
	g := core.NewGraph(core.WithDirected(true), core.WithWeighted())
	// 2) Add directed edge A→B weight=2.
	g.AddEdge("A", "B", 2)
	// 3) Add directed edge A→C weight=1.
	g.AddEdge("A", "C", 1)
	// 4) Add directed edge C→B weight=1.
	g.AddEdge("C", "B", 1)
	// 5) Add directed edge B→D weight=3.
	g.AddEdge("B", "D", 3)
	// 6) Add directed edge C→D weight=5.
	g.AddEdge("C", "D", 5)

	// 7) Run Dijkstra from source "A", requesting the predecessor map via WithReturnPath().
	dist, prev, err := dijkstra.Dijkstra(
		g,
		dijkstra.Source("A"),
		dijkstra.WithReturnPath(),
	)
	// 8) Handle potential errors.
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// 9) Print the distance to "D" and its immediate predecessor.
	//    The shortest path to D is A→C→B→D with total cost 1+1+3 = 5.
	fmt.Printf("dist[D]=%d, prev[D]=%s\n", dist["D"], prev["D"])
	// Output: dist[D]=5, prev[D]=B
}

// ExampleDijkstra_Thresholds demonstrates how to use InfEdgeThreshold and MaxDistance
// to impose “walls” and distance caps. If an edge weight ≥ threshold, we treat it as impassable.
// Complexity: O((V+E) log V).
func ExampleDijkstra_Thresholds() {
	// 1) Create a new weighted graph.
	g := core.NewGraph(core.WithWeighted())
	// 2) Add an edge A—B weight=2.
	g.AddEdge("A", "B", 2)
	// 3) Add an edge B—C weight=4.
	g.AddEdge("B", "C", 4)
	// 4) Add an edge A—C weight=10.
	g.AddEdge("A", "C", 10)

	// 5) Define a threshold = 5. Any edge with weight ≥5 is skipped.
	threshold := int64(5)
	// 6) Run Dijkstra from "A" with InfEdgeThreshold=5.
	//    This causes the direct edge A—C (weight=10) to be ignored.
	dist, _, err := dijkstra.Dijkstra(
		g,
		dijkstra.Source("A"),
		dijkstra.WithInfEdgeThreshold(threshold),
	)
	// 7) Handle any errors.
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// 8) Now the only path from A→C goes A→B→C = 2 + 4 = 6.
	fmt.Printf("dist[C]=%d\n", dist["C"])
	// Output: dist[C]=6
}

// ExampleDijkstra_FromMatrix demonstrates building a graph via an adjacency matrix,
// and then running Dijkstra on it. We show how to convert from Graph → Matrix → Graph.
// Complexity: constructing the matrix is O(V^2), Dijkstra is O((V+E) log V).
func ExampleDijkstra_FromMatrix() {
	// 1) Build an initial weighted graph g0.
	g0 := core.NewGraph(core.WithWeighted())
	// 2) Add directed edges A→B weight=5, B→C weight=7.
	g0.AddEdge("A", "B", 5)
	g0.AddEdge("B", "C", 7)

	// 3) Convert g0 to an adjacency matrix. We set weighted=true so the matrix stores weights.
	am, err := matrix.NewAdjacencyMatrix(g0, matrix.NewMatrixOptions(matrix.WithWeighted(true)))
	if err != nil {
		fmt.Println("error building adjacency matrix:", err)
		return
	}

	// 4) Convert the adjacency matrix back to a new graph g1.
	g1, err := am.ToGraph()
	if err != nil {
		fmt.Println("error converting matrix to graph:", err)
		return
	}

	// 5) Run Dijkstra on g1 from source "A" without requesting prev.
	dist, _, err := dijkstra.Dijkstra(g1, dijkstra.Source("A"))
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// 6) Print distance to "C", which should be 5 + 7 = 12.
	fmt.Printf("dist[C]=%d\n", dist["C"])
	// Output: dist[C]=12
}

// ExampleDijkstra_HouseGraph shows Dijkstra on a small directed, weighted graph.
// Scenario as in buildWeightedMedium.
// Expected: the shortest costs to D and E from A.
func ExampleDijkstra_HouseGraph() {
	// Source graph g:
	//	    (E)
	//	  3/   \4
	//	  /     \
	//	(C)──10─(D)
	//	 |       |
	//	2|       |5
	//	 |       |
	//	(A)──4──(B)
	g := core.NewGraph(core.WithWeighted()) // directed, weighted
	for _, e := range []struct {
		U, V string
		W    int64
	}{
		{"A", "B", 4},
		{"A", "C", 2},
		{"B", "D", 5},
		{"C", "D", 10},
		{"C", "E", 3},
		{"E", "D", 4},
	} {
		g.AddEdge(e.U, e.V, e.W)
	}
	dist, _, _ := dijkstra.Dijkstra(g, dijkstra.Source("A"))
	fmt.Printf("dist[D]=%d dist[E]=%d\n", dist["D"], dist["E"])
	// Output: dist[D]=9 dist[E]=5
}
