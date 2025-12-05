package prim_kruskal_test

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/katalvlaran/lvlath/core"         // core.Graph, core.Edge, and core error types
	"github.com/katalvlaran/lvlath/prim_kruskal" // package under test
	"github.com/stretchr/testify/assert"         // assertion library
)

// buildTriangle constructs a simple undirected, weighted triangle graph:
//
//	A—B (weight 1), B—C (weight 2), A—C (weight 3).
//
// This graph’s MST consists of edges A—B and B—C with total weight 3.
func buildTriangle() *core.Graph {
	// Create a new weighted, undirected graph.
	g := core.NewGraph(core.WithWeighted())
	// Add edges: A<->B(1), B<->C(2), A<->C(3).
	_, _ = g.AddEdge("A", "B", 1)
	_, _ = g.AddEdge("B", "C", 2)
	_, _ = g.AddEdge("A", "C", 3)

	return g
}

// buildMediumGraph creates a connected, weighted graph with n vertices and edgesCount total edges.
// - First, it ensures connectivity by adding a chain V0—V1—...—V(n-1) with random weights [1..10].
// - Then it adds (edgesCount - (n-1)) additional random edges with random weights [1..100].
// The random number generator is seeded deterministically for reproducibility.
func buildMediumGraph(n, edgesCount int) *core.Graph {
	// Create a new weighted, undirected graph.
	g := core.NewGraph(core.WithWeighted())

	// 1. Add n vertices named "V0", "V1", ..., "V(n-1)".
	for i := 0; i < n; i++ {
		_ = g.AddVertex(fmt.Sprintf("V%d", i))
	}

	// 2. Use a new rand.Rand with a fixed seed so that generated edges are always the same.
	r := rand.New(rand.NewSource(42))

	// 3. Ensure basic connectivity by chaining vertices in a line.
	//    For i = 1..n-1, connect V(i-1) to V(i) with a weight in [1..10].
	for i := 1; i < n; i++ {
		weight := 1.0 + r.Float64() + float64(r.Intn(10)) // random weight between 1.0 and 10.0
		_, _ = g.AddEdge(fmt.Sprintf("V%d", i-1), fmt.Sprintf("V%d", i), weight)
	}

	// 4. Add extra random edges to reach edgesCount total edges.
	//    Skip self-loops; allow multiple edges only if they connect different vertices.
	extra := edgesCount - (n - 1)
	for i := 0; i < extra; {
		u := r.Intn(n) // random vertex index for endpoint u
		v := r.Intn(n) // random vertex index for endpoint v
		if u == v {
			// skip loops
			continue
		}
		weight := 1.0 + r.Float64() + float64(r.Intn(100)) // random weight between 1.0 and 100.0

		// AddEdge will fail if multi-edges are disallowed; but default Graph allows only one edge per pair.
		// We do not check the error here since duplicates may be skipped by core.Graph.
		// If duplicate, error is ErrMultiEdgeNotAllowed, and that iteration won’t increase i.
		if _, err := g.AddEdge(fmt.Sprintf("V%d", u), fmt.Sprintf("V%d", v), weight); err == nil {
			i++ // only count successfully added edges
		}
	}

	return g
}

// TestValidation_EmptyOrDisconnected verifies that Prim and Kruskal return ErrDisconnected
// when the graph has no vertices (empty) or when it’s impossible to form a spanning tree.
func TestValidation_EmptyOrDisconnected(t *testing.T) {
	// Create an empty weighted graph (no vertices, no edges).
	g := core.NewGraph(core.WithWeighted())

	// Prim: with root "A" on an empty graph should return ErrDisconnected and empty MST.
	edgesP, totalP, errP := prim_kruskal.Prim(g, "A")
	assert.Empty(t, edgesP)                               // expect no edges returned
	assert.Zero(t, totalP)                                // expect total weight = 0
	assert.ErrorIs(t, errP, prim_kruskal.ErrDisconnected) // expect ErrDisconnected

	// Kruskal: on an empty graph should also return ErrDisconnected and empty MST.
	edgesK, totalK, errK := prim_kruskal.Kruskal(g)
	assert.Empty(t, edgesK)                               // expect no edges returned
	assert.Zero(t, totalK)                                // expect total weight = 0
	assert.ErrorIs(t, errK, prim_kruskal.ErrDisconnected) // expect ErrDisconnected
}

