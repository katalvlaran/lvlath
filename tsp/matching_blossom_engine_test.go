package tsp

import (
	"math"
	"testing"
)

func TestBlossomShrinkRootCycleMakesContractedNodeForestRoot(t *testing.T) {
	problem := internalSeededMatchingProblem(4, 2)

	engine, err := newBlossomEngine(problem, blossomOptions{Eps: DefaultEps})
	if err != nil {
		t.Fatalf("newBlossomEngine: %v", err)
	}

	matched := internalEdgeID(t, engine, 1, 2)
	if err = engine.setMatchedEdge(matched); err != nil {
		t.Fatalf("seed matched edge: %v", err)
	}

	engine.resetForest()
	engine.assignOuterRoot(0)
	engine.assignOuterRoot(3)

	growEvent := blossomEvent{
		kind:    eventGrow,
		edge:    internalEdgeID(t, engine, 3, 2),
		a:       3,
		b:       2,
		aVertex: 3,
		bVertex: 2,
	}
	if err = engine.grow(growEvent); err != nil {
		t.Fatalf("grow: %v", err)
	}

	newNode := engine.nextNode

	shrinkEvent := blossomEvent{
		kind:    eventShrink,
		edge:    internalEdgeID(t, engine, 3, 1),
		a:       3,
		b:       1,
		aVertex: 3,
		bVertex: 1,
	}
	if err = engine.shrink(shrinkEvent); err != nil {
		t.Fatalf("shrink: %v", err)
	}

	if engine.parent[newNode] != noNode {
		t.Fatalf("parent[%d]=%d, want noNode", newNode, engine.parent[newNode])
	}
	if engine.labelEdge[newNode] != noEdge {
		t.Fatalf("labelEdge[%d]=%d, want noEdge", newNode, engine.labelEdge[newNode])
	}
	if engine.treeRoot[newNode] != newNode {
		t.Fatalf("treeRoot[%d]=%d, want %d", newNode, engine.treeRoot[newNode], newNode)
	}

	path, err := engine.pathToRoot(newNode)
	if err != nil {
		t.Fatalf("pathToRoot(new blossom): %v", err)
	}
	if len(path) != 0 {
		t.Fatalf("root blossom path length got %d want 0: %+v", len(path), path)
	}
}

func TestFindBlossomBaseAndCycleSeparatesBaseVertexAndBaseNode(t *testing.T) {
	problem := internalSeededMatchingProblem(6, 6100)

	engine, err := newBlossomEngine(problem, blossomOptions{Eps: DefaultEps})
	if err != nil {
		t.Fatalf("newBlossomEngine: %v", err)
	}

	baseVertex := 3
	baseNode := engine.nextNode
	engine.nextNode++

	engine.base[baseNode] = baseVertex
	engine.members[baseNode] = []int{baseVertex}
	engine.active[baseNode] = true
	engine.active[baseVertex] = false
	engine.inBlossom[baseVertex] = baseNode

	engine.label[baseNode] = blossomOuter
	engine.parent[baseNode] = noNode
	engine.labelEdge[baseNode] = noEdge
	engine.treeRoot[baseNode] = baseNode

	left := 0
	right := 1

	engine.label[left] = blossomOuter
	engine.parent[left] = baseNode
	engine.labelEdge[left] = internalEdgeID(t, engine, left, baseVertex)
	engine.treeRoot[left] = baseNode

	engine.label[right] = blossomOuter
	engine.parent[right] = baseNode
	engine.labelEdge[right] = internalEdgeID(t, engine, right, baseVertex)
	engine.treeRoot[right] = baseNode

	gotBaseVertex, gotBaseNode, pathA, pathB, err := engine.findBlossomBaseAndCycle(left, right)
	if err != nil {
		t.Fatalf("findBlossomBaseAndCycle: %v", err)
	}
	if gotBaseVertex != baseVertex {
		t.Fatalf("base vertex got %d want %d", gotBaseVertex, baseVertex)
	}
	if gotBaseNode != baseNode {
		t.Fatalf("base node got %d want %d", gotBaseNode, baseNode)
	}
	if len(pathA) == 0 || pathA[len(pathA)-1] != baseNode {
		t.Fatalf("pathA does not end at base node: %v", pathA)
	}
	if len(pathB) == 0 {
		t.Fatalf("pathB must contain right arm before base")
	}
}

