// File: methods_adjacent.go
// Role: Neighborhood APIs (Neighbors, NeighborIDs, AdjacencyList) and adjacency helpers.
// Determinism:
//   - Neighbors() sorts by Edge.ID asc.
//   - NeighborIDs() returns unique IDs sorted lex asc.
//   - AdjacencyList() returns per-vertex edgeID slices sorted by Edge.ID asc.
// Concurrency:
//   - Read operations hold muVert or muEdgeAdj read locks as needed.
//   - Helpers are called only under appropriate write locks by mutating code.
// AI-HINT (file):
//   - Neighbors(id): directed edges included only if e.From==id; undirected appear once; result sorted by Edge.ID asc.
//   - NeighborIDs(id): unique, sorted (lex asc).
//   - AdjacencyList(): per-vertex edgeID slices sorted by Edge.ID asc; returned slices are independent (no shared backing).

package core

import "sort"

// Neighbors returns all edges incident to the given vertex id under the graph's neighborhood policy.
//
// Neighborhood policy:
//   - Directed edges: include only edges with e.From == id (outgoing edges).
//   - Undirected edges: include incident edges (mirrored adjacency is used); self-loops appear once.
//
// Implementation:
//   - Stage 1: Validate id is non-empty (ErrEmptyVertexID).
//   - Stage 2: Acquire muVert read lock and muEdgeAdj read lock (in that order) for a consistent snapshot.
//   - Stage 3: Validate vertex existence (ErrVertexNotFound).
//   - Stage 4: Collect incident edges by scanning adjacencyList[id] buckets and mapping edge IDs to *Edge.
//   - Stage 5: Sort the result by Edge.ID ascending.
//   - Stage 6: Return the sorted slice.
//
// Behavior highlights:
//   - Deterministic ordering by Edge.ID ascending.
//   - Returns pointers to live catalog edges (read-only by convention).
//   - Safe against concurrent vertex removal due to consistent lock ordering.
//
// Inputs:
//   - id: vertex identifier.
//
// Returns:
//   - []*Edge: incident edges under the defined policy, sorted by Edge.ID asc.
//   - error: nil on success; otherwise a sentinel error.
//
// Errors:
//   - ErrEmptyVertexID: if id == "".
//   - ErrVertexNotFound: if the vertex does not exist.
//
// Determinism:
//   - Deterministic order by contract: Edge.ID ascending.
//
// Complexity:
//   - Time O(d log d), Space O(d), where d is the number of incident edges collected.
//
// Notes:
//   - This method does not copy Edge objects; treat returned *Edge as immutable.
//   - Map key order is irrelevant because final ordering is enforced by sorting.
//
// AI-Hints:
//   - Use Neighbors(id) for deterministic iteration in algorithms.
//   - Use NeighborIDs(id) when you need unique adjacent vertex IDs rather than edges.
func (g *Graph) Neighbors(id string) ([]*Edge, error) {
	// AI-HINT: empty id → ErrEmptyVertexID; missing vertex → ErrVertexNotFound.
	//          Deterministic order by Edge.ID asc; treat returned *Edge as read-only.
	if id == "" {
		return nil, ErrEmptyVertexID
	}

	// Acquire locks in the same order as mutators (muVert -> muEdgeAdj) to avoid races
	// where a vertex disappears between validation and adjacency snapshotting.
	g.muVert.RLock()
	defer g.muVert.RUnlock()

	// Lock edges+adjacency for reading
	g.muEdgeAdj.RLock()
	defer g.muEdgeAdj.RUnlock()

	if _, ok := g.vertices[id]; !ok {
		return nil, ErrVertexNotFound
	}

	var out []*Edge
	// Iterate all "to" maps for this vertex

	var eid string
	var e *Edge
	for _, edgeSet := range g.adjacencyList[id] {
		for eid = range edgeSet {
			e = g.edges[eid]

			// Defensive guard: adjacency should not reference missing edges, but keep safety tight.
			if e.IsNil() {
				continue
			}
			//if e == nil { continue }

			// Directed policy: only outgoing edges.
			if e.Directed && e.From != id {
				continue
			}
			// Append pointer directly: no copying
			out = append(out, e)
		}
	}
	// Sort by ID to ensure reproducible ordering
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })

	return out, nil
}

