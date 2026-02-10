// File: methods_vertices.go
// Role: Vertex lifecycle & queries.
//
// Determinism:
//   - Vertices() returns IDs sorted lexicographically ascending.
//
// Concurrency:
//   - Vertex catalog protected by muVert.
//   - Adjacency bootstrap under muEdgeAdj (to keep adjacency invariants consistent).
//
// AI-Hints (file):
//   - Vertices() is a stable enumeration surface; rely on it for reproducible outputs.
//   - Degree() follows documented academic policy for loops and directedness.
package core

import "sort"

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
//   - Implement the same pattern for other pointer-backed core types that appear behind interfaces.
func (v *Vertex) IsNil() bool { return v == nil }

// AddVertex inserts a vertex if missing (idempotent).
//
// Implementation:
//   - Stage 1: Validate non-empty ID (ErrEmptyVertexID).
//   - Stage 2: Under muVert write lock, check presence; if missing, allocate Vertex and register it.
//   - Stage 3: Under muEdgeAdj write lock, bootstrap adjacency buckets so edge methods can rely on invariants.
//
// Behavior highlights:
//   - Idempotent: adding an existing vertex is a no-op.
//   - Initializes Metadata map to a non-nil value for convenience in tests and algorithms.
//
// Inputs:
//   - id: vertex identifier; must be non-empty.
//
// Returns:
//   - error: nil on success; ErrEmptyVertexID on invalid input.
//
// Errors:
//   - ErrEmptyVertexID: if id == "".
//
// Determinism:
//   - Deterministic map membership logic; does not depend on iteration order.
//
// Complexity:
//   - Time O(1) amortized, Space O(1) amortized.
//
// Notes:
//   - Lock order is muVert -> muEdgeAdj to avoid lock inversion across vertex/edge code paths.
//   - The adjacency bootstrap is intentionally minimal; it does not create any edges.
//
// AI-Hints:
//   - Prefer AddVertex in setup when you want explicit vertex presence before adding edges.
func (g *Graph) AddVertex(id string) error {
	// AI-HINT: Empty ID returns ErrEmptyVertexID. Idempotent if vertex exists.
	if id == "" {
		return ErrEmptyVertexID
	}

	// Stage 2: Register in the vertex catalog under muVert.
	g.muVert.Lock()
	defer g.muVert.Unlock()

	// Check if vertex already present
	if _, exists := g.vertices[id]; exists {
		return nil // no-op for existing vertex
	}

	// Allocate a new vertex record; Metadata is initialized to a non-nil map by policy.
	g.vertices[id] = &Vertex{ID: id, Metadata: make(map[string]interface{})}

	// Stage 3: Bootstrap adjacency buckets under muEdgeAdj.
	// This preserves adjacency invariants for later edge operations.
	g.muEdgeAdj.Lock()
	ensureAdjacency(g, id, id)
	g.muEdgeAdj.Unlock()

	return nil
}

// HasVertex reports whether the vertex ID exists (empty ID ⇒ false).
//
// Implementation:
//   - Stage 1: Reject empty ID (fast-path).
//   - Stage 2: Acquire muVert read lock and check catalog membership.
//
// Inputs:
//   - id: vertex identifier.
//
// Returns:
//   - bool: true iff the vertex is present.
//
// Errors:
//   - None (pure query).
//
// Determinism:
//   - Deterministic; map membership only.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// AI-Hints:
//   - Use HasVertex as a cheap admission check before operations that do not auto-create vertices.
func (g *Graph) HasVertex(id string) bool {
	// AI-HINT: O(1) membership on vertex catalog; empty id → false.
	if id == "" {
		return false
	}
	// Acquire read lock on vertices
	g.muVert.RLock()
	defer g.muVert.RUnlock()
	_, ok := g.vertices[id]

	return ok
}

