// Package bfs provides tunable options and error definitions
// for breadth‐first search over a core.Graph.
package bfs

import (
	"context"
	"errors"
	"fmt"
)

// Sentinel errors for BFS execution.
var (
	// ErrStartVertexNotFound is returned when the start ID is absent.
	ErrStartVertexNotFound = errors.New("bfs: start vertex not found")

	// ErrGraphNil is returned if a nil graph pointer is passed.
	ErrGraphNil = errors.New("bfs: graph is nil")

	// ErrOptionViolation is returned when an invalid Option is supplied.
	ErrOptionViolation = errors.New("bfs: invalid option supplied")
)

// Option configures BFS behavior via functional arguments.
// If an Option is invalid (e.g. negative depth), it will be recorded
// internally and surfaced as ErrOptionViolation when BFS is invoked.
type Option func(*BFSOptions)

// BFSOptions holds parameters and callbacks to customize BFS execution.
type BFSOptions struct {
	// Ctx allows cancellation and deadlines.
	Ctx context.Context

	// OnEnqueue is called when a vertex is enqueued, before visiting.
	// Receives vertex ID and its depth from the start.
	OnEnqueue func(id string, depth int)

	// OnDequeue is called immediately before visiting a vertex.
	OnDequeue func(id string, depth int)

	// OnVisit is called when visiting a vertex. If it returns an error,
	// BFS aborts and propagates that error.
	OnVisit func(id string, depth int) error

	// MaxDepth, if > 0, stops exploring beyond this depth.
	// A value of 0 explicitly disables any depth limit.
	MaxDepth int

	// FilterNeighbor can skip edges by returning false.
	// Called for each edge curr→neighbor.
	FilterNeighbor func(curr, neighbor string) bool

	// internal error recorded during option parsing
	err error
}

// DefaultOptions returns a BFSOptions with sane defaults:
//   - Context.Background()
//   - no depth limit (MaxDepth == 0)
//   - no filtering (all neighbors allowed)
//   - no-op hooks (OnEnqueue, OnDequeue, OnVisit)
//   - error channel clear.
func DefaultOptions() BFSOptions {
	return BFSOptions{
		Ctx:            context.Background(),
		OnEnqueue:      func(string, int) {},
		OnDequeue:      func(string, int) {},
		OnVisit:        func(string, int) error { return nil },
		MaxDepth:       0,
		FilterNeighbor: func(_, _ string) bool { return true },
		err:            nil,
	}
}

// WithContext sets a custom context for cancellation.
func WithContext(ctx context.Context) Option {
	return func(o *BFSOptions) {
		if ctx != nil {
			o.Ctx = ctx
		}
	}
}

// WithOnEnqueue registers a callback to run on enqueue.
func WithOnEnqueue(fn func(id string, depth int)) Option {
	return func(o *BFSOptions) {
		if fn != nil {
			o.OnEnqueue = fn
		}
	}
}

// WithOnDequeue registers a callback to run on dequeue.
func WithOnDequeue(fn func(id string, depth int)) Option {
	return func(o *BFSOptions) {
		if fn != nil {
			o.OnDequeue = fn
		}
	}
}

// WithOnVisit registers a callback to run on visit; returning an error
// from this callback stops the BFS.
func WithOnVisit(fn func(id string, depth int) error) Option {
	return func(o *BFSOptions) {
		if fn != nil {
			o.OnVisit = fn
		}
	}
}

// WithMaxDepth stops the search at the given depth (exclusive).
//
//	d > 0: limit to depth d
//	d == 0: explicit no depth limit
//	d < 0: invalid option → ErrOptionViolation
func WithMaxDepth(d int) Option {
	return func(o *BFSOptions) {
		switch {
		case d < 0:
			o.err = fmt.Errorf("%w: MaxDepth cannot be negative (%d)", ErrOptionViolation, d)
		case d == 0:
			// explicit "no limit"
			o.MaxDepth = 0
		default:
			o.MaxDepth = d
		}
	}
}

// WithFilterNeighbor skips neighbors when fn returns false.
func WithFilterNeighbor(fn func(curr, neighbor string) bool) Option {
	return func(o *BFSOptions) {
		if fn != nil {
			o.FilterNeighbor = fn
		}
	}
}

// BFSResult holds the outcome of a BFS traversal:
//   - Order: vertices visited, in visit sequence.
//   - Depth: map from vertex ID to its distance (in edges) from the start.
//   - Parent: map from vertex ID to its predecessor in the BFS tree.
type BFSResult struct {
	Order  []string
	Depth  map[string]int
	Parent map[string]string
}

// PathTo reconstructs the path from the start vertex to dest.
// Returns an error if dest was not reached.
func (r *BFSResult) PathTo(dest string) ([]string, error) {
	if _, ok := r.Depth[dest]; !ok {
		return nil, fmt.Errorf("bfs: no path to %q", dest)
	}
	// build reversed path
	path := []string{}
	for cur := dest; ; {
		path = append(path, cur)
		prev, ok := r.Parent[cur]
		if !ok {
			break
		}
		cur = prev
	}
	// reverse to get start → dest
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	return path, nil
}
