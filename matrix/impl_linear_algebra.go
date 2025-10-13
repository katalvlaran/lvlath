// SPDX-License-Identifier: MIT
// Package matrix provides universal operations on any Matrix implementation,
// including element-wise addition, subtraction, matrix multiplication,
// transpose, and scalar scaling. All functions perform strict
// fail-fast validation and return clear errors on dimension mismatches.
//
// Purpose:
//   - Declare canonical linear-algebra kernels (signatures) used across the package.
//   - Define operation tags and shared constants for determinism and error reporting.
//
// Notes:
//   - Implementations live in dedicated kernel files (same package) to keep roles clean.
//   - All kernels must use central validators and return plain sentinels or wrapped via matrixErrorf at the facade.

package matrix

import (
	"fmt"
	"math"
)

// NormZero is the additive identity for norm and accumulation operations.
const NormZero = 0.0

// ZeroSum is the initial sum value for forward/backward substitution and similar.
const ZeroSum = 0.0

// ZeroPivot is the sentinel for detecting a zero pivot in LU/Inverse routines.
const ZeroPivot = 0.0

// Operation name constants for unified error wrapping and reducing magic strings.
const (
	opAdd       = "Add"
	opSub       = "Sub"
	opMul       = "Mul"
	opTranspose = "Transpose"
	opScale     = "Scale"
	opEigen     = "Eigen"
	opInverse   = "Inverse"
	opLU        = "LU"
	opQR        = "QR"
	opHadamard  = "Hadamard"
	opMatVec    = "MatVec"
)

// matrixErrorf wraps an underlying error with the given tag.
func matrixErrorf(tag string, err error) error {
	return fmt.Errorf("%s: %w", tag, err)
}

// Add returns a new Matrix containing the element-wise sum of a and b.
//
// Contract:
//   - a, b must be non-nil and have identical shapes.
//
// Determinism & Performance:
//   - Loop order is fixed (flat 0..n-1 in fast path; i→j in fallback).
//   - Single allocation for the result; no temps inside loops.
//
// Complexity: Time O(r*c), Space O(r*c).
//
// AI-Hints:
//   - If both operands are *Dense, pass them directly to avoid interface dispatch.
//   - ValidateSameShape catches shape bugs early and keeps inner loops branchless.
func Add(a, b Matrix) (Matrix, error) {
	// Validate inputs non-nil
	if err := ValidateNotNil(a); err != nil {
		return nil, matrixErrorf(opAdd, err)
	}
	if err := ValidateNotNil(b); err != nil {
		return nil, matrixErrorf(opAdd, err)
	}
	// Validate shapes match
	if err := ValidateSameShape(a, b); err != nil {
		return nil, matrixErrorf(opAdd, err)
	}

	// Allocate result Dense
	rows, cols := a.Rows(), a.Cols()
	res, err := NewDense(rows, cols)
	if err != nil {
		return nil, matrixErrorf(opAdd, err)
	}

	// Fast path: *Dense × *Dense → single flat loop.
	if da, okA := a.(*Dense); okA {
		if db, okB := b.(*Dense); okB {
			// direct element-wise addition on backing slices
			length := rows * cols
			for idx := 0; idx < length; idx++ { // deterministic 0..n-1
				res.data[idx] = da.data[idx] + db.data[idx]
			}

			return res, nil
		}
	}

	// Fallback: interface path with fixed i→j order.
	var i, j int
	var av, bv float64
	for i = 0; i < rows; i++ {
		for j = 0; j < cols; j++ {
			av, _ = a.At(i, j)       // safe: bounds ensured
			bv, _ = b.At(i, j)       // safe: same shape
			_ = res.Set(i, j, av+bv) // safe: within bounds
		}
	}

	// Return result
	return res, nil
}

