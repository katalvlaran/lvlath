// SPDX-License-Identifier: MIT
// Package: matrix
//
// Purpose:
//   - Provide a single, canonical source of truth for common validation checks.
//   - Keep kernels and public facades small by delegating guard logic here.
//   - Return sentinel errors directly so call sites can wrap uniformly (if desired).
//
// Determinism & Performance:
//   - All validators are deterministic and side-effect free.
//   - Structural checks are O(1); value scans are O(r*c) with fixed traversal order.
//
// AI-Hints:
//   - Use ValidateBinarySameShape before element-wise kernels (Add/Sub/Hadamard).
//   - Use ValidateMulCompatible before MatMul to fail fast on dimension mismatch.
//   - Keep ValidateGraphAdjacency structural; apply ValidateAllFinite separately when needed.

package matrix

import (
	"math"

	"github.com/katalvlaran/lvlath/core"
)

// zeroTol is a tiny tolerance used only internally for guards where appropriate.
// We keep it explicit to avoid "magic numbers" inline.
const zeroTol = 0.0

// isNonFinite REPORTS whether v is NaN or ±Inf.
//
// Implementation:
//   - Stage 1: math.IsNaN(v).
//   - Stage 2: math.IsInf(v, 0).
//
// Behavior highlights:
//   - Pure predicate with IEEE-754 semantics.
//
// Inputs:
//   - v: scalar value.
//
// Returns:
//   - bool: true if v is not a finite real number.
//
// Errors:
//   - None.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Treats both +Inf and -Inf as non-finite.
//
// AI-Hints:
//   - Use this to guard tolerances/bounds before starting an O(n²) scan.
func isNonFinite(v float64) bool {
	return math.IsNaN(v) || math.IsInf(v, 0)
}

// isNaNOrNegInf REPORTS whether v is NaN or -Inf (allows +Inf as "no path").
func isNaNOrNegInf(v float64) bool {
	return math.IsNaN(v) || math.IsInf(v, -1)
}

// isNaNOrPosInf REPORTS whether v is NaN or +Inf (allows -Inf as "no path").
func isNaNOrPosInf(v float64) bool {
	return math.IsNaN(v) || math.IsInf(v, 1)
}

// validateTol NORMALIZES a tolerance to a non-negative finite value.
//
// Implementation:
//   - Stage 1: Reject NaN/±Inf → ErrNaNInf.
//   - Stage 2: If negative, flip sign.
//
// Behavior highlights:
//   - Accepts negative inputs and converts to |tol| to avoid surprising callers.
//
// Inputs:
//   - tol: tolerance value (finite).
//
// Returns:
//   - float64: normalized tol >= 0.
//
// Errors:
//   - ErrNaNInf if tol is NaN/±Inf.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - A tol of 0 enforces exact comparisons.
//
// AI-Hints:
//   - Use validateTol for rtol/atol to make “sign mistakes” non-fatal and predictable.
func validateTol(tol float64) (float64, error) {
	if isNonFinite(tol) {
		return 0, ErrNaNInf
	}
	if tol < 0 {
		tol = -tol
	}

	return tol, nil
}

// validateBounds VALIDATES and NORMALIZES numeric bounds (lo, hi).
//
// Implementation:
//   - Stage 1: Reject NaN/±Inf in either bound → ErrNaNInf.
//   - Stage 2: If lo > hi, swap.
//
// Behavior highlights:
//   - Makes inverted bounds deterministic and non-fatal.
//
// Inputs:
//   - lo, hi: bound endpoints (finite).
//
// Returns:
//   - (lo, hi): normalized so lo <= hi.
//
// Errors:
//   - ErrNaNInf if either bound is NaN/±Inf.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - This is a normalization helper; it does not clamp values by itself.
//
// AI-Hints:
//   - Use this before Clip/threshold logic to avoid writing “lo/hi swap” in every call site.
func validateBounds(lo, hi float64) (float64, float64, error) {
	if isNonFinite(lo) || isNonFinite(hi) {
		return 0, 0, ErrNaNInf
	}
	if lo > hi {
		lo, hi = hi, lo
	}

	return lo, hi, nil
}

