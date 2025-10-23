// SPDX-License-Identifier: MIT
// Package matrix - canonical builders for Dense adjacency and incidence matrices.
// Deterministic, sentinel-accurate, and aligned with contracts.
//
// Purpose:
//   - Deterministic builders for dense adjacency and incidence matrices from core.Graph,
//     honoring Options (Directed/Weighted/AllowLoops/AllowMulti/MetricClosure).
//
// Policy & Contracts:
//   - Adjacency: 0/weight; metric-closure toggles to distances (+Inf as “no edge”, diag=0) then APSP.
//   - Incidence: directed (−1 on source, +1 on target; directed self-loop ⇒ skipped column),
//                undirected (+1/+1; self-loop ⇒ +2 in the single incident row).
//
// Determinism:
//   - First-edge-wins when AllowMulti=false (ordered or unordered key by directedness).
//   - Stable vertex order as provided by caller; no implicit sorting.
//
// AI-Hints:
//   - If you need lex order, pre-sort vertices in the caller.
//   - For sparse graphs, consider future sparse adapters; these are dense by design.

package matrix

import (
	"fmt"
	"math"

	"github.com/katalvlaran/lvlath/core"
)

// defaultWeight - unit weight for unweighted adjacency/incidence writes.
const defaultWeight = 1.0

// unreachableWeight is the placeholder for "no edge" before metric-closure.
// We use 0 in adjacency (standard 0/weight adjacency). During metric-closure
// this turns into +Inf on off-diagonals, while diag is forced to 0.
const unreachableWeight = 0.0

// orderedPair builds (u,v) key for directed de-duplication.
// Complexity: O(1).
func orderedPair(u, v int) pairKey { return pairKey{u: u, v: v} }

// unorderedPair builds {min,max} key for undirected de-duplication.
// Complexity: O(1).
func unorderedPair(u, v int) pairKey {
	if u <= v {
		return pairKey{u: u, v: v}
	}

	return pairKey{u: v, v: u}
}

// lookupIndex resolves a vertex ID to row/col index or returns ErrUnknownVertex.
// Complexity: O(1) expected (hash map).
func lookupIndex(idx map[string]int, id string) (int, error) {
	if i, ok := idx[id]; ok {
		return i, nil
	}

	return 0, fmt.Errorf("matrix: unknown vertex %q: %w", id, ErrUnknownVertex)
}

// allZeroWeights returns true if every edge in the slice has Weight == 0.
// Complexity: O(E) with early-exit on the first non-zero.
func allZeroWeights(edges []*core.Edge) bool {
	for i := 0; i < len(edges); i++ {
		if edges[i].Weight != 0 {
			return false
		}
	}

	return true
}

// isLexSorted returns true if s is non-decreasing in lexicographic order.
// Used defensively for vertex-list order enforcement in wrapper.
// Complexity: O(n).
func isLexSorted(s []string) bool {
	for i := 1; i < len(s); i++ {
		if s[i-1] > s[i] {
			return false
		}
	}

	return true
}

