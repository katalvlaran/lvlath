// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package tsp - Eulerian circuit construction (Hierholzer) for undirected multigraphs.
//
// EulerianCircuit builds an Eulerian *circuit* of an undirected multigraph given
// as adjacency lists, starting and ending at `start`. This implementation uses a
// half-edge representation with explicit twin pointers, guaranteeing O(E) time
// (no quadratic splice/removal) and O(E) extra space.
//
// Preconditions (satisfied by Christofides pipeline after matching):
//   - adj encodes an undirected multigraph: for every entry u→v there exists a
//     corresponding v→u entry (parallel edges allowed).
//   - Every vertex has even degree (Eulerian condition).
//   - 0 ≤ start < len(adj). The dispatcher has already validated this.
//
// Postconditions:
//   - The returned sequence `walk` is a closed Eulerian circuit with
//     walk[0] == walk[len(walk)-1] == start and |walk| = E + 1 vertices.
//   - Deterministic w.r.t. the given adj order (no RNG, tie-breaks by adjacency order).
//
// Complexity:
//   - Time  : O(E) - each half-edge is visited once.
//   - Memory: O(E) - half-edge arrays + per-vertex cursors.
//
// Notes:
//   - The function is intentionally strict and returns sentinel-classified errors
//     for malformed non-Eulerian inputs.
//   - Christofides normalizes its multigraph before calling this function, but
//     EulerianCircuit still validates defensive invariants because it is reusable.
package tsp

import "sort"

