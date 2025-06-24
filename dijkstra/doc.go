// Package dijkstra provides a precise, high-performance implementation of Dijkstra's
// shortest-path algorithm on weighted graphs with non-negative edge weights.
//
// Overview:
//
//   - Dijkstra computes the minimum-cost path from a single source vertex to all
//     reachable vertices in O((V + E) log V) time, where V = |vertices| and E = |edges|.
//   - It relies on a min-heap (priority queue) to always expand the next-closest vertex.
//   - Supports optional path reconstruction, distance caps, and “impassable” edge thresholds.
//
// When to use:
//
//   - In any scenario where you need guaranteed shortest paths on a static weighted graph.
//   - As a foundation for A* or other heuristic searches (substitute heuristics).
//   - As a building block for network routing, traffic simulations, resource allocation,
//     or any domain requiring exact, non-negative shortest paths.
//
// Key features:
//
//   - Functional options allow fine-tuning behavior without changing the API signature.
//   - ReturnPath: if enabled, returns a “predecessor” map, so you can rebuild each path.
//   - MaxDistance: aborts exploration beyond a specified distance, saving work in large graphs.
//   - InfEdgeThreshold: treats any edge with weight ≥ threshold as impassable (infinite cost).
//   - MemoryMode: plan for future “compact” mode that omits predecessor storage (currently Full by default).
//   - Mixed edges support: works correctly on graphs mixing directed and undirected edges.
//
// Performance and complexity:
//
//   - Time:  O((V + E) log V)
//   - Each vertex is extracted at most once from the priority queue (V extracts total).
//   - Each edge relaxation may push one new entry (up to E pushes).
//   - Each heap Push/Pop costs O(log N) where N ≤ V + E, simplified to O(log V).
//   - Space: O(V + E)
//   - O(V) to store distance and (optional) predecessor maps.
//   - O(E) worst-case entries in the heap under “lazy decrease-key” strategy.
//
// Error handling (sentinel errors):
//
//   - ErrEmptySource:
//     Returned if the Source string is empty when calling Dijkstra.
//   - ErrNilGraph:
//     Returned if you pass a nil *core.Graph to Dijkstra.
//   - ErrUnweightedGraph:
//     Returned if the graph is not configured with core.WithWeighted(), since Dijkstra
//     requires non-negative weights.
//   - ErrVertexNotFound:
//     Returned if the specified source vertex does not exist in the graph.
//   - ErrNegativeWeight:
//     Returned if any edge in the graph has a negative weight (detected by a fast O(E) pre-scan).
//   - ErrBadMaxDistance:
//     Returned (via panic) if you set MaxDistance to a negative value.
//   - ErrBadInfThreshold:
//     Returned (via panic) if you set InfEdgeThreshold to zero or a negative value.
//
// API reference:
//
//	func Dijkstra(
//	    g *core.Graph,
//	    opts ...Option,
//	) (dist map[string]int64, prev map[string]string, err error)
//
//	  - g:       pointer to a core.Graph that must be weighted.
//	  - opts:    zero or more functional options, including:
//	      • Source(string):            required, the starting vertex ID.
//	      • WithReturnPath():          if set, returns a predecessor map; otherwise prev == nil.
//	      • WithMaxDistance(int64):    if set, explores only vertices with distance ≤ given value.
//	      • WithInfEdgeThreshold(int64): if set, skips any edge whose weight ≥ threshold.
//	      • WithMemoryMode(MemoryMode): currently Full by default; Compact planned for future.
//	  - dist:    map[v] = minimal distance from Source to v, or math.MaxInt64 if unreachable.
//	  - prev:    map[v] = immediate predecessor of v on one shortest path from Source,
//	              or "" if v is the Source or v is unreachable. Nil if ReturnPath=false.
//	  - err:     one of the sentinel errors (ErrEmptySource, ErrNilGraph, ErrUnweightedGraph,
//	              ErrVertexNotFound, ErrNegativeWeight), or nil on success.
//
// Memory modes:
//
//   - MemoryModeFull (default):   stores a full predecessor map (prev[v] = parent of v) so you can
//     reconstruct any shortest path in O(path length).
//   - MemoryModeCompact (reserved): plans to minimize memory usage by omitting or compressing
//     predecessor data. For now, behaves exactly like Full.
//     In future, enabling Compact may omit prev entirely (prev == nil).
//
// Thread safety:
//
//   - Dijkstra itself is not thread-safe if the same *core.Graph is modified concurrently.
//   - If you need concurrent queries on the same graph, synchronize externally (mutexes, channels, etc.).
//
// See also:
//
//   - core.Graph: graph construction, edge/vertex addition, mixed-edge support.
//   - matrix.NewAdjacencyMatrix / ToGraph: convert to and from adjacency matrices for integration.
//
// Thanks for choosing lvlath! We aim to provide rock-solid graph algorithms that blend
// mathematical rigor, performance, and clarity. If you spot any issue or have suggestions,
// please open an issue or PR on GitHub.
package dijkstra
