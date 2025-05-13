// Package gridgraph provides utilities to treat a 2D grid of integer cell values
// as a graph. It supports:
//
//   - Four‐ or eight‐connectivity (Conn4 or Conn8)
//   - Conversion to a *core.Graph
//   - Identification of connected components of “land” cells
//   - Shortest‐path expansions between components
//
// Cells with value 0 are considered “water” (non‐land); cells with value ≥1 are “land”.
package gridgraph

import (
	"fmt"

	"github.com/katalvlaran/lvlath/core"
)

// Connectivity selects neighbor offsets: orthogonal (Conn4) or including diagonals (Conn8).
type Connectivity int

const (
	// Conn4 uses 4-directional connectivity: N, E, S, W.
	Conn4 Connectivity = iota
	// Conn8 uses 8-directional connectivity: N, NE, E, SE, S, SW, W, NW.
	Conn8
)

// GridGraph treats a 2D integer grid as a graph.
//
// Width, Height define dimensions; CellValues[y][x] is the cell at (x,y).
// Conn chooses adjacency. All operations cost O(W·H·d) time/memory.
type GridGraph struct {
	Width, Height int
	CellValues    [][]int
	Conn          Connectivity
}

// From2D constructs a GridGraph from a non-empty, rectangular 2D slice.
// Returns an error if rows are uneven or input is empty.
//
// Example:
//
//	grid := [][]int{
//	  {0,1,1},
//	  {1,1,0},
//	  {0,1,1},
//	}
//	gg, err := From2D(grid, Conn4)
//	// gg.Width==3, gg.Height==3, gg.Conn==Conn4
func From2D(values [][]int, conn Connectivity) (*GridGraph, error) {
	if len(values) == 0 || len(values[0]) == 0 {
		return nil, ErrEmptyGrid
	}
	h, w := len(values), len(values[0])
	for _, row := range values {
		if len(row) != w {
			return nil, ErrNonRectangular
		}
	}
	// Deep copy to prevent external mutation
	cells := make([][]int, h)
	for y := 0; y < h; y++ {
		cells[y] = make([]int, w)
		copy(cells[y], values[y])
	}
	return &GridGraph{Width: w, Height: h, CellValues: cells, Conn: conn}, nil
}

// InBounds reports whether (x,y) is inside the grid.
func (gg *GridGraph) InBounds(x, y int) bool {
	return x >= 0 && x < gg.Width && y >= 0 && y < gg.Height
}

// neighborOffsets returns the (dx,dy) offsets for the chosen connectivity.
func (gg *GridGraph) neighborOffsets() [][2]int {
	if gg.Conn == Conn8 {
		return [][2]int{
			{0, -1}, {1, -1}, {1, 0}, {1, 1},
			{0, 1}, {-1, 1}, {-1, 0}, {-1, -1},
		}
	}
	// Conn4
	return [][2]int{
		{0, -1}, {1, 0}, {0, 1}, {-1, 0},
	}
}

// cellID formats a Coordinate as "x,y".
func (gg *GridGraph) cellID(x, y int) string {
	return fmt.Sprintf("%d,%d", x, y)
}

// ToCoreGraph converts the GridGraph into a weighted, undirected *core.Graph.
// Each cell at (x,y) becomes a vertex with ID "x,y". Edges connect every pair
// of neighboring cells according to gg.Conn (4‐ or 8‐connectivity), each with
// unit weight (1).
//
// Complexity: O(W·H·d) time and O(W·H + E) memory, where d = 4 or 8, E ≈ W·H·d.
func (gg *GridGraph) ToCoreGraph() *core.Graph {
	g := core.NewGraph(false, true)
	// add vertices
	for y := 0; y < gg.Height; y++ {
		for x := 0; x < gg.Width; x++ {
			id := gg.cellID(x, y)
			g.AddVertex(&core.Vertex{
				ID:       id,
				Metadata: map[string]interface{}{"x": x, "y": y, "value": gg.CellValues[y][x]},
			})
		}
	}
	// add edges
	for y := 0; y < gg.Height; y++ {
		for x := 0; x < gg.Width; x++ {
			uID := gg.cellID(x, y)
			for _, d := range gg.neighborOffsets() {
				nx, ny := x+d[0], y+d[1]
				if !gg.InBounds(nx, ny) {
					continue
				}
				vID := gg.cellID(nx, ny)
				g.AddEdge(uID, vID, 1)
			}
		}
	}
	return g
}

// index maps (x,y) to a row‐major index: y*Width + x.
func (gg *GridGraph) index(x, y int) int {
	return y*gg.Width + x
}

// Coordinate reverts an index to (x,y).
func (gg *GridGraph) Coordinate(idx int) (x, y int) {
	return idx % gg.Width, idx / gg.Width
}
