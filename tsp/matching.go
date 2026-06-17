// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package tsp implements the odd-degree matching stage used by Christofides.
// The production contract is strict: GreedyMatch is deterministic but has no
// formal Christofides ratio, while BlossomMatch must either compute exact MWPM
// for the supported odd-set size or return ErrMatchingUnavailable.
package tsp

import (
	"errors"
	"math"
	"math/bits"

	"github.com/katalvlaran/lvlath/matrix"
)

const (
	// maxExactMatchingOddVertices bounds the exponential exact MWPM bootstrap.
	// The value keeps memory at 2^22 DP states and prevents accidentally treating
	// exponential matching as an unbounded production Blossom replacement.
	maxExactMatchingOddVertices = 22

	// matchingUnmatched marks a local matching vertex that has not been paired.
	matchingUnmatched = -1
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
	odd []int
	n   int
	w   []float64
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

// greedyMatch adds a deterministic greedy perfect matching over odd-degree vertices.
// It is intentionally weaker than MWPM and therefore cannot justify the Christofides
// 1.5 approximation ratio.
//
// Implementation:
//   - Stage 1: Validate even cardinality.
//   - Stage 2: Copy odd vertices into a local shrinkable buffer.
//   - Stage 3: Repeatedly choose one endpoint and scan all remaining finite partners.
//   - Stage 4: Append the selected undirected matching edge to adj.
//
// Behavior highlights:
//   - Deterministic tie-break: lower edge cost, then smaller vertex index.
//   - Propagates non-missing edgeCost errors.
//   - Mutates adj as it succeeds; use greedyMatchAtomic when rollback is required.
//
// Inputs:
//   - odd: odd-degree vertices from the MST.
//   - dist: final complete distance matrix.
//   - adj: mutable Christofides multigraph adjacency.
//
// Returns:
//   - error: nil after all odd vertices are paired.
//
// Errors:
//   - ErrInvalidMatching for odd cardinality.
//   - ErrIncompleteGraph when no finite partner exists.
//   - ErrInvalidTour, ErrNaNInf, ErrNegativeWeight propagated from edgeCost.
//
// Determinism:
//   - Takes the current endpoint from the end of the remaining slice.
//   - Scans candidate partners in increasing remaining-slice order.
//   - Ties are resolved by smaller original vertex index.
//
// Complexity:
//   - Time O(k^2), Space O(k), where k=len(odd).
//
// Notes:
//   - This raw helper is not atomic by itself; Christofides should call greedyMatchAtomic.
//
// AI-Hints:
//   - Do not use greedy matching to claim ChristofidesApproximationRatio.
//   - Do not ignore edgeCost errors; missing edges must not be paired.
func greedyMatch(odd []int, dist matrix.Matrix, adj [][]int) error {
	oddCount := len(odd)
	// oddCount==0 is a valid (degenerate) case - nothing to do.
	if oddCount == 0 {
		return nil
	}
	if (oddCount & 1) == 1 {
		return ErrInvalidMatching
	}

	remaining := append([]int(nil), odd...)

	var (
		fromVertex     int
		toVertex       int
		lastIndex      int
		bestIndex      int
		candidateIndex int
		weight         float64
		bestWeight     float64
		err            error
	)

	// Main matching loop: execute O(k) iterations, matching exactly two vertices per cycle.
	// Pops the last element from the 'remaining' buffer to serve as the baseline endpoint.
	for len(remaining) > 1 {
		lastIndex = len(remaining) - 1
		fromVertex = remaining[lastIndex]
		remaining = remaining[:lastIndex]

		// Structural assertion: verify that the popped vertex fits within the global multigraph scope.
		if fromVertex < 0 || fromVertex >= len(adj) {
			return ErrInvalidMatching
		}

		// Initialize local state to track the cheapest candidate partner available in the remaining pool.
		bestIndex = matchingUnmatched
		bestWeight = math.Inf(1)

		// Linear scan over the remaining pool to discover the absolute closest available partner vertex.
		for candidateIndex = 0; candidateIndex < len(remaining); candidateIndex++ {
			// Bounds check: validate that the candidate vertex is safe for global adjacency mapping.
			toVertex = remaining[candidateIndex]
			if toVertex < 0 || toVertex >= len(adj) {
				return ErrInvalidMatching
			}

			// Retrieve the exact cost of the edge from the underlying TSP distance matrix.
			weight, err = edgeCost(dist, fromVertex, toVertex)
			if err != nil {
				// If the edge is missing (+Inf sentinel), skip it and allow closure/fallback paths to handle.
				if errors.Is(err, ErrIncompleteGraph) {
					continue
				}

				return err
			}

			// Deterministic tie-breaking selection: pick the cheapest weight.
			// If costs match within tolerance, resolve ties by selecting the smaller original vertex index.
			if bestIndex == matchingUnmatched ||
				weight < bestWeight ||
				(math.Abs(weight-bestWeight) <= symTol && toVertex < remaining[bestIndex]) {
				bestIndex = candidateIndex
				bestWeight = weight
			}
		}

		// Graph integrity check: fail if the current vertex is completely isolated from all remaining candidates.
		if bestIndex == matchingUnmatched || math.IsInf(bestWeight, 1) {
			return ErrIncompleteGraph
		}

		// O(1) buffer contraction: replace the selected partner at 'bestIndex' with the element at the
		// end of the slice, then truncate. This preserves efficiency without triggering slice shifts.
		lastIndex = len(remaining) - 1
		toVertex = remaining[bestIndex]
		remaining[bestIndex] = remaining[lastIndex]
		remaining = remaining[:lastIndex]

		// Mutate the multigraph by appending the matched pair as an undirected edge.
		adj[fromVertex] = append(adj[fromVertex], toVertex)
		adj[toVertex] = append(adj[toVertex], fromVertex)
	}

	return nil
}

// snapshotAdjLengths records current adjacency lengths for atomic rollback.
// It does not copy edge values because matching rollback only needs to remove
// edges appended by the current matching attempt.
//
// Implementation:
//   - Stage 1: Allocate one length slice.
//   - Stage 2: Store len(adj[v]) for every vertex.
//
// Behavior highlights:
//   - Does not mutate adj.
//   - O(n) and allocation-light.
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
//   - Do not use deep copies for rollback when only append operations are performed.
func snapshotAdjLengths(adj [][]int) []int {
	lengths := make([]int, len(adj))

	var vertex int
	// Capture the current edge count for each vertex before any matching pairs are appended.
	// This creates a fast, memory-light checkpoint for transaction-like rollbacks.
	for vertex = range adj {
		lengths[vertex] = len(adj[vertex])
	}

	return lengths
}

// rollbackAdj removes matching edges appended after snapshotAdjLengths.
// It restores only slice lengths and therefore preserves existing edge order.
//
// Implementation:
//   - Stage 1: Scan all vertices.
//   - Stage 2: Truncate adj[v] back to the recorded length.
//   - Stage 3: Ignore malformed length snapshots defensively by clamping to current length.
//
// Behavior highlights:
//   - Mutates adj only by truncation.
//   - Preserves all pre-existing adjacency entries.
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
//   - Do not reorder adjacency during rollback; Eulerian traversal depends on stable order.
func rollbackAdj(adj [][]int, lengths []int) {
	var vertex int
	// Iterate through every vertex in the graph to safely truncate the adjacency lists.
	for vertex = range adj {
		if vertex >= len(lengths) {
			continue
		}
		// Defensive firewall: skip processing if the recorded snapshot length is negative
		// or somehow exceeds the current capacity, preventing runtime slice panic.
		if lengths[vertex] < 0 || lengths[vertex] > len(adj[vertex]) {
			continue
		}

		// Perform zero-allocation truncation using regular Go reslicing.
		// This instantly strips away the appended matching edges while keeping underlying memory intact.
		adj[vertex] = adj[vertex][:lengths[vertex]]
	}
}

// greedyMatchAtomic runs greedyMatch and rolls back adjacency on failure.
// It is the Christofides-safe wrapper for deterministic greedy matching.
//
// Implementation:
//   - Stage 1: Snapshot adjacency lengths.
//   - Stage 2: Run greedyMatch.
//   - Stage 3: Roll back all appended edges if greedyMatch returns an error.
//
// Behavior highlights:
//   - Successful execution preserves greedyMatch edge order.
//   - Failed execution leaves adj length-equivalent to its input state.
//   - Does not hide greedyMatch errors.
//
// Inputs:
//   - odd: odd-degree MST vertices.
//   - dist: final complete distance matrix.
//   - adj: mutable Christofides multigraph adjacency.
//
// Returns:
//   - error: nil on success or greedyMatch sentinel failure.
//
// Errors:
//   - Same as greedyMatch.
//
// Determinism:
//   - Same matching order as greedyMatch.
//
// Complexity:
//   - Time O(k^2+n), Space O(k+n).
//
// Notes:
//   - Use this wrapper in Christofides, not raw greedyMatch.
//
// AI-Hints:
//   - Do not call greedyMatch directly from Christofides.
func greedyMatchAtomic(odd []int, dist matrix.Matrix, adj [][]int) (err error) {
	lengths := snapshotAdjLengths(adj)
	defer func() {
		if err != nil {
			rollbackAdj(adj, lengths)
		}
	}()

	return greedyMatch(odd, dist, adj)
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
// It is a bounded bootstrap for small odd sets, not an unbounded Blossom engine.
//
// Implementation:
//   - Stage 1: Reject odd sets above maxExactMatchingOddVertices.
//   - Stage 2: DP over even-cardinality masks using the first unmatched local vertex.
//   - Stage 3: Store the selected pair for deterministic reconstruction.
//   - Stage 4: Reconstruct and verify the symmetric match array.
//
// Behavior highlights:
//   - Exact for k <= maxExactMatchingOddVertices.
//   - Deterministic tie behavior: first local vertex, then increasing partner.
//   - Rejects infeasible perfect matchings with ErrIncompleteGraph.
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
//   - ErrMatchingUnavailable when k exceeds the exact bootstrap cap.
//   - ErrInvalidMatching for malformed local problem.
//   - ErrIncompleteGraph when no perfect matching exists.
//
// Determinism:
//   - Fixed mask order, fixed first-unmatched search, fixed increasing partner scan.
//
// Complexity:
//   - Time O(k^2 * 2^k), Space O(2^k+k), bounded by maxExactMatchingOddVertices.
//
// Notes:
//   - This function proves exactness for small k but must not be marketed as Blossom.
//
// AI-Hints:
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
		return nil, 0, ErrMatchingUnavailable
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
//   - This is shared by exact DP and future Blossom engines.
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

// blossomMatch computes exact MWPM for small odd sets and rejects unsupported large sets.
// The name is retained for compatibility with MatchingAlgo, but this implementation is
// a bounded exact DP bootstrap until a true polynomial Blossom engine is added.
//
// Implementation:
//   - Stage 1: Validate trivial and odd-cardinality cases.
//   - Stage 2: Snapshot adjacency lengths for atomic rollback.
//   - Stage 3: Build a local matching problem over odd vertices.
//   - Stage 4: Run exactSmallPerfectMatching.
//   - Stage 5: Append verified matching pairs or roll back on any failure.
//
// Behavior highlights:
//   - Exact for len(odd) <= maxExactMatchingOddVertices.
//   - Returns ErrMatchingUnavailable for larger odd sets.
//   - Does not silently degrade to greedy matching.
//   - Leaves adj unchanged on failure.
//
// Inputs:
//   - odd: odd-degree MST vertices.
//   - dist: final symmetric complete TSP matrix.
//   - adj: mutable Christofides multigraph adjacency.
//
// Returns:
//   - error: nil on exact matching success.
//
// Errors:
//   - ErrInvalidMatching for malformed odd set or adjacency shape.
//   - ErrMatchingUnavailable when the exact bootstrap cap is exceeded.
//   - ErrIncompleteGraph when no perfect matching exists.
//   - Matrix/TSP sentinels from buildMatchingProblem.
//
// Determinism:
//   - Preserves odd scan order and DP deterministic tie policy.
//
// Complexity:
//   - Time O(n^2+k^2*2^k), Space O(n^2+2^k+k^2), bounded for k<=22.
//
// Notes:
//   - This is not a full Blossom implementation.
//   - Christofides may claim 1.5 for this path because the produced matching is exact MWPM.
//
// AI-Hints:
//   - Do not return ErrMatchingNotImplemented from this production path.
//   - Do not call greedyMatch from here; fallback belongs to the Christofides policy layer.
func blossomMatch(odd []int, dist matrix.Matrix, adj [][]int) (err error) {
	if len(odd) == 0 {
		return nil
	}
	if (len(odd) & 1) == 1 {
		return ErrInvalidMatching
	}

	// Capture a transactional checkpoint of slice lengths before attempting graph mutation.
	// The deferred closure guarantees a clean rollback of all appended edges if subsequent phases fail.
	lengths := snapshotAdjLengths(adj)
	defer func() {
		if err != nil {
			// Trigger state recovery: discard intermediate mutations and restore original graph layout.
			rollbackAdj(adj, lengths)
		}
	}()

	// Construct the independent, localized matching problem sub-graph induced by the odd vertices.
	problem, err := buildMatchingProblem(odd, dist)
	if err != nil {
		return err
	}

	// Invoke the exact subset DP engine to resolve the absolute minimum cost perfect matching.
	match, _, err := exactSmallPerfectMatching(problem)
	if err != nil {
		return err
	}

	return appendPerfectMatching(problem, match, adj)
}
