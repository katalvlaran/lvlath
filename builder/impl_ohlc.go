// SPDX-License-Identifier: MIT
// Package: lvlath/builder
//
// impl_ohlc.go - deterministic OHLC series via discrete-time GBM with intraday steps.
//
// Purpose:
//   - Emit reproducible OHLC arrays for 'days' trading days using a GBM-like path.
//   - Use a small fixed number of intraday steps to form realistic wicks (high/low).
//   - Strict determinism: prefer cfg.rng if present; otherwise fall back to 'seed'.
//
// Contract:
//   - BuildOHLCSeries(days, seed, opts...) → (open[], high[], low[], close[]).
//   - On invalid input (days<1 or bad params) ⇒ return nil slices; never panic.
//   - O(days * steps) time; O(days) memory; steps is a tiny constant.
//
// Determinism policy (aligned with other builders):
//   - If cfg.rng != nil → use cfg.rng (shared stream via WithSeed(...)).
//   - Else → rng := rand.New(rand.NewSource(seed)).
//
// Invariants (by construction after each day):
//   - low ≤ min(open, close) ≤ max(open, close) ≤ high.
//
// AI-Hints:
//   - To make candles even closer to real markets: (a) increase 'steps' to get
//     thinner wicks; (b) add mild vol clustering via an internal multiplier;
//     (c) post-round prices to a tick (e.g., 0.01) outside this function.
//   - If you later expose options (S0, μ, σ, steps, precision), wire them
//     through builderConfig in extractOHLCParams.

package builder

import (
	"math"
)

// -----------------------------
// Defaults specific to OHLC.
// -----------------------------
const (
	defOHLCStart     = 100.0  // Default initial price S0 (>0)
	defOHLCDailyMu   = 0.0005 // Default daily drift μ
	defOHLCDailyVol  = 0.02   // Default daily volatility σ (≥0)
	defIntradaySteps = 8      // Fixed intraday steps per day (small constant)
)

// -----------------------------
// Parameter bundle & resolver.
// -----------------------------

// seqOHLCParams groups resolved knobs for the OHLC generator.
type seqOHLCParams struct {
	S0    float64 // initial price > 0
	mu    float64 // daily drift
	vol   float64 // daily volatility ≥ 0
	steps int     // intraday steps per day ≥ 1
}

// extractOHLCParams maps builderConfig → seqOHLCParams (defaults for now).
func extractOHLCParams(_ builderConfig) seqOHLCParams {
	// Return a fully specified, validated set (defaults are safe).
	return seqOHLCParams{
		S0:    defOHLCStart,
		mu:    defOHLCDailyMu,
		vol:   defOHLCDailyVol,
		steps: defIntradaySteps,
	}
}

// -----------------------------
// Public API.
// -----------------------------

// BuildOHLCSeries returns deterministic OHLC arrays for 'days' trading days.
// Model (discrete GBM per intraday step with Δt = 1/steps):
//
//	S_{t+1} = S_t * exp((μ - 0.5σ²)Δt + σ√Δt * Z),  Z ~ N(0,1).
func BuildOHLCSeries(days int, seed int64, opts ...BuilderOption) (open, high, low, close []float64) {
	// Validate the requested number of days; if invalid, return nil slices.
	if days < 1 {
		return nil, nil, nil, nil
	}

	// Resolve builder options once (ready for future wiring like WithOHLC(...)).
	cfg := newBuilderConfig(opts...)

	// Resolve OHLC parameters (defaults for now).
	p := extractOHLCParams(cfg)

	// Defensive parameter checks (explicit and fast).
	if p.S0 <= 0 || p.vol < 0 || p.steps < 1 {
		return nil, nil, nil, nil
	}

	// RNG selection: prefer shared cfg.rng for global determinism; else local fallback.
	rng := rngFrom(cfg, seed)

	// Pre-allocate outputs exactly once: O(days) memory.
	open = make([]float64, days)  // open[d] at day start
	high = make([]float64, days)  // high[d] max on intraday path
	low = make([]float64, days)   // low[d]  min on intraday path
	close = make([]float64, days) // close[d] after last intraday step

	// Initialize the starting price (strictly positive).
	S := p.S0

	// Precompute intraday constants (avoid recomputing inside loops).
	// Δt per intraday step; using daily μ, σ split across 'steps'.
	dt := 1.0 / float64(p.steps)        // time step
	driftTerm := p.mu - 0.5*p.vol*p.vol // (μ - 0.5 σ²), reused
	noiseScale := p.vol * math.Sqrt(dt) // σ √Δt, reused

	// Declare loop-temporaries once (avoid reallocation in tight loops).
	var (
		d               int     // day index
		s               int     // intraday step index
		dayHigh, dayLow float64 // running extrema for the day
		Z               float64 // standard normal draw
		incr            float64 // log-return increment for the step
		openD, closeD   float64 // aliases for readability
	)

	// Simulate day by day (outer loop) and steps within a day (inner loop).
	for d = 0; d < days; d++ {
		// Record open at the very start of the day (before any intraday step).
		openD = S
		open[d] = openD

		// Initialize daily extrema with the opening price so open is always considered.
		dayHigh = openD
		dayLow = openD

		// Run the fixed number of intraday steps to produce a realistic wick.
		for s = 0; s < p.steps; s++ {
			// Draw a standard normal (deterministic per rng stream).
			Z = rng.NormFloat64()

			// Compute the log-increment for this step: (μ - 0.5σ²)Δt + σ√Δt * Z.
			incr = driftTerm*dt + noiseScale*Z

			// Update the price multiplicatively (GBM).
			S = S * math.Exp(incr)

			// Update daily extrema after the step (the wick body).
			if S > dayHigh {
				dayHigh = S
			}
			if S < dayLow {
				dayLow = S
			}
		}

		// The close is the last price after the final intraday step.
		closeD = S
		close[d] = closeD

		// Finalize high/low to include endpoints explicitly (open & close).
		// (Defensive, although open was used to init and close already visited.)
		if openD > dayHigh {
			dayHigh = openD
		}
		if closeD > dayHigh {
			dayHigh = closeD
		}
		if openD < dayLow {
			dayLow = openD
		}
		if closeD < dayLow {
			dayLow = closeD
		}

		// Commit extrema for the day.
		high[d] = dayHigh
		low[d] = dayLow
	}

	// Return the four deterministic series.
	return open, high, low, close
}