// RemoveVertex deletes a vertex and all incident edges (directed and undirected).
//
// Implementation:
//   - Stage 1: Validate non-empty ID (ErrEmptyVertexID).
//   - Stage 2: Acquire muVert write lock and muEdgeAdj write lock for an atomic topology update.
//   - Stage 3: Verify vertex presence (ErrVertexNotFound).
//   - Stage 4: Scan edge catalog once; for each incident edge remove adjacency references and delete from catalog.
//   - Stage 5: Delete vertex from catalog and cleanup adjacency buckets.
//
// Behavior highlights:
//   - Removes all incident edges deterministically.
//   - Leaves the graph in a consistent state (no dangling adjacency references).
//
// Inputs:
//   - id: vertex identifier to remove; must be non-empty.
//
// Returns:
//   - error: nil on success; otherwise a sentinel.
//
// Errors:
//   - ErrEmptyVertexID: if id == "".
//   - ErrVertexNotFound: if the vertex does not exist.
//
// Determinism:
//   - Deterministic for a fixed graph state; does not depend on iteration order for correctness.
//
// Complexity:
//   - Time O(E) for scanning the edge catalog (+ cleanup cost), Space O(1) extra.
//
// Notes:
//   - This method is intentionally “heavy”: removing a vertex is a topology rewrite.
//   - Requires both write locks; concurrent readers are blocked until completion.
//
// AI-Hints:
//   - Prefer building subgraphs/views when you need “logical removal” without mutating the original.
func (g *Graph) RemoveVertex(id string) error {
	// AI-HINT: Removes all incident edges deterministically; missing vertex → ErrVertexNotFound.
	if id == "" {
		return ErrEmptyVertexID
	}

	// Acquire both locks for atomic removal of vertex + incident edges.
	g.muVert.Lock()
	defer g.muVert.Unlock()

	g.muEdgeAdj.Lock()
	defer g.muEdgeAdj.Unlock()

	// Verify vertex presence
	if _, exists := g.vertices[id]; !exists {
		return ErrVertexNotFound
	}

	// Remove all incident edges (directed or undirected).
	var eid string
	var e *Edge
	for eid, e = range g.edges {
		if e.From == id || e.To == id {
			removeAdjacency(g, e)
			delete(g.edges, eid)
		}
	}

	// Delete the vertex record and cleanup adjacency buckets.
	delete(g.vertices, id)
	// prune any empty nested maps
	cleanupAdjacency(g)

	return nil
}

// Vertices returns all vertex IDs in lexicographic ascending order.
//
// Implementation:
//   - Stage 1: Acquire muVert read lock and copy vertex IDs into a slice.
//   - Stage 2: Sort the slice ascending.
//   - Stage 3: Return the sorted slice.
//
// Behavior highlights:
//   - Stable enumeration surface used for determinism in higher-level algorithms.
//
// Inputs:
//   - None.
//
// Returns:
//   - []string: sorted vertex IDs.
//
// Errors:
//   - None (pure query).
//
// Determinism:
//   - Deterministic output order (lex asc).
//
// Complexity:
//   - Time O(V log V), Space O(V).
//
// AI-Hints:
//   - Use Vertices() for reproducible traversal seeds and stable test assertions.
func (g *Graph) Vertices() []string {
	// AI-HINT: Deterministic ordering by ID asc; rely on it for stable diffs.
	g.muVert.RLock()
	defer g.muVert.RUnlock()

	ids := make([]string, 0, len(g.vertices))
	var id string
	for id = range g.vertices {
		ids = append(ids, id)
	}

	sort.Strings(ids)

	return ids
}

// VertexCount returns the current number of vertices in the graph.
//
// Implementation:
//   - Stage 1: Acquire muVert read lock.
//   - Stage 2: Return len(g.vertices).
//
// Returns:
//   - int: number of vertices.
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
// AI-Hints:
//   - Prefer VertexCount() over len(Vertices()) to avoid O(V log V) sorting costs.
func (g *Graph) VertexCount() int {
	// AI-HINT: O(1) size of vertex catalog; does not allocate.
	g.muVert.RLock()
	defer g.muVert.RUnlock()

	return len(g.vertices)
}

// VerticesMap returns a shallow copy of the vertex catalog (ID -> *Vertex).
//
// Implementation:
//   - Stage 1: Acquire muVert read lock.
//   - Stage 2: Allocate a new map sized to the catalog.
//   - Stage 3: Copy ID -> *Vertex entries into the new map.
//
// Behavior highlights:
//   - Callers can retain the returned map without holding graph locks.
//   - Vertex pointers refer to live objects; treat them as read-only by convention.
//
// Returns:
//   - map[string]*Vertex: shallow copy of the catalog.
//
// Errors:
//   - None (pure query).
//
// Determinism:
//   - Deterministic for membership; returned map iteration order is not deterministic (Go map rule).
//
// Complexity:
//   - Time O(V), Space O(V).
//
// Notes:
//   - Use Vertices() when you need deterministic ordering.
//
// AI-Hints:
//   - Prefer VerticesMap() for membership checks that need a stable snapshot without sorting.
func (g *Graph) VerticesMap() map[string]*Vertex {
	// AI-HINT: Returns a shallow copy (ID → *Vertex); safe to retain by callers.
	g.muVert.RLock()
	defer g.muVert.RUnlock()

	out := make(map[string]*Vertex, len(g.vertices))
	var id string
	var v *Vertex
	for id, v = range g.vertices {
		out[id] = v
	}

	return out
}

