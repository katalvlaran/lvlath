// Package matrix provides core linear algebra primitives for array-based computations.
// Dense is a concrete, row-major implementation of the Matrix interface,
// storing elements in a flat slice for performance and cache friendliness.
package matrix

import (
	"fmt"
	"math"
)

// denseErrorf wraps an underlying error with Dense method context.
// Example message shape: "Dense.Set(3,7): matrix: index out of range".
func denseErrorf(method string, row, col int, err error) error {
	return fmt.Errorf(" Dense.%s(%d,%d): %w", method, row, col, err)
}

// Dense is a concrete row-major matrix.
// r, c are dimensions; data holds r*c elements in row-major order.
// validateNaNInf toggles finite-value enforcement in Set (policy default comes from options.go).
type Dense struct {
	r, c           int       // number of rows and columns
	data           []float64 // flat backing storage (len == r*c)
	validateNaNInf bool      // if true, Set rejects NaN/Inf with ErrNaNInf
}

// Compile-time assertion: *Dense implements the Matrix interface we expose publicly.
var _ Matrix = (*Dense)(nil)

// NewDense creates an r×c Dense initialized to zeros.
// Validates r>0 && c>0; returns ErrInvalidDimensions on failure.
// Complexity: O(r*c) due to zero-fill by make.
func NewDense(rows, cols int) (*Dense, error) {
	// Validate requested shape (strictly positive).
	if rows <= 0 || cols <= 0 {
		return nil, ErrInvalidDimensions
	}
	// Allocate contiguous row-major storage.
	buf := make([]float64, rows*cols) // zero-initialized

	// Initialize with default numeric policy from options.go.
	return &Dense{
		r:              rows,
		c:              cols,
		data:           buf,
		validateNaNInf: DefaultValidateNaNInf,
	}, nil
}

