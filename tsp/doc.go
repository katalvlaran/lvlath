// Package tsp provides Travelling Salesman Problem solvers.
//
// It includes two algorithms on a distance matrix ([][]float64):
//
//   - TSPExact — uses the Held–Karp dynamic‐programming algorithm.
//
//   - Complexity: O(n²·2ⁿ)
//
//   - Memory:     O(n·2ⁿ)
//
//   - Supports “missing” edges via math.Inf(1).
//
//   - TSPApprox — Christofides’ 1.5-approximation (coming soon).
//
//   - Complexity: O(n³)
//
// All functions accept a complete or partially complete distance matrix:
//   - A distance of math.Inf(1) signals “no direct edge.”
//   - If no tour exists, TSPExact returns ErrTSPIncompleteGraph.
//
// Use this package when you need to solve or approximate the TSP
// on small‐to-medium sized instances (n≲16 for TSPExact).
package tsp
