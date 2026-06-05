// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package flow defines deterministic maximum-flow tools for capacity networks
// represented as core.Graph values.
//
// The package answers one practical question: given a source, a sink, and a
// network of finite capacities, how much traffic, power, material, or abstract
// load can be pushed from source to sink without violating any capacity bound?
//
// The canonical entry point is MaxFlow. It returns MaxFlowResult, not just a
// number, because a professional max-flow computation is also a certificate:
// the final residual network proves that no more augmenting path exists, and
// the min-cut partition explains which part of the network is the bottleneck.
//
// # Typical use cases
//
// Data-center traffic engineering:
//   - source: ingress/load-balancer tier;
//   - intermediate vertices: edge switches, core switches, racks, service pools;
//   - sink: target service or egress boundary;
//   - capacity: Gbps, requests/second, or any finite throughput unit.
//
// Smart-grid and energy dispatch:
//   - source: generation portfolio or upstream transmission boundary;
//   - intermediate vertices: generators, substations, transformers, feeders;
//   - sink: demand region;
//   - capacity: MW, MVA, or a normalized engineering capacity unit.
//
// Logistics and supply networks:
//   - source: supplier or warehouse;
//   - intermediate vertices: hubs, ports, depots;
//   - sink: demand center;
//   - capacity: tons/hour, containers/day, or shipment slots.
//
// # Mathematical contract
//
// A capacity network is adapted into a residual network. For each directed arc
// u -> v with capacity c, the residual network stores a forward residual arc
// u -> v and a reverse residual arc v -> u. Augmenting flow by delta along a path
// subtracts delta from each forward residual capacity and adds delta to each
// reverse residual capacity.
//
// A run is complete when no source-to-sink path with residual capacity greater
// than epsilon exists. At that point the max-flow/min-cut theorem applies:
//
//	max-flow value == capacity of the final source-side / sink-side cut
//
// MaxFlowResult exposes this proof:
//   - Value is the computed maximum flow value.
//   - Residual is a detached directed weighted residual graph.
//   - CutSourceSide and CutSinkSide are the final residual reachability partition.
//   - Partial reports interruption before the optimality certificate is complete.
//
// # Graph policy
//
// Input graphs must be weighted because core.Edge.Weight stores capacities.
// Capacity values are float64 and must be finite.
//
// Directed edges become one capacity arc.
//
// Undirected edges are adapted into two directed capacity arcs with the same
// capacity. This models a bidirectional capacity relationship for max-flow
// analysis, not a single shared physical pipe with coupled opposite directions.
// If a domain needs coupled bidirectional capacity, model it explicitly before
// calling this package.
//
// Loops are ignored because a self-loop cannot increase s-t flow.
//
// Parallel edges are aggregated deterministically. This is useful for networks
// where multiple cables, lanes, feeders, or contracts connect the same endpoints.
//
// The input graph is never mutated.
//
// # Numeric policy
//
// Epsilon is the residual threshold:
//   - capacities <= epsilon are treated as absent;
//   - capacities < -epsilon are rejected as ErrNegativeCapacity;
//   - residual values with absolute magnitude <= epsilon are clamped to zero.
//
// NaN and +/-Inf capacities are rejected with ErrNaNInf. Infinite capacity is
// not accepted because it would make residual arithmetic and result certificates
// ambiguous. Use a large explicit engineering capacity if a domain wants a
// practically unconstrained edge.
//
// # Determinism
//
// The package intentionally makes algorithmic tie-breaking stable:
//   - vertex order comes from core.Vertices();
//   - edge ingestion order comes from core.Edges();
//   - residual adjacency lists are sorted once;
//   - BFS and DFS kernels scan residual adjacency in that sorted order;
//   - residual graph edge IDs are deterministic for each residual arc.
//
// This determinism is not cosmetic. It makes examples reproducible, tests stable,
// residual certificates debuggable, and downstream matrix snapshots comparable.
//
// # Algorithms
//
// Dinic is the default algorithm for MaxFlow. It builds BFS level graphs and
// pushes blocking flows through admissible level-increasing arcs. It is the best
// default for larger networks among the currently implemented kernels.
//
// Edmonds-Karp uses BFS to find the shortest augmenting path by number of arcs.
// It is slower, but excellent for auditability, teaching, and path-oriented
// debugging because each augmentation has a simple witness.
//
// Ford-Fulkerson uses deterministic DFS augmenting paths. It is retained for
// compatibility and small/integral-like networks. On arbitrary real capacities,
// DFS path choice can be a poor practical strategy, so WithMaxAugmentations can
// be used as a safety valve.
//
// # Complexity
//
// Let V be the number of vertices and A be the residual adjacency-entry count.
//
// Residual construction costs O(V + E + A log A) time and O(V + A) space.
//
// Dinic has O(V^2 * A) worst-case behavior on general networks, with stronger
// bounds for special capacity classes.
//
// Edmonds-Karp has O(V * A^2) worst-case behavior.
//
// Ford-Fulkerson has O(A * F) behavior in integral-like regimes, where F is the
// number of successful augmenting pushes under the chosen capacities.
//
// CapacityMatrix allocates a dense V x V matrix and therefore costs O(V^2) space.
// It is intended for diagnostics and downstream algebra, not for inner residual
// update loops.
//
// # Error law
//
// Package-defined failures are sentinel-classified. Use errors.Is.
//
// Common sentinel errors:
//   - ErrNilGraph: nil input graph;
//   - ErrEmptyTerminal: empty source or sink ID;
//   - ErrSameTerminal: source and sink are the same vertex;
//   - ErrSourceNotFound / ErrSinkNotFound: missing terminals;
//   - ErrUnweightedGraph: capacities cannot be represented;
//   - ErrInvalidEpsilon: invalid numeric threshold;
//   - ErrInvalidCapacity / ErrNegativeCapacity / ErrNaNInf: bad edge capacity;
//   - ErrAugmentationLimit: configured augmentation limit interrupted a run;
//   - ErrObserverFailure: observer rejected an augmentation event.
//
// Lower-level core errors are preserved with errors.Join where applicable.
//
// # Result interpretation
//
// A successful MaxFlowResult is more than a scalar:
//
//	result.Value
//	    The max-flow value.
//
//	result.Residual
//	    Directed weighted residual graph. If the run is complete, there is no
//	    positive residual path from Source to Sink.
//
//	result.CutSourceSide / result.CutSinkSide
//	    Min-cut certificate. Summing original capacities crossing from source side
//	    to sink side yields result.Value.
//
//	result.Partial
//	    True when cancellation, observer failure, or augmentation limit stopped
//	    the run before optimality was proven.
//
// # Matrix integration
//
// CapacityMatrix produces a deterministic matrix.Dense capacity snapshot using
// the same adapter law as MaxFlow. This is useful for diagnostics, visualization,
// regression tests, and downstream matrix workflows.
//
// Matrix operations are deliberately not used inside the hot residual update
// loops. The residual engine uses compact maps and sorted adjacency slices for
// algorithmic work; matrix artifacts are for pre/post-processing.
//
// # Legacy compatibility
//
// Dinic, EdmondsKarp, and FordFulkerson keep the historical tuple-return API:
//
//	maxFlow, residual, err := flow.Dinic(g, source, sink, flow.DefaultOptions())
//
// New code should prefer:
//
//	result, err := flow.MaxFlow(g, source, sink, flow.WithAlgorithm(flow.AlgorithmDinic))
//
// # AI-Hints
//
// Do not infer undirected residual endpoints through core.Neighbors. Build the
// flow adapter from core.Edges.
//
// Do not preserve input graph directedness on residual output. A residual network
// is mathematically directed and weighted.
//
// Do not add fmt.Printf inside kernels. Use WithObserver for instrumentation.
//
// Do not use string matching for errors. Use errors.Is.
//
// Do not replace proof-style tests with tests that only check the scalar max-flow
// value. A correct package must also prove residual no-path and min-cut capacity.
package flow
