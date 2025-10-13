// SPDX-License-Identifier: MIT

// Package matrix: domain types used by adapters and dense operations.
// This file intentionally contains ONLY domain-facing types (IDs, weights,
// helper keys) and, for now, preserves the public Matrix interface to avoid
// breaking existing code. Errors and options live in dedicated
// files (errors.go, options.go) per the global conventions.
package matrix

// VertexID uniquely identifies a graph vertex (core uses string IDs).
// Determinism relies on lexicographic ordering of these IDs across adapters.
type VertexID string // string-based ID (stable lex order)

// Weight represents an edge weight for adapters/numeric ingestion.
// All weights must be finite under the numeric policy; NaN/Inf is rejected.
type Weight float64 // enforced via ValidateNaNInf policy

// pairKey is an ordered pair (u,v) used to de-duplicate parallel edges under
// "first-edge-wins" policy in directed mode. For undirected mode we normalize
// into {min,max} and still store in pairKey (u=min, v=max). Using ints keeps
// the key compact and hash-friendly.
// Complexity: O(1) to build; used in O(E) scans during ingestion.
type pairKey struct {
	u int // source row index
	v int // destination column index
}

// Matrix represents a two-dimensional mutable array of float64 values.
// We KEEP the interface to preserve current callers, while.
//
// Rationale:
//   - Minimal churn (options/errors refactor only).
//   - Tests and existing impl_* likely depend on the interface.
//
// Complexity notes: all methods are expected O(1) except Clone (O(r*c)).
type Matrix interface {
	// Rows returns the number of rows in the matrix.
	// Complexity: O(1).
	Rows() int

	// Cols returns the number of columns in the matrix.
	// Complexity: O(1).
	Cols() int

	// At retrieves the element at position (i, j).
	// Returns ErrIndexOutOfBounds if i<0, i>=Rows(), j<0 or j>=Cols().
	// Complexity: O(1).
	At(i, j int) (float64, error)

	// Set assigns the value v at position (i, j).
	// Returns ErrIndexOutOfBounds if indices are invalid.
	// Complexity: O(1).
	Set(i, j int, v float64) error

	// Clone returns a deep copy of the matrix.
	// The returned Matrix is independent of the original.
	// Complexity: O(rows*cols).
	Clone() Matrix
}
