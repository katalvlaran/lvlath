// SPDX-License-Identifier: MIT
// Package matrix — public API facades.
//
// Purpose:
//   - Provide thin, well-documented entry points for common tasks across the package.
//   - Avoid any logic duplication — each facade delegates to the canonical implementation.
//   - Keep function names explicit and intention-revealing to improve discoverability.
//
// Determinism & Policy:
//   - Facades never change the loop orders or numeric policy of underlying kernels.
//   - APSP expects +Inf for "no edge" and 0 on the diagonal; facades preserve this contract.
//   - Validation is performed in the kernels; facades only compose or forward.
//
// AI-Hints:
//   - Prefer passing *Dense to unlock fast-paths in kernels (flat-slice loops).
//   - Use NewIdentity/NewZeros to build matrices with explicit shape and neutral elements.
//   - For APSP, call APSPInPlace (delegates to FloydWarshall).
//   - For graph export, AdjacencyToGraph mirrors GraphFromAdjacency for discoverability.

package matrix

import "github.com/katalvlaran/lvlath/core"

// ---------- Constructors & Utilities (O(1) alloc + O(rc) zeroing by runtime) ----------

// NewZeros returns a new zero-initialized *Dense of size rows×cols.
// It is a thin alias of NewDense with an intention-revealing name.
// Deterministic: single allocation; no hidden work;
// Complexity: O(n^2) zero-init (constructor) + O(n) diagonal writes.
//
// Note: Returns (*Dense, error) to surface ErrInvalidDimensions.
func NewZeros(rows, cols int) (*Dense, error) {
	// Delegate directly to the strict constructor (single allocation).
	return NewDense(rows, cols)
}

// NewIdentity returns I_n (n×n identity; ones on the diagonal, zeros elsewhere).
// Determinism: fixed i-loop; single write per diagonal cell.
// Complexity: O(n^2) zeroing (constructor) + O(n) writes on the diagonal.
//
// AI-Hints: Use as a neutral element for inverses/preconditioning/orthogonalization.
func NewIdentity(n int) (*Dense, error) {
	// Allocate an n×n zero matrix via the constructor.
	I, err := NewDense(n, n) // O(1) alloc + O(n^2) zeroing
	if err != nil {
		return nil, err // propagate constructor error unchanged
	}
	// Set the diagonal deterministically in a single loop.
	for i := 0; i < n; i++ { // fixed i order guarantees reproducibility
		_ = I.Set(i, i, 1.0) // Set is bounds-safe; error is not expected after shape validation
	}

	// Return the identity matrix.
	return I, nil
}

// CloneMatrix returns a structural clone of m (same type if m is *Dense).
// Thin wrapper over Matrix.Clone for API discoverability.
// Complexity: O(r*c) copy for dense; implementation-defined otherwise.
func CloneMatrix(m Matrix) Matrix {
	// Delegate to polymorphic clone on the concrete implementation.
	return m.Clone()
}

// ZerosLike returns a new zero matrix with the same shape as m.
// Complexity: O(1) alloc + O(rc) zeroing. Handy to preallocate staging buffers.
//
// AI-Hints: Useful for staging buffers or accumulating into fresh containers.
func ZerosLike(m Matrix) (*Dense, error) {
	// Read shape once and call NewDense with the same dimensions.
	return NewDense(m.Rows(), m.Cols()) // errors (if any) bubble up
}

// IdentityLike returns I with dimension = Rows(m); requires square shape.
// Complexity: O(n^2). Validates square via central validator.
//
// AI-Hints: Handy to build projectors or initialize iterative schemes.
func IdentityLike(m Matrix) (*Dense, error) {
	// Ensure the input is square using the centralized validator.
	if err := ValidateSquare(m); err != nil {
		return nil, matrixErrorf("IdentityLike", err) // wrap with call-site tag
	}
	// Construct the identity of matching dimension.
	return NewIdentity(m.Rows()) // returns (*Dense, error)
}

// ---------- Linear Algebra (facades map 1:1 to kernels; O(rc) unless noted) ----------

// Sum is an alias for Add: element-wise a + b.
// Complexity: O(rc).
//
// AI-Hints: Prefer passing *Dense operands for single flat-loop fast-path.
func Sum(a, b Matrix) (Matrix, error) { return Add(a, b) }

// Diff is an alias for Sub: element-wise a − b.
// Complexity: O(rc).
func Diff(a, b Matrix) (Matrix, error) { return Sub(a, b) }

// Product is an alias for Mul: matrix product a × b.
// Complexity: O(r*n*c).
//
// AI-Hints: Prefer Dense to unlock cache-friendly fast path.
func Product(a, b Matrix) (Matrix, error) { return Mul(a, b) }

