// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

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
//
// Zero-shape law:
//   - 0×0, 0×N, and N×0 are valid structural matrices.
//   - Centering and row normalization treat zero-shape inputs as no-op transforms
//     and return correctly-sized metadata slices.
//   - Covariance/correlation return 0×0 for zero-feature inputs (c==0).
//   - Covariance/correlation still require r>=2 when c>0 because sample statistics
//     need one degree of freedom.
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

// statsDegenerateRowScale is the neutral scale factor for rows whose norm is exactly zero.
//   - A zero-norm row cannot be normalized to unit norm.
//   - The least surprising stable policy is to leave it unchanged.
//   - Using 1 avoids accidental mutation and keeps zero rows zero.
//
// Notes:
//   - This is a package policy constant, not an option.
//   - Making this configurable would split normalization semantics across callers.
//
// AI-Hints:
//   - Do not replace with 0: that would silently erase non-empty rows if a future
//     norm policy classifies a row as degenerate for tolerance reasons.
const statsDegenerateRowScale = 1.0

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
		// Zero-shape law:
		//   - 0×N has N column means, all vacuously 0.
		//   - N×0 has no columns and therefore no means.
		//   - The matrix itself is returned because there are no elements to transform.
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
		// Zero-shape law:
		//   - 0×N has no rows and therefore no row means.
		//   - N×0 has N row means, all vacuously 0.
		//   - The matrix itself is returned because there are no elements to transform.
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
		// Zero-shape law:
		//   - 0×N has no row norms.
		//   - N×0 has N row norms, all vacuously 0.
		//   - The matrix itself is returned because there are no elements to scale.
		return X, norms, nil
	}

	// Stage 2 (Execute): compute L1 norms per row.
	var i, j, base int
	var s, v float64
	var err error

	if d, ok := X.(*Dense); ok {
		for i = 0; i < r; i++ {
			s = 0.0
			base = i * c
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

	// Stage 3 (Prepare scales):
	//   - positive finite norm => scale by 1/norm;
	//   - exact zero norm      => leave the row unchanged;
	//   - non-finite norm      => error, because NaN/Inf is invalid data or overflow,
	//                             not a legitimate degenerate row.
	scale := make([]float64, r)
	for i = 0; i < r; i++ {
		scale[i], err = rowNormScale(norms[i])
		if err != nil {
			return nil, nil, matrixErrorf(opNormalizeRowsL1, err)
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
		// Zero-shape law:
		//   - 0×N has no row norms.
		//   - N×0 has N row norms, all vacuously 0.
		//   - The matrix itself is returned because there are no elements to scale.
		return X, norms, nil
	}

	// Stage 2 (Execute): compute L2 norms per row.
	var i, j int
	var sq, v float64
	var err error

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

	// Stage 3 (Prepare scales):
	//   - positive finite norm => scale by 1/norm;
	//   - exact zero norm      => leave the row unchanged;
	//   - non-finite norm      => error. For L2 this can happen through overflow
	//                             in Σ v² even when individual inputs are finite.
	scale := make([]float64, r)
	for i = 0; i < r; i++ {
		scale[i], err = rowNormScale(norms[i])
		if err != nil {
			return nil, nil, matrixErrorf(opNormalizeRowsL2, err)
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

// covariance computes sample covariance of columns:
//
//	Cov = (Xcᵀ Xc)/(r-1)
//
// Implementation:
//   - Stage 1: Validate X.
//   - Stage 2: If c==0, return a valid 0×0 covariance with empty means.
//   - Stage 3: If c>0, require r>=2 for the sample denominator.
//   - Stage 4: Center columns once.
//   - Stage 5: Compute Cov through Transpose → Mul → Scale.
//
// Behavior highlights:
//   - Zero-feature input is valid and returns 0×0.
//   - Positive-feature input requires at least two observations.
//   - Symmetric output; diagonal equals per-column sample variances.
//
// Inputs:
//   - X: Matrix (r×c). If c>0, r must be >=2.
//
// Returns:
//   - Matrix: covariance matrix with shape c×c.
//   - []float64: column means used for centering, length c.
//
// Errors:
//   - ErrNilMatrix.
//   - ErrDimensionMismatch when c>0 and r<2.
//   - wrapped allocation/kernel errors.
//
// Determinism:
//   - Fixed kernel order through CenterColumns, Transpose, Mul, Scale.
//
// Complexity:
//   - Time O(r*c + r*c²), Space O(r*c + c²).
//
// Notes:
//   - The c==0 case is not a “missing data” error; it is the covariance over an
//     empty feature set.
//
// AI-Hints:
//   - Do not move the r<2 check before the c==0 check.
//   - Empty feature covariance is valid even when r<2.
func covariance(X Matrix) (Matrix, []float64, error) {
	// Stage 1 (Validate): ensure X is present.
	if err := ValidateNotNil(X); err != nil {
		return nil, nil, matrixErrorf(opCovariance, err)
	}

	// Stage 1 (Validate shape): need r>=2 and c>0 for a meaningful sample covariance.
	r, c := X.Rows(), X.Cols()

	// Zero-feature law:
	// A matrix with no columns has an empty feature set. Its covariance lives in
	// feature-space, therefore the mathematically correct result shape is 0×0.
	// This remains valid even when r<2 because no sample variance is requested.
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

// correlation computes Pearson correlation of columns via z-scoring:
//
//	Z = (X - mean) / std
//	Corr = (Zᵀ Z)/(r-1)
//
// Degenerate std==0 columns are mapped to zero columns in Z.
//
// Implementation:
//   - Stage 1: Validate X.
//   - Stage 2: If c==0, return valid 0×0 correlation with empty means/stds.
//   - Stage 3: If c>0, require r>=2 for sample std/correlation.
//   - Stage 4: Center columns.
//   - Stage 5: Compute sample stds.
//   - Stage 6: Build inverse stds, zeroing degenerate columns.
//   - Stage 7: Compute Corr through ScaleCols, Transpose, Mul, Scale.
//
// Behavior highlights:
//   - Zero-feature input is valid and returns 0×0.
//   - Positive-feature input requires at least two observations.
//   - Degenerate finite columns get correlation row/column 0 by construction.
//
// Inputs:
//   - X: Matrix (r×c). If c>0, r must be >=2.
//
// Returns:
//   - Matrix: correlation matrix with shape c×c.
//   - []float64: column means, length c.
//   - []float64: sample standard deviations, length c.
//
// Errors:
//   - ErrNilMatrix.
//   - ErrDimensionMismatch when c>0 and r<2.
//   - wrapped allocation/kernel errors.
//
// Determinism:
//   - Fixed accumulation and kernel order.
//
// Complexity:
//   - Time O(r*c + r*c²), Space O(r*c + c²).
//
// Notes:
//   - The c==0 case has no feature pairs to evaluate, so 0×0 is the correct
//     structural result.
//   - Constant columns are not errors; their std is 0 and their normalized column is 0.
//
// AI-Hints:
//   - Do not move the r<2 check before the c==0 check.
//   - Do not set inverse std of a degenerate column to 1; that would falsely
//     preserve a non-informative centered column.
func correlation(X Matrix) (Matrix, []float64, []float64, error) {
	// Stage 1 (Validate): ensure X is present.
	if err := ValidateNotNil(X); err != nil {
		return nil, nil, nil, matrixErrorf(opCorrelation, err)
	}

	// Stage 1 (Validate shape): need r>=2 and c>0.
	r, c := X.Rows(), X.Cols()
	// Zero-feature law:
	// Correlation is defined over feature pairs. With zero features there are no
	// pairs to evaluate, so the correct structural result is a valid 0×0 matrix
	// with empty means/stds.
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

// ---------- helpers ----------

// rowNormScale converts a computed row norm into a normalization scale.
//
// What:
//   - finite positive norm => 1/norm;
//   - exact zero norm      => statsDegenerateRowScale;
//   - NaN/+Inf/-Inf norm   => ErrNaNInf.
//
// Why:
//   - Zero rows should remain unchanged.
//   - Non-finite norms are not legitimate degeneracy; they indicate invalid
//     input or floating-point overflow and must not be silently treated as a
//     zero/degenerate row.
//
// Errors:
//   - ErrNaNInf when norm is NaN or ±Inf.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// AI-Hints:
//   - Keep this helper shared by L1 and L2 normalization.
//   - Do not inline the norm==0 policy in each normalizer.
func rowNormScale(norm float64) (float64, error) {
	if isNonFinite(norm) {
		return 0, ErrNaNInf
	}
	if norm == 0 {
		return statsDegenerateRowScale, nil
	}

	return 1.0 / norm, nil
}
