package matrix_test

//
//import (
//	"fmt"
//	"sort"
//
//	"github.com/katalvlaran/lvlath/core"
//	"github.com/katalvlaran/lvlath/matrix"
//)
//
//// helper to sort and print slices predictably
//func sortedStrings(in []string) []string {
//	out := append([]string(nil), in...)
//	sort.Strings(out)
//	return out
//}
//
//// ExampleMatrix_roundtrip demonstrates converting between core.Graph,
//// Matrix, AdjacencyMatrix, and back.
//func ExampleMatrix_roundtrip() {
//	// 1) Build initial graph X→Y(7), Y→Z(9)
//	g := core.NewGraph(false, true)
//	g.AddEdge("X", "Y", 7)
//	g.AddEdge("Y", "Z", 9)
//
//	// 2) Lightweight Matrix
//	m1 := matrix.ToMatrix(g)
//	fmt.Println("Matrix[X][Y] =", m1.Data[m1.Index["X"]][m1.Index["Y"]])
//
//	// 3) Full AdjacencyMatrix
//	am := matrix.NewAdjacencyMatrix(g)
//	// Add a new edge Z→X(5)
//	_ = am.AddEdge("Z", "X", 5)
//	fmt.Println("AdjMatrix[Z][X] =", am.Data[am.Index["Z"]][am.Index["X"]])
//
//	// 4) Round-trip back to Graph
//	g2 := am.ToGraph(true)
//	fmt.Println("Graph2 has Z→X?", g2.HasEdge("Z", "X"))
//
//	// Output:
//	// Matrix[X][Y] = 7
//	// AdjMatrix[Z][X] = 5
//	// Graph2 has Z→X? true
//}
//
//// ExampleAdjacencyMatrix_ops shows basic operations on AdjacencyMatrix.
//func ExampleAdjacencyMatrix_ops() {
//	// Start with A–B(4), B–C(6)
//	g := core.NewGraph(false, true)
//	g.AddEdge("A", "B", 4)
//	g.AddEdge("B", "C", 6)
//
//	am := matrix.NewAdjacencyMatrix(g)
//
//	// Add edge C–D(2)
//	_ = am.AddEdge("C", "D", 2)
//	// Remove edge A–B
//	_ = am.RemoveEdge("A", "B")
//
//	// Query neighbors of C
//	nbrs, _ := am.Neighbors("C")
//	fmt.Println("Neighbors of C:", sortedStrings(nbrs))
//
//	// Output:
//	// Neighbors of C: [B D]
//}
