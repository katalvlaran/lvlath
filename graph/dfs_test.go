package graph

import (
	"errors"
	"testing"
)

func TestDFS_EmptyGraph(t *testing.T) {
	g := NewGraph(false, false)
	_, err := g.DFS("X", nil)
	if !errors.Is(err, ErrDFSVertexNotFound) {
		t.Fatalf("expected ErrDFSVertexNotFound, got %v", err)
	}
}

func TestDFS_SingleNode(t *testing.T) {
	g := NewGraph(false, false)
	g.AddVertex(&Vertex{ID: "A"})
	res, err := g.DFS("A", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Order) != 1 || res.Order[0].ID != "A" {
		t.Errorf("expected [A], got %v", res.Order)
	}
}

func TestDFS_LinearGraph(t *testing.T) {
	g := NewGraph(false, false)
	g.AddEdge("A", "B", 0)
	g.AddEdge("B", "C", 0)
	res, err := g.DFS("A", nil)
	if err != nil {
		t.Fatal(err)
	}
	wantOrder := []string{"A", "B", "C"}
	for i, v := range res.Order {
		if v.ID != wantOrder[i] {
			t.Errorf("at %d expected %s, got %s", i, wantOrder[i], v.ID)
		}
	}
	if res.Parent["C"] != "B" {
		t.Errorf("expected parent[C]=B, got %s", res.Parent["C"])
	}
}

func TestDFS_Cycle(t *testing.T) {
	g := NewGraph(false, false)
	g.AddEdge("A", "B", 0)
	g.AddEdge("B", "C", 0)
	g.AddEdge("C", "A", 0)
	res, err := g.DFS("A", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Order) != 3 {
		t.Errorf("expected 3 unique visits, got %d", len(res.Order))
	}
}

func TestDFS_EarlyStop(t *testing.T) {
	g := NewGraph(false, false)
	g.AddEdge("A", "B", 0)
	g.AddEdge("B", "C", 0)
	opts := &DFSOptions{
		OnVisit: func(v *Vertex, depth int) error {
			if v.ID == "B" {
				return errors.New("halt")
			}
			return nil
		},
	}
	res, err := g.DFS("A", opts)
	if err == nil || err.Error() != "halt" {
		t.Fatalf("expected halt error, got %v", err)
	}
	// Order should contain A then B
	if len(res.Order) != 2 || res.Order[1].ID != "B" {
		t.Errorf("unexpected order %v", res.Order)
	}
}
