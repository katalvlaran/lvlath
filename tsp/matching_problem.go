// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package tsp defines the local matching problem used by Christofides.
// The model copies odd-set distances into a deterministic local complete graph
// so matching engines do not depend on external matrix ownership or map order.
package tsp

import (
	"math"
	"math/bits"

	"github.com/katalvlaran/lvlath/matrix"
)

const (
	// matchingUnmatched marks a local matching vertex that has not been paired.
	matchingUnmatched = -1

	// maxExactMatchingOddVertices bounds the internal exponential oracle.
	// The oracle is used only for small-instance verification and optional micro-fast-paths;
	// it must not be the general BlossomMatch engine.
	maxExactMatchingOddVertices = 22
)

// matchingProblem stores the complete graph induced by odd-degree vertices.
// Vertex indices in this structure are local [0..k-1] positions; odd maps them
// back to original TSP matrix vertices.
//
// Implementation:
//   - Stage 1: buildMatchingProblem validates odd-set shape and copies costs.
//   - Stage 2: exactSmallPerfectMatching solves over local indices.
//   - Stage 3: appendPerfectMatching maps local pairs back to original vertices.
//
// Behavior highlights:
//   - Detached from the source matrix.
//   - Row-major local cost layout: w[i*n+j].
//   - Does not mutate Christofides adjacency.
//
// Inputs:
//   - Built from odd-degree MST vertices and the final symmetric distance matrix.
//
// Returns:
//   - Consumed by exact matching helpers.
//
// Errors:
//   - Construction errors are returned by buildMatchingProblem.
//
// Determinism:
//   - Local vertex order preserves the odd slice order produced by MST degree scan.
//
// Complexity:
//   - Space O(k^2), where k=len(odd).
//
// Notes:
//   - This is a matching-local data model, not a public graph representation.
//
// AI-Hints:
//   - Do not expose matchingProblem from public APIs.
//   - Do not reorder odd vertices unless all tie-break tests are updated.
type matchingProblem struct {
	// odd maps local matching indices back to original TSP matrix vertices.
	// odd[local] is used only when appending final matching edges to Christofides adjacency.
	odd []int

	// n is the number of local odd-degree vertices.
	// It must be even for a perfect matching.
	n int

	// w stores local edge costs in row-major n*n layout.
	// w[i*n+j] is the cost between local vertices i and j.
	w []float64
}

// at returns the local matching cost between two local odd-set vertices.
//
// Implementation:
//   - Stage 1: Compute row-major offset i*n+j.
//   - Stage 2: Return the stored cost.
//
// Behavior highlights:
//   - No allocation.
//   - No matrix interface dispatch.
//
// Inputs:
//   - i: local source index.
//   - j: local target index.
//
// Returns:
//   - float64: local complete-graph matching edge cost.
//
// Errors:
//   - None. Callers must pass validated local indices.
//
// Determinism:
//   - Pure indexed lookup.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Diagonal values are unused by the matching DP.
//
// AI-Hints:
//   - Do not use original TSP vertex IDs as indices here; use local indices.
func (p matchingProblem) at(i int, j int) float64 {
	return p.w[i*p.n+j]
}

// matchingCost computes the exact cost of a verified local perfect matching.
// It sums every pair once, using i<match[i] to avoid double-counting symmetric pairs.
//
// Implementation:
//   - Stage 1: Verify the local matching structure.
//   - Stage 2: Scan local vertices in increasing order.
//   - Stage 3: Sum problem.at(i, match[i]) only for i<match[i].
//   - Stage 4: Stabilize the returned cost with round1e9.
//
// Behavior highlights:
//   - Does not mutate problem or match.
//   - Does not inspect adjacency.
//   - Uses the matching-local detached cost matrix.
//
// Inputs:
//   - problem: local complete matching instance.
//   - match: local symmetric perfect matching array.
//
// Returns:
//   - float64: rounded matching cost.
//   - error: nil on valid perfect matching.
//
// Errors:
//   - ErrInvalidMatching from verifyPerfectMatching.
//
// Determinism:
//   - Fixed increasing local-index scan.
//
// Complexity:
//   - Time O(k), Space O(1).
//
// Notes:
//   - The cost is used for verification and tests; Christofides does not need it for adjacency mutation.
//
// AI-Hints:
//   - Do not sum both i→j and j→i.
//   - Do not recompute costs from the original matrix here.
func matchingCost(problem matchingProblem, match []int) (float64, error) {
	if err := verifyPerfectMatching(match); err != nil {
		return 0, err
	}
	if len(problem.odd) != len(match) || problem.n != len(match) {
		return 0, ErrInvalidMatching
	}

	total := 0.0

	for localVertex, partner := range match {
		if localVertex < partner {
			total += problem.at(localVertex, partner)
		}
	}

	return round1e9(total), nil
}

