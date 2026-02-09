// SPDX-License-Identifier: MIT
// Package core_test verifies core.Graph method-level contracts.
//
// Purpose:
//   - Lock in deterministic behaviors for vertex/edge lifecycle and query APIs.
//   - Validate constraint enforcement (weights, loops, multi-edges) without third-party libs.
//   - Provide contract anchors for ordering guarantees (Vertices/Edges/Neighbors sorted by ID).

package core_test

import (
	"testing"

	"github.com/katalvlaran/lvlath/core"
)

// TestGraph_AddRemoveVertex VERIFIES AddVertex/HasVertex/RemoveVertex lifecycle rules.
// Implementation:
//   - Stage 1: Create a default graph.
//   - Stage 2: Assert AddVertex(empty) returns ErrEmptyVertexID.
//   - Stage 3: Add a valid vertex and assert membership.
//   - Stage 4: Assert duplicate AddVertex is a no-op (no error, no count change).
//   - Stage 5: Assert RemoveVertex(empty) and RemoveVertex(missing) return sentinels.
//   - Stage 6: Remove an existing vertex and assert absence.
//
// Behavior highlights:
//   - Enforces empty-ID rejection for vertex insertion/removal.
//   - Enforces idempotent AddVertex semantics.
//
// Inputs:
//   - None (uses package constants).
//
// Returns:
//   - None.
//
// Errors:
//   - core.ErrEmptyVertexID from AddVertex/RemoveVertex with empty ID.
//   - core.ErrVertexNotFound from RemoveVertex with missing ID.
//
// Determinism:
//   - Deterministic (no randomness).
//
// Complexity:
//   - Time O(V log V) dominated by Vertices() sorting for count checks, Space O(V) for returned slice.
//
// Notes:
//   - This test treats Vertices() order as deterministic (covered by separate order anchor).
//
// AI-Hints:
//   - Use short stable IDs (VertexA/VertexX) to keep failure output compact.
//   - Prefer verifying no-op via count deltas rather than internal map inspection.
func TestGraph_AddRemoveVertex(t *testing.T) {
	// Stage 1: Create a default graph (undirected, unweighted, no loops, no multi-edges).
	g := core.NewGraph()

	// Stage 2: Validate empty ID rejection on AddVertex.
	{
		// Attempt to add an empty vertex ID.
		err := g.AddVertex(VertexEmpty)
		// Enforce sentinel error contract.
		MustErrorIs(t, err, core.ErrEmptyVertexID, "AddVertex(empty)")
	}

	// Stage 3: Add a valid vertex and validate membership query.
	{
		// Add VertexA.
		MustErrorNil(t, g.AddVertex(VertexA), "AddVertex(A)")
		// Verify VertexA is present.
		MustEqualBool(t, g.HasVertex(VertexA), true, "HasVertex(A) after AddVertex(A)")
	}

	// Stage 4: Duplicate AddVertex must be a no-op (no error, no count change).
	{
		// Snapshot vertex count before duplicate insert.
		before := len(g.Vertices())
		// Re-add VertexA (must not error).
		MustErrorNil(t, g.AddVertex(VertexA), "AddVertex(A) duplicate")
		// Snapshot vertex count after duplicate insert.
		after := len(g.Vertices())
		// Enforce no-op count invariant.
		MustEqualInt(t, after, before, "duplicate AddVertex(A) must not change vertex count")
	}

	// Stage 5: Remove validations (empty and non-existent).
	{
		// Attempt to remove an empty vertex ID.
		err := g.RemoveVertex(VertexEmpty)
		// Enforce sentinel error contract.
		MustErrorIs(t, err, core.ErrEmptyVertexID, "RemoveVertex(empty)")
	}
	{
		// Attempt to remove a missing vertex ID (VertexX was not added in this test).
		err := g.RemoveVertex(VertexX)
		// Enforce sentinel error contract.
		MustErrorIs(t, err, core.ErrVertexNotFound, "RemoveVertex(X missing)")
	}

	// Stage 6: Remove existing vertex and validate membership query.
	{
		// Remove VertexA.
		MustErrorNil(t, g.RemoveVertex(VertexA), "RemoveVertex(A)")
		// Verify VertexA is absent after removal.
		MustEqualBool(t, g.HasVertex(VertexA), false, "HasVertex(A) after RemoveVertex(A)")
	}
}