// TestValidation_UnweightedOrDirected verifies that both algorithms reject unweighted or directed graphs.
func TestValidation_UnweightedOrDirected(t *testing.T) {
	// 1. Unweighted graph: By default NewGraph() is unweighted and undirected.
	gUnweighted := core.NewGraph()

	// Kruskal on unweighted should error ErrInvalidGraph (requires weighted).
	_, _, errK1 := prim_kruskal.Kruskal(gUnweighted)
	assert.ErrorIs(t, errK1, prim_kruskal.ErrInvalidGraph)

	// Prim on unweighted should error core.ErrBadWeight.
	_, _, errP1 := prim_kruskal.Prim(gUnweighted, "A")
	assert.ErrorIs(t, errP1, prim_kruskal.ErrInvalidGraph)

	// 2. Directed but weighted graph: Create graph with both directed and weighted flags.
	gDirected := core.NewGraph(core.WithDirected(true), core.WithWeighted())

	// Kruskal should error ErrInvalidGraph when graph.Directed() == true.
	_, _, errK2 := prim_kruskal.Kruskal(gDirected)
	assert.ErrorIs(t, errK2, prim_kruskal.ErrInvalidGraph)

	// Prim should also error ErrInvalidGraph when graph.Directed() == true.
	_, _, errP2 := prim_kruskal.Prim(gDirected, "A")
	assert.ErrorIs(t, errP2, prim_kruskal.ErrInvalidGraph)
}

// TestValidation_MissingRoot verifies that Prim returns ErrEmptyRoot when the root string is empty.
func TestValidation_MissingRoot(t *testing.T) {
	// Build a simple triangle to have vertices.
	g := buildTriangle()

	// Call Prim with an empty root. Should return ErrEmptyRoot.
	_, _, err := prim_kruskal.Prim(g, "")
	assert.ErrorIs(t, err, prim_kruskal.ErrEmptyRoot)
}

// TestPrim_Triangle ensures that Prim on the triangle graph picks the correct MST edges and weight.
func TestPrim_Triangle(t *testing.T) {
	// Build our triangle graph: A—B(1), B—C(2), A—C(3).
	g := buildTriangle()

	// Compute MST via Prim, rooted at "A".
	mst, total, err := prim_kruskal.Prim(g, "A")
	assert.NoError(t, err)      // no error expected
	assert.Equal(t, 3.0, total) // MST weight should be 1 + 2 = 3
	assert.Len(t, mst, 2)       // MST must contain exactly 2 edges

	// Verify that edges {A—B, B—C} appear (undirected so order doesn’t matter).
	names := make(map[string]bool, 2)
	for _, e := range mst {
		u, v := e.From, e.To
		if u > v {
			u, v = v, u
		}
		names[fmt.Sprintf("%s-%s", u, v)] = true
	}
	assert.True(t, names["A-B"], "edge A-B must be in MST")
	assert.True(t, names["B-C"], "edge B-C must be in MST")
}

// TestKruskal_Triangle ensures that Kruskal on the triangle graph picks the correct MST edges and weight.
func TestKruskal_Triangle(t *testing.T) {
	// Build our triangle graph: A—B(1), B—C(2), A—C(3).
	g := buildTriangle()

	// Compute MST via Kruskal.
	mst, total, err := prim_kruskal.Kruskal(g)
	assert.NoError(t, err)      // no error expected
	assert.Equal(t, 3.0, total) // MST weight should be 1 + 2 = 3
	assert.Len(t, mst, 2)       // MST must contain exactly 2 edges

	// Verify that edges {A—B, B—C} appear.
	names := make(map[string]bool, 2)
	for _, e := range mst {
		u, v := e.From, e.To
		if u > v {
			u, v = v, u
		}
		names[fmt.Sprintf("%s-%s", u, v)] = true
	}
	assert.True(t, names["A-B"], "edge A-B must be in MST")
	assert.True(t, names["B-C"], "edge B-C must be in MST")
}

// TestSingleVertexGraph verifies behavior when the graph has exactly one vertex.
// - Kruskal should return an empty MST with no error.
// - Prim should return an empty MST with no error, provided root matches that vertex.
func TestSingleVertexGraph(t *testing.T) {
	// Create a graph with one vertex "X".
	g := core.NewGraph(core.WithWeighted())
	_ = g.AddVertex("X")

	// Kruskal on single-vertex graph: no error, empty MST, total weight = 0.
	mstK, totalK, errK := prim_kruskal.Kruskal(g)
	assert.NoError(t, errK)
	assert.Empty(t, mstK)
	assert.Zero(t, totalK)

	// Prim on single-vertex graph with root "X": no error, empty MST, total weight = 0.
	mstP, totalP, errP := prim_kruskal.Prim(g, "X")
	assert.NoError(t, errP)
	assert.Empty(t, mstP)
	assert.Zero(t, totalP)
}

