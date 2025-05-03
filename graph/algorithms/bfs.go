// Package algorithms implements graph algorithms on core.Graph.
package algorithms

import (
	"context"
	"errors"
	"fmt"

	"github.com/katalvlaran/lvlath/graph/core"
)

/*
BFS — Breadth‐First Search

Description:
  BFS explores the graph level by level, starting from a given vertex.
  Useful for finding shortest paths in unweighted graphs, checking connectivity,
  and layering by distance.

Steps:
  1. Initialize:
     - Mark start visited, depth=0, enqueue.
     - Invoke OnEnqueue hook.
  2. Loop until queue empty:
     2.1 Dequeue an item (vertex, depth).
         - Invoke OnDequeue hook.
     2.2 Visit the vertex:
         - Append to result.Order.
         - Invoke OnVisit; if error, abort.
     2.3 Enqueue unvisited neighbors:
         - Mark visited, set parent and depth+1.
         - Invoke OnEnqueue.
  3. Check context cancellation before each dequeue.

Complexity: O(V + E)
Memory:     O(V)
*/

// ErrVertexNotFound is returned when the start vertex is absent.
var ErrVertexNotFound = errors.New("graph: start vertex not found")

// BFSOptions configures the BFS traversal.
type BFSOptions struct {
	// Ctx allows cancellation; if nil, background context is used.
	Ctx context.Context

	// OnEnqueue(v, depth) is called immediately after enqueue.
	OnEnqueue func(v *core.Vertex, depth int)
	// OnDequeue(v, depth) is called just before visiting neighbors.
	OnDequeue func(v *core.Vertex, depth int)
	// OnVisit(v, depth) is called when v is visited.
	// Returning non-nil error aborts traversal, but v is already in Order.
	OnVisit func(v *core.Vertex, depth int) error
}

// BFSResult holds the outcome of a BFS traversal.
type BFSResult struct {
	// Order of visitation.
	Order []*core.Vertex
	// Depth[v.ID] = distance (#edges) from start.
	Depth map[string]int
	// Parent[v.ID] = predecessor ID in BFS tree.
	Parent map[string]string
	// Visited set of all reached vertices.
	Visited map[string]bool
}

// queueItem holds a vertex ID and its BFS depth.
type queueItem struct {
	id    string
	depth int
}

// BFS performs a breadth-first search on g from startID with opts.
// It returns BFSResult or an error.
func BFS(g *core.Graph, startID string, opts *BFSOptions) (*BFSResult, error) {
	// Prepare context and options
	ctx := context.Background()
	if opts != nil && opts.Ctx != nil {
		ctx = opts.Ctx
	}
	// Initialize result and walker
	res := &BFSResult{
		Order:   make([]*core.Vertex, 0),
		Depth:   make(map[string]int),
		Parent:  make(map[string]string),
		Visited: make(map[string]bool),
	}
	w := &walker{
		graph: g,
		opts:  opts,
		res:   res,
		ctx:   ctx,
		queue: make([]queueItem, 0),
	}
	// init and run
	if err := w.init(startID); err != nil {
		return res, err
	}
	if err := w.loop(); err != nil {
		return res, err
	}
	return res, nil
}

// walker encapsulates BFS state for one run.
type walker struct {
	graph *core.Graph
	opts  *BFSOptions
	res   *BFSResult
	ctx   context.Context
	queue []queueItem
}

// init seeds the queue with startID, marks visited, depth=0, and invokes OnEnqueue.
func (w *walker) init(startID string) error {
	if !w.graph.HasVertex(startID) {
		return ErrVertexNotFound
	}
	// seed
	w.res.Visited[startID] = true
	w.res.Depth[startID] = 0
	w.queue = append(w.queue, queueItem{id: startID, depth: 0})
	// hook
	if w.opts != nil && w.opts.OnEnqueue != nil {
		v := w.graph.VerticesMap()[startID]
		w.opts.OnEnqueue(v, 0)
	}
	return nil
}

// loop processes the BFS queue until empty or error/cancellation.
func (w *walker) loop() error {
	for len(w.queue) > 0 {
		// check for cancellation
		if err := w.checkCancel(); err != nil {
			return err
		}
		item := w.dequeue()
		if err := w.visit(item); err != nil {
			return err
		}
		w.enqueueNeighbors(item)
	}
	return nil
}

// checkCancel aborts if context is done.
func (w *walker) checkCancel() error {
	select {
	case <-w.ctx.Done():
		return w.ctx.Err()
	default:
		return nil
	}
}

// dequeue removes the first queueItem, invokes OnDequeue, and returns it.
func (w *walker) dequeue() queueItem {
	item := w.queue[0]
	w.queue = w.queue[1:]
	if w.opts != nil && w.opts.OnDequeue != nil {
		v := w.graph.VerticesMap()[item.id]
		w.opts.OnDequeue(v, item.depth)
	}
	return item
}

// visit records the vertex in Order and calls OnVisit.
func (w *walker) visit(item queueItem) error {
	v := w.graph.VerticesMap()[item.id]
	// append before hook so even if error, visit is recorded
	w.res.Order = append(w.res.Order, v)
	if w.opts != nil && w.opts.OnVisit != nil {
		if err := w.opts.OnVisit(v, item.depth); err != nil {
			return fmt.Errorf("OnVisit error at %q: %w", v.ID, err)
		}
	}
	return nil
}

// enqueueNeighbors enqueues all unvisited neighbors of item.
func (w *walker) enqueueNeighbors(item queueItem) {
	for _, nbr := range w.graph.Neighbors(item.id) {
		if !w.res.Visited[nbr.ID] {
			w.res.Visited[nbr.ID] = true
			w.res.Parent[nbr.ID] = item.id
			d := item.depth + 1
			w.res.Depth[nbr.ID] = d
			if w.opts != nil && w.opts.OnEnqueue != nil {
				w.opts.OnEnqueue(nbr, d)
			}
			w.queue = append(w.queue, queueItem{id: nbr.ID, depth: d})
		}
	}
}
