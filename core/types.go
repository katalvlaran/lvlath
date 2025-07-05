// Package core defines the central Graph, Vertex, and Edge types,
// and provides thread-safe primitives for building, querying, and cloning graphs.
//
// All core APIs use separate sync.RWMutex locks internally (muVert for vertices,
// muEdgeAdj for edges and adjacency), so you can safely mutate your graphs across
// goroutines with minimal contention.
//
// This file declares Vertex, Edge, Graph, GraphOption, EdgeOption,
// sentinel errors, and the NewGraph constructor.
//
// Errors:
//
//	ErrNilVertex         - vertex pointer is nil.
//	ErrEmptyVertexID     - vertex ID is the empty string.
//	ErrVertexNotFound    - requested vertex does not exist.
//	ErrEdgeNotFound      - requested edge does not exist.
//	ErrBadWeight         - non-zero weight provided to an unweighted graph.
//	ErrLoopNotAllowed    - self-loop when loops are disabled.
//	ErrMultiEdgeNotAllowed - attempt to add parallel edge when multi-edges disabled.
package core

import (
	"errors"
	"sync"
)

// Sentinel errors for core graph operations.
var (
	// ErrEmptyVertexID indicates that the provided Vertex has an empty ID.
	ErrEmptyVertexID = errors.New("core: vertex ID is empty")

	// ErrVertexNotFound indicates an operation referenced a non-existent vertex.
	ErrVertexNotFound = errors.New("core: vertex not found")

	// ErrEdgeNotFound indicates an operation referenced a non-existent edge.
	ErrEdgeNotFound = errors.New("core: edge not found")

	// ErrBadWeight indicates a non-zero weight provided to an unweighted graph.
	ErrBadWeight = errors.New("core: bad weight for unweighted graph")

	// ErrLoopNotAllowed indicates a self-loop was attempted when loops are disabled.
	ErrLoopNotAllowed = errors.New("core: self-loop not allowed")

	// ErrMultiEdgeNotAllowed indicates a parallel edge was attempted when multi-edges are disabled.
	ErrMultiEdgeNotAllowed = errors.New("core: multi-edges not allowed")

	// ErrMixedEdgesNotAllowed indicates a mixed direction in edges when mixed-edges are disabled.
	ErrMixedEdgesNotAllowed = errors.New("core: mixed-mode per-edge overrides not allowed")
)

// Vertex represents a node in the graph.
//
// ID uniquely identifies this Vertex within its Graph.
// Metadata stores arbitrary key-value data and is shared on shallow clones.
type Vertex struct {
	// ID is the unique identifier for this Vertex.
	ID string

	// Metadata stores arbitrary user data. It is not deep-copied by Clone.
	Metadata map[string]interface{}
}

// Edge represents a connection between two vertices.
//
// Each Edge has a unique ID, endpoints From→To, integer Weight, and a Directed flag
// that overrides the Graph's default directedness when mixed edges are enabled.
type Edge struct {
	// ID uniquely identifies this edge in the Graph.
	ID string

	// From is the source vertex ID.
	From string

	// To is the destination vertex ID.
	To string

	// Weight is the cost or capacity of the edge.
	Weight int64

	// Directed indicates this edge is one-way (true) or bidirectional (false)
	// when the Graph was constructed with mixed edge support.
	Directed bool
}

// GraphOption configures behavior of a Graph before creation.
type GraphOption func(g *Graph)

// WithDirected sets the default directedness for all new edges
// (true = directed, false = undirected).
func WithDirected(defaultDirected bool) GraphOption {
	return func(g *Graph) { g.directed = defaultDirected }
}

// WithWeighted allows non-zero edge weights in the Graph.
func WithWeighted() GraphOption {
	return func(g *Graph) { g.weighted = true }
}

// WithMultiEdges permits parallel edges between the same vertices.
func WithMultiEdges() GraphOption {
	return func(g *Graph) { g.allowMulti = true }
}

// WithLoops permits self-loops (edges from a vertex to itself).
func WithLoops() GraphOption {
	return func(g *Graph) { g.allowLoops = true }
}

// WithMixedEdges GraphOption to let per-edge directedness overrides take effect:
func WithMixedEdges() GraphOption {
	return func(g *Graph) { g.allowMixed = true }
}

// EdgeOption configures properties of individual edges when added.
type EdgeOption func(*Edge)

// WithEdgeDirected overrides the Graph's default directedness for this edge.
func WithEdgeDirected(directed bool) EdgeOption {
	return func(e *Edge) { e.Directed = directed }
}

// Graph is the core in-memory graph data structure.
//
// It supports: directed vs. undirected, weighted vs. unweighted,
// parallel edges (multi-edges) and self-loops.
// muVert protects vertices map; muEdgeAdj protects edges map and adjacencyList.
// nextEdgeID is an atomic counter for unique Edge.ID generation.
type Graph struct {
	muVert    sync.RWMutex // guards vertices
	muEdgeAdj sync.RWMutex // guards edges and adjacency

	// Configuration flags
	directed   bool // default directedness
	weighted   bool // allow non-zero weights
	allowMulti bool // allow parallel edges
	allowLoops bool // allow self-loops
	allowMixed bool // allow mixed directed edges

	// Storage
	nextEdgeID uint64             // atomic edge ID generator
	vertices   map[string]*Vertex // vertex ID → Vertex
	edges      map[string]*Edge   // edge ID → Edge

	// adjacencyList[(from)Vertex.ID][(to)Vertex.ID][Edge.ID] = struct{}{}
	adjacencyList map[string]map[string]map[string]struct{}
}

// NewGraph creates an empty Graph with the given flags and options.
// By default, Graph is undirected, unweighted, no loops, no multi-edges.
// Complexity: O(1)
func NewGraph(opts ...GraphOption) *Graph {
	g := &Graph{
		vertices:      make(map[string]*Vertex),
		edges:         make(map[string]*Edge),
		adjacencyList: make(map[string]map[string]map[string]struct{}),
	}
	// Apply options
	for _, opt := range opts {
		opt(g)
	}

	return g
}
