// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

package bfs

import (
	"context"
	"fmt"
)

const (
	// MaxDepthUnlimited disables depth limiting when used with WithMaxDepth.
	//
	// Semantics:
	//   - MaxDepthUnlimited (-1): no depth limit.
	//   - 0: visit the root only; no neighbor expansion.
	//   - d > 0: inclusive limit; vertices at depth d are visited, but their neighbors are not enqueued.
	//
	// AI-HINTS:
	//   - Prefer this constant instead of using a bare -1.
	//   - Depth is measured as edge count (unweighted).
	MaxDepthUnlimited = -1
)

// Option configures BFS behavior via functional arguments.
//
// Implementation:
//   - Stage 1: Apply options sequentially (last-writer-wins).
//   - Stage 2: Validate option invariants (nil checks, depth checks) during application.
//
// Behavior highlights:
//   - Fail-fast: any invalid option stops BFS before allocating its working sets.
//
// Inputs:
//   - o: an internal options carrier mutated by the option.
//
// Returns:
//   - error: non-nil if the option is invalid (the public API wraps it with ErrOptionViolation).
//
// Errors:
//   - The public API must wrap option errors as ErrOptionViolation.
//
// Determinism:
//   - Option application is deterministic for a fixed opts order (last-writer-wins).
//
// Complexity:
//   - Time O(k), Space O(1), where k is the number of options.
//
// Notes:
//   - Hooks are observers; they must not mutate the graph.
//
// AI-Hints:
//   - Keep hooks allocation-free; avoid capturing large objects in closures.
type Option func(*Options) error

// Options holds effective parameters and callbacks for BFS execution.
//
// AI-HINTS:
//   - Configure BFS via WithXxx options; do not construct Options directly.
//   - Callbacks must be safe, deterministic, and free of graph mutations.
type Options struct {
	// ctx allows cancellation and deadlines.
	ctx context.Context

	// maxDepth controls frontier expansion depth (inclusive).
	// See MaxDepthUnlimited for semantics.
	maxDepth int

	// fullTraversal enables a BFS-forest traversal that visits all connected components.
	// When enabled, BFS continues with deterministic secondary roots after finishing startID's component.
	fullTraversal bool

	// onEnqueue is called when a vertex is enqueued (discovered), before it is visited.
	onEnqueue func(id string, depth int)

	// onDequeue is called immediately before a vertex is visited (dequeued).
	onDequeue func(id string, depth int)

	// onVisit is called when visiting a vertex.
	// If it returns an error, BFS aborts and returns a partial result with that error.
	onVisit func(id string) error

	// filterNeighbor can skip neighbor relations by returning false.
	// Called for each relation currID → nbrID surfaced by NeighborIDs.
	filterNeighbor func(currID, nbrID string) bool
}

// Default no-op callbacks and policies.
var (
	noOpEnqueue = func(string, int) {}
	noOpDequeue = func(string, int) {}
	noOpVisit   = func(string) error { return nil }
	allowAllNbr = func(string, string) bool { return true }
)

// DefaultOptions returns a fully-initialized Options value.
//
// Behavior highlights:
//   - ctx is context.Background().
//   - maxDepth is MaxDepthUnlimited (no limit).
//   - fullTraversal is disabled.
//   - all callbacks are non-nil no-ops.
//
// Inputs:
//   - none.
//
// Returns:
//   - Options: a valid options carrier for the BFS kernel.
//
// Errors:
//   - none.
//
// Determinism:
//   - deterministic.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Defaults are designed to be safe in hot paths.
//
// AI-Hints:
//   - Defaults are allocation-free and safe; customize only what you need.
func DefaultOptions() Options {
	return Options{
		ctx:            context.Background(),
		maxDepth:       MaxDepthUnlimited,
		fullTraversal:  false,
		onEnqueue:      noOpEnqueue,
		onDequeue:      noOpDequeue,
		onVisit:        noOpVisit,
		filterNeighbor: allowAllNbr,
	}
}

