// File: methods_edges.go
// Role: Edge lifecycle & queries: AddEdge/RemoveEdge/HasEdge/GetEdge/Edges/EdgeCount,
//       plus feature queries and filtered removals. Also: nextEdgeID().
// Determinism:
//   - Edges() returns edges sorted by Edge.ID asc.
//   - nextEdgeID() is monotonic and stable ("e" + decimal).
// Concurrency:
//   - Mutations under muEdgeAdj write lock.
//   - Read queries under muEdgeAdj read lock.
// AI-HINT (file):
//   - Unweighted graphs MUST add edges with weight==0 (else ErrBadWeight).
//   - Per-edge overrides (WithEdgeDirected) require WithMixedEdges(); otherwise ErrMixedEdgesNotAllowed.
//   - Edges() returns deterministic order by Edge.ID asc (stable logs/goldens).

package core

import (
	"sort"
	"strconv"
	"sync/atomic"
)

// edgeIDPrefix is a private textual prefix for edge identifiers.
// Byte form is intentional to allow append to a []byte buffer without fmt.
// Ensures stable human-readable IDs like "e1", "e2", ...
const edgeIDPrefix = 'e'

// AddEdge creates a new edge, optionally directed in a mixed graph.
//
// AI-HINT:
//   - If MixedEdges()==false and opts contain WithEdgeDirected, this returns ErrMixedEdgesNotAllowed.
//   - If Weighted()==false and weight!=0, this returns ErrBadWeight.
//   - If Looped()==false and from==to, this returns ErrLoopNotAllowed.
//   - If Multigraph()==false and (from,to) already has an edge, this returns ErrMultiEdgeNotAllowed.
//
// Steps:
//  1. Validate IDs, weight, loops.
//  2. If opts present without allowMixed ⇒ ErrMixedEdgesNotAllowed.
//  3. Ensure endpoints via AddVertex.
//  4. Lock muEdgeAdj, check multi-edge constraint.
//  5. Generate eid atomically.
//  6. Build Edge struct (global g.directed default), apply opts.
//  7. Store in g.edges.
//  8. ensureAdjacency(from,to); add.
//  9. If !e.Directed && from!=to ⇒ ensureAdjacency(to,from); add (mirror).
//
// Complexity: O(1) amortized (hash-map + nested-map updates).
// Concurrency:
//   - Validates/creates vertices outside muEdgeAdj; adjacency and edge catalog under muEdgeAdj.
func (g *Graph) AddEdge(from, to string, weight int64, opts ...EdgeOption) (string, error) {
	// 1) Input validation
	if from == "" || to == "" {
		return "", ErrEmptyVertexID
	}
	if !g.weighted && weight != 0 { // weight constraint
		return "", ErrBadWeight
	}
	if from == to && !g.allowLoops { // loop constraint
		return "", ErrLoopNotAllowed
	}
	// If any per-edge options are present while mixed-mode is disabled, reject deterministically.
	// Rationale: per-edge overrides (e.g., WithEdgeDirected) are only legal in mixed graphs.
	if len(opts) > 0 && !g.allowMixed {
		return "", ErrMixedEdgesNotAllowed
	}

	// 2) Ensure vertices exist
	if err := g.AddVertex(from); err != nil {
		return "", err
	}
	if err := g.AddVertex(to); err != nil {
		return "", err
	}

	// 3) Insert edge under lock
	g.muEdgeAdj.Lock()
	defer g.muEdgeAdj.Unlock()

	if !g.allowMulti { // Multi-edge existence check
		if inner := g.adjacencyList[from][to]; len(inner) > 0 {
			return "", ErrMultiEdgeNotAllowed
		}
	}

	// 4) Generate a new unique textual edge ID in O(1) without fmt allocations.
	eid := nextEdgeID(g)

	// Construct the Edge with the _global_ default directedness
	e := &Edge{ID: eid, From: from, To: to, Weight: weight, Directed: g.directed}
	// Apply any per-edge overrides (only WithEdgeDirected exists today)
	var opt EdgeOption
	for _, opt = range opts {
		opt(e)
	}
	// Re-check loops in case WithEdgeDirected changed nothing here, but best to keep the guard in case future options interfere.
	if e.From == e.To && !g.allowLoops {
		return "", ErrLoopNotAllowed
	}

	// 5) Store and link adjacency
	g.edges[eid] = e
	ensureAdjacency(g, from, to)
	g.adjacencyList[from][to][eid] = struct{}{}

	// 6) Mirror undirected
	if !e.Directed && from != to {
		ensureAdjacency(g, to, from)
		g.adjacencyList[to][from][eid] = struct{}{}
	}

	return eid, nil
}

// RemoveEdge deletes one edge and its mirror.
// Steps:
//  1. Lock muEdgeAdj.
//  2. Lookup e, ErrEdgeNotFound if missing.
//  3. delete(g.edges, eid), removeAdjacency(e), cleanupAdjacency().
//
// Complexity: O(1) removal + O(V+E) cleanup in degenerate cases (many empty buckets).
// Concurrency: acquires muEdgeAdj write lock only.
func (g *Graph) RemoveEdge(eid string) error {
	// AI-HINT: Removing an absent edge returns ErrEdgeNotFound (no silent ignore).

	// Lock edges+adjacency
	g.muEdgeAdj.Lock()
	defer g.muEdgeAdj.Unlock()
	// Fetch edge
	e, ok := g.edges[eid]
	if !ok {
		return ErrEdgeNotFound
	}
	delete(g.edges, eid)  // Delete from global edges map
	removeAdjacency(g, e) // Remove from adjacencyList[from][to]
	cleanupAdjacency(g)   // Mirror removal for undirected

	return nil
}

