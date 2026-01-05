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
//   - Reusability: ; public APIs consume ...Option.
//
// Design goals:
//   - Deterministic behavior: no global state, no implicit randomness.
//   - No dead switches: each flag impacts behavior and is covered by tests.
//   - Safe by construction: panic only on invalid parameters (programmer error).
//   - Reusability: Options fields are unexported (is internal); public APIs consume ...Option.
//
// Notes:
//   - Core vs Matrix defaults:
//   - core defaults are: undirected, unweighted, loops=false, multi-edges=false.
//   - matrix adapter defaults mirror that spirit, with one intentional difference:
//     DefaultAllowMulti=true so adapters can faithfully represent multi-edge graphs
//     when the source graph was explicitly built with WithMultiEdges().
//   - Directedness mapping (core → matrix):
//   - core can be uniform-directed or mixed-mode per-edge (Edge.Directed).
//   - adapters must remain deterministic: vertex order is stable (ID asc), and
//     edge iteration must be stable (Edge.ID asc) before writing into matrices.
//   - Multi-edge representation:
//   - Dense adjacency has one cell per (u,v); it cannot losslessly represent
//     parallel edges. When AllowMulti=true, a deterministic overwrite policy
//     must be documented (e.g., last-write-wins under stable edge order).
//   - Incidence matrices can represent multi-edges naturally (one column per edge).
//   - Weighted semantics:
//   - core enforces weight==0 unless WithWeighted() was enabled.
//   - matrix.Weighted controls whether adapters export numeric weights or a binary structure.
//   - incidence sign is structural; numeric weights are intentionally ignored there.
//   - Numeric policy is orthogonal and explicit:
//   - validateNaNInf controls whether Set()/ingestion rejects NaN/Inf at all.
//   - allowInfDistances is a narrow exception for +Inf as “no path” in distance matrices.
//     Under validation, NaN and -Inf remain rejected even when allowInfDistances=true.
//   - MetricClosure / APSP:
//   - distance-policy builders require +Inf off-diagonal for “no path”.
//   - those builders must allocate Dense with allowInfDistances=true to make Set(+Inf) legal.
package matrix

// ---------- Defaults (single source of truth) ----------

// Numeric policy.
const (
	// DefaultEpsilon defines the non-negative tolerance used by structural checks.
	DefaultEpsilon = 1e-9

	// DefaultValidateNaNInf toggles strict finite-value validation on ingestion and Set.
	DefaultValidateNaNInf = true

	// DefaultAllowInfDistances permits +Inf values to represent “no path” in
	// APSP/MetricClosure distance-policy matrices.
	//
	// IMPORTANT:
	//   - This is NOT a “dirty-data” mode.
	//   - When ValidateNaNInf is enabled, NaN and -Inf are still rejected; only +Inf
	//     is allowed by this mode.
	DefaultAllowInfDistances = false
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
	DefaultAllowLoops = false

	// DefaultMetricClosure converts adjacency to all-pairs shortest-path distances
	// (Floyd–Warshall) when true: diag=0, off-diag=+Inf if no path.
	// IMPORTANT: ToGraph is unsupported under MetricClosure (ErrMatrixNotImplemented).
	DefaultMetricClosure = false
)

// Export policy (AdjacencyMatrix.ToGraph).
const (
	// DefaultEdgeThreshold a[i,j] > threshold ⇒ edge.
	DefaultEdgeThreshold = 0.5

	// DefaultKeepWeights if true, weight=a[i,j]; else weight=1.
	DefaultKeepWeights = true

	// DefaultBinaryWeights if true, exported edges get weight=1 (ignores a[i,j]).
	DefaultBinaryWeights = false
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
	eps               float64 // >= 0; DefaultEpsilon
	validateNaNInf    bool    // DefaultValidateNaNInf
	allowInfDistances bool    // DefaultAllowInfDistances (+Inf as “no path”)

	// adjacency/incidence build policy
	directed    bool // DefaultDirected
	allowMulti  bool // DefaultAllowMulti
	allowLoops  bool // DefaultAllowLoops
	weighted    bool // DefaultWeighted
	metricClose bool // DefaultMetricClosure

	// export policy (ToGraph)
	edgeThreshold float64 // DefaultEdgeThreshold
	keepWeights   bool    // DefaultKeepWeights
	binaryWeights bool    // DefaultBinaryWeights
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
	if isNonFinite(eps) || eps < 0 {
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
//   - When enabled, NaN and -Inf are always rejected.
//   - +Inf is rejected unless AllowInfDistances is enabled.
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
//     external data with known ±Inf placeholders and sanitizing later or in controlled experiments.
//   - Combine with WithAllowInfDistances for APSP.
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

// WithAllowInfDistances permits +Inf entries to represent “no path” in distance-policy matrices.
// Implementation:
//   - Stage 1: set allowInfDistances=true.
//
// Behavior highlights:
//   - Does NOT imply “allow NaN”: if ValidateNaNInf is enabled, NaN and -Inf are still rejected.
//
// Returns:
//   - Option: functional setter.
//
// Errors:
//   - none (pure setter).
//
// Determinism:
//   - N/A.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Intended for APSP/MetricClosure where +Inf is a semantic sentinel.
//
// AI-Hints:
//   - Use together with WithMetricClosure or APSP builders.
func WithAllowInfDistances() Option {
	return func(o *Options) { o.allowInfDistances = true }
}

// WithDisallowInfDistances disables +Inf-permission mode (default).
// Implementation:
//   - Stage 1: set allowInfDistances=false.
//
// Behavior highlights:
//   - If ValidateNaNInf is enabled, all infinities are rejected.
//
// Returns:
//   - Option: functional setter.
//
// Complexity:
//   - Time O(1), Space O(1).
func WithDisallowInfDistances() Option {
	return func(o *Options) { o.allowInfDistances = false }
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
	if isNonFinite(t) {
		panic(panicEdgeThresholdInvalid)
	}

	return func(o *Options) { o.edgeThreshold = t }
}

// WithKeepWeights keeps numeric weights on ToGraph export (weight=a[i,j]).
// Implementation:
//   - Stage 1: set keepWeights=true and binaryWeights=flase.
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
	return func(o *Options) {
		o.keepWeights = true
		o.binaryWeights = false
	}
}

// WithBinaryWeights forces unit weights on ToGraph export (weight=1).
// Implementation:
//   - Stage 1: no validation needed.
//   - Stage 2: set keepWeights=false and binaryWeights=true.
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
	return func(o *Options) {
		o.binaryWeights = true
		o.keepWeights = false
	}
}