// Sub returns a new Matrix with the element-wise difference a - b.
//
// Contract: non-nil inputs, identical shapes.
// Determinism: fixed loop order (fast: flat; fallback: i→j).
// Complexity: Time O(r*c), Space O(r*c).
//
// AI-Hints:
//   - Use *Dense fast path for heavy workloads.
//   - Keep inputs immutable; this routine allocates a fresh result.
func Sub(a, b Matrix) (Matrix, error) {
	// Validate inputs non-nil
	if err := ValidateNotNil(a); err != nil {
		return nil, matrixErrorf(opSub, err)
	}
	if err := ValidateNotNil(b); err != nil {
		return nil, matrixErrorf(opSub, err)
	}
	// Validate shapes match
	if err := ValidateSameShape(a, b); err != nil {
		return nil, matrixErrorf(opSub, err)
	}

	// Allocate result Dense
	rows, cols := a.Rows(), a.Cols()
	res, err := NewDense(rows, cols)
	if err != nil {
		return nil, matrixErrorf(opSub, err)
	}

	// Fast-path for two Dense matrices
	if da, okA := a.(*Dense); okA {
		if db, okB := b.(*Dense); okB {
			// direct element-wise addition on backing slices
			length := rows * cols
			for idx := 0; idx < length; idx++ {
				res.data[idx] = da.data[idx] - db.data[idx]
			}

			return res, nil
		}
	}

	// Fallback: generic interface loop
	var (
		i, j   int // loop iterators
		av, bv float64
	)
	for i = 0; i < rows; i++ {
		for j = 0; j < cols; j++ {
			av, _ = a.At(i, j)       // safe: bounds ensured
			bv, _ = b.At(i, j)       // safe: same shape
			_ = res.Set(i, j, av-bv) // safe: within bounds
		}
	}

	// Return result
	return res, nil
}

// Mul performs standard matrix multiplication c = a × b.
//
// Contract:
//   - a, b non-nil; a.Cols() == b.Rows().
//
// Determinism & Performance:
//   - Fast path (*Dense×*Dense) uses fixed i→k→j with row-major strides.
//   - Fallback uses fixed i→j→k; both orders are stable across runs.
//
// Complexity: Time O(r*n*c), Space O(r*c).
//
// AI-Hints:
//   - Skip zeros in the inner loop to reduce multiplications on sparse-like rows.
//   - Favor *Dense inputs to unlock cache-friendly flat loops.
func Mul(a, b Matrix) (Matrix, error) {
	// Validate inputs
	if err := ValidateNotNil(a); err != nil {
		return nil, matrixErrorf(opMul, err)
	}
	if err := ValidateNotNil(b); err != nil {
		return nil, matrixErrorf(opMul, err)
	}
	if a.Cols() != b.Rows() {
		return nil, matrixErrorf(opMul, ErrDimensionMismatch)
	}

	// Allocate result Dense
	aRows, aCols, bCols := a.Rows(), a.Cols(), b.Cols()
	res, err := NewDense(aRows, bCols)
	if err != nil {
		return nil, matrixErrorf(opMul, err)
	}
	var (
		i, j, k         int // loop iterators
		av, bv, current float64
	)
	// Fast-path for two Dense matrices
	if da, okA := a.(*Dense); okA {
		if db, okB := b.(*Dense); okB {
			// row-major multiplication into res.data
			// da.data layout: i*aCols + k
			// db.data layout: k*bCols + j
			var rowOffsetA, rowOffsetB, rowOffsetR int
			for i = 0; i < aRows; i++ {
				rowOffsetA = i * aCols
				rowOffsetR = i * bCols
				for k = 0; k < aCols; k++ {
					av = da.data[rowOffsetA+k]
					if av == 0 {
						continue // skip zero for performance
					}
					rowOffsetB = k * bCols
					for j = 0; j < bCols; j++ {
						res.data[rowOffsetR+j] += av * db.data[rowOffsetB+j]
					}
				}
			}
			return res, nil
		}
	}

	// Fallback: generic interface triple-loop (i-j-k)
	for i = 0; i < aRows; i++ {
		for j = 0; j < bCols; j++ {
			current = 0.0
			for k = 0; k < aCols; k++ {
				av, _ = a.At(i, k)
				if av == 0 {
					continue // skip zero for performance
				}
				bv, _ = b.At(k, j)
				current += av * bv // accumulate product
			}
			_ = res.Set(i, j, current)
		}
	}

	// Return result
	return res, nil
}

