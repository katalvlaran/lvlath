package matrix

import (
	"testing"

	"github.com/katalvlaran/lvlath/graph/core"
)

func TestIncidenceMatrixBasic(t *testing.T) {
	g := core.NewGraph(true, true)
	// A→B, B→C
	g.AddEdge("A", "B", 1)
	g.AddEdge("B", "C", 2)

	m := NewIncidenceMatrix(g)
	// should have 3 vertices × 2 edges
	if len(m.Data) != 3 || len(m.Data[0]) != 2 {
		t.Fatalf("unexpected dimensions: got %dx%d", len(m.Data), len(m.Data[0]))
	}

	// Check endpoints
	from, to, err := m.EdgeEndpoints(1)
	if err != nil {
		t.Fatal(err)
	}
	if from != "B" || to != "C" {
		t.Errorf("expected edge 1 endpoints B→C, got %s→%s", from, to)
	}

	// VertexIncidence for "B" should have one -1 and one +1
	row, err := m.VertexIncidence("B")
	if err != nil {
		t.Fatal(err)
	}
	// find exactly one -1 and one +1
	if !contains(row, -1) || !contains(row, 1) {
		t.Errorf("expected row to contain -1 and +1, got %v", row)
	}
}

func contains(slice []int, v int) bool {
	for _, x := range slice {
		if x == v {
			return true
		}
	}
	return false
}