// ValidateNotNil ENSURES the matrix reference is non-nil.
//
// Implementation:
//   - Stage 1: Check interface value against nil.
//   - Stage 2 (optional): If m implements core.Nilable, consult m.IsNil() to detect typed-nil.
//
// Behavior highlights:
//   - Canonical nil-guard for all composite validators.
//
// Inputs:
//   - m: matrix interface value.
//
// Returns:
//   - nil on success.
//
// Errors:
//   - ErrNilMatrix if m == nil.
//   - ErrNilMatrix if m is a typed-nil inside interface AND implements core.Nilable.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - This validates the interface value; it does not validate internal storage.
//
// AI-Hints:
//   - Call this first in any exported operation that dereferences m.
func ValidateNotNil(m Matrix) error {
	// If the matrix is nil, fail with the unified sentinel.
	if m == nil {
		return ErrNilMatrix
	}

	// Optional typed-nil detection without reflect:
	// if the implementation provides core.Nilable, trust its IsNil().
	if n, ok := m.(core.Nilable); ok && n.IsNil() {
		return ErrNilMatrix
	}

	return nil
}

// ValidateSameShape ENSURES matrices a and b have equal dimensions.
//
// Implementation:
//   - Stage 1: Compare Rows.
//   - Stage 2: Compare Cols.
//
// Behavior highlights:
//   - Pure structural check; does not inspect values.
//
// Inputs:
//   - a,b: matrices (must be non-nil).
//
// Returns:
//   - nil on success.
//
// Errors:
//   - ErrDimensionMismatch if shapes differ.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Prefer ValidateBinarySameShape when you also need nil checks.
//
// AI-Hints:
//   - Use this before allocating output buffers to avoid wasted work.
func ValidateSameShape(a, b Matrix) error {
	if a.Rows() != b.Rows() || a.Cols() != b.Cols() {
		return ErrDimensionMismatch
	}

	return nil
}

// ValidateSquare ENSURES m is square (Rows == Cols).
//
// Implementation:
//   - Stage 1: Reject nil.
//   - Stage 2: Compare Rows and Cols.
//
// Behavior highlights:
//   - Structural precondition for spectral / factorization methods.
//
// Inputs:
//   - m: matrix.
//
// Returns:
//   - nil on success.
//
// Errors:
//   - ErrNilMatrix if m is nil.
//   - ErrNonSquare if Rows != Cols.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Use ValidateSquareNonNil when you want an explicit composite name.
//
// AI-Hints:
//   - Apply this before symmetry checks to avoid ambiguous “triangle scans” on non-square matrices.
func ValidateSquare(m Matrix) error {
	if err := ValidateNotNil(m); err != nil {
		return err
	}
	// Check the square condition explicitly.
	if m.Rows() != m.Cols() {
		return ErrNonSquare
		// return ErrDimensionMismatch
	}

	return nil
}

// ValidateVecLen ENSURES vector length matches n.
//
// Implementation:
//   - Stage 1: Reject nil slice.
//   - Stage 2: Check len(x) == n.
//
// Behavior highlights:
//   - Canonical guard for MatVec-like operations.
//
// Inputs:
//   - x: vector slice.
//   - n: expected length.
//
// Returns:
//   - nil on success.
//
// Errors:
//   - ErrNilMatrix if x is nil (generic nil-argument sentinel).
//   - ErrDimensionMismatch if len(x) != n.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Uses ErrNilMatrix as a generic nil-argument sentinel.
//
// AI-Hints:
//   - If you need “nil allowed means zero-vector”, do not use this validator.
func ValidateVecLen(x []float64, n int) error {
	// Disallow nil vectors to avoid subtle bugs in MatVec-like routines.
	if x == nil {
		return ErrNilMatrix
	}
	// Check the exact expected length.
	if len(x) != n {
		return ErrDimensionMismatch
	}

	return nil
}

// ValidateBinarySameShape COMPOSES NotNil(a) → NotNil(b) → SameShape.
//
// Implementation:
//   - Stage 1: ValidateNotNil(a).
//   - Stage 2: ValidateNotNil(b).
//   - Stage 3: ValidateSameShape(a,b).
//
// Behavior highlights:
//   - Standard guard sequence for element-wise operations.
//
// Inputs:
//   - a,b: matrices.
//
// Returns:
//   - nil on success.
//
// Errors:
//   - ErrNilMatrix if any input is nil.
//   - ErrDimensionMismatch if shapes differ.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Keeps wrapping out of validators; callers can wrap with operation context.
//
// AI-Hints:
//   - Use this at public API boundaries; internal kernels may assume non-nil.
func ValidateBinarySameShape(a, b Matrix) error {
	if err := ValidateNotNil(a); err != nil {
		return err
	}
	if err := ValidateNotNil(b); err != nil {
		return err
	}

	return ValidateSameShape(a, b)
}

