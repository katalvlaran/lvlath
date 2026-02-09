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
// It is a named constant to make "forced zero weight" intentional and grep-friendly.
const viewEdgeWeightZero float64 = 0

// UnweightedView creates a non-mutating view of the input graph where weights are forced to 0
// and the resulting graph reports Weighted()==false, while preserving topology and IDs.
//
// Implementation:
//   - Stage 1: Create a new Graph that matches g's orientation/multi/loop/mixed policies,
//     but intentionally does NOT enable WithWeighted().
//   - Stage 2: Snapshot and copy vertices under muVert.RLock (shallow metadata pointer copy).
//   - Stage 3: Snapshot and copy edges under muEdgeAdj.RLock, forcing Weight=0 for every edge.
//   - Stage 4: Carry over nextEdgeID to ensure future AddEdge() IDs cannot collide with copied IDs.
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
	// AI-HINT: Useful when algorithms require zero weights without touching original graph.

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
	g.muVert.RLock()
	for id, v := range g.vertices {
		out.vertices[id] = &Vertex{ID: v.ID, Metadata: v.Metadata}
		out.adjacencyList[id] = make(map[string]map[string]struct{})
	}
	g.muVert.RUnlock()

	// Copy edges with zero weight, preserving IDs and directedness.
	g.muEdgeAdj.RLock()
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
	g.muEdgeAdj.RUnlock()

	// Carry over the edge ID counter so future AddEdge() calls cannot collide with copied IDs.
	atomic.StoreUint64(&out.nextEdgeID, srcNextEdgeID)

	return out
}

// InducedSubgraph creates a non-mutating induced subgraph containing only vertices selected by keep,
// and only edges whose endpoints are both selected.
//
// Implementation:
//   - Stage 1: Create a new Graph that matches g's configuration flags.
//   - Stage 2: Snapshot and copy only kept vertices under muVert.RLock.
//   - Stage 3: Snapshot and copy only edges with both endpoints kept under muEdgeAdj.RLock.
//   - Stage 4: Carry over nextEdgeID to preserve future auto-ID uniqueness.
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
	// AI-HINT: Build problem-specific slices of the graph without side effects on 'g'.

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

	// Copy only kept vertices.
	g.muVert.RLock()
	var id string
	var v *Vertex
	for id, v = range g.vertices {
		if keep[id] {
			out.vertices[id] = &Vertex{ID: v.ID, Metadata: v.Metadata}
			out.adjacencyList[id] = make(map[string]map[string]struct{})
		}
	}
	g.muVert.RUnlock()

	// Copy only edges whose endpoints are both kept; preserve ID and directedness.
	g.muEdgeAdj.RLock()
	// Snapshot the edge ID counter under the same lock as the edge catalog snapshot.
	// Even if the induced subgraph filters out some edges, carrying the counter forward
	// prevents reusing historical IDs and keeps monotonicity aligned with the source graph.
	srcNextEdgeID := atomic.LoadUint64(&g.nextEdgeID)
	var eid string
	var e, ne *Edge
	for eid, e = range g.edges {
		if !keep[e.From] || !keep[e.To] {
			continue
		}
		ne = &Edge{ID: eid, From: e.From, To: e.To, Weight: e.Weight, Directed: e.Directed}
		out.edges[eid] = ne
		ensureAdjacency(out, ne.From, ne.To)
		out.adjacencyList[ne.From][ne.To][eid] = struct{}{}
		if !ne.Directed && ne.From != ne.To {
			ensureAdjacency(out, ne.To, ne.From)
			out.adjacencyList[ne.To][ne.From][eid] = struct{}{}
		}
	}
	g.muEdgeAdj.RUnlock()

	// Carry over the edge ID counter so future AddEdge() calls cannot collide with copied IDs.
	atomic.StoreUint64(&out.nextEdgeID, srcNextEdgeID)

	return out
}
