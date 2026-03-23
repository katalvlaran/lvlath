// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

package dijkstra

import "math"

// DijkstraOptions defines the explicit runtime policy for a single Dijkstra execution.
// The structure contains only contract-changing options that affect path tracking,
// distance cutoffs, and edge-wall interpretation.
//
// Implementation:
//   - Stage 1: Start from DefaultOptions to obtain the canonical baseline policy.
//   - Stage 2: Apply functional options in call order with last-writer-wins semantics.
//   - Stage 3: Pass the finalized policy into the traversal kernel without hidden mutation.
//
// Behavior highlights:
//   - Path tracking is disabled by default to avoid unnecessary predecessor allocation.
//   - MaxDistance defaults to positive infinity, which means "no distance cutoff".
//   - InfEdgeThreshold defaults to positive infinity, which means "no finite edge is treated as a wall".
//
// Inputs:
//   - TrackPaths: enables predecessor tracking for later path reconstruction.
//   - MaxDistance: limits exploration to shortest paths whose distance does not exceed this bound.
//   - InfEdgeThreshold: treats edges with weight greater than or equal to this threshold as impassable.
//
// Returns:
//   - DijkstraOptions: a detached value object consumed by the API and kernel.
//
// Errors:
//   - Option constructors report configuration errors through their returned Option functions.
//
// Determinism:
//   - The structure itself is deterministic value configuration and does not introduce hidden ordering.
//
// Complexity:
//   - Copy and return cost is O(1); the structure contains only scalar fields.
//
// Notes:
//   - sourceID is intentionally not stored here and must be passed explicitly to the public API.
//   - This type contains policy only; it does not own traversal state or graph data.
//
// AI-Hints:
//   - Do not move sourceID back into options; that weakens the input contract.
//   - Do not reintroduce MemoryMode or other dormant switches that do not alter real semantics.
type DijkstraOptions struct {
	TrackPaths       bool
	MaxDistance      float64
	InfEdgeThreshold float64
}

// Option applies a single configuration mutation to DijkstraOptions and may reject
// invalid input with a sentinel error.
//
// Implementation:
//   - Stage 1: Receive the current mutable options state.
//   - Stage 2: Validate the requested option payload.
//   - Stage 3: Apply the mutation or return an error.
//
// Behavior highlights:
//   - Options are applied in call order.
//   - Last writer wins when multiple options target the same field.
//
// Inputs:
//   - *DijkstraOptions: the mutable configuration under construction.
//
// Returns:
//   - error: nil on success, or a sentinel configuration error.
//
// Errors:
//   - ErrNilOption is reported by applyOptions when the option itself is nil.
//   - ErrBadMaxDistance and ErrBadInfEdgeThreshold may be returned by specific options.
//
// Determinism:
//   - Option application order is exactly the call order provided by the caller.
//
// Complexity:
//   - Each option is O(1) and must not allocate large auxiliary state.
//
// Notes:
//   - Panic-based option validation is forbidden.
//
// AI-Hints:
//   - Functional options are part of the public contract; return errors instead of panicking.
//   - Keep options side-effect free beyond mutating the provided config value.
type Option func(*DijkstraOptions) error

// DefaultOptions returns the canonical baseline configuration for a single Dijkstra run.
//
// Implementation:
//   - Stage 1: Construct the zero-allocation baseline policy.
//   - Stage 2: Set explicit infinity-based defaults for optional cutoffs.
//
// Behavior highlights:
//   - Path tracking is disabled by default.
//   - MaxDistance defaults to +Inf, which preserves full reachable exploration.
//   - InfEdgeThreshold defaults to +Inf, which preserves all finite edges.
//
// Inputs:
//   - None.
//
// Returns:
//   - DijkstraOptions: the canonical default configuration.
//
// Errors:
//   - None.
//
// Determinism:
//   - Always returns the same configuration value.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - This function does not validate graph inputs and does not depend on sourceID.
//
// AI-Hints:
//   - Keep +Inf as the explicit "no limit" policy; do not replace it with arbitrary large finite numbers.
func DefaultOptions() DijkstraOptions {
	return DijkstraOptions{
		TrackPaths:       false,
		MaxDistance:      math.Inf(1),
		InfEdgeThreshold: math.Inf(1),
	}
}

