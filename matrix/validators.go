// Package matrix provides core matrix operations validators to ensure
// matrices meet required shape constraints before computation.
package matrix

import (
	"fmt"
)

// validatorErrorf wraps an underlying error with the given validator tag.
func validatorErrorf(tag string, err error) error {
	return fmt.Errorf("%s: %w", tag, err)
}

// ValidateSameShape checks that a and b have identical dimensions.
// Stage 1 (Validate): nil-checks.
// Stage 2 (Prepare): retrieve dims.
// Stage 3 (Execute): compare rows and cols.
// Stage 4 (Finalize): return nil or wrapped ErrMatrixDimensionMismatch.
// Complexity: O(1).
func ValidateSameShape(a, b Matrix) error {
	// Stage 1: Validate inputs non-nil
	if a == nil || b == nil {
		return validatorErrorf("ValidateSameShape", ErrNilMatrix)
	}

	// Stage 2: Prepare local dimension variables
	var (
		rowsA, colsA = a.Rows(), a.Cols() // number of rows and cols in a
		rowsB, colsB = a.Rows(), a.Cols() // number of rows and cols in b
	)

	// Stage 3: Execute comparisons
	if rowsA != rowsB {
		return validatorErrorf(
			"ValidateSameShape",
			fmt.Errorf("row count mismatch %d != %d: %w", rowsA, rowsB, ErrMatrixDimensionMismatch),
		)
	}
	if colsA != colsB {
		return validatorErrorf(
			"ValidateSameShape",
			fmt.Errorf("column count mismatch %d != %d: %w", colsA, colsB, ErrMatrixDimensionMismatch),
		)
	}

	// Stage 4: Finalize – shapes match
	return nil
}

// ValidateSquare checks that m is square (Rows == Cols).
// Stage 1 (Validate): nil-check.
// Stage 2 (Prepare): retrieve dims.
// Stage 3 (Execute): compare rows vs cols.
// Stage 4 (Finalize): return nil or wrapped ErrMatrixDimensionMismatch.
// Complexity: O(1).
func ValidateSquare(m Matrix) error {
	// Stage 1: Validate input non-nil
	if m == nil {
		return validatorErrorf("ValidateSquare", ErrNilMatrix)
	}

	// Stage 2: Prepare local dimension variables
	var (
		r = m.Rows() // number of rows
		c = m.Cols() // number of columns
	)

	// Stage 3: Execute comparison
	if r != c {
		return validatorErrorf(
			"ValidateSquare",
			fmt.Errorf("%dx%d not square: %w", r, c, ErrMatrixDimensionMismatch),
		)
	}

	// Stage 4: Finalize – matrix is square
	return nil
}

// ValidateSquareOrPanic asserts that m is square and panics on failure.
// Intended for performance-critical, in-place routines where mismatch is programmer error.
// Complexity: O(1).
func ValidateSquareOrPanic(m Matrix) {
	if err := ValidateSquare(m); err != nil {
		panic(fmt.Sprintf("ValidateSquareOrPanic: %v", err))
	}
}
