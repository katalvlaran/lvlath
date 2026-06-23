// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package tsp defines tour validation, canonicalization, construction, and cost helpers.
//
// A tour is represented as a closed vertex-index sequence of length n+1:
// tour[0] == tour[n] == start, and every vertex in [0,n) appears exactly once
// before the closing start vertex.
//
// Contracts:
//   - ValidateTour checks closed Hamiltonian cycle shape.
//   - ValidatePermutation checks open permutation shape without a closing vertex.
//   - TourCost computes the sum of directed arc costs in tour order.
//   - Dense and generic cost paths must preserve the same sentinel behavior.
//
// Complexity:
//   - Tour validation is O(n) time and O(n) memory.
//   - Tour cost is O(n) time and O(1) memory, excluding matrix access cost.
//   - Canonical rotation is O(n) time and O(n) memory unless performed in place.
//
// AI-Hints:
//   - Do not use map iteration for tour canonicalization.
//   - Do not treat permutation and closed-tour representations as interchangeable.
//   - Do not let dense and generic cost paths drift in sentinel behavior.
package tsp

import (
	"errors"
	"math"

	"github.com/katalvlaran/lvlath/matrix"
)

// roundScale controls final cost stabilization precision (1e-9).
// Avoids tiny FP drifts across platforms/opt levels without affecting optimality.
const roundScale = 1e9

// ValidatePermutation checks that perm is a permutation of {0..n-1} of length n.
// It does not allocate besides a single O(n) boolean marker slice.
//
// Complexity: O(n) time, O(n) space.
func ValidatePermutation(perm []int, n int) error {
	if len(perm) != n {
		return ErrDimensionMismatch
	}
	if n <= 0 {
		return ErrDimensionMismatch
	}
	seen := make([]bool, n)

	var (
		i int
		v int
	)
	for i = 0; i < n; i++ {
		v = perm[i]
		// Out-of-range element violates the dimension contract.
		if v < 0 || v >= n {
			return ErrDimensionMismatch
		}
		// Duplicate also violates the bijection/dimension contract.
		if seen[v] {
			return ErrDimensionMismatch
		}
		seen[v] = true
	}
	return nil
}

// MakeTourFromPermutation builds a closed Hamiltonian tour from a vertex permutation.
// Steps:
//  1. Validate that perm is a permutation of {0..n-1}.
//  2. Find the index of start in perm; rotate so perm[idx]==start becomes position 0.
//  3. Return a new slice of length n+1 with the closing start at position n.
//
// Contract:
//   - perm is a permutation (ValidatePermutation).
//   - start ∈ [0..n-1] and present in perm.
//   - Returned tour satisfies: len==n+1, tour[0]==tour[n]==start.
//
// Complexity: O(n) time, O(n) space.
func MakeTourFromPermutation(perm []int, n int, start int) ([]int, error) {
	if err := ValidatePermutation(perm, n); err != nil {
		return nil, err
	}
	if start < 0 || start >= n {
		return nil, ErrStartOutOfRange
	}

	// Locate start inside perm.
	var (
		i     int
		pivot = -1
	)
	for i = 0; i < n; i++ {
		if perm[i] == start {
			pivot = i
			break
		}
	}
	if pivot == -1 {
		// The permutation does not contain start - inconsistent input shape.
		return nil, ErrDimensionMismatch
	}

	// Rotate into a fresh [n+1] tour and close with start.
	tour := make([]int, n+1)
	for i = 0; i < n; i++ {
		tour[i] = perm[(pivot+i)%n]
	}
	tour[n] = start
	return tour, nil
}

// ValidateTour verifies a closed Hamiltonian cycle over n vertices.
// Implementation:
//   - Stage 1: reject nil and malformed shape.
//   - Stage 2: verify fixed start/end closure.
//   - Stage 3: scan each non-closing vertex exactly once.
//
// Behavior highlights:
//   - Does not allocate beyond the O(n) seen bitmap.
//   - Does not canonicalize; it only validates.
//
// Inputs:
//   - tour: closed cycle of length n+1.
//   - n: vertex count.
//   - start: required first and last vertex.
//
// Returns:
//   - error: nil if the tour is valid.
//
// Errors:
//   - ErrNilTour for nil slices.
//   - ErrInvalidTour for wrong length, wrong closure, duplicate, or out-of-range vertex.
//   - ErrStartOutOfRange for invalid start.
//
// Determinism:
//   - Fixed left-to-right scan.
//
// Complexity:
//   - Time O(n), Space O(n).
//
// AI-Hints:
//   - Do not use ErrDimensionMismatch for semantic tour violations.
func ValidateTour(tour []int, n int, start int) error {
	if tour == nil {
		return ErrNilTour
	}
	if start < 0 || start >= n {
		return ErrStartOutOfRange
	}
	if n <= 0 || len(tour) != n+1 {
		return ErrInvalidTour
	}
	if tour[0] != start || tour[n] != start {
		return ErrInvalidTour
	}

	seen := make([]bool, n)

	var i, v int
	for i = 0; i < n; i++ {
		v = tour[i]
		if v < 0 || v >= n {
			return ErrInvalidTour
		}
		if seen[v] {
			return ErrInvalidTour
		}
		seen[v] = true
	}

	return nil
}

