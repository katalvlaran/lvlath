// Package tsp — Held–Karp 1-tree (Lagrangian) lower bound for symmetric TSP.
//
// This module computes an admissible lower bound on OPT via the classical
// Held–Karp relaxation:
//
//   - Choose vertex r as the "root". For a multiplier vector π ∈ ℝ^n define
//     reduced costs   c'_{ij} = c_{ij} + π_i + π_j  (symmetric case).
//   - Build a minimum 1-tree T(π): MST on V\{r} using c', plus two cheapest
//     r-incident edges (w.r.t. c').
//   - Bound value (Lagrangian dual):
//     L(π) = cost_c'(T(π)) − 2 · Σ_i π_i
//     where cost_c' sums reduced costs of 1-tree edges.
//     Since Σ_(i,j)∈T (π_i+π_j) = Σ_i deg_T(i)·π_i, this matches the dual form.
//   - Update π by subgradient with components s_i = deg_T(i) − 2
//     (tour feasibility requires deg(i)=2 for every i).
//
// L(π) is a valid *lower bound* on the optimal tour cost for every π, and
// is typically much tighter than degree-1 or MST bounds. We provide:
//   - A deterministic, allocation-conscious implementation.
//   - O(n²) Prim for the MST part (dense distance matrices).
//   - A small subgradient loop with a safe step policy and optional UB feedback.
//
// Scope & safety:
//   - Symmetric instances only (opts.Symmetric == true). For ATSP, a 1-tree
//     bound is not directly admissible; callers must gate usage upstream.
//   - +Inf is allowed (missing edges); if a 1-tree cannot be formed
//     (disconnected V\{r} or <2 finite r-incident edges), ErrIncompleteGraph.
//   - NaN/negative weights are rejected (strict sentinels).
//
// Complexity (per call):
//   - O(iters · n²) time for a fixed iteration budget.
//   - O(n²) memory for dense weights; O(n) working arrays.
//
// Determinism:
//   - No RNG. Prim and r-edge selection break ties by vertex index.
//   - The subgradient schedule is purely arithmetic and reproducible.
//
// Integration notes:
//   - To expose this as a dispatcher choice, add OneTreeBound to BoundAlgo
//     and route Branch-and-Bound to use OneTreeLowerBound when selected.
package tsp

import (
	"math"
	"time"

	"github.com/katalvlaran/lvlath/matrix"
)

// OneTreeConfig controls the subgradient loop and optional wall-clock budget.
//
// A compact, deterministic default works well as a drop-in bound.
// Increasing MaxIter tightens the bound at O(n²) cost per iteration.
type OneTreeConfig struct {
	// MaxIter is the maximum number of subgradient iterations (≥ 1).
	MaxIter int
	// Alpha ∈ (0, 2): step scale. 0.8–1.2 is common; we default to 0.9.
	Alpha float64
	// UB: optional incumbent (feasible tour) cost for adaptive steps.
	// If UB ≤ 0 or +Inf, the schedule ignores UB and uses a decreasing sequence.
	UB float64
	// TimeLimit: optional per-call wall-clock budget (0 disables checks).
	TimeLimit time.Duration
}

// DefaultOneTreeConfig returns conservative, production-grade defaults.
func DefaultOneTreeConfig() OneTreeConfig {
	return OneTreeConfig{
		MaxIter:   32,
		Alpha:     0.9,
		UB:        math.Inf(1), // no UB feedback by default
		TimeLimit: 0,
	}
}

