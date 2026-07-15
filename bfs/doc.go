// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package bfs implements deterministic breadth-first search and deterministic
// weak-connectivity discovery over core.Graph.
//
// -----------------------------------------------------------------------------
// -- WHAT ---------------------------------------------------------------------
//
// bfs provides two public traversal facades:
//
//   - BFS(g, startID, opts...)
//     Deterministic single-source BFS on an unweighted graph.
//
//   - Components(ctx, g)
//     Deterministic weakly-connected component discovery under an undirected
//     relation, even when the graph itself is directed.
//
// Result is the public traversal artifact. It exposes:
//
//   - StartID  - the source vertex of the BFS invocation.
//   - Order    - dequeue and visit order.
//   - Depth    - hop distance in edge count.
//   - Parent   - shortest-path tree or forest predecessor links.
//   - Visited  - discovery set, marked at enqueue time.
//   - Skipped  - count of neighbor relations rejected by FilterNeighbor.
//
// The package is designed as a reusable algorithmic kernel for downstream
// packages that require stable traversal semantics, reproducible outputs,
// errors.Is-compatible failure handling, and operationally safe queue behavior.
//
// -----------------------------------------------------------------------------
// -- WHY ----------------------------------------------------------------------
//
//   - Determinism First:
//     BFS preserves the neighbor order surfaced by core.Graph.NeighborIDs.
//     It does not add a second sorting layer or invent a hidden tie-break rule.
//
//   - Correct Unweighted Semantics:
//     Depth is measured in edge count only. The package intentionally rejects
//     weighted graphs so callers do not accidentally ask BFS the wrong question.
//
//   - Partial Results Are First-Class:
//     Cancellation and runtime interruption return partial result plus error,
//     which makes debugging, monitoring, and controlled early-stop pipelines
//     substantially safer than “error-only” traversal APIs.
//
//   - Policy Without Topology Mutation:
//     Option hooks and FilterNeighbor allow controlled traversal policy without
//     rewriting or cloning the input graph just to express a runtime rule.
//
//   - Memory Discipline:
//     The queue is implemented with head-index dequeue and slot clearing, which
//     avoids the retention behavior of queue = queue[1:] in Go.
//
// -----------------------------------------------------------------------------
// -- DOMAIN SCOPE -------------------------------------------------------------
//
// bfs is intentionally scoped to:
//
//   - single-source BFS from a known StartID,
//   - unweighted shortest-hop traversal,
//   - deterministic full-coverage forest traversal via WithFullTraversal,
//   - deterministic weak connectivity via Components.
//
// The package answers questions such as:
//
//   - “What is reachable from this start vertex?”
//   - “How many hops away is this vertex?”
//   - “What is one shortest hop path to this destination?”
//   - “What are the weakly-connected regions of this graph?”
//
// -----------------------------------------------------------------------------
// -- NON-GOALS ----------------------------------------------------------------
//
// bfs is intentionally NOT:
//
//   - a weighted shortest-path package,
//   - an all-shortest-paths enumerator,
//   - a strong-connectivity or SCC package,
//   - a snapshot-isolated concurrent traversal engine.
//
// Consequences:
//
//   - If the business meaning depends on weight, cost, latency, capacity, or
//     risk, BFS is mathematically the wrong tool.
//   - If you need every shortest path rather than one shortest-path certificate,
//     Parent and PathTo are insufficient by design.
//   - If you need strong connectivity on directed graphs, Components is the
//     wrong primitive because it computes weak connectivity.
//   - If the graph is mutated concurrently during traversal, correctness and
//     reproducibility are not guaranteed.
//
// -----------------------------------------------------------------------------
// -- DETERMINISM --------------------------------------------------------------
//
// Determinism is a package-level contract.
//
//  1. Neighbor ordering
//     BFS processes neighbors in the exact order returned by:
//
//     g.NeighborIDs(u)
//
//     bfs itself does not sort neighbors and does not inspect edges to derive
//     a different ordering.
//
//  2. Visit definition
//     “Visit” means dequeue.
//
//     - Order is dequeue and visit order.
//     - OnVisit is called at dequeue time.
//     - OnDequeue runs immediately before OnVisit.
//
//  3. FullTraversal root ordering
//     WithFullTraversal(), BFS first traverses the component reachable from
//     StartID, then selects secondary roots by iterating g.Vertices() in the
//     order defined by core.
//
// Determinism therefore depends on the enumeration contract of core. If core
// returns deterministic NeighborIDs and Vertices, bfs preserves that contract.
//
// -----------------------------------------------------------------------------
// -- DEPTH SEMANTICS ----------------------------------------------------------
//
// Depth always means unweighted hop distance in edge count.
//
//   - Depth[StartID] == 0
//
//   - For any discovered non-root vertex v with parent p:
//
//     Depth[v] = Depth[p] + 1
//
//   - Parent is recorded at discovery time, which guarantees that the first
//     discovered parent certifies one shortest hop path.
//
// MaxDepth semantics are inclusive:
//
//   - MaxDepthUnlimited (-1) means unlimited traversal.
//   - 0 means root-only traversal.
//   - d > 0 means:
//     vertices at depth d are visited and appear in Order,
//     but their neighbors are not expanded.
//
// This is a public contract. MaxDepth is not exclusive.
//
// -----------------------------------------------------------------------------
// -- PARTIAL RESULT CONTRACT --------------------------------------------------
//
// Once traversal begins, runtime interruption returns:
//
//   - partial *Result
//   - plus a non-nil error
//
// This applies to:
//
//   - context cancellation or deadline,
//   - neighbor enumeration failure,
//   - OnVisit failure.
//
// Result interpretation under early stop:
//
//   - Order contains only vertices that were actually dequeued and processed.
//   - Visited may contain more vertices than Order, because discovery happens at
//     enqueue time and some queued vertices may not yet have been visited.
//
// This asymmetry is intentional and must be preserved.
//
// -----------------------------------------------------------------------------
// -- OPTIONS GOVERNANCE -------------------------------------------------------
//
// Options are explicit policy inputs, not hidden mutable state.
//
//   - Options are applied sequentially.
//   - Last-writer-wins for a field.
//   - Invalid options fail before BFS allocates its working sets.
//   - Nil options are invalid.
//   - Nil callbacks are invalid.
//   - Nil context is invalid.
//
// Key option policies:
//
//   - WithContext(ctx)
//     Sets the traversal context.
//
//   - WithMaxDepth(d)
//     Sets the inclusive depth bound.
//
//   - WithFullTraversal()
//     Extends BFS into a deterministic forest traversal over all components.
//
//   - WithFilterNeighbor(fn)
//     Applies a relation-level filter on (currID, nbrID).
//
//   - WithOnEnqueue / WithOnDequeue / WithOnVisit
//     Register deterministic observer hooks.
//
// FilterNeighbor is relation-level, not edge-level.
// It filters the neighbor relation surfaced by NeighborIDs(currID), not Edge IDs.
//
// -----------------------------------------------------------------------------
// -- ERROR GOVERNANCE ---------------------------------------------------------
//
// Exported package-level sentinels are the single source of truth for protocol
// matching. Callers must compare via errors.Is, never by string.
//
// Primary sentinels:
//
//   - ErrGraphNil
//   - ErrStartVertexNotFound
//   - ErrWeightedGraph
//   - ErrOptionViolation
//   - ErrNeighborFetch
//   - ErrNeighbors         (backward-compatible alias of ErrNeighborFetch)
//   - ErrNoPath
//
// Runtime policy:
//
//   - Option failures are classified under ErrOptionViolation.
//   - Path reconstruction failures are classified under ErrNoPath.
//   - Neighbor enumeration failures must preserve BOTH:
//     1) ErrNeighborFetch classification
//     2) the underlying cause
//
// In other words, neighbor failures are double-wrapped so callers can match the
// bfs-level protocol and still inspect the original lower-level reason.
//
// -----------------------------------------------------------------------------
// -- QUEUE MEMORY LAW ---------------------------------------------------------
//
// BFS queue design is constrained by an explicit memory law:
//
//	NEVER implement BFS dequeue as:
//
//	    queue = queue[1:]
//
// That pattern can retain the backing array and keep old references alive.
// bfs uses a head index and clears consumed slots instead.
//
// This is not an incidental optimization. It is part of the package engineering
// contract and must be preserved in future refactors.
//
// -----------------------------------------------------------------------------
// -- CONCURRENCY GOVERNANCE ---------------------------------------------------
//
// bfs treats the input graph as read-only during a call.
//
// Concurrency policy:
//
//   - Concurrent reads through core are expected usage.
//   - Concurrent topology mutation during BFS is unsupported.
//   - bfs is not a snapshot-isolated traversal engine.
//   - If callers mutate topology during traversal, behavior is best-effort:
//     reproducibility may be lost, observed topology may drift, and traversal
//     may fail with runtime errors surfaced through the normal error protocol.
//
// In other words: if you need reproducible traversal, do not mutate the graph
// concurrently while BFS or Components is running.
//
// -----------------------------------------------------------------------------
// -- PATH AND FOREST GOVERNANCE -----------------------------------------------
//
// PathTo reconstructs one shortest path from StartID to a destination.
//
// Important rule:
//
//   - Under WithFullTraversal, visited does NOT mean “reachable from StartID”.
//     FullTraversal may visit vertices in other components as secondary roots.
//     PathTo therefore remains anchored to StartID and must return ErrNoPath for
//     vertices that were visited by the forest but are not reachable from StartID.
//
// This rule prevents false path reconstruction in multi-component traversals.
//
// -----------------------------------------------------------------------------
// -- COMPONENTS GOVERNANCE ----------------------------------------------------
//
// Components computes weakly-connected components under an undirected relation.
//
// Meaning:
//
//   - Direction does not split component membership.
//   - A -> B -> C is one weak component, not three.
//   - Components is not SCC.
//
// Determinism contract:
//
//   - vertices inside each component are lex-sorted,
//   - component list order is stable and deterministic.
//
// Cancellation contract:
//
//   - on ctx cancellation, Components returns partial ComponentsResult plus ctx.Err().
//
// -----------------------------------------------------------------------------
// -- AI-HINT (LLM/Copilot/ChatGPT/Claude/Gemini/Qwen guidance) ----------------
//
//   - Queue Design:
//     Never use queue = queue[1:] in BFS. Use head-index dequeue and clear slots.
//
//   - Determinism:
//     Traversal order is inherited from NeighborIDs; bfs does not sort neighbors.
//
//   - Error Matching:
//     Match exported sentinels with errors.Is. Preserve underlying causes with %w.
//
//   - Depth:
//     MaxDepth is inclusive. Vertices at depth d are visited; expansion continues
//     only while currentDepth < MaxDepth.
//
//   - Partial Result:
//     On cancel or runtime error after traversal starts, expect partial result
//     plus error, not nil result plus error.
//
//   - FullTraversal:
//     PathTo is valid only for vertices reachable from StartID, even if the
//     forest visited more vertices.
//
//   - Concurrency:
//     Treat the graph as read-only during traversal. Concurrent topology mutation
//     is unsupported if you need correctness and reproducibility.
//
//   - Examples:
//     Do not use time-based cancellation in Example functions. Use
//     context.WithCancel plus a deterministic hook trigger.
//
//   - Tests:
//     Do not compare errors by string. Do not use reflect.DeepEqual or testify
//     for traversal contracts when explicit helpers are available.
//
// -----------------------------------------------------------------------------
// -- SEE ALSO -----------------------------------------------------------------
//
//   - docs/BFS.md  for repository-level tutorial, diagrams, formulas, and recipes.
//   - package GoDoc on BFS, Components, Result, PathTo, and Option helpers for
//     per-symbol contract details.
package bfs