// NeighborIDs returns the unique set of vertex IDs adjacent to id, sorted lexicographically ascending.
//
// Adjacency policy:
//   - For each edge returned by Neighbors(id):
//   - If e.From == id, include e.To.
//   - Else if !e.Directed and e.To == id, include e.From.
//
// Implementation:
//   - Stage 1: Call Neighbors(id) to obtain incident edges and enforce validation.
//   - Stage 2: Build a set of adjacent vertex IDs.
//   - Stage 3: Convert the set to a slice and sort lexicographically.
//   - Stage 4: Return the sorted slice.
//
// Behavior highlights:
//   - Unique output: duplicates are removed.
//   - Deterministic output order (lex asc).
//
// Inputs:
//   - id: vertex identifier.
//
// Returns:
//   - []string: unique adjacent vertex IDs, sorted lex asc.
//   - error: propagated from Neighbors(id).
//
// Errors:
//   - Propagates ErrEmptyVertexID / ErrVertexNotFound from Neighbors(id).
//
// Determinism:
//   - Deterministic output order by contract (lex asc).
//
// Complexity:
//   - Time O(d + k log k), Space O(k), where d is incident edges and k is unique neighbors.
//
// Notes:
//   - For directed edges, only outgoing neighbors are included (consistent with Neighbors policy).
//
// AI-Hints:
//   - Use NeighborIDs when building traversal frontiers to avoid edge duplication.
//   - If you need both in/out neighbors for directed graphs, define a dedicated API explicitly.
func (g *Graph) NeighborIDs(id string) ([]string, error) {
	// AI-HINT: Output is unique and sorted (lex asc); relies on Neighbors(id).
	edges, err := g.Neighbors(id)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}, len(edges))
	for _, e := range edges {
		if e.From == id {
			seen[e.To] = struct{}{}
			continue
		}
		if !e.Directed && e.To == id {
			seen[e.From] = struct{}{}
		}
	}

	ids := make([]string, 0, len(seen))
	for v := range seen {
		ids = append(ids, v)
	}
	sort.Strings(ids)

	return ids, nil
}

// AdjacencyList returns a snapshot mapping each "from" vertex ID to the list of incident edge IDs.
// Each slice is sorted by Edge.ID ascending for deterministic per-vertex enumeration.
//
// Implementation:
//   - Stage 1: Acquire muEdgeAdj read lock for a stable adjacency snapshot.
//   - Stage 2: For each from vertex, collect all edge IDs from nested adjacency buckets.
//   - Stage 3: Sort the collected edge IDs per vertex.
//   - Stage 4: Return the newly allocated map of slices.
//
// Behavior highlights:
//   - Returned slices are freshly allocated and safe to retain and mutate by the caller.
//   - Deterministic ordering within each slice (Edge.ID asc).
//
// Inputs:
//   - None.
//
// Returns:
//   - map[string][]string: snapshot from-vertex -> sorted edge ID list.
//
// Errors:
//   - None (pure query).
//
// Determinism:
//   - Per-vertex slices are deterministic (sorted).
//   - Map key iteration order is not deterministic in Go; callers MUST NOT rely on it.
//
// Complexity:
//   - Time O(V + E + Σ sort(deg(v))), Space O(V + E) for the snapshot.
//
// Notes:
//   - This is a snapshot of adjacency state; it does not expose internal maps.
//   - Use Vertices() to obtain deterministic key order if needed.
//
// AI-Hints:
//   - If you need stable iteration over keys, do: keys := g.Vertices(); then read result[key].
func (g *Graph) AdjacencyList() map[string][]string {
	// AI-HINT: Each slice is freshly allocated and sorted; callers may retain and mutate safely.
	g.muEdgeAdj.RLock()
	defer g.muEdgeAdj.RUnlock()

	result := make(map[string][]string, len(g.adjacencyList))
	for from, toMap := range g.adjacencyList {
		// Fresh buffer per vertex to avoid sharing backing arrays across keys.
		var buf []string
		for _, edgeMap := range toMap {
			for eid := range edgeMap {
				buf = append(buf, eid) // collect all incident edge IDs
			}
		}
		sort.Strings(buf)  // deterministic enumeration
		result[from] = buf // safe to retain by the caller
	}

	return result
}