// TestGraph_AddEdgeConstraints VERIFIES AddEdge constraint enforcement for weights, loops, multi-edges.
// Implementation:
//   - Stage 1: Assert unweighted graph rejects non-zero weight (ErrBadWeight).
//   - Stage 2: Assert weighted graph accepts non-zero weight.
//   - Stage 3: Assert loop-disabled graph rejects self-loop (ErrLoopNotAllowed).
//   - Stage 4: Assert loop-enabled graph accepts self-loop and returns non-empty edge ID.
//   - Stage 5: Assert multi-edge-disabled graph rejects parallel edge (ErrMultiEdgeNotAllowed).
//   - Stage 6: Assert multi-edge-enabled graph accepts parallel edges with distinct IDs.
//
// Behavior highlights:
//   - Fixes sentinel error mapping for invalid edge insertions.
//
// Inputs:
//   - None (uses package constants).
//
// Returns:
//   - None.
//
// Errors:
//   - core.ErrBadWeight when Weighted()==false and weight != 0.
//   - core.ErrLoopNotAllowed when Loops()==false and from==to.
//   - core.ErrMultiEdgeNotAllowed when MultiEdges()==false and endpoints duplicate an existing edge.
//
// Determinism:
//   - Deterministic (no randomness).
//
// Complexity:
//   - Time O(1) per AddEdge membership/constraint check (implementation-dependent), Space O(1) incremental per edge.
//
// Notes:
//   - This test does not assert edge ID format, only non-emptiness and uniqueness when required.
//
// AI-Hints:
//   - Keep constraint tests isolated by building a fresh graph per stage.
//   - Use constants for weights to avoid magic numerics in failure output.
func TestGraph_AddEdgeConstraints(t *testing.T) {
	// Stage 1: Unweighted graph rejects non-zero weight.
	// Create unweighted default graph.
	g := core.NewGraph()
	// Attempt to add a weighted edge on an unweighted graph.
	_, err := g.AddEdge(VertexA, VertexB, Weight5)
	// Enforce sentinel error contract.
	MustErrorIs(t, err, core.ErrBadWeight, "AddEdge(A,B,5) on unweighted graph")

	// Stage 2: Weighted graph accepts non-zero weight and creates the edge.
	// Create weighted graph.
	g = core.NewGraph(core.WithWeighted())
	// Add weighted edge.
	_, err = g.AddEdge(VertexA, VertexB, Weight7)
	// Must succeed.
	MustErrorNil(t, err, "AddEdge(A,B,7) on weighted graph")
	// Membership query must succeed via adjacency.
	MustEqualBool(t, g.HasEdge(VertexA, VertexB), true, "HasEdge(A,B) after AddEdge(A,B,7)")

	// Stage 3: Default graph disallows self-loops.
	// Create default graph (loops disabled).
	g = core.NewGraph()
	// Attempt to add self-loop.
	_, err = g.AddEdge(VertexX, VertexX, Weight0)
	// Enforce sentinel error contract.
	MustErrorIs(t, err, core.ErrLoopNotAllowed, "AddEdge(X,X,0) when loops disabled")

	// Stage 4: Loop-enabled graph accepts self-loops.
	// Create loop-enabled graph.
	g = core.NewGraph(core.WithLoops())
	// Add self-loop.
	loopID, err := g.AddEdge(VertexX, VertexX, Weight0)
	// Must succeed.
	MustErrorNil(t, err, "AddEdge(X,X,0) when loops enabled")
	// ID must be non-empty (format is not a contract here).
	MustNotEqualString(t, loopID, "", "AddEdge(X,X,0) must return non-empty edge ID")
	// Membership query must be true.
	MustEqualBool(t, g.HasEdge(VertexX, VertexX), true, "HasEdge(X,X) after adding self-loop")

	// Stage 5: Multi-edge disallowed by default (second edge with same endpoints must error).
	// Create default graph (multi-edges disabled).
	g = core.NewGraph()
	// Add first edge (must succeed).
	_, err = g.AddEdge(VertexA, VertexB, Weight0)
	MustErrorNil(t, err, "first AddEdge(A,B,0) on default graph")
	// Add second parallel edge (must fail).
	_, err = g.AddEdge(VertexA, VertexB, Weight0)
	MustErrorIs(t, err, core.ErrMultiEdgeNotAllowed, "second AddEdge(A,B,0) on default graph")

	// Stage 6: Multi-edge enabled graph allows parallel edges with distinct IDs.
	// Create graph with multi-edges, weights, and loops enabled to maximize surface.
	g = core.NewGraph(core.WithMultiEdges(), core.WithWeighted(), core.WithLoops())
	// Add first edge.
	e1, err := g.AddEdge(VertexA, VertexB, Weight1)
	MustErrorNil(t, err, "first AddEdge(A,B,1) on multigraph")
	// Add second parallel edge.
	e2, err := g.AddEdge(VertexA, VertexB, Weight2)
	MustErrorNil(t, err, "second AddEdge(A,B,2) on multigraph")
	// Parallel edges must produce distinct IDs.
	MustNotEqualString(t, e1, e2, "parallel AddEdge(A,B,*) must return distinct IDs when multi-edges enabled")
}

// TestGraph_MixedEdgesDirectedOverride VERIFIES per-edge directedness override gating and behavior.
// Implementation:
//   - Stage 1: Non-mixed graph must reject WithEdgeDirected override (ErrMixedEdgesNotAllowed).
//   - Stage 2: Mixed graph must accept override and set Edge.Directed accordingly.
//
// Behavior highlights:
//   - Prevents silent “mixed behavior” on graphs that did not opt into mixed mode.
//
// Inputs:
//   - None (uses package constants).
//
// Returns:
//   - None.
//
// Errors:
//   - core.ErrMixedEdgesNotAllowed from AddEdge when mixed mode is disabled and an EdgeOption is passed.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1) expected, Space O(1) incremental per edge.
//
// Notes:
//   - This test does not assert edge ID format; only the policy gate + Directed flag semantics.
//
// AI-Hints:
//   - Always create mixed graphs explicitly (NewMixedGraph or WithMixedEdges) before using WithEdgeDirected.
func TestGraph_MixedEdgesDirectedOverride(t *testing.T) {
	// Stage 1: Non-mixed graph rejects per-edge override.
	{
		// Create a default (non-mixed) graph.
		g := core.NewGraph()
		// Attempt to override per-edge directedness without mixed mode.
		_, err := g.AddEdge(VertexX, VertexY, Weight0, core.WithEdgeDirected(true))
		// Enforce sentinel gate contract.
		MustErrorIs(t, err, core.ErrMixedEdgesNotAllowed, "AddEdge(X,Y,0,WithEdgeDirected) on non-mixed graph")
	}

	// Stage 2: Mixed graph accepts per-edge override and sets Edge.Directed=true.
	{
		// Create a mixed graph (per-edge directedness overrides allowed).
		g := core.NewMixedGraph()
		// Add an edge overriding directedness to true.
		eid, err := g.AddEdge(VertexX, VertexY, Weight0, core.WithEdgeDirected(true))
		MustErrorNil(t, err, "AddEdge(X,Y,0,WithEdgeDirected(true)) on mixed graph")
		// Read back the edge by ID.
		e, err := g.GetEdge(eid)
		MustErrorNil(t, err, "GetEdge(eid) on mixed graph")
		// Validate per-edge directedness override effect.
		MustEqualBool(t, e.Directed, true, "mixed edge must have Directed=true after WithEdgeDirected(true)")
	}
}

