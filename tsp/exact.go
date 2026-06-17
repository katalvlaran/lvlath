// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package tsp - Held–Karp exact solver (DP O(n²·2ⁿ)) for TSP/ATSP.
//
// TSPExact computes an optimal Hamiltonian cycle using the Held–Karp dynamic
// programming algorithm. Symmetry is NOT required here (ATSP is allowed);
// Christofides-specific symmetry checks are enforced by the dispatcher.
//
// Contracts:
//   - dist is a complete finite square n×n final solver matrix.
//   - diagonal ≈ 0; no NaN/-Inf; negative weights are forbidden.
//   - +Inf off-diagonal is rejected before DP because final TSP kernels consume complete instances.
//   - opts.StartVertex ∈ [0..n−1].
//
// Behavior:
//   - A soft size limit via MaxExactN (default 16) bounds time/space.
//   - If opts.TimeLimit > 0, we periodically check a deadline and return ErrTimeLimit.
//   - Final cost is stabilized to 1e−9 (round1e9) for cross-platform reproducibility.
//
// Complexity:
//   - Time  : O(n²·2ⁿ).
//   - Memory: O(n·2ⁿ) for DP and parent tables.
//
// Returns:
//   - TSResult{Tour, Cost} with tour invariants (len==n+1, start==end==opts.StartVertex).
package tsp

import (
	"errors"
	"math"
	"math/bits"
	"time"

	"github.com/katalvlaran/lvlath/matrix"
)

// MaxExactN bounds problem size for the Held–Karp solver (time/memory guard).
const MaxExactN = 16

// ErrSizeTooLarge signals that n exceeds MaxExactN (pragmatic resource limit).
var ErrSizeTooLarge = errors.New("tsp: exact solver supports at most 16 vertices")

// HeldKarp solves a complete TSP/ATSP instance with the Held-Karp dynamic program.
// It is the canonical exact DP entrypoint and publishes TSPResult metadata directly.
//
// Implementation:
//   - Stage 1: Validate options, final solver matrix shape, and StartVertex.
//   - Stage 2: Execute the existing flat-array Held-Karp DP without changing mathematics.
//   - Stage 3: Publish a detached TSPResult with Exact=true and Optimal=true.
//
// Behavior highlights:
//   - Supports symmetric TSP and asymmetric TSP.
//   - Returns no partial result on timeout.
//   - Preserves the existing deterministic parent reconstruction law.
//
// Inputs:
//   - dist: complete finite square distance matrix.
//   - opts: solver policy; MaxExactN bounds exact DP size.
//
// Returns:
//   - *TSPResult: exact optimal tour result on success.
//   - error: nil on success or a sentinel-classified failure.
//
// Errors:
//   - ErrInvalidOptions from option validation.
//   - ErrNilDistanceMatrix, ErrNonSquare, ErrDimensionMismatch from matrix validation.
//   - ErrNaNInf, ErrNegativeWeight, ErrIncompleteGraph from final solver matrix checks.
//   - ErrStartOutOfRange for invalid StartVertex.
//   - ErrSizeTooLarge when n exceeds the exact DP guard.
//   - ErrTimeLimit when a positive wall-clock budget is exhausted.
//
// Determinism:
//   - Fixed subset-size order, mask order, endpoint order, and predecessor tie policy.
//
// Complexity:
//   - Time O(n^2 * 2^n), Space O(n * 2^n).
//
// Notes:
//   - Use SolveMatrix for facade-level ID attachment and optional metric closure.
//   - This function intentionally does not fall back to heuristics when size limits fail.
//
// AI-Hints:
//   - Do not silently replace Held-Karp with heuristic output on ErrSizeTooLarge.
//   - Do not mark a timed-out result as Optimal unless incumbent support is implemented.
func HeldKarp(dist matrix.Matrix, opts Options) (*TSPResult, error) {
	minimal, err := heldKarpMinimal(dist, opts)
	if err != nil {
		return nil, err
	}

	meta := newSolveMeta(ExactHeldKarp)
	meta.exact = true
	meta.optimal = true

	result := publishTSPResult(minimal, nil, opts, meta, false)
	result.Algorithm = ExactHeldKarp

	return result, nil
}

// TSPExact runs the Held-Karp DP over any matrix.Matrix and returns the legacy
// minimal TSResult projection.
//
// Deprecated: use HeldKarp or SolveMatrix.
func TSPExact(dist matrix.Matrix, opts Options) (TSResult, error) {
	result, err := HeldKarp(dist, opts)
	if result == nil {
		return TSResult{}, err
	}

	return result.Minimal(), err
}

