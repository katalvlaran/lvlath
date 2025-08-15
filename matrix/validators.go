// Package matrix provides core matrix operations validators to ensure
// matrices meet required shape constraints before computation.
package matrix

import (
	"fmt"
)

// ValidateNotNil ensures the Matrix is non-nil.
// Returns ErrNilMatrix if m == nil.
// Complexity: O(1).
func ValidateNotNil(m Matrix) error {
	if m == nil {
		return fmt.Errorf("ValidateNotNil: %w", ErrNilMatrix)
	}
	return nil
}

// validatorErrorf wraps an underlying error with the given validator tag.
func validatorErrorf(tag string, err error) error {
	return fmt.Errorf("%s: %w", tag, err)
}

// ValidateSameShape checks that a and b have identical dimensions.
// Stage 1 (Validate): nil-checks via ValidateNotNil.
// Stage 2 (Prepare): retrieve dims.
// Stage 3 (Execute): compare rows and cols.
// Stage 4 (Finalize): return nil or wrapped ErrMatrixDimensionMismatch.
// Complexity: O(1).
func ValidateSameShape(a, b Matrix) error {
	// Stage 1: Validate non-nil
	if err := ValidateNotNil(a); err != nil {
		return validatorErrorf("ValidateSameShape", err)
	}
	if err := ValidateNotNil(b); err != nil {
		return validatorErrorf("ValidateSameShape", err)
	}

	// Stage 2: Prepare local dimension variables
	rowsA, colsA := a.Rows(), a.Cols() // number of rows and cols in a
	rowsB, colsB := b.Rows(), b.Cols() // number of rows and cols in b

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

	// Stage 4: OK
	return nil
}

// ValidateSquare checks that m is square (Rows == Cols).
// Stage 1 (Validate): nil-check via ValidateNotNil.
// Stage 2 (Prepare): retrieve dims.
// Stage 3 (Execute): compare rows vs cols.
// Stage 4 (Finalize): return nil or wrapped ErrMatrixDimensionMismatch.
// Complexity: O(1).
func ValidateSquare(m Matrix) error {
	// Stage 1: Validate non-nil
	if err := ValidateNotNil(m); err != nil {
		return validatorErrorf("ValidateSquare", err)
	}

	// Stage 2: Prepare local dimension variables
	r, c := m.Rows(), m.Cols()

	// Stage 3: Execute comparison
	if r != c {
		return validatorErrorf(
			"ValidateSquare",
			fmt.Errorf("%dx%d not square: %w", r, c, ErrMatrixDimensionMismatch),
		)
	}

	// Stage 4: OK
	return nil
}
