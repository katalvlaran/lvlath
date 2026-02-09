// SPDX-License-Identifier: MIT

// Package core - deterministic, thread-safe in-memory graphs for serious work.
//
// -----------------------------------------------------------------------------
// -- WHAT ---------------------------------------------------------------------
//
// A single, composable Graph type G=(V,E) with predictable iteration order,
// strict sentinel errors, explicit configuration flags, and no ambient globals.
//
// The package provides a stable, auditable contract suitable for algorithms that
// require reproducibility (tests, golden outputs, deterministic traversals) and
// a simple concurrency model for safe parallel reads and controlled mutations.
//
// -----------------------------------------------------------------------------
// -- WHY ----------------------------------------------------------------------
//
//   - Determinism First:
//     Public enumeration order is documented and stable. This removes flakiness
//     from order-sensitive tests and prevents reproducibility drift.
//
//   - Concurrency Without Drama:
//     Two RWMutexes (muVert, muEdgeAdj) separate vertex/config state from
//     topology state (edges/adjacency) to reduce contention while remaining simple.
//
//   - Mixed-Mode Mastery:
//     Per-edge directedness overrides are explicit and legal only when mixed-mode
//     is enabled. Violations return ErrMixedEdgesNotAllowed (no silent fallback).
//
//   - Identity Discipline:
//     Default Edge IDs are monotonic textual identifiers ("e1","e2",...).
//     This is useful for stable logs, diffs, and external correlation.
//
//   - Practical API Surface:
//     One Graph with explicit flags (Directed/Weighted/Multi/Loops/Mixed) instead
//     of multiple graph types that fragment algorithms.
//
//   - No Magic, No Surprises:
//     Sentinel errors only (errors.Is), no fmt-wrapping of sentinels, and no
//     hidden global state. If a capability is disabled, you receive ErrX.
//
// -----------------------------------------------------------------------------
// -- WHEN ---------------------------------------------------------------------
//
//   - Need reproducible graph results (order-sensitive logic, golden tests).
//   - Need to combine directed/undirected/loops/multi-edges within one instance.
//   - Need safe concurrent reads with a clear lock model and controlled mutations.
//   - Need stable Edge IDs across clones/views for debugging/analytics.
//
// -----------------------------------------------------------------------------
// -- DESIGN INVARIANTS --------------------------------------------------------
//
//  1. Deterministic ordering (public contract):
//     - Vertices()            → IDs sorted lexicographically ascending.
//     - Edges()               → by Edge.ID ascending (string lex order).
//     Note: lex order is over strings, so "e10" sorts between "e1" and "e2".
//     Auto-generated IDs are monotonic in numeric suffix, but API ordering is
//     defined by string sort order.
//     - NeighborIDs(id)       → unique IDs sorted lexicographically ascending.
//     - Neighbors(id)         → edges sorted by Edge.ID ascending.
//
//  2. Sentinel errors only; compare with errors.Is.
//     Exported operations return stable package-level sentinels. Callers must not
//     rely on string matching.
//
//  3. No mutation of caller-owned inputs.
//     Views/clones are explicit and named; they never mutate the source graph.
//
//  4. Concurrency model: two RWMutexes.
//     - muVert    guards vertices catalog and configuration flags (policy state).
//     - muEdgeAdj guards edge catalog and adjacency nested maps (topology state).
//
// -----------------------------------------------------------------------------
// -- CONFIGURATION (GraphOption) ----------------------------------------------
//
// GraphOption values are applied only during construction (NewGraph/NewMixedGraph).
// After construction, flags are immutable.
//
//   - WithDirected(defaultDirected bool)
//     Sets the default edge orientation for newly created edges.
//
//   - WithMixedEdges()
//     Enables per-edge directedness overrides (WithEdgeDirected).
//     Without it, AddEdge(..., WithEdgeDirected(...)) → ErrMixedEdgesNotAllowed.
//
//   - WithWeighted()
//     Permits non-zero edge weights.
//     Without it, AddEdge(weight!=0) → ErrBadWeight.
//
//   - WithMultiEdges()
//     Permits parallel edges between identical endpoints.
//     Without it, a second AddEdge(from,to,...) → ErrMultiEdgeNotAllowed.
//
//   - WithLoops()
//     Permits self-loops (from==to).
//     Without it, AddEdge(v,v,...) → ErrLoopNotAllowed.
//
// Helper constructor:
//   - NewMixedGraph(opts ...GraphOption) = NewGraph(WithMixedEdges(), opts...)
//     Ensures mixed-mode is enabled before any other options are applied.
//
// -----------------------------------------------------------------------------
// -- EDGE OPTIONS (EdgeOption) ------------------------------------------------
//
// EdgeOption values configure a single edge during AddEdge. Options are applied
// sequentially in call order. The first option returning an error aborts AddEdge
// without mutating the graph.
//
//   - WithEdgeDirected(directed bool)
//     Overrides Directed for this edge.
//     Contract: allowed only when MixedEdges()==true; otherwise ErrMixedEdgesNotAllowed.
//
//   - WithID(id string)
//     Assigns a custom Edge.ID for the new edge.
//     Contract:
//
//   - id must be non-empty, else ErrEmptyEdgeID.
//
//   - id must be globally unique within the graph, else ErrEdgeIDConflict.
//     Note:
//
//   - If id matches the canonical auto-ID form "eN", the graph advances its
//     internal auto-ID counter to avoid future collisions.
//
// -----------------------------------------------------------------------------
// -- ERROR SET (sentinels) ----------------------------------------------------
//
//   - ErrEmptyVertexID        - empty vertex ID is illegal.
//   - ErrVertexNotFound       - referenced vertex does not exist.
//   - ErrEdgeNotFound         - referenced edge does not exist.
//   - ErrBadWeight            - non-zero weight in an unweighted graph.
//   - ErrLoopNotAllowed       - self-loop when loops are disabled.
//   - ErrMultiEdgeNotAllowed  - parallel edge when multi-edges are disabled.
//   - ErrMixedEdgesNotAllowed - per-edge directed override when mixed-mode is disabled.
//   - ErrEmptyEdgeID          - empty edge ID is illegal (WithID / SetEdgeID).
//   - ErrEdgeIDConflict       - edge ID collision (WithID / SetEdgeID).
//
// -----------------------------------------------------------------------------
// -- LIFECYCLE MAPS -----------------------------------------------------------
//
// Graph lifecycle (configuration → build → query → transform):
//
//	g := core.NewGraph(WithDirected(false), WithWeighted())
//	_ = g.AddVertex("A")
//	_ = g.AddVertex("B")
//	eid, err := g.AddEdge("A", "B", 10)  // undirected by default here
//	es  := g.Edges()                     // deterministic (Edge.ID asc, lex order)
//	in, out, und, _ := g.Degree("A")     // (in,out,undirected)
//	s   := g.Stats()                     // O(V+E) snapshot of flags & counts
//	g2  := g.Clone()                     // deep copy, preserves edge IDs
//	uv  := core.UnweightedView(g)        // same topology, weight=0, weighted=false
//
// Vertex lifecycle:
//
//   - Create      → AddVertex(id)
//   - Check       → HasVertex(id)
//   - Remove      → RemoveVertex(id)        // removes incident edges deterministically
//   - Enumerate   → Vertices()              // sorted lex asc
//   - Inspect     → VerticesMap()           // shallow copy (ID → *Vertex)
//
// Edge lifecycle:
//
//   - Create      → AddEdge(from,to,weight, opts...)   // policy enforced by sentinels
//   - Check       → HasEdge(from,to)                   // O(1) membership
//   - Inspect     → GetEdge(edgeID), Edges()           // Edges() sorted by ID
//   - Rename      → SetEdgeID(oldID,newID)             // updates catalog + adjacency atomically under lock
//   - Remove      → RemoveEdge(edgeID)                 // mirrors handled automatically
//   - Filter      → FilterEdges(pred)                  // O(E) + cleanup
//
// Adjacency & Neighborhood:
//
//   - Neighbors(id)     → []*Edge         // sorted by Edge.ID (lex asc)
//   - NeighborIDs(id)   → []string        // unique, sorted lex asc
//   - AdjacencyList()   → map[id][]edgeID // per-vertex lists sorted by Edge.ID
//
// Cloning & Views:
//
//   - CloneEmpty() - copy flags + vertices, no edges, carry nextEdgeID.
//   - Clone()      - deep copy (flags + vertices + edges + adjacency), carry nextEdgeID.
//   - UnweightedView(g) - same topology, weight=0, Weighted()=false.
//     Preserves Edge.ID values and carries the edge-ID counter to prevent collisions
//     if you later add edges to the derived graph.
//   - InducedSubgraph(g, keep) - keep subset of vertices + incident edges.
//     Preserves Edge.ID values and carries the edge-ID counter for the same reason.
//
// -----------------------------------------------------------------------------
// -- COMPLEXITY SUMMARY -------------------------------------------------------
//
//	AddVertex / HasVertex / HasEdge              O(1) amortized
//	AddEdge / RemoveEdge                         O(1) amortized
//	Vertices / Edges                             O(V log V) / O(E log E) for ordering
//	Neighbors / NeighborIDs                      O(d log d)  (d = degree(id))
//	AdjacencyList                                O(V+E) assemble + per-vertex sort
//	Degree                                       O(d)
//	CloneEmpty / Clone                           O(V) / O(V+E)
//	Clear                                        O(1) (map reinit + counter reset)
//	Stats                                        O(V+E)
//
// -----------------------------------------------------------------------------
// -- AI-HINT (LLM/Copilot/ChatGPT/Claude/Gemini/Qwen guidance) ----------------
//
//   - Deterministic ordering is part of the public contract.
//     Prefer Vertices()/Edges()/NeighborIDs() outputs directly (already sorted).
//
//   - To use per-edge directedness, you MUST enable mixed-mode at construction time:
//     g := core.NewMixedGraph(WithDirected(false))
//     _, err := g.AddEdge("A","B", 0, WithEdgeDirected(true))
//
//   - Unweighted graphs must add edges with weight==0; otherwise AddEdge returns ErrBadWeight:
//     g := core.NewGraph()             // unweighted by default
//     _, err := g.AddEdge("A","B", 5)  // => ErrBadWeight
//
//   - No silent fallbacks: loops/multi-edges/mixed-mode violations return explicit sentinels.
//     Use errors.Is(err, core.ErrLoopNotAllowed) etc.
//
//   - WithID provides stable external references:
//     _, err := g.AddEdge("A","B", 0, WithID("stable-edge-1"))
//     If you choose IDs of the form "eN", the graph advances its internal counter past N.
//
//   - Cloning and views preserve textual Edge.ID sequence via nextEdgeID carry-over.
//     If you Clone() then AddEdge(), the new edge ID continues monotonic growth.
//
//   - Stats() is an O(V+E) snapshot suitable for diagnostics and tests.
//     If the graph is mutated concurrently, treat Stats() as best-effort telemetry,
//     not as a correctness-critical synchronization primitive.
//
//   - Concurrency model: public methods manage locking internally.
//     Avoid holding external locks around core methods to prevent lock-order coupling.
//
//   - Use InducedSubgraph/UnweightedView to derive subproblems without mutating the input graph.
//     Derived graphs preserve Edge.ID values and keep future AddEdge() IDs unique by carrying
//     the internal edge-ID counter forward.
//
// -----------------------------------------------------------------------------
// -- See also: docs/CORE.md for algorithmic notes, proofs, and extended examples.
package core
