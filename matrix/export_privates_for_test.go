// SPDX-License-Identifier: MIT
//.go:build test

package matrix

// Test-Bridge (White-Box) for Private Kernels and Options Snapshot
//
// Purpose:
//   - Expose UNEXPORTED ew* micro-kernels and internal options snapshot to matrix_test ONLY.
//   - Enable white-box verification of fast-path (*Dense) vs generic fallback, without widening the prod API.
//
// Build Policy:
//   - Compiles ONLY under `-tags test` via `//go:build test` and `// +build test` directives.
//   - File is in package matrix, so it can access private symbols, but it's invisible in production builds.
//
// Provided Surface:
//   - Ew*_*_TestOnly(...) wrappers: thin pass-through to private ew* kernels.
//   - OptionsSnapshot + NewMatrixOptionsSnapshot_TestOnly / GatherOptionsSnapshot_TestOnly:
//     stable, read-only view of internal Options for tests without using package matrix (non-internal) tests.
//
// Behavior & Determinism:
//   - No allocations beyond what the wrapped functions do.
//   - Deterministic wrappers; no side effects.
//
// Risks & Maintenance:
//   - Keep OptionsSnapshot in sync with internal Options fields. If Options changes,
//     update snapshotOf(...) accordingly (tests will catch drift).
//
// AI-Hints:
//   - Prefer keeping ALL test-only bridges co-located here to avoid clutter across files.
//   - If a private helper changes signature, mirror the change here once, not across many tests.

var (
	// ExportedDenseFill exposes Dense.Fill for white-box tests.
	ExportedDenseFill = (*Dense).Fill
	// ExportedNewDenseWithPolicy exposes newDenseWithPolicy for white-box tests.
	ExportedNewDenseWithPolicy = newDenseWithPolicy

	ExportedValidateTol    = validateTol
	ExportedValidateBounds = validateBounds
)

// Panic message exports to avoid "magic strings" in tests.
const (
	PanicEpsilonInvalid_TestOnly       = panicEpsilonInvalid
	PanicEdgeThresholdInvalid_TestOnly = panicEdgeThresholdInvalid
)

// --- ew* micro-kernel bridges -------------------------------------------------

// EwBroadcastSubCols_TestOnly forwards to the private ewBroadcastSubCols kernel.
// Implementation:
//   - Stage 1: Call the private function directly; return its outputs verbatim.
//
// Behavior highlights:
//   - No production API change; test-only surface.
func EwBroadcastSubCols_TestOnly(X Matrix, colMeans []float64) (Matrix, error) {
	return ewBroadcastSubCols(X, colMeans)
}

// EwBroadcastSubRows_TestOnly forwards to ewBroadcastSubRows.
func EwBroadcastSubRows_TestOnly(X Matrix, rowMeans []float64) (Matrix, error) {
	return ewBroadcastSubRows(X, rowMeans)
}

// EwScaleCols_TestOnly forwards to ewScaleCols.
func EwScaleCols_TestOnly(X Matrix, scale []float64) (Matrix, error) {
	return ewScaleCols(X, scale)
}

// EwScaleRows_TestOnly forwards to ewScaleRows.
func EwScaleRows_TestOnly(X Matrix, scale []float64) (Matrix, error) {
	return ewScaleRows(X, scale)
}

// EwReplaceInfNaN_TestOnly forwards to ewReplaceInfNaN.
func EwReplaceInfNaN_TestOnly(X Matrix, val float64) (Matrix, error) {
	return ewReplaceInfNaN(X, val)
}

// EwClipRange_TestOnly forwards to ewClipRange.
func EwClipRange_TestOnly(X Matrix, lo, hi float64) (Matrix, error) {
	return ewClipRange(X, lo, hi)
}

// EwAllClose_TestOnly forwards to ewAllClose.
func EwAllClose_TestOnly(a, b Matrix, rtol, atol float64) (bool, error) {
	return ewAllClose(a, b, rtol, atol)
}

// --- options snapshot bridge --------------------------------------------------

// OptionsSnapshot is a stable, test-facing copy of internal Options fields.
// Purpose:
//   - Allow matrix_test to assert defaults and "last writer wins" semantics
//     without accessing unexported fields directly.
//
// Determinism:
//   - Pure struct copy; no side effects.
type OptionsSnapshot struct {
	Eps               float64
	ValidateNaNInf    bool
	AllowInfDistances bool

	Directed    bool
	AllowMulti  bool
	AllowLoops  bool
	Weighted    bool
	MetricClose bool

	EdgeThreshold float64
	KeepWeights   bool
	BinaryWeights bool
}

// NewMatrixOptionsSnapshot_TestOnly builds Options via public Option funcs and returns a snapshot.
// Implementation:
//   - Stage 1: o := NewMatrixOptions(opts...)
//   - Stage 2: snapshotOf(o)
func NewMatrixOptionsSnapshot_TestOnly(opts ...Option) OptionsSnapshot {
	o := NewMatrixOptions(opts...)

	return snapshotOf(o)
}

// GatherOptionsSnapshot_TestOnly returns a snapshot after internal derivation.
// Implementation:
//   - Stage 1: o := gatherOptions(opts...) // internal constructor
//   - Stage 2: snapshotOf(o)
//
// Notes:
//   - Keep this wrapper in sync if the internal derivation pipeline changes.
func GatherOptionsSnapshot_TestOnly(opts ...Option) OptionsSnapshot {
	o := gatherOptions(opts...)

	return snapshotOf(o)
}

// snapshotOf copies internal fields to a public struct. Keep in sync with Options layout.
// Behavior highlights:
//   - No allocations besides the snapshot value itself.
func snapshotOf(o Options) OptionsSnapshot {
	return OptionsSnapshot{
		Eps:               o.eps,
		ValidateNaNInf:    o.validateNaNInf,
		AllowInfDistances: o.allowInfDistances,

		Directed:    o.directed,
		AllowMulti:  o.allowMulti,
		AllowLoops:  o.allowLoops,
		Weighted:    o.weighted,
		MetricClose: o.metricClose,

		EdgeThreshold: o.edgeThreshold,
		KeepWeights:   o.keepWeights,
		BinaryWeights: o.binaryWeights,
	}
}