// BuildDenseAdjacency CONSTRUCTS a dense adjacency matrix from explicit vertices/edges
// with Options policy (directed/weighted/loops/multi, optional metric-closure).
// Implementation:
//   - Stage 1: validate vertex list and build VertexID→index map.
//   - Stage 2: allocate V×V dense and decide weight policy (degrade to binary if all-zero).
//   - Stage 3: populate entries deterministically; mirror when undirected (except loops).
//   - Stage 4: optional metric-closure (distances via Floyd–Warshall).
//
// Behavior highlights:
//   - First-edge-wins when AllowMulti=false (ordered for directed, unordered for undirected).
//   - Stable order equals provided vertex order; edges are scanned in stable core order.
//
// Inputs:
//   - vertices: canonical vertex order (stable; caller decides lex order if needed).
//   - edges: stable edge list (core contract: by Edge.ID asc).
//   - opts: Options defining Directed/Weighted/AllowLoops/AllowMulti/MetricClosure.
//
// Returns:
//   - vidx: VertexID→index map (row==col index).
//   - mat: V×V dense adjacency.
//   - err: ErrInvalidDimensions (empty vertices), ErrUnknownVertex, ErrInvalidWeight, shape/set errors.
//
// Determinism:
//   - Fixed loops and write order; deterministic output for same inputs/options.
//
// Complexity:
//   - Time O(V^2 + E), Space O(V^2).
//
// Notes:
//   - Unweighted mode writes 1 for present edges; weighted mode uses edge weights.
//
// AI-Hints:
//   - Use MetricClosure to turn adjacency into distances (+Inf as unreachable), diag forced to 0.
func BuildDenseAdjacency(
	vertices []string,
	edges []*core.Edge,
	opts Options,
) (map[string]int, *Dense, error) {
	// --- Stage 1: Validate vertices and build index map ---

	// vertices must exist (empty graph is a valid degenerate case only if we want 0×0),
	// but we adopt a strict policy here: empty vertex set is considered bad shape to
	// avoid accidental empty allocations downstream. Adjust if you need 0×0 matrices.
	if len(vertices) == 0 {
		return nil, nil, fmt.Errorf("BuildDenseAdjacency: empty vertex set: %w", ErrInvalidDimensions)
	}
	V := len(vertices)

	// Build stable vertex→index mapping with linear scan in provided order.
	idx := make(map[string]int, V)
	var i int
	var id string
	for i, id = range vertices {
		// Defensive duplicate check.
		if _, dup := idx[id]; dup {
			return nil, nil, fmt.Errorf("BuildDenseAdjacency: duplicate vertex id %q: %w", id, ErrUnknownVertex)
		}
		idx[id] = i
	}

	// --- Stage 2: Allocate dense V×V and decide weight policy ---
	mat, err := NewDense(V, V)
	if err != nil {
		return nil, nil, fmt.Errorf("BuildDenseAdjacency: NewDense(%d,%d): %w", V, V, err)
	}

	// Determine whether we should use actual edge weights.
	// useWeight starts from opts.Weighted but may be degraded to false if the input
	// graph is effectively unweighted (all weights are 0, or graph flags say unweighted).
	useWeight := opts.weighted
	if useWeight && allZeroWeights(edges) {
		// If the edge slice is present, scan it for a non-zero weight.
		// Early exit on the first non-zero to keep it O(E) best-case.
		useWeight = false // degrade to binary if effectively unweighted
	}

	// --- Stage 3: Populate adjacency entries (deterministic) ---
	directed := opts.directed
	allowMulti := opts.allowMulti
	allowLoops := opts.allowLoops

	// First-edge-wins set when AllowMulti=false.
	// For directed graphs key=(src,dst). For undirected, we normalize to {min,max}.
	seen := make(map[pairKey]struct{}, 64)

	var (
		e        *core.Edge
		ej       int
		src, dst int
		w        float64
		key      pairKey
	)

	for ej = 0; ej < len(edges); ej++ {
		e = edges[ej]
		// Resolve endpoints
		if src, err = lookupIndex(idx, e.From); err != nil {
			return nil, nil, fmt.Errorf("BuildDenseAdjacency: %w", err)
		}
		if dst, err = lookupIndex(idx, e.To); err != nil {
			return nil, nil, fmt.Errorf("BuildDenseAdjacency: %w", err)
		}
		// Loops policy
		if src == dst && !allowLoops {
			continue
		}
		// Multi-edge policy
		if !allowMulti {
			if directed {
				key = orderedPair(src, dst)
			} else {
				key = unorderedPair(src, dst)
			}
			if _, dup := seen[key]; dup {
				// Skip duplicate unordered pair, keep first.
				continue
			}
			// Mark unordered pair.
			seen[key] = struct{}{}
		}

		// Decide adjacency cell value for this edge:
		//   - if useWeight, we preserve float64(e.Weight);
		//   - otherwise we write 1 (binary).
		// NOTE: we do not reject zero-weight *edges* here; under "weighted mode"
		// the earlier degradation logic switches us to binary if all were zero.
		if useWeight {
			w = float64(e.Weight)
			if math.IsNaN(w) || math.IsInf(w, 0) {
				return nil, nil, fmt.Errorf("BuildDenseAdjacency: invalid weight for %q->%q: %w", e.From, e.To, ErrInvalidWeight)
			}
		} else {
			w = defaultWeight
		}

		// Write adjacency cell [src,dst]
		if err = mat.Set(src, dst, w); err != nil {
			return nil, nil, fmt.Errorf("BuildDenseAdjacency: Set(%d,%d): %w", src, dst, err)
		}
		// Mirror for undirected (not for loops)
		if !directed && src != dst {
			if err = mat.Set(dst, src, w); err != nil {
				return nil, nil, fmt.Errorf("BuildDenseAdjacency: Set(%d,%d): %w", dst, src, err)
			}
		}
	}

	// --- Stage 4: Optional metric closure (APSP) ---
	if opts.metricClose {
		// Convert adjacency (0 / weight) into distance matrix:
		//  - diag = 0,
		//  - off-diagonal: 0 → +Inf (no edge), otherwise keep weight.
		if err = initDistancesInPlace(mat); err != nil {
			return nil, nil, fmt.Errorf("BuildDenseAdjacency: %w", err)
		}
		// Run Floyd–Warshall with fixed loop nests (i,k,j) for determinism.
		floydWarshallInPlace(mat)
	} else {
		// In pure adjacency mode, ensure diagonal is 0 (no self-cost) and
		// leave off-diagonal zeros as "no edge".
		for i = 0; i < V; i++ {
			if err = mat.Set(i, i, 0.0); err != nil {
				return nil, nil, fmt.Errorf("BuildDenseAdjacency: Set(%d,%d,0): %w", i, i, err)
			}
		}
	}

	return idx, mat, nil
}

