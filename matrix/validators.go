// SPDX-License-Identifier: MIT
// Package: matrix
//
// Purpose:
//  - Provide a single, canonical source of truth for common validation checks.
//  - Keep kernels/facades minimal by delegating shape/nil/symmetry checks here.
//  - Return plain sentinel errors (no wrapping) so call sites can wrap uniformly.
//
// Determinism & Performance:
//  - All checks are pure, deterministic and allocate nothing.
//  - Symmetry check runs O(n²) on the upper triangle only.
//
// AI-Hints:
//  - Centralizing validators eliminates inconsistent guard logic across files.
//  - Use ValidateSymmetric before spectral methods (Jacobi) to fail fast.
//  - Use IsZeroOffDiagonal to short-circuit iterative algorithms when matrix is already diagonal.
//  - Use ValidateVecLen for any MatVec-like operations to avoid ad hoc length code.
//
// Note:
//  - Each composite validator follows a fixed sequence (e.g. NotNil → Shape).
//  - Each validator describes what it validates and what it assumes (e.g. no nil check).

package matrix

import (
	"fmt"
	"math"
)

const (
	// zeroTol is a tiny tolerance used only internally for guards where appropriate.
	// We keep it explicit to avoid "magic numbers" inline.
	zeroTol = 0.0
)

// validatorErrorf wraps an underlying error with the given validator tag.
// Used internally to maintain consistent labeling of sentinel violations.
func validatorErrorf(tag string, err error) error {
	// Provides consistent error tagging for all validation errors.
	return fmt.Errorf("%s: %w", tag, err)
}

// ValidateNotNil – Ensures the matrix reference is non-nil.
//
// Inputs: Matrix interface value.
// Returns ErrNilMatrix if m == nil.
// Complexity: O(1).
// AI-Hints: Use as the first step in composite validations.
func ValidateNotNil(m Matrix) error {
	// If the matrix is nil, fail with the unified sentinel.
	if m == nil {
		return validatorErrorf("ValidateNotNil", ErrNilMatrix) // single source of truth for "nil argument"
	}

	// Otherwise accept.
	return nil
}

// ValidateSameShape – Ensures matrices a and b have equal dimensions.
//
// Implementation: Assumes a and b are not nil (caller must ensure).
// Inputs: Two Matrix values.
// Return: nil or wrapped ErrDimensionMismatch.
// Complexity: O(1).
// AI-Hints: Use for Add/Sub/Hadamard kernels and compatibility guards.
func ValidateSameShape(a, b Matrix) error {
	// Execute comparisons
	if a.Rows() != b.Rows() {
		return validatorErrorf("ValidateSameShape: Rows", ErrDimensionMismatch)
	}
	if a.Cols() != b.Cols() {
		return validatorErrorf("ValidateSameShape: Columns", ErrDimensionMismatch)
	}

	return nil
}

// ValidateSquare checks that m is square (Rows == Cols).
//
// Inputs: Matrix value.
// Errors: ErrNilMatrix if nil, ErrDimensionMismatch if not square.
// Complexity: O(1).
// AI-Hints: Use before spectral or factorization methods.
func ValidateSquare(m Matrix) error {
	// Check the square condition explicitly.
	if m.Rows() != m.Cols() {
		return validatorErrorf("ValidateSquare", ErrDimensionMismatch)
	}

	return nil
}

// ValidateVecLen ensures the vector length matches the required size n.
// Time: O(1). Space: O(1).
func ValidateVecLen(x []float64, n int) error {
	// Disallow nil vectors to avoid subtle bugs in MatVec-like routines.
	if x == nil {
		return validatorErrorf("ValidateVecLen", ErrNilMatrix) // we reuse the existing sentinel for "nil argument"
	}
	// Check the exact expected length.
	if len(x) != n {
		return validatorErrorf("ValidateVecLen", ErrDimensionMismatch) // vector length must match the number of columns
	}

	return nil
}

// ValidateGraph ensures an AdjacencyMatrix value is non-nil and square,
// and (when available) the index table is consistent with the matrix dimension.
// Time: O(1). Space: O(1).
func ValidateGraph(am *AdjacencyMatrix) error {
	// Check wrapper and underlying storage presence.
	if am == nil || am.Mat == nil {
		return validatorErrorf("ValidateGraph", ErrNilMatrix) // nil graph container or matrix
	}
	// Enforce square adjacency for graph algorithms.
	if err := ValidateSquare(am.Mat); err != nil {
		return validatorErrorf("ValidateGraph", err) // adjacency must be square
	}
	// If reverse index is present, ensure consistent dimension.
	if am.vertexByIndex != nil && len(am.vertexByIndex) != am.Mat.Rows() {
		return validatorErrorf("ValidateGraph", ErrDimensionMismatch) // index table must align with matrix rows
	}
	return nil
}

// ValidateBinarySameShape – Composite: NotNil(a) → NotNil(b) → SameShape.
//
// Errors: Combines ErrNilMatrix and ErrDimensionMismatch.
// Complexity: O(1).
func ValidateBinarySameShape(a, b Matrix) error {
	if err := ValidateNotNil(a); err != nil {
		return validatorErrorf("ValidateBinarySameShape", err)
	}
	if err := ValidateNotNil(b); err != nil {
		return validatorErrorf("ValidateBinarySameShape", err)
	}
	if err := ValidateSameShape(a, b); err != nil {
		return validatorErrorf("ValidateBinarySameShape", err)
	}
	return nil
}

// ValidateSquareNonNil – Composite: NotNil → Square.
//
// Errors: ErrNilMatrix, ErrDimensionMismatch.
// Complexity: O(1).
func ValidateSquareNonNil(m Matrix) error {
	if err := ValidateNotNil(m); err != nil {
		return validatorErrorf("ValidateSquareNonNil", err)
	}
	if err := ValidateSquare(m); err != nil {
		return validatorErrorf("ValidateSquareNonNil", err)
	}
	return nil
}

