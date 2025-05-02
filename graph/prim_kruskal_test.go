package graph

import (
	"testing"
)

func makeTriangle() *Graph {
	g := NewGraph(false, true)
	g.AddEdge("A", "B", 1)
	g.AddEdge("B", "C", 2)
	g.AddEdge("A", "C", 3)
	return g
}

func TestPrim_Triangle(t *testing.T) {
	g := makeTriangle()
	mst, sum, err := g.Prim("A")
	if err != nil {
		t.Fatal(err)
	}
	if sum != 3 {
		t.Errorf("expected total 3, got %d", sum)
	}
	if len(mst) != 2 {
		t.Errorf("expected 2 edges, got %d", len(mst))
	}
}

func TestKruskal_Triangle(t *testing.T) {
	g := makeTriangle()
	mst, sum, err := g.Kruskal()
	if err != nil {
		t.Fatal(err)
	}
	if sum != 3 {
		t.Errorf("expected total 3, got %d", sum)
	}
	if len(mst) != 2 {
		t.Errorf("expected 2 edges, got %d", len(mst))
	}
}
