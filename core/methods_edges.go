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
	"math"
	"sort"
	"strconv"
	"sync/atomic"
)

// edgeIDPrefix is a private textual prefix for edge identifiers.
// Byte form is intentional to allow append to a []byte buffer without fmt.
// Ensures stable human-readable IDs like "e1", "e2", ...
const edgeIDPrefix = 'e'

// IsNil reports whether the receiver should be treated as nil when stored inside interfaces.
//
// Implementation:
//   - Stage 1: Compare the receiver pointer to nil.
//   - Stage 2: Return the result without dereferencing.
//
// Behavior highlights:
//   - Safe for typed-nil stored inside interfaces (no panic).
//   - Reflect-free nil detection used by validators and test helpers via core.Nilable.
//
// Returns:
//   - bool: true iff receiver == nil.
//
// Errors:
//   - None.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Keep this method trivial; do not add deep validation.
//
// AI-Hints:
//   - Use Nilable-aware helpers to correctly detect typed nils behind interfaces.
func (e *Edge) IsNil() bool { return e == nil }

// bumpNextEdgeIDToAtLeast advances g.nextEdgeID to at least usedN.
//
// Implementation:
//   - Stage 1: Load current counter.
//   - Stage 2: If current >= usedN, return.
//   - Stage 3: CAS loop to set max(current, usedN).
//
// Behavior highlights:
//   - Monotonic: never decreases the counter.
//   - Safe under concurrent AddEdge calls due to CAS.
//
// Inputs:
//   - usedN: the numeric suffix N of a canonical "eN" ID that has been consumed.
//
// Returns:
//   - None.
//
// Errors:
//   - None.
//
// Determinism:
//   - Deterministic for a fixed interleaving; never produces collisions.
//
// Complexity:
//   - Time O(1) amortized, Space O(1).
//
// Notes:
//   - If nextEdgeID becomes N, the next auto-generated ID becomes "e(N+1)".
//
// AI-Hints:
//   - This helper is required to keep auto-ID generation collision-free when callers use WithID("eN").
func bumpNextEdgeIDToAtLeast(g *Graph, usedN uint64) {
	for {
		cur := atomic.LoadUint64(&g.nextEdgeID)
		if cur >= usedN {
			return
		}
		if atomic.CompareAndSwapUint64(&g.nextEdgeID, cur, usedN) {
			return
		}
	}
}

