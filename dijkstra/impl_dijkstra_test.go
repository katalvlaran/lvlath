// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

package dijkstra_test

import (
	"math"
	"testing"

	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/dijkstra"
)

// AI-HINTS (file):
//   - This file validates the public Dijkstra contract, not internal implementation accidents.
//   - Each test must anchor a specific package law: input admission, shortest-path correctness,
//     determinism, wall semantics, cutoff semantics, or result-surface semantics.
//   - Use errors.Is protocol checks only; never compare error strings.
//   - Keep graph construction explicit and fail fast on every setup error.
//   - Prefer exact assertions when the contract guarantees exact deterministic output.
//   - Regression anchors must stay narrow and mathematically honest.
//   - Do not weaken deterministic predecessor/path checks into unordered comparisons.

const (
	// testVertexSource is the canonical source vertex used in most fixtures.
	testVertexSource = "A"

	// testVertexMiddle is the primary intermediate vertex used in compact routing fixtures.
	testVertexMiddle = "B"

	// testVertexAlternative is the competing or alternative-route vertex.
	testVertexAlternative = "C"

	// testVertexTarget is the primary target vertex used in multi-step fixtures.
	testVertexTarget = "D"

	// testVertexUnreachable is an explicitly known but disconnected vertex.
	testVertexUnreachable = "Z"

	// testVertexLoop is the canonical self-loop vertex.
	testVertexLoop = "X"

	// testVertexMissing is a source identifier intentionally absent from fixtures.
	testVertexMissing = "missing"

	// testWeightOne is the canonical unit edge weight.
	testWeightOne = 1.0

	// testWeightTwo is the canonical weight for two-step competition fixtures.
	testWeightTwo = 2.0

	// testWeightThree is the canonical weight for medium-route fixtures.
	testWeightThree = 3.0

	// testWeightFour is the canonical heavier finite edge weight.
	testWeightFour = 4.0

	// testWeightFive is the canonical wall-threshold boundary or heavy route weight.
	testWeightFive = 5.0

	// testWeightSix is the canonical two-step total under threshold-wall routing.
	testWeightSix = 6.0

	// testWeightTen is the canonical obviously suboptimal or blocked direct edge weight.
	testWeightTen = 10.0
)

// ----------------------------------------------------------------------------
// Validation block: domain admission and public policy propagation.
// ----------------------------------------------------------------------------

// TestDijkstra_NilGraph verifies that the public API rejects a nil graph pointer.
//
// Implementation:
//   - Stage 1: Call Dijkstra with a nil graph and a non-empty source.
//   - Stage 2: Assert ErrNilGraph through the sentinel protocol.
//
// Behavior highlights:
//   - Nil graph rejection is a public input-contract rule.
//   - The test isolates nil-graph behavior by keeping sourceID non-empty.
//
// AI-Hints:
//   - Keep nil-graph and empty-source coverage separate so failure priority stays explicit.
func TestDijkstra_NilGraph(t *testing.T) {
	result, err := dijkstra.Dijkstra(nil, testVertexSource)

	mustNilState(t, result, true, "Dijkstra(nil graph) result")
	mustErrorIs(t, err, dijkstra.ErrNilGraph)
}

// TestDijkstra_EmptySourceID verifies that the public API rejects an empty source identifier.
//
// Implementation:
//   - Stage 1: Construct a weighted graph.
//   - Stage 2: Call Dijkstra with an empty source identifier.
//   - Stage 3: Assert ErrEmptySourceID through the sentinel protocol.
//
// Behavior highlights:
//   - Empty source rejection is a distinct public input-law.
//   - The test isolates source admission by using a non-nil weighted graph.
//
// AI-Hints:
//   - Do not merge this case with source-not-found; they classify different contract failures.
func TestDijkstra_EmptySourceID(t *testing.T) {
	graph := core.NewGraph(core.WithWeighted())

	result, err := dijkstra.Dijkstra(graph, "")

	mustNilState(t, result, true, "Dijkstra(empty source) result")
	mustErrorIs(t, err, dijkstra.ErrEmptySourceID)
}

// TestDijkstra_UnweightedGraph verifies that Dijkstra rejects graphs that do not expose
// weighted edge semantics.
//
// Implementation:
//   - Stage 1: Construct an unweighted graph.
//   - Stage 2: Insert the source vertex explicitly.
//   - Stage 3: Call Dijkstra with that source.
//   - Stage 4: Assert ErrUnweightedGraph through the sentinel protocol.
//
// Behavior highlights:
//   - Weighted graph support is a hard precondition of the package.
//   - The source is present so the test isolates graph-policy admission.
//
// AI-Hints:
//   - Do not weaken this into a generic “bad graph” assertion.
func TestDijkstra_UnweightedGraph(t *testing.T) {
	graph := core.NewGraph()

	if err := graph.AddVertex(testVertexSource); err != nil {
		t.Fatalf("AddVertex(%q) failed: %v", testVertexSource, err)
	}

	result, err := dijkstra.Dijkstra(graph, testVertexSource)

	mustNilState(t, result, true, "Dijkstra(unweighted graph) result")
	mustErrorIs(t, err, dijkstra.ErrUnweightedGraph)
}

// TestDijkstra_SourceNotFound verifies that the public API rejects a source vertex
// absent from the graph domain.
//
// Implementation:
//   - Stage 1: Construct a weighted graph with a different known vertex.
//   - Stage 2: Call Dijkstra with an absent source.
//   - Stage 3: Assert ErrSourceNotFound through the sentinel protocol.
//
// Behavior highlights:
//   - Missing source classification is distinct from empty source and target lookup failures.
//
// AI-Hints:
//   - Keep source-not-found separate from result-level ErrTargetNotFound semantics.
func TestDijkstra_SourceNotFound(t *testing.T) {
	graph := core.NewGraph(core.WithWeighted())

	if err := graph.AddVertex(testVertexAlternative); err != nil {
		t.Fatalf("AddVertex(%q) failed: %v", testVertexAlternative, err)
	}

	result, err := dijkstra.Dijkstra(graph, testVertexMissing)

	mustNilState(t, result, true, "Dijkstra(missing source) result")
	mustErrorIs(t, err, dijkstra.ErrSourceNotFound)
}

