// Package prim_kruskal defines configuration options and sentinel errors for MST computation.
// It supports selecting between Kruskal and Prim algorithms via MSTOptions.
package prim_kruskal

import (
	"errors"

	"github.com/katalvlaran/lvlath/core"
)

// ErrInvalidGraph indicates that MST algorithms require an undirected, weighted graph.
// Returned when graph is nil, directed, or unweighted.
var ErrInvalidGraph = errors.New("prim_kruskal: MST requires undirected, weighted graph")

// ErrEmptyRoot indicates that no start vertex was specified for Prim.
// Prim cannot run without a valid root string.
var ErrEmptyRoot = errors.New("prim_kruskal: empty root vertex")

// ErrDisconnected indicates that the graph is not fully connected, so a spanning
// tree covering all vertices cannot be formed. It applies when |V| > 1 but MST is impossible.
var ErrDisconnected = errors.New("prim_kruskal: graph is disconnected")

// MethodPrim selects Prim's algorithm (grow from a root using a min-heap).
const MethodPrim = "prim"

// MethodKruskal selects Kruskal's algorithm (sort all edges and union-find).
const MethodKruskal = "kruskal"

// MSTOptions configures which MST algorithm to run, and for Prim, which starting vertex to use.
// Use DefaultOptions() to get a default setup (Kruskal).
//
// Fields:
//
//	Method string — one of MethodPrim or MethodKruskal.
//	Root   string — start vertex ID for Prim; ignored when Method == MethodKruskal.
//
// See: prim_kruskal.Prim, prim_kruskal.Kruskal
// Complexity: O(E log V) for Prim, O(E log E + α(V)·E) for Kruskal.
type MSTOptions struct {
	// Method to use: MethodPrim or MethodKruskal.
	Method string

	// Root is the starting vertex for Prim's algorithm. Unused by Kruskal.
	Root string
}

// Option configures MSTOptions. All Option functions should modify the pointed MSTOptions.
type Option func(*MSTOptions)

// WithMethod returns an Option that sets the algorithm Method.
// Allowed values: MethodPrim, MethodKruskal.
func WithMethod(m string) Option {
	return func(opts *MSTOptions) {
		opts.Method = m
	}
}

// WithRoot returns an Option that sets the starting vertex for Prim's algorithm and ignore by Kruskal.
func WithRoot(root string) Option {
	return func(opts *MSTOptions) {
		opts.Root = root
	}
}

// DefaultOptions returns MSTOptions initialized for Kruskal by default:
//
//	– Method = MethodKruskal
//	– Root   = "" (ignored by Kruskal).
//
// Complexity: O(1) to construct.
func DefaultOptions() MSTOptions {
	return MSTOptions{
		Method: MethodKruskal,
		Root:   "",
	}
}

// Compute selects and runs the MST algorithm based on opts.Method.
//
//	– If opts.Method == MethodKruskal: calls Kruskal(graph).
//	– If opts.Method == MethodPrim:    calls Prim(graph, opts.Root).
//	– Otherwise:                        returns ErrInvalidGraph.
//
// Returns:
//
//	[]core.Edge — slice of edges in MST (empty if graph has single vertex).
//	int64       — total weight of MST (zero if no edges).
//	error       — non-nil if computation cannot proceed.
//
// Note: this is optional scaffolding—methods Prim and Kruskal can still be called directly.
func Compute(graph *core.Graph, opts MSTOptions) ([]core.Edge, int64, error) {
	// Dispatch by method name
	switch opts.Method {
	case MethodKruskal:
		return Kruskal(graph)
	case MethodPrim:
		return Prim(graph, opts.Root)
	default:
		// Unknown method name
		return nil, 0, ErrInvalidGraph
	}
}
