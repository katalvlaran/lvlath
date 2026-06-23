// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package tsp implements dense Blossom alternating-forest search.
// This file owns tight-edge scanning, dual updates, augmenting-path discovery,
// and matching-edge flips over local matchingProblem vertices.
package tsp

import "math"

// blossomEventKind names the next structural action found by tight-edge scanning.
// Events are applied immediately by applyBlossomEvent.
type blossomEventKind uint8

const (
	// eventNone means no applicable tight-edge event was found.
	eventNone blossomEventKind = iota

	// eventGrow grows an alternating tree from an outer node to an unlabeled node.
	eventGrow

	// eventShrink contracts an odd cycle found inside one alternating tree.
	eventShrink

	// eventAugment applies an augmenting path between two outer roots.
	eventAugment

	// eventExpand expands a zero-dual inner blossom.
	eventExpand
)

// blossomEvent describes one structural action discovered by tight-edge scanning.
// It preserves both top-level node ownership and original endpoint ownership,
// which is required for contraction metadata and augmenting-path lifting.
//
// Implementation:
//   - Stage 1: scanOuterNodeTightEdges detects a tight dense edge.
//   - Stage 2: orientEdgeForNodes maps the dense edge endpoints into active top-level nodes.
//   - Stage 3: The event stores both node IDs and original local vertices.
//   - Stage 4: applyBlossomEvent dispatches one mutation without re-deriving endpoint ownership.
//
// Behavior highlights:
//   - Transient value; never stored across dual updates.
//   - a and b are active top-level node IDs.
//   - aVertex is an original local vertex currently owned by a.
//   - bVertex is an original local vertex currently owned by b.
//
// Inputs:
//   - Produced by tight-edge scanning.
//
// Returns:
//   - Consumed by grow, shrink, augment, or expand.
//
// Errors:
//   - Invalid endpoint ownership is rejected before the event is returned.
//
// Determinism:
//   - The first event follows queue order, member order, incident edge order, and edge ID order.
//
// Complexity:
//   - Storage O(1).
//   - Endpoint orientation is O(|members(a)|+|members(b)|) through membership checks.
//
// Notes:
//   - Path lifting must not infer endpoints from node IDs alone.
//   - A contracted blossom may contain many original vertices, so endpoint ownership is essential.
//
// AI-Hints:
//   - Do not remove aVertex/bVertex.
//   - Do not apply events directly inside incident-edge loops.
//   - Do not store original TSP matrix vertices in a/b.
//   - Do not store original TSP graph IDs here; these are local matchingProblem vertices.
type blossomEvent struct {
	// kind selects which structural mutation must be applied.
	kind blossomEventKind

	// edge is the dense local edge that triggered grow, shrink, or augment.
	edge int

	// a is the first active top-level node participating in the event.
	a int

	// b is the second active top-level node participating in the event.
	b int

	// aVertex is the original local endpoint of edge owned by node a.
	aVertex int

	// bVertex is the original local endpoint of edge owned by node b.
	bVertex int
}

// blossomPathStep stores one oriented edge traversal between top-level nodes.
// It is the unit consumed by augmenting-path lifting before mate[] is mutated.
//
// Implementation:
//   - Stage 1: pathToRoot emits steps from an outer node toward its root.
//   - Stage 2: composeAugmentingPath reverses/joins steps into root-to-root order.
//   - Stage 3: liftAugmentingPath expands steps that pass through contracted blossoms.
//
// Behavior highlights:
//   - edge identifies the dense original edge.
//   - fromNode/toNode are top-level nodes at the time the path is composed.
//   - fromVertex/toVertex are original local endpoints owned by those nodes.
//
// Inputs:
//   - Built from alternating forest parent links and event endpoint ownership.
//
// Returns:
//   - Consumed by liftAugmentingPath.
//
// Errors:
//   - Invalid ownership is rejected by step constructors and validation helpers.
//
// Determinism:
//   - Step order follows deterministic parent links and event order.
//
// Complexity:
//   - Storage O(1).
//
// Notes:
//   - This type prevents augment from losing endpoint ownership.
//
// AI-Hints:
//   - Do not collapse this back to []int edge IDs before lifting is complete.
type blossomPathStep struct {
	// edge is the dense local edge traversed by this step.
	edge int

	// fromNode is the active top-level node where this oriented step starts.
	fromNode int

	// toNode is the active top-level node where this oriented step ends.
	toNode int

	// fromVertex is the original local endpoint owned by fromNode.
	fromVertex int

	// toVertex is the original local endpoint owned by toNode.
	toVertex int
}

// blossomDeltaKind names the dual-update reason selected by nextDelta.
type blossomDeltaKind uint8

const (
	// deltaNone means no finite dual update was found.
	deltaNone blossomDeltaKind = iota

	// deltaGrowToUnlabeled makes at least one outer-to-unlabeled edge tight.
	deltaGrowToUnlabeled

	// deltaJoinOuterTrees makes at least one outer-to-outer edge tight.
	deltaJoinOuterTrees

	// deltaExpandInnerBlossom reduces an inner blossom dual to zero.
	deltaExpandInnerBlossom
)

// blossomDelta stores one dense dual update candidate.
// The engine applies the smallest non-negative candidate to create a new tight
// edge or unlock a zero-dual expansion.
//
// AI-Hints:
//   - Do not accumulate matching cost from delta values.
//   - Cost is recomputed from matchingProblem after export.
type blossomDelta struct {
	// kind identifies why this delta is valid.
	kind blossomDeltaKind

	// value is the non-negative dual movement to apply.
	// Small negative values inside eps are clamped before selection.
	value float64

	// edge is the dense edge made tight by this delta.
	// noEdge is valid for inner-blossom expansion deltas.
	edge int

	// node is the node affected by this delta.
	// For grow it is the unlabeled endpoint; for expansion it is the blossom node.
	node int
}

// solve repeatedly augments the current matching until every local vertex is matched.
//
// Implementation:
//   - Stage 1: Compute target pair count.
//   - Stage 2: Search and apply one augmenting structure per iteration.
//   - Stage 3: Verify final matching, partition, and dual state.
//
// Behavior highlights:
//   - Does not call greedy matching.
//   - Does not use the exponential oracle.
//   - Stops only when mate[] covers every original local vertex.
//
// Inputs:
//   - None; consumes engine state.
//
// Returns:
//   - error: nil when the engine reaches a verified perfect matching.
//
// Errors:
//   - ErrIncompleteGraph when no augmenting structure can be found.
//   - ErrInvalidMatching from corrupt forest, contraction, or mate state.
//
// Determinism:
//   - Every search phase scans deterministic dense edge order.
//
// Complexity:
//   - Dense correctness-first polynomial Blossom target with O(k^2) edge storage.
//
// AI-Hints:
//   - Do not add size-based refusal here; BlossomMatch is exact-or-sentinel, not exact-until-k.
func (e *blossomEngine) solve() error {
	targetPairs := e.problem.n / 2

	for e.matchedPairs() < targetPairs {
		if err := e.findAndApplyAugmentation(); err != nil {
			return err
		}

		e.stats.Augmentations++
	}

	return e.verifyOptimalState()
}