// TestDijkstra_NilOption verifies that the public API rejects a nil functional option
// through the explicit sentinel protocol.
//
// Implementation:
//   - Stage 1: Construct a weighted graph with a valid source.
//   - Stage 2: Pass a nil functional option to Dijkstra.
//   - Stage 3: Assert ErrNilOption through the sentinel protocol.
//
// Behavior highlights:
//   - Option admission is explicit and panic-free.
//   - The test validates the public facade, not only applyOptions in isolation.
//
// AI-Hints:
//   - Never replace this with panic-oriented testing.
func TestDijkstra_NilOption(t *testing.T) {
	graph := core.NewGraph(core.WithWeighted())

	if err := graph.AddVertex(testVertexSource); err != nil {
		t.Fatalf("AddVertex(%q) failed: %v", testVertexSource, err)
	}

	var nilOption dijkstra.Option

	result, err := dijkstra.Dijkstra(graph, testVertexSource, nilOption)

	mustNilState(t, result, true, "Dijkstra(nil option) result")
	mustErrorIs(t, err, dijkstra.ErrNilOption)
}

// TestDijkstra_NegativeWeight_PreScan verifies that finite negative edge weights
// are rejected before traversal begins.
//
// Implementation:
//   - Stage 1: Construct a weighted graph with a finite negative edge.
//   - Stage 2: Call Dijkstra from a valid source.
//   - Stage 3: Assert ErrNegativeWeight through the sentinel protocol.
//
// Behavior highlights:
//   - Finite negative weights are invalid for Dijkstra.
//   - The failure must happen through numeric governance, not through path logic.
//
// AI-Hints:
//   - Keep finite negative coverage separate from NaN and -Inf branches.
func TestDijkstra_NegativeWeight_PreScan(t *testing.T) {
	graph := core.NewGraph(core.WithWeighted())

	if _, err := graph.AddEdge(testVertexSource, testVertexMiddle, -testWeightOne); err != nil {
		t.Fatalf("AddEdge(%q,%q,-1) failed: %v", testVertexSource, testVertexMiddle, err)
	}

	result, err := dijkstra.Dijkstra(graph, testVertexSource)

	mustNilState(t, result, true, "Dijkstra(negative weight) result")
	mustErrorIs(t, err, dijkstra.ErrNegativeWeight)
}

// TestDijkstra_NaNWeight_PreScan verifies that NaN edge weights are rejected
// before they can poison frontier ordering or distances.
//
// Implementation:
//   - Stage 1: Construct a weighted graph with one NaN edge.
//   - Stage 2: Call Dijkstra from a valid source.
//   - Stage 3: Assert ErrInvalidWeight through the sentinel protocol.
//
// Behavior highlights:
//   - NaN is invalid numeric input for the package.
//
// AI-Hints:
//   - Keep NaN coverage explicit because it is a live classifier branch.
func TestDijkstra_NaNWeight_PreScan(t *testing.T) {
	graph := core.NewGraph(core.WithWeighted())

	if _, err := graph.AddEdge(testVertexSource, testVertexMiddle, math.NaN()); err != nil {
		t.Fatalf("AddEdge(%q,%q,NaN) failed: %v", testVertexSource, testVertexMiddle, err)
	}

	result, err := dijkstra.Dijkstra(graph, testVertexSource)

	mustNilState(t, result, true, "Dijkstra(NaN weight) result")
	mustErrorIs(t, err, dijkstra.ErrInvalidWeight)
}

// TestDijkstra_NegativeInfinityWeight_PreScan verifies that negative infinity
// edge weights are rejected before traversal begins.
//
// Implementation:
//   - Stage 1: Construct a weighted graph with one -Inf edge.
//   - Stage 2: Call Dijkstra from a valid source.
//   - Stage 3: Assert ErrInvalidWeight through the sentinel protocol.
//
// Behavior highlights:
//   - -Inf is invalid numeric input for the package.
//
// AI-Hints:
//   - Keep -Inf coverage explicit because it is distinct from finite negative weights.
func TestDijkstra_NegativeInfinityWeight_PreScan(t *testing.T) {
	graph := core.NewGraph(core.WithWeighted())

	if _, err := graph.AddEdge(testVertexSource, testVertexMiddle, math.Inf(-1)); err != nil {
		t.Fatalf("AddEdge(%q,%q,-Inf) failed: %v", testVertexSource, testVertexMiddle, err)
	}

	result, err := dijkstra.Dijkstra(graph, testVertexSource)

	mustNilState(t, result, true, "Dijkstra(-Inf weight) result")
	mustErrorIs(t, err, dijkstra.ErrInvalidWeight)
}

// TestDijkstra_BadMaxDistance verifies that invalid MaxDistance configuration
// is surfaced unchanged through the public facade.
//
// Implementation:
//   - Stage 1: Construct a weighted graph with a valid source.
//   - Stage 2: Call Dijkstra with an invalid MaxDistance option.
//   - Stage 3: Assert ErrBadMaxDistance through the sentinel protocol.
//
// Behavior highlights:
//   - Option-policy failures must happen before traversal allocation/work.
//
// AI-Hints:
//   - Keep public option-error propagation explicit in facade-level tests.
func TestDijkstra_BadMaxDistance(t *testing.T) {
	graph := core.NewGraph(core.WithWeighted())

	if err := graph.AddVertex(testVertexSource); err != nil {
		t.Fatalf("AddVertex(%q) failed: %v", testVertexSource, err)
	}

	result, err := dijkstra.Dijkstra(graph, testVertexSource, dijkstra.WithMaxDistance(-testWeightOne))

	mustNilState(t, result, true, "Dijkstra(bad max distance) result")
	mustErrorIs(t, err, dijkstra.ErrBadMaxDistance)
}

