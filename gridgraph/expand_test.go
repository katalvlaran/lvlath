// File: gridgraph/expand_test.go
package gridgraph

import (
	"reflect"
	"testing"
)

// helper to convert (x,y) to index
func idx(gg *GridGraph, x, y int) int {
	return gg.index(x, y)
}

// TestExpandIsland_BasicLine tests a simple 1×3 line with a single water cell between two land cells.
// Grid: [1,0,1], Conn4
// Expected: must convert the middle cell at cost 1, path indices [0,1,2].
func TestExpandIsland_BasicLine(t *testing.T) {
	grid := [][]int{{1, 0, 1}}
	gg, err := From2D(grid, Conn4)
	if err != nil {
		t.Fatalf("From2D error: %v", err)
	}
	comps := gg.ConnectedComponents()
	if len(comps) != 2 {
		t.Fatalf("found %d components; want 2", len(comps))
	}

	path, cost, err := gg.ExpandIsland(0, 1)
	if err != nil {
		t.Fatalf("ExpandIsland error: %v", err)
	}

	wantCost := 1
	wantPath := []int{idx(gg, 0, 0), idx(gg, 1, 0), idx(gg, 2, 0)}

	if cost != wantCost {
		t.Errorf("cost = %d; want %d", cost, wantCost)
	}
	if !reflect.DeepEqual(path, wantPath) {
		t.Errorf("path = %v; want %v", path, wantPath)
	}
}

// TestExpandIsland_MediumRow tests a 1×5 line where two land cells at ends require converting 3 water cells.
// Grid: [1,0,0,0,1], Conn4
// Expected cost = 3, path length = 5.
func TestExpandIsland_MediumRow(t *testing.T) {
	grid := [][]int{{1, 0, 0, 0, 1}}
	gg, _ := From2D(grid, Conn4)
	path, cost, err := gg.ExpandIsland(0, 1)
	if err != nil {
		t.Fatalf("ExpandIsland error: %v", err)
	}

	if cost != 3 {
		t.Errorf("cost = %d; want 3", cost)
	}
	if len(path) != 5 {
		t.Errorf("path length = %d; want 5", len(path))
	}
}

// TestExpandIsland_Diagonal8 tests diagonal connectivity allowing zero-cost direct diagonal path.
// Grid:
//
//	1 0
//	0 1
//
// Conn8: the two land cells touch at corner.
// Expected cost = 0, path = [0,3].
func TestExpandIsland_Diagonal8(t *testing.T) {
	grid := [][]int{
		{1, 0},
		{0, 1},
	}
	gg, _ := From2D(grid, Conn8)
	comps := gg.ConnectedComponents()
	if len(comps) != 1 {
		// All land connected by Conn8 -> one component; treat 0→0 => cost 0, path single cell.
		path, cost, err := gg.ExpandIsland(0, 0)
		if err != nil {
			t.Fatalf("ExpandIsland error: %v", err)
		}
		if cost != 0 {
			t.Errorf("cost = %d; want 0", cost)
		}
		if len(path) != 1 || path[0] != comps[0][0] {
			t.Errorf("path = %v; want [%d]", path, comps[0][0])
		}
	} else {
		// Two separate comps (unexpected), test explicitly 0→1
		path, cost, err := gg.ExpandIsland(0, 1)
		if err != nil {
			t.Fatalf("ExpandIsland error: %v", err)
		}
		if cost != 0 {
			t.Errorf("cost = %d; want 0", cost)
		}
		want := []int{idx(gg, 0, 0), idx(gg, 1, 1)}
		if !reflect.DeepEqual(path, want) {
			t.Errorf("path = %v; want %v", path, want)
		}
	}
}

// TestExpandIsland_InvalidIndices ensures invalid component indices yield ErrComponentIndex.
func TestExpandIsland_InvalidIndices(t *testing.T) {
	grid := [][]int{{1, 0, 1}}
	gg, _ := From2D(grid, Conn4)

	_, _, err := gg.ExpandIsland(-1, 1)
	if err != ErrComponentIndex {
		t.Errorf("src=-1: got %v; want ErrComponentIndex", err)
	}
	_, _, err = gg.ExpandIsland(0, 2)
	if err != ErrComponentIndex {
		t.Errorf("dst=2: got %v; want ErrComponentIndex", err)
	}
}
