// Package tsp - 2-opt local search engine (symmetric 2-opt and asymmetric 2-opt*).
//
// TwoOpt performs deterministic first-improvement 2-opt on a closed tour.
//   - Symmetric case (opts.Symmetric == true): classic 2-opt reverses segment [i..k].
//     Δ = w(a,c) + w(b,d) − w(a,b) − w(c,d), with a=T[i−1], b=T[i], c=T[k], d=T[k+1].
//   - Asymmetric case (opts.Symmetric == false): 2-opt* “tail swap”, no reversal.
//     Replace arcs (a→b),(c→d) with (a→d),(c→b); Δ = w(a,d) + w(c,b) − w(a,b) − w(c,d).
//
// Design:
//   - Deterministic scanning order; no RNG usage (seed reserved for future shuffles).
//   - Strict sentinel errors only (see types.go). No fmt.Errorf in hot paths.
//   - Defensive but allocation-conscious: O(1) per-check; O(n) only on accepted move.
//   - Soft time budget via compatibleTimeBudget + periodic deadline checks.
//   - Cost stabilized to 1e−9 via round1e9.
//
// Contracts:
//   - dist is n×n; validateAll already ran in the dispatcher.
//   - tour is a *closed* Hamiltonian cycle (len==n+1, tour[0]==tour[n]==opts.StartVertex).
//   - For asymmetric instances, the solver uses 2-opt* (does not reverse segments).
//
// Complexity:
//   - One pass: O(n²) candidate checks; first-improvement restarts after each accepted move.
//   - Each accepted move costs O(1) (symmetric) or O(n) (asymmetric, rebuild successor list).
//   - Overall: O(iter*n²) time typical; O(n) extra space on improvements only.
package tsp

import (
	"math"
	"time"

	"github.com/katalvlaran/lvlath/matrix"
)

