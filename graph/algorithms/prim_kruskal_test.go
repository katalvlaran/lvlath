package algorithms

import (
	"errors"
	"reflect"
	"testing"

	"lvlath/graph/core"
)

// helper to build a simple triangle graph: A–B(1), B–C(2), A–C(3)
func makeTriangle() *core.Graph {
	g := core.NewGraph(false, true)
	g.AddEdge("A", "B", 1)
	g.AddEdge("B", "C", 2)
	g.AddEdge("A", "C", 3)
	return g
}

// extractEdges returns a set of edges described as "U-V" strings.
func extractEdges(edges []*core.Edge) map[string]bool {
	s := make(map[string]bool, len(edges))
	for _, e := range edges {
		// normalize order U-V
		u, v := e.From.ID, e.To.ID
		if u > v {
			u, v = v, u
		}
		s[u+"-"+v] = true
	}
	return s
}

func TestPrim_UnweightedOrDirected(t *testing.T) {
	g1 := core.NewGraph(false, false)
	_, _, err1 := Prim(g1, "A")
	if !errors.Is(err1, ErrNotWeighted) {
		t.Errorf("Prim on unweighted: got %v; want %v", err1, ErrNotWeighted)
	}
	g2 := core.NewGraph(true, true)
	_, _, err2 := Prim(g2, "A")
	if !errors.Is(err2, ErrNotWeighted) {
		t.Errorf("Prim on directed: got %v; want %v", err2, ErrNotWeighted)
	}
}

func TestPrim_VertexNotFound(t *testing.T) {
	g := core.NewGraph(false, true)
	_, _, err := Prim(g, "X")
	if !errors.Is(err, ErrVertexNotFound) {
		t.Errorf("Prim missing start: got %v; want %v", err, ErrVertexNotFound)
	}
}

func TestPrim_Triangle(t *testing.T) {
	g := makeTriangle()
	mst, sum, err := Prim(g, "A")
	if err != nil {
		t.Fatal(err)
	}
	if sum != 3 {
		t.Errorf("Prim sum = %d; want 3", sum)
	}
	if len(mst) != 2 {
		t.Errorf("Prim edge count = %d; want 2", len(mst))
	}
	got := extractEdges(mst)
	want := map[string]bool{"A-B": true, "B-C": true}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Prim edges = %v; want %v", got, want)
	}
}

func TestKruskal_UnweightedOrDirected(t *testing.T) {
	g1 := core.NewGraph(false, false)
	_, _, err1 := Kruskal(g1)
	if !errors.Is(err1, ErrNotWeighted) {
		t.Errorf("Kruskal on unweighted: got %v; want %v", err1, ErrNotWeighted)
	}
	g2 := core.NewGraph(true, true)
	_, _, err2 := Kruskal(g2)
	if !errors.Is(err2, ErrNotWeighted) {
		t.Errorf("Kruskal on directed: got %v; want %v", err2, ErrNotWeighted)
	}
}

func TestKruskal_Triangle(t *testing.T) {
	g := makeTriangle()
	mst, sum, err := Kruskal(g)
	if err != nil {
		t.Fatal(err)
	}
	if sum != 3 {
		t.Errorf("Kruskal sum = %d; want 3", sum)
	}
	if len(mst) != 2 {
		t.Errorf("Kruskal edge count = %d; want 2", len(mst))
	}
	got := extractEdges(mst)
	want := map[string]bool{"A-B": true, "B-C": true}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Kruskal edges = %v; want %v", got, want)
	}
}
