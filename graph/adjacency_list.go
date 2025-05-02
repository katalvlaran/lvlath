package graph

// NewGraph creates and returns a new Graph instance.
// directed=true for directed graphs, weighted=true to enable edge weights.
func NewGraph(directed, weighted bool) *Graph {
	return &Graph{
		directed:      directed,
		weighted:      weighted,
		vertices:      make(map[string]*Vertex),
		adjacencyList: make(map[string]map[string][]*Edge),
	}
}

// AddVertex adds a vertex to the graph. If a vertex with the same ID already exists,
// this method does nothing.
func (g *Graph) AddVertex(v *Vertex) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.vertices[v.ID]; exists {
		return
	}
	g.vertices[v.ID] = v
	g.adjacencyList[v.ID] = make(map[string][]*Edge)
}

// RemoveVertex removes the vertex with the given ID, along with all incident edges.
// If the vertex does not exist, this method does nothing.
func (g *Graph) RemoveVertex(id string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	delete(g.vertices, id)
	delete(g.adjacencyList, id)
	for _, nbrs := range g.adjacencyList {
		delete(nbrs, id)
	}
}

// HasVertex returns true if the graph contains a vertex with the given ID.
func (g *Graph) HasVertex(id string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	_, ok := g.vertices[id]
	return ok
}

// AddEdge creates an edge from fromID to toID with the specified weight.
// If either vertex does not exist, it will be auto-added.
// For undirected graphs, a mirror edge is also created.
func (g *Graph) AddEdge(fromID, toID string, weight int64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Auto-add vertices if missing
	if _, exists := g.vertices[fromID]; !exists {
		g.vertices[fromID] = &Vertex{ID: fromID, Metadata: make(map[string]interface{})}
		g.adjacencyList[fromID] = make(map[string][]*Edge)
	}
	if _, exists := g.vertices[toID]; !exists {
		g.vertices[toID] = &Vertex{ID: toID, Metadata: make(map[string]interface{})}
		g.adjacencyList[toID] = make(map[string][]*Edge)
	}

	// Add the edge
	edge := &Edge{From: g.vertices[fromID], To: g.vertices[toID], Weight: weight}
	g.adjacencyList[fromID][toID] = append(g.adjacencyList[fromID][toID], edge)

	// For undirected graphs, add a mirror edge
	if !g.directed {
		rev := &Edge{From: g.vertices[toID], To: g.vertices[fromID], Weight: weight}
		g.adjacencyList[toID][fromID] = append(g.adjacencyList[toID][fromID], rev)
	}
}

// RemoveEdge deletes all edges between fromID and toID.
// For undirected graphs, the mirror edges are also removed.
func (g *Graph) RemoveEdge(fromID, toID string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	delete(g.adjacencyList[fromID], toID)
	if !g.directed {
		delete(g.adjacencyList[toID], fromID)
	}
}

// HasEdge returns true if there is at least one edge from fromID to toID.
func (g *Graph) HasEdge(fromID, toID string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	nbrs, ok := g.adjacencyList[fromID]
	if !ok {
		return false
	}
	return len(nbrs[toID]) > 0
}

// Neighbors returns a slice of unique neighboring vertices for the given vertex ID.
func (g *Graph) Neighbors(id string) []*Vertex {
	g.mu.RLock()
	defer g.mu.RUnlock()

	nbrsMap, ok := g.adjacencyList[id]
	if !ok {
		return nil
	}
	out := make([]*Vertex, 0, len(nbrsMap))
	for neighborID := range nbrsMap {
		out = append(out, g.vertices[neighborID])
	}
	return out
}

// Vertices returns a slice of all vertices in the graph.
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
// Note: In undirected graphs, edges may appear twice (once per direction).
func (g *Graph) Edges() []*Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var out []*Edge
	for _, nbrs := range g.adjacencyList {
		for _, edges := range nbrs {
			out = append(out, edges...)
		}
	}
	return out
}

// Clone returns a deep copy of the graph, including its vertices and edges.
// Note: Metadata maps are copied by reference; implement deep copy if needed.
func (g *Graph) Clone() *Graph {
	g.mu.RLock()
	defer g.mu.RUnlock()

	copyG := NewGraph(g.directed, g.weighted)
	for id, v := range g.vertices {
		// Shallow copy of Metadata; deep copy can be implemented if needed.
		copyG.vertices[id] = &Vertex{ID: v.ID, Metadata: v.Metadata}
		copyG.adjacencyList[id] = make(map[string][]*Edge)
	}
	for fromID, nbrs := range g.adjacencyList {
		for toID, edges := range nbrs {
			for _, e := range edges {
				copyG.adjacencyList[fromID][toID] = append(copyG.adjacencyList[fromID][toID],
					&Edge{From: copyG.vertices[fromID], To: copyG.vertices[toID], Weight: e.Weight})
			}
		}
	}
	return copyG
}
