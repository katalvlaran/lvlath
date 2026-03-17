// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package dfs defines public option, options setters and builders
// for depth-first search workflows over core.Graph.
//
// The file contains configuration and result contracts only.

package dfs

import (
	"context"
	"fmt"
)

// NoDepthLimit disables DFS depth limiting.
// Use it as the MaxDepth value when traversal depth must remain unrestricted.
const NoDepthLimit = -1

// Option configures DFS traversal behavior before execution starts.
//
// Implementation:
//   - Stage 1: Each option mutates DFSOptions during configuration.
//   - Stage 2: Each option validates its own input and may reject invalid values.
//   - Stage 3: DFS applies all options before traversal begins.
//
// Behavior highlights:
//   - Options follow fail-fast validation.
//   - Invalid option input returns ErrOptionViolation.
//
// Inputs:
//   - *DFSOptions: the mutable DFS configuration being assembled.
//
// Returns:
//   - error: nil on success, or ErrOptionViolation-wrapped failure on invalid input.
//
// Errors:
//   - ErrOptionViolation: returned when an option receives invalid explicit input.
//
// Determinism:
//   - Option application is deterministic in call order.
//
// Complexity:
//   - Time O(1) per option, Space O(1).
//
// Notes:
//   - Options configure behavior only; they must not act as runtime state containers.
//
// AI-Hints:
//   - Keep option validation local to the option and finalize cross-field invariants centrally.
//   - Do not let options silently alter core traversal semantics unless documented.
//   - Treat explicitly provided nil callbacks and nil context as invalid input.
type Option func(*DFSOptions) error

// DFSOptions holds configurable DFS traversal parameters.
//
// Implementation:
//   - Stage 1: DefaultOptions provides the baseline configuration.
//   - Stage 2: Option builders selectively override individual fields.
//   - Stage 3: A later finalize/validation stage resolves cross-field invariants.
//
// Behavior highlights:
//   - Options affect traversal policy, not graph structure.
//   - The zero-value is not the canonical configuration; use DefaultOptions.
//
// Inputs:
//   - N/A.
//
// Returns:
//   - N/A.
//
// Errors:
//   - Validation errors are produced by Option builders and finalization code, not by the struct itself.
//
// Determinism:
//   - Determinism is preserved as long as hooks and filters are deterministic.
//
// Complexity:
//   - Accessing fields is O(1); option influence on traversal complexity depends on callback cost.
//
// Notes:
//   - Hooks and filters are user-provided behavior and may dominate runtime if they are expensive.
//
// AI-Hints:
//   - Keep DFSOptions as pure configuration.
//   - Do not store runtime counters or mutable traversal state in this struct.
//   - Callback cost is part of the traversal cost model.
type DFSOptions struct {
	// Ctx controls cancellation and timeout behavior for DFS.
	// A non-nil context is required once explicitly provided through WithContext.
	Ctx context.Context

	// OnVisit runs when a vertex is first entered (pre-order).
	// Returning a non-nil error aborts traversal immediately.
	OnVisit func(id string) error

	// OnExit runs after all reachable descendants of a vertex have been processed
	// and before the vertex is emitted into post-order output.
	// Returning a non-nil error aborts traversal immediately.
	OnExit func(id string) error

	// MaxDepth limits traversal depth.
	// Use NoDepthLimit to keep traversal unrestricted.
	// A depth of 0 means that only the root of a DFS tree may be entered.
	MaxDepth int

	// FilterNeighbor decides whether a candidate neighbor vertex may be traversed.
	// Returning false skips that neighbor.
	FilterNeighbor func(id string) bool

	// FullTraversal enables DFS-forest traversal across all unvisited vertices
	// after the initial root traversal completes.
	FullTraversal bool
}

