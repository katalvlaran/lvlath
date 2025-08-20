// Package tsp — 3-opt local search (symmetric 3-opt and ATSP 3-opt*).
//
// ThreeOpt performs local search over 3-edge exchanges on a closed tour.
// Policies:
//   - First-improvement (default): apply the first strictly improving move.
//   - Best-improvement (opt-in via Options.BestImprovement): scan whole neighborhood and pick the best.
//
// Neighborhood order:
//   - If Options.ShuffleNeighborhood == true, triples (i,j,k) are scanned in a randomized,
//     constraint-respecting cyclic order using rngFromSeed(opts.Seed). seed==0 ⇒ deterministic stream.
//   - If false, a canonical deterministic order is used.
//
// Symmetric vs Asymmetric:
//   - Symmetric (opts.Symmetric==true): classic 3-opt over S1=T[i..j-1], S2=T[j..k-1] with tail S3=T[k..n-1] fixed.
//     We evaluate 7 reconnections in {S1,rev(S1)}×{S2,rev(S2)} \ {identity}.
//     Δ = (a→first(X))+(last(X)→first(Y))+(last(Y)→f) − [(a→b)+(c→d)+(e→f)],
//     where a=T[i−1], b=T[i], c=T[j−1], d=T[j], e=T[k−1], f=T[k]. Internal arcs cancel by symmetry.
//   - Asymmetric (ATSP): 3-opt* without reversals. With fixed tail S3, the only orientation-preserving
//     reconnection is the tail-swap: out = P + S2 + S1 + S3. Δ uses the same three boundary arcs.
//     Candidates that introduce +Inf are rejected.
//
// Contracts & complexity: same defensive guards as two_opt.go; cost stabilized to 1e−9.
package tsp

import (
	"math"
	"time"

	"github.com/katalvlaran/lvlath/matrix"
)

// segKind enumerates segment variants for symmetric 3-opt.
type segKind uint8

const (
	segS1  segKind = iota // segment S1 = T[i..j-1] in forward order
	segS1R                // reversed S1
	segS2                 // segment S2 = T[j..k-1] in forward order
	segS2R                // reversed S2
)

// ThreeOpt returns an improved tour and its stabilized cost.
// Policy is taken from opts.BestImprovement; ATSP uses 3-opt* (tail-swap).
func ThreeOpt(dist matrix.Matrix, initTour []int, opts Options) ([]int, float64, error) {
	return threeOptCore(dist, initTour, opts, opts.BestImprovement)
}

// ThreeOptBest — explicit best-improvement entrypoint (policy forced to best).
func ThreeOptBest(dist matrix.Matrix, initTour []int, opts Options) ([]int, float64, error) {
	return threeOptCore(dist, initTour, opts, true /*bestImprovement*/)
}

