// Package tsp — cost utilities shared by exact/heuristic solvers.
//
// This file provides small, allocation-conscious helpers to compute the
// total cost of a Hamiltonian cycle represented by a vertex index tour.
// They are intentionally minimal and side-effect free.
//
// Design:
//   - Fast path for *matrix.Dense and generic path for any matrix.Matrix.
//   - Strict sentinels from types.go on any invalid input.
//   - Defensive checks (Inf/NaN/negative) even if validate.go was called earlier.
//   - Stable summation: rounded to 1e-9 to avoid cross-platform FP noise.
//
// Complexity:
//   - O(n) time for a tour of length n+1, O(1) extra space.
package tsp

import (
	"math"

	"github.com/katalvlaran/lvlath/matrix"
)

// roundScale controls final cost stabilization precision (1e-9).
// Avoids tiny FP drifts across platforms/opt levels without affecting optimality.
const roundScale = 1e9

// TourCost dispatches to a specialized implementation when dist is *matrix.Dense,
// otherwise uses the generic matrix.Matrix path.
//
// Contract:
//   - tour must represent a closed cycle: len(tour) >= 2 and indices within [0..n-1].
//   - dist must be square (n×n). See validateDistMatrix for full validation.
//   - Returns ErrNonSquare, ErrDimensionMismatch, ErrIncompleteGraph, or ErrNegativeWeight.
//
// Note: This is an internal helper; solvers are expected to validate inputs
//
//	upfront via validateAll, but we still guard against misuse.
//
// Complexity: O(n).
func TourCost(dist matrix.Matrix, tour []int) (float64, error) {
	if dist == nil || tour == nil || len(tour) < 2 {
		return 0, ErrDimensionMismatch
	}
	if d, ok := dist.(*matrix.Dense); ok {
		return tourCostDense(d, tour)
	}
	return tourCostGeneric(dist, tour)
}

// tourCostDense sums costs along the cycle edges tour[i]→tour[i+1] using *matrix.Dense.
//
// Checks performed per edge:
//   - indices in range,
//   - weight finite (no NaN), not ±Inf (⇒ ErrIncompleteGraph),
//   - non-negative (⇒ ErrNegativeWeight).
//
// Complexity: O(n).
func tourCostDense(d *matrix.Dense, tour []int) (float64, error) {
	// Shape guard.
	var (
		nr = d.Rows()
		nc = d.Cols()
	)
	if nr != nc || nr <= 0 {
		return 0, ErrNonSquare
	}

	// Main accumulation.
	var (
		sum float64
		i   int
		u   int
		v   int
		w   float64
		err error
		n   = nr
		L   = len(tour) - 1 // last index used as closing
	)

	for i = 0; i < L; i++ {
		u = tour[i]
		v = tour[i+1]

		// Index range checks.
		if u < 0 || u >= n || v < 0 || v >= n {
			return 0, ErrDimensionMismatch
		}

		// Fetch weight and validate.
		w, err = d.At(u, v)
		if err != nil {
			// Dense.At should only fail on OOB; map to shape sentinel.
			return 0, ErrDimensionMismatch
		}
		if math.IsNaN(w) {
			return 0, ErrDimensionMismatch
		}
		if math.IsInf(w, 0) {
			return 0, ErrIncompleteGraph
		}
		if w < 0 {
			return 0, ErrNegativeWeight
		}

		sum += w
	}

	return round1e9(sum), nil
}

// tourCostGeneric sums costs using the matrix.Matrix interface.
//
// Same checks as tourCostDense; slightly higher call overhead.
// Kept lean to avoid hidden allocations.
//
// Complexity: O(n).
func tourCostGeneric(m matrix.Matrix, tour []int) (float64, error) {
	// Shape guard.
	var (
		nr = m.Rows()
		nc = m.Cols()
	)
	if nr != nc || nr <= 0 {
		return 0, ErrNonSquare
	}

	var (
		sum float64
		i   int
		u   int
		v   int
		w   float64
		err error
		n   = nr
		L   = len(tour) - 1
	)

	for i = 0; i < L; i++ {
		u = tour[i]
		v = tour[i+1]

		if u < 0 || u >= n || v < 0 || v >= n {
			return 0, ErrDimensionMismatch
		}

		w, err = m.At(u, v)
		if err != nil {
			return 0, ErrDimensionMismatch
		}
		if math.IsNaN(w) {
			return 0, ErrDimensionMismatch
		}
		if math.IsInf(w, 0) {
			return 0, ErrIncompleteGraph
		}
		if w < 0 {
			return 0, ErrNegativeWeight
		}

		sum += w
	}

	return round1e9(sum), nil
}

// edgeCost fetches the weight for a single directed edge u→v with strict validation.
// Useful for local-search deltas (2-opt/3-opt) to keep sentinel semantics centralized.
//
// Complexity: O(1).
func edgeCost(m matrix.Matrix, u, v int) (float64, error) {
	// Basic shape and range checks.
	var (
		nr = m.Rows()
		nc = m.Cols()
	)
	if nr != nc || nr <= 0 {
		return 0, ErrNonSquare
	}
	if u < 0 || u >= nr || v < 0 || v >= nr {
		return 0, ErrDimensionMismatch
	}

	w, err := m.At(u, v)
	if err != nil {
		return 0, ErrDimensionMismatch
	}
	if math.IsNaN(w) {
		return 0, ErrDimensionMismatch
	}
	if math.IsInf(w, 0) {
		return 0, ErrIncompleteGraph
	}
	if w < 0 {
		return 0, ErrNegativeWeight
	}
	return w, nil
}

// round1e9 returns x rounded to 1e-9 absolute precision.
// This keeps costs stable across platforms without affecting algorithmic correctness.
//
// Complexity: O(1).
func round1e9(x float64) float64 {
	return math.Round(x*roundScale) / roundScale
}
