// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

package flow

import (
	"context"
	"fmt"
	"math"
)

// defaultEpsilon is the default threshold for treating tiny capacities as zero.
const defaultEpsilon = 1e-9

// Option configures the canonical MaxFlow facade.
//
// Implementation:
//   - Stage 1: Option mutates only the private options accumulator.
//   - Stage 2: applyOptions validates the complete final policy.
//
// Behavior highlights:
//   - Invalid option input returns sentinel errors; options never panic.
//   - Last-writer-wins is used for repeated policy options.
//
// Inputs:
//   - *options: private accumulator owned by applyOptions.
//
// Returns:
//   - error: nil on success, sentinel-classified error on invalid input.
//
// Errors:
//   - ErrInvalidOptions, ErrInvalidEpsilon, ErrObserverFailure through assembly.
//
// Determinism:
//   - Options do not depend on map iteration or runtime heuristics.
//
// Complexity:
//   - Each option is O(1).
//
// AI-Hints:
//   - Do not expose the private options type; public policy must flow through WithXxx.
type Option func(*options) error

type options struct {
	ctx context.Context

	epsilon   float64
	algorithm Algorithm

	levelRebuildInterval int

	observer AugmentationObserver
	verbose  bool

	maxAugmentations int
}

// AugmentationEvent describes one successful residual augmentation.
//
// Implementation:
//   - Stage 1: Kernels build Path from parent links using deterministic traversal.
//   - Stage 2: Kernels publish Delta, Total, and Index after residual update.
//
// Behavior highlights:
//   - Path is caller-owned for the observer call and must not be retained unless copied.
//
// Inputs:
//   - Algorithm: kernel that produced the event.
//   - Path: source-to-sink augmenting path.
//   - Delta: bottleneck capacity pushed through Path.
//   - Total: cumulative flow after Delta.
//   - Index: one-based augmentation count.
//
// Errors:
//   - Observer errors are wrapped with ErrObserverFailure.
//
// Determinism:
//   - Path follows residual adjacency order fixed by residualNetwork.
//
// Complexity:
//   - Event construction is O(path length).
//
// AI-Hints:
//   - Observers must not mutate the input graph or residual internals.
type AugmentationEvent struct {
	Algorithm Algorithm
	Path      []string
	Delta     float64
	Total     float64
	Index     int
}

// AugmentationObserver observes successful augmentation events.
//
// Implementation:
//   - Stage 1: Kernels call the observer after residual state is updated.
//   - Stage 2: Observer failure interrupts the run and returns a partial result.
//
// Behavior highlights:
//   - Observer is read-only by contract.
//   - Returning an error is the supported way to stop a run intentionally.
//
// Inputs:
//   - context.Context: active run context.
//   - AugmentationEvent: detached event payload.
//
// Returns:
//   - error: nil to continue, non-nil to interrupt.
//
// Errors:
//   - Non-nil observer errors are wrapped with ErrObserverFailure.
//
// Determinism:
//   - Observer call order equals augmentation order.
//
// Complexity:
//   - Package overhead is O(1) plus event path copy cost.
//
// AI-Hints:
//   - Do not use fmt.Printf inside kernels; use observers for diagnostics.
type AugmentationObserver func(context.Context, AugmentationEvent) error

// defaultOptionsInternal returns the canonical runtime option defaults.
// It is the single source of default policy for MaxFlow and legacy wrappers.
//
// Implementation:
//   - Stage 1: Use context.Background for runs without caller-provided context.
//   - Stage 2: Use defaultEpsilon for capacity filtering and residual clamping.
//   - Stage 3: Select AlgorithmDinic as the default general-purpose kernel.
//
// Behavior highlights:
//   - Defaults are conservative and deterministic.
//   - No allocation-heavy work is performed.
//   - Legacy DefaultOptions mirrors these values through FlowOptions.
//
// Inputs:
//   - None.
//
// Returns:
//   - options: private runtime configuration.
//
// Errors:
//   - None.
//
// Determinism:
//   - Always returns the same policy values.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Public callers must use Option values, not this private type.
//
// AI-Hints:
//   - Update this helper and DefaultOptions together when adding defaulted policy.
func defaultOptionsInternal() options {
	return options{
		ctx:       context.Background(),
		epsilon:   defaultEpsilon,
		algorithm: AlgorithmDinic,
	}
}