// buildMatchingProblem builds the local complete graph induced by odd vertices.
// It validates local matching shape and copies costs from the shared TSP weight firewall.
//
// Implementation:
//   - Stage 1: Validate even cardinality and copy odd vertices.
//   - Stage 2: Copy the final symmetric complete distance matrix through copyCompleteWeights.
//   - Stage 3: Reject duplicate or out-of-range odd vertices.
//   - Stage 4: Fill a k×k local row-major cost buffer.
//
// Behavior highlights:
//   - Preserves odd order for deterministic tie-breaking.
//   - Rejects malformed odd sets before any adjacency mutation.
//   - Uses weightBuffer instead of repeated matrix.At calls.
//
// Inputs:
//   - odd: odd-degree vertices from MST degree scan.
//   - dist: final symmetric complete TSP matrix.
//
// Returns:
//   - matchingProblem: detached local MWPM instance.
//   - error: nil on success or sentinel-classified failure.
//
// Errors:
//   - ErrInvalidMatching for odd cardinality, duplicates, or out-of-range vertices.
//   - ErrNilDistanceMatrix, ErrNonSquare, ErrNaNInf, ErrNegativeWeight,
//     ErrIncompleteGraph, ErrAsymmetry from copyCompleteWeights.
//
// Determinism:
//   - Fixed increasing local-index scan.
//
// Complexity:
//   - Time O(n^2+k^2), Space O(n^2+k^2).
//
// Notes:
//   - The O(n^2) matrix copy is acceptable because Christofides already runs over dense matrices.
//
// AI-Hints:
//   - Do not sort odd vertices here; MST degree scan already defines deterministic order.
func buildMatchingProblem(odd []int, dist matrix.Matrix) (matchingProblem, error) {
	oddCount := len(odd)
	// Handle the edge case where no odd-degree vertices exist (the MST is already Eulerian).
	if oddCount == 0 {
		return matchingProblem{
			odd: append([]int(nil), odd...),
			n:   0,
			w:   nil,
		}, nil
	}
	if (oddCount & 1) == 1 {
		return matchingProblem{}, ErrInvalidMatching
	}

	// Unpack and validate the global distance model into a continuous local matrix layout.
	// This ensures Numeric Law compliance (no NaN/Inf or negative weights) before matching begins.
	weights, err := copyCompleteWeights(dist, true)
	if err != nil {
		return matchingProblem{}, err
	}

	seen := make([]bool, weights.n)
	localOdd := append([]int(nil), odd...)

	// Pre-declare iteration variables to bypass re-allocation overhead inside the layout loop.
	var (
		localIndex int
		vertex     int
	)
	for localIndex, vertex = range localOdd {
		// Bounds validation: verify that the odd vertex index aligns with the global matrix scope.
		if vertex < 0 || vertex >= weights.n {
			return matchingProblem{}, ErrInvalidMatching
		}
		// Integrity check: fail early if any vertex appears twice in the odd-degree subset.
		if seen[vertex] {
			return matchingProblem{}, ErrInvalidMatching
		}

		// Track vertex registration and save it to the detached local sequence.
		seen[vertex] = true
		localOdd[localIndex] = vertex
	}

	problem := matchingProblem{
		odd: localOdd,
		n:   oddCount,
		w:   make([]float64, oddCount*oddCount),
	}

	// Map the global distances into a tight, row-major k×k local matrix for localized solver caches.
	var (
		row int
		col int
	)
	for row = 0; row < oddCount; row++ {
		for col = 0; col < oddCount; col++ {
			// Skip the diagonal since self-loop matching (matching a vertex to itself) is forbidden.
			if row == col {
				continue
			}

			// Translate local sub-graph indices into global coordinates to pull edge costs.
			problem.w[row*oddCount+col] = weights.at(localOdd[row], localOdd[col])
		}
	}

	return problem, nil
}