// findAndApplyAugmentation grows an alternating forest until one augmenting event is applied.
//
// Implementation:
//   - Stage 1: Reset forest labels and queue.
//   - Stage 2: Add every currently free original vertex top-node as an outer root.
//   - Stage 3: Scan tight edges.
//   - Stage 4: If no tight event exists, apply the smallest valid dual delta and continue.
//
// Behavior highlights:
//   - Uses a head cursor; never shifts queue storage.
//   - All free roots are inserted in increasing original-vertex order.
//   - Dual updates are deterministic dense scans.
//
// Inputs:
//   - None.
//
// Returns:
//   - error: nil after one augmentation is applied.
//
// Errors:
//   - ErrIncompleteGraph when no finite delta exists.
//   - ErrInvalidMatching for inconsistent blossom/forest state.
//
// Determinism:
//   - Root insertion and tight-edge scans follow local vertex/edge order.
//
// Complexity:
//   - O(k^2) per tight scan/delta cycle.
//
// AI-Hints:
//   - Do not replace the queue with queue=queue[1:] retention-prone slicing.
func (e *blossomEngine) findAndApplyAugmentation() error {
	e.resetForest()

	for vertex := 0; vertex < e.problem.n; vertex++ {
		if e.mate[vertex] != noVertex {
			continue
		}

		top := e.inBlossom[vertex]
		if top < 0 || top >= len(e.active) || !e.active[top] {
			return ErrInvalidMatching
		}
		if e.label[top] == blossomUnlabeled {
			e.assignOuterRoot(top)
		}
	}

	for {
		event, ok, err := e.scanTightEdges()
		if err != nil {
			return err
		}
		if ok {
			return e.applyBlossomEvent(event)
		}

		delta, err := e.nextDelta()
		if err != nil {
			return err
		}
		if err = e.applyDelta(delta); err != nil {
			return err
		}
	}
}

func (e *blossomEngine) resetForest() {
	for node := range e.label {
		e.label[node] = blossomUnlabeled
		e.labelEdge[node] = noEdge
		e.treeRoot[node] = noNode
		e.parent[node] = noNode
	}

	e.queue = e.queue[:0]
	e.head = 0
}

// assignOuterRoot inserts an active top-level node as a root of the alternating forest.
// Roots are outer nodes with no parent and no label edge.
//
// Complexity:
//   - Time O(1), Space amortized O(1).
func (e *blossomEngine) assignOuterRoot(node int) {
	e.label[node] = blossomOuter
	e.treeRoot[node] = node
	e.labelEdge[node] = noEdge
	e.parent[node] = noNode
	e.queue = append(e.queue, node)
}

// topNodeOfVertex returns the active top-level node currently owning an original vertex.
// It is the safe accessor for crossing from dense edges over original vertices into
// the contracted blossom forest.
//
// Complexity:
//   - Time O(1), Space O(1).
func (e *blossomEngine) topNodeOfVertex(vertex int) (int, error) {
	if vertex < 0 || vertex >= e.problem.n {
		return noNode, ErrInvalidMatching
	}

	node := e.inBlossom[vertex]
	if node < 0 || node >= len(e.active) || !e.active[node] {
		return noNode, ErrInvalidMatching
	}

	return node, nil
}

// scanTightEdges scans queued outer nodes until one structural event is found.
// If the queue is exhausted, the caller must apply a dual delta and scan again.
//
// Complexity:
//   - Time O(number of scanned incident edges), Space O(1).
func (e *blossomEngine) scanTightEdges() (blossomEvent, bool, error) {
	for e.head < len(e.queue) {
		node := e.queue[e.head]
		e.head++

		if node < 0 || node >= len(e.active) || !e.active[node] || e.label[node] != blossomOuter {
			continue
		}

		event, ok, err := e.scanOuterNodeTightEdges(node)
		if err != nil || ok {
			return event, ok, err
		}
	}

	return blossomEvent{kind: eventNone, edge: noEdge, a: noNode, b: noNode}, false, nil
}

// scanOuterNodeTightEdges scans all dense edges incident to original members of one outer node.
// Tight edges to unlabeled nodes grow the tree; tight edges to outer nodes either shrink
// a same-tree odd cycle or augment across two roots.
//
// Complexity:
//   - Time O(|members(node)| * k) in dense representation, Space O(1).
func (e *blossomEngine) scanOuterNodeTightEdges(node int) (blossomEvent, bool, error) {
	if node < 0 || node >= len(e.members) || !e.active[node] {
		return blossomEvent{}, false, ErrInvalidMatching
	}

	for _, original := range e.members[node] {
		if original < 0 || original >= e.problem.n {
			return blossomEvent{}, false, ErrInvalidMatching
		}

		for _, edgeID := range e.incident[original] {
			if edgeID < 0 || edgeID >= len(e.edges) {
				return blossomEvent{}, false, ErrInvalidMatching
			}
			if !e.isTight(edgeID) {
				continue
			}

			edge := e.edges[edgeID]
			other := edge.v
			if other == original {
				other = edge.u
			}

			otherTop, err := e.topNodeOfVertex(other)
			if err != nil {
				return blossomEvent{}, false, err
			}
			if otherTop == node {
				continue
			}

			aVertex, bVertex, err := e.orientEdgeForNodes(edgeID, node, otherTop)
			if err != nil {
				return blossomEvent{}, false, err
			}

			switch e.label[otherTop] {
			case blossomUnlabeled:
				return blossomEvent{
					kind:    eventGrow,
					edge:    edgeID,
					a:       node,
					b:       otherTop,
					aVertex: aVertex,
					bVertex: bVertex,
				}, true, nil

			case blossomOuter:
				if e.treeRoot[node] == e.treeRoot[otherTop] {
					return blossomEvent{
						kind:    eventShrink,
						edge:    edgeID,
						a:       node,
						b:       otherTop,
						aVertex: aVertex,
						bVertex: bVertex,
					}, true, nil
				}

				return blossomEvent{
					kind:    eventAugment,
					edge:    edgeID,
					a:       node,
					b:       otherTop,
					aVertex: aVertex,
					bVertex: bVertex,
				}, true, nil

			case blossomInner:
				continue

			default:
				return blossomEvent{}, false, ErrInvalidMatching
			}
		}
	}

	return blossomEvent{}, false, nil
}