// OneTreeLowerBound computes the Held–Karp 1-tree lower bound for a symmetric
// instance using 'root' as the distinguished vertex (usually opts.StartVertex).
//
// Returned lower bound is stabilized to 1e−9 for cross-platform consistency.
// The degree vector of the *final* 1-tree is returned for diagnostics.
//
// Errors:
//   - ErrATSPNotSupportedByAlgo for asymmetric instances.
//   - ErrIncompleteGraph if no 1-tree can be formed (disconnected V\{root}
//     or fewer than two finite root edges).
//   - Strict sentinels for NaN/negative weights or shape issues.
//
// Complexity: O(cfg.MaxIter · n²) time, O(n²) memory.
func OneTreeLowerBound(
	dist matrix.Matrix,
	root int,
	symmetric bool,
	cfg OneTreeConfig,
) (lb float64, degrees []int, err error) {
	// Shape guards (dispatcher has validated already; retain cheap checks).
	n := dist.Rows()
	if n != dist.Cols() || n < 2 {
		return 0, nil, ErrNonSquare
	}
	if err = validateStartVertex(n, root); err != nil {
		return 0, nil, err
	}
	if !symmetric {
		return 0, nil, ErrATSPNotSupportedByAlgo
	}
	if cfg.MaxIter <= 0 {
		cfg.MaxIter = 1
	}
	if cfg.Alpha <= 0 || cfg.Alpha >= 2 {
		cfg.Alpha = 0.9
	}

	// Dense prefetch with strict sentinels; +Inf allowed (represents missing edges).
	w := make([]float64, n*n)
	var (
		i, j int
		x    float64
	)
	for i = 0; i < n; i++ { // scan rows of the distance matrix (origin vertices u=i)
		for j = 0; j < n; j++ { // scan columns of the distance matrix (destination vertices v=j)
			x, err = dist.At(i, j)
			if err != nil || math.IsNaN(x) {
				return 0, nil, ErrDimensionMismatch
			}
			if x < 0 {
				return 0, nil, ErrNegativeWeight
			}
			w[i*n+j] = x // write c_{ij} into the dense linear buffer at offset i*n + j
		}
	}

	eng := oneTreeEngine{
		n:      n,
		root:   root,
		w:      w,
		pi:     make([]float64, n),
		deg:    make([]int, n),
		inTree: make([]bool, n),
		parent: make([]int, n),
		key:    make([]float64, n),
	}

	// Sparse deadline checks (every 2048 iterations of the inner MST/scan loops).
	var useDeadline bool   // whether to enforce a per-call wall-clock budget
	var deadline time.Time // absolute time when the budget expires
	var tick uint64
	if cfg.TimeLimit > 0 && compatibleTimeBudget(cfg.TimeLimit) {
		useDeadline = true
		deadline = time.Now().Add(cfg.TimeLimit)
	}
	checkDeadline := func() bool {
		// keep overhead tiny; caller uses a small, fixed number of iterations
		tick++
		if !useDeadline || (tick&2047) != 0 { // only probe wall clock every 2048th tick to amortize
			return false
		}
		return time.Now().After(deadline)
	}

	// Subgradient loop.
	var (
		bestLB    = math.Inf(-1) // best L(π) observed so far
		sumPi     float64        // running Σ π_i
		iter      int            // iteration counter
		norm2     float64        // ||s||² where s_i = deg(i)−2
		redCost   float64        // reduced-cost sum of the current 1-tree, cost_c'(T(π))
		degDiff   int            // s_i = deg(i) − 2 for the current i
		haveUB    bool           // whether a finite positive UB was supplied
		usedUB    float64        // the UB used by the step-size formula
		step      float64        // current step size t
		lastBound float64        // current L(π) before taking the step
	)
	if !math.IsInf(cfg.UB, 0) && cfg.UB > 0 {
		haveUB = true
		usedUB = cfg.UB // capture the incumbent UB to drive the adaptive step t = α·(UB−L)/||s||²
	}

	for iter = 0; iter < cfg.MaxIter; iter++ {
		if checkDeadline() {
			return 0, nil, ErrTimeLimit
		}

		// Build a minimum 1-tree on reduced costs; fills eng.deg and returns cost_c'(T(π)).
		redCost, err = eng.buildOneTreeReduced()
		if err != nil {
			// Includes ErrIncompleteGraph if MST(V\{root}) is disconnected or <2 finite root edges exist.
			return 0, nil, err
		}

		// L(π) = cost_c'(T) − 2·Σπ (classical Held–Karp dual objective).
		sumPi = 0
		for i = 0; i < n; i++ {
			sumPi += eng.pi[i] // accumulate Σ π_i over all vertices
		}
		lastBound = redCost - 2*sumPi // compute the current dual value L(π)
		if lastBound > bestLB {
			bestLB = lastBound
		}

		// Check if T(π) is already a tour: deg(i)==2 for all i ⇒ subgradient s=0 ⇒ ||s||²=0.
		norm2 = 0
		for i = 0; i < n; i++ { // compute the squared 2-norm of the subgradient s
			degDiff = eng.deg[i] - 2 // s_i = deg_T(i) − 2 encodes degree violation at i
			norm2 += float64(degDiff * degDiff)
		}
		if norm2 == 0 {
			// The 1-tree is a Hamiltonian cycle; L(π) equals that tour's reduced-cost sum minus 2Σπ.
			break
		}

		// Step size policy:
		//  - With UB:  t = α · (UB − L) / ||s||²  (standard Held–Karp recipe),
		//    clamped to a non-negative value to avoid moving in the wrong direction.
		//  - Without UB: monotone decreasing schedule t = α / (1 + iter).
		if haveUB {
			step = usedUB - lastBound
			if step < 0 {
				step = 0 // avoid negative step if the current bound exceeds or matches UB
			}
			step = cfg.Alpha * step / norm2 // scale by α and normalize by the subgradient norm²
		} else {
			step = cfg.Alpha / (1.0 + float64(iter)) // diminishing steps ensure stability without UB feedback
		}
		if step == 0 {
			// No useful progress possible under the chosen schedule.
			break
		}

		// π_i ← π_i + t · (deg(i) − 2) for all i (plain subgradient ascent on the dual).
		for i = 0; i < n; i++ {
			eng.pi[i] += step * float64(eng.deg[i]-2)
		}
	}

	// Return the best lower bound seen (stabilized) and the final degrees (diagnostics).
	outDeg := make([]int, n)
	copy(outDeg, eng.deg)

	return round1e9(bestLB), outDeg, nil
}