// InternalVertices returns the live internal vertices map.
//
// Deprecated:
//   - This exposes internal storage without synchronization guarantees and enables mutation
//     that bypasses graph invariants. It remains for legacy tests only.
//
// AI-Hints:
//   - Prefer Vertices(), VerticesMap(), or higher-level views instead of mutating internals.
func (g *Graph) InternalVertices() map[string]*Vertex {
	// AI-HINT: Deprecated live map; do not mutate in user code. Prefer Vertices()/VerticesMap().
	return g.vertices
}

// Degree returns the degree components of the given vertex ID:
//
//   - in: number of incoming directed edges (e.To == id)
//   - out: number of outgoing directed edges (e.From == id)
//   - undirected: contribution from undirected edges
//
// Academic policy:
//   - Directed edges contribute to in/out only.
//   - Undirected edges contribute to undirected only.
//   - Directed self-loop (id -> id) contributes +1 to both in and out.
//   - Undirected self-loop contributes +2 to undirected (classic graph-theory convention).
//
// Implementation:
//   - Stage 1: Validate id and vertex existence under locks.
//   - Stage 2: Scan ALL graph edges (g.edges) to identify incident connections.
//     This is necessary because the standard adjacency list is optimized for
//     outgoing edges and does not efficiently index incoming directed edges.
//   - Stage 3: Accumulate counters based on connectivity and direction.
//
// Inputs:
//   - id: vertex identifier.
//
// Returns:
//   - in: directed in-degree component.
//   - out: directed out-degree component.
//   - undirected: undirected degree component.
//   - err: vertices working errors (ErrEmptyVertexID or ErrVertexNotFound).
//
// Errors:
//   - ErrEmptyVertexID: if id is empty.
//   - ErrVertexNotFound: if the vertex does not exist in the graph.
//
// Determinism:
//   - Deterministic result (counting is order-independent).
//
// Complexity:
//   - Time O(E), Space O(1), where E is the total number of edges in the graph.
//     Note: This is an O(E) operation to ensure correct in-degree calculation without
//     maintaining a separate reverse index.
//
// Notes:
//   - This method acquires global read locks on vertices and edges.
//
// AI-Hints:
//   - Use Degree() when you need loop-aware, policy-defined degree semantics.
//   - Be aware of O(E) cost on very large graphs.
func (g *Graph) Degree(id string) (in, out, undirected int, err error) {
	// AI-HINT:
	//   - Directed self-loop contributes +1 to in and +1 to out.
	//   - Undirected self-loop contributes +2 to undirected.

	if id == "" {
		return 0, 0, 0, ErrEmptyVertexID
	}

	// Acquire locks in the same order as other methods (muVert -> muEdgeAdj)
	// to ensure consistent view and avoid deadlocks.
	g.muVert.RLock()
	defer g.muVert.RUnlock()

	g.muEdgeAdj.RLock()
	defer g.muEdgeAdj.RUnlock()

	// Validate vertex existence strictly
	if _, ok := g.vertices[id]; !ok {
		return 0, 0, 0, ErrVertexNotFound
	}

	// Iterate over ALL edges to correctly capture incoming directed edges (in-degree),
	// which are not present in the standard Neighbors() (outgoing-only) view.
	for _, e := range g.edges {
		// Safety check against nil edges in map (defensive)
		if e.IsNil() {
			continue
		}

		// Check connectivity for directed and undirected edge
		isFrom := e.From == id
		isTo := e.To == id
		if !isFrom && !isTo {
			continue
		}
		if e.Directed {
			if isFrom {
				out++
			}
			if isTo {
				in++
			}
			// Note: A directed self-loop (id->id) triggers both checks,
			// correctly incrementing both 'in' and 'out'.
		} else {
			// Undirected logic
			if e.From == id && e.To == id {
				// Undirected self-loop increases degree by 2 in classic theory.
				undirected += 2
			} else {
				// Standard incidence
				undirected++
			}
		}
	}

	return in, out, undirected, nil
}