// TestGraph_RemoveEdge VERIFIES RemoveEdge sentinel behavior and adjacency cleanup.
// Implementation:
//   - Stage 1: Create a weighted graph and add two edges.
//   - Stage 2: Assert RemoveEdge(missing) returns ErrEdgeNotFound.
//   - Stage 3: Remove an existing edge and assert adjacency is cleaned.
//
// Behavior highlights:
//   - Locks in ErrEdgeNotFound sentinel for unknown IDs.
//   - Locks in undirected mirror cleanup behavior.
//
// Inputs:
//   - None (uses package constants).
//
// Returns:
//   - None.
//
// Errors:
//   - core.ErrEdgeNotFound when removing an unknown edge ID.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1) expected for remove by ID (implementation-dependent), Space O(1) extra.
//
// Notes:
//   - This test assumes HasEdge is safe even if vertices exist but edge is removed.
//
// AI-Hints:
//   - Prefer verifying cleanup via HasEdge(from,to) and HasEdge(to,from) in undirected graphs.
func TestGraph_RemoveEdge(t *testing.T) {
	// Stage 1: Create weighted graph and add two edges.
	g := core.NewGraph(core.WithWeighted())

	// Add edge A-B to later remove.
	eidAB, err := g.AddEdge(VertexA, VertexB, Weight1)
	MustErrorNil(t, err, "AddEdge(A,B,1) setup")

	// Add edge B-C to ensure unrelated edges remain.
	_, err = g.AddEdge(VertexB, VertexC, Weight2)
	MustErrorNil(t, err, "AddEdge(B,C,2) setup")

	// Stage 2: Removing a non-existent edge must yield ErrEdgeNotFound.
	err = g.RemoveEdge(EdgeIDMissing)
	MustErrorIs(t, err, core.ErrEdgeNotFound, "RemoveEdge(missing)")
	// Stage 3: Remove existing A-B and verify undirected adjacency cleanup.
	MustErrorNil(t, g.RemoveEdge(eidAB), "RemoveEdge(eidAB)")

	// Verify forward adjacency removed.
	MustEqualBool(t, g.HasEdge(VertexA, VertexB), false, "HasEdge(A,B) after RemoveEdge(eidAB)")
	// Verify mirror adjacency removed in undirected graph.
	MustEqualBool(t, g.HasEdge(VertexB, VertexA), false, "HasEdge(B,A) after RemoveEdge(eidAB)")
	// Verify unrelated edge remains.
	MustEqualBool(t, g.HasEdge(VertexB, VertexC), true, "HasEdge(B,C) after RemoveEdge(eidAB)")
}

// TestGraph_StatsSnapshot VERIFIES GraphStats matches graph counts, flags, and directed/undirected tallies.
// Implementation:
//   - Stage 1: Create a weighted mixed graph with undirected default.
//   - Stage 2: Add vertices explicitly for deterministic VertexCount.
//   - Stage 3: Add one undirected edge and one directed override edge.
//   - Stage 4: Call Stats() and assert counts/flags/tallies.
//
// Behavior highlights:
//   - Locks in Stats() as a coherent diagnostic snapshot (O(V+E)).
//
// Inputs:
//   - None (uses package constants).
//
// Returns:
//   - None.
//
// Errors:
//   - Propagates sentinels from AddVertex/AddEdge.
//
// Determinism:
//   - Deterministic for a fixed single-threaded graph state.
//
// Complexity:
//   - Time O(V+E) if Stats walks catalogs, Space O(1) extra.
//
// Notes:
//   - DirectedDefault describes the graph default, not the presence of directed edges.
//
// AI-Hints:
//   - Treat Stats() as best-effort for metrics; avoid correctness-critical dependence under concurrent mutation.
func TestGraph_StatsSnapshot(t *testing.T) {
	// Stage 1: Create a weighted mixed graph with an explicit undirected default.
	g := core.NewGraph(core.WithDirected(false), core.WithWeighted(), core.WithMixedEdges())

	// Stage 2: Add vertices explicitly so VertexCount is deterministic.
	MustErrorNil(t, g.AddVertex(VertexA), "AddVertex(A) setup for Stats()")
	MustErrorNil(t, g.AddVertex(VertexB), "AddVertex(B) setup for Stats()")
	MustErrorNil(t, g.AddVertex(VertexC), "AddVertex(C) setup for Stats()")

	// Stage 3: Add one undirected edge (default) and one directed override edge.
	_, err := g.AddEdge(VertexA, VertexB, Weight1)
	MustErrorNil(t, err, "AddEdge(A,B,1) undirected default on mixed graph")
	_, err = g.AddEdge(VertexB, VertexC, Weight2, core.WithEdgeDirected(true))
	MustErrorNil(t, err, "AddEdge(B,C,2,WithEdgeDirected(true)) on mixed graph")

	// Stage 4: Read stats snapshot.
	s := g.Stats()

	// Stage 4: Counts must match public counters.
	MustEqualInt(t, s.VertexCount, g.VertexCount(), "Stats.VertexCount must match VertexCount()")
	MustEqualInt(t, s.EdgeCount, g.EdgeCount(), "Stats.EdgeCount must match EdgeCount()")

	// Stage 4: Flags must reflect constructor options.
	MustEqualBool(t, s.DirectedDefault, false, "Stats.DirectedDefault must be false for WithDirected(false)")
	MustEqualBool(t, s.Weighted, true, "Stats.Weighted must be true for WithWeighted()")
	MustEqualBool(t, s.MixedMode, true, "Stats.MixedMode must be true for WithMixedEdges()")
	MustEqualBool(t, s.AllowsMulti, false, "Stats.AllowsMulti must be false when WithMultiEdges() is not set")
	MustEqualBool(t, s.AllowsLoops, false, "Stats.AllowsLoops must be false when WithLoops() is not set")

	// Stage 4: Directed/undirected edge tallies must match this construction.
	MustEqualInt(t, s.DirectedEdgeCount, Count1, "Stats.DirectedEdgeCount must be 1 (one override-directed edge)")
	MustEqualInt(t, s.UndirectedEdgeCount, Count1, "Stats.UndirectedEdgeCount must be 1 (one default-undirected edge)")
	MustEqualInt(t, s.EdgeCount, Count2, "Stats.EdgeCount must be 2 in this setup")
}

