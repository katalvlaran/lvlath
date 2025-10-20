// Package matrix_test provides comprehensive unit tests for adjacency‐matrix wrappers,
// exercising the 5-stage Blueprint, using lvlath/builder with 8-vertex graphs,
// and verifying all key scenarios with table-driven, parallel tests.
package matrix_test

import (
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/katalvlaran/lvlath/builder"
	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/matrix"
	"github.com/stretchr/testify/require"
)

// TestAdjacency_Blueprint verifies NewAdjacencyMatrix follows the 5-stage Blueprint.
func TestAdjacency_Blueprint(t *testing.T) {
	t.Parallel()
	// Stage 1 (Validate): nil graph should return ErrNilGraph
	am, err := matrix.NewAdjacencyMatrix(nil, matrix.NewMatrixOptions())
	require.Nil(t, am)
	require.ErrorIs(t, err, matrix.ErrGraphNil)

	// Stage 2 (Prepare): build a complete graph of V vertices with weighted, multi-edge, loops
	g, err := builder.BuildGraph(
		[]core.GraphOption{
			core.WithWeighted(), core.WithMultiEdges(), core.WithLoops(),
		},
		[]builder.BuilderOption{},
		builder.Complete(V),
	)
	//g, err := builder.BuildGraph(
	//	[]core.GraphOption{core.WithDirected(false), core.WithWeighted(), core.WithLoops(), core.WithMultiEdges()},
	//	[]builder.BuilderOption{builder.WithSymbNumb("v")}, // если доступна; иначе удалить и зафиксировать нейминг helper’ом
	//	builder.Complete(V),
	//)
	require.NoError(t, err)

	// Stage 3 (Execute): construct adjacency matrix with matching options
	opts := matrix.NewMatrixOptions(
		matrix.WithWeighted(),
		matrix.WithAllowMulti(),
		matrix.WithAllowLoops(),
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
		matrixOpts []matrix.Option
		wantCount  int
	}

	tests := []scenario{
		{
			name:       "Undirected_Unweighted",
			graphOpts:  []core.GraphOption{},
			matrixOpts: []matrix.Option{}, // defaults: undirected, unweighted, collapse parallels, no loops
			wantCount:  V - 1,             // each vertex connects to all others
		},
		{
			name:      "Directed_Weighted",
			graphOpts: []core.GraphOption{core.WithDirected(true), core.WithWeighted()},
			matrixOpts: []matrix.Option{
				matrix.WithDirected(),
				matrix.WithWeighted(),
			},
			wantCount: V - 1, // Complete(V) has edge in both directions
		},
		{
			name:      "WithLoops",
			graphOpts: []core.GraphOption{core.WithLoops()},
			matrixOpts: []matrix.Option{
				matrix.WithAllowLoops(),
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
			g, err := builder.BuildGraph(sc.graphOpts, []builder.BuilderOption{builder.WithSymbNumb("v")}, builder.Complete(V))
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
		[]builder.BuilderOption{},
		builder.Complete(V),
	)
	require.NoError(t, err)

	// Stage 2 (Prepare): build adjacency matrix
	opts := matrix.NewMatrixOptions(matrix.WithDirected(), matrix.WithWeighted())
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
	g, err := builder.BuildGraph([]core.GraphOption{core.WithWeighted()}, []builder.BuilderOption{}, builder.Complete(V))
	require.NoError(t, err)

	// Stage 2 (Prepare): build two adjacency matrices
	opts := matrix.NewMatrixOptions(matrix.WithWeighted())
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
	g, err := builder.BuildGraph([]core.GraphOption{core.WithWeighted()}, []builder.BuilderOption{}, builder.Complete(V))
	require.NoError(t, err)
	am, err := matrix.NewAdjacencyMatrix(g, matrix.NewMatrixOptions())
	require.NoError(t, err)

	// Stage 2 (Execute & Validate): unknown vertex
	_, err = am.Neighbors("unknown")
	require.Error(t, err)
	require.True(t, errors.Is(err, matrix.ErrUnknownVertex))

	// Stage 3 (Finalize): VertexCount on nil receiver panics
	var nilAM *matrix.AdjacencyMatrix
	require.Panics(t, func() { _ = nilAM.VertexCount() })
}

// helper: build n×n Dense filled with +Inf off-diag and 0 on the diagonal
// then apply a list of edges (i,j,val). Keeps deterministic row-major layout.
func buildInfAdj(n int, edges [][3]float64) *matrix.Dense {
	d, _ := matrix.NewDense(n, n)
	// fill with +Inf off-diag, 0 on diag
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i == j {
				_ = d.Set(i, j, 0)
			} else {
				_ = d.Set(i, j, math.Inf(+1))
			}
		}
	}
	// set provided edges (weight as given)
	for _, e := range edges {
		i, j, w := int(e[0]), int(e[1]), e[2]
		_ = d.Set(i, j, w)
	}
	return d
}

