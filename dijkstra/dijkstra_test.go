// Package dijkstra_test contains unit tests for the Dijkstra implementation.
// These tests validate correct behavior under various configurations, including
// basic functionality, directed graphs, mixed edges, MaxDistance, InfEdgeThreshold,
// and edge cases such as single-vertex and self-loop graphs.
package dijkstra_test

import (
	"math"
	"testing"

	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/dijkstra"
)

// ------------------------------------------------------------------------
// 1. Validation Tests: Ensure errors are returned for invalid inputs.
// ------------------------------------------------------------------------

func TestDijkstra_EmptySource(t *testing.T) {
	// When no Source is provided (empty by default), Dijkstra should return ErrEmptySource.
	// Create a weighted graph (so it passes the weighted check) but do not pass Source.
	g := core.NewGraph(core.WithWeighted())
	_, _, err := dijkstra.Dijkstra(g)
	if err != dijkstra.ErrEmptySource {
		t.Fatalf("Expected ErrEmptySource, got %v", err)
	}
}

func TestDijkstra_NilGraphWithoutSource(t *testing.T) {
	// If graph is nil and no Source is provided, ErrEmptySource has priority over ErrNilGraph.
	_, _, err := dijkstra.Dijkstra(nil)
	if err != dijkstra.ErrEmptySource {
		t.Fatalf("Expected ErrEmptySource when graph is nil and Source is empty, got %v", err)
	}
}

func TestDijkstra_NilGraphWithSource(t *testing.T) {
	// If graph is nil but Source is provided, Dijkstra should return ErrNilGraph.
	_, _, err := dijkstra.Dijkstra(nil, dijkstra.Source("X"))
	if err != dijkstra.ErrNilGraph {
		t.Fatalf("Expected ErrNilGraph when graph is nil, got %v", err)
	}
}

func TestDijkstra_UnweightedGraph(t *testing.T) {
	// If the graph is not weighted, Dijkstra must return ErrUnweightedGraph.
	g := core.NewGraph() // unweighted by default
	_, _, err := dijkstra.Dijkstra(g, dijkstra.Source("A"))
	if err != dijkstra.ErrUnweightedGraph {
		t.Fatalf("Expected ErrUnweightedGraph, got %v", err)
	}
}

func TestDijkstra_SourceNotFound(t *testing.T) {
	// If the graph is weighted but does not contain the Source vertex, return ErrVertexNotFound.
	g := core.NewGraph(core.WithWeighted())
	_, _, err := dijkstra.Dijkstra(g, dijkstra.Source("X"))
	if err != dijkstra.ErrVertexNotFound {
		t.Fatalf("Expected ErrVertexNotFound, got %v", err)
	}
}

func TestDijkstra_NegativeWeightDetectedEarly(t *testing.T) {
	// Build a weighted graph with a negative weight edge.
	g := core.NewGraph(core.WithWeighted())
	g.AddEdge("A", "B", -5) // invalid negative weight
	_, _, err := dijkstra.Dijkstra(g, dijkstra.Source("A"))
	if err == nil || err.Error() == "" || err != dijkstra.ErrNegativeWeight && !contains(err.Error(), "negative edge weight") {
		t.Fatalf("Expected ErrNegativeWeight, got %v", err)
	}
}

// ------------------------------------------------------------------------
// 2. Basic Functionality: Small graphs, path correctness without and with ReturnPath.
// ------------------------------------------------------------------------

func TestDijkstra_SimpleTriangle_NoPath(t *testing.T) {
	// Graph: A—B(1), B—C(2), A—C(5), all undirected by default.
	g := core.NewGraph(core.WithWeighted())
	g.AddEdge("A", "B", 1)
	g.AddEdge("B", "C", 2)
	g.AddEdge("A", "C", 5)

	// Compute distances without requesting the predecessor map.
	dist, prev, err := dijkstra.Dijkstra(g, dijkstra.Source("A"))
	if err != nil {
		t.Fatal(err)
	}

	// Distance from A to C should be 3 via A→B→C.
	if got, want := dist["C"], int64(3); got != want {
		t.Errorf("dist[C] = %d; want %d", got, want)
	}
	// prev should be nil when ReturnPath=false.
	if prev != nil {
		t.Errorf("expected nil predecessor map, got %v", prev)
	}
}

