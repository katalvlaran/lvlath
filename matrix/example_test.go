// Package matrix_test provides GoDoc examples for lvlath/matrix,
// demonstrating common adjacency/incidence workflows and small LA snippets.
package matrix_test

import (
	"fmt"

	"github.com/katalvlaran/lvlath/builder"
	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/matrix"
)

// ExampleAdjacencyWorkflow builds a directed, weighted complete graph (no loops, no multi),
// constructs its adjacency matrix, performs a round-trip back to a graph, and then
// demonstrates in-place APSP on a separate adjacency object.
//
// Implementation:
//   - Stage 1: Build a directed, weighted complete graph with V vertices (no loops, no multi).
//   - Stage 2: Construct adjacency (no metric-closure), round-trip back to graph, print counts.
//   - Stage 3: Build another adjacency and run APSP in-place (Floyd–Warshall).
//
// Behavior highlights:
//   - Round-trip export is valid only for non-metric-closure adjacency.
//   - APSP runs in-place; +Inf means “no edge”, diagonal 0 is required.
//
// Inputs:
//   - V: number of vertices in the generated complete graph.
//
// Returns:
//   - Printed counts for vertices and edges after round-trip.
//
// Errors:
//   - Omitted for brevity in example; production code should handle errors.
//
// Determinism:
//   - Vertex order is stable; Complete(V) is deterministic; counts are reproducible.
//
// Complexity:
//   - Build O(V^2), APSP O(V^3). Space is dominated by O(V^2) adjacency.
//
// Notes:
//   - Do NOT export metric-closure matrices back to edges; this is intentionally refused.
//
// AI-Hints:
//   - Use BuildMetricClosure if you specifically need distances and export protection.
//   - Use *Dense-backed matrices to benefit from flat-slice loops internally.
func ExampleAdjacencyWorkflow() {
	const V = 8 // number of vertices; kept as a named constant for clarity

	// (Prepare) Build a directed, weighted complete graph on V vertices, without loops or multi-edges.
	g, _ := builder.BuildGraph(
		[]core.GraphOption{
			core.WithDirected(true), // treat edges as directed
			core.WithWeighted(),     // preserve weights on edges
			core.WithLoops(),        // forbid self-loops to keep edge count deterministic for this demo
			core.WithMultiEdges(),   // forbid parallel edges for a single, deterministic edge set
		},
		[]builder.BuilderOption{
			builder.WithSymbNumb("v"), // stable IDs: v0, v1, ...
		},
		builder.Complete(V), // generator: complete graph on V vertices
	)

	// (Execute) Construct an adjacency matrix without metric-closure (export remains allowed).
	opts := matrix.NewMatrixOptions(
		matrix.WithDirected(),
		matrix.WithWeighted(),
		matrix.WithDisallowLoops(),
		matrix.WithDisallowMulti(),
		// Intentionally omit WithMetricClosure to keep export enabled.
	)
	am, _ := matrix.NewAdjacencyMatrix(g, opts)

	// (Finalize) Reconstruct a graph from the adjacency and print counts (deterministic).
	g2, _ := am.ToGraph()
	fmt.Printf("Vertices: %d, Edges: %d\n", len(g2.Vertices()), len(g2.Edges()))

	// (Extra) Demonstrate APSP on a fresh adjacency; export is not allowed after metric-closure.
	am2, _ := matrix.NewAdjacencyMatrix(g, opts) // fresh adjacency with the same policy
	_ = matrix.APSPInPlace(am2.Mat)              // run Floyd–Warshall in-place on the underlying matrix
	// NOTE: am2 is now a distance matrix; exporting back to edges is intentionally refused by policy.

	// Output:
	// Vertices: 8, Edges: 56
}

// ExampleIncidenceWorkflow builds a directed, weighted path graph, inspects per-vertex
// incidence vectors, and prints each edge’s endpoints.
//
// Implementation:
//   - Stage 1: Build a directed, weighted path on V vertices.
//   - Stage 2: Construct the directed incidence matrix.
//   - Stage 3: Print vertex incidence vectors and edge endpoint pairs.
//
// Behavior highlights:
//   - Incidence uses deterministic ordering; outgoing edges contribute -1, incoming +1.
//
// Inputs:
//   - V: number of vertices in the generated path graph.
//
// Returns:
//   - Printed incidence vectors and an edge list.
//
// Errors:
//   - Omitted in the example; production code should handle them.
//
// Determinism:
//   - Path(V) and incidence ordering are deterministic.
//
// Complexity:
//   - Build O(V), incidence O(V) rows × O(E) columns.
//
// Notes:
//   - Vertex identifiers are stable and printable.
//
// AI-Hints:
//   - Incidence is useful for flow constraints and divergence computations.
func ExampleIncidenceWorkflow() {
	const V = 5 // path length produces V-1 edges

	// (Prepare) Build a directed, weighted path on V vertices.
	g, _ := builder.BuildGraph(
		[]core.GraphOption{core.WithWeighted(), core.WithDirected(true)},
		[]builder.BuilderOption{},
		builder.Path(V),
	)

	// (Execute) Build the directed incidence matrix for this graph.
	im, _ := matrix.NewIncidenceMatrix(g, matrix.NewMatrixOptions(matrix.WithDirected()))

	// (Finalize) Print each vertex’s incidence vector using the actual (stable) vertex IDs.
	fmt.Println("VertexIncidence vectors:")
	for _, id := range g.Vertices() {
		vec, _ := im.VertexIncidence(id)
		fmt.Printf("  %s: %v\n", id, vec)
	}

	// (Finalize) Print each edge’s endpoints in deterministic column order.
	var eggeCount, _ = im.EdgeCount()
	fmt.Println("EdgeEndpoints list:")
	for j := 0; j < eggeCount; j++ {
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

// ExampleMatrixMethods demonstrates Add, Mul, Transpose, and Scale on small matrices.
//
// Implementation:
//   - Stage 1: Construct two 2×2 matrices a and b.
//   - Stage 2: Add them and print one element from the result.
//   - Stage 3: Multiply a 2×3 by a 3×2 and print one element.
//   - Stage 4: Transpose and Scale, printing selected entries.
//
// Behavior highlights:
//   - All kernels are deterministic; *Dense fast-paths are used underneath.
//
// Inputs:
//   - None (literals are used).
//
// Returns:
//   - Printed values for sanity checks.
//
// Errors:
//   - Omitted for brevity in this example.
//
// Determinism:
//   - Fixed traversal orders in dense kernels.
//
// Complexity:
//   - Add O(rc), Mul O(rnc), Transpose O(rc), Scale O(rc).
//
// Notes:
//   - Use AllClose in property tests to compare floats under tolerance.
//
// AI-Hints:
//   - Reuse matrices and vectors in hot paths to minimize allocations.
func ExampleMatrixMethods() {
	// (1) Construct two 2×2 matrices and fill them with small literals.
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

	// (2) Add: c = a + b
	sum, _ := matrix.Add(a, b)
	v, _ := sum.At(1, 1) // pick an element for the sample output
	fmt.Println("sum[1,1] =", v)

	// (3) Multiply a 2×3 by a 3×2 and print one element.
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

	// (4) Transpose and Scale
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
