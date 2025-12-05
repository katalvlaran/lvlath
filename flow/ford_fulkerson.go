package flow

import (
	"fmt"
	"math"

	"github.com/katalvlaran/lvlath/core"
)

// FordFulkerson computes the maximum flow from `source` to `sink` in the
// directed, weighted graph `g` using the Ford–Fulkerson method (DFS-based
// augmenting paths).
//
// It returns:
//   - maxFlow       : the total flow value (float64)
//   - residualGraph : a *core.Graph of remaining capacities, preserving
//     all original graph options (directed, weighted,
//     multi-edges, loops, mixed)
//   - err           : ErrSourceNotFound, ErrSinkNotFound, EdgeError, or
//     context cancellation error
//
// Steps:
//  1. Normalize options (O(1)).
//  2. Validate source and sink exist (O(1)).
//  3. Build initial capacity map via buildCapMap (O(V + E*log d_max)).
//  4. Repeat until no augmenting path:
//     a. Iteratively DFS to find any path s→t with positive capacity (O(E)).
//     b. If none found, break.
//     c. Augment along path, updating capMap (O(path length)).
//     d. Accumulate flow; if opts.Verbose, log path and delta.
//     e. Check ctx for cancellation.
//  5. Reconstruct residual *core.Graph from capMap via buildCoreResidualFromCapMap (O(V + E_res)).
//
// Complexity:
//
//	Time:   O(E * F) where F = maxFlow (sum of all augmentations).
//	Memory: O(V + E) for capMap and DFS stack.
//
// Suitable for small to moderate integral networks; for stronger guarantees,
// consider Edmonds–Karp (BFS) or Dinic (level graph + blocking flow).
func FordFulkerson(
	g *core.Graph,
	source, sink string,
	opts FlowOptions,
) (maxFlow float64, residualGraph *core.Graph, err error) {
	// 1) Normalize options to ensure Ctx and Epsilon are set
	opts.normalize()
	// 1a) Capture context for cancellation checks
	ctx := opts.Ctx

	// 2) Validate that source exists in graph
	if !g.HasVertex(source) {
		return 0, nil, ErrSourceNotFound
	}
	// 2a) Validate that sink exists in graph
	if !g.HasVertex(sink) {
		return 0, nil, ErrSinkNotFound
	}

	// 3) Build the initial capacity map:
	//    capMap[u][v] = total integer capacity from u→v after aggregating
	//    parallel edges and filtering by opts.Epsilon.
	capMap, err := buildCapMap(g, opts)
	if err != nil {
		return 0, nil, err
	}

	// 4) Main Ford–Fulkerson loop: find any augmenting path and push flow
	for {
		// 4a) Check for cancellation before each search
		if err = ctx.Err(); err != nil {
			return maxFlow, nil, err
		}

		// 4b) Prepare for iterative DFS
		// parent[v] = preceding vertex on the augmenting path
		parent := make(map[string]string, len(capMap))
		// minCap[v] = bottleneck capacity from source to v along discovered path
		minCap := make(map[string]float64, len(capMap))
		// visited marks which vertices have been pushed onto the stack
		visited := make(map[string]bool, len(capMap))

		// stackEntry holds a node ID and the current bottleneck to that node
		type stackEntry struct {
			node string  // current vertex ID
			flow float64 // bottleneck capacity so far
		}
		// initialize DFS from source with infinite (MaxInt64) capacity
		stack := []stackEntry{{node: source, flow: math.MaxInt64}}
		visited[source] = true         // mark source visited
		minCap[source] = math.MaxInt64 // source has infinite bottleneck
		found := false                 // indicates if sink is reached

		// 4c) Iterative DFS loop
		for len(stack) > 0 && !found {
			// pop last entry (LIFO)
			entry := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			u := entry.node

			// explore each neighbor v with residual capacity capUV
			for v, capUV := range capMap[u] {
				// skip zero/no capacity or already visited vertices
				if capUV <= 0 || visited[v] {
					continue
				}
				// mark v visited and record its parent
				visited[v] = true
				parent[v] = u

				// compute new bottleneck = min(entry.flow, capUV)
				if entry.flow < capUV {
					minCap[v] = entry.flow
				} else {
					minCap[v] = capUV
				}

				// if we reached sink, we can stop DFS
				if v == sink {
					found = true
					break
				}

				// otherwise push v onto stack to continue search
				stack = append(stack, stackEntry{node: v, flow: minCap[v]})
			}
		}

		// 4d) If no augmenting path found, we're done
		if !found {
			break
		}

		// 4e) The amount to add is the bottleneck at sink
		delta := minCap[sink]

		// 4f) Optionally log the augmenting path and flow
		if opts.Verbose {
			// reconstruct path for logging
			path := []string{sink}
			for cur := sink; cur != source; cur = parent[cur] {
				path = append([]string{parent[cur]}, path...)
			}
			fmt.Printf("augmenting path %v with flow %g\n", path, delta)
		}

		// 4g) Accumulate total flow
		maxFlow += delta

		// 4h) Update residual capacities along the path
		for v := sink; v != source; v = parent[v] {
			u := parent[v]
			// decrease forward edge capacity
			capMap[u][v] -= delta
			// increase reverse edge capacity
			capMap[v][u] += delta
		}
	}

	// 5) Build the final residual graph from capMap,
	//    inheriting all configuration flags from the original graph.
	residualGraph, err = buildCoreResidualFromCapMap(capMap, g, opts)
	if err != nil {
		return maxFlow, nil, err
	}

	// return the computed max flow and the residual graph
	return maxFlow, residualGraph, nil
}
