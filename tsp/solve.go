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
	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/matrix"
)

// SolveWithGraph converts g into a distance matrix (according to its flags),
// optionally applies metric closure (opts.RunMetricClosure), and delegates
// to SolveWithMatrix.
//
// Contracts:
//   - g must be non-nil.
//   - Graph configuration (directed/weighted/loops/multi) is respected via matrix options.
//   - IDs are reconstructed from matrix vertex indices for round-trip fidelity.
//
// Errors: those from validateAll and underlying builders; see types.go.
//
// Complexity:
//   - Building adjacency: O(V^2 + E) (matrix init + edge pass).
//   - Delegation cost: per chosen algorithm (see SolveWithMatrix).
func SolveWithGraph(g *core.Graph, opts Options) (TSResult, error) {
	// Nil graph => invalid shape for building matrices.
	if g == nil {
		return TSResult{}, ErrDimensionMismatch
	}

	// Build matrix options from graph flags + dispatcher policy.
	// AllowMulti=true is safe; the builder will collate according to MatrixOptions semantics.
	var mopts = matrix.NewMatrixOptions(
		matrix.WithDirected(g.Directed()),
		matrix.WithWeighted(g.Weighted()),
		matrix.WithAllowLoops(g.Looped()),
		matrix.WithAllowMulti(true),
		matrix.WithMetricClosure(opts.RunMetricClosure),
	)

	am, err := matrix.NewAdjacencyMatrix(g, mopts)
	if err != nil {
		// NewAdjacencyMatrix returns matrix-level errors; forward them as-is.
		// Upstream validateAll will surface tsp sentinels when we dispatch via SolveWithMatrix.
		return TSResult{}, err
	}

	// Recover stable vertex ordering ids[idx] = id.
	// Map iteration order is irrelevant: we write by canonical index -> stable array.
	var (
		n   = am.Mat.Rows()
		ids = make([]string, n)
	)
	// VertexIndex is id -> index, so invert it.
	var (
		id  string
		idx int
	)
	for id, idx = range am.VertexIndex {
		ids[idx] = id
	}

	// Delegate to matrix dispatcher (unified validation is done there).
	return SolveWithMatrix(am.Mat, ids, opts)
}