// newDenseWithPolicy is an internal helper when tests/constructors
// need to override NaN/Inf validation policy explicitly.
func newDenseWithPolicy(rows, cols int, validateNaNInf bool) (*Dense, error) {
	m, err := NewDense(rows, cols)
	if err != nil {
		return nil, err
	}
	m.validateNaNInf = validateNaNInf
	return m, nil
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

// Shape returns (rows, cols). Provided for convenience in internal code paths.
// (Not required by the public Matrix interface; harmless extra API.)
func (m *Dense) Shape() (rows, cols int) { return m.r, m.c }

// indexOf computes the flat offset for (row,col) or returns a sentinel.
// It does *not* panic; it validates both indices and returns ErrOutOfRange.
// Complexity: O(1).
func (m *Dense) indexOf(row, col int) (int, error) {
	// Validate row index
	if row < 0 || row >= m.r {
		return 0, denseErrorf("At", row, col, ErrOutOfRange)
	}
	// Validate column index
	if col < 0 || col >= m.c {
		return 0, denseErrorf("At", row, col, ErrOutOfRange)
	}

	// Row-major offset: i*cols + j.
	return row*m.c + col, nil
}

// At retrieves element at (row, col).
// Returns ErrOutOfRange on index violation.
// Complexity: O(1).
func (m *Dense) At(row, col int) (float64, error) {
	off, err := m.indexOf(row, col) // bounds check + offset
	if err != nil {
		return 0, err
	}

	return m.data[off], nil // read from flat storage
}

// Set writes value v at (row, col).
// Returns ErrOutOfRange on index violation, ErrNaNInf if validation is enabled.
// Complexity: O(1).
func (m *Dense) Set(row, col int, v float64) error {
	off, err := m.indexOf(row, col) // bounds check + offset
	if err != nil {
		return err
	}
	// Enforce numeric policy if enabled.
	if m.validateNaNInf && (math.IsNaN(v) || math.IsInf(v, 0)) {
		return denseErrorf("Set", row, col, ErrNaNInf)
	}
	m.data[off] = v // store value

	return nil
}

// Clone returns a deep copy of the matrix (data buffer is duplicated).
// Complexity: O(r*c) time and memory.
func (m *Dense) Clone() Matrix {
	cp := make([]float64, len(m.data)) // allocate new buffer
	copy(cp, m.data)                   // deep copy

	return &Dense{
		r:              m.r,
		c:              m.c,
		data:           cp,
		validateNaNInf: m.validateNaNInf, // preserve numeric policy
	}
}

// String provides a simple row-wise dump for debugging/logging.
// Complexity: O(r*c) formatting cost.
func (m *Dense) String() string {
	// Build with Go's default string concatenation; acceptable for debugging.
	// (No fmt reuse to avoid allocations per cell in hot paths.)
	out := ""
	var i, j int
	for i = 0; i < m.r; i++ { // iterate over rows
		out += "["                // open row
		for j = 0; j < m.c; j++ { // iterate over columns
			// Direct offset computation to avoid re-bounds in At.
			out += fmt.Sprintf("%g", m.data[i*m.c+j])
			if j+1 < m.c {
				out += ", " // separate values with comma
			}
		}
		out += "]\n" // close row
	}

	return out
}

// View returns a lightweight window into the same storage without copying.
// Semantics:
//   - Bounds: 0 ≤ r0 < r0+rows ≤ m.r and 0 ≤ c0 < c0+cols ≤ m.c.
//   - Mutations via the view reflect in the base matrix (shared storage).
//   - View is *not* required to satisfy the Matrix interface in Stage-4 (avoid accidental copies).
//
// Complexity: O(1) to create; At/Set on the view are O(1).
func (m *Dense) View(r0, c0, rows, cols int) (*MatrixView, error) {
	// Validate requested window bounds.
	if r0 < 0 || c0 < 0 || rows < 0 || cols < 0 || r0+rows > m.r || c0+cols > m.c {
		return nil, fmt.Errorf("Dense.View(%d,%d,%d,%d): %w", r0, c0, rows, cols, ErrBadShape)
	}

	return &MatrixView{
		base: m,
		r0:   r0, c0: c0,
		r: rows, c: cols,
	}, nil
}

// Induced builds a *copy* submatrix using the given row/column index sets.
// Each index must satisfy 0 ≤ idx < size; duplicates are allowed (rows/cols can repeat).
// Complexity: O(len(rows) * len(cols)).
func (m *Dense) Induced(rowsIdx []int, colsIdx []int) (*Dense, error) {
	rp := len(rowsIdx)
	cp := len(colsIdx)
	// Validate shape (allow 0×k and k×0).
	if rp < 0 || cp < 0 {
		return nil, ErrBadShape
	}
	if rp == 0 || cp == 0 {
		// Allocate empty dimension deterministically (no panics in At/Set for size 0).
		return &Dense{r: rp, c: cp, data: make([]float64, 0), validateNaNInf: m.validateNaNInf}, nil
	}
	// Validate indices and build the submatrix.
	res, err := NewDense(rp, cp)
	if err != nil {
		return nil, err
	}
	// Copy with fixed loop order for determinism.
	for i := 0; i < rp; i++ {
		ri := rowsIdx[i]
		if ri < 0 || ri >= m.r {
			return nil, fmt.Errorf("Dense.Induced: row index %d: %w", ri, ErrOutOfRange)
		}
		for j := 0; j < cp; j++ {
			cj := colsIdx[j]
			if cj < 0 || cj >= m.c {
				return nil, fmt.Errorf("Dense.Induced: col index %d: %w", cj, ErrOutOfRange)
			}
			// Direct linear index in source and destination.
			src := ri*m.c + cj
			dst := i*cp + j
			res.data[dst] = m.data[src]
		}
	}

	return res, nil
}

// MatrixView is a non-owning window into a Dense matrix (shared storage).
// It is intentionally lightweight and does NOT implement the Matrix interface in Stage-4
// to avoid silent copies in algorithms that expect owning semantics.
type MatrixView struct {
	base *Dense // underlying storage owner
	r0   int    // start row in base
	c0   int    // start col in base
	r    int    // number of rows in the view
	c    int    // number of cols in the view
}

// Rows returns the number of rows in the view (O(1)).
func (v *MatrixView) Rows() int { return v.r }

// Cols returns the number of cols in the view (O(1)).
func (v *MatrixView) Cols() int { return v.c }

// At reads a value from the view window (bounds-checked locally; returns ErrOutOfRange).
// Complexity: O(1).
func (v *MatrixView) At(i, j int) (float64, error) {
	if i < 0 || i >= v.r || j < 0 || j >= v.c {
		return 0, fmt.Errorf("MatrixView.At(%d,%d): %w", i, j, ErrOutOfRange)
	}
	return v.base.data[(v.r0+i)*v.base.c+(v.c0+j)], nil
}

// Set writes a value into the view window, honoring base numeric policy.
// Complexity: O(1).
func (v *MatrixView) Set(i, j int, val float64) error {
	if i < 0 || i >= v.r || j < 0 || j >= v.c {
		return fmt.Errorf("MatrixView.Set(%d,%d): %w", i, j, ErrOutOfRange)
	}
	// Reuse base policy; keep one source of truth.
	if v.base.validateNaNInf && (math.IsNaN(val) || math.IsInf(val, 0)) {
		return fmt.Errorf("MatrixView.Set(%d,%d): %w", i, j, ErrNaNInf)
	}
	v.base.data[(v.r0+i)*v.base.c+(v.c0+j)] = val
	return nil
}
