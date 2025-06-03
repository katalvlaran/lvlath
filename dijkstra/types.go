// Package dijkstra defines core types and configuration options
// for Dijkstra's shortest-path algorithm on weighted graphs.
//
// Dijkstra computes the minimum-cost path from a single source vertex to all
// other reachable vertices in a graph with non-negative edge weights.
// The algorithm maintains a priority queue of vertices to explore and
// relaxes edges in increasing order of distance from the source vertex.
//
// Complexity:
//
//	– Time:  O((V + E) log V)   where V = |vertices|, E = |edges|
//	   • Each vertex is extracted from the priority queue at most once (V extracts).
//	   • Each edge relaxation may push into the priority queue (up to E pushes).
//	   • Each heap operation (push/pop) costs O(log V) or O(log (V+E)), simplified to O(log V).
//	– Space: O(V + E)
//	   • O(V) to store distance and predecessor maps.
//	   • O(E) in the priority queue in the worst case (lazy decrease-key).
//
// Options:
//
//	– Source:           ID of the starting vertex (must be non-empty and present in the graph).
//	– ReturnPath:       if true, return the predecessor map for path reconstruction.
//	– MaxDistance:      optional cap on distances to explore; vertices beyond this are skipped.
//	– InfEdgeThreshold: edges with weight >= this threshold are treated as impassable.
//
// Errors (sentinel):
//
//	– ErrEmptySource     if the provided source ID is empty.
//	– ErrNilGraph        if the provided graph pointer is nil.
//	– ErrUnweightedGraph if the graph is not configured to support weights.
//	– ErrVertexNotFound  if the source vertex does not exist in the graph.
//	– ErrNegativeWeight  if a negative edge weight is detected in the graph.
//	– ErrBadMaxDistance  if MaxDistance < 0.
//	– ErrBadInfThreshold if InfEdgeThreshold <= 0.
//
// Example usage:
//
//	// Compute distances and predecessors from "A":
//	dist, prev, err := Dijkstra(
//	    g,
//	    Source("A"),
//	    WithReturnPath(),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Distance to B: %d, parent: %s\n", dist["B"], prev["B"])
package dijkstra

import (
	"errors"
	"math"
)

// Sentinel errors returned by the Dijkstra implementation.
var (
	// ErrEmptySource indicates that the provided source vertex ID is empty.
	ErrEmptySource = errors.New("dijkstra: source vertex ID is empty")

	// ErrNilGraph indicates that a nil *core.Graph was passed to Dijkstra.
	ErrNilGraph = errors.New("dijkstra: graph is nil")

	// ErrUnweightedGraph indicates that the graph was not marked as weighted
	// but Dijkstra requires non-negative weights to compute shortest paths.
	ErrUnweightedGraph = errors.New("dijkstra: graph must be weighted")

	// ErrVertexNotFound indicates that the specified source vertex does not exist
	// in the provided graph.
	ErrVertexNotFound = errors.New("dijkstra: source vertex not found in graph")

	// ErrNegativeWeight indicates that a negative edge weight was detected in the graph.
	ErrNegativeWeight = errors.New("dijkstra: negative edge weight encountered")

	// ErrBadMaxDistance indicates that MaxDistance was set to a negative value,
	// which is not meaningful for a distance threshold.
	ErrBadMaxDistance = errors.New("dijkstra: MaxDistance must be non-negative")

	// ErrBadInfThreshold indicates that InfEdgeThreshold was set to zero or negative,
	// which would treat all edges (including zero-weight edges) as impassable.
	ErrBadInfThreshold = errors.New("dijkstra: InfEdgeThreshold must be positive")
)

// MemoryMode controls how predecessor information is stored during Dijkstra's execution.
//
// Note: Currently only MemoryModeFull is fully supported; MemoryModeCompact is reserved
// for future implementations where predecessor storage is minimized and paths are
// reconstructed via repeated partial computation.
//
// MemoryModeFull    – store complete predecessor map for immediate path reconstruction.
// MemoryModeCompact – minimize memory; omit or compress predecessor data (not yet implemented).
type MemoryMode int

