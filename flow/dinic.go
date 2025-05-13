package flow

import (
	"fmt"
	"math"

	"github.com/katalvlaran/lvlath/core"
)

// Dinic computes the maximum flow from source→sink using Dinic’s algorithm:
//  1. Build level graph via BFS.
//  2. Repeatedly send blocking flows via DFS on the level graph.
//  3. Optionally rebuild the level graph every LevelRebuildInterval augmentations.
//
// Returns the maxFlow value, a residual-capacity copy of the graph, or an error.
//
// Options (nil uses defaults):
//   - Epsilon: treat capacities ≤ Epsilon as zero (default 1e-9).
//   - Verbose:  print each augmentation via fmt.Printf.
//   - LevelRebuildInterval: after this many augmentations, rebuild the level graph.
//
// Complexity: O(E · √V)
// Memory:     O(V + E)
func Dinic(
	g *core.Graph,
	source, sink string,
	opts *FlowOptions,
) (maxFlow float64, residual *core.Graph, err error) {
	// --- 1) Set ε and validate
	eps := 1e-9
	if opts != nil && opts.Epsilon > 0 {
		eps = opts.Epsilon
	}
	if !g.HasVertex(source) {
		return 0, nil, ErrSourceNotFound
	}
	if !g.HasVertex(sink) {
		return 0, nil, ErrSinkNotFound
	}

	// --- 2) Build residual graph (clone vertices, sum parallel edges)
	residual = core.NewGraph(g.Directed(), true)
	for _, v := range g.Vertices() {
		residual.AddVertex(&core.Vertex{ID: v.ID, Metadata: v.Metadata})
	}
	for u, nbrs := range g.AdjacencyList() {
		for vID, edges := range nbrs {
			var capSum float64
			for _, e := range edges {
				c := float64(e.Weight)
				if c < -eps {
					return 0, nil, EdgeError{From: u, To: vID, Cap: c}
				}
				capSum += c
			}
			if capSum > eps {
				residual.AddEdge(u, vID, int64(capSum))
			}
		}
	}

	// helper to build level graph and neighbor lists
	type ctx struct {
		level map[string]int
		next  map[string][]string
	}
	buildLevel := func() (c ctx, ok bool) {
		// BFS to set level[v] = distance (#edges) from source
		level := make(map[string]int, len(residual.Vertices()))
		for _, v := range residual.Vertices() {
			level[v.ID] = -1
		}
		queue := []string{source}
		level[source] = 0

		for i := 0; i < len(queue); i++ {
			u := queue[i]
			for _, e := range residual.AdjacencyList()[u] {
				v := e[0].To.ID
				if level[v] < 0 && float64(e[0].Weight) > eps {
					level[v] = level[u] + 1
					queue = append(queue, v)
				}
			}
		}
		if level[sink] < 0 {
			return ctx{}, false
		}
		// build adjacency for blocking flow (only forward edges in level graph)
		next := make(map[string][]string, len(level))
		for u, nbrs := range residual.AdjacencyList() {
			for _, edges := range nbrs {
				for _, e := range edges {
					v := e.To.ID
					if level[v] == level[u]+1 && float64(e.Weight) > eps {
						next[u] = append(next[u], v)
					}
				}
			}
		}

		return ctx{level: level, next: next}, true
	}

	// --- 3) Dinic main loop
	for {
		state, ok := buildLevel()
		if !ok {
			break // no more augmenting paths
		}
		iter := make(map[string]int, len(state.next))
		augCount := 0

		// blocking‐flow loop
		for {
			// DFS push
			pushed := dfsPush(residual, source, sink, math.Inf(1), state, iter, eps)
			if pushed <= eps {
				break
			}
			maxFlow += pushed
			augCount++
			if opts != nil && opts.Verbose {
				fmt.Printf("Dinic push %.3g (total=%.3g)\n", pushed, maxFlow)
			}
			// rebuild level graph on interval
			if opts != nil && opts.LevelRebuildInterval > 0 &&
				augCount%opts.LevelRebuildInterval == 0 {
				var ok2 bool
				if state, ok2 = buildLevel(); !ok2 {
					break
				}
				iter = make(map[string]int, len(state.next))
			}
		}
	}

	return maxFlow, residual, nil
}

// dfsPush tries to send flow f from u→sink along the level graph.
// Returns amount actually sent (≤ f), updating residual capacities.
func dfsPush(
	residual *core.Graph,
	u, sink string,
	f float64,
	state struct {
		level map[string]int
		next  map[string][]string
	},
	iter map[string]int,
	eps float64,
) float64 {
	if u == sink {
		return f
	}
	neighbors := state.next[u]
	for i := iter[u]; i < len(neighbors); i++ {
		v := neighbors[i]
		iter[u] = i + 1 // advance iterator
		// check current residual capacity
		e := residual.AdjacencyList()[u][v][0]
		capUV := float64(e.Weight)
		if capUV <= eps {
			continue
		}
		// compute how much we can push
		minF := f
		if capUV < minF {
			minF = capUV
		}
		if minF <= eps {
			continue
		}
		// recurse
		pushed := dfsPush(residual, v, sink, minF, state, iter, eps)
		if pushed > eps {
			// reduce forward capacity
			e.Weight = int64(math.Max(0, capUV-pushed))
			// increase reverse capacity
			reList := residual.AdjacencyList()[v][u]
			if len(reList) > 0 {
				reList[0].Weight += int64(pushed)
			} else {
				residual.AddEdge(v, u, int64(pushed))
			}
			return pushed
		}
	}
	return 0
}
