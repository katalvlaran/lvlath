// SPDX-License-Identifier: MIT
// Package: lvlath/builder
//
// impl_config.go — internal configuration and deterministic defaults.
//
// Design:
//   • builderConfig is the single source of truth for all builder knobs.
//   • Defaults are deterministic and documented; no globals.
//   • newBuilderConfig applies options in-order (later overrides earlier).
//
// Deterministic defaults (no surprises):
//   • idFn        = decimalID          ("0","1","2",...)
//   • rng         = nil                 (pure/deterministic unless seeded)
//   • weightFn    = constWeight(defaultConstWeight)
//   • left/right  = "L" / "R"
//   • amplitude   = 1.0
//   • frequency   = 1.0
//   • trendK      = 0.0
//   • noiseSigma  = 0.0
//
// AI-Hints:
//   • Set WithSeed for reproducible RandomSparse/RandomRegular fixtures.
//   • Override WithIDScheme for human-readable labels in examples/golden tests.
//   • WithPartitionPrefix influences K_{m,n} side IDs only (bipartite).
//   • Weight policy matters only if the core graph is weighted.

package builder

import (
	"math/rand" // RNG for stochastic builders
	"strconv"   // decimal vertex IDs ("0","1",...)
)

// builderConfig aggregates all knobs used by constructors.
// It is passed by VALUE to constructors (immutable to callers).
type builderConfig struct {
	// Vertex ID strategy: index -> ID (deterministic).
	idFn func(int) string
	// RNG for stochastic choices; nil means “no randomness”.
	rng *rand.Rand
	// Weight generator for edges; used only for weighted graphs.
	weightFn func(*rand.Rand) int64

	// Bipartite ID prefixes (left/right). Empty → defaults resolved below.
	leftPrefix  string
	rightPrefix string

	// Sequence dataset controls (Pulse/Chirp/OHLC).
	amplitude  float64 // >0
	frequency  float64 // >0 for periodic/ chirp
	trendK     float64 // any real
	noiseSigma float64 // >=0
}

// Deterministic defaults (named, no magic numbers).
const (
	defaultLeftPrefix  = "L"      // bipartite left side label
	defaultRightPrefix = "R"      // bipartite right side label
	defaultAmplitude   = 1.0      // sequence amplitude
	defaultFrequency   = 1.0      // base frequency
	defaultTrend       = 0.0      // linear trend coefficient
	defaultNoiseSigma  = 0.0      // Gaussian noise stdev
	defaultConstWeight = int64(1) // constant edge weight when weighted
)

// newBuilderConfig constructs a config with deterministic defaults and applies
// all options in order. Options may leave some string fields empty; we resolve
// those to defaults here to keep downstream code branch-free.
// Complexity: O(len(opts)) time, O(1) space.
func newBuilderConfig(opts ...BuilderOption) builderConfig {
	// Start with strict, deterministic defaults.
	cfg := builderConfig{
		idFn:        decimalID,                                            // "0","1","2",...
		rng:         nil,                                                  // no RNG unless explicitly set
		weightFn:    func(*rand.Rand) int64 { return defaultConstWeight }, // constant weight
		leftPrefix:  defaultLeftPrefix,                                    // "L"
		rightPrefix: defaultRightPrefix,                                   // "R"
		amplitude:   defaultAmplitude,                                     // 1.0
		frequency:   defaultFrequency,                                     // 1.0
		trendK:      defaultTrend,                                         // 0.0
		noiseSigma:  defaultNoiseSigma,                                    // 0.0
	}

	// Apply options in the given order; last-wins semantics.
	for _, opt := range opts {
		opt(&cfg)
	}

	// Resolve empty bipartite prefixes to defaults (deterministic fallback).
	if cfg.leftPrefix == "" {
		cfg.leftPrefix = defaultLeftPrefix
	}
	if cfg.rightPrefix == "" {
		cfg.rightPrefix = defaultRightPrefix
	}

	// Return by value to encourage immutability for callers.
	return cfg
}

// decimalID renders an index as a base-10 string ("0","1","2",...).
// Deterministic and allocation-light; suitable for golden tests.
func decimalID(i int) string {
	return strconv.Itoa(i)
}
