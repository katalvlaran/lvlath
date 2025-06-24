// Package main demonstrates a simple spectral analysis (power iteration)
// on an adjacency matrix of a graph using lvlath/matrix.
//
// Playground: https://go.dev/play/p/rRmJmeI9Vpa
//
// Scenario:
//
//	In network science, the principal eigenvalue (and eigenvector) of a
//	graph’s adjacency matrix reveals key structural properties: connectivity,
//	centrality, and can seed spectral clustering.  Here we build a small
//	undirected graph, convert it to an adjacency matrix, and use power
//	iteration to approximate its largest eigenvalue/vector.
//
// Graph: a 6-node “barbell” graph—two triangles connected by a single bridge:
//
//	  0
//	 / \
//	1───2---3───4
//	         \ /
//	          5
//
//	Triangle 0–1–2 and triangle 3–4–5 joined by edge 2–3.
//
// Use case:
//
//	Estimate graph connectivity and identify “central” nodes via eigenvector.
//
// Complexity:
//   - Building matrix: O(V + E)
//   - Each power-iteration step: O(V²) for dense matrix
//   - Total: O(I·V²), I = #iterations (here 20).
package main

//
//import (
//	"fmt"
//	"log"
//	"math"
//
//	"github.com/katalvlaran/lvlath/core"
//	"github.com/katalvlaran/lvlath/matrix"
//)
//
//func main9() {
//	// 1) Build the barbell graph
//	g := core.NewGraph(false, false) // undirected, unweighted
//	// Left triangle 0–1–2
//	g.AddEdge("0", "1", 1)
//	g.AddEdge("1", "2", 1)
//	g.AddEdge("2", "0", 1)
//	// Right triangle 3–4–5
//	g.AddEdge("3", "4", 1)
//	g.AddEdge("4", "5", 1)
//	g.AddEdge("5", "3", 1)
//	// Bridge between the two triangles
//	g.AddEdge("2", "3", 1)
//
//	// 2) Convert to adjacency matrix
//	am := matrix.NewAdjacencyMatrix(g)
//	n := len(am.Index)
//
//	// 3) Power iteration to approximate principal eigenvalue/vector
//	// Initialize vector v with 1s
//	v := make([]float64, n)
//	for i := range v {
//		v[i] = 1.0
//	}
//
//	var lambda float64
//	// Repeat: w = A · v; lambda = ||w||; v = w / lambda
//	for iter := 0; iter < 20; iter++ {
//		// w = A * v
//		w := make([]float64, n)
//		for _, uIdx := range am.Index {
//			// sum over neighbors: for dense, iterate all columns
//			for _, vIdx := range am.Index {
//				weight := am.Data[uIdx][vIdx]
//				if weight != 0 {
//					w[uIdx] += float64(weight) * v[vIdx]
//				}
//			}
//			// (no need to use uID here, index mapping suffices)
//		}
//		// Compute norm of w
//		norm := 0.0
//		for _, wi := range w {
//			norm += wi * wi
//		}
//		norm = math.Sqrt(norm)
//		if norm == 0 {
//			log.Fatal("Power iteration diverged to zero vector")
//		}
//		// Normalize to get next v, record eigenvalue estimate
//		for i := range v {
//			v[i] = w[i] / norm
//		}
//		lambda = norm
//	}
//
//	// 4) Display results
//	fmt.Printf("Approximate principal eigenvalue λ ≈ %.5f\n", lambda)
//	fmt.Println("Corresponding eigenvector (node: value):")
//	// Print in node order "0".."5"
//	for node := 0; node < 6; node++ {
//		id := fmt.Sprint(node)
//		idx := am.Index[id]
//		fmt.Printf("  Node %s: %.4f\n", id, v[idx])
//	}
//}
