// Package builder provides internal configuration types and functional options
// for graph constructors. It centralizes common settings such as random number
// generator, vertex ID scheme, and edge weight distribution to keep builder
// implementations DRY and consistent.
//
// The key type is BuilderOption, a function that mutates a builderConfig.
// builderConfig holds three fields:
//   - rng:     *rand.Rand source for randomness (nil → deterministic).
//   - idFn:    IDFn to produce vertex identifiers from integer indices.
//   - weightFn: WeightFn to produce edge weights given an RNG.
//
// Use newBuilderConfig to obtain a config with sensible defaults, then apply
// any number of BuilderOption in order. Later options override earlier ones.
//
// Complexity: newBuilderConfig applies N options in O(N) time, O(1) extra space.
package builder

import (
	"math/rand"
)

// BuilderOption customizes the behavior of a graph constructor.
// It mutates the builderConfig before graph construction begins.
//
// As a rule, option constructors never panic at runtime, and ignore nil inputs.
type BuilderOption func(cfg *builderConfig)

// builderConfig holds the configurable parameters for graph builders:
//   - rng:     source of randomness (nil means deterministic).
//   - idFn:    function mapping index→vertex ID (IDFn).
//   - weightFn: function mapping rng→edge weight (WeightFn).
//
// builderConfig is not safe for concurrent mutation; each builder invocation
// should create its own config via newBuilderConfig.
type builderConfig struct {
	rng      *rand.Rand // optional RNG; nil means deterministic behavior
	idFn     IDFn       // function to generate vertex IDs from indices
	weightFn WeightFn   // function to generate edge weights
}

// newBuilderConfig returns a builderConfig initialized with defaults, then
// applies each provided BuilderOption in order. If opts is empty, returns
// defaults: nil RNG, DefaultIDFn, DefaultWeightFn.
//
// Complexity: O(len(opts)) time, O(1) extra space.
func newBuilderConfig(opts ...BuilderOption) *builderConfig {
	// Initialize defaults
	cfg := &builderConfig{
		rng:      nil,             // no RNG → deterministic ID and weight functions
		idFn:     DefaultIDFn,     // decimal IDs "0","1",…
		weightFn: DefaultWeightFn, // constant DefaultEdgeWeight
	}

	// Apply each option in order; later options override earlier ones
	var opt BuilderOption
	for _, opt = range opts {
		opt(cfg)
	}

	return cfg
}

// WithIDScheme injects a custom IDFn into the builderConfig.
// If idFn is nil, this option is a no-op.
// Complexity: O(1) time, O(1) space.
func WithIDScheme(idFn IDFn) BuilderOption {
	return func(cfg *builderConfig) {
		if idFn != nil {
			cfg.idFn = idFn
		}
	}
}

// WithWeightFn injects a custom WeightFn into the builderConfig.
// If wfn is nil, this option is a no-op.
// Complexity: O(1) time, O(1) space.
func WithWeightFn(wfn WeightFn) BuilderOption {
	return func(cfg *builderConfig) {
		if wfn != nil {
			cfg.weightFn = wfn
		}
	}
}

// WithRand sets an explicit *rand.Rand source for randomness.
// If rng is nil, this option is a no-op and leaves the original RNG.
// Complexity: O(1) time, O(1) space.
func WithRand(rng *rand.Rand) BuilderOption {
	return func(cfg *builderConfig) {
		if rng != nil {
			cfg.rng = rng
		}
	}
}

// WithSeed creates a new *rand.Rand seeded with the given value and
// assigns it as the RNG source. Use this for reproducible randomness.
// Complexity: O(1) time, O(1) space.
func WithSeed(seed int64) BuilderOption {
	return func(cfg *builderConfig) {
		cfg.rng = rand.New(rand.NewSource(seed))
	}
}
