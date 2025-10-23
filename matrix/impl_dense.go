// SPDX-License-Identifier: MIT

// Package matrix - Dense storage (row-major) & safe accessors.
//
// Purpose:
//   - Provide a cache-friendly row-major buffer with the explicit index formula i*cols + j.
//   - Guarantee safety at the public surface: At/Set return errors instead of panicking.
//   - Keep algorithmic determinism (fixed loop orders, no map iteration).
//   - Support no-copy views (MatrixView) and copy-based submatrix extraction (Induced).
//   - Enforce a numeric policy (optional rejection of NaN/Inf) from a single source of truth.
//
// AI-Hints:
//   - Prefer fast-paths on *Dense in hot algebra (see impl_linear_algebra.go): operate on the flat data slice directly.
//   - Use View(r0,c0,h,w) to avoid copies for windows; mutations reflect in the base matrix.
//   - Use Induced(rows, cols) to materialize a submatrix (copy) for independent lifetime/shape.
//   - DefaultValidateNaNInf is on; insert only finite values unless you explicitly disable upstream.
//
// Complexity quicksheet:
//   - NewDense: O(r*c) zero-init; At/Set: O(1); Clone: O(r*c); View: O(1); Induced: O(r'*c').

package matrix

import (
	"fmt"
	"math"
	"strings"
)

// ---------- error context tags ----------

const (
	ctxAt     = "At"      // method tag used in error wrappers
	ctxSet    = "Set"     // method tag used in error wrappers
	ctxApply  = "Apply"   // method tag used in error wrappers
	ctxView   = "View"    // ctor tag for Dense.View
	ctxInduce = "Induced" // ctor/tag for Dense.Induced
)

// ---------- Formatting literals  ----------
const (
	_fmtRowOpen  = "["
	_fmtRowClose = "]\n"
	_fmtSep      = ", "
)

// denseErrorf wraps an error with a uniform Dense context and callsite indices.
// MAIN DESCRIPTION:
//   - Attach method context and coordinates to a sentinel error for diagnostics.
//
// Implementation:
//   - Stage 1: format "Dense.<method>(row,col): %w".
//   - Stage 2: return wrapped error.
//
// Behavior highlights:
//   - Stable, human-friendly messages; preserves sentinel via %w.
//
// Inputs:
//   - method: context tag (ctxAt/ctxSet/ctxApply/...)
//   - row, col: coordinates
//   - err: sentinel (e.g., ErrOutOfRange, ErrNaNInf)
//
// Returns:
//   - error: wrapped with context
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Keep tags in constants for grep-ability and consistency.
//
// AI-Hints:
//   - Prefer to wrap at the nearest detection site for precise coordinates.
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
	_ Matrix       = (*Dense)(nil) // *Dense implements our public Matrix interface
	_ fmt.Stringer = (*Dense)(nil)
)

// NewDense creates an r×c zero matrix using row-major storage.
// MAIN DESCRIPTION:
//   - Public constructor for Dense with strict shape validation and default numeric policy.
//
// Implementation:
//   - Stage 1: validate rows>0 && cols>0; else ErrInvalidDimensions.
//   - Stage 2: allocate zero-filled buffer and initialize policy.
//   - Stage 3: set numeric policy from defaults.
//
// Behavior highlights:
//   - No panics on user errors; returns sentinel errors.
//   - Public constructor forbids empty dimensions to avoid accidental 0×0 matrices.
//
// Inputs:
//   - rows: positive number of rows
//   - cols: positive number of columns
//
// Returns:
//   - *Dense: newly allocated matrix.
//
// Errors:
//   - ErrInvalidDimensions (shape contract violation).
//
// Determinism:
//   - Always allocates the same layout for given (rows, cols).
//   - Fixed zero initialization; no randomness.
//
// Complexity:
//   - Time O(r*c), Space O(r*c).
//
// Notes:
//   - Internal zero-sized cases use newDenseZeroOK.
//
// AI-Hints:
//   - Prefer this ctor for public creation. For subviews, use View().
func NewDense(rows, cols int) (*Dense, error) {
	// Validate shape.
	if rows <= 0 || cols <= 0 {
		return nil, ErrInvalidDimensions
	}
	// Allocate a contiguous flat buffer; make() zero-fills it deterministically.
	buf := make([]float64, rows*cols)

	return &Dense{
		r:              rows,
		c:              cols,
		data:           buf,
		validateNaNInf: DefaultValidateNaNInf,
	}, nil
}

