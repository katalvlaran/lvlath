// Package gridgraph provides utilities to treat a 2D grid of integer cell values
// as a graph. ConnectedComponents identifies contiguous regions (“islands”) of cells
// with equal values ≥ LandThreshold, grouped by that cell value.
package gridgraph

// ConnectedComponents returns a map from each land-value (int) to its
// list of connected components. Each component is represented as a slice of
// Cell structs (coordinates and value). Cells with value < LandThreshold
// are treated as water and excluded.
//
// Complexity: O(W×H×d) time, Memory: O(W×H), where d = number of neighbors (4 or 8).
func (gg *GridGraph) ConnectedComponents() map[int][][]Cell {
	// Early exit for empty grid
	if gg.Width == 0 || gg.Height == 0 {
		return map[int][][]Cell{}
	}

	total := gg.Width * gg.Height
	visited := make([]bool, total)
	components := make(map[int][][]Cell)
	offsets := gg.NeighborOffsets()

	// Traverse every cell
	for y := 0; y < gg.Height; y++ {
		for x := 0; x < gg.Width; x++ {
			value := gg.CellValues[y][x]
			if value < gg.LandThreshold {
				continue // water
			}
			startIdx := gg.index(x, y)
			if visited[startIdx] {
				continue
			}
			// BFS to collect one component of this value
			queue := []int{startIdx}
			visited[startIdx] = true
			var comp []Cell

			for qi := 0; qi < len(queue); qi++ {
				idx := queue[qi]
				x0, y0 := gg.Coordinate(idx)
				comp = append(comp, Cell{X: x0, Y: y0, Value: value})

				// Explore neighbors with same value
				for _, d := range offsets {
					nx, ny := x0+d[0], y0+d[1]
					if !gg.InBounds(nx, ny) {
						continue
					}
					if gg.CellValues[ny][nx] != value {
						continue
					}
					nIdx := gg.index(nx, ny)
					if !visited[nIdx] {
						visited[nIdx] = true
						queue = append(queue, nIdx)
					}
				}
			}

			// Append component to map under its value key
			components[value] = append(components[value], comp)
		}
	}

	return components
}
