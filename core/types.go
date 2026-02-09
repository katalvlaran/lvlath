// SPDX-License-Identifier: MIT

// Package core defines deterministic, thread-safe in-memory graphs and the
// foundational types (Graph/Vertex/Edge) used across lvlath.
//
// Design invariants (public contract, enforced across the codebase):
//  1. Determinism: public enumeration order is stable and documented
//     (Vertices by ID asc, Edges by Edge.ID asc, NeighborIDs by ID asc).
//  2. Sentinel errors only: exported operations return package-level sentinels
//     and are checked via errors.Is; no fmt-wrapping of those sentinels.
//  3. No hidden global state: behavior is explicit via options; no ambient randomness.
//  4. Concurrency: vertex catalog and edge/adjacency storage are protected by
//     separate RWMutexes (muVert, muEdgeAdj) to reduce contention.
//
// Notes:
//   - Graph configuration flags are set only during construction (NewGraph + GraphOption).
//     After construction they are immutable, so internal code may read them without locks
//     to avoid lock inversion in hot paths.
package core

import (
	"errors"
	"sync"
)

var (
	// ErrEmptyVertexID signals that the provided vertex identifier is empty.
	//
	// Contract:
	//   - Any API that accepts a vertex ID MUST reject "" with this sentinel.
	ErrEmptyVertexID = errors.New("core: vertex ID is empty")

	// ErrVertexNotFound indicates that a referenced vertex does not exist.
	//
	// Contract:
	//   - Returned by query/mutation APIs that require a pre-existing vertex.
	ErrVertexNotFound = errors.New("core: vertex not found")

	// ErrEdgeNotFound indicates that a referenced edge (by Edge.ID) was not found.
	//
	// Contract:
	//   - Returned by edge-removal or lookup routines.
	ErrEdgeNotFound = errors.New("core: edge not found")

	// ErrBadWeight reports a non-zero weight on an unweighted graph.
	//
	// Contract:
	//   - On graphs without WithWeighted(), only weight == 0 is allowed.
	ErrBadWeight = errors.New("core: bad weight for unweighted graph")

	// ErrLoopNotAllowed reports a self-loop attempt when loops are disabled.
	//
	// Contract:
	//   - WithLoops() must be set to allow edges (v -> v).
	ErrLoopNotAllowed = errors.New("core: self-loop not allowed")

	// ErrEmptyEdgeID signals that an explicit edge ID was required but empty.
	//
	// Contract:
	//   - AddEdge(..., WithID("")) MUST return ErrEmptyEdgeID.
	//   - SetEdgeID(old,"") MUST return ErrEmptyEdgeID.
	//
	// Notes:
	//   - This sentinel is about edge identifiers (Edge.ID), not vertex IDs.
	ErrEmptyEdgeID = errors.New("core: empty edge ID")

	// ErrEdgeIDConflict signals that an explicit edge ID collides with an existing edge.
	//
	// Contract:
	//   - AddEdge(..., WithID(id)) MUST return ErrEdgeIDConflict if id is already present.
	//   - SetEdgeID(old,id) MUST return ErrEdgeIDConflict if id is already present.
	//
	// Determinism:
	//   - Collision checks are pure map membership checks; no iteration order dependence.
	ErrEdgeIDConflict = errors.New("core: edge with that ID allready exist")

	// ErrMultiEdgeNotAllowed reports a parallel edge attempt when multi-edges are disabled.
	//
	// Contract:
	//   - WithMultiEdges() must be set to allow (u,v) duplication (or directional duplicates).
	ErrMultiEdgeNotAllowed = errors.New("core: multi-edges not allowed")

	// ErrMixedEdgesNotAllowed reports a per-edge directedness override on a non-mixed graph.
	//
	// Contract:
	//   - WithMixedEdges() (or NewMixedGraph) must be set before any WithEdgeDirected(...) override.
	ErrMixedEdgesNotAllowed = errors.New("core: mixed-mode per-edge overrides not allowed")
)