// TestDijkstra_BadInfEdgeThreshold verifies that invalid InfEdgeThreshold configuration
// is surfaced unchanged through the public facade.
//
// Implementation:
//   - Stage 1: Construct a weighted graph with a valid source.
//   - Stage 2: Call Dijkstra with an invalid InfEdgeThreshold option.
//   - Stage 3: Assert ErrBadInfEdgeThreshold through the sentinel protocol.
//
// Behavior highlights:
//   - Option-policy failures must happen before traversal allocation/work.
//
// AI-Hints:
//   - Keep threshold-policy admission separate from runtime wall semantics.
func TestDijkstra_BadInfEdgeThreshold(t *testing.T) {
	graph := core.NewGraph(core.WithWeighted())

	if err := graph.AddVertex(testVertexSource); err != nil {
		t.Fatalf("AddVertex(%q) failed: %v", testVertexSource, err)
	}

	result, err := dijkstra.Dijkstra(graph, testVertexSource, dijkstra.WithInfEdgeThreshold(0))

	mustNilState(t, result, true, "Dijkstra(bad threshold) result")
	mustErrorIs(t, err, dijkstra.ErrBadInfEdgeThreshold)
}

// ----------------------------------------------------------------------------
// Medium contract block: shortest-path correctness under normal regimes.
// ----------------------------------------------------------------------------

// TestDijkstra_Triangle verifies exact shortest-path distances and deterministic
// path reconstruction on a compact weighted triangle.
//
// Implementation:
//   - Stage 1: Construct a weighted triangle graph.
//   - Stage 2: Run Dijkstra with path tracking enabled.
//   - Stage 3: Assert exact distances.
//   - Stage 4: Assert the exact deterministic shortest-path witness.
//
// Behavior highlights:
//   - The shortest path from A to C must go through B.
//   - Exact path equality is valid because the package contract is deterministic.
//
// AI-Hints:
//   - Keep this test exact; do not weaken it into “distance only”.
func TestDijkstra_Triangle(t *testing.T) {
	graph := core.NewGraph(core.WithWeighted())

	if _, err := graph.AddEdge(testVertexSource, testVertexMiddle, testWeightOne); err != nil {
		t.Fatalf("AddEdge(%q,%q,1) failed: %v", testVertexSource, testVertexMiddle, err)
	}
	if _, err := graph.AddEdge(testVertexMiddle, testVertexAlternative, testWeightTwo); err != nil {
		t.Fatalf("AddEdge(%q,%q,2) failed: %v", testVertexMiddle, testVertexAlternative, err)
	}
	if _, err := graph.AddEdge(testVertexSource, testVertexAlternative, testWeightFive); err != nil {
		t.Fatalf("AddEdge(%q,%q,5) failed: %v", testVertexSource, testVertexAlternative, err)
	}

	result, err := dijkstra.Dijkstra(graph, testVertexSource, dijkstra.WithPathTracking())
	if err != nil {
		t.Fatalf("Dijkstra(%q) failed: %v", testVertexSource, err)
	}

	gotA, err := result.DistanceTo(testVertexSource)
	if err != nil {
		t.Fatalf("DistanceTo(%q) failed: %v", testVertexSource, err)
	}
	gotB, err := result.DistanceTo(testVertexMiddle)
	if err != nil {
		t.Fatalf("DistanceTo(%q) failed: %v", testVertexMiddle, err)
	}
	gotC, err := result.DistanceTo(testVertexAlternative)
	if err != nil {
		t.Fatalf("DistanceTo(%q) failed: %v", testVertexAlternative, err)
	}

	mustEqualFloat64(t, gotA, 0.0, "DistanceTo(%q): got=%v want=0", testVertexSource, gotA)
	mustEqualFloat64(t, gotB, testWeightOne, "DistanceTo(%q): got=%v want=%v", testVertexMiddle, gotB, testWeightOne)
	mustEqualFloat64(t, gotC, testWeightThree, "DistanceTo(%q): got=%v want=%v", testVertexAlternative, gotC, testWeightThree)

	path, err := result.PathTo(testVertexAlternative)
	if err != nil {
		t.Fatalf("PathTo(%q) failed: %v", testVertexAlternative, err)
	}
	assertPathEqual(t, path, []string{testVertexSource, testVertexMiddle, testVertexAlternative})
}

