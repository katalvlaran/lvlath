package flow

import (
	"context"
	"fmt"
	"math"

	"github.com/katalvlaran/lvlath/core"
)

// FordFulkerson computes the maximum flow from ⟨source⟩ to ⟨sink⟩ in a capacity network.
//
// Ford–Fulkerson repeatedly finds a path in the residual network with
// positive capacity and augments along it until no such path exists.
//
// Steps:
//  1. **Validation**: ensure source and sink exist.
//  2. **Build residual map**: for every directed (u→v),
//     capacity[u][v] = sum of all parallel edge weights,
//     and capacity[v][u] = 0 initially.
//  3. **Augmentation loop**:
//     a. Run DFS (or BFS) on residual graph to find any path ⟨p⟩
//     from source to sink whose minimum edge‐capacity > ε.
//     b. Let δ = bottleneck capacity along ⟨p⟩.
//     c. For each edge (u→v) in ⟨p⟩:
//     • capacity[u][v] -= δ
//     • capacity[v][u] += δ
//     d. totalFlow += δ.
//     e. Repeat until no augmenting path found.
//  4. **Construct residual core.Graph** (optional).
//
// Complexity: O(E · F) where F ≈ maxFlow / Epsilon
// Memory:     O(V + E) for residual capacity map.
//
// Use Ford–Fulkerson when you need a straightforward max-flow
// implementation and capacities are integral or small. For stronger
// worst‐case guarantees, consider Edmonds–Karp or Dinic.
//
// Returns:
//   - maxFlow: the total flow value found.
//   - residual: a copy of core.Graph annotated with residual capacities as weights.
//   - error: ErrSourceNotFound, ErrSinkNotFound, EdgeError (negative capacity), or context cancellation.
func FordFulkerson(
	ctx context.Context,
	g *core.Graph,
	source, sink string,
	opts *FlowOptions,
) (maxFlow float64, residual *core.Graph, err error) {
	// -- 1. Prepare context and epsilon
	if ctx == nil {
		ctx = context.Background()
	}
	eps := 1e-9
	if opts != nil && opts.Epsilon > 0 {
		eps = opts.Epsilon
	}

	// -- 2. Validate inputs
	if !g.HasVertex(source) {
		return 0, nil, ErrSourceNotFound
	}
	if !g.HasVertex(sink) {
		return 0, nil, ErrSinkNotFound
	}

	// -- 3. Initialize residual capacities
	// resid[u][v] = capacity from u→v
	resid := make(map[string]map[string]float64, len(g.Vertices()))
	for _, v := range g.Vertices() {
		id := v.ID
		resid[id] = make(map[string]float64)
	}
	for _, e := range g.Edges() {
		c := float64(e.Weight)
		if c < -eps {
			return 0, nil, EdgeError{From: e.From.ID, To: e.To.ID, Cap: c}
		}
		resid[e.From.ID][e.To.ID] += c
		// ensure reverse key exists
		if _, ok := resid[e.To.ID][e.From.ID]; !ok {
			resid[e.To.ID][e.From.ID] = 0
		}
	}

	// -- 4. Augmentation loop
	for {
		// a) find augmenting path using DFS
		visited := make(map[string]bool, len(resid))
		path, flow := DFSFindPath(resid, source, sink, visited, math.Inf(1), eps)
		if len(path) == 0 {
			break // no more augmenting path
		}
		if opts != nil && opts.Verbose {
			fmt.Printf("augmenting path %v with δ=%g\n", path, flow)
		}
		// b) apply flow along the path
		for i := 0; i < len(path)-1; i++ {
			u, v := path[i], path[i+1]
			resid[u][v] -= flow
			resid[v][u] += flow
		}
		maxFlow += flow
		// c) check cancellation
		if err = ctx.Err(); err != nil {
			return maxFlow, nil, err
		}
	}

	// -- 5. Build residual core.Graph for return
	residual = core.NewGraph(true, true)
	for id := range resid {
		residual.AddVertex(&core.Vertex{ID: id})
	}
	for u, m := range resid {
		for v, c := range m {
			if c > eps {
				// cast back to int64 for core.Graph
				residual.AddEdge(u, v, int64(c))
			}
		}
	}
	return maxFlow, residual, nil
}

// DFSFindPath performs a DFS in the residual capacity graph to locate
// any source→sink path with capacity > eps. Returns the path and its
// bottleneck flow. If none found, returns empty path.
func DFSFindPath(
	resid map[string]map[string]float64,
	u, sink string,
	visited map[string]bool,
	available float64,
	eps float64,
) ([]string, float64) {
	if u == sink {
		return []string{sink}, available
	}
	visited[u] = true
	for v, capUV := range resid[u] {
		if visited[v] || capUV <= eps {
			continue
		}
		// determine new bottleneck
		b := available
		if capUV < b {
			b = capUV
		}
		path, flow := DFSFindPath(resid, v, sink, visited, b, eps)
		if len(path) > 0 {
			return append([]string{u}, path...), flow
		}
	}
	return nil, 0
}
