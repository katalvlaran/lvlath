// SPDX-License-Identifier: MIT
// Package: matrix
//
// Purpose:
//   - Provide common statistical transforms (centering, normalization, covariance, correlation)
//     as deterministic compositions over canonical kernels (Mul/Transpose/Scale) and ew* micro-kernels.
//   - Keep tight loops centralized in ew* where it improves reuse and consistency.
//
// Exposed API:
//   - CenterColumns(X)   -> (Xc, means)         // subtract per-column mean
//   - CenterRows(X)      -> (Xc, means)         // subtract per-row mean
//   - NormalizeRowsL1(X) -> (Y, norms)          // L1 row normalization (degenerate rows unchanged)
//   - NormalizeRowsL2(X) -> (Y, norms)          // L2 row normalization (degenerate rows unchanged)
//   - Covariance(X)      -> (Cov, means)        // sample covariance of columns: (Xcᵀ Xc)/(r-1)
//   - Correlation(X)     -> (Corr, means, stds) // Pearson corr via z-scoring; degenerate std=0 → zeroed column
//
// Determinism & Performance:
//   - Fixed i→j traversal for all explicit loops.
//   - Dense fast-paths avoid At/Set and operate on row-major flat buffers.
//   - Zero-size matrices (0×N or N×0) are treated as no-ops for centering/normalization.
//
// AI-Hints:
//   - Prefer passing *Dense to unlock flat-slice fast paths.
//   - Sanitize inputs first (ReplaceInfNaN) if NaN/Inf propagation is undesired in downstream statistics.

package matrix

import "math"

// Operation name constants for unified error wrapping and reducing magic strings.
const (
	opCenterColumns   = "CenterColumns"
	opCenterRows      = "CenterRows"
	opNormalizeRowsL1 = "NormalizeRowsL1"
	opNormalizeRowsL2 = "NormalizeRowsL2"
	opCovariance      = "Covariance"
	opCorrelation     = "Correlation"
)

// centerColumns subtracts the per-column mean from every element (column-wise centering).
// Implementation:
//   - Stage 1: Validate X (non-nil) and handle zero-size as a strict no-op.
//   - Stage 2: Compute column means in a deterministic pass (Dense fast-path; At fallback).
//   - Stage 3: Apply ewBroadcastSubCols to produce a centered copy.
//
// Behavior highlights:
//   - Zero-size (0×N or N×0): returns (X, zeroMeans, nil) without allocations.
//   - Deterministic i→j traversal; stable results.
//
// Inputs:
//   - X: input matrix (r×c).
//
// Returns:
//   - Matrix: centered copy (r×c) for r>0 && c>0; otherwise X itself (no-op).
//   - []float64: column means (len=c).
//
// Errors:
//   - ErrNilMatrix from validation.
//   - Wrapped At/NewDense/Set errors from fallback paths.
//
// Determinism:
//   - Fixed loop order; no randomness.
//
// Complexity:
//   - Time O(r*c), Space O(r*c) for output (+ O(c) means).
//
// Notes:
//   - Means are Σ_i X[i,j] / r for r>0.
//
// AI-Hints:
//   - For repeated centering, reuse the returned means to un-center later.
func centerColumns(X Matrix) (Matrix, []float64, error) {
	// Stage 1 (Validate): ensure X is present.
	if err := ValidateNotNil(X); err != nil {
		return nil, nil, matrixErrorf(opCenterColumns, err)
	}

	// Stage 1 (Zero-size policy): centering is a no-op when there are no elements.
	r, c := X.Rows(), X.Cols()
	means := make([]float64, c) // always return correct length for callers
	if r == 0 || c == 0 {
		return X, means, nil
	}

	// Stage 2 (Prepare): accumulate sums into means, then convert to averages.
	// We keep a single slice to avoid an extra allocation for "sums".
	var i, j int
	var v float64

	// Stage 2 (Execute): Dense fast-path uses the row-major flat buffer directly.
	if d, ok := X.(*Dense); ok {
		for i = 0; i < r; i++ { // deterministic row order
			base := i * c           // cache row base offset
			for j = 0; j < c; j++ { // deterministic column order
				means[j] += d.data[base+j] // accumulate sum for column j
			}
		}
	} else {
		// Stage 2 (Execute fallback): use At(i,j) with full error propagation.
		var err error
		for i = 0; i < r; i++ {
			for j = 0; j < c; j++ {
				v, err = X.At(i, j)
				if err != nil {
					return nil, nil, matrixErrorf(opCenterColumns, err)
				}
				means[j] += v
			}
		}
	}

	// Stage 3 (Finalize means): divide sums by r to obtain means.
	invR := 1.0 / float64(r)
	for j = 0; j < c; j++ {
		means[j] *= invR
	}

	// Stage 3 (Apply): broadcast-subtract the means over rows to build the centered copy.
	Xc, err := ewBroadcastSubCols(X, means)
	if err != nil {
		return nil, nil, matrixErrorf(opCenterColumns, err)
	}

	// Return centered matrix and means.
	return Xc, means, nil
}

