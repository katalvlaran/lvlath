// SPDX-License-Identifier: MIT
// Package matrix — Dense storage (row-major) & safe accessors.
//
// Purpose:
//   - Provide a cache-friendly row-major buffer with the explicit index formula i*cols + j.
//   - Guarantee safety at the public surface: At/Set return errors instead of panicking.
//   - Keep algorithmic determinism (fixed loop orders, no map iteration).
//   - Support no-copy views (MatrixView) and copy-based submatrix extraction (Induced).
//   - Enforce a numeric policy (optional rejection of NaN/Inf) from a single source of truth.
//
// AI-Hints:
//   - Prefer fast-paths on *Dense in hot algebra (see methods.go): operate on the flat data slice directly.
//   - Use View(r0,c0,h,w) to avoid copies when you only need a window; mutations reflect in the base matrix.
//   - Use Induced(rows, cols) to materialize a submatrix (copy) for independent lifetime or shape transforms.
//   - Toggle DefaultValidateNaNInf in options to catch numeric issues early during development/testing.
//
// Complexity quicksheet:
//   - NewDense: O(r*c) zero-init; At/Set: O(1); Clone: O(r*c); View: O(1); Induced: O(r'*c').

package matrix

import (
	"fmt"
	"math"
)

// denseErrorf wraps an error with a uniform Dense context and callsite indices.
// Example: "Dense.Set(3,7): matrix: out of range".
func denseErrorf(method string, row, col int, err error) error {
	return fmt.Errorf("Dense.%s(%d,%d): %w", method, row, col, err)
}

// Dense is a concrete row-major matrix.
//   - r,c hold dimensions (rows, cols).
//   - data is a flat buffer of length r*c in row-major order (offset = i*c + j).
//   - validateNaNInf enables optional NaN/Inf rejection in Set (policy default from options.go).
type Dense struct {
	r, c           int       // row and column counts (>=0; zero allowed only for internal zero-OK constructors)
	data           []float64 // contiguous row-major storage (len == r*c)
	validateNaNInf bool      // numeric guard: reject NaN/Inf in Set when true
}

// Compile-time assertions for interface & fmt.Stringer conformance.
var (
	_ Matrix       = (*Dense)(nil) // implements our public Matrix interface
	_ fmt.Stringer = (*Dense)(nil)
)

// NewDense creates an r×c zero matrix using row-major storage.
// Contract: rows>0 && cols>0. Returns ErrInvalidDimensions otherwise.
// Note: internal callers that need zero-sized dimensions should use newDenseZeroOK (below).
func NewDense(rows, cols int) (*Dense, error) {
	// Validate shape (public API does not allow empty dimensions to avoid accidental 0×0).
	if rows <= 0 || cols <= 0 {
		return nil, ErrInvalidDimensions // unified sentinel
	}
	// Allocate a contiguous flat buffer; make() zero-fills it deterministically.
	buf := make([]float64, rows*cols)

	// Initialize with the package-level default numeric policy.
	return &Dense{
		r:              rows,
		c:              cols,
		data:           buf,
		validateNaNInf: DefaultValidateNaNInf,
	}, nil
}

// newDenseZeroOK is an internal constructor that allows rows==0 or cols==0.
// Rationale: incidence builder may need V×0 matrices. We still set policy & invariants.
// Not exported to keep the public surface strict.
func newDenseZeroOK(rows, cols int) (*Dense, error) {
	if rows < 0 || cols < 0 {
		return nil, ErrInvalidDimensions
	}
	// Zero-length buffer is legal when rows==0 or cols==0 (len == rows*cols).
	buf := make([]float64, rows*cols)

	// Initialize with default numeric policy from options.go.
	return &Dense{
		r:              rows,
		c:              cols,
		data:           buf,
		validateNaNInf: DefaultValidateNaNInf,
	}, nil
}

// newDenseWithPolicy is a test/helper constructor to override the numeric policy.
// It preserves the same validation semantics as NewDense.
func newDenseWithPolicy(rows, cols int, validateNaNInf bool) (*Dense, error) {
	m, err := NewDense(rows, cols)
	if err != nil {
		return nil, err
	}
	m.validateNaNInf = validateNaNInf

	return m, nil
}

// Rows returns the row count. No side effects.
// Complexity: O(1).
func (m *Dense) Rows() int { return m.r }

// Cols returns the column count. No side effects.
// Complexity: O(1).
func (m *Dense) Cols() int { return m.c }

// Shape packs Rows() and Cols() into a single call for convenience.
// Complexity: O(1).
func (m *Dense) Shape() (rows, cols int) { return m.r, m.c }

// indexOf computes the row-major offset or returns ErrOutOfRange.
// We keep this unexported and free of panics so public At() / Set() remain safe.
// Complexity: O(1).
func (m *Dense) indexOf(row, col int) (int, error) {
	// Validate row index: 0 ≤ row < r.
	if row < 0 || row >= m.r {
		return 0, denseErrorf("At", row, col, ErrOutOfRange)
	}
	// Validate column index: 0 ≤ col < c.
	if col < 0 || col >= m.c {
		return 0, denseErrorf("At", row, col, ErrOutOfRange)
	}

	// Row-major offset: i*c + j.
	return row*m.c + col, nil
}

