package graph

import (
	"container/heap"
	"errors"
	"math"
)

// ErrDijkstraNotWeighted indicates Dijkstra requires weighted graph.
var ErrDijkstraNotWeighted = errors.New("graph: Dijkstra requires weighted graph")

// Dijkstra computes shortest paths from startID.
// Returns dist[v.ID] and parent map for path reconstruction.
func (g *Graph) Dijkstra(startID string) (map[string]int64, map[string]string, error) {
	if !g.weighted {
		return nil, nil, ErrDijkstraNotWeighted
	}
	if !g.HasVertex(startID) {
		return nil, nil, ErrVertexNotFound
	}

	// init
	dist := make(map[string]int64, len(g.vertices))
	parent := make(map[string]string, len(g.vertices))
	for id := range g.vertices {
		dist[id] = math.MaxInt64
	}
	dist[startID] = 0

	// min-heap of (id, dist)
	pq := &nodePQ{}
	heap.Init(pq)
	heap.Push(pq, &nodeItem{id: startID, dist: 0})

	visited := make(map[string]bool, len(g.vertices))

	for pq.Len() > 0 {
		u := heap.Pop(pq).(*nodeItem)
		if visited[u.id] {
			continue
		}
		visited[u.id] = true

		for _, e := range g.adjacencyList[u.id] {
			for _, edge := range e {
				v := edge.To.ID
				if visited[v] {
					continue
				}
				nd := dist[u.id] + edge.Weight
				if nd < dist[v] {
					dist[v] = nd
					parent[v] = u.id
					heap.Push(pq, &nodeItem{id: v, dist: nd})
				}
			}
		}
	}

	return dist, parent, nil
}

// nodeItem for Dijkstra PQ
type nodeItem struct {
	id   string
	dist int64
}

// nodePQ implements heap.Interface
type nodePQ []*nodeItem

func (pq nodePQ) Len() int           { return len(pq) }
func (pq nodePQ) Less(i, j int) bool { return pq[i].dist < pq[j].dist }
func (pq nodePQ) Swap(i, j int)      { pq[i], pq[j] = pq[j], pq[i] }
func (pq *nodePQ) Push(x interface{}) {
	*pq = append(*pq, x.(*nodeItem))
}
func (pq *nodePQ) Pop() interface{} {
	old := *pq
	n := len(old)
	it := old[n-1]
	*pq = old[:n-1]
	return it
}
