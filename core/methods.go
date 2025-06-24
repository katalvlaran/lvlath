// Package core: high-performance, thread-safe Graph implementations.
//
// This file implements all Graph methods using two RWMutexes to minimize lock
// contention: muVert for vertex map, muEdgeAdj for edges+adjacency.
// Adjacency is a nested map: adjacencyList[from][to][edgeID] = struct{}{}
// providing O(1) existence, insertion, and deletion. Edge IDs are atomic
// counters (“e1”, “e2”, …). All iteration APIs return sorted results.

package core

import (
	"fmt"
	"sort"
	"sync/atomic"
)

const edgeIDPrefix = "e"

//–– Public API ––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––

// AddVertex inserts a vertex if missing.
// Steps:
//  1. Validate non-empty ID (ErrEmptyVertexID).
//  2. Lock muVert, check idempotent add.
//  3. Lazy-init adjacencyList[id] under muEdgeAdj.
//
// Complexity: O(1) amortized.
func (g *Graph) AddVertex(id string) error {
	// Validate input: empty IDs are not allowed
	if id == "" {
		return ErrEmptyVertexID
	}
	g.muVert.Lock()
	defer g.muVert.Unlock()

	// Check if vertex already present
	if _, exists := g.vertices[id]; exists {
		return nil // no-op for existing vertex
	}
	// Insert new Vertex struct with empty Metadata map
	g.vertices[id] = &Vertex{ID: id, Metadata: make(map[string]interface{})}

	// ensure top-level adjacency map exists
	g.muEdgeAdj.Lock()
	ensureAdjacency(g, id, id)
	g.muEdgeAdj.Unlock()

	return nil
}

// HasVertex returns true if the ID exists. Empty ID ⇒ false.
// Complexity: O(1).
func (g *Graph) HasVertex(id string) bool {
	if id == "" {
		return false
	}
	// Acquire read lock on vertices
	g.muVert.RLock()
	defer g.muVert.RUnlock()
	_, ok := g.vertices[id]

	return ok
}

// RemoveVertex deletes a vertex and all incident edges.
// Steps:
//  1. Validate non-empty.
//  2. Lock muVert+muEdgeAdj.
//  3. Iterate edges map once: if e.From==id or (undirected or directed) e.To==id, call removeAdjacency(e).
//  4. Delete from g.edges.
//  5. Delete from g.vertices, then prune empty adjacency with cleanupAdjacency().
//
// Complexity: O(E + V) worst-case.
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

	// Remove all incident edges
	for eid, e := range g.edges {
		if e.From == id || (!e.Directed && e.To == id) || (e.Directed && e.To == id) {
			removeAdjacency(g, e) // remove both directions appropriately
			delete(g.edges, eid)
		}
	}

	// Delete vertex itself
	delete(g.vertices, id)
	// prune any empty nested maps
	cleanupAdjacency(g)

	return nil
}

// AddEdge creates a new edge, optionally directed in a mixed graph.
// Steps:
//  1. Validate IDs, weight, loops.
//  2. If opts present without allowMixed ⇒ ErrMixedEdgesNotAllowed.
//  3. Ensure endpoints via AddVertex.
//  4. Lock muEdgeAdj, check multi-edge constraint.
//  5. Generate eid atomically.
//  6. Build Edge struct (global g.directed default), apply opts.
//  7. Store in g.edges.
//  8. ensureAdjacency(from,to); add.
//  9. If !e.Directed && from!=to ⇒ ensureAdjacency(to,from); add (mirror).
//
// Complexity: O(1) amortized.
func (g *Graph) AddEdge(from, to string, weight int64, opts ...EdgeOption) (string, error) {
	// 1) Input validation
	if from == "" || to == "" {
		return "", ErrEmptyVertexID
	}
	if !g.weighted && weight != 0 { // weight constraint
		return "", ErrBadWeight
	}
	if from == to && !g.allowLoops { // loop constraint
		return "", ErrLoopNotAllowed
	}
	// If user passed any per-edge options but direction and mixed-mode is disabled - reject
	if len(opts) > 0 && !g.directed && !g.allowMixed {
		return "", ErrMixedEdgesNotAllowed
	}

	// 2) Ensure vertices exist
	if err := g.AddVertex(from); err != nil {
		return "", err
	}
	if err := g.AddVertex(to); err != nil {
		return "", err
	}

	// 3) Insert edge under lock
	g.muEdgeAdj.Lock()
	defer g.muEdgeAdj.Unlock()

	if !g.allowMulti { // Multi-edge existence check
		if inner := g.adjacencyList[from][to]; len(inner) > 0 {
			return "", ErrMultiEdgeNotAllowed
		}
	}

	// 4) Generate and apply overrides
	eid := fmt.Sprintf("%s%d", edgeIDPrefix, atomic.AddUint64(&g.nextEdgeID, 1))

	// Construct the Edge with the _global_ default directedness
	e := &Edge{ID: eid, From: from, To: to, Weight: weight, Directed: g.directed}
	// Apply any per-edge overrides (only WithEdgeDirected exists today)
	for _, opt := range opts {
		opt(e)
	}
	// Re-check loops in case WithEdgeDirected changed nothing here, but best to keep the guard in case future options interfere.
	if e.From == e.To && !g.allowLoops {
		return "", ErrLoopNotAllowed
	}

	// 5) Store and link adjacency
	g.edges[eid] = e
	ensureAdjacency(g, from, to)
	g.adjacencyList[from][to][eid] = struct{}{}

	// 6) Mirror undirected
	if !e.Directed && from != to {
		ensureAdjacency(g, to, from)
		g.adjacencyList[to][from][eid] = struct{}{}
	}

	return eid, nil
}

