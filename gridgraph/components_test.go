// File: gridgraph/components_test.go
package gridgraph

import (
	"reflect"
	"sort"
	"testing"
)

// TestConnectedComponents_Simple4 tests ConnectedComponents on a simple 4×3 grid
// with orthogonal connectivity (Conn4).
//
// Grid (1 = land, 0 = water):
//
//	0 1 1 0
//	1 1 0 0
//	0 0 1 1
//
// Expected: 2 islands of sizes 4 and 2.
//
// Complexity: O(W·H·4) time, O(W·H) memory.
func TestConnectedComponents_Simple4(t *testing.T) {
	grid := [][]int{
		{0, 1, 1, 0},
		{1, 1, 0, 0},
		{0, 0, 1, 1},
	}
	gg, err := From2D(grid, Conn4)
	if err != nil {
		t.Fatalf("From2D failed: %v", err)
	}

	comps := gg.ConnectedComponents()
	if len(comps) != 2 {
		t.Fatalf("got %d components; want 2", len(comps))
	}

	// Collect sizes and sort for comparison.
	sizes := []int{len(comps[0]), len(comps[1])}
	sort.Ints(sizes)
	want := []int{2, 4}
	if !reflect.DeepEqual(sizes, want) {
		t.Errorf("component sizes = %v; want %v", sizes, want)
	}
}

// TestConnectedComponents_Diagonal8 tests ConnectedComponents on a 5×5 grid
// using diagonal connectivity (Conn8) to catch “touching corners” islands.
//
// Grid:
//
//	1 0 0 0 1
//	0 1 0 1 0
//	0 0 1 0 0
//	0 1 0 1 0
//	1 0 0 0 1
//
// With Conn8, all 9 ones connect through diagonal hops into a single island.
// Expect: 1 component of size 9.
//
// Complexity: O(W·H·8) time, O(W·H) memory.
func TestConnectedComponents_Diagonal8(t *testing.T) {
	grid := [][]int{
		{1, 0, 0, 0, 1},
		{0, 1, 0, 1, 0},
		{0, 0, 1, 0, 0},
		{0, 1, 0, 1, 0},
		{1, 0, 0, 0, 1},
	}
	gg, err := From2D(grid, Conn8)
	if err != nil {
		t.Fatalf("From2D failed: %v", err)
	}

	comps := gg.ConnectedComponents()
	if len(comps) != 1 {
		t.Fatalf("got %d components; want 1", len(comps))
	}
	if size := len(comps[0]); size != 9 {
		t.Errorf("component size = %d; want 9", size)
	}
}

// TestConnectedComponents_EmptyAndAllWater tests edge cases:
//   - completely water grid → zero components
//   - single‐cell land grid → one component of size 1
func TestConnectedComponents_EmptyAndAllWater(t *testing.T) {
	// All water
	grid1 := [][]int{
		{0, 0},
		{0, 0},
	}
	gg1, _ := From2D(grid1, Conn4)
	comps1 := gg1.ConnectedComponents()
	if len(comps1) != 0 {
		t.Errorf("all-water: got %d components; want 0", len(comps1))
	}

	// Single land cell
	grid2 := [][]int{{0, 1}}
	gg2, _ := From2D(grid2, Conn4)
	comps2 := gg2.ConnectedComponents()
	if len(comps2) != 1 {
		t.Fatalf("single land: got %d components; want 1", len(comps2))
	}
	if len(comps2[0]) != 1 {
		t.Errorf("single land: component size = %d; want 1", len(comps2[0]))
	}
}

// TestConnectedComponents_InvalidRects ensures From2D rejects bad inputs.
func TestConnectedComponents_InvalidRects(t *testing.T) {
	if _, err := From2D(nil, Conn4); err != ErrEmptyGrid {
		t.Errorf("nil grid: got %v; want ErrEmptyGrid", err)
	}
	if _, err := From2D([][]int{{1}, {}}, Conn4); err != ErrNonRectangular {
		t.Errorf("jagged grid: got %v; want ErrNonRectangular", err)
	}
}
