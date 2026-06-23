// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package tsp defines dense weighted Blossom engine state.
// The engine solves maximum-profit perfect matching after transforming local
// minimum-cost MWPM costs into non-negative profits.
package tsp

import "math"

// blossomOptions controls numeric tolerance for the dense Blossom engine.
//
// Implementation:
//   - Stage 1: validateBlossomProblem checks Eps before allocation.
//   - Stage 2: The search uses Eps for slack/tight-edge comparisons.
//   - Stage 3: Verification uses Eps to tolerate harmless floating-point drift.
//
// Behavior highlights:
//   - Eps must be finite and strictly positive.
//   - Options are private to the matching engine.
//
// Inputs:
//   - Built from Options.Eps by blossomMatch.
//
// Returns:
//   - Consumed by solveMinimumWeightPerfectMatching.
//
// Errors:
//   - ErrInvalidOptions when Eps is invalid.
//
// Determinism:
//   - Eps does not affect scan order, only numeric tightness classification.
//
// Complexity:
//   - Storage O(1).
//
// AI-Hints:
//   - Do not silently replace invalid Eps with DefaultEps.
type blossomOptions struct {
	// Eps is the positive finite numeric tolerance used by slack, tight-edge,
	// dual-feasibility, and equality checks inside the Blossom engine.
	Eps float64
}

// blossomStats records deterministic internal telemetry produced by the dense Blossom engine.
// It is intentionally private: public TSP results must not expose Blossom implementation counters
// until those counters become stable API semantics.
//
// Implementation:
//   - Stage 1: Search stages increment counters at the exact mutation point.
//   - Stage 2: solveMinimumWeightPerfectMatching returns the collected value by copy.
//   - Stage 3: Tests and benchmarks use the counters to verify that specific regimes
//     exercise augmentation, contraction, expansion, dual movement, and tight scans.
//
// Behavior highlights:
//   - Counters are monotonic within one engine run.
//   - The struct is copied on return; callers cannot mutate engine state.
//   - Zero values are valid for empty matching instances and trivial direct augmentations.
//
// Inputs:
//   - Populated by blossomEngine methods during search.
//
// Returns:
//   - Returned by solveMinimumWeightPerfectMatching as private package telemetry.
//
// Errors:
//   - None. Counter updates do not fail and must not influence solver decisions.
//
// Determinism:
//   - For fixed problem, options, and implementation, counts are deterministic because
//     edge order, queue order, root insertion, and tie-breaks are deterministic.
//
// Complexity:
//   - Storage O(1).
//   - Update cost O(1) per recorded event.
//
// Notes:
//   - These values are diagnostic, not proof metadata.
//   - Public ApproximationRatio and Exact/Optimal flags must not depend on these counters.
//
// AI-Hints:
//   - Do not expose blossomStats through TSPResult without a public compatibility decision.
//   - Do not use Shrinks==0 as proof that Blossom was unnecessary; some valid instances need no contraction.
type blossomStats struct {
	// Augmentations counts successful augmenting-path applications.
	// One augmentation increases the number of matched original vertices by exactly two.
	// The final value must be problem.n/2 for every non-empty successful perfect matching.
	Augmentations int

	// Shrinks counts odd-cycle contractions into active blossom nodes.
	// It is a regression anchor for general-graph cases where bipartite-style alternating paths are insufficient.
	// A zero value is valid on easy instances that never encounter a same-tree outer/outer tight edge.
	Shrinks int

	// Expansions counts active non-singleton blossom expansions.
	// Weighted search expands only when dual conditions make expansion necessary for continued correctness.
	// This counter must not be artificially incremented for no-op expansion checks.
	Expansions int

	// DualUpdates counts selected dual-delta applications.
	// Every increment means the search exhausted currently tight events and moved the dual system to expose one.
	// Large values are useful benchmark telemetry for hard weighted instances.
	DualUpdates int

	// TightScans counts calls to the tight-edge predicate.
	// The value measures dense scanning work and is deterministic for a fixed scan order.
	// It is not a public complexity promise.
	TightScans int
}

