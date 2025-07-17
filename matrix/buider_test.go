// Package matrix_test contains unit tests for BuildDenseAdjacency and BuildDenseIncidence
// functions in the matrix package, ensuring compliance with expected behavior
// under various MatrixOptions configurations.
package matrix_test

import (
	"testing"

	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/matrix"

	"github.com/stretchr/testify/require"
)

// TestBuildDenseAdjacency_EmptyVertices validates that empty vertex list triggers ErrInvalidDimensions.
func TestBuildDenseAdjacency_EmptyVertices(t *testing.T) {
	_, _, err := matrix.BuildDenseAdjacency([]string{}, nil, matrix.NewMatrixOptions())
	require.ErrorIs(t, err, matrix.ErrInvalidDimensions)
}

// TestBuildDenseAdjacency_NilEdges ensures nil edges slice is treated as no edges, producing zero matrix.
func TestBuildDenseAdjacency_NilEdges(t *testing.T) {
	vertices := []string{"A", "B"}
	idx, mat, err := matrix.BuildDenseAdjacency(vertices, nil, matrix.NewMatrixOptions())
	require.NoError(t, err)
	require.Len(t, idx, 2) // two vertices

	// All entries should be zero
	iA, iB := idx["A"], idx["B"]
	valAB, _ := mat.At(iA, iB)
	require.Equal(t, 0.0, valAB)
	valBA, _ := mat.At(iB, iA)
	require.Equal(t, 0.0, valBA)
}

// TestBuildDenseAdjacency_DirectedVsUndirected tests correct placement of edge weights.
func TestBuildDenseAdjacency_DirectedVsUndirected(t *testing.T) {
	vertices := []string{"A", "B"}
	edges := []*core.Edge{{From: "A", To: "B", Weight: 5}}

	// Directed, unweighted (default weight=1)
	opts := matrix.NewMatrixOptions(matrix.WithDirected(true))
	idx, mat, err := matrix.BuildDenseAdjacency(vertices, edges, opts)
	require.NoError(t, err)
	iA, iB := idx["A"], idx["B"]
	valAB, _ := mat.At(iA, iB)
	require.Equal(t, 1.0, valAB)
	valBA, _ := mat.At(iB, iA)
	require.Equal(t, 0.0, valBA)

	// Undirected, weighted
	opts = matrix.NewMatrixOptions(matrix.WithWeighted(true))
	idx2, mat2, err := matrix.BuildDenseAdjacency(vertices, edges, opts)
	require.NoError(t, err)
	iA2, iB2 := idx2["A"], idx2["B"]
	valAB2, _ := mat2.At(iA2, iB2)
	require.Equal(t, 5.0, valAB2)
	valBA2, _ := mat2.At(iB2, iA2)
	require.Equal(t, 5.0, valBA2)
}

// TestBuildDenseAdjacency_MultiEdgeCollapse tests AllowMulti option handling.
func TestBuildDenseAdjacency_MultiEdgeCollapse(t *testing.T) {
	vertices := []string{"A", "B"}
	edges := []*core.Edge{
		{From: "A", To: "B", Weight: 2},
		{From: "A", To: "B", Weight: 3},
	}

	// AllowMulti=true (default), weighted: second overwrites first
	opts := matrix.NewMatrixOptions(matrix.WithWeighted(true), matrix.WithAllowMulti(true))
	_, mat, err := matrix.BuildDenseAdjacency(vertices, edges, opts)
	require.NoError(t, err)

	iA, iB := 0, 1
	val, _ := mat.At(iA, iB)
	require.Equal(t, 3.0, val)

	// AllowMulti=false, weighted: first weight only
	opts = matrix.NewMatrixOptions(matrix.WithWeighted(true), matrix.WithAllowMulti(false))
	_, mat2, err := matrix.BuildDenseAdjacency(vertices, edges, opts)
	require.NoError(t, err)
	val2, _ := mat2.At(iA, iB)
	require.Equal(t, 2.0, val2)
}

// TestBuildDenseAdjacency_Loops tests AllowLoops option.
func TestBuildDenseAdjacency_Loops(t *testing.T) {
	vertices := []string{"A"}
	edges := []*core.Edge{{From: "A", To: "A", Weight: 7}}

	// AllowLoops=false (default)
	opts := matrix.NewMatrixOptions()
	idx, mat, err := matrix.BuildDenseAdjacency(vertices, edges, opts)
	require.NoError(t, err)
	val, _ := mat.At(idx["A"], idx["A"])
	require.Equal(t, 0.0, val)

	// AllowLoops=true, weighted
	opts = matrix.NewMatrixOptions(matrix.WithAllowLoops(true), matrix.WithWeighted(true))
	idx2, mat2, err := matrix.BuildDenseAdjacency(vertices, edges, opts)
	require.NoError(t, err)
	val2, _ := mat2.At(idx2["A"], idx2["A"])
	require.Equal(t, 7.0, val2)
}

