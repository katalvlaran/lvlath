// Package matrix offers matrix-based graph representations and converters.
//
// The matrix package provides:
//
//   - Lightweight converters (ToMatrix, ToEdgeList) for exporting graphs to
//     linear-algebra routines or external formats.
//   - AdjacencyMatrix with O(1) edge‐weight lookups and O(V²) memory.
//   - IncidenceMatrix for vertex‐by‐edge incidence queries, useful in
//     graph-theoretic analyses.
//
// Matrices are best for dense or small graphs where O(V²) memory and
// O(V² + E) build time are acceptable.
//
// See the examples in this package and core for usage patterns.
package matrix
