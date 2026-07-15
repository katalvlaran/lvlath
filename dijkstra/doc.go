// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package dijkstra implements deterministic single-source shortest paths over
// weighted core.Graph instances using Dijkstra's algorithm.
//
// -----------------------------------------------------------------------------
// -- WHAT ---------------------------------------------------------------------
//
// dijkstra provides one canonical public shortest-path facade plus convenience
// wrappers over the same result contract:
//
//   - Dijkstra(g, sourceID, opts...)
//     Runs deterministic single-source shortest paths and returns a detached
//     Result.
//
//   - Distances(g, sourceID, opts...)
//     Convenience wrapper that returns a detached distance map only.
//
//   - DistanceTo(g, sourceID, targetID, opts...)
//     Convenience wrapper that runs Dijkstra and resolves one target distance.
//
//   - ShortestPathTo(g, sourceID, targetID, opts...)
//     Convenience wrapper that runs Dijkstra with path tracking enabled and
//     returns one deterministic shortest-path witness plus its distance.
//
// Result is the public result artifact. It exposes:
//
//   - SourceID  - the source vertex identifier used for the run.
//   - Distances - finalized shortest-path distances for the known result domain.
//   - Prev      - optional predecessor map used for path reconstruction.
//
// The package is designed as a reusable weighted shortest-path kernel for
// downstream algorithms that require deterministic behavior, explicit numeric
// policy, errors.Is-compatible failure matching, and detached caller-owned
// result data.
//
// -----------------------------------------------------------------------------
// -- WHY ----------------------------------------------------------------------
//
//   - Determinism First:
//     dijkstra preserves the graph relation order surfaced by core and adds an
//     explicit heap tie-break on vertex ID for equal candidate distances.
//
//   - Correct Weighted Semantics:
//     Distances are true weighted shortest-path costs over non-negative edges.
//     The package intentionally rejects invalid numeric input and negative costs.
//
//   - Explicit Result Surface:
//     Distances, reachability, and one shortest-path witness are exposed through
//     Result instead of ad-hoc parallel maps returned from the kernel.
//
//   - Policy Without Topology Mutation:
//     Runtime options such as MaxDistance and InfEdgeThreshold alter traversal
//     policy without rewriting or cloning the input graph.
//
//   - Detached Ownership:
//     Published results are detached from the graph and belong to the caller
//     after return.
//
// -----------------------------------------------------------------------------
// -- DOMAIN SCOPE -------------------------------------------------------------
//
// dijkstra is intentionally scoped to:
//
//   - deterministic single-source shortest paths,
//   - weighted graphs with non-negative finite edge costs,
//   - deterministic path reconstruction of one shortest-path witness when
//     path tracking is enabled,
//   - optional traversal policy through distance cutoff and wall threshold.
//
// The package answers questions such as:
//
//   - “What is the minimum total cost from this source to every known vertex?”
//   - “What is the cost to this specific target?”
//   - “Is this known target reachable under the effective traversal policy?”
//   - “What is one deterministic shortest-path witness to this target?”
//
// -----------------------------------------------------------------------------
// -- NON-GOALS ----------------------------------------------------------------
//
// dijkstra is intentionally NOT:
//
//   - a negative-weight shortest-path package,
//   - an all-pairs shortest-path package,
//   - a snapshot-isolated traversal engine,
//   - a concurrent-mutation-safe graph execution model,
//   - an exhaustive all-shortest-path reconstruction package.
//
// Consequences:
//
//   - If any traversed edge may have a negative finite weight, Dijkstra is
//     mathematically the wrong tool and the package rejects that input.
//   - If you need all-pairs shortest paths, this package does not provide
//     Floyd-Warshall-, Johnson-, or repeated-source orchestration.
//   - If you need an immutable snapshot of a graph under concurrent mutation,
//     this package does not materialize one for you.
//   - If you need every shortest witness rather than one deterministic witness,
//     Prev and PathTo are insufficient by design.
//
// -----------------------------------------------------------------------------
// -- NUMERIC POLICY -----------------------------------------------------------
//
// Numeric semantics are separated by domain and form part of the package contract.
//
//  1. Graph-edge input domain
//
//     Every consumed Edge.Weight must be a finite float64.
//
//     Accepted:
//
//     - finite w >= 0
//
//     Rejected:
//
//     - math.NaN()
//     - math.Inf(1)
//     - math.Inf(-1)
//     - every finite w < 0
//
//  2. Runtime-policy domain
//
//     Positive infinity is valid for:
//
//     - MaxDistance, where +Inf means “no accumulated-distance cutoff”;
//     - InfEdgeThreshold, where +Inf means “no finite edge is blocked”.
//
//     These option-domain meanings do not authorize +Inf graph-edge weights.
//
//  3. Published result domain
//
//     Result.Distances contains:
//
//     - finite non-negative shortest-path distances for reachable vertices;
//     - math.Inf(1) for known vertices that are unreachable under the effective
//     graph and traversal policy.
//
//     A missing target is not represented numerically. It is classified through
//     ErrTargetNotFound.
//
// Edge-weight classification law:
//
//   - math.IsNaN(w)     -> ErrInvalidWeight
//   - math.IsInf(w, 0)  -> ErrInvalidWeight
//   - finite w < 0      -> ErrNegativeWeight
//   - finite w >= 0     -> accepted
//
// Arithmetic-overflow law:
//
//   - If two individually valid finite operands produce +Inf, traversal returns
//     ErrDistanceOverflow.
//   - Arithmetic overflow is not published as ordinary unreachable state.
//   - On overflow, the package returns nil result plus error.
//
// Published-distance law:
//
//   - A known but unreachable vertex remains present in Distances with +Inf.
//   - A known reachable vertex has a finite non-negative distance.
//   - An unknown target is classified through ErrTargetNotFound.
//   - +Inf in Distances is package-owned result state; it is not copied from a
//     graph-edge weight.
//
// -----------------------------------------------------------------------------
// -- GRAPH POLICY -------------------------------------------------------------
//
// dijkstra runs over *core.Graph and requires weighted graph semantics.
//
// Graph policy:
//
//   - g must be non-nil.
//   - g must be weighted.
//   - directed graphs are supported.
//   - undirected graphs are supported.
//   - mixed-edge graphs are supported.
//
// Traversable relations are sourced from:
//
//	g.Neighbors(u)
//
// Endpoint interpretation law:
//
//   - dijkstra does not assume that the traversable neighbor is always edge.To.
//   - For undirected and mixed relations, the effective opposite endpoint is
//     resolved relative to the current vertex.
//   - Directed reverse traversal is forbidden unless the current relation from
//     core.Neighbors(u) is actually traversable from u.
//
// -----------------------------------------------------------------------------
// -- DETERMINISM LAW ----------------------------------------------------------
//
// Determinism is a package-level contract.
//
//  1. Vertex-domain initialization
//     The kernel initializes the known result domain by iterating:
//
//     g.Vertices()
//
//     as surfaced by core.
//
//  2. Relation enumeration
//     Relaxation processes candidate relations in the exact order returned by:
//
//     g.Neighbors(u)
//
//     dijkstra does not inject a second hidden sorting layer for neighbors.
//
//  3. Heap ordering
//     Equal-distance frontier items are ordered by:
//
//     (candidateDistance, vertexID)
//
//     so equal candidate costs are broken by lexicographically smaller vertex IDs.
//
//  4. Predecessor updates
//     Relaxation uses strict improvement only.
//
//     - candidate < currentDistance  -> update
//     - candidate == currentDistance -> do not overwrite predecessor state
//
// Together, these rules make published distances, predecessor selection, and
// reconstructed path witnesses stable for the same graph state, sourceID, and
// runtime policy.
//
// -----------------------------------------------------------------------------
// -- RESULT CONTRACT ----------------------------------------------------------
//
// Result is the detached public shortest-path artifact.
//
// Public result semantics:
//
//   - SourceID
//     The explicit source vertex identifier used for the run.
//
//   - Distances
//     Finalized shortest-path distances over the known result domain.
//     Known but unreachable vertices remain present with value +Inf.
//
//   - Prev
//     Optional predecessor map used for path reconstruction.
//     Prev == nil means path tracking was disabled, not “no path exists”.
//
// Result-surface laws:
//
//   - DistanceTo(targetID)
//     Returns the stored distance when the target is known.
//     Returns ErrTargetNotFound only when the target is absent from the result domain.
//
//   - HasPathTo(targetID)
//     Returns false with nil error for a known target whose distance is +Inf.
//
//   - PathTo(targetID)
//     Requires path tracking to have been enabled.
//     Returns one deterministic shortest-path witness.
//     Validates every predecessor against the result domain.
//     Rejects broken, cyclic, and out-of-domain predecessor state with ErrNoPath.
//     Never returns a partial witness and never loops indefinitely on malformed Prev.
//     Does not enumerate all shortest paths.
//
// -----------------------------------------------------------------------------
// -- OPTIONS GOVERNANCE -------------------------------------------------------
//
// Options are explicit runtime policy inputs, not hidden mutable state.
//
// Dijkstra option law:
//
//   - options are applied sequentially,
//   - last-writer-wins per field,
//   - nil options are invalid,
//   - invalid explicit option input fails before traversal begins.
//
// Public runtime policy includes:
//
//   - WithPathTracking()
//     Enables predecessor tracking for later path reconstruction.
//
//   - WithMaxDistance(max)
//     Limits exploration to shortest paths whose distance does not exceed max.
//
//   - WithInfEdgeThreshold(threshold)
//     Treats edges with weight >= threshold as impassable walls.
//
// Baseline default policy:
//
//   - TrackPaths       = false
//   - MaxDistance      = +Inf
//   - InfEdgeThreshold = +Inf
//
// Important separation:
//
//   - sourceID is not an option.
//   - sourceID is an explicit required input to the public API.
//
// -----------------------------------------------------------------------------
// -- WALL AND CUTOFF POLICY ---------------------------------------------------
//
// dijkstra exposes two traversal-policy gates that are distinct from graph
// validity and distinct from each other.
//
//  1. InfEdgeThreshold
//     Edges with weight >= InfEdgeThreshold are skipped as impassable walls.
//
//     Consequence:
//     a target may remain known but unreachable because all admissible routes
//     to it are blocked by threshold policy.
//
//  2. MaxDistance
//     Candidate paths with distance > MaxDistance are not explored.
//
//     Consequence:
//     a graph path may exist mathematically while the published result still
//     contains +Inf for that target because traversal policy disallowed it.
//
// These are traversal policies, not numeric-validation failures.
//
// -----------------------------------------------------------------------------
// -- ERROR LAW ----------------------------------------------------------------
//
// Exported package-level sentinels are the single source of truth for protocol
// matching. Callers must compare via errors.Is, never by string.
//
// Primary sentinels include:
//
//   - ErrNilGraph
//   - ErrEmptySourceID
//   - ErrSourceNotFound
//   - ErrTargetNotFound
//   - ErrUnweightedGraph
//   - ErrNilOption
//   - ErrBadMaxDistance
//   - ErrBadInfEdgeThreshold
//   - ErrNegativeWeight
//   - ErrInvalidWeight
//   - ErrDistanceOverflow
//   - ErrPathTrackingDisabled
//   - ErrNoPath
//   - ErrEmptyTargetID
//   - ErrNilResult
//
// Wrapping law:
//
//   - When runtime numeric validation adds edge context, the sentinel must be
//     preserved with %w so callers can still match via errors.Is.
//   - Graph-surface runtime failures are wrapped with context and preserved as
//     ordinary Go wrapped errors.
//
// Panic policy:
//
//   - Public option validation is error-returning.
//   - Panic-based option governance is forbidden.
//
// -----------------------------------------------------------------------------
// -- OWNERSHIP LAW ------------------------------------------------------------
//
// Published results are detached and caller-owned.
//
// Ownership contract:
//
//   - The package does not retain a live mutable link from Result back
//     to the graph.
//   - After return, callers may read, clone, and transform published results.
//   - Convenience wrappers that return maps publish detached caller-owned copies.
//
// Caller-mutation consequence:
//
//   - Distances and Prev are caller-owned and therefore mutable after return.
//   - Mutating them changes the meaning of that Result instance.
//   - PathTo validates witness-chain safety and fails with ErrNoPath when mutation
//     leaves the predecessor relation broken, cyclic, or outside Distances.
//   - Clone creates independent maps but does not validate semantic correctness.
//
// Clone law:
//
//   - Result.Clone returns a deep copy.
//   - Prev == nil is preserved exactly and is not rewritten into an empty map.
//
// -----------------------------------------------------------------------------
// -- PARTIAL RESULT LAW -------------------------------------------------------
//
// On failure, the package does not publish a partial Result.
//
// Publication law:
//
//   - validation failures return nil result plus error,
//   - runtime failures during kernel execution return nil result plus error,
//   - detached results are published only after the run completes successfully.
//
// This prevents callers from accidentally depending on incomplete shortest-path
// state as if it were finalized output.
//
// -----------------------------------------------------------------------------
// -- CONCURRENCY LAW ----------------------------------------------------------
//
// dijkstra treats the graph as read-only during a call.
//
// Concurrency policy:
//
//   - concurrent reads through core are expected usage,
//   - concurrent topology mutation during Dijkstra execution is unsupported,
//   - the package does not materialize a snapshot-isolated graph image before traversal.
//
// In other words:
//
//   - if you need reproducible shortest-path results, do not mutate the graph
//     concurrently while dijkstra is running.
//
// -----------------------------------------------------------------------------
// -- COMPLEXITY SUMMARY -------------------------------------------------------
//
// Effective complexity depends on the graph surfaces provided by core.
//
// Public summary:
//
//   - Dijkstra / Distances / DistanceTo / ShortestPathTo
//     Effective time O(V log V + E log V + graph-surface-enumeration cost).
//     Effective space O(V + E_heap), excluding graph-owned storage.
//
// Result-surface summary:
//
//   - DistanceTo / HasPathTo
//     Time O(1), Space O(1).
//
//   - PathTo
//     Time O(k), Space O(k), where k is the number of vertices on the returned path.
//
//   - Clone
//     Time O(V), Space O(V) over copied result entries.
//
// -----------------------------------------------------------------------------
// -- AI-HINT (LLM/Copilot/ChatGPT/Claude/Gemini/Qwen guidance) ----------------
//
//   - Error Matching:
//     Never parse error strings. Use errors.Is with exported sentinels only.
//
//   - Numeric Domain Separation:
//     Graph-edge weights must remain finite.
//     +Inf is valid only in the result domain for unreachability and in the
//     option domain for unbounded policy.
//     Do not transfer +Inf semantics implicitly between these domains.
//
//   - Arithmetic Overflow:
//     Do not publish an overflowed candidate as unreachable +Inf.
//     Preserve ErrDistanceOverflow and suppress partial result publication.
//
//   - Endpoint Law:
//     Never simplify mixed or undirected traversal to edge.To-only logic.
//     The effective opposite endpoint must be resolved relative to the current vertex.
//
//   - Determinism:
//     Do not remove the heap tie-break on vertex ID.
//     Equal-distance ordering affects predecessor stability and user-visible paths.
//
//   - Path Tracking:
//     Prev == nil means path tracking was disabled.
//     It does not mean that the graph has no reachable paths.
//
//   - Path Witness Integrity:
//     Result.Prev is caller-owned after publication.
//     Never follow predecessor links without cycle and domain guards.
//     Broken, cyclic, or out-of-domain witness state must fail with ErrNoPath.
//
//   - Numeric Domain Separation:
//     Graph-edge weights must remain finite.
//     +Inf is valid only in result and policy domains.
//
//   - Custom Options:
//     Option is publicly constructible.
//     Canonical assembly must revalidate finalized state after every option.
//
//   - Options:
//     Do not move sourceID back into options.
//     Do not reintroduce panic-based option validation.
//
//   - Result Publication:
//     Do not publish partial shortest-path state on runtime failure.
//
//   - Concurrency:
//     Treat the graph as read-only during traversal if correctness and
//     reproducibility matter.
//
// -----------------------------------------------------------------------------
// -- SEE ALSO -----------------------------------------------------------------
//
//   - docs/DIJKSTRA.md for repository-level tutorial, formulas, diagrams,
//     algorithm notes, and operational examples.
//   - package GoDoc on Dijkstra, Result, Options, Option helpers,
//     and wrapper APIs for per-symbol contract details.
package dijkstra
