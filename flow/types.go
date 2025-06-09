package flow

import (
	"context"
	"errors"
)

// ErrSourceNotFound is returned when the specified source vertex does not exist in the graph.
var ErrSourceNotFound = errors.New("source vertex not found")

// ErrSinkNotFound is returned when the specified sink vertex does not exist in the graph.
var ErrSinkNotFound = errors.New("sink vertex not found")

// defaultEpsilon is the fallback tolerance for treating small capacities as zero.
// It is used only during initial residual construction (buildCapMap) to filter out
// edges whose aggregated capacity ≤ Epsilon. Once capacities are cast to int64,
// algorithms operate on exact integer values without further Epsilon comparisons.
const defaultEpsilon = 1e-9

// FlowOptions configures all maximum-flow algorithms in this package (Ford-Fulkerson,
// Edmonds-Karp, Dinic). Obtain a ready-to-use set of defaults via DefaultOptions().
//
// Fields:
//
//	Ctx                  - optional context for cancellation/timeouts. If nil,
//	                       DefaultOptions() and normalize() set this to context.Background().
//	Epsilon              - initial capacity filter threshold; capacities ≤ Epsilon
//	                       are discarded when building the initial capMap.
//	Verbose              - if true, each augmentation step will be logged via fmt.Printf.
//	LevelRebuildInterval - for Dinic only: number of augmentations between level-graph rebuilds.
//	                       0 disables automatic rebuilding.
type FlowOptions struct {
	// Ctx is the context used to cancel or timeout long-running computations.
	// It is normalized to context.Background() if left nil.
	Ctx context.Context

	// Epsilon defines the minimum significant capacity when aggregating parallel edges.
	// Any edge whose total capacity ≤ Epsilon is ignored in the initial residual map.
	// Algorithms then manipulate int64 capacities exactly without further rounding.
	Epsilon float64

	// Verbose enables step-by-step logging of each augmenting path and flow push.
	Verbose bool

	// LevelRebuildInterval controls how often (in augmentation count) Dinic rebuilds
	// its level graph. A value of 0 means "never rebuild" until the next BFS naturally.
	LevelRebuildInterval int
}

// normalize applies default values to any zero-value fields on FlowOptions.
// This ensures Ctx is never nil and Epsilon is always positive.
func (opts *FlowOptions) normalize() {
	// If no context provided, default to background to avoid nil dereference.
	if opts.Ctx == nil {
		opts.Ctx = context.Background()
	}
	// If Epsilon is not set or non-positive, reset to the package default.
	if opts.Epsilon <= 0 {
		opts.Epsilon = defaultEpsilon
	}
}

// DefaultOptions returns a FlowOptions populated with production-safe defaults:
//   - Ctx = context.Background()
//   - Epsilon = defaultEpsilon (1e-9)
//   - Verbose = false
//   - LevelRebuildInterval = 0 (no automatic level-graph rebuilds)
func DefaultOptions() FlowOptions {
	return FlowOptions{
		Ctx:                  context.Background(),
		Epsilon:              defaultEpsilon,
		Verbose:              false,
		LevelRebuildInterval: 0,
	}
}