// blossomEdge stores one undirected edge of the dense local matching graph.
// The edge keeps both the original minimization cost and the transformed maximization
// profit so the search can optimize profits while final verification recomputes original cost.
//
// Implementation:
//   - Stage 1: buildBlossomEdges scans local vertices with u<v.
//   - Stage 2: It assigns id=len(edges) before append, making id equal to slice index.
//   - Stage 3: It stores cost from matchingProblem and profit=maxCost-cost.
//   - Stage 4: incident lists store edge IDs for deterministic outer-node scans.
//
// Behavior highlights:
//   - Edges are immutable after construction.
//   - Edge IDs are dense and stable.
//   - u<v is guaranteed by construction.
//   - cost and profit are both finite after validation.
//
// Inputs:
//   - Built from matchingProblem local cost matrix.
//
// Returns:
//   - Consumed by slack, augmenting paths, mateEdge, and verification helpers.
//
// Errors:
//   - Construction rejects NaN, ±Inf, and negative costs before edges are stored.
//
// Determinism:
//   - Edge order is lexicographic by local endpoints: u ascending, then v ascending.
//   - Equal-cost tie-breaking can safely use smaller edge ID.
//
// Complexity:
//   - Storage O(1) per edge.
//   - Whole dense edge set uses O(k^2) space.
//
// Notes:
//   - profit is an internal optimization objective, not the published matching cost.
//   - Every perfect matching has k/2 edges, so maxCost-cost preserves argmin over costs.
//
// AI-Hints:
//   - Do not compute final matching cost from profit.
//   - Do not reorder edges after construction; event tie-breaks rely on stable IDs.
type blossomEdge struct {
	// id is the dense edge identifier and equals the index in blossomEngine.edges.
	// It is used by incident lists, labelEdge, mateEdge, and augmenting-path edge sequences.
	// The ID must remain stable for the lifetime of the engine.
	id int

	// u is the smaller local endpoint in [0, problem.n).
	// Dense construction guarantees u<v and never creates self-loops.
	// It is an index into matchingProblem, not the original TSP matrix.
	u int

	// v is the larger local endpoint in [0, problem.n).
	// Dense construction guarantees v>u and stores the undirected edge once.
	// It is an index into matchingProblem, not the original TSP matrix.
	v int

	// cost is the original local minimization weight.
	// matchingCost uses this model to recompute final MWPM cost after export.
	// This value must not be mutated by dual updates.
	cost float64

	// profit is maxCost-cost for maximum-weight Blossom search.
	// It is non-negative because maxCost is computed over all finite local edges.
	// This value exists only to drive the internal weighted matching objective.
	profit float64
}

// blossomLabel classifies active top-level nodes in the alternating forest.
// Unlabeled nodes are not yet in the forest; outer nodes are S-nodes;
// inner nodes are T-nodes reached through a matched edge.
type blossomLabel uint8

const (
	// blossomUnlabeled marks an active node outside the current alternating forest.
	blossomUnlabeled blossomLabel = iota

	// blossomOuter marks an S-node that scans tight outgoing edges.
	blossomOuter

	// blossomInner marks a T-node reached from an outer node through a tight edge.
	blossomInner
)

