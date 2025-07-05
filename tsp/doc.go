// Package tsp provides Travelling Salesman Problem solvers on distance matrices,
// offering both exact and approximate algorithms with unified API and error handling.
//
// What & Why:
//   - Solve the TSP on a complete or partially complete metric graph represented
//     as an n×n [][]float64 distance matrix.
//   - Use TSPExact (Held–Karp) for n ≲ 16 when an optimal solution is required.
//   - Use TSPApprox (Christofides 1.5-approx) for larger n when a fast, near-optimal
//     solution suffices.
//
// Algorithms:
//
//   - TSPExact(dist, opts):
//     – Exact Held–Karp dynamic programming.
//     – Time:    O(n² · 2ⁿ)
//     – Memory:  O(n · 2ⁿ)
//
//   - TSPApprox(dist, opts):
//     – Christofides’ algorithm with greedy perfect matching.
//     – Time:    O(n³)
//     – Memory:  O(n²)
//
// Input Matrix Requirements:
//
//	– Square (n×n) matrix.
//	– dist[i][i] == 0.
//	– dist[i][j] ≥ 0 and dist[i][j] == dist[j][i].
//	– math.Inf(1) signals “missing edge” (used to represent partial graphs).
//
// Options:
//   - type Options struct{}            – placeholder for future parameters.
//   - func DefaultOptions() Options    – always use to initialize opts.
//
// Errors:
//   - ErrBadInput       – invalid matrix (empty, ragged, negative weights,
//     non-zero diagonal, asymmetry).
//   - ErrIncompleteGraph– no Hamiltonian cycle exists (disconnected graph).
//
// See: docs/TSP.md for full tutorial with math, pseudocode, diagrams, and best practices.
package tsp
