// SPDX-License-Identifier: MIT
// Package: matrix
//
// Purpose:
//   - Provide common statistical transforms (centering, normalization, covariance, correlation)
//     as thin, deterministic compositions over canonical kernels and ew* micro-kernels.
//   - Keep tight loops factored (ew*) and reuse existing matrix kernels (Mul/Transpose/Scale/MatVec).
//
// Exposed API:
//   - CenterColumns(X)  -> (Xc, means)           // subtract per-column mean
//   - CenterRows(X)     -> (Xc, means)           // subtract per-row mean
//   - NormalizeRowsL1(X)-> (Y, norms)            // L1 row normalization (degenerate rows stay zero)
//   - NormalizeRowsL2(X)-> (Y, norms)            // L2 row normalization (degenerate rows stay zero)
//   - Covariance(X)     -> (Cov, means)          // sample covariance of columns: (Xcᵀ Xc)/(n-1)
//   - Correlation(X)    -> (Corr, means, stds)   // Pearson corr via z-scoring; degenerate std=0 → zeroed column
//
// Determinism & Performance:
//   - Deterministic loop orders everywhere.
//   - O(r*c) for centering/normalization; O(r*c + c^2) for covariance/correlation (via Mul).
//   - Dense fast-paths used where appropriate; otherwise fall back to safe At/Set.
//
// AI-Hints:
//   - Reuse returned means/stds to unscale or to feed downstream models.
//   - For very tall-skinny X, prefer blocked/TSQR outside this package for better cache behavior.
//   - Sanitize inputs first (ReplaceInfNaN) to avoid NaN propagation in statistics.

package matrix

import "math"

// centerColumns: Xc = X − mean_by_col(X). Returns Xc and means.
// Time: O(r*c). Space: O(r*c).
func centerColumns(X Matrix) (Matrix, []float64, error) {
	// Validate presence of X.
	if err := ValidateNotNil(X); err != nil {
		return nil, nil, matrixErrorf("CenterColumns", err)
	}

	// Read dimensions once.
	r, c := X.Rows(), X.Cols()

	// Compute column sums via MatVec on transpose inside ColSums.
	colSums, err := ColSums(X)
	if err != nil {
		return nil, nil, matrixErrorf("CenterColumns", err)
	}

	// Convert sums to means (one pass; O(c)).
	means := make([]float64, c)
	if r > 0 {
		inv := 1.0 / float64(r) // precompute reciprocal
		for j := 0; j < c; j++ {
			means[j] = colSums[j] * inv
		}
	} // for r==0, means all-zeros is a harmless neutral

	// Subtract column means (broadcast over rows).
	Xc, err := ewBroadcastSubCols(X, means)
	if err != nil {
		return nil, nil, matrixErrorf("CenterColumns", err)
	}

	// Return centered matrix and means.
	return Xc, means, nil
}

// centerRows: subtract per-row mean. Returns Xc and row means.
// Time: O(r*c). Space: O(r*c).
func centerRows(X Matrix) (Matrix, []float64, error) {
	// Validate X presence.
	if err := ValidateNotNil(X); err != nil {
		return nil, nil, matrixErrorf("CenterRows", err)
	}

	// Read dimensions once.
	r, c := X.Rows(), X.Cols()

	// Compute per-row sums with MatVec inside RowSums.
	rowSums, err := RowSums(X)
	if err != nil {
		return nil, nil, matrixErrorf("CenterRows", err)
	}

	// Convert sums to means.
	means := make([]float64, r)
	if c > 0 {
		inv := 1.0 / float64(c)
		for i := 0; i < r; i++ {
			means[i] = rowSums[i] * inv
		}
	} // for c==0, means zeros

	// Subtract row means with broadcast over columns.
	Xc, err := ewBroadcastSubRows(X, means)
	if err != nil {
		return nil, nil, matrixErrorf("CenterRows", err)
	}

	// Return centered matrix and means.
	return Xc, means, nil
}

