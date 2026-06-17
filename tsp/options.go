// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package tsp defines solver policy options for Traveling Salesman Problem
// algorithms. Options are explicit contract switches: they select algorithms,
// matching policy, local-search limits, metric-closure behavior, deterministic
// randomness, and exact-solver guards.
package tsp

import "time"

// MatchingFallbackPolicy controls what Christofides does when the requested
// minimum-weight perfect matching engine is unavailable.
//
// Implementation:
//   - Stage 1: The Christofides kernel asks the selected MatchingAlgo for a matching.
//   - Stage 2: If Blossom/MWPM is unavailable, this policy decides whether to fail
//     or explicitly degrade to deterministic greedy matching.
//   - Stage 3: When greedy fallback is used, TSPResult clears the formal ratio and
//     records ErrMatchingFallback as a warning.
//
// Behavior highlights:
//   - Fallback is explicit policy, not a hidden rescue path.
//   - MatchingFallbackReject is the safe default.
//   - MatchingFallbackGreedy keeps the pipeline feasible but removes the 1.5 guarantee.
//
// Inputs:
//   - Used through Options.MatchingFallbackPolicy.
//
// Returns:
//   - No direct return value; consumed by Christofides matching dispatch.
//
// Errors:
//   - Invalid enum values are rejected by validateOptionsStandalone with ErrInvalidOptions.
//
// Determinism:
//   - Policy selection is deterministic and does not depend on runtime state.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - This policy is meaningful when MatchingAlgo==BlossomMatch and exact MWPM is unavailable
//     for the current odd-set size.
//   - Explicit GreedyMatch is not considered a fallback; it is the selected weaker algorithm.
//
// AI-Hints:
//   - Do not silently fallback from BlossomMatch to GreedyMatch without this policy.
//   - Do not keep ChristofidesApproximationRatio when greedy fallback is used.
type MatchingFallbackPolicy int

const (
	// MatchingFallbackReject returns ErrMatchingUnavailable when exact MWPM cannot run.
	MatchingFallbackReject MatchingFallbackPolicy = iota

	// MatchingFallbackGreedy allows deterministic greedy matching as a non-fatal degradation.
	MatchingFallbackGreedy
)

const (
	// DefaultEps is the minimal strictly-better improvement accepted by local search.
	// A candidate move is accepted only when delta < -DefaultEps under default options.
	DefaultEps = 1e-12

	// DefaultTwoOptMaxIters caps accepted 2-opt moves across one local-search run.
	// Zero in Options means unlimited; DefaultOptions uses this finite production guard.
	DefaultTwoOptMaxIters = 10_000

	// DefaultThreeOptMaxMoves caps accepted 3-opt moves across one local-search run.
	// ThreeOpt has a larger neighborhood than 2-opt, so it receives a distinct policy knob.
	DefaultThreeOptMaxMoves = 10_000
)

const (
	// DefaultMaxExactN is the opt-in Auto-policy cap for exact Held-Karp selection.
	DefaultMaxExactN = MaxExactN

	// NoApproximationRatio marks solvers or fallback modes with no proven ratio.
	NoApproximationRatio = 0.0

	// ChristofidesApproximationRatio is the formal ratio only when true MWPM is used.
	ChristofidesApproximationRatio = 1.5
)

