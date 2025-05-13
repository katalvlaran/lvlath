// Package algorithms implements graph algorithms on core.Graph.
//
// # BFS — Breadth-First Search
//
// Breadth-First Search explores the graph level by level, starting from a
// given vertex. It is ideal for finding shortest paths in unweighted graphs,
// checking connectivity, and layering vertices by distance.
//
// Steps:
//  1. Initialize:
//     - Mark start visited, depth=0, enqueue.
//     - Invoke OnEnqueue hook.
//  2. Loop until queue empty:
//     2.1 Dequeue an item (vertex, depth).
//     - Invoke OnDequeue hook.
//     2.2 Visit the vertex:
//     - Append to result.Order.
//     - Invoke OnVisit; if error, abort.
//     2.3 Enqueue unvisited neighbors:
//     - Mark visited, set parent and depth+1.
//     - Invoke OnEnqueue.
//  3. Check context cancellation before each dequeue.
//
// Time complexity: O(V + E)
// Memory usage:    O(V)
package algorithms

import (
	"context"
	"fmt"

	"github.com/katalvlaran/lvlath/core"
)

// ErrVertexNotFound is returned when the start vertex does not exist.
var ErrVertexNotFound = fmt.Errorf("algorithms: start vertex %q not found")

// BFSOptions configures traversal behavior.
type BFSOptions struct {
	// Ctx allows cancellation; if nil, context.Background() is used.
	Ctx context.Context

	// OnEnqueue(v, depth) is called immediately after v is enqueued.
	OnEnqueue func(v *core.Vertex, depth int)
	// OnDequeue(v, depth) is called just before v is dequeued.
	OnDequeue func(v *core.Vertex, depth int)
	// OnVisit(v, depth) is called when v is visited.
	// If it returns an error, traversal aborts (v is already in Order).
	OnVisit func(v *core.Vertex, depth int) error
}

// BFSResult holds the outcome of a BFS traversal.
type BFSResult struct {
	// Order is the sequence of visited vertices.
	Order []*core.Vertex
	// Depth maps vertex ID → distance (#edges) from start.
	Depth map[string]int
	// Parent maps vertex ID → predecessor ID in the BFS tree.
	Parent map[string]string
	// Visited tracks which vertices have been reached.
	Visited map[string]bool
}

// queueItem pairs a vertex ID with its BFS depth.
type queueItem struct {
	id    string
	depth int
}

// Complexity: O(V + E), Memory: O(V)
// BFS performs a breadth-first search on g from startID using opts.
// It returns a BFSResult and any error encountered (e.g. ErrVertexNotFound,
// context.Canceled, or a user-supplied OnVisit error).
func BFS(g *core.Graph, startID string, opts *BFSOptions) (*BFSResult, error) {
	// Prepare context and options
	ctx := context.Background()
	if opts != nil && opts.Ctx != nil {
		ctx = opts.Ctx
	}

	// Initialize result container
	res := &BFSResult{
		Order:   make([]*core.Vertex, 0),
		Depth:   make(map[string]int),
		Parent:  make(map[string]string),
		Visited: make(map[string]bool),
	}

	// walker encapsulates BFS state
	w := &walker{
		graph: g,
		opts:  opts,
		res:   res,
		ctx:   ctx,
		queue: make([]queueItem, 0, len(g.Vertices())),
	}

	// Seed queue
	if err := w.init(startID); err != nil {
		return res, err
	}
	// Process queue
	if err := w.loop(); err != nil {
		return res, err
	}

	return res, nil
}

// walker holds the mutable state for one BFS execution.
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
	// mark visited & depth=0
	w.res.Visited[startID] = true
	w.res.Depth[startID] = 0
	w.queue = append(w.queue, queueItem{id: startID, depth: 0})
	// OnEnqueue hook
	if w.opts != nil && w.opts.OnEnqueue != nil {
		v := w.graph.VerticesMap()[startID]
		w.opts.OnEnqueue(v, 0)
	}

	return nil
}

// loop processes the BFS queue until empty or error/cancellation.
func (w *walker) loop() error {
	for len(w.queue) > 0 {
		// cancellation
		select {
		case <-w.ctx.Done():
			return w.ctx.Err()
		default:
		}
		item := w.dequeue()
		if err := w.visit(item); err != nil {
			return err
		}
		w.enqueueNeighbors(item)
	}

	return nil
}

func (w *walker) dequeue() queueItem {
	item := w.queue[0]
	w.queue = w.queue[1:]
	if w.opts != nil && w.opts.OnDequeue != nil {
		v := w.graph.VerticesMap()[item.id]
		w.opts.OnDequeue(v, item.depth)
	}

	return item
}

func (w *walker) visit(item queueItem) error {
	v := w.graph.VerticesMap()[item.id]
	w.res.Order = append(w.res.Order, v)
	if w.opts != nil && w.opts.OnVisit != nil {
		if err := w.opts.OnVisit(v, item.depth); err != nil {
			return fmt.Errorf("OnVisit error at %q: %w", v.ID, err)
		}
	}

	return nil
}

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
