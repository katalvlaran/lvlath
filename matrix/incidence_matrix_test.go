package matrix_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/matrix"
)

func TestIncidenceMatrixBasic(t *testing.T) {
	// Build a directed graph A→B(1), B→C(2)
	g := core.NewGraph(true, true)
	g.AddEdge("A", "B", 1)
	g.AddEdge("B", "C", 2)

	m := matrix.NewIncidenceMatrix(g)

	// Dimensions: 3 vertices × 2 edges
	require.Len(t, m.Data, 3)
	require.Len(t, m.Data[0], 2)

	// EdgeEndpoints
	from, to, err := m.EdgeEndpoints(1)
	require.NoError(t, err)
	require.Equal(t, "B", from)
	require.Equal(t, "C", to)

	// VertexIncidence for "B" contains exactly one -1 and one +1
	row, err := m.VertexIncidence("B")
	require.NoError(t, err)
	require.Contains(t, row, -1)
	require.Contains(t, row, 1)

	// Unknown vertex / edge errors
	_, err = m.VertexIncidence("X")
	require.Error(t, err)
	_, _, err = m.EdgeEndpoints(5)
	require.Error(t, err)
}
