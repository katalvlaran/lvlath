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
//   - The function is deliberately error-less to keep the Christofides pipeline lean.
//     Input shape is validated earlier (mst + matching) and by the dispatcher.
//   - Defensive guards avoid panics on malformed inputs but do not attempt recovery;
//     such cases would be surfaced later (e.g., by tour validation).
package tsp

// EulerianCircuit returns a closed Eulerian walk (Hierholzer) over adj starting at start.
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
		// Isolated graph: the only "circuit" is the start itself (deg=0 everywhere).
		// Christofides never passes such a graph, but keep behavior defined.
		//return []int{start}, nil
		return nil, ErrNonEulerian
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

	pending := make(map[uint64]int, halfEdgeCount/2+1)
	nextEdgeID := 0
	var previousEdgeID int
	var ok bool
	for fromVertex = 0; fromVertex < n; fromVertex++ {
		head[fromVertex] = make([]int, 0, len(adj[fromVertex]))

		for _, toVertex = range adj[fromVertex] {
			edgeID = nextEdgeID
			nextEdgeID++

			to[edgeID] = toVertex
			head[fromVertex] = append(head[fromVertex], edgeID)

			// Undirected key; parallel edges are paired sequentially per key.
			key := packUndirectedKey(fromVertex, toVertex)
			previousEdgeID, ok = pending[key]
			if !ok || previousEdgeID == -1 {
				pending[key] = edgeID
				continue
			}

			twin[edgeID] = previousEdgeID
			twin[previousEdgeID] = edgeID
			pending[key] = -1
		}
	}

	for _, unmatchedEdgeID := range pending {
		if unmatchedEdgeID != -1 {
			return nil, ErrNonEulerian
		}
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

	// circuit is produced in reverse of the traversal, but it is a valid closed walk
	// starting and ending at `start`. No reallocation beyond O(E).
	return circuit, nil
}

// packUndirectedKey encodes an undirected pair {u,v} as a uint64 key.
// The scheme supports vertex ids up to 2^32-1 (well beyond practical limits here).
func packUndirectedKey(u, v int) uint64 {
	var (
		a = uint64(u)
		b = uint64(v)
	)
	// Order the endpoints to make the key direction-agnostic.
	if a < b {
		return (a << 32) | b
	}

	return (b << 32) | a
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
