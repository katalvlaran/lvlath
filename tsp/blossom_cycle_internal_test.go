// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

package tsp

import "testing"

func TestBlossomShrinkStoresCycleAsSingleSourceOfTruth(t *testing.T) {
	problem := internalSeededMatchingProblem(4, 10)

	engine, err := newBlossomEngine(problem, blossomOptions{Eps: DefaultEps})
	if err != nil {
		t.Fatalf("newBlossomEngine: %v", err)
	}

	root := 0
	left := 1
	right := 2

	engine.label[root] = blossomOuter
	engine.treeRoot[root] = root
	engine.parent[root] = noNode
	engine.labelEdge[root] = noEdge

	engine.label[left] = blossomOuter
	engine.treeRoot[left] = root
	engine.parent[left] = root
	engine.labelEdge[left] = internalEdgeID(t, engine, left, root)

	engine.label[right] = blossomOuter
	engine.treeRoot[right] = root
	engine.parent[right] = root
	engine.labelEdge[right] = internalEdgeID(t, engine, right, root)

	event := blossomEvent{
		kind:    eventShrink,
		edge:    internalEdgeID(t, engine, left, right),
		a:       left,
		b:       right,
		aVertex: left,
		bVertex: right,
	}

	newNode := engine.nextNode

	if err := engine.shrink(event); err != nil {
		t.Fatalf("shrink: %v", err)
	}

	steps, err := engine.cycleSteps(newNode)
	if err != nil {
		t.Fatalf("cycleSteps: %v", err)
	}

	if len(steps) != 3 {
		t.Fatalf("cycle length got %d want 3", len(steps))
	}

	for index, step := range steps {
		if step.node == noNode {
			t.Fatalf("step[%d].node=noNode", index)
		}
		if step.edgeToNext == noEdge {
			t.Fatalf("step[%d].edgeToNext=noEdge", index)
		}
		if !engine.nodeContainsVertex(step.node, step.vertexToNext) {
			t.Fatalf("step[%d].vertexToNext=%d not owned by node %d", index, step.vertexToNext, step.node)
		}

		next := steps[(index+1)%len(steps)]
		if !engine.nodeContainsVertex(next.node, step.nextVertex) {
			t.Fatalf("step[%d].nextVertex=%d not owned by next node %d", index, step.nextVertex, next.node)
		}
	}
}

func TestBlossomCycleStepsRejectSingletonAndMalformedCycle(t *testing.T) {
	problem := internalSeededMatchingProblem(4, 11)

	engine, err := newBlossomEngine(problem, blossomOptions{Eps: DefaultEps})
	if err != nil {
		t.Fatalf("newBlossomEngine: %v", err)
	}

	if _, err := engine.cycleSteps(0); err != ErrInvalidMatching {
		t.Fatalf("singleton cycleSteps got %v", err)
	}

	node := engine.nextNode
	engine.nextNode++
	engine.cycles[node] = []blossomCycleStep{
		{node: 0, edgeToNext: internalEdgeID(t, engine, 0, 1), vertexToNext: 0, nextVertex: 1},
		{node: 1, edgeToNext: internalEdgeID(t, engine, 1, 0), vertexToNext: 1, nextVertex: 0},
	}

	if _, err := engine.cycleSteps(node); err != ErrInvalidMatching {
		t.Fatalf("even malformed cycle got %v", err)
	}
}

func TestBlossomRestoreChildrenAsTopLevelUsesCycleSteps(t *testing.T) {
	problem := internalSeededMatchingProblem(4, 12)

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

	if err := engine.restoreChildrenAsTopLevel(node); err != nil {
		t.Fatalf("restoreChildrenAsTopLevel: %v", err)
	}

	for vertex := 0; vertex < 3; vertex++ {
		if engine.inBlossom[vertex] != vertex {
			t.Fatalf("inBlossom[%d]=%d want %d", vertex, engine.inBlossom[vertex], vertex)
		}
		if !engine.active[vertex] {
			t.Fatalf("child %d not reactivated", vertex)
		}
	}
	if len(engine.cycles[node]) != 0 {
		t.Fatalf("contracted node cycles not cleared")
	}
}
