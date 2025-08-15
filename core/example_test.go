package core_test

import (
	"fmt"
	"sort"

	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/matrix"
)

// sortAsc returns a sorted copy of ids.
func sortAsc(ids []string) []string {
	out := append([]string(nil), ids...)
	sort.Strings(out)

	return out
}

// extractEdgeIDs is a helper to pull IDs from []*core.Edge.
func extractEdgeIDs(edges []*core.Edge) []string {
	out := make([]string, len(edges))
	for i, e := range edges {
		out[i] = e.ID
	}

	return out
}

// ExampleGraph_Simple demonstrates basic undirected, unweighted usage.
func ExampleGraph_Simple() {
	// 1) Create an undirected, unweighted graph:
	g := core.NewGraph()

	// 2) Add edges (auto-creates vertices A, B, C)
	_, _ = g.AddEdge("A", "B", 0)
	_, _ = g.AddEdge("B", "C", 0)

	// 3) Inspect vertices and edge IDs:
	fmt.Println("Vertices:", sortAsc(g.Vertices()))
	fmt.Println("Edges IDs:", sortAsc(extractEdgeIDs(g.Edges())))

	// 4) Query existence and then remove B:
	fmt.Println("HasEdge B→C?", g.HasEdge("B", "C"))
	_ = g.RemoveVertex("B")
	fmt.Println("After RemoveVertex(B):", sortAsc(g.Vertices()))
	fmt.Println("HasEdge A→B?", g.HasEdge("A", "B"))

	// Output:
	// Vertices: [A B C]
	// Edges IDs: [e1 e2]
	// HasEdge B→C? true
	// After RemoveVertex(B): [A C]
	// HasEdge A→B? false
}

// ExampleGraph_Advanced shows weighted multi-edges and self-loops.
func ExampleGraph_Advanced() {
	// Enable weights, multi-edges, and loops:
	g := core.NewGraph(
		core.WithWeighted(),
		core.WithMultiEdges(),
		core.WithLoops(),
	)

	// Create parallel edges A→B with weights 5 and 7:
	_, _ = g.AddEdge("A", "B", 5)
	_, _ = g.AddEdge("A", "B", 7)

	// Create a self-loop on C with weight 3:
	loopID, _ := g.AddEdge("C", "C", 3)

	// List and sort all edge IDs:
	edgeIDs := sortAsc(extractEdgeIDs(g.Edges()))
	fmt.Println("All edge IDs:", edgeIDs)

	// Show each edge’s weight:
	for _, eid := range edgeIDs {
		// lookup in map
		for _, e := range g.Edges() {
			if e.ID == eid {
				fmt.Printf("%s weight=%d\n", eid, e.Weight)
			}
		}
	}

	// Identify the loop:
	fmt.Println("Self-loop ID:", loopID)

	// Output (order may vary by ID but sorted above):
	// All edge IDs: [e1 e2 e3]
	// e1 weight=5
	// e2 weight=7
	// e3 weight=3
	// Self-loop ID: e3
}

// extractNeighborInfo formats each neighbor edge as To(w=Weight,d=Directed).
func extractNeighborInfo(edges []*core.Edge) []string {
	out := make([]string, len(edges))
	for i, e := range edges {
		out[i] = fmt.Sprintf("%s(w=%d,d=%v)", e.To, e.Weight, e.Directed)
	}
	sort.Strings(out)

	return out
}

// ExampleGraph_Special builds a 10-vertex graph demonstrating
// directed edges, undirected overrides, self-loops, parallel edges,
// weighted edges, and then inspects matrices.
//
// Graph configuration:
//   - weighted
//   - multi-edges
//   - self-loops
//   - default undirected
func ExampleGraph_Special() {
	// 1) Create a weighted, multi-edge, loop-enabled, mixed-mode graph
	g := core.NewGraph(
		core.WithMixedEdges(),
		core.WithWeighted(),
		core.WithMultiEdges(),
		core.WithLoops(),
	)

	// 2) Add a variety of edges:
	_, _ = g.AddEdge("A", "B", 10, core.WithEdgeDirected(true)) // directed A→B
	_, _ = g.AddEdge("B", "C", 5, core.WithEdgeDirected(false)) // explicit undirected B—C
	_, _ = g.AddEdge("C", "C", 2)                               // self-loop on C
	_, _ = g.AddEdge("D", "E", 3)                               // first D—E
	_, _ = g.AddEdge("D", "E", 4)                               // parallel D—E
	_, _ = g.AddEdge("F", "G", 1, core.WithEdgeDirected(true))  // directed F→G
	_, _ = g.AddEdge("G", "F", 1, core.WithEdgeDirected(false)) // explicit undirected G—F
	_, _ = g.AddEdge("H", "I", 7)                               // H—I
	_, _ = g.AddEdge("I", "J", 8)                               // I—J
	_, _ = g.AddEdge("J", "J", 6)                               // self-loop on J

	// 3) Inspect basic graph properties
	fmt.Println("Vertices:", sortAsc(g.Vertices()))
	fmt.Println("Edges:", sortAsc(extractEdgeIDs(g.Edges())))
	fmt.Printf("Directed A→B? %v\n", g.HasEdge("A", "B"))
	fmt.Printf("Undirected B—C? %v\n", g.HasEdge("B", "C"))
	fmt.Printf("Loop on C? %v\n", g.HasEdge("C", "C"))
	fmt.Printf("DE parallel weights: %v\n",
		[]int64{g.Edges()[4].Weight, g.Edges()[5].Weight}) // weights 3 and 4

	// 4) Neighbors of D (should list both parallel edges)
	nbD, _ := g.Neighbors("D")
	fmt.Println("Neighbors of D:", extractNeighborInfo(nbD))

	// 5) Build an adjacency matrix matching graph features
	mo := matrix.NewMatrixOptions(
		matrix.WithDirected(true),   // include directed arcs
		matrix.WithWeighted(true),   // preserve actual weights
		matrix.WithAllowMulti(true), // show parallel edges
		matrix.WithAllowLoops(true), // include self-loops
	)
	am, _ := matrix.NewAdjacencyMatrix(g, mo)

	// 6) Query the matrix: A→B should be 10.0
	iA, iB := am.VertexIndex["A"], am.VertexIndex["B"]
	wAB, _ := am.Mat.At(iA, iB)
	fmt.Printf("Adjacency A→B weight: %f\n", wAB)

	// Output:
	// Vertices: [A B C D E F G H I J]
	// Edges: [e1 e10 e2 e3 e4 e5 e6 e7 e8 e9]
	// Directed A→B? true
	// Undirected B—C? true
	// Loop on C? true
	// DE parallel weights: [3 4]
	// Neighbors of D: [E(w=3,d=false) E(w=4,d=false)]
	// Adjacency A→B weight: 10.000000
}
