// SPDX-License-Identifier: MIT

// Package matrix: functional configuration for graph→matrix adapters and
// numeric policy. This file defines:
//   - Option / options (internal state)
//   - documented defaults (constants)
//   - WithX constructors with strong validation (panic on nonsensical values)
//   - gatherOptions helper (internal) that enforces invariants.
//
// Design goals:
//   - Deterministic behavior: no global state, no implicit randomness.
//   - No dead options: each switch affects behavior and is test-covered.
//   - Validation in constructors only; algorithms never panic on user input.
package matrix

import (
	"math"
)

// ---------- Defaults (single source of truth) ----------

// Numeric policy.
const (
	DefaultEpsilon        = 1e-9 // eps ≥ 0
	DefaultValidateNaNInf = true // reject NaN/±Inf on ingestion and Set
)

// DEFAULTS — single source of truth for zero-value behavior.
// These constants must reflect the intended defaults in NewMatrixOptions.
const (
	// DefaultDirected controls whether edges are treated as directed.
	// false ⇒ undirected (mirror [u,v] into [v,u], except loops).
	DefaultDirected = false

	// DefaultWeighted controls whether actual edge weights are preserved.
	// false ⇒ build binary adjacency/incidence with unit entries.
	DefaultWeighted = false

	// DefaultAllowMulti allows multiple parallel edges between the same endpoints.
	// true ⇒ include all by default; row/col will reflect the last write in a
	// dense adjacency cell (structural limitation). When false, policy is
	// first-edge-wins; see comments in builders.
	DefaultAllowMulti = true

	// DefaultAllowLoops includes self-loops when true.
	// NOTE: older comments in types.go said "zero value ⇒ AllowLoops=true".
	// We correct that here to match the actual intended default: false.
	DefaultAllowLoops = false

	// DefaultMetricClosure converts adjacency to all-pairs shortest path distances
	// (Floyd–Warshall) when true: diag=0, off-diag=+Inf if no path.
	// IMPORTANT: ToGraph is unsupported under MetricClosure (ErrMatrixNotImplemented).
	DefaultMetricClosure = false
)

// Export policy (AdjacencyMatrix.ToGraph).
const (
	DefaultEdgeThreshold = 0.5  // a[i,j] > threshold ⇒ edge
	DefaultKeepWeights   = true // if true, weight=a[i,j]; else weight=1
	// 'undirected' for export is inferred from 'directed' (see gatherOptions).
)

// ---------- Public option type (functional) ----------

// Option mutates internal options. Safe to apply repeatedly (idempotent).
// Constructors MUST panic only on nonsensical values (programmer error).
type Option func(*Options)

// Options stores the effective configuration after applying Option setters.
// It is intentionally unexported to prevent external mutation and to keep
// public API as `...opts ...Option` on entry points.
type Options struct {
	// numeric policy
	eps            float64 // >= 0; DefaultEpsilon
	validateNaNInf bool    // DefaultValidateNaNInf

	// adjacency/incidence build policy
	directed    bool // DefaultDirected
	allowMulti  bool // DefaultAllowMulti
	allowLoops  bool // DefaultAllowLoops
	weighted    bool // DefaultWeighted
	metricClose bool // DefaultMetricClosure

	// export policy (ToGraph)
	edgeThreshold float64 // DefaultEdgeThreshold
	undirected    bool    // inferred from 'directed' if not overridden internally
	keepWeights   bool    // DefaultKeepWeights
}

// ---------- Constructors (WithX) ----------

// WithEpsilon sets the numeric tolerance eps used by structural checks.
// Panics if eps < 0 or not finite.
// Complexity: O(1).
func WithEpsilon(eps float64) Option {
	if math.IsNaN(eps) || math.IsInf(eps, 0) || eps < 0 {
		panic("matrix: WithEpsilon: eps must be finite, non-negative")
	}
	return func(o *Options) { // assign validated epsilon
		o.eps = eps
	}
}

// WithValidateNaNInf enables strict finite-value validation.
// Complexity: O(1).
func WithValidateNaNInf() Option {
	return func(o *Options) { o.validateNaNInf = true }
}

// WithNoValidateNaNInf disables NaN/Inf validation (use with care).
// Complexity: O(1).
func WithNoValidateNaNInf() Option {
	return func(o *Options) { o.validateNaNInf = false }
}