// TestGraph_ClearPreservesFlagsAndResetsState VERIFIES Clear() empties the graph but preserves flags.
// Implementation:
//   - Stage 1: Create a configured graph and add at least one edge.
//   - Stage 2: Call Clear().
//   - Stage 3: Assert counts are zero and configuration flags are unchanged.
//   - Stage 4: Assert edge ID counter resets ("e1" is returned for the first new edge).
//
// Behavior highlights:
//   - Clear() is a topology reset, not a configuration reset.
//
// Inputs:
//   - None (uses package constants).
//
// Returns:
//   - None.
//
// Errors:
//   - Propagates any sentinels from AddEdge.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1) expected, Space O(1) extra.
//
// Notes:
//   - "e1" reset is a documented ID contract (types.go + doc.go).
//
// AI-Hints:
//   - Use Clear() to reuse configured graphs without reallocating options repeatedly.
func TestGraph_ClearPreservesFlagsAndResetsState(t *testing.T) {
	// Stage 1: Create a configured graph and add an edge to advance the internal ID counter.
	g := core.NewGraph(core.WithDirected(true), core.WithWeighted(), core.WithMultiEdges())

	// Add one edge to ensure the graph is non-empty before Clear().
	_, err := g.AddEdge(VertexA, VertexB, Weight5)
	MustErrorNil(t, err, "AddEdge(A,B,5) setup for Clear()")

	// Stage 2: Clear the graph.
	g.Clear()

	// Stage 3: Counts must be zero.
	MustEqualInt(t, g.VertexCount(), Count0, "VertexCount() after Clear()")
	MustEqualInt(t, g.EdgeCount(), Count0, "EdgeCount() after Clear()")

	// Stage 3: Flags must be preserved.
	MustEqualBool(t, g.Directed(), true, "Directed() must be preserved after Clear()")
	MustEqualBool(t, g.Weighted(), true, "Weighted() must be preserved after Clear()")
	MustEqualBool(t, g.Multigraph(), true, "Multigraph() must be preserved after Clear()")

	// Stage 4: First edge ID after Clear must reset to "e1".
	eid, err := g.AddEdge(VertexA, VertexB, Weight5)
	MustErrorNil(t, err, "AddEdge(A,B,5) after Clear()")
	MustEqualString(t, eid, EdgeIDFirst, "first edge ID after Clear() must be EdgeIDFirst")
}

// TestGraph_Queries VERIFIES HasEdge mirror behavior, Neighbors ordering, Vertices ordering, and Edges inventory count.
// Implementation:
//   - Stage 1: Create a weighted, loop-enabled graph.
//   - Stage 2: Add one undirected edge V1-V2 and one self-loop V1-V1.
//   - Stage 3: Assert HasEdge mirrors undirected adjacency.
//   - Stage 4: Assert Neighbors(V1) returns edges sorted by Edge.ID and includes exactly two edges.
//   - Stage 5: Assert Vertices() returns sorted vertex IDs.
//   - Stage 6: Assert Edges() returns exactly two edges in this setup.
//
// Behavior highlights:
//   - Locks in deterministic ordering contracts for Vertices() and Neighbors().
//
// Inputs:
//   - None (uses package constants).
//
// Returns:
//   - None.
//
// Errors:
//   - Propagates any sentinel errors from AddVertex/AddEdge/Neighbors.
//
// Determinism:
//   - Vertices() order is deterministic (sorted).
//   - Neighbors() order is deterministic (sorted by Edge.ID).
//
// Complexity:
//   - Time O(k log k) for sorting in Vertices/Neighbors, Space O(k) for returned slices.
//
// Notes:
//   - This test uses minimal topology to keep ordering expectations unambiguous.
//
// AI-Hints:
//   - To validate determinism, always check sortedness of returned IDs rather than relying on insertion order.
func TestGraph_Queries(t *testing.T) {
	// Stage 1: Use a weighted, loop-enabled graph.
	g := core.NewGraph(core.WithWeighted(), core.WithLoops())

	// Stage 2: Add one undirected edge V1–V2 and one self-loop V1–V1.
	MustErrorNil(t, g.AddVertex(VertexV1), "AddVertex(V1)")
	_, err := g.AddEdge(VertexV1, VertexV2, Weight0)
	MustErrorNil(t, err, "AddEdge(V1,V2,0)")
	_, err = g.AddEdge(VertexV1, VertexV1, Weight1)
	MustErrorNil(t, err, "AddEdge(V1,V1,1)")

	// Stage 3: Undirected edge must be mirrored for membership queries.
	MustEqualBool(t, g.HasEdge(VertexV2, VertexV1), true, "HasEdge(V2,V1) mirror for undirected edge")

	// Stage 4: Neighbors must return edges sorted by Edge.ID.
	nbs, err := g.Neighbors(VertexV1)
	MustErrorNil(t, err, "Neighbors(V1)")

	// Extract neighbor IDs in returned order.
	ids := make([]string, 0, len(nbs))
	for _, e := range nbs {
		ids = append(ids, e.ID)
	}

	// Validate sorted-by-ID contract.
	MustSortedStrings(t, ids, "Neighbors(V1) IDs must be sorted asc")
	// Validate neighbor count contract.
	MustEqualInt(t, len(ids), Count2, "Neighbors(V1) must contain exactly 2 edges (V1-V2 and V1-V1)")

	// Stage 5: Vertices() must return sorted IDs.
	vs := g.Vertices()
	MustSortedStrings(t, vs, "Vertices() must be sorted asc")

	// Stage 6: Edges inventory must include exactly two edges.
	ees := g.Edges()
	MustEqualInt(t, len(ees), Count2, "Edges() must contain exactly 2 edges in this setup")
}