// threeOptCore contains the shared engine. No logs/panics; strict sentinels only.
func threeOptCore(dist matrix.Matrix, initTour []int, opts Options, bestImprovement bool) ([]int, float64, error) {
	// Tour shape & invariants (the dispatcher already validated matrix shape).
	if initTour == nil || len(initTour) < 2 {
		return nil, 0, ErrDimensionMismatch
	}
	n := len(initTour) - 1
	if n < 2 { // a closed tour requires at least two vertices (n≥2)
		return nil, 0, ErrDimensionMismatch
	}
	// Validate the cycle invariants: closure, unique vertices, fixed start.
	if err := ValidateTour(initTour, n, opts.StartVertex); err != nil {
		return nil, 0, err
	}

	// Prefetch weights into a dense buffer to eliminate interface overhead in hot loops.
	w := make([]float64, n*n)
	var (
		i, j int     // matrix indices reused across loops
		x    float64 // temporary weight holder
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
			w[i*n+j] = x // linearized index; avoids [][] bounds/indirection in hot path
		}
	}
	at := func(u, v int) float64 { return w[u*n+v] } // fast weight accessor

	// Working copy and baseline tour cost (strict validation of current edges).
	cur := make([]int, n+1)
	copy(cur, initTour)              // keep caller’s slice immutable
	cost, err := TourCost(dist, cur) // verifies no NaN/+Inf on existing arcs
	if err != nil {
		return nil, 0, err
	}

	// Policy knobs.
	eps := opts.Eps                 // accept only Δ < −eps (eps≥0 validated beforehand)
	maxMoves := opts.TwoOptMaxIters // 0 ⇒ unlimited number of accepted moves

	// RNG for randomized triple order: enabled only when ShuffleNeighborhood is set.
	var rng randLite // tiny shim interface with Intn(int)
	if opts.ShuffleNeighborhood {
		rng = rngFromSeed(opts.Seed)
	}

	// Soft deadline (cheap periodic checks; negligible overhead).
	var (
		useDeadline bool      // whether to enforce a time budget
		deadline    time.Time // absolute deadline if enabled
		steps       int       // Δ-evaluation counter for sparse checks
	)
	if compatibleTimeBudget(opts.TimeLimit) && opts.TimeLimit > 0 {
		useDeadline = true
		deadline = time.Now().Add(opts.TimeLimit)
	}
	// Check every 4096 Δ evaluations; this keeps the check overhead tiny.
	checkDeadline := func() bool {
		steps++
		if !useDeadline || (steps&4095) != 0 { // every 4096 Δ-evals
			return false
		}

		return time.Now().After(deadline)
	}

	// Neighborhood templates.
	// Symmetric: the 7 distinct reconnections (X,Y) with X,Y ∈ {S1,S1R,S2,S2R}\{identity}.
	tryXSym := [...]segKind{segS1R, segS1, segS2R, segS1R, segS2, segS2R, segS2}
	tryYSym := [...]segKind{segS2, segS2R, segS1R, segS2R, segS1R, segS1, segS1}
	// ATSP (3-opt*): single orientation-preserving reconnection under fixed-tail model.
	const tryXATSP = segS2 // X=S2
	const tryYATSP = segS1 // Y=S1

	// Main improvement loop.
	accepted := 0
	for {
		found := false // did we discover an improving candidate in this sweep?

		// Best-improvement bookkeeping for a single sweep.
		bestDelta := 0.0            // most negative Δ seen so far
		var bestI, bestJ, bestK int // triple indices for the best move
		var bestX, bestY segKind    // segment choices for symmetric case

		// Randomized cyclic offset for the outermost index i (optional when rng!=nil).
		offI := 0
		if rng != nil && n > 3 {
			offI = rng.Intn(maxi(1, n-3)) // safe even at minimal n
		}

		// Enumerate all triples 1≤i<j<k≤n−1 with optional cyclic offsets to reduce structure bias.
		var (
			k                            int     // k index
			ii, jj, kk, m                int     // loop counters
			spanJ, spanK, offJ, offK     int     // per-level spans and offsets
			a, b, c, d, e, f             int     // boundary vertices around (i,j,k)
			xFirst, xLast, yFirst, yLast int     // boundary endpoints for X and Y
			xk, yk                       segKind // chosen segment kinds
			w1, w2, w3                   float64 // new boundary arc weights
			removed                      float64 // weight of removed arcs
			delta                        float64 // candidate improvement (negative is good)
		)
		for ii = 0; ii < n-3; ii++ {
			i = 1 + ((ii + offI) % (n - 3)) // ensure i ∈ [1..n-3] with cyclic shift

			spanJ = (n - 2) - i // j ∈ (i..n-2] ⇒ span of length (n-2)-i
			if spanJ <= 0 {
				continue // no feasible j for this i
			}
			offJ = 0
			if rng != nil {
				offJ = rng.Intn(spanJ) // independent cyclic offset per i
			}

			for jj = 0; jj < spanJ; jj++ {
				j = i + 1 + ((jj + offJ) % spanJ) // j ∈ [i+1..n-2]

				spanK = (n - 1) - j // k ∈ (j..n-1] ⇒ span of length (n-1)-j
				if spanK <= 0 {
					continue // no feasible k for this (i,j)
				}
				offK = 0
				if rng != nil {
					offK = rng.Intn(spanK) // independent cyclic offset per (i,j)
				}

				for kk = 0; kk < spanK; kk++ {
					k = j + 1 + ((kk + offK) % spanK) // k ∈ [j+1..n-1]

					// Boundary vertices around the three cuts:
					a, b = cur[i-1], cur[i]
					c, d = cur[j-1], cur[j]
					e, f = cur[k-1], cur[k]
					removed = at(a, b) + at(c, d) + at(e, f)

					if opts.Symmetric {
						// Evaluate 7 symmetric reconnections (X,Y).
						for m = 0; m < 7; m++ {
							if checkDeadline() {
								return nil, 0, ErrTimeLimit
							}
							xk = tryXSym[m]
							yk = tryYSym[m]

							// Determine boundary endpoints for X and Y under the chosen orientation.
							xFirst, xLast = segFirstLast(xk, b, c, d, e)
							yFirst, yLast = segFirstLast(yk, b, c, d, e)

							// New boundary arcs: (a→first(X)), (last(X)→first(Y)), (last(Y)→f).
							w1 = at(a, xFirst)
							w2 = at(xLast, yFirst)
							w3 = at(yLast, f)
							if math.IsInf(w1, 0) || math.IsInf(w2, 0) || math.IsInf(w3, 0) {
								continue // would introduce missing arc(s)
							}
							delta = (w1 + w2 + w3) - removed
							if delta >= -eps {
								continue // not strictly improving under tolerance
							}

							if !bestImprovement {
								// First-improvement: apply immediately and restart sweep.
								cur = apply3OptSym(cur, i, j, k, xk, yk)
								cost += delta
								accepted++
								found = true
							} else if delta < bestDelta {
								// Best-improvement: remember the best move within this sweep.
								bestDelta, bestI, bestJ, bestK, bestX, bestY = delta, i, j, k, xk, yk
								found = true
							}
							if found && !bestImprovement {
								break // restart after an accepted first-improvement move
							}
						}
					} else {
						// ATSP — 3-opt* tail-swap (orientation-preserving, no reversals).
						if checkDeadline() {
							return nil, 0, ErrTimeLimit
						}
						// Boundary endpoints for X=S2 and Y=S1.
						xFirst, xLast = segFirstLast(tryXATSP, b, c, d, e) // (d,e)
						yFirst, yLast = segFirstLast(tryYATSP, b, c, d, e) // (b,c)

						w1 = at(a, xFirst)     // a→d
						w2 = at(xLast, yFirst) // e→b
						w3 = at(yLast, f)      // c→f
						if math.IsInf(w1, 0) || math.IsInf(w2, 0) || math.IsInf(w3, 0) {
							continue // would introduce missing arc(s)
						}
						delta = (w1 + w2 + w3) - removed
						if delta >= -eps {
							continue // not improving
						}

						if !bestImprovement {
							cur = apply3OptATSP(cur, i, j, k) // out = P + S2 + S1 + S3
							cost += delta
							accepted++
							found = true
						} else if delta < bestDelta {
							bestDelta, bestI, bestJ, bestK, bestX, bestY = delta, i, j, k, tryXATSP, tryYATSP
							found = true
						}
					}

					// Early exit for first-improvement policy; best-improvement keeps scanning.
					if found && !bestImprovement {
						break
					}
				}
				if found && !bestImprovement {
					break
				}
			}
			if found && !bestImprovement {
				break
			}
		}

		// Best-improvement: apply the remembered best move once per sweep.
		if bestImprovement && found {
			if opts.Symmetric {
				cur = apply3OptSym(cur, bestI, bestJ, bestK, bestX, bestY)
			} else {
				cur = apply3OptATSP(cur, bestI, bestJ, bestK)
			}
			cost += bestDelta
			accepted++
		}

		// Termination guards.
		if !found {
			break // local optimum for the chosen neighborhood/policy
		}
		if maxMoves > 0 && accepted >= maxMoves {
			break // hit user-specified move cap
		}
	}

	_ = CanonicalizeOrientationInPlace(cur)
	if verr := ValidateTour(cur, n, opts.StartVertex); verr != nil {
		return nil, 0, verr
	}

	return cur, round1e9(cost), nil
}

