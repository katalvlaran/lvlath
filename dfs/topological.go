// Package dfs provides core algorithms on directed graphs, including
// topological sort.
//
// TopologicalSort computes a linear ordering of vertices such that for
// every directed edge u→v, u appears before v in the ordering.
// If the graph contains a cycle, ErrCycleDetected is returned.
// If neighbor iteration fails, ErrNeighborFetch is returned.
//
// Complexity:
//
//   - Time:   O(V + E) (each vertex and edge visited once)
//   - Memory: O(V)     (recursion stack and state map)
package dfs

import (
	"context"
	"errors"
	"fmt"

	"github.com/katalvlaran/lvlath/core"
)

// ErrNeighborFetch indicates a failure to retrieve neighbors from the graph.
var ErrNeighborFetch = errors.New("dfs: failed to fetch neighbors")

// TopoOption configures optional behavior for TopologicalSort.
type TopoOption func(*topoOptions)

// topoOptions holds settings for TopologicalSort, currently only cancellation.
type topoOptions struct {
	ctx context.Context // allows cancellation; defaults to Background
}

// defaultTopoOptions returns the default options (Background context).
func defaultTopoOptions() topoOptions {
	return topoOptions{ctx: context.Background()}
}

// WithCancelContext returns a TopoOption that sets the cancellation context.
// Passing a nil context has no effect.
func WithCancelContext(ctx context.Context) TopoOption {
	return func(o *topoOptions) {
		if ctx != nil {
			o.ctx = ctx
		}
	}
}

// topoSorter encapsulates state for a topological sort traversal.
type topoSorter struct {
	graph *core.Graph    // the graph being sorted
	opts  topoOptions    // traversal options (cancellation)
	state map[string]int // visitation state: 0=White,1=Gray,2=Black
	order []string       // recorded post-order sequence
}

// TopologicalSort computes a topological ordering of all vertices in g.
// If g is nil, returns ErrGraphNil.
// If g is undirected, returns an error.
// If a cycle is detected, returns ErrCycleDetected.
// If neighbor lookup fails, returns ErrNeighborFetch.
// You may pass WithCancelContext(ctx) to enable cancellation.
func TopologicalSort(g *core.Graph, options ...TopoOption) ([]string, error) {
	// 1. Validate graph pointer
	if g == nil {
		return nil, ErrGraphNil
	}
	// 2. Only directed graphs are supported
	if !g.Directed() {
		return nil, fmt.Errorf("dfs: TopologicalSort requires directed graph")
	}
	// 3. Apply optional settings
	opts := defaultTopoOptions()
	for _, opt := range options {
		opt(&opts)
	}
	// 4. Initialize sorter state
	verts := g.Vertices() // sorted list of vertex IDs
	sorter := &topoSorter{
		graph: g,
		opts:  opts,
		state: make(map[string]int, len(verts)), // all vertices start as White (0)
		order: make([]string, 0, len(verts)),    // capacity hint for post-order
	}
	// 5. Drive DFS from every unvisited vertex
	for _, v := range verts {
		if sorter.state[v] == White {
			if err := sorter.visit(v); err != nil {
				return nil, err
			}
		}
	}
	// 6. Reverse post-order to produce topological order
	for i, j := 0, len(sorter.order)-1; i < j; i, j = i+1, j-1 {
		sorter.order[i], sorter.order[j] = sorter.order[j], sorter.order[i]
	}

	return sorter.order, nil
}

// visit performs a DFS from id, marking states and detecting cycles.
// It respects cancellation, skips any undirected edges, and wraps neighbor errors.
func (t *topoSorter) visit(id string) error {
	// 1. Cancellation check at entry
	select {
	case <-t.opts.ctx.Done():
		return t.opts.ctx.Err()
	default:
	}
	// 2. Cycle detection: if already Gray, we found a back-edge
	if t.state[id] == Gray {
		return ErrCycleDetected
	}
	// 3. Already fully processed (Black)? then skip
	if t.state[id] == Black {
		return nil
	}
	// 4. Mark as in-progress (Gray)
	t.state[id] = Gray

	// 5. Retrieve neighbors (incoming undirected edges are filtered next)
	neighbors, err := t.graph.Neighbors(id)
	if err != nil {
		// Wrap in sentinel ErrNeighborFetch so callers can check via errors.Is
		return fmt.Errorf("%w: %v", ErrNeighborFetch, err)
	}
	// 6. Explore each outgoing directed edge
	for _, e := range neighbors {
		// 6a. Only honor edges explicitly marked Directed (skip undirected)
		if !e.Directed {
			continue
		}
		// 6b. Safety: skip edges that do not originate from current id
		if e.From != id {
			continue
		}
		// 6c. Recurse into neighbor
		if err = t.visit(e.To); err != nil {
			return err
		}
	}

	// 7. Mark as fully explored (Black)
	t.state[id] = Black
	// 8. Record in post-order list
	t.order = append(t.order, id)

	return nil
}
