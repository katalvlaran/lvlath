// File: view.go
// Role: Non-mutating graph views (cloning topology with altered properties).
// Determinism:
//   - Preserves vertex/edge IDs and directedness. No reordering guarantees beyond core rules.
// Concurrency:
//   - Read locks on source; result is a fresh graph instance.
// AI-HINT (file):
//   - Views do NOT mutate the input Graph.
//   - UnweightedView returns Weighted()==false and sets all edge weights to 0.
//   - InducedSubgraph keeps only vertices in 'keep' and edges with both endpoints kept.

package core

import "sync/atomic"

// viewEdgeWeightZero is the canonical weight value used by views that enforce unweighted semantics.
const viewEdgeWeightZero float64 = 0

// UnweightedView creates a non-mutating view of the input graph where weights are forced to 0
// and the resulting graph reports Weighted()==false, while preserving topology and IDs.
//
// Implementation:
//   - Stage 1: Acquire GLOBAL READ LOCK (Vert + EdgeAdj) to ensure atomic snapshot.
//   - Stage 2: Copy vertices (shallow Metadata).
//   - Stage 3: Copy edges (forcing Weight=0).
//   - Stage 4: Release locks.
//
// Concurrency snapshot:
//   - Atomic: The view represents a consistent state at the moment of creation.
//   - No "gap" between vertex and edge copying.
//
// Behavior highlights:
//   - Does not mutate the source graph.
//   - Preserves Edge.ID and Directed for every edge; only Weight is changed.
//   - Preserves determinism rules of the core package (ordering is defined by public APIs).
//
// Inputs:
//   - g: source graph (must be non-nil by caller convention).
//
// Returns:
//   - *Graph: a new graph instance with identical topology and Weight forced to zero.
//
// Errors:
//   - None.
//
// Determinism:
//   - Deterministic for a fixed source snapshot; uses read locks for stable catalogs.
//
// Complexity:
//   - Time O(V+E), Space O(V+E) for the copied catalogs.
//
// Notes:
//   - This is a *view* implemented as a copy: the returned graph is independent and mutable.
//   - Vertex.Metadata is shallow-copied (pointer copy); if deep-copy is required, callers must do it externally.
//
// AI-Hints:
//   - Use UnweightedView when an algorithm requires zero weights but you must preserve original weights elsewhere.
func UnweightedView(g *Graph) *Graph {
	// LOCK ORDER: muVert -> muEdgeAdj.
	// Atomic snapshot requirement: we cannot allow topology changes during copy.
	g.muVert.RLock()
	defer g.muVert.RUnlock()
	g.muEdgeAdj.RLock()
	defer g.muEdgeAdj.RUnlock()

	// Build a graph with same directedness/mode but unweighted.
	opts := []GraphOption{WithDirected(g.Directed())}
	if g.Multigraph() {
		opts = append(opts, WithMultiEdges())
	}
	if g.Looped() {
		opts = append(opts, WithLoops())
	}
	if g.MixedEdges() {
		opts = append(opts, WithMixedEdges())
	}
	out := NewGraph(opts...)

	// Copy vertices
	// ALIASING: Metadata is shared.
	for id, v := range g.vertices {
		out.vertices[id] = &Vertex{ID: v.ID, Metadata: v.Metadata}
		out.adjacencyList[id] = make(map[string]map[string]struct{})
	}

	// Snapshot the edge ID counter under the same lock as the edge catalog snapshot.
	// This ensures the view continues generating IDs strictly after the last ID used by 'g'.
	srcNextEdgeID := atomic.LoadUint64(&g.nextEdgeID)
	var eid string
	var e, ne *Edge
	for eid, e = range g.edges {
		// Force weight to zero regardless of the source weight; directedness and IDs are preserved.
		ne = &Edge{ID: eid, From: e.From, To: e.To, Weight: viewEdgeWeightZero, Directed: e.Directed}
		out.edges[eid] = ne
		ensureAdjacency(out, ne.From, ne.To)
		out.adjacencyList[ne.From][ne.To][eid] = struct{}{}

		if !ne.Directed && ne.From != ne.To {
			ensureAdjacency(out, ne.To, ne.From)
			out.adjacencyList[ne.To][ne.From][eid] = struct{}{}
		}
	}

	// Carry over the edge ID counter so future AddEdge() calls cannot collide with copied IDs.
	atomic.StoreUint64(&out.nextEdgeID, srcNextEdgeID)

	return out
}