// WithDirected builds directed adjacency/incidence (no mirroring).
// Complexity: O(1).
func WithDirected() Option {
	return func(o *Options) { o.directed = true }
}

// WithUndirected builds undirected adjacency/incidence (mirror [u,v]→[v,u], except loops).
// Complexity: O(1).
func WithUndirected() Option {
	return func(o *Options) { o.directed = false }
}

// WithAllowMulti includes parallel edges in the ingestion phase.
// Complexity: O(1).
func WithAllowMulti() Option {
	return func(o *Options) { o.allowMulti = true }
}

// WithDisallowMulti enforces "first-edge-wins" de-duplication:
//   - Directed: key is ordered pair (u,v)
//   - Undirected: key is unordered {min,max}
//
// Complexity: O(1).
func WithDisallowMulti() Option {
	return func(o *Options) { o.allowMulti = false }
}

// WithAllowLoops includes self-loops (u==v) during ingestion.
// Complexity: O(1).
func WithAllowLoops() Option {
	return func(o *Options) { o.allowLoops = true }
}

// WithDisallowLoops ignores self-loops during ingestion.
// Complexity: O(1).
func WithDisallowLoops() Option {
	return func(o *Options) { o.allowLoops = false }
}

// WithWeighted preserves actual edge weights if the input graph is weighted.
// If the source graph is effectively unweighted (all weights 0), builders
// will degrade to binary adjacency (1) to avoid an all-zero matrix.
// Complexity: O(1).
func WithWeighted() Option {
	return func(o *Options) { o.weighted = true }
}

// WithUnweighted forces binary adjacency/incidence (unit weights).
// Complexity: O(1).
func WithUnweighted() Option {
	return func(o *Options) { o.weighted = false }
}

// WithMetricClosure converts adjacency into all-pairs shortest-path distances
// via Floyd–Warshall (APSP). Under this mode, ToGraph is intentionally
// unsupported and MUST return ErrMatrixNotImplemented.
// Complexity: O(1) to set (O(n^3) later when applied).
func WithMetricClosure() Option {
	return func(o *Options) { o.metricClose = true }
}

// WithEdgeThreshold sets threshold for ToGraph export (a[i,j] > t ⇒ edge).
// Panics if t is NaN or ±Inf.
// Complexity: O(1).
func WithEdgeThreshold(t float64) Option {
	if math.IsNaN(t) || math.IsInf(t, 0) {
		panic("matrix: WithEdgeThreshold: threshold must be finite")
	}
	return func(o *Options) { o.edgeThreshold = t }
}

// WithKeepWeights keeps numeric weights on ToGraph export (weight=a[i,j]).
// Complexity: O(1).
func WithKeepWeights() Option {
	return func(o *Options) { o.keepWeights = true }
}

// WithBinaryWeights forces unit weights on ToGraph export (weight=1).
// Complexity: O(1).
func WithBinaryWeights() Option {
	return func(o *Options) { o.keepWeights = false }
}

// ---------- Defaults & aggregation ----------

// defaultOptions returns the documented defaults (single source of truth).
// Complexity: O(1).
func defaultOptions() Options {
	return Options{
		// numeric policy
		eps:            DefaultEpsilon,
		validateNaNInf: DefaultValidateNaNInf,

		// build policy
		directed:    DefaultDirected,
		allowMulti:  DefaultAllowMulti,
		allowLoops:  DefaultAllowLoops,
		weighted:    DefaultWeighted,
		metricClose: DefaultMetricClosure,

		// export policy
		edgeThreshold: DefaultEdgeThreshold,
		keepWeights:   DefaultKeepWeights,
		// undirected inferred in gatherOptions
	}
}

// gatherOptions applies user-provided Option setters on top of defaults and
// finalizes derived invariants (e.g., export undirected = !directed).
// This is the canonical internal entry in api/impl layers.
// Complexity: O(k), k=len(user).
func gatherOptions(user ...Option) Options {
	o := defaultOptions() // start from documented defaults
	for _, set := range user {
		set(&o) // apply in order; last-writer-wins semantics
	}
	// Derive export 'undirected' from build 'directed' (unless future code
	// chooses to override explicitly inside export path).
	o.undirected = !o.directed
	return o
}
