package bfs_test

import (
	"context"
	"fmt"
	"time"

	"github.com/katalvlaran/lvlath/bfs"
	"github.com/katalvlaran/lvlath/core"
)

// ExampleBFS_GridTraversal demonstrates BFS layering on a 3×3 grid (9 vertices).
// We expect to see the start at "0_0", then its 2 neighbors {"0_1","1_0"}, then the next frontier, etc.
func ExampleBFS_GridTraversal() {
	// Build a 3×3 undirected grid: vertices "i_j" for 0 ≤ i,j < 3
	g := core.NewGraph()
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			// connect to right neighbor
			if j+1 < 3 {
				g.AddEdge(fmt.Sprintf("%d_%d", i, j), fmt.Sprintf("%d_%d", i, j+1), 0)
			}
			// connect to down neighbor
			if i+1 < 3 {
				g.AddEdge(fmt.Sprintf("%d_%d", i, j), fmt.Sprintf("%d_%d", i+1, j), 0)
			}
		}
	}

	// BFS from top-left corner
	res, err := bfs.BFS(g, "0_0")
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// Print the visit order; should follow non-decreasing Manhattan distance
	fmt.Println(res.Order)
	// Output:
	// [0_0 0_1 1_0 0_2 1_1 2_0 1_2 2_1 2_2]
}

// ExampleBFS_ShortestPathNetwork finds the fewest-hop path in a larger network of 11 vertices.
// Two competing routes exist from "A" to "K": one of length 4, another length 3.
func ExampleBFS_ShortestPathNetwork() {
	// Create an undirected graph with 11 nodes
	nodes := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K"}
	g := core.NewGraph()
	for _, u := range nodes {
		g.AddVertex(u) // Not required, Vertices will be crete automatically
	}
	// Route1: A–B–C–D–K (4 hops)
	g.AddEdge("A", "B", 0)
	g.AddEdge("B", "C", 0)
	g.AddEdge("C", "D", 0)
	g.AddEdge("D", "K", 0)
	// Route2: A–E–F–K (3 hops)
	g.AddEdge("A", "E", 0)
	g.AddEdge("E", "F", 0)
	g.AddEdge("F", "K", 0)
	// Some extra branches to other nodes
	g.AddEdge("C", "G", 0)
	g.AddEdge("G", "H", 0)
	g.AddEdge("D", "I", 0)
	g.AddEdge("I", "J", 0)

	// Run BFS and reconstruct path
	res, err := bfs.BFS(g, "A")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	path, err := res.PathTo("K")
	if err != nil {
		fmt.Println("no path:", err)
		return
	}
	fmt.Println(path)
	// Output:
	// [A E F K]
}

// ExampleBFS_DepthLimitOnChain shows applying WithMaxDepth to a linear chain of 10 vertices.
// With depth=2 we only visit the first three nodes.
func ExampleBFS_DepthLimitOnChain() {
	// Build a chain v0→v1→...→v9 (10 vertices)
	g := core.NewGraph()
	for i := 0; i < 9; i++ {
		u := fmt.Sprintf("v%d", i)
		v := fmt.Sprintf("v%d", i+1)
		g.AddEdge(u, v, 0)
	}

	// Limit depth to 2: should see v0, v1, v2 only
	res, err := bfs.BFS(g, "v0", bfs.WithMaxDepth(2))
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(res.Order)
	// Output:
	// [v0 v1 v2]
}

// ExampleBFS_FilterAndMixed demonstrates filtering and mixed-edge handling on a 5-node graph.
// Node U–V is undirected, V→W directed, W–X undirected, X→Y directed. We then filter out X→W.
func ExampleBFS_FilterAndMixed() {
	// Mixed-mode graph
	g := core.NewGraph(core.WithMixedEdges())
	// U–V undirected
	g.AddEdge("U", "V", 0, core.WithEdgeDirected(false))
	// V→W directed
	g.AddEdge("V", "W", 0, core.WithEdgeDirected(true))
	// W–X undirected
	g.AddEdge("W", "X", 0, core.WithEdgeDirected(false))
	// X→Y directed
	g.AddEdge("X", "Y", 0, core.WithEdgeDirected(true))

	// Filter to block traversal back to W from X
	filter := func(curr, nbr string) bool {
		// block the reverse of W–X
		return !(curr == "X" && nbr == "W")
	}

	res, err := bfs.BFS(g, "U", bfs.WithFilterNeighbor(filter))
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(res.Order)
	// Output:
	// [U V W X Y]
}

// ExampleBFS_HooksAndCancellation demonstrates OnEnqueue, OnDequeue, OnVisit hooks
// alongside context cancellation on a 7-node chain.
func ExampleBFS_HooksAndCancellation() {
	// Build chain of 7 vertices: n0→...→n6
	g := core.NewGraph()
	for i := 0; i < 6; i++ {
		g.AddEdge(fmt.Sprintf("n%d", i), fmt.Sprintf("n%d", i+1), 0)
	}

	// Cancel after visiting 4 nodes
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	var enqSeq, deqSeq, visSeq []string

	// after depth 4, we call cancel()
	hookVisit := func(id string, d int) error {
		visSeq = append(visSeq, fmt.Sprintf("V[%s@%d]", id, d))
		if d == 4 {
			cancel() // force mid-traversal cancellation
		}
		return nil
	}

	_, err := bfs.BFS(
		g, "n0",
		bfs.WithContext(ctx),
		bfs.WithOnEnqueue(func(id string, d int) { enqSeq = append(enqSeq, fmt.Sprintf("E[%s@%d]", id, d)) }),
		bfs.WithOnDequeue(func(id string, d int) { deqSeq = append(deqSeq, fmt.Sprintf("D[%s@%d]", id, d)) }),
		bfs.WithOnVisit(hookVisit),
	)

	fmt.Println("error:", err) // We ignore cancellation error for the example output
	fmt.Println("Enqueued:", enqSeq)
	fmt.Println("Dequeued:", deqSeq)
	fmt.Println("Visited: ", visSeq)
	// Output:
	// error: context canceled
	// Enqueued: [E[n0@0] E[n1@1] E[n2@2] E[n3@3] E[n4@4]]
	// Dequeued: [D[n0@0] D[n1@1] D[n2@2] D[n3@3] D[n4@4]]
	// Visited:  [V[n0@0] V[n1@1] V[n2@2] V[n3@3] V[n4@4]]
}
