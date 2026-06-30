// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

package tsp

import (
	"math"
	"testing"
)

func TestBlossomSetAndClearMatchedEdgeMaintainSymmetry(t *testing.T) {
	problem := internalSeededMatchingProblem(4, 20)

	engine, err := newBlossomEngine(problem, blossomOptions{Eps: DefaultEps})
	if err != nil {
		t.Fatalf("newBlossomEngine: %v", err)
	}

	edgeID := internalEdgeID(t, engine, 0, 1)

	if err := engine.setMatchedEdge(edgeID); err != nil {
		t.Fatalf("setMatchedEdge: %v", err)
	}
	if engine.mate[0] != 1 || engine.mate[1] != 0 {
		t.Fatalf("mate symmetry broken after set: mate=%v", engine.mate)
	}
	if engine.mateEdge[0] != edgeID || engine.mateEdge[1] != edgeID {
		t.Fatalf("mateEdge symmetry broken after set: mateEdge=%v", engine.mateEdge)
	}

	if err := engine.clearMatchedEdge(edgeID); err != nil {
		t.Fatalf("clearMatchedEdge: %v", err)
	}
	if engine.mate[0] != noVertex || engine.mate[1] != noVertex {
		t.Fatalf("mate not cleared: mate=%v", engine.mate)
	}
	if engine.mateEdge[0] != noEdge || engine.mateEdge[1] != noEdge {
		t.Fatalf("mateEdge not cleared: mateEdge=%v", engine.mateEdge)
	}
}

func TestBlossomFlipRejectsNonAlternatingPathBeforeMutation(t *testing.T) {
	problem := internalSeededMatchingProblem(4, 21)

	engine, err := newBlossomEngine(problem, blossomOptions{Eps: DefaultEps})
	if err != nil {
		t.Fatalf("newBlossomEngine: %v", err)
	}

	first := internalEdgeID(t, engine, 0, 1)
	second := internalEdgeID(t, engine, 2, 3)

	beforeMate := append([]int(nil), engine.mate...)
	beforeMateEdge := append([]int(nil), engine.mateEdge...)

	if err := engine.flipAugmentingPath([]int{first, second}); err != ErrInvalidMatching {
		t.Fatalf("non-alternating path got %v", err)
	}

	for vertex := range engine.mate {
		if engine.mate[vertex] != beforeMate[vertex] {
			t.Fatalf("mate mutated at %d: got %d want %d", vertex, engine.mate[vertex], beforeMate[vertex])
		}
		if engine.mateEdge[vertex] != beforeMateEdge[vertex] {
			t.Fatalf("mateEdge mutated at %d: got %d want %d", vertex, engine.mateEdge[vertex], beforeMateEdge[vertex])
		}
	}
}

func TestBlossomFlipAugmentingPathMatchesThreeEdgePath(t *testing.T) {
	problem := internalSeededMatchingProblem(4, 22)

	engine, err := newBlossomEngine(problem, blossomOptions{Eps: DefaultEps})
	if err != nil {
		t.Fatalf("newBlossomEngine: %v", err)
	}

	left := internalEdgeID(t, engine, 0, 1)
	middle := internalEdgeID(t, engine, 1, 2)
	right := internalEdgeID(t, engine, 2, 3)

	if err := engine.setMatchedEdge(middle); err != nil {
		t.Fatalf("seed middle match: %v", err)
	}

	if err := engine.flipAugmentingPath([]int{left, middle, right}); err != nil {
		t.Fatalf("flipAugmentingPath: %v", err)
	}

	if engine.mate[0] != 1 || engine.mate[1] != 0 {
		t.Fatalf("left edge not matched after flip: mate=%v", engine.mate)
	}
	if engine.mate[2] != 3 || engine.mate[3] != 2 {
		t.Fatalf("right edge not matched after flip: mate=%v", engine.mate)
	}
}

func TestBlossomComposeAugmentingPathUsesRootToRootOrientation(t *testing.T) {
	problem := internalSeededMatchingProblem(6, 23)

	engine, err := newBlossomEngine(problem, blossomOptions{Eps: DefaultEps})
	if err != nil {
		t.Fatalf("newBlossomEngine: %v", err)
	}

	left := []blossomPathStep{
		{edge: internalEdgeID(t, engine, 3, 1), fromNode: 3, toNode: 1, fromVertex: 3, toVertex: 1},
		{edge: internalEdgeID(t, engine, 1, 0), fromNode: 1, toNode: 0, fromVertex: 1, toVertex: 0},
	}
	right := []blossomPathStep{
		{edge: internalEdgeID(t, engine, 4, 2), fromNode: 4, toNode: 2, fromVertex: 4, toVertex: 2},
		{edge: internalEdgeID(t, engine, 2, 5), fromNode: 2, toNode: 5, fromVertex: 2, toVertex: 5},
	}
	bridge := blossomPathStep{
		edge:       internalEdgeID(t, engine, 3, 4),
		fromNode:   3,
		toNode:     4,
		fromVertex: 3,
		toVertex:   4,
	}

	path, err := engine.composeAugmentingPath(left, bridge, right)
	if err != nil {
		t.Fatalf("composeAugmentingPath: %v", err)
	}

	wantFrom := []int{0, 1, 3, 4, 2}
	wantTo := []int{1, 3, 4, 2, 5}

	if len(path) != len(wantFrom) {
		t.Fatalf("path length got %d want %d: %+v", len(path), len(wantFrom), path)
	}

	for index := range path {
		if path[index].fromNode != wantFrom[index] || path[index].toNode != wantTo[index] {
			t.Fatalf("step %d got %d->%d want %d->%d",
				index,
				path[index].fromNode,
				path[index].toNode,
				wantFrom[index],
				wantTo[index],
			)
		}
	}
}

func TestBlossomK8Seed05RejectsInvalidLiftCandidateAndMatchesOracle(t *testing.T) {
	problem := internalSeededMatchingProblem(8, 5)

	_, wantCost, err := exactMatchingOracleForTest(problem)
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}

	match, gotCost, stats, err := solveMinimumWeightPerfectMatching(problem, blossomOptions{Eps: DefaultEps})
	if err != nil {
		t.Fatalf("blossom: %v stats=%+v", err, stats)
	}

	if err = verifyPerfectMatching(match); err != nil {
		t.Fatalf("verify blossom: %v match=%v stats=%+v", err, match, stats)
	}

	if math.Abs(gotCost-wantCost) > DefaultEps {
		t.Fatalf("cost got %.12f want %.12f match=%v stats=%+v", gotCost, wantCost, match, stats)
	}
}