// blossomCycleStep describes one directed step in a contracted odd blossom cycle.
// Each step owns one child top-level node and the dense edge that connects this
// child to the next child in the deterministic cycle order.
//
// Implementation:
//   - Stage 1: allocateBlossomNode builds child nodes in cycle order.
//   - Stage 2: buildBlossomCycleSteps resolves the edge between every adjacent child pair.
//   - Stage 3: orientEdgeForNodes stores original endpoint ownership for each boundary edge.
//   - Stage 4: liftThroughBlossom walks these steps to reconstruct original augmenting paths.
//
// Behavior highlights:
//   - node is the current child in the cycle.
//   - edgeToNext connects node to the next cycle child.
//   - vertexToNext is the original local vertex inside node used by edgeToNext.
//   - nextVertex is the original local vertex inside the next child used by edgeToNext.
//
// Inputs:
//   - Built only during blossom contraction.
//
// Returns:
//   - Consumed by path lifting, expansion, and contraction invariant tests.
//
// Errors:
//   - Construction helpers reject invalid child nodes, invalid dense edges, and endpoint mismatches.
//
// Determinism:
//   - Cycle order is inherited from parent paths and the closing shrink edge.
//   - Equal structural cases keep the parent-chain order chosen by shrink.
//
// Complexity:
//   - Storage O(1) per cycle child.
//   - Whole contracted cycle storage is O(c), where c is the blossom cycle length.
//
// Notes:
//   - This metadata is the minimum information required to lift paths through contracted blossoms.
//   - childEdges with a single closing edge is not enough for correct augmentation.
//
// AI-Hints:
//   - Do not store only the closing edge.
//   - Do not infer endpoint ownership later from edge IDs alone; store it at contraction time.
type blossomCycleStep struct {
	// node is the active top-level child represented by this cycle step.
	// It may be an original singleton vertex or a previously contracted blossom.
	node int

	// edgeToNext is the dense edge connecting node to the next cycle child.
	// For the last step, it connects back to the first cycle child.
	edgeToNext int

	// vertexToNext is the original local vertex inside node used by edgeToNext.
	// It lets path lifting enter or leave nested blossoms without guessing endpoints.
	vertexToNext int

	// nextVertex is the original local vertex inside the next cycle child used by edgeToNext.
	// It is paired with vertexToNext by the same dense edge.
	nextVertex int
}

const (
	// noVertex marks the absence of an original local vertex.
	noVertex = -1

	// noEdge marks the absence of a dense blossomEdge identifier.
	noEdge = -1

	// noNode marks the absence of an active or allocated blossom node.
	noNode = -1
)

// blossomEngine owns all mutable state for dense weighted Blossom search.
//
// Implementation:
//   - Stage 1: newBlossomEngine builds dense edge storage and initializes duals.
//   - Stage 2: solve repeatedly finds and applies augmenting structures.
//   - Stage 3: exportMatching converts mate[] into a verified local match array.
//
// Behavior highlights:
//   - All vertex IDs are local matchingProblem indices.
//   - Edges are dense and deterministic: u asc, v asc.
//   - No map iteration is used for matching order.
//   - Original vertices and contracted blossoms share node IDs.
//
// Inputs:
//   - matchingProblem and blossomOptions.
//
// Returns:
//   - Used internally by solveMinimumWeightPerfectMatching.
//
// Errors:
//   - Construction and verification errors are returned by helpers.
//
// Determinism:
//   - Dense edge order is fixed.
//   - Queue order is FIFO over deterministic scan order.
//
// Complexity:
//   - Edge storage O(k^2).
//   - Engine node storage O(k).
//
// AI-Hints:
//   - Do not expose this engine outside package tsp.
//   - Do not store original TSP matrix indices in mate[]; use local indices.
type blossomEngine struct {
	// problem is the detached local MWPM instance over odd-degree vertices.
	// All engine vertex indices are local positions in this problem.
	problem matchingProblem

	// eps is the positive finite tolerance used for slack and dual comparisons.
	// It is copied from blossomOptions and never mutated.
	eps float64

	// edges stores every dense undirected edge in deterministic u<v order.
	// edge.id is equal to its index in this slice.
	edges []blossomEdge

	// incident maps each original local vertex to dense edge IDs touching it.
	// It lets outer-node scans enumerate candidate edges without map iteration.
	incident [][]int

	// mate stores the matched original local vertex for each original vertex.
	// mate[v]==noVertex means v is currently unmatched.
	mate []int

	// mateEdge stores the dense edge ID that realizes mate[v].
	// mateEdge[v]==noEdge when v is unmatched.
	mateEdge []int

	// inBlossom maps every original local vertex to its current active top-level node.
	// It is updated during contraction and expansion.
	inBlossom []int

	// base stores the base original vertex for every active node or contracted blossom.
	// For singleton vertices base[v]==v.
	base []int

	// active reports whether a node ID is currently a top-level search node.
	// Contracted children are inactive until expansion restores them.
	active []bool

	// parent stores alternating-forest parent node IDs.
	// noNode marks a root or inactive/unlabeled node.
	parent []int

	// children stores child top-level node IDs for contracted blossom nodes.
	// The order matches cycles[node][i].node and is kept for simple membership scans.
	children [][]int

	// cycles stores ordered cycle metadata for every contracted blossom node.
	// cycles[node][i] describes children[node][i] and the edge from it to the next child.
	cycles [][]blossomCycleStep

	// members stores original local vertices contained in each node.
	// It is the source of truth for updating inBlossom during shrink/expand.
	members [][]int

	// nextNode is the next available contracted-blossom node ID.
	// It starts at problem.n and must stay below len(active).
	nextNode int

	// label stores the alternating-forest label for each active node.
	// Values are blossomUnlabeled, blossomOuter, or blossomInner.
	label []blossomLabel

	// labelEdge stores the edge through which a node received its current label.
	// noEdge marks roots and unlabeled nodes.
	labelEdge []int

	// treeRoot stores the outer root node for each labeled node.
	// It distinguishes same-tree shrink events from cross-tree augment events.
	treeRoot []int

	// queue stores outer nodes awaiting tight-edge scans.
	// The engine uses head as a cursor and never shifts this slice.
	queue []int

	// head is the FIFO cursor into queue.
	// It avoids retention-prone queue=queue[1:] operations.
	head int

	// dual stores active node dual variables used by the weighted Blossom search.
	// Original vertices and contracted blossoms share this dense node-indexed array.
	dual []float64

	// stats records private deterministic telemetry for tests and benchmarks.
	// It is not part of the public TSPResult contract.
	stats blossomStats
}

