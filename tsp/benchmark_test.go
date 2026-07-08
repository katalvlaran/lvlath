// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package tsp benchmarks the private dense Blossom MWPM engine.
//
// These benchmarks intentionally live in package tsp rather than tsp_test,
// because solveMinimumWeightPerfectMatching is not public API. Exposing test-only
// wrappers just to benchmark private internals would pollute the package surface.
package tsp

import "testing"

var benchmarkBlossomMatchSink []int

// benchmarkBlossomMWPM runs the dense Blossom engine on a seeded complete
// matching problem and verifies the exported perfect matching.
//
// Implementation:
//   - Stage 1: Build the seeded dense problem before timer reset.
//   - Stage 2: Run Blossom in the hot loop.
//   - Stage 3: Verify the perfect matching to prevent dead-code elimination.
//
// Behavior highlights:
//   - Measures the real exact MWPM engine used by Christofides BlossomMatch.
//   - Does not route through Christofides.
//   - Does not include matrix construction cost.
//
// Inputs:
//   - b: Go benchmark harness.
//   - k: even matching cardinality.
//   - seed: deterministic problem seed.
//
// Returns:
//   - None.
//
// Errors:
//   - Benchmark fails immediately on Blossom or verification errors.
//
// Determinism:
//   - Problem generation and solver tie-breaks are deterministic.
//
// Complexity:
//   - Solver complexity is the dense Blossom engine complexity for k vertices.
//   - Benchmark setup stores O(k^2) weights.
//
// Notes:
//   - Keep this benchmark private until a top-level matching package exists.
//
// AI-Hints:
//   - Do not replace this with Christofides benchmarks; they measure different regimes.
//   - Do not expose solveMinimumWeightPerfectMatching for this benchmark alone.
func benchmarkBlossomMWPM(b *testing.B, k int, seed int64) {
	b.Helper()

	problem := internalSeededMatchingProblem(k, seed)
	opts := blossomOptions{Eps: DefaultEps}

	b.ReportAllocs()
	b.ResetTimer()

	for iteration := 0; iteration < b.N; iteration++ {
		match, _, _, err := solveMinimumWeightPerfectMatching(problem, opts)
		if err != nil {
			b.Fatalf("Blossom MWPM failed: %v", err)
		}
		if err = verifyPerfectMatching(match); err != nil {
			b.Fatalf("verify perfect matching: %v", err)
		}

		benchmarkBlossomMatchSink = match
	}
}

func BenchmarkBlossomMWPM_K32Dense(b *testing.B) {
	benchmarkBlossomMWPM(b, 32, 3200)
}

func BenchmarkBlossomMWPM_K64Dense(b *testing.B) {
	benchmarkBlossomMWPM(b, 64, 6400)
}

func BenchmarkBlossomDenseK128Dense(b *testing.B) {
	benchmarkBlossomMWPM(b, 128, 12800)
}
