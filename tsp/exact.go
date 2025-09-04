// Package tsp - Held–Karp exact solver (DP O(n²·2ⁿ)) for TSP/ATSP.
//
// TSPExact computes an optimal Hamiltonian cycle using the Held–Karp dynamic
// programming algorithm. Symmetry is NOT required here (ATSP is allowed);
// Christofides-specific symmetry checks are enforced by the dispatcher.
//
// Contracts (already enforced by the dispatcher before calling this function):
//   - dist is a square n×n matrix, n ≥ 2.
//   - diagonal ≈ 0; no NaN; negative weights are forbidden (sentinel).
//   - +Inf is allowed and means “no direct edge”; if no cycle exists ⇒ ErrIncompleteGraph.
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

// TSPExact runs the Held–Karp DP over any matrix.Matrix (symmetric or asymmetric).
func TSPExact(dist matrix.Matrix, opts Options) (TSResult, error) {
	// Light shape guards (full validation was done by the dispatcher).
	if dist == nil {
		return TSResult{}, ErrNonSquare
	}
	var (
		nr = dist.Rows()
		nc = dist.Cols()
	)
	if nr != nc || nr <= 0 {
		return TSResult{}, ErrNonSquare
	}
	if nr < 2 {
		return TSResult{}, ErrDimensionMismatch
	}
	if nr > MaxExactN {
		return TSResult{}, ErrSizeTooLarge
	}
	var n = nr

	// Start vertex range.
	if err := validateStartVertex(n, opts.StartVertex); err != nil {
		return TSResult{}, err
	}

	// Prefetch weights into a dense 1D buffer w[i*n + j] to remove interface overhead
	// from the DP hot loops. Also enforce sentinel semantics here:
	// NaN → ErrDimensionMismatch; negative → ErrNegativeWeight; +Inf is allowed.
	w := make([]float64, n*n)
	var (
		i, j int
		wij  float64
		err  error
	)
	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			wij, err = dist.At(i, j)
			if err != nil {
				return TSResult{}, ErrDimensionMismatch
			}
			if math.IsNaN(wij) {
				return TSResult{}, ErrDimensionMismatch
			}
			if wij < 0 {
				return TSResult{}, ErrNegativeWeight
			}
			w[i*n+j] = wij
		}
	}

	// Soft time budget: cheap deadline checks at a low fixed cadence.
	var (
		useDeadline bool
		deadline    time.Time
		step        int
	)
	if compatibleTimeBudget(opts.TimeLimit) && opts.TimeLimit > 0 {
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
		k    int
		prev int
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