// EulerianCircuit returns a closed Eulerian walk over an undirected multigraph adjacency
// list using Hierholzer's algorithm. Parallel edges are preserved and consumed exactly once.
//
// Implementation:
//   - Stage 1: Validate start and adjacency shape.
//   - Stage 2: Validate even degree for every vertex.
//   - Stage 3: Build reciprocal half-edge/twin metadata for the undirected multigraph.
//   - Stage 4: Run iterative Hierholzer traversal from start.
//   - Stage 5: Reverse the finished stack into walk order and verify all half-edges were consumed.
//
// Behavior highlights:
//   - Does not mutate adj.
//   - Supports parallel edges.
//   - Rejects malformed non-reciprocal half-edge representations.
//   - Returns a closed walk when at least one edge is present.
//
// Inputs:
//   - adj: undirected multigraph adjacency encoded as reciprocal half-edge lists.
//   - start: local vertex where the Eulerian walk starts and ends.
//
// Returns:
//   - []int: closed Eulerian walk.
//   - error: nil when adj is a valid Eulerian multigraph.
//
// Errors:
//   - ErrDimensionMismatch for empty adjacency or malformed endpoints.
//   - ErrInvalidVertex for invalid start.
//   - ErrNonEulerian for odd degree, missing reciprocal twins, disconnected edge components,
//     or incomplete edge consumption.
//
// Determinism:
//   - Uses deterministic adjacency order and stack traversal.
//
// Complexity:
//   - Time O(V+E), Space O(V+E).
//
// Notes:
//   - Christofides should call canonicalizeUndirectedMultigraph before this function.
//
// AI-Hints:
//   - Do not deduplicate parallel edges.
//   - Do not accept odd-degree vertices even when a partial walk exists.
func EulerianCircuit(adj [][]int, start int) ([]int, error) {
	// Fast guards to avoid panics on malformed wiring; dispatcher ensures valid ranges.
	n := len(adj)
	if n == 0 {
		return nil, ErrDimensionMismatch
	}
	if start < 0 || start >= n {
		//start = 0 // defensive; upstream validation guarantees this is not taken in practice
		return nil, ErrStartOutOfRange
	}

	halfEdgeCount := 0
	var fromVertex, toVertex, degree int
	for fromVertex = 0; fromVertex < n; fromVertex++ {
		degree = len(adj[fromVertex])
		if (degree & 1) == 1 {
			return nil, ErrNonEulerian
		}
		halfEdgeCount += degree

		for _, toVertex = range adj[fromVertex] {
			if toVertex < 0 || toVertex >= n {
				return nil, ErrDimensionMismatch
			}
		}
	}

	if halfEdgeCount == 0 {
		// Isolated graph: the unique zero-edge Eulerian circuit is the start vertex itself.
		// This keeps the defensive contract total for valid start vertices and avoids
		// special-case handling in callers that accept degenerate multigraphs.
		return []int{start}, nil
	}

	// Half-edge storage:
	//   to[e]   - destination vertex of half-edge e
	//   twin[e] - opposite half-edge id (e ↔ twin[e]); -1 if unmatched (defensive)
	//   used[e] - visitation mark
	//   head[v] - list of incident half-edge ids for vertex v (stack semantics)
	to := make([]int, halfEdgeCount)
	twin := make([]int, halfEdgeCount)
	used := make([]bool, halfEdgeCount)
	head := make([][]int, n)

	// Initialize twin[] with -1 to make unmatched pairs explicit (defensive).
	var edgeID int
	for edgeID = 0; edgeID < halfEdgeCount; edgeID++ {
		twin[edgeID] = -1
	}

	pending := make(map[uint64][]int, halfEdgeCount)

	nextEdgeID := 0
	for fromVertex = 0; fromVertex < n; fromVertex++ {
		head[fromVertex] = make([]int, 0, len(adj[fromVertex]))

		for _, toVertex = range adj[fromVertex] {
			if fromVertex == toVertex {
				return nil, ErrNonEulerian
			}

			edgeID = nextEdgeID
			nextEdgeID++

			to[edgeID] = toVertex
			head[fromVertex] = append(head[fromVertex], edgeID)

			oppositeKey := packDirectedHalfEdgeKey(toVertex, fromVertex)
			oppositeEdges := pending[oppositeKey]

			if len(oppositeEdges) > 0 {
				twinID := oppositeEdges[len(oppositeEdges)-1]
				oppositeEdges = oppositeEdges[:len(oppositeEdges)-1]

				if len(oppositeEdges) == 0 {
					delete(pending, oppositeKey)
				} else {
					pending[oppositeKey] = oppositeEdges
				}

				twin[edgeID] = twinID
				twin[twinID] = edgeID
				continue
			}

			key := packDirectedHalfEdgeKey(fromVertex, toVertex)
			pending[key] = append(pending[key], edgeID)
		}
	}

	if len(pending) != 0 {
		return nil, ErrNonEulerian
	}

	cursor := make([]int, n)
	stack := make([]int, 0, halfEdgeCount+1)
	circuit := make([]int, 0, halfEdgeCount+1)

	var twinID int
	stack = append(stack, start)

	for len(stack) > 0 {
		fromVertex = stack[len(stack)-1]

		for cursor[fromVertex] < len(head[fromVertex]) && used[head[fromVertex][cursor[fromVertex]]] {
			cursor[fromVertex]++
		}

		if cursor[fromVertex] == len(head[fromVertex]) {
			// No more edges - emit vertex and pop.
			circuit = append(circuit, fromVertex)
			stack = stack[:len(stack)-1]
			continue
		}

		edgeID = head[fromVertex][cursor[fromVertex]]
		twinID = twin[edgeID]
		if twinID < 0 {
			return nil, ErrNonEulerian
		}

		used[edgeID] = true
		used[twinID] = true

		stack = append(stack, to[edgeID])
	}

	if len(circuit) == 0 || circuit[0] != start || circuit[len(circuit)-1] != start {
		return nil, ErrNonEulerian
	}

	for edgeID = 0; edgeID < halfEdgeCount; edgeID++ {
		if !used[edgeID] {
			return nil, ErrNonEulerian
		}
	}

	if len(circuit) != halfEdgeCount/2+1 {
		return nil, ErrNonEulerian
	}

	// circuit is produced in reverse of the traversal, but it is a valid closed walk
	// starting and ending at `start`. No reallocation beyond O(E).
	return circuit, nil
}

