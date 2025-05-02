package graph

import (
	"errors"
	"testing"
)

func TestBFS_EmptyGraph(t *testing.T) {
	g := NewGraph(false, false)
	_, err := g.BFS("X", nil)
	if !errors.Is(err, ErrVertexNotFound) {
		t.Fatalf("expected ErrVertexNotFound, got %v", err)
	}
}

func TestBFS_SingleNode(t *testing.T) {
	g := NewGraph(false, false)
	g.AddVertex(&Vertex{ID: "A"})
	res, err := g.BFS("A", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Order) != 1 || res.Order[0].ID != "A" {
		t.Errorf("expected order [A], got %v", res.Order)
	}
	if d := res.Depth["A"]; d != 0 {
		t.Errorf("expected depth[A]=0, got %d", d)
	}
}

func TestBFS_LinearGraph(t *testing.T) {
	g := NewGraph(false, false)
	g.AddEdge("A", "B", 0)
	g.AddEdge("B", "C", 0)
	res, err := g.BFS("A", nil)
	if err != nil {
		t.Fatal(err)
	}
	wantOrder := []string{"A", "B", "C"}
	for i, v := range res.Order {
		if v.ID != wantOrder[i] {
			t.Errorf("at %d expected %s, got %s", i, wantOrder[i], v.ID)
		}
	}
	if res.Depth["C"] != 2 {
		t.Errorf("expected depth[C]=2, got %d", res.Depth["C"])
	}
	if p := res.Parent["C"]; p != "B" {
		t.Errorf("expected parent[C]=B, got %s", p)
	}
}

func TestBFS_Cycle(t *testing.T) {
	g := NewGraph(false, false)
	g.AddEdge("A", "B", 0)
	g.AddEdge("B", "C", 0)
	g.AddEdge("C", "A", 0)
	res, err := g.BFS("A", nil)
	if err != nil {
		t.Fatal(err)
	}
	visited := res.Order
	if len(visited) != 3 {
		t.Errorf("expected 3 unique visits, got %d", len(visited))
	}
}

func TestBFS_EarlyStop(t *testing.T) {
	g := NewGraph(false, false)
	g.AddEdge("A", "B", 0)
	g.AddEdge("B", "C", 0)
	opts := &BFSOptions{
		OnVisit: func(v *Vertex, depth int) error {
			if v.ID == "B" {
				return errors.New("stop at B")
			}
			return nil
		},
	}
	res, err := g.BFS("A", opts)
	if err == nil || err.Error() != "stop at B" {
		t.Fatalf("expected stop error, got %v", err)
	}
	// Order should contain A then B
	if len(res.Order) != 2 || res.Order[1].ID != "B" {
		t.Errorf("unexpected order %v", res.Order)
	}
}