// exactSmallPerfectMatching computes exact MWPM by subset dynamic programming.
// It is an internal exponential oracle for tests and tiny fast-paths, not the
// general BlossomMatch engine.
//
// Implementation:
//   - Stage 1: Reject malformed local matching problems.
//   - Stage 2: Reject inputs above maxExactMatchingOddVertices unless a caller explicitly gates them.
//   - Stage 3: DP over even-cardinality masks using the first unmatched local vertex.
//   - Stage 4: Reconstruct and verify the symmetric match array.
//
// Behavior highlights:
//   - Exact for k <= maxExactMatchingOddVertices.
//   - Deterministic tie behavior: first local vertex, then increasing partner.
//   - Must not be used as the general production path for BlossomMatch.
//
// Inputs:
//   - problem: local complete matching instance.
//
// Returns:
//   - []int: match[i]=j local-index pairing.
//   - float64: exact total matching cost.
//   - error: nil on success or sentinel-classified failure.
//
// Errors:
//   - ErrInvalidMatching for malformed local problem.
//   - ErrIncompleteGraph when no perfect matching exists.
//
// Determinism:
//   - Fixed mask order, fixed first-unmatched search, fixed increasing partner scan.
//
// Complexity:
//   - Time O(k^2 * 2^k), Space O(2^k+k).
//
// Notes:
//   - Production BlossomMatch must not rely on this function for large k.
//
// AI-Hints:
//   - Do not advertise this function as Blossom.
//   - Do not raise maxExactMatchingOddVertices without benchmark and memory review.
func exactSmallPerfectMatching(problem matchingProblem) ([]int, float64, error) {
	oddCount := problem.n
	// Edge case: an empty set of odd vertices requires no matching layout.
	if oddCount == 0 {
		return []int{}, 0, nil
	}
	// Validate invariants: a perfect matching is mathematically impossible for an odd number of vertices.
	// Also ensures structural integrity of the flat-matrix problem slices.
	if (oddCount&1) == 1 || len(problem.odd) != oddCount || len(problem.w) != oddCount*oddCount {
		return nil, 0, ErrInvalidMatching
	}
	// Safeguard firewall: prevents exponential time O(k^2 * 2^k) and memory blowup.
	if oddCount > maxExactMatchingOddVertices {
		return nil, 0, ErrInvalidMatching
	}

	// Initialize dynamic programming tables. Dimension equals 2^k subsets.
	totalMasks := 1 << uint(oddCount)
	dp := make([]float64, totalMasks)
	parent := make([][2]int, totalMasks)

	// Seed tables with default values. All states are initially unreachable (+Inf),
	// except the empty vertex subset (mask 0), whose coverage cost is exactly 0.
	var mask int
	for mask = range dp {
		dp[mask] = math.Inf(1)
		parent[mask] = [2]int{matchingUnmatched, matchingUnmatched}
	}
	dp[0] = 0

	// Pre-allocate iterator variables to eliminate overhead inside the hot loop.
	var (
		first            int
		partner          int
		maskWithoutFirst int
		nextMask         int
		cost             float64
		candidate        float64
	)
	for mask = 1; mask < totalMasks; mask++ {
		// Optimization: a perfect matching can only cover an even number of vertices.
		// Skip masks containing an odd count of set bits (active vertices).
		if (bits.OnesCount(uint(mask)) & 1) == 1 {
			continue
		}

		// Find the first available (free) vertex in the current subset mask.
		// This breaks symmetry and fixes traversal order, guaranteeing strict determinism.
		first = matchingUnmatched
		for partner = 0; partner < oddCount; partner++ {
			// Check if the vertex (bit index) is present in the current mask.
			if (mask & (1 << uint(partner))) != 0 {
				first = partner
				break
			}
		}
		// Defensive invariant: if a mask is non-empty and even, a free vertex must be found.
		if first == matchingUnmatched {
			continue
		}

		// Isolate the mask by temporarily removing the chosen 'first' vertex from it.
		maskWithoutFirst = mask ^ (1 << uint(first))

		// Iterate over all potential partners for the 'first' vertex from the remaining pool.
		for partner = first + 1; partner < oddCount; partner++ {
			// If the potential partner is missing from the subset, skip to the next index.
			if (maskWithoutFirst & (1 << uint(partner))) == 0 {
				continue
			}

			// Extract the edge cost. If the edge is missing (+Inf), this pair cannot be part of the subgraph.
			cost = problem.at(first, partner)
			if math.IsInf(cost, 1) {
				continue
			}

			// Compute the sub-mask (state) that existed BEFORE adding the (first, partner) pair.
			nextMask = maskWithoutFirst ^ (1 << uint(partner))
			if math.IsInf(dp[nextMask], 1) {
				continue
			}

			// Optimality check: if the current combination yields a lower total weight, update the state.
			candidate = dp[nextMask] + cost
			if candidate < dp[mask] {
				// Record the new minimum subset cost and preserve the pair for the traceback routine.
				dp[mask] = candidate
				parent[mask] = [2]int{first, partner}
			}
		}
	}

	// Verify the final state (full mask where all vertices are covered).
	// If the cost remains +Inf, the graph does not possess a valid perfect matching.
	fullMask := totalMasks - 1
	if math.IsInf(dp[fullMask], 1) {
		return nil, 0, ErrIncompleteGraph
	}

	// Initialize the resulting match array with default unassigned tokens (-1).
	match := make([]int, oddCount)
	var index int
	for index = range match {
		match[index] = matchingUnmatched
	}

	// Traceback phase: reconstruct pairings by backtracking through stored parent state transitions.
	for mask = fullMask; mask != 0; {
		pair := parent[mask]
		first = pair[0]
		partner = pair[1]

		// Index bounds auditing during traceback to catch DP corruption before memory mutation occurs.
		if first == matchingUnmatched || partner == matchingUnmatched || first == partner ||
			first < 0 || first >= oddCount || partner < 0 || partner >= oddCount {
			return nil, 0, ErrInvalidMatching
		}

		match[first] = partner
		match[partner] = first

		// Clear bit flags of the processed pair to transition back to the source sub-mask.
		mask ^= 1 << uint(first)
		mask ^= 1 << uint(partner)
	}

	// Final structural audit: validate matching properties before returning to Christofides facade.
	if err := verifyPerfectMatching(match); err != nil {
		return nil, 0, err
	}

	return match, dp[fullMask], nil
}