// applyOptions builds and validates the private runtime option set.
// It implements last-writer-wins semantics for repeated options.
//
// Implementation:
//   - Stage 1: Start from defaultOptionsInternal.
//   - Stage 2: Reject nil Option values before applying them.
//   - Stage 3: Apply options in caller order.
//   - Stage 4: Normalize nil context to context.Background.
//   - Stage 5: Validate epsilon, algorithm, rebuild interval, and augmentation limit.
//
// Behavior highlights:
//   - Validation happens before graph traversal or residual allocation.
//   - Nil options are classified as ErrInvalidOptions.
//   - Invalid epsilon is classified as ErrInvalidEpsilon.
//
// Inputs:
//   - user: variadic Option list from MaxFlow or adapters.
//
// Returns:
//   - options: finalized private runtime policy.
//   - error: nil on success or sentinel-classified option error.
//
// Errors:
//   - ErrInvalidOptions for nil/unknown/negative option policy.
//   - ErrInvalidEpsilon for negative, NaN, or infinite epsilon.
//
// Determinism:
//   - Option application order is exactly caller order.
//
// Complexity:
//   - Time O(k), Space O(1), where k is number of options.
//
// Notes:
//   - This helper must remain side-effect free except for calling Option closures.
//
// AI-Hints:
//   - Do not validate graph state here; use validators.go for graph contracts.
func applyOptions(user ...Option) (options, error) {
	cfg := defaultOptionsInternal()

	for i, opt := range user {
		if opt == nil {
			return options{}, fmt.Errorf("flow: option #%d: %w", i, ErrInvalidOptions)
		}
		if err := opt(&cfg); err != nil {
			return options{}, err
		}
	}

	if cfg.ctx == nil {
		cfg.ctx = context.Background()
	}
	if math.IsNaN(cfg.epsilon) || math.IsInf(cfg.epsilon, 0) || cfg.epsilon < 0 {
		return options{}, ErrInvalidEpsilon
	}
	if cfg.levelRebuildInterval < 0 {
		return options{}, ErrInvalidOptions
	}
	if cfg.maxAugmentations < 0 {
		return options{}, ErrInvalidOptions
	}

	switch cfg.algorithm {
	case AlgorithmDinic, AlgorithmEdmondsKarp, AlgorithmFordFulkerson:
		return cfg, nil
	default:
		return options{}, ErrInvalidOptions
	}
}

// WithContext sets the cancellation context for MaxFlow.
//
// Implementation:
//   - Stage 1: Store context on the private options accumulator.
//   - Stage 2: applyOptions replaces nil with context.Background.
//
// Behavior highlights:
//   - Nil context is accepted and normalized to context.Background.
//
// Inputs:
//   - ctx: cancellation and deadline context.
//
// Returns:
//   - Option: option closure for MaxFlow.
//
// Errors:
//   - None directly.
//
// Determinism:
//   - Context changes cancellation timing only, not traversal order.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// AI-Hints:
//   - Use deterministic canceled contexts in tests instead of time-based flakiness.
func WithContext(ctx context.Context) Option {
	return func(o *options) error {
		o.ctx = ctx
		return nil
	}
}

// WithEpsilon sets the capacity threshold used by residual traversal.
//
// Implementation:
//   - Stage 1: Validate epsilon immediately.
//   - Stage 2: Store epsilon for capacity filtering and residual clamping.
//
// Behavior highlights:
//   - Capacities <= epsilon are treated as absent.
//   - Residual values with absolute value <= epsilon are clamped to zero.
//
// Inputs:
//   - eps: non-negative finite threshold.
//
// Returns:
//   - Option: option closure for MaxFlow.
//
// Errors:
//   - ErrInvalidEpsilon when eps is negative, NaN, or Inf.
//
// Determinism:
//   - Epsilon affects eligibility, not ordering.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// AI-Hints:
//   - Do not use epsilon to hide negative capacities below -epsilon.
func WithEpsilon(eps float64) Option {
	return func(o *options) error {
		if math.IsNaN(eps) || math.IsInf(eps, 0) || eps < 0 {
			return ErrInvalidEpsilon
		}
		o.epsilon = eps
		return nil
	}
}