// Vertex is the canonical node record used by Graph; the unique key is Vertex.ID.
// Metadata is an opaque, caller-managed payload that the core does not interpret.
//
// Implementation:
//   - Stage 1: Graph stores vertices in a map keyed by Vertex.ID.
//   - Stage 2: Algorithms treat Metadata as an opaque pointer; Clone is shallow.
//
// Behavior highlights:
//   - Vertex identity is stable for the lifetime of the graph.
//   - Metadata ownership belongs to the caller; core never mutates it.
//
// Inputs:
//   - ID: unique identifier within a Graph; must be non-empty.
//   - Metadata: arbitrary user payload; may be nil.
//
// Returns:
//   - N/A (data type).
//
// Errors:
//   - N/A (data type).
//
// Determinism:
//   - Vertex.ID is the stable ordering key for public enumeration.
//
// Complexity:
//   - N/A (data type).
//
// Notes:
//   - Graph.Clone copies the Metadata map pointer; callers must deep-copy if required.
//
// AI-Hints:
//   - Prefer short, stable IDs for deterministic logs and golden tests.
//   - Keep Metadata small and immutable if used concurrently.
type Vertex struct {
	// ID uniquely identifies a vertex within a single Graph instance.
	ID string

	// Metadata holds arbitrary user data. Graph.Clone performs a shallow copy
	// of this map pointer; if deep-copy is required, callers must handle it externally.
	Metadata map[string]interface{}
}

// Edge is the canonical connection record used by Graph; the unique key is Edge.ID.
// Endpoints are stored as vertex IDs to keep Graph storage compact and deterministic.
//
// Implementation:
//   - Stage 1: AddEdge constructs a baseline Edge with graph defaults.
//   - Stage 2: AddEdge applies EdgeOptions sequentially (first error aborts).
//   - Stage 3: Edge is registered in the edge catalog and adjacency is updated.
//
// Behavior highlights:
//   - ID is unique within a graph for the graph lifetime.
//   - For undirected edges (Directed=false), adjacency is mirrored.
//
// Inputs:
//   - ID: unique edge identifier; auto-generated unless WithID is used.
//   - From/To: endpoint vertex IDs.
//   - Weight: must be 0 unless the graph is WithWeighted().
//   - Directed: effective directionality (may be overridden per-edge only in mixed mode).
//
// Returns:
//   - N/A (data type).
//
// Errors:
//   - N/A (data type).
//
// Determinism:
//   - Edge.ID is the stable ordering key for public enumeration.
//
// Complexity:
//   - N/A (data type).
//
// Notes:
//   - Auto-generated IDs are of the form "eN" where N is a monotonically increasing counter.
//
// AI-Hints:
//   - Treat returned *Edge pointers from getters as read-only to avoid data races.
type Edge struct {
	// ID is a unique string identifier for the edge (auto-generated by default, or provided via WithID).
	ID string

	// From is the source vertex ID (for undirected edges this is one endpoint).
	From string

	// To is the destination vertex ID (for undirected edges this is the other endpoint).
	To string

	// Weight is the edge cost/capacity; must be 0 unless the graph is WithWeighted().
	Weight float64

	// Directed = true means the edge is asymmetric (From -> To only).
	// Directed = false means the edge is symmetric (From <-> To, adjacency mirrored).
	Directed bool
}

// GraphOption mutates only a newly constructed Graph inside NewGraph.
//
// Implementation:
//   - Stage 1: NewGraph allocates empty catalogs.
//   - Stage 2: Options are applied deterministically in call order.
//
// Behavior highlights:
//   - Options are construction-time only.
//   - Invalid option arguments (if any) panic immediately (programmer error).
//
// Inputs:
//   - g: the graph instance being constructed.
//
// Returns:
//   - N/A.
//
// Errors:
//   - N/A (NewGraph does not return errors; invalid option arguments must panic).
//
// Determinism:
//   - Deterministic application order (left-to-right).
//
// Complexity:
//   - Time O(1) per option, Space O(1).
//
// Notes:
//   - Core avoids hidden global state; behavior changes must be explicit via options.
//
// AI-Hints:
//   - Prefer explicit options over implicit defaults when writing reproducible tests.
type GraphOption func(g *Graph)

// WithDirected sets the default directedness for all future edges created in this graph.
// Per-edge overrides are only allowed in mixed mode (WithMixedEdges + WithEdgeDirected).
//
// Implementation:
//   - Stage 1: Store defaultDirected into g.directed during construction.
//
// Behavior highlights:
//   - Affects only the default for future edges (not existing edges).
//
// Inputs:
//   - defaultDirected: true for directed-by-default graphs; false for undirected-by-default.
//
// Returns:
//   - GraphOption: construction-time mutator.
//
// Errors:
//   - None.
//
// Determinism:
//   - Deterministic; constant-time flag set.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Mixed mode controls whether individual edges may override Directed.
//
// AI-Hints:
//   - Use WithMixedEdges() if your workload needs both directed and undirected edges in one graph.
//   - Sets only the default; per-edge override requires WithMixedEdges().
func WithDirected(defaultDirected bool) GraphOption {
	return func(g *Graph) { g.directed = defaultDirected }
}