// Transpose returns a new Matrix with rows and columns swapped.
//
// Contract: m non-nil.
// Determinism: fixed i→j; fast path copies via flat indices.
// Complexity: Time O(r*c), Space O(r*c).
//
// AI-Hints:
//   - Transpose of *Dense is fastest with flat slice copies.
//   - For small matrices the generic path is fine.
func Transpose(m Matrix) (Matrix, error) {
	// Validate input non-nil
	if err := ValidateNotNil(m); err != nil {
		return nil, matrixErrorf(opTranspose, err)
	}

	// Allocate result Dense with flipped dimensions
	rows, cols := m.Rows(), m.Cols()
	res, err := NewDense(cols, rows) // dims flipped
	if err != nil {
		return nil, matrixErrorf(opTranspose, err)
	}

	// Fast-path for Dense → Dense
	var i, j int // loop iterators
	if dm, ok := m.(*Dense); ok {
		// data[i*cols + j] → res.data[j*rows + i]
		var baseSrc int
		for i = 0; i < rows; i++ {
			baseSrc = i * cols
			for j = 0; j < cols; j++ {
				res.data[j*rows+i] = dm.data[baseSrc+j]
			}
		}
		return res, nil
	}

	// Fallback: generic interface loop
	var v float64
	for i = 0; i < rows; i++ {
		for j = 0; j < cols; j++ {
			v, _ = m.At(i, j)    // safe: bounds ensured
			_ = res.Set(j, i, v) // safe: within bounds
		}
	}

	// Return result
	return res, nil
}

// Scale returns a new Matrix with each element of m multiplied by alpha.
//
// Contract: m non-nil.
// Determinism: flat loop (fast) or i→j (fallback).
// Complexity: Time O(r*c), Space O(r*c).
//
// AI-Hints:
//   - If you only need a view-like behavior, consider deferring scaling
//     to the next kernel to avoid an extra allocation.
func Scale(m Matrix, alpha float64) (Matrix, error) {
	// Validate input non-nil
	if err := ValidateNotNil(m); err != nil {
		return nil, matrixErrorf(opScale, err)
	}

	// Allocate result Dense
	rows, cols := m.Rows(), m.Cols()
	res, err := NewDense(rows, cols)
	if err != nil {
		return nil, matrixErrorf(opScale, err)
	}

	// Fast-path for Dense → Dense
	if dm, ok := m.(*Dense); ok {
		n := rows * cols
		for idx := 0; idx < n; idx++ {
			res.data[idx] = dm.data[idx] * alpha
		}
		return res, nil
	}

	// Fallback: generic interface loop
	var i, j int
	var v float64
	for i = 0; i < rows; i++ {
		for j = 0; j < cols; j++ {
			v, _ = m.At(i, j)          // safe: bounds ensured
			_ = res.Set(i, j, v*alpha) // safe: within bounds
		}
	}

	// Return result
	return res, nil
}

// Hadamard returns element-wise product a ⊙ b as a new Matrix (Dense).
//
// Contract: a,b non-nil; identical shapes.
// Fast-path: *Dense×*Dense runs a single flat loop 0..n-1 (deterministic).
// Determinism: flat loop (fast) or i→j (fallback).
// Complexity: Time O(r*c), Space O(r*c).
//
// AI-Hints:
//   - Prefer *Dense operands to exploit flat-slice throughput.
//   - This is bandwidth-bound; keep data contiguous and avoid tiny tiles.
func Hadamard(a, b Matrix) (Matrix, error) {
	// Validate 'a' is not nil.
	if err := ValidateNotNil(a); err != nil {
		return nil, matrixErrorf(opHadamard, err)
	}
	// Validate 'b' is not nil.
	if err := ValidateNotNil(b); err != nil {
		return nil, matrixErrorf(opHadamard, err)
	}
	// Validate shapes match exactly.
	if err := ValidateSameShape(a, b); err != nil {
		return nil, matrixErrorf(opHadamard, err)
	}

	// Allocate the result Dense with the same shape.
	rows, cols := a.Rows(), a.Cols()
	res, err := NewDense(rows, cols)
	if err != nil {
		return nil, matrixErrorf(opHadamard, err)
	}

	// Fast-path: both operands are *Dense → operate on flat slices directly.
	if da, okA := a.(*Dense); okA {
		if db, okB := b.(*Dense); okB {
			var n, idx int                // predeclare loop variables to avoid per-iteration allocations
			n = rows * cols               // total number of elements
			for idx = 0; idx < n; idx++ { // fixed order ensures deterministic accumulation
				res.data[idx] = da.data[idx] * db.data[idx] // element-wise product
			}

			return res, nil // return immediately on fast-path
		}
	}

	// Fallback: generic interface loop using At/Set (bounds-safe, shape already validated).
	var i, j int // loop indices (predeclared)
	var av, bv float64
	for i = 0; i < rows; i++ { // fixed i-outer loop
		for j = 0; j < cols; j++ { // fixed j-inner loop
			av, _ = a.At(i, j)       // read a(i,j)
			bv, _ = b.At(i, j)       // read b(i,j)
			_ = res.Set(i, j, av*bv) // write result(i,j); Set is safe w.r.t. bounds/policy
		}
	}

	// Return the computed result (Dense implements Matrix).
	return res, nil
}

