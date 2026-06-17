// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package tsp - Branch-and-Bound (exact search with admissible lower bounds).
//
// TSPBranchAndBound enumerates Hamiltonian cycles via a depth-first
// Branch-and-Bound (BnB) search with deterministic branching, admissible
// lower bounds, and a soft time budget. Both symmetric TSP and asymmetric
// ATSP are supported.
//
// Rationale (succinct):
//  1. Strict input shape and invariants are enforced by the dispatcher;
//     here we prefetch the distance matrix into a dense buffer to remove
//     interface overhead in hot loops.
//  2. Optional seeding of an initial upper bound (UB): Christofides (+ 2-opt)
//     for symmetric TSP, or a deterministic trivial ring polished by 2-opt*
//     for ATSP. A good UB dramatically strengthens pruning.
//     UB costs are stored unrounded inside the search state; display rounding
//     happens only when publishing TSPResult.
//  3. Search: DFS with a degree-1 relaxation lower bound (LB):
//     - For vertices whose outgoing edge is not yet fixed, add minOut[v].
//     - For vertices whose incoming edge is not yet fixed, add minIn[v].
//     - LB_extra = max( sum(minOut), sum(minIn) ).
//     - LB = costSoFar + LB_extra. This bound is admissible (≤ OPT).
//     Prune whenever LB ≥ UB − eps.
//  4. Branching order: from the current “last”, try next vertices v in
//     ascending w[last→v] (index tiebreak). This tightens UB early while
//     remaining fully deterministic.
//  5. Soft time limit: every DFS node checks the shared stopped flag and deadline.
//     When the deadline expires, the whole recursion stops and a valid incumbent
//     is returned as a governed partial result when available.
//
// Complexity:
//   - Worst case exponential in n (exact search). Practical speed comes from pruning.
//   - Per node: O(n) bound + O(1) state updates.
//   - Memory: O(n) for the current path + O(n) for visited + O(n²) for precomputes
//     (min-in/out, neighbor orders).
//
// Governance:
//   - Options.BoundAlgo:
//     NoBound      → disables the lower bound (testing only).
//     SimpleBound  → degree-1 relaxation (implemented here).
//     OneTreeBound → root-only Held–Karp (1-tree) bound; see bound_onetree.go.

package tsp

import (
	"math"
	"sort"
	"time"

	"github.com/katalvlaran/lvlath/matrix"
)

const (
	// branchAndBoundNoIncumbent marks that no feasible Hamiltonian cycle has been recorded.
	branchAndBoundNoIncumbent = -1.0
)

// bbEngine holds all search data and policies.
// We use a dedicated engine struct (instead of anonymous closures) to keep
// dependencies explicit, testing simpler, and hot-path state predictable.
type bbEngine struct {
	// Configuration / policy
	n         int
	start     int
	symmetric bool
	useBound  bool
	eps       float64

	// Time budget
	useDeadline bool
	deadline    time.Time
	stopped     bool
	// Search telemetry
	nodesExpanded int

	// Graph data (detached dense buffer): weights.at(u,v) is cost u→v.
	weights weightBuffer

	// Precomputes for bound / branching order
	minOut []float64 // per-vertex minimal outgoing edge (excluding self)
	minIn  []float64 // per-vertex minimal incoming edge (excluding self)
	order  [][]int   // for each u: v≠u sorted by w[u→v] (index tiebreak)

	// Current search state
	visited []bool // which vertices are on the current path
	path    []int  // path[0:depth], path[0] == start

	// Current best incumbent (UB)
	bestTour []int
	bestCost float64

	// Whether a feasible Hamiltonian cycle has been found
	foundAny bool
}

// at is a fast accessor into the detached dense weight buffer.
func (e *bbEngine) at(u, v int) float64 { return e.weights.at(u, v) }

