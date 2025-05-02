package graph

import "sync"

// Vertex represents a node within the graph.
// ID is a unique identifier for the vertex.
// Metadata can store arbitrary key-value information associated with the vertex.
type Vertex struct {
	ID       string
	Metadata map[string]interface{}
}

// Edge represents a connection between two vertices in the graph.
// From and To are pointers to the source and target vertices.
// Weight is an int64 value representing the edge's weight.
// Note: Weight is stored regardless of the Graph.weighted flag,
// but algorithms should check Graph.weighted to decide whether to use it.
type Edge struct {
	From   *Vertex
	To     *Vertex
	Weight int64
}

// Graph is the core data structure for representing a graph.
// It supports directed and undirected graphs, weighted edges, and multiedges.
// Internally, it uses an adjacency list representation.
// mu protects concurrent access to the graph's internal state.
type Graph struct {
	mu            sync.RWMutex
	directed      bool
	weighted      bool
	vertices      map[string]*Vertex
	adjacencyList map[string]map[string][]*Edge
}

// VerticesMap exposes the internal map of vertices for package use.
// It should be used carefully and read-only.
func (g *Graph) VerticesMap() map[string]*Vertex {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.InternalVertices()
}

// InternalVertices returns the vertices map. Only for internal use.
func (g *Graph) InternalVertices() map[string]*Vertex {
	return g.vertices
}