// MatVec computes y = m * x for a column vector x.
//
// Contract: m non-nil; x non-nil; len(x) == m.Cols().
// Fast-path: *Dense performs one pass per row with flat indexing.
// Determinism: fixed i→j loop order.
// Complexity: Time O(r*c), Space O(r) for y.
//
// AI-Hints:
//   - Use *Dense to keep a single pass per row with flat indexing.
//   - Skipping zero x[j] helps when x is sparse-ish.
func MatVec(m Matrix, x []float64) ([]float64, error) {
	// Validate m is not nil.
	if err := ValidateNotNil(m); err != nil {
		return nil, matrixErrorf(opMatVec, err)
	}
	// Validate x is not nil and match with number of columns
	if err := ValidateVecLen(x, m.Cols()); err != nil {
		return nil, matrixErrorf(opMatVec, err)
	}
	// Prepare result vector y with length rows.
	rows, cols := m.Rows(), m.Cols()
	y := make([]float64, rows) // allocate exactly rows outputs

	// Fast-path: *Dense allows flat, row-major dot-products.
	if d, ok := m.(*Dense); ok {
		var i, j, base int // indices and row base offset
		var acc, xv float64
		for i = 0; i < d.r; i++ { // iterate rows deterministically
			acc = 0                   // reset accumulator per row
			base = i * d.c            // compute flat base offset for row i
			for j = 0; j < d.c; j++ { // iterate columns
				xv = x[j]    // read x(j) once per iteration
				if xv != 0 { // micro-optimization: skip zero multiplications
					acc += d.data[base+j] * xv // accumulate a(i,j)*x(j)
				}
			}
			y[i] = acc // store y(i)
		}

		return y, nil // return on fast-path
	}

	// Fallback: interface-based dot-products via At.
	var i, j int               // loop indices
	var mv float64             // temporary to hold m(i,j)
	for i = 0; i < rows; i++ { // iterate rows
		y[i] = 0                   // initialize y(i) to zero
		for j = 0; j < cols; j++ { // iterate columns
			mv, _ = m.At(i, j) // read m(i,j)
			y[i] += mv * x[j]  // accumulate
		}
	}

	return y, nil // return computed vector
}

