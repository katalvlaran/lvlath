// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package tsp contracts and expands odd blossoms in dense weighted matching.
// The code keeps original-vertex membership explicit so contraction never loses
// ownership information needed by expansion and final matching export.
package tsp

// shrink contracts an odd alternating cycle into one active top-level blossom node.
// The contraction is required when a tight edge connects two outer nodes that already
// belong to the same alternating tree: the discovered odd cycle cannot be handled by
// bipartite alternating-path logic and must become a single searchable object.
//
// Implementation:
//   - Stage 1: Validate and locate the least common ancestor of the two outer nodes.
//   - Stage 2: Build the ordered cycle arms from both outer nodes toward the common base.
//   - Stage 3: Allocate a new blossom node with base, children, child-edge metadata, and members.
//   - Stage 4: Deactivate child top-level nodes and remap every original member through inBlossom.
//   - Stage 5: Publish the new blossom as an outer node in the same alternating tree.
//   - Stage 6: Enqueue the blossom so its external tight edges can be scanned.
//
// Behavior highlights:
//   - Mutates only contraction/forest ownership state.
//   - Does not change mate[] directly.
//   - Preserves original-vertex ownership through members.
//   - The new blossom inherits outer status because it represents an odd outer cycle.
//
// Inputs:
//   - edgeID: dense tight edge connecting outer nodes a and b.
//   - a: first active outer top-level node.
//   - b: second active outer top-level node in the same tree.
//
// Returns:
//   - error: nil after successful contraction.
//
// Errors:
//   - ErrInvalidMatching for invalid edge IDs, inactive nodes, different tree roots,
//     malformed cycle shape, exhausted blossom node capacity, or corrupted base ownership.
//
// Determinism:
//   - Base detection follows deterministic parent paths.
//   - Child order follows left path then reversed right path.
//   - Queue append order is deterministic.
//
// Complexity:
//   - Time O(k), Space O(k), where k is the local odd-set size.
//
// Notes:
//   - This method must store enough cycle metadata for future augmenting-path lifting.
//   - A blossom with missing child-edge metadata is not mathematically complete.
//
// AI-Hints:
//   - Do not shrink cross-tree outer/outer edges; those are augmenting events.
//   - Do not update mate[] here.
//   - Do not discard original members; expand/export depend on them.
func (e *blossomEngine) shrink(event blossomEvent) error {
	if event.kind != eventShrink {
		return ErrInvalidMatching
	}
	if !e.nodeContainsVertex(event.a, event.aVertex) || !e.nodeContainsVertex(event.b, event.bVertex) {
		return ErrInvalidMatching
	}

	base, pathA, pathB, err := e.findBlossomBaseAndCycle(event.a, event.b)
	if err != nil {
		return err
	}

	newNode, err := e.allocateBlossomNode(base, pathA, pathB, event.edge)
	if err != nil {
		return err
	}

	if err = e.contractCycleIntoBlossom(newNode); err != nil {
		return err
	}

	e.label[newNode] = blossomOuter
	e.treeRoot[newNode] = e.treeRoot[event.a]
	e.parent[newNode] = noNode
	e.labelEdge[newNode] = noEdge
	e.queue = append(e.queue, newNode)

	e.stats.Shrinks++

	return nil
}

// nodePathToRoot returns active node IDs from node to its alternating-tree root.
// It is used by shrink to find the common base node of an odd cycle.
//
// Complexity:
//   - Time O(k), Space O(k).
func (e *blossomEngine) nodePathToRoot(node int) ([]int, error) {
	if node < 0 || node >= len(e.active) || !e.active[node] {
		return nil, ErrInvalidMatching
	}

	path := make([]int, 0, e.problem.n)

	for current := node; current != noNode; current = e.parent[current] {
		if current < 0 || current >= len(e.active) {
			return nil, ErrInvalidMatching
		}

		path = append(path, current)

		if e.treeRoot[current] == current {
			break
		}
	}

	return path, nil
}

