package bfs_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/katalvlaran/lvlath/bfs"
	"github.com/katalvlaran/lvlath/core"
)

// TestBFS_Errors verifies that invalid inputs and options are rejected.
func TestBFS_Errors(t *testing.T) {
	// nil graph
	if _, err := bfs.BFS(nil, "A"); !errors.Is(err, bfs.ErrGraphNil) {
		t.Errorf("nil graph: want ErrGraphNil, got %v", err)
	}
	// start vertex not found
	g := core.NewGraph()
	if _, err := bfs.BFS(g, "missing"); !errors.Is(err, bfs.ErrStartVertexNotFound) {
		t.Errorf("missing start: want ErrStartVertexNotFound, got %v", err)
	}
	// weighted graph unsupported
	gW := core.NewGraph(core.WithWeighted())
	gW.AddVertex("A")
	if _, err := bfs.BFS(gW, "A"); !errors.Is(err, bfs.ErrWeightedGraph) {
		t.Errorf("weighted graph: want ErrWeightedGraph, got %v", err)
	}
	// negative MaxDepth is a violation
	g2 := core.NewGraph()
	g2.AddVertex("A")
	if _, err := bfs.BFS(g2, "A", bfs.WithMaxDepth(-1)); !errors.Is(err, bfs.ErrOptionViolation) {
		t.Errorf("negative depth: want ErrOptionViolation, got %v", err)
	}
}

// TestBFS_SimpleTraversal covers the trivial one-vertex graph.
func TestBFS_SimpleTraversal(t *testing.T) {
	g := core.NewGraph()
	g.AddVertex("A")
	res, err := bfs.BFS(g, "A")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := []string{"A"}; !reflect.DeepEqual(res.Order, want) {
		t.Errorf("Order = %v; want %v", res.Order, want)
	}
	if d := res.Depth["A"]; d != 0 {
		t.Errorf("Depth[A] = %d; want 0", d)
	}
}

// TestCycleAndDepths covers a simple cycle and checks depths.
func TestCycleAndDepths(t *testing.T) {
	// A–B–C–D–A undirected cycle
	g := core.NewGraph(core.WithLoops(), core.WithMultiEdges())
	g.AddEdge("A", "B", 0)
	g.AddEdge("B", "C", 0)
	g.AddEdge("C", "D", 0)
	g.AddEdge("D", "A", 0)

	res, err := bfs.BFS(g, "A")
	if err != nil {
		t.Fatal(err)
	}
	// Must start at A
	if res.Order[0] != "A" {
		t.Errorf("first vertex = %s; want A", res.Order[0])
	}
	// Next two must be B and D in any order
	layer1 := map[string]bool{res.Order[1]: true, res.Order[2]: true}
	if !layer1["B"] || !layer1["D"] {
		t.Errorf("depth-1 layer = %v; want {B,D}", res.Order[1:3])
	}
	// Finally C
	if res.Order[3] != "C" {
		t.Errorf("last vertex = %s; want C", res.Order[3])
	}

	// Depth checks
	if got, want := res.Depth["A"], 0; got != want {
		t.Errorf("Depth[A] = %d; want %d", got, want)
	}
	for _, v := range []string{"B", "D"} {
		if got, want := res.Depth[v], 1; got != want {
			t.Errorf("Depth[%s] = %d; want %d", v, got, want)
		}
	}
	if got, want := res.Depth["C"], 2; got != want {
		t.Errorf("Depth[C] = %d; want %d", got, want)
	}
}

// TestBFS_Disconnected ensures BFS only explores the component of the start vertex.
func TestBFS_Disconnected(t *testing.T) {
	g := core.NewGraph()
	g.AddEdge("X", "Y", 0) // component 1
	g.AddEdge("P", "Q", 0) // component 2

	resX, _ := bfs.BFS(g, "X")
	if !reflect.DeepEqual(resX.Order, []string{"X", "Y"}) {
		t.Errorf("From X: got %v; want [X Y]", resX.Order)
	}
	resP, _ := bfs.BFS(g, "P")
	if !reflect.DeepEqual(resP.Order, []string{"P", "Q"}) {
		t.Errorf("From P: got %v; want [P Q]", resP.Order)
	}
}

// TestBFS_MaxDepth verifies WithMaxDepth behavior for positive, zero (no limit), and large depths.
func TestBFS_MaxDepth(t *testing.T) {
	g := core.NewGraph()
	g.AddEdge("A", "B", 0)
	g.AddEdge("B", "C", 0)
	// depth = 1 should only visit A,B
	if res, _ := bfs.BFS(g, "A", bfs.WithMaxDepth(1)); !reflect.DeepEqual(res.Order, []string{"A", "B"}) {
		t.Errorf("MaxDepth=1: got %v; want [A B]", res.Order)
	}
	// depth = 0 => explicit no limit => visits all
	if res, _ := bfs.BFS(g, "A", bfs.WithMaxDepth(0)); !reflect.DeepEqual(res.Order, []string{"A", "B", "C"}) {
		t.Errorf("MaxDepth=0: got %v; want [A B C]", res.Order)
	}
	// depth > graph size => same full traversal
	if res, _ := bfs.BFS(g, "A", bfs.WithMaxDepth(10)); !reflect.DeepEqual(res.Order, []string{"A", "B", "C"}) {
		t.Errorf("MaxDepth=10: got %v; want [A B C]", res.Order)
	}
}