// At returns the value at (row, col) or ErrOutOfRange.
// Public surface never panics.
// Complexity: O(1).
func (m *Dense) At(row, col int) (float64, error) {
	off, err := m.indexOf(row, col) // bounds check + offset calc
	if err != nil {
		return 0, err // propagate sentinel
	}

	return m.data[off], nil // direct flat read
}

// Set stores v at (row, col) or returns an error (bounds or numeric policy).
// If validateNaNInf is true, NaN and ±Inf are rejected with ErrNaNInf.
// Complexity: O(1).
func (m *Dense) Set(row, col int, v float64) error {
	off, err := m.indexOf(row, col) // bounds check + offset calc
	if err != nil {
		return err // propagate sentinel
	}
	// Numeric policy: optional finite-only enforcement.
	if m.validateNaNInf && (math.IsNaN(v) || math.IsInf(v, 0)) {
		return denseErrorf("Set", row, col, ErrNaNInf)
	}
	m.data[off] = v // direct flat write

	return nil
}

// Clone returns a deep copy (new buffer, same numeric policy).
// Complexity: O(r*c) copy time and memory.
func (m *Dense) Clone() Matrix {
	cp := make([]float64, len(m.data)) // allocate same length
	copy(cp, m.data)                   // deep copy bytes

	return &Dense{
		r:              m.r,
		c:              m.c,
		data:           cp,
		validateNaNInf: m.validateNaNInf, // preserve guard policy
	}
}

// String provides a readable row-wise dump for diagnostics.
// It is not intended for hot paths; complexity  with formatting overhead.
// Complexity: O(r*c)
func (m *Dense) String() string {
	out := "" // simple builder; sufficient for diagnostics
	var i, j int
	for i = 0; i < m.r; i++ { // iterate rows deterministically
		out += "["                // open row
		for j = 0; j < m.c; j++ { // iterate cols
			// Direct offset (no At) avoids redundant bound checks.
			out += fmt.Sprintf("%g", m.data[i*m.c+j])
			if j+1 < m.c {
				out += ", " //separate values with comma + space
			}
		}
		out += "]\n" // close row
	}

	return out
}

// View creates a no-copy window [r0:r0+rows, c0:c0+cols) over the same storage.
// Mutations via the view are reflected in the base matrix (shared buffer).
// Bounds: 0 ≤ r0 ≤ r0+rows ≤ m.r and 0 ≤ c0 ≤ c0+cols ≤ m.c. Otherwise ErrBadShape.
// Complexity to create: O(1); subsequent At/Set: O(1).
func (m *Dense) View(r0, c0, rows, cols int) (*MatrixView, error) {
	// Validate proposed window; allow zero-area views.
	if r0 < 0 || c0 < 0 || rows < 0 || cols < 0 || r0+rows > m.r || c0+cols > m.c {
		return nil, fmt.Errorf("Dense.View(%d,%d,%d,%d): %w", r0, c0, rows, cols, ErrBadShape)
	}

	return &MatrixView{
		base: m,    // share storage
		r0:   r0,   // top row in base
		c0:   c0,   // left col in base
		r:    rows, // view height
		c:    cols, // view width
	}, nil
}

// Induced materializes a copy submatrix using explicit index sets.
// Duplicates are allowed; each index must satisfy 0 ≤ idx < dim.
// Zero-sized results (0×k or k×0) are allowed and return a legal Dense.
// Complexity: O(len(rowsIdx) * len(colsIdx)).
func (m *Dense) Induced(rowsIdx, colsIdx []int) (*Dense, error) {
	rp := len(rowsIdx) // result rows
	cp := len(colsIdx) // result cols

	// Negative sizes make no sense (guard anyway for API consistency).
	if rp < 0 || cp < 0 {
		return nil, ErrBadShape
	}

	// Allow zero-sized outputs; allocate a consistent zero-length buffer.
	if rp == 0 || cp == 0 {
		return &Dense{
			r:              rp,
			c:              cp,
			data:           make([]float64, 0),
			validateNaNInf: m.validateNaNInf,
		}, nil
	}

	// Allocate the result with the strict constructor.
	res, err := NewDense(rp, cp)
	if err != nil {
		return nil, err
	}

	// Deterministic double loop; direct offset math in both matrices.
	var i, j int
	var ri, cj int
	var src, dst int
	for i = 0; i < rp; i++ {
		ri = rowsIdx[i]
		if ri < 0 || ri >= m.r {
			return nil, fmt.Errorf("Dense.Induced: row index %d: %w", ri, ErrOutOfRange)
		}
		for j = 0; j < cp; j++ {
			cj = colsIdx[j]
			if cj < 0 || cj >= m.c {
				return nil, fmt.Errorf("Dense.Induced: col index %d: %w", cj, ErrOutOfRange)
			}
			// Direct linear index in source and destination.
			src = ri*m.c + cj // source offset in base
			dst = i*cp + j    // destination offset in result
			res.data[dst] = m.data[src]
		}
	}

	return res, nil
}

