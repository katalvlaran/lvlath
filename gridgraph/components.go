package gridgraph

// ConnectedComponents finds all contiguous regions (“islands”) of land cells
// (CellValues[y][x] ≥ 1), according to gg.Conn connectivity.
// Returns a slice of components; each component is a slice of cell‐indices
// (row‐major) in arbitrary order.
//
// To convert an index back to (x,y), use coordinate(idx).
//
// Time:   O(W·H·d), where d = 4 or 8.
// Memory: O(W·H) for visited flags and output.
func (gg *GridGraph) ConnectedComponents() [][]int {
	total := gg.Width * gg.Height
	seen := make([]bool, total)
	var comps [][]int
	offsets := gg.neighborOffsets()

	for y := 0; y < gg.Height; y++ {
		for x := 0; x < gg.Width; x++ {
			if gg.CellValues[y][x] < 1 {
				continue // water
			}
			i0 := gg.index(x, y)
			if seen[i0] {
				continue
			}
			// BFS to collect component
			queue := []int{i0}
			seen[i0] = true
			var comp []int

			for qi := 0; qi < len(queue); qi++ {
				u := queue[qi]
				comp = append(comp, u)
				ux, uy := gg.Coordinate(u)
				for _, d := range offsets {
					vx, vy := ux+d[0], uy+d[1]
					if !gg.InBounds(vx, vy) || gg.CellValues[vy][vx] < 1 {
						continue
					}
					vi := gg.index(vx, vy)
					if !seen[vi] {
						seen[vi] = true
						queue = append(queue, vi)
					}
				}
			}
			comps = append(comps, comp)
		}
	}
	return comps
}
