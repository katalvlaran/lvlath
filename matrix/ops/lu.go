// Package ops provides advanced matrix operations for the lvlath/matrix package.
package ops

import (
	"fmt"

	"github.com/katalvlaran/lvlath/matrix"
)

// LU performs Doolittle LU decomposition on a square matrix m.
// It returns L (unit lower triangular) and U (upper triangular) matrices.
// Returns an error if m is not square (ErrMatrixDimensionMismatch).
// Time Complexity: O(n³), where n = m.Rows(); Memory: O(n²) for L and U.
func LU(m matrix.Matrix) (matrix.Matrix, matrix.Matrix, error) {
	// Stage 1: Validate input is square
	rows, cols := m.Rows(), m.Cols() // get dimensions
	if rows != cols {                // check square
		// wrap sentinel error with context
		return nil, nil, fmt.Errorf("LU: non-square matrix %dx%d: %w", rows, cols, matrix.ErrMatrixDimensionMismatch)
	}
	n := rows // common dimension

	// Stage 2: Prepare L and U matrices
	L, err := matrix.NewDense(n, n) // allocate L
	if err != nil {
		return nil, nil, fmt.Errorf("LU: %w", err)
	}
	U, err := matrix.NewDense(n, n) // allocate U
	if err != nil {
		return nil, nil, fmt.Errorf("LU: %w", err)
	}
	// Initialize L diagonal to 1 (unit lower triangular)
	for i := 0; i < n; i++ {
		_ = L.Set(i, i, 1)
	}

	// Stage 3: Execute decomposition
	var (
		i, j, k    int     // loop indices
		sum        float64 // accumulator for dot products
		lVal, uVal float64
		aVal       float64 // holds A[i][j] or A[j][i]
		uDiag      float64 // diagonal element of U
	)
	// for each pivot row i
	for i = 0; i < n; i++ {
		// Compute U's row i for columns j >= i
		for j = i; j < n; j++ {
			sum = 0                 // reset accumulator
			for k = 0; k < i; k++ { // sum L[i][k]*U[k][j]
				lVal, _ = L.At(i, k)
				uVal, _ = U.At(k, j)
				sum += lVal * uVal // accumulate product
			}
			aVal, _ = m.At(i, j)      // original A[i][j]
			_ = U.Set(i, j, aVal-sum) // set U[i][j]
		}
		// Compute L's column i for rows j > i
		for j = i + 1; j < n; j++ {
			sum = 0                 // reset accumulator
			for k = 0; k < i; k++ { // sum L[j][k]*U[k][i]
				lVal, _ = L.At(j, k)
				uVal, _ = U.At(k, i)
				sum += lVal * uVal // accumulate product
			}
			aVal, _ = m.At(j, i)  // original A[j][i]
			uDiag, _ = U.At(i, i) // U's pivot
			// set L[j][i] = (A[j][i] - sum) / U[i][i]
			_ = L.Set(j, i, (aVal-sum)/uDiag)
		}
	}

	// Stage 4: Finalize and return
	return L, U, nil
}
