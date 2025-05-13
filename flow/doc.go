// Package flow provides maximum-flow algorithms on *core.Graph.
//
// Supported methods:
//   - Ford–Fulkerson (DFS augmenting paths),  O(E·maxFlow)
//   - Edmonds–Karp (BFS shortest augmenting), O(V·E^2)
//   - Dinic (level graph + blocking flow),   O(E·√V)
//
// Use flow when modeling network capacities, resource distribution,
// and partitioning problems requiring max-flow computation.
package flow
