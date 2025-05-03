// Package algorithms implements graph algorithms on core.Graph.
/*
DFS — Depth‐First Search

Description:
  DFS explores as far as possible along each branch before backtracking.
  Useful for connectivity, cycle detection, topological sorting.

Steps:
  1. Initialize:
     - Validate start vertex.
     - Prepare visited set, depth and parent maps.
  2. Recursively traverse:
     2.1 Check for cancellation.
     2.2 Mark current visited, record depth.
     2.3 Append to Order, invoke OnVisit.
     2.4 For each unvisited neighbor:
         - Set Parent, recurse with depth+1.
     2.5 After all children, invoke OnExit.
  3. Unwind recursion to finish.

Complexity: O(V + E)
Memory:     O(V) for visited + call stack up to depth V.
*/
package algorithms

import (
	"context"
	"fmt"

	"github.com/katalvlaran/lvlath/graph/core"
)

// ErrDFSVertexNotFound is returned when the start vertex does not exist.
var ErrDFSVertexNotFound = ErrVertexNotFound

// DFSOptions configures the DFS traversal.
type DFSOptions struct {
	// Ctx allows cancellation; if nil, background context is used.
	Ctx context.Context
	// OnVisit(v, depth) is called when v is first visited.
	// Returning a non-nil error aborts traversal (v is already in Order).
	OnVisit func(v *core.Vertex, depth int) error
	// OnExit(v, depth) is called after all descendants of v are processed.
	OnExit func(v *core.Vertex, depth int)
}

// DFSResult holds the outcome of a DFS traversal.
type DFSResult struct {
	// Order of visitation.
	Order []*core.Vertex
	// Depth[v.ID] = recursion depth from start.
	Depth map[string]int
	// Parent[v.ID] = predecessor ID in DFS tree.
	Parent map[string]string
	// Visited set of all reached vertices.
	Visited map[string]bool
}

// DFS performs a depth‐first search on g from startID using opts.
func DFS(g *core.Graph, startID string, opts *DFSOptions) (*DFSResult, error) {
	// Prepare context
	ctx := context.Background()
	if opts != nil && opts.Ctx != nil {
		ctx = opts.Ctx
	}

	// Initialize result container
	res := &DFSResult{
		Order:   make([]*core.Vertex, 0),
		Depth:   make(map[string]int),
		Parent:  make(map[string]string),
		Visited: make(map[string]bool),
	}

	// Create walker
	w := &dfsWalker{
		g:    g,
		opts: opts,
		res:  res,
		ctx:  ctx,
	}

	// Validate start vertex
	if !g.HasVertex(startID) {
		return res, ErrDFSVertexNotFound
	}

	// Begin recursion
	if err := w.traverse(startID, 0); err != nil {
		return res, err
	}
	return res, nil
}

// dfsWalker holds DFS state for one run.
type dfsWalker struct {
	g    *core.Graph
	opts *DFSOptions
	res  *DFSResult
	ctx  context.Context
}

// traverse visits id at given depth, recurses into neighbors.
func (w *dfsWalker) traverse(id string, depth int) error {
	// Cancellation check
	if err := w.checkCancel(); err != nil {
		return err
	}

	// Mark visited and record depth
	w.res.Visited[id] = true
	w.res.Depth[id] = depth

	// Get vertex pointer
	v := w.g.VerticesMap()[id]

	// Record order before hooks (so it's logged even if OnVisit errors)
	w.res.Order = append(w.res.Order, v)

	// OnVisit hook
	if w.opts != nil && w.opts.OnVisit != nil {
		if err := w.opts.OnVisit(v, depth); err != nil {
			return fmt.Errorf("OnVisit error at %q: %w", id, err)
		}
	}

	// Recurse into neighbors
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

// checkCancel aborts traversal if context is done.
func (w *dfsWalker) checkCancel() error {
	select {
	case <-w.ctx.Done():
		return w.ctx.Err()
	default:
		return nil
	}
}