// --------------------------- Deprecated Aliases ---------------------------

// DisableValidateNaNInf disables NaN/Inf validation.
// Deprecated: Use WithNoValidateNaNInf.
// NOTE: This alias is kept for backwards compatibility and may be removed
// in a future major release (see CHANGELOG).
func DisableValidateNaNInf() Option { return WithNoValidateNaNInf() }

// WithThreshold sets threshold for ToGraph export.
// Deprecated: Use WithEdgeThreshold.
func WithThreshold(t float64) Option { return WithEdgeThreshold(t) }

// WithBinary forces unit weights on export.
// Deprecated: Use WithBinaryWeights.
func WithBinary() Option { return WithBinaryWeights() }

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
	return gatherOptions(opts...)
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
	o := Options{
		// numeric policy
		eps:               DefaultEpsilon,
		validateNaNInf:    DefaultValidateNaNInf,
		allowInfDistances: DefaultAllowInfDistances,

		// build policy
		directed:    DefaultDirected,
		allowMulti:  DefaultAllowMulti,
		allowLoops:  DefaultAllowLoops,
		weighted:    DefaultWeighted,
		metricClose: DefaultMetricClosure,

		// export policy
		edgeThreshold: DefaultEdgeThreshold,
		keepWeights:   DefaultKeepWeights,
		binaryWeights: DefaultBinaryWeights,
		// undirected inferred in gatherOptions
	}

	finalizeOptions(&o)

	return o
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
	o := Options{
		// numeric policy
		eps:               DefaultEpsilon,
		validateNaNInf:    DefaultValidateNaNInf,
		allowInfDistances: DefaultAllowInfDistances,

		// build policy
		directed:    DefaultDirected,
		allowMulti:  DefaultAllowMulti,
		allowLoops:  DefaultAllowLoops,
		weighted:    DefaultWeighted,
		metricClose: DefaultMetricClosure,

		// export policy
		edgeThreshold: DefaultEdgeThreshold,
		keepWeights:   DefaultKeepWeights,
		binaryWeights: DefaultBinaryWeights,
	}
	for _, set := range user {
		set(&o) // apply in order; last-writer-wins semantics
	}

	finalizeOptions(&o)

	return o
}

// finalizeOptions enforces derived invariants in exactly one place.
// Implementation:
//   - Stage 1: Derive distance-policy requirements (MetricClosure ⇒ allowInfDistances).
//   - Stage 2: Normalize export weight-mode flags into a single consistent state.
//
// Behavior highlights:
//   - Centralized invariant enforcement prevents drift across call sites.
//
// Inputs:
//   - o: pointer to Options to normalize.
//
// Returns:
//   - None (mutates *o).
//
// Errors:
//   - None (invariants are internal; option constructors handle validation/panics).
//
// Determinism:
//   - Deterministic for a fixed o state.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - This function MUST be called after applying all Option setters.
//
// AI-Hints:
//   - If you add a new option that implies others, encode that implication here (single source of truth).
func finalizeOptions(o *Options) {
	// Distance-policy requires +Inf as “no path”.
	if o.metricClose {
		o.allowInfDistances = true
	}
	// If metric closure is requested, the distance policy must allow +Inf sentinels.
	if o.metricClose {
		o.allowInfDistances = true
	}

	// Export weight mode must be internally consistent.
	if o.binaryWeights {
		o.keepWeights = false
	} else if o.keepWeights {
		o.binaryWeights = false
	} else {
		// Defensive normalization: ensure a stable default.
		o.keepWeights = true
		o.binaryWeights = false
	}
}