// deadlineExpired reports whether the Branch-and-Bound wall-clock budget is exhausted.
// It is intentionally checked before every node expansion so a timeout stops the whole
// DFS tree instead of merely returning from one recursive frame.
//
// Implementation:
//   - Stage 1: Return false when TimeLimit is disabled.
//   - Stage 2: Compare current wall-clock time to the fixed deadline.
//
// Behavior highlights:
//   - Does not mutate engine state.
//   - Used by dfs and finalization.
//
// Inputs:
//   - None; reads engine deadline policy.
//
// Returns:
//   - bool: true when the deadline has passed.
//
// Errors:
//   - None.
//
// Determinism:
//   - Deterministic when TimeLimit==0; wall-clock dependent only when a budget is active.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - The caller sets e.stopped when this returns true.
//
// AI-Hints:
//   - Do not use sparse counters as NodesExpanded.
//   - Do not keep searching sibling branches after this returns true.
func (e *bbEngine) deadlineExpired() bool {
	return e.useDeadline && time.Now().After(e.deadline)
}

// initWeights attaches a validated complete weight buffer to the search engine.
// Branch-and-Bound consumes final solver input, so missing +Inf edges must already
// be rejected by copyCompleteWeights before search starts.
//
// Implementation:
//   - Stage 1: Verify that the buffer order matches the engine order.
//   - Stage 2: Store the detached weight buffer for O(1) hot-loop access.
//
// Behavior highlights:
//   - Does not copy weights again.
//   - Does not mutate the source matrix.
//   - Keeps B&B sentinel classification aligned with all final solver kernels.
//
// Inputs:
//   - weights: detached complete row-major TSP weights.
//
// Returns:
//   - error: nil when the buffer matches engine dimensions.
//
// Errors:
//   - ErrDimensionMismatch when weights.n does not match e.n.
//
// Determinism:
//   - Pure shape check and assignment.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Build weights with copyCompleteWeights, not copyClosureReadyWeights.
//
// AI-Hints:
//   - Do not allow +Inf into B&B final search state.
//   - Do not reintroduce a custom matrix.At prefetch loop here.
func (e *bbEngine) initWeights(weights weightBuffer) error {
	if weights.n != e.n {
		return ErrDimensionMismatch
	}

	e.weights = weights

	return nil
}

// tourCost computes an unrounded tour cost from the engine weight buffer.
// It is used for incumbent bounds; display stabilization happens only when
// publishing TSPResult.
//
// Implementation:
//   - Stage 1: Validate closed-tour length and fixed start.
//   - Stage 2: Scan all arcs in tour order.
//   - Stage 3: Accumulate raw float64 cost without round1e9.
//
// Behavior highlights:
//   - Does not mutate the tour.
//   - Rejects malformed tours before they become pruning bounds.
//   - Reads from weightBuffer, not matrix.Matrix.
//
// Inputs:
//   - tour: closed Hamiltonian cycle candidate.
//
// Returns:
//   - float64: raw unrounded cycle cost.
//   - error: nil on valid finite tour.
//
// Errors:
//   - ErrInvalidTour for malformed tour, out-of-range vertices, or missing closure.
//   - ErrIncompleteGraph for impossible +Inf arc, defensive only after copyCompleteWeights.
//
// Determinism:
//   - Fixed left-to-right arc scan.
//
// Complexity:
//   - Time O(n), Space O(1).
//
// Notes:
//   - This helper intentionally does not call TourCost because TourCost rounds for public metadata.
//
// AI-Hints:
//   - Do not use rounded public result costs as Branch-and-Bound UB.
func (e *bbEngine) tourCost(tour []int) (float64, error) {
	if err := ValidateTour(tour, e.n, e.start); err != nil {
		return 0, err
	}

	var (
		index int
		from  int
		to    int
		cost  float64
		sum   float64
	)
	for index = 0; index < e.n; index++ {
		from = tour[index]
		to = tour[index+1]
		cost = e.at(from, to)

		if math.IsInf(cost, 0) {
			return 0, ErrIncompleteGraph
		}

		sum += cost
	}

	return sum, nil
}

