package algorithms

import (
	"errors"
	"testing"

	"github.com/katalvlaran/lvlath/graph/core"
)

func TestDijkstra_UnweightedError(t *testing.T) {
	g := core.NewGraph(false, false) // undirected, unweighted
	g.AddVertex(&core.Vertex{ID: "A"})
	_, _, err := Dijkstra(g, "A")
	if !errors.Is(err, ErrDijkstraNotWeighted) {
		t.Fatalf("expected ErrDijkstraNotWeighted, got %v", err)
	}
}

func TestDijkstra_VertexNotFound(t *testing.T) {
	g := core.NewGraph(false, true)
	_, _, err := Dijkstra(g, "X")
	if !errors.Is(err, ErrVertexNotFound) {
		t.Fatalf("expected ErrVertexNotFound, got %v", err)
	}
}

func TestDijkstra_SimpleTriangle(t *testing.T) {
	// A–B (1), B–C (2), A–C (5)
	g := core.NewGraph(false, true)
	g.AddEdge("A", "B", 1)
	g.AddEdge("B", "C", 2)
	g.AddEdge("A", "C", 5)

	dist, parent, err := Dijkstra(g, "A")
	if err != nil {
		t.Fatal(err)
	}
	// Check direct vs indirect path
	if got := dist["C"]; got != 3 {
		t.Errorf("dist[C]=%d; want 3", got)
	}
	if parent["C"] != "B" {
		t.Errorf("parent[C]=%q; want B", parent["C"])
	}
}

func TestDijkstra_MultiplePaths(t *testing.T) {
	// Graph:
	// A→B(2), A→C(1), C→B(1), B→D(3), C→D(5)
	g := core.NewGraph(true, true)
	g.AddEdge("A", "B", 2)
	g.AddEdge("A", "C", 1)
	g.AddEdge("C", "B", 1)
	g.AddEdge("B", "D", 3)
	g.AddEdge("C", "D", 5)

	dist, parent, err := Dijkstra(g, "A")
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		vertex string
		wantD  int64
		wantP  string
	}{
		{"B", 2, "C"}, // A→C→B
		{"D", 5, "B"}, // A→C→B→D
		{"C", 1, "A"}, // A→C
	}
	for _, c := range cases {
		if dist[c.vertex] != c.wantD {
			t.Errorf("dist[%s]=%d; want %d", c.vertex, dist[c.vertex], c.wantD)
		}
		if parent[c.vertex] != c.wantP {
			t.Errorf("parent[%s]=%q; want %q", c.vertex, parent[c.vertex], c.wantP)
		}
	}
}