// centerRows subtracts the per-row mean from every element (row-wise centering).
// Implementation:
//   - Stage 1: Validate X (non-nil) and handle zero-size as a strict no-op.
//   - Stage 2: Compute row means deterministically (Dense fast-path; At fallback).
//   - Stage 3: Apply ewBroadcastSubRows to produce a centered copy.
//
// Behavior highlights:
//   - Zero-size (0×N or N×0): returns (X, zeroMeans, nil) without allocations.
//   - Deterministic i→j traversal; stable results.
//
// Inputs:
//   - X: input matrix (r×c).
//
// Returns:
//   - Matrix: centered copy (r×c) for r>0 && c>0; otherwise X itself (no-op).
//   - []float64: row means (len=r).
//
// Errors:
//   - ErrNilMatrix from validation.
//   - Wrapped At/NewDense/Set errors from fallback paths.
//
// Determinism:
//   - Fixed loop order; no randomness.
//
// Complexity:
//   - Time O(r*c), Space O(r*c) for output (+ O(r) means).
//
// Notes:
//   - Means are Σ_j X[i,j] / c for c>0.
//
// AI-Hints:
//   - Useful for per-sample baselining before distance/similarity computations.
func centerRows(X Matrix) (Matrix, []float64, error) {
	// Stage 1 (Validate): ensure X is present.
	if err := ValidateNotNil(X); err != nil {
		return nil, nil, matrixErrorf(opCenterRows, err)
	}

	// Stage 1 (Zero-size policy): centering is a no-op when there are no elements.
	r, c := X.Rows(), X.Cols()
	means := make([]float64, r) // correct length for row means
	if r == 0 || c == 0 {
		return X, means, nil
	}

	// Stage 2 (Execute): compute mean per row.
	var i, j int
	var s, v float64

	if d, ok := X.(*Dense); ok {
		for i = 0; i < r; i++ {
			s = 0.0
			base := i * c
			for j = 0; j < c; j++ {
				s += d.data[base+j]
			}
			means[i] = s / float64(c)
		}
	} else {
		var err error
		for i = 0; i < r; i++ {
			s = 0.0
			for j = 0; j < c; j++ {
				v, err = X.At(i, j)
				if err != nil {
					return nil, nil, matrixErrorf(opCenterRows, err)
				}
				s += v
			}
			means[i] = s / float64(c)
		}
	} // for c==0, means zeros

	// Stage 3 (Apply): broadcast-subtract row means over columns.
	Xc, err := ewBroadcastSubRows(X, means)
	if err != nil {
		return nil, nil, matrixErrorf(opCenterRows, err)
	}

	// Return centered matrix and means.
	return Xc, means, nil
}