// TestTwoIsolatedVertices verifies that disconnected graph with two isolated vertices
// returns ErrDisconnected from both Prim and Kruskal.
func TestTwoIsolatedVertices(t *testing.T) {
	// Create a graph with two vertices "A" and "B", but no edge between them.
	g := core.NewGraph(core.WithWeighted())
	_ = g.AddVertex("A")
	_ = g.AddVertex("B")

	// Kruskal should detect disconnection: ErrDisconnected.
	_, _, errK := prim_kruskal.Kruskal(g)
	assert.ErrorIs(t, errK, prim_kruskal.ErrDisconnected)

	// Prim from "A" should also detect disconnection: ErrDisconnected.
	_, _, errP := prim_kruskal.Prim(g, "A")
	assert.ErrorIs(t, errP, prim_kruskal.ErrDisconnected)
}

// TestParallelEdgesSelection verifies that when multiple edges exist between same vertices (multi-edges),
// both Prim and Kruskal pick the lighter edge in the MST.
func TestParallelEdgesSelection(t *testing.T) {
	// Create a graph that allows multi-edges and is weighted.
	g := core.NewGraph(core.WithWeighted(), core.WithMultiEdges())

	// Add two parallel edges between A and B: one with weight 5, one with weight 1.
	_, err1 := g.AddEdge("A", "B", 5)
	assert.NoError(t, err1)
	_, err2 := g.AddEdge("A", "B", 1)
	assert.NoError(t, err2)

	// Kruskal should choose only the weight‐1 edge: total = 1, MST size = 1.
	mstK, totalK, errK := prim_kruskal.Kruskal(g)
	assert.NoError(t, errK)
	assert.Equal(t, 1.0, totalK)
	assert.Len(t, mstK, 1)

	// Prim from root "A" should also pick the weight‐1 edge: total = 1, MST size = 1.
	mstP, totalP, errP := prim_kruskal.Prim(g, "A")
	assert.NoError(t, errP)
	assert.Equal(t, 1.0, totalP)
	assert.Len(t, mstP, 1)
}

// TestMixedEdgesFlagIgnored verifies that if graph is created with WithMixedEdges (allow per‐edge directedness),
// but a truly directed edge is inserted, both Prim and Kruskal should error ErrInvalidGraph,
// because MST requires a purely undirected graph.
func TestMixedEdgesFlagIgnored(t *testing.T) {
	// Create a graph that allows mixed edges (per-edge directed overrides) and is weighted.
	g := core.NewGraph(core.WithWeighted(), core.WithMixedEdges())

	// Add a directed edge override: A->B with weight 1.
	_, err := g.AddEdge("A", "B", 1, core.WithEdgeDirected(true))
	assert.NoError(t, err)

	// Kruskal should detect directed edge presence and return ErrInvalidGraph.
	_, _, errK := prim_kruskal.Kruskal(g)
	assert.ErrorIs(t, errK, prim_kruskal.ErrInvalidGraph)

	// Prim from "A" should also detect directed edge and return ErrInvalidGraph.
	_, _, errP := prim_kruskal.Prim(g, "A")
	assert.ErrorIs(t, errP, prim_kruskal.ErrInvalidGraph)
}

// TestComparison_MediumGraph compares Prim vs. Kruskal on a larger randomly generated graph.
// Ensures both algorithms produce the same total weight and correct number of edges.
func TestComparison_MediumGraph(t *testing.T) {
	// Build a “medium” graph with 10 vertices and 20 total edges.
	g := buildMediumGraph(10, 20)

	// Compute MST via Kruskal.
	mstK, totalK, errK := prim_kruskal.Kruskal(g)
	assert.NoError(t, errK)                  // no error expected
	assert.Len(t, mstK, len(g.Vertices())-1) // MST size must be |V|-1

	// Compute MST via Prim, rooted at "V0".
	mstP, totalP, errP := prim_kruskal.Prim(g, "V0")
	assert.NoError(t, errP)                  // no error expected
	assert.Len(t, mstP, len(g.Vertices())-1) // MST size must be |V|-1
	const tolerance = 1e-10

	// The total weights produced by both methods must match.
	//assert.Equal(t, totalK, totalP)
	assert.InDelta(t, totalK, totalP, tolerance)
}