// newBlossomEngine constructs the dense weighted Blossom engine.
//
// Implementation:
//   - Stage 1: Validate local problem and Blossom options.
//   - Stage 2: Build dense edge/incident storage with max-profit transformation.
//   - Stage 3: Allocate matching, blossom, forest, and dual state.
//   - Stage 4: Initialize original vertices as active top-level singleton blossoms.
//
// Behavior highlights:
//   - Allocates once for the dense correctness-first engine.
//   - Uses node capacity 2*k-1 for original vertices plus contracted blossoms.
//   - Does not start the search.
//
// Inputs:
//   - problem: local MWPM instance.
//   - opts: numeric Blossom policy.
//
// Returns:
//   - *blossomEngine: initialized engine.
//   - error: nil on success or sentinel-classified failure.
//
// Errors:
//   - ErrInvalidOptions, ErrInvalidMatching, ErrNaNInf, ErrNegativeWeight, ErrAsymmetry.
//
// Determinism:
//   - Allocation and edge order are deterministic.
//
// Complexity:
//   - Time O(k^2), Space O(k^2).
//
// AI-Hints:
//   - Do not allocate blossoms lazily with maps; dense node IDs keep contraction deterministic.
func newBlossomEngine(problem matchingProblem, opts blossomOptions) (*blossomEngine, error) {
	if err := validateBlossomProblem(problem, opts); err != nil {
		return nil, err
	}

	edges, incident, maxCost, err := buildBlossomEdges(problem)
	if err != nil {
		return nil, err
	}

	nodeCapacity := 1
	if problem.n > 0 {
		nodeCapacity = 2*problem.n - 1
	}

	engine := &blossomEngine{
		problem:   problem,
		eps:       opts.Eps,
		edges:     edges,
		incident:  incident,
		mate:      makeFilledInt(problem.n, noVertex),
		mateEdge:  makeFilledInt(problem.n, noEdge),
		inBlossom: makeFilledIdentity(problem.n, nodeCapacity),
		base:      makeFilledInt(nodeCapacity, noVertex),
		active:    make([]bool, nodeCapacity),
		parent:    makeFilledInt(nodeCapacity, noNode),
		children:  make([][]int, nodeCapacity),
		cycles:    make([][]blossomCycleStep, nodeCapacity),
		label:     make([]blossomLabel, nodeCapacity),
		labelEdge: makeFilledInt(nodeCapacity, noEdge),
		treeRoot:  makeFilledInt(nodeCapacity, noNode),
		queue:     make([]int, 0, problem.n),
		dual:      make([]float64, nodeCapacity),
		members:   make([][]int, nodeCapacity),
		nextNode:  problem.n,
	}

	for vertex := 0; vertex < problem.n; vertex++ {
		engine.base[vertex] = vertex
		engine.active[vertex] = true
		engine.dual[vertex] = maxCost
		engine.members[vertex] = []int{vertex}
	}

	return engine, nil
}

