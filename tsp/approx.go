package tsp

import (
	"errors"
	"fmt"
	"math"
)

// TSPApprox computes a 1.5-approximation to the Travelling Salesman
// Problem on a metric (symmetric, triangle-inequality) distance matrix
// using the Christofides algorithm.
//
// Steps:
//  1. Build a minimum spanning tree (MST) of the complete graph via Prim.
//  2. Find all vertices of odd degree in the MST.
//  3. Compute a minimum-weight perfect matching on the induced odd-degree subgraph.
//     Here we use a greedy heuristic: repeatedly match the two closest unmatched.
//  4. Combine MST edges and matching edges into a multigraph (Eulerian).
//  5. Find an Eulerian tour (Hierholzer’s algorithm).
//  6. Shortcut repeated vertices to obtain a Hamiltonian cycle.
//
// Input: n×n symmetric matrix dist, dist[i][i]==0, dist[i][j]=dist[j][i].
//
//	math.Inf(1) entries indicate missing edges (non-metric).
//
// Returns a TSResult with Tour (n+1 length, start and end at 0) and Cost.
// Returns ErrTSPIncompleteGraph if the matrix is invalid or a cycle cannot be formed.
//
// Time:    O(n² + n³) dominated by matching heuristic and Euler tour.
// Memory:  O(n²)
func TSPApprox(dist [][]float64) (TSResult, error) {
	n := len(dist)
	if n == 0 {
		return TSResult{}, errors.New("tsp: empty matrix")
	}
	// -- 1. Validate symmetric metric matrix --
	for i := 0; i < n; i++ {
		if len(dist[i]) != n {
			return TSResult{}, fmt.Errorf("tsp: row %d length %d, want %d", i, len(dist[i]), n)
		}
		for j := 0; j < n; j++ {
			if i == j {
				if dist[i][j] != 0 {
					return TSResult{}, fmt.Errorf("tsp: dist[%d][%d]=%v; self-distance must be 0", i, j, dist[i][j])
				}
			} else {
				if dist[i][j] != dist[j][i] {
					return TSResult{}, fmt.Errorf("tsp: non-symmetric at [%d][%d]=%v vs [%d][%d]=%v",
						i, j, dist[i][j], j, i, dist[j][i])
				}
				if math.IsInf(dist[i][j], 1) {
					return TSResult{}, ErrTSPIncompleteGraph
				}
			}
		}
	}

	// -- 2. Build MST on complete graph via Prim --
	mstAdj := make([][]int, n)    // adjacency list of MST
	used := make([]bool, n)       // in-MST flag
	minEdge := make([]float64, n) // best edge weight to add
	parent := make([]int, n)      // parent in MST
	for i := range minEdge {
		minEdge[i] = math.Inf(1)
		parent[i] = -1
	}
	minEdge[0] = 0
	for iter := 0; iter < n; iter++ {
		// pick unused vertex with minimal minEdge
		u, best := -1, math.Inf(1)
		for v := 0; v < n; v++ {
			if !used[v] && minEdge[v] < best {
				best, u = minEdge[v], v
			}
		}
		if u < 0 {
			return TSResult{}, ErrTSPIncompleteGraph
		}
		used[u] = true
		if parent[u] >= 0 {
			mstAdj[u] = append(mstAdj[u], parent[u])
			mstAdj[parent[u]] = append(mstAdj[parent[u]], u)
		}
		// relax neighbors
		for v := 0; v < n; v++ {
			if !used[v] && dist[u][v] < minEdge[v] {
				minEdge[v] = dist[u][v]
				parent[v] = u
			}
		}
	}

	// -- 3. Find odd-degree vertices in MST --
	var odd []int
	for v := 0; v < n; v++ {
		if len(mstAdj[v])%2 == 1 {
			odd = append(odd, v)
		}
	}

	// -- 4. Greedy perfect matching among odd vertices --
	// We'll match nearest pairs repeatedly.
	paired := make([]bool, n)
	for len(odd) > 0 {
		u := odd[0]
		paired[u] = true
		odd = odd[1:]
		// find closest to u
		bestIdx, bestDist := -1, math.Inf(1)
		for i, v := range odd {
			if dist[u][v] < bestDist {
				bestDist = dist[u][v]
				bestIdx = i
			}
		}
		if bestIdx < 0 {
			return TSResult{}, ErrTSPIncompleteGraph
		}
		v := odd[bestIdx]
		// add matching edge u–v
		mstAdj[u] = append(mstAdj[u], v)
		mstAdj[v] = append(mstAdj[v], u)
		paired[v] = true
		// remove v from odd list
		odd = append(odd[:bestIdx], odd[bestIdx+1:]...)
	}

	// -- 5. Find Eulerian tour on the multigraph (Hierholzer) --
	// Build edge multiset with indices
	type edge struct{ to, idx int }
	multi := make([][]edge, n)
	for u := 0; u < n; u++ {
		for _, v := range mstAdj[u] {
			multi[u] = append(multi[u], edge{v, u})
		}
	}
	var tour []int
	var dfsEuler func(u int)
	dfsEuler = func(u int) {
		for len(multi[u]) > 0 {
			e := multi[u][len(multi[u])-1]
			multi[u] = multi[u][:len(multi[u])-1]
			// remove corresponding reverse edge
			for i, re := range multi[e.to] {
				if re.to == u {
					multi[e.to] = append(multi[e.to][:i], multi[e.to][i+1:]...)
					break
				}
			}
			dfsEuler(e.to)
		}
		tour = append(tour, u)
	}
	dfsEuler(0)

	// -- 6. Shortcut to Hamiltonian cycle --
	visitedTour := make([]bool, n)
	var cycle []int
	for _, v := range tour {
		if !visitedTour[v] {
			cycle = append(cycle, v)
			visitedTour[v] = true
		}
	}
	cycle = append(cycle, cycle[0]) // close cycle

	// compute cost
	var total float64
	for i := 0; i < len(cycle)-1; i++ {
		u, v := cycle[i], cycle[i+1]
		total += dist[u][v]
	}

	return TSResult{Tour: cycle, Cost: total}, nil
}
