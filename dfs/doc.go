// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package dfs implements deterministic depth-first traversal, deterministic cycle
// witness detection, and deterministic directed topological ordering over core.Graph.
//
// -----------------------------------------------------------------------------
// -- WHAT ---------------------------------------------------------------------
//
// dfs provides four public algorithmic facades and two convenience wrappers:
//
//   - DFS(g, startID, opts...)
//     Deterministic single-source depth-first traversal.
//
//   - Forest(g, opts...)
//     Deterministic full-graph DFS forest traversal across disconnected topology.
//
//   - DetectCycles(g)
//     Deterministic cyclicity detection with canonical witness-cycle reporting.
//
//   - HasCycle(g)
//     Deterministic summary helper over DetectCycles.
//
//   - TopologicalSort(g, options...)
//     Deterministic DFS-based topological ordering for directed graphs.
//
//   - TopologicalSortContext(ctx, g)
//     Convenience wrapper for TopologicalSort with explicit cancellation context.
//
// Result is the public traversal artifact. It exposes:
//
//   - Order            - DFS finish order (post-order), not discovery order.
//   - Depth            - DFS-tree depth.
//   - Parent           - DFS-tree predecessor relation.
//   - Visited          - set of vertices actually entered by traversal.
//   - SkippedNeighbors - count of candidate neighbors rejected by FilterNeighbor.
//
// CycleDetectionResult is the public cycle-detection artifact. It exposes:
//
//   - HasCycle - summary boolean indicating whether at least one witness cycle exists.
//   - Cycles   - deterministic canonical witness-cycle set.
//
// The package is designed as a reusable algorithmic kernel for downstream graph
// analytics that require strict determinism, errors.Is-compatible failure handling,
// explicit witness contracts, and disciplined traversal semantics.
//
// -----------------------------------------------------------------------------
// -- WHY ----------------------------------------------------------------------
//
//   - Determinism First:
//     dfs preserves the graph's deterministic root and neighbor order instead of
//     inventing hidden sorting rules or accidental traversal tie-breaks.
//
//   - Post-Order Is a Public Contract:
//     Result.Order is authoritative finish order, which makes downstream uses
//     such as dependency unwinding, stack-safe teardown planning, and topo-style
//     reasoning reproducible and auditable.
//
//   - Witness Cycles Instead of Exhaustive Enumeration:
//     DetectCycles proves cyclicity and returns deterministic canonical witnesses
//     without promising NP-hard exhaustive listing of all simple cycles.
//
//   - Strict Failure Governance:
//     Invalid options fail before traversal allocation, runtime failures preserve
//     sentinel-based classification, and DFS partial results remain usable where
//     mathematically safe.
//
//   - Practical Algorithm Surface:
//     Hooks, depth limits, filtering, forest traversal, cycle discovery, and topo
//     ordering are exposed as composable, deterministic package primitives rather
//     than ad-hoc snippets duplicated across downstream code.
//
// -----------------------------------------------------------------------------
// -- DOMAIN SCOPE -------------------------------------------------------------
//
// dfs is intentionally scoped to:
//
//   - deterministic single-source DFS,
//   - deterministic full-graph DFS forest traversal,
//   - deterministic cycle witness detection for directed and undirected traversal
//     semantics where applicable,
//   - deterministic DFS-based topological ordering of directed dependency graphs.
//
// The package answers questions such as:
//
//   - “What is the deterministic DFS finish order from this start vertex?”
//   - “What DFS-tree parent and depth structure was constructed?”
//   - “Does this graph contain at least one cycle witness?”
//   - “Which canonical witness loops were discovered by the DFS-based method?”
//   - “What deterministic topological execution order satisfies this DAG?”
//
// -----------------------------------------------------------------------------
// -- NON-GOALS ----------------------------------------------------------------
//
// dfs is intentionally NOT:
//
//   - a graph storage or builder package,
//   - a shortest-path package,
//   - an exhaustive all-simple-cycles enumerator,
//   - a snapshot-isolated concurrent traversal engine.
//
// Consequences:
//
//   - DFS depth is DFS-tree depth, not shortest-hop or weighted distance.
//   - DetectCycles does not guarantee exhaustive enumeration of all mathematically
//     possible simple cycles.
//   - The package does not own graph topology and does not grow builder-style APIs.
//   - If the graph is mutated concurrently during traversal, correctness and
//     reproducibility are not guaranteed.
//
// -----------------------------------------------------------------------------
// -- DETERMINISM LAW ----------------------------------------------------------
//
// Determinism is a package-level contract.
//
//  1. Root ordering
//     DFS forest root selection follows:
//
//     g.Vertices()
//
//     as surfaced by core.Graph.
//
//  2. Neighbor ordering
//     DFS, DetectCycles, and TopologicalSort process candidate relations in the
//     exact order returned by:
//
//     g.Neighbors(id)
//
//     dfs itself does not inject hidden fallback sorting.
//
//  3. Finish-order semantics
//     Result.Order records finish order only.
//     It is appended when a vertex exits, not when it is discovered.
//
//  4. Cycle witness ordering
//     Witness cycles are canonicalized deterministically and emitted in stable order.
//
//  5. Topological order
//     TopologicalSort is deterministic under deterministic graph root order,
//     neighbor order, and stable option policy.
//
// -----------------------------------------------------------------------------
// -- DFS SEMANTICS ------------------------------------------------------------
//
// DFS is a depth-first traversal that builds a DFS tree, or a DFS forest when
// full traversal is enabled.
//
// Public Result semantics:
//
//   - Order
//     Finish order (post-order).
//
//   - Depth
//     DFS-tree depth measured from the root of the current DFS tree.
//
//   - Parent
//     DFS-tree predecessor of each actually entered non-root vertex.
//
//   - Visited
//     Set of vertices actually entered by traversal.
//
//   - SkippedNeighbors
//     Count of candidate neighbor relations rejected by FilterNeighbor.
//
// Structural laws:
//
//   - DFS-tree roots have depth 0.
//   - DFS-tree roots never appear in Parent.
//   - Parent[v] is assigned only when v is actually entered as a child.
//   - Filtered or rejected neighbors do not acquire Parent entries.
//   - Order is authoritative only on successful completion.
//
// FullTraversal law:
//
//   - WithFullTraversal() converts single-tree DFS into a deterministic DFS forest.
//   - Each newly selected forest root starts with Depth[root] == 0.
//   - Parent remains absent for every forest root.
//
// -----------------------------------------------------------------------------
// -- PARTIAL RESULT CONTRACT --------------------------------------------------
//
// Once DFS execution begins, aborting runtime failures may return:
//
//   - partial *Result
//   - plus a non-nil error
//
// This applies to:
//
//   - context cancellation or deadline,
//   - neighbor enumeration failure,
//   - OnVisit failure,
//   - OnExit failure.
//
// Result interpretation under early stop:
//
//   - Visited, Depth, and Parent may contain partial but meaningful structural state.
//   - Order is not exposed as authoritative partial finish order.
//   - On aborting failures, Order may therefore be cleared to prevent downstream
//     consumers from treating incomplete finish order as mathematically valid output.
//
// -----------------------------------------------------------------------------
// -- CYCLE DETECTION CONTRACT -------------------------------------------------
//
// DetectCycles reports cyclicity and returns a deterministic witness set of
// canonicalized cycles discovered by the DFS-based method.
//
// It does NOT promise:
//
//   - exhaustive enumeration of all simple cycles,
//   - canonical representatives for every possible equivalent traversal witness,
//   - path-enumeration semantics.
//
// Public CycleDetectionResult semantics:
//
//   - HasCycle
//     True iff at least one witness cycle was discovered.
//
//   - Cycles
//     Deterministic canonical witness cycles represented as closed sequences:
//
//     [v0, v1, ..., vk, v0]
//
// Canonicalization law:
//
//   - Directed witness cycles preserve orientation during canonicalization.
//   - Undirected witness cycles may treat reversed orientation as equivalent.
//   - Canonicalization must not corrupt caller-visible backing storage by aliasing
//     tricks or in-place slice growth.
//
// -----------------------------------------------------------------------------
// -- TOPOLOGICAL ORDERING CONTRACT --------------------------------------------
//
// TopologicalSort is defined for graph instances that satisfy the package's
// directed-graph policy check.
//
// Validation law:
//
//   - graph-policy rejection returns ErrGraphNotDirected.
//
// Ordering law:
//
//   - only directed outgoing relations contribute dependency constraints,
//
//   - the returned order must satisfy:
//
//     for every directed dependency u -> v,
//     index(u) < index(v)
//
// Cycle law:
//
//   - if a directed cycle is discovered during topological traversal,
//     TopologicalSort returns ErrCycleDetected.
//
// Mixed-edge law:
//
//   - after graph-policy acceptance, undirected relations do not contribute
//     dependency edges to the topological order.
//
// -----------------------------------------------------------------------------
// -- OPTIONS GOVERNANCE -------------------------------------------------------
//
// Options are explicit traversal policy inputs, not hidden mutable state.
//
// DFS option law:
//
//   - options are applied sequentially,
//   - last-writer-wins per field,
//   - invalid explicit option input fails before traversal working state is allocated.
//
// Public DFS option surface includes policy controls for:
//
//   - context cancellation,
//   - pre-order observation,
//   - post-order observation,
//   - inclusive depth limiting,
//   - neighbor filtering,
//   - full-graph forest traversal.
//
// Topological sort uses a narrower option surface specialized for its needs.
// TopologicalSortContext exists as a convenience wrapper for explicit cancellation.
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
//   - ErrNeighborFetch
//   - ErrCycleDetected
//   - ErrGraphNotDirected
//   - ErrOptionViolation
//
// Runtime policy:
//
//   - explicit option failures are classified under ErrOptionViolation,
//   - neighbor enumeration failures must preserve ErrNeighborFetch classification
//     while retaining the underlying cause through wrapping,
//   - context cancellation and deadline failures are propagated through the
//     standard context sentinels,
//   - hook failures are returned with wrapping so callers can preserve both
//     category and origin.
//
// -----------------------------------------------------------------------------
// -- CONCURRENCY & SNAPSHOT POLICY --------------------------------------------
//
// dfs reads the graph progressively during execution.
//
// Concurrency policy:
//
//   - concurrent reads through core are expected usage,
//   - concurrent topology mutation during DFS, DetectCycles, or TopologicalSort
//     is unsupported,
//   - the package does not materialize a global immutable traversal snapshot
//     before execution.
//
// In other words:
//
//   - if callers need reproducible traversal, the graph must be treated as
//     read-only for the duration of the call.
//
// -----------------------------------------------------------------------------
// -- RESULT OWNERSHIP ---------------------------------------------------------
//
// Returned result slices and maps belong to the caller after the function returns.
//
// Ownership law:
//
//   - dfs does not retain caller-visible result containers for background mutation,
//   - callers may read, copy, or transform returned results after the call,
//   - callers remain responsible for any further synchronization if they share
//     those results across goroutines.
//
// -----------------------------------------------------------------------------
// -- COMPLEXITY SUMMARY -------------------------------------------------------
//
//   - DFS / Forest
//     Time O(V+E), Space O(V), excluding hook and filter cost.
//
//   - DetectCycles
//     Time O(V+E+W*L), Space O(V+Lmax+W*L),
//     where W is the number of stored witness cycles and L is average witness length.
//
//     Equivalent summation view:
//
//     O(V+E+Σ|witness_i|)
//
//   - TopologicalSort
//     Time O(V+E), Space O(V).
//
// -----------------------------------------------------------------------------
// -- AI-HINT (LLM/Copilot/ChatGPT/Claude/Gemini/Qwen guidance) ----------------
//
//   - Neighbor Semantics:
//     Never assume the traversable neighbor of an edge is always edge.To.
//     Undirected and mixed-edge traversal must resolve the neighbor from the
//     current vertex context.
//
//   - Finish Order:
//     Result.Order is post-order finish order, not discovery order.
//
//   - Witness Contract:
//     DetectCycles returns witness cycles, not an exhaustive cycle listing.
//
//   - Error Matching:
//     Do not replace sentinel-based errors with string-based checks.
//     Use errors.Is for package error protocol.
//
//   - Cycle Normalization:
//     Do not use in-place append(s, s...) patterns in normalization helpers when
//     aliasing can mutate caller-visible or shared backing storage.
//
//   - Concurrency:
//     Concurrent graph mutation during traversal is unsupported.
//
//   - Tests:
//     Order-sensitive tests must follow the package determinism law rather than
//     unordered comparisons that weaken the contract.
//
//   - Forest Semantics:
//     Depth resets per DFS-tree root under full traversal.
//     Root vertices must not appear in Parent.
//
// -----------------------------------------------------------------------------
// -- SEE ALSO -----------------------------------------------------------------
//
//   - docs/DFS.md for repository-level tutorial, formulas, diagrams, examples,
//     witness semantics, and operational guidance.
//   - package GoDoc on DFS, Forest, DetectCycles, HasCycle, TopologicalSort,
//     TopologicalSortContext, Result, and CycleDetectionResult for per-symbol
//     contract details.
package dfs