// AddEdge creates a new edge from `from` to `to` with an optional weight and per-edge options.
// The method enforces graph capabilities (weighted, multi-edge, loop, and mixed-edge policy)
// and publishes catalog/adjoining adjacency state as one topology transaction.
//
// Transactional Safety:
//   - This operation is TOPOLOGICALLY ATOMIC for edge publication.
//   - It acquires muVert.Lock() and muEdgeAdj.Lock() in that order.
//   - While both locks are held, endpoints cannot be concurrently removed and the edge
//     catalog/adjoining adjacency buckets cannot be observed half-published.
//   - If endpoints are missing, they are created as part of the same transaction before
//     edge publication.
//
// Implementation:
//   - Stage 1: Validate stateless public inputs: endpoint IDs, weight policy, loop policy,
//     and nil EdgeOption values.
//   - Stage 2: Acquire muVert.Lock() to start the topology transaction.
//   - Stage 3: Ensure `from` and `to` exist in the vertex catalog.
//   - Stage 4: Acquire muEdgeAdj.Lock() to protect edge catalog and adjacency mutation.
//   - Stage 5: Reject forbidden parallel edges before allocation/publication.
//   - Stage 6: Build a baseline unpublished Edge from validated endpoints, weight, and
//     graph default directedness.
//   - Stage 7: Apply EdgeOptions in call order.
//   - Stage 8: Re-validate option effects: endpoints and weight are immutable; directedness
//     override requires mixed mode; loop policy remains enforced.
//   - Stage 9: Assign an explicit or generated Edge.ID and bump the auto-ID counter when needed.
//   - Stage 10: Publish the edge in the catalog and update primary/mirrored adjacency buckets.
//   - Stage 11: Release locks through deferred unlocks.
//
// Behavior highlights:
//   - Strict sentinel errors only (classify with errors.Is).
//   - No panics as part of the public contract.
//   - Nil EdgeOption values fail fast before lock acquisition, vertex auto-creation, or adjacency mutation.
//   - Endpoint and weight mutation by custom EdgeOption values is rejected before edge publication.
//   - Directedness mutation is allowed only when mixed mode is enabled.
//   - No lock gap exists between endpoint creation and edge insertion.
//
// Inputs:
//   - from: source vertex ID; must be non-empty.
//   - to: destination vertex ID; must be non-empty.
//   - weight: edge weight; must be finite and must be 0 unless WithWeighted() was set.
//   - opts: optional per-edge mutators; each value must be non-nil.
//
// Returns:
//   - string: assigned Edge.ID (auto-generated or explicit).
//   - error: nil on success; otherwise a sentinel error.
//
// Errors:
//   - ErrEmptyVertexID: if from == "" or to == "".
//   - ErrBadWeight: if weight != 0 on an unweighted graph.
//   - ErrNaNInf: if weight is NaN/Inf.
//   - ErrLoopNotAllowed: if from == to and loops are disabled.
//   - ErrNilEdgeOption: if opts contains a nil EdgeOption value.
//   - ErrMultiEdgeNotAllowed: if a parallel edge is attempted and multi-edges are disabled.
//   - ErrInvalidEdgeOption: if an EdgeOption mutates Edge.From, Edge.To, or Edge.Weight.
//   - ErrMixedEdgesNotAllowed: if directedness is changed without mixed mode.
//   - ErrEmptyEdgeID / ErrEdgeIDConflict: from WithID / ValidateEdgeID rules.
//
// Determinism:
//   - Option validation order is stable (call order).
//   - Option application order is stable (call order).
//   - Auto-ID assignment uses a monotonic counter (no randomness, no time).
//
// Complexity:
//   - Time O(1) amortized.
//   - Space O(1) amortized.
//
// Notes:
//   - AddEdge may auto-create missing endpoint vertices before a later EdgeOption error occurs.
//     Such vertices are valid catalog state; the edge itself is not published on error.
//   - Vertices are created while muVert is held to prevent concurrent RemoveVertex from
//     deleting endpoints during edge publication.
//
// AI-Hints:
//   - Use WithID for stable cross-run edge references.
//   - For undirected edges, adjacency is mirrored automatically.
//   - Do not write custom EdgeOption values that rewrite endpoints or weights.
func (g *Graph) AddEdge(from, to string, weight float64, opts ...EdgeOption) (string, error) {
	// 1. Input validation (stateless)
	if from == "" || to == "" {
		return "", ErrEmptyVertexID
	}
	if math.IsNaN(weight) || math.IsInf(weight, 0) {
		return "", ErrNaNInf
	}
	if !g.weighted && weight != 0 {
		return "", ErrBadWeight
	}
	if from == to && !g.allowLoops {
		return "", ErrLoopNotAllowed
	}

	var opt EdgeOption
	for _, opt = range opts {
		if opt == nil {
			return "", ErrNilEdgeOption
		}
	}

	// 2. Start Transaction: Lock Vertices (Write Lock)
	// We must hold this lock to prevent concurrent RemoveVertex from deleting
	// 'from' or 'to' while we are in the process of linking them.
	g.muVert.Lock()
	defer g.muVert.Unlock()

	// 3. Ensure Vertices Exist (Inlined logic to avoid deadlock via g.AddVertex)
	if _, ok := g.vertices[from]; !ok {
		g.vertices[from] = &Vertex{ID: from, Metadata: make(map[string]interface{})}
	}
	if _, ok := g.vertices[to]; !ok {
		g.vertices[to] = &Vertex{ID: to, Metadata: make(map[string]interface{})}
	}

	// 4. Lock Edges & Adjacency
	g.muEdgeAdj.Lock()
	defer g.muEdgeAdj.Unlock()

	// 5. Enforce Multi-edge Policy
	if !g.allowMulti {
		if inner := g.adjacencyList[from][to]; len(inner) > 0 {
			return "", ErrMultiEdgeNotAllowed
		}
	}

	// 6. Build Baseline Edge
	e := &Edge{From: from, To: to, Weight: weight, Directed: g.directed}

	// 7. Apply Options
	// Note: Options are simple mutators; they do not require extra locks.
	for _, opt = range opts {
		if err := opt(g, e); err != nil {
			return "", err
		}
	}

	// EdgeOptions are not allowed to rewrite topology-owned fields.
	// If this guard is removed, a custom option can make edge catalog endpoints
	// disagree with adjacency buckets, corrupting RemoveEdge/Neighbors/HasEdge.
	if e.From != from || e.To != to || e.Weight != weight {
		return "", ErrInvalidEdgeOption
	}

	// Directedness is the only topology-semantic field that may be overridden,
	// and only when the graph explicitly opted into mixed-mode semantics.
	if !g.allowMixed && e.Directed != g.directed {
		return "", ErrMixedEdgesNotAllowed
	}
	// Re-check loops guard (options might not respect initial check, though they should)
	if e.From == e.To && !g.allowLoops {
		return "", ErrLoopNotAllowed
	}

	// 8. Assign edge ID:
	//    - If WithID was used, e.ID is already set and validated for uniqueness.
	//    - Otherwise, generate a new unique textual edge ID in O(1) without fmt allocations.
	if e.ID == "" {
		e.ID = nextEdgeID(g)
	} else {
		// Collision check for explicit ID
		if _, exists := g.edges[e.ID]; exists {
			return "", ErrEdgeIDConflict
		}
		if num, ok := matchesAutoIDPattern(e.ID); ok {
			bumpNextEdgeIDToAtLeast(g, num)
		}
	}

	// 9. Store Edge & Update Adjacency
	g.edges[e.ID] = e
	// Forward Adjacency
	ensureAdjacency(g, from, to)
	g.adjacencyList[from][to][e.ID] = struct{}{}
	// Mirror Adjacency (Undirected)
	if !e.Directed && from != to {
		ensureAdjacency(g, to, from)
		g.adjacencyList[to][from][e.ID] = struct{}{}
	}

	return e.ID, nil
}

