// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran
//
// File: api.go
// Role: Thin, deterministic public facade exposing constructors and read-only getters.
// Policy:
//   - No algorithms or hidden state here.
//   - Concurrency model and invariants are defined in doc.go and types.go.
//   - Detached container/value surfaces and zero-copy alias accessors are both
//   intentional parts of the public contract; callers must choose explicitly
//   based on their ownership needs.
//
// AI-HINT (file):
//   - Use NewMixedGraph(...) before passing WithEdgeDirected(...) to AddEdge.
//   - Stats() is an O(V+E) summary surface for quick admissions/diagnostics.

package core

// NOTE: This file exposes a thin, well-documented public API facade
//       (constructors and read-only getters) on top of the core types.
//       It intentionally contains *no* algorithmic complexity or hidden state.
//       All operations are deterministic and concurrency-safe per the locking
//       model described in types.go (muVert, muEdgeAdj).

// NewMixedGraph creates a new Graph that allows per-edge directedness overrides via EdgeOption,
// while preserving deterministic option application order.
//
// Implementation:
//   - Stage 1: Prepend WithMixedEdges() to the caller-provided options.
//   - Stage 2: Delegate to NewGraph(...) to validate and apply options deterministically.
//
// Behavior highlights:
//   - Enables WithEdgeDirected(...) on AddEdge; without mixed-mode this is rejected.
//   - Does not mutate the caller's opts slice (no hidden side-effects).
//
// Inputs:
//   - opts: additional GraphOption values applied after enabling mixed-mode.
//
// Returns:
//   - *Graph: a fresh configured instance with allowMixed enabled.
//   - error: nil on success; otherwise a stable sentinel error.
//
// Errors:
//   - ErrNilGraphOption: if opts contains a nil GraphOption value.
//   - Any stable sentinel returned by a provided GraphOption via NewGraph.
//
// Determinism:
//   - Options are applied left-to-right, with WithMixedEdges() always first.
//
// Complexity:
//   - Time O(len(opts)), Space O(len(opts)) for the composed options slice.
//
// Notes:
//   - Prefer this constructor when you plan to mix directed and undirected edges in one graph.
//
// AI-Hints:
//   - Use NewMixedGraph(...) instead of remembering to prepend WithMixedEdges() manually.
func NewMixedGraph(opts ...GraphOption) (*Graph, error) {
	// AI-HINT: Prefer this constructor if you plan per-edge directed overrides.
	//          Without mixed mode, AddEdge(..., WithEdgeDirected(...)) returns ErrMixedEdgesNotAllowed.

	// Prepend WithMixedEdges() as the very first option to guarantee that
	// any later per-edge assumptions (in future methods) see allowMixed == true.
	// We allocate a new slice to avoid mutating the caller's slice (no side-effects).
	mixed := make([]GraphOption, 0, len(opts)+1) // allocate exact capacity to avoid reallocation
	mixed = append(mixed, WithMixedEdges())      // first option sets mixed-mode flag
	mixed = append(mixed, opts...)               // then apply caller-provided options deterministically
	// Delegate to NewGraph to keep construction logic centralized and uniform.
	return NewGraph(mixed...)
}

// Weighted reports the construction-time "weighted" capability flag.
// If false, AddEdge rejects non-zero weights with ErrBadWeight.
//
// Implementation:
//   - Stage 1: Acquire muVert read lock to observe configuration consistently.
//   - Stage 2: Return the immutable flag value.
//
// Behavior highlights:
//   - Pure query: no mutation, no iteration, no allocations.
//
// Returns:
//   - bool: true if non-zero weights are permitted.
//
// Errors:
//   - None.
//
// Determinism:
//   - Deterministic for a fixed graph instance (flags are immutable after construction).
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - This reports a policy flag, not whether any stored edge currently has Weight != 0.
//
// AI-Hints:
//   - Gate weighted algorithms by g.Weighted() before reading edge.Weight.
func (g *Graph) Weighted() bool {
	// AI-HINT: If this returns false, AddEdge with non-zero weight returns ErrBadWeight.
	g.muVert.RLock()         // acquire read lock on vertex/config state
	defer g.muVert.RUnlock() // release lock via defer for clarity and safety

	return g.weighted
}