// TestBFS_FilterNeighbor shows how filtering prunes certain edges.
func TestBFS_FilterNeighbor(t *testing.T) {
	g := core.NewGraph()
	g.AddEdge("A", "B", 0)
	g.AddEdge("B", "C", 0)
	// filter out B→C
	res, _ := bfs.BFS(g, "A",
		bfs.WithFilterNeighbor(func(curr, nbr string) bool {
			return !(curr == "B" && nbr == "C")
		}),
	)
	if want := []string{"A", "B"}; !reflect.DeepEqual(res.Order, want) {
		t.Errorf("FilterNeighbor: got %v; want %v", res.Order, want)
	}
}

// TestBFS_SelfLoopAndParallelDedup ensures that loops and parallel edges do not enqueue twice.
func TestBFS_SelfLoopAndParallelDedup(t *testing.T) {
	g := core.NewGraph(core.WithLoops(), core.WithMultiEdges())
	g.AddEdge("A", "A", 0) // self-loop
	g.AddEdge("A", "B", 0)
	g.AddEdge("A", "B", 0) // parallel
	res, _ := bfs.BFS(g, "A")
	if want := []string{"A", "B"}; !reflect.DeepEqual(res.Order, want) {
		t.Errorf("SelfLoop/Parallel: got %v; want %v", res.Order, want)
	}
}

// TestBFS_Hooks asserts that hooks fire in the expected sequence and count.
func TestBFS_Hooks(t *testing.T) {
	g := core.NewGraph()
	g.AddEdge("A", "B", 0)
	g.AddEdge("B", "C", 0)

	var enq, deq, vis []string
	makeEntry := func(prefix, id string, d int) string {
		return prefix + ":" + id + "@" + strconv.Itoa(d)
	}

	_, err := bfs.BFS(
		g, "A",
		bfs.WithOnEnqueue(func(id string, d int) { enq = append(enq, makeEntry("e", id, d)) }),
		bfs.WithOnDequeue(func(id string, d int) { deq = append(deq, makeEntry("d", id, d)) }),
		bfs.WithOnVisit(func(id string, d int) error { vis = append(vis, makeEntry("v", id, d)); return nil }),
	)
	if err != nil {
		t.Fatal(err)
	}

	// We expect BFS depths A@0, B@1, C@2
	wantDepths := []string{"A@0", "B@1", "C@2"}
	for i, suffix := range wantDepths {
		if !strings.HasSuffix(enq[i], suffix) {
			t.Errorf("OnEnqueue[%d] = %q, want suffix %q", i, enq[i], suffix)
		}
		if !strings.HasSuffix(deq[i], suffix) {
			t.Errorf("OnDequeue[%d] = %q, want suffix %q", i, deq[i], suffix)
		}
		if !strings.HasSuffix(vis[i], suffix) {
			t.Errorf("OnVisit[%d] = %q, want suffix %q", i, vis[i], suffix)
		}
	}
}

// TestBFS_PathTo covers both trivial (start→start) and unreachable targets.
func TestBFS_PathTo(t *testing.T) {
	g := core.NewGraph()
	g.AddVertex("X")
	res, _ := bfs.BFS(g, "X")
	if path, _ := res.PathTo("X"); !reflect.DeepEqual(path, []string{"X"}) {
		t.Errorf("PathTo start: got %v; want [X]", path)
	}
	_, err := res.PathTo("Y")
	if err == nil || !strings.Contains(err.Error(), "no path") {
		t.Errorf("PathTo unreachable: expected error, got %v", err)
	}
}

// TestBFS_Cancellation verifies that a cancelled context halts BFS promptly.
func TestBFS_Cancellation(t *testing.T) {
	g := core.NewGraph()
	// build a longer chain
	for i := 0; i < 100; i++ {
		u, v := fmt.Sprintf("v%d", i), fmt.Sprintf("v%d", i+1)
		g.AddEdge(u, v, 0)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediate
	if _, err := bfs.BFS(g, "v0", bfs.WithContext(ctx)); !errors.Is(err, context.Canceled) {
		t.Errorf("Cancellation: want context.Canceled, got %v", err)
	}
}

// TestBFS_ConcurrentSafety ensures two concurrent BFS runs on the same graph do not interfere.
func TestBFS_ConcurrentSafety(t *testing.T) {
	g := core.NewGraph()
	g.AddEdge("A", "B", 0)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() { _, err := bfs.BFS(g, "A"); errs <- err }()
	}
	for i := 0; i < 2; i++ {
		if err := <-errs; err != nil {
			t.Errorf("Concurrent run #%d: unexpected error %v", i, err)
		}
	}
}
