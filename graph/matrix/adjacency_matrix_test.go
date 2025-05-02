package matrix

import (
	"reflect"
	"testing"

	"lvlath/graph/core"
)

func TestAdjacencyMatrixBasic(t *testing.T) {
	g := core.NewGraph(false, true)
	g.AddEdge("A", "B", 5)
	mat := NewAdjacencyMatrix(g)

	// Test Neighbors
	nb, err := mat.Neighbors("A")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(nb, []string{"B"}) {
		t.Errorf("expected [B], got %v", nb)
	}

	// Round-trip back to Graph
	g2 := mat.ToGraph(true)
	if !g2.HasEdge("A", "B") || !g2.HasEdge("B", "A") {
		t.Errorf("ToGraph failed to recreate undirected edge")
	}
}

func TestAdjacencyMatrixAddRemove(t *testing.T) {
	g := core.NewGraph(false, true)
	mat := NewAdjacencyMatrix(g)

	// unknown vertices should error
	if err := mat.AddEdge("X", "Y", 3); err == nil {
		t.Errorf("expected error for unknown vertices")
	}

	// Add vertices then test
	g.AddEdge("X", "Y", 3)
	mat = NewAdjacencyMatrix(g)
	if err := mat.RemoveEdge("X", "Y"); err != nil {
		t.Fatal(err)
	}
	nb, err := mat.Neighbors("X")
	if err != nil {
		t.Fatal(err)
	}
	if len(nb) != 0 {
		t.Errorf("expected no neighbors after RemoveEdge, got %v", nb)
	}
}