// TestGraph_CloneEmptyAndClone VERIFIES CloneEmpty vertex-only behavior and Clone deep-copy behavior.
// Implementation:
//   - Stage 1: Build a graph with multi-edges, weights, and loops.
//   - Stage 2: CloneEmpty preserves vertices but drops all edges.
//   - Stage 3: Clone preserves vertices and edge count.
//   - Stage 4: Clone deep-copies Edge objects (pointers must not alias).
//
// Behavior highlights:
//   - Preserves functional correctness without mutating Edge objects in tests.
//
// Inputs:
//   - None (uses package constants).
//
// Returns:
//   - None.
//
// Errors:
//   - Propagates any errors from AddEdge/GetEdge.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(V+E) for Clone/CloneEmpty, Space O(V+E) for clone storage.
//
// Notes:
//   - Edge values are treated as read-only by contract; deep-copy is verified via pointer inequality.
//
// AI-Hints:
//   - Prefer pointer inequality to prove deep-copy while keeping the “Edge is read-only” contract intact.
func TestGraph_CloneEmptyAndClone(t *testing.T) {
	// Stage 1: Build a graph with multi-edges, weights, and loops.
	g := core.NewGraph(core.WithWeighted(), core.WithMultiEdges(), core.WithLoops())

	// Add two parallel edges so clone has non-trivial inventory.
	eid1, err := g.AddEdge(VertexA, VertexB, Weight1)
	MustErrorNil(t, err, "AddEdge(A,B,1)")
	_, err = g.AddEdge(VertexA, VertexB, Weight2)
	MustErrorNil(t, err, "AddEdge(A,B,2)")

	// Stage 2: CloneEmpty must preserve vertices and have zero edges.
	ce := g.CloneEmpty()
	MustSameStringSet(t, g.Vertices(), ce.Vertices(), "CloneEmpty preserves vertex IDs")
	MustEqualInt(t, len(ce.Edges()), 0, "CloneEmpty has zero edges")

	// Stage 3: Clone must preserve vertex IDs and edge count.
	c := g.Clone()
	MustSameStringSet(t, g.Vertices(), c.Vertices(), "Clone preserves vertex IDs")
	MustEqualInt(t, len(c.Edges()), len(g.Edges()), "Clone preserves edge count")

	// Stage 4: Deep-copy contract: cloned edge objects must not alias original objects.
	origEdge, err := g.GetEdge(eid1)
	MustErrorNil(t, err, "GetEdge(eid1) on original graph")
	cloneEdge, err := c.GetEdge(eid1)
	MustErrorNil(t, err, "GetEdge(eid1) on cloned graph")

	// Verify pointers differ (deep copy).
	MustEqualBool(t, origEdge != cloneEdge, true, "Clone deep-copy: edge pointers must not alias")
	// Verify scalar value preserved (exact int-like weight).
	MustEqualBool(t, origEdge.Weight == cloneEdge.Weight, true, "Clone deep-copy: edge weights must be preserved")
}

// TestGraph_LoopsAndDirection VERIFIES self-loop behavior in undirected vs directed graphs.
// Implementation:
//   - Stage 1: Undirected + loops enabled: AddEdge(X,X,0) yields exactly one neighbor and one edge.
//   - Stage 2: Directed + loops enabled: AddEdge(Y,Y,0) yields exactly one neighbor with Directed==true.
//
// Behavior highlights:
//   - Ensures self-loop inventory is not duplicated in undirected mirror logic.
//
// Inputs:
//   - None (uses package constants).
//
// Returns:
//   - None.
//
// Errors:
//   - Propagates errors from AddEdge/Neighbors.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1) setup + O(deg(v)) for Neighbors, Space O(deg(v)) for returned slice.
//
// Notes:
//   - This test assumes Edges() is de-duplicated by Edge.ID in undirected mode.
//
// AI-Hints:
//   - Self-loop handling is a common source of double-counting bugs; keep this anchor test stable.
func TestGraph_LoopsAndDirection(t *testing.T) {
	// Stage 1: Undirected loop-enabled graph.
	{
		// Create undirected graph with loops enabled.
		g := core.NewGraph(core.WithLoops())

		// Add self-loop on X.
		eid, err := g.AddEdge(VertexX, VertexX, Weight0)
		MustErrorNil(t, err, "AddEdge(X,X,0) undirected loops-enabled")

		// Neighbors(X) must return the loop exactly once.
		nbs, err := g.Neighbors(VertexX)
		MustErrorNil(t, err, "Neighbors(X) undirected loop")
		MustEqualInt(t, len(nbs), Count1, "Neighbors(X) undirected self-loop appears once")

		// Edges() must yield exactly one edge for a self-loop.
		ees := g.Edges()
		MustEqualInt(t, len(ees), Count1, "Edges() undirected self-loop yields one edge")
		MustEqualString(t, ees[0].ID, eid, "Edges()[0].ID equals AddEdge returned ID (undirected loop)")
	}

	// Stage 2: Directed loop-enabled graph.
	{
		// Create directed graph with loops enabled.
		g := core.NewGraph(core.WithLoops(), core.WithDirected(true))

		// Add self-loop on Y.
		eid, err := g.AddEdge(VertexY, VertexY, Weight0)
		MustErrorNil(t, err, "AddEdge(Y,Y,0) directed loops-enabled")

		// Neighbors(Y) must return the loop once.
		nbs, err := g.Neighbors(VertexY)
		MustErrorNil(t, err, "Neighbors(Y) directed loop")
		MustEqualInt(t, len(nbs), Count1, "Neighbors(Y) directed self-loop appears once")

		// Directed flag must be true for directed self-loop edge.
		MustEqualBool(t, nbs[0].Directed, true, "Neighbors(Y)[0].Directed must be true in directed graph")
		// ID must match AddEdge return.
		MustEqualString(t, nbs[0].ID, eid, "Neighbors(Y)[0].ID equals AddEdge returned ID (directed loop)")
	}
}

// TestGraph_MultiEdges VERIFIES parallel-edge semantics and weight preservation when enabled.
// Implementation:
//   - Stage 1: Create a multi-edge, weighted graph.
//   - Stage 2: Add two parallel edges A-B with weights 1 and 2.
//   - Stage 3: Assert IDs differ.
//   - Stage 4: Read edges by ID and assert weights match.
//
// Behavior highlights:
//   - Locks in ID uniqueness under multi-edge policy.
//
// Inputs:
//   - None (uses package constants).
//
// Returns:
//   - None.
//
// Errors:
//   - Propagates errors from AddEdge/GetEdge.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1) per AddEdge/GetEdge (implementation-dependent), Space O(1) extra.
//
// Notes:
//   - Weight equality is exact here because the inputs are integer-like constants.
//
// AI-Hints:
//   - Prefer validating edge attributes via GetEdge(id) instead of scanning Edges().
func TestGraph_MultiEdges(t *testing.T) {
	// Stage 1: Enable multi-edges and weights.
	g := core.NewGraph(core.WithMultiEdges(), core.WithWeighted())

	// Stage 2: Add parallel edges A-B with different weights.
	e1, err := g.AddEdge(VertexA, VertexB, Weight1)
	MustErrorNil(t, err, "AddEdge(A,B,1)")
	e2, err := g.AddEdge(VertexA, VertexB, Weight2)
	MustErrorNil(t, err, "AddEdge(A,B,2)")

	// Stage 3: IDs must differ.
	MustNotEqualString(t, e1, e2, "parallel edges must produce distinct IDs")

	// Stage 4: Validate stored weights by reading edges back by ID.
	edge1, err := g.GetEdge(e1)
	MustErrorNil(t, err, "GetEdge(e1)")
	edge2, err := g.GetEdge(e2)
	MustErrorNil(t, err, "GetEdge(e2)")

	// Compare weights exactly (integer-like float64 constants).
	MustEqualBool(t, edge1.Weight == float64(Weight1), true, "edge1 weight must equal 1")
	MustEqualBool(t, edge2.Weight == float64(Weight2), true, "edge2 weight must equal 2")
}