// RemoveEdge deletes one edge by ID and unlinks its sparse adjacency references.
//
// Implementation:
//   - Stage 1: Validate non-empty edge ID (ErrEmptyEdgeID).
//   - Stage 2: Acquire muEdgeAdj.Lock() because only edge catalog and adjacency index mutate.
//   - Stage 3: Lookup the edge in the authoritative edge catalog.
//   - Stage 4: Delete the edge catalog entry and remove its adjacency references.
//   - Stage 5: Prune empty sparse adjacency buckets.
//
// Behavior highlights:
//   - Does not remove endpoint vertices, even if they become isolated.
//   - Does not acquire muVert because vertex membership is unchanged.
//   - Leaves public AdjacencyList() able to report isolated endpoints through g.vertices.
//
// Inputs:
//   - eid: edge identifier; must be non-empty.
//
// Returns:
//   - error: nil on success; otherwise a sentinel error.
//
// Errors:
//   - ErrEmptyEdgeID: if eid == "".
//   - ErrEdgeNotFound: if no edge with eid exists.
//
// Determinism:
//   - Deterministic final graph state; no iteration-order dependency.
//
// Complexity:
//   - Time O(B) because cleanup may scan sparse buckets; O(1) average without cleanup cost.
//   - Space O(1).
//
// Notes:
//   - Edge deletion changes edge topology, not vertex membership.
//
// AI-Hints:
//   - Do not add muVert locking here; it creates unnecessary contention and can invert lock order.
//   - Removing the last incident edge of a vertex must not remove the vertex from g.vertices.
func (g *Graph) RemoveEdge(eid string) error {
	// Edge-only mutation: vertex catalog is untouched.
	if eid == "" { // validate edgeID
		return ErrEmptyEdgeID
	}
	// Lock edges+adjacency
	g.muEdgeAdj.Lock()
	defer g.muEdgeAdj.Unlock()
	// Fetch edge
	e, ok := g.edges[eid]
	if !ok {
		return ErrEdgeNotFound
	}

	// Remove the authoritative edge record, then unlink sparse adjacency index entries.
	delete(g.edges, eid)  // Delete from global edges map
	removeAdjacency(g, e) // Remove from adjacencyList[from][to]
	cleanupAdjacency(g)   // Mirror removal for undirected

	return nil
}