func TestBlossomDualOuterBlossomPreservesInternalSlack(t *testing.T) {
	problem := internalSeededMatchingProblem(4, 2)

	engine, err := newBlossomEngine(problem, blossomOptions{Eps: DefaultEps})
	if err != nil {
		t.Fatalf("newBlossomEngine: %v", err)
	}

	matched := internalEdgeID(t, engine, 1, 2)
	if err = engine.setMatchedEdge(matched); err != nil {
		t.Fatalf("seed matched edge: %v", err)
	}

	engine.resetForest()
	engine.assignOuterRoot(0)
	engine.assignOuterRoot(3)

	growEvent := blossomEvent{
		kind:    eventGrow,
		edge:    internalEdgeID(t, engine, 3, 2),
		a:       3,
		b:       2,
		aVertex: 3,
		bVertex: 2,
	}
	if err = engine.grow(growEvent); err != nil {
		t.Fatalf("grow: %v", err)
	}

	newNode := engine.nextNode

	shrinkEvent := blossomEvent{
		kind:    eventShrink,
		edge:    internalEdgeID(t, engine, 3, 1),
		a:       3,
		b:       1,
		aVertex: 3,
		bVertex: 1,
	}
	if err = engine.shrink(shrinkEvent); err != nil {
		t.Fatalf("shrink: %v", err)
	}

	steps, err := engine.cycleSteps(newNode)
	if err != nil {
		t.Fatalf("cycleSteps: %v", err)
	}

	internalEdge := steps[0].edgeToNext
	before := engine.slack(internalEdge)

	if err = engine.applyDelta(blossomDelta{
		kind:  deltaJoinOuterTrees,
		value: 3,
		edge:  noEdge,
		node:  noNode,
	}); err != nil {
		t.Fatalf("applyDelta: %v", err)
	}

	after := engine.slack(internalEdge)
	if math.Abs(after-before) > DefaultEps {
		t.Fatalf("internal slack changed: before %.12f after %.12f", before, after)
	}
}

func TestBlossomDualInnerExpansionUsesHalfBlossomDual(t *testing.T) {
	problem := internalSeededMatchingProblem(4, 2)

	engine, err := newBlossomEngine(problem, blossomOptions{Eps: DefaultEps})
	if err != nil {
		t.Fatalf("newBlossomEngine: %v", err)
	}

	node := engine.nextNode
	engine.nextNode++

	engine.base[node] = 0
	engine.members[node] = []int{0, 1, 2}
	engine.cycles[node] = []blossomCycleStep{
		{node: 0, edgeToNext: internalEdgeID(t, engine, 0, 1), vertexToNext: 0, nextVertex: 1},
		{node: 1, edgeToNext: internalEdgeID(t, engine, 1, 2), vertexToNext: 1, nextVertex: 2},
		{node: 2, edgeToNext: internalEdgeID(t, engine, 2, 0), vertexToNext: 2, nextVertex: 0},
	}
	engine.active[node] = true
	engine.label[node] = blossomInner
	engine.dual[node] = 10

	best := blossomDelta{
		kind:  deltaNone,
		value: math.Inf(1),
		edge:  noEdge,
		node:  noNode,
	}

	if err = engine.considerExpandDeltas(&best); err != nil {
		t.Fatalf("considerExpandDeltas: %v", err)
	}

	if best.kind != deltaExpandInnerBlossom {
		t.Fatalf("kind got %v want deltaExpandInnerBlossom", best.kind)
	}
	if math.Abs(best.value-5) > DefaultEps {
		t.Fatalf("delta got %.12f want 5", best.value)
	}
}

