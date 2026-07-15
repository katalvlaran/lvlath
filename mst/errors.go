// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package mst defines sentinel errors for MST graph policy,
// option policy, result access, and numeric validation.
package mst

import "errors"

// ErrInvalidGraph indicates that the input graph violates the MST graph policy.
// It is kept as a broad compatibility sentinel and may be joined with precise graph-policy errors.
var ErrInvalidGraph = errors.New("mst: invalid graph for MST")

// ErrNilGraph indicates that a nil *core.Graph was provided.
// MST algorithms require an existing graph instance before any topology can be inspected.
var ErrNilGraph = errors.New("mst: nil graph")

// ErrDirectedGraph indicates that the graph-level default direction is directed.
// Minimum spanning trees are defined for undirected weighted graphs in this package.
var ErrDirectedGraph = errors.New("mst: MST requires an undirected graph")

// ErrDirectedEdge indicates that at least one edge has directed edge-level semantics.
// Mixed or directed edge catalogs are rejected because MST consumes an undirected relation.
var ErrDirectedEdge = errors.New("mst: MST requires no directed edges")

// ErrUnweightedGraph indicates that the graph does not expose weighted edge semantics.
// MST algorithms need finite edge weights to define an ordering over candidates.
var ErrUnweightedGraph = errors.New("mst: MST requires a weighted graph")

// ErrEmptyGraph indicates that the graph has no vertices.
// It is joined with ErrDisconnected because strict MST cannot span an empty graph.
var ErrEmptyGraph = errors.New("mst: graph has no vertices")

// ErrDisconnected indicates that strict tree mode cannot connect all vertices.
// Forest mode is the explicit opt-in policy for disconnected graphs.
var ErrDisconnected = errors.New("mst: graph is disconnected")

// ErrEmptyRoot indicates that Prim was requested without a root in strict tree mode.
// Prim needs a concrete start vertex unless explicit forest mode chooses component roots.
var ErrEmptyRoot = errors.New("mst: empty root vertex")

// ErrUnsupportedAlgorithm indicates that the requested algorithm is not recognized.
// It classifies option/dispatch failures, not graph topology failures.
var ErrUnsupportedAlgorithm = errors.New("mst: unsupported MST algorithm")

// ErrInvalidOption indicates that an option value is structurally invalid.
// This sentinel is reserved for option states that do not have a more specific sentinel.
var ErrInvalidOption = errors.New("mst: invalid option")

// ErrNilOption indicates that a nil Option was passed to option assembly.
// Public option assembly never panics on ordinary user-provided nil options.
var ErrNilOption = errors.New("mst: nil option")

// ErrNaNInfWeight indicates that at least one graph edge has a NaN or infinite weight.
// MST ordering requires finite weights; negative finite weights remain valid.
var ErrNaNInfWeight = errors.New("mst: edge weight is NaN or Inf")

// ErrNilResult indicates that a method requiring a non-nil Result receiver was called on nil.
// Result helpers use this sentinel instead of panicking on nil receivers.
var ErrNilResult = errors.New("mst: nil result")
