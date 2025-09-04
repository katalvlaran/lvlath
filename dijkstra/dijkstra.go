// Package dijkstra implements Dijkstra's shortest-path algorithm on weighted graphs.
//
// Dijkstra computes the minimum-cost path from a single source vertex to all
// other reachable vertices in a graph with non-negative edge weights.
// It processes vertices in order of increasing distance using a min-heap priority queue,
// relaxing edges and updating distances accordingly.
//
// Complexity:
//
//   - Time:  O((V + E) log V)
//   - Each vertex is extracted at most once: V extractions from the heap.
//   - Each edge relaxation may push a new entry into the heap: up to E pushes.
//   - Each heap operation (Push/Pop) costs O(log N), where N ≤ V + E. Simplified to O(log V).
//   - Space: O(V + E)
//   - O(V) for distance and predecessor maps.
//   - O(E) worst-case for entries in the heap under “lazy-decrease-key”.
//
// Notes on implementation choices:
//
//   - We perform an upfront scan of all edges (O(E)) to detect negative weights and fail fast.
//   - We treat any edge with weight ≥ InfEdgeThreshold as an impassable “wall”.
//   - We stop exploring once the minimum distance in the heap exceeds MaxDistance.
//   - We use a “lazy” decrease-key strategy: pushing duplicates into the heap and ignoring stale entries.
package dijkstra

import (
	"container/heap"
	"fmt"
	"math"

	"github.com/katalvlaran/lvlath/core"
)

// Dijkstra computes shortest distances from the source vertex (Options.Source)
// to all other vertices in the weighted graph g. It accepts functional options
// to customize behavior (ReturnPath, MaxDistance, InfEdgeThreshold, etc.).
//
// Returns:
//
//   - dist: map from vertex ID to minimum distance (math.MaxInt64 if unreachable).
//   - prev: optional predecessor map if ReturnPath=true (nil otherwise).
//     prev[v] == u means the shortest path to v goes through u.
//     For unreachable v, prev[v] == "".
//   - err:  error if inputs are invalid or if a negative weight is detected.
//
// Preconditions and validation (in order):
//  1. Source string must be non-empty (ErrEmptySource).
//  2. g must be non-nil (ErrNilGraph).
//  3. g must be weighted (ErrUnweightedGraph).
//  4. g must contain Source (ErrVertexNotFound).
//  5. No edge in g can have negative weight (ErrNegativeWeight).
//
// Options customization:
//
//   - WithReturnPath(): return predecessor map.
//   - WithMaxDistance(x): vertices with distance > x are not explored (x ≥ 0).
//   - WithInfEdgeThreshold(t): edges with weight ≥ t are skipped (t > 0).
//
// Complexity:
//
//   - Time:  O((V + E) log V)
//   - Space: O(V + E)
func Dijkstra(g *core.Graph, opts ...Option) (map[string]int64, map[string]string, error) {
	// 1) Build and validate Options
	cfg := DefaultOptions("") // default options
	var opt Option
	for _, opt = range opts { // apply each functional option
		opt(&cfg)
	}

	// 2) Validate Source is provided
	if cfg.Source == "" {
		return nil, nil, ErrEmptySource
	}

	// 3) Validate graph is non-nil
	if g == nil {
		return nil, nil, ErrNilGraph
	}

	// 4) Validate graph supports weights
	if !g.Weighted() {
		return nil, nil, ErrUnweightedGraph
	}

	// 5) Validate Source exists in the graph
	if !g.HasVertex(cfg.Source) {
		return nil, nil, ErrVertexNotFound
	}

	// 6) Validate MaxDistance non-negative (option constructor already panics if negative).
	//    Here we ensure cfg.MaxDistance is ≥ 0. In theory, MaxDistance < 0 is impossible because
	//    WithMaxDistance panics. But if user bypassed WithMaxDistance, cfg.MaxDistance==MaxInt64.

	// 7) Validate InfEdgeThreshold > 0 (option constructor already panics if ≤ 0).
	//    Default is MaxInt64, so this check is effectively always satisfied here.

	// 8) Pre-scan all edges to detect negative weights. Fail fast with ErrNegativeWeight.
	var e *core.Edge
	for _, e = range g.Edges() {
		if e.Weight < 0 {
			// Return the sentinel error with context of which edge failed.
			return nil, nil, fmt.Errorf("%w: edge %s→%s weight=%d", ErrNegativeWeight, e.From, e.To, e.Weight)
		}
	}

	// 9) Prepare data structures for the algorithm.
	//    Let V = number of vertices.
	V := len(g.Vertices())

	//    dist maps each vertex ID to its current best-known distance from Source.
	dist := make(map[string]int64, V)

	//    If ReturnPath or MemoryModeFull, we allocate prev map to track predecessors.
	//    Otherwise prev remains nil to save memory.
	var prev map[string]string
	if cfg.ReturnPath || cfg.MemoryMode == MemoryModeFull {
		prev = make(map[string]string, V)
	} else {
		prev = nil
	}

	//    visited marks whether we have finalized the shortest distance for a vertex.
	visited := make(map[string]bool, V)

	//    Initialize a priority queue (min-heap) for (vertex, distance) pairs.
	pq := make(nodePQ, 0, V) // capacity V is a reasonable starting point

	// 10) Initialize runner with all maps and the heap.
	r := &runner{
		g:       g,
		options: cfg,
		dist:    dist,
		prev:    prev,
		visited: visited,
		pq:      pq,
	}

	// 11) Initialize algorithm state and run main loop.
	r.init()
	if err := r.process(); err != nil {
		return nil, nil, err
	}

	// 12) Once done, if ReturnPath is false, we return prev as nil.
	if !cfg.ReturnPath {
		return r.dist, nil, nil
	}

	// 13) Otherwise, return both dist and prev.
	return r.dist, r.prev, nil
}

