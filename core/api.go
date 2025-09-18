// SPDX-License-Identifier: MIT
//
// File: api.go
// Role: Thin, deterministic public facade exposing constructors and read-only getters.
// Policy:
//   - No algorithms or hidden state here.
//   - Concurrency model and invariants are defined in types.go/doc.go.
//   - Every exported function documents complexity and locking strategy.
// AI-HINT (file):
//   - Use NewMixedGraph(...) before passing WithEdgeDirected(...) to AddEdge.
//   - Stats() is O(V+E) snapshot; rely on it for quick admissions/diagnostics.

package core

// NOTE: This file exposes a thin, well-documented public API facade
//       (constructors and read-only getters) on top of the core types.
//       It intentionally contains *no* algorithmic complexity or hidden state.
//       All operations are deterministic and concurrency-safe per the locking
//       model described in types.go (muVert, muEdgeAdj).

// NewMixedGraph constructs a new Graph with mixed-mode enabled and then applies
// any additional GraphOption values deterministically (left-to-right).
//
// Rationale:
//   - Mixed-mode allows per-edge directedness overrides via WithEdgeDirected.
//   - This helper is sugar for NewGraph(WithMixedEdges(), opts...).
//   - Keeping option order stable preserves determinism for all callers.
//
// Complexity: O(1) allocations + O(len(opts)) option applications.
// Concurrency: safe to call from multiple goroutines; it creates a fresh object.
func NewMixedGraph(opts ...GraphOption) *Graph {
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

// Weighted reports whether the graph treats edge weights as meaningful.
//
// Contract:
//   - Returns the construction-time flag (immutable after NewGraph).
//   - Read is protected by muVert for consistent visibility.
//   - No allocations, no mutations.
//
// Complexity: O(1).
// Concurrency: safe; uses read lock.
func (g *Graph) Weighted() bool {
	// AI-HINT: If this returns false, AddEdge with non-zero weight returns ErrBadWeight.
	g.muVert.RLock()         // acquire read lock on vertex/config state
	defer g.muVert.RUnlock() // release lock via defer for clarity and safety

	return g.weighted
}

// Directed reports whether new edges default to directed.
//
// Contract:
//   - Returns the construction-time flag (immutable after NewGraph).
//   - Read is protected by muVert for consistent visibility.
//   - No allocations, no mutations.
//
// Complexity: O(1).
// Concurrency: safe; uses read lock.
func (g *Graph) Directed() bool {
	// AI-HINT: Default orientation for new edges; does NOT count current directed edges.
	g.muVert.RLock()         // acquire read lock on vertex/config state
	defer g.muVert.RUnlock() // release lock via defer for clarity and safety

	return g.directed
}

// Looped reports whether the graph's edges could be looped.
//
// Contract:
//   - Returns the construction-time flag (immutable after NewGraph).
//   - Read is protected by muVert for consistent visibility.
//   - No allocations, no mutations.
//
// Complexity: O(1).
// Concurrency: safe; uses read lock.
func (g *Graph) Looped() bool {
	// AI-HINT: If false, AddEdge(v,v,...) returns ErrLoopNotAllowed.
	g.muVert.RLock()         // acquire read lock on vertex/config state
	defer g.muVert.RUnlock() // release lock via defer for clarity and safety

	return g.allowLoops
}

// Multigraph reports whether this Graph permits parallel edges (multi-edges).
//
// Contract:
//   - Returns the construction-time flag (immutable after NewGraph).
//   - Read is protected by muVert for consistent visibility.
//   - No allocations, no mutations.
//
// Complexity: O(1).
// Concurrency: safe; uses read lock.
func (g *Graph) Multigraph() bool {
	// AI-HINT: If false, adding a second edge between same endpoints returns ErrMultiEdgeNotAllowed.
	g.muVert.RLock()         // acquire read lock on vertex/config state
	defer g.muVert.RUnlock() // ensure lock is released even on panic (there shouldn't be any)

	return g.allowMulti // return the immutable configuration flag
}

// MixedEdges reports whether this Graph permits per-edge directedness overrides.
//
// Contract:
//   - Returns the construction-time flag (immutable after NewGraph).
//   - Read is protected by muVert for consistent visibility.
//   - No allocations, no mutations.
//
// Complexity: O(1).
// Concurrency: safe; uses read lock.
func (g *Graph) MixedEdges() bool {
	// AI-HINT: If false, per-edge overrides (WithEdgeDirected) are rejected with ErrMixedEdgesNotAllowed.
	g.muVert.RLock()         // acquire read lock on vertex/config state
	defer g.muVert.RUnlock() // release lock via defer for clarity and safety

	return g.allowMixed // return mixed-mode configuration flag
}

// Stats produces an O(V+E) read-only summary of the graph's configuration and size.
//
// Semantics:
//   - DirectedDefault mirrors the graph's default edge orientation.
//   - Weighted/AllowsMulti/AllowsLoops/MixedMode are construction-time flags.
//   - VertexCount/EdgeCount reflect catalog sizes at the time of the call.
//   - DirectedEdgeCount / UndirectedEdgeCount are derived by scanning edge catalog.
//
// Locking strategy:
//   - Acquire muVert.RLock to read flags and vertex count, then release it.
//   - Acquire muEdgeAdj.RLock to scan edges and compute edge counters.
//   - Never hold both locks at once to avoid lock-ordering issues and minimize contention.
//
// Complexity: O(V+E).
// Concurrency: safe; uses read locks only and allocates a small result struct.
func (g *Graph) Stats() *GraphStats {
	// AI-HINT: Deterministic, read-only summary for assertions and tests.
	//          DirectedEdgeCount/UndirectedEdgeCount scan edge catalog once (O(E)).

	// First phase: capture configuration flags and vertex count under muVert.
	g.muVert.RLock() // lock config/vertices for consistent reads
	stats := GraphStats{
		DirectedDefault: g.directed,      // record default orientation
		Weighted:        g.weighted,      // record weight policy
		AllowsMulti:     g.allowMulti,    // record multi-edge policy
		AllowsLoops:     g.allowLoops,    // record loop policy
		MixedMode:       g.allowMixed,    // record mixed-mode policy
		VertexCount:     len(g.vertices), // snapshot of vertex catalog size
		// Edge counters will be filled in second phase under muEdgeAdj.
	}
	g.muVert.RUnlock() // release muVert ASAP to minimize contention

	// Second phase: compute edge counters under muEdgeAdj.
	g.muEdgeAdj.RLock()            // lock edge catalog and adjacency for consistent scanning
	stats.EdgeCount = len(g.edges) // snapshot of edge catalog size
	var e *Edge
	for _, e = range g.edges { // single pass over all edges (O(E))
		if e.Directed { // classify by Directed flag
			stats.DirectedEdgeCount++ // directed edge encountered
		} else {
			stats.UndirectedEdgeCount++ // undirected edge encountered
		}
	}
	g.muEdgeAdj.RUnlock() // release edges/adjacency lock

	// Return a pointer to the fully populated, immutable-by-convention summary.
	return &stats
}