// newDenseZeroOK is an internal constructor that allows rows==0 or cols==0.
// MAIN DESCRIPTION:
//   - Internal factory for legal 0×N or N×0 shapes used by builders.
//
// Implementation:
//   - Stage 1: validate rows>=0 && cols>=0.
//   - Stage 2: allocate len(rows*cols) buffer (possibly zero).
//
// Behavior highlights:
//   - Same numeric policy as public constructor.
//   - Used by builders to produce legal 0×k or k×0 matrices when needed.
//
// Inputs:
//   - rows, cols: non-negative dimensions.
//
// Returns:
//   - *Dense or ErrInvalidDimensions.
//
// Complexity:
//   - Time O(r*c).
//
// Inputs:
//   - rows, cols: non-negative dimensions.
//
// Returns:
//   - *Dense or ErrInvalidDimensions on negatives.
//
// Complexity:
//   - Time O(rows*cols), Space O(rows*cols).
func newDenseZeroOK(rows, cols int) (*Dense, error) {
	if rows < 0 || cols < 0 {
		return nil, ErrInvalidDimensions
	}
	// Zero-length buffer is legal when rows==0 or cols==0 (len == rows*cols).
	buf := make([]float64, rows*cols)

	return &Dense{
		r:              rows,
		c:              cols,
		data:           buf,
		validateNaNInf: DefaultValidateNaNInf,
	}, nil
}

// newDenseWithPolicy is a helper for tests/builders to override numeric policy.
// MAIN DESCRIPTION:
//   - Construct Dense with strict shape validation, then set validateNaNInf explicitly.
//
// Implementation:
//   - Stage 1: call NewDense(rows, cols).
//   - Stage 2: set policy flag.
//
// Behavior highlights:
//   - Centralized creation semantics.
//   - Intended for package internals and tests.
//
// Inputs:
//   - rows, cols; validateNaNInf.
//
// Returns:
//   - *Dense or error from NewDense.
//
// Complexity:
//   - Time O(rows*cols), Space O(rows*cols).
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
// MAIN DESCRIPTION:
//   - Bounds-check (row,col) and compute flat offset for row-major storage.
//
// Implementation:
//   - Stage 1: validate 0 ≤ row < m.r and 0 ≤ col < m.c.
//   - Stage 2: compute row*m.c + col.
//
// Behavior highlights:
//   - Error is wrapped with the caller's method context.
//   - Returns a sentinel (ErrOutOfRange) without adding context; public
//     methods (At/Set) will wrap with coordinates and method name.
//
// Inputs:
//   - method: caller identifier (ctxAt/ctxSet/...)
//   - row, col: coordinates.
//
// Returns:
//   - (offset, nil) on success; (0, ErrOutOfRange) otherwise.
//
// Errors:
//   - ErrOutOfRange when indices are invalid
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Keep unexported to avoid accidental panics at public surface.
//
// AI-Hints:
//   - Reuse in At/Set to keep identical bound semantics.
func (m *Dense) indexOf(row, col int) (int, error) {
	if row < 0 || row >= m.r {
		return 0, ErrOutOfRange
	}
	if col < 0 || col >= m.c {
		return 0, ErrOutOfRange
	}

	// Row-major offset: i*c + j.
	return row*m.c + col, nil
}

