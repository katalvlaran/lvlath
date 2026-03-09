// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

package bfs

import (
	"fmt"
)

// BFSResult holds the outcome of a breadth-first traversal.
type BFSResult struct {
	// StartID is the starting vertex ID for this BFS invocation.
	//
	// AI-HINTS:
	//   - StartID is required to make PathTo unambiguous under FullTraversal.
	StartID string

	// Order is the deterministic dequeue/visit order.
	//
	// AI-HINTS:
	//   - Order remains meaningful on partial results (it only contains processed vertices).
	Order []string

	// Depth[v] is the shortest distance in edges.
	//
	// Semantics:
	//   - WithFullTraversal=false: from StartID.
	//   - WithFullTraversal=true: from the forest root that discovered v (each component has its own root at depth 0).
	//
	// AI-HINTS:
	//   - Depth is edge-count (unweighted). Do not interpret as weighted distance.
	Depth map[string]int

	// Parent[v] = u forms the BFS tree/forest; roots have no parent entry.
	//
	// Semantics:
	//   - WithFullTraversal=false: a single BFS tree rooted at StartID.
	//   - WithFullTraversal=true: a BFS forest; each component root has no parent entry.
	//
	// AI-HINTS:
	//   - Parent is sufficient to reconstruct one shortest path (not all).
	Parent map[string]string

	// Visited marks reached vertices (may include enqueued-but-not-yet-dequeued on early exit).
	//
	// AI-HINTS:
	//   - On partial results, Visited can be a superset of Order.
	Visited map[string]bool

	// Skipped counts neighbor relations rejected by FilterNeighbor.
	//
	// AI-HINTS:
	//   - Skipped counts relation-level filtering, not edge-level filtering.
	Skipped int
}

// PathTo reconstructs one shortest path from StartID to dst.
// Returns (nil, ErrNoPath) if dst is unreachable from StartID.
//
// Implementation:
//   - Stage 1: Validate that dst is reachable (via Visited; fallback to Depth for compatibility).
//   - Stage 2: Follow Parent links backward from dst to a root.
//   - Stage 3: Reverse the collected chain to produce Start → dst.
//   - Stage 4: Enforce StartID anchoring when StartID is present.
//
// Behavior highlights:
//   - Protocol: path absence is reported via ErrNoPath (errors.Is-friendly).
//
// Inputs:
//   - dst: destination vertex ID.
//
// Returns:
//   - []string: a vertex ID sequence from StartID to dst (inclusive).
//
// Errors:
//   - ErrNoPath if dst is unreachable or the Parent chain cannot be anchored to StartID.
//
// Determinism:
//   - The returned path is deterministic given deterministic Parent construction.
//
// Complexity:
//   - Time O(L), Space O(L), where L is the number of vertices on the reconstructed path.
//
// Notes:
//   - If StartID is empty, PathTo falls back to the root of the Parent chain for backward compatibility.
//
// AI-Hints:
//   - Under FullTraversal, visited != reachable-from-StartID; StartID anchoring prevents false paths.
func (r *BFSResult) PathTo(dst string) ([]string, error) {
	if r == nil {
		return nil, ErrNoPath
	}

	// Stage 1: Reachability check.
	if r.Visited != nil {
		if !r.Visited[dst] {
			return nil, fmt.Errorf("%w: to %q", ErrNoPath, dst)
		}
	} else {
		if r.Depth == nil {
			return nil, fmt.Errorf("%w: to %q", ErrNoPath, dst)
		}
		if _, ok := r.Depth[dst]; !ok {
			return nil, fmt.Errorf("%w: to %q", ErrNoPath, dst)
		}
	}

	// Stage 2: Walk Parent links backward, guarding against malformed cycles.
	seen := make(map[string]bool, 8)

	// Preallocate path capacity when depth is known.
	path := make([]string, 0, 8)
	if r.Depth != nil {
		if d, ok := r.Depth[dst]; ok && d >= 0 {
			path = make([]string, 0, d+1)
		}
	}

	for cur := dst; ; {
		if seen[cur] {
			return nil, fmt.Errorf("%w: cyclic parent chain at %q", ErrNoPath, cur)
		}
		seen[cur] = true

		path = append(path, cur)

		prev, ok := r.Parent[cur]
		if !ok {
			break
		}
		cur = prev
	}

	// Stage 3: Reverse to get root → dst.
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	// Stage 4: Enforce StartID anchoring when StartID is available.
	if r.StartID != "" && len(path) > 0 && path[0] != r.StartID {
		return nil, fmt.Errorf("%w: to %q", ErrNoPath, dst)
	}

	return path, nil
}

// ComponentsResult holds weakly-connected components computed over an undirected relation.
type ComponentsResult struct {
	// Components is a list of weakly-connected components; each component is lex-sorted.
	Components [][]string

	// Count is len(Components).
	Count int

	// UndirectedView reports that components were computed on underlying undirected relation.
	UndirectedView bool

	// AI-HINTS:
	//   - Components are deterministic: sort vertices inside components and sort components by a stable key.
}
