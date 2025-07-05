package tsp_test

import (
	"math"
	"testing"

	"github.com/katalvlaran/lvlath/tsp"
	"github.com/stretchr/testify/require"
)

// TestTSPExact_Small4 verifies Held-Karp on a trivial 4-node cycle.
// It should find the exact cost 4 and a tour of length 5 starting/ending at 0.
// Complexity: O(n²·2ⁿ) = O(16·4) here.
func TestTSPExact_Small4(t *testing.T) {
	dist := [][]float64{
		{0, 1, 2, 1},
		{1, 0, 1, 2},
		{2, 1, 0, 1},
		{1, 2, 1, 0},
	}
	// Call TSPExact with default options
	res, err := tsp.TSPExact(dist, tsp.DefaultOptions())
	require.NoError(t, err)          // no error expected
	require.Len(t, res.Tour, 5)      // n+1 vertices in tour
	require.Equal(t, 0, res.Tour[0]) // must start at 0
	require.Equal(t, 0, res.Tour[4]) // must end at 0
	require.Equal(t, 4.0, res.Cost)  // exact cost = 4
}

// TestTSPExact_Medium8 verifies Held-Karp on an 8-node cycle.
// Optimum cost == 8.
func TestTSPExact_Medium8(t *testing.T) {
	dist := makeCycleDist(8)
	res, err := tsp.TSPExact(dist, tsp.DefaultOptions())
	require.NoError(t, err)
	require.Len(t, res.Tour, 9) // 8+1 vertices
	require.Equal(t, 0, res.Tour[0])
	require.Equal(t, 0, res.Tour[8])
	require.Equal(t, 8.0, res.Cost) // cost = 8
}

// TestTSPExact_Disconnected ensures ErrIncompleteGraph when the graph
// truly has no Hamiltonian cycle (one vertex is completely isolated).
func TestTSPExact_Disconnected(t *testing.T) {

	const n = 5
	dist := makeCycleDist(n)

	// Isolate vertex 2 by removing all its edges to others
	for v := 0; v < n; v++ {
		if v == 2 {
			continue
		}
		dist[2][v] = math.Inf(1) // no edge 2→v
		dist[v][2] = math.Inf(1) // no edge v→2
	}

	_, err := tsp.TSPExact(dist, tsp.DefaultOptions())
	require.ErrorIs(t, err, tsp.ErrIncompleteGraph)
}

// TestTSPExact_BadInput covers invalid inputs according to specification.
func TestTSPExact_BadInput(t *testing.T) {
	// 1) Empty matrix
	_, err := tsp.TSPExact([][]float64{}, tsp.DefaultOptions())
	require.ErrorIs(t, err, tsp.ErrBadInput)

	// 2) Non-square matrix
	nonSquare := [][]float64{{0, 1}, {1}}
	_, err = tsp.TSPExact(nonSquare, tsp.DefaultOptions())
	require.ErrorIs(t, err, tsp.ErrBadInput)

	// 3) Negative weight
	neg := [][]float64{
		{0, -1, 2},
		{-1, 0, 1},
		{2, 1, 0},
	}
	_, err = tsp.TSPExact(neg, tsp.DefaultOptions())
	require.ErrorIs(t, err, tsp.ErrBadInput)

	// 4) Asymmetric matrix
	asym := [][]float64{
		{0, 1, 2},
		{2, 0, 1}, // dist[1][0] != dist[0][1]
		{2, 1, 0},
	}
	_, err = tsp.TSPExact(asym, tsp.DefaultOptions())
	require.ErrorIs(t, err, tsp.ErrBadInput)
}
