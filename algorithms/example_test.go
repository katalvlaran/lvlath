package algorithms_test

import (
	"fmt"

	"github.com/katalvlaran/lvlath/algorithms"
	"github.com/katalvlaran/lvlath/core"
)

////////////////////////////////////////////////////////////////////////////////
// Helper builders for example graphs
////////////////////////////////////////////////////////////////////////////////

// buildSimpleChain constructs an undirected, unweighted path graph:
//
//	A — B — C
func buildSimpleChain() *core.Graph {
	g := core.NewGraph(false, false) // undirected, unweighted
	g.AddEdge("A", "B", 0)
	g.AddEdge("B", "C", 0)
	return g
}

// buildMediumDAG constructs an undirected, unweighted “diamond”‐shaped graph:
//
//	  A
//	 / \
//	B   C
//	 \ /
//	  D
//	 / \
//	E   F
func buildMediumDiamond() *core.Graph {
	g := core.NewGraph(false, false)
	for _, e := range []struct{ U, V string }{
		{"A", "B"}, {"A", "C"},
		{"B", "D"}, {"C", "D"},
		{"D", "E"}, {"D", "F"},
	} {
		g.AddEdge(e.U, e.V, 0)
	}
	return g
}

// buildWeightedTriangle constructs an undirected, weighted triangle:
//
//	A ——(1)—— B
//	 \      /
//	 (5)  (2)
//	   \ /
//	    C
func buildWeightedTriangle() *core.Graph {
	g := core.NewGraph(false, true) // undirected, weighted
	g.AddEdge("A", "B", 1)
	g.AddEdge("B", "C", 2)
	g.AddEdge("A", "C", 5)
	return g
}

// buildWeightedMedium constructs a small directed, weighted graph:
//
//	 A→B(4)   C→E(3)
//	 ↓    ↘    ↙
//	C(2)  D(5)
//	  ↘    ↙
//	   D(10)
func buildWeightedMedium() *core.Graph {
	g := core.NewGraph(true, true) // directed, weighted
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
	return g
}

////////////////////////////////////////////////////////////////////////////////
// BFS Examples
////////////////////////////////////////////////////////////////////////////////

// ExampleBFS_SimpleChain shows a breadth-first search on a simple path graph.
// Scenario:
//
//	Graph: A—B—C (undirected, unweighted)
//	Start vertex: "A"
//
// Expected output: visitation order A, then B, then C.
func ExampleBFS_SimpleChain() {
	g := buildSimpleChain()
	result, _ := algorithms.BFS(g, "A", nil)
	for _, v := range result.Order {
		fmt.Print(v.ID)
	}
	// Output: ABC
}

// ExampleBFS_MediumDiamond shows BFS on a 6-node “diamond” graph.
// Scenario:
//
//	 Graph:
//				   A
//	              / \
//	             B   C
//	              \ /
//	               D
//	              / \
//	             E   F
//	 Start vertex: "A"
//
// Expected output: layer by layer: A, then B,C, then D, then E,F.
func ExampleBFS_MediumDiamond() {
	g := buildMediumDiamond()
	result, _ := algorithms.BFS(g, "A", nil)
	for _, v := range result.Order {
		fmt.Print(v.ID)
	}
	// Output: ABCDEF
}

////////////////////////////////////////////////////////////////////////////////
// DFS Examples
////////////////////////////////////////////////////////////////////////////////

// ExampleDFS_SimpleChain shows depth-first search on a simple path graph.
// Scenario:
//
//	Graph: A—B—C (undirected, unweighted)
//	Start vertex: "A"
//
// Expected output: visits A, then B, then C (single path).
func ExampleDFS_SimpleChain() {
	g := buildSimpleChain()
	result, _ := algorithms.DFS(g, "A", nil)
	for _, v := range result.Order {
		fmt.Print(v.ID)
	}
	// Output: ABC
}

