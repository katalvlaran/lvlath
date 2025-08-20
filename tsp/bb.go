// Package tsp — Branch-and-Bound (exact search with admissible lower bounds).
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
//  3. Search: DFS with a degree-1 relaxation lower bound (LB):
//     - For vertices whose outgoing edge is not yet fixed, add minOut[v].
//     - For vertices whose incoming edge is not yet fixed, add minIn[v].
//     - LB_extra = max( sum(minOut), sum(minIn) ).
//     - LB = costSoFar + LB_extra. This bound is admissible (≤ OPT).
//     Prune whenever LB ≥ UB − eps.
//  4. Branching order: from the current “last”, try next vertices v in
//     ascending w[last→v] (index tiebreak). This tightens UB early while
//     remaining fully deterministic.
//  5. Soft time limit: rare deadline checks (every 4096 node events) keep
//     overhead negligible.
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
	steps       int // sparse deadline checks counter

	// Graph data (dense buffer): w[u*n+v]
	w []float64

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

// at is a fast accessor into the dense weight buffer.
func (e *bbEngine) at(u, v int) float64 { return e.w[u*e.n+v] }

// deadlineCheck performs a rare deadline test (every 4096 node events).
func (e *bbEngine) deadlineCheck() bool {
	e.steps++
	if !e.useDeadline || (e.steps&4095) != 0 {
		return false
	}

	return time.Now().After(e.deadline)
}

