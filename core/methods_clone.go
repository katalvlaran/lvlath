// File: methods_clone.go
// Role: Cloning and clearing graph instances.
// Determinism:
//   - CloneEmpty/Clone carry over nextEdgeID to keep textual edge IDs monotonic on the clone.
// Concurrency:
//   - Read locks for snapshotting; no mutation of the source graph.
// AI-HINT (file):
//   - CloneEmpty/Clone carry nextEdgeID so future AddEdge() continues textual sequence on the clone.
//   - Clear() preserves flags but resets catalogs and counter.

package core

import "sync/atomic"

// CloneEmpty returns a new Graph with identical configuration and vertices, but no edges.
//
// Implementation:
//   - Stage 1: Acquire muVert and muEdgeAdj read locks to snapshot flags and vertex catalog safely.
//   - Stage 2: Construct a new Graph with equivalent GraphOptions (flags only).
//   - Stage 3: Carry over nextEdgeID to preserve the textual edge ID sequence on the clone.
//   - Stage 4: Copy vertices (shallow metadata pointer copy) and initialize empty per-vertex adjacency maps.
//   - Stage 5: Return the clone.
//
// Behavior highlights:
//   - Preserves configuration flags (Directed default, Weighted, MultiEdges, Loops, MixedMode).
//   - Preserves vertex identities.
//   - Drops all edges (edge catalog and adjacency remain empty).
//
// Inputs:
//   - None.
//
// Returns:
//   - *Graph: new graph instance with copied flags and vertices, no edges.
//
// Errors:
//   - None.
//
// Determinism:
//   - Deterministic for a fixed source graph state.
//
// Complexity:
//   - Time O(V), Space O(V).
//
// Notes:
//   - Vertex.Metadata is shallow-copied (the map pointer is reused).
//     If deep copy is required, callers must implement it externally.
//   - nextEdgeID is carried over to prevent future auto-generated IDs from "rewinding".
//
// AI-Hints:
//   - Use CloneEmpty when you need the same vertex universe but want to rebuild topology from scratch.
func (g *Graph) CloneEmpty() *Graph {
	// Acquire locks to ensure consistent vertex snapshot.
	g.muVert.RLock()
	defer g.muVert.RUnlock()
	// Lock edges too, to ensure we don't snapshot whilst Clear() or internal reorg is happening,
	// maintaining strict Lock ordering (Vert -> EdgeAdj).
	g.muEdgeAdj.RLock()
	defer g.muEdgeAdj.RUnlock()

	// 1. Copy configuration
	opts := []GraphOption{WithDirected(g.directed)}
	if g.weighted {
		opts = append(opts, WithWeighted())
	}
	if g.allowMulti {
		opts = append(opts, WithMultiEdges())
	}
	if g.allowLoops {
		opts = append(opts, WithLoops())
	}
	if g.allowMixed {
		opts = append(opts, WithMixedEdges())
	}

	clone := NewGraph(opts...)

	// 2. Carry over nextEdgeID
	atomic.StoreUint64(&clone.nextEdgeID, atomic.LoadUint64(&g.nextEdgeID))

	// 3. Copy vertices (Shallow Copy of Metadata)
	// ALIASING WARNING: Metadata map is shared between source and clone.
	var id string
	var v *Vertex
	for id, v = range g.vertices {
		clone.vertices[id] = &Vertex{ID: v.ID, Metadata: v.Metadata}
		clone.adjacencyList[id] = make(map[string]map[string]struct{})
	}

	return clone
}