// findBlossomBaseAndCycle locates the base and two cycle arms for shrink.
// The returned base is an original local vertex, while left and right are active
// top-level node paths used to allocate the contracted blossom.
//
// Complexity:
//   - Time O(k^2) with straightforward path intersection, Space O(k).
func (e *blossomEngine) findBlossomBaseAndCycle(a int, b int) (int, []int, []int, error) {
	if e.treeRoot[a] != e.treeRoot[b] {
		return noVertex, nil, nil, ErrInvalidMatching
	}

	pathA, err := e.nodePathToRoot(a)
	if err != nil {
		return noVertex, nil, nil, err
	}

	pathB, err := e.nodePathToRoot(b)
	if err != nil {
		return noVertex, nil, nil, err
	}

	commonA := -1
	commonB := -1

	for indexA, nodeA := range pathA {
		for indexB, nodeB := range pathB {
			if nodeA == nodeB {
				commonA = indexA
				commonB = indexB
				break
			}
		}
		if commonA != -1 {
			break
		}
	}

	if commonA == -1 || commonB == -1 {
		return noVertex, nil, nil, ErrInvalidMatching
	}

	baseNode := pathA[commonA]
	if baseNode < 0 || baseNode >= len(e.base) || e.base[baseNode] == noVertex {
		return noVertex, nil, nil, ErrInvalidMatching
	}

	left := append([]int(nil), pathA[:commonA+1]...)
	right := append([]int(nil), pathB[:commonB]...)

	return e.base[baseNode], left, right, nil
}

// allocateBlossomNode reserves and initializes one contracted blossom node.
// It records deterministic child order, base vertex, boundary edge, member vertices,
// and zero initial blossom dual.
//
// Complexity:
//   - Time O(k), Space O(k).
func (e *blossomEngine) allocateBlossomNode(base int, left []int, right []int, edgeID int) (int, error) {
	if base < 0 || base >= e.problem.n {
		return noNode, ErrInvalidMatching
	}
	if edgeID < 0 || edgeID >= len(e.edges) {
		return noNode, ErrInvalidMatching
	}
	if e.nextNode >= len(e.active) {
		return noNode, ErrInvalidMatching
	}

	newNode := e.nextNode
	e.nextNode++

	cycle := make([]int, 0, len(left)+len(right))
	cycle = append(cycle, left...)

	for index := len(right) - 1; index >= 0; index-- {
		cycle = append(cycle, right[index])
	}

	if len(cycle) < 3 || (len(cycle)&1) == 0 {
		return noNode, ErrInvalidMatching
	}

	members := make([]int, 0, e.problem.n)

	for _, child := range cycle {
		if child < 0 || child >= len(e.active) || !e.active[child] {
			return noNode, ErrInvalidMatching
		}

		members = append(members, e.members[child]...)
	}

	cycleSteps, err := e.buildBlossomCycleSteps(cycle, edgeID)
	if err != nil {
		return noNode, err
	}

	e.base[newNode] = base
	e.children[newNode] = cycle
	e.cycles[newNode] = cycleSteps
	e.members[newNode] = members
	e.dual[newNode] = 0

	return newNode, nil
}

// contractCycleIntoBlossom makes a newly allocated blossom the active owner of its members.
// Child top-level nodes become inactive; every original member vertex is remapped through inBlossom.
//
// Complexity:
//   - Time O(k), Space O(1).
func (e *blossomEngine) contractCycleIntoBlossom(node int) error {
	if node < e.problem.n || node >= len(e.active) {
		return ErrInvalidMatching
	}
	if len(e.children[node]) == 0 || len(e.members[node]) == 0 {
		return ErrInvalidMatching
	}

	for _, child := range e.children[node] {
		if child < 0 || child >= len(e.active) || !e.active[child] {
			return ErrInvalidMatching
		}

		e.active[child] = false
	}

	for _, vertex := range e.members[node] {
		if vertex < 0 || vertex >= e.problem.n {
			return ErrInvalidMatching
		}

		e.inBlossom[vertex] = node
	}

	e.active[node] = true

	return nil
}