// WithWeighted enables non-zero edge weights in this graph.
//
// Implementation:
//   - Stage 1: Set g.weighted = true during construction.
//
// Behavior highlights:
//   - Without this option, AddEdge rejects weight != 0 with ErrBadWeight.
//
// Returns:
//   - GraphOption: construction-time mutator.
//
// Errors:
//   - None.
//
// Determinism:
//   - Deterministic; constant-time flag set.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// AI-Hints:
//   - Keep graphs unweighted if you only need topology; it simplifies inputs and tests.
//   - Without this, AddEdge(weight!=0) returns ErrBadWeight.
func WithWeighted() GraphOption {
	return func(g *Graph) { g.weighted = true }
}

// WithMultiEdges permits parallel edges between the same endpoints.
//
// Implementation:
//   - Stage 1: Set g.allowMulti = true during construction.
//
// Behavior highlights:
//   - Without this option, AddEdge on an existing (from,to) bucket returns ErrMultiEdgeNotAllowed.
//
// Returns:
//   - GraphOption: construction-time mutator.
//
// Errors:
//   - None.
//
// Determinism:
//   - Deterministic; constant-time flag set.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// AI-Hints:
//   - Enable multi-edges for multigraph models (e.g., multiple relations between entities).
//   - Without this, a second AddEdge(from,to,...) returns ErrMultiEdgeNotAllowed.
func WithMultiEdges() GraphOption {
	return func(g *Graph) { g.allowMulti = true }
}

// WithLoops permits self-loops (edges from a vertex to itself).
//
// Implementation:
//   - Stage 1: Set g.allowLoops = true during construction.
//
// Behavior highlights:
//   - Without this option, AddEdge(v,v,...) returns ErrLoopNotAllowed.
//
// Returns:
//   - GraphOption: construction-time mutator.
//
// Errors:
//   - None.
//
// Determinism:
//   - Deterministic; constant-time flag set.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// AI-Hints:
//   - Keep loops disabled unless your math model explicitly includes self transitions.
//   - Without this, AddEdge(v,v,...) returns ErrLoopNotAllowed.
func WithLoops() GraphOption {
	return func(g *Graph) { g.allowLoops = true }
}

// WithMixedEdges enables per-edge directedness overrides (mixed mode).
// In mixed mode, individual edges may specify Directed=true/false via EdgeOption.
//
// Implementation:
//   - Stage 1: Set g.allowMixed = true during construction.
//
// Behavior highlights:
//   - Without this option, WithEdgeDirected(...) returns ErrMixedEdgesNotAllowed.
//
// Returns:
//   - GraphOption: construction-time mutator.
//
// Errors:
//   - None.
//
// Determinism:
//   - Deterministic; constant-time flag set.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// AI-Hints:
//   - Mixed mode is a capability flag; keep it off if all edges share the same orientation.
func WithMixedEdges() GraphOption {
	return func(g *Graph) { g.allowMixed = true }
}

// EdgeOption configures a single edge during AddEdge.
//
// Implementation:
//   - Stage 1: AddEdge builds a baseline Edge (defaults + endpoints + weight).
//   - Stage 2: AddEdge applies EdgeOptions sequentially; the first error aborts the operation.
//
// Behavior highlights:
//   - Options MUST NOT panic as part of the public contract.
//   - Options MUST NOT mutate global graph state as a side effect.
//   - Options should be O(1) and deterministic.
//
// Inputs:
//   - g: the owning graph (read-only for options; do not mutate global state).
//   - e: the edge being configured (options may mutate only this edge).
//
// Returns:
//   - error: nil on success; otherwise a sentinel error (ErrMixedEdgesNotAllowed, ErrEmptyEdgeID, ErrEdgeIDConflict, ...).
//
// Errors:
//   - Must return only stable sentinels (checked via errors.Is).
//
// Determinism:
//   - Deterministic given deterministic inputs; option order is call-order stable.
//
// Complexity:
//   - Time O(1) per option, Space O(1).
//
// Notes:
//   - AddEdge applies options under the edge/adjacency write lock; options must not take additional locks.
//
// AI-Hints:
//   - Keep options “local”: modify only *e and validate against graph capability flags.
type EdgeOption func(g *Graph, e *Edge) error

