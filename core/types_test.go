// SPDX-License-Identifier: MIT
// Package core_test verifies core.Graph configuration, identity contracts, and cloning semantics.
//
// Purpose:
//   - Lock in option flags, vertex lifecycle rules, ID uniqueness under concurrency.
//   - Demonstrate read-only map snapshots and deep-copy behavior (no pointer aliasing).

package core_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/katalvlaran/lvlath/core"
)

// TestGraph_Options ASSERTS GraphOption flags are applied correctly.
//
// Implementation:
//   - Stage 1: Build a feature-rich graph via NewGraphFull().
//   - Stage 2: Assert Directed defaults to false.
//   - Stage 3: Assert Weighted is enabled.
//   - Stage 4: Assert empty vertex ID is absent.
//   - Stage 5: Assert WithDirected(true) overrides.
//   - Stage 6: Assert multi-edge policy rejects duplicates when disabled.
//
// Behavior highlights:
//   - Documents option semantics explicitly.
//
// Inputs:
//   - None.
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal on any contract mismatch.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Multi-edge rejection is a sentinel contract (ErrMultiEdgeNotAllowed).
//
// AI-Hints:
//   - Prefer option tests to stay minimal: assert flags and one representative behavior per flag.
func TestGraph_Options(t *testing.T) {
	g := NewGraphFull()

	MustFalse(t, g.Directed(), "Directed() default must be false (undirected)")
	MustTrue(t, g.Weighted(), "Weighted() must be true on NewGraphFull")
	MustFalse(t, g.HasVertex(VertexEmpty), "HasVertex(empty) must be false")

	dg := core.NewGraph(core.WithDirected(true))
	MustTrue(t, dg.Directed(), "WithDirected(true) must set Directed()==true")

	sg := core.NewGraph()
	_, err := sg.AddEdge(VertexX, VertexY, Weight0)
	MustNoError(t, err, "AddEdge(X,Y,0) first on default graph")

	_, err = sg.AddEdge(VertexX, VertexY, Weight0)
	MustErrorIs(t, err, core.ErrMultiEdgeNotAllowed, "AddEdge(X,Y,0) second on default graph")
}

// TestGraph_VertexLifecycle ASSERTS AddVertex/HasVertex/RemoveVertex invariants.
//
// Implementation:
//   - Stage 1: Create a graph.
//   - Stage 2: Reject empty ID on AddVertex.
//   - Stage 3: Add a vertex and validate presence.
//   - Stage 4: Duplicate AddVertex is no-op.
//   - Stage 5: RemoveVertex rejects empty and missing IDs.
//   - Stage 6: Remove existing vertex and validate absence.
//
// Behavior highlights:
//   - Locks in sentinel errors for empty/missing IDs.
//
// Inputs:
//   - None.
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal on mismatch.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(k log k) due to Vertices() sorting during count checks (implementation-dependent).
//
// Notes:
//   - This test relies on Vertices() being stable and safe.
//
// AI-Hints:
//   - Keep vertex IDs short and consistent to avoid noise in failure output.
func TestGraph_VertexLifecycle(t *testing.T) {
	g := NewGraphFull()

	err := g.AddVertex(VertexEmpty)
	MustErrorIs(t, err, core.ErrEmptyVertexID, "AddVertex(empty)")

	MustNoError(t, g.AddVertex(VertexV1), "AddVertex(V1)")
	MustTrue(t, g.HasVertex(VertexV1), "HasVertex(V1) after AddVertex(V1)")

	before := len(g.Vertices())
	MustNoError(t, g.AddVertex(VertexV1), "AddVertex(V1) duplicate")
	after := len(g.Vertices())
	MustEqualInt(t, after, before, "duplicate AddVertex(V1) must not change vertex count")

	err = g.RemoveVertex("Z")
	MustErrorIs(t, err, core.ErrVertexNotFound, "RemoveVertex(Z missing)")

	err = g.RemoveVertex(VertexEmpty)
	MustErrorIs(t, err, core.ErrEmptyVertexID, "RemoveVertex(empty)")

	MustNoError(t, g.RemoveVertex(VertexV1), "RemoveVertex(V1)")
	MustFalse(t, g.HasVertex(VertexV1), "HasVertex(V1) after RemoveVertex(V1)")
}

