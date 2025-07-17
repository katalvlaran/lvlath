// Package matrix provides functions to build Dense adjacency and incidence matrices from core.Graph definitions,
// preserving semantic fidelity, enforcing strict fail-fast validation, and adhering to the lvlath coding standards.
package matrix

import (
	"errors"
	"fmt"
	"math"

	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/matrix/ops"
)

const (
	// defaultWeight is used for unweighted graphs when opts.Weighted=false
	defaultWeight = 1.0
	// undirectedPairSize normalizes vertex pair ordering for undirected edges
	undirectedPairSize = 2
)

// ErrInvalidWeight is returned when an edge weight is NaN or infinite.
var ErrInvalidWeight = errors.New("matrix: invalid edge weight")

// lookupIndex returns the index for the given vertex key or an error if missing.
func lookupIndex(idx map[string]int, key string) (int, error) {
	// Validate that key exists in map
	if i, ok := idx[key]; ok {
		return i, nil // found index
	}
	// Fail-fast on unknown vertex
	return 0, fmt.Errorf("lookupIndex: unknown vertex %q: %w", key, ErrMatrixUnknownVertex)
}

// shouldSkipMulti determines whether an undirected edge between u and v should be skipped
// due to multi-edge collapse.
func shouldSkipMulti(u, v int, seen map[[2]int]bool) bool {
	// Normalize order so key is consistent
	if u > v {
		u, v = v, u
	}
	key := [2]int{u, v}
	if seen[key] {
		return true // already seen, skip
	}
	seen[key] = true // mark as seen

	return false
}

// applyMetricClosure fills zero entries with +Inf and runs APSP via Floyd–Warshall.
// Complexity: O(V^2 + V^3)
func applyMetricClosure(mat *Dense) error {
	rows, cols := mat.Rows(), mat.Cols()
	// Stage 1 (Validate): ensure square matrix
	if rows != cols {
		return fmt.Errorf("applyMetricClosure: non-square %dx%d: %w", rows, cols, ErrMatrixDimensionMismatch)
	}
	// Stage 2 (Execute): replace zeros (excluding diagonal) with +Inf
	var u, v int
	var val float64
	for u = 0; u < rows; u++ {
		for v = 0; v < cols; v++ {
			if u == v {
				continue // skip diagonal
			}
			// safe at-bound access
			val, _ = mat.At(u, v)
			if val == 0 {
				mat.Set(u, v, math.Inf(1))
			}
		}
	}

	// Stage 3 (Execute): run all-pairs shortest paths
	return ops.FloydWarshall(mat)
}

