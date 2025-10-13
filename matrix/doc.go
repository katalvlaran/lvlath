// SPDX-License-Identifier: MIT

// Package matrix provides deterministic, allocation-aware linear algebra
// and graph-by-matrix utilities used across lvlath.
//
// The matrix package enables efficient construction, manipulation, and analysis of
// graph representations in matrix form, supporting both adjacency and incidence
// matrices, along with spectral (eigen-) analysis of graphs.
//
// # Philosophy
//
// - Determinism first: all loops have fixed orders; results are bit-stable across runs.
// - Clear contracts: +Inf denotes “no edge” in adjacency/distances; diag must be 0 for APSP.
// - Zero surprises: fast-paths on *Dense are single-source-of-truth kernels; fallbacks behave identically.
// - No silent policy drift: errors are explicit (dimension/shape/nil/singularity).
//
// What
//
//		(core kernels)
//	  - Add / Sub / Hadamard / Scale / Transpose - element-wise ops and transforms.
//	  - Mul / MatVec - matrix×matrix and matrix×vector.
//	  - Eigen (Jacobi) - symmetric eigen decomposition (values + eigenvectors).
//	  - LU / Inverse / QR - basic factorizations without pivoting (deterministic).
//	  - FloydWarshall - APSP in-place (k→i→j); +Inf=no edge; diag=0.
//	    (graph-related)
//	  - AdjacencyMatrix - dense adjacency with stable vertex order and policies (directed/weighted/loops/multi).
//	  - NewAdjacencyMatrix / ToGraph - round-trip between Core Graph and adjacency (metric-closure guarded).
//	  - MetricClosure - APSP over adjacency (distances); marks matrix as non-exportable to edges.
//	  - DegreeVector - row-sum per vertex; loops count as 1; +Inf/NaN ignored.
//	  - Neighbors - adjacency scan with deterministic order.
//	    (facades & convenience)
//	  - T / Sum / Diff / Product / HadamardProd / ScaleBy / MatVecMul - thin aliases to kernels.
//	  - RowSums/ColSums - reductions via MatVec, useful for stochastic normalization and features.
//	  - ZerosLike / IdentityLike / NewZeros / NewIdentity - intention-revealing constructors.
//	  - Symmetrize - (A + Aᵀ)/2 to repair asymmetry for spectral methods.
//	  - BuildMetricClosure - adjacency build + APSP + policy flag.
//	  - CenterColumns / CenterRows - mean-centering by columns/rows (PCA/DTW/Regression).
//	  - NormalizeRowsL1 / NormalizeRowsL2 - row stochasticization / L2-normalization.
//	  - Covariance / Correlation - sample covariance and Pearson correlation (columns).
//	  - AllClose - PR-friendly equality with tolerance.
//	  - Clip / ReplaceInfNaN - pipeline protection for stochastic simulations.
//
// Why
//
//	Dense adjacency allows O(1) edge lookup and predictable throughput;
//	spectral and optimization methods benefit from deterministic kernels.
//	Graph workflows (BFS / DFS / Dijkstra / Prim / Kruskal / Flows / DTW / TSP) lean on these matrices,
//	while analytics (PCA / Regression / HMM / Monte-Carlo / SAX) rely on centering, scaling, and reductions.
//
// Complexity (typical)
//
// - Add / Sub / Hadamard / Scale / Transpose: O(rc) time, O(rc) space
// - Mul:                                      O(rnc) time, O(rc) space
// - MatVec:                                   O(rc) time, O(r) space
// - Eigen (Jacobi):                           O(maxIter*n³) time, O(n²) space
// - LU / Inverse / QR:                        O(n³) time, O(n²) space
// - FloydWarshall:                            O(n³) time, O(1) extra (in-place)
// - DegreeVector / RowSums / ColSums:         O(n²) time, O(n) space
// - Covariance / Correlation:                 O(rc + c²) time, O(rc + c²) space
//
// Options (adjacency build)
//
//	Directed:    orient edges; Undirected: mirror without duplicating loops.
//	Weighted:    keep edge weights; otherwise degrade to binary (1).
//	AllowMulti:  preserve first edge (deterministic) when false; allow parallels when true.
//	AllowLoops:  admit self-loops; DegreeVector counts a loop as 1 if present.
//	MetricClosure: distance matrices are marked non-exportable via ToGraph.
//
// Sentinel Errors
//
//	ErrNilMatrix, ErrDimensionMismatch, ErrNotSymmetric, ErrSingular,
//	ErrUnknownVertex, ErrGraphNil, ErrMatrixNotImplemented, ErrEigenFailed (or policy name).
//
// Best practices
// - Prefer *Dense to unlock fast-paths and minimize interface dispatch.
// - Use +Inf for missing edges and keep diagonal 0 for APSP/MetricClosure.
// - Pin determinism in tests: repeat runs, AllClose with agreed ε.
// - ReplaceInfNaN / Clip before statistics to harden pipelines.
//
// See
//   - package-level docs and tests for usage patterns and guarantees
//   - docs/matrix.md for a full tutorial and best practices.
package matrix
