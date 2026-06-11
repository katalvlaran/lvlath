// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package tsp - unified dispatcher for TSP solvers.
//
// This file provides the canonical entry points to run TSP algorithms:
//
//   - SolveWithGraph: accept *core.Graph, build an adjacency matrix (optionally
//     with metric closure), derive stable vertex IDs, then delegate to SolveWithMatrix.
//   - SolveWithMatrix: accept a distance matrix + optional IDs and route to the
//     requested algorithm (Christofides / Held–Karp / TwoOptOnly / ThreeOptOnly / …),
//     applying strict validation and optional local-search post-passes.
//
// Design principles:
//   - Deterministic: seed routing to heuristics; no time-based randomness.
//   - Strict sentinels: only errors from types.go; no fmt.Errorf where a sentinel suffices.
//   - Hot-path discipline: no hidden allocations; preallocate slices where needed.
//   - Algorithmic clarity: doc strings with complexity and contracts.
//   - Stable cost: all returned costs are rounded to 1e−9 to prevent FP drift.
package tsp

import (
	"errors"

	"github.com/katalvlaran/lvlath/matrix"
)

// trivialRing returns a canonical Hamiltonian cycle [start, start+1, …, n−1, 0, …, start]
// with closure; it allocates exactly n+1 integers and performs no matrix lookups.
//
// Contracts:
//   - 0 ≤ start < n; n ≥ 2.
//
// Complexity: O(n) time, O(n) space.
func trivialRing(n int, start int) ([]int, error) {
	if n < 2 {
		return nil, ErrDimensionMismatch
	}
	if start < 0 || start >= n {
		return nil, ErrStartOutOfRange
	}
	out := make([]int, n+1)

	var (
		i   int // loop iterator
		pos = 0 // independent index of the entry into the resulting slice.
	)

	// Fill from start to n-1.
	for i = start; i < n; i++ {
		out[pos] = i
		pos++
	}
	// Then wrap from 0 to start-1.
	for i = 0; i < start; i++ {
		out[pos] = i
		pos++
	}

	// Close the cycle by returning to start.
	out[n] = start

	return out, nil
}

