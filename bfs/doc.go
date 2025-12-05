// Package bfs provides a production-grade breadth-first search over a core.Graph,
// returning unweighted shortest-path distances, parent links, and visit order.
//
// What
//
//   - Explore vertices in non-decreasing distance (edge count) from a start vertex.
//   - Returns a BFSResult containing:
//   - Order: visit sequence
//   - Depth: map from vertex → distance (edges) from start
//   - Parent: map from vertex → its predecessor in the BFS tree
//   - Supports functional hooks at three stages:
//   - OnEnqueue (before a vertex is enqueued)
//   - OnDequeue (immediately before visiting)
//   - OnVisit   (when visiting; may abort with an error)
//   - Allows filtering of individual neighbor edges via WithFilterNeighbor.
//   - Honors MaxDepth limit (d>0) or explicit “no limit” (d==0).
//   - Respects directed, undirected, and mixed-direction graphs.
//
// Why
//
//   - Compute unweighted shortest paths in O(V + E) time.
//   - Discover reachable subgraphs, connected components, and level layering.
//   - Foundation for flow, matching, reachability, and other graph algorithms.
//
// Determinism
//
//	Because core.Neighbors returns edges sorted by Edge.ID, and BFS enqueues
//	neighbors in that order, the visit sequence is fully reproducible.
//
// Mixed-Edges Support
//
//	When core.WithMixedEdges is enabled on your Graph, individual edges may be
//	marked Directed or undirected. BFS will:
//	  - Follow directed edges only in their proper direction (edge.From→edge.To).
//	  - For undirected edges, treat them bidirectionally.
//	Use WithFilterNeighbor to prune specific directions or relationships.
//
// Complexity (V = |Vertices|, E = |Edges|)
//
//   - Time:   O(V + E)   (each vertex and edge seen at most once)
//   - Memory: O(V)       (for queue, Depth map, Parent map, visited set)
//
// Usage
//
//		// Basic BFS with no options:
//		result, err := bfs.BFS(g, "start")
//		if err != nil {
//	      // handle one of:
//	      // ErrGraphNil, ErrStartVertexNotFound, ErrWeightedGraph, ErrOptionViolation, ErrNeighbors, or hook errors
//		}
//
//		// With functional options:
//		result, err := bfs.BFS(
//		    g, "start",
//		    bfs.WithContext(ctx),
//		    bfs.WithMaxDepth(3),
//		    bfs.WithFilterNeighbor(func(curr, nbr string) bool { return curr != "skip" }),
//		    bfs.WithOnEnqueue(func(id string, depth int) { /* ... */ }),
//		    bfs.WithOnDequeue(func(id string, depth int) { /* ... */ }),
//		    bfs.WithOnVisit(func(id string, depth int) error { /* ... */ return nil }),
//		)
//
// Options
//
//   - DefaultOptions(): background Context, no-op hooks, no depth limit, no filtering.
//   - WithContext(ctx):            set a custom context for cancellation.
//   - WithMaxDepth(d):             stop exploring beyond depth d (>0).
//   - WithFilterNeighbor(fn):      skip edges for which fn(curr,neighbor)==false.
//   - WithOnEnqueue(fn):           hook before a vertex is enqueued.
//   - WithOnDequeue(fn):           hook immediately before visiting a vertex.
//   - WithOnVisit(fn):             hook during visit; returning error aborts BFS.
//
// Errors
//
//   - ErrGraphNil             if the graph pointer is nil.
//   - ErrStartVertexNotFound  if the start vertex does not exist.
//   - ErrWeightedGraph        if run on a weighted graph.
//   - ErrOptionViolation      if invalid Option (e.g. negative MaxDepth).
//   - ErrNeighbors            if core.Neighbors fails for any vertex.
//   - Wrapped user-supplied hook errors from OnVisit.
//
// See: docs/BFS.md for detailed tutorial, pseudocode, and diagrams.
package bfs