const (
	// MemoryModeFull stores all predecessors to allow direct path recovery.
	MemoryModeFull MemoryMode = iota

	// MemoryModeCompact reduces memory footprint; requires external path derivation.
	// At present, MemoryModeCompact does not alter behavior (equivalent to Full).
	MemoryModeCompact
)

// Options configures the behavior of the Dijkstra algorithm.
//
// Source           – starting vertex ID (must be non-empty and present in the graph).
// ReturnPath       – if true, return the predecessor map; otherwise prev map is nil.
// MaxDistance      – optional cap on distances to explore (vertices beyond are skipped).
//
//	Must be ≥ 0. Default is math.MaxInt64 (no cap).
//
// InfEdgeThreshold – treat edges with weight ≥ this threshold as impassable obstacles.
//
//	Must be > 0. Default is math.MaxInt64 (no obstacles).
type Options struct {
	Source           string     // The ID of the source vertex
	MemoryMode       MemoryMode // Controls how predecessors are stored (Full or Compact)
	ReturnPath       bool       // Whether to return the predecessor map
	MaxDistance      int64      // Maximum distance to explore
	InfEdgeThreshold int64      // Weight threshold above which edges are non-traversable
}

// Option represents a functional option for configuring Dijkstra.
type Option func(*Options)

// WithMemoryMode sets the memory mode for storing predecessor information.
// MemoryModeFull: store full predecessor map.
// MemoryModeCompact: minimize memory (behavior currently same as Full).
func WithMemoryMode(mode MemoryMode) Option {
	return func(o *Options) {
		o.MemoryMode = mode
	}
}

// Source sets the Source field of Options to the given string.
// Must be called to specify the starting vertex ID.
func Source(str string) Option {
	return func(o *Options) {
		o.Source = str
	}
}

// WithReturnPath enables generation of the predecessor map in the result.
// If false (default), the predecessor map is not returned (prev == nil).
func WithReturnPath() Option {
	return func(o *Options) {
		o.ReturnPath = true
	}
}

// WithMaxDistance sets a maximum distance threshold.
// Vertices whose shortest distance would exceed this value are not explored.
// Must pass a non-negative value; negative values cause ErrBadMaxDistance.
// Default (if not set) is math.MaxInt64 (no cap).
func WithMaxDistance(max int64) Option {
	return func(o *Options) {
		if max < 0 {
			// Panic to signal invalid configuration early.
			// In Go, panic in Option constructors is acceptable for invalid arguments.
			panic(ErrBadMaxDistance.Error())
		}
		o.MaxDistance = max
	}
}

// WithInfEdgeThreshold defines a weight threshold above which edges are
// considered non-traversable (treated as infinite weight).
// Edges with weight ≥ threshold are skipped entirely.
// Must pass a positive value; zero or negative cause ErrBadInfThreshold.
// Default (if not set) is math.MaxInt64 (no edges treated as impassable).
func WithInfEdgeThreshold(threshold int64) Option {
	return func(o *Options) {
		if threshold <= 0 {
			panic(ErrBadInfThreshold.Error())
		}
		o.InfEdgeThreshold = threshold
	}
}

// DefaultOptions returns an Options struct initialized with sensible defaults
// for the given source vertex ID. Use this as a starting point for further
// functional-options overrides.
//
// Defaults:
//   - Source:           <as passed> (no validation here; validated in Dijkstra).
//   - MemoryMode:       MemoryModeFull (predecessor map fully stored).
//   - ReturnPath:       false (predecessor map not returned).
//   - MaxDistance:      math.MaxInt64 (no distance limit; explore all reachable).
//   - InfEdgeThreshold: math.MaxInt64 (no edges treated as impassable).
func DefaultOptions(source string) Options {
	return Options{
		Source:           source,
		MemoryMode:       MemoryModeFull,
		ReturnPath:       false,
		MaxDistance:      math.MaxInt64,
		InfEdgeThreshold: math.MaxInt64,
	}
}
