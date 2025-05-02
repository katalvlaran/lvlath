package core

// AddVertex inserts v into the graph if absent.
// Thread-safe.
func (g *Graph) AddVertex(v *Vertex) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, ok := g.vertices[v.ID]; ok {
		return
	}
	g.vertices[v.ID] = v
	g.adjacencyList[v.ID] = make(map[string][]*Edge)
}

// RemoveVertex deletes the vertex id and all its incident edges.
// Thread-safe.
func (g *Graph) RemoveVertex(id string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	delete(g.vertices, id)
	delete(g.adjacencyList, id)
	for _, nbrs := range g.adjacencyList {
		delete(nbrs, id)
	}
}

// HasVertex reports whether id exists in the graph.
// Thread-safe.
func (g *Graph) HasVertex(id string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	_, ok := g.vertices[id]
	return ok
}

// AddEdge creates an edge from → to with given weight.
// Auto-adds missing vertices. Mirror-inserts for undirected graphs.
// Thread-safe.
func (g *Graph) AddEdge(fromID, toID string, weight int64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// ensure vertices exist
	if _, ok := g.vertices[fromID]; !ok {
		g.vertices[fromID] = &Vertex{ID: fromID, Metadata: make(map[string]interface{})}
		g.adjacencyList[fromID] = make(map[string][]*Edge)
	}
	if _, ok := g.vertices[toID]; !ok {
		g.vertices[toID] = &Vertex{ID: toID, Metadata: make(map[string]interface{})}
		g.adjacencyList[toID] = make(map[string][]*Edge)
	}

	// add the directed edge
	edge := &Edge{From: g.vertices[fromID], To: g.vertices[toID], Weight: weight}
	g.adjacencyList[fromID][toID] = append(g.adjacencyList[fromID][toID], edge)

	// mirror for undirected
	if !g.directed {
		rev := &Edge{From: g.vertices[toID], To: g.vertices[fromID], Weight: weight}
		g.adjacencyList[toID][fromID] = append(g.adjacencyList[toID][fromID], rev)
	}
}

// RemoveEdge deletes **all** edges between fromID and toID.
// For undirected graphs, mirrors are also removed.
// Thread-safe.
func (g *Graph) RemoveEdge(fromID, toID string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	delete(g.adjacencyList[fromID], toID)
	if !g.directed {
		delete(g.adjacencyList[toID], fromID)
	}
}

// HasEdge reports whether at least one edge exists from → to.
// Thread-safe.
func (g *Graph) HasEdge(fromID, toID string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if nbrs, ok := g.adjacencyList[fromID]; ok {
		return len(nbrs[toID]) > 0
	}
	return false
}

// Neighbors returns each unique *Vertex reachable from id.
// Thread-safe.
func (g *Graph) Neighbors(id string) []*Vertex {
	g.mu.RLock()
	defer g.mu.RUnlock()

	nbrsMap, ok := g.adjacencyList[id]
	if !ok {
		return nil
	}
	out := make([]*Vertex, 0, len(nbrsMap))
	for nid := range nbrsMap {
		out = append(out, g.vertices[nid])
	}
	return out
}

// Vertices returns a slice of all vertices in the graph.
// Thread-safe.
func (g *Graph) Vertices() []*Vertex {
	g.mu.RLock()
	defer g.mu.RUnlock()

	out := make([]*Vertex, 0, len(g.vertices))
	for _, v := range g.vertices {
		out = append(out, v)
	}
	return out
}

// Edges returns a flat slice of all edges (may duplicate for undirected).
// Thread-safe.
func (g *Graph) Edges() []*Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var out []*Edge
	for _, nbrs := range g.adjacencyList {
		for _, es := range nbrs {
			out = append(out, es...)
		}
	}
	return out
}
