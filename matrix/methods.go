// Package matrix provides universal operations on any Matrix implementation,
// including element-wise addition, subtraction, matrix multiplication,
// transpose, and scalar scaling. All functions perform strict
// fail-fast validation and return clear errors on dimension mismatches.
package matrix

import (
	"errors"
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
	opAdd           = "Add"
	opSub           = "Sub"
	opMul           = "Mul"
	opTranspose     = "Transpose"
	opScale         = "Scale"
	opEigen         = "Eigen"
	opFloydWarshall = "FloydWarshall"
	opInverse       = "Inverse"
	opLU            = "LU"
	opQR            = "QR"
)

// ErrSingular is returned when a zero pivot is encountered during inversion.
var ErrSingular = errors.New("ops: matrix is singular")

// ErrNotSymmetric is returned when the input matrix is not symmetric.
var ErrNotSymmetric = errors.New("ops: matrix is not symmetric")

// ErrEigenFailed is returned if the algorithm does not converge within max iterations.
var ErrEigenFailed = errors.New("ops: eigen decomposition did not converge")

// ErrNilMatrix indicates that a nil Matrix was passed to an operation.
var ErrNilMatrix = errors.New("matrix: nil Matrix")

// matrixErrorf wraps an underlying error with the given tag.
func matrixErrorf(tag string, err error) error {
	return fmt.Errorf("%s: %w", tag, err)
}