// HasEdge reports whether at least one edge exists from 'from' to 'to'.
// It is safe to call with unknown vertex IDs; missing adjacency buckets return false.
// Implementation:
//   - Stage 1: Reject empty IDs early (fast-path, no locks).
//   - Stage 2: Acquire muEdgeAdj read lock for a stable adjacency snapshot.
//   - Stage 3: Probe adjacency buckets defensively and return membership.
//
// Behavior highlights:
//   - Undirected edges are mirrored on insertion, so HasEdge works in both directions.
//   - Missing vertices or missing adjacency buckets return false (never panic).
//
// Inputs:
//   - from: source vertex ID (non-empty for meaningful queries).
//   - to: destination vertex ID (non-empty for meaningful queries).
//
// Returns:
//   - bool: true if at least one edge from->to exists, otherwise false.
//
// Errors:
//   - N/A (pure query; never returns sentinel errors).
//
// Determinism:
//   - Deterministic; relies only on map membership (no iteration order).
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Uses adjacency buckets instead of scanning edges for performance.
//
// AI-Hints:
//   - Use HasEdge as an O(1) membership check; do not build temporary slices just to test existence.
//   - For undirected graphs, checking (to,from) is equivalent due to mirrored adjacency.
func (g *Graph) HasEdge(from, to string) bool {
	// AI-HINT: O(1) membership by adjacency; unknown vertices must return false (never panic).
	if from == "" || to == "" {
		return false
	}
	g.muEdgeAdj.RLock()
	defer g.muEdgeAdj.RUnlock()

	// Probe the first-level bucket defensively to avoid relying on implicit nil-map behavior.
	inner, ok := g.adjacencyList[from]
	if !ok || inner == nil {
		return false
	}

	// Second-level probe is safe even if inner[to] is nil; len(nil)==0 by contract.
	return len(inner[to]) > 0
}

// SetEdgeID renames an existing edge’s identifier from oldID to newID.
//
// Implementation:
//   - Stage 1: Validate inputs (non-empty).
//   - Stage 2: Acquire muEdgeAdj write lock.
//   - Stage 3: Lookup oldID (ErrEdgeNotFound) and ensure newID is free (ErrEdgeIDConflict).
//   - Stage 4: Update edge catalog key and edge.ID.
//   - Stage 5: Rewrite adjacency buckets that store the edge ID (from->to and mirror if undirected).
//   - Stage 6: Cleanup empty adjacency buckets.
//   - Stage 7: If newID is canonical "eN", bump auto-ID counter to avoid future collisions.
//
// Inputs:
//   - oldID: existing edge identifier (must be non-empty).
//   - newID: desired edge identifier (must be non-empty).
//
// Returns:
//   - error: nil on success; otherwise a sentinel error.
//
// Errors:
//   - ErrEmptyEdgeID: if oldID == "" or newID == "".
//   - ErrEdgeNotFound: if oldID does not exist.
//   - ErrEdgeIDConflict: if newID already exists.
//
// Determinism:
//   - Deterministic; map membership checks and deterministic updates.
//
// Complexity:
//   - Time O(1) average (map updates); Space O(1).
//
// Notes:
//   - This operation is atomic with respect to edge queries due to muEdgeAdj write lock.
//
// AI-Hints:
//   - Use SetEdgeID to migrate from auto-IDs to stable IDs after building a graph topology.
func (g *Graph) SetEdgeID(oldID, newID string) error {
	// Validate inputs first (fast path).
	if oldID == "" || newID == "" {
		return ErrEmptyEdgeID
	}

	// AI-HINT: This operation mutates edge catalog and adjacency maps; it MUST use the write lock.
	g.muEdgeAdj.Lock()         // exclusive lock: protects map writes (g.edges and adjacencyList buckets)
	defer g.muEdgeAdj.Unlock() // ensure unlock on all paths
	e, exist := g.edges[oldID] // attempt to find edge by its unique ID
	if !exist {                // if not found, return the canonical sentinel
		return ErrEdgeNotFound
	}
	// No-op rename is allowed.
	if oldID == newID {
		return nil
	}
	// Ensure the target ID is free.
	if _, exist = g.edges[newID]; exist {
		return ErrEdgeIDConflict
	}

	// Remove the old catalog entry first to keep uniqueness invariant simple.
	delete(g.edges, oldID)
	// Rewrite the edge record and re-insert into the catalog.
	e.ID = newID
	g.edges[newID] = e

	// Update adjacency: from->to always contains this edge.
	if m := g.adjacencyList[e.From][e.To]; m != nil {
		delete(m, oldID)
		m[newID] = struct{}{}
	} else {
		ensureAdjacency(g, e.From, e.To)
		g.adjacencyList[e.From][e.To][newID] = struct{}{}
	}

	// Update mirror adjacency for undirected non-loop edges.
	if !e.Directed && e.From != e.To {
		if m := g.adjacencyList[e.To][e.From]; m != nil {
			delete(m, oldID)
			m[newID] = struct{}{}
		} else {
			ensureAdjacency(g, e.To, e.From)
			g.adjacencyList[e.To][e.From][newID] = struct{}{}
		}
	}

	// Cleanup empty adjacency buckets to keep membership checks fast.
	cleanupAdjacency(g)

	// If the new ID matches canonical auto-ID form "eN", bump the counter.
	if num, ok := matchesAutoIDPattern(newID); ok {
		bumpNextEdgeIDToAtLeast(g, num)
	}

	return nil
}