// TestDijkstra_DirectedGraph verifies shortest-path behavior on a directed weighted graph.
//
// Implementation:
//   - Stage 1: Construct a directed weighted graph.
//   - Stage 2: Run Dijkstra with path tracking enabled.
//   - Stage 3: Assert exact target distances.
//   - Stage 4: Assert the exact deterministic shortest-path witness.
//
// Behavior highlights:
//   - Directed edges must never be traversed backward.
//   - The target route is chosen by shortest-path cost under directed-only semantics.
//
// AI-Hints:
//   - Keep directed coverage distinct from mixed-edge and tie-break coverage.
func TestDijkstra_DirectedGraph(t *testing.T) {
	graph := core.NewGraph(core.WithDirected(true), core.WithWeighted())

	if _, err := graph.AddEdge(testVertexSource, testVertexMiddle, testWeightTwo); err != nil {
		t.Fatalf("AddEdge(%q,%q,2) failed: %v", testVertexSource, testVertexMiddle, err)
	}
	if _, err := graph.AddEdge(testVertexSource, testVertexAlternative, testWeightOne); err != nil {
		t.Fatalf("AddEdge(%q,%q,1) failed: %v", testVertexSource, testVertexAlternative, err)
	}
	if _, err := graph.AddEdge(testVertexAlternative, testVertexMiddle, testWeightFive); err != nil {
		t.Fatalf("AddEdge(%q,%q,5) failed: %v", testVertexAlternative, testVertexMiddle, err)
	}
	if _, err := graph.AddEdge(testVertexMiddle, testVertexTarget, testWeightThree); err != nil {
		t.Fatalf("AddEdge(%q,%q,3) failed: %v", testVertexMiddle, testVertexTarget, err)
	}
	if _, err := graph.AddEdge(testVertexAlternative, testVertexTarget, testWeightTen); err != nil {
		t.Fatalf("AddEdge(%q,%q,10) failed: %v", testVertexAlternative, testVertexTarget, err)
	}

	result, err := dijkstra.Dijkstra(graph, testVertexSource, dijkstra.WithPathTracking())
	if err != nil {
		t.Fatalf("Dijkstra(%q) failed: %v", testVertexSource, err)
	}

	gotB, err := result.DistanceTo(testVertexMiddle)
	if err != nil {
		t.Fatalf("DistanceTo(%q) failed: %v", testVertexMiddle, err)
	}
	gotC, err := result.DistanceTo(testVertexAlternative)
	if err != nil {
		t.Fatalf("DistanceTo(%q) failed: %v", testVertexAlternative, err)
	}
	gotD, err := result.DistanceTo(testVertexTarget)
	if err != nil {
		t.Fatalf("DistanceTo(%q) failed: %v", testVertexTarget, err)
	}

	mustEqualFloat64(t, gotB, testWeightTwo, "DistanceTo(%q): got=%v want=%v", testVertexMiddle, gotB, testWeightTwo)
	mustEqualFloat64(t, gotC, testWeightOne, "DistanceTo(%q): got=%v want=%v", testVertexAlternative, gotC, testWeightOne)
	mustEqualFloat64(t, gotD, testWeightFive, "DistanceTo(%q): got=%v want=%v", testVertexTarget, gotD, testWeightFive)

	path, err := result.PathTo(testVertexTarget)
	if err != nil {
		t.Fatalf("PathTo(%q) failed: %v", testVertexTarget, err)
	}
	assertPathEqual(t, path, []string{testVertexSource, testVertexMiddle, testVertexTarget})
}

// TestDijkstra_MixedGraph verifies shortest-path behavior on a graph that contains
// both directed and undirected edges.
//
// Implementation:
//   - Stage 1: Construct a weighted mixed-edge graph.
//   - Stage 2: Add a directed edge, an undirected edge, and a directed edge.
//   - Stage 3: Run Dijkstra with path tracking enabled.
//   - Stage 4: Assert exact distances and the exact witness path.
//
// Behavior highlights:
//   - Per-edge mixed semantics must be honored.
//   - Undirected traversal must resolve the opposite endpoint relative to the current vertex.
//
// AI-Hints:
//   - Keep mixed-edge coverage explicit; it protects real endpoint-law behavior.
func TestDijkstra_MixedGraph(t *testing.T) {
	graph := core.NewGraph(core.WithWeighted(), core.WithMixedEdges())

	if _, err := graph.AddEdge(testVertexSource, testVertexMiddle, testWeightTwo, core.WithEdgeDirected(true)); err != nil {
		t.Fatalf("AddEdge(%q,%q,2,directed) failed: %v", testVertexSource, testVertexMiddle, err)
	}
	if _, err := graph.AddEdge(testVertexMiddle, testVertexAlternative, testWeightThree, core.WithEdgeDirected(false)); err != nil {
		t.Fatalf("AddEdge(%q,%q,3,undirected) failed: %v", testVertexMiddle, testVertexAlternative, err)
	}
	if _, err := graph.AddEdge(testVertexAlternative, testVertexTarget, testWeightOne, core.WithEdgeDirected(true)); err != nil {
		t.Fatalf("AddEdge(%q,%q,1,directed) failed: %v", testVertexAlternative, testVertexTarget, err)
	}
	if _, err := graph.AddEdge(testVertexSource, testVertexTarget, testWeightTen, core.WithEdgeDirected(true)); err != nil {
		t.Fatalf("AddEdge(%q,%q,10,directed) failed: %v", testVertexSource, testVertexTarget, err)
	}

	result, err := dijkstra.Dijkstra(graph, testVertexSource, dijkstra.WithPathTracking())
	if err != nil {
		t.Fatalf("Dijkstra(%q) failed: %v", testVertexSource, err)
	}

	gotB, err := result.DistanceTo(testVertexMiddle)
	if err != nil {
		t.Fatalf("DistanceTo(%q) failed: %v", testVertexMiddle, err)
	}
	gotC, err := result.DistanceTo(testVertexAlternative)
	if err != nil {
		t.Fatalf("DistanceTo(%q) failed: %v", testVertexAlternative, err)
	}
	gotD, err := result.DistanceTo(testVertexTarget)
	if err != nil {
		t.Fatalf("DistanceTo(%q) failed: %v", testVertexTarget, err)
	}

	mustEqualFloat64(t, gotB, testWeightTwo, "DistanceTo(%q): got=%v want=%v", testVertexMiddle, gotB, testWeightTwo)
	mustEqualFloat64(t, gotC, testWeightFive, "DistanceTo(%q): got=%v want=%v", testVertexAlternative, gotC, testWeightFive)
	mustEqualFloat64(t, gotD, testWeightSix, "DistanceTo(%q): got=%v want=%v", testVertexTarget, gotD, testWeightSix)

	path, err := result.PathTo(testVertexTarget)
	if err != nil {
		t.Fatalf("PathTo(%q) failed: %v", testVertexTarget, err)
	}
	assertPathEqual(t, path, []string{testVertexSource, testVertexMiddle, testVertexAlternative, testVertexTarget})
}