// Directed reports the graph-wide default directedness applied to newly created edges.
// Per-edge overrides require mixed-mode (MixedEdges()==true).
//
// Implementation:
//   - Stage 1: Acquire muVert read lock to observe configuration consistently.
//   - Stage 2: Return the immutable default directedness flag.
//
// Behavior highlights:
//   - Pure policy query: does not scan edges.
//
// Returns:
//   - bool: true if new edges default to directed orientation.
//
// Errors:
//   - None.
//
// Determinism:
//   - Deterministic for a fixed graph instance.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - This does not indicate whether the graph currently contains directed edges
//     (use HasDirectedEdges() or Stats().DirectedEdgeCount for that).
//
// AI-Hints:
//   - Use g.Directed() to decide default edge semantics when generating topology programmatically.
func (g *Graph) Directed() bool {
	// AI-HINT: Default orientation for new edges; does NOT count current directed edges.
	g.muVert.RLock()         // acquire read lock on vertex/config state
	defer g.muVert.RUnlock() // release lock via defer for clarity and safety

	return g.directed
}

// Looped reports whether self-loops (from==to) are permitted by policy.
// If false, AddEdge(v,v,...) rejects the operation with ErrLoopNotAllowed.
//
// Implementation:
//   - Stage 1: Acquire muVert read lock to observe configuration consistently.
//   - Stage 2: Return the immutable loops policy flag.
//
// Returns:
//   - bool: true if self-loops are permitted.
//
// Errors:
//   - None.
//
// Determinism:
//   - Deterministic for a fixed graph instance.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - This is a policy flag; existing self-loops can only exist if this was enabled at creation time.
//
// AI-Hints:
//   - Gate loop-sensitive algorithms by g.Looped() before assuming v->v edges may exist.
func (g *Graph) Looped() bool {
	// AI-HINT: If false, AddEdge(v,v,...) returns ErrLoopNotAllowed.
	g.muVert.RLock()         // acquire read lock on vertex/config state
	defer g.muVert.RUnlock() // release lock via defer for clarity and safety

	return g.allowLoops
}

// Multigraph reports whether parallel edges between the same endpoints are permitted by policy.
// If false, AddEdge(from,to,...) rejects duplicates with ErrMultiEdgeNotAllowed.
//
// Implementation:
//   - Stage 1: Acquire muVert read lock to observe configuration consistently.
//   - Stage 2: Return the immutable multi-edge policy flag.
//
// Returns:
//   - bool: true if parallel edges are permitted.
//
// Errors:
//   - None.
//
// Determinism:
//   - Deterministic for a fixed graph instance.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Multi-edge checks in AddEdge are membership checks over adjacency buckets.
//
// AI-Hints:
//   - If you need multi-edges, enable WithMultiEdges() at construction time; this is immutable later.
func (g *Graph) Multigraph() bool {
	// AI-HINT: If false, adding a second edge between same endpoints returns ErrMultiEdgeNotAllowed.
	g.muVert.RLock()         // acquire read lock on vertex/config state
	defer g.muVert.RUnlock() // ensure lock is released even on panic (there shouldn't be any)

	return g.allowMulti // return the immutable configuration flag
}