// applyBlossomEvent dispatches one tight-edge structural event.
// It is the only switch that mutates forest structure from event records.
//
// Complexity:
//   - Event-dependent: grow O(1), augment/shrink/expand O(k).
func (e *blossomEngine) applyBlossomEvent(event blossomEvent) error {
	switch event.kind {
	case eventGrow:
		return e.grow(event)

	case eventShrink:
		return e.shrink(event)

	case eventAugment:
		return e.augment(event)

	case eventExpand:
		return e.expand(event.a)

	case eventNone:
		return ErrInvalidMatching

	default:
		return ErrInvalidMatching
	}
}

// grow extends an alternating tree through a tight edge to an unlabeled top-level node.
// The unlabeled node becomes inner; its matched partner blossom becomes outer and is queued.
//
// Complexity:
//   - Time O(1), Space amortized O(1).
func (e *blossomEngine) grow(event blossomEvent) error {
	if event.kind != eventGrow {
		return ErrInvalidMatching
	}

	fromOuter := event.a
	toUnlabeled := event.b

	if fromOuter < 0 || fromOuter >= len(e.active) || toUnlabeled < 0 || toUnlabeled >= len(e.active) {
		return ErrInvalidMatching
	}
	if !e.active[fromOuter] || !e.active[toUnlabeled] {
		return ErrInvalidMatching
	}
	if e.label[fromOuter] != blossomOuter || e.label[toUnlabeled] != blossomUnlabeled {
		return ErrInvalidMatching
	}
	if !e.nodeContainsVertex(fromOuter, event.aVertex) || !e.nodeContainsVertex(toUnlabeled, event.bVertex) {
		return ErrInvalidMatching
	}

	e.label[toUnlabeled] = blossomInner
	e.labelEdge[toUnlabeled] = event.edge
	e.parent[toUnlabeled] = fromOuter
	e.treeRoot[toUnlabeled] = e.treeRoot[fromOuter]

	base := e.base[toUnlabeled]
	if base < 0 || base >= e.problem.n {
		return ErrInvalidMatching
	}

	mate := e.mate[base]
	if mate == noVertex {
		return ErrInvalidMatching
	}

	mateTop, err := e.topNodeOfVertex(mate)
	if err != nil {
		return err
	}

	e.label[mateTop] = blossomOuter
	e.labelEdge[mateTop] = e.mateEdge[base]
	e.parent[mateTop] = toUnlabeled
	e.treeRoot[mateTop] = e.treeRoot[fromOuter]
	e.queue = append(e.queue, mateTop)

	return nil
}

// slack computes dual slack for one dense edge under current top-level ownership.
// A non-positive slack within eps means the edge is tight and may drive grow,
// shrink, or augment events.
//
// Complexity:
//   - Time O(1), Space O(1).
func (e *blossomEngine) slack(edgeID int) float64 {
	edge := e.edges[edgeID]

	uTop := e.inBlossom[edge.u]
	vTop := e.inBlossom[edge.v]

	return e.dual[uTop] + e.dual[vTop] - 2*edge.profit
}

// isTight reports whether an edge is tight under the current dual state.
// It increments TightScans for deterministic internal telemetry.
//
// Complexity:
//   - Time O(1), Space O(1).
func (e *blossomEngine) isTight(edgeID int) bool {
	e.stats.TightScans++

	return e.slack(edgeID) <= e.eps
}

// improveDelta updates best when candidate is a better non-negative dual movement.
// Ties inside eps are resolved by smaller dense edge ID when an edge is present.
//
// Complexity:
//   - Time O(1), Space O(1).
func (e *blossomEngine) improveDelta(best *blossomDelta, candidate blossomDelta) {
	if candidate.value < 0 && math.Abs(candidate.value) <= e.eps {
		candidate.value = 0
	}
	if candidate.value < -e.eps || math.IsNaN(candidate.value) || math.IsInf(candidate.value, 0) {
		return
	}
	if best.kind == deltaNone || candidate.value < best.value-e.eps ||
		(math.Abs(candidate.value-best.value) <= e.eps && candidate.edge != noEdge && candidate.edge < best.edge) {
		*best = candidate
	}
}

// nextDelta selects the smallest dual update that can create a new event.
// It considers outer-to-unlabeled growth, outer-to-outer joins, and zero-dual
// inner-blossom expansion candidates.
//
// Complexity:
//   - Time O(k^2), Space O(1).
func (e *blossomEngine) nextDelta() (blossomDelta, error) {
	best := blossomDelta{
		kind:  deltaNone,
		value: math.Inf(1),
		edge:  noEdge,
		node:  noNode,
	}

	if err := e.considerGrowDeltas(&best); err != nil {
		return best, err
	}
	if err := e.considerJoinDeltas(&best); err != nil {
		return best, err
	}
	if err := e.considerExpandDeltas(&best); err != nil {
		return best, err
	}

	if best.kind == deltaNone || math.IsInf(best.value, 1) {
		return best, ErrIncompleteGraph
	}

	return best, nil
}

// considerGrowDeltas finds dual updates that make outer-to-unlabeled edges tight.
//
// Complexity:
//   - Time O(k^2), Space O(1).
func (e *blossomEngine) considerGrowDeltas(best *blossomDelta) error {
	for _, edge := range e.edges {
		uTop, err := e.topNodeOfVertex(edge.u)
		if err != nil {
			return err
		}
		vTop, err := e.topNodeOfVertex(edge.v)
		if err != nil {
			return err
		}
		if uTop == vTop {
			continue
		}

		uOuter := e.label[uTop] == blossomOuter
		vOuter := e.label[vTop] == blossomOuter

		if uOuter && e.label[vTop] == blossomUnlabeled {
			e.improveDelta(best, blossomDelta{kind: deltaGrowToUnlabeled, value: e.slack(edge.id), edge: edge.id, node: vTop})
		}
		if vOuter && e.label[uTop] == blossomUnlabeled {
			e.improveDelta(best, blossomDelta{kind: deltaGrowToUnlabeled, value: e.slack(edge.id), edge: edge.id, node: uTop})
		}
	}

	return nil
}

// considerJoinDeltas finds dual updates that make outer-to-outer edges tight.
// Same-tree joins become shrink events; different-root joins become augment events.
//
// Complexity:
//   - Time O(k^2), Space O(1).
func (e *blossomEngine) considerJoinDeltas(best *blossomDelta) error {
	for _, edge := range e.edges {
		uTop, err := e.topNodeOfVertex(edge.u)
		if err != nil {
			return err
		}
		vTop, err := e.topNodeOfVertex(edge.v)
		if err != nil {
			return err
		}
		if uTop == vTop {
			continue
		}
		if e.label[uTop] == blossomOuter && e.label[vTop] == blossomOuter {
			e.improveDelta(best, blossomDelta{
				kind:  deltaJoinOuterTrees,
				value: e.slack(edge.id) / 2,
				edge:  edge.id,
				node:  noNode,
			})
		}
	}

	return nil
}