// TestDijkstra_MaxDistanceCutoff verifies that vertices beyond the configured
// maximum distance remain at +Inf and are not finalized.
//
// Implementation:
//   - Stage 1: Construct a weighted chain graph.
//   - Stage 2: Run Dijkstra with MaxDistance set to one edge.
//   - Stage 3: Assert finite distance for the near vertex.
//   - Stage 4: Assert +Inf for vertices beyond the cutoff.
//
// Behavior highlights:
//   - MaxDistance is a traversal policy, not a graph-validity rule.
//   - Vertices beyond the cutoff remain known but unreachable in the published result.
//
// AI-Hints:
//   - Keep +Inf assertions explicit; cutoff semantics are part of the public result law.
func TestDijkstra_MaxDistanceCutoff(t *testing.T) {
	graph := core.NewGraph(core.WithWeighted())

	if _, err := graph.AddEdge(testVertexSource, testVertexMiddle, testWeightOne); err != nil {
		t.Fatalf("AddEdge(%q,%q,1) failed: %v", testVertexSource, testVertexMiddle, err)
	}
	if _, err := graph.AddEdge(testVertexMiddle, testVertexAlternative, testWeightOne); err != nil {
		t.Fatalf("AddEdge(%q,%q,1) failed: %v", testVertexMiddle, testVertexAlternative, err)
	}
	if _, err := graph.AddEdge(testVertexAlternative, testVertexTarget, testWeightOne); err != nil {
		t.Fatalf("AddEdge(%q,%q,1) failed: %v", testVertexAlternative, testVertexTarget, err)
	}

	result, err := dijkstra.Dijkstra(graph, testVertexSource, dijkstra.WithMaxDistance(testWeightOne))
	if err != nil {
		t.Fatalf("Dijkstra(%q) failed: %v", testVertexSource, err)
	}

	gotB, err := result.DistanceTo(testVertexMiddle)
	if err != nil {
		t.Fatalf("DistanceTo(%q) failed: %v", testVertexMiddle, err)
	}
	gotC, err := result.DistanceTo(testVertexAlternative)
	if err != nil {
		t.Fatalf("DistanceTo(%q) failed: %v", testVertexAlternative, err)
	}
	gotD, err := result.DistanceTo(testVertexTarget)
	if err != nil {
		t.Fatalf("DistanceTo(%q) failed: %v", testVertexTarget, err)
	}

	mustEqualFloat64(t, gotB, testWeightOne, "DistanceTo(%q): got=%v want=%v", testVertexMiddle, gotB, testWeightOne)
	assertInfDistance(t, gotC)
	assertInfDistance(t, gotD)
}

// TestDijkstra_InfEdgeThresholdWall verifies that edges at or above the configured
// threshold are skipped as walls during relaxation.
//
// Implementation:
//   - Stage 1: Construct a graph with one heavy direct edge and one lighter two-step route.
//   - Stage 2: Run Dijkstra with a wall threshold below the heavy direct edge.
//   - Stage 3: Assert that the lighter two-step route determines the result.
//
// Behavior highlights:
//   - Threshold policy is distinct from invalid-weight policy.
//   - Heavy finite edges may be legal graph data yet intentionally non-traversable.
//
// AI-Hints:
//   - Keep threshold-wall coverage separate from +Inf-edge wall coverage.
func TestDijkstra_InfEdgeThresholdWall(t *testing.T) {
	graph := core.NewGraph(core.WithWeighted())

	if _, err := graph.AddEdge(testVertexSource, testVertexMiddle, testWeightTwo); err != nil {
		t.Fatalf("AddEdge(%q,%q,2) failed: %v", testVertexSource, testVertexMiddle, err)
	}
	if _, err := graph.AddEdge(testVertexMiddle, testVertexAlternative, testWeightFour); err != nil {
		t.Fatalf("AddEdge(%q,%q,4) failed: %v", testVertexMiddle, testVertexAlternative, err)
	}
	if _, err := graph.AddEdge(testVertexSource, testVertexAlternative, testWeightTen); err != nil {
		t.Fatalf("AddEdge(%q,%q,10) failed: %v", testVertexSource, testVertexAlternative, err)
	}

	result, err := dijkstra.Dijkstra(graph, testVertexSource, dijkstra.WithInfEdgeThreshold(testWeightFive))
	if err != nil {
		t.Fatalf("Dijkstra(%q) failed: %v", testVertexSource, err)
	}

	gotC, err := result.DistanceTo(testVertexAlternative)
	if err != nil {
		t.Fatalf("DistanceTo(%q) failed: %v", testVertexAlternative, err)
	}

	mustEqualFloat64(t, gotC, testWeightSix, "DistanceTo(%q): got=%v want=%v", testVertexAlternative, gotC, testWeightSix)
}

