// Package bfs provides breadth-first search over a core.Graph,
// returning unweighted shortest-path distances, parent links, and visit order.
//
// BFS explores vertices in increasing distance from a start vertex,
// with optional hooks, depth limiting, and neighbor filtering.
package bfs

import (
	"context"
	"errors"
	"fmt"

	"github.com/katalvlaran/lvlath/core"
)

// ErrWeightedGraph is returned when BFS is run on a weighted graph.
var ErrWeightedGraph = errors.New("bfs: weighted graphs not supported")

// ErrNeighbors is returned when fetching neighbors from the graph fails.
var ErrNeighbors = errors.New("bfs: neighbor iteration error")

// queueItem pairs a vertex ID with its BFS depth and its parent's ID.
type queueItem struct {
	id     string
	depth  int
	parent string // empty for root
}

// walker encapsulates mutable BFS state.
type walker struct {
	graph   *core.Graph
	opts    BFSOptions
	ctx     context.Context
	queue   []queueItem
	visited map[string]bool
	res     *BFSResult
}

// BFS runs breadth-first search on g starting from startID,
// applying any number of functional Options.
// Returns ErrGraphNil or ErrStartVertexNotFound for invalid input,
// ErrWeightedGraph for weighted graphs, ErrOptionViolation for bad options,
// ErrNeighbors for graph failures, or any user-supplied hook error.
func BFS(g *core.Graph, startID string, opts ...Option) (*BFSResult, error) {
	if g == nil {
		return nil, ErrGraphNil
	}
	// Build options and catch any invalid ones immediately
	o := DefaultOptions()
	for _, opt := range opts {
		opt(&o)
	}
	if o.err != nil {
		return nil, o.err
	}

	// Validate start vertex
	if !g.HasVertex(startID) {
		return nil, ErrStartVertexNotFound
	}
	// Disallow weighted graphs
	if g.Weighted() {
		return nil, ErrWeightedGraph
	}

	// Prepare walker
	vertices := g.Vertices()
	n := len(vertices)
	w := &walker{
		graph:   g,
		opts:    o,
		ctx:     o.Ctx,
		queue:   make([]queueItem, 0, n),
		visited: make(map[string]bool, n),
		res: &BFSResult{
			Order:  make([]string, 0, n),
			Depth:  make(map[string]int, n),
			Parent: make(map[string]string, n),
		},
	}

	// Seed queue with start vertex (no parent)
	w.enqueue(startID, 0, "")
	// Main loop
	return w.res, w.loop()
}

// enqueue marks id visited at depth d, calls OnEnqueue, records its parent,
// and adds it to the queue.
func (w *walker) enqueue(id string, d int, parent string) {
	w.visited[id] = true
	w.res.Depth[id] = d
	if parent != "" {
		w.res.Parent[id] = parent
	}
	w.opts.OnEnqueue(id, d)
	w.queue = append(w.queue, queueItem{id: id, depth: d, parent: parent})
}

// loop processes the queue until empty, error, or cancellation.
func (w *walker) loop() error {
	for len(w.queue) > 0 {
		// cancellation check (once per loop)
		select {
		case <-w.ctx.Done():
			return w.ctx.Err()
		default:
		}

		item := w.dequeue()
		if err := w.visit(item); err != nil {
			return err
		}
		if err := w.enqueueNeighbors(item); err != nil {
			return err
		}
	}
	return nil
}

// dequeue pops the first item, invokes OnDequeue, and returns it.
func (w *walker) dequeue() queueItem {
	item := w.queue[0]
	w.queue = w.queue[1:]
	w.opts.OnDequeue(item.id, item.depth)
	return item
}

// visit records the vertex in Order and calls OnVisit.
func (w *walker) visit(item queueItem) error {
	w.res.Order = append(w.res.Order, item.id)
	if err := w.opts.OnVisit(item.id, item.depth); err != nil {
		return fmt.Errorf("bfs: OnVisit error at %q: %w", item.id, err)
	}
	return nil
}

// enqueueNeighbors retrieves neighbors, applies filtering and MaxDepth,
// and enqueues each unseen neighbor. Returns ErrNeighbors on lookup failure.
func (w *walker) enqueueNeighbors(item queueItem) error {
	neighbors, err := w.graph.NeighborIDs(item.id)
	if err != nil {
		return fmt.Errorf("%w: failed to get neighbors of %q: %v", ErrNeighbors, item.id, err)
	}
	for _, nbr := range neighbors {
		// cancellation check inside neighbor iteration
		select {
		case <-w.ctx.Done():
			return w.ctx.Err()
		default:
		}

		// now you have the true neighbor
		if !w.opts.FilterNeighbor(item.id, nbr) {
			continue
		}
		nextDepth := item.depth + 1
		if w.opts.MaxDepth > 0 && nextDepth > w.opts.MaxDepth {
			continue
		}

		// first time seen?
		if !w.visited[nbr] {
			w.enqueue(nbr, nextDepth, item.id)
		}
	}
	return nil
}
