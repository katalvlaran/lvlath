// SPDX-License-Identifier: MIT
// Package: matrix
//
// Purpose:
//   - Canonical dense APSP (Floyd–Warshall) implementation with deterministic loop order.
//   - Shared by adjacency/metric-closure paths; in-place, O(n³) time, O(1) extra space.
//
// Contract:
//   - Square matrix; +Inf means “no path”; diagonal must be 0 before calling.
//
// Performance note:
//   - This is a dense, in-place O(n³) routine chosen for simplicity and determinism
//     on moderate n (e.g., TSP metric-closure). For very large, sparse graphs consider
//     alternatives (Johnson’s algorithm, repeated Dijkstra on sparse structures, or
//     min-plus semiring over sparse matrices).

package matrix

import (
	"fmt"
	"math"
)

// Operation name constants for unified error wrapping and reducing magic strings.
const opFloydWarshall = "FloydWarshall"

// InitDistancesInPlace CONVERTS adjacency (0 / w) to a distance matrix in-place.
// Implementation:
//   - Stage 1: validate square shape (rows==cols), otherwise ErrDimensionMismatch.
//   - Stage 2: row-major rewrite: diag=0; off-diagonal 0 → +Inf; non-zero stays unchanged.
//
// Behavior highlights:
//   - No extra allocations; deterministic nested loops over rows then cols.
//
// Inputs:
//   - mat: *Dense adjacency matrix (square).
//
// Returns:
//   - error: ErrDimensionMismatch for non-square; Set errors wrapped with coordinates.
//
// Errors:
//   - ErrDimensionMismatch (non-square), plus any Set error from Dense.
//
// Determinism:
//   - Fixed i/j loop order; stable rewrites.
//
// Complexity:
//   - Time O(n^2), Space O(1).
//
// Notes:
//   - Used by adjacency metric-closure before Floyd–Warshall.
//
// AI-Hints:
//   - Use when starting from 0/weight adjacency; then run FloydWarshall.
func InitDistancesInPlace(mat *Dense) error {
	// Guard nil pointer to preserve "no panics" discipline even for internal helpers.
	if mat == nil {
		return fmt.Errorf("initDistancesInPlace: nil matrix: %w", ErrNilMatrix)
	}
	r, c := mat.Rows(), mat.Cols()
	if r != c {
		return fmt.Errorf("InitDistancesInPlace: non-square %dx%d: %w", r, c, ErrDimensionMismatch)
	}

	// Rewrite values row-by-row in a fixed order for determinism.
	var i, j int
	var v float64
	var err error
	for i = 0; i < r; i++ {
		for j = 0; j < c; j++ {
			if i == j {
				// Diagonal initialization:
				//   - Base rule: dist(i,i) starts at 0.
				//   - If an explicit self-loop with negative weight exists, it must be preserved:
				//       dist(i,i) = min(0, w_loop) = w_loop when w_loop < 0.
				v, err = mat.At(i, j) // read current diagonal value (may contain loop weight)
				if err != nil {
					return fmt.Errorf("initDistancesInPlace: At(%d,%d): %w", i, j, err)
				}
				// Reject non-finite values early; distances use +Inf only off-diagonal.
				if isNonFinite(v) {
					return fmt.Errorf("initDistancesInPlace: invalid diagonal [%d,%d]=%v: %w", i, j, v, ErrInvalidWeight)
				}
				// Keep negative self-loop weight; otherwise enforce 0.
				if v < 0.0 {
					if err = mat.Set(i, j, v); err != nil {
						return fmt.Errorf("initDistancesInPlace: Set(%d,%d,%v): %w", i, j, v, err)
					}
				} else {
					if err = mat.Set(i, j, 0.0); err != nil {
						return fmt.Errorf("initDistancesInPlace: Set(%d,%d,0): %w", i, j, err)
					}
				}

				continue
			}

			// Off-diagonal rewrite:
			//   - adjacency 0 means "no edge" -> convert to +Inf ("no path" sentinel)
			//   - non-zero finite values are direct edges and must remain as-is
			//   - +Inf is allowed and left unchanged (idempotent initialization)
			v, err = mat.At(i, j) // read current adjacency value
			if err != nil {
				return fmt.Errorf("initDistancesInPlace: At(%d,%d): %w", i, j, err)
			}
			// Reject NaN and -Inf; +Inf is only meaningful as "no path" sentinel.
			if isNaNOrNegInf(v) {
				return fmt.Errorf("initDistancesInPlace: invalid entry [%d,%d]=%v: %w", i, j, v, ErrInvalidWeight)
			}
			// Convert absent edge marker into distance sentinel.
			if v == 0.0 {
				if err = mat.Set(i, j, math.Inf(1)); err != nil {
					return fmt.Errorf("initDistancesInPlace: Set(%d,%d,+Inf): %w", i, j, err)
				}
			}
		}
	}

	return nil
}

