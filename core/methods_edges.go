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

// AddEdge creates a new edge (from -> to) with an optional weight and per-edge options.
// The method enforces graph capabilities (weighted/multi/loops/mixed) and updates adjacency.
//
// Implementation:
//   - Stage 1: Validate inputs against graph capabilities (empty IDs, weight policy, loop policy).
//   - Stage 2: Ensure endpoints exist (AddVertex).
//   - Stage 3: Under muEdgeAdj write lock:
//   - enforce multi-edge policy,
//   - construct a baseline Edge (defaults + endpoints + weight),
//   - apply EdgeOptions in call order (first error aborts),
//   - assign an auto-ID if no explicit ID was provided,
//   - store in edge catalog and update adjacency (mirror if undirected).
//
// Behavior highlights:
//   - Strict sentinel errors only (errors.Is).
//   - No panics as part of the public contract.
//   - Deterministic behavior given deterministic inputs.
//
// Inputs:
//   - from: source vertex ID (must be non-empty).
//   - to: destination vertex ID (must be non-empty).
//   - weight: edge weight; must be 0 unless WithWeighted() was set.
//   - opts: optional per-edge mutators (WithID, WithEdgeDirected, ...).
//
// Returns:
//   - string: the assigned Edge.ID (auto-generated or explicit).
//   - error: nil on success; otherwise a sentinel error.
//
// Errors:
//   - ErrEmptyVertexID: if from == "" or to == "".
//   - ErrBadWeight: if weight != 0 on an unweighted graph.
//   - ErrLoopNotAllowed: if from == to and loops are disabled.
//   - ErrMultiEdgeNotAllowed: if a parallel edge is attempted and multi-edges are disabled.
//   - ErrMixedEdgesNotAllowed: if a directedness override is attempted without mixed mode.
//   - ErrEmptyEdgeID / ErrEdgeIDConflict: from WithID / ValidateEdgeID rules.
//
// Determinism:
//   - Option application order is stable (call order).
//   - Auto-ID assignment uses a monotonic counter (no randomness, no time).
//
// Complexity:
//   - Time O(1) amortized (hash-map + nested-map updates + O(len(opts)) option calls).
//   - Space O(1) amortized.
//
// Notes:
//   - Vertices are ensured before the edge lock to avoid lock inversion with vertex mutations.
//
// AI-Hints:
//   - Use WithID for stable cross-run edge references.
//   - For undirected edges, adjacency is mirrored automatically.
func (g *Graph) AddEdge(from, to string, weight float64, opts ...EdgeOption) (string, error) {
	// Input validation
	if from == "" || to == "" {
		return "", ErrEmptyVertexID
	}
	if !g.weighted && weight != 0 { // weight constraint
		return "", ErrBadWeight
	}
	if from == to && !g.allowLoops { // loop constraint
		return "", ErrLoopNotAllowed
	}

	// Ensure vertices exist
	if err := g.AddVertex(from); err != nil {
		return "", err
	}
	if err := g.AddVertex(to); err != nil {
		return "", err
	}

	// Insert edge under lock
	g.muEdgeAdj.Lock()
	defer g.muEdgeAdj.Unlock()

	// Build baseline edge (ID assigned later)
	e := &Edge{From: from, To: to, Weight: weight, Directed: g.directed}

	if !g.allowMulti { // Multi-edge existence check
		if inner := g.adjacencyList[from][to]; len(inner) > 0 {
			return "", ErrMultiEdgeNotAllowed
		}
	}

	// Apply EdgeOptions deterministically in call order.
	//    Any option error aborts the operation without mutating graph catalogs.
	for _, opt := range opts {
		if err := opt(g, e); err != nil {
			return "", err
		}
	}

	// Re-check loops in case WithEdgeDirected changed nothing here,
	//    but best to keep the guard in case future options interfere (future options must not bypass loop policy).
	if e.From == e.To && !g.allowLoops {
		return "", ErrLoopNotAllowed
	}

	// Assign edge ID:
	//    - If WithID was used, e.ID is already set and validated for uniqueness.
	//    - Otherwise, generate a new unique textual edge ID in O(1) without fmt allocations.
	if e.ID == "" {
		e.ID = nextEdgeID(g)
	} else {
		// Final defensive collision check (covers future options that may set IDs).
		if _, exists := g.edges[e.ID]; exists {
			return "", ErrEdgeIDConflict
		}
		if num, ok := matchesAutoIDPattern(e.ID); ok && num >= 0 {
			bumpNextEdgeIDToAtLeast(g, num)
		}
	}

	// Store edge and update adjacency.
	g.edges[e.ID] = e
	ensureAdjacency(g, from, to)
	g.adjacencyList[from][to][e.ID] = struct{}{}

	// Mirror undirected edges (excluding loops).
	if !e.Directed && from != to {
		ensureAdjacency(g, to, from)
		g.adjacencyList[to][from][e.ID] = struct{}{}
	}

	return e.ID, nil
}

// RemoveEdge deletes one edge by ID and removes its adjacency references.
//
// Implementation:
//   - Stage 1: Acquire muEdgeAdj write lock.
//   - Stage 2: Lookup edge in catalog; if absent return ErrEdgeNotFound.
//   - Stage 3: Delete from catalog, remove adjacency references, cleanup empty buckets.
//
// Inputs:
//   - eid: edge identifier.
//
// Returns:
//   - error: nil on success; ErrEdgeNotFound if missing.
//
// Errors:
//   - ErrEdgeNotFound: if no edge with eid exists.
//
// Determinism:
//   - Deterministic; pure map membership + deletions.
//
// Complexity:
//   - Time O(1) average for catalog + adjacency deletions (cleanup may scan buckets).
//   - Space O(1).
//
// AI-Hints:
//   - Removing an absent edge returns ErrEdgeNotFound (no silent ignore).
func (g *Graph) RemoveEdge(eid string) error {
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
	// No-op rename is allowed.
	if oldID == newID {
		return nil
	}

	// AI-HINT: This operation mutates edge catalog and adjacency maps; it MUST use the write lock.
	g.muEdgeAdj.Lock()         // exclusive lock: protects map writes (g.edges and adjacencyList buckets)
	defer g.muEdgeAdj.Unlock() // ensure unlock on all paths
	e, exist := g.edges[oldID] // attempt to find edge by its unique ID
	if !exist {                // if not found, return the canonical sentinel
		return ErrEdgeNotFound
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
	if num, ok := matchesAutoIDPattern(newID); ok && num >= 0 {
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
//   - safe for concurrent use; acquires muEdgeAdj read lock.
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
//   - The returned pointer refers to the live catalog object.
//     Callers MUST treat it as immutable; mutation would break graph invariants.
//   - If you need a stable snapshot object for external ownership, copy the Edge value.
//
// AI-Hints:
//   - Use errors.Is(err, ErrEdgeNotFound) to branch without string matching.
//   - Prefer GetEdge over scanning Edges() when you already have the ID.
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

// FilterEdges removes all edges failing the predicate.
//
// Implementation:
//   - Stage 1: Acquire muEdgeAdj write lock.
//   - Stage 2: Scan edge catalog; for edges where pred(e)==false remove adjacency + delete from catalog.
//   - Stage 3: Cleanup adjacency buckets.
//
// Inputs:
//   - pred: pure predicate; MUST NOT mutate the graph.
//
// Returns:
//   - None.
//
// Determinism:
//   - Deterministic for deterministic pred; scan order does not affect final set.
//
// Complexity:
//   - Time O(E) average + cleanup cost, Space O(1) extra.
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
