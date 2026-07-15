// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package dfs defines public traversal types, and result structures
// for depth-first search workflows over core.Graph.
//
// The file contains configuration and result contracts only.
package dfs

// VertexState describes the traversal state of a vertex in DFS-based algorithms.
//
// Implementation:
//   - Stage 1: A vertex starts in White.
//   - Stage 2: It becomes Gray when entered and placed in the active DFS path.
//   - Stage 3: It becomes Black after all outgoing traversal work is complete.
//
// Behavior highlights:
//   - White means undiscovered.
//   - Gray means currently active in the DFS recursion path.
//   - Black means fully explored.
//
// Inputs:
//   - N/A.
//
// Returns:
//   - N/A.
//
// Errors:
//   - N/A.
//
// Determinism:
//   - State transitions are deterministic under deterministic traversal order.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - The type is intentionally small and strictly typed to prevent accidental mixing
//     with unrelated integer values.
//
// AI-Hints:
//   - Use VertexState instead of raw integers in DFS, topological sort, and cycle detection.
//   - Gray is the active-path state used for back-edge detection.
//   - Black must mean fully explored, not merely visited once.
type VertexState uint8

const (
	// White marks a vertex that has not been entered yet.
	White VertexState = iota

	// Gray marks a vertex that is currently active in the DFS recursion path.
	Gray

	// Black marks a vertex whose traversal work has been fully completed.
	Black
)

// Result captures the observable outcome of DFS traversal.
//
// Implementation:
//   - Stage 1: Visited is recorded at vertex entry.
//   - Stage 2: Parent and Depth are assigned as DFS-tree state is formed.
//   - Stage 3: Order is appended on vertex finish.
//
// Behavior highlights:
//   - Order is post-order, not pre-order.
//   - Parent and Depth describe the DFS forest produced by traversal policy.
//
// Inputs:
//   - N/A.
//
// Returns:
//   - N/A.
//
// Errors:
//   - N/A.
//
// Determinism:
//   - Deterministic under deterministic graph root order, neighbor order, hooks, and filters.
//
// Complexity:
//   - Storage is O(V) for visited vertices, plus O(V) post-order output.
//
// Notes:
//   - The result is owned by the caller after DFS returns.
//
// AI-Hints:
//   - Order is finish order.
//   - Depth is DFS-tree depth, not general shortest-path distance.
//   - Parent is assigned only when a vertex is actually entered.
type Result struct {
	// Order records vertices in DFS finish order (post-order).
	Order []string

	// Depth maps each visited vertex ID to its DFS-tree depth.
	// In single-source mode, depth is measured from startID.
	// In full-traversal mode, depth is measured from the root of the
	// corresponding DFS tree in the traversal forest.
	Depth map[string]int

	// Parent maps each visited non-root vertex ID to the vertex from which it was first entered.
	// Root vertices of DFS trees do not appear in this map.
	Parent map[string]string

	// Visited reports which vertices were entered by the traversal.
	Visited map[string]bool

	// SkippedNeighbors counts candidate neighbors rejected by FilterNeighbor.
	SkippedNeighbors int
}

// CycleDetectionResult captures the observable outcome of DFS-based cycle detection.
//
// Implementation:
//   - Stage 1: Record graph cyclicity as a boolean summary.
//   - Stage 2: Store the canonical witness cycles discovered during traversal.
//
// Behavior highlights:
//   - Cycles contains a deterministic witness set, not an exhaustive enumeration of all simple cycles.
//   - The result is owned by the caller after DetectCycles returns.
//
// Inputs:
//   - N/A.
//
// Returns:
//   - N/A.
//
// Errors:
//   - N/A.
//
// Determinism:
//   - Deterministic under deterministic graph root order, neighbor order, and canonicalization policy.
//
// Complexity:
//   - Storage is O(W·L), where W = number of stored witness cycles and L = average witness length.
//
// Notes:
//   - Cycles are returned as closed sequences of the form [v0, v1, ..., v0].
//
// AI-Hints:
//   - HasCycle is the summary answer.
//   - Cycles is a witness set, not a proof of exhaustive cycle enumeration.
//   - Do not infer shortest paths or graph reachability semantics from cycle witness order.
type CycleDetectionResult struct {
	// HasCycle reports whether at least one cycle witness was found.
	HasCycle bool

	// Cycles stores canonical closed witness cycles in deterministic order.
	Cycles [][]string
}

// IsNil reports whether the receiver should be treated as nil when stored inside interfaces.
//
// Implementation:
//   - Stage 1: Compare the receiver pointer to nil.
//   - Stage 2: Return the result without dereferencing.
//
// Behavior highlights:
//   - Safe for typed-nil stored inside interfaces.
//   - Reflect-free nil detection used by tests through core.Nilable.
//
// Inputs:
//   - N/A.
//
// Returns:
//   - bool: true iff receiver == nil.
//
// Errors:
//   - N/A.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Keep this method trivial and side-effect free.
//
// AI-Hints:
//   - Use Nilable-aware test helpers to detect typed nil correctly behind interfaces.
func (r *Result) IsNil() bool {
	return r == nil
}

// IsNil reports whether the receiver should be treated as nil when stored inside interfaces.
//
// Implementation:
//   - Stage 1: Compare the receiver pointer to nil.
//   - Stage 2: Return the result without dereferencing.
//
// Behavior highlights:
//   - Safe for typed-nil stored inside interfaces.
//   - Reflect-free nil detection used by tests through core.Nilable.
//
// Inputs:
//   - N/A.
//
// Returns:
//   - bool: true iff receiver == nil.
//
// Errors:
//   - N/A.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Keep this method trivial and side-effect free.
//
// AI-Hints:
//   - Use Nilable-aware test helpers to detect typed nil correctly behind interfaces.
func (r *CycleDetectionResult) IsNil() bool {
	return r == nil
}
