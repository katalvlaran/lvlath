// Package matrix provides universal operations on any Matrix implementation,
// including element-wise addition, subtraction, matrix multiplication,
// transpose, and scalar scaling. All functions perform strict
// fail-fast validation and return clear errors on dimension mismatches.
package matrix

import (
	"errors"
	"fmt"
)

// ErrNilMatrix indicates that a nil Matrix was passed to an operation.
var ErrNilMatrix = errors.New("matrix: nil Matrix")

// matrixErrorf wraps an underlying error with the given tag.
func matrixErrorf(tag string, err error) error {
	return fmt.Errorf("%s: %w", tag, err)
}

// Add returns a new Matrix containing the element-wise sum of a and b.
// Stage 1 (Validate): nil-checks and shape match.
// Stage 2 (Prepare): allocate result Dense.
// Stage 3 (Execute): loop over elements.
// Stage 4 (Finalize): return result.
// Complexity: O(r·c) time and memory.
func Add(a, b Matrix) (Matrix, error) {
	// Stage 1: Validate inputs
	if a == nil || b == nil {
		return nil, fmt.Errorf("Add: %w", ErrNilMatrix)
	}
	if err := ValidateSameShape(a, b); err != nil {
		return nil, fmt.Errorf("Add: %w", err)
	}
	// Stage 2: Prepare result
	rows, cols := a.Rows(), a.Cols()
	res, err := NewDense(rows, cols)
	if err != nil {
		return nil, matrixErrorf("Add", err)
	}
	// Stage 3: Execute element-wise addition
	var ( // loop vars
		i, j   int
		av, bv float64
	)
	for i = 0; i < rows; i++ {
		for j = 0; j < cols; j++ {
			av, _ = a.At(i, j) // safe: shapes match
			bv, _ = b.At(i, j)
			_ = res.Set(i, j, av+bv) // safe: within bounds
		}
	}

	// Stage 4: Return result
	return res, nil
}

// Sub returns a new Matrix containing the element-wise difference a - b.
// Follows the same blueprint as Add.
// Complexity: O(r·c).
func Sub(a, b Matrix) (Matrix, error) {
	// Stage 1: Validate inputs
	if a == nil || b == nil {
		return nil, fmt.Errorf("Sub: %w", ErrNilMatrix)
	}
	if err := ValidateSameShape(a, b); err != nil {
		return nil, fmt.Errorf("Sub: %w", err)
	}

	// Stage 2: Prepare result
	rows, cols := a.Rows(), a.Cols()
	res, err := NewDense(rows, cols)
	if err != nil {
		return nil, matrixErrorf("Sub", err)
	}
	// Stage 3: Execute subtraction
	var ( // loop vars
		i, j   int
		av, bv float64
	)
	for i = 0; i < rows; i++ {
		for j = 0; j < cols; j++ {
			av, _ = a.At(i, j)
			bv, _ = b.At(i, j)
			_ = res.Set(i, j, av-bv)
		}
	}

	// Stage 4: Return result
	return res, nil
}

// Mul performs standard matrix multiplication of a and b (a × b).
// Stage 1 (Validate): nil-check and inner-dimension match.
// Stage 2 (Prepare): allocate result Dense.
// Stage 3 (Execute): triple loop, with fast-path for *Dense.
// Stage 4 (Finalize): return result.
// Complexity: O(r*n*c) time and O(r*c) memory.
func Mul(a, b Matrix) (Matrix, error) {
	// Stage 1: Validate inputs
	if a == nil || b == nil {
		return nil, matrixErrorf("Mul", ErrNilMatrix)
	}
	if a.Cols() != b.Rows() {
		return nil, matrixErrorf("Mul", ErrMatrixDimensionMismatch)
	}
	// Stage 2: Prepare result
	r, n, c := a.Rows(), a.Cols(), b.Cols()
	res, err := NewDense(r, c)
	if err != nil {
		return nil, matrixErrorf("Mul", err)
	}
	// Multiply: res[i][j] = sum_k a[i][k]*b[k][j]
	var ( // loop vars
		i, j, k         int
		av, bv, current float64
	)
	for i = 0; i < r; i++ {
		for k = 0; k < n; k++ {
			av, _ = a.At(i, k)
			if av == 0 {
				continue // skip zero for performance
			}
			for j = 0; j < c; j++ {
				bv, _ = b.At(k, j)
				// accumulate product
				current, _ = res.At(i, j)
				_ = res.Set(i, j, current+av*bv)
			}
		}
	}

	// Stage 4: Return result
	return res, nil
}

// Transpose returns a new Matrix where rows and columns of m are swapped.
// Stage 1 (Validate): nil-check.
// Stage 2 (Prepare): allocate Dense(cols×rows).
// Stage 3 (Execute): double loop copy.
// Stage 4 (Finalize): return.
// Complexity: O(r·c).
func Transpose(m Matrix) Matrix {
	// Stage 1: Validate input
	if m == nil {
		panic("Transpose: nil Matrix") // programmer error
	}
	// Stage 2: Prepare result
	rows, cols := m.Rows(), m.Cols()
	res, _ := NewDense(cols, rows) // dims flipped
	var (                          // loop vars
		i, j int
		v    float64
	)
	// Stage 3: Execute copy
	for i = 0; i < rows; i++ {
		for j = 0; j < cols; j++ {
			v, _ = m.At(i, j)
			_ = res.Set(j, i, v)
		}
	}
	// Stage 4: Return result
	return res
}

// Scale returns a new Matrix where each element of m is multiplied by alpha.
// Stage 1 (Validate): nil-check.
// Stage 2 (Prepare): allocate Dense(rows×cols).
// Stage 3 (Execute): double loop scaling.
// Stage 4 (Finalize): return.
// Complexity: O(r·c).
func Scale(m Matrix, alpha float64) Matrix {
	// Stage 1: Validate input
	if m == nil {
		panic("Scale: nil Matrix") // programmer error
	}
	// Stage 2: Prepare result
	rows, cols := m.Rows(), m.Cols()
	res, _ := NewDense(rows, cols)
	var ( // loop vars
		i, j int
		v    float64
	)
	// Stage 3: Execute scaling
	for i = 0; i < rows; i++ {
		for j = 0; j < cols; j++ {
			v, _ = m.At(i, j)
			_ = res.Set(i, j, v*alpha)
		}
	}

	// Stage 4: Return result
	return res
}
