package matrix_test

import (
	"fmt"

	"github.com/katalvlaran/lvlath/graph/core"
	"github.com/katalvlaran/lvlath/graph/matrix"
)

// ExampleMatrix_roundtrip shows conversion between core.Graph and both
// lightweight Matrix and full AdjacencyMatrix.
func ExampleMatrix_roundtrip() {
	// Build a weighted graph: X→Y(7), Y→Z(9)
	g := core.NewGraph(false, true)
	g.AddEdge("X", "Y", 7)
	g.AddEdge("Y", "Z", 9)

	// 1) Lightweight Matrix
	m1 := matrix.ToMatrix(g)
	fmt.Println("Matrix[X][Y] =", m1.Data[m1.Index["X"]][m1.Index["Y"]])

	// 2) Full AdjacencyMatrix
	am := matrix.NewAdjacencyMatrix(g)
	// Add a new edge Z→X(5)
	_ = am.AddEdge("Z", "X", 5)
	fmt.Println("AdjMatrix[Z][X] =", am.Data[am.Index["Z"]][am.Index["X"]])

	// 3) Round-trip back
	g2 := am.ToGraph(true)
	fmt.Println("Graph2 has Z→X?", g2.HasEdge("Z", "X"))

	// Output:
	// Matrix[X][Y] = 7
	// AdjMatrix[Z][X] = 5
	// Graph2 has Z→X? true
}