// HadamardProd is an alias for Hadamard: element-wise product a ⊙ b.
// Complexity: O(rc).
func HadamardProd(a, b Matrix) (Matrix, error) { return Hadamard(a, b) }

// T is an alias for Transpose: returns mᵀ.
// Complexity: O(rc).
//
// AI-Hints: Good for small helpers and chaining.
func T(m Matrix) (Matrix, error) { return Transpose(m) }

// ScaleBy is an alias for Scale: α*m.
// Complexity: O(rc).
func ScaleBy(m Matrix, alpha float64) (Matrix, error) { return Scale(m, alpha) }

// MatVecMul is an alias for MatVec: y = m·x.
// Complexity: O(rc).
//
// AI-Hints: For repeated calls with same shape, reuse x/y slices outside.
func MatVecMul(m Matrix, x []float64) ([]float64, error) { return MatVec(m, x) }

// EigenSym calls the canonical Jacobi eigen-decomposition (symmetric input).
// Complexity: O(maxIter · n^3). Numeric policy unchanged.
// Note: Under the hood it calls Eigen; symmetric validation lives in kernels.
func EigenSym(m Matrix, tol float64, maxIter int) ([]float64, Matrix, error) {
	// Delegate directly to the kernel. The kernel performs ValidateNotNil/Square/Symmetric.
	return Eigen(m, tol, maxIter)
}

// InverseOf is an alias for Inverse: returns A^{-1} (no pivoting; deterministic).
// Complexity: O(n^3).
func InverseOf(m Matrix) (Matrix, error) { return Inverse(m) }

// LUDecompose is an alias for LU: returns (L, U) with unit diagonal on L.
// Complexity: O(n^3).
func LUDecompose(m Matrix) (Matrix, Matrix, error) { return LU(m) }

// QRDecompose is an alias for QR: returns (Q, R) via Householder reflections.
// Complexity: O(n^3).
func QRDecompose(m Matrix) (Matrix, Matrix, error) { return QR(m) }

// ---------- APSP / Metric Closure (graph kernels; O(n^3)) ----------

// APSPInPlace runs Floyd–Warshall in-place on m (all-pairs shortest paths).
// Thin alias to FloydWarshall; provided for graph-oriented API discoverability.
// Contract: m square; +Inf for “no edge”; diagonal 0. Deterministic k→i→j loop order.
// AI-Hints: For *Dense, the fast path uses a single in-slice triple loop.
func APSPInPlace(m Matrix) error { return FloydWarshall(m) }

// MetricClosure mutates am.Mat to all-pairs shortest-path distances (APSP).
// Contract: am != nil; square; +Inf denotes "no path" off-diagonal; diagonal 0.
// Deterministic: delegates to FloydWarshall on the underlying matrix.
func MetricClosure(am *AdjacencyMatrix) error {
	// Guard nil pointer early with the package sentinel via centralized validation.
	if err := ValidateGraph(am); err != nil {
		return matrixErrorf("MetricClosure", err) // wrap with context
	}

	// Delegate to APSP kernel on the underlying matrix.
	return FloydWarshall(am.Mat) // O(n^3), in-place
}

// BuildMetricClosure constructs adjacency from g and then converts it to metric-closure
// (Floyd–Warshall in-place). It also marks the returned adjacency as metricClose=true
// to ensure ToGraph() properly refuses exporting distances as edges.
//
// AI-Hints:
//   - Use for TSP/DTW pipelines where you want pairwise shortest-path distances immediately.
//   - Keep in mind the diagonal=0 and +Inf for unreachable pairs policy.
func BuildMetricClosure(g *core.Graph, opts Options) (*AdjacencyMatrix, error) {
	// Build adjacency deterministically using the adapter builder.
	am, err := NewAdjacencyMatrix(g, opts) // may fail on invalid graph/options
	if err != nil {
		return nil, err // bubble as-is
	}
	// Run APSP in place on the underlying matrix.
	if err = FloydWarshall(am.Mat); err != nil {
		return nil, err // bubble kernel error
	}
	// Mark the policy flag so ToGraph() refuses exporting distances as edges.
	am.opts.metricClose = true // explicit flag (no reflective tricks)

	// Return the mutated adjacency wrapper.
	return am, nil
}

// ---------- Graph <-> Adjacency helpers (thin; no hidden semantics) ----------

// BuildAdjacency constructs a deterministic adjacency matrix from a core.Graph.
// Thin alias to NewAdjacencyMatrix; exposed in API to improve discoverability.
// AI-Hints: Pass Options that match your graph semantics (directed/loops/multi/weighted).
func BuildAdjacency(g *core.Graph, opts Options) (*AdjacencyMatrix, error) {
	return NewAdjacencyMatrix(g, opts)
}