// oneTreeEngine holds mutable state for building 1-trees on reduced costs.
// Arrays are reused across iterations to avoid per-iteration allocations.
type oneTreeEngine struct {
	n    int       // number of vertices
	root int       // distinguished root vertex r
	w    []float64 // dense weights, length n*n (original costs c_{ij})

	// Lagrange multipliers (π_i).
	pi []float64

	// Working state for building a 1-tree.
	deg    []int  // degree in the current 1-tree
	inTree []bool // whether a vertex is already included in the MST over V\{root}
	parent []int  // Prim parent for MST over V\{root}
	key    []float64
}

// reduced returns c'_{uv} = c_{uv} + π_u + π_v (symmetric case).
func (e *oneTreeEngine) reduced(u, v int) float64 {
	return e.w[u*e.n+v] + e.pi[u] + e.pi[v]
}

// zeroDegrees clears the degree vector (O(n)).
func (e *oneTreeEngine) zeroDegrees() {
	var i int
	for i = 0; i < e.n; i++ {
		e.deg[i] = 0 // reset degree counters before constructing the next 1-tree
	}
}

// buildOneTreeReduced builds a minimum 1-tree on reduced costs:
//   - MST over V\{root} via Prim in O(n²);
//   - two cheapest root edges (w.r.t. reduced costs) then added.
//
// It fills e.deg and returns the *reduced-cost* total.
//
// If the 1-tree cannot be formed (disconnected V\{root} or <2 finite root edges),
// ErrIncompleteGraph is returned.
func (e *oneTreeEngine) buildOneTreeReduced() (float64, error) {
	var inf = math.Inf(1)
	e.zeroDegrees()

	// ---- Prim over V \ {root} using reduced costs.
	var (
		v, u, best, iter int     // vertex indices / loop counters
		c                float64 // working reduced cost
		costReduced      float64 // total reduced cost of the 1-tree edges
	)

	// Initialize keys/parents/inTree for the (n−1)-vertex MST.
	for v = 0; v < e.n; v++ {
		e.inTree[v] = false
		e.parent[v] = -1
		e.key[v] = inf
	}
	// Pick a deterministic start in V\{root}.
	start := 0
	if start == e.root { // ensure the start vertex is not the root (root is excluded from the MST)
		start = 1 // pick the next available vertex
	}
	e.key[start] = 0 // seed Prim: first extraction will choose 'start'

	// Extract the minimum-key vertex (n−1 times), add it, and relax neighbors.
	for iter = 0; iter < e.n-1; iter++ {
		// Find best = argmin_{v ∉ tree, v≠root} key[v] with index tiebreak.
		best = -1
		for v = 0; v < e.n; v++ { // scan all vertices to pick the next MST vertex
			if v == e.root || e.inTree[v] {
				continue // skip the root and vertices already included into the MST
			}
			if best == -1 || e.key[v] < e.key[best] || (e.key[v] == e.key[best] && v < best) {
				best = v // maintain the current minimum by key, breaking ties by vertex index
			}
		}
		if best == -1 || math.IsInf(e.key[best], 0) {
			// Disconnected V\{root}: no MST ⇒ no 1-tree can be formed.
			return 0, ErrIncompleteGraph
		}

		// Include 'best'. If it has a parent, account for that edge.
		e.inTree[best] = true     // mark 'best' as part of the MST
		if e.parent[best] != -1 { // if not the starting vertex (which has no parent yet)
			u = e.parent[best]     // retrieve the MST parent of 'best'
			c = e.reduced(best, u) // reduced cost of the selected MST edge (best,u)
			costReduced += c       // accumulate the reduced-cost total
			e.deg[best]++          // update endpoint degrees in the emerging 1-tree
			e.deg[u]++
		}

		// Relax edges from 'best' to remaining vertices in V\{root}.
		for v = 0; v < e.n; v++ { // try improving keys for all not-yet-in-tree vertices
			if v == e.root || e.inTree[v] || v == best {
				continue // skip root, already-in-tree vertices, and self
			}
			c = e.reduced(best, v)
			if c < e.key[v] { // if joining via 'best' yields a smaller reduced cost
				e.key[v] = c       // update the tentative key (edge weight)
				e.parent[v] = best // and remember 'best' as the new parent
			}
		}
	}

	// ---- Add the two cheapest root edges by reduced cost.
	var (
		m1To, m2To int     // endpoints of the best and second-best root edges
		m1, m2     float64 // their reduced costs
	)
	m1, m2 = inf, inf
	m1To, m2To = -1, -1

	for v = 0; v < e.n; v++ { // scan all non-root vertices to find the two cheapest root edges
		if v == e.root {
			continue // root-to-root is invalid; we need r→v with v≠r
		}
		c = e.reduced(e.root, v)
		if c < m1 || (c == m1 && v < m1To) {
			// Promote current minimum to second minimum, then update the minimum.
			m2, m2To = m1, m1To
			m1, m1To = c, v
		} else if c < m2 || (c == m2 && v < m2To) {
			m2, m2To = c, v // update the second-best root edge
		}
	}
	if math.IsInf(m1, 0) || math.IsInf(m2, 0) {
		// Fewer than two finite root edges ⇒ no 1-tree exists.
		return 0, ErrIncompleteGraph
	}

	costReduced += m1 + m2 // include the two root edges into the reduced-cost total
	e.deg[e.root] += 2     // root has degree 2 in any 1-tree
	e.deg[m1To]++          // increment degrees at the endpoints of the two root edges
	e.deg[m2To]++

	return costReduced, nil
}