// HasEdge reports whether at least one edge from→to exists.
//
// Determinism: constant-time membership via nested maps; no allocations.
// Works for undirected graphs as AddEdge mirrors adjacency automatically.
// Complexity: O(1).
// Concurrency: read lock on muEdgeAdj.
func (g *Graph) HasEdge(from, to string) bool {
	// AI-HINT: O(1) membership by adjacency; undirected edges are mirrored, so HasEdge works both ways.
	if from == "" || to == "" {
		return false
	}
	g.muEdgeAdj.RLock()
	defer g.muEdgeAdj.RUnlock()

	return len(g.adjacencyList[from][to]) > 0
}

// GetEdge returns a pointer to the Edge with the given edgeID if it exists,
// or ErrEdgeNotFound if no such edge is present.
//
// Contract:
//   - The returned *Edge must be treated as read-only by callers.
//   - Errors are strict sentinels (checked via errors.Is).
//   - No mutation of graph state occurs.
//
// Complexity: O(1) average time (hash map lookup).
// Concurrency: safe; uses the edges/adjacency read lock.
func (g *Graph) GetEdge(edgeID string) (*Edge, error) {
	// AI-HINT: Use errors.Is(err, ErrEdgeNotFound) to gate fallbacks; returned *Edge is read-only by convention.
	g.muEdgeAdj.RLock()         // lock edges/adjacency map for a consistent snapshot
	defer g.muEdgeAdj.RUnlock() // ensure unlock on all paths
	e, ok := g.edges[edgeID]    // attempt to find edge by its unique ID
	if !ok {                    // if not found, return the canonical sentinel
		return nil, ErrEdgeNotFound
	}

	return e, nil // happy path: return read-only pointer to the cataloged edge
}

// Edges returns all edges sorted by Edge.ID asc (stable, deterministic order).
// Complexity: O(E log E) for sorting; O(E) to assemble the slice.
// Concurrency: read lock on muEdgeAdj.
func (g *Graph) Edges() []*Edge {
	// AI-HINT: Deterministic ordering by Edge.ID asc; rely on it for golden tests.
	g.muEdgeAdj.RLock()
	defer g.muEdgeAdj.RUnlock()
	out := make([]*Edge, 0, len(g.edges))
	var e *Edge
	for _, e = range g.edges {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })

	return out
}

// EdgeCount returns total number of edges.
// Complexity: O(1).
// Concurrency: read lock on muEdgeAdj.
func (g *Graph) EdgeCount() int {
	// AI-HINT: O(1) size of edge catalog; does not allocate.
	g.muEdgeAdj.RLock()
	defer g.muEdgeAdj.RUnlock()

	return len(g.edges)
}

//–– Additional methods –––––––––––––––––––––––––––––––––––––––––––––––––––––

// HasDirectedEdges reports whether there exists at least one edge with Directed == true.
// Complexity: O(E).
// Concurrency: read lock on muEdgeAdj.
func (g *Graph) HasDirectedEdges() bool {
	// AI-HINT: Quick capability probe for mixed/directed algorithms; O(E) scan.
	g.muEdgeAdj.RLock()
	defer g.muEdgeAdj.RUnlock()
	var e *Edge
	for _, e = range g.edges {
		if e.Directed {
			return true
		}
	}

	return false
}

// FilterEdges removes all edges failing the predicate.
//
// Contract:
//   - pred is pure; must not mutate the graph.
//   - After removals, adjacency is cleaned to keep HasEdge/iterations fast.
//
// Complexity: O(E) scan + O(V+E) cleanup in worst case.
// Concurrency: write lock on muEdgeAdj.
func (g *Graph) FilterEdges(pred func(*Edge) bool) {
	// AI-HINT: Removes edges not satisfying pred; adjacency is cleaned; graph stays consistent.
	g.muEdgeAdj.Lock()
	defer g.muEdgeAdj.Unlock()
	var eid string
	var e *Edge
	for eid, e = range g.edges {
		if !pred(e) {
			removeAdjacency(g, e)
			delete(g.edges, eid)
		}
	}

	cleanupAdjacency(g)
}

// nextEdgeID returns a new unique textual edge ID.
//
// Determinism:
//   - Uses a monotonic uint64 counter (g.nextEdgeID) incremented atomically.
//   - Produces "e" + decimal digits (no locale/time/randomness).
//
// Performance:
//   - Avoids fmt.Sprintf to remove heap churn in hot paths.
//
// Concurrency:
//   - Safe for concurrent callers; atomic.AddUint64 is used to fetch the next number.
func nextEdgeID(g *Graph) string {
	// AI-HINT: Monotonic textual IDs ("e1","e2",...); Clone carries sequence to keep continuity.
	n := atomic.AddUint64(&g.nextEdgeID, 1) // atomically reserve the next sequence number
	buf := make([]byte, 0, 1+20)            // "e" + up to 20 digits for uint64
	buf = append(buf, edgeIDPrefix)         // textual prefix
	buf = strconv.AppendUint(buf, n, 10)    // base-10 digits

	return string(buf) // convert to immutable string
}