func TestDijkstra_SimpleTriangle_WithPath(t *testing.T) {
	// Same triangle graph, but request path reconstruction.
	g := core.NewGraph(core.WithWeighted())
	g.AddEdge("A", "B", 1)
	g.AddEdge("B", "C", 2)
	g.AddEdge("A", "C", 5)

	// Compute distances and prev map.
	dist, prev, err := dijkstra.Dijkstra(g, dijkstra.Source("A"), dijkstra.WithReturnPath())
	if err != nil {
		t.Fatal(err)
	}

	// Check distance values.
	if dist["A"] != 0 || dist["B"] != 1 || dist["C"] != 3 {
		t.Errorf("Unexpected distances: %v", dist)
	}

	// Check predecessor chain: B←A, C←B.
	if prev["B"] != "A" {
		t.Errorf("prev[B] = %q; want %q", prev["B"], "A")
	}
	if prev["C"] != "B" {
		t.Errorf("prev[C] = %q; want %q", prev["C"], "B")
	}
}

func TestDijkstra_ChainWithPath(t *testing.T) {
	// Graph:
	// A—B—C—D—E
	//      |
	//      F—G
	g := core.NewGraph(core.WithWeighted())
	g.AddEdge("A", "B", 1)
	g.AddEdge("B", "C", 1)
	g.AddEdge("C", "D", 1)
	g.AddEdge("D", "E", 1)
	g.AddEdge("D", "F", 1)
	g.AddEdge("F", "G", 1)

	// Compute with path reconstruction.
	dist, prev, err := dijkstra.Dijkstra(g, dijkstra.Source("A"), dijkstra.WithReturnPath())
	if err != nil {
		t.Fatal(err)
	}

	// Expected distances.
	expectedDistances := map[string]int64{
		"A": 0,
		"B": 1,
		"C": 2,
		"D": 3,
		"E": 4,
		"F": 4,
		"G": 5,
	}
	for v, want := range expectedDistances {
		if got := dist[v]; got != want {
			t.Errorf("dist[%s] = %d; want %d", v, got, want)
		}
	}

	// Check a few predecessor links: B←A, C←B, D←C.
	if prev["B"] != "A" || prev["C"] != "B" || prev["D"] != "C" {
		t.Errorf("Unexpected predecessors: %v", prev)
	}
}

// ------------------------------------------------------------------------
// 3. Directed Graph Tests: Ensure correct handling of one-way edges.
// ------------------------------------------------------------------------

func TestDijkstra_MediumDirectedGraph(t *testing.T) {
	// Directed graph:
	// A→B(2), A→C(1), C→B(1), B→D(3), C→D(5)
	g := core.NewGraph(core.WithDirected(true), core.WithWeighted())
	g.AddEdge("A", "B", 2)
	g.AddEdge("A", "C", 1)
	g.AddEdge("C", "B", 1)
	g.AddEdge("B", "D", 3)
	g.AddEdge("C", "D", 5)

	// Compute without requesting prev map.
	dist, prev, err := dijkstra.Dijkstra(g, dijkstra.Source("A"))
	if err != nil {
		t.Fatal(err)
	}

	// Expected: dist[B]=2 (via A→C→B), dist[C]=1, dist[D]=5 (via A→C→B→D).
	if dist["C"] != 1 {
		t.Errorf("dist[C] = %d; want %d", dist["C"], 1)
	}
	if dist["B"] != 2 {
		t.Errorf("dist[B] = %d; want %d", dist["B"], 2)
	}
	if dist["D"] != 5 {
		t.Errorf("dist[D] = %d; want %d", dist["D"], 5)
	}
	// prev should be nil because ReturnPath was not requested.
	if prev != nil {
		t.Errorf("expected nil prev, got %v", prev)
	}
}

// ------------------------------------------------------------------------
// 4. Mixed Edges: Verify behavior when graph contains both directed and undirected edges.
// ------------------------------------------------------------------------

func TestDijkstra_MixedEdges(t *testing.T) {
	// Mixed graph (WithWeighted + WithMixedEdges allows both directed and undirected edges).
	g := core.NewGraph(core.WithWeighted(), core.WithMixedEdges())

	// Add A→B (directed weight 2).
	_, _ = g.AddEdge("A", "B", 2, core.WithEdgeDirected(true))

	// Add B—C (undirected weight 3).
	_, _ = g.AddEdge("B", "C", 3, core.WithEdgeDirected(false))

	// Add C→D (directed weight 1).
	_, _ = g.AddEdge("C", "D", 1, core.WithEdgeDirected(true))

	// Compute with path reconstruction.
	dist, prev, err := dijkstra.Dijkstra(g, dijkstra.Source("A"), dijkstra.WithReturnPath())
	if err != nil {
		t.Fatal(err)
	}

	// Expected distances:
	// A:0, B:2, C:5 (via A→B—C), D:6 (via A→B—C→D).
	if dist["A"] != 0 {
		t.Errorf("dist[A] = %d; want %d", dist["A"], 0)
	}
	if dist["B"] != 2 {
		t.Errorf("dist[B] = %d; want %d", dist["B"], 2)
	}
	if dist["C"] != 5 {
		t.Errorf("dist[C] = %d; want %d", dist["C"], 5)
	}
	if dist["D"] != 6 {
		t.Errorf("dist[D] = %d; want %d", dist["D"], 6)
	}

	// Check predecessor chain: B←A, C←B, D←C.
	if prev["B"] != "A" {
		t.Errorf("prev[B] = %q; want %q", prev["B"], "A")
	}
	if prev["C"] != "B" {
		t.Errorf("prev[C] = %q; want %q", prev["C"], "B")
	}
	if prev["D"] != "C" {
		t.Errorf("prev[D] = %q; want %q", prev["D"], "C")
	}

	// Confirm that Dijkstra did not traverse backward along directed edge A←B:
	// dist[A] stays 0.
	if dist["A"] != 0 {
		t.Errorf("dist[A] changed unexpectedly to %d", dist["A"])
	}
}

