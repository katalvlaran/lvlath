// Package ops provides advanced matrix operations for the lvlath/matrix package.
// Inverse computes the inverse of a square matrix using LU decomposition
// and forward/backward substitution, following strict fail-fast and Go-idiomatic patterns.
package ops

import (
	"errors"
	"fmt"

	"github.com/katalvlaran/lvlath/matrix"
)

const (
	ZeroSum   = 0.0 // ?
	ZeroPivot = 0.0 // ?
)

// ErrSingular is returned when a zero pivot is encountered during inversion.
var ErrSingular = errors.New("ops: matrix is singular")

// Inverse returns the inverse of the square matrix m, or an error if m is not square or singular.
// Blueprint:
//
//	Stage 1 (Validate): ensure m is square.
//	Stage 2 (Decompose): A = L·U via Doolittle.
//	Stage 3 (Prepare): allocate result matrix and scratch slices.
//	Stage 4 (Execute): for each identity column eᵢ, solve L·y = eᵢ then U·x = y.
//	Stage 5 (Finalize): assemble columns into the inverse and return.
//
// Complexity: O(n³) time, O(n²) memory, where n = m.Rows().
func Inverse(m matrix.Matrix) (matrix.Matrix, error) {
	// Stage 1: Validate input shape
	var (
		rows, cols int   // matrix dimensions
		err        error // general error holder
	)
	rows = m.Rows()   // get number of rows
	cols = m.Cols()   // get number of columns
	if rows != cols { // enforce square matrix
		return nil, fmt.Errorf("Inverse: non-square %dx%d: %w", rows, cols, matrix.ErrMatrixDimensionMismatch)
	}

	// Stage 2: LU decomposition
	var (
		L, U matrix.Matrix // lower and upper triangular matrices
	)
	L, U, err = LU(m) // perform Doolittle LU
	if err != nil {   // fail-fast on decomposition error
		return nil, fmt.Errorf("Inverse: %w", err)
	}

	// Stage 3: Prepare result container and workspaces
	var (
		inv matrix.Matrix // resulting inverse matrix
	)
	inv, err = matrix.NewDense(rows, cols) // allocate Dense(rows×cols)
	if err != nil {                        // fail-fast on allocation error
		return nil, fmt.Errorf("Inverse: %w", err)
	}
	y := make([]float64, rows) // scratch for forward substitution
	x := make([]float64, rows) // scratch for backward substitution

	// Stage 4: Compute each column of the inverse
	var (
		col, i, k  int     // loop indices
		sum, pivot float64 // arithmetic helpers
		aVal       float64 // fetched matrix value
	)
	for col = 0; col < cols; col++ { // for each basis vector e_col
		// Forward substitution: L·y = e_col
		for i = 0; i < rows; i++ {
			sum = ZeroSum           // reset accumulator
			for k = 0; k < i; k++ { // sum L[i][k]*y[k]
				aVal, _ = L.At(i, k) // fetch L[i][k]
				sum += aVal * y[k]   // accumulate
			}
			if i == col { // basis entry
				y[i] = 1.0 - sum // e_col[i] == 1
			} else {
				y[i] = -sum // e_col[i] == 0
			}
		}

		// Backward substitution: U·x = y
		for i = rows - 1; i >= 0; i-- {
			sum = ZeroSum                  // reset accumulator
			for k = i + 1; k < cols; k++ { // sum U[i][k]*x[k]
				aVal, _ = U.At(i, k) // fetch U[i][k]
				sum += aVal * x[k]   // accumulate
			}
			pivot, _ = U.At(i, i)   // fetch diagonal U[i][i]
			if pivot == ZeroPivot { // singular check
				return nil, fmt.Errorf("Inverse: zero pivot at %d: %w", i, ErrSingular)
			}
			x[i] = (y[i] - sum) / pivot // solve for x[i]
		}

		// Write solution x into column col of inv
		for i = 0; i < rows; i++ {
			_ = inv.Set(i, col, x[i]) // assign inv[i][col]
		}
	}

	// Stage 5: Return computed inverse
	return inv, nil
}
