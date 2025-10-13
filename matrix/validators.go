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

// ValidateNotNil ensures a Matrix reference is not nil.
// Returns ErrNilMatrix if m == nil.
// Complexity: O(1).
func ValidateNotNil(m Matrix) error {
	// If the matrix is nil, fail with the unified sentinel.
	if m == nil {
		return validatorErrorf("ValidateNotNil", ErrNilMatrix) // single source of truth for "nil argument"
	}

	// Otherwise accept.
	return nil
}

// validatorErrorf wraps an underlying error with the given validator tag.
func validatorErrorf(tag string, err error) error {
	return fmt.Errorf("%s: %w", tag, err)
}

// ValidateSameShape checks that two matrices have identical shape.
// Stage 1 (Validate): nil-checks via ValidateNotNil.
// Stage 2 (Execute): compare rows and cols.
// Stage 3 (Finalize): return nil or wrapped ErrMatrixDimensionMismatch.
// Complexity: O(1).
func ValidateSameShape(a, b Matrix) error {
	// Disallow nil inputs to avoid ambiguous panics downstream.
	if a == nil || b == nil {
		return validatorErrorf("ValidateSameShape", ErrNilMatrix)
	}

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
// Complexity: O(1).
func ValidateSquare(m Matrix) error {
	// Validate non-nil
	if m == nil {
		return validatorErrorf("ValidateSquare", ErrNilMatrix)
	}

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

// ValidateSymmetric checks A is symmetric within tolerance tol:
// |A[i,j] - A[j,i]| ≤ tol for all i<j.
// Returns ErrNilMatrix/ErrDimensionMismatch on structural issues, ErrNaNInf on bad tol,
// ErrAsymmetry on violation.
// Complexity: O(n^2) where n = Rows(A). Space: O(1).
func ValidateSymmetric(m Matrix, tol float64) error {
	// Guard nil first.
	if m == nil {
		return validatorErrorf("ValidateSymmetric", ErrNilMatrix) // avoid dereferencing nil
	}
	// Require a square matrix for symmetry.
	if err := ValidateSquare(m); err != nil {
		return validatorErrorf("ValidateSymmetric", err) // propagate dimension sentinel
	}
	// Normalize tolerance to a non-negative finite value.
	if math.IsNaN(tol) || math.IsInf(tol, 0) {
		// Use existing numeric sentinel rather than inventing a new one.
		return validatorErrorf("ValidateSymmetric", ErrNaNInf) // invalid tolerance is considered a numeric policy violation
	}
	if tol < 0 {
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
			// If deviation exceeds tolerance, fail immediately — fast negative path.
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
	if tol < 0 {
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
