// Package core provides a high-performance, thread-safe in-memory Graph
// implementation with a minimal, composable API surface.
//
// The Graph G = (V,E) supports a rich mix of behaviors:
//
//   - Directed vs. undirected edges (WithDirected)
//   - Global vs. per-edge orientation in “mixed” graphs (WithMixedEdges + WithEdgeDirected)
//   - Weighted vs. unweighted edges (WithWeighted)
//   - Parallel edges / multi-graphs (WithMultiEdges)
//   - Self-loops (WithLoops)
//   - Constant-time edge operations via nested maps:
//     adjacencyList[from][to][edgeID] = struct{}{}
//   - Collision-free atomic Edge.ID generation (“e1”, “e2”, …)
//   - Separate sync.RWMutex for vertices (muVert) and edges+adjacency (muEdgeAdj)
//     to minimize lock contention under concurrency
//
// Why use core.Graph?
//
//   - Single type, composable flags — no explosion of separate graph types.
//   - Deterministic iteration — Vertices(), Edges(), NeighborIDs() all return sorted results.
//   - Flexible mixing — combine directed, undirected, loops, weights, multi-edges in one graph.
//   - Clone support — CloneEmpty (vertices+flags), Clone (deep copy of edges+adjacency).
//   - Extensible utility methods — degree counts, clear, filter, path-prelude, …
//
// Configuration Options (GraphOption):
//
//	– WithDirected(defaultDirected bool)
//	    Sets the default orientation of new edges.
//	    • Directed graphs store only “from→to” pointers.
//	    • Undirected graphs mirror edges in adjacencyList[to][from].
//
//	– WithMixedEdges()
//	    Allows per-edge overrides via EdgeOption.WithEdgeDirected().
//	    Without it, any override returns ErrMixedEdgesNotAllowed.
//
//	– WithWeighted()
//	    Permits non-zero weights globally; otherwise AddEdge(weight≠0) → ErrBadWeight.
//
//	– WithMultiEdges()
//	    Allows multiple parallel edges between the same endpoints.
//	    Otherwise a second AddEdge(from,to) → ErrMultiEdgeNotAllowed.
//
//	– WithLoops()
//	    Permits self-loops (from == to); otherwise AddEdge(v,v) → ErrLoopNotAllowed.
//
// EdgeOptions:
//
//	– WithEdgeDirected(directed bool)
//	    Override the graph’s default direction per-edge (mixed mode only).
//
// Core Methods:
//
//	// Vertex lifecycle
//	AddVertex(id string) error         // O(1)
//	HasVertex(id string) bool          // O(1)
//	RemoveVertex(id string) error      // O(deg(v)+M)
//
//	// Edge lifecycle
//	AddEdge(from,to string, weight int64, opts ...EdgeOption) (edgeID string, err error) // O(1)†
//	RemoveEdge(edgeID string) error   // O(1)
//	HasEdge(from,to string) bool      // O(1)
//
//	// Query
//	Neighbors(id string) ([]*Edge, error)   // O(d·log d), loops appear once, multi-edges repeated
//	NeighborIDs(id string) ([]string, error)// O(d·log d), unique, sorted
//	AdjacencyList() map[string][]string      // O(V+E)
//	Vertices() []string                      // O(V·log V)
//	Edges() []*Edge                          // O(E·log E)
//
//	// Counts & degrees
//	Degree(id string) (in,out,undirected int, err error) // in/out counts + undirected count (loops, mirrors)
//	VertexCount() int                    // O(1)
//	EdgeCount() int                      // O(1)
//
//	// Maintenance
//	Clear()                              // O(1): reset maps, counter; preserve flags
//	FilterEdges(pred func(*Edge) bool)   // O(E): remove edges failing predicate
//
//	// Cloning
//	CloneEmpty() *Graph                  // O(V): copy vertices+flags only
//	Clone() *Graph                       // O(V+E): deep-copy vertices+edges+adjacency
//
//	// Shallow view
//	VerticesMap() map[string]*Vertex     // O(V): read-only copy of vertices
//	InternalVertices() map[string]*Vertex// live map (no locking!)
//
// Edge struct fields:
//
//	ID       string   // “e1”, “e2”, …
//	From     string   // source vertex ID
//	To       string   // destination vertex ID
//	Weight   int64    // cost/capacity (zero in unweighted graphs)
//	Directed bool     // true=one-way, false=bidirectional (mixed graphs only)
//
// Errors:
//
//		ErrEmptyVertexID       – zero-length vertex ID
//		ErrVertexNotFound      – missing vertex
//		ErrEdgeNotFound        – missing edge
//		ErrBadWeight           – non-zero weight on unweighted graph
//		ErrLoopNotAllowed      – self-loop when loops disabled
//		ErrMultiEdgeNotAllowed – parallel edge when multi-edges disabled
//		ErrMixedEdgesNotAllowed – per-edge override without mixed-mode
//
//	 also amortized constant time: atomic ID generation + nested-map insertion.
//
// For a deep dive (pseudocode, math formulation, examples), see docs/CORE.md.
package core
