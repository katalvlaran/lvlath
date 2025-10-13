// SPDX-License-Identifier: MIT
// Package: matrix
//
// Purpose:
//   - Provide small, *private* element-wise and broadcast kernels (ew*) to avoid
//     duplicating tight loops across higher-level ops (stats, sanitize).
//   - Keep all loops deterministic and cache-friendly with Dense fast-paths.
//
// Design:
//   - All ew* are UNEXPORTED by design (internal micro-kernels).
//   - Public API uses these via thin wrappers (e.g., stats.go, ops_sanitize_compare.go).
//
// Determinism & Performance:
//   - Fixed loop orders (i→j or flat 0..n-1).
//   - Dense fast-path operates on a single flat buffer (row-major).
//   - No hidden allocations beyond the output Dense; O(r*c) time and space.
//
// AI-Hints:
//   - Prefer passing *Dense to unlock the flat-slice fast path.
//   - Keep broadcast arrays (colMeans/rowMeans/scale) precomputed and reused across calls.
//   - Avoid re-allocations in hot paths by pooling inputs/outputs at a higher layer if needed.

package matrix

import (
	"math"
)

// ewBroadcastSubCols computes out[i,j] = X[i,j] - colMeans[j].
// Time: O(r*c). Space: O(r*c). Deterministic i→j loops.
//
// AI-Hint: Use for column-centering and z-scoring.
func ewBroadcastSubCols(X Matrix, colMeans []float64) (Matrix, error) {
	// Validate matrix presence using centralized validator.
	if err := ValidateNotNil(X); err != nil {
		return nil, matrixErrorf("broadcastSubCols", err)
	}
	// Read shape once (O(1)).
	r, c := X.Rows(), X.Cols()
	// Check broadcast vector length.
	if len(colMeans) != c {
		return nil, matrixErrorf("broadcastSubCols", ErrDimensionMismatch)
	}
	// Allocate result dense (O(1) alloc + O(r*c) zeroing by runtime).
	out, err := NewDense(r, c)
	if err != nil {
		return nil, matrixErrorf("broadcastSubCols", err)
	}

	// Dense fast-path: single pass over the flat row-major buffer.
	if d, ok := X.(*Dense); ok {
		// Iterate rows deterministically.
		for i := 0; i < r; i++ {
			base := i * c // cache the base offset for row i
			// Iterate columns deterministically.
			for j := 0; j < c; j++ {
				// Subtract the column mean from each element (one read, one write).
				out.data[base+j] = d.data[base+j] - colMeans[j]
			}
		}
		return out, nil
	}

	// Generic fallback via At/Set (still deterministic).
	for i := 0; i < r; i++ {
		for j := 0; j < c; j++ {
			v, e := X.At(i, j)
			if e != nil {
				return nil, matrixErrorf("broadcastSubCols", e)
			}
			_ = out.Set(i, j, v-colMeans[j]) // bounds-safe write
		}
	}
	return out, nil
}

// ewBroadcastSubRows computes out[i,j] = X[i,j] - rowMeans[i].
// Time: O(r*c). Space: O(r*c). Deterministic i→j loops.
func ewBroadcastSubRows(X Matrix, rowMeans []float64) (Matrix, error) {
	// Validate matrix presence.
	if err := ValidateNotNil(X); err != nil {
		return nil, matrixErrorf("broadcastSubRows", err)
	}
	// Read shape once.
	r, c := X.Rows(), X.Cols()
	// Check broadcast vector length.
	if len(rowMeans) != r {
		return nil, matrixErrorf("broadcastSubRows", ErrDimensionMismatch)
	}
	// Allocate result dense.
	out, err := NewDense(r, c)
	if err != nil {
		return nil, matrixErrorf("broadcastSubRows", err)
	}

	// Dense fast-path.
	if d, ok := X.(*Dense); ok {
		for i := 0; i < r; i++ {
			base := i * c     // base offset for row i
			rm := rowMeans[i] // cache row mean once per row
			for j := 0; j < c; j++ {
				out.data[base+j] = d.data[base+j] - rm
			}
		}
		return out, nil
	}

	// Generic fallback.
	for i := 0; i < r; i++ {
		rm := rowMeans[i] // read once per row
		for j := 0; j < c; j++ {
			v, e := X.At(i, j)
			if e != nil {
				return nil, matrixErrorf("broadcastSubRows", e)
			}
			_ = out.Set(i, j, v-rm)
		}
	}
	return out, nil
}