// Eigen performs Jacobi eigen-decomposition on a symmetric matrix m.
// It returns eigenvalues and eigenvectors Q (columns of Q).
//
// Contract:
//   - m non-nil and square; symmetry within tol (|A[i,j]-A[j,i]| ≤ tol).
//
// Determinism & Performance:
//   - Pivot selection scans upper triangle in fixed i→j order.
//   - Rotations are applied in fixed order; tie-breaking is stable.
//   - Fast path uses *Dense for data-parallel updates.
//
// Complexity: Time O(maxIter * n^3), Space O(n^2).
//
// AI-Hints:
//   - Choose tol ~ 1e-9..1e-12 for double; cap maxIter to avoid stalls.
//   - Precondition by symmetrizing if input comes from numerically noisy ops.
func Eigen(m Matrix, tol float64, maxIter int) ([]float64, Matrix, error) {
	// Validate: notNil; Square; Symmetric;
	if err := ValidateSymmetric(m, tol); err != nil {
		return nil, nil, matrixErrorf(opEigen, err) // unify error wrapping
	}
	// Prepare working copy A and orthogonal accumulator Q
	n := m.Rows()               // n - number of rows (and columns), cols - number of columns
	aRaw := m.Clone()           // aRaw is a working copy of m to avoid modifying the original
	qRaw, err := NewDense(n, n) // qRaw is a newly allocated zero dense matrix
	var i, j int                // loop iterators over rows and columns
	if err != nil {
		return nil, nil, matrixErrorf(opEigen, err)
	}
	// Initialize Q as identity: Q[i,i] = 1
	for i = 0; i < n; i++ {
		_ = qRaw.Set(i, i, 1.0)
	}

	// Detect if we can use fast-path on *Dense
	// if aRaw is actually *Dense, then useFast=true
	Adense, useFast := aRaw.(*Dense)

	// Jacobi rotations
	var (
		iter               int     // iteration counter
		base               int     // helper offset into the flat data slice
		p, q               int     // current pivot indices
		maxOff, off        float64 // maxOff - current max |A[p,q]|; off - temporary
		app, aqq           float64 // diagonal entries A[p,p], A[q,q]
		aip, aiq, qip, qiq float64 // temporaries for A[i,p], A[i,q] and Q[i,p], Q[i,q]
		new_ip, new_iq     float64 // updated values for A[i,p] and A[i,q]
		apq                float64 // off-diagonal entry A[p,q]
		theta, t           float64 // intermediate rotation parameters
		c, s               float64 // cosine and sine of the rotation angle
	)
	for iter = 0; iter < maxIter; iter++ {
		// J.1: Find pivot (p,q) maximizing |A[p,q]|
		maxOff = 0.0
		if useFast {
			// fast-path: operate directly on data []float64
			for i = 0; i < n; i++ {
				base = i * n
				for j = i + 1; j < n; j++ {
					// off = |A[i,j]|
					off = math.Abs(Adense.data[base+j])
					if off > maxOff {
						maxOff, p, q = off, i, j
					}
				}
			}
		} else {
			// fallback: interface-based path via At
			for i = 0; i < n; i++ {
				for j = i + 1; j < n; j++ {
					off, _ = aRaw.At(i, j)
					off = math.Abs(off)
					if off > maxOff {
						maxOff, p, q = off, i, j
					}
				}
			}
		}

		// J.2: Check convergence: if maxOff < tol, break
		if maxOff < tol {
			break
		}

		// J.3: Compute rotation parameters from A[p,p], A[q,q], A[p,q]
		if useFast {
			app = Adense.data[p*n+p]
			aqq = Adense.data[q*n+q]
			apq = Adense.data[p*n+q]
		} else {
			app, _ = aRaw.At(p, p)
			aqq, _ = aRaw.At(q, q)
			apq, _ = aRaw.At(p, q)
		}
		// θ = (aqq−app)/(2*apq)
		theta = (aqq - app) / (2 * apq)
		// t = sign(θ) / (|θ|+√(θ²+1))
		t = math.Copysign(1.0/(math.Abs(theta)+math.Sqrt(theta*theta+1)), theta)
		// c = 1/√(1+t²), s = t*c
		c = 1.0 / math.Sqrt(t*t+1)
		s = t * c

		// J.4: Apply rotation to A
		if useFast {
			// fast-path: update two pairs of elements in data at once
			for i = 0; i < n; i++ {
				if i == p || i == q {
					continue
				}
				// original A[i,p], A[i,q]
				aip = Adense.data[i*n+p]
				aiq = Adense.data[i*n+q]
				// new values
				new_ip = c*aip - s*aiq
				new_iq = s*aip + c*aiq
				// assign symmetrically to [i,p] and [p,i], [i,q] and [q,i]
				Adense.data[i*n+p], Adense.data[p*n+i] = new_ip, new_ip
				Adense.data[i*n+q], Adense.data[q*n+i] = new_iq, new_iq
			}
			// update diagonals and zero out A[p,q], A[q,p]
			Adense.data[p*n+p] = c*c*app - 2*c*s*apq + s*s*aqq
			Adense.data[q*n+q] = s*s*app + 2*c*s*apq + c*c*aqq
			Adense.data[p*n+q], Adense.data[q*n+p] = 0, 0
		} else {
			// fallback via At/Set
			for i = 0; i < n; i++ {
				if i == p || i == q {
					continue
				}
				aip, _ = aRaw.At(i, p)
				aiq, _ = aRaw.At(i, q)
				_ = aRaw.Set(i, p, c*aip-s*aiq)
				_ = aRaw.Set(p, i, c*aip-s*aiq)
				_ = aRaw.Set(i, q, s*aip+c*aiq)
				_ = aRaw.Set(q, i, s*aip+c*aiq)
			}
			_ = aRaw.Set(p, p, c*c*app-2*c*s*apq+s*s*aqq)
			_ = aRaw.Set(q, q, s*s*app+2*c*s*apq+c*c*aqq)
			_ = aRaw.Set(p, q, 0.0)
			_ = aRaw.Set(q, p, 0.0)
		}

		// J.5: Accumulate rotation into Q
		if useFast {
			// here qRaw is also expected to be *Dense, but this works anyway
			for i = 0; i < n; i++ {
				qip = qRaw.data[i*n+p] // Q[i,p]
				qiq = qRaw.data[i*n+q] // Q[i,q]
				qRaw.data[i*n+p] = c*qip - s*qiq
				qRaw.data[i*n+q] = s*qip + c*qiq
			}
		} else {
			for i = 0; i < n; i++ {
				qip, _ = qRaw.At(i, p)
				qiq, _ = qRaw.At(i, q)
				_ = qRaw.Set(i, p, c*qip-s*qiq)
				_ = qRaw.Set(i, q, s*qip+c*qiq)
			}
		}
	}

	// Check convergence
	// after exiting the loop, recompute maxOff to ensure convergence
	maxOff = 0
	for i = 0; i < n; i++ {
		for j = i + 1; j < n; j++ {
			off, _ = aRaw.At(i, j)
			if m := math.Abs(off); m > maxOff {
				maxOff = m
			}
		}
	}
	if maxOff >= tol {
		return nil, nil, matrixErrorf(opEigen, ErrMatrixEigenFailed)
	}

	// Extract eigenvalues from diagonal of A
	eigs := make([]float64, n)
	if useFast {
		for i = 0; i < n; i++ {
			eigs[i] = Adense.data[i*n+i]
		}
	} else {
		var v float64
		for i = 0; i < n; i++ {
			v, _ = aRaw.At(i, i)
			eigs[i] = v
		}
	}

	// Return eigenvalues and eigenvectors
	return eigs, qRaw, nil
}

