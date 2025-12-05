// SPDX-License-Identifier: MIT
// Package: lvlath/builder
//
// impl_sequences_shared.go - shared defaults and helpers for sequence builders.
//
// Purpose:
//   - Hold cross-sequence defaults (amplitude/noise/trend).
//   - Provide deterministic RNG selection with cfg.rng priority.
//   - Provide small named numeric constants to avoid magic literals.
//
// Contract:
//   - Pure helpers (no global state). Safe to import from impl_pulse.go / impl_chirp.go / impl_ohlc.go.

package builder

import (
	"math/rand"
)

// -----------------------------
// Shared defaults (cross-file).
// -----------------------------
const (
	defAmp        = 1.0 // Default amplitude for Pulse/Chirp A (>0).
	defSigma      = 0.0 // Default Gaussian noise sigma (≥0); 0 disables noise.
	defTrendSlope = 0.0 // Default linear trend increment per sample.
)

// -----------------------------
// Tiny numeric named constants.
// -----------------------------
const (
	unitZero  = 0.0 // named zero to avoid magic 0.0
	unitOne   = 1.0 // named one to avoid magic 1.0
	triDouble = 2.0 // factor used in triangular wave: 2*frac-1
	triCenter = 1.0 // center offset used in triangular wave
)

// centerVertexID is a fixed, documented hub ID used by Star/Wheel/Platonic(withCenter).
const centerVertexID = "Center"

// chord represents an undirected shell edge between two vertex indices U<V.
type chord struct {
	U int // first endpoint index (0-based)
	V int // second endpoint index (0-based), strictly greater than U
}

// edgePair is an order-sensitive undirected edge between two canonical IDs.
type edgePair struct {
	U string // endpoint 1
	V string // endpoint 2
}

// rngFrom returns cfg.rng if present (shared stream), else a local rand
// seeded by 'seed'. This keeps determinism across composed calls.
func rngFrom(cfg builderConfig, seed int64) *rand.Rand {
	if cfg.rng != nil {
		return cfg.rng
	}

	return rand.New(rand.NewSource(seed))
}