// normalizeRowsL1: each row i scaled to L1==1 when possible.
// Degenerate rows remain zero. Returns Y and per-row norms.
// Time: O(r*c). Space: O(r*c).
func normalizeRowsL1(X Matrix) (Matrix, []float64, error) {
	// Validate presence.
	if err := ValidateNotNil(X); err != nil {
		return nil, nil, matrixErrorf("NormalizeRowsL1", err)
	}

	// Read shape.
	r, c := X.Rows(), X.Cols()

	// Allocate norms slice.
	norms := make([]float64, r)

	// Fast-path when X is Dense: compute ||row||_1 via flat pass.
	if d, ok := X.(*Dense); ok {
		for i := 0; i < r; i++ {
			s := 0.0
			base := i * c
			for j := 0; j < c; j++ {
				v := d.data[base+j]
				if v < 0 {
					v = -v
				} // abs
				s += v
			}
			norms[i] = s
		}
	} else {
		// Fallback via At.
		for i := 0; i < r; i++ {
			s := 0.0
			for j := 0; j < c; j++ {
				v, e := X.At(i, j)
				if e != nil {
					return nil, nil, matrixErrorf("NormalizeRowsL1", e)
				}
				if v < 0 {
					v = -v
				}
				s += v
			}
			norms[i] = s
		}
	}

	// Build reciprocal scale factors per row (1/norm if norm>0 else 0).
	scale := make([]float64, r)
	for i := 0; i < r; i++ {
		if norms[i] > 0 {
			scale[i] = 1.0 / norms[i]
		} else {
			scale[i] = 0.0
		}
	}

	// Apply row-wise scaling.
	Y, err := ewScaleRows(X, scale)
	if err != nil {
		return nil, nil, matrixErrorf("NormalizeRowsL1", err)
	}

	// Return normalized matrix and original norms.
	return Y, norms, nil
}

// normalizeRowsL2: each row i scaled to L2==1 when possible.
// Time: O(r*c). Space: O(r*c).
func normalizeRowsL2(X Matrix) (Matrix, []float64, error) {
	// Validate presence.
	if err := ValidateNotNil(X); err != nil {
		return nil, nil, matrixErrorf("NormalizeRowsL2", err)
	}

	// Read shape.
	r, c := X.Rows(), X.Cols()

	// Allocate norms slice.
	norms := make([]float64, r)

	// Dense fast-path for ||row||_2.
	if d, ok := X.(*Dense); ok {
		for i := 0; i < r; i++ {
			sq := 0.0
			base := i * c
			for j := 0; j < c; j++ {
				v := d.data[base+j]
				sq += v * v
			}
			norms[i] = math.Sqrt(sq)
		}
	} else {
		// Generic fallback via At.
		for i := 0; i < r; i++ {
			sq := 0.0
			for j := 0; j < c; j++ {
				v, e := X.At(i, j)
				if e != nil {
					return nil, nil, matrixErrorf("NormalizeRowsL2", e)
				}
				sq += v * v
			}
			norms[i] = math.Sqrt(sq)
		}
	}

	// Build reciprocal scale factors.
	scale := make([]float64, r)
	for i := 0; i < r; i++ {
		if norms[i] > 0 {
			scale[i] = 1.0 / norms[i]
		} else {
			scale[i] = 0.0
		}
	}

	// Apply row-wise scaling.
	Y, err := ewScaleRows(X, scale)
	if err != nil {
		return nil, nil, matrixErrorf("NormalizeRowsL2", err)
	}

	// Return normalized matrix and original norms.
	return Y, norms, nil
}

