// Package core defines the central Graph, Vertex and Edge types,
// and provides thread-safe primitives for building and cloning graphs.
//
// All core APIs use sync.RWMutex internally—so you can safely mutate your
// graphs across goroutines without extra locking.
package core

import "sync"

// Vertex represents a node in the graph.
// ID is a unique identifier. Metadata can hold arbitrary key-value data.
type Vertex struct {
	ID       string
	Metadata map[string]interface{}
}

// Edge represents a connection between two vertices.
// From → To, with an int64 Weight.
// Note: Weight is stored always, but only honored if Graph.Weighted() == true; algorithms decide whether to use it.
type Edge struct {
	From   *Vertex
	To     *Vertex
	Weight int64
}

// Graph is the core data structure for a (multi)graph.
// It supports directed vs undirected, weighted vs unweighted edges,
// - directed: if true, edges have orientation.
// - weighted: if true, edge weights are meaningful.
// and uses an adjacency list under the hood.
// All mutations are protected by an internal mutex.
type Graph struct {
	mu            sync.RWMutex
	directed      bool
	weighted      bool
	vertices      map[string]*Vertex
	adjacencyList map[string]map[string][]*Edge
}

// NewGraph constructs an empty Graph.
// directed=true→directed graph, weighted=true→honor weights.
func NewGraph(directed, weighted bool) *Graph {
	return &Graph{
		directed:      directed,
		weighted:      weighted,
		vertices:      make(map[string]*Vertex),
		adjacencyList: make(map[string]map[string][]*Edge),
	}
}

// CloneEmpty returns a new Graph with the same set of vertices,
// but with no edges. Metadata maps are shared.
func (g *Graph) CloneEmpty() *Graph {
	g.mu.RLock()
	defer g.mu.RUnlock()

	clone := NewGraph(g.directed, g.weighted)
	for id, v := range g.vertices {
		clone.vertices[id] = &Vertex{
			ID:       v.ID,
			Metadata: v.Metadata, // shallow copy; share Metadata
		}
		clone.adjacencyList[id] = make(map[string][]*Edge)
	}
	return clone
}

// Clone returns a deep copy of the Graph: all vertices and edges.
// Note: Metadata maps are shared (shallow). To deep-copy Metadata, iterate yourself.
func (g *Graph) Clone() *Graph {
	clone := g.CloneEmpty()

	g.mu.RLock()
	defer g.mu.RUnlock()

	for fromID, nbrs := range g.adjacencyList {
		for toID, edges := range nbrs {
			for _, e := range edges {
				clone.adjacencyList[fromID][toID] = append(
					clone.adjacencyList[fromID][toID],
					&Edge{
						From:   clone.vertices[fromID],
						To:     clone.vertices[toID],
						Weight: e.Weight,
					},
				)
			}
		}
	}
	return clone
}

// VerticesMap exposes the internal map of vertices for package use.
// It should be used carefully and read-only.
func (g *Graph) VerticesMap() map[string]*Vertex {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Return a shallow copy of the map so caller cannot mutate g.vertices.
	out := make(map[string]*Vertex, len(g.vertices))
	for id, v := range g.vertices {
		out[id] = v
	}
	return out
}

// InternalVertices returns the vertices map. Only for internal use.
func (g *Graph) InternalVertices() map[string]*Vertex {
	return g.vertices
}

// Directed returns the directed flag.
func (g *Graph) Directed() bool {
	return g.directed
}

// Weighted returns the weighted flag.
func (g *Graph) Weighted() bool {
	return g.weighted
}

// AdjacencyList returns the adjacencyList.
func (g *Graph) AdjacencyList() map[string]map[string][]*Edge {
	return g.adjacencyList
}