// WithEdgeDirected overrides directedness for a single edge.
//
// Implementation:
//   - Stage 1: Validate that the graph was constructed with WithMixedEdges().
//   - Stage 2: Assign e.Directed = directed.
//
// Behavior highlights:
//   - Requires WithMixedEdges mode; otherwise returns ErrMixedEdgesNotAllowed.
//
// Inputs:
//   - directed: desired directedness for this edge.
//
// Returns:
//   - EdgeOption: per-edge mutator.
//
// Errors:
//   - ErrMixedEdgesNotAllowed: if the graph does not allow per-edge directedness overrides.
//
// Determinism:
//   - Deterministic; constant-time flag set.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - This option does not change endpoints; only directionality.
//
// AI-Hints:
//   - Use mixed mode only when you truly need both orientations in one graph.
func WithEdgeDirected(directed bool) EdgeOption {
	return func(g *Graph, e *Edge) error {
		// Mixed-mode is a construction-time capability flag (immutable after NewGraph).
		if !g.allowMixed {
			return ErrMixedEdgesNotAllowed
		}
		// Apply the per-edge override.
		e.Directed = directed
		return nil
	}
}

// WithID assigns a custom identifier to the edge created by AddEdge.
//
// Implementation:
//   - Stage 1: Validate id is non-empty (ErrEmptyEdgeID).
//   - Stage 2: Under AddEdge's edge lock, check the edge catalog for collisions (ErrEdgeIDConflict).
//   - Stage 3: Set e.ID = id.
//   - Stage 4: If id matches the canonical auto-ID form "eN", bump the auto-ID counter
//     so that future auto-generated IDs never collide with the explicit "eN".
//
// Behavior highlights:
//   - Works regardless of mixed mode.
//   - Deterministic: collision is a strict membership check, not a scan.
//
// Inputs:
//   - id: desired edge identifier; must be non-empty and globally unique within the graph.
//
// Returns:
//   - EdgeOption: per-edge mutator.
//
// Errors:
//   - ErrEmptyEdgeID: if id == "".
//   - ErrEdgeIDConflict: if id is already present in the edge catalog.
//
// Determinism:
//   - Deterministic; does not depend on map iteration order.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - If you choose IDs of the canonical auto form "eN", the graph will advance its auto-ID counter past N.
//
// AI-Hints:
//   - Use WithID for stable external references (golden tests, trace correlation, interop).
//   - Prefer non-auto-shaped IDs (e.g., "road_A_B") when you do not want to affect the auto-ID counter.
func WithID(id string) EdgeOption {

	return func(g *Graph, e *Edge) error {
		// Reject empty identifiers.
		if id == "" {
			return ErrEmptyEdgeID
		}

		// AddEdge applies options under muEdgeAdj write lock, so reading g.edges is safe here.
		if _, exists := g.edges[id]; exists {
			return ErrEdgeIDConflict
		}

		// Assign the custom identifier.
		e.ID = id

		// If the explicit ID looks like an auto-generated "eN", bump the counter to avoid collisions.
		if num, ok := matchesAutoIDPattern(e.ID); ok && num >= 0 {
			bumpNextEdgeIDToAtLeast(g, num)
		}

		return nil
	}
}

// Graph is a thread-safe, deterministic in-memory graph.
//
// Concurrency model:
//   - muVert protects vertex-level state (vertices map).
//   - muEdgeAdj protects edges map and adjacencyList.
//   - Locks are separated to reduce contention in algorithms that iterate neighbors frequently.
//
// Storage model:
//   - vertices: map[string]*Vertex - vertex catalog by ID.
//   - edges:    map[string]*Edge   - edge catalog by unique Edge.ID.
//   - adjacencyList: map[fromID]map[toID]map[edgeID]struct{} for O(1) membership.
//
// Configuration flags:
//   - directed/weighted/allowMulti/allowLoops/allowMixed are set only during construction
//     and remain immutable afterwards.
//
// ID generation:
//   - Auto edge IDs are "eN" where N is a monotonically increasing counter.
//   - WithID/SetEdgeID bump the counter when assigning canonical "eN" IDs.
//
// Determinism:
//   - Public enumeration order is stable and documented at package level.
//
// Notes:
//   - Clone is expected to carry over the counter to prevent collisions.
type Graph struct {
	// muVert guards vertex map and configuration flags to allow safe, low-contention reads.
	muVert sync.RWMutex

	// muEdgeAdj guards edge catalog and adjacency list for consistent graph topology updates.
	muEdgeAdj sync.RWMutex

	// Configuration flags (read under muVert; written only during construction).
	directed   bool // default directedness for future edges
	weighted   bool // allow non-zero weights
	allowMulti bool // allow parallel edges
	allowLoops bool // allow self-loops
	allowMixed bool // allow per-edge directedness overrides (mixed mode)

	// nextEdgeID is the next sequential ID number to use for auto-generated edge IDs.
	// It is incremented atomically. Custom edge IDs (via WithID) overwrite ID, but don't change the counting process.
	nextEdgeID uint64

	// Vertex catalog: ID -> *Vertex (caller-visible IDs, stable across runtime).
	vertices map[string]*Vertex

	// Edge catalog: EdgeID -> *Edge (unique identities across the graph’s lifetime).
	edges map[string]*Edge

	// Adjacency: fromID -> toID -> edgeID -> unit. Provides O(1) membership checks
	// and deterministic extraction (final sorting happens at API boundaries).
	adjacencyList map[string]map[string]map[string]struct{}
}