// RotateTourToStart returns a fresh copy of the tour shifted so that
// out[0] == start and out[n] == start. The input may be either a closed tour
// (len==n+1) or a raw path (len==n, no closing vertex). In the raw-path case,
// the function appends the closing start.
//
// Pre-conditions:
//   - start must appear in tour at least once within the first n elements.
//
// Complexity: O(n) time, O(n) space.
func RotateTourToStart(tour []int, start int) ([]int, error) {
	if len(tour) == 0 {
		return nil, ErrDimensionMismatch
	}

	// Determine n (number of unique vertices).
	var (
		n        int
		isClosed bool
	)
	if tour[0] == tour[len(tour)-1] {
		n = len(tour) - 1
		isClosed = true
	} else {
		n = len(tour)
	}
	if start < 0 || start >= n {
		return nil, ErrStartOutOfRange
	}

	// Find start in the first n entries.
	var (
		i     int
		pivot = -1
	)
	for i = 0; i < n; i++ {
		if tour[i] == start {
			pivot = i
			break
		}
	}
	if pivot == -1 {
		return nil, ErrDimensionMismatch
	}

	// Build rotated copy and close it.
	out := make([]int, n+1)

	for i = 0; i < n; i++ {
		out[i] = tour[(pivot+i)%n]
	}
	out[n] = start

	_ = isClosed // documented above; not used in fast path.
	return out, nil
}

// CanonicalizeOrientationInPlace fixes the tour direction under a fixed start.
// If the right neighbor tour[1] is lexicographically “worse” than the left
// neighbor tour[n-1], the interior segment [1..n-1] is reversed in place.
// This yields a unique canonical orientation for the same cyclic order.
//
// Requirements:
//   - len(tour) == n+1 and tour[0]==tour[n] (already closed).
//   - The permutation part is assumed valid.
//
// Complexity: O(n) time, O(1) space.
func CanonicalizeOrientationInPlace(tour []int) error {
	if len(tour) < 3 {
		return ErrDimensionMismatch
	}
	var n = len(tour) - 1
	if tour[0] != tour[n] {
		return ErrDimensionMismatch
	}
	// Compare right vs left neighbor of start (indices 1 and n-1).
	if tour[1] > tour[n-1] {
		return reverseArcInPlace(tour, 1, n-1)
	}
	return nil
}

// reverseArcInPlace reverses the inclusive segment tour[i..k] in place,
// keeping the closing vertex intact. This is the primitive used by 2-opt.
//
// Contracts:
//   - The tour is closed: len(tour)==n+1 and tour[0]==tour[n].
//   - Indices satisfy: 1 ≤ i < k ≤ n-1.
//
// Complexity: O(k-i) time, O(1) space.
func reverseArcInPlace(tour []int, i, k int) error {
	var n = len(tour) - 1
	if n < 2 {
		return ErrDimensionMismatch
	}
	if tour[0] != tour[n] {
		return ErrDimensionMismatch
	}
	if i < 1 || k > n-1 || i >= k {
		return ErrDimensionMismatch
	}
	for i < k {
		tour[i], tour[k] = tour[k], tour[i]
		i++
		k--
	}
	return nil
}

// TourCost computes the total cost of a closed Hamiltonian tour.
// Implementation:
//   - Stage 1: validate nil matrix and nil/short tour inputs.
//   - Stage 2: choose Dense fast-path or generic matrix.Matrix path.
//   - Stage 3: each path validates edge indices and numeric weights.
//
// Behavior highlights:
//   - Does not mutate the tour or matrix.
//   - Dense and generic paths preserve identical sentinel classification.
//
// Inputs:
//   - dist: square distance matrix.
//   - tour: closed cycle of vertex indices.
//
// Returns:
//   - float64: rounded total cost.
//   - error: sentinel-classified failure.
//
// Errors:
//   - ErrNilDistanceMatrix joined with matrix nil sentinels.
//   - ErrNilTour for nil tour.
//   - ErrInvalidTour for malformed non-nil tour.
//   - ErrNonSquare for bad matrix shape.
//   - ErrNaNInf for NaN/-Inf.
//   - ErrIncompleteGraph for +Inf tour edges.
//   - ErrNegativeWeight for negative finite weights.
//
// Determinism:
//   - Fixed tour order scan from index 0 to len(tour)-2.
//
// Complexity:
//   - Time O(len(tour)), Space O(1).
//
// AI-Hints:
//   - Do not classify NaN as ErrDimensionMismatch.
//   - Keep Dense and generic path error classes equivalent.
func TourCost(dist matrix.Matrix, tour []int) (float64, error) {
	if err := matrix.ValidateNotNil(dist); err != nil {
		return 0, errors.Join(ErrNilDistanceMatrix, err)
	}
	if tour == nil {
		return 0, ErrNilTour
	}
	if len(tour) < 2 {
		return 0, ErrInvalidTour
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
			return 0, ErrInvalidTour
		}

		// Fetch weight and validate.
		w, err = d.At(u, v)
		if err != nil {
			// Dense.At should only fail on OOB; map to shape sentinel.
			return 0, ErrDimensionMismatch
		}
		if math.IsNaN(w) {
			return 0, errors.Join(ErrNaNInf, matrix.ErrNaNInf)
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
			return 0, ErrInvalidTour
		}

		w, err = m.At(u, v)
		if err != nil {
			return 0, ErrDimensionMismatch
		}
		if math.IsNaN(w) {
			return 0, errors.Join(ErrNaNInf, matrix.ErrNaNInf)
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
	if err := matrix.ValidateNotNil(m); err != nil {
		return 0, errors.Join(ErrNilDistanceMatrix, err)
	}

	nr := m.Rows()
	nc := m.Cols()

	if nr != nc || nr <= 0 {
		return 0, ErrNonSquare
	}
	if u < 0 || u >= nr || v < 0 || v >= nr {
		return 0, ErrInvalidTour
	}

	w, err := m.At(u, v)
	if err != nil {
		return 0, errors.Join(ErrDimensionMismatch, err)
	}
	if math.IsNaN(w) || math.IsInf(w, -1) {
		return 0, errors.Join(ErrNaNInf, matrix.ErrNaNInf)
	}
	if math.IsInf(w, 1) {
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