func TestBlossomExpansionDeltaIsAppliedImmediately(t *testing.T) {
	problem := internalSeededMatchingProblem(4, 2)

	engine, err := newBlossomEngine(problem, blossomOptions{Eps: DefaultEps})
	if err != nil {
		t.Fatalf("newBlossomEngine: %v", err)
	}

	node := engine.nextNode
	engine.nextNode++

	engine.base[node] = 0
	engine.members[node] = []int{0, 1, 2}
	engine.cycles[node] = []blossomCycleStep{
		{node: 0, edgeToNext: internalEdgeID(t, engine, 0, 1), vertexToNext: 0, nextVertex: 1},
		{node: 1, edgeToNext: internalEdgeID(t, engine, 1, 2), vertexToNext: 1, nextVertex: 2},
		{node: 2, edgeToNext: internalEdgeID(t, engine, 2, 0), vertexToNext: 2, nextVertex: 0},
	}

	// Parent edge enters base child directly in this smoke test.
	parent := 3
	parentEdge := internalEdgeID(t, engine, parent, 0)

	engine.active[parent] = true
	engine.label[parent] = blossomOuter
	engine.treeRoot[parent] = parent

	engine.active[node] = true
	engine.label[node] = blossomInner
	engine.treeRoot[node] = parent
	engine.parent[node] = parent
	engine.labelEdge[node] = parentEdge
	engine.dual[node] = 2

	for _, step := range engine.cycles[node] {
		engine.active[step.node] = false
		for _, vertex := range engine.members[step.node] {
			engine.inBlossom[vertex] = node
		}
	}

	delta := blossomDelta{
		kind:  deltaExpandInnerBlossom,
		value: 1,
		edge:  noEdge,
		node:  node,
	}

	if err = engine.applyDelta(delta); err != nil {
		t.Fatalf("applyDelta: %v", err)
	}
	if err = engine.expand(delta.node); err != nil {
		t.Fatalf("expand: %v", err)
	}

	if engine.active[node] {
		t.Fatalf("expanded blossom node remains active")
	}
	if len(engine.cycles[node]) != 0 {
		t.Fatalf("expanded blossom cycle metadata not cleared")
	}
	if engine.dual[node] != 0 {
		t.Fatalf("expanded blossom dual got %.12f want 0", engine.dual[node])
	}
}

func TestBlossomExpandInnerUsesEntryChildForParentEdge(t *testing.T) {
	problem := internalSeededMatchingProblem(6, 23)

	engine, err := newBlossomEngine(problem, blossomOptions{Eps: DefaultEps})
	if err != nil {
		t.Fatalf("newBlossomEngine: %v", err)
	}

	node := engine.nextNode
	engine.nextNode++

	engine.base[node] = 0
	engine.members[node] = []int{0, 1, 2}
	engine.cycles[node] = []blossomCycleStep{
		{node: 0, edgeToNext: internalEdgeID(t, engine, 0, 1), vertexToNext: 0, nextVertex: 1},
		{node: 1, edgeToNext: internalEdgeID(t, engine, 1, 2), vertexToNext: 1, nextVertex: 2},
		{node: 2, edgeToNext: internalEdgeID(t, engine, 2, 0), vertexToNext: 2, nextVertex: 0},
	}

	// The valid entry->base expansion path is 2 -> 1 -> 0.
	// Therefore 2-1 must be matched and 1-0 must be unmatched.
	if err = engine.setMatchedEdge(internalEdgeID(t, engine, 1, 2)); err != nil {
		t.Fatalf("set internal matched edge 1-2: %v", err)
	}

	for _, step := range engine.cycles[node] {
		engine.active[step.node] = false
		for _, vertex := range engine.members[step.node] {
			engine.inBlossom[vertex] = node
		}
	}

	parent := 3
	parentEdge := internalEdgeID(t, engine, parent, 2)

	engine.active[parent] = true
	engine.label[parent] = blossomOuter
	engine.treeRoot[parent] = parent

	engine.active[node] = true
	engine.label[node] = blossomInner
	engine.parent[node] = parent
	engine.labelEdge[node] = parentEdge
	engine.treeRoot[node] = parent
	engine.dual[node] = 0

	if err = engine.expandInnerBlossomInForest(node); err != nil {
		t.Fatalf("expandInnerBlossomInForest: %v", err)
	}

	entryChild := 2
	if engine.parent[entryChild] != parent {
		t.Fatalf("entry child parent got %d want %d", engine.parent[entryChild], parent)
	}
	if engine.labelEdge[entryChild] != parentEdge {
		t.Fatalf("entry child labelEdge got %d want %d", engine.labelEdge[entryChild], parentEdge)
	}
	if engine.label[entryChild] != blossomInner {
		t.Fatalf("entry child label got %v want blossomInner", engine.label[entryChild])
	}
}