// TestDijkstra_MultiEdgeChoosesBestDistance verifies that, when multi-edges are enabled,
// the best parallel edge governs the shortest path.
//
// Implementation:
//   - Stage 1: Construct a weighted multi-edge graph.
//   - Stage 2: Add two distinct A->B edges with different weights.
//   - Stage 3: Run Dijkstra with path tracking enabled.
//   - Stage 4: Assert that the lighter edge determines the final result.
//
// Behavior highlights:
//   - Multi-edge support must preserve shortest-path optimality.
//   - The test protects true distance optimality, not merely edge insertion order.
//
// AI-Hints:
//   - Keep multi-edge optimality separate from equal-cost tie-break coverage.
func TestDijkstra_MultiEdgeChoosesBestDistance(t *testing.T) {
	graph := core.NewGraph(core.WithWeighted(), core.WithMultiEdges())

	if _, err := graph.AddEdge(testVertexSource, testVertexMiddle, testWeightFive); err != nil {
		t.Fatalf("AddEdge(%q,%q,5) failed: %v", testVertexSource, testVertexMiddle, err)
	}
	if _, err := graph.AddEdge(testVertexSource, testVertexMiddle, testWeightTwo); err != nil {
		t.Fatalf("AddEdge(%q,%q,2) failed: %v", testVertexSource, testVertexMiddle, err)
	}
	if _, err := graph.AddEdge(testVertexMiddle, testVertexAlternative, testWeightOne); err != nil {
		t.Fatalf("AddEdge(%q,%q,1) failed: %v", testVertexMiddle, testVertexAlternative, err)
	}
	if _, err := graph.AddEdge(testVertexSource, testVertexAlternative, testWeightTen); err != nil {
		t.Fatalf("AddEdge(%q,%q,10) failed: %v", testVertexSource, testVertexAlternative, err)
	}

	result, err := dijkstra.Dijkstra(graph, testVertexSource, dijkstra.WithPathTracking())
	if err != nil {
		t.Fatalf("Dijkstra(%q) failed: %v", testVertexSource, err)
	}

	gotB, err := result.DistanceTo(testVertexMiddle)
	if err != nil {
		t.Fatalf("DistanceTo(%q) failed: %v", testVertexMiddle, err)
	}
	gotC, err := result.DistanceTo(testVertexAlternative)
	if err != nil {
		t.Fatalf("DistanceTo(%q) failed: %v", testVertexAlternative, err)
	}

	mustEqualFloat64(t, gotB, testWeightTwo, "DistanceTo(%q): got=%v want=%v", testVertexMiddle, gotB, testWeightTwo)
	mustEqualFloat64(t, gotC, testWeightThree, "DistanceTo(%q): got=%v want=%v", testVertexAlternative, gotC, testWeightThree)

	path, err := result.PathTo(testVertexAlternative)
	if err != nil {
		t.Fatalf("PathTo(%q) failed: %v", testVertexAlternative, err)
	}
	assertPathEqual(t, path, []string{testVertexSource, testVertexMiddle, testVertexAlternative})
}

// ----------------------------------------------------------------------------
// Special / regression block: endpoint law, determinism, published result-law.
// ----------------------------------------------------------------------------

// TestDijkstra_UndirectedReverseEndpoint_Regression anchors the regression for
// undirected edges whose stored direction is opposite to the traversal source.
//
// Implementation:
//   - Stage 1: Construct a weighted undirected graph.
//   - Stage 2: Add the edge as B--A rather than A--B.
//   - Stage 3: Run Dijkstra from A.
//   - Stage 4: Assert that B is still reachable with distance 1.
//
// Behavior highlights:
//   - Undirected traversal must resolve the other endpoint relative to the current vertex.
//   - Stored edge direction must not break undirected reachability.
//
// AI-Hints:
//   - Do not “simplify” endpoint resolution to edge.To and keep this test by accident.
func TestDijkstra_UndirectedReverseEndpoint_Regression(t *testing.T) {
	graph := core.NewGraph(core.WithWeighted())

	if _, err := graph.AddEdge(testVertexMiddle, testVertexSource, testWeightOne); err != nil {
		t.Fatalf("AddEdge(%q,%q,1) failed: %v", testVertexMiddle, testVertexSource, err)
	}

	result, err := dijkstra.Dijkstra(graph, testVertexSource)
	if err != nil {
		t.Fatalf("Dijkstra(%q) failed: %v", testVertexSource, err)
	}

	got, err := result.DistanceTo(testVertexMiddle)
	if err != nil {
		t.Fatalf("DistanceTo(%q) failed: %v", testVertexMiddle, err)
	}

	mustEqualFloat64(t, got, testWeightOne, "DistanceTo(%q): got=%v want=%v", testVertexMiddle, got, testWeightOne)
}

// TestDijkstra_TieBreakEqualShortestPaths anchors deterministic predecessor selection
// when two equal-cost shortest routes compete for the same target.
//
// Implementation:
//   - Stage 1: Construct a graph with equal-cost routes A->B->D and A->C->D.
//   - Stage 2: Run Dijkstra with path tracking enabled.
//   - Stage 3: Assert that D keeps B as its predecessor.
//   - Stage 4: Assert the exact deterministic witness path.
//
// Behavior highlights:
//   - Heap tie-break by vertex ID must stabilize equal-distance frontier order.
//   - Strict-improvement-only updates must prevent equal-cost predecessor overwrite.
//
// AI-Hints:
//   - Do not weaken this into “one of several valid parents”.
func TestDijkstra_TieBreakEqualShortestPaths(t *testing.T) {
	graph := core.NewGraph(core.WithWeighted())

	if _, err := graph.AddEdge(testVertexSource, testVertexMiddle, testWeightOne); err != nil {
		t.Fatalf("AddEdge(%q,%q,1) failed: %v", testVertexSource, testVertexMiddle, err)
	}
	if _, err := graph.AddEdge(testVertexSource, testVertexAlternative, testWeightOne); err != nil {
		t.Fatalf("AddEdge(%q,%q,1) failed: %v", testVertexSource, testVertexAlternative, err)
	}
	if _, err := graph.AddEdge(testVertexMiddle, testVertexTarget, testWeightOne); err != nil {
		t.Fatalf("AddEdge(%q,%q,1) failed: %v", testVertexMiddle, testVertexTarget, err)
	}
	if _, err := graph.AddEdge(testVertexAlternative, testVertexTarget, testWeightOne); err != nil {
		t.Fatalf("AddEdge(%q,%q,1) failed: %v", testVertexAlternative, testVertexTarget, err)
	}

	result, err := dijkstra.Dijkstra(graph, testVertexSource, dijkstra.WithPathTracking())
	if err != nil {
		t.Fatalf("Dijkstra(%q) failed: %v", testVertexSource, err)
	}

	mustEqualString(
		t,
		result.Prev[testVertexTarget],
		testVertexMiddle,
		"Prev[%q]: got=%q want=%q",
		testVertexTarget,
		result.Prev[testVertexTarget],
		testVertexMiddle,
	)

	path, err := result.PathTo(testVertexTarget)
	if err != nil {
		t.Fatalf("PathTo(%q) failed: %v", testVertexTarget, err)
	}
	assertPathEqual(t, path, []string{testVertexSource, testVertexMiddle, testVertexTarget})
}