// expand expands a zero-dual inner blossom while preserving alternating-forest semantics.
// It is invoked during weighted search when an inner contracted blossom reaches zero dual
// and must be reopened so tight-edge scanning can continue on its children.
//
// Implementation:
//   - Stage 1: Validate active contracted blossom shape.
//   - Stage 2: Ignore non-inner or positive-dual blossoms.
//   - Stage 3: Expand children through expandInnerBlossomInForest.
//   - Stage 4: Deactivate the contracted node and record expansion telemetry.
//
// Behavior highlights:
//   - Does not clear forest labels blindly.
//   - Preserves the represented matching cost.
//   - Reactivates child nodes in deterministic cycle order.
//   - Keeps inBlossom ownership consistent for original vertices.
//
// Inputs:
//   - node: active contracted blossom node.
//
// Returns:
//   - error: nil when expansion is not needed or succeeds.
//
// Errors:
//   - ErrInvalidMatching for invalid node, missing cycle metadata, corrupted children,
//     or broken ownership restoration.
//
// Determinism:
//   - Children are restored in cycle order.
//
// Complexity:
//   - Time O(c + members), Space O(1).
//
// Notes:
//   - Structural cleanup remains in restoreChildrenAsTopLevel.
//   - Search expansion must preserve enough labels for continued alternating-tree processing.
//
// AI-Hints:
//   - Do not call restoreChildrenAsTopLevel directly from weighted search expansion.
//   - Do not reset all child labels to blossomUnlabeled here.
func (e *blossomEngine) expand(node int) error {
	if node < e.problem.n || node >= len(e.active) {
		return ErrInvalidMatching
	}
	if !e.active[node] {
		return ErrInvalidMatching
	}
	if len(e.children[node]) == 0 || len(e.cycles[node]) == 0 {
		return ErrInvalidMatching
	}

	if e.label[node] != blossomInner || e.dual[node] > e.eps {
		return nil
	}

	if err := e.expandInnerBlossomInForest(node); err != nil {
		return err
	}

	e.active[node] = false
	e.stats.Expansions++

	return nil
}

// expandInnerBlossomInForest reopens a zero-dual inner blossom inside the active forest.
// The expanded child sequence must continue to represent the same alternating structure
// so later scans and augmentations remain mathematically meaningful.
//
// Implementation:
//   - Stage 1: Validate the contracted node, label, and cycle metadata.
//   - Stage 2: Reactivate children in stored cycle order.
//   - Stage 3: Restore original-vertex ownership for every child member.
//   - Stage 4: Mark the base child as the continuation point of the parent forest edge.
//   - Stage 5: Assign deterministic labels to other children by walking the odd cycle.
//   - Stage 6: Clear the contracted node storage without losing child state.
//
// Behavior highlights:
//   - Preserves matching cost.
//   - Preserves tree root identity.
//   - Does not mutate mate[].
//   - Does not reorder children.
//
// Inputs:
//   - node: active inner contracted blossom whose dual is zero within eps.
//
// Returns:
//   - error: nil after forest-preserving expansion.
//
// Errors:
//   - ErrInvalidMatching for invalid labels, missing base child, malformed cycle,
//     or ownership corruption.
//
// Determinism:
//   - Cycle order controls all child restoration decisions.
//
// Complexity:
//   - Time O(c + members), Space O(1), where c is cycle length.
//
// Notes:
//   - This is a conservative expansion policy; tests must cover zero-dual inner expansion.
//
// AI-Hints:
//   - Do not collapse this into restoreChildrenAsTopLevel.
//   - Do not reset labelEdge/treeRoot for every child.
func (e *blossomEngine) expandInnerBlossomInForest(node int) error {
	if node < e.problem.n || node >= len(e.cycles) {
		return ErrInvalidMatching
	}
	if e.label[node] != blossomInner {
		return ErrInvalidMatching
	}
	if e.dual[node] > e.eps {
		return ErrInvalidMatching
	}

	baseChildIndex, err := e.baseChildIndex(node)
	if err != nil {
		return err
	}

	root := e.treeRoot[node]
	parent := e.parent[node]
	labelEdge := e.labelEdge[node]

	for _, step := range e.cycles[node] {
		child := step.node
		if child < 0 || child >= len(e.active) {
			return ErrInvalidMatching
		}

		e.active[child] = true

		for _, vertex := range e.members[child] {
			if vertex < 0 || vertex >= e.problem.n {
				return ErrInvalidMatching
			}

			e.inBlossom[vertex] = child
		}

		e.treeRoot[child] = root
	}

	for offset := 0; offset < len(e.cycles[node]); offset++ {
		index := (baseChildIndex + offset) % len(e.cycles[node])
		child := e.cycles[node][index].node

		if offset == 0 {
			e.label[child] = blossomInner
			e.parent[child] = parent
			e.labelEdge[child] = labelEdge
			continue
		}

		if (offset & 1) == 1 {
			e.label[child] = blossomOuter
			e.parent[child] = e.cycles[node][(index-1+len(e.cycles[node]))%len(e.cycles[node])].node
			e.labelEdge[child] = e.cycles[node][(index-1+len(e.cycles[node]))%len(e.cycles[node])].edgeToNext
			e.queue = append(e.queue, child)
			continue
		}

		e.label[child] = blossomInner
		e.parent[child] = e.cycles[node][(index-1+len(e.cycles[node]))%len(e.cycles[node])].node
		e.labelEdge[child] = e.cycles[node][(index-1+len(e.cycles[node]))%len(e.cycles[node])].edgeToNext
	}

	e.children[node] = nil
	e.cycles[node] = nil
	e.members[node] = nil
	e.base[node] = noVertex
	e.parent[node] = noNode
	e.label[node] = blossomUnlabeled
	e.labelEdge[node] = noEdge
	e.treeRoot[node] = noNode

	return nil
}

