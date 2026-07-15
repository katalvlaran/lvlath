// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

package dijkstra

import "math"

// Options defines the explicit runtime policy for a single Dijkstra execution.
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
//   - Options: a detached value object consumed by the API and kernel.
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
type Options struct {
	TrackPaths       bool
	MaxDistance      float64
	InfEdgeThreshold float64
}

// Option applies a single configuration mutation to Options and may reject
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
//   - *Options: the mutable configuration under construction.
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
//   - Positive infinity in MaxDistance or InfEdgeThreshold belongs exclusively
//     to the runtime-policy domain and does not permit non-finite edge weights.
//
// AI-Hints:
//   - Functional options are part of the public contract; return errors instead of panicking.
//   - Keep options side-effect free beyond mutating the provided config value.
//   - Caller-defined Option values are revalidated by canonical assembly.
//   - Do not add exported TestOnly bridges to observe applyOptions.
type Option func(*Options) error

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
//   - Options: the canonical default configuration.
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
func DefaultOptions() Options {
	return Options{
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
	return func(opts *Options) error {
		opts.TrackPaths = true
		return nil
	}
}

// WithMaxDistance sets an inclusive upper bound for published finite
// shortest-path distances.
// The option limits traversal by total accumulated path cost and does not alter
// the graph or classify individual edges as invalid.
//
// Implementation:
//   - Stage 1: Reject NaN, negative infinity, and finite negative values.
//   - Stage 2: Accept finite non-negative values or positive infinity.
//   - Stage 3: Store the accepted value as the execution MaxDistance policy.
//
// Behavior highlights:
//   - The bound is inclusive: a candidate distance equal to max remains admissible.
//   - A candidate distance greater than max is not relaxed.
//   - When the minimum frontier item exceeds max, traversal terminates.
//   - Positive infinity means “no distance cutoff”.
//   - Vertices excluded by the cutoff remain known in Result.Distances with +Inf.
//
// Inputs:
//   - max: the inclusive maximum accumulated path distance.
//     Valid domain: finite max >= 0 or +Inf.
//
// Returns:
//   - Option: a deterministic functional option that updates MaxDistance.
//
// Errors:
//   - ErrBadMaxDistance if max is NaN, -Inf, or a finite negative value.
//
// Determinism:
//   - The accepted value is stored exactly.
//   - Repeated MaxDistance options follow call order and last-writer-wins semantics.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Zero is valid. It allows the source and any vertices reachable exclusively
//     through zero-weight paths to retain finite distance.
//   - Positive infinity belongs to the option domain and does not permit +Inf
//     graph-edge weights.
//
// AI-Hints:
//   - Do not replace +Inf with an arbitrary large finite constant.
//   - Do not silently clamp invalid input.
//   - Do not confuse MaxDistance with InfEdgeThreshold: MaxDistance governs total
//     path cost, while InfEdgeThreshold governs each individual edge.
func WithMaxDistance(max float64) Option {
	return func(options *Options) error {
		if err := validateMaxDistance(max); err != nil {
			return err
		}

		options.MaxDistance = max

		return nil
	}
}

// WithInfEdgeThreshold sets an inclusive edge-local wall threshold.
// Every valid finite edge whose weight is greater than or equal to threshold
// is skipped during relaxation without mutating graph topology.
//
// Implementation:
//   - Stage 1: Reject NaN, negative infinity, zero, and finite negative values.
//   - Stage 2: Accept a finite strictly positive threshold or positive infinity.
//   - Stage 3: Store the accepted value as the execution wall policy.
//
// Behavior highlights:
//   - The comparison is inclusive: weight == threshold is blocked.
//   - Positive infinity means that no valid finite edge is blocked.
//   - The option changes edge admissibility, not edge validity.
//   - Blocking every route to a known vertex leaves its published distance at +Inf.
//
// Inputs:
//   - threshold: the minimum finite edge weight treated as impassable.
//     Valid domain: finite threshold > 0 or +Inf.
//
// Returns:
//   - Option: a deterministic functional option that updates InfEdgeThreshold.
//
// Errors:
//   - ErrBadInfEdgeThreshold if threshold is NaN, -Inf, zero,
//     or a finite negative value.
//
// Determinism:
//   - The accepted value is stored exactly.
//   - Repeated threshold options follow call order and last-writer-wins semantics.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Positive infinity is a policy-domain sentinel meaning “disable finite-edge
//     wall filtering”; it is not a valid core.Edge.Weight.
//   - Threshold policy is independent of MaxDistance.
//   - Zero is invalid because valid zero-weight edges must remain traversable.
//
// AI-Hints:
//   - Do not reinterpret this option as permission to store +Inf edge weights.
//   - Keep the wall comparison as weight >= threshold.
//   - Do not merge this option with MaxDistance; one is edge-local and the other
//     governs accumulated path cost.
func WithInfEdgeThreshold(threshold float64) Option {
	return func(options *Options) error {
		if err := validateInfEdgeThreshold(threshold); err != nil {
			return err
		}

		options.InfEdgeThreshold = threshold

		return nil
	}
}

// applyOptions builds the finalized Dijkstra configuration from the canonical
// defaults and the provided functional options.
// The assembler validates both option-returned errors and the complete state
// produced by every option.
//
// Implementation:
//   - Stage 1: Start from DefaultOptions.
//   - Stage 2: Reject nil option functions.
//   - Stage 3: Apply each option in caller-provided order.
//   - Stage 4: Revalidate the complete config after every option.
//   - Stage 5: Publish the finalized detached value.
//
// Behavior highlights:
//   - Nil options are rejected explicitly.
//   - Last writer wins when repeated valid options target the same field.
//   - Caller-defined options cannot bypass numeric invariants.
//   - Assembly stops at the first option or finalized-state failure.
//
// Inputs:
//   - opts: zero or more functional options.
//
// Returns:
//   - Options: the finalized runtime policy.
//   - error: nil on success or the first option/configuration failure.
//
// Errors:
//   - ErrNilOption if a nil option is encountered.
//   - ErrBadMaxDistance if an option leaves MaxDistance invalid.
//   - ErrBadInfEdgeThreshold if an option leaves InfEdgeThreshold invalid.
//   - Any error returned directly by an option.
//
// Determinism:
//   - Option application and validation follow exact caller-provided order.
//
// Complexity:
//   - Time O(n), Space O(1), where n is the number of options.
//
// Notes:
//   - This function performs configuration assembly only.
//   - It does not inspect graph topology or allocate traversal state.
//
// AI-Hints:
//   - Keep this helper unexported.
//   - Do not create production TestOnly bridges around it.
//   - Do not remove finalized-state validation merely because built-in options
//     already validate their own payloads.
func applyOptions(opts ...Option) (Options, error) {
	config := DefaultOptions()

	for _, option := range opts {
		if option == nil {
			return Options{}, ErrNilOption
		}

		if err := option(&config); err != nil {
			return Options{}, err
		}
		if err := validateOptions(config); err != nil {
			return Options{}, err
		}
	}

	return config, nil
}