// considerExpandDeltas finds inner blossoms whose dual can be reduced to zero.
//
// Complexity:
//   - Time O(k), Space O(1).
func (e *blossomEngine) considerExpandDeltas(best *blossomDelta) error {
	for node := e.problem.n; node < len(e.active); node++ {
		if !e.active[node] || e.label[node] != blossomInner {
			continue
		}

		e.improveDelta(best, blossomDelta{
			kind:  deltaExpandInnerBlossom,
			value: e.dual[node],
			edge:  noEdge,
			node:  node,
		})
	}

	return nil
}

// applyDelta moves the active dual system by one selected non-negative delta.
// The movement preserves dual feasibility while creating at least one new tight edge
// or enabling expansion of a zero-dual inner blossom.
//
// Implementation:
//   - Stage 1: Reject NaN, ±Inf, or negative delta values outside eps.
//   - Stage 2: Scan active nodes in deterministic node-index order.
//   - Stage 3: Decrease outer-node duals and increase inner-node duals.
//   - Stage 4: Leave unlabeled active nodes unchanged because they are outside the current forest.
//   - Stage 5: Clamp tiny near-zero drift within eps and reject truly negative active duals.
//   - Stage 6: Record one dual update in stats.
//
// Behavior highlights:
//   - Does not mutate mate or blossom membership.
//   - Does not scan edges directly.
//   - The caller must rescan tight edges after this method returns.
//   - Unlabeled nodes are explicitly ignored, not accidentally skipped.
//
// Inputs:
//   - delta: selected dual movement from nextDelta.
//
// Returns:
//   - error: nil when the dual state remains feasible.
//
// Errors:
//   - ErrInvalidMatching for invalid delta values, impossible labels, or negative active duals.
//
// Determinism:
//   - Fixed active node-index scan.
//   - No map iteration or random choice.
//
// Complexity:
//   - Time O(B), Space O(1), where B is allocated blossom node capacity O(k).
//
// Notes:
//   - Matching cost is never accumulated from dual values.
//   - Cost is recomputed from original matchingProblem costs after export.
//
// AI-Hints:
//   - Do not update inactive child blossoms.
//   - Do not treat unlabeled active nodes as outer nodes.
//   - Do not remove the near-zero clamp; floating-point drift around zero is expected.
func (e *blossomEngine) applyDelta(delta blossomDelta) error {
	if delta.value < -e.eps || math.IsNaN(delta.value) || math.IsInf(delta.value, 0) {
		return ErrInvalidMatching
	}

	for node := range e.active {
		if !e.active[node] {
			continue
		}

		switch e.label[node] {
		case blossomOuter:
			e.dual[node] -= delta.value

		case blossomInner:
			e.dual[node] += delta.value

		case blossomUnlabeled:
			continue

		default:
			return ErrInvalidMatching
		}

		if math.Abs(e.dual[node]) <= e.eps {
			e.dual[node] = 0
		}
		if e.dual[node] < -e.eps {
			return ErrInvalidMatching
		}
	}

	e.stats.DualUpdates++

	return nil
}

// augment applies one endpoint-aware augmenting path discovered between two outer trees.
// It composes a root-to-root path, lifts contracted blossoms into original edge sequences,
// validates alternation, then flips matching state atomically at the mate[] level.
//
// Implementation:
//   - Stage 1: Reconstruct the left outer-to-root endpoint-aware path.
//   - Stage 2: Reconstruct the right outer-to-root endpoint-aware path.
//   - Stage 3: Build the connecting bridge step from event endpoint ownership.
//   - Stage 4: Compose a root-to-root path in correct orientation.
//   - Stage 5: Lift all contracted blossoms into original dense edge IDs.
//   - Stage 6: Flip the validated augmenting sequence and verify mate symmetry.
//
// Behavior highlights:
//   - Mutates mate[] only after lifting succeeds.
//   - Does not mutate contraction state.
//   - Rejects non-alternating lifted paths.
//   - Supports nested contracted blossoms through recursive lifting.
//
// Inputs:
//   - event: eventAugment with edge, node, and endpoint ownership fields populated.
//
// Returns:
//   - error: nil after a successful matching augmentation.
//
// Errors:
//   - ErrInvalidMatching for malformed event, invalid paths, failed lifting,
//     duplicate edges, broken alternation, or mate/mateEdge corruption.
//
// Determinism:
//   - Parent paths, bridge orientation, candidate lifting, and flip order are deterministic.
//
// Complexity:
//   - Time O(k + L + C), Space O(k + L + C), where L is lifted path length
//     and C is candidate lifting overhead for crossed blossoms.
//
// Notes:
//   - This method must not accept event values without endpoint ownership.
//
// AI-Hints:
//   - Do not flip the top-level path directly.
//   - Do not mutate mate[] before liftAugmentingPath returns successfully.
func (e *blossomEngine) augment(event blossomEvent) error {
	if event.kind != eventAugment {
		return ErrInvalidMatching
	}

	leftPath, err := e.pathToRoot(event.a)
	if err != nil {
		return err
	}

	rightPath, err := e.pathToRoot(event.b)
	if err != nil {
		return err
	}

	bridge := blossomPathStep{
		edge:       event.edge,
		fromNode:   event.a,
		toNode:     event.b,
		fromVertex: event.aVertex,
		toVertex:   event.bVertex,
	}

	path, err := e.composeAugmentingPath(leftPath, bridge, rightPath)
	if err != nil {
		return err
	}

	lifted, err := e.liftAugmentingPath(path)
	if err != nil {
		return err
	}

	if err = e.flipAugmentingPath(lifted); err != nil {
		return err
	}

	return e.verifyMatchingSymmetry()
}

// pathToRoot returns oriented path steps from an outer node toward its alternating-tree root.
// It records endpoint ownership at every parent edge so contracted blossoms can later be lifted.
//
// Implementation:
//   - Stage 1: Validate active outer start node.
//   - Stage 2: Follow parent links until the root.
//   - Stage 3: Convert every labelEdge into an oriented blossomPathStep.
//   - Stage 4: Stop when the current node is its own tree root.
//
// Behavior highlights:
//   - Does not mutate engine state.
//   - Does not expand blossoms.
//   - Returns node/endpoint-aware steps, not bare edge IDs.
//
// Inputs:
//   - node: active outer top-level node.
//
// Returns:
//   - []blossomPathStep: path from node toward root.
//   - error: nil on valid forest chain.
//
// Errors:
//   - ErrInvalidMatching for inactive nodes, non-outer start node,
//     missing label edges, or corrupt parent links.
//
// Determinism:
//   - Follows exactly one parent chain.
//
// Complexity:
//   - Time O(k*m), Space O(k), where m is average membership scan cost.
//
// Notes:
//   - composeAugmentingPath owns final orientation.
//
// AI-Hints:
//   - Do not return []int here; augment needs endpoint ownership.
func (e *blossomEngine) pathToRoot(node int) ([]blossomPathStep, error) {
	if node < 0 || node >= len(e.active) || !e.active[node] {
		return nil, ErrInvalidMatching
	}
	if e.label[node] != blossomOuter {
		return nil, ErrInvalidMatching
	}

	steps := make([]blossomPathStep, 0, e.problem.n)

	for current := node; current != noNode; current = e.parent[current] {
		if current < 0 || current >= len(e.active) {
			return nil, ErrInvalidMatching
		}

		if e.treeRoot[current] == current {
			break
		}

		parent := e.parent[current]
		if parent == noNode || e.labelEdge[current] == noEdge {
			return nil, ErrInvalidMatching
		}

		step, err := e.makePathStep(e.labelEdge[current], current, parent)
		if err != nil {
			return nil, err
		}

		steps = append(steps, step)
	}

	return steps, nil
}

