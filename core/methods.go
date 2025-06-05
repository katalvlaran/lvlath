// Package core: high-performance Graph method implementations
//
// This file provides thread-safe, O(1) (amortized) operations for
// vertex and edge management on the Graph type defined in types.go.
// We leverage separate RWMutex locks for vertices (muVert) and
// edges+adjacency (muEdgeAdj) to minimize contention.
// Adjacency is stored as a nested map: adjacencyList[from][to][edgeID] = struct{}{},
// allowing constant-time existence, insertion, and deletion of edges.

package core

import (
	"fmt"
	"sort"
	"sync/atomic"
)

const (
	edgeIDPrefix = "e"
)

// AddVertex inserts a new vertex with the given ID into the Graph.
// Returns ErrEmptyVertexID if id is empty.
// If the vertex already exists, this is a no-op (idempotent).
// Complexity: O(1) amortized.
func (g *Graph) AddVertex(id string) error {
	// Validate input: empty IDs are not allowed
	if id == "" {
		return ErrEmptyVertexID // empty ID
	}
	// Acquire write lock on vertices only
	g.muVert.Lock()
	defer g.muVert.Unlock()

	// Check if vertex already present
	if _, exists := g.vertices[id]; exists {
		return nil // no-op for existing vertex
	}
	// Insert new Vertex struct with empty Metadata map
	g.vertices[id] = &Vertex{ID: id, Metadata: make(map[string]interface{})}

	// Initialize adjacencyList entry for this vertex (lazy map-of-maps)
	g.muEdgeAdj.Lock()
	g.ensureAdjID(id)
	g.muEdgeAdj.Unlock()

	return nil
}

// HasVertex reports whether a vertex with the given ID exists in the graph.
// Complexity: O(1).
func (g *Graph) HasVertex(id string) bool {
	if id == "" {
		return false // empty ID considered absent
	}
	// Acquire read lock on vertices
	g.muVert.RLock()
	defer g.muVert.RUnlock()
	_, exists := g.vertices[id]

	return exists
}

// RemoveVertex deletes the vertex and all incident edges from the graph.
// Returns ErrEmptyVertexID if id is empty, ErrVertexNotFound if vertex does not exist.
// Complexity: O(deg(v) + M) where deg(v) is total edges incident and M is unique neighbors.
func (g *Graph) RemoveVertex(id string) error {
	if id == "" {
		return ErrEmptyVertexID
	}
	// Lock vertices and edges+adjacency to prevent races
	g.muVert.Lock()
	defer g.muVert.Unlock()
	g.muEdgeAdj.Lock()
	defer g.muEdgeAdj.Unlock()

	// Verify vertex presence
	if _, exists := g.vertices[id]; !exists {
		return ErrVertexNotFound
	}
	// Remove all edges where id is either from or to
	for eid, e := range g.edges {
		if e.From == id || (e.To == id && !e.Directed) || (e.To == id && e.Directed) {
			removeEdgeFromAdj(g, eid, e)
			delete(g.edges, eid)
		}
	}

	// Remove vertex itself
	delete(g.vertices, id)
	// Cleanup empty adjacency entries
	errorCleanupAdjacency(g)

	return nil
}