// DefaultOptions returns the canonical DFS configuration.
//
// Implementation:
//   - Stage 1: Install the default background context.
//   - Stage 2: Disable optional hooks and neighbor filtering.
//   - Stage 3: Disable depth limiting and full-graph traversal by default.
//
// Behavior highlights:
//   - The returned value is ready for further Option-based customization.
//   - The default traversal is single-source and unlimited in depth.
//
// Inputs:
//   - N/A.
//
// Returns:
//   - DFSOptions: the baseline DFS configuration.
//
// Errors:
//   - N/A.
//
// Determinism:
//   - Deterministic by construction.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Callers should prefer DefaultOptions over relying on the struct zero-value.
//
// AI-Hints:
//   - Use DefaultOptions as the single baseline before applying options.
//   - Keep NoDepthLimit as the only supported sentinel for unrestricted depth.
func DefaultOptions() DFSOptions {
	return DFSOptions{
		Ctx:            context.Background(),
		OnVisit:        nil,
		OnExit:         nil,
		MaxDepth:       NoDepthLimit,
		FilterNeighbor: nil,
		FullTraversal:  false,
	}
}

// WithContext sets the traversal context.
//
// Implementation:
//   - Stage 1: Validate the provided context.
//   - Stage 2: Store it in DFSOptions for later traversal checks.
//
// Behavior highlights:
//   - A nil context is invalid explicit input.
//
// Inputs:
//   - ctx: traversal context used for cancellation and timeout.
//
// Returns:
//   - Option: an option that stores the validated context.
//
// Errors:
//   - ErrOptionViolation: if ctx is nil.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Context cancellation semantics remain provided by the standard library.
//
// AI-Hints:
//   - Do not silently accept nil context when it was explicitly provided.
//   - Preserve context.Canceled and context.DeadlineExceeded as underlying causes.
func WithContext(ctx context.Context) Option {
	return func(o *DFSOptions) error {
		if ctx == nil {
			return ErrOptionViolation
		}

		o.Ctx = ctx
		return nil
	}
}

// WithOnVisit installs a pre-order hook.
//
// Implementation:
//   - Stage 1: Validate the callback.
//   - Stage 2: Store it for traversal-time execution.
//
// Behavior highlights:
//   - The hook runs immediately when a vertex is entered.
//
// Inputs:
//   - fn: pre-order callback.
//
// Returns:
//   - Option: an option that stores the callback.
//
// Errors:
//   - ErrOptionViolation: if fn is nil.
//
// Determinism:
//   - Deterministic if fn itself is deterministic.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Hook runtime cost becomes part of DFS runtime cost.
//
// AI-Hints:
//   - Treat an explicit nil callback as invalid input, not as a silent no-op.
//   - Keep hook behavior side-effect aware and deterministic when order matters.
func WithOnVisit(fn func(id string) error) Option {
	return func(o *DFSOptions) error {
		if fn == nil {
			return ErrOptionViolation
		}

		o.OnVisit = fn
		return nil
	}
}

// WithOnExit installs a post-order hook.
//
// Implementation:
//   - Stage 1: Validate the callback.
//   - Stage 2: Store it for traversal-time execution.
//
// Behavior highlights:
//   - The hook runs after a vertex has finished exploring all reachable descendants.
//   - The hook executes before the vertex is appended to DFS post-order output.
//
// Inputs:
//   - fn: post-order callback.
//
// Returns:
//   - Option: an option that stores the callback.
//
// Errors:
//   - ErrOptionViolation: if fn is nil.
//
// Determinism:
//   - Deterministic if fn itself is deterministic.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Hook runtime cost becomes part of DFS runtime cost.
//
// AI-Hints:
//   - Treat an explicit nil callback as invalid input, not as a silent no-op.
//   - Keep hook behavior side-effect aware and deterministic when order matters.
func WithOnExit(fn func(id string) error) Option {
	return func(o *DFSOptions) error {
		if fn == nil {
			return ErrOptionViolation
		}

		o.OnExit = fn
		return nil
	}
}

// WithMaxDepth sets the maximum DFS depth.
//
// Implementation:
//   - Stage 1: Validate the requested limit.
//   - Stage 2: Store it in DFSOptions.
//
// Behavior highlights:
//   - NoDepthLimit disables depth limiting.
//   - A depth of 0 allows entering only the root of each DFS tree.
//
// Inputs:
//   - limit: maximum traversal depth, or NoDepthLimit.
//
// Returns:
//   - Option: an option that stores the validated depth limit.
//
// Errors:
//   - ErrOptionViolation: if limit is less than NoDepthLimit.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Negative values other than NoDepthLimit are invalid.
//
// AI-Hints:
//   - Do not invent additional negative sentinels.
//   - Keep NoDepthLimit as the only supported unrestricted-depth value.
func WithMaxDepth(limit int) Option {
	return func(o *DFSOptions) error {
		if limit < NoDepthLimit {
			return ErrOptionViolation
		}

		o.MaxDepth = limit
		return nil
	}
}

