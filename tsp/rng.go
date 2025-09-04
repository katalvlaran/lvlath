// Package tsp - RNG utilities shared by heuristic solvers.
//
// This file centralizes deterministic random generation for all TSP heuristics.
//
// Goals:
//   - Determinism: same seed ⇒ identical results across platforms.
//   - Encapsulation: a single RNG factory; no time-based sources hidden anywhere.
//   - Safety: no panics or logging; only sentinel errors from types.go when needed.
//   - Performance: no hidden allocations in hot paths; O(1) helpers, O(n) shuffles.
//
// Concurrency:
//   - math/rand.Rand is NOT goroutine-safe. Do not share a *rand.Rand across goroutines.
//   - Use deriveRNG to create independent streams for parallel restarts or workers.
package tsp

import "math/rand"

// defaultRNGSeed is the fixed “zero” seed used when callers pass seed==0.
// The value is arbitrary but stable to keep reproducible defaults.
const defaultRNGSeed int64 = 1

// rngFromSeed returns a deterministic *rand.Rand.
// Policy: seed==0 ⇒ use defaultRNGSeed; otherwise use the provided seed verbatim.
//
// Complexity: O(1).
func rngFromSeed(seed int64) *rand.Rand {
	var s int64
	s = seed
	if s == 0 {
		s = defaultRNGSeed
	}
	return rand.New(rand.NewSource(s))
}

// deriveSeed mixes a parent seed and a stream identifier into a new 64-bit seed.
//
// Rationale:
//   - We want independent substreams derived from a base RNG (e.g., multi-start heuristics).
//   - We apply a SplitMix64-style avalanche mix to eliminate correlations.
//
// Notes:
//   - Constants are the canonical SplitMix64 multipliers/finalizer. They provide strong
//     bit diffusion; small changes in inputs produce large, well-distributed output changes.
//
// Complexity: O(1).
func deriveSeed(parent int64, stream uint64) int64 {
	// SplitMix64-style finalizer; see Vigna 2014 for the constants and rationale.
	var x uint64
	x = uint64(parent) ^ (stream + 0x9e3779b97f4a7c15)
	x += 0x9e3779b97f4a7c15
	x = (x ^ (x >> 30)) * 0xbf58476d1ce4e5b9
	x = (x ^ (x >> 27)) * 0x94d049bb133111eb
	x ^= x >> 31
	return int64(x)
}

// deriveRNG creates an independent deterministic RNG stream based on a base RNG
// and a stream identifier. If base==nil, defaultRNGSeed is used as the parent.
// Otherwise, base.Int63() is consumed once to decorrelate consecutive derivations,
// then mixed with the stream via deriveSeed.
//
// Usage:
//   - Call during setup (not in hot loops) to create per-worker/per-restart RNGs.
//
// Complexity: O(1).
func deriveRNG(base *rand.Rand, stream uint64) *rand.Rand {
	var parent int64
	if base == nil {
		parent = defaultRNGSeed
	} else {
		// Int63() advances base state; this is intentional to avoid identical
		// children when the same stream id is reused by mistake.
		parent = base.Int63()
	}
	return rand.New(rand.NewSource(deriveSeed(parent, stream)))
}

// shuffleIntsInPlace performs an in-place Fisher–Yates shuffle of a using rng.
// If rng==nil, a deterministic default stream is used (seed==0 policy).
//
// Complexity: O(n) time, O(1) extra space.
func shuffleIntsInPlace(a []int, rng *rand.Rand) {
	var n int
	n = len(a)
	if n <= 1 {
		return
	}

	var (
		r *rand.Rand
		i int
		j int
	)
	r = rng
	if r == nil {
		r = rngFromSeed(0)
	}

	for i = n - 1; i > 0; i-- {
		j = r.Intn(i + 1)
		a[i], a[j] = a[j], a[i]
	}
}

// permRange returns a permutation of 0..n-1 generated deterministically from rng.
// If rng==nil, the default deterministic stream is used. For n<0, returns ErrDimensionMismatch.
// Allocation is required by contract (the returned permutation slice).
//
// Complexity: O(n) time, O(n) space.
func permRange(n int, rng *rand.Rand) ([]int, error) {
	if n < 0 {
		return nil, ErrDimensionMismatch
	}
	p := make([]int, n)

	var i int
	for i = 0; i < n; i++ {
		p[i] = i
	}
	shuffleIntsInPlace(p, rng)
	return p, nil
}
