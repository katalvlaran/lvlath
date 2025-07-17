// Package ops provides advanced matrix operations for the lvlath/matrix package.
// QR computes the QR decomposition of a square matrix using Householder reflections,
// returning orthogonal Q and upper-triangular R such that m = Q×R.
package ops

import (
	"fmt"
	"math"

	"github.com/katalvlaran/lvlath/matrix"
)

const NormZero = 0.0 // ?

// QR returns Q and R for the decomposition m = Q×R.
// It returns ErrDimensionMismatch if m is not square.
// Complexity: O(n³) time, O(n²) memory where n = m.Rows().
func QR(m matrix.Matrix) (matrix.Matrix, matrix.Matrix, error) {
	// Stage 1: Validate input dimensions
	var (
		rows, cols, n int // matrix dimensions
		err           error
	)
	rows = m.Rows()   // get number of rows
	cols = m.Cols()   // get number of columns
	if rows != cols { // enforce square matrix
		return nil, nil, fmt.Errorf("QR: non-square %dx%d: %w", rows, cols, matrix.ErrMatrixDimensionMismatch)
	}
	n = rows // common dimension

	// Stage 2: Prepare working matrices and Householder vector
	var (
		A matrix.Matrix // working copy of input
		Q matrix.Matrix // orthogonal matrix accumulator
	)
	A = m.Clone()                  // deep copy to preserve original
	Q, err = matrix.NewDense(n, n) // allocate Q as n×n
	if err != nil {
		return nil, nil, fmt.Errorf("QR: %w", err)
	}
	// initialize Q to identity matrix
	{
		var i int
		for i = 0; i < n; i++ {
			_ = Q.Set(i, i, 1.0) // set diagonal to 1
		}
	}
	// allocate Householder vector once
	v := make([]float64, n)

	// Stage 3: Execute Householder reflections
	var (
		k, i, j    int     // loop indices
		sum, alpha float64 // accumulators and reflection scalar
		norm, beta float64 // vector norm and beta = vᵀv
		val        float64 // temporary value holder
		tau        float64 // 2/β factor
	)
	for k = 0; k < n; k++ {
		// 3.1: Compute norm of A[k:n][k]
		norm = NormZero
		for i = k; i < n; i++ {
			val, _ = A.At(i, k) // fetch A[i][k]
			norm += val * val   // accumulate square
		}
		norm = math.Sqrt(norm) // take square root
		if norm == NormZero {
			continue // skip zero column
		}
		// 3.2: Compute reflection scalar alpha = -sign(A[k][k]) * norm
		val, _ = A.At(k, k) // pivot element
		alpha = -math.Copysign(norm, val)
		// 3.3: Build Householder vector v
		for i = 0; i < n; i++ {
			v[i] = NormZero // clear vector entry
		}
		for i = k; i < n; i++ {
			val, _ = A.At(i, k) // fetch A[i][k]
			v[i] = val          // copy into v
		}
		v[k] -= alpha // adjust first component
		// 3.4: Compute beta = vᵀv
		beta = NormZero
		for i = k; i < n; i++ {
			beta += v[i] * v[i]
		}
		tau = 2.0 / beta // compute tau

		// 3.5: Apply reflection to A (update R)
		for j = k; j < n; j++ {
			// compute projection coefficient sum = vᵀ A[:,j]
			sum = NormZero
			for i = k; i < n; i++ {
				val, _ = A.At(i, j)
				sum += v[i] * val
			}
			// A[:,j] -= tau * v * sum
			for i = k; i < n; i++ {
				val, _ = A.At(i, j)
				_ = A.Set(i, j, val-tau*v[i]*sum)
			}
		}

		// 3.6: Apply reflection to Q
		for j = 0; j < n; j++ {
			// compute projection coefficient sum = vᵀ Q[:,j]
			sum = NormZero
			for i = k; i < n; i++ {
				val, _ = Q.At(i, j)
				sum += v[i] * val
			}
			// Q[:,j] -= tau * v * sum
			for i = k; i < n; i++ {
				val, _ = Q.At(i, j)
				_ = Q.Set(i, j, val-tau*v[i]*sum)
			}
		}
	}

	// Stage 4: Finalize and return Q and R (R is the current A)
	R := A
	return Q, R, nil
}
