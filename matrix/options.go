// SPDX-License-Identifier: MIT

// Package matrix: functional configuration for graph→matrix adapters and
// numeric policy. This file defines:
//   - Option / Options (functional options with internal state),
//   - documented defaults (constants),
//   - WithX constructors with strong validation (panic on nonsensical values),
//   - gatherOptions helper (internal) that enforces invariants.
//
// Design goals:
//   - Deterministic behavior: no global state, no implicit randomness.
//   - No dead switches: each flag impacts behavior and is covered by tests.
//   - Safe by construction: panic only on invalid parameters (programmer error).
//   - Reusability: Options is internal; public APIs consume ...Option.
//
// Notes:
//   - Core vs Matrix defaults: core graphs by default are undirected, unweighted,
//     no loops, no multi-edges. Matrix adapters have the same spirit except that
//     the adapter default here is AllowMulti = true (i.e., if a source graph
//     contained parallel edges, the adapter would include them). In practice
//     core will not output parallel edges unless the graph was constructed with
//     multi-edges enabled, so the defaults remain consistent in the common case.
//   - Numeric policy (validateNaNInf): adapters (e.g., adjacency/incidence
//     builders) propagate this flag into newly created Dense matrices via an
//     internal constructor (newDenseWithPolicy). There is no toggle for existing
//     matrices; the flag is effective at creation time.
//   - Incidence matrices do not use numeric edge weights for signs; any Weighted
//     toggle is ignored by the incidence builder. This is documented here to
//     avoid confusion; the builder will prefer correctness over the flag.
package matrix

import (
	"math"
)

// ---------- Defaults (single source of truth) ----------

// Numeric policy.
const (
	// DefaultEpsilon defines the non-negative tolerance used by structural checks.
	DefaultEpsilon = 1e-9
	// DefaultValidateNaNInf toggles strict finite-value validation on ingestion and Set.
	DefaultValidateNaNInf = true
)

// DEFAULTS - single source of truth for zero-value behavior.
// These constants MUST reflect the intended defaults in defaultOptions.
const (
	// DefaultDirected controls whether edges are treated as directed.
	// false ⇒ undirected (mirror [u,v] into [v,u], except loops).
	DefaultDirected = false

	// DefaultWeighted controls whether actual edge weights are preserved.
	// false ⇒ build binary adjacency/incidence with unit entries.
	DefaultWeighted = false

	// DefaultAllowMulti allows multiple parallel edges between the same endpoints.
	// true ⇒ include all by default; row/col reflects the last write in a dense
	// adjacency cell (structural limitation). When false, the policy is
	// first-edge-wins; see comments in builders.
	DefaultAllowMulti = true

	// DefaultAllowLoops includes self-loops when true.
	// NOTE: older comments elsewhere may suggest otherwise; the intended default is false.
	DefaultAllowLoops = false

	// DefaultMetricClosure converts adjacency to all-pairs shortest-path distances
	// (Floyd–Warshall) when true: diag=0, off-diag=+Inf if no path.
	// IMPORTANT: ToGraph is unsupported under MetricClosure (ErrMatrixNotImplemented).
	DefaultMetricClosure = false
)

// Export policy (AdjacencyMatrix.ToGraph).
const (
	// DefaultEdgeThreshold: a[i,j] > threshold ⇒ edge.
	DefaultEdgeThreshold = 0.5
	// DefaultKeepWeights: if true, weight=a[i,j]; else weight=1.
	DefaultKeepWeights = true
	// 'undirected' for export is inferred from 'directed' (see gatherOptions).
)

// ---------- Internal panic messages (no magic strings) ----------

const (
	panicEpsilonInvalid       = "matrix: WithEpsilon: eps must be finite, non-negative"
	panicEdgeThresholdInvalid = "matrix: WithEdgeThreshold: threshold must be finite"
)

// ---------- Public option type (functional) ----------

// Option mutates internal options. Safe to apply repeatedly (idempotent).
// Constructors MUST panic only on nonsensical values (programmer error).
type Option func(*Options)