// TestDijkstra_PrevNilWhenPathTrackingDisabled verifies that predecessor storage
// stays absent unless path tracking is requested explicitly.
//
// Implementation:
//   - Stage 1: Construct a weighted graph.
//   - Stage 2: Run Dijkstra without WithPathTracking.
//   - Stage 3: Assert that Prev remains nil.
//   - Stage 4: Assert that PathTo fails with ErrPathTrackingDisabled.
//
// Behavior highlights:
//   - Path tracking is explicit and opt-in.
//   - Distances remain usable without predecessor storage.
//
// AI-Hints:
//   - Do not infer path tracking implicitly in the base API.
func TestDijkstra_PrevNilWhenPathTrackingDisabled(t *testing.T) {
	graph := core.NewGraph(core.WithWeighted())

	if _, err := graph.AddEdge(testVertexSource, testVertexMiddle, testWeightOne); err != nil {
		t.Fatalf("AddEdge(%q,%q,1) failed: %v", testVertexSource, testVertexMiddle, err)
	}

	result, err := dijkstra.Dijkstra(graph, testVertexSource)
	if err != nil {
		t.Fatalf("Dijkstra(%q) failed: %v", testVertexSource, err)
	}

	mustNilState(t, result.Prev, true, "Prev when path tracking disabled")

	_, err = result.PathTo(testVertexMiddle)
	mustErrorIs(t, err, dijkstra.ErrPathTrackingDisabled)
}

// TestDijkstra_PathTrackingEnabled verifies that explicit path tracking produces
// non-nil predecessor storage and usable witness reconstruction.
//
// Implementation:
//   - Stage 1: Construct a weighted chain graph.
//   - Stage 2: Run Dijkstra with WithPathTracking.
//   - Stage 3: Assert that Prev is non-nil and contains the expected parents.
//   - Stage 4: Assert the exact deterministic witness path.
//
// Behavior highlights:
//   - Path tracking is explicit.
//   - The produced predecessor map must align with the reconstructed witness.
//
// AI-Hints:
//   - Keep enabled and disabled tracking covered by separate tests.
func TestDijkstra_PathTrackingEnabled(t *testing.T) {
	graph := core.NewGraph(core.WithWeighted())

	if _, err := graph.AddEdge(testVertexSource, testVertexMiddle, testWeightOne); err != nil {
		t.Fatalf("AddEdge(%q,%q,1) failed: %v", testVertexSource, testVertexMiddle, err)
	}
	if _, err := graph.AddEdge(testVertexMiddle, testVertexAlternative, testWeightOne); err != nil {
		t.Fatalf("AddEdge(%q,%q,1) failed: %v", testVertexMiddle, testVertexAlternative, err)
	}

	result, err := dijkstra.Dijkstra(graph, testVertexSource, dijkstra.WithPathTracking())
	if err != nil {
		t.Fatalf("Dijkstra(%q) failed: %v", testVertexSource, err)
	}

	mustNilState(t, result.Prev, false, "Prev when path tracking enabled")
	mustEqualString(
		t,
		result.Prev[testVertexMiddle],
		testVertexSource,
		"Prev[%q]: got=%q want=%q",
		testVertexMiddle,
		result.Prev[testVertexMiddle],
		testVertexSource,
	)
	mustEqualString(
		t,
		result.Prev[testVertexAlternative],
		testVertexMiddle,
		"Prev[%q]: got=%q want=%q",
		testVertexAlternative,
		result.Prev[testVertexAlternative],
		testVertexMiddle,
	)

	path, err := result.PathTo(testVertexAlternative)
	if err != nil {
		t.Fatalf("PathTo(%q) failed: %v", testVertexAlternative, err)
	}
	assertPathEqual(t, path, []string{testVertexSource, testVertexMiddle, testVertexAlternative})
}

// TestDijkstra_InfiniteWeightActsAsWall verifies that +Inf edge weights are treated
// as impassable walls rather than as invalid numeric input.
//
// Implementation:
//   - Stage 1: Construct a weighted graph with one +Inf edge.
//   - Stage 2: Run Dijkstra from the finite source.
//   - Stage 3: Assert that the +Inf target remains unreachable with no validation error.
//
// Behavior highlights:
//   - +Inf is valid graph data under this package contract.
//   - Unreachable known targets remain present in the result domain with +Inf distance.
//
// AI-Hints:
//   - Keep +Inf wall coverage separate from NaN and -Inf invalid-weight coverage.
func TestDijkstra_InfiniteWeightActsAsWall(t *testing.T) {
	graph := core.NewGraph(core.WithWeighted())

	if _, err := graph.AddEdge(testVertexSource, testVertexMiddle, math.Inf(1)); err != nil {
		t.Fatalf("AddEdge(%q,%q,+Inf) failed: %v", testVertexSource, testVertexMiddle, err)
	}

	result, err := dijkstra.Dijkstra(graph, testVertexSource)
	if err != nil {
		t.Fatalf("Dijkstra(%q) failed: %v", testVertexSource, err)
	}

	got, err := result.DistanceTo(testVertexMiddle)
	if err != nil {
		t.Fatalf("DistanceTo(%q) failed: %v", testVertexMiddle, err)
	}
	assertInfDistance(t, got)
}