// InducedSubgraph creates a non-mutating induced subgraph containing only vertices selected by keep.
//
// Implementation:
//   - Stage 1: Acquire GLOBAL READ LOCK (Vert + EdgeAdj).
//   - Stage 2: Filter and copy vertices.
//   - Stage 3: Filter and copy edges (only if both endpoints exist in 'keep').
//
// Concurrency snapshot:
//   - Atomic: Prevents "phantom edges" where an endpoint might be deleted concurrently.
//
// Behavior highlights:
//   - Does not mutate the source graph.
//   - Preserves Edge.ID, Directed, and Weight for retained edges.
//   - Drops all edges that cross the cut (one endpoint not kept).
//
// Inputs:
//   - g: source graph (must be non-nil by caller convention).
//   - keep: map of vertex IDs to retain; keep[id]==true means "retain id".
//
// Returns:
//   - *Graph: a new graph containing only the induced topology.
//
// Errors:
//   - None.
//
// Determinism:
//   - Deterministic for a fixed source snapshot; uses read locks for stable catalogs.
//
// Complexity:
//   - Time O(V+E), Space O(V'+E') where V'/E' are retained sizes.
//
// Notes:
//   - The order of iteration over keep is irrelevant; retention is membership-based.
//   - Vertex.Metadata is shallow-copied (pointer copy).
//
// AI-Hints:
//   - Use InducedSubgraph to focus algorithms on a region of interest without changing the original graph.
func InducedSubgraph(g *Graph, keep map[string]bool) *Graph {
	// LOCK ORDER: muVert -> muEdgeAdj.
	g.muVert.RLock()
	defer g.muVert.RUnlock()
	g.muEdgeAdj.RLock()
	defer g.muEdgeAdj.RUnlock()

	// Reuse the same configuration as g (including weighted flag).
	opts := []GraphOption{WithDirected(g.Directed())}
	if g.Weighted() {
		opts = append(opts, WithWeighted())
	}
	if g.Multigraph() {
		opts = append(opts, WithMultiEdges())
	}
	if g.Looped() {
		opts = append(opts, WithLoops())
	}
	if g.MixedEdges() {
		opts = append(opts, WithMixedEdges())
	}
	out := NewGraph(opts...)

	// Copy kept vertices
	for id, v := range g.vertices {
		if keep[id] {
			out.vertices[id] = &Vertex{ID: v.ID, Metadata: v.Metadata}
			out.adjacencyList[id] = make(map[string]map[string]struct{})
		}
	}

	// Copy valid edges
	srcNextEdgeID := atomic.LoadUint64(&g.nextEdgeID)
	var eid string
	var e, ne *Edge
	var ok bool

	for eid, e = range g.edges {
		// Filter: both endpoints must be in the keep set.
		if !keep[e.From] || !keep[e.To] {
			continue
		}

		ne = &Edge{ID: eid, From: e.From, To: e.To, Weight: e.Weight, Directed: e.Directed}
		out.edges[eid] = ne
		ensureAdjacency(out, ne.From, ne.To)
		out.adjacencyList[ne.From][ne.To][eid] = struct{}{}

		if !ne.Directed && ne.From != ne.To {
			if _, ok = out.adjacencyList[ne.To][ne.From]; !ok {
				out.adjacencyList[ne.To][ne.From] = make(map[string]struct{})
			}
			out.adjacencyList[ne.To][ne.From][eid] = struct{}{}
		}
	}

	// Carry over the edge ID counter so future AddEdge() calls cannot collide with copied IDs.
	atomic.StoreUint64(&out.nextEdgeID, srcNextEdgeID)

	return out
}