// normalizeRowsL1 scales each row to have L1-norm == 1 when possible.
// Implementation:
//   - Stage 1: Validate X (non-nil) and handle zero-size as a strict no-op.
//   - Stage 2: Compute per-row L1 norms deterministically.
//   - Stage 3: Build row scale factors (1/norm); for norm==0 use scale=1 to keep the row unchanged.
//   - Stage 4: Apply ewScaleRows to produce a normalized copy.
//
// Behavior highlights:
//   - Degenerate rows (norm==0) are left unchanged (stable policy).
//
// Inputs:
//   - X: input matrix (r×c).
//
// Returns:
//   - Matrix: normalized copy (r×c) for r>0 && c>0; otherwise X itself (no-op).
//   - []float64: L1 norms (len=r).
//
// Errors:
//   - ErrNilMatrix from validation.
//   - Wrapped At/NewDense/Set errors from fallback paths.
//
// Determinism:
//   - Fixed i→j traversal; no randomness.
//
// Complexity:
//   - Time O(r*c), Space O(r*c) for output (+ O(r) norms + O(r) scales).
//
// Notes:
//   - L1 norm is Σ_j |x_ij|.
//
// AI-Hints:
//   - Prefer L1 normalization for row-stochastic style preprocessing.
func normalizeRowsL1(X Matrix) (Matrix, []float64, error) {
	// Stage 1 (Validate): ensure X is present.
	if err := ValidateNotNil(X); err != nil {
		return nil, nil, matrixErrorf(opNormalizeRowsL1, err)
	}

	// Stage 1 (Zero-size policy): no elements => no-op.
	r, c := X.Rows(), X.Cols()
	norms := make([]float64, r)
	if r == 0 || c == 0 {
		return X, norms, nil
	}

	// Stage 2 (Execute): compute L1 norms per row.
	var i, j int
	var s, v float64

	if d, ok := X.(*Dense); ok {
		for i = 0; i < r; i++ {
			s = 0.0
			base := i * c
			for j = 0; j < c; j++ {
				v = d.data[base+j]
				if v < 0 {
					v = -v // abs
				}
				s += v
			}
			norms[i] = s
		}
	} else {
		var err error
		for i = 0; i < r; i++ {
			s = 0.0
			for j = 0; j < c; j++ {
				v, err = X.At(i, j)
				if err != nil {
					return nil, nil, matrixErrorf(opNormalizeRowsL1, err)
				}
				if v < 0 {
					v = -v
				}
				s += v
			}
			norms[i] = s
		}
	}

	// Stage 3 (Prepare scales): 1/norm for normal rows; 1 for degenerate rows (leave unchanged).
	scale := make([]float64, r)
	for i = 0; i < r; i++ {
		if norms[i] > 0 {
			scale[i] = 1.0 / norms[i]
		} else {
			scale[i] = 1.0 // preserves the row exactly (avoids "zeroing" underflow rows)
		}
	}

	// Stage 4 (Apply): scale rows via the canonical ew micro-kernel.
	Y, err := ewScaleRows(X, scale)
	if err != nil {
		return nil, nil, matrixErrorf(opNormalizeRowsL1, err)
	}

	// Return normalized matrix and original norms.
	return Y, norms, nil
}

// normalizeRowsL2 scales each row to have L2-norm == 1 when possible.
// Implementation:
//   - Stage 1: Validate X (non-nil) and handle zero-size as a strict no-op.
//   - Stage 2: Compute per-row L2 norms deterministically.
//   - Stage 3: Build row scale factors (1/norm); for norm==0 use scale=1 to keep the row unchanged.
//   - Stage 4: Apply ewScaleRows to produce a normalized copy.
//
// Behavior highlights:
//   - Degenerate rows (norm==0) are left unchanged (stable policy).
//
// Inputs/Returns/Errors/Determinism:
//   - Same structure as normalizeRowsL1.
//
// Complexity:
//   - Time O(r*c), Space O(r*c) (+ O(r) auxiliary slices).
//
// Notes:
//   - L2 norm is sqrt(Σ_j x_ij^2).
//
// AI-Hints:
//   - L2 normalization is typical before cosine similarity pipelines.
func normalizeRowsL2(X Matrix) (Matrix, []float64, error) {
	// Stage 1 (Validate): ensure X is present.
	if err := ValidateNotNil(X); err != nil {
		return nil, nil, matrixErrorf(opNormalizeRowsL2, err)
	}

	// Stage 1 (Zero-size policy): no elements => no-op.
	r, c := X.Rows(), X.Cols()
	norms := make([]float64, r)
	if r == 0 || c == 0 {
		return X, norms, nil
	}

	// Stage 2 (Execute): compute L2 norms per row.
	var i, j int
	var sq, v float64

	if d, ok := X.(*Dense); ok {
		for i = 0; i < r; i++ {
			sq = 0.0
			base := i * c
			for j = 0; j < c; j++ {
				v = d.data[base+j]
				sq += v * v
			}
			norms[i] = math.Sqrt(sq)
		}
	} else {
		var err error
		for i = 0; i < r; i++ {
			sq = 0.0
			for j = 0; j < c; j++ {
				v, err = X.At(i, j)
				if err != nil {
					return nil, nil, matrixErrorf(opNormalizeRowsL2, err)
				}
				sq += v * v
			}
			norms[i] = math.Sqrt(sq)
		}
	}

	// Stage 3 (Prepare scales): 1/norm for normal rows; 1 for degenerate rows (leave unchanged).
	scale := make([]float64, r)
	for i = 0; i < r; i++ {
		if norms[i] > 0 {
			scale[i] = 1.0 / norms[i]
		} else {
			scale[i] = 1.0
		}
	}

	// Stage 4 (Apply): scale rows via ew micro-kernel.
	Y, err := ewScaleRows(X, scale)
	if err != nil {
		return nil, nil, matrixErrorf(opNormalizeRowsL2, err)
	}

	// Return normalized matrix and original norms.
	return Y, norms, nil
}

