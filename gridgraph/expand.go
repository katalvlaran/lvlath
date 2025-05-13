package gridgraph

import (
	"container/list"
)

// ExpandIsland finds a minimum‐conversion path of “water” cells (CellValues<1)
// to connect any cell in component srcComp to any cell in component dstComp,
// as identified by ConnectedComponents(). Each water‐cell conversion costs 1.
// Returns the sequence of cell‐indices (row‐major) representing the path
// (including the start and end land cells) and the total conversion cost.
//
// Behavior:
//  1. Validate component indices.
//  2. Multi‐source 0–1‐BFS from all srcComp cells:
//     • Moving into an existing land cell   → cost 0
//     • Moving into a water cell             → cost 1
//  3. Stop when any dstComp cell is reached.
//  4. Reconstruct path via predecessors map.
//
// Complexity: O(W·H · log(W·H)) worst‐case if using a priority deque,
//
//	but on average O(W·H) for grid sizes.
//
// Memory:     O(W·H) for distance, visited, and prev pointers.
func (gg *GridGraph) ExpandIsland(srcComp, dstComp int) (path []int, cost int, err error) {
	comps := gg.ConnectedComponents()
	if srcComp < 0 || srcComp >= len(comps) || dstComp < 0 || dstComp >= len(comps) {
		return nil, 0, ErrComponentIndex
	}
	src := comps[srcComp]
	dstSet := make(map[int]struct{}, len(comps[dstComp]))
	for _, i := range comps[dstComp] {
		dstSet[i] = struct{}{}
	}

	N := gg.Width * gg.Height
	const inf = int(^uint(0) >> 1)
	dist := make([]int, N)
	prev := make([]int, N)
	for i := range dist {
		dist[i] = inf
		prev[i] = -1
	}

	// 0–1 BFS: deque processes cost0 at front, cost1 at back
	dq := list.New()
	for _, i := range src {
		dist[i] = 0
		dq.PushFront(i)
	}

	offsets := gg.neighborOffsets()
	target := -1

	for dq.Len() > 0 {
		e := dq.Front()
		dq.Remove(e)
		u := e.Value.(int)
		if _, ok := dstSet[u]; ok {
			target = u
			break
		}
		ux, uy := gg.Coordinate(u)
		for _, d := range offsets {
			vx, vy := ux+d[0], uy+d[1]
			if !gg.InBounds(vx, vy) {
				continue
			}
			v := gg.index(vx, vy)
			step := 0
			if gg.CellValues[vy][vx] < 1 {
				step = 1
			}
			nd := dist[u] + step
			if nd < dist[v] {
				dist[v] = nd
				prev[v] = u
				if step == 0 {
					dq.PushFront(v)
				} else {
					dq.PushBack(v)
				}
			}
		}
	}

	if target < 0 {
		return nil, 0, ErrNoPath
	}
	// Reconstruct path
	for at := target; at >= 0; at = prev[at] {
		path = append([]int{at}, path...)
	}
	return path, dist[target], nil
}