// ------------------------------------------------------------------------
// 5. MaxDistance Tests: Ensure that vertices with distance > MaxDistance are not explored.
// ------------------------------------------------------------------------

func TestDijkstra_MaxDistanceLimits(t *testing.T) {
	// Linear graph: A—B(1)—C(1)—D(1)
	g := core.NewGraph(core.WithWeighted())
	g.AddEdge("A", "B", 1)
	g.AddEdge("B", "C", 1)
	g.AddEdge("C", "D", 1)

	// Set MaxDistance = 1: only A and B are within threshold.
	dist, _, err := dijkstra.Dijkstra(
		g,
		dijkstra.Source("A"),
		dijkstra.WithMaxDistance(1),
	)
	if err != nil {
		t.Fatal(err)
	}

	// dist[A]=0, dist[B]=1, dist[C] and dist[D] remain ∞ (unvisited).
	if dist["A"] != 0 {
		t.Errorf("dist[A] = %d; want %d", dist["A"], 0)
	}
	if dist["B"] != 1 {
		t.Errorf("dist[B] = %d; want %d", dist["B"], 1)
	}
	if dist["C"] != math.MaxInt64 {
		t.Errorf("dist[C] = %d; want %d (unreachable)", dist["C"], math.MaxInt64)
	}
	if dist["D"] != math.MaxInt64 {
		t.Errorf("dist[D] = %d; want %d (unreachable)", dist["D"], math.MaxInt64)
	}
}

func TestDijkstra_MaxDistanceZero(t *testing.T) {
	// Graph: A—B(1)
	g := core.NewGraph(core.WithWeighted())
	g.AddEdge("A", "B", 1)

	// Set MaxDistance = 0: only the source itself should be processed.
	dist, _, err := dijkstra.Dijkstra(
		g,
		dijkstra.Source("A"),
		dijkstra.WithMaxDistance(0),
	)
	if err != nil {
		t.Fatal(err)
	}

	// dist[A]=0, dist[B] remains ∞.
	if dist["A"] != 0 {
		t.Errorf("dist[A] = %d; want %d", dist["A"], 0)
	}
	if dist["B"] != math.MaxInt64 {
		t.Errorf("dist[B] = %d; want %d (unreachable)", dist["B"], math.MaxInt64)
	}
}

// ------------------------------------------------------------------------
// 6. InfEdgeThreshold Tests: Ensure “impassable” edges are skipped appropriately.
// ------------------------------------------------------------------------

func TestDijkstra_InfThreshold_DefaultBehavior(t *testing.T) {
	// If InfEdgeThreshold is not set, default is MaxInt64, so no edges are impassable.
	// Graph: A—B(10), B—C(20)
	g := core.NewGraph(core.WithWeighted())
	g.AddEdge("A", "B", 10)
	g.AddEdge("B", "C", 20)

	dist, _, err := dijkstra.Dijkstra(g, dijkstra.Source("A"))
	if err != nil {
		t.Fatal(err)
	}

	// dist[C] should equal 30.
	if dist["C"] != 30 {
		t.Errorf("dist[C] = %d; want %d", dist["C"], 30)
	}
}

func TestDijkstra_InfThresholdStopsHeavyEdge(t *testing.T) {
	// Graph: A—B(2), B—C(4), A—C(10)
	g := core.NewGraph(core.WithWeighted())
	g.AddEdge("A", "B", 2)
	g.AddEdge("B", "C", 4)
	g.AddEdge("A", "C", 10)

	// Set InfEdgeThreshold = 5: edges with weight ≥5 are skipped, so A—C(10) is ignored.
	dist, _, err := dijkstra.Dijkstra(
		g,
		dijkstra.Source("A"),
		dijkstra.WithInfEdgeThreshold(5),
	)
	if err != nil {
		t.Fatal(err)
	}

	// Now the shortest path from A to C is A→B→C with total cost 6.
	if dist["C"] != 6 {
		t.Errorf("dist[C] = %d; want %d", dist["C"], 6)
	}
}

