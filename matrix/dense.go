// Package matrix provides core linear algebra primitives for array-based computations.
// Dense is a concrete, row-major implementation of the Matrix interface,
// storing elements in a flat slice for performance and cache friendliness.
package matrix

import (
	"errors"
	"fmt"
)

// ErrInvalidDimensions indicates that requested matrix dimensions are non-positive.
var ErrInvalidDimensions = errors.New("matrix: dimensions must be > 0")

// ErrIndexOutOfBounds indicates that a row or column index is outside valid range.
var ErrIndexOutOfBounds = errors.New("matrix: index out of bounds")

// denseErrorf wraps an underlying error with Dense method context.
func denseErrorf(method string, row, col int, err error) error {
	return fmt.Errorf("Dense.%s(%d,%d): %w", method, row, col, err)
}

// Dense is a row-major matrix of float64 values.
// r is rows, c is columns, and data holds r*c elements in row-major order.
type Dense struct {
	r, c int       // number of rows and columns
	data []float64 // flat backing storage, length == r*c
}

// NewDense creates an r×c Dense matrix initialized to zeros.
// Stage 1 (Validate): ensure rows and cols > 0.
// Stage 2 (Prepare): allocate flat backing slice.
// Stage 3 (Finalize): return new Dense or ErrInvalidDimensions.
// Complexity: O(r*c) time and memory.
func NewDense(rows, cols int) (*Dense, error) {
	// Validate dimensions
	if rows <= 0 || cols <= 0 {
		return nil, ErrInvalidDimensions
	}
	// Allocate flat slice
	data := make([]float64, rows*cols)

	// Return initialized Dense
	return &Dense{r: rows, c: cols, data: data}, nil
}

// Rows returns the number of rows in the matrix.
// Complexity: O(1).
func (m *Dense) Rows() int {
	return m.r // return stored row count
}

// Cols returns the number of columns in the matrix.
// Complexity: O(1).
func (m *Dense) Cols() int {
	return m.c // return stored column count
}

// indexOf computes the flat index for (row, col) or returns ErrIndexOutOfBounds.
// Stage 1 (Validate): check 0 ≤ row < r and 0 ≤ col < c.
// Stage 2 (Execute): compute and return linear index.
// Complexity: O(1).
func (m *Dense) indexOf(row, col int) (int, error) {
	// Validate row index
	if row < 0 || row >= m.r {
		return 0, denseErrorf("At", row, col, ErrIndexOutOfBounds)
	}
	// Validate column index
	if col < 0 || col >= m.c {
		return 0, denseErrorf("At", row, col, ErrIndexOutOfBounds)
	}

	// Compute flat offset
	return row*m.c + col, nil
}

// At retrieves the element at (row, col).
// Stage 1 (Validate): bounds check via indexOf.
// Stage 2 (Execute): read from data slice.
// Stage 3 (Finalize): return value or wrapped error.
// Complexity: O(1).
func (m *Dense) At(row, col int) (float64, error) {
	// Compute flat index or error
	idx, err := m.indexOf(row, col)
	if err != nil {
		return 0, err
	}

	// Return stored value
	return m.data[idx], nil
}

// Set assigns value v at (row, col).
// Stage 1 (Validate): bounds check via indexOf.
// Stage 2 (Execute): write into data slice.
// Stage 3 (Finalize): return error or nil.
// Complexity: O(1).
func (m *Dense) Set(row, col int, v float64) error {
	// Compute flat index or error
	idx, err := m.indexOf(row, col)
	if err != nil {
		return err
	}
	// Assign value
	m.data[idx] = v

	return nil
}

// Clone returns a deep copy of the Dense matrix.
// Complexity: O(r*c) time and memory for copy.
func (m *Dense) Clone() Matrix {
	// Allocate new slice for data copy
	copyData := make([]float64, len(m.data))
	// Copy all elements into new slice
	copy(copyData, m.data)

	return &Dense{r: m.r, c: m.c, data: copyData}
}

// String implements fmt.Stringer for easy debugging.
// Stage 1 (Execute): build per-row strings.
// Stage 2 (Finalize): return concatenated representation.
// Complexity: O(r*c) for string construction.
func (m *Dense) String() string {
	var s string
	var i, j int
	for i = 0; i < m.r; i++ { // iterate over rows
		s += "["                  // open row
		for j = 0; j < m.c; j++ { // iterate over columns
			// compute flat index directly for performance
			s += fmt.Sprintf("%g", m.data[i*m.c+j])
			if j < m.c-1 {
				s += ", " // separate values with comma
			}
		}
		s += "]\n" // close row
	}

	return s
}