// runner holds the mutable state for a single Dijkstra execution.
type runner struct {
	g       *core.Graph       // The input graph; read-only within Dijkstra.
	options Options           // Configuration options (Source, thresholds, etc.).
	dist    map[string]int64  // Maps vertex ID → current best distance from Source.
	prev    map[string]string // Maps vertex ID → predecessor on the shortest path.
	visited map[string]bool   // Tracks if a vertex's distance is finalized.
	pq      nodePQ            // Min-heap of *nodeItem for lazy priority queue.
}

// init sets up initial distances, predecessors, visited flags, and pushes Source=0 into the heap.
func (r *runner) init() {
	// Retrieve a sorted list of vertices (core.Vertices returns sorted IDs).
	vertices := r.g.Vertices()

	// 1) Initialize dist[v] = +∞ (MaxInt64) for all vertices v.
	//    Initialize visited[v] = false.
	//    If prev is allocated, set prev[v] = "" for all v.
	for _, v := range vertices {
		r.dist[v] = math.MaxInt64
		r.visited[v] = false
		if r.prev != nil {
			r.prev[v] = "" // no predecessor yet
		}
	}

	// 2) Distance to the source is zero.
	r.dist[r.options.Source] = 0

	// 3) Initialize the priority queue. heap.Init ensures the internal heap invariants hold.
	heap.Init(&r.pq)

	// 4) Push the source vertex with distance 0 onto the heap.
	//    We represent this as *nodeItem{id: Source, dist: 0}.
	heap.Push(&r.pq, &nodeItem{
		id:   r.options.Source,
		dist: 0,
	})
}

// process is the core loop of Dijkstra's algorithm. It repeatedly extracts the vertex
// with the minimum distance from the source and relaxes its outgoing edges.
//
// Loop termination conditions:
//
//   - The heap becomes empty (all reachable vertices processed).
//   - The minimum distance in the heap exceeds MaxDistance (no need to explore farther).
//
// Returns an error if any invalid edge weight is found during relaxation (should not happen due to pre-scan).
func (r *runner) process() error {
	// Unpack local references for brevity.
	cfg := r.options
	var u string
	var d int64
	for r.pq.Len() > 0 {
		// 1) Pop the smallest-distance item from the heap.
		item := heap.Pop(&r.pq).(*nodeItem)
		u = item.id
		d = item.dist

		// 2) If this vertex was already visited (finalized), skip stale heap entry.
		if r.visited[u] {
			continue
		}

		// 3) If this distance exceeds MaxDistance, stop exploring any further vertices.
		//    Do NOT mark u as visited, as we never relax it. We simply break.
		if d > cfg.MaxDistance {
			break
		}

		// 4) Mark u as visited. Its shortest distance d is now final.
		r.visited[u] = true

		// 5) Relax all outgoing edges from u.
		if err := r.relax(u); err != nil {
			return err
		}
	}

	return nil
}