// heldKarpMinimal contains the existing Held-Karp DP implementation.
// It remains private so canonical public code can publish TSPResult while the
// compatibility wrapper can still project the same mathematics to TSResult.
func heldKarpMinimal(dist matrix.Matrix, opts Options) (TSResult, error) {
	if err := validateOptionsStandalone(opts); err != nil {
		return TSResult{}, err
	}

	weights, err := copyCompleteWeights(dist, false)
	if err != nil {
		return TSResult{}, err
	}
	n := weights.n
	maxExactN := opts.MaxExactN
	if maxExactN == 0 {
		maxExactN = DefaultMaxExactN
	}
	if n > maxExactN {
		return TSResult{}, ErrSizeTooLarge
	}
	if err = validateStartVertex(n, opts.StartVertex); err != nil {
		return TSResult{}, err
	}

	// Use the shared detached weight buffer so exact DP preserves the same
	// numeric and sentinel policy as local search and Branch-and-Bound.
	w := weights.w

	// Soft time budget: cheap deadline checks at a low fixed cadence.
	var (
		useDeadline bool
		deadline    time.Time
		step        int
	)
	if opts.TimeLimit > 0 {
		useDeadline = true
		deadline = time.Now().Add(opts.TimeLimit)
	}
	checkDeadline := func() bool {
		// Increment a local counter and check the wall clock every 1024 invocations.
		// This keeps overhead negligible vs. DP work in tight loops.
		step++
		if !useDeadline || (step&1023) != 0 {
			return false
		}
		return time.Now().After(deadline)
	}

	// DP tables in a flat layout to avoid [][] indexing overhead:
	//   dp[mask*n + j]     - min cost to visit the set "mask" and end at j (mask always contains "start"),
	//   parent[mask*n + j] - predecessor of j in the optimal transition into (mask, j).
	totalMasks := 1 << uint(n)
	dp := make([]float64, totalMasks*n)
	parent := make([]int, totalMasks*n)

	// Initialize dp to +Inf and parent to −1.
	for idx := 0; idx < totalMasks*n; idx++ {
		dp[idx] = math.Inf(1)
		parent[idx] = -1
	}

	start := opts.StartVertex
	startBit := 1 << uint(start)
	baseMask := startBit
	dp[baseMask*n+start] = 0 // base state: at start, only start visited

	// Precompute lists of masks by popcount to avoid repeated popcount in hot loops.
	// We only keep masks that include the start bit.
	masksBySize := make([][]int, n+1)
	var mask int
	for mask = 0; mask < totalMasks; mask++ {
		if (mask & startBit) == 0 {
			continue
		}
		ps := bits.OnesCount(uint(mask))
		if ps >= 1 && ps <= n {
			masksBySize[ps] = append(masksBySize[ps], mask)
		}
	}

	// Main DP: grow subset size |mask| from 2..n.
	var (
		size int
		jbit int
		kbit int
		j, k int
		prev int
		wij  float64
	)
	for size = 2; size <= n; size++ {
		for _, mask = range masksBySize[size] {
			// For each possible endpoint j in "mask", j ≠ start:
			for j = 0; j < n; j++ {
				jbit = 1 << uint(j)
				if j == start || (mask&jbit) == 0 {
					continue
				}
				prev = mask ^ jbit // predecessor subset w/o j
				// Relax over all k ∈ prev: dp[mask,j] = min_k dp[prev,k] + w[k→j].
				var best float64
				best = math.Inf(1)
				var argk = -1

				for k = 0; k < n; k++ {
					kbit = 1 << uint(k)
					if (prev & kbit) == 0 {
						continue
					}
					var base = dp[prev*n+k]
					if math.IsInf(base, 1) {
						continue // unreachable state
					}
					wij = w[k*n+j]
					if math.IsInf(wij, 0) {
						continue // no edge k→j
					}
					var cand = base + wij
					if cand < best {
						best = cand
						argk = k
					}
				}
				if argk >= 0 {
					dp[mask*n+j] = best
					parent[mask*n+j] = argk
				}

				if checkDeadline() {
					return TSResult{}, ErrTimeLimit
				}
			}
		}
	}

	// Close the tour back to start: choose the best last vertex j and add w[j→start].
	all := totalMasks - 1
	var (
		bestCost = math.Inf(1)
		last     = -1
	)
	for j = 0; j < n; j++ {
		if j == start {
			continue
		}
		var base = dp[all*n+j]
		if math.IsInf(base, 1) {
			continue
		}
		wij = w[j*n+start]
		if math.IsInf(wij, 0) {
			continue
		}
		var total = base + wij
		if total < bestCost {
			bestCost = total
			last = j
		}
	}
	if last < 0 || math.IsInf(bestCost, 1) {
		return TSResult{}, ErrIncompleteGraph
	}

	// Reconstruct the optimal tour by walking parents backward from (mask=all, j=last).
	tour := make([]int, n+1)
	tour[0] = start
	tour[n] = start

	mask = all
	cur := last
	for idx := n - 1; idx >= 1; idx-- {
		tour[idx] = cur
		prev = parent[mask*n+cur]
		mask ^= 1 << uint(cur) // remove cur from the subset
		cur = prev
	}

	// Canonicalize direction (fixed start) and enforce final tour invariants.
	_ = CanonicalizeOrientationInPlace(tour)
	if verr := ValidateTour(tour, n, start); verr != nil {
		return TSResult{}, verr
	}

	return TSResult{
		Tour: tour,
		Cost: round1e9(bestCost),
	}, nil
}