// matchedPairs counts committed original-vertex matching pairs.
//
// Complexity:
//   - Time O(k), Space O(1).
func (e *blossomEngine) matchedPairs() int {
	pairs := 0

	for vertex, mate := range e.mate {
		if vertex < mate {
			pairs++
		}
	}

	return pairs
}

// exportMatching returns a verified local symmetric perfect matching array.
//
// Implementation:
//   - Stage 1: Copy mate[] into a detached match slice.
//   - Stage 2: Reject unmatched vertices.
//   - Stage 3: Verify perfect matching symmetry.
//
// Behavior highlights:
//   - Does not expose engine storage.
//   - Does not compute costs.
//
// Inputs:
//   - None; reads engine mate[].
//
// Returns:
//   - []int: detached local match array.
//   - error: nil when every vertex is matched.
//
// Errors:
//   - ErrIncompleteGraph when any vertex remains unmatched.
//   - ErrInvalidMatching when mate[] is corrupt.
//
// Determinism:
//   - Fixed increasing vertex scan.
//
// Complexity:
//   - Time O(k), Space O(k).
//
// AI-Hints:
//   - Do not export contracted blossom node IDs; matching is over original local vertices only.
func (e *blossomEngine) exportMatching() ([]int, error) {
	match := make([]int, e.problem.n)

	for vertex := 0; vertex < e.problem.n; vertex++ {
		if e.mate[vertex] == noVertex {
			return nil, ErrIncompleteGraph
		}

		match[vertex] = e.mate[vertex]
	}

	if err := verifyPerfectMatching(match); err != nil {
		return nil, err
	}

	return match, nil
}

// verifyMatchingSymmetry checks mate[] and mateEdge[] consistency.
//
// Complexity:
//   - Time O(k), Space O(1).
func (e *blossomEngine) verifyMatchingSymmetry() error {
	for vertex := 0; vertex < e.problem.n; vertex++ {
		mate := e.mate[vertex]
		if mate == noVertex {
			continue
		}
		if mate < 0 || mate >= e.problem.n {
			return ErrInvalidMatching
		}
		if mate == vertex {
			return ErrInvalidMatching
		}
		if e.mate[mate] != vertex {
			return ErrInvalidMatching
		}
		if e.mateEdge[vertex] == noEdge || e.mateEdge[mate] != e.mateEdge[vertex] {
			return ErrInvalidMatching
		}
	}

	return nil
}

// verifyTopLevelPartition checks that every original vertex belongs to one active top-level node.
// It protects shrink/expand code from orphaning vertices.
//
// Complexity:
//   - Time O(k), Space O(1).
func (e *blossomEngine) verifyTopLevelPartition() error {
	for vertex := 0; vertex < e.problem.n; vertex++ {
		node := e.inBlossom[vertex]
		if node < 0 || node >= len(e.active) || !e.active[node] {
			return ErrInvalidMatching
		}
	}

	return nil
}

// verifyDualFeasibility checks non-negative active dual variables within tolerance.
// Full edge-slack verification belongs in the dual subsystem because it depends on
// top-level blossom membership and transformed profit.
//
// Complexity:
//   - Time O(k), Space O(1).
func (e *blossomEngine) verifyDualFeasibility() error {
	for node := 0; node < len(e.active); node++ {
		if !e.active[node] {
			continue
		}
		if e.dual[node] < -e.eps {
			return ErrInvalidMatching
		}
	}

	return nil
}

