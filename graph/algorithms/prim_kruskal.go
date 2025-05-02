// Package algorithms implements graph algorithms on core.Graph.
package algorithms

import (
	"container/heap"
	"errors"
	"sort"

	"lvl_algorithms/graph/core"
)

/*
Prim & Kruskal — Minimum Spanning Tree (MST)

Description:
  Given an undirected, connected, weighted graph, an MST is a subset of edges
  connecting all vertices with the minimum possible total edge weight.

Algorithms:

1. Prim’s Algorithm
   - Greedy, grows one tree:
     a. Pick a start vertex, mark it visited.
     b. Push all edges from visited set into a min‐heap.
     c. While heap not empty and MST has < V-1 edges:
         i.   Pop minimum‐weight edge (u→v).
         ii.  If v is already visited, skip.
         iii. Add edge to MST, mark v visited.
         iv.  Push all edges from v to unvisited neighbors into heap.
   - Complexity: O(E log E) ≈ O(E log V)
   - Memory:     O(E)

2. Kruskal’s Algorithm
   - Greedy, picks global minimum edges:
     a. Sort all edges by weight ascending.
     b. Initialize disjoint‐set (DSU) for vertices.
     c. For each edge (u–v) in order:
         i.   If u and v are in different sets:
               • Union their sets
               • Add edge to MST
     d. Stop when MST has V-1 edges.
   - Complexity: O(E log E + E·α(V)) ≈ O(E log V)
   - Memory:     O(E + V)
*/

// ErrNotWeighted indicates MST algorithms require an undirected, weighted graph.
var ErrNotWeighted = errors.New("graph: MST requires weighted undirected graph")

// Prim computes an MST using Prim’s algorithm starting from startID.
// Returns the edges in the MST (in selection order) and the total weight.
func Prim(g *core.Graph, startID string) ([]*core.Edge, int64, error) {
	// Validate: must be undirected & weighted
	if g.Directed() || !g.Weighted() {
		return nil, 0, ErrNotWeighted
	}
	if !g.HasVertex(startID) {
		return nil, 0, ErrVertexNotFound
	}

	// State for Prim’s run
	visited := make(map[string]bool, len(g.Vertices()))
	mst := make([]*core.Edge, 0, len(g.Vertices())-1)
	var total int64

	// Min‐heap of candidate edges
	edgeHeap := &edgePQ{}
	heap.Init(edgeHeap)

	// Seed from startID
	visited[startID] = true
	for _, edges := range g.AdjacencyList()[startID] {
		for _, e := range edges {
			heap.Push(edgeHeap, e)
		}
	}

	// Main loop
	for edgeHeap.Len() > 0 && len(mst) < len(g.Vertices())-1 {
		e := heap.Pop(edgeHeap).(*core.Edge)
		// e.From is in visited by construction; skip if e.To already visited
		if visited[e.To.ID] {
			continue
		}
		// Accept edge
		visited[e.To.ID] = true
		mst = append(mst, e)
		total += e.Weight

		// Push new frontier edges from newly visited vertex
		for _, nbrEdges := range g.AdjacencyList()[e.To.ID] {
			for _, ne := range nbrEdges {
				if !visited[ne.To.ID] {
					heap.Push(edgeHeap, ne)
				}
			}
		}
	}

	return mst, total, nil
}

// Kruskal computes an MST using Kruskal’s algorithm.
// Returns the edges in the MST (in ascending‐weight order) and the total weight.
func Kruskal(g *core.Graph) ([]*core.Edge, int64, error) {
	// Validate: must be undirected & weighted
	if g.Directed() || !g.Weighted() {
		return nil, 0, ErrNotWeighted
	}

	// Filter unique edges (one per undirected pair)
	edges := kruskalFilterUniqueEdges(g)
	// Sort by ascending weight
	sort.Slice(edges, func(i, j int) bool {
		return edges[i].Weight < edges[j].Weight
	})

	// Disjoint‐set union (DSU) structures
	parent := make(map[string]string, len(g.Vertices()))
	rank := make(map[string]int, len(g.Vertices()))
	for _, v := range g.Vertices() {
		parent[v.ID] = v.ID
		rank[v.ID] = 0
	}
	var find func(string) string
	find = func(u string) string {
		if parent[u] != u {
			parent[u] = find(parent[u])
		}
		return parent[u]
	}
	union := func(u, v string) {
		ru, rv := find(u), find(v)
		if ru == rv {
			return
		}
		// Union by rank
		if rank[ru] < rank[rv] {
			parent[ru] = rv
		} else {
			parent[rv] = ru
			if rank[ru] == rank[rv] {
				rank[ru]++
			}
		}
	}

	mst := make([]*core.Edge, 0, len(g.Vertices())-1)
	var total int64

	// Main loop: pick smallest edge that connects two separate sets
	for _, e := range edges {
		u, v := e.From.ID, e.To.ID
		if find(u) != find(v) {
			union(u, v)
			mst = append(mst, e)
			total += e.Weight
			// Stop once we have V-1 edges
			if len(mst) == len(g.Vertices())-1 {
				break
			}
		}
	}

	return mst, total, nil
}

// kruskalFilterUniqueEdges returns one representative per undirected edge.
func kruskalFilterUniqueEdges(g *core.Graph) []*core.Edge {
	all := g.Edges()
	seen := make(map[string]map[string]bool, len(all))
	uniq := make([]*core.Edge, 0, len(all))
	for _, e := range all {
		u, v := e.From.ID, e.To.ID
		if u == v {
			continue // skip self‐loops
		}
		// Normalize order for undirected uniqueness
		if u > v {
			u, v = v, u
		}
		if seen[u] == nil {
			seen[u] = make(map[string]bool)
		}
		if !seen[u][v] {
			seen[u][v] = true
			uniq = append(uniq, e)
		}
	}
	return uniq
}

// edgePQ implements heap.Interface over []*core.Edge ordered by Weight.
type edgePQ []*core.Edge

func (pq edgePQ) Len() int            { return len(pq) }
func (pq edgePQ) Less(i, j int) bool  { return pq[i].Weight < pq[j].Weight }
func (pq edgePQ) Swap(i, j int)       { pq[i], pq[j] = pq[j], pq[i] }
func (pq *edgePQ) Push(x interface{}) { *pq = append(*pq, x.(*core.Edge)) }
func (pq *edgePQ) Pop() interface{} {
	old := *pq
	n := len(old)
	e := old[n-1]
	*pq = old[:n-1]
	return e
}
