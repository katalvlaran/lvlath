package tsp

import "math"

// MinimumSpanningTree computes a minimum spanning tree on the complete metric graph
// represented by the n×n distance matrix dist. It returns:
//
//	parents — slice of length n where parents[v] is the parent of v in the MST (or -1 for root);
//	adj     — adjacency list (n slices) of the MST edges;
//	err     — ErrDimensionMismatch if dist is not square.
//
// Time:  O(n²)  using Prim’s algorithm.
// Space: O(n²) for the output adjacency lists.
func MinimumSpanningTree(dist [][]float64) (parents []int, adj [][]int, err error) {
	n := len(dist)
	// Validate square matrix
	for i := 0; i < n; i++ {
		if len(dist[i]) != n {
			return nil, nil, ErrDimensionMismatch
		}
	}
	// Track which vertices are in the tree
	inMST := make([]bool, n)
	// Best edge weight to connect each vertex to the growing MST
	bestCost := make([]float64, n)
	// Parent pointer for each vertex
	parents = make([]int, n)
	// Initialize adjacency lists
	adj = make([][]int, n)

	// 1) Initialization: set all costs to +Inf, no parent
	for v := range bestCost {
		bestCost[v] = math.Inf(1)
		parents[v] = -1
	}
	// 2) Start from vertex 0
	bestCost[0] = 0

	// 3) Grow the MST one vertex at a time
	for it := 0; it < n; it++ {
		// (a) Find vertex u not in MST with minimal bestCost[u]
		u, minW := -1, math.Inf(1)
		for v := 0; v < n; v++ {
			if !inMST[v] && bestCost[v] < minW {
				minW, u = bestCost[v], v
			}
		}
		// If no such u, the graph is disconnected
		if u < 0 {
			return nil, nil, ErrIncompleteGraph
		}
		// (b) Add u to MST
		inMST[u] = true
		if parents[u] >= 0 {
			// Record edge u↔parent in adjacency lists
			p := parents[u]
			adj[u] = append(adj[u], p)
			adj[p] = append(adj[p], u)
		}
		// (c) Update bestCost and parent for remaining vertices
		for v := 0; v < n; v++ {
			if !inMST[v] && dist[u][v] < bestCost[v] {
				bestCost[v] = dist[u][v]
				parents[v] = u
			}
		}
	}

	return parents, adj, nil
}