// GetNamedEdges returns all edges whose ID is not in the canonical auto-generated “eN” form.
//
// Implementation:
//   - Stage 1: Acquire muEdgeAdj read lock.
//   - Stage 2: Filter edges whose IDs do not match matchesAutoIDPattern.
//   - Stage 3: Sort results by Edge.ID asc for deterministic order.
//
// Behavior highlights:
// - Deterministic output order (Edge.ID lex asc).
// - The returned slice container is freshly allocated.
// - Slice elements alias live catalog edges.
// - Structural edge fields MUST be treated as immutable once published in a graph.
//
// Returns:
//   - []*Edge: edges with non-auto-shaped IDs, sorted by ID.
//
// Determinism:
//   - Deterministic output order (lexicographic ID sort).
//
// Complexity:
//   - Time O(E log E), Space O(E).
//
// NOTES:
//   - Reordering or truncating the returned slice does not mutate the graph.
//   - Retaining a returned *Edge does not pin graph membership or extend graph locks.
//   - Use external value copies if detached mutable edge ownership is required.
func (g *Graph) GetNamedEdges() []*Edge {
	g.muEdgeAdj.RLock()
	defer g.muEdgeAdj.RUnlock()
	named := make([]*Edge, 0, len(g.edges))
	for _, e := range g.edges {
		// Pattern check: skip anything that looks like "e" + numbers
		if _, match := matchesAutoIDPattern(e.ID); match {
			continue // id = "e"+number, skip (auto-ID)
		}
		named = append(named, e)
	}
	// Sort result lexicographically by ID to ensure stable order
	sort.Slice(named, func(i, j int) bool { return named[i].ID < named[j].ID })

	return named
}