// WithPathTracking enables predecessor tracking so that the result can later
// reconstruct a concrete shortest path witness.
//
// Implementation:
//   - Stage 1: Mark TrackPaths as enabled.
//
// Behavior highlights:
//   - This option does not allocate predecessor storage by itself.
//   - The kernel decides whether to allocate Prev based on the finalized config.
//
// Inputs:
//   - None.
//
// Returns:
//   - Option: a functional option that enables path tracking.
//
// Errors:
//   - None.
//
// Determinism:
//   - Always enables the same field in the same way.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Use this only when the caller actually needs path reconstruction.
//
// AI-Hints:
//   - Tracking predecessors is a contract-level request and should remain explicit.
//   - Do not infer it implicitly from wrapper internals except in dedicated wrapper APIs.
func WithPathTracking() Option {
	return func(opts *DijkstraOptions) error {
		opts.TrackPaths = true
		return nil
	}
}

// WithMaxDistance limits exploration to shortest paths whose distance does not
// exceed the provided bound.
//
// Implementation:
//   - Stage 1: Validate the numeric input.
//   - Stage 2: Store the accepted bound in the config.
//
// Behavior highlights:
//   - Positive infinity means "no distance cutoff".
//   - Distances beyond the cutoff remain +Inf in the final result.
//
// Inputs:
//   - max: the maximum allowed shortest-path distance.
//
// Returns:
//   - Option: a functional option that updates MaxDistance.
//
// Errors:
//   - ErrBadMaxDistance if max is NaN, negative, or negative infinity.
//
// Determinism:
//   - The accepted value is applied exactly as provided.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Zero is valid and keeps only the source at finite distance unless zero-cost edges extend reachability.
//
// AI-Hints:
//   - Do not reject +Inf; it is the canonical "no cutoff" value.
//   - Do not silently clamp bad values; reject them explicitly.
func WithMaxDistance(max float64) Option {
	return func(opts *DijkstraOptions) error {
		switch {
		case math.IsNaN(max):
			return ErrBadMaxDistance
		case math.IsInf(max, -1):
			return ErrBadMaxDistance
		case max < 0:
			return ErrBadMaxDistance
		default:
			opts.MaxDistance = max
			return nil
		}
	}
}

// WithInfEdgeThreshold defines the edge weight at or above which edges are treated
// as impassable walls by the traversal kernel.
//
// Implementation:
//   - Stage 1: Validate the numeric threshold.
//   - Stage 2: Store the accepted threshold in the config.
//
// Behavior highlights:
//   - Positive infinity means "no finite edge is blocked by threshold".
//   - Finite edges with weight >= threshold are skipped during relaxation.
//
// Inputs:
//   - threshold: the minimum weight treated as a wall.
//
// Returns:
//   - Option: a functional option that updates InfEdgeThreshold.
//
// Errors:
//   - ErrBadInfEdgeThreshold if threshold is NaN, non-positive, or negative infinity.
//
// Determinism:
//   - The accepted threshold is applied exactly as provided.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - A zero threshold would incorrectly classify zero-weight edges as impassable and is therefore invalid.
//
// AI-Hints:
//   - Keep this policy explicit; do not hide wall semantics behind undocumented heuristics.
//   - Do not collapse this option into MaxDistance; they govern different stages of the algorithm.
func WithInfEdgeThreshold(threshold float64) Option {
	return func(opts *DijkstraOptions) error {
		switch {
		case math.IsNaN(threshold):
			return ErrBadInfEdgeThreshold
		case math.IsInf(threshold, -1):
			return ErrBadInfEdgeThreshold
		case threshold <= 0:
			return ErrBadInfEdgeThreshold
		default:
			opts.InfEdgeThreshold = threshold
			return nil
		}
	}
}

