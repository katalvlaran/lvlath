// Package gridgraph provides utilities to treat a 2D grid of integer cell values
// as a graph. ExpandIsland finds the minimum-cost path of water-to-land conversions
// connecting two sets of cells.
package gridgraph

// ExpandIsland computes a minimal-cost path to connect any cell in src to any cell in dst.
// src and dst are slices of Cell representing two island regions (as returned by ConnectedComponents()).
// Moving into a land Cell (Value ≥ LandThreshold) costs 0; into a water Cell costs 1.
// Returns the sequence of Cells along the shortest path (including endpoints) and total cost.
// O(W×H×d) time and O(W×H) memory.
func (gg *GridGraph) ExpandIsland(src, dst []Cell) (path []Cell, cost int, err error) {
	if len(src) == 0 || len(dst) == 0 {
		return nil, 0, ErrComponentIndex
	}
	// Map destination indices for O(1) lookup
	N := gg.Width * gg.Height
	dstSet := make(map[int]struct{}, len(dst))
	for _, c := range dst {
		idx := gg.index(c.X, c.Y)
		dstSet[idx] = struct{}{}
	}

	// Distance and previous arrays
	const inf = int(^uint(0) >> 1)
	dist := make([]int, N)
	prev := make([]int, N)
	for i := range dist {
		dist[i] = inf
		prev[i] = -1
	}

	// Custom deque implementation
	capDeque := N + 1
	deque := make([]int, capDeque)
	head, tail := 0, 0

	// Initialize deque with all source cells
	for _, c := range src {
		i := gg.index(c.X, c.Y)
		dist[i] = 0
		// push front
		head = (head - 1 + capDeque) % capDeque
		deque[head] = i
	}

	offsets := gg.NeighborOffsets()
	target := -1

	// 0-1 BFS
	for head != tail {
		// pop front
		u := deque[head]
		head = (head + 1) % capDeque
		// Check if reached any dst
		if _, ok := dstSet[u]; ok {
			target = u
			break
		}
		// Explore neighbors
		x0, y0 := gg.Coordinate(u)
		for _, d := range offsets {
			nx, ny := x0+d[0], y0+d[1]
			if !gg.InBounds(nx, ny) {
				continue
			}
			v := gg.index(nx, ny)
			step := 0
			if gg.CellValues[ny][nx] < gg.LandThreshold {
				step = 1
			}
			nd := dist[u] + step
			if nd < dist[v] {
				dist[v] = nd
				prev[v] = u
				if step == 0 {
					// push front
					head = (head - 1 + capDeque) % capDeque
					deque[head] = v
				} else {
					// push back
					deque[tail] = v
					tail = (tail + 1) % capDeque
				}
			}
		}
	}

	if target < 0 {
		return nil, 0, ErrNoPath
	}

	// Reconstruct path of indices
	var idxPath []int
	for at := target; at >= 0; at = prev[at] {
		idxPath = append([]int{at}, idxPath...)
	}

	// Convert indices to Cells
	path = make([]Cell, len(idxPath))
	for i, idx := range idxPath {
		x, y := gg.Coordinate(idx)
		path[i] = Cell{X: x, Y: y, Value: gg.CellValues[y][x]}
	}
	cost = dist[target]

	return path, cost, nil
}