// AddEdge creates a new edge with optional per-edge directed override(from 'from' to 'to' by default),
// and with the given weight and options, returns its unique Edge.ID.
// Handles parallel edges, loops, weights per configuration.
// For undirected (Directed=false), we mirror adjacency two ways,
// also enforces that per-edge directedness overrides (EdgeOption) are only allowed when the graph was constructed with WithMixedEdges().
//
// Returns ErrEmptyVertexID, ErrBadWeight, ErrLoopNotAllowed, ErrMultiEdgeNotAllowed, ErrMixedEdgesNotAllowed.
// Complexity: O(1).
func (g *Graph) AddEdge(from, to string, weight int64, opts ...EdgeOption) (string, error) {
	// 1) Input validation
	if from == "" || to == "" {
		return "", ErrEmptyVertexID
	}
	// 2) Weight constraint
	if !g.weighted && weight != 0 {
		return "", ErrBadWeight
	}
	// 3) Loop constraint
	if from == to && !g.allowLoops {
		return "", ErrLoopNotAllowed
	}
	// 4) If user passed any per-edge options but direction and mixed-mode is disabled - reject
	if len(opts) > 0 && !g.directed && !g.allowMixed {
		return "", ErrMixedEdgesNotAllowed
	}
	// 5) Ensure both endpoints exist (idempotent)
	if err := g.AddVertex(from); err != nil {
		return "", err
	}
	if err := g.AddVertex(to); err != nil {
		return "", err
	}

	// 6) Lock everything around edges & adjacency
	g.muEdgeAdj.Lock()
	defer g.muEdgeAdj.Unlock()

	// 7) Multi-edge existence check
	if !g.allowMulti {
		if inner, ok := g.adjacencyList[from][to]; ok && len(inner) > 0 {
			return "", ErrMultiEdgeNotAllowed
		}
	}

	// 8) Generate a new atomic Edge.ID
	eid := fmt.Sprintf("%s%d", edgeIDPrefix, atomic.AddUint64(&g.nextEdgeID, 1))

	// 9) Construct the Edge with the _global_ default directedness
	e := &Edge{ID: eid, From: from, To: to, Weight: weight, Directed: g.directed}
	// 10) Apply any per-edge overrides (only WithEdgeDirected exists today)
	for _, opt := range opts {
		opt(e)
	}
	// 11) Re-check loops in case WithEdgeDirected changed nothing here,
	//     but best to keep the guard in case future options interfere.
	if e.From == e.To && !g.allowLoops {
		return "", ErrLoopNotAllowed
	}

	// 12) Store in the global map
	g.edges[eid] = e

	// 13) Insert into nested adjacencyList[from][to][eid]
	g.ensureAdjMap(from, to)
	g.adjacencyList[from][to][eid] = struct{}{}

	// 14) If this edge is undirected, mirror it for the reverse adjacency
	//     (loops skip the mirror)
	if !e.Directed && from != to {
		g.ensureAdjMap(to, from)
		g.adjacencyList[to][from][eid] = struct{}{}
	}

	return eid, nil
}

// RemoveEdge deletes the edge with the given ID (and its mirror) from the graph,
// updating both global map and adjacency nested maps.
// Returns ErrEdgeNotFound if no such edge exists.
// Complexity: O(1).
func (g *Graph) RemoveEdge(eid string) error {
	// Lock edges+adjacency
	g.muEdgeAdj.Lock()
	defer g.muEdgeAdj.Unlock()
	// Fetch edge
	e, ok := g.edges[eid]
	if !ok {
		return ErrEdgeNotFound
	}
	delete(g.edges, eid)         // Delete from global edges map
	removeEdgeFromAdj(g, eid, e) // Remove from adjacencyList[from][to]
	errorCleanupAdjacency(g)     // Mirror removal for undirected

	return nil
}

// HasEdge reports true if at least one edge from 'from' to 'to' exists.
// Complexity: O(1).
func (g *Graph) HasEdge(from, to string) bool {
	if from == "" || to == "" {
		return false
	}
	g.muEdgeAdj.RLock()
	defer g.muEdgeAdj.RUnlock()
	// Check nested map existence and non-empty
	if inner, ok := g.adjacencyList[from][to]; ok && len(inner) > 0 {
		return true
	}

	return false
}

// Neighbors returns all edges incident to vertex 'id'.
// For directed edges, returns outgoing; for undirected, returns both directions.
// Result is a slice of *Edge pointers, sorted by Edge.ID for determinism.
// Complexity: O(d log d), where d is number of incident edges.
func (g *Graph) Neighbors(id string) ([]*Edge, error) {
	if id == "" {
		return nil, ErrEmptyVertexID
	}
	// Ensure vertex exists
	g.muVert.RLock()
	if _, ok := g.vertices[id]; !ok {
		g.muVert.RUnlock()
		return nil, ErrVertexNotFound
	}
	g.muVert.RUnlock()

	// Lock edges+adjacency for reading
	g.muEdgeAdj.RLock()
	defer g.muEdgeAdj.RUnlock()

	var out []*Edge
	// Iterate all "to" maps for this vertex
	for _, edgeSet := range g.adjacencyList[id] {
		for eid := range edgeSet {
			e := g.edges[eid]
			// For directed, include only if e.From == id
			if e.Directed && e.From != id {
				continue
			}
			// Append pointer directly: no copying
			out = append(out, e)
		}
	}
	// Sort by ID to ensure reproducible ordering
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })

	return out, nil
}

