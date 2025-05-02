package graph

import (
	"reflect"
	"testing"
)

func TestAdjacencyMatrix_Basic(t *testing.T) {
	g := NewGraph(false, true)
	g.AddEdge("A", "B", 5)
	mat := NewAdjacencyMatrix(g)

	// Check neighbors
	nb, err := mat.Neighbors("A")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(nb, []string{"B"}) {
		t.Errorf("expected [B], got %v", nb)
	}

	// Round-trip
	g2 := mat.ToGraph(true)
	if !g2.HasEdge("A", "B") || !g2.HasEdge("B", "A") {
		t.Errorf("ToGraph failed")
	}
}

func TestAdjacencyMatrix_AddRemove(t *testing.T) {
	g := NewGraph(false, true)
	mat := NewAdjacencyMatrix(g)
	if err := mat.AddEdge("X", "Y", 3); err == nil {
		t.Errorf("expected error for unknown vertices")
	}
	// build graph first
	g.AddEdge("X", "Y", 3)
	mat = NewAdjacencyMatrix(g)
	if err := mat.RemoveEdge("X", "Y"); err != nil {
		t.Fatal(err)
	}
	nb, _ := mat.Neighbors("X")
	if len(nb) != 0 {
		t.Errorf("expected no neighbors after RemoveEdge, got %v", nb)
	}
}