// GetEdge returns the edge with the given edgeID, or ErrEdgeNotFound if absent.
//
// Implementation:
//   - Stage 1: Acquire muEdgeAdj read lock for a consistent catalog snapshot.
//   - Stage 2: Lookup g.edges[edgeID].
//   - Stage 3: Return the pointer (read-only by convention) or ErrEdgeNotFound.
//
// Behavior highlights:
//   - O(1) average lookup time.
//   - Does not allocate.
//   - Does not mutate graph state.
//
// Inputs:
//   - edgeID: edge identifier (Edge.ID) to lookup.
//
// Returns:
//   - *Edge: pointer to the cataloged edge (treat as read-only).
//   - error: nil on success; ErrEdgeNotFound if missing.
//
// Errors:
//   - ErrEdgeNotFound: if edgeID is not present in the edge catalog.
//
// Determinism:
//   - Deterministic for a fixed graph state (pure map membership check).
//
// Complexity:
//   - Time O(1) average, Space O(1).
//
// Notes:
//   - The returned pointer aliases the live catalog object.
//   - Edge.ID, Edge.From, Edge.To, Edge.Weight, and Edge.Directed MUST be treated
//     as immutable once the edge is published in the graph.
//   - Retaining the pointer does not pin membership and does not extend graph locks.
//   - If detached mutable ownership is required, copy the Edge value externally.
//
// AI-Hints:
//   - Use errors.Is(err, ErrEdgeNotFound) to branch without string matching.
//   - Prefer GetEdge over scanning Edges() when you already have the ID.
func (g *Graph) GetEdge(edgeID string) (*Edge, error) {
	if edgeID == "" { // validate edgeID
		return nil, ErrEmptyEdgeID
	}
	// AI-HINT: Use errors.Is(err, ErrEdgeNotFound) to gate fallbacks; returned *Edge is read-only by convention.
	g.muEdgeAdj.RLock()         // lock edges/adjacency map for a consistent snapshot
	defer g.muEdgeAdj.RUnlock() // ensure unlock on all paths
	e, ok := g.edges[edgeID]    // attempt to find edge by its unique ID
	if !ok {                    // if not found, return the canonical sentinel
		return nil, ErrEdgeNotFound
	}

	return e, nil // happy path: return read-only pointer to the cataloged edge
}

// Edges returns all edges sorted by Edge.ID ascending (stable, deterministic order).
//
// Implementation:
//   - Stage 1: Acquire muEdgeAdj read lock for a stable catalog snapshot.
//   - Stage 2: Copy all *Edge pointers into a pre-sized slice.
//   - Stage 3: Sort the slice by Edge.ID ascending.
//   - Stage 4: Return the sorted slice.
//
// Behavior highlights:
//   - Deterministic ordering independent of Go map iteration order.
//   - Returns pointers to live catalog edges (read-only by convention).
//
// Inputs:
//   - None.
//
// Returns:
//   - []*Edge: all edges sorted by Edge.ID ascending.
//
// Errors:
//   - None (pure query).
//
// Determinism:
//   - Deterministic order by contract: Edge.ID ascending.
//
// Complexity:
//   - Time O(E log E), Space O(E).
//
// Notes:
//   - Prefer EdgeCount() when you need only counts (O(1) vs O(E log E)).
//   - The returned slice is newly allocated; callers may retain and reorder it safely.
//   - Do not mutate the returned *Edge objects.
//
// AI-Hints:
//   - Use Edges() for stable logs/golden outputs and deterministic diffing.
//   - If you need only IDs, consider extracting IDs and comparing sorted slices in tests.
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

// EdgeCount returns the current number of edges in the graph.
//
// Implementation:
//   - Stage 1: Acquire muEdgeAdj read lock for a consistent catalog snapshot.
//   - Stage 2: Return len(g.edges).
//
// Behavior highlights:
//   - O(1) fast-path.
//   - No allocations.
//   - Independent of map iteration order.
//
// Inputs:
//   - None.
//
// Returns:
//   - int: number of edges currently present.
//
// Errors:
//   - None (pure query).
//
// Determinism:
//   - Deterministic for a fixed graph state.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Counts the edge catalog entries; undirected edges still count as 1 edge.
//
// AI-Hints:
//   - Prefer EdgeCount() over len(Edges()) to avoid O(E log E) sorting cost.
func (g *Graph) EdgeCount() int {
	// AI-HINT: O(1) size of edge catalog; does not allocate.
	g.muEdgeAdj.RLock()
	defer g.muEdgeAdj.RUnlock()

	return len(g.edges)
}