// segFirstLast maps a segment kind to its first/last vertex endpoints
// given boundary markers: b=T[i], c=T[j-1], d=T[j], e=T[k-1].
// For reversed segments, endpoints swap as expected.
func segFirstLast(kind segKind, b, c, d, e int) (first, last int) {
	switch kind {
	case segS1:
		return b, c
	case segS1R:
		return c, b
	case segS2:
		return d, e
	default: // segS2R
		return e, d
	}
}

// apply3OptSym assembles out = P + X + Y + S3 (then closes with start).
// P=T[:i], S1=T[i:j], S2=T[j:k], S3=T[k:n].
func apply3OptSym(tour []int, i, j, k int, X, Y segKind) []int {
	n := len(tour) - 1
	P, S1, S2, S3 := tour[:i], tour[i:j], tour[j:k], tour[k:n]

	out := make([]int, 0, n+1)
	out = append(out, P...)

	emit := func(seg []int, reverse bool) {
		if !reverse {
			out = append(out, seg...)
			return
		}

		var t = len(seg) - 1
		for ; t >= 0; t-- {
			out = append(out, seg[t])
		}
	}

	// Emit X then Y according to selected orientations.
	switch X {
	case segS1:
		emit(S1, false)
	case segS1R:
		emit(S1, true)
	case segS2:
		emit(S2, false)
	default:
		emit(S2, true)
	}

	// Emit Y.
	switch Y {
	case segS1:
		emit(S1, false)
	case segS1R:
		emit(S1, true)
	case segS2:
		emit(S2, false)
	default:
		emit(S2, true)
	}

	// Tail unchanged and closure by start.
	out = append(out, S3...)
	out = append(out, tour[0])
	return out
}

// apply3OptATSP assembles the tail-swap out = P + S2 + S1 + S3 (no reversals).
func apply3OptATSP(tour []int, i, j, k int) []int {
	n := len(tour) - 1
	P, S1, S2, S3 := tour[:i], tour[i:j], tour[j:k], tour[k:n]

	out := make([]int, 0, n+1)
	out = append(out, P...)
	out = append(out, S2...)
	out = append(out, S1...)
	out = append(out, S3...)
	out = append(out, tour[0])

	return out
}

// maxi returns the maximum of two ints.
func maxi(a, b int) int {
	if a > b {
		return a
	}

	return b
}

// randLite is a tiny shim: any RNG that implements Intn(int) (e.g., *rand.Rand).
// The actual instance comes from rngFromSeed(opts.Seed); seed==0 ⇒ deterministic stream.
type randLite interface {
	Intn(n int) int
}
