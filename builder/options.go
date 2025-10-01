// SPDX-License-Identifier: MIT
// Package: lvlath/builder
//
// options.go — functional options for the builder package.
//
// Contract (strict):
//   • Options are functional (type Option func(*builderConfig)).
//   • Option constructors VALIDATE and PANIC on meaningless inputs
//     (per lvlath 99-rules). Algorithms themselves MUST NOT panic.
//   • Determinism is explicit: seeding is done via WithSeed or WithRand.
//   • No hidden globals; everything flows through builderConfig.
//
// AI-Hints:
//   • Prefer WithSeed for reproducible stochastic builders (Random*).
//   • Use WithIDScheme to align vertex labels across tests/golden files.
//   • WithPartitionPrefix controls K_{m,n} labels; empty values mean
//     “use defaults”, not an error (deterministic fallback).
//   • WithWeightFn affects weighted graphs only; core controls whether
//     weights are observed.

package builder

import (
	"math/rand" // RNG source for stochastic builders
)

// BuilderOption customizes the behavior of a constructor by mutating a
// builderConfig instance before graph construction begins.
// Complexity: applying N options costs O(N) time, O(1) space.
type BuilderOption func(*builderConfig)

// WithIDScheme sets the deterministic vertex ID generator: idx -> string.
// Panics on nil to surface programmer error early and keep invariants tight.
// Complexity: O(1) time, O(1) space.
func WithIDScheme(fn func(int) string) BuilderOption {
	if fn == nil {
		// Fail fast: option constructors validate and panic (99-rules).
		panic("builder: WithIDScheme(nil)")
	}
	return func(c *builderConfig) {
		// Assign the provided function; used by all topology builders.
		c.idFn = fn
	}
}

// WithRand provides an explicit RNG for stochastic builders.
// Panics on nil; prefer WithSeed for reproducible runs.
// Complexity: O(1) time, O(1) space.
func WithRand(r *rand.Rand) BuilderOption {
	if r == nil {
		// Fail fast to avoid silent non-determinism later.
		panic("builder: WithRand(nil)")
	}
	return func(c *builderConfig) {
		// Attach the RNG; callers decide the seed policy.
		c.rng = r
	}
}

// WithSeed creates a new *rand.Rand with the given seed (deterministic).
// Use this in tests and examples to lock outcomes.
// Complexity: O(1) time, O(1) space.
func WithSeed(seed int64) BuilderOption {
	return func(c *builderConfig) {
		// Seeded source → reproducible shuffles/draws.
		c.rng = rand.New(rand.NewSource(seed))
	}
}

// WithWeightFn overrides the per-edge weight generator.
// The function receives the (possibly nil) RNG and MUST be pure w.r.t.
// input RNG state to preserve determinism. Panics on nil.
// Complexity: O(1) time, O(1) space.
func WithWeightFn(fn func(*rand.Rand) int64) BuilderOption {
	if fn == nil {
		// Fail fast; weight policy must be explicit if customized.
		panic("builder: WithWeightFn(nil)")
	}
	return func(c *builderConfig) {
		// Store generator; used only when the core graph is weighted.
		c.weightFn = fn
	}
}

// WithPartitionPrefix sets bipartite side labels (left/right).
// Empty values are allowed and interpreted as “use defaults” in config.
// Complexity: O(1) time, O(1) space.
func WithPartitionPrefix(left, right string) BuilderOption {
	return func(c *builderConfig) {
		// Store as provided; defaults will be resolved in newBuilderConfig.
		c.leftPrefix, c.rightPrefix = left, right
	}
}

// WithAmplitude sets the sequence amplitude A (>0) for datasets (Pulse/Chirp/OHLC).
// Panics if A <= 0 to avoid degenerate outputs.
// Complexity: O(1) time, O(1) space.
func WithAmplitude(A float64) BuilderOption {
	if A <= 0 {
		panic("builder: WithAmplitude(A<=0)")
	}
	return func(c *builderConfig) {
		// Deterministic scalar controlling signal scale.
		c.amplitude = A
	}
}

// WithFrequency sets the base frequency f0 (>0) for chirps/periodic pulses.
// Panics if f0 <= 0.
// Complexity: O(1) time, O(1) space.
func WithFrequency(f0 float64) BuilderOption {
	if f0 <= 0 {
		panic("builder: WithFrequency(f0<=0)")
	}
	return func(c *builderConfig) {
		// Fundamental frequency parameter for signal synthesis.
		c.frequency = f0
	}
}

// WithTrend sets the linear trend coefficient k for sequences.
// Any real value is accepted (including 0).
// Complexity: O(1) time, O(1) space.
func WithTrend(k float64) BuilderOption {
	return func(c *builderConfig) {
		// Adds k*t to samples; exact usage is defined in impl_sequences.go.
		c.trendK = k
	}
}

// WithNoise sets Gaussian noise sigma (>=0) for sequences.
// Panics if sigma < 0. Noise draws are seeded by c.rng.
// Complexity: O(1) time, O(1) space.
func WithNoise(sigma float64) BuilderOption {
	if sigma < 0 {
		panic("builder: WithNoise(sigma<0)")
	}
	return func(c *builderConfig) {
		// Standard deviation for additive noise; 0 means noiseless.
		c.noiseSigma = sigma
	}
}