// TestGraph_HasEdgeUnknownVertices ANCHORS the contract: HasEdge must be safe for unknown vertex IDs.
// Implementation:
//   - Stage 1: Create an empty graph.
//   - Stage 2: Call HasEdge(U,V) and assert false (and no panic).
//
// Behavior highlights:
//   - Keeps HasEdge usable as a fast-path membership predicate.
//
// Inputs:
//   - None (uses package constants).
//
// Returns:
//   - None.
//
// Errors:
//   - None (pure predicate).
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1) expected, Space O(1).
//
// Notes:
//   - This test is intentionally minimal; any panic fails the test implicitly.
//
// AI-Hints:
//   - Keep HasEdge safe even when vertices are not created (avoid forced AddVertex in callers).
func TestGraph_HasEdgeUnknownVertices(t *testing.T) {
	// Stage 1: Querying an empty graph with unknown vertices must be safe and return false.
	g := core.NewGraph()
	// Stage 2: Validate predicate result.
	MustEqualBool(t, g.HasEdge(VertexU, VertexV), false, "HasEdge(U,V) on unknown vertices must be false")
}

// TestGraph_UnweightedViewCarriesNextEdgeID VERIFIES UnweightedView preserves edge-ID counter to avoid collisions.
// Implementation:
//   - Stage 1: Create a weighted source graph and add edges to advance ID counter.
//   - Stage 2: Build UnweightedView and assert it is unweighted.
//   - Stage 3: Assert copied edge weight is forced to zero in the view.
//   - Stage 4: AddEdge on the view must increase edge count and must not reuse an existing copied ID.
//   - Stage 5: Assert previously copied edge is still retrievable and unchanged in endpoints.
//
// Behavior highlights:
//   - Prevents ID collision bugs where new edges overwrite copied edges in the view.
//
// Inputs:
//   - None (uses package constants).
//
// Returns:
//   - None.
//
// Errors:
//   - Propagates errors from AddEdge/GetEdge.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(E) for view construction (copy), plus O(1) AddEdge, Space O(E) for copied edges.
//
// Notes:
//   - The contract is about counter carry-over, not ID formatting.
//
// AI-Hints:
//   - When debugging view/subgraph behaviors, always add a new edge after construction to probe ID collision risk.
func TestGraph_UnweightedViewCarriesNextEdgeID(t *testing.T) {
	// Stage 1: Build a weighted source graph and add edges to advance edge-ID counter.
	src := core.NewGraph(core.WithWeighted())

	// Add first weighted edge and retain its ID.
	eid1, err := src.AddEdge(VertexA, VertexB, Weight1)
	MustErrorNil(t, err, "src.AddEdge(A,B,1)")
	// Add second weighted edge to further advance the counter.
	_, err = src.AddEdge(VertexB, VertexC, Weight2)
	MustErrorNil(t, err, "src.AddEdge(B,C,2)")

	// Stage 2: Build the unweighted view and validate Weighted()==false.
	view := core.UnweightedView(src)
	MustEqualBool(t, view.Weighted(), false, "UnweightedView(src) must return an unweighted graph")

	// Stage 3: Validate forced weight=0 for copied edges.
	e1, err := view.GetEdge(eid1)
	MustErrorNil(t, err, "view.GetEdge(eid1)")
	MustEqualBool(t, e1.Weight == float64(Weight0), true, "UnweightedView must force copied edge weights to 0")

	// Stage 4: AddEdge must not reuse an existing edge ID in the view.
	before := view.EdgeCount()
	newID, err := view.AddEdge(VertexX, VertexY, Weight0)
	MustErrorNil(t, err, "view.AddEdge(X,Y,0)")
	MustEqualInt(t, view.EdgeCount(), before+Count1, "AddEdge on view must increase edge count by 1")
	MustNotEqualString(t, newID, eid1, "AddEdge on view must not collide with copied edge IDs")

	// Stage 5: Previously copied edge must still exist and keep endpoints.
	e1After, err := view.GetEdge(eid1)
	MustErrorNil(t, err, "view.GetEdge(eid1) after adding new edge")
	MustEqualString(t, e1After.From, e1.From, "copied edge From must be preserved after AddEdge on view")
	MustEqualString(t, e1After.To, e1.To, "copied edge To must be preserved after AddEdge on view")
}