// WithAlgorithm selects the maximum-flow kernel.
//
// Implementation:
//   - Stage 1: Validate the Algorithm constant.
//   - Stage 2: Store the selected kernel in options.
//
// Behavior highlights:
//   - AlgorithmDinic is the default.
//   - Legacy wrappers force the corresponding algorithm explicitly.
//
// Inputs:
//   - alg: AlgorithmDinic, AlgorithmEdmondsKarp, or AlgorithmFordFulkerson.
//
// Returns:
//   - Option: option closure for MaxFlow.
//
// Errors:
//   - ErrInvalidOptions for unknown algorithms.
//
// Determinism:
//   - Selection is explicit and stable.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// AI-Hints:
//   - Do not add heuristic auto-selection without a separate explicit option.
func WithAlgorithm(alg Algorithm) Option {
	return func(o *options) error {
		switch alg {
		case AlgorithmDinic, AlgorithmEdmondsKarp, AlgorithmFordFulkerson:
			o.algorithm = alg
			return nil
		default:
			return ErrInvalidOptions
		}
	}
}

// WithLevelRebuildInterval sets Dinic level-graph rebuild cadence.
//
// Implementation:
//   - Stage 1: Validate the non-negative interval.
//   - Stage 2: Store the interval for Dinic only.
//
// Behavior highlights:
//   - Zero disables augmentation-count-triggered rebuilds.
//   - The option is ignored by Edmonds-Karp and Ford-Fulkerson.
//
// Inputs:
//   - n: non-negative augmentation interval.
//
// Returns:
//   - Option: option closure for MaxFlow.
//
// Errors:
//   - ErrInvalidOptions when n is negative.
//
// Determinism:
//   - Rebuilds occur after deterministic augmentation counts.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// AI-Hints:
//   - Keep this as a performance policy; it must not change final max-flow value.
func WithLevelRebuildInterval(n int) Option {
	return func(o *options) error {
		if n < 0 {
			return ErrInvalidOptions
		}
		o.levelRebuildInterval = n
		return nil
	}
}

// WithObserver registers an augmentation observer.
//
// Implementation:
//   - Stage 1: Store the observer callback.
//   - Stage 2: Kernels invoke the observer after each successful augmentation.
//
// Behavior highlights:
//   - Nil observer disables observation.
//   - Observer errors interrupt the run and produce partial result semantics.
//
// Inputs:
//   - observer: optional read-only callback.
//
// Returns:
//   - Option: option closure for MaxFlow.
//
// Errors:
//   - Observer runtime errors are wrapped with ErrObserverFailure.
//
// Determinism:
//   - Observer call order equals augmentation order.
//
// Complexity:
//   - Time O(1) to configure; runtime overhead is observer-defined.
//
// AI-Hints:
//   - Prefer observers over direct stdout logging inside algorithms.
func WithObserver(observer AugmentationObserver) Option {
	return func(o *options) error {
		o.observer = observer
		return nil
	}
}

// WithMaxAugmentations sets a hard limit on successful augmenting pushes.
// It is a safety valve for algorithms whose practical runtime depends heavily on path choice.
//
// Implementation:
//   - Stage 1: Validate the limit as a non-negative integer.
//   - Stage 2: Store the limit on the private options accumulator.
//   - Stage 3: Kernels check the limit only after proving another augmenting path exists.
//
// Behavior highlights:
//   - Zero means unlimited.
//   - The limit counts successful pushes, not BFS/DFS search attempts.
//   - Reaching the limit returns ErrAugmentationLimit with a partial result.
//
// Inputs:
//   - n: maximum number of successful augmentations; zero disables the limit.
//
// Returns:
//   - Option: option closure for MaxFlow.
//
// Errors:
//   - ErrInvalidOptions when n is negative.
//
// Determinism:
//   - The same graph, algorithm, epsilon, and traversal order reach the limit at the same point.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - This option is most useful for AlgorithmFordFulkerson.
//   - Dinic and Edmonds-Karp also honor the option for uniform runtime governance.
//
// AI-Hints:
//   - Do not check the limit before knowing whether another augmenting path exists;
//     otherwise a graph that finishes exactly at the limit could be reported as failed.
func WithMaxAugmentations(n int) Option {
	return func(o *options) error {
		if n < 0 {
			return ErrInvalidOptions
		}

		o.maxAugmentations = n
		return nil
	}
}

