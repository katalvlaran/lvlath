// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package mst computes minimum spanning trees and minimum spanning forests
// over weighted, undirected core.Graph instances.
//
// Domain scope:
//
//   - Strict MST: connect every vertex with exactly |V|-1 selected edges.
//   - Explicit MSF: return one minimum spanning tree per connected component.
//   - Algorithms: Kruskal and Prim.
//   - Input model: *core.Graph with weighted, undirected graph policy.
//   - Result model: MSTResult as the canonical public artifact.
//
// What MST solves:
//
//   - Given an undirected weighted graph G=(V,E), a minimum spanning tree is a
//     subset T of edges that connects all vertices with minimum total finite cost:
//
//     minimize:   sum w(e), for every e in T
//     subject to: T spans V, T is acyclic, and |T| = |V| - 1
//
//     Equivalent compact form:
//
//     min_{T ⊆ E} Σ_{e ∈ T} w(e),   |T| = |V| - 1
//
//     CodeCogs view:
//     https://latex.codecogs.com/svg.image?\min_{T\subseteq%20E}\sum_{e\in%20T}w(e),\quad|T|=|V|-1
//
//   - For a disconnected graph, explicit forest mode applies the same objective
//     independently to each connected component:
//
//     MSF(G) = union of MST(C_i) for every connected component C_i.
//
// Why MST matters:
//
//   - Network design:
//     MST gives the cheapest backbone that still connects every required site.
//     Examples include fiber rings, emergency radio relays, satellite mesh control
//     links, industrial sensor networks, and campus / data-center cabling.
//
//   - VLSI and physical design:
//     MST-style topologies are useful as a deterministic baseline for global
//     routing, clock distribution sketches, pin clustering, and low-cost net
//     connection before more specialized Steiner-tree or timing-aware refinements.
//
//   - Smart grids and green energy:
//     MST helps reason about minimum-cost connectivity among substations,
//     batteries, renewable sources, and critical loads while keeping disconnected
//     islanded microgrids explicit through forest mode.
//
//   - Big data and machine learning:
//     MST is a standard primitive for clustering, outlier analysis, graph-based
//     manifold approximations, and “cut the largest tree edges” segmentation.
//
//   - Algorithmic subroutines:
//     MST appears inside approximation algorithms, clustering pipelines,
//     graph partitioning, network sparsification, and topology preconditioners.
//
// Non-goals:
//
//   - Directed optimum branching/arborescence is not MST and is intentionally rejected.
//   - Shortest path computation is not MST; use a shortest-path package for route queries.
//   - Steiner tree optimization is not implemented here; MST can be a baseline but
//     does not introduce new Steiner points.
//   - Dynamic MST maintenance is out of scope; every call snapshots the graph.
//   - The package does not mutate the input graph.
//   - Matrix-backed APIs must not be documented here unless their exact public
//     signatures and sentinel laws exist in the package.
//
// Public API:
//
//   - MinimumSpanningTree(graph *core.Graph, opts ...Option) (*MSTResult, error)
//     Canonical facade. It assembles options, validates graph policy, snapshots graph
//     data, dispatches to one algorithm, and returns a detached result artifact.
//
//   - Kruskal(graph *core.Graph) (*MSTResult, error)
//     Focused strict-tree wrapper selecting AlgorithmKruskal.
//
//   - Prim(graph *core.Graph, root string) (*MSTResult, error)
//     Focused strict-tree wrapper selecting AlgorithmPrim with an explicit root.
//
//   - MSTResult
//     Canonical result artifact containing Algorithm, Mode, Root, Edges,
//     TotalWeight, VertexCount, ComponentCount, and ComponentRoots.
//
// Result interpretation:
//
//   - Algorithm reports the kernel selected by the facade.
//   - Mode reports strict tree or explicit forest semantics.
//   - Root is meaningful for Prim; in forest mode it can describe the first
//     deterministic component root when no explicit root is supplied.
//   - Edges contains detached core.Edge values selected by the MST/MSF.
//   - TotalWeight is the sum of selected edge weights.
//   - VertexCount is the number of vertices in the validated snapshot.
//   - ComponentCount is 1 for a successful strict MST and can be greater than 1
//     for explicit forest mode.
//   - ComponentRoots gives deterministic public component roots.
//
// Options and regimes:
//
//   - DefaultOptions returns AlgorithmKruskal + ModeStrictTree.
//   - WithAlgorithm selects AlgorithmKruskal or AlgorithmPrim.
//   - WithRoot supplies the explicit Prim root.
//   - WithForest enables explicit minimum spanning forest mode.
//   - WithStrictTree restores strict spanning tree mode after WithForest.
//   - Option application is deterministic and follows caller-provided order.
//
// Graph policy:
//
//   - graph must be non-nil.
//   - graph must be weighted.
//   - graph must be undirected at graph level.
//   - graph must not contain directed edge-level overrides.
//   - Self-loops are ignored because they cannot connect two components.
//   - Parallel edges are allowed; the lighter useful candidate may be selected.
//   - Negative finite weights are allowed and mathematically valid for MST.
//   - NaN and ±Inf weights are rejected before sorting or heap operations.
//
// Connectivity policy:
//
//   - ModeStrictTree is the default and returns ErrDisconnected when the graph
//     cannot be spanned by one tree.
//   - ModeForest is explicit opt-in via WithForest and returns a minimum spanning
//     forest instead of silently downgrading strict tree semantics.
//   - Empty graphs are classified as disconnected and empty through joined sentinels
//     where the validator exposes both conditions.
//
// Determinism:
//
//   - Vertex order is inherited from core.Vertices().
//   - Edge order is inherited from core.Edges().
//   - Kruskal stable-sorts by finite Weight; equal-weight candidates keep core edge order.
//   - Prim's heap orders by finite Weight, then unique Edge.ID.
//   - ComponentRoots are deterministic public roots, normalized by lexicographic vertex ID.
//   - Examples print invariant outputs unless edge order itself is the demonstrated contract.
//
// Error law:
//
//   - Use errors.Is for classification.
//   - Invalid graph failures preserve ErrInvalidGraph plus precise sentinels where applicable.
//   - ErrNilGraph reports nil graph input.
//   - ErrUnweightedGraph reports missing weighted semantics.
//   - ErrDirectedGraph reports directed graph policy.
//   - ErrDirectedEdge reports directed edge-level overrides.
//   - ErrEmptyGraph reports an empty graph.
//   - ErrDisconnected reports strict tree mode failure on empty or disconnected graphs.
//   - ErrNaNInfWeight reports non-finite edge weights.
//   - ErrEmptyRoot reports missing Prim root in strict mode.
//   - core.ErrVertexNotFound reports an absent non-empty Prim root.
//   - ErrNilOption reports nil Option values.
//   - ErrUnsupportedAlgorithm reports unsupported algorithm policy.
//   - ErrInvalidOption reports invalid option state.
//   - ErrNilResult reports invalid nil result helper access.
//
// Complexity:
//
//   - Kruskal: O(E log E + E·α(V)) time, O(E + V) space.
//     Sorting dominates; DSU operations are effectively near-constant amortized.
//
//   - Prim: O(E log E) time, O(E + V) space for the current edge-frontier heap.
//     This package does not currently use a decrease-key vertex heap; therefore
//     O(E log V) must not be claimed for the current implementation.
//
//   - Result clone helpers are O(E + C), where C is the component count.
//
// Result ownership:
//
//   - MSTResult contains detached core.Edge values.
//   - No live *core.Edge pointers are retained in public results.
//   - MSTResult.Clone returns detached slice fields.
//   - MSTResult.EdgeValues returns a caller-owned edge slice.
//
// Concurrency:
//
//   - Algorithms snapshot vertices and edges before the kernel starts.
//   - Concurrent mutation of the same graph during snapshot construction is unsupported;
//     callers must synchronize graph writes.
//
// AI-Hints:
//
//   - Do not reintroduce tuple-return public APIs; MSTResult is the canonical result.
//   - Do not use epsilon inside sort or heap comparators; reject NaN/Inf and compare finite weights directly.
//   - Do not use edge.To as the next endpoint for undirected traversal; resolve endpoints relative to the source vertex.
//   - Do not silently return a forest from strict tree mode; require WithForest.
//   - Do not document matrix-backed APIs unless the exact public signatures exist.
//   - Do not claim Prim is O(E log V) unless the implementation changes to a vertex-key decrease-key policy.
package mst