func TestDijkstra_InfObstacle_3x3GridCorrected(t *testing.T) {
	// Build 3×3 grid of vertices "0,0" to "2,2" with edges weight=1.
	g := core.NewGraph(core.WithWeighted())
	coords := []string{"0,0", "0,1", "0,2", "1,0", "1,1", "1,2", "2,0", "2,1", "2,2"}
	for _, v := range coords {
		g.AddVertex(v)
	}
	// Connect horizontally and vertically where applicable with weight=1.
	g.AddEdge("0,0", "0,1", 1)
	g.AddEdge("0,0", "1,0", 1)
	g.AddEdge("0,1", "0,2", 1)
	g.AddEdge("1,0", "2,0", 1)
	g.AddEdge("1,1", "1,2", 1)
	g.AddEdge("2,1", "2,2", 1)

	// Now make row y=1 into an “impassable wall” by adding edges with weight=threshold.
	threshold := int64(5)
	// Add or replace edges at ("1,0"→"1,1") and ("1,1"→"1,2") with weight=5.
	g.AddEdge("1,0", "1,1", threshold)
	g.AddEdge("1,1", "1,2", threshold)

	// Execute Dijkstra with InfEdgeThreshold = 5. Edges with weight ≥5 are skipped.
	dist, _, err := dijkstra.Dijkstra(
		g,
		dijkstra.Source("0,0"),
		dijkstra.WithInfEdgeThreshold(threshold),
	)
	if err != nil {
		t.Fatal(err)
	}

	// Now vertex "1,1" is unreachable (it lies behind the “wall”).
	if dist["1,1"] != math.MaxInt64 {
		t.Errorf("Expected '1,1' unreachable (MaxInt64), got %d", dist["1,1"])
	}
}

// ------------------------------------------------------------------------
// 7. Edge Cases: Single vertex, Empty graph, Self-loop.
// ------------------------------------------------------------------------

func TestDijkstra_SingleVertex_ReturnsZero(t *testing.T) {
	// Graph with a single vertex "Solo" and no edges.
	g := core.NewGraph(core.WithWeighted())
	g.AddVertex("Solo")

	// Compute with ReturnPath.
	dist, prev, err := dijkstra.Dijkstra(g, dijkstra.Source("Solo"), dijkstra.WithReturnPath())
	if err != nil {
		t.Fatal(err)
	}

	// For the only vertex, distance is 0 and prev["Solo"] == "".
	if d := dist["Solo"]; d != 0 {
		t.Errorf("dist[\"Solo\"] = %d; want %d", d, 0)
	}
	if p := prev["Solo"]; p != "" {
		t.Errorf("prev[\"Solo\"] = %q; want empty string", p)
	}
}

func TestDijkstra_EmptyGraph_ReturnsVertexNotFound(t *testing.T) {
	// Graph is weighted but contains no vertices.
	g := core.NewGraph(core.WithWeighted())
	// Do not add any vertex, request Source="Any".
	_, _, err := dijkstra.Dijkstra(g, dijkstra.Source("Any"))
	if err != dijkstra.ErrVertexNotFound {
		t.Errorf("Expected ErrVertexNotFound for empty graph, got %v", err)
	}
}

func TestDijkstra_SelfLoopZeroWeight(t *testing.T) {
	// Graph with self-loop allowed and weight=0.
	g := core.NewGraph(core.WithWeighted(), core.WithLoops())
	_, _ = g.AddEdge("X", "X", 0)

	// Compute with ReturnPath.
	dist, prev, err := dijkstra.Dijkstra(g, dijkstra.Source("X"), dijkstra.WithReturnPath())
	if err != nil {
		t.Fatal(err)
	}

	// Distance from X to itself is 0, and prev["X"] == "".
	if d := dist["X"]; d != 0 {
		t.Errorf("dist[\"X\"] = %d; want %d", d, 0)
	}
	if p := prev["X"]; p != "" {
		t.Errorf("prev[\"X\"] = %q; want empty string", p)
	}
}

// ------------------------------------------------------------------------
// 8. Test Helper: Check if substring is in error message.
// ------------------------------------------------------------------------

func contains(full, substr string) bool {
	return len(full) >= len(substr) && (full == substr || (len(full) > len(substr) && (full[:len(substr)] == substr || full[len(full)-len(substr):] == substr || stringIndex(full, substr) >= 0)))
}

// stringIndex returns the index of substr in str or -1 if not found.
// This is a minimal reimplementation of strings.Index to avoid imports.
func stringIndex(str, substr string) int {
	for i := 0; i+len(substr) <= len(str); i++ {
		if str[i:i+len(substr)] == substr {
			return i
		}
	}

	return -1
}
