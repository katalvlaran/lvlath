// Package tsp_test verifies the odd-vertex matching step via test-only hooks.
// We stay in tsp_test (external view) and use tsp.TestHookGreedyMatch /
// tsp.TestHookBlossomMatch which exist only under `go test`.
package tsp_test

import (
	"errors"
	"testing"

	"github.com/katalvlaran/lvlath/tsp"
)

// hasUndirectedEdge returns true iff adj contains exactly one u–v and one v–u entry.
// This checks that a single undirected edge was added in the multigraph.
func hasUndirectedEdge(adj [][]int, u, v int) bool {
	return countDir(adj, u, v) == 1 && countDir(adj, v, u) == 1
}

// countDir counts how many times v appears in adj[u] (parallel edges allowed).
func countDir(adj [][]int, u, v int) int {
	var c int
	var i int
	for i = 0; i < len(adj[u]); i++ { // scan row u explicitly (no short var decls in loop)
		if adj[u][i] == v {
			c++
		}
	}
	return c
}

// equalAdj checks structural equality of two adjacency lists (order-sensitive).
func equalAdj(a, b [][]int) bool {
	var rowsA, rowsB = len(a), len(b)
	var rowsAi, rowsBi int // entries len
	if rowsA != rowsB {
		return false
	}
	var i, j int                // loop iterators
	for i = 0; i < rowsA; i++ { // compare each row
		rowsAi, rowsBi = len(a[i]), len(b[i])
		if rowsAi != rowsBi { // row length must match
			return false
		}
		for j = 0; j < rowsAi; j++ { // compare entries
			if a[i][j] != b[i][j] {
				return false
			}
		}
	}

	return true
}

// -----------------------------------------------------------------------------
// 1) K4 with a unique optimum: greedy must pick {0–1, 2–3} over any cross pairs.
// -----------------------------------------------------------------------------

func TestGreedyMatch_K4_UniquePairs(t *testing.T) {
	// Symmetric 4×4 metric with unique optimal pairs.
	a := [][]float64{
		{0, 1, 5, 5},
		{1, 0, 5, 5},
		{5, 5, 0, 1},
		{5, 5, 1, 0},
	}
	m := testDense{a: a}

	odd := []int{0, 1, 2, 3} // even-size odd set
	adj := make([][]int, 4)  // start with empty simple graph
	tsp.TestHookGreedyMatch(odd, m, adj)

	// Expect exactly (0–1) and (2–3).
	if !hasUndirectedEdge(adj, 0, 1) {
		t.Fatalf("missing or duplicated edge 0–1 in adjacency: %+v", adj)
	}
	if !hasUndirectedEdge(adj, 2, 3) {
		t.Fatalf("missing or duplicated edge 2–3 in adjacency: %+v", adj)
	}
	// No cross edges should exist.
	if countDir(adj, 0, 2)+countDir(adj, 2, 0)+countDir(adj, 1, 3)+countDir(adj, 3, 1) != 0 {
		t.Fatalf("unexpected cross edges present: %+v", adj)
	}
}

// -----------------------------------------------------------------------------
// 2) Tiebreak determinism on K6 with equal weights.
// Greedy pops u from the tail of 'odd' and pairs with the smallest-id partner.
// With odd=[0,1,2,3,4,5] the deterministic result is (5–0),(3–1),(2–4).
// (Earlier expectation (4–1),(3–2) was incorrect — see explanation.)
// -----------------------------------------------------------------------------

func TestGreedyMatch_K6_TieBreakDeterminism(t *testing.T) {
	const n = 6
	// All off-diagonal distances equal; diagonal is zero.
	a := make([][]float64, n)
	var i, j int
	for i = 0; i < n; i++ {
		a[i] = make([]float64, n)
		for j = 0; j < n; j++ {
			if i == j {
				a[i][j] = 0
			} else {
				a[i][j] = 1
			}
		}
	}
	m := testDense{a: a}

	// Expected deterministic mapping under the implemented policy:
	// u pops from the tail; among remaining, pick smallest-id partner.
	// Pairs: (5–0), (3–1), (2–4).
	var neigh = [n]int{5, 3, 4, 1, 2, 0} // neighbor[u] = v

	var rep int
	for rep = 0; rep < 3; rep++ { // repeat to lock determinism
		odd := []int{0, 1, 2, 3, 4, 5}
		adj := make([][]int, n)

		tsp.TestHookGreedyMatch(odd, m, adj)

		// Each vertex must have degree 1 and neighbor equal to the expected one.
		var u int
		for u = 0; u < n; u++ {
			if len(adj[u]) != 1 {
				t.Fatalf("deg[%d]=%d; want 1; adj=%+v", u, len(adj[u]), adj)
			}
			if adj[u][0] != neigh[u] {
				t.Fatalf("unexpected partner for %d: got %d, want %d; adj=%+v",
					u, adj[u][0], neigh[u], adj)
			}
		}
	}
}

// -----------------------------------------------------------------------------
// 3) Blossom placeholder: must return ErrMatchingNotImplemented AND not mutate.
// -----------------------------------------------------------------------------

func TestBlossomMatch_Sentinel_NoMutation(t *testing.T) {
	// Tiny symmetric matrix; actual values irrelevant to the sentinel.
	a := [][]float64{
		{0, 1, 1},
		{1, 0, 1},
		{1, 1, 0},
	}
	m := testDense{a: a}

	odd := []int{0, 1} // any even-sized subset
	adj := [][]int{
		{1},
		{0, 2},
		{1},
	}
	// Deep copy adjacency for equality checks.
	before := make([][]int, len(adj))
	var r int
	for r = 0; r < len(adj); r++ {
		before[r] = append([]int(nil), adj[r]...)
	}

	err := tsp.TestHookBlossomMatch(odd, m, adj)
	if !errors.Is(err, tsp.ErrMatchingNotImplemented) {
		t.Fatalf("want ErrMatchingNotImplemented, got %v", err)
	}
	if !equalAdj(before, adj) { // no mutation allowed
		t.Fatalf("adjacency mutated by blossomMatch; before=%+v after=%+v", before, adj)
	}
}