// relax examines each edge outgoing from vertex u and attempts to improve distances to its neighbors.
// It respects the InfEdgeThreshold and ignores any edge weight ≥ that threshold (treating them as impassable).
// If a shorter path to neighbor v is found (newDist < dist[v]), we update dist[v], prev[v], and push a new heap entry.
//
// Assumes r.dist[u] is finalized before calling relax(u).
func (r *runner) relax(u string) error {
	// 1) Retrieve the list of edges incident to u. core.Neighbors returns all edges
	//    for which e.From == u if e.Directed == true, and both directions if e.Directed == false.
	neighbors, err := r.g.Neighbors(u)
	if err != nil {
		// Wrap core error with context
		return fmt.Errorf("dijkstra: failed to get neighbors of %q: %w", u, err)
	}

	// 2) For each edge e in neighbors, attempt relaxation.
	var e *core.Edge
	var v string
	var w int64
	var newDist int64
	for _, e = range neighbors {
		// e.From and e.To are vertex IDs for this edge.
		// e.Directed indicates if this edge is one-way; if true, the edge is valid only if e.From == u.
		// If e.Directed is false, the edge is undirected, and e appears in both u's and neighbor's adjacency lists.

		// Filter out directed edges that do not originate from u.
		// This ensures we do not “walk backwards” along a directed edge that appears in the neighbor list.
		if e.Directed && e.From != u {
			continue // skip edges that do not actually go out of u
		}

		v = e.To     // neighbor vertex
		w = e.Weight // edge weight from u → v

		//  Skip any edge that is marked as impassable by InfEdgeThreshold.
		//  If w >= InfEdgeThreshold, we treat the edge as “infinite” and do not traverse.
		if w >= r.options.InfEdgeThreshold {
			continue
		}

		// Safety check: though we pre-scanned for negative weights, double-check nonetheless.
		if w < 0 {
			return fmt.Errorf("%w: edge %s→%s weight=%d", ErrNegativeWeight, u, v, w)
		}

		// Compute candidate distance if we go from Source → … → u → v.
		newDist = r.dist[u] + w

		// If newDist exceeds MaxDistance, we skip relaxing this neighbor.
		if newDist > r.options.MaxDistance {
			continue
		}

		// If newDist is not strictly better than the current dist[v], skip.
		//     Note: we use “<” rather than “≤” to avoid pushing duplicates when distances are equal.
		if newDist >= r.dist[v] {
			continue
		}

		// We have found a strictly shorter path to v. Update dist[v].
		r.dist[v] = newDist

		// If ReturnPath is requested, record u as the predecessor of v.
		// If prev is nil (ReturnPath=false and MemoryModeCompact), this line is skipped.
		if r.prev != nil {
			r.prev[v] = u
		}

		// Push the updated distance for v onto the heap.
		// This is the “lazy-decrease-key” pattern: we do not remove old entries,
		// but instead ignore them later when popped if visited[v] is already true.
		heap.Push(&r.pq, &nodeItem{
			id:   v,
			dist: newDist,
		})
	}

	return nil
}

// nodeItem represents a vertex and its current distance from the source.
// It is stored in the priority queue to order vertices by increasing distance.
type nodeItem struct {
	id   string // vertex ID
	dist int64  // distance from source
}

// nodePQ is a min-heap (priority queue) of *nodeItem, ordered by nodeItem.dist ascending.
// We use the “lazy-decrease-key” approach: when we find a shorter distance to an existing vertex v,
// we push a new *nodeItem onto the heap. The outdated entry remains but is ignored when popped
// (checked via visited[v]).
type nodePQ []*nodeItem

// Len returns the number of items in the heap.
func (pq nodePQ) Len() int { return len(pq) }

// Less defines the comparison: smaller dist → higher priority.
func (pq nodePQ) Less(i, j int) bool { return pq[i].dist < pq[j].dist }

// Swap swaps two elements in the heap.
func (pq nodePQ) Swap(i, j int) { pq[i], pq[j] = pq[j], pq[i] }

// Push adds a new element x onto the heap.
// Called by heap.Push; x must be of type *nodeItem.
func (pq *nodePQ) Push(x interface{}) { *pq = append(*pq, x.(*nodeItem)) }

// Pop removes and returns the smallest element from the heap.
// Called by heap.Pop; returns interface{} that must be cast to *nodeItem.
func (pq *nodePQ) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[:n-1]

	return item
}