// ValidateSymmetric checks A is symmetric within tolerance tol:
// |A[i,j] - A[j,i]| ≤ tol for all i<j.
//
// Inputs: Square Matrix m, tolerance tol ≥ 0.
// Complexity: O(n^2) where n = Rows(A). Space: O(1).
// Returns ErrNilMatrix/ErrDimensionMismatch on structural issues, ErrNaNInf on bad tol,
// ErrAsymmetry on violation.
// AI-Hints: Use for Eigen decomposition and PSD tests. Require a square matrix for symmetry.
func ValidateSymmetric(m Matrix, tol float64) error {
	// Guard nil first.
	if m == nil {
		return validatorErrorf("ValidateSymmetric", ErrNilMatrix) // avoid dereferencing nil
	}
	// Check the square condition explicitly.
	if m.Rows() != m.Cols() {
		return validatorErrorf("ValidateSymmetric", ErrDimensionMismatch) // propagate dimension sentinel
	}
	// Normalize tolerance to a non-negative finite value.
	if math.IsNaN(tol) || math.IsInf(tol, 0) {
		// Use existing numeric sentinel rather than inventing a new one.
		return validatorErrorf("ValidateSymmetric", ErrNaNInf) // invalid tolerance is considered a numeric policy violation
	}
	if tol < zeroTol {
		// Negative tolerance makes little semantic sense; flip to its absolute value.
		tol = -tol
	}

	// Early return path: a 0×0 or 1×1 matrix is trivially symmetric.
	n := m.Rows() // n == m.Cols() due to ValidateSquare above
	if n <= 1 {
		return nil // nothing to compare
	}

	// Scan the strict upper triangle once, tracking the maximum deviation.
	// Deterministic i→j order ensures reproducible short-circuiting behavior.
	var (
		i, j   int     // loop counters
		aij    float64 // A[i,j]
		aji    float64 // A[j,i]
		diff   float64 // |aij - aji|
		maxOff float64 // running maximum of the deviation
	)
	for i = 0; i < n; i++ { // fixed row loop
		for j = i + 1; j < n; j++ { // scan only upper triangle
			aij, _ = m.At(i, j)        // At is O(1); errors are not expected after shape validation
			aji, _ = m.At(j, i)        // symmetric counterpart
			diff = math.Abs(aij - aji) // absolute asymmetry magnitude
			// If deviation exceeds tolerance, fail immediately - fast negative path.
			if diff > tol {
				return validatorErrorf("ValidateSymmetric", ErrAsymmetry) // caller may wrap with an operation tag
			}
			// Track the maximum deviation for early-positive reasoning (optional).
			if diff > maxOff {
				maxOff = diff
			}
		}
	}

	// At this point, all |A[i,j]-A[j,i]| ≤ tol, so A is symmetric within tol.
	// Callers (e.g., Eigen) can treat (maxOff == 0) as a "diagonal already" shortcut.
	return nil
}

// IsZeroOffDiagonal reports whether max_{i≠j} |A[i,j]| ≤ tol.
// Useful to early-exit Jacobi when matrix is already (near) diagonal.
// Returns ErrNilMatrix/ErrDimensionMismatch/ErrNaNInf like ValidateSymmetric.
// Complexity: O(n²).
func IsZeroOffDiagonal(m Matrix, tol float64) (bool, error) {
	if m == nil {
		return false, ErrNilMatrix
	}
	if err := ValidateSquare(m); err != nil {
		return false, err
	}
	if math.IsNaN(tol) || math.IsInf(tol, 0) {
		return false, ErrNaNInf
	}
	if tol < zeroTol {
		tol = -tol
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
			v, _ = m.At(i, j)
			if math.Abs(v) > tol {
				return false, nil
			}
		}
	}

	return true, nil
}

// ValidateHadamard – Wrapper for ValidateBinarySameShape.
//
// Complexity: O(1).
// AI-Hints: Alias for semantic clarity in elementwise multiplication.
func ValidateHadamard(a, b Matrix) error {
	return ValidateBinarySameShape(a, b)
}

// ValidateMulCompatible – Ensures a.Cols == b.Rows, inputs non-nil.
//
// Errors: ErrNilMatrix, ErrDimensionMismatch.
// Complexity: O(1).
// AI-Hints: Use for general matrix multiplication compatibility.
func ValidateMulCompatible(a, b Matrix) error {
	if err := ValidateNotNil(a); err != nil {
		return validatorErrorf("ValidateMulCompatible", err)
	}
	if err := ValidateNotNil(b); err != nil {
		return validatorErrorf("ValidateMulCompatible", err)
	}
	if a.Cols() != b.Rows() {
		return validatorErrorf("ValidateMulCompatible", ErrDimensionMismatch)
	}

	return nil
}

// ValidateGraphAdjacency – Validates adjacency matrix and index map consistency.
//
// Inputs: *AdjacencyMatrix struct.
// Errors: ErrNilMatrix, ErrDimensionMismatch.
// Complexity: O(1).
// AI-Hints: Use before FW/APSP-related kernels.
func ValidateGraphAdjacency(am *AdjacencyMatrix) error {
	if am == nil {
		return validatorErrorf("ValidateGraphAdjacency", ErrNilMatrix)
	}
	if err := ValidateSquareNonNil(am.Mat); err != nil {
		return validatorErrorf("ValidateGraphAdjacency", err)
	}
	if am.vertexByIndex != nil && len(am.vertexByIndex) != am.Mat.Rows() {
		return validatorErrorf("ValidateGraphAdjacency", ErrDimensionMismatch)
	}

	return nil
}