// Inverse computes A^{-1} via Doolittle LU without pivoting (deterministic).
//
// Contract: m non-nil and square; ErrSingular on zero pivot.
//
// Determinism & Performance:
//   - Fixed loop orders for forward/backward substitution.
//   - Fast path for *Dense avoids interface dispatch.
//
// Complexity: Time O(n^3), Space O(n^2).
//
// AI-Hints:
//   - Upstream pivoting changes numeric stability; we intentionally keep none
//     for determinism. Detect near-zero pivots before calling if needed.
func Inverse(m Matrix) (Matrix, error) {
	// Validate input non‐nil and square
	if err := ValidateNotNil(m); err != nil {
		return nil, matrixErrorf(opInverse, err)
	}
	if err := ValidateSquare(m); err != nil {
		return nil, matrixErrorf(opInverse, err)
	}

	// LU decomposition (Doolittle)
	Lmat, Umat, err := LU(m)
	if err != nil {
		return nil, matrixErrorf(opInverse, err)
	}

	// Prepare result container and scratch arrays
	n := m.Rows()
	invDense, err := NewDense(n, n)
	if err != nil {
		return nil, matrixErrorf(opInverse, err)
	}

	var (
		col, i, k int                  // loop iterators
		sum       float64              // ?
		pivot     float64              // ?
		y         = make([]float64, n) // forward substitution workspace
		x         = make([]float64, n) // backward substitution workspace
	)
	// Fast‐path: detect *Dense for L, U, and inv
	Ld, okL := Lmat.(*Dense)
	Ud, okU := Umat.(*Dense)
	if okL && okU {
		// row‐major stride
		var baseUi, baseLi int // ??
		for col = 0; col < n; col++ {
			// 4.1 Forward substitution: L*y = e_col
			for i = 0; i < n; i++ {
				sum = 0.0
				baseLi = i * n
				for k = 0; k < i; k++ {
					sum += Ld.data[baseLi+k] * y[k]
				}
				if i == col {
					y[i] = 1.0 - sum
				} else {
					y[i] = -sum
				}
			}
			// 4.2 Backward substitution: U*x = y
			for i = n - 1; i >= 0; i-- {
				sum = 0.0
				baseUi = i * n
				for k = i + 1; k < n; k++ {
					sum += Ud.data[baseUi+k] * x[k]
				}
				pivot = Ud.data[baseUi+i]
				if pivot == 0 {
					return nil, matrixErrorf(opInverse, ErrSingular)
				}
				x[i] = (y[i] - sum) / pivot
			}
			// 4.3 Write x into column col of inv
			for i = 0; i < n; i++ {
				invDense.data[i*n+col] = x[i]
			}
		}

		return invDense, nil
	}

	// Fallback: generic interface version
	var v float64 // ?
	for col = 0; col < n; col++ {
		// Forward substitution: L*y = e_col
		for i = 0; i < n; i++ {
			sum = 0.0
			for k = 0; k < i; k++ {
				v, _ = Lmat.At(i, k)
				sum += v * y[k]
			}
			if i == col {
				y[i] = 1.0 - sum
			} else {
				y[i] = -sum
			}
		}
		// Backward substitution: U*x = y
		for i = n - 1; i >= 0; i-- {
			sum = 0.0
			for k = i + 1; k < n; k++ {
				v, _ = Umat.At(i, k)
				sum += v * x[k]
			}
			pivot, _ = Umat.At(i, i)
			if pivot == 0 {
				return nil, matrixErrorf(opInverse, ErrSingular)
			}
			x[i] = (y[i] - sum) / pivot
		}
		// Write x into column col of inv
		for i = 0; i < n; i++ {
			_ = invDense.Set(i, col, x[i])
		}
	}

	return invDense, nil
}