// Computes sample covariance of columns: Cov = (Xcᵀ * Xc)/(r-1).
// Implementation:
//   - Stage 1: Validate X, require r>=2 (sample denominator).
//   - Stage 2: Center columns once; then accumulate outer-products with deterministic loops.
//
// Behavior highlights:
//   - Symmetric output; diagonal equals per-column sample variances.
//
// Inputs:
//   - X: Matrix (r×c), r>=2.
//
// Returns:
//   - Matrix: Covariance (c×c).
//   - []float64: column means used for centering.
//
// Errors:
//   - ErrNilMatrix, ErrDimensionMismatch (r<2), wrapped alloc/At/Set errors.
//
// Determinism:
//   - Fixed k→j→i accumulation (outer products), symmetric fill.
//
// Complexity:
//   - Time O(r*c + r*c^2), Space O(c^2).
//
// Notes:
//   - Uses direct loops to avoid dependency on external MatMul; leverages Dense fast-path.
//   - Result is positive semi-definite on well-formed data (modulo numeric noise).
//
// AI-Hints:
//   - Reuse means with downstream correlation.
//   - For very large c, consider block accumulation outside this package.
func covariance(X Matrix) (Matrix, []float64, error) {
	// Stage 1 (Validate): ensure X is present.
	if err := ValidateNotNil(X); err != nil {
		return nil, nil, matrixErrorf(opCovariance, err)
	}

	// Stage 1 (Validate shape): need r>=2 and c>0 for a meaningful sample covariance.
	r, c := X.Rows(), X.Cols()

	// Empty feature-set policy:
	// If there are no columns (c==0), the covariance is a valid degenerate 0×0 matrix.
	if c == 0 {
		z, err := NewDense(0, 0)
		if err != nil {
			return nil, nil, matrixErrorf(opCovariance, err)
		}
		return z, make([]float64, 0), nil
	}
	// Sample covariance requires at least two observations when c>0.
	if r < 2 {
		return nil, nil, matrixErrorf(opCovariance, ErrDimensionMismatch)
	}

	// Stage 2 (Center): reuse the canonical centering implementation.
	Xc, means, err := centerColumns(X)
	if err != nil {
		return nil, nil, matrixErrorf(opCovariance, err)
	}

	// Stage 3 (Compute): Cov = (Xcᵀ Xc)/(r-1) via canonical kernels.
	Xct, err := Transpose(Xc)
	if err != nil {
		return nil, nil, matrixErrorf(opCovariance, err)
	}
	G, err := Mul(Xct, Xc)
	if err != nil {
		return nil, nil, matrixErrorf(opCovariance, err)
	}
	Cov, err := Scale(G, 1.0/float64(r-1))
	if err != nil {
		return nil, nil, matrixErrorf(opCovariance, err)
	}

	return Cov, means, nil
}

