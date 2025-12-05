package flow

import (
	"context"
	"fmt"
	"math"

	"github.com/katalvlaran/lvlath/core"
)

// Dinic computes the maximum flow from `source` to `sink` in the
// directed, weighted graph `g` using Dinic’s algorithm (level graph + blocking flows).
//
// It returns:
//   - maxFlow       : the total flow value (float64)
//   - residualGraph : a *core.Graph of remaining capacities, preserving
//     all original graph options (directed, weighted,
//     multi-edges, loops, mixed)
//   - err           : ErrSourceNotFound, ErrSinkNotFound, EdgeError,
//     or context cancellation error
//
// Steps:
//  1. Normalize options and capture context (O(1)).
//  2. Validate that `source` and `sink` exist in `g` (O(1)).
//  3. Build initial capacity map via buildCapMap
//     (O(V + E*log d_max) due to neighbor sorting).
//  4. Repeat until no more augmenting paths:
//     a. Check for cancellation (O(1)).
//     b. BFS to build the level graph: distance from source for each vertex (O(V + E)).
//     c. If sink unreachable, break.
//     d. Build adjacency list `next` for edges in level graph (O(E)).
//     e. DFS-based blocking flow pushes until none remains,
//     optionally rebuilding level graph every LevelRebuildInterval augmentations.
//  5. Construct final residual graph via buildCoreResidualFromCapMap
//     (O(V + E_res)), inheriting all flags from `g`.
//
// Complexity:
//
//	Time:   O(min(V^(2/3), √E) * E) in general; O(E*√V) on unit‐capacity networks.
//	Memory: O(V + E) for capMap and auxiliary maps (level, next, iter).
func Dinic(
	g *core.Graph,
	source, sink string,
	opts FlowOptions,
) (maxFlow float64, residualGraph *core.Graph, err error) {
	// 1) Normalize options (set default Ctx and Epsilon if needed)
	opts.normalize()
	ctx := opts.Ctx

	// 2) Validate presence of source and sink
	if !g.HasVertex(source) {
		return 0, nil, ErrSourceNotFound
	}
	if !g.HasVertex(sink) {
		return 0, nil, ErrSinkNotFound
	}

	// 3) Build initial capacity map: capMap[u][v] = total float64 capacity from u→v
	capMap, err := buildCapMap(g, opts)
	if err != nil {
		return 0, nil, err
	}

	// 4) Main loop: level graph + blocking flows
	augmentCount := 0
	for {
		// 4a) Cancellation check before BFS
		if err = ctx.Err(); err != nil {
			return maxFlow, nil, err
		}

		// 4b) BFS to compute levels
		level := make(map[string]int, len(capMap))
		for u := range capMap {
			level[u] = -1
		}
		queue := []string{source}
		level[source] = 0
		for i := 0; i < len(queue); i++ {
			u := queue[i]
			for v, capUV := range capMap[u] {
				if capUV > 0 && level[v] < 0 {
					level[v] = level[u] + 1
					queue = append(queue, v)
				}
			}
		}
		// 4c) If sink unreachable in level graph, we're done
		if level[sink] < 0 {
			break
		}

		// 4d) Build level‐graph adjacency: next[u] = neighbors v at level+1
		next := make(map[string][]string, len(capMap))
		for u, nbrs := range capMap {
			for v, capUV := range nbrs {
				if capUV > 0 && level[v] == level[u]+1 {
					next[u] = append(next[u], v)
				}
			}
		}

		// 4e) DFS‐based blocking flow
		iter := make(map[string]int, len(next))
		for {
			// 4e.i) Cancellation check before each DFS push
			if err = ctx.Err(); err != nil {
				return maxFlow, nil, err
			}
			pushed := dfsDinicPush(ctx, capMap, next, iter, source, sink, math.MaxInt64)
			if pushed == 0 {
				break
			}
			maxFlow += pushed
			augmentCount++
			if opts.Verbose {
				fmt.Printf("Dinic: pushed %g, total %g\n", pushed, maxFlow)
			}
			// 4e.ii) Optionally rebuild level graph
			if opts.LevelRebuildInterval > 0 && augmentCount%opts.LevelRebuildInterval == 0 {
				break
			}
		}
	}

	// 5) Construct the final residual graph from capMap,
	//    inheriting all flags from the original graph.
	residualGraph, err = buildCoreResidualFromCapMap(capMap, g, opts)
	if err != nil {
		return maxFlow, nil, err
	}

	return maxFlow, residualGraph, nil
}

// dfsDinicPush recursively pushes flow along the level graph.
// It respects cancellation via ctx, updates capMap in-place,
// and returns the amount actually sent.
func dfsDinicPush(
	ctx context.Context,
	capMap map[string]map[string]float64,
	next map[string][]string,
	iter map[string]int,
	u, sink string,
	available float64,
) float64 {
	// Check for cancellation at DFS entry
	if err := ctx.Err(); err != nil {
		return 0
	}
	// If we reached sink, return the available flow
	if u == sink {
		return available
	}
	// Iterate over neighbors in level graph, starting from iter[u]
	for i := iter[u]; i < len(next[u]); i++ {
		iter[u] = i + 1
		v := next[u][i]
		capUV := capMap[u][v]
		if capUV <= 0 {
			continue
		}
		// Determine how much we can send: min(available, capUV)
		send := available
		if capUV < send {
			send = capUV
		}
		if send == 0 {
			continue
		}
		// Recurse to push from v toward sink
		pushed := dfsDinicPush(ctx, capMap, next, iter, v, sink, send)
		if pushed > 0 {
			// On success, update residual capacities
			capMap[u][v] -= pushed
			capMap[v][u] += pushed

			return pushed
		}
	}

	return 0
}
