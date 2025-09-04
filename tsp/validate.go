// Package tsp - validation utilities shared by exact/heuristic solvers.
//
// This file contains small, tight, and well-documented helpers that:
//  1. Validate Options combinations (algo ↔ symmetric, bounds, limits).
//  2. Validate distance matrices (shape, diagonal, negativity, ∞, symmetry).
//  3. Validate/normalize auxiliary inputs (IDs, start vertex).
//
// Design principles:
//   - Deterministic, side-effect free functions.
//   - No logging, no panics on user input - only sentinel errors from types.go.
//   - O(n²) worst-case where n is the matrix size; no hidden allocations.
package tsp

import (
	"math"
	"time"

	"github.com/katalvlaran/lvlath/matrix"
)

// symTol is a structural tolerance for symmetry/diagonal checks in matrices.
// It is independent from Options.Eps (which governs "improvement" in local search).
const symTol = 1e-12

// validateAll verifies Options + distance matrix + optional vertex IDs.
// It returns n (matrix order) on success.
//
// Contract:
//   - dist must be non-nil, square, and of size n≥2 for non-trivial TSP.
//   - ids is optional; if provided, len(ids) must equal n and contain unique, non-empty strings.
//   - Symmetry is enforced if required by the selected algorithm (e.g., Christofides).
//   - If opts.RunMetricClosure==false, +Inf off-diagonal entries are rejected.
//
// Complexity: O(n²) time, O(n) extra space when ids!=nil (uniqueness check).
func validateAll(dist matrix.Matrix, ids []string, opts Options) (int, error) {
	var (
		n   int
		err error
	)

	// Stage 1: Options-only sanity.
	if err = validateOptionsStandalone(opts); err != nil {
		return 0, err
	}

	// Stage 2: Matrix shape/values with algorithm-driven symmetry requirement.
	n, err = validateDistMatrix(dist, mustEnforceSymmetry(opts), opts.RunMetricClosure, symTol)
	if err != nil {
		return 0, err
	}

	// Stage 3: Start vertex range (after n is known).
	if err = validateStartVertex(n, opts.StartVertex); err != nil {
		return 0, err
	}

	// Stage 4: Optional IDs validation.
	if ids != nil {
		if err = validateIDs(ids, n); err != nil {
			return 0, err
		}
	}

	return n, nil
}

// validateOptionsStandalone checks internal consistency of Options without
// referencing matrices or tours. Algo↔Symmetric constraints are enforced here.
//
// Complexity: O(1).
func validateOptionsStandalone(opts Options) error {
	// TimeLimit must be non-negative (negative durations are undefined).
	if opts.TimeLimit < 0 {
		return ErrDimensionMismatch
	}
	// Eps is the acceptance tolerance for Δ<−Eps. A negative epsilon would invert
	// the acceptance logic and break optimality guarantees ⇒ reject.
	if opts.Eps < 0 {
		return ErrDimensionMismatch
	}
	// TwoOpt/ThreeOpt iteration bound must be non-negative (0 ⇒ unlimited).
	if opts.TwoOptMaxIters < 0 {
		return ErrDimensionMismatch
	}
	// Christofides requires a symmetric (metric) TSP instance.
	if opts.Algo == Christofides && !opts.Symmetric {
		return ErrATSPNotSupportedByAlgo
	}

	// OneTreeBound is symmetric-only (Held–Karp 1-tree).
	if opts.BoundAlgo == OneTreeBound && !opts.Symmetric {
		return ErrATSPNotSupportedByAlgo
	}

	// Accept only known algorithms; dispatcher may still return a runtime sentinel later.
	switch opts.Algo {
	case Christofides, ExactHeldKarp, TwoOptOnly, ThreeOptOnly, BranchAndBound:
		// ok
	default:
		return ErrUnsupportedAlgorithm
	}

	// ShuffleNeighborhood may be set regardless of Seed; seed==0 ⇒ deterministic stream.
	return nil
}

// mustEnforceSymmetry tells whether the chosen algorithm *requires* symmetry.
//
// Rationale:
//   - Christofides: strictly symmetric (and metric).
//   - ExactHeldKarp: supports both symmetric/ATSP.
//   - TwoOptOnly: supports both; no hard mathematical restriction.
//   - ThreeOptOnly/BranchAndBound: permissive for now.
//
// Complexity: O(1).
func mustEnforceSymmetry(opts Options) bool {
	if opts.Algo == Christofides {
		return true
	}

	return opts.Symmetric
}