// Options stores the effective configuration after applying Option setters.
// It is intentionally unexported to prevent external mutation; public entry
// points accept `...Option` and internally resolve them via gatherOptions.
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
// Implementation:
//   - Stage 1: validate eps is finite and ≥ 0.
//   - Stage 2: return a setter that writes eps into Options.
//
// Behavior highlights:
//   - Strict validation in constructor; panics on nonsensical values.
//
// Inputs:
//   - eps: non-negative finite tolerance.
//
// Returns:
//   - Option: functional setter.
//
// Errors:
//   - Panics with a stable message when eps is invalid.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Applies to structural numeric checks (e.g., AllClose-like, symmetry).
//     Larger eps relaxes equality checks; use judiciously.
//
// AI-Hints:
//   - Prefer small positive eps (e.g., 1e-9) for double-precision data or unless dealing with noisy data.
func WithEpsilon(eps float64) Option {
	if math.IsNaN(eps) || math.IsInf(eps, 0) || eps < 0 {
		panic(panicEpsilonInvalid)
	}

	// Assign validated epsilon
	return func(o *Options) { o.eps = eps }
}

// WithValidateNaNInf enables strict finite-value validation.
// Implementation:
//   - Stage 1: set validateNaNInf=true.
//
// Behavior highlights:
//   - Enforced at Dense creation and Set/Clone/Transpose/Induced/IdentityLike.
//
// Returns:
//   - Option: functional setter.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Affects newly created matrices via builders; existing matrices keep their policy.
//   - This is the default; use WithNoValidateNaNInf to relax.
//
// AI-Hints:
//   - Keep this enabled in data-clean pipelines; disable only when ingesting
//     external data with known ±Inf placeholders and sanitizing later or  in controlled experiments.
func WithValidateNaNInf() Option {
	return func(o *Options) { o.validateNaNInf = true }
}

// WithNoValidateNaNInf disables NaN/Inf validation (use with care).
// Implementation:
//   - Stage 1: set validateNaNInf=false.
//
// Behavior highlights:
//   - Allows ±Inf/NaN to pass through on newly created matrices.
//
// Returns:
//   - Option: functional setter.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - This flag propagates only on creation; existing matrices are unaffected.
//
// AI-Hints:
//   - Combine with data sanitization ops (ReplaceInfNaN, Clip) if you disable checks.
func WithNoValidateNaNInf() Option {
	return func(o *Options) { o.validateNaNInf = false }
}

// WithDirected builds directed adjacency/incidence (no mirroring).
// Implementation:
//   - Stage 1: set directed=true; export undirected is derived later.
//
// Behavior highlights:
//   - Export 'undirected' is computed as !directed in gatherOptions.
//
// Returns:
//   - Option: functional setter.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Incidence sign conventions depend on this flag.
//   - For export, undirected is inferred; no separate public toggle required.
//
// AI-Hints:
//   - Use WithUndirected for mirrored graphs without asymmetric weights.
//   - Combine with AllowMulti/AllowLoops explicitly for reproducibility.
func WithDirected() Option {
	return func(o *Options) { o.directed = true }
}

// WithUndirected builds undirected adjacency/incidence (mirror [u,v]→[v,u], except loops).
// Implementation:
//   - Stage 1: set directed=false.
//
// Behavior highlights:
//   - Export undirected field will be set to true by gatherOptions.
//
// Returns:
//   - Option: functional setter.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Loops remain on the diagonal and are not mirrored.
//
// AI-Hints:
//   - Prefer undirected for symmetric datasets; pair with Weighted as needed.
//   - In undirected mode, multi-edge deduplication normalizes pair keys.
func WithUndirected() Option {
	return func(o *Options) { o.directed = false }
}

// WithAllowMulti includes parallel edges in the ingestion phase.
// Implementation:
//   - Stage 1: set allowMulti=true.
//
// Behavior highlights:
//   - Adjacency cells may be updated multiple times; last-write-wins on Dense.
//
// Returns:
//   - Option: functional setter.
//
// Determinism:
//   - Deterministic if edge iteration order is stable (builders enforce this).
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Undirected mode uses normalized keys {min,max} for de-duplication.
//
// AI-Hints:
//   - Prefer disallowing multi unless your pipeline requires parallel edges.
//   - If you need strict first-edge semantics, use WithDisallowMulti.
func WithAllowMulti() Option {
	return func(o *Options) { o.allowMulti = true }
}

// WithDisallowMulti enforces "first-edge-wins" de-duplication.
// Implementation:
//   - Stage 1: set allowMulti=false.
//
// Behavior highlights:
//   - Directed: key uses ordered pair (u,v); Undirected: unordered {min,max}.
//
// Returns:
//   - Option: functional setter.
//
// Determinism:
//   - Requires stable vertex indexing (builders guarantee lexicographic order).
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Prevents late edges from overwriting earlier weights.
//
// AI-Hints:
//   - Prefer this when ingest order is meaningful (ETL pipelines).
func WithDisallowMulti() Option {
	return func(o *Options) { o.allowMulti = false }
}

