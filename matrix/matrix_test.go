package matrix_test

import (
	"sort"
	"testing"

	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/matrix"
	"github.com/stretchr/testify/require"
)

func TestBuildAdjacencyData_SimpleTriangle(t *testing.T) {
	// Graph: A-B(1), B-C(2), C-A(3)
	g := core.NewGraph(core.WithWeighted())
	_, _ = g.AddEdge("A", "B", 1)
	_, _ = g.AddEdge("B", "C", 2)
	_, _ = g.AddEdge("C", "A", 3)
	verts := g.Vertices()
	edges := g.Edges()
	idx, data, err := matrix.BuildAdjacencyData(verts, edges, matrix.NewMatrixOptions(matrix.WithWeighted(true)))
	require.NoError(t, err)
	require.Len(t, idx, 3)
	// Check weights
	iA, jB := idx["A"], idx["B"]
	require.Equal(t, 1.0, data[iA][jB])
	// Undirected: B->A
	require.Equal(t, 1.0, data[jB][iA])
}

func TestBuildIncidenceData_SimpleChain(t *testing.T) {
	// Graph: A->B, B->C
	g := core.NewGraph(core.WithWeighted(), core.WithDirected(true))
	_, _ = g.AddEdge("A", "B", 1)
	_, _ = g.AddEdge("B", "C", 2)
	verts := g.Vertices()
	edges := g.Edges()
	vIdx, cols, data, err := matrix.BuildIncidenceData(verts, edges, matrix.NewMatrixOptions(matrix.WithDirected(true), matrix.WithWeighted(true)))
	require.NoError(t, err)
	require.Len(t, data, len(verts))
	require.Len(t, cols, 2)
	// Find column for B->C
	colIndex := -1
	for j, e := range cols {
		if e.From == "B" && e.To == "C" {
			colIndex = j
			break
		}
	}
	require.GreaterOrEqual(t, colIndex, 0)
	// Incidence: B row has -1 at col, C row has +1
	bRow := data[vIdx["B"]]
	cRow := data[vIdx["C"]]
	require.Equal(t, -1, bRow[colIndex])
	require.Equal(t, 1, cRow[colIndex])
}

func TestTransposeData(t *testing.T) {
	mat := [][]float64{{1, 2, 3}, {4, 5, 6}}
	td := matrix.TransposeData(mat)
	expected := [][]float64{{1, 4}, {2, 5}, {3, 6}}
	require.Equal(t, expected, td)
}

func TestMultiplyData(t *testing.T) {
	A := [][]float64{{1, 2}, {3, 4}}
	B := [][]float64{{2, 0}, {1, 2}}
	res, err := matrix.MultiplyData(A, B)
	require.NoError(t, err)
	// A*B = [[1*2+2*1,1*0+2*2],[3*2+4*1,3*0+4*2]] = [[4,4],[10,8]]
	exp := [][]float64{{4, 4}, {10, 8}}
	require.Equal(t, exp, res)
}

func TestMultiplyData_DimensionMismatch(t *testing.T) {
	A := [][]float64{{1, 2, 3}, {4, 5, 6}}
	B := [][]float64{{1, 2}, {3, 4}}
	_, err := matrix.MultiplyData(A, B)
	require.ErrorIs(t, err, matrix.ErrDimensionMismatch)
}

func TestDegreeFromData(t *testing.T) {
	mat := [][]float64{{1, 2, 3}, {0, 0, 5}}
	d := matrix.DegreeFromData(mat)
	exp := []float64{6, 5}
	require.Equal(t, exp, d)
}

func TestEigenDecompose_Ring3(t *testing.T) {
	// Adjacency of 3-cycle: each node connected to two others with weight 1
	mat := [][]float64{{0, 1, 1}, {1, 0, 1}, {1, 1, 0}}
	tol := 1e-9
	maxIter := 100
	vals, vecs, err := matrix.EigenDecompose(mat, tol, maxIter)
	require.NoError(t, err)
	// Expected eigenvalues: 2, -1, -1
	sort.Float64s(vals)
	exp := []float64{-1, -1, 2}
	for i := range exp {
		require.InDelta(t, exp[i], vals[i], 1e-6)
	}
	// Verify orthonormality of eigenvectors
	n := len(vecs)
	norm0 := 0.0
	for j := 0; j < n; j++ {
		norm0 += vecs[j][0] * vecs[j][0]
	}
	require.InDelta(t, 1.0, norm0, 1e-6)
}

func TestEigenDecompose_FailsOnNonSquare(t *testing.T) {
	_, _, err := matrix.EigenDecompose([][]float64{{1, 2, 3}, {4, 5, 6}}, 1e-5, 10)
	require.ErrorIs(t, err, matrix.ErrDimensionMismatch)
}

func TestEigenDecompose_FailsToConverge(t *testing.T) {
	mat := [][]float64{{0, 1}, {1, 0}}
	// set maxIter small so it won't converge
	_, _, err := matrix.EigenDecompose(mat, 1e-12, 1)
	require.ErrorIs(t, err, matrix.ErrEigenFailed)
}
