package tsp_test

import (
	"math"
	"testing"

	"github.com/katalvlaran/lvlath/tsp"
	"github.com/stretchr/testify/require"
)

// makeCycleDist builds the distance matrix for an n-node cycle graph.
//   - dist[i][j] = min(|i - j|, n - |i - j|)
//
// The optimal Hamiltonian cycle cost on this graph equals n.
func makeCycleDist(n int) [][]float64 {
	dist := make([][]float64, n) // allocate n rows
	for i := 0; i < n; i++ {
		dist[i] = make([]float64, n) // each row has n columns
		for j := 0; j < n; j++ {
			// compute the direct distance on the cycle
			d := math.Abs(float64(i - j))
			// wrap-around distance if going the other way is shorter
			dist[i][j] = math.Min(d, float64(n)-d)
		}
	}
	return dist
}

// TestTSPApprox_Small4 verifies that Christofides on a 4-node cycle
// finds the exact tour of cost 4 and returns a tour of length 5.
func TestTSPApprox_Small4(t *testing.T) {
	// Define distance matrix for the 4-cycle
	dist := [][]float64{
		{0, 1, 2, 1},
		{1, 0, 1, 2},
		{2, 1, 0, 1},
		{1, 2, 1, 0},
	}

	// Call the approximate solver with default options
	res, err := tsp.TSPApprox(dist, tsp.DefaultOptions())

	// Expect no error for a valid metric cycle
	require.NoError(t, err)

	// Tour length should be n+1 = 5
	require.Len(t, res.Tour, 5)

	// The tour must start and end at vertex 0
	require.Equal(t, 0, res.Tour[0])
	require.Equal(t, 0, res.Tour[4])

	// On a perfect cycle, Christofides returns the exact optimal cost = 4
	require.Equal(t, 4.0, res.Cost)
}

// TestTSPApprox_Medium8 verifies that Christofides on an 8-node cycle
// returns the exact tour of cost 8 and a tour of length 9.
func TestTSPApprox_Medium8(t *testing.T) {
	// Generate distance matrix for the 8-cycle
	dist := makeCycleDist(8)

	// Solve approximately
	res, err := tsp.TSPApprox(dist, tsp.DefaultOptions())

	// Expect no error
	require.NoError(t, err)

	// Tour length should be 8+1 = 9
	require.Len(t, res.Tour, 9)

	// Must start and end at 0
	require.Equal(t, 0, res.Tour[0])
	require.Equal(t, 0, res.Tour[8])

	// Expected cost = 8 on the cycle
	require.Equal(t, 8.0, res.Cost)
}

// TestTSPApprox_Disconnected ensures ErrIncompleteGraph is returned
// when the distance matrix has missing edges (Inf), breaking connectivity.
func TestTSPApprox_Disconnected(t *testing.T) {
	// Use a 6-node cycle as base
	n := 6
	dist := makeCycleDist(n)

	// Remove the edge between vertices 2 and 3 in both directions
	dist[2][3] = math.Inf(1)
	dist[3][2] = math.Inf(1)

	// Attempt to solve; should error out due to incomplete graph
	_, err := tsp.TSPApprox(dist, tsp.DefaultOptions())
	require.ErrorIs(t, err, tsp.ErrIncompleteGraph)
}

// TestTSPApprox_BadInput covers invalid-input scenarios for TSPApprox:
//   - Empty matrix
//   - Non-square (ragged) matrix
//   - Negative distances
//   - Asymmetric matrix
func TestTSPApprox_BadInput(t *testing.T) {
	// 1) Empty matrix should yield ErrBadInput
	_, err := tsp.TSPApprox([][]float64{}, tsp.DefaultOptions())
	require.ErrorIs(t, err, tsp.ErrBadInput)

	// 2) Ragged matrix: rows of unequal length
	ragged := [][]float64{{0, 1}, {1}}
	_, err = tsp.TSPApprox(ragged, tsp.DefaultOptions())
	require.ErrorIs(t, err, tsp.ErrBadInput)

	// 3) Negative distances are not allowed
	neg := [][]float64{
		{0, -1, 2},
		{-1, 0, 1},
		{2, 1, 0},
	}
	_, err = tsp.TSPApprox(neg, tsp.DefaultOptions())
	require.ErrorIs(t, err, tsp.ErrBadInput)

	// 4) Asymmetric matrix violates metric requirements
	asym := [][]float64{
		{0, 1, 2},
		{2, 0, 1}, // dist[1][0] != dist[0][1]
		{2, 1, 0},
	}
	_, err = tsp.TSPApprox(asym, tsp.DefaultOptions())
	require.ErrorIs(t, err, tsp.ErrBadInput)
}