// precomputeMinima computes per-vertex minOut/minIn excluding self-loops.
// If any vertex has no finite outgoing or incoming edge to other vertices,
// the instance is infeasible for TSP/ATSP and we return ErrIncompleteGraph.
func (e *bbEngine) precomputeMinima() error {
	var (
		inf    = math.Inf(1)
		v, u   int
		mo, mi float64
		cOut   float64
		cIn    float64
	)
	e.minOut = make([]float64, e.n)
	e.minIn = make([]float64, e.n)
	for v = 0; v < e.n; v++ {
		mo, mi = inf, inf
		for u = 0; u < e.n; u++ {
			if u == v {
				continue
			}
			cOut = e.at(v, u)
			if cOut < mo {
				mo = cOut
			}
			cIn = e.at(u, v)
			if cIn < mi {
				mi = cIn
			}
		}
		e.minOut[v] = mo
		e.minIn[v] = mi
		if math.IsInf(mo, 0) || math.IsInf(mi, 0) {
			return ErrIncompleteGraph
		}
	}

	return nil
}

// neighborOrder implements sort.Interface for a row of neighbors ordered by weight.
type neighborOrder struct {
	u   int
	row []int
	e   *bbEngine
}

func (no neighborOrder) Len() int { return len(no.row) }
func (no neighborOrder) Less(i, j int) bool {
	vi, vj := no.row[i], no.row[j]
	wi, wj := no.e.at(no.u, vi), no.e.at(no.u, vj)
	if wi == wj {
		return vi < vj
	}

	return wi < wj
}
func (no *neighborOrder) Swap(i, j int) { no.row[i], no.row[j] = no.row[j], no.row[i] }

// buildNeighborOrder produces, for each u, the list of v≠u sorted by ascending w[u→v]
// (and then by v). Deterministic branching reduces UB time-to-tighten and keeps runs reproducible.
func (e *bbEngine) buildNeighborOrder() {
	var u, v int
	e.order = make([][]int, e.n)
	for u = 0; u < e.n; u++ {
		row := make([]int, 0, e.n-1)
		for v = 0; v < e.n; v++ {
			if v != u {
				row = append(row, v)
			}
		}
		no := neighborOrder{u: u, row: row, e: e}
		sort.Sort(&no)
		e.order[u] = no.row
	}
}

// recordUB commits a new incumbent upper bound without rounding its cost.
// The bound participates in pruning, so cost must remain the raw accumulated
// value; round1e9 is applied only when publishing TSPResult.
func (e *bbEngine) recordUB(tour []int, cost float64) {
	copy(e.bestTour, tour)
	e.bestCost = cost
	e.foundAny = true
}

// seedUB optionally initializes UB via heuristics to accelerate pruning.
// Symmetric: Christofides (+ optional 2-opt). If still no UB, fall back to a
// canonical trivial ring with validation and optional TwoOpt(*/2-opt*) polishing.
func (e *bbEngine) seedUB(dist matrix.Matrix, opts Options) {
	e.bestCost = math.Inf(1)
	e.bestTour = make([]int, e.n+1)

	// Symmetric seed - Christofides; safe fallbacks are handled inside TSPApprox.
	if opts.Symmetric {
		if res, err := TSPApprox(dist, opts); err == nil {
			if cost, costErr := e.tourCost(res.Tour); costErr == nil {
				e.recordUB(res.Tour, cost)
			}
		}
	}

	// If UB is still +Inf, try a deterministic trivial ring (then optional 2-opt).
	if math.IsInf(e.bestCost, 0) {
		base, berr := trivialRing(e.n, e.start)
		if berr == nil {
			if c0, cerr := e.tourCost(base); cerr == nil {
				e.recordUB(base, c0)
				if opts.EnableLocalSearch && e.n >= 4 {
					if improvedTour, _, improveErr := TwoOpt(dist, base, opts); improveErr == nil {
						if improvedCost, improvedCostErr := e.tourCost(improvedTour); improvedCostErr == nil {
							e.recordUB(improvedTour, improvedCost)
						}
					}
				}
			}
		}
	}
}

