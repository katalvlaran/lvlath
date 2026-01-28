// SPDX-License-Identifier: MIT

// Package core - deterministic, thread-safe in-memory graphs for serious work.
//
// -----------------------------------------------------------------------------
// -- WHAT ---------------------------------------------------------------------
//
//	A single, composable Graph type G=(V,E) with predictable iteration order,
//	strict sentinel errors, zero hidden globals, and explicit functional options.
//
// -- WHY ----------------------------------------------------------------------
//   - Determinism First: all public enumerations have a documented, stable order.
//     This removes “heisenbugs” in tests, reproducibility drifts and flaky CI.
//   - Concurrency Without Drama: two RWMutexes (muVert, muEdgeAdj) minimize
//     contention across vertex/edge heavy codepaths while keeping the model simple.
//   - Mixed-Mode Mastery: per-edge directedness overrides are explicit, legal
//     only in mixed mode, and guarded by a sentinel (no silent fallbacks).
//   - Identity Discipline: monotonic textual Edge.ID (“e1”, “e2”, …) - perfect
//     for stable logs, diffs, golden tests, and distributed trace heuristics.
//   - Practical API Surface: one Graph with knobs (Directed/Weighted/Multi/Loops/
//     Mixed), instead of an exploding matrix of types. Algorithms compose cleanly.
//   - No Magic, No Surprises: sentinels only (errors.Is), no fmt-wrapping, no
//     ambient globals, no soft-fallbacks - if a capability is disabled, you see ErrX.
//
// -- WHEN ---------------------------------------------------------------------
//   - Need reproducible graph results (order-sensitive logic, golden tests).
//   - Need to combine directed/undirected/loops/multiedges in 1 instance.
//   - Need safe concurrent reads/writes with a clear lock model.
//   - Need stable Edge IDs across clones/views for debugging/analytics.
//
// -----------------------------------------------------------------------------
// -- DESIGN INVARIANTS --------------------------------------------------------
//
//  1. Deterministic ordering (public contract):
//     Vertices()            → IDs sorted lex asc
//     Edges()               → by Edge.ID asc
//     - Order is lexicographic over strings (e.g., "e10" sorts between "e1" and "e2").
//     NeighborIDs(id)       → IDs sorted lex asc
//     Algorithms rely on this stability; tests must assume it.
//  2. Sentinel errors only; compare with errors.Is. No fmt-wrapping of sentinels.
//  3. No mutation of caller-owned inputs; views/clones are explicit and named.
//  4. Concurrency model: two RWMutexes
//     muVert    - vertices map + config flags
//     muEdgeAdj - edges catalog + adjacency nested maps
//
// -----------------------------------------------------------------------------
// -- CONFIGURATION (GraphOption) - explicit, deterministic flags --------------
//
//   - WithDirected(defaultDirected bool)
//     Sets the default edge orientation (true=directed).
//
//   - WithMixedEdges()
//     Allows per-edge overrides (WithEdgeDirected). Without it, attempts to
//     pass EdgeOption return ErrMixedEdgesNotAllowed (no silent degradation).
//
//   - WithWeighted()
//     Permits non-zero edge weights. Otherwise AddEdge(weight≠0) → ErrBadWeight.
//
//   - WithMultiEdges()
//     Permits parallel edges between identical endpoints. Otherwise a second
//     AddEdge(from,to) → ErrMultiEdgeNotAllowed.
//
//   - WithLoops()
//     Permits self-loops (from==to). Otherwise AddEdge(v,v) → ErrLoopNotAllowed.
//
// -- EdgeOption ---------------------------------------------------------------
//
//   - WithEdgeDirected(directed bool)
//     Per-edge orientation override (legal only in mixed mode).
//
// -- Helper constructor -------------------------------------------------------
//   - NewMixedGraph(opts ...GraphOption) = NewGraph(WithMixedEdges(), opts...)
//
// -----------------------------------------------------------------------------
// -- ERROR SET (sentinels) ----------------------------------------------------
//   - ErrEmptyVertexID        - empty vertex ID is illegal
//   - ErrVertexNotFound       - vertex does not exist
//   - ErrEdgeNotFound         - edge does not exist
//   - ErrBadWeight            - non-zero weight in an unweighted graph
//   - ErrLoopNotAllowed       - self-loop when loops are disabled
//   - ErrMultiEdgeNotAllowed  - parallel edge when multi-edges are disabled
//   - ErrMixedEdgesNotAllowed - per-edge overrides when mixed mode is disabled
//
// -----------------------------------------------------------------------------
// -- LIFECYCLE MAPS -----------------------------------------------------------
//
// Graph lifecycle (configuration → build → query → transform):
//
//	g := core.NewGraph(WithDirected(false), WithWeighted())
//	g.AddVertex("A"); g.AddVertex("B")
//	eid, err := g.AddEdge("A", "B", 10)  // undirected by default here
//	es  := g.Edges()                     // deterministic (Edge.ID asc)
//	deg := g.Degree("A")                 // (in,out,undirected)
//	s   := g.Stats()                     // O(V+E) snapshot of flags & counts
//	g2  := g.Clone()                     // deep copy, preserves edge IDs
//	uv  := core.UnweightedView(g)        // same topology, weight=0, weighted=false
//
// Vertex lifecycle:
//
//   - Create      → AddVertex(id)
//   - Check       → HasVertex(id)
//   - Remove      → RemoveVertex(id)        // removes incident edges deterministically
//   - Enumerate   → Vertices()              // sorted
//   - Inspect     → VerticesMap()           // shallow copy (ID → *Vertex)
//
// Edge lifecycle:
//
//   - Create      → AddEdge(from,to,weight, opts...)   // mixed-mode gate enforced
//   - Check       → HasEdge(from,to)                   // O(1) membership
//   - Inspect     → GetEdge(edgeID), Edges()           // Edges() sorted by ID
//   - Remove      → RemoveEdge(edgeID)                 // mirrors handled automatically
//   - Filter      → FilterEdges(pred)                  // O(E) + cleanup
//
// Adjacency & Neighborhood:
//
//   - Neighbors(id)   → []*Edge        // sorted by Edge.ID
//   - NeighborIDs(id) → []string       // unique, sorted
//   - AdjacencyList() → map[id][]edgeID (per-vertex lists sorted by Edge.ID)
//
// Cloning & Views:
//
//   - CloneEmpty() - copy flags + vertices, no edges, carry nextEdgeID
//   - Clone()      - deep copy (flags + vertices + edges + adjacency), carry nextEdgeID
//   - UnweightedView(g) - same topology, weight=0, Weighted()=false.
//     Preserves Edge.ID values and carries the edge-ID counter to prevent collisions
//     if you later add edges to the derived graph.
//   - InducedSubgraph(g, keep) - keep subset of vertices + incident edges.
//     Preserves Edge.ID values and carries the edge-ID counter for the same reason.
//
// -----------------------------------------------------------------------------
// -- COMPLEXITY SUMMARY -------------------------------------------------------
//
//	AddVertex / HasVertex / HasEdge            O(1) amortized
//	AddEdge / RemoveEdge                       O(1) amortized (+ O(1) membership)
//	Vertices / Edges                           O(V log V) / O(E log E) for ordering
//	Neighbors / NeighborIDs                    O(d log d)  (d = degree(id))
//	AdjacencyList                              O(V+E) to assemble + per-vertex sort
//	Degree                                     O(d)
//	CloneEmpty / Clone                         O(V) / O(V+E)
//	Clear                                      O(1) (map reinit + counter reset)
//	Stats                                      O(V+E)
//
// -----------------------------------------------------------------------------
// -- AI-HINT (LLM/Copilot/ChatGPT/Claude/Gemini/Qwen guidance) ----------------
//
//   - Always assume deterministic ordering when you need stable tests or diffs.
//     Prefer Vertices()/Edges()/NeighborIDs() outputs directly (already sorted).
//
//   - To use per-edge directedness, you MUST construct the graph with WithMixedEdges(),
//     otherwise AddEdge(..., WithEdgeDirected(...)) → ErrMixedEdgesNotAllowed.
//     g := core.NewGraph(WithDirected(false), WithMixedEdges())
//     _, _ = g.AddEdge("A","B", 0, WithEdgeDirected(true))
//
//   - Unweighted graphs must add edges with weight==0; else → ErrBadWeight.
//     g := core.NewGraph()                // unweighted by default
//     _, err := g.AddEdge("A","B", 5)     // => ErrBadWeight
//
//   - No silent fallbacks: loops/multi-edges disabled → explicit sentinels.
//     Check with errors.Is(err, core.ErrLoopNotAllowed) etc.
//
//   - Cloning preserves textual Edge.ID sequence via nextEdgeID carry-over.
//     If you Clone() then AddEdge(), the new edge ID continues monotonic growth.
//
//   - Stats() returns a snapshot suitable for diagnostics and metrics.
//     If the graph is mutated concurrently, Stats() may reflect a moving target;
//     treat it as best-effort and avoid building correctness-critical logic on it.
//
//   - Concurrency model is simple: public methods take care of locking internally.
//     Do not hold external locks around core methods; let the package manage locks.
//
//   - If you only need a subset or weightless view - use InducedSubgraph/UnweightedView.
//     They do NOT mutate the input graph.
//
//     They preserve Edge.ID values and keep future AddEdge() IDs unique by carrying
//     the internal edge-ID counter forward.
//
// -----------------------------------------------------------------------------
// -- See also: docs/CORE.md for algorithmic notes, proofs, and extended examples.
package core
