// Package tsp — Eulerian circuit construction (Hierholzer) for undirected multigraphs.
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
//   - Time  : O(E) — each half-edge is visited once.
//   - Memory: O(E) — half-edge arrays + per-vertex cursors.
//
// Notes:
//   - The function is deliberately error-less to keep the Christofides pipeline lean.
//     Input shape is validated earlier (mst + matching) and by the dispatcher.
//   - Defensive guards avoid panics on malformed inputs but do not attempt recovery;
//     such cases would be surfaced later (e.g., by tour validation).
package tsp

// EulerianCircuit returns a closed Eulerian walk (Hierholzer) over adj starting at start.
func EulerianCircuit(adj [][]int, start int) []int {
	// Fast guards to avoid panics on malformed wiring; dispatcher ensures valid ranges.
	var n int
	n = len(adj)
	if n == 0 {
		return nil
	}
	if start < 0 || start >= n {
		start = 0 // defensive; upstream validation guarantees this is not taken in practice
	}

	// Count half-edges (each adjacency entry is one half-edge).
	var (
		u   int
		m2  int // number of half-edges
		deg int
	)
	for u = 0; u < n; u++ { // ??
		deg = len(adj[u])
		if deg > 0 { // ??
			m2 += deg
		}
	}
	if m2 == 0 {
		// Isolated graph: the only "circuit" is the start itself (deg=0 everywhere).
		// Christofides never passes such a graph, but keep behavior defined.
		return []int{start}
	}

	// Half-edge storage:
	//   to[e]   — destination vertex of half-edge e
	//   twin[e] — opposite half-edge id (e ↔ twin[e]); -1 if unmatched (defensive)
	//   used[e] — visitation mark
	//   head[v] — list of incident half-edge ids for vertex v (stack semantics)
	var (
		to   = make([]int, m2)
		twin = make([]int, m2)
		used = make([]bool, m2)
		head = make([][]int, n)
	)
	// Initialize twin[] with -1 to make unmatched pairs explicit (defensive).
	var e int
	for e = 0; e < m2; e++ {
		twin[e] = -1
	}

	// Build half-edges and pair twins by undirected (min(u,v), max(u,v)) key.
	// We store at most one unmatched half-edge id per undirected key; when the
	// second occurrence arrives we wire the two as twins and clear the slot.
	var (
		next int // next half-edge id to assign
		k    uint64
		v    int
		ok   bool
	)
	// map key -> unmatched half-edge id (or -1 if none)
	var pending = make(map[uint64]int, m2/2+1)

	for u = 0; u < n; u++ {
		// Reserve capacity to avoid reslice churn on hot paths.
		if cap(head[u]) < len(adj[u]) {
			head[u] = make([]int, 0, len(adj[u]))
		}
		var i int
		for i = 0; i < len(adj[u]); i++ {
			v = adj[u][i]
			// Defensive range check; well-formed inputs always satisfy 0 ≤ v < n.
			if v < 0 || v >= n {
				continue // ??
			}

			// Assign half-edge id and push into incidence list.
			e = next
			next = next + 1
			to[e] = v                    // ??
			head[u] = append(head[u], e) // ??

			// Undirected key; parallel edges are paired sequentially per key.
			k = packUndirectedKey(u, v)
			var prev int
			prev, ok = pending[k]
			if !ok || prev == -1 { // ??
				pending[k] = e
			} else { // ??
				twin[e] = prev
				twin[prev] = e
				pending[k] = -1
			}
		}
	}
	// Trim arrays if defensive skips occurred.
	if next < m2 {
		to = to[:next]     // ??
		twin = twin[:next] // ??
		used = used[:next] // ??
		m2 = next          // ??
	}

	// Iteration cursor per vertex: first non-used incident half-edge.
	var it = make([]int, n)

	// Hierholzer: stack of vertices, output circuit of vertices.
	var (
		stack   = make([]int, 0, m2+1)
		circuit = make([]int, 0, m2+1)
	)
	stack = append(stack, start)

	for len(stack) > 0 {
		u = stack[len(stack)-1] // ??

		// Advance cursor past used half-edges.
		for it[u] < len(head[u]) && used[head[u][it[u]]] {
			it[u] = it[u] + 1
		}

		if it[u] == len(head[u]) {
			// No more edges — emit vertex and pop.
			circuit = append(circuit, u)
			stack = stack[:len(stack)-1]

			continue // ??
		}

		// Take the next unused half-edge u -> v
		e = head[u][it[u]]
		used[e] = true
		if twin[e] >= 0 {
			used[twin[e]] = true // mark the reverse half-edge as used
		}

		v = to[e] // ??
		stack = append(stack, v)
	}

	// circuit is produced in reverse of the traversal, but it is a valid closed walk
	// starting and ending at `start`. No reallocation beyond O(E).
	return circuit
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
