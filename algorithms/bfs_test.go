package algorithms

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/katalvlaran/lvlath/core"
)

func TestBFS_EmptyGraph(t *testing.T) {
	g := core.NewGraph(false, false)
	_, err := BFS(g, "X", nil)
	if !errors.Is(err, ErrVertexNotFound) {
		t.Fatalf("expected ErrVertexNotFound, got %v", err)
	}
}

func TestBFS_SingleNode(t *testing.T) {
	g := core.NewGraph(false, false)
	g.AddVertex(&core.Vertex{ID: "A"})
	res, err := BFS(g, "A", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := extractIDs(res.Order); !reflect.DeepEqual(got, []string{"A"}) {
		t.Errorf("Order = %v; want [A]", got)
	}
	if d := res.Depth["A"]; d != 0 {
		t.Errorf("Depth[A] = %d; want 0", d)
	}
	if len(res.Parent) != 0 {
		t.Errorf("Parent should be empty, got %v", res.Parent)
	}
}

func TestBFS_LinearGraph(t *testing.T) {
	g := core.NewGraph(false, false)
	g.AddEdge("A", "B", 0)
	g.AddEdge("B", "C", 0)
	res, err := BFS(g, "A", nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"A", "B", "C"}
	if got := extractIDs(res.Order); !reflect.DeepEqual(got, want) {
		t.Errorf("Order = %v; want %v", got, want)
	}
	if res.Depth["C"] != 2 {
		t.Errorf("Depth[C] = %d; want 2", res.Depth["C"])
	}
	if parent := res.Parent["C"]; parent != "B" {
		t.Errorf("Parent[C] = %q; want B", parent)
	}
}

func TestBFS_Cycle(t *testing.T) {
	g := core.NewGraph(false, false)
	g.AddEdge("A", "B", 0)
	g.AddEdge("B", "C", 0)
	g.AddEdge("C", "A", 0)
	res, err := BFS(g, "A", nil)
	if err != nil {
		t.Fatal(err)
	}
	got := extractIDs(res.Order)
	if len(got) != 3 {
		t.Errorf("visited %d vertices; want 3", len(got))
	}
}

func TestBFS_EarlyStop(t *testing.T) {
	g := core.NewGraph(false, false)
	g.AddEdge("A", "B", 0)
	g.AddEdge("B", "C", 0)

	opts := &BFSOptions{
		OnVisit: func(v *core.Vertex, depth int) error {
			if v.ID == "B" {
				return errors.New("stop at B")
			}
			return nil
		},
	}
	res, err := BFS(g, "A", opts)
	if err == nil || err.Error() != `OnVisit error at "B": stop at B` {
		t.Fatalf("expected stop error at B, got %v", err)
	}
	if got := extractIDs(res.Order); !reflect.DeepEqual(got, []string{"A", "B"}) {
		t.Errorf("Order = %v; want [A B]", got)
	}
}

// Test that cancellation via context works.
func TestBFS_Cancellation(t *testing.T) {
	g := core.NewGraph(false, false)
	// large chain
	for i := 0; i < 1000; i++ {
		// build cyclic IDs 'A'...'Z' by casting to rune first
		r1 := 'A' + rune(i%26)
		r2 := 'A' + rune((i+1)%26)
		id1 := string(r1)
		id2 := string(r2)
		g.AddEdge(id1, id2, 0)
	}
	ctx, cancel := context.WithCancel(context.Background())
	// cancel immediately
	cancel()
	_, err := BFS(g, "A", &BFSOptions{Ctx: ctx})
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

// extractIDs helper for tests.
func extractIDs(vs []*core.Vertex) []string {
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = v.ID
	}

	return out
}