// RemoveEdge deletes one edge and its mirror.
// Steps:
//  1. Lock muEdgeAdj.
//  2. Lookup e, ErrEdgeNotFound if missing.
//  3. delete(g.edges, eid), removeAdjacency(e), cleanupAdjacency().
//
// Complexity: O(1) + O(V+E) on cleanup.
func (g *Graph) RemoveEdge(eid string) error {
	// Lock edges+adjacency
	g.muEdgeAdj.Lock()
	defer g.muEdgeAdj.Unlock()
	// Fetch edge
	e, ok := g.edges[eid]
	if !ok {
		return ErrEdgeNotFound
	}
	delete(g.edges, eid)  // Delete from global edges map
	removeAdjacency(g, e) // Remove from adjacencyList[from][to]
	cleanupAdjacency(g)   // Mirror removal for undirected

	return nil
}

// HasEdge checks for any edge from→to in O(1).
// Mirrors inserted in AddEdge so this works for undirected too.
func (g *Graph) HasEdge(from, to string) bool {
	if from == "" || to == "" {
		return false
	}
	g.muEdgeAdj.RLock()
	defer g.muEdgeAdj.RUnlock()

	return len(g.adjacencyList[from][to]) > 0
}

// Neighbors lists *all* edges touching id.
//   - Directed edges: only those with e.From==id.
//   - Undirected edges: both directions, but loop appears once.
//
// Sorted by Edge.ID.
// Complexity: O(d log d).
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

// NeighborIDs returns unique, sorted vertex IDs adjacent to id.
//
//	e.From==id ⇒ include e.To.
//	e.To==id && !e.Directed ⇒ include e.From.
//
// Complexity: O(d log d).
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
	ids := make([]string, 0, len(seen))
	for v := range seen {
		ids = append(ids, v)
	}
	sort.Strings(ids)

	return ids, nil
}

//–– Helpers ––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––

func ensureAdjacency(g *Graph, from, to string) {
	if g.adjacencyList[from] == nil {
		g.adjacencyList[from] = make(map[string]map[string]struct{})
	}
	if g.adjacencyList[from][to] == nil {
		g.adjacencyList[from][to] = make(map[string]struct{})
	}
}

// removeAdjacency deletes eid from both directions:
//   - from→to, and if undirected (e.Directed==false && from!=to), also to→from.
func removeAdjacency(g *Graph, e *Edge) {
	if m := g.adjacencyList[e.From][e.To]; m != nil {
		delete(m, e.ID)
		if len(m) == 0 {
			delete(g.adjacencyList[e.From], e.To)
		}
	}
	if !e.Directed && e.From != e.To {
		if m := g.adjacencyList[e.To][e.From]; m != nil {
			delete(m, e.ID)
			if len(m) == 0 {
				delete(g.adjacencyList[e.To], e.From)
			}
		}
	}
}

// cleanupAdjacency prunes any empty nested maps so HasEdge stays correct.
func cleanupAdjacency(g *Graph) {
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
			removeAdjacency(g, e)
			delete(g.edges, eid)
		}
	}
	cleanupAdjacency(g)
}