// Clone returns a deep topology copy of the Graph: configuration, vertices, edges, and adjacency.
//
// Implementation:
//   - Stage 1: Acquire BOTH muVert and muEdgeAdj Rlocks (atomic snapshot).
//   - Stage 2: Create new Graph and copy configuration.
//   - Stage 3: Copy vertices (shallow metadata).
//   - Stage 4: Copy edges and rebuild adjacency.
//   - Stage 5: Release locks.
//
// Behavior highlights:
//   - ATOMIC: The clone represents the graph state at a single instant.
//   - Preserves Edge.ID, endpoints, weights, and directedness.
//   - Vertex.Metadata is shallow-copied (shared pointer).
//
// Inputs:
//   - None.
//
// Returns:
//   - *Graph: cloned graph instance.
//
// Errors:
//   - None.
//
// Determinism:
//   - Deterministic for a fixed source graph state.
//
// Complexity:
//   - Time O(V + E), Space O(V + E).
//
// Notes:
//   - Edges are copied as new Edge structs; callers must still treat returned edges as immutable.
//   - nextEdgeID carry-over ensures future AddEdge IDs remain monotonic on the clone.
//
// AI-Hints:
//   - Use Clone when algorithms need a sandbox graph to mutate without affecting the original.
func (g *Graph) Clone() *Graph {
	// LOCK ORDER: muVert -> muEdgeAdj.
	// We must hold BOTH locks to ensure the vertex set doesn't change
	// while we are copying edges that refer to them.
	g.muVert.RLock()
	defer g.muVert.RUnlock()

	g.muEdgeAdj.RLock()
	defer g.muEdgeAdj.RUnlock()

	// 1. Setup new graph with same config
	opts := []GraphOption{WithDirected(g.directed)}
	if g.weighted {
		opts = append(opts, WithWeighted())
	}
	if g.allowMulti {
		opts = append(opts, WithMultiEdges())
	}
	if g.allowLoops {
		opts = append(opts, WithLoops())
	}
	if g.allowMixed {
		opts = append(opts, WithMixedEdges())
	}
	clone := NewGraph(opts...)

	// 2. Copy Vertices
	// ALIASING WARNING: Metadata is shared.
	for id, v := range g.vertices {
		clone.vertices[id] = &Vertex{ID: v.ID, Metadata: v.Metadata}
		clone.adjacencyList[id] = make(map[string]map[string]struct{})
	}

	// 3. Copy Edges & Adjacency
	var (
		eid   string
		e, ne *Edge
		ok    bool
	)
	for eid, e = range g.edges {
		// Duplicate Edge struct
		ne = &Edge{ID: eid, From: e.From, To: e.To, Weight: e.Weight, Directed: e.Directed}
		clone.edges[eid] = ne

		// Rebuild Adjacency: Forward
		if _, ok = clone.adjacencyList[e.From][e.To]; !ok {
			clone.adjacencyList[e.From][e.To] = make(map[string]struct{})
		}
		clone.adjacencyList[e.From][e.To][eid] = struct{}{}

		// Rebuild Adjacency: Backward (if undirected)
		if !e.Directed && e.From != e.To {
			if _, ok = clone.adjacencyList[e.To][e.From]; !ok {
				clone.adjacencyList[e.To][e.From] = make(map[string]struct{})
			}
			clone.adjacencyList[e.To][e.From][eid] = struct{}{}
		}
	}

	// 4. Preserve ID Counter
	atomic.StoreUint64(&clone.nextEdgeID, atomic.LoadUint64(&g.nextEdgeID))

	return clone
}

// Clear resets the graph to an empty state while preserving configuration flags.
//
// Implementation:
//   - Stage 1: Acquire muVert and muEdgeAdj write locks to perform an atomic reset.
//   - Stage 2: Reinitialize vertices, edges, and adjacencyList maps.
//   - Stage 3: Reset nextEdgeID to 0 (future auto edge IDs resume from "e1").
//   - Stage 4: Release locks.
//
// Behavior highlights:
//   - Preserves flags: Directed default, Weighted, MultiEdges, Loops, MixedMode.
//   - Drops all vertices and edges.
//   - Resets ID counter deterministically.
//
// Inputs:
//   - None.
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
//   - Time O(1) for map reallocation, Space O(1) new empty maps (old maps become GC-eligible).
//
// Notes:
//   - Not safe to call concurrently with readers/writers; it acquires both write locks.
//   - After Clear(), the graph is equivalent to a newly constructed graph with the same options.
//
// AI-Hints:
//   - Prefer Clear() for reuse in benchmarks/tests to avoid repeated allocations from NewGraph.
func (g *Graph) Clear() {
	// AI-HINT: Resets vertices/edges/adjacency and nextEdgeID; configuration flags remain unchanged.
	g.muVert.Lock()
	g.muEdgeAdj.Lock()
	// reset maps
	g.vertices = make(map[string]*Vertex)
	g.edges = make(map[string]*Edge)
	g.adjacencyList = make(map[string]map[string]map[string]struct{})
	atomic.StoreUint64(&g.nextEdgeID, 0)

	g.muEdgeAdj.Unlock()
	g.muVert.Unlock()
}