// At returns the value at (row, col) or ErrOutOfRange.
// MAIN DESCRIPTION:
//   - Safe element read at coordinates.
//
// Implementation:
//   - Stage 1: compute offset via indexOf (bounds check).
//   - Stage 2: load from flat buffer.
//
// Behavior highlights:
//   - Never panics on out-of-range; returns sentinel error.
//
// Inputs:
//   - row, col: zero-based indices.
//
// Returns:
//   - (value, nil) on success; (0, ErrOutOfRange) on invalid indices.
//
// Errors:
//   - ErrOutOfRange when out of bounds
//
// Determinism:
//   - Stable access cost; no allocations.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Uses direct data[] to avoid double checking.
//
// AI-Hints:
//   - Prefer At in external code; internal hot paths may index directly.
func (m *Dense) At(row, col int) (float64, error) {
	off, err := m.indexOf(row, col)
	if err != nil {
		return 0, denseErrorf(ctxAt, row, col, err) // wrap with context
	}

	return m.data[off], nil
}

// Set stores v at (row, col) or returns an error (bounds or numeric policy).
// MAIN DESCRIPTION:
//   - Safe element write with optional finite-only policy.
//
// Implementation:
//   - Stage 1: compute offset via indexOf (bounds check).
//   - Stage 2: enforce numeric policy (reject NaN/±Inf when enabled).
//   - Stage 3: write into flat buffer.
//
// Behavior highlights:
//   - Never panics; returns sentinel errors.
//   - Numeric policy is a per-instance flag preserved by Clone.
//
// Inputs:
//   - row, col: element coordinates.
//   - v      : value to store.
//
// Returns:
//   - nil on success; errors on invalid indices.
//
// Errors:
//   - ErrOutOfRange for bounds; ErrNaNInf for invalid numbers
//
// Determinism:
//   - Direct flat write; fixed order irrelevant here.
//
// Determinism:
//   - Stable, no side-effects beyond the cell.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Policy flag is carried by Clone/Induced/View (single source of truth).
//
// AI-Hints:
//   - Keep policy ON in production data flows; disable only in controlled ingestion.
func (m *Dense) Set(row, col int, v float64) error {
	off, err := m.indexOf(row, col)
	if err != nil {
		return denseErrorf(ctxSet, row, col, err) // wrap with context
	}
	// Numeric policy: optional finite-only enforcement.
	if m.validateNaNInf && (math.IsNaN(v) || math.IsInf(v, 0)) {
		return denseErrorf(ctxSet, row, col, ErrNaNInf)
	}
	m.data[off] = v // direct flat write

	return nil
}

// Clone returns a deep copy (new buffer, same numeric policy).
// MAIN DESCRIPTION:
//   - Produce an independent Dense with identical shape/data/policy.
//
// Implementation:
//   - Stage 1: allocate new buffer len==r*c.
//   - Stage 2: copy data and flags.
//
// Behavior highlights:
//   - Independence: mutations do not affect the original.
//
// Returns:
//   - Matrix: *Dense implementing Matrix.
//
// Determinism:
//   - Stable double loop cost reduced to single copy.
//
// Complexity:
//   - Time O(r*c), Space O(r*c).
//
// Notes:
//   - Returned dynamic type is *Dense.
//
// AI-Hints:
//   - For structural copy with transform, consider Apply on clone.
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
// MAIN DESCRIPTION:
//   - Render matrix rows as lines with comma-separated values.
//
// Implementation:
//   - Stage 1: iterate rows/cols deterministically.
//   - Stage 2: append values formatted with %g.
//
// Behavior highlights:
//   - Intended for debugging; not for hot paths.
//