// ewScaleCols computes out[i,j] = X[i,j] * scale[j].
// Time: O(r*c). Space: O(r*c). Deterministic i→j loops.
//
// AI-Hint: use factors as 1/std for z-scoring, or 0 for degenerate columns. O(r*c).
func ewScaleCols(X Matrix, scale []float64) (Matrix, error) {
	// Validate matrix presence.
	if err := ValidateNotNil(X); err != nil {
		return nil, matrixErrorf("scaleCols", err)
	}
	// Read shape once.
	r, c := X.Rows(), X.Cols()
	// Validate scale length.
	if len(scale) != c {
		return nil, matrixErrorf("scaleCols", ErrDimensionMismatch)
	}
	// Allocate result dense.
	out, err := NewDense(r, c)
	if err != nil {
		return nil, matrixErrorf("scaleCols", err)
	}

	// Dense fast-path.
	if d, ok := X.(*Dense); ok {
		for i := 0; i < r; i++ {
			base := i * c // row base offset
			for j := 0; j < c; j++ {
				out.data[base+j] = d.data[base+j] * scale[j]
			}
		}
		return out, nil
	}

	// Generic fallback.
	for i := 0; i < r; i++ {
		for j := 0; j < c; j++ {
			v, e := X.At(i, j)
			if e != nil {
				return nil, matrixErrorf("scaleCols", e)
			}
			_ = out.Set(i, j, v*scale[j])
		}
	}
	return out, nil
}

// ewScaleRows computes out[i,j] = X[i,j] * scale[i].
// Time: O(r*c). Space: O(r*c). Deterministic i→j loops.
//
// AI-Hint: use for L1/L2 row-normalization. O(r*c).
func ewScaleRows(X Matrix, scale []float64) (Matrix, error) {
	// Validate matrix presence.
	if err := ValidateNotNil(X); err != nil {
		return nil, matrixErrorf("scaleRows", err)
	}
	// Read shape once.
	r, c := X.Rows(), X.Cols()
	// Validate scale length.
	if len(scale) != r {
		return nil, matrixErrorf("scaleRows", ErrDimensionMismatch)
	}
	// Allocate result dense.
	out, err := NewDense(r, c)
	if err != nil {
		return nil, matrixErrorf("scaleRows", err)
	}

	// Dense fast-path.
	if d, ok := X.(*Dense); ok {
		for i := 0; i < r; i++ {
			base := i * c  // row base offset
			sf := scale[i] // scale factor for row i
			for j := 0; j < c; j++ {
				out.data[base+j] = d.data[base+j] * sf
			}
		}
		return out, nil
	}

	// Generic fallback.
	for i := 0; i < r; i++ {
		sf := scale[i] // row scale once per row
		for j := 0; j < c; j++ {
			v, e := X.At(i, j)
			if e != nil {
				return nil, matrixErrorf("scaleRows", e)
			}
			_ = out.Set(i, j, v*sf)
		}
	}
	return out, nil
}

// ewReplaceInfNaN copies X replacing any {±Inf, NaN} by val (finite).
// Time: O(r*c). Space: O(r*c). Deterministic flat loop on Dense fast-path.
func ewReplaceInfNaN(X Matrix, val float64) (Matrix, error) {
	// Validate input matrix.
	if err := ValidateNotNil(X); err != nil {
		return nil, matrixErrorf("ReplaceInfNaN", err)
	}
	// Validate 'val' is finite per numeric policy.
	if math.IsNaN(val) || math.IsInf(val, 0) {
		return nil, matrixErrorf("ReplaceInfNaN", ErrNaNInf)
	}
	// Read shape and allocate result.
	r, c := X.Rows(), X.Cols()
	out, err := NewDense(r, c)
	if err != nil {
		return nil, matrixErrorf("ReplaceInfNaN", err)
	}

	// Dense fast-path: direct flat slice iteration.
	if d, ok := X.(*Dense); ok {
		n := r * c
		for idx := 0; idx < n; idx++ {
			v := d.data[idx] // read element
			// Replace if not finite.
			if math.IsNaN(v) || math.IsInf(v, 0) {
				v = val
			}
			out.data[idx] = v // write element
		}
		return out, nil
	}

	// Generic fallback.
	for i := 0; i < r; i++ {
		for j := 0; j < c; j++ {
			v, e := X.At(i, j)
			if e != nil {
				return nil, matrixErrorf("ReplaceInfNaN", e)
			}
			if math.IsNaN(v) || math.IsInf(v, 0) {
				v = val
			}
			_ = out.Set(i, j, v) // bounds-safe write
		}
	}
	return out, nil
}