// TestBuildDenseAdjacency_MetricClosure tests APSP metric closure.
func TestBuildDenseAdjacency_MetricClosure(t *testing.T) {
	vertices := []string{"A", "B", "C"}
	edges := []*core.Edge{
		{From: "A", To: "B", Weight: 1},
		{From: "B", To: "C", Weight: 1},
	}
	opts := matrix.NewMatrixOptions(matrix.WithWeighted(true), matrix.WithMetricClosure(true))
	idx, mat, err := matrix.BuildDenseAdjacency(vertices, edges, opts)
	require.NoError(t, err)
	iA, iC := idx["A"], idx["C"]
	// shortest path A->C is 2
	val, _ := mat.At(iA, iC)
	require.Equal(t, float64(2), val)
}

// TestBuildDenseIncidence_EmptyVertices validates ErrInvalidDimensions for zero vertices.
func TestBuildDenseIncidence_EmptyVertices(t *testing.T) {
	_, _, _, err := matrix.BuildDenseIncidence([]string{}, nil, matrix.NewMatrixOptions())
	require.ErrorIs(t, err, matrix.ErrInvalidDimensions)
}

// TestBuildDenseIncidence_NilEdges ensures nil edges yields zero-column matrix.
func TestBuildDenseIncidence_NilEdges(t *testing.T) {
	vertices := []string{"A", "B"}
	idx, cols, mat, err := matrix.BuildDenseIncidence(vertices, nil, matrix.NewMatrixOptions())
	require.NoError(t, err)
	require.Len(t, idx, 2)
	require.Empty(t, cols)
	require.Equal(t, 0, mat.Cols())
}

// TestBuildDenseIncidence_DirectedVsUndirected tests incidence entries for directed and undirected.
func TestBuildDenseIncidence_DirectedVsUndirected(t *testing.T) {
	vertices := []string{"A", "B"}
	edges := []*core.Edge{{From: "A", To: "B", Weight: 0}}

	// Directed
	opts := matrix.NewMatrixOptions(matrix.WithDirected(true))
	idxD, colsD, matD, err := matrix.BuildDenseIncidence(vertices, edges, opts)
	require.NoError(t, err)
	require.Len(t, colsD, 1)
	iA, iB := idxD["A"], idxD["B"]
	neg, _ := matD.At(iA, 0)
	pos, _ := matD.At(iB, 0)
	require.Equal(t, -1.0, neg)
	require.Equal(t, 1.0, pos)

	// Undirected
	opts = matrix.NewMatrixOptions()
	idxU, _, matU, err := matrix.BuildDenseIncidence(vertices, edges, opts)
	require.NoError(t, err)
	posA, _ := matU.At(idxU["A"], 0)
	posB, _ := matU.At(idxU["B"], 0)
	require.Equal(t, 1.0, posA)
	require.Equal(t, 1.0, posB)
}

// TestBuildDenseIncidence_MultiEdgeCollapse tests collapse behavior for incidence.
func TestBuildDenseIncidence_MultiEdgeCollapse(t *testing.T) {
	vertices := []string{"A", "B"}
	edges := []*core.Edge{
		{From: "A", To: "B", Weight: 0},
		{From: "A", To: "B", Weight: 0},
	}

	// AllowMulti=true (default)
	opts := matrix.NewMatrixOptions(matrix.WithAllowMulti(true))
	_, cols, _, err := matrix.BuildDenseIncidence(vertices, edges, opts)
	require.NoError(t, err)
	require.Len(t, cols, 2)

	// AllowMulti=false
	opts = matrix.NewMatrixOptions(matrix.WithAllowMulti(false))
	_, cols2, _, err := matrix.BuildDenseIncidence(vertices, edges, opts)
	require.NoError(t, err)
	require.Len(t, cols2, 1)
}

// TestBuildDenseIncidence_Loops tests AllowLoops option for incidence.
func TestBuildDenseIncidence_Loops(t *testing.T) {
	vertices := []string{"A"}
	edges := []*core.Edge{{From: "A", To: "A", Weight: 0}}

	// AllowLoops=false
	opts := matrix.NewMatrixOptions()
	_, cols, _, err := matrix.BuildDenseIncidence(vertices, edges, opts)
	require.NoError(t, err)
	require.Empty(t, cols)

	// AllowLoops=true
	opts = matrix.NewMatrixOptions(matrix.WithAllowLoops(true))
	_, cols2, mat2, err := matrix.BuildDenseIncidence(vertices, edges, opts)
	require.NoError(t, err)
	require.Len(t, cols2, 1)
	val, _ := mat2.At(0, 0)
	require.Equal(t, 1.0, val)
}