// Options defines explicit solver policy for TSP and ATSP algorithms.
// Use DefaultOptions and override fields; the zero value is intentionally not a
// complete production policy because several enum fields need explicit defaults.
//
// Implementation:
//   - Stage 1: DefaultOptions builds a complete safe policy.
//   - Stage 2: validateOptionsStandalone rejects invalid numeric knobs and enum values.
//   - Stage 3: finalizeSolverOptions resolves Auto and matrix-size-dependent defaults.
//
// Behavior highlights:
//   - No hidden exact-to-heuristic fallback unless Algo==Auto.
//   - No hidden Blossom-to-greedy fallback unless MatchingFallbackPolicy allows it.
//   - TimeLimit==0 means unlimited, not disabled.
//   - Seed==0 means a fixed deterministic default RNG stream where shuffling is enabled.
//
// Inputs:
//   - Passed to SolveMatrix, SolveGraph, and direct solver entrypoints.
//
// Returns:
//   - Options is consumed by solvers; it does not own external memory.
//
// Errors:
//   - ErrInvalidOptions for invalid enum values, negative caps, negative TimeLimit, or invalid Eps.
//   - ErrATSPNotSupportedByAlgo for symmetric-only algorithms used with asymmetric policy.
//
// Determinism:
//   - Same options and same matrix order produce the same selected algorithm and route policy.
//   - ShuffleNeighborhood is deterministic under Seed; Seed==0 maps to a fixed internal seed.
//
// Complexity:
//   - Validation and finalization are O(1), except Auto selection depends only on matrix order.
//
// Notes:
//   - MatchingAlgo==GreedyMatch is the safe default until unbounded polynomial MWPM is implemented.
//   - Exact-small MWPM is available through BlossomMatch but does not make BlossomMatch safe
//     as a large-instance default.
//   - ThreeOptMaxMoves is independent from TwoOptMaxIters; do not reuse one cap for both kernels.
//
// AI-Hints:
//   - Do not silently clamp invalid options; reject them with ErrInvalidOptions.
//   - Do not use TwoOptMaxIters as the long-term 3-opt cap.
//   - Do not make Auto the default algorithm without an explicit compatibility decision.
type Options struct {
	// StartVertex selects the start/end vertex index [0..n-1].
	StartVertex int

	// Algo selects the top-level solver path.
	Algo Algorithm

	// Symmetric controls TSP/ATSP matrix validation.
	Symmetric bool

	// MatchingAlgo chooses the odd-vertex matching engine for Christofides.
	MatchingAlgo MatchingAlgo

	// MatchingFallbackPolicy controls Blossom/MWPM unavailable behavior.
	MatchingFallbackPolicy MatchingFallbackPolicy

	// BoundAlgo controls Branch-and-Bound lower-bound strategy.
	BoundAlgo BoundAlgo

	// RunMetricClosure runs Floyd-Warshall before solving direct matrix input.
	RunMetricClosure bool

	// EnableLocalSearch enables local-search post-passes where the dispatcher supports them.
	EnableLocalSearch bool

	// TwoOptMaxIters bounds accepted 2-opt moves. Zero means unlimited.
	TwoOptMaxIters int

	// ThreeOptMaxMoves bounds accepted 3-opt moves. Zero means unlimited.
	ThreeOptMaxMoves int

	// BestImprovement selects best-improvement policy where a local-search kernel supports it.
	BestImprovement bool

	// ShuffleNeighborhood enables deterministic seed-controlled neighborhood shuffling.
	ShuffleNeighborhood bool

	// Eps is the minimal accepted improvement threshold.
	Eps float64

	// TimeLimit optionally bounds wall-clock time. Zero means unlimited.
	TimeLimit time.Duration

	// Seed controls deterministic randomized components.
	Seed int64

	// MaxExactN bounds exact Held-Karp selection when Algo==Auto.
	MaxExactN int
}

// DefaultOptions returns a complete deterministic production policy.
// The default path is currently Christofides with explicit GreedyMatch because
// exact MWPM is bounded to small odd sets until a true Blossom engine is implemented.
//
// Implementation:
//   - Stage 1: Select the symmetric Christofides pipeline.
//   - Stage 2: Select deterministic greedy matching without hidden fallback.
//   - Stage 3: Enable local search with bounded move counts and deterministic RNG policy.
//
// Behavior highlights:
//   - Returns a fully populated Options value.
//   - Does not enable metric closure by default.
//   - Keeps Auto opt-in rather than silently switching algorithms by size.
//
// Inputs:
//   - None.
//
// Returns:
//   - Options: detached value safe for caller mutation.
//
// Errors:
//   - None.
//
// Determinism:
//   - Seed==0 is interpreted by RNG consumers as a fixed deterministic stream.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Switch MatchingAlgo to BlossomMatch only after the MWPM implementation is production-ready.
//   - MatchingFallbackReject remains the default in both current and future modes.
//
// AI-Hints:
//   - Do not reintroduce hidden Blossom->Greedy fallback here.
//   - Do not set Algo=Auto by default in a patch release.
func DefaultOptions() Options {
	return Options{
		StartVertex:            0,
		Algo:                   Christofides,
		Symmetric:              true,
		MatchingAlgo:           GreedyMatch,
		MatchingFallbackPolicy: MatchingFallbackReject,
		BoundAlgo:              NoBound,
		RunMetricClosure:       false,
		EnableLocalSearch:      true,
		TwoOptMaxIters:         DefaultTwoOptMaxIters,
		ThreeOptMaxMoves:       DefaultThreeOptMaxMoves,
		BestImprovement:        false,
		ShuffleNeighborhood:    false,
		Eps:                    DefaultEps,
		TimeLimit:              0,
		Seed:                   0,
		MaxExactN:              DefaultMaxExactN,
	}
}
