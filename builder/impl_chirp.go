// SPDX-License-Identifier: MIT
// Package: lvlath/builder
//
// impl_chirp.go - deterministic linear audio chirp generator.
//
// Purpose:
//   - Produce a 1-D linear chirp (frequency sweep from f0 to f1) for tests/demos.
//   - Optional linear trend and Gaussian noise.
//   - Strict determinism with the same policy as BuildPulse.
//
// Contract:
//   - BuildAudioChirp(n, seed, opts...) returns a slice of length n (or nil).
//   - O(n) time, O(n) memory. No panics. No global state.
//
// Determinism policy (aligned with builders):
//   - If cfg.rng != nil → use cfg.rng (shared stream via WithSeed(...)).
//   - Else → rng := rand.New(rand.NewSource(seed)).
//
// AI-Hints:
//   - Need exponential sweep? Swap linear fi with geometric interpolation.
//   - Want phase-continuous multi-chirp sequences? Reuse the same theta accumulator.

package builder

import (
	"math"
)

// -----------------------------
// Defaults specific to chirp.
// -----------------------------

const (
	defChirpF0 = 0.02 // start frequency (cycles/sample) > 0
	defChirpF1 = 0.25 // end   frequency (cycles/sample) > 0
)

// Precompute 2π to avoid repeated multiplications in the loop (micro-optimization).
const tau = 2.0 * math.Pi // τ = 2π

// -----------------------------
// Param bundle & resolver.
// -----------------------------
type seqChirpParams struct {
	amp   float64 // amplitude > 0
	f0    float64 // start freq > 0
	f1    float64 // end   freq > 0
	sigma float64 // noise sigma ≥ 0
	trend float64 // linear trend increment per sample
}

func extractChirpParams(_ builderConfig) seqChirpParams {
	return seqChirpParams{
		amp:   defAmp, // from impl_pulse.go (package-level const)
		f0:    defChirpF0,
		f1:    defChirpF1,
		sigma: defSigma,      // from impl_pulse.go (package-level const)
		trend: defTrendSlope, // from impl_pulse.go (package-level const)
	}
}

// -----------------------------
// Public API.
// -----------------------------

// BuildAudioChirp returns a length-n linear chirp: f sweeps from f0 to f1.
// Model:
//   - fi  = f0 + (f1 − f0) * i/(n−1)  (cycles/sample)
//   - θᵢ₊₁ = θᵢ + τ * fi               (phase accumulator, τ=2π)
//   - yᵢ  = A * sin(θᵢ) + trend*i + noise
func BuildAudioChirp(n int, seed int64, opts ...BuilderOption) []float64 {
	// Validate size early.
	if n < 1 {
		return nil
	}

	// Resolve builder options.
	cfg := newBuilderConfig(opts...)

	// Resolve chirp parameters.
	p := extractChirpParams(cfg)
	if p.amp <= 0 || p.f0 <= 0 || p.f1 <= 0 || p.sigma < 0 {
		return nil
	}

	// RNG selection (shared vs local).
	rng := rngFrom(cfg, seed)

	// Allocate output buffer.
	out := make([]float64, n)

	// Phase accumulator (start at 0 for reproducibility).
	theta := unitZero

	// Predeclare loop temporaries to avoid reallocation.
	var (
		t   float64 // normalized position in [0,1]
		fi  float64 // instantaneous frequency at sample i
		val float64 // sample value before store
	)

	// Fill deterministically.
	for i := 0; i < n; i++ {
		// Linear interpolation factor t in [0,1].
		if n > 1 {
			t = float64(i) / float64(n-1)
		} else {
			t = unitZero
		}

		// Instantaneous frequency fi (linear sweep).
		fi = p.f0 + (p.f1-p.f0)*t

		// Update phase (discrete-time integration with dt=1).
		theta += tau * fi

		// Base sinusoid.
		val = p.amp * math.Sin(theta)

		// Linear trend (predictable slope).
		val += p.trend * float64(i)

		// Additive Gaussian noise (optional).
		if p.sigma > 0 {
			val += p.sigma * rng.NormFloat64()
		}

		// Store sample.
		out[i] = val
	}

	return out
}
