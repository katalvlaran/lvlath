// Package prim_kruskal provides an implementation of Kruskal’s Minimum Spanning Tree algorithm.
// It assumes an undirected, weighted *core.Graph and produces a slice of edges forming the MST.
package prim_kruskal

import (
	"sort"

	"github.com/katalvlaran/lvlath/core"
)

// Kruskal computes the Minimum Spanning Tree (MST) of an undirected, weighted graph.
// It uses a disjoint-set (union-find) data structure with path compression and union by rank.
//
// Error Conditions:
//   - ErrInvalidGraph  : if graph is nil, or graph.Directed() == true, or graph.Weighted() == false.
//   - ErrDisconnected  : if |V| == 0 or |V| > 1 but graph is not fully connected.
//
// Steps:
//  1. Validate: graph != nil, graph.Weighted(), !graph.Directed() and !graph.HasDirectedEdges()..
//  2. Retrieve sorted vertex IDs; if len(vertices)==0 → ErrDisconnected.
//     If len(vertices)==1 → trivial MST (empty, weight=0).
//  3. Collect all edges via graph.Edges(), skip self-loops (e.From == e.To).
//  4. Sort edges by ascending Weight (use sort.SliceStable to maintain deterministic order for equal weights).
//  5. Initialize DSU maps parent[] and rank[] for each vertex in vertices.
//  6. Loop over sorted edges: for each edge (u,v), if find(u) != find(v), then union(u,v) and include edge in MST.
//  7. Once MST has |V|-1 edges, break. After loop, if MST edge count < |V|-1 → ErrDisconnected.
//
// Complexity: O(E log E + α(V)·E) ≈ O(E log V). Memory: O(E + V).
func Kruskal(graph *core.Graph) ([]core.Edge, int64, error) {
	// 1. Validate that graph is non-nil, weighted, undirected and have no direct edges.
	if graph == nil || !graph.Weighted() || graph.Directed() || graph.HasDirectedEdges() {
		// Return ErrInvalidGraph for any invalid condition.
		return nil, 0, ErrInvalidGraph
	}

	// 2. Retrieve all vertex IDs in sorted order for determinism.
	vertices := graph.Vertices()
	// If no vertices exist, there is no spanning tree but also no edges;
	// by convention, we consider this a disconnected graph for |V| == 0.
	if len(vertices) == 0 {
		return nil, 0, ErrDisconnected
	}
	// If exactly one vertex, the MST is trivially empty with total weight 0.
	if len(vertices) == 1 {
		// Return an empty slice (no edges) and zero weight.
		return []core.Edge{}, 0, nil
	}

	// 3. Collect all edges from graph, skipping self-loops to avoid trivial cycles.
	allEdges := graph.Edges()                     // []*core.Edge sorted by Edge.ID
	edges := make([]*core.Edge, 0, len(allEdges)) // filtered slice
	for _, e := range allEdges {
		if e.From == e.To {
			// Skip self-loops entirely: they cannot be part of a spanning tree.
			continue
		}
		edges = append(edges, e)
	}

	// 4. Sort edges by ascending weight (stable sort ensures deterministic tie-breaking
	//    based on original Edge.ID order from graph.Edges()).
	sort.SliceStable(edges, func(i, j int) bool {
		return edges[i].Weight < edges[j].Weight
	})

	// 5. Initialize disjoint-set (union-find) structures.
	//    parent maps each vertex to its parent in the DSU; initially parent[v] = v.
	parent := make(map[string]string, len(vertices))
	//    rank keeps track of tree depth to optimize unions.
	rank := make(map[string]int, len(vertices))
	for _, vid := range vertices {
		parent[vid] = vid
		rank[vid] = 0
	}

	// Iterative find with path compression to avoid deep recursion.
	find := func(u string) string {
		// Walk up until the root (parent[u] == u).
		for parent[u] != u {
			// Path compression: make u point to its grandparent.
			parent[u] = parent[parent[u]]
			u = parent[u]
		}

		return u
	}

	// Union by rank merges two disjoint sets.
	union := func(u, v string) {
		rootU := find(u)
		rootV := find(v)
		if rootU == rootV {
			// Already in the same set; no action needed.
			return
		}
		// Attach smaller-rank tree under larger-rank root.
		if rank[rootU] < rank[rootV] {
			parent[rootU] = rootV
		} else {
			parent[rootV] = rootU
			// If ranks are equal, increment the resulting root's rank by 1.
			if rank[rootU] == rank[rootV] {
				rank[rootU]++
			}
		}
	}

	// 6. Build MST by iterating over sorted edges.
	var (
		mst         []core.Edge // resulting edges in the MST
		totalWeight int64       // sum of weights
		numVerts    = len(vertices)
	)
	for _, e := range edges {
		u := e.From // one endpoint
		v := e.To   // the other endpoint
		// Check if endpoints are in different components.
		if find(u) != find(v) {
			// If disjoint, merge sets and include this edge in MST.
			union(u, v)
			mst = append(mst, *e)   // dereference *core.Edge to core.Edge
			totalWeight += e.Weight // accumulate weight
			// If we have |V|-1 edges, MST is complete.
			if len(mst) == numVerts-1 {
				break
			}
		}
	}

	// 7. If MST does not contain exactly |V|-1 edges, graph was disconnected.
	if len(mst) < numVerts-1 {
		return nil, 0, ErrDisconnected
	}

	// 8. Return the built MST and its total weight.
	return mst, totalWeight, nil
}
