// Package matrix_test provides GoDoc examples for lvlath/matrix,
// demonstrating common adjacency‐matrix workflows.
package matrix_test

import (
	"fmt"

	"github.com/katalvlaran/lvlath/builder"
	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/matrix"
)

// ExampleAdjacencyWorkflow shows how to build a graph, compute all‐pairs shortest paths,
// and reconstruct the graph via adjacency‐matrix round‐trip.
func ExampleAdjacencyWorkflow() {
	// (Prepare): build a directed, weighted, loop‐enabled complete graph
	g, _ := builder.BuildGraph(
		[]core.GraphOption{
			core.WithDirected(true), // treat edges as directed
			core.WithWeighted(),     // preserve weights
			core.WithLoops(),        // allow self‐loops
			core.WithMultiEdges(),   // allow parallel edges
		},
		builder.Complete(V), // complete graph on V vertices
	)

	// (Execute): construct adjacency matrix with metric closure enabled
	opts := matrix.NewMatrixOptions(
		matrix.WithDirected(true),
		matrix.WithWeighted(true),
		matrix.WithAllowLoops(true),
		matrix.WithAllowMulti(true),
		matrix.WithMetricClosure(true), // fill missing edges with +Inf and run APSP
	)
	am, _ := matrix.NewAdjacencyMatrix(g, opts)

	// (Execute): run Floyd–Warshall on underlying matrix to finalize distances
	_ = matrix.FloydWarshall(am.Mat)

	// (Finalize): reconstruct graph and display vertex & edge counts
	g2, _ := am.ToGraph()
	fmt.Printf("Vertices: %d, Edges: %d\n", len(g2.Vertices()), len(g2.Edges()))

	// Output:
	// Vertices: 8, Edges: 56
}

// ExampleIncidenceWorkflow builds a simple path graph, inspects per-vertex incidence,
// and prints each edge’s endpoints.
func ExampleIncidenceWorkflow() {
	const V = 5

	// (Prepare): build a directed, weighted path on V vertices
	g, _ := builder.BuildGraph(
		[]core.GraphOption{core.WithWeighted(), core.WithDirected(true)},
		builder.Path(V),
	)

	// (Execute): build its incidence matrix (directed as our graph)
	im, _ := matrix.NewIncidenceMatrix(g, matrix.NewMatrixOptions(matrix.WithDirected(true)))

	// (Finalize): print each vertex’s incidence vector using the actual IDs
	fmt.Println("VertexIncidence vectors:")
	for _, id := range g.Vertices() {
		vec, _ := im.VertexIncidence(id)
		fmt.Printf("  %s: %v\n", id, vec)
	}

	// (Finalize): print each edge’s endpoints
	fmt.Println("EdgeEndpoints list:")
	for j := 0; j < im.EdgeCount(); j++ {
		from, to, _ := im.EdgeEndpoints(j)
		fmt.Printf("  edge %d: %s→%s\n", j, from, to)
	}

	// Output:
	// VertexIncidence vectors:
	//   0: [-1 0 0 0]
	//   1: [1 -1 0 0]
	//   2: [0 1 -1 0]
	//   3: [0 0 1 -1]
	//   4: [0 0 0 1]
	// EdgeEndpoints list:
	//   edge 0: 0→1
	//   edge 1: 1→2
	//   edge 2: 2→3
	//   edge 3: 3→4
}

// ExampleMatrixMethods demonstrates Add, Sub, Mul, Transpose, and Scale on small data.
func ExampleMatrixMethods() {
	// 1) Create two 2×2 matrices
	a, _ := matrix.NewDense(2, 2)
	b, _ := matrix.NewDense(2, 2)
	_ = a.Set(0, 0, 1)
	_ = a.Set(0, 1, 2)
	_ = a.Set(1, 0, 3)
	_ = a.Set(1, 1, 4)
	_ = b.Set(0, 0, 5)
	_ = b.Set(0, 1, 6)
	_ = b.Set(1, 0, 7)
	_ = b.Set(1, 1, 8)

	// 2) Demonstrate Add
	sum, _ := matrix.Add(a, b)
	v, _ := sum.At(1, 1)
	fmt.Println("sum[1,1] =", v)

	// 3) Demonstrate Mul for a 2×3 × 3×2
	m, _ := matrix.NewDense(2, 3)
	n, _ := matrix.NewDense(3, 2)
	_ = m.Set(0, 0, 1)
	_ = m.Set(0, 1, 2)
	_ = m.Set(0, 2, 3)
	_ = m.Set(1, 0, 4)
	_ = m.Set(1, 1, 5)
	_ = m.Set(1, 2, 6)
	_ = n.Set(0, 0, 7)
	_ = n.Set(0, 1, 8)
	_ = n.Set(1, 0, 9)
	_ = n.Set(1, 1, 10)
	_ = n.Set(2, 0, 11)
	_ = n.Set(2, 1, 12)
	prod, _ := matrix.Mul(m, n)
	v, _ = prod.At(1, 0)
	fmt.Println("prod[1,0] =", v)

	// 4) Transpose & Scale
	t, _ := matrix.Transpose(a)
	s, _ := matrix.Scale(a, 2.5)
	x, _ := t.At(1, 0)
	y, _ := s.At(0, 1)
	fmt.Println("transpose[1,0] =", x)
	fmt.Println("scale[0,1] =", y)

	// Output:
	// sum[1,1] = 12
	// prod[1,0] = 139
	// transpose[1,0] = 2
	// scale[0,1] = 5
}
