// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

//.go:build test

package matrix

// Test-Bridge (White-Box) for Private Kernels and Options Snapshot.
//
// Purpose:
//   - Expose selected unexported kernels, helpers, and internal Options state to tests.
//   - Enable white-box verification of Dense fast-paths, generic fallbacks,
//     option finalization, zero-shape law, and numeric policy without widening
//     the production API.
//
// Build Policy:
//   - Compiles only when tests are run with `-tags test`.
//   - This file is intentionally test-only and must never become part of the
//     production build.
//
// Provided Surface:
//   - Ew*_*_TestOnly wrappers: thin pass-through to private ew* kernels.
//   - OptionsSnapshot: stable read-only view of internal Options.
//   - Selected helper exports for focused package-level tests.
//
// Behavior & Determinism:
//   - Wrappers do not add logic, allocation, mutation, or alternative policy.
//   - All behavior remains owned by the wrapped production functions.
//
// Risks & Maintenance:
//   - Keep OptionsSnapshot in sync with Options.
//   - If Options gains a field that affects behavior, snapshotOf must expose it
//     unless there is a deliberate reason not to test it.
//
// AI-Hints:
//   - Keep all test-only bridges co-located here.
//   - Do not add production-only conveniences to this file.
//   - Do not hide errors in snapshot constructors; tests must see option failures.
//   - If a private helper changes signature, mirror the change here once, not across many tests.

var (
	// ExportedDenseFill exposes Dense.Fill for white-box tests.
	//
	// Important:
	//   - Fill is policy-respecting. It does NOT bypass Dense numeric validation.
	//   - For dirty NaN/Inf fixtures, allocate Dense with validation disabled.
	ExportedDenseFill = (*Dense).Fill

	// ExportedNewDenseWithPolicy exposes newDenseWithPolicy for tests that need
	// exact Dense numeric-policy construction.
	ExportedNewDenseWithPolicy = newDenseWithPolicy

	// ExportedValidateTol exposes validateTol for boundary tests.
	ExportedValidateTol = validateTol

	// ExportedValidateBounds exposes validateBounds for Clip/sanitizer tests.
	ExportedValidateBounds = validateBounds

	// ExportedRowNormScale exposes the shared row-normalization scale policy.
	// It is useful for P5 tests that assert zero-norm and non-finite norm behavior.
	ExportedRowNormScale = rowNormScale
)

// --- ew* micro-kernel bridges -------------------------------------------------

// EwBroadcastSubCols_TestOnly forwards to the private ewBroadcastSubCols kernel.
//
// Behavior highlights:
//   - No policy change.
//   - No production API change.
//   - Useful for comparing Dense fast-path against generic fallback.
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
//
// Purpose:
//   - Allow external tests to assert defaults, finalization, and last-writer-wins
//     semantics without accessing unexported Options fields directly.
//
// Maintenance law:
//   - If an Options field influences runtime behavior, expose it here unless
//     there is a deliberate reason not to white-box test it.
//
// Determinism:
//   - Pure struct copy; no side effects.
type OptionsSnapshot struct {
	// Numeric policy.
	Eps               float64
	ValidateNaNInf    bool
	AllowInfDistances bool

	// Graph/matrix build policy.
	Directed    bool
	AllowMulti  bool
	AllowLoops  bool
	Weighted    bool
	MetricClose bool

	// Zero-weight preservation policy.
	PreserveZeroWeights bool

	// Export policy.
	EdgeThreshold    float64
	EdgeThresholdSet bool
	KeepWeights      bool
	BinaryWeights    bool
}

// NewMatrixOptionsSnapshot_TestOnly builds Options through the public resolver
// and returns a snapshot.
//
// Behavior highlights:
//   - Uses NewMatrixOptions, not manual field construction.
//   - Returns errors explicitly so tests can assert invalid-option behavior.
//
// AI-Hints:
//   - Do not collapse this back to a panic or zero snapshot on error.
func NewMatrixOptionsSnapshot_TestOnly(opts ...Option) (OptionsSnapshot, error) {
	o, err := NewMatrixOptions(opts...)
	if err != nil {
		return OptionsSnapshot{}, err
	}

	return snapshotOf(o), nil
}

// GatherOptionsSnapshot_TestOnly returns a snapshot after internal option derivation.
//
// Purpose:
//   - Allows tests to verify internal gatherOptions/finalizeOptions behavior directly.
//   - This is useful for defaults and derived invariants such as:
//     MetricClosure => AllowInfDistances;
//     BinaryWeights => !KeepWeights;
//     !Weighted     => !PreserveZeroWeights.
//
// AI-Hints:
//   - Keep this wrapper thin.
//   - Do not swallow errors.
func GatherOptionsSnapshot_TestOnly(opts ...Option) (OptionsSnapshot, error) {
	o, err := gatherOptions(opts...)
	if err != nil {
		return OptionsSnapshot{}, err
	}

	return snapshotOf(o), nil
}

// snapshotOf copies internal Options fields to a public test-facing struct.
//
// Behavior highlights:
//   - No allocation beyond returned value.
//   - No finalization; caller must pass already-finalized Options when needed.
//
// AI-Hints:
//   - Keep in sync with Options.
//   - Missing behavior-bearing fields here should be treated as test coverage debt.
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

		PreserveZeroWeights: o.preserveZeroWeights,

		EdgeThreshold:    o.edgeThreshold,
		EdgeThresholdSet: o.edgeThresholdSet,
		KeepWeights:      o.keepWeights,
		BinaryWeights:    o.binaryWeights,
	}
}