// GraphStats MAIN DESCRIPTION.
// GraphStats is a read-only result object summarizing graph state.
//
// Implementation:
//   - Filled by (*Graph).Stats() by scanning catalogs (implementation elsewhere).
//
// Behavior highlights:
//   - Pure data type: no methods, no side effects.
//
// Determinism:
//   - Field meanings are stable and part of the public contract.
//
// Complexity:
//   - N/A (data type).
//
// AI-Hints:
//   - Use Stats() in tests as a deterministic admission check (counts, feature flags).
type GraphStats struct {
	// DirectedDefault is the graph-wide default orientation (true=directed).
	DirectedDefault bool

	// Weighted is true when non-zero edge weights are permitted.
	Weighted bool

	// AllowsMulti is true when parallel edges are permitted.
	AllowsMulti bool

	// AllowsLoops is true when self-loops are permitted.
	AllowsLoops bool

	// MixedMode is true when per-edge Directed overrides are permitted.
	MixedMode bool

	// VertexCount is the number of vertices present in the graph.
	VertexCount int

	// EdgeCount is the number of edges present in the graph.
	EdgeCount int

	// DirectedEdgeCount is the number of edges with Directed == true.
	DirectedEdgeCount int

	// UndirectedEdgeCount is the number of edges with Directed == false.
	UndirectedEdgeCount int
}

// NewGraph constructs an empty graph and applies GraphOption values deterministically.
//
// Implementation:
//   - Stage 1: Allocate empty vertex/edge catalogs and adjacency map.
//   - Stage 2: Apply options in call order (left-to-right).
//
// Behavior highlights:
//   - Construction-time options only; subsequent behavior is controlled by method contracts.
//   - nextEdgeID starts at 0; the first auto-generated edge becomes "e1".
//
// Inputs:
//   - opts: zero or more GraphOption values applied in call order.
//
// Returns:
//   - *Graph: configured empty graph.
//
// Errors:
//   - None (invalid option arguments must panic inside the option).
//
// Determinism:
//   - Deterministic option application order.
//
// Complexity:
//   - Time O(len(opts)), Space O(1) excluding map growth.
//
// AI-Hints:
//   - Keep option sets explicit to make tests reproducible.
func NewGraph(opts ...GraphOption) *Graph {
	// Allocate a new Graph with empty maps; sizes grow amortized O(1).
	g := &Graph{
		vertices:      make(map[string]*Vertex),
		edges:         make(map[string]*Edge),
		adjacencyList: make(map[string]map[string]map[string]struct{}),
	}

	// Apply user-provided options deterministically (left-to-right).
	var opt GraphOption
	for _, opt = range opts {
		opt(g)
	}

	// Return the configured, empty graph. All mutations happen via methods
	// that enforce invariants, determinism, and sentinel errors.
	return g
}

// Nilable provides an explicit, reflect-free mechanism to treat typed-nil receivers
// stored inside interfaces as nil during validation and testing.
//
// Implementation:
//   - Stage 1: Callers accept an interface value that may hold a typed nil pointer.
//   - Stage 2: If the dynamic value implements Nilable, callers invoke IsNil().
//   - Stage 3: If IsNil reports true, the value is treated as nil by the caller.
//
// Behavior highlights:
//   - Avoids reflect in foundational validators and test helpers.
//   - Keeps nil-detection O(1) and deterministic.
//   - Optional: types that do not implement Nilable are checked with regular `== nil`.
//
// Returns:
//   - bool: true if the receiver should be treated as nil.
//
// Errors:
//   - None. IsNil MUST NOT panic and MUST NOT allocate.
//
// Determinism:
//   - Deterministic and side-effect free (required).
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - In Go, an interface can be non-nil while holding a typed nil pointer.
//
// AI-Hints:
//   - Implement IsNil() on pointer-backed types that are commonly stored behind interfaces.
type Nilable interface {
	// IsNil reports whether the receiver should be treated as nil.
	// It MUST be side-effect free, deterministic, non-allocating, and MUST NOT panic.
	IsNil() bool
}
