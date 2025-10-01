// SPDX-License-Identifier: MIT
// Package: lvlath/builder
//
// impl_pulse.go — deterministic rectangular/triangular pulse generator.
//
// Purpose (single responsibility):
//   • Provide a reproducible 1-D pulse sequence for tests, demos and fixtures.
//   • Shape controls: rectangular (duty ∈ [0,1]) or triangular (0..A envelope).
//   • Optional linear trend and additive Gaussian noise, both deterministic.
//
// Contract:
//   • BuildPulse(n, seed, opts...) returns a slice of length n (or nil on invalid input).
//   • Strict determinism per (n, seed, options); no panics; no global state.
//   • O(n) time and O(n) memory; tiny constant factors.
//
// Options & policy (kept minimal, extension-ready):
//   • Options are resolved via newBuilderConfig(opts...).
//   • extractPulseParams(cfg) centralizes parameter sourcing (defaults for now).
//   • Noise via rng.NormFloat64()*sigma; sigma≥0 is required.
//
// Determinism & testing:
//   • For a fixed seed and defaults, the first K samples are stable (golden-friendly).
//   • Duty=0.5 uses the same branchless comparison as general duty (via frac<duty).
//
// AI-Hints (practical):
//   • To expose user-facing knobs (A/f0/duty/triangular/sigma/trend), add WithPulse(...)
//     that populates builderConfig; then wire it in extractPulseParams.
//   • For DC offsets or piecewise trends, stack them after base and before noise.
//   • For rectangular waveforms, frac := mod(i*f0,1) is faster than trig checks.

package builder

import (
	"math"
)

// -----------------------------------------------------------------------------
// File-local defaults (no magic numbers; cohesive to the pulse generator).
// -----------------------------------------------------------------------------

const (
	defBaseFreq   = 0.125 // Default base frequency f0 in cycles/sample (>0). Period ≈ 8.
	defDuty       = 0.5   // Default rectangular duty cycle in [0,1].
	defTriangular = false // Default shape: false=rectangular, true=triangular.
)

// -----------------------------------------------------------------------------
// Parameter bundle and resolver (keeps impl decoupled from builderConfig).
// -----------------------------------------------------------------------------

// seqPulseParams holds all resolved knobs for the pulse generator.
// Keeping a single struct makes validation and future expansion straightforward.
type seqPulseParams struct {
	amp        float64 // amplitude > 0
	f0         float64 // base frequency > 0 (cycles/sample)
	duty       float64 // rectangular duty in [0,1]
	triangular bool    // rectangular(false) or triangular(true)
	sigma      float64 // Gaussian noise sigma ≥ 0
	trend      float64 // linear trend increment per sample
}

// extractPulseParams maps builderConfig → seqPulseParams.
// Current version uses stable defaults; later you can read cfg fields here.
func extractPulseParams(_ builderConfig) seqPulseParams {
	// Return a fully specified bundle so BuildPulse does not branch on cfg.
	return seqPulseParams{
		amp:        defAmp,        // stable default amplitude
		f0:         defBaseFreq,   // stable default frequency
		duty:       defDuty,       // stable default duty
		triangular: defTriangular, // stable default shape
		sigma:      defSigma,      // stable default noise level
		trend:      defTrendSlope, // stable default trend
	}
}

// -----------------------------------------------------------------------------
// Public API — deterministic pulse generator
// -----------------------------------------------------------------------------

// BuildPulse returns a length-n pulse sequence with optional trend and noise.
// Shape:
//   - Rectangular: y ∈ {0, A} chosen by phase fraction < duty.
//   - Triangular:  y ∈ [0, A] via 1 − |2*frac − 1| (no trig).
//
// Additions:
//   - Linear trend: y += trend * i.
//   - Gaussian noise: y += sigma * N(0,1) (deterministic per seed).
//
// Validation:
//   - If n < 1 ⇒ return nil (invalid request).
//   - If parameters are invalid (A≤0, f0≤0, duty∉[0,1], sigma<0) ⇒ return nil.
//
// Complexity:
//   - O(n) time, O(n) memory, constant-small overhead.
func BuildPulse(n int, seed int64, opts ...BuilderOption) []float64 {
	// Early size check avoids any allocations or RNG setup on invalid input.
	if n < 1 {
		return nil // Contract: invalid input → no data, never panic.
	}

	// Resolve deterministic builder configuration once (O(len(opts))).
	// Even if not used today, this makes the function future-proof for options.
	cfg := newBuilderConfig(opts...) // Immutable config for this call.

	// Resolve pulse parameters from cfg (defaults in this version).
	p := extractPulseParams(cfg) // Keeps BuildPulse free from cfg-specific logic.
	// Defensive parameter validation (fast and explicit; all defaults pass).
	if p.amp <= 0 || p.f0 <= 0 || p.sigma < 0 || p.duty < 0 || p.duty > 1 {
		return nil // Invalid parameterization → no data (no panics by design).
	}

	// RNG selection: prefer cfg.rng to honor global determinism; otherwise fall back to 'seed'.
	rng := rngFrom(cfg, seed)

	// Allocate the output buffer exactly once (tight O(n) memory).
	out := make([]float64, n) // Pre-sized result slice.

	// Precompute common constants (micro-optimizations inside the loop).
	periodFrac := p.f0 // i*f0 gives the phase fraction per sample modulo 1.

	// Predeclare loop temporaries to avoid reallocation in tight loops.
	var (
		frac float64 // phase fraction in [0,1)
		base float64 // base waveform before trend/noise
		tri  float64 // triangular [0,1] envelope
	)

	// Fill all samples in a single pass — O(n) time.
	for i := 0; i < n; i++ {
		// Compute phase fraction in [0,1): frac = (i*f0) mod 1.
		// Using Mod avoids trig overhead and keeps rectangular/triangular unified.
		frac = math.Mod(float64(i)*periodFrac, unitOne)

		// Branch on shape — keep the math simple and branchless within each case.
		if p.triangular {
			// Triangle in [0,1]: 1 − |2*frac − 1|.
			tri = unitOne - math.Abs(triDouble*frac-triCenter) // Normalized triangular envelope.

			base = p.amp * tri // Scale to [0..A].
		} else {
			// Rectangular in {0, A}: on when frac < duty, off otherwise.
			if frac < p.duty {
				base = p.amp
			} else {
				base = unitZero
			}
		}

		// Add predictable linear trend (index-normalized for golden tests).
		base += p.trend * float64(i)

		// Add Gaussian noise only if enabled (sigma>0 keeps default paths clean).
		if p.sigma > 0 {
			base += p.sigma * rng.NormFloat64()
		}

		// Commit the final sample value.
		out[i] = base
	}

	// Return the fully populated, deterministic sequence.
	return out
}
