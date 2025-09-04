// Package tsp - Minimum Spanning Tree (Prim O(n²)) for dense metric graphs.
//
// MinimumSpanningTree builds an MST over a non-negative dense distance matrix.
// This is step (1) of the Christofides pipeline. We implement Prim in O(n²) to
// avoid heap allocations and keep predictable memory use on dense instances.
//
// Contracts (validated earlier by the dispatcher on Christofides):
//   - dist is square n×n, n ≥ 2 for non-trivial TSP;
//   - diagonal ≈ 0, no negative weights, no NaN;
//   - for Christofides: no +Inf edges if metric closure is off.
//
// Behavior here remains defensive:
//   - Shape/At() errors → ErrNonSquare / ErrDimensionMismatch.
//   - Any negative weight → ErrNegativeWeight.
//   - +Inf edges make vertices unreachable → ErrIncompleteGraph.
//
// Return values:
//   - totalW : total MST weight (useful for bounds; Christofides itself doesn’t need it);
//   - adj    : undirected adjacency lists of the MST (simple graph, no parallel edges).
//
// Complexity:
//   - Time  : O(n²) (Prim without a heap).
//   - Memory: O(n) for state + O(n) lists (2(n−1) adjacency entries).
package tsp

import (
	"math"

	"github.com/katalvlaran/lvlath/matrix"
)

// MinimumSpanningTree runs Prim’s algorithm in O(n²) over any matrix.Matrix.
// Fast-path to *matrix.Dense is kept as an internal branch if needed later.
func MinimumSpanningTree(dist matrix.Matrix) (totalW float64, adj [][]int, err error) {
	if dist == nil {
		return 0, nil, ErrNonSquare
	}
	if d, ok := dist.(*matrix.Dense); ok {
		return mstDense(d)
	}
	return mstGeneric(dist)
}

// mstDense - Prim O(n²) using *matrix.Dense; avoids interface indirection in hot loops.
func mstDense(d *matrix.Dense) (float64, [][]int, error) {
	var (
		nr = d.Rows()
		nc = d.Cols()
	)
	if nr != nc || nr <= 0 {
		return 0, nil, ErrNonSquare
	}
	// n==1 is well-defined: empty tree with weight 0.
	if nr == 1 {
		return 0, make([][]int, 1), nil
	}
	var (
		n        = nr
		inMST    = make([]bool, n)
		bestCost = make([]float64, n)
		parent   = make([]int, n)
		adj      = make([][]int, n)
		i        int
	)

	for i = 0; i < n; i++ {
		bestCost[i] = math.Inf(1)
		parent[i] = -1
	}

	// Start from vertex 0 (choice is arbitrary; MST does not depend on it).
	bestCost[0] = 0

	var (
		iter  int
		u     int
		minW  float64
		v     int
		w     float64
		total float64
	)

	for iter = 0; iter < n; iter++ {
		// Pick the non-tree vertex with minimal connection cost.
		u = -1
		minW = math.Inf(1)
		for v = 0; v < n; v++ {
			if !inMST[v] && bestCost[v] < minW {
				minW = bestCost[v]
				u = v
			}
		}
		if u == -1 {
			// Some vertices remained unreachable (e.g., +Inf edges).
			return 0, nil, ErrIncompleteGraph
		}

		// Add u to the tree; connect to its parent if it has one.
		inMST[u] = true
		if parent[u] != -1 {
			adj[u] = append(adj[u], parent[u])
			adj[parent[u]] = append(adj[parent[u]], u)
			// bestCost[u] is the weight of edge (parent[u], u)
			total += bestCost[u]
		}

		// Relax neighbors of u.
		for v = 0; v < n; v++ {
			if inMST[v] {
				continue
			}
			var err error
			w, err = d.At(u, v)
			if err != nil {
				return 0, nil, ErrDimensionMismatch
			}
			if math.IsNaN(w) {
				return 0, nil, ErrDimensionMismatch
			}
			if w < 0 {
				return 0, nil, ErrNegativeWeight
			}
			// +Inf is allowed to pass through (keeps bestCost[v]==+Inf) and will trigger
			// ErrIncompleteGraph when no reachable vertex remains.
			if w < bestCost[v] {
				bestCost[v] = w
				parent[v] = u
			}
		}
	}

	return round1e9(total), adj, nil
}

// mstGeneric - Prim O(n²) via the matrix.Matrix interface.
func mstGeneric(m matrix.Matrix) (float64, [][]int, error) {
	var (
		nr = m.Rows()
		nc = m.Cols()
	)
	if nr != nc || nr <= 0 {
		return 0, nil, ErrNonSquare
	}
	if nr == 1 {
		return 0, make([][]int, 1), nil
	}
	var (
		n        = nr
		inMST    = make([]bool, n)
		bestCost = make([]float64, n)
		parent   = make([]int, n)
		adj      = make([][]int, n)
		i        int
	)
	for i = 0; i < n; i++ {
		bestCost[i] = math.Inf(1)
		parent[i] = -1
	}
	bestCost[0] = 0

	var (
		iter  int
		u     int
		minW  float64
		v     int
		w     float64
		total float64
	)

	for iter = 0; iter < n; iter++ {
		u = -1
		minW = math.Inf(1)
		for v = 0; v < n; v++ {
			if !inMST[v] && bestCost[v] < minW {
				minW = bestCost[v]
				u = v
			}
		}
		if u == -1 {
			return 0, nil, ErrIncompleteGraph
		}

		inMST[u] = true
		if parent[u] != -1 {
			adj[u] = append(adj[u], parent[u])
			adj[parent[u]] = append(adj[parent[u]], u)
			total += bestCost[u]
		}

		for v = 0; v < n; v++ {
			if inMST[v] {
				continue
			}
			var err error
			w, err = m.At(u, v)
			if err != nil {
				return 0, nil, ErrDimensionMismatch
			}
			if math.IsNaN(w) {
				return 0, nil, ErrDimensionMismatch
			}
			if w < 0 {
				return 0, nil, ErrNegativeWeight
			}
			if w < bestCost[v] {
				bestCost[v] = w
				parent[v] = u
			}
		}
	}

	return round1e9(total), adj, nil
}
