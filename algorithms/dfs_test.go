package algorithms

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/katalvlaran/lvlath/core"
)

func TestDFS_EmptyGraph(t *testing.T) {
	g := core.NewGraph(false, false)
	_, err := DFS(g, "X", nil)
	if !errors.Is(err, ErrDFSVertexNotFound) {
		t.Fatalf("expected ErrDFSVertexNotFound, got %v", err)
	}
}

func TestDFS_SingleNode(t *testing.T) {
	g := core.NewGraph(false, false)
	g.AddVertex(&core.Vertex{ID: "A"})
	res, err := DFS(g, "A", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := extractIDs(res.Order); !reflect.DeepEqual(got, []string{"A"}) {
		t.Errorf("Order = %v; want [A]", got)
	}
	if d := res.Depth["A"]; d != 0 {
		t.Errorf("Depth[A] = %d; want 0", d)
	}
}

func TestDFS_LinearGraph(t *testing.T) {
	g := core.NewGraph(false, false)
	g.AddEdge("A", "B", 0)
	g.AddEdge("B", "C", 0)
	res, err := DFS(g, "A", nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"A", "B", "C"}
	if got := extractIDs(res.Order); !reflect.DeepEqual(got, want) {
		t.Errorf("Order = %v; want %v", got, want)
	}
	if parent := res.Parent["C"]; parent != "B" {
		t.Errorf("Parent[C] = %q; want B", parent)
	}
}

func TestDFS_Cycle(t *testing.T) {
	g := core.NewGraph(false, false)
	g.AddEdge("A", "B", 0)
	g.AddEdge("B", "C", 0)
	g.AddEdge("C", "A", 0)
	res, err := DFS(g, "A", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Order) != 3 {
		t.Errorf("visited %d vertices; want 3", len(res.Order))
	}
}

func TestDFS_EarlyStop(t *testing.T) {
	g := core.NewGraph(false, false)
	g.AddEdge("A", "B", 0)
	g.AddEdge("B", "C", 0)

	opts := &DFSOptions{
		OnVisit: func(v *core.Vertex, depth int) error {
			if v.ID == "B" {
				return errors.New("halt at B")
			}
			return nil
		},
	}
	res, err := DFS(g, "A", opts)
	if err == nil || err.Error() != `OnVisit error at "B": halt at B` {
		t.Fatalf("expected halt error at B, got %v", err)
	}
	if got := extractIDs(res.Order); !reflect.DeepEqual(got, []string{"A", "B"}) {
		t.Errorf("Order = %v; want [A B]", got)
	}
}

func TestDFS_Cancellation(t *testing.T) {
	g := core.NewGraph(false, false)
	// build a deep chain to force cancellation
	for i := 0; i < 1000; i++ {
		u := fmt.Sprintf("N%d", i)
		v := fmt.Sprintf("N%d", i+1)
		g.AddEdge(u, v, 0)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediate cancellation

	_, err := DFS(g, "N0", &DFSOptions{Ctx: ctx})
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