// LU performs Doolittle decomposition A = L*U with unit diagonal on L (no pivoting).
//
// Contract: m non-nil and square.
//
// Determinism & Performance:
//   - Fixed i→{j≥i} for U then {j>i}→i for L.
//   - Fast path for *Dense uses row-major offsets.
//
// Complexity: Time O(n^3), Space O(n^2).
//
// AI-Hints:
//   - For stability-sensitive workflows consider pivoting upstream;
//     here we trade stability for determinism.
func LU(m Matrix) (Matrix, Matrix, error) {
	// Validate input non‐nil and square
	if err := ValidateNotNil(m); err != nil {
		return nil, nil, matrixErrorf(opLU, err)
	}
	if err := ValidateSquare(m); err != nil {
		return nil, nil, matrixErrorf(opLU, err)
	}

	// Allocate L and U
	n := m.Rows()
	Lraw, err := NewDense(n, n)
	if err != nil {
		return nil, nil, matrixErrorf(opLU, err)
	}
	Uraw, err := NewDense(n, n)
	if err != nil {
		return nil, nil, matrixErrorf(opLU, err)
	}

	// Initialize L diagonal to 1 (unit lower triangular)
	for i := 0; i < n; i++ {
		Lraw.data[i*n+i] = 1.0
	}

	// Detect fast‐path on *Dense
	// mRaw holds the input data if m is *Dense
	mRaw, useFast := m.(*Dense)
	var (
		i, j, k int     // loop iterators
		sum     float64 // ?
		pivot   float64 // ?
	)
	// Execute Doolittle decomposition
	if useFast {
		// Fast‐path: operate directly on flat slices
		var baseI, baseJ int
		for i = 0; i < n; i++ {
			// Compute U[i][j] for j >= i
			for j = i; j < n; j++ {
				sum = 0.0
				baseI = i * n
				for k = 0; k < i; k++ {
					sum += Lraw.data[baseI+k] * Uraw.data[k*n+j]
				}
				Uraw.data[baseI+j] = mRaw.data[baseI+j] - sum
			}
			// Compute L[j][i] for j > i
			for j = i + 1; j < n; j++ {
				sum = 0.0
				baseJ = j * n
				for k = 0; k < i; k++ {
					sum += Lraw.data[baseJ+k] * Uraw.data[k*n+i]
				}
				pivot = Uraw.data[i*n+i]
				Lraw.data[baseJ+i] = (mRaw.data[baseJ+i] - sum) / pivot
			}
		}
	} else {
		// Fallback: generic interface version
		var a, l, u float64 // ?
		for i = 0; i < n; i++ {
			// Compute U[i][j] for j >= i
			for j = i; j < n; j++ {
				sum = 0.0
				for k = 0; k < i; k++ {
					l, _ = Lraw.At(i, k)
					u, _ = Uraw.At(k, j)
					sum += l * u
				}
				a, _ = m.At(i, j)
				_ = Uraw.Set(i, j, a-sum)
			}
			// Compute L[j][i] for j > i
			for j = i + 1; j < n; j++ {
				sum = 0.0
				for k = 0; k < i; k++ {
					l, _ = Lraw.At(j, k)
					u, _ = Uraw.At(k, i)
					sum += l * u
				}
				a, _ = m.At(j, i)
				pivot, _ = Uraw.At(i, i)
				_ = Lraw.Set(j, i, (a-sum)/pivot)
			}
		}
	}

	// Return L and U
	return Lraw, Uraw, nil
}