// BuildDenseAdjacency constructs a V×V adjacency matrix in Dense form.
//   - vertices: ordered slice of unique vertex IDs (length V).
//   - edges:    slice of *core.Edge to populate (may be empty or nil).
//   - opts:     MatrixOptions controlling directed, weighted, loops, multi-edge, and metric closure.
//
// Returns:
//   - idx: map from vertex ID to its row/column index.
//   - mat: pointer to a Dense matrix of size V×V with populated weights.
//   - err: any error encountered (invalid dims, unknown vertex, invalid weight).
//
// Stage 1 (Validate): ensure vertices non-empty.
// Stage 2 (Prepare): build index map and allocate matrix.
// Stage 3 (Execute): iterate edges, set weights, collapse duplicates.
// Stage 4 (Finalize): apply metric closure if requested.
// Stage 5 (Return): return built structures or error.
func BuildDenseAdjacency(vertices []string, edges []*core.Edge, opts MatrixOptions) (map[string]int, *Dense, error) {
	// Stage 1: Validate inputs
	if len(vertices) == 0 {
		return nil, nil, ErrInvalidDimensions // no vertices to build
	}

	// Stage 2: Prepare index map and allocate matrix
	var (
		i    int
		id   string
		V    = len(vertices)           // number of vertices
		idx  = make(map[string]int, V) // vertex→index map
		mat  *Dense                    // result matrix
		err  error                     // error placeholder
		edge *core.Edge                // current edge
		seen = make(map[[2]int]bool)   // for undirected multi-edge tracking
		src  int                       // source index
		dst  int                       // destination index
		w    float64                   // edge weight
	)
	for i, id = range vertices {
		idx[id] = i // assign each vertex an index
	}
	mat, err = NewDense(V, V) // allocate zero-filled matrix
	if err != nil {
		return nil, nil, fmt.Errorf("BuildDenseAdjacency: %w", err)
	}

	// Stage 3: Execute edge population
	for _, edge = range edges {
		// lookup source index or fail
		src, err = lookupIndex(idx, edge.From)
		if err != nil {
			return nil, nil, fmt.Errorf("BuildDenseAdjacency: %w", err)
		}
		// lookup target index or fail
		dst, err = lookupIndex(idx, edge.To)
		if err != nil {
			return nil, nil, fmt.Errorf("BuildDenseAdjacency: %w", err)
		}
		// skip loops if not allowed
		if src == dst && !opts.AllowLoops {
			continue
		}
		// collapse undirected multi-edges
		if !opts.AllowMulti && !opts.Directed && shouldSkipMulti(src, dst, seen) {
			continue
		}
		// compute weight
		if opts.Weighted {
			w = float64(edge.Weight)
		} else {
			w = defaultWeight
		}
		// validate weight is finite
		if math.IsNaN(w) || math.IsInf(w, 0) {
			return nil, nil, ErrInvalidWeight
		}
		// set matrix entry
		if err = mat.Set(src, dst, w); err != nil {
			return nil, nil, fmt.Errorf("BuildDenseAdjacency: set(%d,%d): %w", src, dst, err)
		}
		// mirror for undirected graphs
		if !opts.Directed {
			if err = mat.Set(dst, src, w); err != nil {
				return nil, nil, fmt.Errorf("BuildDenseAdjacency: mirror set(%d,%d): %w", dst, src, err)
			}
		}
	}

	// Stage 4: Optional metric closure
	if opts.MetricClosure {
		if err = applyMetricClosure(mat); err != nil {
			return nil, nil, fmt.Errorf("BuildDenseAdjacency: metric closure: %w", err)
		}
	}

	// Stage 5: Return built index and matrix
	return idx, mat, nil
}

// BuildDenseIncidence constructs a V×E incidence matrix.
//
// Stage 1 (Validate): ensure vertices non-empty.
// Stage 2 (Prepare): build index and filter & dedupe edges.
// Stage 3 (Allocate): create Dense of size V×E'.
// Stage 4 (Populate): fill -1/+1 entries.
//
// Complexity: O(V + E + V·E)
// Memory: O(V·E)
func BuildDenseIncidence(
	vertices []string,
	edges []*core.Edge,
	opts MatrixOptions,
) (map[string]int, []*core.Edge, *Dense, error) {
	// Stage 1: Validate
	if len(vertices) == 0 {
		return nil, nil, nil, ErrInvalidDimensions
	}

	// Stage 2: Prepare index and columns
	var (
		i        int
		id, u, v string
		e        *core.Edge
		key      = [2]string{}
		V        = len(vertices)           // number of vertices
		idx      = make(map[string]int, V) // VertexID→row
		cols     = make([]*core.Edge, 0, len(edges))
		seen     = make(map[[2]string]bool) // for undirected dedupe
	)
	for i, id = range vertices {
		idx[id] = i
	}
	for _, e = range edges {
		// Skip loops if disallowed
		if e.From == e.To && !opts.AllowLoops {
			continue
		}
		u, v = e.From, e.To
		// Collapse undirected duplicates
		if !opts.Directed && !opts.AllowMulti {
			if u > v {
				u, v = v, u
			}
			key = [2]string{u, v}
			if seen[key] {
				continue
			}
			seen[key] = true
		}
		cols = append(cols, e)
	}

	// Stage 3: Allocate matrix
	E := len(cols)
	mat, err := NewDense(V, E)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("BuildDenseIncidence: %w", err)
	}

	// Stage 4: Populate
	var j, iFrom, iTo int
	for j, e = range cols {
		// Lookup indices (fail-fast)
		iFrom, err = lookupIndex(idx, e.From)
		if err != nil {
			return nil, nil, nil, err
		}
		iTo, err = lookupIndex(idx, e.To)
		if err != nil {
			return nil, nil, nil, err
		}
		// Mark source/target
		if opts.Directed {
			_ = mat.Set(iFrom, j, -1) // source
			_ = mat.Set(iTo, j, +1)   // target
		} else {
			_ = mat.Set(iFrom, j, +1) // both endpoints
			_ = mat.Set(iTo, j, +1)
		}
	}

	return idx, cols, mat, nil
}