// composeAugmentingPath combines two outer-root paths and the connecting tight edge.
// It returns a root-to-root oriented path: leftRoot -> leftOuter -> rightOuter -> rightRoot.
//
// Implementation:
//   - Stage 1: Validate and append the left path in reverse orientation.
//   - Stage 2: Append the connecting event edge from leftOuter to rightOuter.
//   - Stage 3: Append the right path in its stored outer-to-root orientation.
//   - Stage 4: Verify path continuity by node and endpoint ownership.
//
// Behavior highlights:
//   - Does not mutate matching state.
//   - Preserves endpoint ownership across contracted nodes.
//   - Keeps the bridge edge orientation from the discovered event.
//
// Inputs:
//   - left: path from leftOuter to leftRoot.
//   - bridge: event step from leftOuter to rightOuter.
//   - right: path from rightOuter to rightRoot.
//
// Returns:
//   - []blossomPathStep: oriented root-to-root augmenting path.
//   - error: nil when composition is structurally valid.
//
// Errors:
//   - ErrInvalidMatching for invalid edge IDs or disconnected step chain.
//
// Determinism:
//   - Reversal and append order are fixed.
//
// Complexity:
//   - Time O(len(left)+len(right)), Space O(len(left)+len(right)+1).
//
// Notes:
//   - Lifting happens after composition, not during composition.
//
// AI-Hints:
//   - Do not use left+bridge+reverse(right); that orients the path incorrectly.
func (e *blossomEngine) composeAugmentingPath(
	left []blossomPathStep,
	bridge blossomPathStep,
	right []blossomPathStep,
) ([]blossomPathStep, error) {
	if bridge.edge < 0 || bridge.edge >= len(e.edges) {
		return nil, ErrInvalidMatching
	}

	out := make([]blossomPathStep, 0, len(left)+1+len(right))

	for index := len(left) - 1; index >= 0; index-- {
		out = append(out, reversePathStep(left[index]))
	}

	out = append(out, bridge)
	out = append(out, right...)

	if err := e.verifyPathStepContinuity(out); err != nil {
		return nil, err
	}

	return out, nil
}

// isMatchedEdge reports whether edgeID is currently the committed mate edge.
// It checks mate and mateEdge symmetrically, so stale one-sided state is not
// mistaken for a valid matched edge.
//
// Complexity:
//   - Time O(1), Space O(1).
func (e *blossomEngine) isMatchedEdge(edgeID int) bool {
	edge := e.edges[edgeID]

	return e.mate[edge.u] == edge.v &&
		e.mate[edge.v] == edge.u &&
		e.mateEdge[edge.u] == edgeID &&
		e.mateEdge[edge.v] == edgeID
}

// verifyPathStepContinuity checks that consecutive oriented steps meet at the same node.
// When the shared node is a singleton, the original endpoint must also match;
// when it is a contracted blossom, liftAugmentingPath will reconstruct the internal route.
//
// Complexity:
//   - Time O(len(steps)), Space O(1).
func (e *blossomEngine) verifyPathStepContinuity(steps []blossomPathStep) error {
	for index := 1; index < len(steps); index++ {
		prev := steps[index-1]
		next := steps[index]

		if prev.toNode != next.fromNode {
			return ErrInvalidMatching
		}

		if prev.toNode < e.problem.n && prev.toVertex != next.fromVertex {
			return ErrInvalidMatching
		}
	}

	return nil
}

// liftAugmentingPath expands an endpoint-aware top-level path into original dense edge IDs.
// The function inserts internal blossom cycle paths whenever consecutive external steps
// pass through a contracted blossom.
//
// Implementation:
//   - Stage 1: Start with one empty candidate path.
//   - Stage 2: Insert internal paths at the first contracted start node when needed.
//   - Stage 3: Append each external edge as a one-edge segment.
//   - Stage 4: Insert internal blossom paths between consecutive steps that share a contracted node.
//   - Stage 5: Insert an internal path to the base of the final contracted node when needed.
//   - Stage 6: Select the first fully alternating augmenting candidate.
//
// Behavior highlights:
//   - Does not mutate mate[].
//   - Preserves every external event edge.
//   - Supports nested contracted blossoms through recursive path choices.
//   - Rejects paths that do not alternate matched/unmatched edge status.
//
// Inputs:
//   - steps: root-to-root oriented top-level path.
//
// Returns:
//   - []int: lifted original dense edge IDs ready for flipAugmentingPath.
//   - error: nil when a valid augmenting sequence is found.
//
// Errors:
//   - ErrInvalidMatching for malformed steps, broken ownership, or no alternating lift.
//
// Determinism:
//   - Candidate order is forward cycle path first, backward cycle path second.
//   - The first valid candidate is selected deterministically.
//
// Complexity:
//   - Worst-case O(2^b * L) for b crossed blossoms, but b is bounded by nesting in one augmenting path.
//   - Typical dense search remains governed by O(k^2) scans plus O(k) lifting.
//
// Notes:
//   - This is correctness-first; performance tuning belongs after shrink/augment tests pass.
//
// AI-Hints:
//   - Do not mutate mate[] before this function succeeds.
//   - Do not ignore alternate cycle direction when the first candidate fails alternation.
func (e *blossomEngine) liftAugmentingPath(steps []blossomPathStep) ([]int, error) {
	if len(steps) == 0 {
		return nil, ErrInvalidMatching
	}

	candidates := [][]int{{}}

	first := steps[0]
	if first.fromNode >= e.problem.n {
		choices, err := e.liftThroughBlossomChoices(first.fromNode, e.base[first.fromNode], first.fromVertex)
		if err != nil {
			return nil, err
		}
		candidates = e.extendAlternatingCandidates(candidates, choices)
		if len(candidates) == 0 {
			return nil, ErrInvalidMatching
		}
	}

	for index, step := range steps {
		if step.edge < 0 || step.edge >= len(e.edges) {
			return nil, ErrInvalidMatching
		}

		if index > 0 {
			prev := steps[index-1]
			if prev.toNode != step.fromNode {
				return nil, ErrInvalidMatching
			}

			if prev.toVertex != step.fromVertex {
				if step.fromNode < e.problem.n {
					return nil, ErrInvalidMatching
				}

				choices, err := e.liftThroughBlossomChoices(step.fromNode, prev.toVertex, step.fromVertex)
				if err != nil {
					return nil, err
				}

				candidates = e.extendAlternatingCandidates(candidates, choices)
				if len(candidates) == 0 {
					return nil, ErrInvalidMatching
				}
			}
		}

		candidates = e.extendAlternatingCandidates(candidates, [][]int{{step.edge}})
		if len(candidates) == 0 {
			return nil, ErrInvalidMatching
		}
	}

	last := steps[len(steps)-1]
	if last.toNode >= e.problem.n {
		choices, err := e.liftThroughBlossomChoices(last.toNode, last.toVertex, e.base[last.toNode])
		if err != nil {
			return nil, err
		}

		candidates = e.extendAlternatingCandidates(candidates, choices)
		if len(candidates) == 0 {
			return nil, ErrInvalidMatching
		}
	}

	for _, candidate := range candidates {
		if err := e.verifyAugmentingEdgeSequence(candidate); err == nil {
			return candidate, nil
		}
	}

	return nil, ErrInvalidMatching
}