// restoreChildrenAsTopLevel reverses one blossom contraction.
// It reactivates child nodes, restores inBlossom ownership for original vertices,
// clears child forest labels, and releases the contracted node storage.
//
// Complexity:
//   - Time O(k), Space O(1).
func (e *blossomEngine) restoreChildrenAsTopLevel(node int) error {
	if node < e.problem.n || node >= len(e.children) {
		return ErrInvalidMatching
	}

	for _, child := range e.children[node] {
		if child < 0 || child >= len(e.active) {
			return ErrInvalidMatching
		}

		e.active[child] = true

		for _, vertex := range e.members[child] {
			if vertex < 0 || vertex >= e.problem.n {
				return ErrInvalidMatching
			}

			e.inBlossom[vertex] = child
		}

		e.label[child] = blossomUnlabeled
		e.labelEdge[child] = noEdge
		e.parent[child] = noNode
		e.treeRoot[child] = noNode
	}

	e.members[node] = nil
	e.children[node] = nil
	e.cycles[node] = nil
	e.base[node] = noVertex

	return nil
}

// edgeBetweenAdjacentTreeNodes returns the label edge between parent-adjacent forest nodes.
// It is used while building a contracted blossom cycle from two paths to their common base.
//
// Implementation:
//   - Stage 1: Check whether left is the child of right.
//   - Stage 2: Check whether right is the child of left.
//   - Stage 3: Return the child label edge when exactly one parent relation matches.
//
// Behavior highlights:
//   - Does not mutate engine state.
//   - Rejects non-adjacent nodes.
//   - Uses forest parent links, not graph adjacency scans.
//
// Inputs:
//   - left: first active top-level node.
//   - right: second active top-level node.
//
// Returns:
//   - int: dense edge ID connecting the two forest-adjacent nodes.
//   - error: nil when the forest relation is valid.
//
// Errors:
//   - ErrInvalidMatching for invalid node IDs, non-adjacent nodes, or missing label edge.
//
// Determinism:
//   - Fixed parent-direction checks.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - The closing shrink edge is handled separately; this helper handles only parent-chain edges.
//
// AI-Hints:
//   - Do not search all dense edges here; parent links already define the alternating-tree edge.
func (e *blossomEngine) edgeBetweenAdjacentTreeNodes(left int, right int) (int, error) {
	if left < 0 || left >= len(e.parent) || right < 0 || right >= len(e.parent) {
		return noEdge, ErrInvalidMatching
	}

	if e.parent[left] == right {
		if e.labelEdge[left] == noEdge {
			return noEdge, ErrInvalidMatching
		}

		return e.labelEdge[left], nil
	}

	if e.parent[right] == left {
		if e.labelEdge[right] == noEdge {
			return noEdge, ErrInvalidMatching
		}

		return e.labelEdge[right], nil
	}

	return noEdge, ErrInvalidMatching
}

