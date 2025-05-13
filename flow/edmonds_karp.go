package flow

import (
	"context"
	"fmt"
	"math"

	"github.com/katalvlaran/lvlath/core"
)

// EdmondsKarp computes the maximum flow from source→sink
// using the Edmonds–Karp algorithm (BFS for shortest augmenting paths).
//
// It returns:
//   - maxFlow: total flow value
//   - residual: residual-capacity graph after flow
//   - err: non-nil on missing vertices or negative capacities.
//
// Options (nil uses defaults):
//   - Epsilon: capacities ≤ Epsilon treated as zero (default 1e-9)
//   - Verbose:  print each augmentation via fmt.Printf
//
// Complexity: O(V · E²)
// Memory:     O(V + E)
func EdmondsKarp(
	ctx context.Context,
	g *core.Graph,
	source, sink string,
	opts *FlowOptions,
) (maxFlow float64, residual *core.Graph, err error) {
	// 1) Set epsilon
	eps := 1e-9
	if opts != nil && opts.Epsilon > 0 {
		eps = opts.Epsilon
	}

	// 2) Validate presence of source/sink
	if !g.HasVertex(source) {
		return 0, nil, ErrSourceNotFound
	}
	if !g.HasVertex(sink) {
		return 0, nil, ErrSinkNotFound
	}

	// 3) Build residual graph (copy vertices, sum parallel edges)
	residual = core.NewGraph(g.Directed(), true)
	for _, v := range g.Vertices() {
		// share Metadata shallowly
		residual.AddVertex(&core.Vertex{ID: v.ID, Metadata: v.Metadata})
	}
	for u, nbrs := range g.AdjacencyList() {
		for vID, edges := range nbrs {
			// sum parallel edge capacities
			var capSum float64
			for _, e := range edges {
				if float64(e.Weight) < -eps {
					return 0, nil, EdgeError{From: u, To: vID, Cap: float64(e.Weight)}
				}
				capSum += float64(e.Weight)
			}
			if capSum > eps {
				residual.AddEdge(u, vID, int64(capSum))
			}
		}
	}

	// 4) Main loop: find BFS augmenting paths until none remain
	for {
		path, bottle := bfsAugmentingPath(ctx, residual, source, sink, eps)
		if len(path) == 0 || bottle <= eps {
			break
		}
		if opts != nil && opts.Verbose {
			fmt.Printf("augmenting path %v with flow %.3g\n", path, bottle)
		}
		maxFlow += bottle

		// 5) Augment along the path
		for i := 0; i < len(path)-1; i++ {
			u, v := path[i], path[i+1]
			// decrease forward capacity
			for _, e := range residual.AdjacencyList()[u][v] {
				e.Weight = int64(math.Max(0, float64(e.Weight)-bottle))
			}
			// increase reverse capacity
			found := false
			for _, re := range residual.AdjacencyList()[v][u] {
				re.Weight = int64(float64(re.Weight) + bottle)
				found = true
			}
			if !found {
				// create reverse edge if missing
				residual.AddEdge(v, u, int64(bottle))
			}
		}
	}

	return maxFlow, residual, nil
}

// bfsAugmentingPath finds the shortest (fewest-edges) path in residual
// from source→sink with positive capacity > eps, and returns that path
// plus its bottleneck capacity. Returns nil if no path found.
func bfsAugmentingPath(
	ctx context.Context,
	g *core.Graph,
	source, sink string,
	eps float64,
) ([]string, float64) {
	// parent[v] = predecessor of v on the path
	parent := make(map[string]string, len(g.Vertices()))
	// capMap[v] = bottleneck capacity from source→v
	capMap := map[string]float64{source: math.Inf(1)}
	visited := map[string]bool{source: true}

	queue := []string{source}
	for len(queue) > 0 {
		// context cancellation check
		select {
		case <-ctx.Done():
			return nil, 0
		default:
		}
		u := queue[0]
		queue = queue[1:]
		for _, nbr := range g.Neighbors(u) {
			v := nbr.ID
			if visited[v] {
				continue
			}
			// sum capacity of all parallel edges u→v
			var capSum float64
			for _, e := range g.AdjacencyList()[u][v] {
				capSum += float64(e.Weight)
			}
			if capSum <= eps {
				continue
			}
			visited[v] = true
			parent[v] = u
			capMap[v] = math.Min(capMap[u], capSum)
			if v == sink {
				// reconstruct path
				path := []string{sink}
				for cur := sink; cur != source; {
					p := parent[cur]
					path = append([]string{p}, path...)
					cur = p
				}
				return path, capMap[sink]
			}
			queue = append(queue, v)
		}
	}
	return nil, 0
}