// liftThroughBlossom returns the deterministic first alternating internal path between two vertices.
// It is a convenience wrapper around liftThroughBlossomChoices for callers that do not need
// to preserve multiple parity candidates.
//
// Complexity:
//   - Time O(c + nested lifting), Space O(c).
func (e *blossomEngine) liftThroughBlossom(node int, entryVertex int, exitVertex int) ([]int, error) {
	choices, err := e.liftThroughBlossomChoices(node, entryVertex, exitVertex)
	if err != nil {
		return nil, err
	}
	if len(choices) == 0 {
		return nil, ErrInvalidMatching
	}

	return choices[0], nil
}

// liftThroughBlossomChoices returns all deterministic internal edge sequences between two
// original vertices represented by a contracted blossom.
//
// Implementation:
//   - Stage 1: Validate contracted node and trivial same-vertex route.
//   - Stage 2: Locate the cycle child that owns entryVertex and exitVertex.
//   - Stage 3: Build forward and backward cycle paths.
//   - Stage 4: Keep internally alternating candidates only.
//
// Behavior highlights:
//   - Does not mutate engine state.
//   - Handles nested child blossoms recursively.
//   - Returns choices in deterministic forward-then-backward order.
//
// Inputs:
//   - node: contracted blossom node.
//   - entryVertex: original local vertex where the outer path enters node.
//   - exitVertex: original local vertex where the outer path leaves node.
//
// Returns:
//   - [][]int: candidate internal dense edge sequences.
//   - error: nil when at least one structurally valid route can be considered.
//
// Errors:
//   - ErrInvalidMatching for singleton misuse, missing cycle metadata, or missing ownership.
//
// Determinism:
//   - Forward direction is considered first.
//   - Backward direction is considered second.
//
// Complexity:
//   - Time O(c + nested choices), Space O(c + nested choices).
//
// Notes:
//   - The caller still verifies whole augmenting-path alternation after insertion.
//
// AI-Hints:
//   - Do not choose by shortest path; Blossom lifting is parity-sensitive.
func (e *blossomEngine) liftThroughBlossomChoices(node int, entryVertex int, exitVertex int) ([][]int, error) {
	if entryVertex == exitVertex {
		return [][]int{{}}, nil
	}
	if node < e.problem.n {
		return nil, ErrInvalidMatching
	}
	if node < 0 || node >= len(e.cycles) || len(e.cycles[node]) == 0 {
		return nil, ErrInvalidMatching
	}

	fromChild, err := e.cycleChildIndexContainingVertex(node, entryVertex)
	if err != nil {
		return nil, err
	}

	toChild, err := e.cycleChildIndexContainingVertex(node, exitVertex)
	if err != nil {
		return nil, err
	}

	choices := make([][]int, 0, 2)

	forward, err := e.cyclePathBetweenDirection(node, fromChild, toChild, true, entryVertex, exitVertex)
	if err == nil && e.edgeSequenceAlternates(forward) {
		choices = append(choices, forward)
	}

	backward, err := e.cyclePathBetweenDirection(node, fromChild, toChild, false, entryVertex, exitVertex)
	if err == nil && e.edgeSequenceAlternates(backward) {
		choices = append(choices, backward)
	}

	if len(choices) == 0 {
		return nil, ErrInvalidMatching
	}

	return choices, nil
}

// cycleChildIndexContainingVertex finds which child of a contracted blossom owns vertex.
//
// Complexity:
//   - Time O(c*m), Space O(1).
func (e *blossomEngine) cycleChildIndexContainingVertex(node int, vertex int) (int, error) {
	if node < e.problem.n || node >= len(e.cycles) {
		return -1, ErrInvalidMatching
	}

	for index, step := range e.cycles[node] {
		if e.nodeContainsVertex(step.node, vertex) {
			return index, nil
		}
	}

	return -1, ErrInvalidMatching
}

// cyclePathBetween returns the deterministic first internally alternating cycle path
// between two child indices of a contracted blossom.
//
// Complexity:
//   - Time O(c + nested lifting), Space O(c).
func (e *blossomEngine) cyclePathBetween(node int, fromChild int, toChild int) ([]int, error) {
	steps, err := e.cyclePathBetweenDirection(node, fromChild, toChild, true, e.base[e.cycles[node][fromChild].node], e.base[e.cycles[node][toChild].node])
	if err == nil && e.edgeSequenceAlternates(steps) {
		return steps, nil
	}

	return e.cyclePathBetweenDirection(node, fromChild, toChild, false, e.base[e.cycles[node][fromChild].node], e.base[e.cycles[node][toChild].node])
}