// lowerBound implements the degree-1 relaxation (admissible for TSP/ATSP).
// In a Hamiltonian cycle each vertex has out-degree 1 and in-degree 1. For vertices
// where outgoing/incoming is not yet fixed by the partial path, the eventual edge
// cost is ≥ minOut[v] / ≥ minIn[v]. Therefore:
//
//	LB_extra ≥ max( sum(minOut over out-unfixed), sum(minIn over in-unfixed) )
//
// and LB = costSoFar + LB_extra is a valid lower bound on any completion.
func (e *bbEngine) lowerBound(costSoFar float64, last int, depth int) float64 {
	if !e.useBound {
		return costSoFar // NoBound policy (for testing/benchmarking).
	}
	var sumOut, sumIn float64
	sumOut, sumIn = 0, 0

	// Outgoing is fixed for all visited vertices except 'last';
	// incoming is fixed for all visited vertices except 'start'.
	var v int
	for v = 0; v < e.n; v++ {
		if e.visited[v] {
			if v == last {
				sumOut += e.minOut[v]
			}
			if v == e.start {
				sumIn += e.minIn[v]
			}
		} else {
			sumOut += e.minOut[v]
			sumIn += e.minIn[v]
		}
	}
	extra := sumOut
	if sumIn > extra {
		extra = sumIn
	}

	return costSoFar + extra
}

// commit writes the final start-closure and records a new incumbent upper bound.
// The total cost is raw search cost and must not be rounded before pruning completes.
//
// Implementation:
//   - Stage 1: Close the current path with the fixed start vertex.
//   - Stage 2: Copy the full path into bestTour.
//   - Stage 3: Store raw bestCost and mark that an incumbent exists.
//
// Behavior highlights:
//   - Does not allocate.
//   - Does not round bestCost.
//
// Inputs:
//   - total: raw complete-cycle cost.
//
// Returns:
//   - None.
//
// Errors:
//   - None. Caller validates edge feasibility before commit.
//
// Determinism:
//   - Copies the current deterministic DFS path.
//
// Complexity:
//   - Time O(n), Space O(1).
//
// Notes:
//   - Publication-time rounding belongs to result().
//
// AI-Hints:
//   - Do not call round1e9 here; rounded UB can change pruning.
func (e *bbEngine) commit(total float64) {
	e.path[e.n] = e.start
	copy(e.bestTour, e.path)
	e.bestCost = total
	e.foundAny = true
}

// dfs performs the core search: deterministic branching + pruning by LB ≥ UB − eps.
func (e *bbEngine) dfs(last int, depth int, costSoFar float64) {
	if e.stopped {
		return
	}
	if e.deadlineExpired() {
		e.stopped = true
		return
	}
	e.nodesExpanded++

	// Prune by lower bound.
	if lb := e.lowerBound(costSoFar, last, depth); lb >= e.bestCost-e.eps {
		return
	}

	// If we used all vertices, close the cycle at start.
	if depth == e.n {
		c := e.at(last, e.start)
		if math.IsInf(c, 0) {
			return // missing closing edge
		}
		total := costSoFar + c
		if total < e.bestCost-e.eps {
			e.commit(total)
		}

		return
	}

	// Branch: iterate neighbors of 'last' in the precomputed order.
	var (
		v int
		c float64
	)
	for _, v = range e.order[last] {
		if e.stopped {
			return
		}
		if e.visited[v] {
			continue
		}
		c = e.at(last, v)
		if math.IsInf(c, 0) {
			continue
		}
		e.visited[v] = true
		e.path[depth] = v
		e.dfs(v, depth+1, costSoFar+c)
		e.visited[v] = false

		if e.stopped {
			return
		}
	}
}

// result publishes the current Branch-and-Bound incumbent as a canonical TSPResult.
// Implementation:
//   - Stage 1: Detach the best tour.
//   - Stage 2: Stabilize cost.
//   - Stage 3: Attach exact/optimal/timeout/search metadata.
//
// Behavior highlights:
//   - Requires e.bestTour to contain a valid incumbent.
//   - Does not mutate engine state.
//
// Returns:
//   - *TSPResult: detached result snapshot.
//
// Complexity:
//   - Time O(n), Space O(n).
//
// AI-Hints:
//   - Do not expose engine pointers or mutable slices through TSPResult.
func (e *bbEngine) result(optimal bool, timedOut bool) *TSPResult {
	return &TSPResult{
		Tour:          append([]int(nil), e.bestTour...),
		Cost:          round1e9(e.bestCost),
		Algorithm:     BranchAndBound,
		Exact:         true,
		Optimal:       optimal,
		TimedOut:      timedOut,
		NodesExpanded: e.nodesExpanded,
		Symmetric:     e.symmetric,
	}
}

