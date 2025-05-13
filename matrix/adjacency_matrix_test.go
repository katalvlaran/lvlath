package matrix_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/matrix"
)

func TestAdjacencyMatrix_RoundTripAndOps(t *testing.T) {
	// Build a simple weighted graph: A→B(5), B→C(7)
	g := core.NewGraph(false, true)
	g.AddEdge("A", "B", 5)
	g.AddEdge("B", "C", 7)

	// 1) Round-trip: Graph → AdjacencyMatrix → Graph
	am := matrix.NewAdjacencyMatrix(g)
	g2 := am.ToGraph(true)
	require.True(t, g2.HasEdge("A", "B"))
	require.True(t, g2.HasEdge("B", "C"))

	// 2) AddEdge: C→D(3)
	err := am.AddEdge("C", "D", 3)
	require.NoError(t, err)
	nb, err := am.Neighbors("C")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"B", "D"}, nb)

	// 3) RemoveEdge: A→B
	err = am.RemoveEdge("A", "B")
	require.NoError(t, err)
	nb, err = am.Neighbors("A")
	require.NoError(t, err)
	require.NotContains(t, nb, "B")

	// 4) Unknown-vertex errors
	_, err = am.Neighbors("X")
	require.Error(t, err)
	err = am.AddEdge("X", "Y", 1)
	require.Error(t, err)
}
