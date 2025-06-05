package prim_kruskal_test

import (
	"fmt"

	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/prim_kruskal"
)

// ExampleKruskal_Triangle demonstrates Kruskal’s algorithm on the same triangle graph.
// The MST is the same set {A–B, B–C} with total weight = 3.
// Playground: [![Playground – Prim](https://img.shields.io/badge/Go_Playground-Prim-blue?logo=go)](https://go.dev/play/p/cWR3GQU2luz)
func ExampleKruskal_Triangle() {
	// 1. Construct a new weighted, undirected graph.
	g := core.NewGraph(core.WithWeighted())
	// 2. Add edges to form the triangle:
	g.AddEdge("A", "B", 1) // A—B with weight 1
	g.AddEdge("B", "C", 2) // B—C with weight 2
	g.AddEdge("A", "C", 4) // A—C with weight 4

	// 3. Run Kruskal’s algorithm.
	edges, total, err := prim_kruskal.Kruskal(g)
	if err != nil {
		// If any error occurs (e.g., disconnected), print it and exit.
		fmt.Println("error:", err)
		return
	}

	// 4. Print the total weight and the list of edges in the MST.
	fmt.Printf("Total: %d, Edges: ", total)
	for i, e := range edges {
		if i > 0 {
			fmt.Print(" ")
		}
		fmt.Printf("%s-%s", e.From, e.To)
	}
	// Output: Total: 3, Edges: A-B B-C
}

// ExamplePrim_Pentagon demonstrates Prim’s algorithm on a simple 5‐vertex pentagon graph.
// Vertices: A, B, C, D, E. Edges: A–B (1), B–C (2), C–D (3), D–E (5), A–E (12)
// The MST in this graph is edges {A–B, B–C, C-D, D-E} with total weight = 11.
// Playground: [![Playground – Prim](https://img.shields.io/badge/Go_Playground-Prim-blue?logo=go)](https://go.dev/play/p/2P5c7LC2Ac-)
func ExamplePrim_Pentagon() {
	// Construct triangle graph: A–B(1), B–C(2), C–D(3), D–E(5), A–E(12)
	g := core.NewGraph(core.WithWeighted())
	g.AddEdge("A", "B", 1)
	g.AddEdge("A", "E", 12)
	g.AddEdge("B", "C", 2)
	g.AddEdge("C", "D", 3)
	g.AddEdge("D", "E", 5)

	edges, total, err := prim_kruskal.Prim(g, "A")
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Printf("Total: %d, Edges: ", total)
	for i, e := range edges {
		if i > 0 {
			fmt.Print(" ")
		}
		fmt.Printf("%s-%s", e.From, e.To)
	}
	// Output: Total: 11, Edges: A-B B-C C-D D-E
}

// ExampleKruskal_MediumGraph demonstrates Kruskal’s algorithm on a larger 4‐vertex graph((letter envelope)).
// Vertices: A, B, C, D
// Edges:
//
//	A—B (4), B—C (2), C—D (5), D—A (4),
//	A—C (1), B—D (3).
//
// The MST has 3 edges: {A–C, C–B, B–D} with total weight = 6.
// Playground: [![Playground – Kruskal_medium](https://img.shields.io/badge/Go_Playground-Kruskal-blue?logo=go)](https://go.dev/play/p/aDggwYQ8H4Q)
func ExampleKruskal_MediumGraph() {
	// Medium graph: A–B(4), A–C(1), C–B(2), B–D(3), C–D(5), D–A(4)
	g := core.NewGraph(core.WithWeighted())
	g.AddEdge("A", "B", 4)
	g.AddEdge("A", "C", 1)
	g.AddEdge("C", "B", 2)
	g.AddEdge("B", "D", 3)
	g.AddEdge("C", "D", 5)
	g.AddEdge("D", "A", 4)

	edges, total, err := prim_kruskal.Kruskal(g)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Printf("Total: %d, Edges: ", total)
	for i, e := range edges {
		if i > 0 {
			fmt.Print(" ")
		}
		fmt.Printf("%s-%s", e.From, e.To)
	}
	// Output: Total: 6, Edges: A-C C-B B-D
}

// ExamplePrim_LargeGraph demonstrates Prim’s algorithm on a larger 7‐vertex graph.
// Vertices: A, B, C, D, E, F, G
// Edges:
//
//	B—C (1), D—E (1), A—B (2), E—G (2), F—G (3),
//	A—C (3), B—D (4), C—E (5), E—F (6), D—F (7).
//
// The MST has 6 edges: {A–B, B–C, B–D, D–E, E–G, E–F} with total weight = 16.
// Playground: [![Playground – Prim_large](https://img.shields.io/badge/Go_Playground-Kruskal-blue?logo=go)](https://go.dev/play/p/EwPJLIM1y31)
func ExamplePrim_LargeGraph() {
	// 1. Construct a new weighted, undirected graph.
	g := core.NewGraph(core.WithWeighted())

	// 2. Add all vertices one by one.
	g.AddVertex("A")
	g.AddVertex("B")
	g.AddVertex("C")
	g.AddVertex("D")
	g.AddVertex("E")
	g.AddVertex("F")
	g.AddVertex("G")

	// 3. Add edges with the specified weights (alternative path, will be skipped by MST).
	g.AddEdge("A", "B", 2)
	g.AddEdge("B", "C", 1)
	g.AddEdge("D", "E", 1)
	g.AddEdge("E", "G", 2)
	g.AddEdge("F", "G", 3)
	g.AddEdge("A", "C", 3)
	g.AddEdge("B", "D", 4)
	g.AddEdge("C", "E", 5)
	g.AddEdge("E", "F", 6)
	g.AddEdge("D", "F", 7)

	// 4. Run Prim’s algorithm, starting from vertex "A".
	edges, total, err := prim_kruskal.Prim(g, "A")
	if err != nil {
		// If graph were invalid or disconnected, print error and return.
		fmt.Println("error:", err)
		return
	}

	// 5. Print the total weight and the list of edges in the MST, in the order Prim discovered them.
	fmt.Printf("Total: %d, Edges: ", total)
	for i, e := range edges {
		if i > 0 {
			fmt.Print(" ")
		}
		fmt.Printf("%s-%s", e.From, e.To)
	}
	// Output: Total: 16, Edges: A-B B-C B-D D-E E-G E-F
}

func ExamplePrim_ErrDisconnected() {
	g := core.NewGraph(core.WithWeighted())
	// Attempt to run Prim with root "A" on an empty graph.
	_, _, err := prim_kruskal.Prim(g, "A")
	fmt.Println(err)
	// Output: prim_kruskal: graph is disconnected
}

func ExampleKruskal_ErrDisconnected() {
	g := core.NewGraph(core.WithWeighted())
	// Attempt to run Kruskal on an empty graph.
	_, _, err := prim_kruskal.Kruskal(g)
	fmt.Println(err)
	// Output: prim_kruskal: graph is disconnected
}