// BranchAndBoundSolve runs exact Branch-and-Bound search and publishes TSPResult directly.
// It is the canonical BnB entrypoint for callers that need timeout, exactness, and
// search telemetry metadata without going through SolveMatrix.
//
// Implementation:
//   - Stage 1: Delegate to the result-native Branch-and-Bound engine.
//   - Stage 2: Return the canonical result unchanged.
//
// Behavior highlights:
//   - Full completion returns Exact=true and Optimal=true.
//   - Timeout with a feasible incumbent returns a non-nil partial result and ErrTimeLimit.
//   - Timeout without an incumbent returns nil and ErrTimeLimit.
//
// Inputs:
//   - dist: final complete TSP/ATSP distance matrix.
//   - opts: finalized or caller-provided BnB policy.
//
// Returns:
//   - *TSPResult: optimal or partial result.
//   - error: nil on full completion, ErrTimeLimit for partial timeout, or validation sentinel.
//
// Errors:
//   - ErrTimeLimit if a positive time budget is exceeded.
//   - ErrIncompleteGraph if no Hamiltonian cycle exists (or all closures are +Inf).
//   - Strict validation sentinels for malformed inputs (see types.go).
//
// Determinism:
//   - Branching order is sorted by edge weight and vertex index.
//
// Complexity:
//   - Worst-case exponential time, O(n^2) precompute space plus O(n) search state.
//
// Notes:
//   - Use SolveMatrix when IDs or metric closure must be attached by the facade.
//
// AI-Hints:
//   - Do not project this result to TSResult before checking TimedOut and NodesExpanded.
//   - Do not mark timed-out partial results as Optimal.
func BranchAndBoundSolve(dist matrix.Matrix, opts Options) (*TSPResult, error) {
	return runBranchAndBoundResult(dist, opts)
}

// TSPBranchAndBound is the legacy entrypoint for exact BnB search.
// It returns only the minimal TSResult projection and therefore discards partial-result
// metadata such as TimedOut and NodesExpanded.
//
// Deprecated: use BranchAndBoundSolve or SolveMatrix.
func TSPBranchAndBound(dist matrix.Matrix, opts Options) (TSResult, error) {
	result, err := BranchAndBoundSolve(dist, opts)
	if err != nil {
		return TSResult{}, err
	}
	if result == nil {
		return TSResult{}, nil
	}

	return result.Minimal(), nil
}

