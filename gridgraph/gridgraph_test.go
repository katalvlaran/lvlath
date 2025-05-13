// File: gridgraph/gridgraph_test.go
package gridgraph

import (
	"reflect"
	"testing"
)

//----------------------------------------------------------------------------//
// From2D and InBounds
//----------------------------------------------------------------------------//

// TestFrom2D_Errors verifies that From2D correctly rejects empty or ragged inputs.
// Complexity: O(WH) for validation only, Memory: O(1) aside from error.
func TestFrom2D_Errors(t *testing.T) {
	cases := []struct {
		name string
		grid [][]int
		conn Connectivity
		err  error
	}{
		{"EmptyRows", [][]int{}, Conn4, ErrEmptyGrid},
		{"EmptyCols", [][]int{{}}, Conn8, ErrEmptyGrid},
		{"NonRectangular", [][]int{{1, 2}, {3}}, Conn4, ErrNonRectangular},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := From2D(tc.grid, tc.conn)
			if err != tc.err {
				t.Errorf("From2D(%v) error = %v; want %v", tc.grid, err, tc.err)
			}
		})
	}
}

// TestInBounds checks InBounds on a 3×2 grid.
// Scenario: width=3, height=2.
// Valid: (0,0),(2,1); Invalid: (-1,0),(3,1),(1,2).
func TestInBounds(t *testing.T) {
	grid := [][]int{
		{0, 1, 0},
		{1, 0, 1},
	}
	gg, _ := From2D(grid, Conn4)

	valid := [][2]int{{0, 0}, {2, 1}, {1, 1}}
	for _, xy := range valid {
		if !gg.InBounds(xy[0], xy[1]) {
			t.Errorf("InBounds(%d,%d)=false; want true", xy[0], xy[1])
		}
	}
	invalid := [][2]int{{-1, 0}, {3, 0}, {1, 2}, {2, -1}}
	for _, xy := range invalid {
		if gg.InBounds(xy[0], xy[1]) {
			t.Errorf("InBounds(%d,%d)=true; want false", xy[0], xy[1])
		}
	}
}

//----------------------------------------------------------------------------//
// ToCoreGraph
//----------------------------------------------------------------------------//

// TestToCoreGraph_Conn4 verifies horizontal and vertical edges only.
// Grid:
//
//	1 0
//	1 1
//
// Conn4: edges between (0,0)-(0,1) and (0,1)-(1,1), etc.
// Expected vertices=4, and no diagonal edges.
func TestToCoreGraph_Conn4(t *testing.T) {
	grid := [][]int{
		{1, 0},
		{1, 1},
	}
	gg, _ := From2D(grid, Conn4)
	cg := gg.ToCoreGraph()

	// Expect 4 vertices
	if len(cg.Vertices()) != 4 {
		t.Errorf("Vertices count = %d; want 4", len(cg.Vertices()))
	}

	// Horizontal & vertical edges should exist
	have := []struct{ u, v string }{
		{"0,0", "0,1"},
		{"0,1", "1,1"},
	}
	for _, e := range have {
		if !cg.HasEdge(e.u, e.v) {
			t.Errorf("Edge %s↔%s missing under Conn4", e.u, e.v)
		}
	}

	// Diagonals must NOT exist under Conn4
	if cg.HasEdge("0,0", "1,1") {
		t.Error("Unexpected diagonal edge 0,0↔1,1 under Conn4")
	}
}

// TestToCoreGraph_Conn8 verifies that diagonal connectivity is honored.
// Grid:
//
//	1 0
//	0 1
//
// Conn8: diagonal (0,0)-(1,1) should connect at zero manhattan cost.
// Expected HasEdge("0,0","1,1") == true.
func TestToCoreGraph_Conn8(t *testing.T) {
	grid := [][]int{
		{1, 0},
		{0, 1},
	}
	gg, _ := From2D(grid, Conn8)
	cg := gg.ToCoreGraph()

	// Diagonal edges should exist
	if !cg.HasEdge("0,0", "1,1") {
		t.Error("Expected diagonal edge 0,0↔1,1 under Conn8")
	}
	// Also verify the four cardinal neighbors
	if !cg.HasEdge("0,0", "0,1") {
		t.Error("Expected vertical edge 0,0↔0,1 under Conn8")
	}
	if !cg.HasEdge("0,0", "1,0") {
		t.Error("Expected horizontal edge 0,0↔1,0 under Conn8")
	}
}

//----------------------------------------------------------------------------//
// ConnectedComponents
//----------------------------------------------------------------------------//

// TestConnectedComponents_Basic tests two separate islands in a 3×3 grid.
// Grid:
//
//	1 1 0
//	1 0 0
//	0 0 1
//
// Conn4: expects two components of sizes {3,1}.
func TestConnectedComponents_Basic(t *testing.T) {
	grid := [][]int{
		{1, 1, 0},
		{1, 0, 0},
		{0, 0, 1},
	}
	gg, _ := From2D(grid, Conn4)
	comps := gg.ConnectedComponents()

	if len(comps) != 2 {
		t.Fatalf("Components count = %d; want 2", len(comps))
	}
	sizes := []int{len(comps[0]), len(comps[1])}
	want := []int{3, 1}
	if !reflect.DeepEqual(sizes, want) && !reflect.DeepEqual(sizes, []int{1, 3}) {
		t.Errorf("Component sizes = %v; want %v (any order)", sizes, want)
	}
}

// TestConnectedComponents_Conn8 merges diagonal cells into single component.
// Grid:
//
//	1 0 1
//	0 1 0
//	1 0 1
//
// Conn8: all ones connected through center -> single component of size 5.
func TestConnectedComponents_Conn8(t *testing.T) {
	grid := [][]int{
		{1, 0, 1},
		{0, 1, 0},
		{1, 0, 1},
	}
	gg, _ := From2D(grid, Conn8)
	comps := gg.ConnectedComponents()

	if len(comps) != 1 {
		t.Fatalf("Components count = %d; want 1", len(comps))
	}
	if len(comps[0]) != 5 {
		t.Errorf("Component size = %d; want 5", len(comps[0]))
	}
}
