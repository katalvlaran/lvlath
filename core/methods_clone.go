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
// Determinism & Identity:
//   - Carries over nextEdgeID so that future AddEdge calls on the clone continue the same
//     textual sequence and never collide with existing edges in the clone.
//
// Complexity: O(V) to copy vertices and initialize adjacency.
func (g *Graph) CloneEmpty() *Graph {
	// AI-HINT: No edges are copied; vertices + flags copied; nextEdgeID carried.
	g.muVert.RLock()
	defer g.muVert.RUnlock()
	g.muEdgeAdj.RLock()
	defer g.muEdgeAdj.RUnlock()
	// Copy configuration via options
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
	// Preserve the textual edge ID sequence to avoid collisions on future AddEdge.
	// (We load under the current lock; store into the clone without contention.)
	atomic.StoreUint64(&clone.nextEdgeID, atomic.LoadUint64(&g.nextEdgeID))
	// Copy vertices
	var id string
	var v *Vertex
	for id, v = range g.vertices {
		clone.vertices[id] = &Vertex{ID: v.ID, Metadata: v.Metadata}
		clone.adjacencyList[id] = make(map[string]map[string]struct{})
	}

	return clone
}

// Clone returns a deep copy of the Graph: configuration, vertices, edges, and adjacency.
//
// Determinism & Identity:
//   - Carries over nextEdgeID to keep edge-ID textual sequence monotonic on the clone.
//
// Complexity: O(V + E)
func (g *Graph) Clone() *Graph {
	// AI-HINT: Deep copy of vertices, edges, adjacency; IDs and Directedness preserved; nextEdgeID carried.
	clone := g.CloneEmpty()
	g.muEdgeAdj.RLock()
	defer g.muEdgeAdj.RUnlock()
	// Copy edges and adjacency
	var (
		eid   string
		e, ne *Edge
		ok    bool
	)
	for eid, e = range g.edges {
		// Duplicate Edge struct
		ne = &Edge{ID: eid, From: e.From, To: e.To, Weight: e.Weight, Directed: e.Directed}
		clone.edges[eid] = ne
		// Append to nested adjacency maps
		if _, ok = clone.adjacencyList[e.From][e.To]; !ok {
			clone.adjacencyList[e.From][e.To] = make(map[string]struct{})
		}
		clone.adjacencyList[e.From][e.To][eid] = struct{}{}
		if !e.Directed && e.From != e.To {
			if _, ok = clone.adjacencyList[e.To][e.From]; !ok {
				clone.adjacencyList[e.To][e.From] = make(map[string]struct{})
			}
			clone.adjacencyList[e.To][e.From][eid] = struct{}{}
		}
	}

	return clone
}

// Clear resets the graph to an empty state while preserving configuration flags.
//
// Behavior:
//   - Reinitializes vertices/edges/adjacency maps.
//   - Resets nextEdgeID to 0 (textual edge IDs will resume from "e1").
//   - Directed/Weighted/Multi/Loops/Mixed flags are preserved.
//
// Complexity: O(1) for map reallocation; no iteration over existing entries.
// Concurrency: acquires both write locks; not safe to call concurrently with readers.
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
