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

// Neighbors lists *all* edges touching id.
//   - Directed edges: only those with e.From==id.
//   - Undirected edges: both directions, but loop appears once.
//
// Sorted by Edge.ID.
// Complexity: O(d log d).
func (g *Graph) Neighbors(id string) ([]*Edge, error) {
	// AI-HINT: empty id → ErrEmptyVertexID; missing vertex → ErrVertexNotFound.
	//          Deterministic order by Edge.ID asc; treat returned *Edge as read-only.
	if id == "" {
		return nil, ErrEmptyVertexID
	}
	// Ensure vertex exists
	g.muVert.RLock()
	if _, ok := g.vertices[id]; !ok {
		g.muVert.RUnlock()
		return nil, ErrVertexNotFound
	}
	g.muVert.RUnlock()

	// Lock edges+adjacency for reading
	g.muEdgeAdj.RLock()
	defer g.muEdgeAdj.RUnlock()

	var out []*Edge
	// Iterate all "to" maps for this vertex

	var eid string
	var e *Edge
	for _, edgeSet := range g.adjacencyList[id] {
		for eid = range edgeSet {
			e = g.edges[eid]
			// For directed, include only if e.From == id
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

// NeighborIDs returns unique, sorted vertex IDs adjacent to id.
//
//	e.From==id ⇒ include e.To.
//	e.To==id && !e.Directed ⇒ include e.From.
//
// Complexity: O(d log d).
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
		} else if !e.Directed && e.To == id {
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

// AdjacencyList returns a snapshot mapping each vertex ID to the list of
// incident edge IDs. For determinism, each slice is sorted by Edge.ID asc.
//
// Notes:
//   - The order of map keys is unspecified in Go; callers must not rely on it.
//   - Slices are freshly allocated and safe to retain by the caller.
//
// Complexity: O(V + E) to assemble + O(sum_deg log deg) to sort per-vertex slices.
// Concurrency: safe; holds edges/adjacency read lock for the duration of the snapshot.
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

//–– Helpers ––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––––

// ensureAdjacency guarantees the presence of nested maps for (from,to).
// Must be called under muEdgeAdj write lock by mutating code paths.
// Complexity: O(1) amortized.
func ensureAdjacency(g *Graph, from, to string) {
	// AI-HINT: Called only under muEdgeAdj write lock by mutating codepaths.
	if g.adjacencyList[from] == nil {
		g.adjacencyList[from] = make(map[string]map[string]struct{})
	}
	if g.adjacencyList[from][to] == nil {
		g.adjacencyList[from][to] = make(map[string]struct{})
	}
}

// removeAdjacency deletes e.ID from both directions:
//   - from→to always;
//   - if e is undirected and not a self-loop, also to→from.
//
// Must be called under muEdgeAdj write lock.
// Complexity: O(1) average.
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

// cleanupAdjacency prunes empty nested maps after removals to keep HasEdge fast.
// Must be called under muEdgeAdj write lock.
// Complexity: O(V + E) worst-case when many empty buckets exist.
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
