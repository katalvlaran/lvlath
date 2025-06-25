// Package flow implements a collection of maximum-flow algorithms on graphs
// represented by *core.Graph. It provides flexible, high-performance routines
// for computing the maximum feasible flow from a source to a sink in a network,
// supporting directed and mixed-edge graphs with weights, parallel edges, and loops.
//
// The key algorithms offered are:
//
//   - Ford–Fulkerson
//
//   - Method: depth-first search to find any augmenting path.
//
//   - Time:   O(E · F), where F is the total flow pushed (integral networks).
//
//   - Memory: O(V + E) for the residual capacity map and DFS stack.
//
//   - Use when simplicity and moderate capacities suffice.
//
//   - Edmonds–Karp
//
//   - Method: breadth-first search for shortest (fewest-edge) augmenting paths.
//
//   - Time:   O(V · E²) in the worst case with integer capacities.
//
//   - Memory: O(V + E) for residual map and BFS queues.
//
//   - Guarantees polynomial worst-case behavior.
//
//   - Dinic
//
//   - Method: level graph construction + blocking-flow via DFS.
//
//   - Time:   O(E · √V) on unit-capacity networks (general networks often near O(E·√V)).
//
//   - Memory: O(V + E) for level map, adjacency slices, and recursion state.
//
//   - High practical performance on dense or high-capacity graphs.
//
// # Graph Support
//
// All algorithms operate on *core.Graph, respecting its configuration flags:
//
//	– Directed or undirected edges (with per-edge mixed direction support).
//	– Weighted edges (capacity values).
//	– Optional multi-edges (parallel edges aggregated).
//	– Optional loops (ignored for augmenting-path search).
//
// Capacities are represented as int64, but an initial Epsilon threshold
// (float64) allows filtering very small weights when aggregating parallel edges.
//
// # API
//
// FlowOptions configures all three algorithms:
//
//	type FlowOptions struct {
//	    Ctx                  context.Context // for cancellation / timeouts
//	    Epsilon              float64         // ignore capacities ≤ Epsilon during build
//	    Verbose              bool            // log each augmentation step
//	    LevelRebuildInterval int             // Dinic only: rebuild level graph every N pushes
//	}
//
// Use DefaultOptions() to obtain production-safe defaults:
//
//	opts := flow.DefaultOptions()
//	// opts.Ctx = context.Background()
//	// opts.Epsilon = 1e-9
//	// opts.Verbose = false
//	// opts.LevelRebuildInterval = 0
//
// The core entry points all share the same signature:
//
//	func FordFulkerson(
//	    g *core.Graph,
//	    source, sink string,
//	    opts FlowOptions,
//	) (maxFlow int64, residual *core.Graph, err error)
//
//	func EdmondsKarp(
//	    g *core.Graph,
//	    source, sink string,
//	    opts FlowOptions,
//	) (maxFlow int64, residual *core.Graph, err error)
//
//	func Dinic(
//	    g *core.Graph,
//	    source, sink string,
//	    opts FlowOptions,
//	) (maxFlow int64, residual *core.Graph, err error)
//
// Each returns the computed maximum flow value and a **residual graph**
// that preserves all original configuration flags (directedness, weighting,
// loops, multi-edges, mixed-edges). The residual graph’s edges correspond
// to remaining forward capacity and newly created reverse edges.
//
// # Errors
//
//	ErrSourceNotFound - if the source vertex is missing in the input graph.
//	ErrSinkNotFound   - if the sink vertex is missing.
//	EdgeError         - if a negative capacity (beyond Epsilon) is encountered.
//	context.Canceled / context.DeadlineExceeded - if opts.Ctx is canceled.
//
// # Integration
//
//   - Relies on github.com/katalvlaran/lvlath/core for graph storage and iteration.
//   - Compatible with github.com/katalvlaran/lvlath/matrix for matrix-based pre-/post-processing.
//
// See: docs/FLOW.md for in‐depth tutorial, pseudocode, ASCII/mermaid diagrams, and pitfalls.
package flow
