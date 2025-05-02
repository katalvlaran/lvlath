package core

import (
	"reflect"
	"sort"
	"testing"
)

// TestAddVertexHasVertex covers AddVertex and HasVertex.
func TestAddVertexHasVertex(t *testing.T) {
	g := NewGraph(false, false)
	if g.HasVertex("A") {
		t.Error("empty graph should not have A")
	}
	g.AddVertex(&Vertex{ID: "A"})
	if !g.HasVertex("A") {
		t.Error("graph should have A after AddVertex")
	}
	// idempotence
	g.AddVertex(&Vertex{ID: "A"})
	if len(g.Vertices()) != 1 {
		t.Errorf("AddVertex duplicate should not increase count; got %d", len(g.Vertices()))
	}
}

// TestRemoveVertex checks that RemoveVertex deletes vertex and incident edges.
func TestRemoveVertex(t *testing.T) {
	// Undirected
	g := NewGraph(false, false)
	g.AddEdge("A", "B", 1)
	g.RemoveVertex("A")
	if g.HasVertex("A") {
		t.Error("A should be removed")
	}
	if g.HasEdge("B", "A") {
		t.Error("mirror edge B→A should be removed")
	}

	// Directed
	g2 := NewGraph(true, false)
	g2.AddEdge("X", "Y", 1)
	g2.RemoveVertex("Y")
	if g2.HasVertex("Y") {
		t.Error("Y should be removed")
	}
	if g2.HasEdge("X", "Y") {
		t.Error("edge X→Y should be removed")
	}
}

// TestAddEdgeHasEdgeMultiedges verifies auto-add, HasEdge, and multiedges.
func TestAddEdgeHasEdgeMultiedges(t *testing.T) {
	g := NewGraph(false, true)
	// auto-add vertices
	g.AddEdge("A", "B", 5)
	if !g.HasVertex("A") || !g.HasVertex("B") {
		t.Fatal("AddEdge should auto-add vertices")
	}
	// single edge
	if !g.HasEdge("A", "B") {
		t.Error("expected edge A→B")
	}
	// undirected mirror
	if !g.HasEdge("B", "A") {
		t.Error("expected mirror edge B→A in undirected")
	}
	// multiedges
	g.AddEdge("A", "B", 7)
	edges := g.Edges()
	countAB := 0
	for _, e := range edges {
		if e.From.ID == "A" && e.To.ID == "B" {
			countAB++
		}
	}
	if countAB != 2 {
		t.Errorf("expected 2 distinct A→B edges, got %d", countAB)
	}
}

// TestRemoveEdge covers RemoveEdge in directed and undirected.
func TestRemoveEdge(t *testing.T) {
	// Directed
	g := NewGraph(true, false)
	g.AddEdge("X", "Y", 2)
	g.RemoveEdge("X", "Y")
	if g.HasEdge("X", "Y") {
		t.Error("directed RemoveEdge failed")
	}
	// Undirected
	g2 := NewGraph(false, false)
	g2.AddEdge("U", "V", 3)
	g2.RemoveEdge("U", "V")
	if g2.HasEdge("U", "V") || g2.HasEdge("V", "U") {
		t.Error("undirected RemoveEdge should remove both directions")
	}
}

// TestNeighbors ensures unique neighbors, even with multiedges.
func TestNeighbors(t *testing.T) {
	g := NewGraph(false, false)
	g.AddEdge("1", "2", 0)
	g.AddEdge("1", "2", 0) // duplicate
	nb := g.Neighbors("1")
	if len(nb) != 1 || nb[0].ID != "2" {
		t.Errorf("neighbors should be [2], got %v", nb)
	}
	// Nonexistent
	if nn := g.Neighbors("X"); nn != nil {
		t.Errorf("Neighbors of missing vertex should be nil, got %v", nn)
	}
}

// TestVerticesEdges checks Vertices() and Edges() output sizes.
func TestVerticesEdges(t *testing.T) {
	g := NewGraph(false, false)
	g.AddVertex(&Vertex{ID: "A"})
	g.AddVertex(&Vertex{ID: "B"})
	g.AddEdge("A", "B", 1)
	vs := g.Vertices()
	if !reflect.DeepEqual(sortedIDs(vs), []string{"A", "B"}) {
		t.Errorf("Vertices = %v; want [A B]", sortedIDs(vs))
	}
	es := g.Edges()
	if len(es) != 2 {
		t.Errorf("Edges length = %d; want 2 (A→B & B→A)", len(es))
	}
}

// TestSelfLoop behavior: core allows self-loop and mirror for undirected.
func TestSelfLoop(t *testing.T) {
	g := NewGraph(false, false)
	g.AddEdge("Z", "Z", 10)
	if !g.HasEdge("Z", "Z") {
		t.Error("self-loop Z→Z should exist")
	}
	es := g.Edges()
	// Undirected mirror also adds second loop
	loopCount := 0
	for _, e := range es {
		if e.From.ID == "Z" && e.To.ID == "Z" {
			loopCount++
		}
	}
	if loopCount != 2 {
		t.Errorf("expected 2 self-loop edges (mirror), got %d", loopCount)
	}

	// Directed self-loop
	g2 := NewGraph(true, false)
	g2.AddEdge("W", "W", 5)
	es2 := g2.Edges()
	count := 0
	for _, e := range es2 {
		if e.From.ID == "W" && e.To.ID == "W" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 directed self-loop, got %d", count)
	}
}

// sortedIDs helper for comparison
func sortedIDs(vs []*Vertex) []string {
	ids := make([]string, len(vs))
	for i, v := range vs {
		ids[i] = v.ID
	}
	sort.Strings(ids)
	return ids
}
