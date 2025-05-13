// Package core provides the fundamental graph data structures and thread-safe
// operations for in-memory graphs.
//
// Features:
//   - Graph: directed/undirected, weighted/unweighted
//   - Thread-safe mutations with R/W locks
//   - Core types: Vertex, Edge
//   - Cloning: CloneEmpty (same vertices, no edges), Clone (deep copy of edges)
//
// Use core when you need low-level control over vertices, edges, and safe
// graph mutations. Algorithms and higher‑level abstractions build on this package.
package core