// GraphFromAdjacency exports a core.Graph from AdjacencyMatrix with the given options.
// Thin alias to (*AdjacencyMatrix).ToGraph with identical behavior.
func GraphFromAdjacency(am *AdjacencyMatrix, optFns ...Option) (*core.Graph, error) {
	return am.ToGraph(optFns...)
}

// AdjacencyToGraph is a discoverability alias for GraphFromAdjacency.
// Same semantics; different entry name for API cohesion.
func AdjacencyToGraph(am *AdjacencyMatrix, optFns ...Option) (*core.Graph, error) {
	// Delegate to the canonical adapter export.
	return am.ToGraph(optFns...)
}

// DegreeVector returns per-vertex row sums on an adjacency (loops count as 1).
// Thin alias to (*AdjacencyMatrix).DegreeVector; documented here for API cohesion.
func DegreeVector(am *AdjacencyMatrix) ([]float64, error) {
	return am.DegreeVector()
}

// ---------- Convenience facades (compositions only; no loop duplication) ----------

// Symmetrize returns (m + mᵀ)/2. Deterministic composition: Transpose → Add → Scale.
// Complexity: O(rc).
//
// AI-Hints: Useful in spectral methods (PCA, Laplacians) to repair asymmetry drift.
func Symmetrize(m Matrix) (Matrix, error) {
	// Transpose first; kernel validates non-nil input.
	mt, err := Transpose(m) // O(rc)
	if err != nil {
		return nil, matrixErrorf("Symmetrize", err) // wrap with context
	}
	// Add original and transpose; shapes are guaranteed identical.
	sum, err := Add(m, mt) // O(rc)
	if err != nil {
		return nil, matrixErrorf("Symmetrize", err) // wrap
	}

	// Scale by 0.5 to complete the symmetrization.
	return Scale(sum, 0.5) // O(rc)
}

// RowSums returns vector r where r[i] = sum_j m[i,j].
// Implementation: MatVec(m, ones(cols)). No custom loops.
// Complexity: O(rc).
//
// AI-Hints: Used by Markov/stochastic normalization, degree-like features, DTW bands, etc.
func RowSums(m Matrix) ([]float64, error) {
	// Build an all-ones vector of length equal to the number of columns.
	cols := m.Cols()              // O(1) read of dimension
	ones := make([]float64, cols) // allocate the vector once
	for j := 0; j < cols; j++ {   // deterministic fill
		ones[j] = 1.0 // neutral element for summation
	}

	// Multiply m by the ones vector to get per-row sums.
	return MatVec(m, ones) // O(rc), kernel validates lengths
}

// ColSums returns vector c where c[j] = sum_i m[i,j].
// Implementation: T(m) then MatVec with ones(rows).
// Complexity: O(rc).
//
// AI-Hints: Useful for indegree-like stats, column-normalization, PCA centering.
func ColSums(m Matrix) ([]float64, error) {
	// Transpose m first.
	mt, err := Transpose(m) // O(rc)
	if err != nil {
		return nil, matrixErrorf("ColSums", err) // wrap with context
	}
	// Build an all-ones vector of length equal to the (transposed) number of columns,
	// which equals the original number of rows.
	rows := mt.Cols()             // == m.Rows()
	ones := make([]float64, rows) // allocate the vector once
	for i := 0; i < rows; i++ {   // deterministic fill
		ones[i] = 1.0 // neutral element for summation
	}
	// Multiply to get per-column sums of the original matrix.
	return MatVec(mt, ones) // O(rc)
}

// ---------- Sanitization & numeric compare (thin wrappers → ew*) ----------

// Clip returns a copy of m with elements clamped into [lo, hi] (both finite).
//
//	out[i,j] = min(max(A[i,j], lo), hi).
//
// Supports lo<=hi; both can be ±Inf. Deterministic. O(r*c).
// Time: O(r*c). Space: O(r*c). Deterministic.
//
// Policy: If lo > hi, bounds are swapped (normalized). NaN/Inf bounds are rejected.
// AI-Hints:
//   - helps enforce constraints (e.g., probabilities ∈ [0,1]) before normalization.
//   - protects simulation pipelines (GBM/Monte-Carlo) from outliers.
func Clip(m Matrix, lo, hi float64) (Matrix, error) {
	// Delegate to the private element-wise kernel (centralizes the loop).
	return ewClipRange(m, lo, hi) // errors are already wrapped with "Clip" tag inside
}