// checkAugmentationLimit verifies whether another successful push is allowed.
// It must be called only after an augmenting path has been found.
//
// Implementation:
//   - Stage 1: Treat zero limit as unlimited.
//   - Stage 2: Compare the current successful augmentation count against the limit.
//   - Stage 3: Return ErrAugmentationLimit only when the next push would exceed policy.
//
// Behavior highlights:
//   - Avoids false failures when the optimum is reached exactly at the configured limit.
//   - Does not mutate algorithm state.
//
// Inputs:
//   - count: number of successful augmentations already applied.
//   - limit: configured maximum; zero means unlimited.
//
// Returns:
//   - error: nil when another augmentation is allowed.
//
// Errors:
//   - ErrAugmentationLimit when count >= limit and limit > 0.
//
// Determinism:
//   - Pure numeric predicate.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// AI-Hints:
//   - Call this after path discovery and before addResidual.
func checkAugmentationLimit(count int, limit int) error {
	if limit > 0 && count >= limit {
		return ErrAugmentationLimit
	}

	return nil
}

// optionsFromLegacy converts FlowOptions into canonical Option values.
// It lets legacy wrappers reuse MaxFlow without duplicating policy logic.
//
// Implementation:
//   - Stage 1: Force the algorithm selected by the legacy wrapper.
//   - Stage 2: Transfer context, epsilon, Dinic rebuild interval, and augmentation limit.
//   - Stage 3: Add a verbose observer bridge when legacy Verbose is enabled.
//
// Behavior highlights:
//   - Legacy wrappers remain source-compatible.
//   - Canonical validation still happens inside applyOptions.
//   - New option fields should be bridged here only when they preserve legacy semantics.
//
// Inputs:
//   - legacy: historical FlowOptions value.
//   - algorithm: Algorithm forced by Dinic/EdmondsKarp/FordFulkerson wrapper.
//
// Returns:
//   - []Option: canonical option list for MaxFlow.
//
// Errors:
//   - None directly; returned options are validated later by applyOptions.
//
// Determinism:
//   - Output option order is fixed.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - This helper does not call MaxFlow itself.
//
// AI-Hints:
//   - Do not copy algorithm code into wrappers; bridge through this helper.
func optionsFromLegacy(legacy FlowOptions, algorithm Algorithm) []Option {
	opts := []Option{
		WithAlgorithm(algorithm),
		WithContext(legacy.Ctx),
		WithEpsilon(legacy.Epsilon),
		WithLevelRebuildInterval(legacy.LevelRebuildInterval),
		WithMaxAugmentations(legacy.MaxAugmentations),
	}

	if legacy.Verbose {
		opts = append(opts, withLegacyVerboseObserver())
	}

	return opts
}

// withLegacyVerboseObserver marks canonical runtime options as legacy-verbose.
// The actual formatted output is emitted by notifyAugmentation, not by kernels.
//
// Implementation:
//   - Stage 1: Set cfg.verbose to true.
//   - Stage 2: Leave observer unchanged so user-provided observers are not replaced.
//   - Stage 3: notifyAugmentation prints when cfg.verbose is true.
//
// Behavior highlights:
//   - Keeps fmt.Printf outside algorithmic kernels.
//   - Preserves legacy Verbose behavior through the shared observer publication path.
//   - Does not allocate or capture heavy state.
//
// Inputs:
//   - None.
//
// Returns:
//   - Option: internal compatibility option.
//
// Errors:
//   - None.
//
// Determinism:
//   - Verbose output order follows deterministic augmentation order.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - This option is intentionally private.
//
// AI-Hints:
//   - Do not install a dummy observer here; notifyAugmentation already handles verbose.
func withLegacyVerboseObserver() Option {
	return func(o *options) error {
		o.verbose = true
		return nil
	}
}