// floydWarshallInPlace RUNS dense APSP closure on a square *Dense in place.
// Implementation:
//   - Stage 1: read order n once; alias row-major buffer for tight loops.
//   - Stage 2: triple loop k→i→j with early-continue if i→k or k→j is +Inf.
//
// Behavior highlights:
//   - Strict improvement only (cand < current), providing deterministic tie behavior.
//
// Inputs:
//   - d: *Dense square distance matrix (+Inf marks unreachable, diagonal 0).
//
// Determinism:
//   - Fixed loop order (k, then i, then j).
//
// Complexity:
//   - Time O(n^3), Space O(1). No allocations in hot loops.
//
// Notes:
//   - Negative cycles propagate to negative diagonals for nodes on reachable cycles.
//
// AI-Hints:
//   - Prefer calling through FloydWarshall so the fast path is selected automatically.
func floydWarshallInPlace(d *Dense) {
	// Read matrix order once; upstream guarantees square shape.
	n := d.r // direct field access avoids a virtual call

	// Predeclare all loop counters and temporaries to avoid per-iteration allocations.
	var (
		k, i, j      int     // loop indices
		baseK, baseI int     // row base offsets for K and I in the flat buffer
		ik, ij, kj   float64 // distances d[i,k], d[i,j], d[k,j]
		cand         float64 // candidate path length via k: d[i,k] + d[k,j]
	)

	// Local alias to the flat row-major buffer; this does not change bounds checks,
	// it just shortens the access path and helps the compiler with CSE.
	data := d.data

	// Triple nested loops with a deterministic order matching tests and other ops.
	for k = 0; k < n; k++ { // outer: pick intermediate vertex k
		baseK = k * n // compute once per k

		for i = 0; i < n; i++ { // middle: source vertex i
			ik = data[i*n+k]       // current shortest distance i→k
			if math.IsInf(ik, 1) { // if i cannot reach k,
				continue // no path via k can improve i→j
			}
			baseI = i * n // compute once per i

			for j = 0; j < n; j++ { // inner: destination vertex j
				kj = data[baseK+j]     // current shortest distance k→j
				if math.IsInf(kj, 1) { // if k cannot reach j,
					continue // skip candidate computation
				}
				ij = data[baseI+j] // current shortest distance i→j
				cand = ik + kj     // candidate path length via k
				if cand < ij {     // strict improvement only (deterministic tie rule)
					data[baseI+j] = cand // relax edge i→j in place
				}
			}
		}
	}
}

// FloydWarshall COMPUTES all-pairs shortest paths in-place over a Matrix.
// Implementation:
//   - Stage 1: validate non-nil and square shape (ValidateSquare).
//   - Stage 2: if *Dense, use fast path; otherwise run generic interface triple loop.
//
// Behavior highlights:
//   - +Inf denotes “no path”; diagonal must be 0 on entry.
//
// Inputs:
//   - m: Matrix (square). Use +Inf for no-edge and 0 on the diagonal.
//
// Returns:
//   - error: ErrNilMatrix, ErrDimensionMismatch, or Matrix.At/Set errors (interface path).
//
// Errors:
//   - ErrNilMatrix (nil m), ErrDimensionMismatch (non-square), wrapped Matrix access errors.
//
// Determinism:
//   - Fixed (k,i,j) loop order in both fast and generic paths.
//
// Complexity:
//   - Time O(n^3); Extra space O(1).
//
// Notes:
//   - Running FW again on its output is idempotent (no further relaxations).
//
// AI-Hints:
//   - Prefer passing *Dense to trigger the zero-overhead fast path.
//   - Ensure diagonal zeros before calling; replace missing edges with +Inf.
//   - For very sparse graphs consider a sparse APSP instead; this routine is dense.
func FloydWarshall(m Matrix) error {
	// Validate: non-nil; square (shape n×n).
	if err := ValidateSquareNonNil(m); err != nil {
		return matrixErrorf(opFloydWarshall, err)
	}

	// Fast-path: direct dense traversal via a single source of truth.
	if d, ok := m.(*Dense); ok {
		floydWarshallInPlace(d) // single source of truth for Dense
		return nil
	}

	// Generic interface fallback (no extra allocations).
	n := m.Rows() // equals m.Cols() after ValidateSquare

	// Predeclare indices and temporaries; reuse across loops to reduce GC pressure.
	var (
		k, i, j       int     // loop indices
		dik, dkj, dij float64 // distances m[i,k], m[k,j], m[i,j]
		cand          float64 // candidate via k: m[i,k] + m[k,j]
		err           error   // error accumulator
	)

	for k = 0; k < n; k++ { // fixed outer loop (intermediate vertex)
		for i = 0; i < n; i++ { // fixed middle loop (source)
			dik, err = m.At(i, k) // read m(i,k)
			if err != nil {
				return matrixErrorf(opFloydWarshall, err)
			}
			if math.IsInf(dik, 1) { // no path i→k
				continue
			}
			for j = 0; j < n; j++ { // fixed inner loop (destination)
				dkj, err = m.At(k, j) // read m(k,j)
				if err != nil {
					return matrixErrorf(opFloydWarshall, err)
				}
				if math.IsInf(dkj, 1) { // no path k→j
					continue
				}
				dij, err = m.At(i, j) // read current m(i,j)
				if err != nil {
					return matrixErrorf(opFloydWarshall, err)
				}
				cand = dik + dkj // compute candidate
				if cand < dij {  // relax if strictly better
					if err = m.Set(i, j, cand); err != nil {
						return matrixErrorf(opFloydWarshall, err)
					}
				}
			}
		}
	}

	return nil
}