// BuildDenseIncidence CONSTRUCTS a dense incidence matrix from a vertex-id list and an edge list,
// applying Options policy deterministically.
// Implementation:
//   - Stage 1: validate vertex list and build VertexID→row index.
//   - Stage 2: compute effective column list (filter loops/multi deterministically).
//   - Stage 3: allocate V×E' dense (allow zero columns).
//   - Stage 4: populate columns with signed/undirected marks.
//
// Behavior highlights:
//   - Directed: −1 at source row, +1 at target row; directed self-loop ⇒ skipped column (not materialized).
//   - Undirected: +1 at both endpoints; undirected self-loop ⇒ +2 in the single row.
//   - DisallowMulti: first-edge-wins (ordered for directed; unordered for undirected).
//   - Columns preserve stable input order post filtering/dedup.
//
// Inputs:
//   - vertices: canonical vertex order (stable; caller decides lex order).
//   - edges: stable edge sequence (core contract: by Edge.ID asc).
//   - opts: Directed/AllowMulti/AllowLoops (Weighted is irrelevant for incidence).
//
// Returns:
//   - vidx: VertexID→row index.
//   - cols: effective column-aligned edges after filtering/dedup.
//   - mat: V×E' dense with entries in {−1,0,+1} (and +2 for undirected loops).
//   - err: ErrInvalidDimensions (empty vertices), ErrUnknownVertex, dense Set/shape errors.
//
// Determinism:
//   - Stable rows/columns given stable inputs/options.
//
// Complexity:
//   - Time O(V + E), Space O(V + E) plus V×E' for dense data.
//
// Notes:
//   - Directed self-loop is algebraically zero; the column is skipped for a minimal basis.
//
// AI-Hints:
//   - Use AllowMulti=false to get a canonical set of columns without duplicates.
func BuildDenseIncidence(
	vertices []string,
	edges []*core.Edge,
	opts Options,
) (map[string]int, []*core.Edge, *Dense, error) {
	// --- Stage 1: Validate and index ---

	// Empty vertex set is considered invalid shape for incidence (no rows).
	if len(vertices) == 0 {
		return nil, nil, nil, fmt.Errorf("BuildDenseIncidence: empty vertex set: %w", ErrInvalidDimensions)
	}
	V := len(vertices)
	// Build a stable vertex→row index map in the provided order; check duplicates defensively.
	idx := make(map[string]int, V)
	var i int
	var id string
	for i, id = range vertices {
		if _, dup := idx[id]; dup {
			return nil, nil, nil, fmt.Errorf("BuildDenseIncidence: duplicate vertex id %q: %w", id, ErrUnknownVertex)
		}
		idx[id] = i
	}

	// --- Stage 2: Compute effective column list deterministically ---

	// Deduplicate when AllowMulti=false using a pairKey set; directed uses ordered (u,v),
	// undirected uses unordered {min,max}. Keep the *first* occurrence (stable scan order).
	directed := opts.directed
	allowMulti := opts.allowMulti
	allowLoops := opts.allowLoops

	eff := make([]*core.Edge, 0, len(edges))
	seen := make(map[pairKey]struct{}, 64)

	// Stable single pass over edges to construct the effective column list.
	var (
		e    *core.Edge
		ej   int
		u, v int
		key  pairKey
		ok   bool
	)
	for ej = 0; ej < len(edges); ej++ {
		e = edges[ej] // address is safe (backed by the slice)
		// Resolve endpoints to row indices; unknown vertex is a hard error.
		if u, ok = idx[e.From]; !ok {
			return nil, nil, nil, fmt.Errorf("BuildDenseIncidence: unknown source %q: %w", e.From, ErrUnknownVertex)
		}
		if v, ok = idx[e.To]; !ok {
			return nil, nil, nil, fmt.Errorf("BuildDenseIncidence: unknown target %q: %w", e.To, ErrUnknownVertex)
		}

		// Loops policy:
		// - If loops are disallowed: skip.
		// - If directed and u==v: skip zero column entirely (not materialized).
		if u == v {
			if !allowLoops {
				continue // policy: ignore self-loops when AllowLoops=false
			}
			if directed {
				continue // skip directed self-loop column
			}
		}

		// Multi-edge policy (first-edge-wins when disallowed)
		if !allowMulti {
			if directed {
				// Directed: ordered pair (u,v).
				key = orderedPair(u, v)
			} else {
				key = unorderedPair(u, v)
			}
			if _, dup := seen[key]; dup {
				continue // ignore duplicate; keep the first occurrence (stable)
			}
			seen[key] = struct{}{} // record this pair as seen
		}
		// Append the edge pointer to the column list; order preserved (stable).
		eff = append(eff, e)
	}

	// --- Stage 3: Allocate V×E' dense ---
	Ep := len(eff)
	var mat *Dense
	var err error
	if Ep == 0 {
		// allow zero-column incidence
		if mat, err = newDenseZeroOK(V, 0); err != nil {
			return nil, nil, nil, fmt.Errorf("BuildDenseIncidence: newDenseZeroOK(%d,0): %w", V, err)
		}
	} else {
		if mat, err = NewDense(V, Ep); err != nil {
			return nil, nil, nil, fmt.Errorf("BuildDenseIncidence: NewDense(%d,%d): %w", V, Ep, err)
		}
	}

	// --- Stage 4: Populate columns ---

	// Fill one column per effective edge with the correct signs per policy.
	var j, su, sv int
	for j = 0; j < Ep; j++ {
		e = eff[j]
		su, _ = idx[e.From]
		sv, _ = idx[e.To]

		if directed {
			// Directed incidence: su!=sv (directed loops already skipped).
			if err = mat.Set(su, j, -1.0); err != nil { // mark source with −1.
				return nil, nil, nil, fmt.Errorf("BuildDenseIncidence: Set(%d,%d,-1): %w", su, j, err)
			}
			if err = mat.Set(sv, j, +1.0); err != nil { // mark target with +1.
				return nil, nil, nil, fmt.Errorf("BuildDenseIncidence: Set(%d,%d,+1): %w", sv, j, err)
			}
			continue
		}

		// Undirected incidence:
		if su == sv {
			// Self-loop contributes +2 in the single incident row.
			if err = mat.Set(su, j, 2.0); err != nil {
				return nil, nil, nil, fmt.Errorf("BuildDenseIncidence: Set(%d,%d,+2): %w", su, j, err)
			}
			continue
		}
		// Non-loop undirected: +1 at each endpoint.
		if err = mat.Set(su, j, 1.0); err != nil {
			return nil, nil, nil, fmt.Errorf("BuildDenseIncidence: Set(%d,%d,+1): %w", su, j, err)
		}
		if err = mat.Set(sv, j, 1.0); err != nil {
			return nil, nil, nil, fmt.Errorf("BuildDenseIncidence: Set(%d,%d,+1): %w", sv, j, err)
		}
	}

	return idx, eff, mat, nil
}