// validateStartVertex verifies that start∈[0..n-1].
//
// Complexity: O(1).
func validateStartVertex(n int, start int) error {
	if start < 0 || start >= n {
		return ErrStartOutOfRange
	}

	return nil
}

// validateIDs enforces len(ids)==n, non-empty strings, and uniqueness.
//
// Complexity: O(n) time and O(n) extra space.
func validateIDs(ids []string, n int) error {
	if len(ids) != n {
		return ErrDimensionMismatch
	}
	seen := make(map[string]struct{}, n)

	var (
		i  int    // loop index
		id string // current ID under validation
		ok bool   // presence flag in the 'seen' set
	)
	for i = 0; i < n; i++ { // scan each ID
		id = ids[i] // read ID at position i
		// Empty or duplicate IDs violate the shape/uniqueness contract.
		if id == "" {
			return ErrDimensionMismatch
		}
		if _, ok = seen[id]; ok {
			return ErrDimensionMismatch
		}
		seen[id] = struct{}{} // mark ID as seen
	}

	return nil
}

// validateDistMatrix performs full matrix validation:
//   - non-nil, square, n>=2,
//   - diagonal ≈ 0 (|a_ii| ≤ tol), finite,
//   - no negative off-diagonal distances,
//   - if !allowInf: reject +Inf/−Inf off-diagonal,
//   - if symmetric==true: |a_ij − a_ji| ≤ tol,
//   - NaN anywhere is invalid.
//
// Returns n (matrix order) on success.
//
// Complexity: O(n²).
func validateDistMatrix(dist matrix.Matrix, symmetric bool, allowInf bool, tol float64) (int, error) {
	// Stage 1: shape checks (non-nil, square).
	if dist == nil {
		return 0, ErrDimensionMismatch
	}
	var (
		nr int
		nc int
	)
	nr = dist.Rows()
	nc = dist.Cols()
	if nr != nc || nr <= 0 {
		return 0, ErrNonSquare
	}
	if nr == 1 {
		// Trivial n==1 instance: treat as invalid for general solvers (we require n>=2).
		return 0, ErrDimensionMismatch
	}
	var n int
	n = nr // the matrix order

	// Stage 2: diagonal, negativity, infinity, symmetry.
	var (
		i, j     int     // loop indices
		aij, aji float64 // matrix entries a[i][j] and a[j][i]
		err      error
		abs      float64 // scratch for |value|
	)

	// Diagonal: a_ii ≈ 0 within tol, finite.
	for i = 0; i < n; i++ { // iterate diagonal positions
		aij, err = dist.At(i, i) // read diagonal entry
		if err != nil {
			return 0, ErrDimensionMismatch
		}
		if math.IsNaN(aij) || math.IsInf(aij, 0) {
			return 0, ErrDimensionMismatch
		}
		abs = aij // absolute value without allocations
		if abs < 0 {
			abs = -abs // abs(aij)
		}
		if abs > tol {
			return 0, ErrNonZeroDiagonal
		}
	}

	// Off-diagonal scan.
	for i = 0; i < n; i++ { // rows
		for j = 0; j < n; j++ { // cols
			if i == j {
				continue // skip diagonal (already checked)
			}
			aij, err = dist.At(i, j) // read off-diagonal entry
			if err != nil {
				return 0, ErrDimensionMismatch
			}
			if math.IsNaN(aij) {
				return 0, ErrDimensionMismatch
			}
			if aij < 0 {
				return 0, ErrNegativeWeight
			}
			if math.IsInf(aij, 0) && !allowInf {
				return 0, ErrIncompleteGraph
			}
		}
	}

	// Symmetry (if required).
	if symmetric {
		for i = 0; i < n; i++ { // upper triangle
			for j = i + 1; j < n; j++ { // avoid double work
				aij, err = dist.At(i, j) // a_ij
				if err != nil {
					return 0, ErrDimensionMismatch
				}
				aji, err = dist.At(j, i) // a_ji
				if err != nil {
					return 0, ErrDimensionMismatch
				}
				abs = aij - aji // difference to test symmetry
				if abs < 0 {
					abs = -abs // |a_ij - a_ji|
				}
				if abs > tol {
					return 0, ErrAsymmetry
				}
			}
		}
	}

	return n, nil
}

// compatibleTimeBudget returns whether the remaining time budget is positive.
// Policy: 0 means "unlimited".
//
// Complexity: O(1).
func compatibleTimeBudget(tl time.Duration) bool {
	if tl == 0 {
		return true
	}
	// Negative handled in validateOptionsStandalone; here treat >0 as allowed.
	return tl > 0
}