func TestBlossomExpandInnerReparentsOuterContinuationChild(t *testing.T) {
	problem := internalSeededMatchingProblem(6, 23)

	engine, err := newBlossomEngine(problem, blossomOptions{Eps: DefaultEps})
	if err != nil {
		t.Fatalf("newBlossomEngine: %v", err)
	}

	node := engine.nextNode
	engine.nextNode++

	engine.base[node] = 0
	engine.members[node] = []int{0, 1, 2}
	engine.cycles[node] = []blossomCycleStep{
		{node: 0, edgeToNext: internalEdgeID(t, engine, 0, 1), vertexToNext: 0, nextVertex: 1},
		{node: 1, edgeToNext: internalEdgeID(t, engine, 1, 2), vertexToNext: 1, nextVertex: 2},
		{node: 2, edgeToNext: internalEdgeID(t, engine, 2, 0), vertexToNext: 2, nextVertex: 0},
	}

	// Expansion path 2 -> 1 -> 0 needs 2-1 matched.
	if err = engine.setMatchedEdge(internalEdgeID(t, engine, 1, 2)); err != nil {
		t.Fatalf("set internal matched edge 1-2: %v", err)
	}

	for _, step := range engine.cycles[node] {
		engine.active[step.node] = false
		for _, vertex := range engine.members[step.node] {
			engine.inBlossom[vertex] = node
		}
	}

	parent := 3
	outerChild := 4
	parentEdge := internalEdgeID(t, engine, parent, 2)
	outerEdge := internalEdgeID(t, engine, 0, outerChild)

	if err = engine.setMatchedEdge(outerEdge); err != nil {
		t.Fatalf("setMatchedEdge outerEdge: %v", err)
	}

	engine.active[parent] = true
	engine.label[parent] = blossomOuter
	engine.treeRoot[parent] = parent

	engine.active[node] = true
	engine.label[node] = blossomInner
	engine.parent[node] = parent
	engine.labelEdge[node] = parentEdge
	engine.treeRoot[node] = parent
	engine.dual[node] = 0

	engine.active[outerChild] = true
	engine.label[outerChild] = blossomOuter
	engine.parent[outerChild] = node
	engine.labelEdge[outerChild] = outerEdge
	engine.treeRoot[outerChild] = parent

	if err = engine.expandInnerBlossomInForest(node); err != nil {
		t.Fatalf("expandInnerBlossomInForest: %v", err)
	}

	baseChild := 0
	if engine.parent[outerChild] != baseChild {
		t.Fatalf("outer child parent got %d want base child %d", engine.parent[outerChild], baseChild)
	}
	if engine.labelEdge[outerChild] != outerEdge {
		t.Fatalf("outer child labelEdge got %d want %d", engine.labelEdge[outerChild], outerEdge)
	}
	if engine.treeRoot[outerChild] != parent {
		t.Fatalf("outer child treeRoot got %d want %d", engine.treeRoot[outerChild], parent)
	}
}

func TestBlossomRefreshAllocatedBasesUsesCommittedMatching(t *testing.T) {
	problem := internalSeededMatchingProblem(6, 23)

	engine, err := newBlossomEngine(problem, blossomOptions{Eps: DefaultEps})
	if err != nil {
		t.Fatalf("newBlossomEngine: %v", err)
	}

	node := engine.nextNode
	engine.nextNode++

	engine.base[node] = 0
	engine.members[node] = []int{0, 1, 2}
	engine.cycles[node] = []blossomCycleStep{
		{node: 0, edgeToNext: internalEdgeID(t, engine, 0, 1), vertexToNext: 0, nextVertex: 1},
		{node: 1, edgeToNext: internalEdgeID(t, engine, 1, 2), vertexToNext: 1, nextVertex: 2},
		{node: 2, edgeToNext: internalEdgeID(t, engine, 2, 0), vertexToNext: 2, nextVertex: 0},
	}

	engine.active[node] = true

	for _, step := range engine.cycles[node] {
		engine.active[step.node] = false
		for _, vertex := range engine.members[step.node] {
			engine.inBlossom[vertex] = node
		}
	}

	if err = engine.setMatchedEdge(internalEdgeID(t, engine, 0, 1)); err != nil {
		t.Fatalf("set internal matched edge: %v", err)
	}

	if err = engine.refreshAllocatedBlossomBases(); err != nil {
		t.Fatalf("refreshAllocatedBlossomBases: %v", err)
	}

	if engine.base[node] != 2 {
		t.Fatalf("base got %d want 2", engine.base[node])
	}
}