// ExampleDFS_MediumDiamond shows DFS on the “diamond” graph.
// Scenario same as BFS example.
// Note: DFS order depends on neighbor iteration; one possible valid
// traversal is A→B→D→E→F→C.
func ExampleDFS_MediumDiamond() {
	g := buildMediumDiamond()
	result, _ := algorithms.DFS(g, "A", nil)
	for _, v := range result.Order {
		fmt.Print(v.ID)
	}
	// Possible Output: ABDEFC
}

////////////////////////////////////////////////////////////////////////////////
// Dijkstra Examples
////////////////////////////////////////////////////////////////////////////////

// ExampleDijkstra_Triangle shows Dijkstra’s shortest paths on a weighted triangle.
// Scenario:
//
//	Graph: A—(1)—B—(2)—C and A—(5)—C
//	Start: "A"
//
// Expected: distance to C = 3 via A→B→C.
func ExampleDijkstra_Triangle() {
	g := buildWeightedTriangle()
	dist, _, _ := algorithms.Dijkstra(g, "A")
	fmt.Println("dist[C] =", dist["C"])
	// Output: dist[C] = 3
}

// ExampleDijkstra_MediumGraph shows Dijkstra on a small directed, weighted graph.
// Scenario as in buildWeightedMedium.
// Expected: the shortest costs to D and E from A.
func ExampleDijkstra_MediumGraph() {
	g := buildWeightedMedium()
	dist, _, _ := algorithms.Dijkstra(g, "A")
	fmt.Printf("dist[D]=%d dist[E]=%d\n", dist["D"], dist["E"])
	// Output: dist[D]=9 dist[E]=5
}

////////////////////////////////////////////////////////////////////////////////
// Prim & Kruskal MST Examples
////////////////////////////////////////////////////////////////////////////////

// ExamplePrim_Triangle shows Prim’s MST on the weighted triangle.
// Scenario: same as Dijkstra triangle.
// Expected: picks edges A–B(1) and B–C(2), total weight 3.
func ExamplePrim_Triangle() {
	g := buildWeightedTriangle()
	edges, total, _ := algorithms.Prim(g, "A")
	fmt.Println("total weight:", total)
	for _, e := range edges {
		fmt.Printf("%s-%s ", e.From.ID, e.To.ID)
	}
	// Output:
	// total weight: 3
	// A-B B-C
}

// ExamplePrim_MediumGraph shows Prim’s MST on the small directed, weighted graph.
// Note: Prim ignores direction; treats as undirected.
// Expected: minimal tree connecting all nodes.
func ExamplePrim_MediumGraph() {
	g := buildWeightedMedium()
	edges, total, _ := algorithms.Prim(g, "A")
	fmt.Println("total weight:", total)
	for _, e := range edges {
		fmt.Printf("%s-%s ", e.From.ID, e.To.ID)
	}
	// Possible Output:
	// total weight: 9
	// A-C C-E E-D A-B
}

// ExampleKruskal_Triangle shows Kruskal’s MST on the weighted triangle.
// Scenario same as Prim triangle.
// Expected: same result as Prim.
func ExampleKruskal_Triangle() {
	g := buildWeightedTriangle()
	edges, total, _ := algorithms.Kruskal(g)
	fmt.Println("total weight:", total)
	for _, e := range edges {
		fmt.Printf("%s-%s ", e.From.ID, e.To.ID)
	}
	// Output:
	// total weight: 3
	// A-B B-C
}

// ExampleKruskal_MediumGraph shows Kruskal’s MST on the small directed, weighted graph.
// Expected: same set of edges (undirected) as Prim’s tree.
func ExampleKruskal_MediumGraph() {
	g := buildWeightedMedium()
	edges, total, _ := algorithms.Kruskal(g)
	fmt.Println("total weight:", total)
	for _, e := range edges {
		fmt.Printf("%s-%s ", e.From.ID, e.To.ID)
	}
	// Possible Output:
	// total weight: 9
	// A-C A-B C-E E-D
}
