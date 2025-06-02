// Package dfs defines types and options for depth-first search traversal,
// including cancellation, pre-/post-order hooks, depth limiting, neighbor filtering,
// full-graph (forest) traversal, and basic diagnostics.
package dfs

import (
	"context"
	"errors"
)

// VertexState represents the DFS visitation state of a vertex.
const (
	White = iota // White: the vertex has not been visited yet.
	Gray         // Gray: the vertex is in the recursion stack (visiting).
	Black        // Black: the vertex and all its descendants have been fully explored.
)

var (
	// ErrGraphNil is returned when a nil *core.Graph is passed to DFS,
	// TopologicalSort, or DetectCycles.
	ErrGraphNil = errors.New("dfs: graph is nil")

	// ErrStartVertexNotFound indicates that the specified start vertex ID
	// does not exist in the graph.
	ErrStartVertexNotFound = errors.New("dfs: start vertex not found")

	// ErrCycleDetected indicates that a cycle was encountered during
	// TopologicalSort or DetectCycles.
	ErrCycleDetected = errors.New("dfs: cycle detected")
)

// Option configures optional behavior of DFS traversal.
// Use with DFS(g, startID, opts...).
type Option func(*DFSOptions)

// DFSOptions holds configurable parameters for DFS traversal.
// It controls hooks, limits, filtering, full-graph mode, and diagnostics.
// Complexity remains O(V+E) when filters and hooks are O(1).
type DFSOptions struct {
	// Ctx allows cancellation or timeouts; defaults to context.Background().
	// Cancelling the context will abort DFS early.
	Ctx context.Context

	// OnVisit, if non-nil, is invoked immediately upon discovering a vertex (pre-order).
	// Returning an error aborts traversal with that error.
	OnVisit func(id string) error

	// OnExit, if non-nil, is invoked after exploring all descendants of a vertex
	// have been explored (post-order), before appending to result.Order.
	// Returning an error aborts traversal and leaves Order empty.
	OnExit func(id string) error

	// MaxDepth, if non-negative, limits recursion to the given depth.
	// A depth of 0 visits only the start vertex. Default is -1 (no limit).
	MaxDepth int

	// FilterNeighbor, if non-nil, is called for each neighbor ID before recurse.
	// Return true to traverse into that neighbor, false to skip it.
	FilterNeighbor func(id string) bool

	// FullTraversal, if true, runs DFS from every unvisited vertex in the graph,
	// covering disconnected components (forest traversal). Default is false.
	FullTraversal bool

	// SkippedNeighbors tracks how many neighbor vertices were skipped
	// due to FilterNeighbor returning false. Useful for diagnostics.
	SkippedNeighbors int
}

// DefaultOptions returns a DFSOptions struct with:
//   - Background context
//   - No pre-/post-order hooks
//   - No depth limit (MaxDepth = -1)
//   - No neighbor filtering
//   - Single-source traversal (FullTraversal = false)
func DefaultOptions() DFSOptions {
	return DFSOptions{
		Ctx:              context.Background(),
		OnVisit:          nil,
		OnExit:           nil,
		MaxDepth:         -1,
		FilterNeighbor:   nil,
		FullTraversal:    false,
		SkippedNeighbors: 0,
	}
}

// WithContext returns an Option that sets the Context for DFS traversal.
// Passing a nil context has no effect (Background is retained).
func WithContext(ctx context.Context) Option {
	return func(o *DFSOptions) {
		if ctx != nil {
			o.Ctx = ctx // use provided context for cancellation
		}
	}
}

// WithOnVisit returns an Option that installs fn as a pre-order hook.
// The hook is called when a vertex is first discovered.
func WithOnVisit(fn func(id string) error) Option {
	return func(o *DFSOptions) {
		o.OnVisit = fn
	}
}

// WithOnExit returns an Option that installs fn as a post-order hook.
// The hook is called after a vertex’s descendants have been fully explored.
func WithOnExit(fn func(id string) error) Option {
	return func(o *DFSOptions) {
		o.OnExit = fn
	}
}

// WithMaxDepth returns an Option that limits traversal depth to limit.
// A limit of 0 means only the start vertex is visited.
func WithMaxDepth(limit int) Option {
	return func(o *DFSOptions) {
		o.MaxDepth = limit
	}
}

// WithFilterNeighbor returns an Option that filters neighbor IDs.
// If fn(id) == false, that neighbor is skipped and counted in SkippedNeighbors.
func WithFilterNeighbor(fn func(id string) bool) Option {
	return func(o *DFSOptions) {
		o.FilterNeighbor = fn
	}
}

// WithFullTraversal returns an Option that enables full-graph traversal.
// When set, DFS will restart from each unvisited vertex, covering disconnected components.
func WithFullTraversal() Option {
	return func(o *DFSOptions) {
		o.FullTraversal = true
	}
}

// DFSResult captures the outcome of a depth-first traversal.
// It reports post-order, discovery depths, parent links, and visited flags,
// as well as diagnostics like SkippedNeighbors.
type DFSResult struct {
	// Order records vertices in the sequence they finished (post-order).
	Order []string

	// Depth maps each vertex ID to its distance (#edges) from the start.
	Depth map[string]int

	// Parent maps each vertex ID to the ID of the vertex from which it was first discovered.
	// The start vertex will not appear in this map for each DFS tree.
	Parent map[string]string

	// Visited flags which vertices were reached during the traversal.
	Visited map[string]bool

	// SkippedNeighborsMirror reports how many neighbors were skipped
	// due to FilterNeighbor returning false, aggregated across all trees.
	SkippedNeighbors int
}