// applyOptions builds the finalized Dijkstra configuration from the canonical defaults
// and the provided functional options.
//
// Implementation:
//   - Stage 1: Start from DefaultOptions.
//   - Stage 2: Apply each option in call order.
//   - Stage 3: Stop on the first configuration failure.
//
// Behavior highlights:
//   - Nil options are rejected explicitly.
//   - Last writer wins when multiple options target the same field.
//
// Inputs:
//   - opts: zero or more functional options.
//
// Returns:
//   - DijkstraOptions: the finalized runtime policy.
//   - error: a sentinel configuration error when option assembly fails.
//
// Errors:
//   - ErrNilOption if a nil option is encountered.
//   - Any error returned by an individual option.
//
// Determinism:
//   - Option application order is stable and equals the caller-provided order.
//
// Complexity:
//   - Time O(n), Space O(1), where n is the number of options.
//
// Notes:
//   - This function performs configuration assembly only; it does not inspect the graph.
//
// AI-Hints:
//   - Keep all option finalization centralized here instead of scattering it across API entry points.
//   - Do not add hidden default-repair logic that silently changes caller intent.
func applyOptions(opts ...Option) (DijkstraOptions, error) {
	config := DefaultOptions()

	for _, opt := range opts {
		if opt == nil {
			return DijkstraOptions{}, ErrNilOption
		}
		if err := opt(&config); err != nil {
			return DijkstraOptions{}, err
		}
	}

	return config, nil
}

//////////////////////////////////////////////////////////////////////
//
//  Everything below is strictly for testing purposes. !! TEST ONLY !!
//
//////////////////////////////////////////////////////////////////////

// DefaultOptionsSnapshot_TestOnly returns the canonical default option snapshot
// for external contract tests.
//
// Implementation:
//   - Stage 1: Delegate to DefaultOptions.
//   - Stage 2: Return the detached snapshot unchanged.
//
// Behavior highlights:
//   - This helper exposes the documented default configuration to external tests.
//   - The helper does not mutate package state.
//
// Inputs:
//   - None.
//
// Returns:
//   - DijkstraOptions: the canonical default option snapshot.
//
// Errors:
//   - None.
//
// Determinism:
//   - Always returns the same value for the same package version.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - This helper exists only to support external test-package contract verification.
//
// AI-Hints:
//   - Keep this helper as a thin bridge to the canonical default constructor.
//   - Do not add hidden normalization or extra policy changes here.
func DefaultOptionsSnapshot_TestOnly() DijkstraOptions {
	return DefaultOptions()
}

// GatherOptionsSnapshot_TestOnly applies functional options through the canonical
// internal assembly path and returns the finalized option snapshot for external tests.
//
// Implementation:
//   - Stage 1: Delegate directly to applyOptions.
//   - Stage 2: Return the finalized snapshot or the assembly error unchanged.
//
// Behavior highlights:
//   - This helper exposes the real option assembly semantics to external tests.
//   - Last-writer-wins and nil-option behavior remain exactly those of applyOptions.
//
// Inputs:
//   - opts: zero or more functional options.
//
// Returns:
//   - DijkstraOptions: the finalized option snapshot.
//   - error: any canonical option-assembly error.
//
// Errors:
//   - ErrNilOption if a nil option is encountered.
//   - Any error returned by an individual option.
//
// Determinism:
//   - Deterministic for the same option sequence.
//
// Complexity:
//   - Time O(n), Space O(1), where n is the number of options.
//
// Notes:
//   - This helper exists only to support external test-package contract verification.
//
// AI-Hints:
//   - Keep this helper a direct bridge to applyOptions.
//   - Do not fork option-assembly logic for tests.
func GatherOptionsSnapshot_TestOnly(opts ...Option) (DijkstraOptions, error) {
	return applyOptions(opts...)
}
