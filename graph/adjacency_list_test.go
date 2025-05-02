package graph

import (
	"testing"
)

func TestAddAndHasVertex(t *testing.T) {
	g := NewGraph(false, true)
	if g.HasVertex("A") {
		t.Errorf("empty graph should not have any vertices")
	}
	g.AddVertex(&Vertex{ID: "A"})
	if !g.HasVertex("A") {
		t.Errorf("expected vertex A to exist")
	}
}

func TestAddEdgeAndHasEdge(t *testing.T) {
	g := NewGraph(false, true)
	g.AddEdge("A", "B", 5)
	if !g.HasVertex("A") || !g.HasVertex("B") {
		t.Errorf("AddEdge should auto-add vertices")
	}
	if !g.HasEdge("A", "B") {
		t.Errorf("expected edge A→B to exist")
	}
	if !g.HasEdge("B", "A") {
		t.Errorf("in undirected graph edge B→A should also exist")
	}
}

func TestRemoveVertex(t *testing.T) {
	g := NewGraph(false, true)
	g.AddEdge("A", "B", 1)
	g.RemoveVertex("A")
	if g.HasVertex("A") || g.HasEdge("A", "B") {
		t.Errorf("vertex A and its edges should be removed")
	}
	if g.HasEdge("B", "A") {
		t.Errorf("mirror edge B→A should be removed too")
	}
}

func TestRemoveEdge(t *testing.T) {
	g := NewGraph(true, true)
	g.AddEdge("X", "Y", 2)
	g.RemoveEdge("X", "Y")
	if g.HasEdge("X", "Y") {
		t.Errorf("edge X→Y should be removed")
	}
}

func TestNeighborsVerticesEdges(t *testing.T) {
	g := NewGraph(false, true)
	g.AddEdge("1", "2", 0)
	g.AddEdge("1", "3", 0)
	n := g.Neighbors("1")
	if len(n) != 2 {
		t.Errorf("expected 2 neighbors, got %d", len(n))
	}
	vs := g.Vertices()
	if len(vs) != 3 {
		t.Errorf("expected 3 vertices, got %d", len(vs))
	}
	es := g.Edges()
	if len(es) < 2 {
		t.Errorf("expected at least 2 edges, got %d", len(es))
	}
}

func TestCloneIndependence(t *testing.T) {
	g := NewGraph(false, true)
	g.AddEdge("A", "B", 3)
	copyG := g.Clone()
	// изменим вес в оригинале
	edges := g.adjacencyList["A"]["B"]
	edges[0].Weight = 42
	// в копии должно остаться старое значение
	if copyG.adjacencyList["A"]["B"][0].Weight != 3 {
		t.Errorf("clone should be deep copy of edges")
	}
}