// helper: wrap Dense into a minimal AdjacencyMatrix for tests with stable order.
func wrapAdjForTest(d *matrix.Dense, directed, allowLoops bool) *matrix.AdjacencyMatrix {
	n := d.Rows()
	idx := make(map[string]int, n)
	rev := make([]string, n)
	for i := 0; i < n; i++ {
		id := string(rune('A' + i)) // A,B,C,... deterministic ids
		idx[id] = i
		rev[i] = id
	}
	return &matrix.AdjacencyMatrix{
		Mat:           d,
		VertexIndex:   idx,
		vertexByIndex: rev,
		opts: matrix.Options{
			directed:   directed,
			weighted:   true,
			allowMulti: false,
			allowLoops: allowLoops,
		},
	}
}

func almostEqualSlice(a, b []float64, eps float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if math.Abs(a[i]-b[i]) > eps {
			return false
		}
	}
	return true
}

func TestDegreeVector_Directed_Unweighted(t *testing.T) {
	// Graph: 4 vertices: A,B,C,D
	// Edges: A→B, A→C, B→C, C→C(loop), no edges from D
	// All weights = 1; +Inf = no edge; loop counts as 1.
	d := buildInfAdj(4, [][3]float64{
		{0, 1, 1}, // A→B
		{0, 2, 1}, // A→C
		{1, 2, 1}, // B→C
		{2, 2, 1}, // C→C (loop)
	})
	am := wrapAdjForTest(d, true, true)

	got, err := am.DegreeVector()
	if err != nil {
		t.Fatalf("DegreeVector error: %v", err)
	}
	// A: 2 (B,C); B:1 (C); C:1 (loop only ⇒ counts as 1); D:0
	want := []float64{2, 1, 1, 0}
	if !almostEqualSlice(got, want, 1e-12) {
		t.Fatalf("DegreeVector mismatch.\n got: %v\nwant: %v", got, want)
	}
}

func TestDegreeVector_Undirected_Unweighted(t *testing.T) {
	// Undirected: edges mirrored in adjacency.
	// A—B, B—C; degrees: deg(A)=1, deg(B)=2, deg(C)=1, deg(D)=0
	d := buildInfAdj(4, [][3]float64{
		{0, 1, 1}, {1, 0, 1}, // A—B
		{1, 2, 1}, {2, 1, 1}, // B—C
	})
	am := wrapAdjForTest(d, false, false)

	got, err := am.DegreeVector()
	if err != nil {
		t.Fatalf("DegreeVector error: %v", err)
	}
	want := []float64{1, 2, 1, 0}
	if !almostEqualSlice(got, want, 1e-12) {
		t.Fatalf("DegreeVector mismatch.\n got: %v\nwant: %v", got, want)
	}
}

func TestDegreeVector_LoopWeightedCountsAsOne(t *testing.T) {
	// Single vertex with a heavy loop (weight 7) must count as exactly 1.
	d := buildInfAdj(1, [][3]float64{
		{0, 0, 7}, // loop weight should not matter
	})
	am := wrapAdjForTest(d, false, true)

	got, err := am.DegreeVector()
	if err != nil {
		t.Fatalf("DegreeVector error: %v", err)
	}
	want := []float64{1}
	if !almostEqualSlice(got, want, 1e-12) {
		t.Fatalf("DegreeVector mismatch.\n got: %v\nwant: %v", got, want)
	}
}
