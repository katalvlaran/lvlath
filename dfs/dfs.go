// Package dfs implements depth‑first search (single‑source and forest) on core.Graph.
// It supports directed, undirected, and per‑edge mixed‑direction edges, cancellation,
// pre‑ and post‑order hooks, depth and neighbor limits, full‑graph traversal, and diagnostics.
//
// Key features:
//   - DFS(g, startID, opts...): traverse from a root or full forest via WithFullTraversal
//   - Mixed‑edge: honors Edge.Directed when core.WithMixedEdges is enabled
//   - Hooks: OnVisit (pre‑order) & OnExit (post‑order) with error aborts
//   - Limits: MaxDepth, FilterNeighbor, SkippedNeighbors diagnostic count
//   - Cancellation via context.Context
//
// Complexity:
//
//   - Time:   O(V + E) for traversal (where V = vertices, E = edges), plus overhead of hooks and filters.
//   - Memory: O(V) for recursion stack and metadata maps.
//
// Options:
//
//   - WithContext(ctx)          allows cancellation via context.Context.
//   - WithOnVisit(fn)           pre-order hook on vertex discovery; error aborts traversal.
//   - WithOnExit(fn)            post-order hook after exploring descendants, before recording.
//   - WithMaxDepth(limit)       stops recursion beyond given depth (>=0).
//   - WithFilterNeighbor(fn)    filters neighbor IDs; return false to skip.
//
// Errors:
//
//   - ErrGraphNil               if g is nil.
//   - ErrStartVertexNotFound    if startID is missing.
//   - context.Canceled          if ctx is done.
//   - any error returned by OnVisit or OnExit.
package dfs

import (
	"fmt"

	"github.com/katalvlaran/lvlath/core"
)

// dfsWalker encapsulates state during DFS.
type dfsWalker struct {
	graph *core.Graph // underlying graph
	opts  DFSOptions  // traversal options
	res   *DFSResult  // result collector
}

// DFS performs depth‑first search on graph g. If opts include WithFullTraversal,
// it covers all disconnected components; otherwise, it starts only from startID.
// Returns DFSResult or error if aborted by context or hook.
func DFS(g *core.Graph, startID string, opts ...Option) (*DFSResult, error) {
	// 1. Validate input graph
	if g == nil {
		return nil, ErrGraphNil
	}

	// 2. Apply options
	dopts := DefaultOptions()
	var fn Option
	for _, fn = range opts {
		fn(&dopts)
	}

	// 3. Single‑source mode: verify startID
	if !dopts.FullTraversal && !g.HasVertex(startID) {
		return nil, ErrStartVertexNotFound
	}

	// 4. Initialize result with capacity hint
	vertices := g.Vertices()
	res := &DFSResult{
		Order:   make([]string, 0, len(vertices)),
		Depth:   make(map[string]int, len(vertices)),
		Parent:  make(map[string]string, len(vertices)),
		Visited: make(map[string]bool, len(vertices)),
	}

	walker := &dfsWalker{graph: g, opts: dopts, res: res}

	// 5. Traverse: forest or single tree
	if dopts.FullTraversal {
		for _, v := range vertices {
			if !res.Visited[v] {
				if err := walker.traverse(v, 0); err != nil {
					return res, err
				}
			}
		}
	} else {
		if err := walker.traverse(startID, 0); err != nil {
			return res, err
		}
	}

	// 6. Expose diagnostics
	res.SkippedNeighbors = walker.opts.SkippedNeighbors

	return res, nil
}

// traverse visits vertex id at given depth, recursing to neighbors.
// It honors context cancellation, depth limit, hooks, filtering, and mixed‑edge rules.
func (w *dfsWalker) traverse(id string, depth int) error {
	// 1. Cancellation check
	select {
	case <-w.opts.Ctx.Done():
		return w.opts.Ctx.Err()
	default:
	}

	// 2. Depth limit: stop if exceeded
	if w.opts.MaxDepth >= 0 && depth > w.opts.MaxDepth {
		return nil
	}

	// 3. Mark visited and record depth
	w.res.Visited[id] = true
	w.res.Depth[id] = depth

	// 4. Pre‑order hook
	if w.opts.OnVisit != nil {
		if err := w.opts.OnVisit(id); err != nil {
			// abort and clear post‑order
			w.res.Order = nil

			return fmt.Errorf("dfs: OnVisit hook for %q: %w", id, err)
		}
	}

	// 5. Fetch neighbors once
	nbs, err := w.graph.Neighbors(id)
	if err != nil {
		w.res.Order = nil

		return fmt.Errorf("dfs: Neighbors(%q): %w", id, err)
	}

	// 6. Explore each neighbor
	var e *core.Edge
	var nid string
	for _, e = range nbs {
		nid = e.To

		// Skip reverse edges in mixed/undirected
		if !e.Directed && !w.graph.Directed() && nid == id {
			continue
		}

		// Skip self‑loops if disallowed
		if nid == id && !w.graph.Looped() {
			continue
		}

		// Neighbor filtering
		if w.opts.FilterNeighbor != nil && !w.opts.FilterNeighbor(nid) {
			w.opts.SkippedNeighbors++
			continue
		}

		// Recurse on unvisited
		if !w.res.Visited[nid] {
			w.res.Parent[nid] = id
			if err = w.traverse(nid, depth+1); err != nil {
				return err
			}
		}
	}

	// 7. Post‑order hook
	if w.opts.OnExit != nil {
		if err = w.opts.OnExit(id); err != nil {
			w.res.Order = nil

			return fmt.Errorf("dfs: OnExit hook for %q: %w", id, err)
		}
	}

	// 8. Record finish order
	w.res.Order = append(w.res.Order, id)

	return nil
}
