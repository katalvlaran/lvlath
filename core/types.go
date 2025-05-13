// Package core defines the central Graph, Vertex, and Edge types,
// and provides thread-safe primitives for building, querying, and cloning graphs.
//
// All core APIs use sync.RWMutex internally, so you can safely mutate your
// graphs across goroutines without extra locking.
package core

import "sync"

// Vertex represents a node in the graph.
//
// ID is a unique identifier within its Graph.
// Metadata can hold arbitrary key-value data and is shared on shallow clones.
type Vertex struct {
	// ID uniquely identifies this Vertex.
	ID string

	// Metadata stores arbitrary user data. It is not deep-copied by Clone.
	Metadata map[string]interface{}
}

// Edge represents a connection between two vertices.
//
// From → To, with an integer Weight. Weight is stored regardless of
// whether the Graph is marked weighted; algorithms decide whether to use it.
type Edge struct {
	// From is the source vertex.
	From *Vertex

	// To is the destination vertex.
	To *Vertex

	// Weight of the edge. Algorithms may ignore this if Graph.Weighted() is false.
	Weight int64
}

// Graph is the core in-memory graph data structure.
//
// It supports directed vs. undirected and weighted vs. unweighted graphs.
// Internally uses an adjacency list protected by a sync.RWMutex for
// concurrent safety. Methods either acquire a read lock (RLock) for queries
// or a write lock (Lock) for mutations.
type Graph struct {
	mu            sync.RWMutex
	directed      bool
	weighted      bool
	vertices      map[string]*Vertex
	adjacencyList map[string]map[string][]*Edge
}

// NewGraph constructs an empty Graph.
//   - directed=true  ⇒ edges have orientation.
//   - weighted=true  ⇒ edge weights are meaningful.
//
// The returned Graph is safe for concurrent use.
func NewGraph(directed, weighted bool) *Graph {
	return &Graph{
		directed:      directed,
		weighted:      weighted,
		vertices:      make(map[string]*Vertex),
		adjacencyList: make(map[string]map[string][]*Edge),
	}
}

// CloneEmpty returns a new Graph with the same set of vertices but no edges.
//
// Metadata maps are shared (shallow copy). The new Graph has its own mutex
// and adjacency structure.
func (g *Graph) CloneEmpty() *Graph {
	g.mu.RLock()
	defer g.mu.RUnlock()

	clone := NewGraph(g.directed, g.weighted)
	for id, v := range g.vertices {
		// share Metadata map intentionally
		clone.vertices[id] = &Vertex{ID: v.ID, Metadata: v.Metadata}
		clone.adjacencyList[id] = make(map[string][]*Edge)
	}

	return clone
}

// Clone returns a deep copy of the Graph: all vertices and edges.
//
// Vertex.Metadata maps are shared (shallow). To deep-copy Metadata,
// iterate and copy each map yourself.
//
// Concurrency: holds a read lock while iterating.
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

// VerticesMap returns a shallow copy of the internal vertex map.
//
// The returned map maps vertex IDs to *Vertex pointers.
// Modifying the returned map will not affect the original Graph.
//
// Concurrency: acquires a read lock.
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

// InternalVertices exposes the internal vertices map directly.
//
// This bypasses locking and is intended only for internal package use.
func (g *Graph) InternalVertices() map[string]*Vertex {
	return g.vertices
}

// Directed reports whether the graph treats edges as directed.
//
// No locking needed: directed is immutable after creation.
func (g *Graph) Directed() bool {
	return g.directed
}

// Weighted reports whether the graph treats edge Weights as meaningful.
//
// No locking needed: weighted is immutable after creation.
func (g *Graph) Weighted() bool {
	return g.weighted
}

// AdjacencyList exposes the internal adjacency list map.
//
// The returned map is the underlying structure; modifying it may corrupt
// the Graph. Use only for read-only internal operations.
func (g *Graph) AdjacencyList() map[string]map[string][]*Edge {
	return g.adjacencyList
}