// TwoOpt runs deterministic first-improvement 2-opt / 2-opt* starting from initTour.
// Returns the improved tour (same start) and its stabilized cost.
func TwoOpt(dist matrix.Matrix, initTour []int, opts Options) ([]int, float64, error) {
	// --- Shape & invariants (cheap; full matrix validation is done earlier).
	if initTour == nil || len(initTour) < 2 {
		return nil, 0, ErrDimensionMismatch
	}
	n := len(initTour) - 1
	if n < 2 { // a closed cycle needs at least two distinct vertices
		return nil, 0, ErrDimensionMismatch
	}
	if err := ValidateTour(initTour, n, opts.StartVertex); err != nil {
		return nil, 0, err
	}

	// Prefetch weights into a dense 1D buffer w[i*n + j] to remove interface indirection
	// from hot loops. We also enforce sentinel semantics:
	//   - NaN          → ErrDimensionMismatch (ill-posed input),
	//   - negative     → ErrNegativeWeight   (forbidden),
	//   - +Inf allowed → candidate moves that rely on +Inf are simply rejected.
	w := make([]float64, n*n)
	{
		var (
			i, j int     // matrix indices; declared outside loops to avoid rebinds
			x    float64 // temporary holder for At(i,j)
			err  error
		)
		for i = 0; i < n; i++ {
			for j = 0; j < n; j++ {
				x, err = dist.At(i, j)
				if err != nil {
					return nil, 0, ErrDimensionMismatch
				}
				if math.IsNaN(x) {
					return nil, 0, ErrDimensionMismatch
				}
				if x < 0 {
					return nil, 0, ErrNegativeWeight
				}
				// Store in linearized form for cache-friendly reads: w[u*n+v] ~ At(u,v).
				w[i*n+j] = x
			}
		}
	}
	at := func(u, v int) float64 { return w[u*n+v] } // hot-path accessor with zero allocations

	// Current working tour (copy to keep the input immutable).
	cur := make([]int, n+1)
	copy(cur, initTour)

	// Baseline cost with strict checks (rejects +Inf/NaN on existing edges).
	cost, err := TourCost(dist, cur)
	if err != nil {
		return nil, 0, err
	}

	// Policy knobs.
	eps := opts.Eps
	if eps < 0 {
		// Defensive clamp: validateOptionsStandalone already forbids negative eps,
		// but we keep this to guarantee acceptance rule Δ < −eps is well-posed.
		eps = 0
	}
	maxIters := opts.TwoOptMaxIters // 0 ⇒ unlimited (until local optimum)

	// Soft deadline (checked sparsely to keep overhead negligible).
	var (
		useDeadline bool      // whether we enforce a wall-clock time budget
		deadline    time.Time // absolute deadline if enabled
		step        int       // iteration counter to throttle checks
	)
	if compatibleTimeBudget(opts.TimeLimit) && opts.TimeLimit > 0 {
		useDeadline = true
		deadline = time.Now().Add(opts.TimeLimit)
	}
	// Check every 2048 iterations (~cheap). This preserves throughput in tight loops.
	checkDeadline := func() bool {
		step++
		if !useDeadline || (step&2047) != 0 {
			return false
		}

		return time.Now().After(deadline)
	}

	// Main first-improvement loop: restart scan after every accepted move.
	accepted := 0
	for {
		improved := false // toggled to true exactly when a move is applied

		// Variables reused in inner loops to avoid re-declarations in hot path.
		var (
			a, b, c, d    int     // boundary endpoints around (i,k)
			delta         float64 // candidate improvement (negative is good)
			wab, wad, wcb float64 // baseline / new arcs (ATSP)
			wcd, wac, wbd float64 // baseline / new arcs (symmetric)
			i, k          int     // candidate cut indices, 1 ≤ i < k ≤ n−1
		)

		// Scan all candidate pairs (i,k) with 1 ≤ i < k ≤ n−1.
		// In 2-opt* (ATSP) we skip (i==1 && k==n−1) to avoid creating start→start directly.
		for i = 1; i <= n-2; i++ {
			for k = i + 1; k <= n-1; k++ {
				if !opts.Symmetric && i == 1 && k == n-1 {
					// Would connect start→start in the middle of the tour; skip.
					continue
				}

				// a=T[i−1], b=T[i], c=T[k], d=T[k+1]
				a = cur[i-1]
				b = cur[i]
				c = cur[k]
				d = cur[k+1]

				if opts.Symmetric {
					// Classic 2-opt reversal on [i..k].
					wab = at(a, b)
					wcd = at(c, d)
					wac = at(a, c)
					wbd = at(b, d)

					// If the new edges do not exist, reject this candidate.
					if math.IsInf(wac, 0) || math.IsInf(wbd, 0) {
						continue
					}
					// Δ = new − old; accept strictly improving (beyond tolerance).
					delta = (wac + wbd) - (wab + wcd)
					if delta < -eps {
						// Apply by in-place reversal of segment [i..k] (O(k−i+1)).
						if err = reverseArcInPlace(cur, i, k); err != nil {
							return nil, 0, err
						}
					} else {
						continue // not improving
					}
				} else {
					// ATSP - 2-opt* (no reversal). Replace (a→b),(c→d) with (a→d),(c→b).
					wab = at(a, b)
					wcd = at(c, d)
					wad = at(a, d)
					wcb = at(c, b)

					if math.IsInf(wad, 0) || math.IsInf(wcb, 0) {
						continue // candidate would introduce missing arcs
					}
					delta = (wad + wcb) - (wab + wcd)
					if delta < -eps {
						// Apply by rewiring successors and rebuilding the sequence (O(n)).
						cur = applyTwoOptStar(cur, opts.StartVertex, i, k)
					} else {
						continue // not improving
					}
				}

				// Update cost and bookkeeping after an accepted move.
				cost += delta
				accepted++
				improved = true

				// Guards.
				if maxIters > 0 && accepted >= maxIters {
					_ = CanonicalizeOrientationInPlace(cur)

					return cur, round1e9(cost), nil
				}
				if checkDeadline() {
					return nil, 0, ErrTimeLimit
				}

				// First-improvement policy: restart scanning from the beginning.
				break
			}
			if improved {
				break
			}
		}

		if !improved {
			// Local optimum under the chosen neighborhood.
			break
		}
	}

	_ = CanonicalizeOrientationInPlace(cur)
	// Defensive: keep invariants tight and explicit before returning.
	if verr := ValidateTour(cur, n, opts.StartVertex); verr != nil {
		return nil, 0, verr
	}

	return cur, round1e9(cost), nil
}

// applyTwoOptStar applies an asymmetric 2-opt* move on a closed tour.
//
// Notation (closed tour T of length n+1):
//
//	i,k are indices with 1 ≤ i < k ≤ n−1.
//	a=T[i−1], b=T[i], c=T[k], d=T[k+1].
//
// The move removes arcs (a→b) and (c→d), and adds (a→d) and (c→b),
// without reversing the [i..k] segment. We implement it by:
//  1. Building a successor array succ[v] from the current tour.
//  2. Rewiring succ[a]=d and succ[c]=b.
//  3. Reconstructing a fresh sequence by following succ from start.
//
// Complexity: O(n) time, O(n) space (executed only on *accepted* moves).
func applyTwoOptStar(tour []int, start, i, k int) []int {
	n := len(tour) - 1

	// Build succ[v] from tour - one pass over n arcs of the cycle.
	succ := make([]int, n)
	var p, u, v int
	for p = 0; p < n; p++ {
		u = tour[p]
		v = tour[p+1]
		succ[u] = v
	}
	a := tour[i-1]
	b := tour[i]
	c := tour[k]
	d := tour[k+1]

	// Rewire successors to encode the 2-opt* swap: (a→d) and (c→b).
	succ[a] = d
	succ[c] = b

	// Rebuild the sequence by chasing successors from the fixed start vertex.
	out := make([]int, n+1)
	cur := start
	var idx int
	for idx = 0; idx < n; idx++ {
		out[idx] = cur
		cur = succ[cur]
	}
	out[n] = start // close the cycle

	return out
}