// prepareSolverDistanceMatrix prepares direct matrix input for final TSP solver kernels.
// Implementation:
//   - Stage 1: Validate option policy before allocation.
//   - Stage 2: Validate pre-closure TSP distance semantics with complete=false.
//   - Stage 3: If metric closure is disabled, return the original matrix.
//   - Stage 4: If metric closure is enabled, copy values into a detached Dense matrix.
//   - Stage 5: Run matrix.APSPInPlace to compute Floyd-Warshall closure.
//   - Stage 6: Validate final TSP solver matrix with complete=true.
//
// Behavior highlights:
//   - Does not mutate caller-owned direct matrix inputs.
//   - Uses matrix.APSPInPlace as the single APSP kernel.
//   - Uses TSP validation only for TSP-specific laws: no negative weights and final completeness.
//
// Inputs:
//   - dist: direct distance matrix.
//   - opts: explicit solver policy.
//
// Returns:
//   - matrix.Matrix: original matrix or detached metric-closed matrix.
//   - bool: true when closure was applied.
//   - int: final matrix order.
//   - error: sentinel-classified failure.
//
// Errors:
//   - ErrInvalidOptions from option validation.
//   - ErrNilDistanceMatrix / ErrNonSquare / ErrDimensionMismatch from structure validation.
//   - ErrNaNInf, ErrNonZeroDiagonal, ErrNegativeWeight, ErrIncompleteGraph, ErrAsymmetry.
//   - ErrNegativeWeight joined with matrix.ErrNegativeCycle if APSP detects a negative cycle.
//
// Determinism:
//   - Copy is row-major.
//   - APSP uses matrix.FloydWarshall deterministic k→i→j order.
//   - Final validation uses deterministic row-major / upper-triangle scans.
//
// Complexity:
//   - Without closure: Time O(n^2), Space O(1).
//   - With closure: Time O(n^3), Space O(n^2).
//
// Notes:
//   - This is a policy-stage helper, not a second TSP algorithm.
//   - Graph inputs that already asked matrix.NewAdjacencyMatrix for metric closure should clear
//     RunMetricClosure before delegating to direct matrix solving to avoid double closure.
//
// AI-Hints:
//   - Do not replace this with “allow +Inf in validateAll”; that lets incomplete
//     final TSP instances reach kernels.
//   - Do not call matrix.InitDistancesInPlace here; direct matrix input already uses
//     +Inf as the missing-edge sentinel, not raw 0-as-no-edge adjacency.
func prepareSolverDistanceMatrix(dist matrix.Matrix, opts Options) (matrix.Matrix, bool, int, error) {
	if err := validateOptionsStandalone(opts); err != nil {
		return nil, false, 0, err
	}

	n, err := validateSolverDistanceMatrix(dist, mustEnforceSymmetry(opts), false, symTol)
	if err != nil {
		return nil, false, 0, err
	}

	if !opts.RunMetricClosure {
		n, err = validateSolverDistanceMatrix(dist, mustEnforceSymmetry(opts), true, symTol)
		if err != nil {
			return nil, false, 0, err
		}

		return dist, false, n, nil
	}

	closed, err := matrix.NewPreparedDense(n, n, matrix.WithAllowInfDistances())
	if err != nil {
		return nil, false, 0, errors.Join(ErrDimensionMismatch, err)
	}

	var (
		row, col int
		value    float64
		readErr  error
	)
	for row = 0; row < n; row++ {
		for col = 0; col < n; col++ {
			value, readErr = dist.At(row, col)
			if readErr != nil {
				return nil, false, 0, errors.Join(ErrDimensionMismatch, readErr)
			}

			if row == col {
				value = 0
			}

			if err = closed.Set(row, col, value); err != nil {
				return nil, false, 0, errors.Join(ErrNaNInf, err)
			}
		}
	}

	if err = matrix.APSPInPlace(closed); err != nil {
		if errors.Is(err, matrix.ErrNegativeCycle) {
			return nil, false, 0, errors.Join(ErrNegativeWeight, err)
		}
		if errors.Is(err, matrix.ErrNaNInf) {
			return nil, false, 0, errors.Join(ErrNaNInf, err)
		}

		return nil, false, 0, err
	}

	n, err = validateSolverDistanceMatrix(closed, mustEnforceSymmetry(opts), true, symTol)
	if err != nil {
		return nil, false, 0, err
	}

	return closed, true, n, nil
}