// ValidateSquareNonNil COMPOSES NotNil → Square.
//
// Implementation:
//   - Stage 1: ValidateNotNil.
//   - Stage 2: ValidateSquare.
//
// Behavior highlights:
//   - Named composite to clarify intent at call sites.
//
// Inputs:
//   - m: matrix.
//
// Returns:
//   - nil on success.
//
// Errors:
//   - ErrNilMatrix, ErrNonSquare.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Equivalent to ValidateSquare but kept for readability in composites.
//
// AI-Hints:
//   - Prefer this in exported graph/spectral APIs where “square required” is part of the contract.
func ValidateSquareNonNil(m Matrix) error {
	return ValidateSquare(m)
}

// ValidateMulCompatible ENSURES a.Cols == b.Rows, inputs non-nil.
//
// Implementation:
//   - Stage 1: ValidateNotNil(a), ValidateNotNil(b).
//   - Stage 2: Compare a.Cols() and b.Rows().
//
// Behavior highlights:
//   - Canonical dimension guard for MatMul.
//
// Inputs:
//   - a,b: matrices.
//
// Returns:
//   - nil on success.
//
// Errors:
//   - ErrNilMatrix if any input is nil.
//   - ErrDimensionMismatch if a.Cols != b.Rows.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - This does not check overflow/NaN behavior in multiplication.
//
// AI-Hints:
//   - ValidateMulCompatible early to avoid allocating output on invalid shapes.
func ValidateMulCompatible(a, b Matrix) error {
	if err := ValidateNotNil(a); err != nil {
		return err
	}
	if err := ValidateNotNil(b); err != nil {
		return err
	}
	if a.Cols() != b.Rows() {
		return ErrDimensionMismatch
	}

	return nil
}

// ValidateSymmetric ENSURES A is symmetric within tolerance tol:
// |A[i,j] - A[j,i]| <= tol for all i<j.
//
// Implementation:
//   - Stage 1: ValidateSquare (includes nil guard).
//   - Stage 2: Normalize tol via validateTol.
//   - Stage 3: Scan upper triangle; reject non-finite entries; compare deviations.
//
// Behavior highlights:
//   - Deterministic traversal order and early-exit on first violation.
//
// Inputs:
//   - m: square matrix.
//   - tol: finite tolerance (negative allowed; treated as |tol|).
//
// Returns:
//   - nil on success.
//
// Errors:
//   - ErrNilMatrix, ErrNonSquare from structure checks..
//   - ErrNaNInf if tol is NaN/±Inf or if any compared entry is non-finite.
//   - ErrAsymmetry if symmetry violation exceeds tol.
//
// Determinism:
//   - Deterministic (fixed i→j scan).
//
// Complexity:
//   - Time O(n²), Space O(1).
//
// Notes:
//   - Non-finite entries make symmetry comparisons ill-defined and are rejected.
//
// AI-Hints:
//   - Use tol=0 for strict symmetry on exact constructions (identity, Laplacians with exact weights).
func ValidateSymmetric(m Matrix, tol float64) error {
	// Guard nil first.
	if err := ValidateSquare(m); err != nil {
		return err
	}
	// Normalize tolerance to a non-negative finite value.
	t, err := validateTol(tol)
	if err != nil {
		return err
	}

	n := m.Rows()
	if n <= 1 {
		return nil
	}

	// Scan the strict upper triangle once, tracking the maximum deviation.
	// Deterministic i→j order ensures reproducible short-circuiting behavior.
	var i, j int         // loop counters
	var aij, aji float64 // A[i,j] and A[j,i]
	for i = 0; i < n; i++ {
		for j = i + 1; j < n; j++ {
			aij, err = m.At(i, j)
			if err != nil {
				return err
			}
			aji, err = m.At(j, i)
			if err != nil {
				return err
			}

			if isNonFinite(aij) || isNonFinite(aji) {
				return ErrNaNInf
			}

			if math.Abs(aij-aji) > t {
				return ErrAsymmetry
			}
		}
	}

	// At this point, all |A[i,j]-A[j,i]| ≤ tol, so A is symmetric within tol.
	// Callers (e.g., Eigen) can treat (maxOff == 0) as a "diagonal already" shortcut.
	return nil
}

