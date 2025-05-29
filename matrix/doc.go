// Package matrix provides matrix-based representations of graphs and related utilities.
//
// The matrix package enables efficient construction, manipulation, and analysis of
// graph representations in matrix form, supporting both adjacency and incidence
// matrices, along with spectral (eigen-) analysis of graphs.
//
// What
//   - BuildAdjacency: construct an N×N weighted or unweighted adjacency matrix
//     from a list of vertices and edges.
//   - BuildIncidence: construct a V×E incidence matrix to map vertices to edges.
//   - Transpose: compute the transpose of a matrix, swapping rows and columns.
//   - Multiply: perform matrix multiplication (adjacency matrix product).
//   - DegreeVector: compute per-vertex degree (sum of row entries).
//   - SpectralAnalysis: perform eigen decomposition (Jacobi rotations) to obtain
//     eigenvalues and eigenvectors of a symmetric adjacency matrix.
//   - ToGraph: reconstruct a \*core.Graph from an AdjacencyMatrix, preserving
//     directed/weighted/multi-edge/loop options.
//
// Why
//
//	Adjacency matrices offer O(1) edge-weight lookup ideal for dense graphs,
//	while incidence matrices facilitate algebraic graph operations and
//	Eulerian path/circuit checks. Spectral methods (e.g. clustering,
//	centrality) leverage linear algebra on the adjacency matrix.
//
// Complexity
//
//	BuildAdjacency:   O(V+E) time, O(V²) memory
//	BuildIncidence:   O(V+E) time, O(V·E) memory
//	Transpose:        O(N²) time, O(N²) memory
//	Multiply:         O(N³) time, O(N²) memory
//	DegreeVector:     O(N²) time, O(N) memory
//	SpectralAnalysis: O(N³) time, O(N²) memory
//	ToGraph:          O(V²+E) time, O(V+E) memory
//
// Options
//
//	MatrixOptions configures behavior when building matrices:
//	  - Directed:   treat edges as directed (true) or undirected (false)
//	  - Weighted:   preserve edge weights (true) or treat all edges as weight 1
//	  - AllowMulti: include parallel (multi-) edges (true) or collapse duplicates
//	  - AllowLoops:  include self-loops (true) or skip them
//
//	Create options via:
//	  opts := matrix.NewMatrixOptions(
//	      matrix.WithDirected(true),
//	      matrix.WithWeighted(true),
//	      matrix.WithAllowMulti(false),
//	      matrix.WithAllowLoops(false),
//	  )
//
// Sentinel Errors
//
//	ErrUnknownVertex     // referenced vertex not found in index
//	ErrDimensionMismatch // incompatible matrix dimensions
//	ErrNonBinaryIncidence// non-±1 entry in unweighted incidence matrix
//	ErrEigenFailed       // eigen decomposition did not converge
//	ErrNilGraph          // nil *core.Graph passed to constructor
//
// See docs/matrix.md for a full tutorial and best practices.
package matrix