// verifyPerfectMatching validates the local symmetric match array.
// It prevents malformed DP or future matching engines from mutating Christofides
// adjacency with duplicate, self, or one-sided pairs.
//
// Implementation:
//   - Stage 1: Reject odd-length match arrays.
//   - Stage 2: Validate each partner index.
//   - Stage 3: Check symmetry match[match[i]]==i and no self-pair.
//
// Behavior highlights:
//   - No allocation.
//   - Does not inspect matrix weights.
//
// Inputs:
//   - match: local match array.
//
// Returns:
//   - error: nil when match is a perfect matching.
//
// Errors:
//   - ErrInvalidMatching for malformed arrays.
//
// Determinism:
//   - Fixed increasing local-index scan.
//
// Complexity:
//   - Time O(k), Space O(1).
//
// Notes:
//   - This is shared by every exact matching engine before adjacency mutation.
//
// AI-Hints:
//   - Do not append matching edges before this verification succeeds.
func verifyPerfectMatching(match []int) error {
	// Structural check: a symmetric matching array must have an even length.
	if (len(match) & 1) == 1 {
		return ErrInvalidMatching
	}

	var (
		index   int
		partner int
	)
	// Iterate through the array to assert bounds safety, loop absence, and reflection.
	for index, partner = range match {
		if partner < 0 || partner >= len(match) {
			return ErrInvalidMatching
		}
		if partner == index {
			return ErrInvalidMatching
		}
		if match[partner] != index {
			return ErrInvalidMatching
		}
	}

	return nil
}

