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

/*


// =========== ALL RELATIONS ===========
//
// ---------- formal hierarchy ----------
var hierarchyRelations = []PeopleRelation{
	// Rector → Deans
	{From: "Albus_Dumbledore", To: "Dr_Manhattan", Relation: 6, Reasons: "Rector trusts a quantum demigod to run Multiversal Studies."},
	{From: "Albus_Dumbledore", To: "Aristotle", Relation: 6, Reasons: "Appoints the Lyceum founder Dean of Classical Thought."},
	{From: "Albus_Dumbledore", To: "Leonardo_da_Vinci", Relation: 6, Reasons: "Names the Renaissance polymath Dean of Innovation & Design."},
	{From: "Albus_Dumbledore", To: "Sun_Tzu", Relation: 6, Reasons: "Grants the ancient strategist the helm of Leadership Faculty."},

	// Dr Manhattan → Chairs
	{From: "Dr_Manhattan", To: "John_von_Neumann", Relation: 6, Reasons: "Delegates math & CS to the inventor of game theory."},
	{From: "Dr_Manhattan", To: "Isaac_Newton", Relation: 6, Reasons: "Puts gravity’s father in charge of Math–Physics synergy."},
	{From: "Dr_Manhattan", To: "Hippocrates_of_Kos", Relation: 6, Reasons: "Hands biomedical faculty to the ‘father of medicine’."},
	// Aristotle → Chairs
	{From: "Aristotle", To: "Immanuel_Kant", Relation: 6, Reasons: "Passes ethical curriculum to the critic of pure reason."},
	{From: "Aristotle", To: "William_Shakespeare", Relation: 6, Reasons: "Commissions the Bard to cultivate rhetoric & drama."},
	// Leonardo da Vinci → Chairs
	{From: "Leonardo_da_Vinci", To: "Ludwig_van_Beethoven", Relation: 6, Reasons: "Oversees sonic creativity via the symphonic titan."},
	{From: "Leonardo_da_Vinci", To: "Neil_Armstrong", Relation: 6, Reasons: "Places first Moon-walker over Aerospace & Exploration."},
	// Sun Tzu → Chair
	{From: "Sun_Tzu", To: "Saitama", Relation: 6, Reasons: "Assigns the fastest man to head Sports & Performance doctrine."},

	// ---------- Chairs → Professors ----------
	// John von Neumann
	{From: "John_von_Neumann", To: "Ada_Lovelace", Relation: 6, Reasons: "Guides algorithmic heritage in Computational Dept."},
	{From: "John_von_Neumann", To: "Alan_Turing", Relation: 6, Reasons: "Co-chairs theoretical CS & AI foundations."},
	// Isaac Newton
	{From: "Isaac_Newton", To: "Albert_Einstein", Relation: 6, Reasons: "Relativity under the shoulders of giants."},
	{From: "Isaac_Newton", To: "Galileo_Galilei", Relation: 6, Reasons: "Continues the telescope-to-tensor lineage."},
	{From: "Isaac_Newton", To: "Archimedes_of_Syracuse", Relation: 6, Reasons: "Links levers to calculus in Mechanics unit."},
	// Hippocrates
	{From: "Hippocrates_of_Kos", To: "Marie_Curie", Relation: 6, Reasons: "Radiology & oath meet in Medical Sciences."},
	{From: "Hippocrates_of_Kos", To: "Jane_Goodall", Relation: 6, Reasons: "Primate health & ethics supervision."},
	{From: "Hippocrates_of_Kos", To: "Sigmund_Freud", Relation: 6, Reasons: "Mind–body symposium stewardship."},
	// Immanuel Kant
	{From: "Immanuel_Kant", To: "Noam_Chomsky", Relation: 6, Reasons: "Language & categories cross-examined."},
	{From: "Immanuel_Kant", To: "Karl_Marx", Relation: 6, Reasons: "Historical materialism under critical reason."},
	// William Shakespeare
	{From: "William_Shakespeare", To: "Homer", Relation: 6, Reasons: "Epic & tragedy joint seminars."},
	{From: "William_Shakespeare", To: "Claude_Monet", Relation: 6, Reasons: "Word-painting meets plein-air colour."},
	{From: "William_Shakespeare", To: "Pablo_Picasso", Relation: 6, Reasons: "Cubist stagecraft experiments."},
	{From: "William_Shakespeare", To: "Michelangelo_Buonarroti", Relation: 6, Reasons: "Sistine storytelling in set design."},
	// Ludwig van Beethoven
	{From: "Ludwig_van_Beethoven", To: "Wolfgang_Amadeus_Mozart", Relation: 6, Reasons: "Bridges classical clarity to romantic storm."},
	// Neil Armstrong
	{From: "Neil_Armstrong", To: "Charles_Darwin", Relation: 6, Reasons: "From Beagle to Eagle—evolution of exploration."},
	// Saitama
	{From: "Saitama", To: "Nelson_Mandela", Relation: 6, Reasons: "Sprint legend channels unity through sport diplomacy."},
}

// professors & curators — списки из positionOf
var professors = []string{
	"Marie_Curie", "Albert_Einstein", "Galileo_Galilei", "Ada_Lovelace",
	"Alan_Turing", "Jane_Goodall", "Sigmund_Freud", "Noam_Chomsky",
	"Karl_Marx", "Michelangelo_Buonarroti", "Pablo_Picasso",
	"Wolfgang_Amadeus_Mozart", "Homer", "Archimedes_of_Syracuse",
	"Charles_Darwin", "Claude_Monet", "Nelson_Mandela",
}
var curators = []string{
	"Sherlock_Holmes", "Princess_Leia_Organa", "Gandalf", "Yoda",
	"Hayao_Miyazaki", "Hans_Zimmer", "Johan_Liebert", "Lara_Croft",
	"Griffith", "Armin_Arlert", "Shuri", "Motoko_Kusanagi",
	"Captain_America_Steve_Rogers", "Dr_Stephen_Strange",
	"Tyler_Durden", "Rapunzel_Tangled", "Satoru_Gojo", "Neo_Thomas_Anderson",
}

// BuildMentorshipEdges returns 306 directed edges Professor → Curator.
func BuildMentorshipEdges() []PeopleRelation {
	var edges []PeopleRelation
	for _, prof := range professors {
		for _, cur := range curators {
			edges = append(edges, PeopleRelation{
				From:     prof,
				To:       cur,
				Relation: 6,
				Reasons:  fmt.Sprintf("Academic mentorship: %s supervises curator %s.", prof, cur),
			})
		}
	}
	return edges
}
*/
