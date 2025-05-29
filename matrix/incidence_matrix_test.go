package matrix_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/matrix"
)

// TestIncidenceMatrixBasic covers basic integration scenarios for IncidenceMatrix.
func TestIncidenceMatrixBasic(t *testing.T) {
	// Build a directed, weighted graph A→B(1), B→C(2)
	g := core.NewGraph(core.WithDirected(true), core.WithWeighted())
	_, _ = g.AddEdge("A", "B", 1)
	_, _ = g.AddEdge("B", "C", 2)

	opts := matrix.NewMatrixOptions(
		matrix.WithDirected(true),
		matrix.WithWeighted(true),
	)
	m, err := matrix.NewIncidenceMatrix(g, opts)
	require.NoError(t, err)

	// Dimensions: V rows × E columns
	expectedVerts := len(g.Vertices())
	require.Equal(t, expectedVerts, m.VertexCount())
	require.Equal(t, 2, m.EdgeCount())

	// EdgeEndpoints
	from, to, err := m.EdgeEndpoints(1)
	require.NoError(t, err)
	require.Equal(t, "B", from)
	require.Equal(t, "C", to)

	// VertexIncidence for "B": contains exactly one -1 and one +1
	row, err := m.VertexIncidence("B")
	require.NoError(t, err)
	countNeg1, countPos1 := 0, 0
	for _, v := range row {
		switch v {
		case -1:
			countNeg1++
		case +1:
			countPos1++
		}
	}
	require.Equal(t, 1, countNeg1)
	require.Equal(t, 1, countPos1)

	// Unknown vertex error
	_, err = m.VertexIncidence("X")
	require.ErrorIs(t, err, matrix.ErrUnknownVertex)

	// EdgeEndpoints out-of-range error
	_, _, err = m.EdgeEndpoints(5)
	require.ErrorIs(t, err, matrix.ErrDimensionMismatch)
}
