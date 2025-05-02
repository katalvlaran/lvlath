package core

import (
	"reflect"
	"testing"
)

func TestNewGraphFlags(t *testing.T) {
	for _, tc := range []struct {
		d, w bool
	}{
		{false, false},
		{false, true},
		{true, false},
		{true, true},
	} {
		g := NewGraph(tc.d, tc.w)
		if g.Directed() != tc.d {
			t.Errorf("Directed() = %v; want %v", g.Directed(), tc.d)
		}
		if g.Weighted() != tc.w {
			t.Errorf("Weighted() = %v; want %v", g.Weighted(), tc.w)
		}
	}
}

func TestCloneEmpty(t *testing.T) {
	g := NewGraph(false, true)
	// add vertices and edges
	g.AddVertex(&Vertex{ID: "A"})
	g.AddEdge("A", "B", 2)
	g.AddEdge("B", "C", 3)

	clone := g.CloneEmpty()
	// Should have same vertices
	origIDs := sortedIDs(g.Vertices())
	clonedIDs := sortedIDs(clone.Vertices())
	if !reflect.DeepEqual(origIDs, clonedIDs) {
		t.Errorf("CloneEmpty vertices = %v; want %v", clonedIDs, origIDs)
	}
	// But no edges
	if len(clone.Edges()) != 0 {
		t.Errorf("CloneEmpty edges count = %d; want 0", len(clone.Edges()))
	}
}

func TestCloneIndependence(t *testing.T) {
	g := NewGraph(false, true)
	g.AddEdge("A", "B", 5)

	// Clone before modifying the original graph's edges
	clone := g.Clone()

	// Now mutate the original graph's edge weight
	edges := g.Edges()
	edges[0].Weight = 42

	// The clone should retain the original weight 5
	cloneEdges := clone.Edges()
	if cloneEdges[0].Weight != 5 {
		t.Errorf("Clone edge weight = %d; want 5 (original unaffected)", cloneEdges[0].Weight)
	}
}

func TestVerticesMapReadOnly(t *testing.T) {
	g := NewGraph(false, false)
	g.AddVertex(&Vertex{ID: "X"})
	vm := g.VerticesMap()
	vm["Y"] = &Vertex{ID: "Y"} // attempt mutation
	// Should not affect underlying graph
	if g.HasVertex("Y") {
		t.Error("VerticesMap should expose read-only map; mutation leaked")
	}
}
