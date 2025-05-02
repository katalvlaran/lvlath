package graph

import (
	"testing"
)

func TestDijkstra_Simple(t *testing.T) {
	g := NewGraph(true, true)
	g.AddEdge("A", "B", 1)
	g.AddEdge("B", "C", 2)
	g.AddEdge("A", "C", 5)

	dist, parent, err := g.Dijkstra("A")
	if err != nil {
		t.Fatal(err)
	}
	if dist["C"] != 3 {
		t.Errorf("expected dist C=3, got %d", dist["C"])
	}
	if parent["C"] != "B" {
		t.Errorf("expected parent[C]=B, got %s", parent["C"])
	}
}