// TestDijkstra_SourceToSelf_ZeroDistance verifies that the source remains at zero
// distance and reconstructs a single-vertex witness when tracking is enabled.
//
// Implementation:
//   - Stage 1: Construct a weighted graph with a single source vertex.
//   - Stage 2: Run Dijkstra with path tracking enabled.
//   - Stage 3: Assert zero distance to the source.
//   - Stage 4: Assert the exact single-vertex source path.
//
// Behavior highlights:
//   - Source distance is always zero.
//   - Source path reconstruction is a first-class published behavior.
//
// AI-Hints:
//   - Keep source-to-self coverage explicit because it anchors base result semantics.
func TestDijkstra_SourceToSelf_ZeroDistance(t *testing.T) {
	graph := core.NewGraph(core.WithWeighted())

	if err := graph.AddVertex(testVertexSource); err != nil {
		t.Fatalf("AddVertex(%q) failed: %v", testVertexSource, err)
	}

	result, err := dijkstra.Dijkstra(graph, testVertexSource, dijkstra.WithPathTracking())
	if err != nil {
		t.Fatalf("Dijkstra(%q) failed: %v", testVertexSource, err)
	}

	got, err := result.DistanceTo(testVertexSource)
	if err != nil {
		t.Fatalf("DistanceTo(%q) failed: %v", testVertexSource, err)
	}
	mustEqualFloat64(t, got, 0.0, "DistanceTo(%q): got=%v want=0", testVertexSource, got)

	path, err := result.PathTo(testVertexSource)
	if err != nil {
		t.Fatalf("PathTo(%q) failed: %v", testVertexSource, err)
	}
	assertPathEqual(t, path, []string{testVertexSource})
}

// TestDijkstra_SelfLoopZeroWeight verifies that a zero-weight self-loop does not
// corrupt source distance or source-path semantics.
//
// Implementation:
//   - Stage 1: Construct a weighted graph with loops enabled.
//   - Stage 2: Add a zero-weight self-loop.
//   - Stage 3: Run Dijkstra with path tracking enabled.
//   - Stage 4: Assert zero distance, empty self-parent, and a single-vertex source path.
//
// Behavior highlights:
//   - Zero-weight self-loops are valid when loops are enabled.
//   - Strict-improvement-only updates must keep the source witness stable.
//
// AI-Hints:
//   - Keep self-loop coverage separate from plain source-to-self coverage.
func TestDijkstra_SelfLoopZeroWeight(t *testing.T) {
	graph := core.NewGraph(core.WithWeighted(), core.WithLoops())

	if _, err := graph.AddEdge(testVertexLoop, testVertexLoop, 0.0); err != nil {
		t.Fatalf("AddEdge(%q,%q,0) failed: %v", testVertexLoop, testVertexLoop, err)
	}

	result, err := dijkstra.Dijkstra(graph, testVertexLoop, dijkstra.WithPathTracking())
	if err != nil {
		t.Fatalf("Dijkstra(%q) failed: %v", testVertexLoop, err)
	}

	got, err := result.DistanceTo(testVertexLoop)
	if err != nil {
		t.Fatalf("DistanceTo(%q) failed: %v", testVertexLoop, err)
	}
	mustEqualFloat64(t, got, 0.0, "DistanceTo(%q): got=%v want=0", testVertexLoop, got)

	mustEqualString(
		t,
		result.Prev[testVertexLoop],
		"",
		"Prev[%q]: got=%q want=%q",
		testVertexLoop,
		result.Prev[testVertexLoop],
		"",
	)

	path, err := result.PathTo(testVertexLoop)
	if err != nil {
		t.Fatalf("PathTo(%q) failed: %v", testVertexLoop, err)
	}
	assertPathEqual(t, path, []string{testVertexLoop})
}

// TestDijkstra_UnreachableRemainsInf verifies that known but disconnected vertices
// remain in the result domain with +Inf distance, false reachability, and ErrNoPath
// when path tracking is enabled.
//
// Implementation:
//   - Stage 1: Construct a weighted graph with one connected component and one isolated vertex.
//   - Stage 2: Run Dijkstra with path tracking enabled.
//   - Stage 3: Assert +Inf distance for the isolated vertex.
//   - Stage 4: Assert HasPathTo reports false and PathTo reports ErrNoPath.
//
// Behavior highlights:
//   - Known unreachable vertices remain part of the result domain.
//   - Missing-target and unreachable-target semantics remain distinct.
//
// AI-Hints:
//   - Keep disconnected known-vertex coverage explicit; it protects the result-domain law.
func TestDijkstra_UnreachableRemainsInf(t *testing.T) {
	graph := core.NewGraph(core.WithWeighted())

	if _, err := graph.AddEdge(testVertexSource, testVertexMiddle, testWeightOne); err != nil {
		t.Fatalf("AddEdge(%q,%q,1) failed: %v", testVertexSource, testVertexMiddle, err)
	}
	if err := graph.AddVertex(testVertexUnreachable); err != nil {
		t.Fatalf("AddVertex(%q) failed: %v", testVertexUnreachable, err)
	}

	result, err := dijkstra.Dijkstra(graph, testVertexSource, dijkstra.WithPathTracking())
	if err != nil {
		t.Fatalf("Dijkstra(%q) failed: %v", testVertexSource, err)
	}

	got, err := result.DistanceTo(testVertexUnreachable)
	if err != nil {
		t.Fatalf("DistanceTo(%q) failed: %v", testVertexUnreachable, err)
	}
	assertInfDistance(t, got)

	hasPath, err := result.HasPathTo(testVertexUnreachable)
	if err != nil {
		t.Fatalf("HasPathTo(%q) failed: %v", testVertexUnreachable, err)
	}
	mustEqualBool(t, hasPath, false, "HasPathTo(%q): got=%v want=false", testVertexUnreachable, hasPath)

	_, err = result.PathTo(testVertexUnreachable)
	mustErrorIs(t, err, dijkstra.ErrNoPath)
}