// cyclePathBetweenDirection builds one directed path along a contracted blossom cycle.
// It recursively lifts through child blossoms whenever a child is itself contracted.
//
// Complexity:
//   - Time O(c + nested lifting), Space O(c + nested lifting).
func (e *blossomEngine) cyclePathBetweenDirection(
	node int,
	fromChild int,
	toChild int,
	forward bool,
	entryVertex int,
	exitVertex int,
) ([]int, error) {
	if node < e.problem.n || node >= len(e.cycles) {
		return nil, ErrInvalidMatching
	}

	cycle := e.cycles[node]
	if len(cycle) < 3 || (len(cycle)&1) == 0 {
		return nil, ErrInvalidMatching
	}
	if fromChild < 0 || fromChild >= len(cycle) || toChild < 0 || toChild >= len(cycle) {
		return nil, ErrInvalidMatching
	}

	out := make([]int, 0, len(cycle))
	currentIndex := fromChild
	currentVertex := entryVertex

	for currentIndex != toChild {
		currentNode := cycle[currentIndex].node

		if forward {
			step := cycle[currentIndex]

			internal, err := e.liftInsideChild(currentNode, currentVertex, step.vertexToNext)
			if err != nil {
				return nil, err
			}
			out = append(out, internal...)
			out = append(out, step.edgeToNext)

			currentVertex = step.nextVertex
			currentIndex++
			if currentIndex == len(cycle) {
				currentIndex = 0
			}

			continue
		}

		previousIndex := currentIndex - 1
		if previousIndex < 0 {
			previousIndex = len(cycle) - 1
		}

		previousStep := cycle[previousIndex]

		internal, err := e.liftInsideChild(currentNode, currentVertex, previousStep.nextVertex)
		if err != nil {
			return nil, err
		}
		out = append(out, internal...)
		out = append(out, previousStep.edgeToNext)

		currentVertex = previousStep.vertexToNext
		currentIndex = previousIndex
	}

	finalNode := cycle[currentIndex].node
	internal, err := e.liftInsideChild(finalNode, currentVertex, exitVertex)
	if err != nil {
		return nil, err
	}

	out = append(out, internal...)

	return out, nil
}

// liftInsideChild lifts a path inside one cycle child.
// Singleton children require the same entry/exit vertex; contracted child blossoms recurse.
//
// Complexity:
//   - Time O(1) for singleton children; recursive cost for contracted children.
func (e *blossomEngine) liftInsideChild(child int, entryVertex int, exitVertex int) ([]int, error) {
	if child < 0 || child >= len(e.members) {
		return nil, ErrInvalidMatching
	}
	if !e.nodeContainsVertex(child, entryVertex) || !e.nodeContainsVertex(child, exitVertex) {
		return nil, ErrInvalidMatching
	}

	if child < e.problem.n {
		if entryVertex != exitVertex {
			return nil, ErrInvalidMatching
		}

		return nil, nil
	}

	return e.liftThroughBlossom(child, entryVertex, exitVertex)
}

// extendAlternatingCandidates appends every segment choice to every prefix and keeps
// only prefixes that still satisfy local alternating edge parity.
//
// Complexity:
//   - Time O(p*c*s), Space O(p*c*s), where p is prefix count, c is choice count, s is segment length.
func (e *blossomEngine) extendAlternatingCandidates(prefixes [][]int, choices [][]int) [][]int {
	out := make([][]int, 0, len(prefixes)*len(choices))

	for _, prefix := range prefixes {
		for _, choice := range choices {
			if !e.canAppendAlternating(prefix, choice) {
				continue
			}

			next := make([]int, 0, len(prefix)+len(choice))
			next = append(next, prefix...)
			next = append(next, choice...)
			out = append(out, next)
		}
	}

	return out
}

// canAppendAlternating reports whether segment can be appended without breaking
// matched/unmatched alternation at the join or inside the segment.
//
// Complexity:
//   - Time O(len(segment)), Space O(1).
func (e *blossomEngine) canAppendAlternating(prefix []int, segment []int) bool {
	if len(segment) == 0 {
		return true
	}

	previousKnown := false
	previousMatched := false

	if len(prefix) > 0 {
		previousKnown = true
		previousMatched = e.isMatchedEdge(prefix[len(prefix)-1])
	}

	for _, edgeID := range segment {
		if edgeID < 0 || edgeID >= len(e.edges) {
			return false
		}

		currentMatched := e.isMatchedEdge(edgeID)
		if previousKnown && currentMatched == previousMatched {
			return false
		}

		previousKnown = true
		previousMatched = currentMatched
	}

	return true
}

// flipAugmentingPath toggles matched status along an alternating edge sequence.
// Existing matched edges are cleared first, then previously unmatched edges are set,
// preventing transient conflicts from rejecting a valid augmenting path.
//
// Complexity:
//   - Time O(len(edges)), Space O(len(edges)).
func (e *blossomEngine) flipAugmentingPath(edges []int) error {
	if err := e.verifyAugmentingEdgeSequence(edges); err != nil {
		return err
	}

	wasMatched := make([]bool, len(edges))

	for index, edgeID := range edges {
		wasMatched[index] = e.isMatchedEdge(edgeID)
	}

	for index, edgeID := range edges {
		if wasMatched[index] {
			if err := e.clearMatchedEdge(edgeID); err != nil {
				return err
			}
		}
	}

	for index, edgeID := range edges {
		if !wasMatched[index] {
			if err := e.setMatchedEdge(edgeID); err != nil {
				return err
			}
		}
	}

	return nil
}

// setMatchedEdge commits one dense edge as the mate relation of its endpoints.
// It rejects conflicting pre-existing mates and updates mate and mateEdge together.
//
// Complexity:
//   - Time O(1), Space O(1).
func (e *blossomEngine) setMatchedEdge(edgeID int) error {
	if edgeID < 0 || edgeID >= len(e.edges) {
		return ErrInvalidMatching
	}

	edge := e.edges[edgeID]

	if e.mate[edge.u] != noVertex && e.mate[edge.u] != edge.v {
		return ErrInvalidMatching
	}
	if e.mate[edge.v] != noVertex && e.mate[edge.v] != edge.u {
		return ErrInvalidMatching
	}

	e.mate[edge.u] = edge.v
	e.mate[edge.v] = edge.u
	e.mateEdge[edge.u] = edgeID
	e.mateEdge[edge.v] = edgeID

	return nil
}

// clearMatchedEdge removes one currently matched dense edge.
// Both endpoints must agree that edgeID is their mate edge; otherwise the matching
// state is corrupt and ErrInvalidMatching is returned.
//
// Complexity:
//   - Time O(1), Space O(1).
func (e *blossomEngine) clearMatchedEdge(edgeID int) error {
	if edgeID < 0 || edgeID >= len(e.edges) {
		return ErrInvalidMatching
	}

	edge := e.edges[edgeID]

	if e.mate[edge.u] != edge.v || e.mate[edge.v] != edge.u {
		return ErrInvalidMatching
	}

	e.mate[edge.u] = noVertex
	e.mate[edge.v] = noVertex
	e.mateEdge[edge.u] = noEdge
	e.mateEdge[edge.v] = noEdge

	return nil
}

