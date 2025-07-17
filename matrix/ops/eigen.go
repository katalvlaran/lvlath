// Package ops provides advanced matrix operations for the lvlath/matrix package.
// Eigen computes all eigenvalues and eigenvectors of a real symmetric matrix
// using the Jacobi rotation method.
package ops

import (
	"errors"
	"fmt"
	"math"

	"github.com/katalvlaran/lvlath/matrix"
)

// ErrNotSymmetric is returned when the input matrix is not symmetric.
var ErrNotSymmetric = errors.New("ops: matrix is not symmetric")

// ErrEigenFailed is returned if the algorithm does not converge within max iterations.
var ErrEigenFailed = errors.New("ops: eigen decomposition did not converge")

// Eigen performs Jacobi eigenvalue decomposition on a symmetric matrix m.
// It returns a slice of eigenvalues and a matrix of eigenvectors Q (columns of Q).
// tol specifies convergence threshold for off-diagonal elements.
// maxIter caps the number of sweeps.
// Returns ErrMatrixDimensionMismatch, ErrNotSymmetric, or ErrEigenFailed.
// Complexity: O(n³) time per sweep, worst-case O(maxIter·n³); Memory: O(n²).
func Eigen(m matrix.Matrix, tol float64, maxIter int) ([]float64, matrix.Matrix, error) {
	// Stage 1: Validate input
	var (
		n        = m.Rows() // number of rows
		cols     = m.Cols() // number of columns
		err      error      // error holder
		i, j     int
		aij, aji float64
	)
	if n != cols { // must be square
		return nil, nil, fmt.Errorf("Eigen: non-square %dx%d: %w", n, cols, matrix.ErrMatrixDimensionMismatch)
	}
	// check symmetry m[i][j] == m[j][i]
	for i = 0; i < n; i++ {
		for j = i + 1; j < n; j++ {
			aij, _ = m.At(i, j) // element (i,j)
			aji, _ = m.At(j, i) // element (j,i)
			if math.Abs(aij-aji) > tol {
				return nil, nil, ErrNotSymmetric // fail-fast on asymmetry
			}
		}
	}

	// Stage 2: Prepare A (work) and Q (eigenvectors)
	var (
		A matrix.Matrix // working copy
		Q matrix.Matrix // accumulate rotations
	)
	A = m.Clone()                  // deep copy input
	Q, err = matrix.NewDense(n, n) // allocate Q as identity
	if err != nil {
		return nil, nil, fmt.Errorf("Eigen: %w", err)
	}

	// initialize Q to identity
	for i = 0; i < n; i++ {
		_ = Q.Set(i, i, 1.0) // set diagonal ones
	}

	// Stage 3: Execute Jacobi rotations
	var (
		iter               int     // iteration counter
		p, q               int     // pivot indices
		maxOff             float64 // maximum off-diagonal value
		theta, t           float64 // rotation parameters
		c, s               float64 // cosine and sine
		off, aip, aiq, apq float64 // temporary values
	)
	for iter = 0; iter < maxIter; iter++ {
		// find largest off-diagonal |A[p][q]|
		maxOff = 0.0
		for i = 0; i < n; i++ {
			for j = i + 1; j < n; j++ {
				off, _ = A.At(i, j)
				if math.Abs(off) > maxOff {
					maxOff = off
					p, q = i, j
				}
			}
		}
		if maxOff < tol {
			break // converged
		}
		// compute rotation angle theta
		aip, _ = A.At(p, p)
		aiq, _ = A.At(q, q)
		apq, _ = A.At(p, q)
		theta = (aiq - aip) / (2 * apq)
		t = math.Copysign(1.0/(math.Abs(theta)+math.Sqrt(theta*theta+1)), theta)
		c = 1.0 / math.Sqrt(t*t+1) // cosine
		s = t * c                  // sine

		// apply rotation to A
		for i = 0; i < n; i++ {
			if i != p && i != q {
				// update A[i][p] and A[i][q]
				aip, _ = A.At(i, p)
				aiq, _ = A.At(i, q)
				_ = A.Set(i, p, c*aip-s*aiq)
				_ = A.Set(p, i, c*aip-s*aiq)
				_ = A.Set(i, q, s*aip+c*aiq)
				_ = A.Set(q, i, s*aip+c*aiq)
			}
		}
		// update diagonal entries
		_ = A.Set(p, p, c*c*aip-2*c*s*apq+s*s*aiq)
		_ = A.Set(q, q, s*s*aip+2*c*s*apq+c*c*aiq)
		_ = A.Set(p, q, 0.0)
		_ = A.Set(q, p, 0.0)

		// accumulate into Q
		for i = 0; i < n; i++ {
			aip, _ = Q.At(i, p)
			aiq, _ = Q.At(i, q)
			_ = Q.Set(i, p, c*aip-s*aiq)
			_ = Q.Set(i, q, s*aip+c*aiq)
		}
	}

	if iter == maxIter {
		return nil, nil, ErrEigenFailed // did not converge
	}

	// Stage 4: Finalize eigenvalues and return
	eigs := make([]float64, n)
	for i = 0; i < n; i++ {
		eigs[i], _ = A.At(i, i) // diagonal elements are eigenvalues
	}

	return eigs, Q, nil
}