// MixedEdges reports whether per-edge Directed overrides are permitted via EdgeOption
// (specifically WithEdgeDirected(...)) during AddEdge.
//
// Implementation:
//   - Stage 1: Acquire muVert read lock to observe configuration consistently.
//   - Stage 2: Return the immutable mixed-mode policy flag.
//
// Returns:
//   - bool: true if per-edge Directed overrides are permitted.
//
// Errors:
//   - None.
//
// Determinism:
//   - Deterministic for a fixed graph instance.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - If false, AddEdge(..., WithEdgeDirected(...)) returns ErrMixedEdgesNotAllowed.
//
// AI-Hints:
//   - Prefer NewMixedGraph(...) when you intend to mix directed and undirected edges in one instance.
func (g *Graph) MixedEdges() bool {
	// AI-HINT: If false, per-edge overrides (WithEdgeDirected) are rejected with ErrMixedEdgesNotAllowed.
	g.muVert.RLock()         // acquire read lock on vertex/config state
	defer g.muVert.RUnlock() // release lock via defer for clarity and safety

	return g.allowMixed // return mixed-mode configuration flag
}

// Stats produces a deterministic, read-only diagnostic summary of configuration flags,
// catalog sizes, and edge directedness counts.
//
// Snapshot category:
//   - BestEffortSnapshot under concurrent mutation.
//   - Strict for a fixed graph state with no concurrent mutation.
//
// Implementation:
//   - Stage 1: Acquire muVert.RLock(), snapshot immutable configuration flags and vertex count,
//     then release muVert.RLock().
//   - Stage 2: Acquire muEdgeAdj.RLock(), snapshot edge count and scan the edge catalog once,
//     then release muEdgeAdj.RLock().
//   - Stage 3: Return the populated GraphStats value object.
//
// Behavior highlights:
//   - Avoids holding both locks simultaneously, reducing contention for diagnostics.
//   - The returned *GraphStats is detached from the graph and immutable by convention.
//   - Under concurrent mutation, vertex/config fields and edge counters may come from
//     different read phases; callers must not treat it as a linearizable topology snapshot.
//
// Returns:
//   - *GraphStats: detached summary of policy flags and catalog counters.
//
// Errors:
//   - None.
//
// Determinism:
//   - Deterministic for a fixed graph state.
//   - Under concurrent mutation, each phase is internally consistent, but the whole
//     result is best-effort telemetry rather than a strict transaction snapshot.
//
// Complexity:
//   - Time O(E), Space O(1) plus the returned struct.
//
// Notes:
//   - DirectedDefault reports the default policy for new edges, not whether directed edges exist.
//   - Use HasDirectedEdges or DirectedEdgeCount when gating algorithms by current topology.
//
// AI-Hints:
//   - Use Stats() to gate algorithms quickly (e.g., ensure Weighted==true before reading weights).
//   - Do not use Stats() as a strict concurrent admission primitive unless the caller owns
//     external synchronization.
func (g *Graph) Stats() *GraphStats {
	// AI-HINT: Deterministic, read-only summary for assertions and tests.
	//          DirectedEdgeCount/UndirectedEdgeCount scan edge catalog once (O(E)).

	// First phase: capture configuration flags and vertex count under muVert.
	g.muVert.RLock()            // lock config/vertices for consistent reads
	defer g.muVert.RUnlock()    // release muVert ASAP to minimize contention
	g.muEdgeAdj.RLock()         // lock edge catalog and adjacency for consistent scanning
	defer g.muEdgeAdj.RUnlock() // release edges/adjacency lock

	stats := GraphStats{
		DirectedDefault: g.directed,      // record default orientation
		Weighted:        g.weighted,      // record weight policy
		AllowsMulti:     g.allowMulti,    // record multi-edge policy
		AllowsLoops:     g.allowLoops,    // record loop policy
		MixedMode:       g.allowMixed,    // record mixed-mode policy
		VertexCount:     len(g.vertices), // snapshot of vertex catalog size
		EdgeCount:       len(g.edges),    // snapshot of edge catalog size
	}

	// Second phase: compute edge counters under muEdgeAdj.
	var e *Edge
	for _, e = range g.edges { // single pass over all edges (O(E))
		if e.Directed { // classify by Directed flag
			stats.DirectedEdgeCount++ // directed edge encountered
		} else {
			stats.UndirectedEdgeCount++ // undirected edge encountered
		}
	}

	// Return a pointer to the fully populated, immutable-by-convention summary.
	return &stats
}