// ReplaceInfNaN returns a copy of m where any {±Inf, NaN} are replaced by 'val' (finite).
// Time: O(r*c). Space: O(r*c). Deterministic.
//
// Policy: 'val' must be finite; otherwise ErrNaNInf is returned.
// AI-Hints:
//   - Use ReplaceInfNaN before statistics to avoid NaN propagation.
//   - stabilize downstream stats and ML features.
func ReplaceInfNaN(m Matrix, val float64) (Matrix, error) {
	// Delegate to the private ew* sanitizer (centralizes numeric checks and loops).
	return ewReplaceInfNaN(m, val) // errors are wrapped with "ReplaceInfNaN" tag inside
}

// AllClose checks element-wise |a-b| ≤ atol + rtol*|b| for identical shapes.
// Returns (true,nil) if all elements satisfy the relation; (false,nil) otherwise.
// NaN != anything; +Inf equals +Inf; -Inf equals -Inf. Deterministic.
// Time: O(r*c). Space: O(1). Deterministic.
//
// Policy:
//   - a and b must be non-nil and have identical shapes.
//   - rtol, atol are treated as |rtol|, |atol| (negative values are normalized).
//
// AI-Hints:
//   - AllClose with small atol/rtol is ideal for invariance tests in unit tests.
func AllClose(a, b Matrix, rtol, atol float64) (bool, error) {
	// ??
	return ewAllClose(a, b, rtol, atol)
}

// ---------- Statistics (public surface → internal implementations) ----------

// CenterColumns returns a centered copy: Xc = X − mean(X, by columns) and the column means.
// Returns Xc and the column means (length = Cols(X)).
// Implementation: ColSums + divide by rows to get means; then ewBroadcastSubCols.
// Determinism: fixed loops and pure compositions. O(r*c).
// Time: O(r*c). Space: O(r*c).
//
// AI-Hints: feed means into PCA/Regression; reuse for z-scoring.
func CenterColumns(X Matrix) (Matrix, []float64, error) { return centerColumns(X) }

// CenterRows returns a centered copy: Xc[i,*] = X[i,*] − mean(X[i,*]) for each row.
// Returns Xc and the row means. O(r*c).
// Implementation: RowSums + divide by cols; then ewBroadcastSubRows.
// Time: O(r*c). Space: O(r*c). Deterministic.
//
// AI-Hints: useful for DTW/row shifts; for stochastic models, NormalizeRows* is better.
func CenterRows(X Matrix) (Matrix, []float64, error) { return centerRows(X) }

// NormalizeRowsL1 returns Y where each row i is scaled to L1-norm = 1 (if possible).
// Degenerate rows (norm==0) remain zero. Also returns the norms per row.
// Implementation: compute per-row L1 norms (fast-path for Dense), build scale factors 1/norm (or 0),
// then ewScaleRows to produce Y.
// Determinism: fixed i→j passes. O(r*c).
// Time: O(r*c). Space: O(r*c). Deterministic.
//
// AI-Hints: produce row-stochastic matrices for Markov chains.
func NormalizeRowsL1(X Matrix) (Matrix, []float64, error) { return normalizeRowsL1(X) }

// NormalizeRowsL2 scales each row to have L2-norm == 1 when possible; returns Y and per-row norms.
// Degenerate rows (norm==0) remain zero rows by design.
// Implementation: compute per-row L2 norms via √(Σ v^2); then ewScaleRows with 1/norm (or 0).
// Time: O(r*c). Space: O(r*c). Deterministic.
//
// AI-Hints: common for cosine similarity / spectral features.
func NormalizeRowsL2(X Matrix) (Matrix, []float64, error) { return normalizeRowsL2(X) }

// Covariance computes sample covariance of columns: Cov = (Xcᵀ Xc)/(n-1).
// Returns Cov and column means.
// Determinism: compositions only; all loops fixed. O(r*c + c^2*min(r,c)).
// Time: O(r*c + c^2) (via one Transpose + one Mul + one Scale). Space: O(r*c + c^2).
//
// Notes:
//   - Requires r >= 2 to avoid division by zero; else ErrDimensionMismatch.
//   - Uses CenterColumns then reuses canonical kernels (Transpose/Mul/Scale).
func Covariance(X Matrix) (Matrix, []float64, error) { return covariance(X) }

// Correlation computes Pearson correlation of columns via z-scoring:
//
//	Z = (X - mean) / std,  std^2 = Σ (Xc)^2 / (n-1),  degenerate std==0 ⇒ column zeroed.
//	Corr = (Zᵀ Z)/(n-1).
//
// Returns Corr, means, stds.
// Time: O(r*c + c^2). Space: O(r*c + c^2).
func Correlation(X Matrix) (Matrix, []float64, []float64, error) { return correlation(X) }