// initPrefetch loads the matrix into a dense buffer and applies strict sentinels.
// NaN and negative weights are rejected; +Inf is allowed (represents missing edges).
func (e *bbEngine) initPrefetch(dist matrix.Matrix) error {
	var (
		i, j int
		x    float64
		err  error
	)
	e.w = make([]float64, e.n*e.n)
	for i = 0; i < e.n; i++ {
		for j = 0; j < e.n; j++ {
			x, err = dist.At(i, j)
			if err != nil || math.IsNaN(x) {
				return ErrDimensionMismatch
			}
			if x < 0 {
				return ErrNegativeWeight
			}
			e.w[i*e.n+j] = x
		}
	}

	return nil
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

// recordUB commits a new incumbent (UB) with stabilized cost.
func (e *bbEngine) recordUB(tour []int, cost float64) {
	copy(e.bestTour, tour)
	e.bestCost = round1e9(cost)
}

// seedUB optionally initializes UB via heuristics to accelerate pruning.
// Symmetric: Christofides (+ optional 2-opt). If still no UB, fall back to a
// canonical trivial ring with validation and optional TwoOpt(*/2-opt*) polishing.
func (e *bbEngine) seedUB(dist matrix.Matrix, opts Options) {
	e.bestCost = math.Inf(1)
	e.bestTour = make([]int, e.n+1)

	// Symmetric seed — Christofides; safe fallbacks are handled inside TSPApprox.
	if opts.Symmetric {
		if res, err := TSPApprox(dist, opts); err == nil {
			e.recordUB(res.Tour, res.Cost)
		}
	}

	// If UB is still +Inf, try a deterministic trivial ring (then optional 2-opt).
	if math.IsInf(e.bestCost, 0) {
		base, berr := trivialRing(e.n, e.start)
		if berr == nil {
			if c0, cerr := TourCost(dist, base); cerr == nil {
				e.recordUB(base, c0)
				if opts.EnableLocalSearch && e.n >= 4 {
					if imp, ic, ierr := TwoOpt(dist, base, opts); ierr == nil {
						e.recordUB(imp, ic) // TwoOpt already returns stabilized cost.
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

// commit writes the final start-closure and records a new incumbent (UB).
func (e *bbEngine) commit(total float64, depth int) {
	e.path[e.n] = e.start
	copy(e.bestTour, e.path)
	e.bestCost = round1e9(total)
	e.foundAny = true
}

// dfs performs the core search: deterministic branching + pruning by LB ≥ UB − eps.
func (e *bbEngine) dfs(last int, depth int, costSoFar float64) {
	// Sparse time check (practically free).
	if e.deadlineCheck() {
		return
	}

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
			e.commit(total, depth)
		}

		return
	}

	// Branch: iterate neighbors of 'last' in the precomputed order.
	var (
		v int
		c float64
	)
	for _, v = range e.order[last] {
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
	}
}

// TSPBranchAndBound is the public entrypoint for exact BnB search.
// It prepares the engine, runs the search, and returns the optimal tour/cost.
//
// Errors:
//   - ErrTimeLimit if a positive time budget is exceeded.
//   - ErrIncompleteGraph if no Hamiltonian cycle exists (or all closures are +Inf).
//   - Strict validation sentinels for malformed inputs (see types.go).
func TSPBranchAndBound(dist matrix.Matrix, opts Options) (TSResult, error) {
	// Lightweight shape guard (full validation already performed in SolveWithMatrix).
	var n int
	n = dist.Rows()
	if n != dist.Cols() || n < 2 {
		return TSResult{}, ErrNonSquare
	}
	if err := validateStartVertex(n, opts.StartVertex); err != nil {
		return TSResult{}, err
	}

	// Engine initialization (no anonymous closures).
	var e bbEngine
	e.n = n
	e.start = opts.StartVertex
	e.symmetric = opts.Symmetric
	e.eps = opts.Eps
	if e.eps < 0 {
		e.eps = 0
	}
	e.useBound = (opts.BoundAlgo != NoBound)

	// Deadline setup.
	if compatibleTimeBudget(opts.TimeLimit) && opts.TimeLimit > 0 {
		e.useDeadline = true
		e.deadline = time.Now().Add(opts.TimeLimit)
	}

	// Prefetch and precomputes.
	if err := e.initPrefetch(dist); err != nil {
		return TSResult{}, err
	}
	if err := e.precomputeMinima(); err != nil {
		return TSResult{}, err
	}
	e.buildNeighborOrder()

	// Search state.
	e.visited = make([]bool, n)
	e.path = make([]int, n+1)
	e.path[0] = e.start
	e.visited[e.start] = true

	// Optional UB seeding (greatly helps pruning, correctness unaffected).
	e.seedUB(dist, opts)

	// Optional: root-only 1-tree (Held–Karp) lower bound to tighten pruning
	// before entering DFS. Safe for symmetric instances; correctness unaffected.
	if opts.BoundAlgo == OneTreeBound && opts.Symmetric {
		cfg := DefaultOneTreeConfig()
		// If we already have a finite incumbent, pass it to the step schedule.
		if !math.IsInf(e.bestCost, 0) && e.bestCost > 0 {
			cfg.UB = e.bestCost
		}
		if lb, _, err := OneTreeLowerBound(dist, e.start, true, cfg); err == nil {
			// If LB >= UB−eps at the root, the incumbent is optimal; return it.
			if !math.IsInf(e.bestCost, 0) && lb >= e.bestCost-e.eps {
				_ = CanonicalizeOrientationInPlace(e.bestTour)
				if verr := ValidateTour(e.bestTour, n, e.start); verr == nil {
					return TSResult{Tour: e.bestTour, Cost: round1e9(e.bestCost)}, nil
				}
			}
		}
	}

	// Run DFS.
	e.dfs(e.start, 1, 0)

	// Finalization.
	if e.useDeadline && time.Now().After(e.deadline) {
		// Time budget reached (even if an incumbent exists).
		return TSResult{}, ErrTimeLimit
	}
	if !e.foundAny && math.IsInf(e.bestCost, 0) {
		return TSResult{}, ErrIncompleteGraph
	}
	_ = CanonicalizeOrientationInPlace(e.bestTour)
	if err := ValidateTour(e.bestTour, n, e.start); err != nil {
		return TSResult{}, err
	}

	return TSResult{Tour: e.bestTour, Cost: round1e9(e.bestCost)}, nil
}
