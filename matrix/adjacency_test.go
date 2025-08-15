// Package matrix_test provides comprehensive unit tests for adjacency‐matrix wrappers,
// exercising the 5-stage Blueprint, using lvlath/builder with 8-vertex graphs,
// and verifying all key scenarios with table-driven, parallel tests.
package matrix_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/katalvlaran/lvlath/builder"
	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/matrix"
	"github.com/stretchr/testify/require"
)

// TestAdjacency_Blueprint verifies NewAdjacencyMatrix follows the 5-stage Blueprint.
func TestAdjacency_Blueprint(t *testing.T) {
	t.Parallel()
	// Stage 1 (Validate): nil graph should return ErrMatrixNilGraph
	am, err := matrix.NewAdjacencyMatrix(nil, matrix.NewMatrixOptions())
	require.Nil(t, am)
	require.ErrorIs(t, err, matrix.ErrMatrixNilGraph)

	// Stage 2 (Prepare): build a complete graph of V vertices with weighted, multi-edge, loops
	g, err := builder.BuildGraph(
		[]core.GraphOption{
			core.WithWeighted(), core.WithMultiEdges(), core.WithLoops(),
		},
		builder.Complete(V),
	)
	require.NoError(t, err)

	// Stage 3 (Execute): construct adjacency matrix with matching options
	opts := matrix.NewMatrixOptions(
		matrix.WithWeighted(true),
		matrix.WithAllowMulti(true),
		matrix.WithAllowLoops(true),
	)
	am, err = matrix.NewAdjacencyMatrix(g, opts)
	require.NoError(t, err)
	require.NotNil(t, am)

	// Stage 4 (Finalize): verify VertexCount matches V
	require.Equal(t, V, am.VertexCount())
}

// TestNeighbors_TableDriven covers Directed/Undirected, Weighted/Unweighted,
// Multi-edge collapse, and Loops scenarios using Complete graph.
func TestNeighbors_TableDriven(t *testing.T) {
	t.Parallel()

	type scenario struct {
		name       string
		graphOpts  []core.GraphOption
		matrixOpts []matrix.MatrixOption
		wantCount  int
	}

	tests := []scenario{
		{
			name:       "Undirected_Unweighted",
			graphOpts:  []core.GraphOption{},
			matrixOpts: []matrix.MatrixOption{}, // defaults: undirected, unweighted, collapse parallels, no loops
			wantCount:  V - 1,                   // each vertex connects to all others
		},
		{
			name:      "Directed_Weighted",
			graphOpts: []core.GraphOption{core.WithDirected(true), core.WithWeighted()},
			matrixOpts: []matrix.MatrixOption{
				matrix.WithDirected(true),
				matrix.WithWeighted(true),
			},
			wantCount: V - 1, // Complete(V) has edge in both directions
		},
		{
			name:      "WithLoops",
			graphOpts: []core.GraphOption{core.WithLoops()},
			matrixOpts: []matrix.MatrixOption{
				matrix.WithAllowLoops(true),
			},
			wantCount: V - 1, // self-loop plus V-1 neighbors
		},
	}

	for _, sc := range tests {
		sc := sc // capture
		t.Run(sc.name, func(t *testing.T) {
			t.Parallel()
			// Stage 1 (Validate): nothing to validate here
			// Stage 2 (Prepare): build graph per scenario
			g, err := builder.BuildGraph(sc.graphOpts, builder.Complete(V, builder.WithSymbNumb("v")))
			require.NoError(t, err)

			// Stage 3 (Execute): build adjacency matrix
			am, err := matrix.NewAdjacencyMatrix(g, matrix.NewMatrixOptions(sc.matrixOpts...))
			require.NoError(t, err)

			// Stage 4 (Finalize): pick a representative vertex and get neighbors
			u := fmt.Sprintf("v%d", 0)
			neighbors, err := am.Neighbors(u)
			require.NoError(t, err)
			require.Len(t, neighbors, sc.wantCount)
			// ensure no unknown vertices
			for _, v := range neighbors {
				_, ok := am.VertexIndex[v]
				require.True(t, ok, "neighbor %q must be valid vertex", v)
			}
		})
	}
}

// TestToGraph_RoundTrip ensures ToGraph reconstructs the original Graph.
func TestToGraph_RoundTrip(t *testing.T) {
	t.Parallel()
	// Stage 1 (Validate): build original complete, directed, weighted graph
	orig, err := builder.BuildGraph(
		[]core.GraphOption{core.WithDirected(true), core.WithWeighted()},
		builder.Complete(V),
	)
	require.NoError(t, err)

	// Stage 2 (Prepare): build adjacency matrix
	opts := matrix.NewMatrixOptions(matrix.WithDirected(true), matrix.WithWeighted(true))
	am, err := matrix.NewAdjacencyMatrix(orig, opts)
	require.NoError(t, err)

	// Stage 3 (Execute): reconstruct graph
	g2, err := am.ToGraph()
	require.NoError(t, err)

	// Stage 4 (Finalize): compare vertex and edge counts
	require.Equal(t, len(orig.Vertices()), len(g2.Vertices()))
	require.Equal(t, len(orig.Edges()), len(g2.Edges()))
}

// TestAdjacency_Idempotency ensures repeated NewAdjacencyMatrix calls yield identical matrices.
func TestAdjacency_Idempotency(t *testing.T) {
	t.Parallel()
	// Stage 1 (Validate): build baseline graph
	g, err := builder.BuildGraph([]core.GraphOption{core.WithWeighted()}, builder.Complete(V))
	require.NoError(t, err)

	// Stage 2 (Prepare): build two adjacency matrices
	opts := matrix.NewMatrixOptions(matrix.WithWeighted(true))
	am1, err1 := matrix.NewAdjacencyMatrix(g, opts)
	am2, err2 := matrix.NewAdjacencyMatrix(g, opts)
	require.NoError(t, err1)
	require.NoError(t, err2)

	// Stage 3 (Execute): compare indices
	require.Equal(t, am1.VertexIndex, am2.VertexIndex)

	// Stage 4 (Finalize): compare every cell
	n := am1.VertexCount()
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			v1, _ := am1.Mat.At(i, j)
			v2, _ := am2.Mat.At(i, j)
			require.Equal(t, v1, v2, "cell (%d,%d) mismatch", i, j)
		}
	}
}

// TestNeighbors_ErrorCases covers unknown-vertex error and VertexCount panic.
func TestNeighbors_ErrorCases(t *testing.T) {
	t.Parallel()
	// Stage 1 (Prepare): build default graph
	g, err := builder.BuildGraph([]core.GraphOption{core.WithWeighted()}, builder.Complete(V))
	require.NoError(t, err)
	am, err := matrix.NewAdjacencyMatrix(g, matrix.NewMatrixOptions())
	require.NoError(t, err)

	// Stage 2 (Execute & Validate): unknown vertex
	_, err = am.Neighbors("unknown")
	require.Error(t, err)
	require.True(t, errors.Is(err, matrix.ErrMatrixUnknownVertex))

	// Stage 3 (Finalize): VertexCount on nil receiver panics
	var nilAM *matrix.AdjacencyMatrix
	require.Panics(t, func() { _ = nilAM.VertexCount() })
}
