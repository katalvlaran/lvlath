package tsp_test

import (
	"math"
	"testing"

	"github.com/katalvlaran/lvlath/tsp"
	"github.com/stretchr/testify/require"
)

func makeCycleDist(n int) [][]float64 {
	// distances along a cycle: dist(i,j)=min(|i-j|, n-|i-j|)
	dist := make([][]float64, n)
	for i := range dist {
		dist[i] = make([]float64, n)
		for j := range dist {
			d := math.Abs(float64(i - j))
			dist[i][j] = math.Min(d, float64(n)-d)
		}
	}
	return dist
}

func TestTSPExact_Small4(t *testing.T) {
	// 4‐node cycle distances; optimum cycle cost = 4
	dist := [][]float64{
		{0, 1, 2, 1},
		{1, 0, 1, 2},
		{2, 1, 0, 1},
		{1, 2, 1, 0},
	}
	res, err := tsp.TSPExact(dist)
	require.NoError(t, err)
	// must start and end at 0, length = n+1
	require.Len(t, res.Tour, 5)
	require.Equal(t, 0, res.Tour[0])
	require.Equal(t, 0, res.Tour[len(res.Tour)-1])
	// cost = 4 exactly
	require.Equal(t, 4.0, res.Cost)
}

func TestTSPExact_Medium8(t *testing.T) {
	// 8‐node cycle; optimum cycle cost = 8
	dist := makeCycleDist(8)
	res, err := tsp.TSPExact(dist)
	require.NoError(t, err)
	require.Len(t, res.Tour, 9)
	require.Equal(t, 0, res.Tour[0])
	require.Equal(t, 0, res.Tour[len(res.Tour)-1])
	require.Equal(t, 8.0, res.Cost)
}

func TestTSPExact_Disconnected(t *testing.T) {
	// introduce an infinite distance to break connectivity
	dist := makeCycleDist(5)
	dist[1][2] = math.Inf(1)
	dist[2][1] = math.Inf(1)
	_, err := tsp.TSPExact(dist)
	require.ErrorIs(t, err, tsp.ErrTSPIncompleteGraph)
}