// HasDirectedEdges reports whether at least one edge with Directed == true exists.
//
// Implementation:
//   - Stage 1: Acquire muEdgeAdj read lock for a stable catalog snapshot.
//   - Stage 2: Scan edge catalog and return true on first directed edge.
//   - Stage 3: If none found, return false.
//
// Behavior highlights:
//   - Early-exit scan (best-case O(1) if a directed edge is found early).
//   - Does not allocate.
//
// Inputs:
//   - None.
//
// Returns:
//   - bool: true if any directed edge exists; otherwise false.
//
// Errors:
//   - None (pure query).
//
// Determinism:
//   - Deterministic for a fixed graph state (existence check).
//
// Complexity:
//   - Time O(E) worst-case, Space O(1).
//
// Notes:
//   - In mixed-mode graphs, this can be true even if the default orientation is undirected.
//
// AI-Hints:
//   - Use HasDirectedEdges() as a cheap gate for algorithms that need directed semantics.
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

// FilterEdges returns detached copies of all edges for which pred returns true.
//
// Implementation:
//   - Stage 1: Reject nil predicate with ErrNilEdgePredicate.
//   - Stage 2: Acquire muEdgeAdj.RLock() for a stable edge-catalog snapshot.
//   - Stage 3: Scan the edge catalog and pass detached Edge values to pred.
//   - Stage 4: Append matching copies to the result slice.
//   - Stage 5: Sort returned copies by Edge.ID ascending.
//
// Behavior highlights:
//   - Read-only query; does not mutate edge catalog or adjacency index.
//   - Predicate receives detached Edge values, not live catalog pointers.
//   - Returned slice and Edge values are caller-owned copies.
//
// Inputs:
//   - pred: non-nil pure predicate over an Edge value copy.
//
// Returns:
//   - []Edge: matching detached edge copies sorted by Edge.ID ascending.
//   - error: nil on success; otherwise a sentinel error.
//
// Errors:
//   - ErrNilEdgePredicate: if pred == nil.
//
// Determinism:
//   - Output order is deterministic: Edge.ID ascending.
//   - Predicate call order is not part of the contract.
//
// Complexity:
//   - Time O(E log E) due to final sorting, Space O(k) for k matches.
//
// Notes:
//   - Use RemoveEdgesWhere for mutating bulk deletion.
//
// AI-Hints:
//   - Do not reintroduce destructive filtering under this name; FilterEdges is a query surface.
//   - If caller needs pointers, they can GetEdge(match.ID) explicitly after filtering.
func (g *Graph) FilterEdges(pred func(Edge) bool) ([]Edge, error) {
	if pred == nil {
		return nil, ErrNilEdgePredicate
	}

	g.muEdgeAdj.RLock()
	defer g.muEdgeAdj.RUnlock()

	out := make([]Edge, 0)
	var value Edge
	for _, e := range g.edges {
		if e.IsNil() {
			continue
		}
		value = *e
		if pred(value) {
			out = append(out, value)
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })

	return out, nil
}

// RemoveEdgesWhere removes every edge for which pred returns true.
//
// Implementation:
//   - Stage 1: Reject nil predicate with ErrNilEdgePredicate.
//   - Stage 2: Acquire muEdgeAdj write lock for an atomic removal pass.
//   - Stage 3: Scan the edge catalog once.
//   - Stage 4: Pass each predicate a detached Edge value copy.
//   - Stage 5: For matching edges, remove adjacency and delete from the catalog.
//   - Stage 6: Cleanup empty adjacency buckets once after the bulk mutation.
//
// Behavior highlights:
//   - Removes matching edges; keeps non-matching edges.
//   - The predicate receives a value copy, so it cannot mutate cataloged edge fields.
//   - The graph remains catalog/adjacency-consistent after every successful call.
//   - Predicate execution occurs while muEdgeAdj is held; predicates must be pure and must
//     not call graph methods or try to mutate the graph.
//
// Inputs:
//   - pred: non-nil pure predicate over a detached Edge value.
//
// Returns:
//   - int: number of removed edges.
//   - error: nil on success; otherwise a sentinel error.
//
// Errors:
//   - ErrNilEdgePredicate: if pred == nil.
//
// Determinism:
//   - Deterministic for a deterministic predicate; map scan order does not affect the final set.
//
// Complexity:
//   - Time O(E + B), where B is adjacency cleanup bucket count.
//   - Space O(1) extra.
//
// Notes:
//   - This method is the preferred value-copy bulk-removal API.
//   - Public AdjacencyList() still reports isolated vertices after their last edge is removed.
//
// AI-Hints:
//   - Use RemoveEdgesWhere for contract-safe bulk deletion.
//   - Do not call g.Edges, g.Neighbors, AddEdge, RemoveEdge, or RemoveVertex from pred.
func (g *Graph) RemoveEdgesWhere(pred func(Edge) bool) (int, error) {
	if pred == nil {
		return 0, ErrNilEdgePredicate
	}

	g.muEdgeAdj.Lock()
	defer g.muEdgeAdj.Unlock()

	removed := 0
	var value Edge
	for eid, e := range g.edges {
		if e.IsNil() {
			delete(g.edges, eid)
			removed++
			continue
		}

		value = *e
		if pred(value) {
			removeAdjacency(g, e)
			delete(g.edges, eid)
			removed++
		}
	}

	cleanupAdjacency(g)

	return removed, nil
}

