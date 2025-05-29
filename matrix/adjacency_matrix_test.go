package matrix_test

import (
	"testing"

	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/matrix"
	"github.com/stretchr/testify/require"
)

func TestAdjacencyMatrix_RoundTrip(t *testing.T) {
	// Build a simple weighted graph: A→B(5), B→C(7)
	g := core.NewGraph(core.WithWeighted())
	_, _ = g.AddEdge("A", "B", 5)
	_, _ = g.AddEdge("B", "C", 7)

	opts := matrix.NewMatrixOptions(matrix.WithWeighted(true), matrix.WithDirected(true))
	// Directed matrix: each edge produces a single non-zero entry
	am, err := matrix.NewAdjacencyMatrix(g, opts)
	require.NoError(t, err)

	g2, err := am.ToGraph()
	require.NoError(t, err)
	require.True(t, g2.HasEdge("A", "B"))
	require.True(t, g2.HasEdge("B", "C"))

	// VertexCount and EdgeCount
	reqVerts := am.VertexCount()
	expVerts := len(g.Vertices())
	require.Equal(t, expVerts, reqVerts)

	reqEdges := am.EdgeCount()
	// Directed graph has 2 edges
	require.Equal(t, 2, reqEdges)
}

func TestAdjacencyMatrix_TransposeMultiply(t *testing.T) {
	// Graph: undirected triangle A-B, B-C, C-A weight 1
	g := core.NewGraph(core.WithWeighted())
	_, _ = g.AddEdge("A", "B", 1)
	_, _ = g.AddEdge("B", "C", 1)
	_, _ = g.AddEdge("C", "A", 1)

	opts := matrix.NewMatrixOptions(matrix.WithWeighted(true))
	am, err := matrix.NewAdjacencyMatrix(g, opts)
	require.NoError(t, err)

	// Transpose equals original for symmetric
	amT := am.Transpose()
	require.Equal(t, am.Data, amT.Data)

	// Multiply A*A should yield A^2
	prod, err := am.Multiply(am)
	require.NoError(t, err)
	// In triangle each vertex has 2 two-step paths to itself
	n := am.VertexCount()
	for i := 0; i < n; i++ {
		require.Equal(t, 2.0, prod.Data[i][i])
	}
}

func TestAdjacencyMatrix_ErrorCases(t *testing.T) {
	// nil graph
	_, err := matrix.NewAdjacencyMatrix(nil, matrix.NewMatrixOptions())
	require.ErrorIs(t, err, matrix.ErrNilGraph)

	// Dimension mismatch in Multiply
	g := core.NewGraph()
	am, err := matrix.NewAdjacencyMatrix(g, matrix.NewMatrixOptions())
	require.NoError(t, err)
	_ = g.AddVertex("X")
	am2, err := matrix.NewAdjacencyMatrix(g, matrix.NewMatrixOptions())
	require.NoError(t, err)
	_, err = am.Multiply(am2)
	require.ErrorIs(t, err, matrix.ErrDimensionMismatch)
}