// QR computes Q,R for A = Q*R via Householder reflections.
//
// Contract: m non-nil and square.
//
// Determinism & Performance:
//   - Householder steps are applied in fixed column order.
//   - Fast path for *Dense reduces accessor overhead.
//
// Complexity: Time O(n^3), Space O(n^2).
//
// AI-Hints:
//   - For tall-skinny matrices consider blocked/TSQR variants outside this package
//     if you need better cache behavior or parallelism.
func QR(m Matrix) (Matrix, Matrix, error) {
	// Validate input non‐nil and square
	if err := ValidateNotNil(m); err != nil {
		return nil, nil, matrixErrorf(opQR, err)
	}
	if err := ValidateSquare(m); err != nil {
		return nil, nil, matrixErrorf(opQR, err)
	}
	n := m.Rows()

	// Prepare working copy A and orthogonal accumulator Q
	Araw := m.Clone()
	Qraw, err := NewDense(n, n)
	if err != nil {
		return nil, nil, matrixErrorf(opQR, err)
	}
	// initialize Q to identity: Q[i,i]=1
	for i := 0; i < n; i++ {
		Qraw.data[i*n+i] = 1.0
	}

	// Detect fast‐path on *Dense for A
	Ad, useFast := Araw.(*Dense)

	// Allocate Householder vector
	v := make([]float64, n)

	// Perform Householder reflections
	var (
		i, j, k    int     // loop indices
		norm, beta float64 // vector norm and β = vᵀv
		alpha, tau float64 // reflection scalar and 2/β factor
		sum, aij   float64 // accumulators and temporary values

	)
	for k = 0; k < n; k++ {
		// 4.1: Compute norm of A[k:n][k]
		norm = 0.0
		if useFast {
			for i = k; i < n; i++ {
				aij = Ad.data[i*n+k]
				norm += aij * aij
			}
		} else {
			for i = k; i < n; i++ {
				aij, _ = Araw.At(i, k)
				norm += aij * aij
			}
		}
		norm = math.Sqrt(norm)
		if norm == 0.0 {
			continue // skip zero column
		}

		// 4.2: Compute alpha = -sign(A[k,k]) * norm
		if useFast {
			aij = Ad.data[k*n+k]
		} else {
			aij, _ = Araw.At(k, k)
		}
		alpha = -math.Copysign(norm, aij)

		// 4.3: Build Householder vector v
		for i = 0; i < n; i++ {
			v[i] = 0.0
		}
		if useFast {
			for i = k; i < n; i++ {
				v[i] = Ad.data[i*n+k]
			}
		} else {
			for i = k; i < n; i++ {
				v[i], _ = Araw.At(i, k)
			}
		}
		v[k] -= alpha

		// 4.4: Compute β = vᵀv and τ = 2/β
		beta = 0.0
		for i = k; i < n; i++ {
			beta += v[i] * v[i]
		}
		tau = 2.0 / beta

		// 4.5: Apply reflection to A (update R)
		for j = k; j < n; j++ {
			sum = 0.0
			if useFast {
				for i = k; i < n; i++ {
					sum += v[i] * Ad.data[i*n+j]
				}
				for i = k; i < n; i++ {
					Ad.data[i*n+j] -= tau * v[i] * sum
				}
			} else {
				for i = k; i < n; i++ {
					aij, _ = Araw.At(i, j)
					sum += v[i] * aij
				}
				for i = k; i < n; i++ {
					aij, _ = Araw.At(i, j)
					_ = Araw.Set(i, j, aij-tau*v[i]*sum)
				}
			}
		}

		// 4.6: Apply reflection to Q
		for j = 0; j < n; j++ {
			sum = 0.0
			if useFast {
				for i = k; i < n; i++ {
					sum += v[i] * Qraw.data[i*n+j]
				}
				for i = k; i < n; i++ {
					Qraw.data[i*n+j] -= tau * v[i] * sum
				}
			} else {
				for i = k; i < n; i++ {
					aij, _ = Qraw.At(i, j)
					sum += v[i] * aij
				}
				for i = k; i < n; i++ {
					aij, _ = Qraw.At(i, j)
					_ = Qraw.Set(i, j, aij-tau*v[i]*sum)
				}
			}
		}
	}

	// Finalize R = Araw and return Q, R
	return Qraw, Araw, nil
}