// WithAllowLoops includes self-loops (u==v) during ingestion.
// Implementation:
//   - Stage 1: set allowLoops=true.
//
// Behavior highlights:
//   - Loops affect diagonal entries only.
//
// Returns:
//   - Option: functional setter.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - For incidence, undirected loop contributes +2 in a single row.
//
// AI-Hints:
//   - Enable when loops carry semantic meaning (e.g., self-transitions).
func WithAllowLoops() Option {
	return func(o *Options) { o.allowLoops = true }
}

// WithDisallowLoops ignores self-loops during ingestion.
// Implementation:
//   - Stage 1: set allowLoops=false.
//
// Behavior highlights:
//   - Diagonal contributions are suppressed.
//
// Returns:
//   - Option: functional setter.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Use to simplify structures where loops are artifacts.
//
// AI-Hints:
//   - Pairs well with binary adjacency for structural analyses.
func WithDisallowLoops() Option {
	return func(o *Options) { o.allowLoops = false }
}

// WithWeighted preserves actual edge weights if the input graph is weighted.
// Implementation:
//   - Stage 1: set weighted=true.
//
// Behavior highlights:
//   - Adjacency: stores edge weights (or 1 if graph is effectively unweighted).
//   - Incidence: structure-only; weights are ignored by design.
//
// Returns:
//   - Option: functional setter.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - If the source graph has only zeros, builders may promote to binary 1s
//     to avoid an all-zero adjacency.
//   - Incidence ignoring weights is intentional and documented; see file header.
//
// AI-Hints:
//   - Combine with MetricClosure to obtain distance matrices (APSP) from weights.
func WithWeighted() Option {
	return func(o *Options) { o.weighted = true }
}

// WithUnweighted forces binary adjacency/incidence (unit weights).
// Implementation:
//   - Stage 1: no validation needed.
//   - Stage 2: set weighted=false.
//
// Behavior highlights:
//   - Adjacency becomes {0,1}; incidence remains sign-based.
//
// Returns:
//   - Option: functional setter.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Useful for topology-only analyses.
//
// AI-Hints:
//   - Pair with DisallowMulti for clean simple graphs.
func WithUnweighted() Option {
	return func(o *Options) { o.weighted = false }
}

// WithMetricClosure converts adjacency into all-pairs shortest-path distances via Floyd–Warshall (APSP).
// Implementation:
//   - Stage 1: no validation needed.
//   - Stage 2: set metricClose=true (APSP later performed in builders).
//
// Behavior highlights:
//   - Produces distance-policy matrices: diag=0, +Inf denotes no path.
//   - ToGraph is intentionally unsupported and must return ErrMatrixNotImplemented.
//
// Returns:
//   - Option: functional setter.
//
// Errors:
//   - none here; APSP may surface errors during execution.
//
// Determinism:
//   - Floyd–Warshall uses a fixed k→i→j loop order.
//
// Complexity:
//   - Time O(1) to set; APSP later is O(n^3).
//
// Notes:
//   - Input adjacency is not mutated when using non-in-place facade.
//
// AI-Hints:
//   - Prefer APSPFromAdjacency for convenience if available at API level.
func WithMetricClosure() Option {
	return func(o *Options) { o.metricClose = true }
}

// WithEdgeThreshold sets threshold for ToGraph export (a[i,j] > t ⇒ edge).
// Implementation:
//   - Stage 1: set edgeThreshold=t.
//
// Behavior highlights:
//   - Affects only export; has no impact on in-memory computations.
//
// Inputs:
//   - t: finite threshold (NaN/±Inf disallowed).
//
// Returns:
//   - Option: functional setter.
//
// Errors:
//   - Panics with a stable message when t is invalid.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Choose t around half the unit if using binary adjacency (e.g., 0.5).
//
// AI-Hints:
//   - For weighted graphs, pick a domain-relevant cutoff.
func WithEdgeThreshold(t float64) Option {
	if math.IsNaN(t) || math.IsInf(t, 0) {
		panic(panicEdgeThresholdInvalid)
	}

	return func(o *Options) { o.edgeThreshold = t }
}