// packDirectedHalfEdgeKey encodes an ordered half-edge direction u->v.
// Unlike packUndirectedKey, endpoint order is significant; this is required when
// pairing reciprocal half-edges for parallel undirected edges.
//
// Implementation:
//   - Stage 1: Store source vertex in the high 32 bits.
//   - Stage 2: Store destination vertex in the low 32 bits.
//
// Behavior highlights:
//   - Pure helper.
//   - Distinguishes u->v from v->u.
//   - Supports correct twin pairing for parallel edges.
//
// Inputs:
//   - u: source local vertex ID.
//   - v: destination local vertex ID.
//
// Returns:
//   - uint64: directed half-edge key.
//
// Errors:
//   - None. Caller must validate endpoint bounds.
//
// Determinism:
//   - Pure bit packing.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Vertex IDs above 2^32-1 are outside the packing contract.
//
// AI-Hints:
//   - Do not replace this with packUndirectedKey in Eulerian half-edge pairing.
//   - Twin pairing requires opposite directed keys.
func packDirectedHalfEdgeKey(u int, v int) uint64 {
	return uint64(uint32(u))<<32 | uint64(uint32(v))
}

// ShortcutEulerianToHamiltonian converts an Eulerian vertex sequence (with revisits)
// into a Hamiltonian cycle by skipping the first revisits and then closing the tour.
// This is the standard “shortcutting” step in Christofides:
//
//	Input:  euler - a vertex sequence of arbitrary length (often O(E)).
//	        n     - number of unique vertices (0..n-1).
//	        start - required starting vertex of the resulting tour.
//
// Algorithm:
//   - Maintain a visited[n] boolean array.
//   - Scan euler left-to-right; append a vertex v the first time it is seen.
//   - After the scan, ensure every vertex 0..n-1 was seen exactly once.
//   - Rotate the resulting n-length cycle so it starts at `start` and close it.
//
// Contracts:
//   - 0 ≤ v < n for every v ∈ euler; otherwise ErrDimensionMismatch.
//   - start ∈ [0..n-1].
//
// Returns:
//   - tour of length n+1 with tour[0]==tour[n]==start,
//   - ErrDimensionMismatch if euler misses some vertices or has out-of-range entries,
//   - ErrStartOutOfRange if start is invalid.
//
// Complexity: O(len(euler) + n) time, O(n) space.
func ShortcutEulerianToHamiltonian(euler []int, n int, start int) ([]int, error) {
	if n <= 0 {
		return nil, ErrDimensionMismatch
	}
	if start < 0 || start >= n {
		return nil, ErrStartOutOfRange
	}

	visited := make([]bool, n)
	cycle := make([]int, 0, n) // collect first occurrences

	var (
		idx int
		v   int
	)
	for idx = 0; idx < len(euler); idx++ {
		v = euler[idx]
		if v < 0 || v >= n {
			return nil, ErrDimensionMismatch
		}
		if !visited[v] {
			visited[v] = true
			cycle = append(cycle, v)
		}
	}

	// Ensure all vertices were seen exactly once.
	if len(cycle) != n {
		return nil, ErrDimensionMismatch
	}
	var i int
	for i = 0; i < n; i++ {
		if !visited[i] {
			return nil, ErrDimensionMismatch
		}
	}

	// Rotate to start and close.
	var p = -1
	for i = 0; i < n; i++ {
		if cycle[i] == start {
			p = i
			break
		}
	}
	if p == -1 {
		return nil, ErrDimensionMismatch
	}

	tour := make([]int, n+1)
	for i = 0; i < n; i++ {
		tour[i] = cycle[(p+i)%n]
	}
	tour[n] = start
	return tour, nil
}

