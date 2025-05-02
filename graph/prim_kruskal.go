package graph

import (
	"container/heap"
	"errors"
	"sort"
)

// ErrNotWeighted indicates MST algorithms require weighted, undirected graphs.
var ErrNotWeighted = errors.New("graph: MST requires weighted undirected graph")

// Prim builds a minimum spanning tree starting from startID.
// Returns list of edges in MST and total weight.
func (g *Graph) Prim(startID string) ([]*Edge, int64, error) {
	if g.directed || !g.weighted {
		return nil, 0, ErrNotWeighted
	}
	if !g.HasVertex(startID) {
		return nil, 0, ErrVertexNotFound
	}

	visited := make(map[string]bool)
	mst := make([]*Edge, 0, len(g.vertices)-1)
	var total int64

	// Min-heap of candidate edges
	pq := &edgePQ{}
	heap.Init(pq)
	visited[startID] = true
	// push all edges from start
	/*for _, e := range g.adjacencyList[startID][startID] { // - не понятно для чего?
		// no self-edges; skip
	}*/
	for _, nbrs := range g.adjacencyList[startID] {
		for _, e := range nbrs {
			heap.Push(pq, e)
		}
	}

	for pq.Len() > 0 && len(mst) < len(g.vertices)-1 {
		e := heap.Pop(pq).(*Edge)
		if visited[e.To.ID] {
			continue
		}
		visited[e.To.ID] = true
		mst = append(mst, e)
		total += e.Weight
		// add new edges
		/*for _, ne := range g.adjacencyList[e.To.ID][e.To.ID] { // - не понятно для чего?
		}*/
		for _, nbrs := range g.adjacencyList[e.To.ID] {
			for _, ne := range nbrs {
				if !visited[ne.To.ID] {
					heap.Push(pq, ne)
				}
			}
		}
	}

	return mst, total, nil
}

// edgePQ implements heap.Interface for []*Edge by Weight.
type edgePQ []*Edge

func (pq edgePQ) Len() int            { return len(pq) }
func (pq edgePQ) Less(i, j int) bool  { return pq[i].Weight < pq[j].Weight }
func (pq edgePQ) Swap(i, j int)       { pq[i], pq[j] = pq[j], pq[i] }
func (pq *edgePQ) Push(x interface{}) { *pq = append(*pq, x.(*Edge)) }
func (pq *edgePQ) Pop() interface{} {
	old := *pq
	n := len(old)
	e := old[n-1]
	*pq = old[:n-1]
	return e
}

// Kruskal builds a minimum spanning tree using disjoint-set union.
func (g *Graph) Kruskal() ([]*Edge, int64, error) {
	if g.directed || !g.weighted {
		return nil, 0, ErrNotWeighted
	}

	// Prepare unique edges
	edges := filterUniqueEdges(g)
	sort.Slice(edges, func(i, j int) bool {
		return edges[i].Weight < edges[j].Weight
	})

	// DSU init
	parent := make(map[string]string, len(g.vertices))
	rank := make(map[string]int, len(g.vertices))
	for id := range g.vertices {
		parent[id] = id
		rank[id] = 0
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
		if rank[ru] < rank[rv] {
			parent[ru] = rv
		} else {
			parent[rv] = ru
			if rank[ru] == rank[rv] {
				rank[ru]++
			}
		}
	}

	// Kruskal main loop
	mst := make([]*Edge, 0, len(g.vertices)-1)
	var total int64
	for _, e := range edges {
		u, v := e.From.ID, e.To.ID
		if find(u) != find(v) {
			union(u, v)
			mst = append(mst, e)
			total += e.Weight
		}
	}
	return mst, total, nil
}