// WithKeepWeights keeps numeric weights on ToGraph export (weight=a[i,j]).
// Implementation:
//   - Stage 1: set keepWeights=true.
//
// Behavior highlights:
//   - Edge weights reflect matrix entries.
//
// Returns:
//   - Option: functional setter.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Combine with Weighted adjacency for meaningful magnitudes.
//
// AI-Hints:
//   - Set an appropriate EdgeThreshold to filter weak links.
func WithKeepWeights() Option {
	return func(o *Options) { o.keepWeights = true }
}

// WithBinaryWeights forces unit weights on ToGraph export (weight=1).
// Implementation:
//   - Stage 1: no validation needed.
//   - Stage 2: set keepWeights=false.
//
// Behavior highlights:
//   - Edges become unweighted on export, irrespective of matrix entries.
//
// Returns:
//   - Option: functional setter.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Useful for producing topological graphs from weighted matrices.
//
// AI-Hints:
//   - Pair with a threshold to sparsify exports.
func WithBinaryWeights() Option {
	return func(o *Options) { o.keepWeights = false }
}

// --------------------------- Deprecated Aliases ---------------------------

// DisableValidateNaNInf disables NaN/Inf validation.
// Deprecated: Use WithNoValidateNaNInf.
// NOTE: This alias is kept for backwards compatibility and may be removed
// in a future major release (see CHANGELOG).
func DisableValidateNaNInf() Option {
	return WithNoValidateNaNInf()
}

// --------------------------- Option Resolution ---------------------------

// NewMatrixOptions resolves option setters against documented defaults.
// Implementation:
//   - Stage 1: start from defaultOptions() (single source of truth).
//   - Stage 2: apply opt in order; last-writer-wins semantics.
//   - Stage 3: return the internal Options value.
//
// Behavior highlights:
//   - Pure function; no side effects beyond producing a value.
//
// Inputs:
//   - opts: zero or more functional setters.
//
// Returns:
//   - Options: internal struct with effective configuration.
//
// Determinism:
//   - Stable for a given sequence of opts.
//
// Complexity:
//   - Time O(k), Space O(1) for k=len(opts).
//
// Notes:
//   - Most public entry points accept ...Option and call gatherOptions.
//   - The resulting Options is intended for internal consumption by builders.
//
// AI-Hints:
//   - Compose options close to the adapter call-site for clarity.
//   - Group related flags together (e.g., Directed + DisallowMulti).
func NewMatrixOptions(opts ...Option) Options {
	o := defaultOptions()
	for _, opt := range opts {
		opt(&o)
	}

	return o
}

// defaultOptions returns the documented defaults (single source of truth).
// Implementation:
//   - Stage 1: fill fields from Default* constants.
//   - Stage 2: leave derived 'undirected' to be finalized by gatherOptions.
//
// Behavior highlights:
//   - Ensures defaults and comments never diverge.
//
// Returns:
//   - Options: defaults snapshot.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Keep this in sync with constants above.
//
// AI-Hints:
//   - Use NewMatrixOptions() to override selectively.
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
// finalizes derived invariants (e.g., export 'undirected' = !build.directed).
// This is the canonical internal entry in api/impl layers.
// Implementation:
//   - Stage 1: start from defaultOptions().
//   - Stage 2: apply setters in order (last-writer-wins).
//   - Stage 3: derive export.undirected = !build.directed.
//
// Behavior highlights:
//   - Derivations in one place prevent drift across call sites.
//     -
//   - Centralized point ensuring internal invariants before builder use.
//
// Inputs:
//   - user: sequence of Option setters.
//
// Returns:
//   - Options: fully resolved configuration, with derived 'undirected' set accordingly.
//
// Determinism:
//   - Stable for a given sequence of setters.
//
// Complexity:
//   - Time O(k), Space O(1) for k=len(user).
//
// Notes:
//   - Builders may ignore Weighted for incidence (sign-based).
//   - MetricClosure triggers APSP in adjacency builder only.
//   - If future export needs a distinct undirected toggle, set it here.
//
// AI-Hints:
//   - Prefer gatherOptions(...) over ad-hoc defaulting in callers.
func gatherOptions(user ...Option) Options {
	o := defaultOptions() // start from documented defaults
	for _, set := range user {
		set(&o) // apply in order; last-writer-wins semantics
	}
	// Derive export 'undirected' from build 'directed'.
	o.undirected = !o.directed

	return o
}