// ValidateEdgeID checks whether ID is non-empty and not already in use.
//
// Implementation:
//   - Stage 1: Reject empty IDs (ErrEmptyEdgeID).
//   - Stage 2: Under muEdgeAdj read lock, check edge catalog membership (ErrEdgeIDConflict).
//
// Inputs:
//   - ID: desired edge identifier.
//
// Returns:
//   - error: nil if valid; otherwise a sentinel.
//
// Errors:
//   - ErrEmptyEdgeID: if ID == "".
//   - ErrEdgeIDConflict: if ID already exists.
//
// Complexity:
//   - Time O(1), Space O(1).
func (g *Graph) ValidateEdgeID(ID string) error {
	if ID == "" {
		return ErrEmptyEdgeID
	}

	g.muEdgeAdj.RLock()
	defer g.muEdgeAdj.RUnlock()
	if _, exist := g.edges[ID]; exist {
		return ErrEdgeIDConflict
	}

	return nil
}

// nextEdgeID returns a new unique canonical auto edge ID "eN".
//
// Implementation:
//   - Stage 1: Atomically increment g.nextEdgeID and capture the new value N.
//   - Stage 2: Build the string "e" + base-10 digits without fmt.
//
// Returns:
//   - string: canonical auto edge ID.
//
// Determinism:
//   - Monotonic counter; no randomness.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// NOTES:
//   - Safe for concurrent callers; atomic.AddUint64 is used to fetch the next number.
func nextEdgeID(g *Graph) string {
	// AI-HINT: Monotonic textual IDs ("e1","e2",...); Clone carries sequence to keep continuity.
	n := atomic.AddUint64(&g.nextEdgeID, 1) // atomically reserve the next sequence number
	buf := make([]byte, 0, 1+20)            // "e" + up to 20 digits for uint64
	buf = append(buf, edgeIDPrefix)         // textual prefix
	buf = strconv.AppendUint(buf, n, 10)    // base-10 digits

	return string(buf) // convert to immutable string
}

// matchesAutoIDPattern reports whether id is a canonical auto-generated edge ID of the form "eN".
//
// Canonical form rules:
//   - Prefix must be 'e'.
//   - Digits must be base-10.
//   - No leading zeros ("e01" is NOT canonical).
//   - N must be >= 1.
//
// Returns:
//   - (uint64, bool): parsed N and true if canonical; otherwise (0,false).
//
// Determinism:
//   - Pure parsing; deterministic.
//
// Complexity:
//   - Time O(len(id)), Space O(1).
func matchesAutoIDPattern(id string) (uint64, bool) {
	// Require at least "e" + 1 digit.
	if len(id) < 2 {
		return 0, false
	}

	// Require canonical prefix.
	if id[0] != edgeIDPrefix {
		return 0, false
	}

	// Reject leading zeros to keep canonical form strict and unambiguous.
	if id[1] == '0' {
		return 0, false
	}

	// Parse base-10 digits.
	n, err := strconv.ParseUint(id[1:], 10, 64)
	if err != nil || n == 0 {
		return 0, false
	}

	return n, true
}
