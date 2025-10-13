// SPDX-License-Identifier: MIT
// Package: matrix
//
// Purpose:
//   - Canonical dense APSP (Floyd–Warshall) implementation with deterministic loop order.
//   - Shared by adjacency/metric-closure paths; in-place, O(n³) time, O(1) extra space.
//
// Contract:
//   - Square matrix; +Inf means “no path”; diagonal must be 0 before calling.

package matrix

import (
	"fmt"
	"math"
)

// Operation name constants for unified error wrapping and reducing magic strings.
const opFloydWarshall = "FloydWarshall"

// initDistancesInPlace converts adjacency (0 / w) -> distance matrix in-place:
//
//	diag = 0; off-diagonal 0 -> +Inf; non-zero -> unchanged.
//
// Requires square matrix. Returns ErrDimensionMismatch otherwise.
// Complexity: O(n^2).
func initDistancesInPlace(mat *Dense) error {
	r, c := mat.Rows(), mat.Cols()
	if r != c {
		return fmt.Errorf("initDistancesInPlace: non-square %dx%d: %w", r, c, ErrDimensionMismatch)
	}

	// Rewrite values row-by-row in a fixed order for determinism.
	var i, j int
	var v float64
	for i = 0; i < r; i++ {
		for j = 0; j < c; j++ {
			if i == j {
				// Distance from a node to itself is zero.
				if err := mat.Set(i, j, 0.0); err != nil {
					return fmt.Errorf("initDistancesInPlace: Set(%d,%d,0): %w", i, j, err)
				}
				continue
			}
			// Read current adjacency value.
			v, _ = mat.At(i, j) // safe after shape validation
			if v == 0.0 {
				// No direct edge: set +Inf to represent "no path" initially.
				if err := mat.Set(i, j, math.Inf(1)); err != nil {
					return fmt.Errorf("initDistancesInPlace: Set(%d,%d,+Inf): %w", i, j, err)
				}
			}
		}
	}

	return nil
}

// floydWarshallInPlace runs APSP closure on a square *Dense in-place.
//
// Policy (assumed by callers):
//   - +Inf (math.Inf(1)) denotes "no path" off-diagonal.
//   - The diagonal MUST be 0 before calling (distance to self).
//
// Loop order is fixed (k → i → j) for deterministic accumulation.
// Time: O(n^3); Extra space: O(1). No allocations inside the hot loops.
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

// FloydWarshall computes all-pairs shortest paths in-place on m.
//
// Contract:
//   - m must be square (n×n).
//   - +Inf denotes “no edge” off-diagonal; the diagonal MUST be 0.
//
// Determinism:
//   - Loop order is fixed (k → i → j), ensuring stable accumulation order.
//
// Complexity: Time O(n^3), Extra space O(1) (fully in-place).
//
// AI-Hints:
//   - Prefer passing *Dense to trigger the zero-overhead fast path.
//   - Ensure diagonal zeros before calling; replace missing edges with +Inf.
//   - For very sparse graphs consider a sparse APSP instead; this routine is dense.
func FloydWarshall(m Matrix) error {
	// Validate: non-nil; square (shape n×n).
	if err := ValidateSquare(m); err != nil {
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