// ensureAdjacency guarantees that adjacencyList[from] and adjacencyList[from][to] are initialized.
//
// Implementation:
//   - Stage 1: If adjacencyList[from] is nil, allocate it.
//   - Stage 2: If adjacencyList[from][to] is nil, allocate it.
//
// Behavior highlights:
//   - O(1) amortized nested-map initialization.
//   - No-op when buckets already exist.
//
// Inputs:
//   - g: target graph.
//   - from: source vertex ID.
//   - to: destination vertex ID.
//
// Returns:
//   - None.
//
// Errors:
//   - None.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1) amortized, Space O(1) amortized.
//
// Notes:
//   - Must be called ONLY under muEdgeAdj write lock by mutating code paths.
//
// AI-Hints:
//   - Keep this helper small and allocation-minimal; it is on mutation hot paths.
func ensureAdjacency(g *Graph, from, to string) {
	// AI-HINT: Called only under muEdgeAdj write lock by mutating codepaths.
	if g.adjacencyList[from] == nil {
		g.adjacencyList[from] = make(map[string]map[string]struct{})
	}
	if g.adjacencyList[from][to] == nil {
		g.adjacencyList[from][to] = make(map[string]struct{})
	}
}

// removeAdjacency removes e.ID from adjacency buckets for the edge endpoints.
//
// Removal policy:
//   - Always remove from e.From -> e.To.
//   - If the edge is undirected and not a self-loop, also remove from e.To -> e.From.
//
// Implementation:
//   - Stage 1: Delete e.ID from the primary bucket.
//   - Stage 2: If the bucket becomes empty, prune the nested map entry.
//   - Stage 3: If undirected non-loop, repeat for the mirrored bucket.
//
// Behavior highlights:
//   - Keeps adjacency buckets compact by pruning empty nested maps.
//
// Inputs:
//   - g: target graph.
//   - e: edge to unlink (must be non-nil).
//
// Returns:
//   - None.
//
// Errors:
//   - None.
//
// Determinism:
//   - Deterministic for a fixed graph state.
//
// Complexity:
//   - Time O(1) average, Space O(1).
//
// Notes:
//   - Must be called ONLY under muEdgeAdj write lock.
//
// AI-Hints:
//   - Always pair catalog deletion (delete(g.edges,e.ID)) with removeAdjacency to avoid dangling adjacency references.
func removeAdjacency(g *Graph, e *Edge) {
	// AI-HINT: Removes e.ID from from→to and (if undirected non-loop) to→from; write lock required.
	if m := g.adjacencyList[e.From][e.To]; m != nil {
		delete(m, e.ID)
		if len(m) == 0 {
			delete(g.adjacencyList[e.From], e.To)
		}
	}
	if !e.Directed && e.From != e.To {
		if m := g.adjacencyList[e.To][e.From]; m != nil {
			delete(m, e.ID)
			if len(m) == 0 {
				delete(g.adjacencyList[e.To], e.From)
			}
		}
	}
}

// cleanupAdjacency prunes empty nested adjacency buckets after removals.
//
// Implementation:
//   - Stage 1: Scan adjacencyList top-level keys.
//   - Stage 2: For each nested toMap, remove empty edgeSet buckets.
//   - Stage 3: Remove top-level entries whose nested map becomes empty.
//
// Behavior highlights:
//   - Maintains compact adjacency structure to keep HasEdge and scans fast.
//   - Safe to call repeatedly; idempotent relative to empty-state pruning.
//
// Inputs:
//   - g: target graph.
//
// Returns:
//   - None.
//
// Errors:
//   - None.
//
// Determinism:
//   - Deterministic given a fixed map state.
//
// Complexity:
//   - Time O(V + B) where B is the number of (from,to) buckets scanned; worst-case O(V+E) in dense maps.
//   - Space O(1).
//
// Notes:
//   - Must be called ONLY under muEdgeAdj write lock.
//
// AI-Hints:
//   - Call cleanupAdjacency after bulk removals (FilterEdges, RemoveVertex) rather than after every single deletion if batching is possible.
func cleanupAdjacency(g *Graph) {
	// AI-HINT: Prunes empty buckets after removals; write lock required.
	for u, toMap := range g.adjacencyList {
		for v, edgeSet := range toMap {
			if len(edgeSet) == 0 {
				delete(toMap, v)
			}
		}
		if len(toMap) == 0 {
			delete(g.adjacencyList, u)
		}
	}
}
