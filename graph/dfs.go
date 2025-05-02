package graph

import (
	"context"
)

// DFSOptions configures depth-first search behavior.
type DFSOptions struct {
	// Ctx is optional. If non-nil, traversal aborts when ctx.Done() is signaled.
	Ctx context.Context

	// OnVisit(v, depth) is called when v is first visited.
	// Returning a non-nil error aborts traversal,
	// but v will already have been added to Order.
	OnVisit func(v *Vertex, depth int) error

	// OnExit(v, depth) is called after all descendants of v have been processed.
	OnExit func(v *Vertex, depth int)
}

// DFSResult holds the outcome of a DFS traversal.
type DFSResult struct {
	Order   []*Vertex         // sequence of visited vertices
	Depth   map[string]int    // Depth[v.ID] = recursion depth
	Parent  map[string]string // Parent[v.ID] = predecessor ID
	Visited map[string]bool   // set of all visited IDs
}

// ErrDFSVertexNotFound is returned when the start vertex does not exist.
var ErrDFSVertexNotFound = ErrVertexNotFound

// DFS performs depth-first search starting at startID.
// If opts is nil, uses sane defaults (no callbacks, background context).
func (g *Graph) DFS(startID string, opts *DFSOptions) (*DFSResult, error) {
	// Prepare options & context
	topts := DFSOptions{}
	ctx := context.Background()
	if opts != nil {
		topts = *opts
		if opts.Ctx != nil {
			ctx = opts.Ctx
		}
	}

	// Validate start
	if !g.HasVertex(startID) {
		return nil, ErrDFSVertexNotFound
	}

	// Init result
	res := &DFSResult{
		Order:   make([]*Vertex, 0),
		Depth:   make(map[string]int),
		Parent:  make(map[string]string),
		Visited: make(map[string]bool),
	}

	return res, g.dfsTraverse(startID, 0, &topts, res, ctx)
}

// dfsTraverse is the recursive core of DFS.
func (g *Graph) dfsTraverse(id string, depth int, opts *DFSOptions, res *DFSResult, ctx context.Context) error {
	// Cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Mark visit
	res.Visited[id] = true
	res.Depth[id] = depth
	v := g.vertex(id)

	// Record order
	res.Order = append(res.Order, v)

	// Visit hook
	if opts.OnVisit != nil {
		if err := opts.OnVisit(v, depth); err != nil {
			return err
		}
	}

	// Recurse neighbors
	for _, nbr := range g.Neighbors(id) {
		if !res.Visited[nbr.ID] {
			res.Parent[nbr.ID] = id
			if err := g.dfsTraverse(nbr.ID, depth+1, opts, res, ctx); err != nil {
				return err
			}
		}
	}

	// Exit hook
	if opts.OnExit != nil {
		opts.OnExit(v, depth)
	}
	return nil
}
