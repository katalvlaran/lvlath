// Package algorithms implements graph algorithms on core.Graph.
//
// # DFS — Depth-First Search
//
// Depth-First Search explores as far as possible along each branch
// before backtracking. It is useful for connectivity analysis,
// cycle detection, and topological sorting.
//
// Steps:
//  1. Initialize:
//     - Validate start vertex.
//     - Prepare visited set, depth and parent maps.
//  2. Recursively traverse:
//     2.1 Check for cancellation.
//     2.2 Mark current visited, record depth.
//     2.3 Append to Order, invoke OnVisit.
//     2.4 For each unvisited neighbor:
//     - Set Parent, recurse with depth+1.
//     2.5 After all children, invoke OnExit.
//  3. Unwind recursion to finish.
//
// Time complexity: O(V + E)
// Memory usage:    O(V) for visited + recursion stack
package algorithms

import (
	"context"
	"fmt"

	"github.com/katalvlaran/lvlath/core"
)

// ErrDFSVertexNotFound is returned when the start vertex is absent.
var ErrDFSVertexNotFound = ErrVertexNotFound

// DFSOptions configures the DFS traversal.
type DFSOptions struct {
	// Ctx allows cancellation; if nil, background context is used.
	Ctx context.Context
	// OnVisit(v, depth) is called when v is first visited.
	// Returning an error aborts traversal (v is in Order).
	OnVisit func(v *core.Vertex, depth int) error

	// OnExit(v, depth) is called after all descendants of v are processed.
	OnExit func(v *core.Vertex, depth int)
}

// DFSResult holds the outcome of a DFS traversal.
type DFSResult struct {
	// Order is the sequence of visited vertices.
	Order []*core.Vertex
	// Depth[v.ID] = recursion depth from start.
	Depth map[string]int
	// Parent[v.ID] = predecessor in DFS tree.
	Parent map[string]string
	// Visited tracks reached vertices.
	Visited map[string]bool
}

// dfsWalker encapsulates DFS state.
type dfsWalker struct {
	g    *core.Graph
	opts *DFSOptions
	res  *DFSResult
	ctx  context.Context
}

// Complexity: O(V + E), Memory: O(V)
// DFS performs a depth-first search on g from startID using opts.
// Returns a DFSResult or an error (ErrDFSVertexNotFound, context.Canceled, OnVisit error).
func DFS(g *core.Graph, startID string, opts *DFSOptions) (*DFSResult, error) {
	// prepare context
	ctx := context.Background()
	if opts != nil && opts.Ctx != nil {
		ctx = opts.Ctx
	}

	// init result
	res := &DFSResult{
		Order:   make([]*core.Vertex, 0),
		Depth:   make(map[string]int),
		Parent:  make(map[string]string),
		Visited: make(map[string]bool),
	}
	w := &dfsWalker{g: g, opts: opts, res: res, ctx: ctx}

	// validate
	if !g.HasVertex(startID) {
		return res, ErrDFSVertexNotFound
	}
	// start recursion
	if err := w.traverse(startID, 0); err != nil {
		return res, err
	}

	return res, nil
}

func (w *dfsWalker) traverse(id string, depth int) error {
	// cancellation
	select {
	case <-w.ctx.Done():
		return w.ctx.Err()
	default:
	}

	// mark and record
	w.res.Visited[id] = true
	w.res.Depth[id] = depth
	v := w.g.VerticesMap()[id]
	w.res.Order = append(w.res.Order, v)

	// OnVisit hook
	if w.opts != nil && w.opts.OnVisit != nil {
		if err := w.opts.OnVisit(v, depth); err != nil {
			return fmt.Errorf("OnVisit error at %q: %w", id, err)
		}
	}

	// recurse
	for _, nbr := range w.g.Neighbors(id) {
		if !w.res.Visited[nbr.ID] {
			w.res.Parent[nbr.ID] = id
			if err := w.traverse(nbr.ID, depth+1); err != nil {
				return err
			}
		}
	}

	// OnExit hook
	if w.opts != nil && w.opts.OnExit != nil {
		w.opts.OnExit(v, depth)
	}

	return nil
}