// verifyOptimalState checks final structural invariants before export.
//
// Complexity:
//   - Time O(k), Space O(1).
func (e *blossomEngine) verifyOptimalState() error {
	if err := e.verifyMatchingSymmetry(); err != nil {
		return err
	}
	if err := e.verifyTopLevelPartition(); err != nil {
		return err
	}
	if err := e.verifyDualFeasibility(); err != nil {
		return err
	}

	return nil
}

// validateBlossomProblem checks local MWPM shape and numeric policy before allocation.
//
// Implementation:
//   - Stage 1: Validate Eps.
//   - Stage 2: Validate even local order and flat cost matrix shape.
//   - Stage 3: Validate finite non-negative off-diagonal costs.
//   - Stage 4: Validate local cost symmetry defensively.
//
// Behavior highlights:
//   - No mutation.
//   - Rejects invalid local models before engine allocation.
//
// Inputs:
//   - problem: local matching problem.
//   - opts: Blossom numeric policy.
//
// Returns:
//   - error: nil when the engine can be constructed.
//
// Errors:
//   - ErrInvalidOptions for invalid Eps.
//   - ErrInvalidMatching for malformed local dimensions.
//   - ErrNaNInf for NaN or ±Inf local costs.
//   - ErrNegativeWeight for negative finite local costs.
//   - ErrAsymmetry for non-symmetric local costs.
//
// Determinism:
//   - Fixed row-major validation order.
//
// Complexity:
//   - Time O(k^2), Space O(1).
//
// AI-Hints:
//   - Do not trust matchingProblem blindly; it is private but still validates external matrix data.
func validateBlossomProblem(problem matchingProblem, opts blossomOptions) error {
	if opts.Eps <= 0 || math.IsNaN(opts.Eps) || math.IsInf(opts.Eps, 0) {
		return ErrInvalidOptions
	}
	if problem.n < 0 || (problem.n&1) == 1 {
		return ErrInvalidMatching
	}
	if len(problem.odd) != problem.n || len(problem.w) != problem.n*problem.n {
		return ErrInvalidMatching
	}

	for row := 0; row < problem.n; row++ {
		for col := 0; col < problem.n; col++ {
			value := problem.at(row, col)
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return ErrNaNInf
			}
			if row != col && value < 0 {
				return ErrNegativeWeight
			}
			if row < col && math.Abs(value-problem.at(col, row)) > opts.Eps {
				return ErrAsymmetry
			}
		}
	}

	return nil
}

// makeFilledInt returns an int slice initialized with value.
//
// Complexity:
//   - Time O(n), Space O(n).
func makeFilledInt(n int, value int) []int {
	out := make([]int, n)
	for index := range out {
		out[index] = value
	}

	return out
}

// makeFilledIdentity returns a length cap slice with indices [0,n) initialized to themselves.
// Remaining entries are initialized to noNode so unallocated blossom nodes are explicit.
//
// Complexity:
//   - Time O(capacity), Space O(capacity).
func makeFilledIdentity(n int, capacity int) []int {
	out := makeFilledInt(capacity, noNode)
	for index := 0; index < n; index++ {
		out[index] = index
	}

	return out
}

