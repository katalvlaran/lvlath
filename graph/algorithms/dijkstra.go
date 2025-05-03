// Package algorithms implements graph algorithms on core.Graph.
/*
Dijkstra — Shortest Paths in Weighted Graphs

Description:
  Computes the minimum‐cost path from a source vertex to all other
  vertices in a graph with non-negative edge weights.

Use cases:
  - Routing, navigation, resource allocation.
  - Building blocks for A* and other heuristics.

Algorithm outline:
 1. Validate:
    - Graph must be marked weighted.
    - Start vertex must exist.
 2. Initialization:
    - dist[v] = ∞ for all v, except dist[start] = 0.
    - parent[v] = "".
    - visited[v] = false.
    - Push (start, 0) into a min‐heap priority queue.
 3. Main loop:
    while pq not empty:
      a. u := pop minimum‐dist vertex from pq
      b. if visited[u] continue
      c. visited[u] = true
      d. for each edge u→w with weight wt:
           if !visited[w] and dist[u]+wt < dist[w]:
             dist[w] = dist[u] + wt
             parent[w] = u
             push (w, dist[w]) into pq
 4. Return dist and parent maps.

Time complexity: O((V + E) log V)
Memory:     O(V + E)
*/
package algorithms

import (
	"container/heap"
	"errors"
	"math"

	"github.com/katalvlaran/lvlath/graph/core"
)

// ErrDijkstraNotWeighted indicates that Dijkstra requires a weighted graph.
var ErrDijkstraNotWeighted = errors.New("graph: Dijkstra requires weighted graph")

// Dijkstra finds shortest paths from startID in g.
// Returns a map of distances and a parent map for path reconstruction.
func Dijkstra(g *core.Graph, startID string) (map[string]int64, map[string]string, error) {
	// 1. Validate prerequisites
	if !g.Weighted() {
		return nil, nil, ErrDijkstraNotWeighted
	}
	if !g.HasVertex(startID) {
		return nil, nil, ErrVertexNotFound
	}

	// 2. Initialize worker and run algorithm
	r := &dijkstraRunner{
		g:       g,
		start:   startID,
		dist:    make(map[string]int64),
		parent:  make(map[string]string),
		visited: make(map[string]bool),
		pq:      make(nodePQ, 0, len(g.Vertices())),
	}
	r.init()         // set up dist, parent, visited, and initial PQ
	r.processQueue() // run main loop
	return r.dist, r.parent, nil
}

// dijkstraRunner holds state for a single execution.
type dijkstraRunner struct {
	g       *core.Graph
	start   string
	dist    map[string]int64
	parent  map[string]string
	visited map[string]bool
	pq      nodePQ
}

// init sets up distance, parent, visited maps and pushes the start node.
func (r *dijkstraRunner) init() {
	// Initialize all distances to infinity
	for _, v := range r.g.Vertices() {
		r.dist[v.ID] = math.MaxInt64
		r.parent[v.ID] = ""
		r.visited[v.ID] = false
	}
	// Distance to start is zero
	r.dist[r.start] = 0
	// Build initial priority queue containing only the start node
	heap.Init(&r.pq)
	heap.Push(&r.pq, &nodeItem{id: r.start, dist: 0})
}

// processQueue runs the main Dijkstra loop until PQ is empty.
func (r *dijkstraRunner) processQueue() {
	for r.pq.Len() > 0 {
		uItem := heap.Pop(&r.pq).(*nodeItem)
		u := uItem.id
		// Skip already‐visited vertices
		if r.visited[u] {
			continue
		}
		// Mark as visited so we don't revisit
		r.visited[u] = true
		// Relax each outgoing edge
		r.relaxEdges(u)
	}
}

// relaxEdges attempts to improve distances for neighbors of u.
func (r *dijkstraRunner) relaxEdges(u string) {
	// For each outgoing edge from u
	for _, edges := range r.g.AdjacencyList()[u] {
		for _, e := range edges {
			v := e.To.ID
			if r.visited[v] {
				continue
			}
			newDist := r.dist[u] + e.Weight
			// If a shorter path to v is found, update and push to PQ
			// Update even on equal distance so that, e.g., indirect path
			// via already-processed node can override direct same-cost edge.
			if newDist <= r.dist[v] {
				r.dist[v] = newDist
				r.parent[v] = u
				heap.Push(&r.pq, &nodeItem{id: v, dist: newDist})
			}
		}
	}
}

// nodeItem is an entry in the priority queue.
type nodeItem struct {
	id   string // vertex ID
	dist int64  // current best distance from start
}

// nodePQ implements heap.Interface for []*nodeItem,
// ordering by smallest dist first.
type nodePQ []*nodeItem

func (pq nodePQ) Len() int            { return len(pq) }
func (pq nodePQ) Less(i, j int) bool  { return pq[i].dist < pq[j].dist }
func (pq nodePQ) Swap(i, j int)       { pq[i], pq[j] = pq[j], pq[i] }
func (pq *nodePQ) Push(x interface{}) { *pq = append(*pq, x.(*nodeItem)) }
func (pq *nodePQ) Pop() interface{} {
	old := *pq
	n := len(old)
	it := old[n-1]
	*pq = old[:n-1]
	return it
}