// covariance: sample covariance of columns = (Xcᵀ Xc)/(n-1).
// Returns Cov and column means.
// Time: O(r*c + c^2).
func covariance(X Matrix) (Matrix, []float64, error) {
	// Validate presence.
	if err := ValidateNotNil(X); err != nil {
		return nil, nil, matrixErrorf("Covariance", err)
	}

	// Require at least two rows for unbiased sample covariance.
	r := X.Rows()
	if r < 2 {
		return nil, nil, matrixErrorf("Covariance", ErrDimensionMismatch)
	}

	// Center columns once.
	Xc, means, err := CenterColumns(X)
	if err != nil {
		return nil, nil, matrixErrorf("Covariance", err)
	}

	// Compute Cov = (Xcᵀ Xc)/(r-1).
	Xct, err := Transpose(Xc)
	if err != nil {
		return nil, nil, matrixErrorf("Covariance", err)
	}
	M, err := Mul(Xct, Xc)
	if err != nil {
		return nil, nil, matrixErrorf("Covariance", err)
	}
	Cov, err := Scale(M, 1.0/float64(r-1))
	if err != nil {
		return nil, nil, matrixErrorf("Covariance", err)
	}

	// Return covariance matrix and column means.
	return Cov, means, nil
}

// correlation: Pearson via z-score columns; degenerate std→zeroed column.
// Returns Corr, means, stds.
// Time: O(r*c + c^2).
func correlation(X Matrix) (Matrix, []float64, []float64, error) {
	// Validate presence.
	if err := ValidateNotNil(X); err != nil {
		return nil, nil, nil, matrixErrorf("Correlation", err)
	}

	// Require at least two rows for unbiased denominators.
	r, c := X.Rows(), X.Cols()
	if r < 2 {
		return nil, nil, nil, matrixErrorf("Correlation", ErrDimensionMismatch)
	}

	// Center columns and obtain means.
	Xc, means, err := CenterColumns(X)
	if err != nil {
		return nil, nil, nil, matrixErrorf("Correlation", err)
	}

	// Compute per-column sample std:
	//   std[j] = sqrt( Σ_i Xc[i,j]^2 / (r-1) ).
	stds := make([]float64, c)

	if d, ok := Xc.(*Dense); ok {
		// Dense fast-path: sum squares per column in a single pass.
		sumsq := make([]float64, c)
		for i := 0; i < r; i++ {
			base := i * c
			for j := 0; j < c; j++ {
				v := d.data[base+j]
				sumsq[j] += v * v
			}
		}
		inv := 1.0 / float64(r-1)
		for j := 0; j < c; j++ {
			stds[j] = math.Sqrt(sumsq[j] * inv)
		}
	} else {
		// Generic fallback via At.
		sumsq := make([]float64, c)
		var v float64
		for i := 0; i < r; i++ {
			for j := 0; j < c; j++ {
				v, err = Xc.At(i, j)
				if err != nil {
					return nil, nil, nil, matrixErrorf("Correlation", err)
				}
				sumsq[j] += v * v
			}
		}
		inv := 1.0 / float64(r-1)
		for j := 0; j < c; j++ {
			stds[j] = math.Sqrt(sumsq[j] * inv)
		}
	}

	// Build inverse std factors for z-scoring; degenerate std==0 → factor 0 (zero the column).
	invStd := make([]float64, c)
	for j := 0; j < c; j++ {
		if stds[j] > 0 {
			invStd[j] = 1.0 / stds[j]
		} else {
			invStd[j] = 0.0
		}
	}

	// Compute Z = Xc * diag(invStd) via broadcast scaling across columns.
	Z, err := ewScaleCols(Xc, invStd)
	if err != nil {
		return nil, nil, nil, matrixErrorf("Correlation", err)
	}

	// Corr = (Zᵀ Z)/(r-1).
	Zt, err := Transpose(Z)
	if err != nil {
		return nil, nil, nil, matrixErrorf("Correlation", err)
	}
	G, err := Mul(Zt, Z)
	if err != nil {
		return nil, nil, nil, matrixErrorf("Correlation", err)
	}
	Corr, err := Scale(G, 1.0/float64(r-1))
	if err != nil {
		return nil, nil, nil, matrixErrorf("Correlation", err)
	}

	// Return correlation matrix, means, stds.
	return Corr, means, stds, nil
}
