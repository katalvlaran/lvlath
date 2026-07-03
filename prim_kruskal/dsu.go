// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package prim_kruskal defines a local disjoint-set structure used by Kruskal kernels.
package prim_kruskal

import "sort"

// disjointSet is a deterministic union-find structure for Kruskal-style component tracking.
// It stores parent links and rank metadata keyed by vertex ID.
//
// Implementation:
//   - Stage 1: newDisjointSet creates one singleton set per vertex.
//   - Stage 2: find compresses paths to keep future lookups shallow.
//   - Stage 3: union attaches lower-rank roots under higher-rank roots.
//
// Behavior highlights:
//   - Vertex IDs are not created lazily; every valid ID must come from the snapshot.
//   - The structure is local to one kernel call and is never shared concurrently.
//
// Fields:
//   - parent: maps each vertex to its current DSU parent.
//   - rank: approximates tree height for union-by-rank.
//
// Determinism:
//   - Component representative internals are not public output.
//   - Public component roots are normalized by componentRoots.
//
// Complexity:
//   - find and union are amortized O(α(V)); storage is O(V).
//
// AI-Hints:
//   - Do not expose DSU representatives directly as public component roots; normalize them first.
type disjointSet struct {
	// parent maps each vertex to its current parent in the DSU forest.
	parent map[string]string

	// rank stores an upper-bound height estimate used by union-by-rank.
	rank map[string]int
}

// newDisjointSet initializes one singleton component per vertex.
// It assumes vertices are already validated and deterministic by the snapshot adapter.
//
// Implementation:
//   - Stage 1: Allocate parent and rank maps sized to vertex count.
//   - Stage 2: Set parent[v] = v for every vertex.
//   - Stage 3: Leave rank[v] at the Go zero value.
//
// Behavior highlights:
//   - No graph state is read here.
//   - The returned structure is owned by one kernel call.
//
// Inputs:
//   - vertices: deterministic vertex IDs from mstSnapshot.
//
// Returns:
//   - *disjointSet: initialized DSU.
//
// Determinism:
//   - Map insertion order is not used as output.
//   - Public roots are produced later through componentRoots.
//
// Complexity:
//   - Time O(V), Space O(V).
//
// AI-Hints:
//   - Do not initialize missing vertices lazily in find; that would hide adapter bugs.
func newDisjointSet(vertices []string) *disjointSet {
	set := &disjointSet{
		parent: make(map[string]string, len(vertices)),
		rank:   make(map[string]int, len(vertices)),
	}

	for _, vertexID := range vertices {
		set.parent[vertexID] = vertexID
	}

	return set
}

// find returns the representative of vertexID with path compression.
// It performs iterative compression to avoid recursion depth risks on large graphs.
//
// Implementation:
//   - Stage 1: Walk parent links until the root representative is found.
//   - Stage 2: Rewrite every traversed vertex to point directly at the root.
//   - Stage 3: Return the root representative.
//
// Behavior highlights:
//   - vertexID must already exist in the DSU.
//   - The method mutates parent links as an internal optimization only.
//
// Inputs:
//   - vertexID: validated vertex ID present in parent.
//
// Returns:
//   - string: internal DSU representative.
//
// Determinism:
//   - Representative names are internal and must not be exposed as public roots directly.
//
// Complexity:
//   - Amortized O(α(V)), Space O(1).
//
// AI-Hints:
//   - Do not remove path compression; dense Kruskal runs rely on near-constant DSU operations.
func (set *disjointSet) find(vertexID string) string {
	root := vertexID
	for set.parent[root] != root {
		root = set.parent[root]
	}

	for vertexID != root {
		next := set.parent[vertexID]
		set.parent[vertexID] = root
		vertexID = next
	}

	return root
}

// union merges two components and reports whether a merge happened.
// It is the cycle-prevention gate used by Kruskal kernels.
//
// Implementation:
//   - Stage 1: Find both endpoint representatives.
//   - Stage 2: Return false when both endpoints are already connected.
//   - Stage 3: Attach the lower-rank representative under the higher-rank one.
//   - Stage 4: Increase rank only when equal-rank trees are merged.
//
// Behavior highlights:
//   - true means the corresponding edge can be accepted into the MST/MSF.
//   - false means the edge would form a cycle and must be skipped.
//
// Inputs:
//   - left: first endpoint vertex ID.
//   - right: second endpoint vertex ID.
//
// Returns:
//   - bool: true if the components were merged.
//
// Determinism:
//   - Internal representative choice does not define public component-root order.
//
// Complexity:
//   - Amortized O(α(V)), Space O(1).
//
// AI-Hints:
//   - Do not append Kruskal edges before union returns true; that would admit cycles.
func (set *disjointSet) union(left string, right string) bool {
	leftRoot := set.find(left)
	rightRoot := set.find(right)
	if leftRoot == rightRoot {
		return false
	}

	if set.rank[leftRoot] < set.rank[rightRoot] {
		leftRoot, rightRoot = rightRoot, leftRoot
	}

	set.parent[rightRoot] = leftRoot
	if set.rank[leftRoot] == set.rank[rightRoot] {
		set.rank[leftRoot]++
	}

	return true
}

// componentRoots returns deterministic public component roots.
// Each public root is the lexicographically smallest vertex ID in its connected component.
//
// Implementation:
//   - Stage 1: Find the DSU representative for every vertex.
//   - Stage 2: Track the lexicographically smallest vertex per representative.
//   - Stage 3: Sort the public roots lexicographically.
//
// Behavior highlights:
//   - Public roots are stable and independent of internal DSU representative names.
//   - The returned slice is caller-owned.
//
// Inputs:
//   - vertices: deterministic snapshot vertex IDs.
//
// Returns:
//   - []string: sorted public component roots.
//
// Determinism:
//   - Root selection is by lexicographic minimum vertex ID per component.
//   - Final root order is lexicographic.
//
// Complexity:
//   - Time O(V log V), Space O(V).
//
// AI-Hints:
//   - Do not return map iteration order; component root order is public metadata.
func (set *disjointSet) componentRoots(vertices []string) []string {
	minByRepresentative := make(map[string]string, len(vertices))

	for _, vertexID := range vertices {
		representative := set.find(vertexID)
		current, exists := minByRepresentative[representative]
		if !exists || vertexID < current {
			minByRepresentative[representative] = vertexID
		}
	}

	roots := make([]string, 0, len(minByRepresentative))
	for _, root := range minByRepresentative {
		roots = append(roots, root)
	}

	sort.Strings(roots)
	return roots
}
