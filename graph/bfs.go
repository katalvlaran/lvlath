package graph

import (
	"context"
	"errors"
)

// ErrVertexNotFound is returned when the start vertex does not exist.
var ErrVertexNotFound = errors.New("graph: start vertex not found")

// BFSOptions configures breadth-first search behavior.
type BFSOptions struct {
	// Ctx is optional. If non-nil, traversal aborts when ctx.Done() is signaled.
	Ctx context.Context

	// OnEnqueue(v, depth) is called immediately after v is enqueued.
	OnEnqueue func(v *Vertex, depth int)

	// OnDequeue(v, depth) is called just before v is visited.
	OnDequeue func(v *Vertex, depth int)

	// OnVisit(v, depth) is called the moment v is visited.
	// Returning a non-nil error aborts the traversal,
	// but v will already have been added to Order.
	OnVisit func(v *Vertex, depth int) error
}

// BFSResult holds the outcome of a BFS traversal.
type BFSResult struct {
	Order   []*Vertex         // sequence of visited vertices
	Depth   map[string]int    // Depth[v.ID] = distance from start
	Parent  map[string]string // Parent[v.ID] = predecessor ID
	Visited map[string]bool   // set of all visited IDs
}

// BFS performs breadth-first search starting at startID.
// If opts is nil, uses sane defaults (no callbacks, background context).
func (g *Graph) BFS(startID string, opts *BFSOptions) (*BFSResult, error) {
	// Prepare options & context
	topts := BFSOptions{}
	ctx := context.Background()
	if opts != nil {
		topts = *opts
		if opts.Ctx != nil {
			ctx = opts.Ctx
		}
	}

	// Validate start
	if !g.HasVertex(startID) {
		return nil, ErrVertexNotFound
	}

	// Init result
	res := &BFSResult{
		Order:   make([]*Vertex, 0),
		Depth:   make(map[string]int),
		Parent:  make(map[string]string),
		Visited: make(map[string]bool),
	}

	return res, g.bfsTraverse(startID, &topts, res, ctx)
}

// bfsTraverse contains the core BFS loop.
func (g *Graph) bfsTraverse(startID string, opts *BFSOptions, res *BFSResult, ctx context.Context) error {
	// Internal queue element
	type item struct {
		id    string
		depth int
	}

	// Seed queue
	queue := []item{{startID, 0}}
	res.Visited[startID] = true
	res.Depth[startID] = 0

	// Initial enqueue callback
	if opts.OnEnqueue != nil {
		opts.OnEnqueue(g.vertex(startID), 0)
	}

	// Loop
	for len(queue) > 0 {
		// Cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Dequeue
		it := queue[0]
		queue = queue[1:]
		v := g.vertex(it.id)

		// Dequeue hook
		if opts.OnDequeue != nil {
			opts.OnDequeue(v, it.depth)
		}

		// Record visit
		res.Order = append(res.Order, v)

		// Visit hook (may abort)
		if opts.OnVisit != nil {
			if err := opts.OnVisit(v, it.depth); err != nil {
				return err
			}
		}

		// Enqueue neighbors
		for _, nbr := range g.Neighbors(it.id) {
			if !res.Visited[nbr.ID] {
				res.Visited[nbr.ID] = true
				res.Parent[nbr.ID] = it.id
				nd := it.depth + 1
				res.Depth[nbr.ID] = nd

				if opts.OnEnqueue != nil {
					opts.OnEnqueue(nbr, nd)
				}
				queue = append(queue, item{nbr.ID, nd})
			}
		}
	}

	return nil
}

// vertex retrieves *Vertex by ID under read lock. Caller must ensure it exists.
func (g *Graph) vertex(id string) *Vertex {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.vertices[id]
}