// NeighborIDs returns the IDs of all adjacent vertices to id,
// honoring directed, undirected, and per-edge overrides.
// Complexity: O(d log d)
func (g *Graph) NeighborIDs(id string) ([]string, error) {
	edges, err := g.Neighbors(id)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(edges))
	for _, e := range edges {
		if e.From == id {
			seen[e.To] = struct{}{}
		} else if !e.Directed && e.To == id {
			seen[e.From] = struct{}{}
		}
	}
	var ids []string
	for v := range seen {
		ids = append(ids, v)
	}
	sort.Strings(ids)

	return ids, nil
}

// Additional methods:
////////////////////

// AdjacencyList exposes a flattened map from vertex to edge IDs.
// Complexity: O(V + E)
func (g *Graph) AdjacencyList() map[string][]string {
	g.muEdgeAdj.RLock()
	defer g.muEdgeAdj.RUnlock()
	result := make(map[string][]string, len(g.adjacencyList))
	for from, toMap := range g.adjacencyList {
		for _, edgeMap := range toMap {
			for eid := range edgeMap {
				result[from] = append(result[from], eid)
			}
		}
	}

	return result
}

// InternalVertices exposes the live internal vertices map (read-only).
func (g *Graph) InternalVertices() map[string]*Vertex {
	return g.vertices
}

// Weighted reports whether the graph treats edge weights as meaningful.
func (g *Graph) Weighted() bool {
	return g.weighted
}

// Directed reports whether new edges default to directed.
func (g *Graph) Directed() bool {
	return g.directed
}

// Looped reports whether the graph's edges could be looped.
func (g *Graph) Looped() bool {
	return g.allowLoops
}

// VerticesMap returns a shallow copy of the vertex map.
// Complexity: O(V)
func (g *Graph) VerticesMap() map[string]*Vertex {
	g.muVert.RLock()
	defer g.muVert.RUnlock()
	out := make(map[string]*Vertex, len(g.vertices))
	for id, v := range g.vertices {
		out[id] = v
	}

	return out
}

// CloneEmpty returns a new Graph with identical configuration and vertices, but no edges.
// Complexity: O(V)
func (g *Graph) CloneEmpty() *Graph {
	g.muVert.RLock()
	defer g.muVert.RUnlock()
	g.muEdgeAdj.RLock()
	defer g.muEdgeAdj.RUnlock()
	// Copy configuration via options
	opts := []GraphOption{WithDirected(g.directed)}
	if g.weighted {
		opts = append(opts, WithWeighted())
	}
	if g.allowMulti {
		opts = append(opts, WithMultiEdges())
	}
	if g.allowLoops {
		opts = append(opts, WithLoops())
	}
	if g.allowMixed {
		opts = append(opts, WithMixedEdges())
	}
	clone := NewGraph(opts...)
	// Copy vertices
	for id, v := range g.vertices {
		clone.vertices[id] = &Vertex{ID: v.ID, Metadata: v.Metadata}
		clone.adjacencyList[id] = make(map[string]map[string]struct{})
	}

	return clone
}

// Clone returns a deep copy of the Graph: configuration, vertices, edges, and adjacency.
// Complexity: O(V + E)
func (g *Graph) Clone() *Graph {
	clone := g.CloneEmpty()
	g.muEdgeAdj.RLock()
	defer g.muEdgeAdj.RUnlock()
	// Copy edges and adjacency
	for eid, e := range g.edges {
		// Duplicate Edge struct
		ne := &Edge{ID: eid, From: e.From, To: e.To, Weight: e.Weight, Directed: e.Directed}
		clone.edges[eid] = ne
		// Append to nested adjacency maps
		if _, ok := clone.adjacencyList[e.From][e.To]; !ok {
			clone.adjacencyList[e.From][e.To] = make(map[string]struct{})
		}
		clone.adjacencyList[e.From][e.To][eid] = struct{}{}
		if !e.Directed && e.From != e.To {
			if _, ok := clone.adjacencyList[e.To][e.From]; !ok {
				clone.adjacencyList[e.To][e.From] = make(map[string]struct{})
			}
			clone.adjacencyList[e.To][e.From][eid] = struct{}{}
		}
	}

	return clone
}