func TestBlossomShrinkRedirectsForestDescendantsFromContractedChildren(t *testing.T) {
	problem := internalSeededMatchingProblem(6, 23)

	engine, err := newBlossomEngine(problem, blossomOptions{Eps: DefaultEps})
	if err != nil {
		t.Fatalf("newBlossomEngine: %v", err)
	}

	newNode := engine.nextNode
	engine.nextNode++

	engine.base[newNode] = 3
	engine.members[newNode] = []int{3, 4, 5}
	engine.cycles[newNode] = []blossomCycleStep{
		{node: 3, edgeToNext: internalEdgeID(t, engine, 3, 5), vertexToNext: 3, nextVertex: 5},
		{node: 5, edgeToNext: internalEdgeID(t, engine, 5, 4), vertexToNext: 5, nextVertex: 4},
		{node: 4, edgeToNext: internalEdgeID(t, engine, 4, 3), vertexToNext: 4, nextVertex: 3},
	}

	// Active descendants outside the soon-to-be-contracted blossom.
	engine.active[1] = true
	engine.label[1] = blossomOuter
	engine.parent[1] = 2
	engine.labelEdge[1] = internalEdgeID(t, engine, 1, 2)
	engine.treeRoot[1] = 3

	engine.active[2] = true
	engine.label[2] = blossomInner
	engine.parent[2] = 4
	engine.labelEdge[2] = internalEdgeID(t, engine, 2, 4)
	engine.treeRoot[2] = 3

	// Old cycle children are active before contraction.
	for _, child := range []int{3, 4, 5} {
		engine.active[child] = true
	}

	if err = engine.contractCycleIntoBlossom(newNode); err != nil {
		t.Fatalf("contractCycleIntoBlossom: %v", err)
	}

	engine.label[newNode] = blossomOuter
	engine.parent[newNode] = noNode
	engine.labelEdge[newNode] = noEdge
	engine.treeRoot[newNode] = newNode

	if err = engine.redirectForestThroughContractedBlossom(newNode); err != nil {
		t.Fatalf("redirectForestThroughContractedBlossom: %v", err)
	}

	if engine.parent[2] != newNode {
		t.Fatalf("parent[2] got %d want contracted blossom %d", engine.parent[2], newNode)
	}
	if engine.treeRoot[1] != newNode {
		t.Fatalf("treeRoot[1] got %d want %d", engine.treeRoot[1], newNode)
	}
	if engine.treeRoot[2] != newNode {
		t.Fatalf("treeRoot[2] got %d want %d", engine.treeRoot[2], newNode)
	}

	if _, err = engine.pathToRoot(1); err != nil {
		t.Fatalf("pathToRoot after redirect: %v", err)
	}
}

func TestBlossomApplyDeltaDoesNotClampPositiveSubToleranceMovement(t *testing.T) {
	problem := internalNearTieMatchingProblem(4, 80)

	engine, err := newBlossomEngine(problem, blossomOptions{Eps: DefaultEps})
	if err != nil {
		t.Fatalf("newBlossomEngine: %v", err)
	}

	engine.active[0] = true
	engine.label[0] = blossomOuter

	before := engine.dual[0]
	delta := engine.dualTolerance() / 2

	if err = engine.applyDelta(blossomDelta{
		kind:  deltaGrowToUnlabeled,
		value: delta,
	}); err != nil {
		t.Fatalf("applyDelta: %v", err)
	}

	if engine.dual[0] == before {
		t.Fatalf("positive sub-tolerance delta was clamped: before %.17g after %.17g delta %.17g tol %.17g",
			before, engine.dual[0], delta, engine.dualTolerance())
	}
}

func TestBlossomImproveDeltaUsesNarrowSelectionTolerance(t *testing.T) {
	problem := internalNearTieMatchingProblem(4, 86)

	engine, err := newBlossomEngine(problem, blossomOptions{Eps: DefaultEps})
	if err != nil {
		t.Fatalf("newBlossomEngine: %v", err)
	}

	wide := engine.dualTolerance() / 2
	narrow := engine.dualSelectionTolerance()

	if !(wide > narrow) {
		t.Fatalf("test requires wide tolerance > narrow tolerance, wide=%g narrow=%g", wide, narrow)
	}

	best := blossomDelta{
		kind:  deltaGrowToUnlabeled,
		value: wide,
		edge:  1,
		node:  1,
	}

	better := blossomDelta{
		kind:  deltaGrowToUnlabeled,
		value: narrow,
		edge:  99,
		node:  2,
	}

	engine.improveDelta(&best, better)

	if best.edge != better.edge {
		t.Fatalf("improveDelta kept larger candidate: got edge=%d value=%.17g want edge=%d value=%.17g wideTol=%.17g narrowTol=%.17g",
			best.edge, best.value, better.edge, better.value, wide, narrow)
	}
}