// TestGraph_AtomicEdgeIDs ASSERTS concurrent AddEdge yields unique IDs.
//
// Implementation:
//   - Stage 1: Create feature-rich graph (multi-edge enabled).
//   - Stage 2: Spawn NAtomicEdgeIDs goroutines adding edges A->B with varying weights.
//   - Stage 3: Goroutines send errors/IDs to channels (no *testing.T inside goroutines).
//   - Stage 4: Assert no errors, and set size equals NAtomicEdgeIDs.
//
// Behavior highlights:
//   - Locks in uniqueness property of edge IDs under contention.
//
// Inputs:
//   - None.
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal if any AddEdge fails or uniqueness is violated.
//
// Determinism:
//   - Schedule nondeterministic; uniqueness assertion deterministic.
//
// Complexity:
//   - Time O(N), Space O(N).
//
// Notes:
//   - This test does not assert the *format* of IDs (only uniqueness/non-emptiness).
//
// AI-Hints:
//   - If you later formalize ID format, extend this test with a parser and pattern assertions.
func TestGraph_AtomicEdgeIDs(t *testing.T) {
	g := NewGraphFull()

	idCh := make(chan string, NAtomicEdgeIDs)
	errCh := make(chan error, NAtomicEdgeIDs)

	var wg sync.WaitGroup
	wg.Add(NAtomicEdgeIDs)

	var i int
	for i = 0; i < NAtomicEdgeIDs; i++ {
		go func(i int) {
			defer wg.Done()

			eid, err := g.AddEdge(VertexA, VertexB, float64(i))
			if err != nil {
				errCh <- err
				return
			}
			if eid == "" {
				errCh <- fmt.Errorf("empty edge ID returned")
				return
			}
			idCh <- eid
		}(i)
	}

	wg.Wait()
	close(idCh)
	close(errCh)

	MustNoErrorsFromChan(t, errCh, "Atomic edge IDs")

	ids := make(map[string]struct{}, NAtomicEdgeIDs)

	for eid := range idCh {
		ids[eid] = struct{}{}
	}

	MustEqualInt(t, len(ids), NAtomicEdgeIDs, "unique edge IDs count")
}

// TestGraph_AdjacencyMap ASSERTS HasEdge is safe and respects add/remove.
//
// Implementation:
//   - Stage 1: Create a graph.
//   - Stage 2: Verify HasEdge is false on empty graph.
//   - Stage 3: Add an edge, verify HasEdge true.
//   - Stage 4: Remove edge, verify HasEdge false.
//
// Behavior highlights:
//   - Ensures membership queries are safe even before vertices exist.
//
// Inputs:
//   - None.
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal on mismatch.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1) per query under nested-map lookup design, Space O(1).
//
// Notes:
//   - If HasEdge policy changes, update this test only when contract changes.
//
// AI-Hints:
//   - Keep HasEdge usable as a fast-path predicate (must never panic on unknown IDs).
func TestGraph_AdjacencyMap(t *testing.T) {
	g := NewGraphFull()

	MustFalse(t, g.HasEdge(VertexP, VertexQ), "HasEdge(P,Q) on empty graph must be false")

	eid, err := g.AddEdge(VertexP, VertexQ, Weight0)
	MustNoError(t, err, "AddEdge(P,Q,0)")
	MustTrue(t, g.HasEdge(VertexP, VertexQ), "HasEdge(P,Q) after AddEdge(P,Q)")

	MustNoError(t, g.RemoveEdge(eid), "RemoveEdge(eid)")
	MustFalse(t, g.HasEdge(VertexP, VertexQ), "HasEdge(P,Q) after RemoveEdge")
}