// nodeContainsVertex reports whether node currently represents vertex.
// It checks explicit membership instead of assuming singleton identity, because
// contracted blossoms own multiple original local vertices.
//
// Implementation:
//   - Stage 1: Validate vertex range.
//   - Stage 2: Validate node range.
//   - Stage 3: Scan members[node] for the original vertex.
//
// Behavior highlights:
//   - Does not mutate engine state.
//   - Works for singleton and contracted blossom nodes.
//   - Treats missing membership as false rather than panicking.
//
// Inputs:
//   - node: engine node ID.
//   - vertex: original local matchingProblem vertex.
//
// Returns:
//   - bool: true when vertex belongs to node.
//
// Errors:
//   - None.
//
// Determinism:
//   - Fixed member scan order.
//
// Complexity:
//   - Time O(|members(node)|), Space O(1).
//
// Notes:
//   - This helper is intentionally simple; membership arrays are already explicit.
//
// AI-Hints:
//   - Do not replace this with inBlossom[vertex]==node when checking inactive child ownership.
func (e *blossomEngine) nodeContainsVertex(node int, vertex int) bool {
	if vertex < 0 || vertex >= e.problem.n {
		return false
	}
	if node < 0 || node >= len(e.members) {
		return false
	}

	for _, member := range e.members[node] {
		if member == vertex {
			return true
		}
	}

	return false
}

// verifyAugmentingEdgeSequence validates the lifted original-edge path before mutation.
// A valid augmenting path starts and ends with unmatched edges and alternates matched/unmatched
// status throughout the sequence.
//
// Implementation:
//   - Stage 1: Reject empty or even-length edge sequences.
//   - Stage 2: Validate every edge ID and reject duplicates.
//   - Stage 3: Require the first and last edges to be currently unmatched.
//   - Stage 4: Require strict matched/unmatched alternation between neighbors.
//
// Behavior highlights:
//   - Does not mutate mate[].
//   - Checks original dense edge IDs only.
//   - Protects flipAugmentingPath from corrupt path lifting.
//
// Inputs:
//   - edges: lifted original dense edge sequence.
//
// Returns:
//   - error: nil when the sequence is a valid augmenting path.
//
// Errors:
//   - ErrInvalidMatching for invalid IDs, duplicates, wrong parity, or broken alternation.
//
// Determinism:
//   - Fixed left-to-right scan.
//
// Complexity:
//   - Time O(len(edges)), Space O(|E|) for duplicate detection.
//
// Notes:
//   - This validation is intentionally strict; a failed lift must not partially mutate matching state.
//
// AI-Hints:
//   - Do not weaken first/last unmatched checks.
//   - Do not call setMatchedEdge before this validation succeeds.
func (e *blossomEngine) verifyAugmentingEdgeSequence(edges []int) error {
	if len(edges) == 0 || (len(edges)&1) == 0 {
		return ErrInvalidMatching
	}

	seen := make([]bool, len(e.edges))

	for index, edgeID := range edges {
		if edgeID < 0 || edgeID >= len(e.edges) {
			return ErrInvalidMatching
		}
		if seen[edgeID] {
			return ErrInvalidMatching
		}
		seen[edgeID] = true

		currentMatched := e.isMatchedEdge(edgeID)

		if index == 0 || index == len(edges)-1 {
			if currentMatched {
				return ErrInvalidMatching
			}
		}

		if index > 0 {
			previousMatched := e.isMatchedEdge(edges[index-1])
			if currentMatched == previousMatched {
				return ErrInvalidMatching
			}
		}
	}

	return nil
}

// edgeSequenceAlternates reports whether a non-empty sequence alternates matched status.
// Empty sequences are valid neutral internal paths.
//
// Complexity:
//   - Time O(len(edges)), Space O(1).
func (e *blossomEngine) edgeSequenceAlternates(edges []int) bool {
	for index := 1; index < len(edges); index++ {
		if e.isMatchedEdge(edges[index]) == e.isMatchedEdge(edges[index-1]) {
			return false
		}
	}

	return true
}

// orientEdgeForNodes returns the original endpoints of edgeID as owned by a and b.
// It is the canonical bridge from dense edge endpoints to contracted top-level
// blossom ownership.
//
// Implementation:
//   - Stage 1: Validate edge and active node IDs.
//   - Stage 2: Read the dense edge endpoints.
//   - Stage 3: Test both endpoint orientations against node memberships.
//   - Stage 4: Return endpoints in the same order as nodes a and b.
//
// Behavior highlights:
//   - Does not mutate engine state.
//   - Accepts singleton and contracted nodes.
//   - Rejects edges whose endpoints are not split across the requested nodes.
//
// Inputs:
//   - edgeID: dense local edge ID.
//   - a: first active top-level node.
//   - b: second active top-level node.
//
// Returns:
//   - int: original local vertex owned by a.
//   - int: original local vertex owned by b.
//   - error: nil when endpoint ownership is valid.
//
// Errors:
//   - ErrInvalidMatching for invalid edge ID, inactive nodes, same-node ownership,
//     or endpoint membership mismatch.
//
// Determinism:
//   - Checks the two possible orientations in fixed order.
//
// Complexity:
//   - Time O(|members(a)|+|members(b)|), Space O(1).
//
// Notes:
//   - This helper is used by event creation and cycle metadata construction.
//
// AI-Hints:
//   - Do not guess orientation from edge.u<edge.v.
//   - Do not use this helper with original TSP matrix vertex IDs.
func (e *blossomEngine) orientEdgeForNodes(edgeID int, a int, b int) (int, int, error) {
	if edgeID < 0 || edgeID >= len(e.edges) {
		return noVertex, noVertex, ErrInvalidMatching
	}
	if a < 0 || a >= len(e.active) || b < 0 || b >= len(e.active) {
		return noVertex, noVertex, ErrInvalidMatching
	}
	if !e.active[a] || !e.active[b] || a == b {
		return noVertex, noVertex, ErrInvalidMatching
	}

	edge := e.edges[edgeID]

	if e.nodeContainsVertex(a, edge.u) && e.nodeContainsVertex(b, edge.v) {
		return edge.u, edge.v, nil
	}
	if e.nodeContainsVertex(a, edge.v) && e.nodeContainsVertex(b, edge.u) {
		return edge.v, edge.u, nil
	}

	return noVertex, noVertex, ErrInvalidMatching
}

// reversePathStep reverses one oriented path step.
// It is used when composing the left root-to-outer path from a stored outer-to-root chain.
//
// Complexity:
//   - Time O(1), Space O(1).
func reversePathStep(step blossomPathStep) blossomPathStep {
	return blossomPathStep{
		edge:       step.edge,
		fromNode:   step.toNode,
		toNode:     step.fromNode,
		fromVertex: step.toVertex,
		toVertex:   step.fromVertex,
	}
}

// makePathStep constructs one validated oriented step between two active top-level nodes.
//
// Complexity:
//   - Time O(|members(fromNode)|+|members(toNode)|), Space O(1).
func (e *blossomEngine) makePathStep(edgeID int, fromNode int, toNode int) (blossomPathStep, error) {
	fromVertex, toVertex, err := e.orientEdgeForNodes(edgeID, fromNode, toNode)
	if err != nil {
		return blossomPathStep{}, err
	}

	return blossomPathStep{
		edge:       edgeID,
		fromNode:   fromNode,
		toNode:     toNode,
		fromVertex: fromVertex,
		toVertex:   toVertex,
	}, nil
}
