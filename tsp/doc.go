// Package tsp provides Travelling Salesman Problem (TSP/ATSP) solvers over distance matrices with a consistent API,
// strict sentinel errors, deterministic behavior, and stable cost rounding (1e-9). The package exposes exact search,
// classical approximation, and local-search improvements behind a single dispatcher.
//
// # What & Why
//
// Given an n×n distance matrix dist, tsp computes a Hamiltonian cycle (tour)
// visiting all vertices once and returning to the start.
//
//   - Exact: Held–Karp dynamic programming (ExactHeldKarp) and Branch-and-Bound
//     (BranchAndBound) with admissible lower bounds.
//   - Approximation (symmetric metric): Christofides 1.5-approx (Christofides).
//   - Local search: deterministic 2-opt / 3-opt post-passes (TwoOptOnly, ThreeOptOnly)
//     usable standalone or via dispatcher post-processing.
//
// # Algorithms & Complexity
//
//	ExactHeldKarp (Held–Karp DP) — supports TSP and ATSP
//	  Time:   O(n²·2ⁿ)     Memory: O(n·2ⁿ)
//	  Guards: MaxExactN (=16) to bound resources.
//
//	BranchAndBound (exact DFS with pruning) — supports TSP and ATSP
//	  Bound:  degree-1 relaxation (admissible) and optional root-only 1-tree LB (TSP only).
//	  Branch: neighbors sorted by weight then index (deterministic).
//	  Time:   exponential    Memory: O(n) path + O(n²) precomputes.
//
//	Christofides (1.5-approx) — symmetric metric TSP only
//	  Pipeline: MST → minimum perfect matching (Blossom when available; else Greedy) →
//	            Eulerian circuit → shortcut to tour.
//	  Time:   typically O(n²) on dense metric instances.
//
//	TwoOptOnly / ThreeOptOnly (local search) — TSP and ATSP
//	  2-opt (TSP): segment reversal; Δ = (a→c)+(b→d)−(a→b)−(c→d).
//	  2-opt* (ATSP): tail swap without reversals.
//	  3-opt: 7 reconnections (TSP) / 3-opt* (ATSP).
//	  Deterministic first/best-improvement, optional shuffled enumeration via Seed.
//
// # Determinism & Stability
//
//   - No time-based randomness. Any randomized scan uses Seed; Seed==0 gives fixed stream.
//   - Tie-breaks use indices. Costs are rounded to 1e-9 (round1e9) to avoid FP drift.
//   - CanonicalizeOrientationInPlace fixes tour direction under a fixed start vertex.
//
// # Input Requirements
//
//	dist must be a square n×n matrix, n≥2.  Diagonal ≈ 0 (|a_ii| ≤ 1e-12).  No negatives.
//	NaN is invalid.  +Inf denotes “missing edge” (allowed in most solvers; see below).
//
//	Symmetry (dist[i][j]==dist[j][i]) is required when:
//	  • opts.Algo == Christofides
//	  • opts.BoundAlgo == OneTreeBound
//	  • or opts.Symmetric == true (explicit user request)
//
//	If opts.RunMetricClosure==false the validator rejects +Inf off-diagonal entries.
//	Otherwise, matrix-level metric closure (e.g., Floyd–Warshall) may be applied upstream.
//
// # Options
//
//	type Options struct {
//	    StartVertex int           // start/end vertex [0..n-1] (default 0)
//	    Algo        Algorithm     // Christofides / ExactHeldKarp / TwoOptOnly / ThreeOptOnly / BranchAndBound
//	    Symmetric   bool          // require symmetry where needed (true by default)
//	    MatchingAlgo MatchingAlgo // Christofides: GreedyMatch or BlossomMatch (fallback to Greedy on sentinel)
//	    BoundAlgo   BoundAlgo     // BranchAndBound: NoBound / SimpleBound / OneTreeBound (TSP only)
//	    RunMetricClosure bool     // allow solving partially connected graphs via closure
//	    EnableLocalSearch bool    // run 2-opt (and 3-opt) post-passes where applicable
//	    TwoOptMaxIters int        // cap accepted moves (0=unlimited)
//	    BestImprovement bool      // LS policy: best vs first improvement
//	    ShuffleNeighborhood bool  // shuffle candidate order (deterministic via Seed)
//	    Eps         float64       // minimal strict improvement (default 1e-12)
//	    TimeLimit   time.Duration // soft wall-clock budget (0=none)
//	    Seed        int64         // deterministic RNG seed (0=stable default)
//	}
//
//	func DefaultOptions() Options
//
// # Errors (strict sentinels)
//
//	ErrNonSquare, ErrNegativeWeight, ErrAsymmetry, ErrNonZeroDiagonal,
//	ErrIncompleteGraph, ErrDimensionMismatch, ErrStartOutOfRange,
//	ErrMatchingNotImplemented, ErrUnsupportedAlgorithm, ErrTimeLimit,
//	ErrNodeLimit, ErrATSPNotSupportedByAlgo.
//
// Errors are never wrapped with fmt.Errorf where a sentinel suffices.
//
// # Results
//
//	type TSResult struct {
//	    Tour []int    // len==n+1, Tour[0]==Tour[n]==StartVertex, each 0..n-1 appears once
//	    Cost float64  // rounded to 1e-9
//	}
//
// # Mathematics (references)
//
//	2-opt Δ:  (a→c)+(b→d)−(a→b)−(c→d)
//	1-tree dual bound (Held–Karp, symmetric):
//	  L(π) = cost_{c'}(T(π)) − 2·Σ π_i,
//	  c'_{ij} = c_{ij} + π_i + π_j.
//	Costs are stabilized by round1e9 for cross-platform reproducibility.
//
// See: docs/TSP.md for full tutorial with math, pseudocode, diagrams, and best practices.
package tsp