// applyOptions applies opts sequentially and returns the effective Options.
//
// Implementation:
//   - Stage 1: Initialize defaults.
//   - Stage 2: Apply each Option (nil option is rejected).
//   - Stage 3: Enforce defensive postconditions (non-nil ctx/callbacks).
//
// Behavior highlights:
//   - Fail-fast: any invalid option stops processing immediately.
//
// Inputs:
//   - opts: functional options; last-writer-wins for each field.
//
// Returns:
//   - Options: fully-initialized options value.
//   - error: a descriptive error (to be wrapped as ErrOptionViolation by the public API).
//
// Errors:
//   - Returns an error for nil options, nil ctx, nil callbacks, or invalid depth.
//
// Determinism:
//   - Stable for the same opts order; last-writer-wins is deterministic.
//
// Complexity:
//   - Time O(k), Space O(1).
//
// Notes:
//   - Options should not allocate; heavy configuration belongs outside hot paths.
//
// AI-Hints:
//   - Passing nil as an Option is rejected to prevent silent misconfiguration.
func applyOptions(opts ...Option) (Options, error) {
	o := DefaultOptions()

	for i, opt := range opts {
		if opt == nil {
			return Options{}, fmt.Errorf("nil option at index %d", i)
		}
		if err := opt(&o); err != nil {
			return Options{}, err
		}
	}

	if o.ctx == nil {
		return Options{}, fmt.Errorf("context is nil")
	}
	if o.onEnqueue == nil {
		return Options{}, fmt.Errorf("onEnqueue is nil")
	}
	if o.onDequeue == nil {
		return Options{}, fmt.Errorf("onDequeue is nil")
	}
	if o.onVisit == nil {
		return Options{}, fmt.Errorf("onVisit is nil")
	}
	if o.filterNeighbor == nil {
		return Options{}, fmt.Errorf("filterNeighbor is nil")
	}

	return o, nil
}

// WithContext sets the context used for cancellation.
//
// AI-HINTS:
//   - Passing nil is invalid; use context.Background() instead.
//   - Avoid time-based cancellation in Examples; use WithCancel + deterministic triggers.
func WithContext(ctx context.Context) Option {
	return func(o *Options) error {
		if ctx == nil {
			return fmt.Errorf("context is nil")
		}
		o.ctx = ctx
		return nil
	}
}

// WithMaxDepth sets an inclusive BFS depth limit.
//
// Semantics:
//   - d == MaxDepthUnlimited: no limit.
//   - d == 0: visit the root only; no neighbor expansion.
//   - d > 0: inclusive limit; expansion stops when currentDepth == d.
//   - d < MaxDepthUnlimited: invalid.
//
// AI-HINTS:
//   - Depth is measured in edge count (unweighted).
//   - Inclusive means vertices at depth d are visited, but their neighbors are not enqueued.
func WithMaxDepth(d int) Option {
	return func(o *Options) error {
		if d < MaxDepthUnlimited {
			return fmt.Errorf("maxDepth is invalid (%d)", d)
		}
		o.maxDepth = d
		return nil
	}
}

// WithFullTraversal enables BFS-forest traversal (visit all components).
//
// AI-HINTS:
//   - This mode is useful for building a forest.
//   - PathTo must still anchor to StartID; do not interpret forest paths as start-reachable paths.
func WithFullTraversal() Option {
	return func(o *Options) error {
		o.fullTraversal = true
		return nil
	}
}

// WithFilterNeighbor sets a relation-level neighbor filter.
//
// AI-HINTS:
//   - This filter operates on (currID, nbrID) pairs returned by NeighborIDs.
//   - Do not assume edge-level visibility (Edge.ID is not available here).
func WithFilterNeighbor(fn func(currID, nbrID string) bool) Option {
	return func(o *Options) error {
		if fn == nil {
			return fmt.Errorf("filterNeighbor is nil")
		}
		o.filterNeighbor = fn
		return nil
	}
}

// WithOnEnqueue registers a hook called when a vertex is enqueued.
//
// AI-HINTS:
//   - Hooks must be observers; do not mutate the graph.
//   - Keep hooks allocation-free for minimal overhead.
func WithOnEnqueue(fn func(id string, depth int)) Option {
	return func(o *Options) error {
		if fn == nil {
			return fmt.Errorf("onEnqueue is nil")
		}
		o.onEnqueue = fn
		return nil
	}
}

// WithOnDequeue registers a hook called immediately before a vertex is visited.
//
// AI-HINTS:
//   - This hook is a good place for deterministic cancellation triggers.
func WithOnDequeue(fn func(id string, depth int)) Option {
	return func(o *Options) error {
		if fn == nil {
			return fmt.Errorf("onDequeue is nil")
		}
		o.onDequeue = fn
		return nil
	}
}

// WithOnVisit registers a hook called when visiting a vertex.
//
// AI-HINTS:
//   - Returning an error stops BFS and returns a partial result.
//   - Use sentinel errors in your own code and wrap them for classification.
func WithOnVisit(fn func(id string) error) Option {
	return func(o *Options) error {
		if fn == nil {
			return fmt.Errorf("onVisit is nil")
		}
		o.onVisit = fn
		return nil
	}
}
