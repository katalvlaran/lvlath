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

// UnweightedView returns a new Graph with identical topology but with all edge
// weights set to zero and the weighted flag turned off. The input graph is not
// mutated. Edge IDs and directedness are preserved.
//
// Complexity: O(V + E). Concurrency: read locks only on source.
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
	var eid string
	var e, ne *Edge
	for eid, e = range g.edges {
		ne = &Edge{ID: eid, From: e.From, To: e.To, Weight: 0, Directed: e.Directed}
		out.edges[eid] = ne
		ensureAdjacency(out, ne.From, ne.To)
		out.adjacencyList[ne.From][ne.To][eid] = struct{}{}
		if !ne.Directed && ne.From != ne.To {
			ensureAdjacency(out, ne.To, ne.From)
			out.adjacencyList[ne.To][ne.From][eid] = struct{}{}
		}
	}
	g.muEdgeAdj.RUnlock()

	return out
}

// InducedSubgraph returns a new Graph induced by the set "keep" of vertex IDs:
// the result contains only vertices v where keep[v] is true, and all edges whose
// endpoints are both in keep. The input graph is not mutated.
//
// Complexity: O(V + E). Concurrency: read locks only on source.
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

	return out
}