// buildBlossomEdges builds dense local edges and max-profit weights.
// The transformation profit=maxCost-cost preserves the optimal perfect matching
// because every perfect matching has exactly k/2 edges.
//
// Implementation:
//   - Stage 1: Scan all upper-triangle costs and find maxCost.
//   - Stage 2: Allocate dense edge and incident storage.
//   - Stage 3: Emit edges in deterministic u asc, v asc order.
//   - Stage 4: Store profit=maxCost-cost for max-weight matching.
//
// Behavior highlights:
//   - Does not mutate problem.
//   - No map iteration.
//   - Produces non-negative profits.
//
// Inputs:
//   - problem: validated local matching problem.
//
// Returns:
//   - []blossomEdge: dense local edge list.
//   - [][]int: incident edge IDs by local vertex.
//   - float64: maximum original cost used by the profit transform.
//   - error: nil on success.
//
// Errors:
//   - ErrNaNInf for NaN or ±Inf cost.
//   - ErrNegativeWeight for negative finite cost.
//
// Determinism:
//   - Fixed upper-triangle scan and append order.
//
// Complexity:
//   - Time O(k^2), Space O(k^2).
//
// AI-Hints:
//   - Do not transform with reciprocal weights; equal matching cardinality makes maxCost-cost safe.
func buildBlossomEdges(problem matchingProblem) ([]blossomEdge, [][]int, float64, error) {
	if problem.n == 0 {
		return nil, make([][]int, 0), 0, nil
	}

	maxCost := 0.0

	for u := 0; u < problem.n; u++ {
		for v := u + 1; v < problem.n; v++ {
			cost := problem.at(u, v)
			if math.IsNaN(cost) || math.IsInf(cost, 0) {
				return nil, nil, 0, ErrNaNInf
			}
			if cost < 0 {
				return nil, nil, 0, ErrNegativeWeight
			}
			if cost > maxCost {
				maxCost = cost
			}
		}
	}

	edges := make([]blossomEdge, 0, problem.n*(problem.n-1)/2)
	incident := make([][]int, problem.n)

	for u := 0; u < problem.n; u++ {
		for v := u + 1; v < problem.n; v++ {
			cost := problem.at(u, v)
			edgeID := len(edges)

			edge := blossomEdge{
				id:     edgeID,
				u:      u,
				v:      v,
				cost:   cost,
				profit: maxCost - cost,
			}

			edges = append(edges, edge)
			incident[u] = append(incident[u], edgeID)
			incident[v] = append(incident[v], edgeID)
		}
	}

	return edges, incident, maxCost, nil
}

// solveMinimumWeightPerfectMatching solves the local MWPM instance exactly.
// It is the production engine behind MatchingAlgo==BlossomMatch.
//
// Implementation:
//   - Stage 1: Handle empty matching as a valid zero-cost case.
//   - Stage 2: Validate even cardinality before engine construction.
//   - Stage 3: Build the dense Blossom engine.
//   - Stage 4: Run weighted Blossom search until every vertex is matched.
//   - Stage 5: Export and verify the perfect matching and its original cost.
//
// Behavior highlights:
//   - Exact.
//   - Does not call greedy matching.
//   - Does not return size-based matching unavailability.
//   - Uses matchingCost on original costs, not transformed profits.
//
// Inputs:
//   - problem: local complete MWPM instance.
//   - opts: Blossom numeric policy.
//
// Returns:
//   - []int: local match array match[i]=j.
//   - float64: rounded original matching cost.
//   - blossomStats: internal deterministic search telemetry.
//   - error: nil on success or sentinel-classified failure.
//
// Errors:
//   - ErrInvalidOptions.
//   - ErrInvalidMatching.
//   - ErrIncompleteGraph.
//   - ErrNaNInf, ErrNegativeWeight, ErrAsymmetry.
//
// Determinism:
//   - Dense edge order, queue order, contraction order, and export order are deterministic.
//
// Complexity:
//   - Dense correctness-first target is polynomial in k and uses O(k^2) edge storage.
//
// Notes:
//   - exactSmallPerfectMatching remains only as a small oracle/micro-fast-path; this function
//     must not use it as the general large-k implementation.
//
// AI-Hints:
//   - Do not route large instances to a bounded oracle; this engine owns general MWPM semantics.
//   - Do not compute public proof metadata from transformed profit.
func solveMinimumWeightPerfectMatching(
	problem matchingProblem,
	opts blossomOptions,
) ([]int, float64, blossomStats, error) {
	if problem.n == 0 {
		return []int{}, 0, blossomStats{}, nil
	}
	if (problem.n & 1) == 1 {
		return nil, 0, blossomStats{}, ErrInvalidMatching
	}

	engine, err := newBlossomEngine(problem, opts)
	if err != nil {
		return nil, 0, blossomStats{}, err
	}

	if err = engine.solve(); err != nil {
		return nil, 0, engine.stats, err
	}

	match, err := engine.exportMatching()
	if err != nil {
		return nil, 0, engine.stats, err
	}

	cost, err := matchingCost(problem, match)
	if err != nil {
		return nil, 0, engine.stats, err
	}

	return match, cost, engine.stats, nil
}