// MatrixView is a non-owning window into a Dense (shared storage).
// Not implementing Matrix interface to avoid accidental copies in ops.
type MatrixView struct {
	base *Dense // underlying storage owner
	r0   int    // top-left row offset in base
	c0   int    // top-left col offset in base
	r    int    // view height
	c    int    // view width
}

// Rows returns the number of rows in the view.
// Complexity: O(1).
func (v *MatrixView) Rows() int { return v.r }

// Cols returns the number of columns in the view.
// Complexity: O(1).
func (v *MatrixView) Cols() int { return v.c }

// At reads element (i,j) in the view or returns ErrOutOfRange.
// Complexity: O(1).
func (v *MatrixView) At(i, j int) (float64, error) {
	if i < 0 || i >= v.r || j < 0 || j >= v.c {
		return 0, fmt.Errorf("MatrixView.At(%d,%d): %w", i, j, ErrOutOfRange)
	}

	// Translate to base coordinates and load directly from the flat buffer.
	return v.base.data[(v.r0+i)*v.base.c+(v.c0+j)], nil
}

// Set writes element (i,j) in the view, honoring the base numeric policy.
// Complexity: O(1).
func (v *MatrixView) Set(i, j int, val float64) error {
	if i < 0 || i >= v.r || j < 0 || j >= v.c {
		return fmt.Errorf("MatrixView.Set(%d,%d): %w", i, j, ErrOutOfRange)
	}
	// Reuse base policy (single source of truth).
	if v.base.validateNaNInf && (math.IsNaN(val) || math.IsInf(val, 0)) {
		return fmt.Errorf("MatrixView.Set(%d,%d): %w", i, j, ErrNaNInf)
	}
	v.base.data[(v.r0+i)*v.base.c+(v.c0+j)] = val // write through

	return nil
}

// Do visits each element (i,j) in row-major order and calls f(i,j,v).
// If f returns false, the iteration stops early.
// No allocations; deterministic order; read-only with respect to the callback.
func (m *Dense) Do(f func(i, j int, v float64) bool) {
	var i, j, base int // predeclare loop counters and base offset
	var v float64      // temporary for current value

	for i = 0; i < m.r; i++ { // iterate rows deterministically
		base = i * m.c            // compute flat base offset for row i
		for j = 0; j < m.c; j++ { // iterate columns
			v = m.data[base+j] // read current element
			if !f(i, j, v) {   // invoke callback; stop if it returns false
				return // early exit requested by caller
			}
		}
	}
}

// Apply replaces each element with f(i,j,v) in-place.
// Respects numeric policy: when validateNaNInf is true, rejects NaN/Inf via ErrNaNInf.
// Deterministic row-major order; no extra allocations.
func (m *Dense) Apply(f func(i, j int, v float64) float64) error {
	var i, j, base int // predeclare loop counters and base offset
	var v, nv float64  // old and new values

	for i = 0; i < m.r; i++ { // iterate rows
		base = i * m.c            // base offset for row i
		for j = 0; j < m.c; j++ { // iterate columns
			v = m.data[base+j]       // read current value
			nv = f(i, j, v)          // compute new value
			if m.validateNaNInf && ( // enforce numeric policy if enabled
			math.IsNaN(nv) || math.IsInf(nv, 0)) {
				return denseErrorf("Apply", i, j, ErrNaNInf) // wrap with coordinates
			}
			m.data[base+j] = nv // write back new value
		}
	}

	return nil // success
}

/*

matrix/doc.go

// База
matrix/types.go
matrix/errors.go
matrix/validators.go           // (единая точка) ValidateNotNil/ValidateSquare/ValidateSameShape/... ValidateVecLen/ValidateGraph/ValidateSymmetric
matrix/options.go
matrix/impl_dense.go

// Графовые адаптеры
matrix/impl_graph_adjacency.go // AdjacencyMatrix, VertexCount, Neighbors, DegreeVector, ..., + BuildDenseAdjacency, build*FromGraph, ToGraph, returnEdge, утилиты pair/lookup
matrix/impl_graph_incidence.go // IncidenceMatrix + BuildDenseIncidence

// Ядра — линалг
matrix/impl_linear_algebra.go          // (full: core+factorization+eigen) Add, Sub, Hadamard, Transpose, Scale, Mul, MatVec +  LU, QR, Inverse + Eigen

// Ядра — граф
matrix/impl_floydwarshall.go   // FloydWarshall, floydWarshallInPlace, initDistancesInPlace

// Элементные и статистика
matrix/ops_elementwise.go
matrix/ops_sanitize_compare.go // Clip(), AllClose(), ReplaceInfNaN() — публичные над ew*
matrix/stats.go                // (full: center+normalize+covcorr)

// Публичные фасады
matrix/api.go                  // (full: construct+linalg+graph)


*/