// Add returns a new Matrix containing the element-wise sum of a and b.
// Stage 1 (Validate): nil-checks and shape match.
// Stage 2 (Prepare): allocate result Dense.
// Stage 3 (Execute): fast-path for *Dense or fallback to interface.
// Stage 4 (Finalize): return result.
// Time Complexity: O(r·c); Space Complexity: O(r·c).
// Complexity: O(r·c) time and memory.
func Add(a, b Matrix) (Matrix, error) {
	// Stage 1: Validate inputs non-nil
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

	// Stage 2: Allocate result Dense
	rows, cols := a.Rows(), a.Cols()
	res, err := NewDense(rows, cols)
	if err != nil {
		return nil, matrixErrorf(opAdd, err)
	}

	// Stage 3: Fast-path for two Dense matrices
	if da, okA := a.(*Dense); okA {
		if db, okB := b.(*Dense); okB {
			// direct element-wise addition on backing slices
			length := rows * cols
			for idx := 0; idx < length; idx++ {
				res.data[idx] = da.data[idx] + db.data[idx]
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
			_ = res.Set(i, j, av+bv) // safe: within bounds
		}
	}

	// Stage 4: Return result
	return res, nil
}

// Sub returns a new Matrix containing the element-wise difference a - b.
// Stage 1 (Validate): nil-checks and shape match.
// Stage 2 (Prepare): allocate result Dense.
// Stage 3 (Execute): loop over elements.
// Stage 4 (Finalize): return result.
// Complexity: O(r·c) time and memory.
func Sub(a, b Matrix) (Matrix, error) {
	// Stage 1: Validate inputs non-nil
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

	// Stage 2: Allocate result Dense
	rows, cols := a.Rows(), a.Cols()
	res, err := NewDense(rows, cols)
	if err != nil {
		return nil, matrixErrorf(opSub, err)
	}

	// Stage 3: Fast-path for two Dense matrices
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
	if err := ValidateNotNil(a); err != nil {
		return nil, matrixErrorf(opMul, err)
	}
	if err := ValidateNotNil(b); err != nil {
		return nil, matrixErrorf(opMul, err)
	}
	if a.Cols() != b.Rows() {
		return nil, matrixErrorf(opMul, ErrMatrixDimensionMismatch)
	}

	// Stage 2: Allocate result Dense
	aRows, aCols, bCols := a.Rows(), a.Cols(), b.Cols()
	res, err := NewDense(aRows, bCols)
	if err != nil {
		return nil, matrixErrorf(opMul, err)
	}
	var (
		i, j, k         int // loop iterators
		av, bv, current float64
	)
	// Stage 3: Fast-path for two Dense matrices
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

	// Stage 4: Return result
	return res, nil
}

// Transpose returns a new Matrix where rows and columns of m are swapped.
// Stage 1 (Validate): nil-check.
// Stage 2 (Prepare): allocate Dense(cols×rows).
// Stage 3 (Execute): fast-path for *Dense or fallback to interface.
// Stage 4 (Finalize): return result.
// Time Complexity: O(r·c); Space Complexity: O(r·c).
func Transpose(m Matrix) (Matrix, error) {
	// Stage 1: Validate input non-nil
	if err := ValidateNotNil(m); err != nil {
		return nil, matrixErrorf(opTranspose, err)
	}

	// Stage 2: Allocate result Dense with flipped dimensions
	rows, cols := m.Rows(), m.Cols()
	res, err := NewDense(cols, rows) // dims flipped
	if err != nil {
		return nil, matrixErrorf(opTranspose, err)
	}

	// Stage 3: Fast-path for Dense → Dense
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

	// Stage 4: Return result
	return res, nil
}

// Scale returns a new Matrix where each element of m is multiplied by alpha.
// Stage 1 (Validate): nil-check.
// Stage 2 (Prepare): allocate Dense(rows×cols).
// Stage 3 (Execute): double loop scaling.
// Stage 4 (Finalize): return.
// Complexity: O(r·c).
func Scale(m Matrix, alpha float64) (Matrix, error) {
	// Stage 1: Validate input non-nil
	if err := ValidateNotNil(m); err != nil {
		return nil, matrixErrorf(opScale, err)
	}

	// Stage 2: Allocate result Dense
	rows, cols := m.Rows(), m.Cols()
	res, err := NewDense(rows, cols)
	if err != nil {
		return nil, matrixErrorf(opScale, err)
	}

	// Stage 3: Fast-path for Dense → Dense
	var i, j int // loop iterators
	if dm, ok := m.(*Dense); ok {
		n := rows * cols
		for idx := 0; idx < n; idx++ {
			res.data[idx] = dm.data[idx] * alpha
		}
		return res, nil
	}

	// Fallback: generic interface loop
	var v float64
	for i = 0; i < rows; i++ {
		for j = 0; j < cols; j++ {
			v, _ = m.At(i, j)          // safe: bounds ensured
			_ = res.Set(i, j, v*alpha) // safe: within bounds
		}
	}

	// Stage 4: Return result
	return res, nil
}

// Eigen performs Jacobi eigenvalue decomposition on a symmetric matrix m.
// It returns a slice of eigenvalues and a matrix of eigenvectors Q (columns of Q).
// tol specifies convergence threshold for off-diagonal elements.
// maxIter caps the number of Jacobi sweeps.
// Returns ErrMatrixDimensionMismatch, ErrNotSymmetric, or ErrEigenFailed.
// Time Complexity: O(maxIter·n³); Space Complexity: O(n²).
func Eigen(m Matrix, tol float64, maxIter int) ([]float64, Matrix, error) {
	// Stage 1: Validate input non-nil and square
	if err := ValidateNotNil(m); err != nil {
		return nil, nil, matrixErrorf(opEigen, err)
	}
	var (
		n, cols  = m.Rows(), m.Cols() // n — number of rows (and columns), cols — number of columns
		i, j     int                  // loop iterators over rows and columns
		aij, aji float64              // off-diagonal entries for symmetry check
	)
	if n != cols {
		// if not square — error out immediately
		return nil, nil, matrixErrorf(opEigen, ErrMatrixDimensionMismatch)
	}

	// Stage 2: Check symmetry within tolerance
	for i = 0; i < n; i++ {
		for j = i + 1; j < n; j++ {
			// read A[i,j] and A[j,i]
			aij, _ = m.At(i, j)
			aji, _ = m.At(j, i)
			// if they differ by more than tol — not symmetric
			if math.Abs(aij-aji) > tol {
				return nil, nil, matrixErrorf(opEigen, ErrNotSymmetric)
			}
		}
	}

	// Stage 3: Prepare working copy A and orthogonal accumulator Q
	aRaw := m.Clone()           // aRaw is a working copy of m to avoid modifying the original
	qRaw, err := NewDense(n, n) // qRaw is a newly allocated zero dense matrix
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

	// Stage 4: Jacobi rotations
	var (
		iter               int     // iteration counter
		base               int     // helper offset into the flat data slice
		p, q               int     // current pivot indices
		maxOff, off        float64 // maxOff — current max |A[p,q]|; off — temporary
		app, aqq           float64 // diagonal entries A[p,p], A[q,q]
		aip, aiq, qip, qiq float64 // temporaries for A[i,p], A[i,q] and Q[i,p], Q[i,q]
		new_ip, new_iq     float64 // updated values for A[i,p] and A[i,q]
		apq                float64 // off-diagonal entry A[p,q]
		theta, t           float64 // intermediate rotation parameters
		c, s               float64 // cosine and sine of the rotation angle
	)
	for iter = 0; iter < maxIter; iter++ {
		// 4.1: Find pivot (p,q) maximizing |A[p,q]|
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

		// 4.2: Check convergence: if maxOff < tol, break
		if maxOff < tol {
			break
		}

		// 4.3: Compute rotation parameters from A[p,p], A[q,q], A[p,q]
		if useFast {
			app = Adense.data[p*n+p]
			aqq = Adense.data[q*n+q]
			apq = Adense.data[p*n+q]
		} else {
			app, _ = aRaw.At(p, p)
			aqq, _ = aRaw.At(q, q)
			apq, _ = aRaw.At(p, q)
		}
		// θ = (aqq−app)/(2·apq)
		theta = (aqq - app) / (2 * apq)
		// t = sign(θ) / (|θ|+√(θ²+1))
		t = math.Copysign(1.0/(math.Abs(theta)+math.Sqrt(theta*theta+1)), theta)
		// c = 1/√(1+t²), s = t·c
		c = 1.0 / math.Sqrt(t*t+1)
		s = t * c

		// 4.4: Apply rotation to A
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

		// 4.5: Accumulate rotation into Q
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

	// Stage 5: Check convergence
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
		return nil, nil, matrixErrorf(opEigen, ErrEigenFailed)
	}

	// Stage 6: Extract eigenvalues from diagonal of A
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

	// Stage 7: Return eigenvalues and eigenvectors
	return eigs, qRaw, nil
}

// FloydWarshall computes the shortest‐path distances between all pairs of vertices
// in‐place on the provided matrix m. m must be square, with +Inf representing
// absent edges. Returns ErrMatrixDimensionMismatch if m is not square, or any
// error from At/Set. Detects *Dense for a fast in‐slice inner loop.
// Time Complexity: O(n³); Space Complexity: O(1) extra.
func FloydWarshall(m Matrix) error {
	// Stage 1: Validate non‐nil and square
	if err := ValidateNotNil(m); err != nil {
		return matrixErrorf(opFloydWarshall, err)
	}
	if err := ValidateSquare(m); err != nil {
		return matrixErrorf(opFloydWarshall, err)
	}

	n := m.Rows()
	var (
		i, j, k      int     // loop iterators
		baseK, baseI int     // ??
		ik, ij       float64 // ??
		candidate    float64 // ?
	)
	// Stage 2: Attempt fast‐path on *Dense
	if d, ok := m.(*Dense); ok {
		// operate directly on d.data (row-major length n*n)
		data := d.data
		for k = 0; k < n; k++ {
			baseK = k * n
			for i = 0; i < n; i++ {
				ik = data[i*n+k]
				if ik == math.Inf(1) {
					continue // no path i→k
				}

				baseI = i * n
				for j = 0; j < n; j++ {
					// current i→j
					ij = data[baseI+j]
					// candidate through k
					candidate = ik + data[baseK+j]
					if candidate < ij {
						data[baseI+j] = candidate
					}
				}
			}
		}

		return nil
	}

	// Stage 3: Generic interface fallback
	var (
		dik, dkj, dij float64
		err           error
	)
	for k = 0; k < n; k++ {
		for i = 0; i < n; i++ {
			// load d(i,k)
			dik, err = m.At(i, k)
			if err != nil {
				return matrixErrorf(opFloydWarshall, err)
			}
			if dik == math.Inf(1) {
				continue
			}

			for j = 0; j < n; j++ {
				// load d(k,j)
				dkj, err = m.At(k, j)
				if err != nil {
					return matrixErrorf(opFloydWarshall, err)
				}
				if dkj == math.Inf(1) {
					continue
				}
				// load d(i,j)
				dij, err = m.At(i, j)
				if err != nil {
					return matrixErrorf(opFloydWarshall, err)
				}
				// relax
				if dik+dkj < dij {
					if err = m.Set(i, j, dik+dkj); err != nil {
						return matrixErrorf(opFloydWarshall, err)
					}
				}
			}
		}
	}

	return nil
}

// Inverse returns the inverse of the square matrix m, or an error if m is not square or singular.
// Blueprint:
//
//	Stage 1 (Validate): ensure m is non‐nil and square.
//	Stage 2 (Decompose): A = L·U via Doolittle LU.
//	Stage 3 (Prepare): allocate result Dense and workspaces.
//	Stage 4 (Execute): for each basis vector eᵢ, solve L·y = eᵢ then U·x = y.
//	Stage 5 (Write): store x as column i of the inverse.
//	Stage 6 (Return): return the computed inverse.
//
// Time Complexity: O(n³); Space Complexity: O(n²).
func Inverse(m Matrix) (Matrix, error) {
	// Stage 1: Validate input non‐nil and square
	if err := ValidateNotNil(m); err != nil {
		return nil, matrixErrorf(opInverse, err)
	}
	if err := ValidateSquare(m); err != nil {
		return nil, matrixErrorf(opInverse, err)
	}

	n := m.Rows()

	// Stage 2: LU decomposition (Doolittle)
	Lmat, Umat, err := LU(m)
	if err != nil {
		return nil, matrixErrorf(opInverse, err)
	}

	// Stage 3: Prepare result container and scratch arrays
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
			// 4.1 Forward substitution: L·y = e_col
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
			// 4.2 Backward substitution: U·x = y
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
		// Forward substitution: L·y = e_col
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
		// Backward substitution: U·x = y
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

// LU performs Doolittle LU decomposition on a square matrix m.
// It returns L (unit lower triangular) and U (upper triangular) matrices.
// Returns ErrMatrixDimensionMismatch if m is not square, or any error from allocation.
// Time Complexity: O(n³); Space Complexity: O(n²).
func LU(m Matrix) (Matrix, Matrix, error) {
	// Stage 1: Validate input non‐nil and square
	if err := ValidateNotNil(m); err != nil {
		return nil, nil, matrixErrorf(opLU, err)
	}
	if err := ValidateSquare(m); err != nil {
		return nil, nil, matrixErrorf(opLU, err)
	}

	n := m.Rows()

	// Stage 2: Allocate L and U
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
	// Stage 3: Execute Doolittle decomposition
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

	// Stage 4: Return L and U
	return Lraw, Uraw, nil
}

// QR returns Q and R for the decomposition m = Q×R using Householder reflections.
// It returns ErrMatrixDimensionMismatch if m is not square.
// Time Complexity: O(n³); Space Complexity: O(n²).
func QR(m Matrix) (Matrix, Matrix, error) {
	// Stage 1: Validate input non‐nil and square
	if err := ValidateNotNil(m); err != nil {
		return nil, nil, matrixErrorf(opQR, err)
	}
	if err := ValidateSquare(m); err != nil {
		return nil, nil, matrixErrorf(opQR, err)
	}
	n := m.Rows()

	// Stage 2: Prepare working copy A and orthogonal accumulator Q
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

	// Stage 3: Allocate Householder vector
	v := make([]float64, n)

	// Stage 4: Perform Householder reflections
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

	// Stage 5: Finalize R = Araw and return Q, R
	return Qraw, Araw, nil
}
