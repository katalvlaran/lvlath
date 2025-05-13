// Package algorithms implements classic graph algorithms on core.Graph.
//
// It provides free-function implementations of:
//
//   - Traversals
//     – BFS (Breadth-First Search)
//     – DFS (Depth-First Search)
//
//   - Shortest paths
//     – Dijkstra
//
//   - Minimum spanning trees
//     – Prim
//     – Kruskal
//
// All functions accept *core.Graph and return simple Go types (slices, maps).
// Hookable options (BFSOptions, DFSOptions) let you inject custom logic
// during traversal. See README.md for usage examples and detailed docs.
package algorithms
