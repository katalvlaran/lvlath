// Package builder provides internal helper functions and types
// for configuring edge‐weight distributions in graph constructors.
package builder

import (
	"fmt"
	"math"
	"math/rand"
)

// DefaultEdgeWeight is the default weight assigned to each edge when no
// custom WeightFn is provided.
const DefaultEdgeWeight float64 = 1

// WeightFn produces an edge weight given an optional *rand.Rand source.
// It must be deterministic for a given RNG seed; panics in constructors
// indicate programmer error in configuration.
type WeightFn func(rng *rand.Rand) float64

// DefaultWeightFn always returns the constant DefaultEdgeWeight.
// Complexity: O(1) time, O(1) space. Never panics.
func DefaultWeightFn(_ *rand.Rand) float64 {
	return DefaultEdgeWeight
}

// ConstantWeightFn returns a WeightFn that always yields the provided value.
// Panics if value < 0.
// Complexity: O(1) time, O(1) space.
func ConstantWeightFn(value float64) WeightFn {
	if value < 0 {
		panic(fmt.Sprintf("ConstantWeightFn: value must be ≥ 0, got %g", value))
	}

	return func(_ *rand.Rand) float64 {
		return value
	}
}

// UniformWeightFn returns a WeightFn sampling uniformly in [min, max] inclusive.
// Panics if min < 0 or max < min.
// If rng is nil, yields DefaultEdgeWeight to maintain deterministic fallback.
// Complexity: O(1) time, O(1) space.
func UniformWeightFn(min, max float64) WeightFn {
	if min < 0 || max < min {
		panic(fmt.Sprintf("UniformWeightFn: require 0 ≤ min ≤ max, got min=%g, max=%g", min, max))
	}

	if min < 0 || max < min {
		panic(fmt.Sprintf("UniformWeightFn: require 0 ≤ min ≤ max, got min=%g, max=%g", min, max))
	}
	return func(rng *rand.Rand) float64 {
		if rng == nil {
			return DefaultEdgeWeight
		}

		if max == min {
			// Degenerate interval: constant
			return min
		}
		// Continuous uniform on [min, max) (Float64() returns [0,1))
		span := max - min

		return min + rng.Float64()*span
	}
}

// From1To100WeightFn returns a random weight uniformly in [1,100].
// Equivalent to UniformWeightFn(1,100).
// Complexity: O(1) time, O(1) space.
// Never panics.
func From1To100WeightFn(rng *rand.Rand) float64 {
	return UniformWeightFn(1, 100)(rng)
}

// NormalWeightFn returns a WeightFn sampling from N(mean, stddev),
// rounding to nearest integer and clipping to [0, MaxInt64].
// Panics if stddev < 0.
// If rng is nil, yields DefaultEdgeWeight.
// Complexity: O(1) time, O(1) space.
func NormalWeightFn(mean, stddev float64) WeightFn {
	if stddev < 0 {
		panic(fmt.Sprintf("NormalWeightFn: stddev must be ≥ 0, got %f", stddev))
	}
	maxVal := float64(math.MaxInt64) // !! math.Inf(1)

	return func(rng *rand.Rand) float64 {
		if rng == nil {
			return DefaultEdgeWeight
		}
		sample := rng.NormFloat64()*stddev + mean
		if sample < 0 {
			return 0
		}
		if sample > maxVal {
			return math.MaxInt64
		}

		return math.Round(sample)
	}
}

// ExponentialWeightFn returns a WeightFn sampling from an exponential distribution
// with rate λ, i.e. PDF λ e^(−λx). Panics if rate ≤ 0.
// If rng is nil, yields DefaultEdgeWeight.
// Complexity: O(1) time, O(1) space.
func ExponentialWeightFn(rate float64) WeightFn {
	if rate <= 0 {
		panic(fmt.Sprintf("ExponentialWeightFn: rate must be > 0, got %f", rate))
	}
	return func(rng *rand.Rand) float64 {
		if rng == nil {
			return DefaultEdgeWeight
		}
		// rng.ExpFloat64 gives mean 1/λ when scaled accordingly;
		// dividing by rate yields mean 1/rate.
		return math.Round(rng.ExpFloat64() / rate)
	}
}

// resolveWeightFn returns the first non-nil WeightFn in wfn,
// or DefaultWeightFn if none provided.
// Complexity: O(1) time, O(1) space.
func resolveWeightFn(wfn ...WeightFn) WeightFn {
	if len(wfn) > 0 && wfn[0] != nil {
		return wfn[0]
	}

	return DefaultWeightFn
}

// WithConstantWeight sets a fixed edge weight via ConstantWeightFn.
// Complexity: O(1).
func WithConstantWeight(w float64) BuilderOption {
	return WithWeightFn(ConstantWeightFn(w))
}

// WithUniformWeight sets weights ∼ U[min,max] via UniformWeightFn.
// Complexity: O(1).
func WithUniformWeight(min, max float64) BuilderOption {
	return WithWeightFn(UniformWeightFn(min, max))
}

// WithNormalWeight sets weights ∼ N(mean,stddev) via NormalWeightFn.
// Complexity: O(1).
func WithNormalWeight(mean, stddev float64) BuilderOption {
	return WithWeightFn(NormalWeightFn(mean, stddev))
}

// WithExponentialWeight sets weights ∼ Exp(rate) via ExponentialWeightFn.
// Complexity: O(1).
func WithExponentialWeight(rate float64) BuilderOption {
	return WithWeightFn(ExponentialWeightFn(rate))
}