// IsZeroOffDiagonal REPORTS whether max_{i!=j} |A[i,j]| <= tol.
//
// Implementation:
//   - Stage 1: ValidateSquare (includes nil guard).
//   - Stage 2: Normalize tol via validateTol.
//   - Stage 3: Scan all off-diagonal entries; reject non-finite; compare |v| to tol.
//
// Behavior highlights:
//   - Fast early-exit when a single entry exceeds tolerance.
//
// Inputs:
//   - m: square matrix.
//   - tol: finite tolerance (negative allowed; treated as |tol|).
//
// Returns:
//   - bool: true if all off-diagonal entries are within tol.
//
// Errors:
//   - ErrNilMatrix, ErrNonSquare from structure checks.
//   - ErrNaNInf if tol is NaN/±Inf or if any inspected entry is non-finite.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(n²), Space O(1).
//
// Notes:
//   - This is a numeric predicate; it does not mutate the matrix.
//
// AI-Hints:
//   - Use this to short-circuit diagonal-only fast paths (Jacobi, repeated deflation loops).
func IsZeroOffDiagonal(m Matrix, tol float64) (bool, error) {
	if err := ValidateSquare(m); err != nil {
		return false, err
	}

	t, err := validateTol(tol)
	if err != nil {
		return false, err
	}

	n := m.Rows()
	if n <= 1 {
		return true, nil
	}

	var i, j int
	var v float64
	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			if i == j {
				continue
			}
			v, err = m.At(i, j)
			if err != nil {
				return false, err
			}
			if isNonFinite(v) {
				return false, ErrNaNInf
			}
			if math.Abs(v) > t {
				return false, nil
			}
		}
	}

	return true, nil
}

// ValidateGraphAdjacency VALIDATES an adjacency-matrix wrapper structurally.
//
// Implementation:
//   - Stage 1: Reject nil wrapper or nil Mat.
//   - Stage 2: ValidateSquare(Mat).
//   - Stage 3: If VertexIndex is present, ensure its size matches N and indices are in [0,N).
//
// Behavior highlights:
//   - Structural-only: does not inspect matrix values (finite-ness is a separate concern).
//
// Inputs:
//   - am: adjacency matrix wrapper.
//
// Returns:
//   - nil on success.
//
// Errors:
//   - ErrNonSquare if Mat is not square.
//   - ErrDimensionMismatch if index metadata contradicts dimensions.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1) to O(|V|) depending on index metadata scan, Space O(1).
//
// Notes:
//   - Value validation (NaN/Inf) must be done explicitly via ValidateAllFinite if required.
//
// AI-Hints:
//   - Keep this check at graph API boundaries; do not pay O(n²) scans unless the algorithm needs them.
func ValidateGraphAdjacency(am *AdjacencyMatrix) error {
	if am == nil || am.Mat == nil {
		return ErrNilMatrix
	}
	if err := ValidateSquare(am.Mat); err != nil {
		return err
	}

	// If the type exposes VertexIndex, validate its basic consistency without allocating.
	if am.VertexIndex != nil {
		n := am.Mat.Rows()
		if len(am.VertexIndex) != n {
			return ErrDimensionMismatch
		}
		for _, idx := range am.VertexIndex {
			if idx < 0 || idx >= n {
				return ErrDimensionMismatch
			}
		}
	}

	return nil
}

// ValidateAllFinite ENSURES every entry of m is finite.
//
// Implementation:
//   - Stage 1: Reject nil.
//   - Stage 2: Scan all entries in deterministic row-major order.
//
// Behavior highlights:
//   - Explicit value-level validator (opt-in).
//
// Inputs:
//   - m: matrix.
//
// Returns:
//   - nil on success.
//
// Errors:
//   - ErrNilMatrix if m is nil.
//   - ErrNaNInf if any entry is NaN/±Inf.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(r*c), Space O(1).
//
// Notes:
//   - Use only when the algorithm cannot tolerate non-finite values.
//
// AI-Hints:
//   - Apply after raw ingest (Fill) when you want to enforce “finite-only” preconditions.
func ValidateAllFinite(m Matrix) error {
	err := ValidateNotNil(m)
	if err != nil {
		return err
	}

	r, c := m.Rows(), m.Cols()
	var i, j int
	var v float64
	for i = 0; i < r; i++ {
		for j = 0; j < c; j++ {
			v, err = m.At(i, j)
			if err != nil {
				return err
			}
			if isNonFinite(v) {
				return ErrNaNInf
			}
		}
	}

	return nil
}
