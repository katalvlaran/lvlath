// File: gridgraph/example_test.go
package gridgraph_test

import (
	"fmt"

	"github.com/katalvlaran/lvlath/gridgraph"
)

////////////////////////////////////////////////////////////////////////////////
// Example: ConnectedComponents
////////////////////////////////////////////////////////////////////////////////

// ExampleGridGraph_ConnectedComponents demonstrates how to identify
// contiguous “islands” of non-zero cells in a 2D grid.
// Scenario:
//
//   - Grid values: 0 = water, 1,2,3 = different land/resource IDs
//   - Conn4: 4-directional adjacency (N/E/S/W)
//   - Expect three islands:
//     – ID 1 cluster at {(1,0),(2,0),(0,1),(1,1)}
//     – ID 2 cluster at {(4,0),(3,1),(4,1),(2,2),(3,2)}
//     – ID 3 single cell at {(0,2)}
//
// Complexity: O(W·H·4), Memory: O(W·H)
func ExampleGridGraph_ConnectedComponents() {
	grid := [][]int{
		{0, 1, 1, 0, 2},
		{1, 1, 0, 2, 2},
		{3, 0, 2, 2, 0},
	}
	gg, _ := gridgraph.From2D(grid, gridgraph.Conn4)

	comps := gg.ConnectedComponents()
	fmt.Println("components:", len(comps))
	for i, comp := range comps {
		fmt.Printf("component %d:", i)
		for _, idx := range comp {
			x, y := gg.Coordinate(idx)
			fmt.Printf(" (%d,%d)", x, y)
		}
		fmt.Println()
	}

	// Output:
	// components: 3
	// component 0: (1,0) (2,0) (0,1) (1,1)
	// component 1: (4,0) (3,1) (4,1) (2,2) (3,2)
	// component 2: (0,2)
}

////////////////////////////////////////////////////////////////////////////////
// Example: ExpandIsland
////////////////////////////////////////////////////////////////////////////////

// ExampleGridGraph_ExpandIsland demonstrates computing the minimal
// water‐cell conversions to connect two islands in the grid.
// Scenario:
//
//   - Same grid and Conn4 as above.
//   - Connect component 0 (ID 1 cluster) to component 1 (ID 2 cluster).
//   - Each water cell converted costs 1.
//
// Expected: shortest path through water cells only, e.g. (1,1)->(1,2)->(2,2).
//
// Complexity: O(W·H) on average, Memory: O(W·H)
func ExampleGridGraph_ExpandIsland() {
	grid := [][]int{
		{0, 1, 1, 0, 2},
		{1, 1, 0, 2, 2},
		{3, 0, 2, 2, 0},
	}
	gg, _ := gridgraph.From2D(grid, gridgraph.Conn4)

	_ = gg.ConnectedComponents()
	// Connect comp 0 → comp 1:
	path, cost, _ := gg.ExpandIsland(0, 1)

	fmt.Printf("Convert %d water cells along path:\n", cost)
	for _, idx := range path {
		x, y := gg.Coordinate(idx)
		fmt.Printf("(%d,%d) ", x, y)
	}
	// Output:
	// Convert 2 water cells along path:
	// (1,1) (1,2) (2,2) (2,1) (3,1)
}
