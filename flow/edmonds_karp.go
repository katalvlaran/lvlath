package flow

import (
	"fmt"
	"math"

	"github.com/katalvlaran/lvlath/core"
)

// EdmondsKarp computes the maximum flow from `source` to `sink` in the
// directed, weighted graph `g` using the Edmonds–Karp algorithm, which
// finds shortest augmenting paths via BFS.
//
// It returns:
//   - maxFlow       : the total flow value (int64)
//   - residualGraph : a *core.Graph of remaining capacities, preserving
//     all original graph options (directed, weighted,
//     multi-edges, loops, mixed)
//   - err           : ErrSourceNotFound, ErrSinkNotFound, EdgeError,
//     or context cancellation error
//
// Steps:
//  1. Normalize options and capture context (O(1)).
//  2. Validate that `source` and `sink` exist in `g` (O(1)).
//  3. Build the initial capacity map via buildCapMap
//     (O(V + E*log d_max) for neighbor sorting).
//  4. Repeat until no augmenting path:
//     a. Check for cancellation (O(1)).
//     b. BFS from source to sink to find the shortest augmenting path
//     and record the bottleneck for each vertex (O(V + E)).
//     c. If no path or bottleneck == 0, break.
//     d. Augment flow along the path (O(path length)).
//     e. Optionally log the path if opts.Verbose.
//  5. Construct the final residual graph via buildCoreResidualFromCapMap
//     (O(V + E_res)), inheriting all graph flags from `g`.
//
// Complexity:
//
//	Time:   O(V * E²) worst-case on integer capacities (BFS per augmentation).
//	Memory: O(V + E) for capMap and BFS auxiliary maps.
func EdmondsKarp(
	g *core.Graph,
	source, sink string,
	opts FlowOptions,
) (maxFlow float64, residualGraph *core.Graph, err error) {
	// 1) Ensure opts.Ctx and opts.Epsilon are set to defaults if zero-valued.
	opts.normalize()
	// 1a) Capture the normalized context for cancellation.
	ctx := opts.Ctx

	// 2) Validate presence of source vertex in graph.
	if !g.HasVertex(source) {
		return 0, nil, ErrSourceNotFound
	}
	// 2a) Validate presence of sink vertex in graph.
	if !g.HasVertex(sink) {
		return 0, nil, ErrSinkNotFound
	}

	// 3) Build the initial capacity map: capMap[u][v] is total capacity
	//    from u→v after summing parallel edges and filtering by Epsilon.
	capMap, err := buildCapMap(g, opts)
	if err != nil {
		return 0, nil, err
	}

	// 4) Main loop: find and augment along shortest paths until none remain.
	for {
		// 4a) Cancellation check before starting BFS.
		if err = ctx.Err(); err != nil {
			return maxFlow, nil, err
		}

		// 4b) Prepare BFS auxiliary structures:
		//    parent[v] = preceding vertex on shortest path to v.
		parent := make(map[string]string, len(capMap))
		//    bottle[v] = bottleneck capacity from source to v.
		bottle := make(map[string]float64, len(capMap))

		// Initialize all bottlenecks to zero.
		for u := range capMap {
			bottle[u] = 0
		}
		// Source has effectively infinite initial capacity.
		bottle[source] = math.MaxInt64

		// Initialize BFS queue with the source vertex.
		queue := []string{source}
		found := false // flag to indicate if we reached sink

		// 4c) BFS loop over queue to build level graph and record parents.
		for i := 0; i < len(queue); i++ {
			// 4c.i) Cancellation check inside BFS iterations.
			if err = ctx.Err(); err != nil {
				return maxFlow, nil, err
			}

			u := queue[i] // dequeue next vertex
			// Explore each neighbor v with positive residual capacity.
			for v, capUV := range capMap[u] {
				// Only consider v if there's capacity and we haven't visited it yet.
				if capUV > 0 && bottle[v] == 0 {
					// Record the parent to reconstruct path later.
					parent[v] = u

					// Compute new bottleneck = min(capUV, bottle[u]).
					if capUV < bottle[u] {
						bottle[v] = capUV
					} else {
						bottle[v] = bottle[u]
					}

					// If we've reached the sink, we can stop BFS early.
					if v == sink {
						found = true
						break
					}

					// Otherwise, enqueue v for further exploration.
					queue = append(queue, v)
				}
			}
			// If sink was found, break out of BFS outer loop.
			if found {
				break
			}
		}

		// 4d) Check if an augmenting path was found and has positive bottleneck.
		flowValue := bottle[sink]
		if !found || flowValue == 0 {
			// No more augmenting paths - algorithm terminates.
			break
		}

		// 4e) If verbose logging is enabled, reconstruct and print the path.
		if opts.Verbose {
			// Rebuild path from sink to source via parent links.
			var path []string
			for v := sink; v != source; v = parent[v] {
				// Prepend each vertex to build path in correct order.
				path = append([]string{v}, path...)
			}
			// Prepend the source at the beginning.
			path = append([]string{source}, path...)
			fmt.Printf("augmenting path %v with flow %g\n", path, flowValue)
		}

		// 4f) Increase total max flow by the bottleneck of this path.
		maxFlow += flowValue

		// 4g) Update residual capacities along the path:
		//     decrease forward edges and increase reverse edges.
		for v := sink; v != source; v = parent[v] {
			u := parent[v]
			// Subtract the flow from the forward edge u→v.
			capMap[u][v] -= flowValue
			// Add the flow to the reverse edge v→u.
			capMap[v][u] += flowValue
		}
	}

	// 5) Construct the final residual graph from capMap,
	//    inheriting all flags (directed, weighted, multi-edges, loops, mixed).
	residualGraph, err = buildCoreResidualFromCapMap(capMap, g, opts)
	if err != nil {
		return maxFlow, nil, err
	}

	// Return the computed maximum flow and the residual graph.
	return maxFlow, residualGraph, nil
}
