package tsp_test

import (
	"fmt"
	"math"

	"github.com/katalvlaran/lvlath/tsp"
)

////////////////////////////////////////////////////////////////////////////////
// Helper distance generators
////////////////////////////////////////////////////////////////////////////////

// build4Cycle returns the 4-node cycle distance matrix:
//
//	0 ↔1↔ 1
//	↓     ↓
//	3 ↔1↔ 2
//
// optimal tour cost = 4.
func build4Cycle() [][]float64 {
	return [][]float64{
		{0, 1, 2, 1},
		{1, 0, 1, 2},
		{2, 1, 0, 1},
		{1, 2, 1, 0},
	}
}

// build8Cycle returns the 8-node cycle distance matrix,
// where dist(i,j)=min(|i−j|,8−|i−j|).  Optimal cost = 8.
func build8Cycle() [][]float64 {
	const n = 8
	mat := make([][]float64, n)
	for i := 0; i < n; i++ {
		mat[i] = make([]float64, n)
		for j := 0; j < n; j++ {
			d := math.Abs(float64(i - j))
			mat[i][j] = math.Min(d, float64(n)-d)
		}
	}
	return mat
}

// breakEdge returns a copy of mat with the edge (u,v) removed
// (distance = ∞) to simulate a disconnected graph.
func breakEdge(mat [][]float64, u, v int) [][]float64 {
	n := len(mat)
	copyMat := make([][]float64, n)
	for i := range mat {
		copyMat[i] = append([]float64(nil), mat[i]...)
	}
	copyMat[u][v] = math.Inf(1)
	copyMat[v][u] = math.Inf(1)
	return copyMat
}

////////////////////////////////////////////////////////////////////////////////
// Held–Karp Exact TSP Examples
////////////////////////////////////////////////////////////////////////////////

// ExampleTSPExact_Small demonstrates the Held–Karp dynamic‐programming TSP
// on a 4‐node cycle.
//
// Graph (cycle):
//
//	0 —1— 1
//	|     |
//	3 —1— 2
//
// Complexity: O(n²·2ⁿ), Memory: O(n·2ⁿ)
// Expected optimal tour: [0 → 1 → 2 → 3 → 0], cost = 4.
func ExampleTSPExact_Small() {
	mat := build4Cycle()
	res, err := tsp.TSPExact(mat)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Exact 4-cycle cost: %.0f\n", res.Cost)
	// Output:
	// Exact 4-cycle cost: 4
}

// ExampleTSPExact_Medium demonstrates Held–Karp on an 8‐node cycle.
//
// Graph: nodes 0–7 arranged in a ring, unit‐distance to neighbors.
//
// Complexity: O(n²·2ⁿ), Memory: O(n·2ⁿ)
// Expected optimal cost = 8.
func ExampleTSPExact_Medium() {
	mat := build8Cycle()
	res, err := tsp.TSPExact(mat)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Exact 8-cycle cost: %.0f\n", res.Cost)
	// Output:
	// Exact 8-cycle cost: 8
}

// ExampleTSPExact_Disconnected shows error on an incomplete graph.
//
// Start with a 5-cycle, then remove edge 1–2 (make it ∞).
// Expect ErrTSPIncompleteGraph.
func ExampleTSPExact_Disconnected() {
	// 5-cycle using first 5 rows of build8Cycle
	mat := build8Cycle()[0:5]
	mat = breakEdge(mat, 1, 2)
	_, err := tsp.TSPExact(mat)
	fmt.Printf("Error: %v\n", err)
	// Output:
	// Error: tsp: incomplete distance matrix
}

////////////////////////////////////////////////////////////////////////////////
// Christofides 1.5-Approximation Examples
////////////////////////////////////////////////////////////////////////////////

// ExampleTSPApprox_Small demonstrates Christofides’ algorithm on a 4-cycle.
//
// Even though approximate, on a metric cycle it finds the exact tour.
//
// Complexity: O(n³), Memory: O(n²)
// Expected cost = 4.
func ExampleTSPApprox_Small() {
	mat := build4Cycle()
	res, err := tsp.TSPApprox(mat)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Approx 4-cycle cost: %.0f\n", res.Cost)
	// Output:
	// Approx 4-cycle cost: 4
}

// ExampleTSPApprox_Medium demonstrates Christofides on an 8-node cycle.
//
// Complexity: O(n³), Memory: O(n²)
// Expected cost = 8.
func ExampleTSPApprox_Medium() {
	mat := build8Cycle()
	res, err := tsp.TSPApprox(mat)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Approx 8-cycle cost: %.0f\n", res.Cost)
	// Output:
	// Approx 8-cycle cost: 8
}

// ExampleTSPApprox_Disconnected shows Christofides error on a disconnected graph.
//
// Break one edge in a 6-node cycle, expect ErrTSPIncompleteGraph.
func ExampleTSPApprox_Disconnected() {
	mat := build8Cycle()[0:6]
	mat = breakEdge(mat, 2, 3)
	_, err := tsp.TSPApprox(mat)
	fmt.Printf("Error: %v\n", err)
	// Output:
	// Error: tsp: incomplete distance matrix
}