// runBranchAndBoundResult executes exact Branch-and-Bound and can publish partial timeout results.
// Implementation:
//   - Stage 1: Validate options, matrix, and start vertex.
//   - Stage 2: Initialize dense engine buffers and deterministic branching order.
//   - Stage 3: Seed an incumbent upper bound.
//   - Stage 4: Run DFS with admissible pruning.
//   - Stage 5: Publish optimal, incomplete, or partial-timeout result.
//
// Behavior highlights:
//   - Exact when completed without timeout.
//   - On timeout, returns a partial TSPResult only if a feasible incumbent exists.
//   - Legacy wrappers may discard partial metadata, but canonical SolveMatrix preserves it.
//
// Inputs:
//   - dist: final complete TSP/ATSP distance matrix.
//   - opts: finalized solver policy.
//
// Returns:
//   - *TSPResult: optimal or partial result.
//   - error: nil, ErrTimeLimit, ErrIncompleteGraph, or validation sentinel.
//
// Errors:
//   - ErrInvalidOptions / ErrUnsupportedAlgorithm.
//   - ErrNilDistanceMatrix / ErrNonSquare / ErrDimensionMismatch.
//   - ErrNaNInf / ErrNegativeWeight / ErrIncompleteGraph.
//   - ErrStartOutOfRange.
//   - ErrTimeLimit with non-nil result when an incumbent exists.
//
// Determinism:
//   - Branch order is sorted by edge weight then vertex index.
//   - Deadline checks are sparse but do not change correctness, only whether partial result is returned.
//
// Complexity:
//   - Worst-case exponential time.
//   - Space O(n^2) precompute + O(n) search state.
//
// AI-Hints:
//   - Do not clear a valid incumbent on timeout.
//   - Do not mark timed-out results as Optimal.
func runBranchAndBoundResult(dist matrix.Matrix, opts Options) (*TSPResult, error) {
	if err := validateOptionsStandalone(opts); err != nil {
		return nil, err
	}

	weights, err := copyCompleteWeights(dist, opts.Symmetric)
	if err != nil {
		return nil, err
	}
	n := weights.n
	if err = validateStartVertex(n, opts.StartVertex); err != nil {
		return nil, err
	}

	// Engine initialization (no anonymous closures).
	var engine bbEngine
	engine.n = n
	engine.start = opts.StartVertex
	engine.symmetric = opts.Symmetric
	engine.eps = opts.Eps
	engine.useBound = opts.BoundAlgo != NoBound

	// Deadline setup.
	if opts.TimeLimit > 0 {
		engine.useDeadline = true
		engine.deadline = time.Now().Add(opts.TimeLimit)
	}

	// Attach already validated final solver weights.
	if err = engine.initWeights(weights); err != nil {
		return nil, err
	}
	if err = engine.precomputeMinima(); err != nil {
		return nil, err
	}
	engine.buildNeighborOrder()

	// Search state.
	engine.visited = make([]bool, n)
	engine.path = make([]int, n+1)
	engine.path[0] = engine.start
	engine.visited[engine.start] = true

	if engine.deadlineExpired() {
		engine.stopped = true
		return nil, ErrTimeLimit
	}

	// Optional UB seeding (greatly helps pruning, correctness unaffected).
	engine.seedUB(dist, opts)

	if engine.deadlineExpired() {
		engine.stopped = true
		if engine.foundAny && !math.IsInf(engine.bestCost, 0) {
			_ = CanonicalizeOrientationInPlace(engine.bestTour)
			if err = ValidateTour(engine.bestTour, n, engine.start); err != nil {
				return nil, err
			}
			return engine.result(false, true), ErrTimeLimit
		}
		return nil, ErrTimeLimit
	}

	// Optional: root-only 1-tree (Held–Karp) lower bound to tighten pruning
	// before entering DFS. Safe for symmetric instances; correctness unaffected.
	if opts.BoundAlgo == OneTreeBound && opts.Symmetric {
		cfg := DefaultOneTreeConfig()
		// If we already have a finite incumbent, pass it to the step schedule.
		if !math.IsInf(engine.bestCost, 0) && engine.bestCost > 0 {
			cfg.UB = engine.bestCost
		}

		if lowerBound, _, boundErr := OneTreeLowerBound(dist, engine.start, true, cfg); boundErr == nil {
			// If LB >= UB−eps at the root, the incumbent is optimal; return it.
			if !math.IsInf(engine.bestCost, 0) && lowerBound >= engine.bestCost-engine.eps {
				_ = CanonicalizeOrientationInPlace(engine.bestTour)
				if err = ValidateTour(engine.bestTour, n, engine.start); err != nil {
					return nil, err
				}

				return engine.result(true, false), nil
			}
		}
	}

	// Run DFS.
	engine.dfs(engine.start, 1, 0)

	// Finalization.
	if engine.stopped || engine.deadlineExpired() {
		engine.stopped = true
		if engine.foundAny && !math.IsInf(engine.bestCost, 0) {
			_ = CanonicalizeOrientationInPlace(engine.bestTour)
			if err = ValidateTour(engine.bestTour, n, engine.start); err != nil {
				return nil, err
			}

			return engine.result(false, true), ErrTimeLimit
		}

		return nil, ErrTimeLimit
	}

	if !engine.foundAny && math.IsInf(engine.bestCost, 0) {
		return nil, ErrIncompleteGraph
	}

	_ = CanonicalizeOrientationInPlace(engine.bestTour)
	if err = ValidateTour(engine.bestTour, n, engine.start); err != nil {
		return nil, err
	}

	return engine.result(true, false), nil
}