// SolveWithMatrix validates inputs and routes to the chosen algorithm.
// Optionally performs local search post-passes when EnableLocalSearch is true
// (heuristics only; exact solvers return optimal tours as-is).
//
// Contracts:
//   - dist must be a square matrix; n ≥ 2 for non-trivial TSP.
//   - ids may be nil; if provided, len(ids)==n with unique, non-empty strings.
//   - Symmetry is enforced when required by the algorithm or opts.Symmetric.
//
// Errors: strict sentinels from types.go (e.g., ErrNonSquare, ErrAsymmetry,
// ErrIncompleteGraph, ErrUnsupportedAlgorithm, ErrATSPNotSupportedByAlgo).
//
// Complexity: validation O(n^2); the rest per algorithm:
//   - Christofides: O(n^2) for Prim + O(k^2) greedy matching (or blossom when present) +
//     O(E) Hierholzer + O(n) shortcut; typical dense cost bounded by O(n^2).
//   - Held–Karp:   O(n^2·2^n).
//   - TwoOptOnly:  O(iter·n^2) (see two_opt.go).
//   - ThreeOptOnly: O(iter·n^3) (see three_opt.go).
func SolveWithMatrix(dist matrix.Matrix, ids []string, opts Options) (TSResult, error) {
	// Stage 1 - unified validation (Options + matrix + ids).
	n, err := validateAll(dist, ids, opts)
	if err != nil {
		return TSResult{}, err
	}

	// Stage 2 - route by algorithm.
	var res TSResult
	switch opts.Algo {
	case Christofides:
		// Christofides requires symmetric metric; validated in validateAll.
		// 1) Build a feasible tour via TSPApprox.
		res, err = TSPApprox(dist, opts)
		if err != nil {
			return TSResult{}, err
		}

		// 2) Optional local search post-pass.
		//    If BestImprovement==false → a single TwoOpt pass (fast).
		//    If BestImprovement==true  → hybrid “2-opt → 3-opt (best) → 2-opt polish”
		//    (user opted in for stronger but slower refinement).
		if opts.EnableLocalSearch && compatibleTimeBudget(opts.TimeLimit) && n >= 4 {
			tour := res.Tour
			cost := res.Cost

			// Always start with a cheap 2-opt phase.
			if t2, c2, e2 := TwoOpt(dist, tour, opts); e2 == nil {
				tour, cost = t2, c2
			} else {
				return TSResult{}, e2
			}

			if opts.BestImprovement {
				// Stronger middle pass: best-improvement 3-opt (ThreeOpt reads policy from opts).
				if t3, c3, e3 := ThreeOpt(dist, tour, opts); e3 == nil {
					tour, cost = t3, c3
				} else {
					return TSResult{}, e3
				}
				// Final quick polish: one more 2-opt (often squeezes a bit more).
				if t4, c4, e4 := TwoOpt(dist, tour, opts); e4 == nil {
					tour, cost = t4, c4
				} else {
					return TSResult{}, e4
				}
			}

			// Keep canonical orientation and invariants.
			_ = CanonicalizeOrientationInPlace(tour)
			if verr := ValidateTour(tour, n, opts.StartVertex); verr == nil {
				res.Tour = tour
				res.Cost = round1e9(cost)
			}
		}

		return res, nil

	case ExactHeldKarp:
		// Exact DP; no post-pass needed.
		res, err = TSPExact(dist, opts)
		if err != nil {
			return TSResult{}, err
		}
		// Stabilize cost for cross-platform consistency.
		res.Cost = round1e9(res.Cost)

		return res, nil

	case TwoOptOnly:
		// Build a canonical initial tour (deterministic), then run TwoOpt.
		var base []int
		base, err = trivialRing(n, opts.StartVertex)
		if err != nil {
			return TSResult{}, err
		}
		var (
			best []int
			cost float64
		)
		best, cost, err = TwoOpt(dist, base, opts)
		if err != nil {
			return TSResult{}, err
		}
		_ = CanonicalizeOrientationInPlace(best)
		if verr := ValidateTour(best, n, opts.StartVertex); verr != nil {
			return TSResult{}, verr
		}

		return TSResult{Tour: best, Cost: round1e9(cost)}, nil

	case ThreeOptOnly:
		// Canonical initial tour; deterministic seed.
		var base []int
		base, err = trivialRing(n, opts.StartVertex)
		if err != nil {
			return TSResult{}, err
		}

		// Optional warm-up 2-opt pass (fast).
		if opts.EnableLocalSearch && n >= 4 {
			if tour2, _, err2 := TwoOpt(dist, base, opts); err2 == nil {
				base = tour2
			} else {
				return TSResult{}, err2
			}
		}

		// 3-opt with user-selected policy (first/best) and optional shuffle.
		var (
			best []int
			cost float64
		)
		best, cost, err = ThreeOpt(dist, base, opts)
		if err != nil {
			return TSResult{}, err
		}

		// Optional final 2-opt polish (cheap).
		if opts.EnableLocalSearch && n >= 4 {
			if tour2, cost2, err2 := TwoOpt(dist, best, opts); err2 == nil {
				best, cost = tour2, cost2
			} else {
				return TSResult{}, err2
			}
		}

		_ = CanonicalizeOrientationInPlace(best)
		if verr := ValidateTour(best, n, opts.StartVertex); verr != nil {
			return TSResult{}, verr
		}

		return TSResult{Tour: best, Cost: round1e9(cost)}, nil

	case BranchAndBound:
		res, err = TSPBranchAndBound(dist, opts)
		if err != nil {
			return TSResult{}, err
		}

		return res, nil

	default:
		return TSResult{}, ErrUnsupportedAlgorithm
	}
}

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