// Edges returns all edges sorted by their ID.
// Complexity: O(E·logE)
func (g *Graph) Edges() []*Edge {
	g.muEdgeAdj.RLock()
	defer g.muEdgeAdj.RUnlock()
	out := make([]*Edge, 0, len(g.edges))
	for _, e := range g.edges {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })

	return out
}

// Vertices returns all vertex IDs in sorted order.
// Complexity: O(V·logV)
func (g *Graph) Vertices() []string {
	g.muVert.RLock()
	defer g.muVert.RUnlock()
	ids := make([]string, 0, len(g.vertices))
	for id := range g.vertices {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	return ids
}

// Degree returns (in, out, undirected) degrees of id.
func (g *Graph) Degree(id string) (in, out, undirected int, err error) {
	edges, err := g.Neighbors(id)
	if err != nil {
		return 0, 0, 0, err
	}
	for _, e := range edges {
		if e.From == id && e.To == id {
			undirected++ // self-loop
		} else if e.From == id {
			out++
			if !e.Directed {
				undirected++
			}
		} else {
			// undirected incoming
			out++
			undirected++
		}
	}

	return in, out, undirected, nil
}

// HasDirectedEdges reports whether there is at least one edge.Directed == true.
func (g *Graph) HasDirectedEdges() bool {
	g.muEdgeAdj.RLock()
	defer g.muEdgeAdj.RUnlock()
	for _, e := range g.edges {
		if e.Directed {
			return true
		}
	}

	return false
}

// EdgeCount returns total number of edges. O(1).
func (g *Graph) EdgeCount() int {
	g.muEdgeAdj.RLock()
	defer g.muEdgeAdj.RUnlock()

	return len(g.edges)
}

// VertexCount returns total number of vertices. O(1).
func (g *Graph) VertexCount() int {
	g.muVert.RLock()
	defer g.muVert.RUnlock()

	return len(g.vertices)
}

// Clear resets the graph to empty state (vertices, edges) but preserves flags.
func (g *Graph) Clear() {
	g.muVert.Lock()
	g.muEdgeAdj.Lock()
	// reset maps
	g.vertices = make(map[string]*Vertex)
	g.edges = make(map[string]*Edge)
	g.adjacencyList = make(map[string]map[string]map[string]struct{})
	atomic.StoreUint64(&g.nextEdgeID, 0)
	g.muEdgeAdj.Unlock()
	g.muVert.Unlock()
}

// FilterEdges removes all edges failing the predicate.
func (g *Graph) FilterEdges(pred func(*Edge) bool) {
	g.muEdgeAdj.Lock()
	defer g.muEdgeAdj.Unlock()
	for eid, e := range g.edges {
		if !pred(e) {
			removeEdgeFromAdj(g, eid, e)
			delete(g.edges, eid)
		}
	}
	errorCleanupAdjacency(g)
}

// Internal helper methods:
////////////////////

// ensureAdjID makes adjacencyList[id] non-nil.
func (g *Graph) ensureAdjID(id string) {
	if _, ok := g.adjacencyList[id]; !ok {
		// Create outer map for "from" key
		g.adjacencyList[id] = make(map[string]map[string]struct{})
	}
}

// ensureAdjMap ensures adjacencyList[from][to] initialized.
func (g *Graph) ensureAdjMap(from, to string) {
	g.ensureAdjID(from)
	if g.adjacencyList[from][to] == nil {
		g.adjacencyList[from][to] = make(map[string]struct{})
	}
}

// removeEdgeFromAdj deletes eid from both directions if needed.
func removeEdgeFromAdj(g *Graph, eid string, e *Edge) {
	// from -> to
	if m := g.adjacencyList[e.From][e.To]; m != nil {
		delete(m, eid)
		if len(m) == 0 {
			delete(g.adjacencyList[e.From], e.To)
		}
	}
	// mirror when undirected
	if !e.Directed && e.From != e.To {
		if m := g.adjacencyList[e.To][e.From]; m != nil {
			delete(m, eid)
			if len(m) == 0 {
				delete(g.adjacencyList[e.To], e.From)
			}
		}
	}
}

// errorCleanupAdjacency removes empty nested maps.
func errorCleanupAdjacency(g *Graph) {
	for u, m := range g.adjacencyList {
		for v, em := range m {
			if len(em) == 0 {
				delete(m, v)
			}
		}
		if len(m) == 0 {
			delete(g.adjacencyList, u)
		}
	}
}
