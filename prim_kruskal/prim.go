// Package prim_kruskal provides an implementation of Prim’s Minimum Spanning Tree (MST) algorithm.
// It assumes an undirected, weighted *core.Graph and grows the MST from a specified root vertex using a min‐heap.
package prim_kruskal

import (
	"container/heap"

	"github.com/katalvlaran/lvlath/core"
)

// Prim computes the Minimum Spanning Tree (MST) of an undirected, weighted graph
// by growing outwards from a specified root vertex using a min‐heap.
//
// Error Conditions:
//   - ErrInvalidGraph      : if graph is nil, or graph.Directed() == true, or graph.Weighted() == false.
//   - ErrEmptyRoot         : if the provided root string is empty.
//   - core.ErrVertexNotFound: if the root vertex does not exist in the graph.
//   - ErrDisconnected      : if |V| == 0 (empty graph) or |V| > 1 but the graph is not fully connected.
//
// Steps:
//  1. Validate: graph != nil, graph.Weighted(), !graph.Directed() and !graph.HasDirectedEdges().
//  2. Retrieve sorted vertex IDs; if len(vertices)==0 → ErrDisconnected.
//     If len(vertices)==1, check that root matches the single vertex → return trivial empty MST.
//  3. Validate root: root != "", graph.HasVertex(root).
//  4. Initialize:
//     - visited map to track which vertices are already in MST.
//     - pq (min‐heap) to hold candidate edges ordered by weight.
//     - mark root as visited and push all edges adjacent to root into pq.
//  5. While pq not empty and MST has < |V|-1 edges:
//     a. Pop the smallest‐weight edge (u→v) from pq.
//     b. If v is already visited, skip (this edge would form a cycle).
//     c. Otherwise, add (u→v) to MST, mark v as visited, accumulate weight.
//     d. Push all edges from v to as‐yet‐unvisited neighbors into pq.
//  6. If MST size < |V|-1 after loop → ErrDisconnected.
//  7. Return MST edges and total weight.
//
// Complexity: O(E log V) time, O(V + E) memory.
func Prim(graph *core.Graph, root string) ([]core.Edge, float64, error) {
	// 1. Validate that graph is non-nil, weighted, undirected and have no direct edges.
	if graph == nil || !graph.Weighted() || graph.Directed() || graph.HasDirectedEdges() {
		// Return ErrInvalidGraph for any invalid condition.
		return nil, 0, ErrInvalidGraph
	}

	// 2. Retrieve all vertex IDs in sorted order (core.Graph.Vertices() returns sorted).
	vertices := graph.Vertices()
	// If no vertices, we cannot form any MST: treat as disconnected.
	if len(vertices) == 0 {
		return nil, 0, ErrDisconnected
	}
	// If exactly one vertex, MST is trivially empty (no edges) if root matches that vertex.
	if len(vertices) == 1 {
		if vertices[0] != root {
			// If the single vertex does not match the requested root, that root doesn't exist.
			return nil, 0, core.ErrVertexNotFound
		}

		// Single‐vertex MST: empty edge list, zero total weight, no error.
		return []core.Edge{}, 0, nil
	}

	// 3. Validate root is non-empty and actually exists in the graph.
	if root == "" {
		return nil, 0, ErrEmptyRoot
	}
	if !graph.HasVertex(root) {
		return nil, 0, core.ErrVertexNotFound
	}

	// 4. Initialize visited set and MST container.
	n := len(vertices)                  // total number of vertices
	visited := make(map[string]bool, n) // mark visited vertices
	mst := make([]core.Edge, 0, n-1)    // will hold up to n-1 edges
	var totalWeight float64             // sum of weights in MST

	// 4a. Prepare the priority queue (min‐heap) of *core.Edge pointers.
	pq := &edgePQ{} // our custom edge priority queue
	heap.Init(pq)   // initialize internal slice

	// 4b. Mark root as visited and push all edges adjacent to root.
	visited[root] = true
	neighbors, err := graph.Neighbors(root) // get all outgoing/undirected edges from root
	if err != nil {
		// If Neighbors returned an error (e.g., vertex not found), propagate it.
		return nil, 0, err
	}
	for _, e := range neighbors {
		// Only consider edges whose other endpoint is not yet visited.
		if !visited[e.To] {
			heap.Push(pq, e) // push pointer to *core.Edge
		}
	}

	// 5. Main loop: extract smallest edge and expand MST until we have n-1 edges.
	for pq.Len() > 0 && len(mst) < n-1 {
		// Pop the minimal‐weight edge from the heap.
		e := heap.Pop(pq).(*core.Edge)
		v := e.To
		// If this endpoint is already visited, skip to avoid cycles.
		if visited[v] {
			continue
		}
		// 5a. Include this edge (u→v) in MST.
		visited[v] = true       // mark new vertex as visited
		mst = append(mst, *e)   // append the edge value (dereference pointer)
		totalWeight += e.Weight // accumulate its weight

		// 5b. Push all edges from newly visited vertex v to unvisited neighbors.
		nextNeighbors, err := graph.Neighbors(v)
		if err != nil {
			// Propagate any error encountered while fetching neighbors.
			return nil, 0, err
		}
		for _, ne := range nextNeighbors {
			if !visited[ne.To] {
				heap.Push(pq, ne)
			}
		}
	}

	// 6. If we did not collect exactly n-1 edges, the graph must be disconnected.
	if len(mst) < n-1 {
		return nil, 0, ErrDisconnected
	}

	// 7. Return the completed MST and its total weight.
	return mst, totalWeight, nil
}

// edgePQ implements heap.Interface for a min‐heap of *core.Edge, ordered by Weight.
type edgePQ []*core.Edge

// Len returns the number of edges in the priority queue.
// Complexity: O(1).
func (pq edgePQ) Len() int { return len(pq) }

// Less reports whether element i should sort before j.
// We compare by edge.Weight for ascending order.
// Complexity: O(1).
func (pq edgePQ) Less(i, j int) bool { return pq[i].Weight < pq[j].Weight }

// Swap swaps elements at indices i and j.
// Complexity: O(1).
func (pq edgePQ) Swap(i, j int) { pq[i], pq[j] = pq[j], pq[i] }

// Push appends a new *core.Edge to the heap.
// Type‐assert x to *core.Edge and append to underlying slice.
// Called by heap.Push. Complexity: O(log N) amortized.
func (pq *edgePQ) Push(x interface{}) { *pq = append(*pq, x.(*core.Edge)) }

// Pop removes and returns the smallest‐weight *core.Edge from the heap.
// Called by heap.Pop. Complexity: O(log N) amortized.
func (pq *edgePQ) Pop() interface{} {
	old := *pq
	n := len(old)
	edge := old[n-1] // smallest element after heap adjustments
	*pq = old[:n-1]  // shrink slice

	return edge
}