// WithFilterNeighbor installs a neighbor filter.
//
// Implementation:
//   - Stage 1: Validate the callback.
//   - Stage 2: Store it for candidate-neighbor checks during traversal.
//
// Behavior highlights:
//   - Returning false prevents traversal into the candidate neighbor.
//
// Inputs:
//   - fn: neighbor filter callback.
//
// Returns:
//   - Option: an option that stores the callback.
//
// Errors:
//   - ErrOptionViolation: if fn is nil.
//
// Determinism:
//   - Deterministic if fn itself is deterministic.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - The filter currently receives only the candidate neighbor ID.
//
// AI-Hints:
//   - Filtering affects traversal reachability and diagnostics, not graph topology.
//   - Avoid expensive filter logic when O(V+E) behavior matters.
func WithFilterNeighbor(fn func(id string) bool) Option {
	return func(o *DFSOptions) error {
		if fn == nil {
			return ErrOptionViolation
		}

		o.FilterNeighbor = fn
		return nil
	}
}

// WithFullTraversal enables DFS-forest traversal.
//
// Implementation:
//   - Stage 1: Enable traversal restart from later unvisited roots.
//   - Stage 2: Allow coverage of disconnected components.
//
// Behavior highlights:
//   - The traversal becomes a DFS forest instead of a single DFS tree.
//
// Inputs:
//   - N/A.
//
// Returns:
//   - Option: an option that enables full traversal.
//
// Errors:
//   - N/A.
//
// Determinism:
//   - Root visitation order remains governed by the graph vertex order.
//
// Complexity:
//   - Time O(1), Space O(1) for configuration itself.
//
// Notes:
//   - Depth values reset at each DFS-tree root in forest mode.
//
// AI-Hints:
//   - In full traversal mode, Depth is measured from each tree root, not from a single global origin.
func WithFullTraversal() Option {
	return func(o *DFSOptions) error {
		o.FullTraversal = true
		return nil
	}
}

// buildDFSOptions assembles the canonical DFS configuration before traversal starts.
//
// Implementation:
//   - Stage 1: Start from DefaultOptions as the single baseline configuration.
//   - Stage 2: Apply options in the exact order provided by the caller.
//   - Stage 3: Abort immediately on the first invalid or failing option.
//
// Behavior highlights:
//   - Configuration is fail-fast.
//   - A nil Option value is invalid explicit input.
//   - The returned DFSOptions value contains traversal policy only.
//
// Inputs:
//   - opts: ordered DFS option builders.
//
// Returns:
//   - DFSOptions: the assembled DFS configuration.
//   - error: nil on success, or an option-construction failure.
//
// Errors:
//   - ErrOptionViolation: returned when an option is nil or invalid.
//   - Any custom Option error is returned unchanged so that callers may preserve its category.
//
// Determinism:
//   - Deterministic in caller-provided option order.
//
// Complexity:
//   - Time O(k), Space O(1), where k = len(opts).
//
// Notes:
//   - This helper centralizes option application so DFS-based algorithms do not duplicate
//     configuration logic.
//
// AI-Hints:
//   - Use this helper as the single entry point for DFS option assembly.
//   - Do not duplicate option application logic across algorithms.
//   - Treat invalid explicit option input as configuration failure, not runtime traversal failure.
func buildDFSOptions(opts ...Option) (DFSOptions, error) {
	// Start from the canonical baseline configuration.
	options := DefaultOptions()

	// Apply options in caller order so last-writer-wins semantics remain explicit.
	for index, opt := range opts {
		// Reject a nil Option value explicitly to avoid a panic on call.
		if opt == nil {
			return DFSOptions{}, fmt.Errorf("%w: option at index %d is nil", ErrOptionViolation, index)
		}

		// Apply the option and stop immediately if it rejects the input.
		if err := opt(&options); err != nil {
			return DFSOptions{}, err
		}
	}

	return options, nil
}