// String HUMAN-READABLE dump of rows for diagnostics.
// Implementation:
//   - Stage 1: iterate rows/cols deterministically.
//   - Stage 2: write values into strings.Builder with standard delimiters.
//
// Behavior highlights:
//   - Not for hot paths; intended for logs and debugging.
//
// Returns:
//   - string: multi-line representation of matrix.
//
// Determinism:
//   - Fixed traversal order.
//
// Complexity:
//   - Time O(r*c), Space O(r*c) for formatting.
//
// AI-Hints:
//   - For large matrices prefer printing a few rows/cols or summarize.
func (m *Dense) String() string {
	var b strings.Builder
	var i, j, base int
	for i = 0; i < m.r; i++ { // iterate rows deterministically
		b.WriteString(_fmtRowOpen) // open row
		base = i * m.c
		for j = 0; j < m.c; j++ { // iterate cols
			b.WriteString(fmt.Sprintf("%g", m.data[base+j]))
			if j+1 < m.c {
				b.WriteString(_fmtSep) //separate values with comma + space
			}
		}
		b.WriteString(_fmtRowClose) // close row
	}

	return b.String()
}

// View creates a no-copy window [r0:r0+rows, c0:c0+cols) over the same storage.
// MAIN DESCRIPTION:
//   - Lightweight submatrix referencing the base buffer (shared storage).
//
// Implementation:
//   - Stage 1: validate window bounds; allow zero-area.
//   - Stage 2: return MatrixView with offsets.
//
// Behavior highlights:
//   - Writes via view reflect in base; policy is inherited.
//
// Inputs:
//   - r0,c0: top-left offsets; rows, cols: window size (≥0).
//
// Returns:
//   - *MatrixView or error.
//
// Errors:
//   - ErrBadShape when the window is invalid.
//
// Determinism:
//   - Constant-time creation; fixed access order in methods.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - View does not implement Matrix on purpose to avoid accidental copies in ops.
//
// AI-Hints:
//   - Use for sliding-window ops; copy only when lifetime must be independent.
func (m *Dense) View(r0, c0, rows, cols int) (*MatrixView, error) {
	if r0 < 0 || c0 < 0 || rows < 0 || cols < 0 || r0+rows > m.r || c0+cols > m.c {
		return nil, fmt.Errorf("Dense.%s(%d,%d,%d,%d): %w", ctxView, r0, c0, rows, cols, ErrBadShape)
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
// MAIN DESCRIPTION:
//   - Copy rows/cols at the given index lists (duplicates allowed).
//
// Implementation:
//   - Stage 1: handle zero-sized result (legal).
//   - Stage 2: allocate result via NewDense.
//   - Stage 3: nested loops with direct offset math; bounds-check each index.
//
// Behavior highlights:
//   - Policy is preserved from the base (validateNaNInf).
//   - Duplicates in index sets are allowed (repeated rows/cols in the result).
//
// Inputs:
//   - rowsIdx: indices into [0..m.r).
//   - colsIdx: indices into [0..m.c).
//
// Returns:
//   - *Dense: independent copy with size len(rowsIdx)×len(colsIdx).
//
// Errors:
//   - ErrOutOfRange (index outside bounds).
//
// Determinism:
//   - Fixed nested loops i→j.
//
// Complexity:
//   - Time O(rp*cp), Space O(rp*cp).
//
// Notes:
//   - Zero-area returns legal Dense with zero-length buffer.
//
// AI-Hints:
//   - Use when the result must be independent (e.g., transform downstream).
func (m *Dense) Induced(rowsIdx, colsIdx []int) (*Dense, error) {
	rp := len(rowsIdx) // result rows
	cp := len(colsIdx) // result cols
	// Zero-area: legal Dense, shared policy
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
	// Preserve numeric policy from the base (critical for consistency).
	res.validateNaNInf = m.validateNaNInf

	// Deterministic double loop; direct offset math in both matrices.
	var i, j int
	var ri, cj int
	var src, dst int
	for i = 0; i < rp; i++ {
		ri = rowsIdx[i]
		if ri < 0 || ri >= m.r {
			return nil, fmt.Errorf("Dense.%s: row index %d: %w", ctxInduce, ri, ErrOutOfRange)
		}
		for j = 0; j < cp; j++ {
			cj = colsIdx[j]
			if cj < 0 || cj >= m.c {
				return nil, fmt.Errorf("Dense.%s: col index %d: %w", ctxInduce, cj, ErrOutOfRange)
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
// MAIN DESCRIPTION:
//   - Safe read within the view bounds; translates to base coordinates.
//
// Implementation:
//   - Stage 1: check 0≤i<r and 0≤j<c.
//   - Stage 2: return base.data[(r0+i)*base.c + (c0+j)].
//
// Behavior highlights:
//   - Never panics; returns sentinel on violation.
//
// Complexity:
//   - Time O(1), Space O(1).
func (v *MatrixView) At(i, j int) (float64, error) {
	if i < 0 || i >= v.r || j < 0 || j >= v.c {
		return 0, fmt.Errorf("MatrixView.At(%d,%d): %w", i, j, ErrOutOfRange)
	}

	// Translate to base coordinates and load directly from the flat buffer.
	return v.base.data[(v.r0+i)*v.base.c+(v.c0+j)], nil
}

// Set writes element (i,j) in the view, honoring the base numeric policy.
// MAIN DESCRIPTION:
//   - Safe write-through into the base buffer with policy enforcement.
//
// Implementation:
//   - Stage 1: check bounds.
//   - Stage 2: validate finite when base policy is enabled.
//   - Stage 3: write-through into base.data.
//
// Behavior highlights:
//   - Shares the base Dense policy; no separate flags in the view.
//
// Complexity:
//   - Time O(1), Space O(1).
func (v *MatrixView) Set(i, j int, val float64) error {
	if i < 0 || i >= v.r || j < 0 || j >= v.c {
		return fmt.Errorf("MatrixView.Set(%d,%d): %w", i, j, ErrOutOfRange)
	}
	if v.base.validateNaNInf && (math.IsNaN(val) || math.IsInf(val, 0)) {
		return fmt.Errorf("MatrixView.Set(%d,%d): %w", i, j, ErrNaNInf)
	}
	v.base.data[(v.r0+i)*v.base.c+(v.c0+j)] = val // write through

	return nil
}

// Do visits each element (i,j) in row-major order and calls f(i,j,v).
// MAIN DESCRIPTION:
//   - Read-only visitor; stops early when f returns false.
//
// Implementation:
//   - Stage 1: nested loops - double for-loop over rows then cols; compute base offset per row.
//   - Stage 2: call f on each element; stop when f returns false.
//
// Behavior highlights:
//   - Read-only with respect to the callback; no allocations; deterministic order.
//
// Inputs:
//   - f: callback returning continue/stop flag (false to stop early).
//
// Determinism:
//   - Fixed i→j order.
//
// Complexity:
//   - Time O(r*c), Space O(1).
//
// AI-Hints:
//   - Use to accumulate stats without temporary allocations.
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
// MAIN DESCRIPTION:
//   - In-place map with policy enforcement and deterministic order.
//
// Implementation:
//   - Stage 1: nested loops - double for-loop over rows then cols; compute new value via f.
//   - Stage 2: compute new value; reject NaN/Inf if policy enabled.
//   - Stage 3: write back.
//
// Behavior highlights:
//   - Deterministic row-major order; no extra allocations.
//   - Respects validateNaNInf (rejects NaN/±Inf when enabled).
//   - Early error aborts; elements written before the error remain updated.
//
// Inputs:
//   - f: transformer from (i,j,v) to new value.
//
// Returns:
//   - error: ErrNaNInf when transformer produced non-finite (if policy ON).
//
// Determinism:
//   - Fixed i→j order; side effects are predictable.
//
// Complexity:
//   - Time O(r*c), Space O(1).
//
// Notes:
//   - For all-or-nothing semantics, transform into a clone and swap on success.
//
// AI-Hints:
//   - Keep transforms pure; avoid capturing external mutable state.
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
				return denseErrorf(ctxApply, i, j, ErrNaNInf) // wrap with coordinates
			}
			m.data[base+j] = nv // write back new value
		}
	}

	return nil // success
}