// ewClipRange copies X clamping each entry into [lo, hi] (both finite).
// Time: O(r*c). Space: O(r*c). Deterministic flat loop on Dense fast-path.
//
// Note: Bounds must be finite; if lo > hi, they are swapped (normalized).
func ewClipRange(X Matrix, lo, hi float64) (Matrix, error) {
	// Validate input matrix.
	if err := ValidateNotNil(X); err != nil {
		return nil, matrixErrorf("Clip", err)
	}
	// Require finite bounds (respect package numeric policy).
	if math.IsNaN(lo) || math.IsNaN(hi) || math.IsInf(lo, 0) || math.IsInf(hi, 0) {
		return nil, matrixErrorf("Clip", ErrNaNInf)
	}
	// Normalize bound order to avoid surprising errors.
	if lo > hi {
		lo, hi = hi, lo // swap
	}
	// Read shape and allocate output.
	r, c := X.Rows(), X.Cols()
	out, err := NewDense(r, c)
	if err != nil {
		return nil, matrixErrorf("Clip", err)
	}

	// Dense fast-path: single pass with branchy clamp (predictable).
	if d, ok := X.(*Dense); ok {
		n := r * c
		for idx := 0; idx < n; idx++ {
			v := d.data[idx] // read
			if v < lo {
				v = lo
			} else if v > hi {
				v = hi
			} // clamp into [lo,hi]
			out.data[idx] = v // write
		}
		return out, nil
	}

	// Generic fallback via At/Set.
	for i := 0; i < r; i++ {
		for j := 0; j < c; j++ {
			v, e := X.At(i, j)
			if e != nil {
				return nil, matrixErrorf("Clip", e)
			}
			if v < lo {
				v = lo
			} else if v > hi {
				v = hi
			}
			_ = out.Set(i, j, v)
		}
	}
	return out, nil
}

// AllClose checks element-wise |a-b| ≤ atol + rtol*|b| for identical shapes.
// Returns (true,nil) if all elements satisfy the relation; (false,nil) otherwise.
// Time: O(r*c). Space: O(1). Deterministic.
//
// Policy:
//   - a and b must be non-nil and have identical shapes.
//   - rtol, atol are treated as |rtol|, |atol| (negative values are normalized).
func ewAllClose(a, b Matrix, rtol, atol float64) (bool, error) {
	// Normalize tolerances to non-negative values (negative inputs are accepted but abs-ed).
	if math.IsNaN(rtol) || math.IsNaN(atol) || math.IsInf(rtol, 0) || math.IsInf(atol, 0) {
		return false, matrixErrorf("AllClose", ErrNaNInf) // invalid tolerance
	}
	if rtol < 0 {
		rtol = -rtol
	}
	if atol < 0 {
		atol = -atol
	}

	// Validate presence and shape equality using central validators.
	if err := ValidateNotNil(a); err != nil {
		return false, matrixErrorf("AllClose", err)
	}
	if err := ValidateNotNil(b); err != nil {
		return false, matrixErrorf("AllClose", err)
	}
	if err := ValidateSameShape(a, b); err != nil {
		return false, matrixErrorf("AllClose", err)
	}

	// Read shape once (O(1)).
	r, c := a.Rows(), a.Cols()

	// Dense fast-path: operate over flat slices when both are *Dense.
	if da, okA := a.(*Dense); okA {
		if db, okB := b.(*Dense); okB {
			n := r * c // total number of elements
			for idx := 0; idx < n; idx++ {
				// Compute absolute difference and RHS tolerance bound.
				diff := da.data[idx] - db.data[idx]
				if diff < 0 {
					diff = -diff
				} // |a-b|
				absb := db.data[idx]
				if absb < 0 {
					absb = -absb
				} // |b|
				// Check |a-b| ≤ atol + rtol*|b|.
				if diff > (atol + rtol*absb) {
					return false, nil // early-exit on first violation
				}
			}
			return true, nil // all ok
		}
	}

	// Generic fallback via At (bounds-safe; still deterministic).
	var av, bv, diff, absb float64
	for i := 0; i < r; i++ {
		for j := 0; j < c; j++ {
			av, _ = a.At(i, j) // read a(i,j)
			bv, _ = b.At(i, j) // read b(i,j)
			diff = av - bv     // difference
			if diff < 0 {
				diff = -diff
			} // abs
			absb = bv
			if absb < 0 {
				absb = -absb
			} // abs
			// Compare to tolerance threshold.
			if diff > (atol + rtol*absb) {
				return false, nil
			}
		}
	}

	return true, nil
}
