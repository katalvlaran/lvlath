// Package dfs implements depth‑first search traversal, cycle detection,
// and topological sort on a core.Graph, supporting both directed and
// undirected graphs where appropriate.
//
// What:
//
//   - DFS (Depth‑First Search): explores as far as possible along each
//     branch before backtracking. Supports:
//   - Pre‑order and post‑order hooks
//   - Cancellation via context.Context
//   - Depth limiting
//   - Neighbor filtering
//   - DetectCycles: enumerates all simple cycles in directed or undirected
//     graphs using vertex coloring (White, Gray, Black) with back‑edge
//     recording and canonical signature deduplication.
//   - TopologicalSort: computes a linear ordering of vertices in a directed
//     acyclic graph (DAG), returning ErrCycleDetected if cycles exist.
//
// Why:
//   - Build and analyze dependency graphs (build systems, package managers, task schedulers)
//   - Determine safe execution orders in DAGs
//   - Detect cycles to prevent infinite loops or inconsistent states
//   - Provide a foundation for SCC detection, connectivity, and pathfinding
//
// Key Types & Constants:
//
//   - VertexState: White, Gray, Black (visitation markers)
//   - Option: functional options for DFS behavior
//   - DFSOptions: holds Context, hooks, MaxDepth, FilterNeighbor
//   - DFSResult: collects post‑order, Depth, Parent, Visited maps
//
// Complexity:
//
//   - DFS:            Time O(V+E), Memory O(V)
//   - DetectCycles:   Time O(V+E + C*L²), Memory O(V+L\_max)
//     (C=#cycles, L=avg cycle length; normalization is O(L²))
//   - TopologicalSort\:Time O(V+E), Memory O(V)
//
// Errors:
//
//   - ErrGraphNil             graph pointer is nil
//   - ErrStartVertexNotFound  start vertex ID not in graph
//   - ErrCycleDetected        cycle discovered in DAG operations
//   - context.Canceled        DFS canceled via context
//   - hook errors             propagated from OnVisit or OnExit
//
// Functions:
//
//   - DFS(g \*core.Graph, startID string, opts ...Option) (\*DFSResult, error)
//     perform depth‑first traversal from startID
//   - DetectCycles(g \*core.Graph) (bool, \[]\[]string, error)
//     report existence and list of simple cycles
//   - TopologicalSort(g \*core.Graph) (\[]string, error)
//     return topological order or ErrCycleDetected
//   - DefaultOptions(), WithContext(), WithOnVisit(), WithOnExit(),
//     WithMaxDepth(), WithFilterNeighbor()
//
// See docs/DFS.md for detailed tutorial, pseudocode, diagrams, and performance analysis.
package dfs
