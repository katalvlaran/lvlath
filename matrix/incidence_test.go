// Package matrix_test provides comprehensive unit tests for incidence‐matrix wrappers,
// exercising the 5-stage Blueprint, using lvlath/builder with 8-vertex graphs,
// and verifying key scenarios with table-driven, parallel tests.
package matrix_test

import (
	"fmt"
	"testing"

	"github.com/katalvlaran/lvlath/builder"
	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/matrix"
	"github.com/stretchr/testify/require"
)

// Number of vertices in all test graphs
const V = 8

// Number of edges in an undirected complete graph
const EComplete = V * (V - 1) / 2

// TestIncidence_Blueprint verifies NewIncidenceMatrix follows the 5-stage Blueprint.
func TestIncidence_Blueprint(t *testing.T) {
	t.Parallel()

	// Stage 1 (Validate): nil graph should return ErrGraphNil
	im, err := matrix.NewIncidenceMatrix(nil, matrix.NewMatrixOptions())
	require.Nil(t, im)
	require.ErrorIs(t, err, matrix.ErrGraphNil)

	// Stage 2 (Prepare): build a complete undirected graph of V vertices
	g, err := builder.BuildGraph([]core.GraphOption{core.WithWeighted()}, builder.Complete(V))
	require.NoError(t, err)

	// Stage 3 (Execute): construct incidence matrix with default options
	im, err = matrix.NewIncidenceMatrix(g, matrix.NewMatrixOptions())
	require.NoError(t, err)
	require.NotNil(t, im)

	// Stage 4 (Finalize): verify VertexCount and EdgeCount
	require.Equal(t, V, im.VertexCount(), "vertex count should equal V")
	require.Equal(t, EComplete, im.EdgeCount(), "edge count should equal V*(V-1)/2")
}

// TestVertexIncidence_TableDriven covers Path graph scenarios:
// Undirected vs Directed, verifying per-vertex incidence degrees and signs.
func TestVertexIncidence_TableDriven(t *testing.T) {
	t.Parallel()

	type scenario struct {
		name       string
		coreOpts   []core.GraphOption
		matrixOpts []matrix.Option
		// expected number of non-zero entries per vertex
		wantDeg []int
		// if directed: expected negative count (outgoing) and positive count (incoming) per vertex
		wantNeg, wantPos []int
	}

	tests := []scenario{
		{
			name:       "Undirected_Path",
			coreOpts:   nil,                           // default undirected
			matrixOpts: nil,                           // default collapse parallels, no loops
			wantDeg:    []int{1, 2, 2, 2, 2, 2, 2, 1}, // path endpoints degree 1, internal degree 2
		},
		{
			name:       "Directed_Path",
			coreOpts:   []core.GraphOption{core.WithDirected(true)},
			matrixOpts: []matrix.Option{matrix.WithDirected(true)},
			wantDeg:    []int{1, 2, 2, 2, 2, 2, 2, 1}, // same count but signed
			wantNeg:    []int{1, 1, 1, 1, 1, 1, 1, 0}, // outgoing edges count
			wantPos:    []int{0, 1, 1, 1, 1, 1, 1, 1}, // incoming edges count
		},
	}

	for _, sc := range tests {
		sc := sc // capture
		t.Run(sc.name, func(t *testing.T) {
			t.Parallel()

			// Stage 2 (Prepare): build a path graph of V vertices
			g, err := builder.BuildGraph(sc.coreOpts, builder.Path(V, builder.WithSymbNumb("v")))
			require.NoError(t, err)

			// Stage 3 (Execute): construct incidence matrix
			im, err := matrix.NewIncidenceMatrix(g, matrix.NewMatrixOptions(sc.matrixOpts...))
			require.NoError(t, err)

			// Stage 4 (Finalize): for each vertex, check incidence row
			for i := 0; i < V; i++ {
				id := im.Edges[0].From[:0] // dummy to satisfy lint; will overwrite
				// vertices are named "v0", "v1", ..., "v7"
				id = fmt.Sprintf("v%d", i)
				row, err := im.VertexIncidence(id)
				require.NoError(t, err)

				// count non-zero, negative, and positive entries
				nonZero, neg, pos := 0, 0, 0
				for _, v := range row {
					if v != 0 {
						nonZero++
					}
					if v < 0 {
						neg++
					}
					if v > 0 {
						pos++
					}
				}

				require.Equalf(t, sc.wantDeg[i], nonZero,
					"vertex %q should have %d non-zero incidences", id, sc.wantDeg[i])

				// for directed scenarios, also check sign counts
				if len(sc.wantNeg) > 0 {
					require.Equalf(t, sc.wantNeg[i], neg,
						"vertex %q should have %d negative entries", id, sc.wantNeg[i])
					require.Equalf(t, sc.wantPos[i], pos,
						"vertex %q should have %d positive entries", id, sc.wantPos[i])
				}
			}
		})
	}
}

// TestEdgeEndpoints_Cases covers invalid and valid column index cases.
func TestEdgeEndpoints_Cases(t *testing.T) {
	t.Parallel()

	// Stage 2 (Prepare): build a path graph and its incidence matrix
	g, err := builder.BuildGraph([]core.GraphOption{core.WithWeighted()}, builder.Path(V))
	require.NoError(t, err)
	im, err := matrix.NewIncidenceMatrix(g, matrix.NewMatrixOptions())
	require.NoError(t, err)

	// Stage 3 (Execute & Validate): invalid indices should error
	_, _, err = im.EdgeEndpoints(-1)
	require.ErrorIs(t, err, matrix.ErrDimensionMismatch)
	_, _, err = im.EdgeEndpoints(im.EdgeCount())
	require.ErrorIs(t, err, matrix.ErrDimensionMismatch)

	// Stage 4 (Finalize): valid indices return matching endpoints
	for j := 0; j < im.EdgeCount(); j++ {
		from, to, err := im.EdgeEndpoints(j)
		require.NoError(t, err)
		edge := im.Edges[j]
		require.Equal(t, edge.From, from)
		require.Equal(t, edge.To, to)
	}
}

// TestIncidence_Idempotency ensures repeated NewIncidenceMatrix calls yield identical results.
func TestIncidence_Idempotency(t *testing.T) {
	t.Parallel()

	// Stage 1 (Validate): build baseline path graph
	g, err := builder.BuildGraph([]core.GraphOption{core.WithWeighted()}, builder.Path(V))
	require.NoError(t, err)

	// Stage 2 (Prepare): build two incidence matrices
	im1, err1 := matrix.NewIncidenceMatrix(g, matrix.NewMatrixOptions())
	im2, err2 := matrix.NewIncidenceMatrix(g, matrix.NewMatrixOptions())
	require.NoError(t, err1)
	require.NoError(t, err2)

	// Stage 3 (Execute): compare VertexIndex maps
	require.Equal(t, im1.VertexIndex, im2.VertexIndex)

	// Stage 4 (Finalize): compare Edges slices
	require.Equal(t, im1.Edges, im2.Edges)
}