// TestGraph_UnweightedViewFunctionalSnapshot VERIFIES UnweightedView preserves topology and forces weights to zero.
// Implementation:
//   - Stage 1: Build a weighted directed source graph with two edges.
//   - Stage 2: Build UnweightedView(src).
//   - Stage 3: Assert Weighted()==false and inventories (Vertices, Edge IDs) match.
//   - Stage 4: For each edge ID, assert From/To/Directed preserved and Weight==0 in the view.
//   - Stage 5: Assert mutating the view does not mutate the source graph.
//
// Behavior highlights:
//   - Non-destructive, deterministic transform suitable for unweighted algorithms.
//
// Inputs:
//   - None (uses package constants).
//
// Returns:
//   - None.
//
// Errors:
//   - Propagates errors from AddEdge/GetEdge.
//
// Determinism:
//   - Deterministic for a fixed input graph.
//
// Complexity:
//   - Time O(V+E) to build + O(E) to verify, Space O(V+E) for the view.
//
// Notes:
//   - Edge.ID preservation is a key debugging and stability feature.
//
// AI-Hints:
//   - Use UnweightedView when you need BFS/DFS-like behavior on a weighted input.
func TestGraph_UnweightedViewFunctionalSnapshot(t *testing.T) {
	// Stage 1: Build weighted directed source graph.
	src := core.NewGraph(core.WithDirected(true), core.WithWeighted())

	// Add two edges with non-zero weights.
	id1, err := src.AddEdge(VertexA, VertexB, Weight1)
	MustErrorNil(t, err, "src.AddEdge(A,B,1)")
	id2, err := src.AddEdge(VertexB, VertexC, Weight7)
	MustErrorNil(t, err, "src.AddEdge(B,C,7)")

	// Stage 2: Build view.
	view := core.UnweightedView(src)

	// Stage 3: Inventories must match; view must be unweighted.
	MustEqualBool(t, view.Weighted(), false, "UnweightedView must return Weighted()==false")
	MustSameStringSet(t, view.Vertices(), src.Vertices(), "UnweightedView must preserve vertex ID set")
	MustSameStringSet(t, ExtractEdgeIDs(view.Edges()), ExtractEdgeIDs(src.Edges()), "UnweightedView must preserve edge ID set")

	// Stage 4: Per-edge topology and directedness must be preserved; weight forced to zero.
	ids := []string{id1, id2}
	for _, eid := range ids {
		orig, err := src.GetEdge(eid)
		MustErrorNil(t, err, "src.GetEdge(eid)")
		cpy, err := view.GetEdge(eid)
		MustErrorNil(t, err, "view.GetEdge(eid)")

		MustEqualString(t, cpy.From, orig.From, "UnweightedView must preserve Edge.From")
		MustEqualString(t, cpy.To, orig.To, "UnweightedView must preserve Edge.To")
		MustEqualBool(t, cpy.Directed == orig.Directed, true, "UnweightedView must preserve Edge.Directed")
		MustEqualBool(t, cpy.Weight == float64(Weight0), true, "UnweightedView must force Edge.Weight==0")
	}

	// Stage 5: Mutating view must not mutate src.
	before := src.EdgeCount()
	_, err = view.AddEdge(VertexX, VertexY, Weight0)
	MustErrorNil(t, err, "view.AddEdge(X,Y,0)")
	MustEqualInt(t, src.EdgeCount(), before, "mutating view must not change src.EdgeCount()")
}

// TestGraph_InducedSubgraphCarriesNextEdgeID VERIFIES InducedSubgraph preserves edge-ID counter to avoid collisions.
// Implementation:
//   - Stage 1: Create a weighted source graph and add edges A-B and B-C.
//   - Stage 2: Induce subgraph keeping only {A,B}; assert it keeps exactly one edge (A-B).
//   - Stage 3: AddEdge on subgraph must increase edge count and must not reuse an existing kept ID.
//   - Stage 4: Assert previously kept edge is still retrievable and unchanged in endpoints.
//
// Behavior highlights:
//   - Prevents ID collision bugs in induced subgraphs similar to view-collision class.
//
// Inputs:
//   - None (uses package constants).
//
// Returns:
//   - None.
//
// Errors:
//   - Propagates errors from AddEdge/GetEdge.
//
// Determinism:
//   - Deterministic for fixed input graph.
//
// Complexity:
//   - Time O(E) to build induced subgraph (copy/filter), Space O(E_sub).
//
// Notes:
//   - This test asserts the kept-edge count for keep={A,B} based on the constructed source graph.
//
// AI-Hints:
//   - When changing subgraph policies, keep the “collision probe” pattern: build → add edge → re-read kept edge.
func TestGraph_InducedSubgraphCarriesNextEdgeID(t *testing.T) {
	// Stage 1: Create a weighted source graph with two edges.
	src := core.NewGraph(core.WithWeighted())

	// Add edge A-B and retain its ID (it must be included in keep={A,B}).
	eidAB, err := src.AddEdge(VertexA, VertexB, Weight1)
	MustErrorNil(t, err, "src.AddEdge(A,B,1)")
	// Add edge B-C (will be excluded by keep={A,B}).
	_, err = src.AddEdge(VertexB, VertexC, Weight2)
	MustErrorNil(t, err, "src.AddEdge(B,C,2)")

	// Stage 2: Keep only {A,B}; induced subgraph must keep exactly the A-B edge.
	keep := map[string]bool{VertexA: true, VertexB: true}
	sub := core.InducedSubgraph(src, keep)

	MustEqualInt(t, sub.EdgeCount(), Count1, "InducedSubgraph keep={A,B} must keep exactly 1 edge")

	eAB, err := sub.GetEdge(eidAB)
	MustErrorNil(t, err, "sub.GetEdge(eidAB)")

	// Stage 3: AddEdge must not reuse an existing kept edge ID.
	before := sub.EdgeCount()
	newID, err := sub.AddEdge(VertexA, VertexD, Weight3)
	MustErrorNil(t, err, "sub.AddEdge(A,D,3)")
	MustEqualInt(t, sub.EdgeCount(), before+Count1, "AddEdge on subgraph must increase edge count by 1")
	MustNotEqualString(t, newID, eidAB, "new subgraph edge ID must not collide with kept eidAB")

	// Stage 4: Previously kept edge must still exist and keep endpoints.
	eABAfter, err := sub.GetEdge(eidAB)
	MustErrorNil(t, err, "sub.GetEdge(eidAB) after adding new edge")
	MustEqualString(t, eABAfter.From, eAB.From, "kept edge From must be preserved after AddEdge on subgraph")
	MustEqualString(t, eABAfter.To, eAB.To, "kept edge To must be preserved after AddEdge on subgraph")
}

