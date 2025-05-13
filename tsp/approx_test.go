package tsp_test

import (
	"math"
	"testing"

	"github.com/katalvlaran/lvlath/tsp"
	"github.com/stretchr/testify/require"
)

func TestTSPApprox_Small4(t *testing.T) {
	dist := [][]float64{
		{0, 1, 2, 1},
		{1, 0, 1, 2},
		{2, 1, 0, 1},
		{1, 2, 1, 0},
	}
	res, err := tsp.TSPApprox(dist)
	require.NoError(t, err)
	require.Len(t, res.Tour, 5)
	require.Equal(t, 0, res.Tour[0])
	require.Equal(t, 0, res.Tour[len(res.Tour)-1])
	// Christofides guarantees ≤1.5·opt = 6, but we expect exact here
	require.Equal(t, 4.0, res.Cost)
}

func TestTSPApprox_Medium8(t *testing.T) {
	dist := makeCycleDist(8)
	res, err := tsp.TSPApprox(dist)
	require.NoError(t, err)
	require.Len(t, res.Tour, 9)
	require.Equal(t, 0, res.Tour[0])
	require.Equal(t, 0, res.Tour[len(res.Tour)-1])
	// cost should be the exact 8 on a cycle
	require.Equal(t, 8.0, res.Cost)
}

func TestTSPApprox_Disconnected(t *testing.T) {
	dist := makeCycleDist(6)
	dist[3][4] = math.Inf(1)
	dist[4][3] = math.Inf(1)
	_, err := tsp.TSPApprox(dist)
	require.ErrorIs(t, err, tsp.ErrTSPIncompleteGraph)
}