// appendPerfectMatching appends verified local matching pairs to the multigraph.
// It maps local matching indices back to original TSP vertex indices.
//
// Implementation:
//   - Stage 1: Verify the local match array.
//   - Stage 2: Scan local indices in increasing order.
//   - Stage 3: Append each undirected pair exactly once when i<match[i].
//
// Behavior highlights:
//   - Mutates adj only after complete verification.
//   - Preserves deterministic pair append order.
//   - Allows parallel edges because Christofides uses a multigraph.
//
// Inputs:
//   - problem: local matching problem with odd-to-original mapping.
//   - match: verified local match array.
//   - adj: mutable Christofides multigraph adjacency.
//
// Returns:
//   - error: nil after append.
//
// Errors:
//   - ErrInvalidMatching for malformed mappings or adjacency shape.
//
// Determinism:
//   - Fixed increasing local-index scan; each pair appended once.
//
// Complexity:
//   - Time O(k), Space O(1).
//
// Notes:
//   - Caller owns rollback if append must be atomic under later failure.
//
// AI-Hints:
//   - Do not append both i<j and j<i in separate iterations.
func appendPerfectMatching(problem matchingProblem, match []int, adj [][]int) error {
	if err := verifyPerfectMatching(match); err != nil {
		return err
	}
	if len(problem.odd) != len(match) {
		return ErrInvalidMatching
	}

	// Pre-declare iteration primitives to enforce a zero-allocation hot loop profile.
	var (
		local   int
		partner int
		u       int
		v       int
	)
	for local, partner = range match {
		if local > partner {
			continue
		}

		// De-reference local sub-graph indices back into original global TSP vertex IDs.
		u = problem.odd[local]
		v = problem.odd[partner]

		// Bounds firewall: protect the global multigraph against mismatched or corrupt index mappings.
		if u < 0 || u >= len(adj) || v < 0 || v >= len(adj) {
			return ErrInvalidMatching
		}

		// Inject the resolved undirected matching pair into the global Christofides multigraph.
		adj[u] = append(adj[u], v)
		adj[v] = append(adj[v], u)
	}

	return nil
}

// snapshotAdjLengths records current adjacency lengths for atomic rollback.
// It does not copy edge values because matching engines only append edges.
//
// Implementation:
//   - Stage 1: Allocate one length slice.
//   - Stage 2: Store len(adj[v]) for every vertex in deterministic vertex order.
//
// Behavior highlights:
//   - Does not mutate adj.
//   - Does not inspect edge values.
//   - Suitable for Blossom and greedy matching append transactions.
//
// Inputs:
//   - adj: Christofides multigraph adjacency.
//
// Returns:
//   - []int: per-vertex original lengths.
//
// Errors:
//   - None.
//
// Determinism:
//   - Fixed increasing vertex-index scan.
//
// Complexity:
//   - Time O(n), Space O(n).
//
// Notes:
//   - Pair with rollbackAdj in deferred failure paths.
//
// AI-Hints:
//   - Do not deep-copy adjacency for append-only rollback.
//   - Do not reorder adjacency during snapshotting.
func snapshotAdjLengths(adj [][]int) []int {
	lengths := make([]int, len(adj))

	for vertex := range adj {
		lengths[vertex] = len(adj[vertex])
	}

	return lengths
}

// rollbackAdj removes matching edges appended after snapshotAdjLengths.
// It restores slice lengths only and therefore preserves existing edge order.
//
// Implementation:
//   - Stage 1: Scan all vertices.
//   - Stage 2: Validate snapshot bounds defensively.
//   - Stage 3: Truncate adj[v] back to the recorded length.
//
// Behavior highlights:
//   - Mutates adj only by truncation.
//   - Preserves all pre-existing adjacency entries.
//   - Ignores malformed snapshot entries instead of panicking.
//
// Inputs:
//   - adj: mutable adjacency to restore.
//   - lengths: snapshot returned by snapshotAdjLengths.
//
// Returns:
//   - None.
//
// Errors:
//   - None.
//
// Determinism:
//   - Fixed increasing vertex-index scan.
//
// Complexity:
//   - Time O(n), Space O(1).
//
// Notes:
//   - This helper is intentionally private to the matching stage.
//
// AI-Hints:
//   - Do not sort adjacency during rollback; Eulerian traversal depends on stable order.
func rollbackAdj(adj [][]int, lengths []int) {
	for vertex := range adj {
		if vertex >= len(lengths) {
			continue
		}
		if lengths[vertex] < 0 || lengths[vertex] > len(adj[vertex]) {
			continue
		}

		adj[vertex] = adj[vertex][:lengths[vertex]]
	}
}