// canonicalizeUndirectedMultigraph rebuilds a reciprocal undirected multigraph adjacency
// from half-edge adjacency lists. It is used by Christofides before EulerianCircuit
// so the Eulerian stage receives a clean reciprocal multigraph representation.
//
// Implementation:
//   - Stage 1: Validate vertex bounds while counting undirected half-edges by canonical key.
//   - Stage 2: Reject odd half-edge counts per undirected key.
//   - Stage 3: Rebuild adjacency by emitting count/2 reciprocal undirected edges.
//   - Stage 4: Reject odd final degrees.
//
// Behavior highlights:
//   - Preserves parallel edges.
//   - Does not use map iteration for output order.
//   - Repairs harmless reciprocal-order drift in mutable adjacency.
//   - Rejects genuinely malformed odd half-edge counts.
//
// Inputs:
//   - adj: undirected multigraph adjacency encoded as half-edge lists.
//
// Returns:
//   - [][]int: canonical reciprocal adjacency.
//   - error: nil when the multigraph can represent an Eulerian undirected graph.
//
// Errors:
//   - ErrDimensionMismatch for out-of-range endpoints.
//   - ErrNonEulerian for self-loops, odd half-edge counts, or odd final degrees.
//
// Determinism:
//   - Keys are sorted before adjacency emission.
//
// Complexity:
//   - Time O(E log E), Space O(E).
//
// Notes:
//   - Christofides has O(n^2) matrix stages, so O(E log E) here is negligible.
//   - EulerianCircuit remains strict; this helper belongs to the Christofides pipeline.
//
// AI-Hints:
//   - Do not deduplicate parallel edges.
//   - Do not sort each adjacency list independently after emission; key order is the deterministic policy.
func canonicalizeUndirectedMultigraph(adj [][]int) ([][]int, error) {
	n := len(adj)
	if n == 0 {
		return nil, ErrDimensionMismatch
	}

	counts := make(map[uint64]int)
	keys := make([]uint64, 0)

	for from := 0; from < n; from++ {
		for _, to := range adj[from] {
			if to < 0 || to >= n {
				return nil, ErrDimensionMismatch
			}
			if from == to {
				return nil, ErrNonEulerian
			}

			key := canonicalUndirectedKey(from, to)
			if _, exists := counts[key]; !exists {
				keys = append(keys, key)
			}
			counts[key]++
		}
	}

	sort.Slice(keys, func(i int, j int) bool {
		return keys[i] < keys[j]
	})

	out := make([][]int, n)

	for _, key := range keys {
		count := counts[key]
		if (count & 1) == 1 {
			return nil, ErrNonEulerian
		}

		u, v := unpackCanonicalUndirectedKey(key)
		multiplicity := count / 2

		for edge := 0; edge < multiplicity; edge++ {
			out[u] = append(out[u], v)
			out[v] = append(out[v], u)
		}
	}

	for vertex := 0; vertex < n; vertex++ {
		if (len(out[vertex]) & 1) == 1 {
			return nil, ErrNonEulerian
		}
	}

	return out, nil
}

// canonicalUndirectedKey encodes an unordered vertex pair {u,v} into the same stable
// uint64 format used by Eulerian and Christofides multigraph normalization.
//
// Implementation:
//   - Stage 1: Order endpoints so u <= v.
//   - Stage 2: Pack the ordered endpoints into a uint64 key.
//
// Behavior highlights:
//   - Pure helper.
//   - Keeps canonicalization code independent from adjacency order.
//   - Equivalent in behavior to packUndirectedKey.
//
// Inputs:
//   - u: first local vertex ID.
//   - v: second local vertex ID.
//
// Returns:
//   - uint64: canonical undirected pair key.
//
// Errors:
//   - None. Caller must validate endpoint bounds.
//
// Determinism:
//   - Endpoint order does not affect output.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Prefer one key helper internally to avoid duplicate semantics.
//
// AI-Hints:
//   - Consider implementing this as a thin wrapper over packUndirectedKey.
func canonicalUndirectedKey(u int, v int) uint64 {
	if u > v {
		u, v = v, u
	}

	return uint64(uint32(u))<<32 | uint64(uint32(v))
}

// unpackCanonicalUndirectedKey decodes a canonical uint64 undirected-pair key into
// ordered endpoints u<=v.
//
// Implementation:
//   - Stage 1: Read the high 32 bits as u.
//   - Stage 2: Read the low 32 bits as v.
//   - Stage 3: Convert both endpoints back to int.
//
// Behavior highlights:
//   - Pure helper.
//   - Assumes the key was created by canonicalUndirectedKey or packUndirectedKey.
//   - Does not validate the decoded endpoints against a graph size.
//
// Inputs:
//   - key: packed canonical undirected pair.
//
// Returns:
//   - int: lower endpoint.
//   - int: higher endpoint.
//
// Errors:
//   - None.
//
// Determinism:
//   - Pure bit unpacking.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - The decoded pair is ordered by construction.
//
// AI-Hints:
//   - Do not use this for directed edge IDs.
func unpackCanonicalUndirectedKey(key uint64) (int, int) {
	return int(uint32(key >> 32)), int(uint32(key))
}