// Compute Pearson correlation of columns via z-scoring: Corr = (Zᵀ Z)/(r-1),
// where Z = (X − mean) * diag(1/std). Degenerate std==0 → that column becomes all zeros.
// Implementation:
//   - Stage 1: Validate X, require r>=2; center columns (means).
//   - Stage 2: Compute sample stds per column; build invStd with 0 for degenerate columns.
//   - Stage 3: Z = Xc * diag(invStd) via ewScaleCols; Corr = (Zᵀ Z)/(r-1) via loops.
//
// Behavior highlights:
//   - Symmetric; diagonal is 1 for non-degenerate columns, 0 for degenerate (std==0).
//
// Inputs:
//   - X: Matrix (r×c), r>=2.
//
// Returns:
//   - Matrix: Correlation (c×c).
//   - []float64: column means.
//   - []float64: column stds (sample).
//
// Errors:
//   - ErrNilMatrix, ErrDimensionMismatch (r<2), wrapped alloc/At/Set errors.
//
// Determinism:
//   - Fixed accumulation order; stable output.
//
// Complexity:
//   - Time O(r*c + r*c^2), Space O(c^2).
//
// Notes:
//   - Scale-invariant: correlation(α*X) == correlation(X) for α>0.
//
// AI-Hints:
//   - correlation is scale-invariant: Corr(α*X) == Corr(X) for α>0 on non-degenerate columns.
//   - Degenerate columns (std==0) become zero columns/rows in Corr by construction.
func correlation(X Matrix) (Matrix, []float64, []float64, error) {
	// Stage 1 (Validate): ensure X is present.
	if err := ValidateNotNil(X); err != nil {
		return nil, nil, nil, matrixErrorf(opCorrelation, err)
	}

	// Stage 1 (Validate shape): need r>=2 and c>0.
	r, c := X.Rows(), X.Cols()
	// Empty feature-set policy:
	// If there are no columns (c==0), the correlation is a valid degenerate 0×0 matrix.
	if c == 0 {
		z, err := NewDense(0, 0)
		if err != nil {
			return nil, nil, nil, matrixErrorf(opCorrelation, err)
		}
		return z, make([]float64, 0), make([]float64, 0), nil
	}
	// Sample correlation requires at least two observations when c>0.
	if r < 2 {
		return nil, nil, nil, matrixErrorf(opCorrelation, ErrDimensionMismatch)
	}

	// Stage 2 (Center): subtract column means.
	Xc, means, err := centerColumns(X)
	if err != nil {
		return nil, nil, nil, matrixErrorf(opCorrelation, err)
	}

	// Stage 3 (Compute std): std[j] = sqrt( Σ_i Xc[i,j]^2 / (r-1) ).
	stds := make([]float64, c)
	sumsq := make([]float64, c) // accumulate squared sums deterministically
	inv := 1.0 / float64(r-1)

	var i, j int
	var v float64

	if d, ok := Xc.(*Dense); ok {
		for i = 0; i < r; i++ {
			base := i * c
			for j = 0; j < c; j++ {
				v = d.data[base+j]
				sumsq[j] += v * v
			}
		}
	} else {
		for i = 0; i < r; i++ {
			for j = 0; j < c; j++ {
				v, err = Xc.At(i, j)
				if err != nil {
					return nil, nil, nil, matrixErrorf(opCorrelation, err)
				}
				sumsq[j] += v * v
			}
		}
	}

	for j = 0; j < c; j++ {
		stds[j] = math.Sqrt(sumsq[j] * inv)
	}

	// Stage 4 (Build invStd): degenerate std==0 => invStd=0 (zero-out the column).
	invStd := make([]float64, c)
	for j = 0; j < c; j++ {
		if stds[j] > 0 {
			invStd[j] = 1.0 / stds[j]
		} else {
			invStd[j] = 0.0
		}
	}

	// Stage 5 (Z-score): Z = Xc * diag(invStd) via ewScaleCols.
	Z, err := ewScaleCols(Xc, invStd)
	if err != nil {
		return nil, nil, nil, matrixErrorf(opCorrelation, err)
	}

	// Stage 6 (Corr): Corr = (Zᵀ Z)/(r-1).
	Zt, err := Transpose(Z)
	if err != nil {
		return nil, nil, nil, matrixErrorf(opCorrelation, err)
	}
	G, err := Mul(Zt, Z)
	if err != nil {
		return nil, nil, nil, matrixErrorf(opCorrelation, err)
	}
	Corr, err := Scale(G, 1.0/float64(r-1))
	if err != nil {
		return nil, nil, nil, matrixErrorf(opCorrelation, err)
	}

	// Return correlation matrix, means, stds.
	return Corr, means, stds, nil
}
