// Package core provides the fundamental in-memory Graph implementation.
//
// It offers thread-safe methods to mutate and query vertices and edges.
// All mutations acquire a write lock; queries acquire a read lock.
package core

// AddVertex inserts v into the graph if absent.
// If a vertex with the same ID already exists, this is a no-op.
// Thread-safe: acquires a write lock.
//
// Complexity: O(1)
func (g *Graph) AddVertex(v *Vertex) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.vertices[v.ID]; exists {
		return
	}
	g.vertices[v.ID] = v
	g.adjacencyList[v.ID] = make(map[string][]*Edge)
}

// RemoveVertex deletes the vertex with the given ID and all its incident edges.
// If the vertex is not present, this is a no-op.
// Thread-safe: acquires a write lock.
//
// Complexity: O(V + E) in the worst case, where V is number of vertices
// and E is number of edges adjacent to id.
func (g *Graph) RemoveVertex(id string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Remove the vertex itself
	delete(g.vertices, id)
	delete(g.adjacencyList, id)

	// Remove any edges pointing to id
	for fromID, nbrs := range g.adjacencyList {
		delete(nbrs, id)
		g.adjacencyList[fromID] = nbrs
	}
}

// HasVertex reports whether the graph contains a vertex with the given ID.
// Thread-safe: acquires a read lock.
//
// Complexity: O(1)
func (g *Graph) HasVertex(id string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	_, ok := g.vertices[id]
	return ok
}

// AddEdge creates an edge from fromID to toID with the specified weight.
// If either vertex does not exist, it is auto-added.
// For undirected graphs, also inserts the mirror edge.
// Thread-safe: acquires a write lock.
//
// Complexity: O(1) amortized per edge insertion.
func (g *Graph) AddEdge(fromID, toID string, weight int64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Ensure both vertices exist
	if _, ok := g.vertices[fromID]; !ok {
		g.vertices[fromID] = &Vertex{ID: fromID, Metadata: make(map[string]interface{})}
		g.adjacencyList[fromID] = make(map[string][]*Edge)
	}
	if _, ok := g.vertices[toID]; !ok {
		g.vertices[toID] = &Vertex{ID: toID, Metadata: make(map[string]interface{})}
		g.adjacencyList[toID] = make(map[string][]*Edge)
	}

	// Add directed edge
	edge := &Edge{From: g.vertices[fromID], To: g.vertices[toID], Weight: weight}
	g.adjacencyList[fromID][toID] = append(g.adjacencyList[fromID][toID], edge)

	// Mirror for undirected graphs
	if !g.directed {
		rev := &Edge{From: g.vertices[toID], To: g.vertices[fromID], Weight: weight}
		g.adjacencyList[toID][fromID] = append(g.adjacencyList[toID][fromID], rev)
	}
}

// RemoveEdge deletes all edges from fromID to toID.
// For undirected graphs, also removes the mirror edges.
// If no such edges exist, this is a no-op.
// Thread-safe: acquires a write lock.
//
// Complexity: O(1)
func (g *Graph) RemoveEdge(fromID, toID string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	delete(g.adjacencyList[fromID], toID)
	if !g.directed {
		delete(g.adjacencyList[toID], fromID)
	}
}

// HasEdge reports whether at least one edge exists from fromID to toID.
// Thread-safe: acquires a read lock.
//
// Complexity: O(1)
func (g *Graph) HasEdge(fromID, toID string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if nbrs, ok := g.adjacencyList[fromID]; ok {
		return len(nbrs[toID]) > 0
	}

	return false
}

// Neighbors returns each unique vertex reachable from the given ID.
// If the vertex does not exist, returns nil.
// Thread-safe: acquires a read lock.
//
// Complexity: O(d) where d is the out-degree of id.
func (g *Graph) Neighbors(id string) []*Vertex {
	g.mu.RLock()
	defer g.mu.RUnlock()

	nbrsMap, exists := g.adjacencyList[id]
	if !exists {
		return nil
	}
	out := make([]*Vertex, 0, len(nbrsMap))
	for nbrID := range nbrsMap {
		out = append(out, g.vertices[nbrID])
	}

	return out
}

// Vertices returns a slice of all vertices in the graph.
// Thread-safe: acquires a read lock.
//
// Complexity: O(V)
func (g *Graph) Vertices() []*Vertex {
	g.mu.RLock()
	defer g.mu.RUnlock()

	out := make([]*Vertex, 0, len(g.vertices))
	for _, v := range g.vertices {
		out = append(out, v)
	}

	return out
}

// Edges returns a flat slice of all edges in the graph.
// In undirected graphs, each edge appears twice (once per direction).
// Thread-safe: acquires a read lock.
//
// Complexity: O(E)
func (g *Graph) Edges() []*Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var out []*Edge
	for _, nbrs := range g.adjacencyList {
		for _, edgeList := range nbrs {
			out = append(out, edgeList...)
		}
	}

	return out
}
