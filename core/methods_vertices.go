// File: methods_vertices.go
// Role: Vertex lifecycle & queries.
// Determinism:
//   - Vertices() returns IDs sorted lex asc.
// Concurrency:
//   - Vertex catalog protected by muVert; adjacency bootstrap under muEdgeAdj.
// AI-HINT (file):
//   - Vertices() returns sorted IDs (lex asc).
//   - Degree() policy: directed edges contribute to in/out; undirected to undirected; self-loop rules documented below.

package core

import "sort"

// AddVertex inserts a vertex if missing (idempotent).
//
// Steps:
//  1. Validate non-empty ID (ErrEmptyVertexID).
//  2. Under muVert, check presence; if missing, allocate Vertex and register it.
//  3. Under muEdgeAdj, lazy-init adjacencyList[id] to keep invariants consistent.
//
// Complexity: O(1) amortized.
// Concurrency: muVert write lock (catalog), then muEdgeAdj write lock (bootstrap).
func (g *Graph) AddVertex(id string) error {
	// AI-HINT: Empty ID returns ErrEmptyVertexID. Idempotent if vertex exists.

	// Validate input: empty IDs are not allowed
	if id == "" {
		return ErrEmptyVertexID
	}
	g.muVert.Lock()
	defer g.muVert.Unlock()

	// Check if vertex already present
	if _, exists := g.vertices[id]; exists {
		return nil // no-op for existing vertex
	}
	// Insert new Vertex struct with empty Metadata map
	g.vertices[id] = &Vertex{ID: id, Metadata: make(map[string]interface{})}

	// ensure top-level adjacency map exists
	g.muEdgeAdj.Lock()
	ensureAdjacency(g, id, id)
	g.muEdgeAdj.Unlock()

	return nil
}

// HasVertex returns true if the vertex ID exists (empty ID ⇒ false).
// Complexity: O(1). Concurrency: muVert read lock.
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

// RemoveVertex deletes the vertex and all incident edges (directed and undirected).
//
// Steps:
//  1. Validate non-empty ID.
//  2. Acquire muVert + muEdgeAdj write locks.
//  3. Scan edges once; for each incident edge call removeAdjacency and delete from catalog.
//  4. Delete vertex from catalog and prune empty adjacency buckets.
//
// Complexity: O(E + V) worst-case.
// Concurrency: requires both write locks; not safe concurrently with readers.
func (g *Graph) RemoveVertex(id string) error {
	// AI-HINT: Removes all incident edges deterministically; missing vertex → ErrVertexNotFound.
	if id == "" {
		return ErrEmptyVertexID
	}
	// Lock vertices and edges+adjacency to prevent races
	g.muVert.Lock()
	defer g.muVert.Unlock()
	g.muEdgeAdj.Lock()
	defer g.muEdgeAdj.Unlock()

	// Verify vertex presence
	if _, exists := g.vertices[id]; !exists {
		return ErrVertexNotFound
	}

	// Remove all incident edges
	var eid string
	var e *Edge
	for eid, e = range g.edges {
		if e.From == id || (!e.Directed && e.To == id) || (e.Directed && e.To == id) {
			removeAdjacency(g, e) // remove both directions appropriately
			delete(g.edges, eid)
		}
	}

	// Delete vertex itself
	delete(g.vertices, id)
	// prune any empty nested maps
	cleanupAdjacency(g)

	return nil
}

// Vertices returns all vertex IDs in lexicographic ascending order.
// Determinism: stable enumeration order (algorithms rely on it).
// Complexity: O(V log V). Concurrency: muVert read lock.
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

// VertexCount returns total number of vertices.
// Complexity: O(1). Concurrency: muVert read lock.
func (g *Graph) VertexCount() int {
	// AI-HINT: O(1) size of vertex catalog; does not allocate.
	g.muVert.RLock()
	defer g.muVert.RUnlock()

	return len(g.vertices)
}

// VerticesMap returns a shallow copy of the vertex catalog (ID -> *Vertex).
// Notes:
//   - Pointers refer to live Vertex objects; treat them as read-only by convention.
//
// Complexity: O(V). Concurrency: muVert read lock.
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

// InternalVertices returns the *live* internal vertices map.
//
// Deprecated: This function exposes the internal storage without synchronization
// guarantees for callers and allows mutation that bypasses graph invariants.
// It remains for legacy tests only. Prefer Vertices() / VerticesMap() / views.
func (g *Graph) InternalVertices() map[string]*Vertex {
	// AI-HINT: Deprecated live map; do not mutate in user code. Prefer Vertices()/VerticesMap().
	return g.vertices
}

//–– Additional methods –––––––––––––––––––––––––––––––––––––––––––––––––––––

// Degree returns the degree components of the given vertex ID:
//
//	in         — number of incoming directed edges (e.To == id)
//	out        — number of outgoing directed edges (e.From == id)
//	undirected — contribution from undirected edges
//
// Academic policy (documented in doc.go):
//   - Directed edges contribute to in/out only.
//   - Undirected edges contribute to undirected only.
//   - Directed self-loop (id -> id) contributes +1 to both in and out.
//   - Undirected self-loop contributes +2 to undirected (standard graph-theory convention).
//
// Determinism:
//   - Iteration relies on Neighbors(id), which returns edges sorted by Edge.ID asc.
//
// Complexity: O(d) where d is the degree of id (including mirrored undirected entries).
// Concurrency: safe; acquires read locks inside Neighbors.
func (g *Graph) Degree(id string) (in, out, undirected int, err error) {
	// AI-HINT:
	//   - Directed self-loop contributes +1 to in and +1 to out.
	//   - Undirected self-loop contributes +2 to undirected (classic theory).
	//   - Uses Neighbors(id); deterministic iteration by Edge.ID asc.
	edges, err := g.Neighbors(id)
	if err != nil {
		return 0, 0, 0, err
	}
	var e *Edge
	for _, e = range edges {
		if e.Directed {
			if e.From == id && e.To == id {
				// Directed self-loop counts as both incoming and outgoing once.
				in++
				out++
				continue
			}
			if e.From == id {
				out++
				continue
			}
			if e.To == id {
				in++
				continue
			}
			// Should not happen; neighbors() ensures relevance.
			continue
		}
		// Undirected edges:
		if e.From == id && e.To == id {
			// Undirected self-loop increases degree by 2 in classic theory.
			undirected += 2
		} else {
			undirected++
		}
	}

	return in, out, undirected, nil
}