// buildBlossomCycleSteps builds ordered edge-to-next metadata for one blossom cycle.
// The resulting slice has the same length as cycle, and step i connects cycle[i]
// to cycle[(i+1)%len(cycle)].
//
// Implementation:
//   - Stage 1: Validate odd cycle length.
//   - Stage 2: Resolve parent-chain edges for adjacent cycle nodes.
//   - Stage 3: Use closingEdgeID for the final cycle edge.
//   - Stage 4: Orient every dense edge by child-node ownership.
//
// Behavior highlights:
//   - Stores enough metadata for path lifting.
//   - Does not mutate engine state.
//   - Rejects incomplete cycle edge descriptions immediately.
//
// Inputs:
//   - cycle: active child nodes in deterministic blossom cycle order.
//   - closingEdgeID: tight edge that closes the odd cycle.
//
// Returns:
//   - []blossomCycleStep: ordered cycle metadata.
//   - error: nil when every cycle edge is valid.
//
// Errors:
//   - ErrInvalidMatching for malformed cycle shape, invalid child nodes,
//     invalid edge IDs, or endpoint ownership mismatch.
//
// Determinism:
//   - Fixed cycle-index scan.
//   - Closing edge is always the last step.
//
// Complexity:
//   - Time O(c * m), where c is cycle length and m is average child membership.
//   - Space O(c).
//
// Notes:
//   - This replaces the invalid single-edge childEdges model.
//
// AI-Hints:
//   - Do not store only closingEdgeID.
//   - Do not allow len(steps) != len(cycle).
func (e *blossomEngine) buildBlossomCycleSteps(cycle []int, closingEdgeID int) ([]blossomCycleStep, error) {
	if len(cycle) < 3 || (len(cycle)&1) == 0 {
		return nil, ErrInvalidMatching
	}

	steps := make([]blossomCycleStep, len(cycle))

	for index, node := range cycle {
		if node < 0 || node >= len(e.active) || !e.active[node] {
			return nil, ErrInvalidMatching
		}

		nextIndex := index + 1
		if nextIndex == len(cycle) {
			nextIndex = 0
		}

		nextNode := cycle[nextIndex]

		edgeID := closingEdgeID
		if index+1 < len(cycle) {
			var err error
			edgeID, err = e.edgeBetweenAdjacentTreeNodes(node, nextNode)
			if err != nil {
				return nil, err
			}
		}

		vertexToNext, nextVertex, err := e.orientEdgeForNodes(edgeID, node, nextNode)
		if err != nil {
			return nil, err
		}

		steps[index] = blossomCycleStep{
			node:         node,
			edgeToNext:   edgeID,
			vertexToNext: vertexToNext,
			nextVertex:   nextVertex,
		}
	}

	return steps, nil
}

// baseChildIndex returns the child cycle index that owns the blossom base vertex.
//
// Complexity:
//   - Time O(c*m), Space O(1).
func (e *blossomEngine) baseChildIndex(node int) (int, error) {
	if node < e.problem.n || node >= len(e.cycles) {
		return -1, ErrInvalidMatching
	}

	base := e.base[node]
	if base < 0 || base >= e.problem.n {
		return -1, ErrInvalidMatching
	}

	for index, step := range e.cycles[node] {
		if e.nodeContainsVertex(step.node, base) {
			return index, nil
		}
	}

	return -1, ErrInvalidMatching
}