// TestGraph_CloneMethods ASSERTS CloneEmpty and Clone semantics.
//
// Implementation:
//   - Stage 1: Create a graph and add representative edges.
//   - Stage 2: CloneEmpty keeps vertices but drops edges.
//   - Stage 3: Clone preserves edge IDs and does not alias edge objects.
//
// Behavior highlights:
//   - Deep-copy is verified by pointer inequality (no mutation of returned Edge objects).
//
// Inputs:
//   - None.
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal on mismatch.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(V+E), Space O(V+E) for cloning.
//
// Notes:
//   - Edge objects returned by Graph APIs are treated as read-only by contract.
//
// AI-Hints:
//   - Verify deep-copy by pointer identity, not by mutating Weight (avoids contract violations).
func TestGraph_CloneMethods(t *testing.T) {
	g := NewGraphFull()

	eidXY, err := g.AddEdge(VertexX, VertexY, Weight1)
	MustNoError(t, err, "AddEdge(X,Y,1)")
	_, err = g.AddEdge(VertexY, VertexY, Weight2)
	MustNoError(t, err, "AddEdge(Y,Y,2)")

	ce := g.CloneEmpty()
	MustSameStringSet(t, g.Vertices(), ce.Vertices(), "CloneEmpty preserves vertices")
	MustEqualInt(t, len(ce.Edges()), 0, "CloneEmpty has no edges")

	c := g.Clone()
	MustSameStringSet(t, g.Vertices(), c.Vertices(), "Clone preserves vertices")
	MustSameStringSet(t, ExtractEdgeIDs(g.Edges()), ExtractEdgeIDs(c.Edges()), "Clone preserves edge IDs")

	orig, err := g.GetEdge(eidXY)
	MustNoError(t, err, "GetEdge(eidXY) on original")

	cl, err := c.GetEdge(eidXY)
	MustNoError(t, err, "GetEdge(eidXY) on clone")

	MustTrue(t, orig != cl, "Clone deep-copy: edge pointers must not alias")
}

// TestGraph_VerticesMapReadOnly ASSERTS VerticesMap returns a safe snapshot.
//
// Implementation:
//   - Stage 1: Add vertex Z.
//   - Stage 2: Read VerticesMap snapshot.
//   - Stage 3: Mutate snapshot.
//   - Stage 4: Assert original graph is unchanged.
//
// Behavior highlights:
//   - Prevents external mutation through returned maps.
//
// Inputs:
//   - None.
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal if snapshot is not read-only.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(V) for snapshot copy, Space O(V).
//
// Notes:
//   - This locks in “defensive copy” behavior for maps.
//
// AI-Hints:
//   - Prefer snapshot APIs when you want a safe iteration without holding graph locks.
func TestGraph_VerticesMapReadOnly(t *testing.T) {
	g := NewGraphFull()

	MustNoError(t, g.AddVertex("Z"), "AddVertex(Z)")

	vm := g.VerticesMap()
	vm["NEW"] = &core.Vertex{ID: "NEW"}

	MustFalse(t, g.HasVertex("NEW"), "VerticesMap must be read-only snapshot")
}

// TestGraph_HasVertexConcurrency ASSERTS concurrent HasVertex/AddVertex does not panic.
//
// Implementation:
//   - Stage 1: Create graph.
//   - Stage 2: Spawn M goroutines adding vertices and M goroutines reading HasVertex.
//   - Stage 3: Wait; test passes if no panic.
//
// Behavior highlights:
//   - This is a race/panic detector; validate with `go test -race`.
//
// Inputs:
//   - None.
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal only if a panic occurs (implicit).
//
// Determinism:
//   - Nondeterministic schedule; expected stability.
//
// Complexity:
//   - Time O(M), Space O(1) extra.
//
// Notes:
//   - This test intentionally does not assert final counts: it targets safety, not outcome.
//
// AI-Hints:
//   - Keep this test lightweight; rely on -race to detect unsynchronized access.
func TestGraph_HasVertexConcurrency(t *testing.T) {
	g := NewGraphFull()

	const M = 50

	var wg sync.WaitGroup
	wg.Add(2 * M)

	var i int
	for i = 0; i < M; i++ {
		go func(i int) {
			defer wg.Done()
			_ = g.AddVertex(fmt.Sprintf("V%d", i))
		}(i)

		go func(i int) {
			defer wg.Done()
			_ = g.HasVertex(fmt.Sprintf("V%d", i))
		}(i)
	}

	wg.Wait()
}