// solvePreparedMatrix routes an already validated, final TSP distance matrix to the selected kernel.
// Implementation:
//   - Stage 1: Dispatch by opts.Algo.
//   - Stage 2: Run the chosen kernel.
//   - Stage 3: Apply local-search post-pass where the selected policy permits it.
//   - Stage 4: Canonicalize and validate the final tour.
//   - Stage 5: Return the legacy minimal result for canonical wrapping.
//
// Behavior highlights:
//   - Assumes prepareSolverDistanceMatrix and validateIDs already ran.
//   - Does not run metric closure.
//   - Does not mutate caller-owned matrix data.
//
// Inputs:
//   - dist: final complete solver matrix.
//   - opts: final options with RunMetricClosure=false.
//   - n: matrix order.
//
// Returns:
//   - TSResult: minimal successful solver output.
//   - error: sentinel-classified failure.
//
// Errors:
//   - Kernel-specific sentinels from TSPApprox, TSPExact, TwoOpt, ThreeOpt, TSPBranchAndBound.
//   - ErrUnsupportedAlgorithm for unknown algorithm.
//
// Determinism:
//   - Fixed switch dispatch and kernel-level tie-breaks.
//   - Tour canonicalization is applied before return.
//
// Complexity:
//   - Depends on selected algorithm.
//
// AI-Hints:
//   - Do not call this from public code.
//   - Do not add validation here except final invariant checks; public facades own validation.
func solvePreparedMatrix(dist matrix.Matrix, opts Options, n int) (TSResult, error) {
	switch opts.Algo {
	case Christofides:
		// Christofides requires symmetric metric; validated in validateAll.
		// 1) Build a feasible tour via TSPApprox.
		result, err := TSPApprox(dist, opts)
		if err != nil {
			return TSResult{}, err
		}

		// 2) Optional local search post-pass.
		//    If BestImprovement==false → a single TwoOpt pass (fast).
		//    If BestImprovement==true  → hybrid “2-opt → 3-opt (best) → 2-opt polish”
		//    (user opted in for stronger but slower refinement).
		if opts.EnableLocalSearch && compatibleTimeBudget(opts.TimeLimit) && n >= 4 {
			tour := result.Tour
			cost := result.Cost

			// Always start with a cheap 2-opt phase.
			tour2, cost2, err2 := TwoOpt(dist, tour, opts)
			if err2 != nil {
				return TSResult{}, err2
			}
			tour, cost = tour2, cost2

			if opts.BestImprovement {
				// Stronger middle pass: best-improvement 3-opt (ThreeOpt reads policy from opts).
				tour3, cost3, err3 := ThreeOpt(dist, tour, opts)
				if err3 != nil {
					return TSResult{}, err3
				}
				tour, cost = tour3, cost3

				// Final quick polish: one more 2-opt (often squeezes a bit more).
				tour4, cost4, err4 := TwoOpt(dist, tour, opts)
				if err4 != nil {
					return TSResult{}, err4
				}
				tour, cost = tour4, cost4
			}

			// Keep canonical orientation and invariants.
			_ = CanonicalizeOrientationInPlace(tour)
			if err = ValidateTour(tour, n, opts.StartVertex); err != nil {
				return TSResult{}, err
			}

			result.Tour = tour
			result.Cost = round1e9(cost)
		}

		return result, nil

	case ExactHeldKarp:
		// Exact DP; no post-pass needed.
		result, err := TSPExact(dist, opts)
		if err != nil {
			return TSResult{}, err
		}
		// Stabilize cost for cross-platform consistency.
		result.Cost = round1e9(result.Cost)
		return result, nil

	case TwoOptOnly:
		// Build a canonical initial tour (deterministic), then run TwoOpt.
		base, err := trivialRing(n, opts.StartVertex)
		if err != nil {
			return TSResult{}, err
		}

		tour, cost, err := TwoOpt(dist, base, opts)
		if err != nil {
			return TSResult{}, err
		}

		_ = CanonicalizeOrientationInPlace(tour)
		if err = ValidateTour(tour, n, opts.StartVertex); err != nil {
			return TSResult{}, err
		}

		return TSResult{Tour: tour, Cost: round1e9(cost)}, nil

	case ThreeOptOnly:
		// Canonical initial tour; deterministic seed.
		base, err := trivialRing(n, opts.StartVertex)
		if err != nil {
			return TSResult{}, err
		}

		// Optional warm-up 2-opt pass (fast).
		if opts.EnableLocalSearch && n >= 4 {
			tour2, _, err2 := TwoOpt(dist, base, opts)
			if err2 != nil {
				return TSResult{}, err2
			}
			base = tour2
		}

		// 3-opt with user-selected policy (first/best) and optional shuffle.
		tour, cost, err := ThreeOpt(dist, base, opts)
		if err != nil {
			return TSResult{}, err
		}

		// Optional final 2-opt polish (cheap).
		if opts.EnableLocalSearch && n >= 4 {
			tour2, cost2, err2 := TwoOpt(dist, tour, opts)
			if err2 != nil {
				return TSResult{}, err2
			}
			tour, cost = tour2, cost2
		}

		_ = CanonicalizeOrientationInPlace(tour)
		if err = ValidateTour(tour, n, opts.StartVertex); err != nil {
			return TSResult{}, err
		}

		return TSResult{Tour: tour, Cost: round1e9(cost)}, nil

	case BranchAndBound:
		return TSPBranchAndBound(dist, opts)

	default:
		return TSResult{}, ErrUnsupportedAlgorithm
	}
}

// nearestNeighbor (optional) - kept private for future use.
// Deterministic NN from start with a simple tie-breaker (smallest index).
// Not wired by default to keep dispatcher minimal and predictable.
// If you decide to use it later, validateAll must have allowed complete matrices.
//
// Complexity: O(n^2) time, O(n) space.
//
// func nearestNeighbor(dist matrix.Matrix, start int) ([]int, error) { … }
//
// We intentionally omit its body here - it will be introduced when we add
// richer initializers for TwoOpt/ThreeOpt per stages 6–7.