// TestGraph_InducedSubgraphFunctionalCorrectness VERIFIES InducedSubgraph keeps exactly requested vertices and internal edges.
// Implementation:
//   - Stage 1: Create a weighted graph and add edges A-B, B-C, and A-C.
//   - Stage 2: Induce subgraph with keep={A,C}.
//   - Stage 3: Assert vertices are exactly {A,C}.
//   - Stage 4: Assert only the A-C edge remains and preserves weight/directness.
//
// Behavior highlights:
//   - InducedSubgraph is a deterministic filter that does not mutate the source graph.
//
// Inputs:
//   - None (uses package constants).
//
// Returns:
//   - None.
//
// Errors:
//   - Propagates sentinel errors from AddEdge/GetEdge.
//
// Determinism:
//   - Deterministic for fixed input and keep-set.
//
// Complexity:
//   - Time O(V+E) to filter/copy, Space O(V_sub+E_sub) for subgraph storage.
//
// Notes:
//   - This test intentionally uses a keep-set that excludes a “bridge” vertex to ensure edges are filtered correctly.
//
// AI-Hints:
//   - Prefer InducedSubgraph when you need an isolated working set of vertices for an algorithm stage.
func TestGraph_InducedSubgraphFunctionalCorrectness(t *testing.T) {
	// Stage 1: Build a weighted graph with a triangle A-B-C.
	src := core.NewGraph(core.WithWeighted())

	_, err := src.AddEdge(VertexA, VertexB, Weight1)
	MustErrorNil(t, err, "src.AddEdge(A,B,1)")
	_, err = src.AddEdge(VertexB, VertexC, Weight2)
	MustErrorNil(t, err, "src.AddEdge(B,C,2)")
	idAC, err := src.AddEdge(VertexA, VertexC, Weight3)
	MustErrorNil(t, err, "src.AddEdge(A,C,3)")

	// Stage 2: Induce keep={A,C}.
	keep := map[string]bool{VertexA: true, VertexC: true}
	sub := core.InducedSubgraph(src, keep)

	// Stage 3: Vertices must be exactly {A,C}.
	MustSameStringSet(t, sub.Vertices(), []string{VertexA, VertexC}, "InducedSubgraph must keep exactly {A,C}")

	// Stage 4: Only A-C edge remains and preserves weight.
	MustEqualInt(t, sub.EdgeCount(), Count1, "InducedSubgraph keep={A,C} must keep exactly 1 edge")
	e, err := sub.GetEdge(idAC)
	MustErrorNil(t, err, "sub.GetEdge(idAC)")
	MustEqualString(t, e.From, VertexA, "kept edge must have From==A")
	MustEqualString(t, e.To, VertexC, "kept edge must have To==C")
	MustEqualBool(t, e.Weight == float64(Weight3), true, "kept edge must preserve Weight==3")

	// Edges incident to removed vertex must not exist (and predicate must be safe).
	MustEqualBool(t, sub.HasEdge(VertexA, VertexB), false, "sub.HasEdge(A,B) must be false when B is not kept")
	MustEqualBool(t, sub.HasEdge(VertexB, VertexC), false, "sub.HasEdge(B,C) must be false when B is not kept")
}

// TestGraph_EdgesAreSorted ANCHORS the contract: Edges() must be sorted by Edge.ID ascending.
// Implementation:
//   - Stage 1: Create a multi-edge weighted graph and add multiple parallel edges.
//   - Stage 2: Extract IDs from Edges() and assert sortedness.
//
// Behavior highlights:
//   - Deterministic ordering simplifies downstream algorithms and stable tests.
//
// Inputs:
//   - None (uses package constants).
//
// Returns:
//   - None.
//
// Errors:
//   - Propagates errors from AddEdge.
//
// Determinism:
//   - Deterministic: Edges() order is stable and sorted by ID.
//
// Complexity:
//   - Time O(E log E) if Edges() sorts internally, Space O(E) for returned slice.
//
// Notes:
//   - This test checks sortedness only; uniqueness is covered by AddEdge/multi-edge tests and ID uniqueness tests.
//
// AI-Hints:
//   - If you change edge-ID representation, keep ordering deterministic (lexicographic over IDs is simplest).
func TestGraph_EdgesAreSorted(t *testing.T) {
	// Stage 1: Create a multigraph and add multiple edges so sorting is observable.
	g := core.NewGraph(core.WithMultiEdges(), core.WithWeighted())

	// Add three parallel edges.
	_, err := g.AddEdge(VertexA, VertexB, Weight1)
	MustErrorNil(t, err, "AddEdge(A,B,1)")
	_, err = g.AddEdge(VertexA, VertexB, Weight2)
	MustErrorNil(t, err, "AddEdge(A,B,2)")
	_, err = g.AddEdge(VertexA, VertexB, Weight3)
	MustErrorNil(t, err, "AddEdge(A,B,3)")

	// Stage 2: Validate stable sorted order by Edge.ID (lexicographic).
	ees := g.Edges()
	ids := ExtractEdgeIDs(ees)
	MustSortedStrings(t, ids, "Edges() IDs must be sorted asc")
}

// TestGraph_AddEdge_WithID_OK verifies that AddEdge honors WithID for a unique, non-empty edge identifier and that
// the created edge is addressable through the edge catalog under that exact ID.
//
// Implementation:
//   - Stage 1: Construct a default Graph (unweighted, undirected).
//   - Stage 2: AddEdge(A,B,0,WithID(customID)) and assert it succeeds.
//   - Stage 3: Assert the returned edge ID equals customID (no auto-ID fallback).
//   - Stage 4: Lookup via GetEdge(customID) and assert Edge.ID == customID.
//   - Stage 5: Assert HasEdge(A,B) reflects the insertion.
//
// Behavior highlights:
//   - Confirms that WithID bypasses auto-generated IDs for that edge.
//   - Confirms catalog key consistency: returned ID == GetEdge lookup key == Edge.ID field.
//
// Inputs:
//   - None.
//
// Returns:
//   - None.
//
// Errors:
//   - The test fails if AddEdge returns an error, if the returned ID mismatches,
//     if GetEdge cannot retrieve the edge, or if HasEdge does not observe the insertion.
//
// Determinism:
//   - Deterministic: explicit ID and deterministic catalog operations; no iteration-order dependencies.
//
// Complexity:
//   - Time O(1) average per operation, Space O(1) (excluding graph allocations).
//
// Notes:
//   - Uses Weight0 to satisfy the default unweighted graph constraint.
//
// AI-Hints:
//   - Use WithID to create stable external references (golden tests, trace correlation, interop).
func TestGraph_AddEdge_WithID_OK(t *testing.T) {
	g := core.NewGraph()
	eid, err := g.AddEdge(VertexA, VertexB, Weight0, core.WithID("customID123"))
	MustErrorNil(t, err, "AddEdge(A,B,0,WithID) should succeed with unique ID")
	MustEqualString(t, eid, "customID123", "returned edge ID should match provided ID")
	// Check the edge is retrievable and has the correct ID
	e, err := g.GetEdge("customID123")
	MustErrorNil(t, err, "GetEdgeByID(customID123) after AddEdge")
	MustNotNil(t, e, "GetEdgeByID should return an Edge")
	MustEqualString(t, e.ID, "customID123", "Edge.ID should be the custom ID")
	MustEqualBool(t, g.HasEdge(VertexA, VertexB), true, "HasEdge(A,B) should reflect the new edge")
}
