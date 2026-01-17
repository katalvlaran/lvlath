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
// Contract:
//   - When numeric validation is enabled, values must be finite (no NaN/±Inf).
//   - No sign or range restriction is imposed here beyond finiteness.
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
//
// Contract:
//   - Matrix may be zero-size (0×0, 0×N, N×0).
//   - At/Set MUST return ErrOutOfRange on invalid indices.
//   - Set may additionally return ErrNaNInf when numeric policy rejects a value.
//   - Clone returns a deep copy independent of the original.
//
// Complexity notes:
//   - Rows/Cols: O(1)
//   - At/Set: O(1)
//   - Clone: expected O(r*c) for dense implementations.
type Matrix interface {
	// Rows returns the number of rows in the matrix.
	// Complexity: O(1).
	Rows() int

	// Cols returns the number of columns in the matrix.
	// Complexity: O(1).
	Cols() int

	// At retrieves the element at position (i, j).
	// Returns ErrOutOfRange if i<0, i>=Rows(), j<0 or j>=Cols().
	// Concrete implementations may return additional sentinel errors per
	// numeric policy.
	//Complexity: O(1).
	At(i, j int) (float64, error)

	// Set assigns the value v at position (i, j).
	// Returns ErrOutOfRange if indices are invalid. Implementations may also
	// return ErrNaNInf when v is not finite, depending on numeric policy.
	// Complexity: O(1).
	Set(i, j int, v float64) error

	// Clone returns a deep copy of the matrix.
	// The returned Matrix is independent of the original.
	// Complexity: O(rows*cols).
	Clone() Matrix
}

// Nilable MAIN DESCRIPTION (2+ lines, no marketing).
// Provides an explicit, reflect-free mechanism to treat typed-nil receivers
// stored inside a Matrix interface as nil during validation.
//
// Implementation:
//   - Stage 1: Callers pass a Matrix interface to validators (e.g., ValidateNotNil).
//   - Stage 2: If the dynamic value implements Nilable, validators call IsNil().
//   - Stage 3: If IsNil reports true, validators return ErrNilMatrix.
//
// Behavior highlights:
//   - Avoids reflect in hot, foundational validators.
//   - Keeps nil-detection O(1) and deterministic.
//   - Optional: implementations that do not provide IsNil() are treated normally.
//
// Inputs:
//   - (receiver): the concrete Matrix implementation (often a pointer type).
//
// Returns:
//   - bool: true if the receiver should be treated as nil.
//
// Errors:
//   - None. IsNil MUST NOT panic and MUST NOT allocate.
//
// Determinism:
//   - Deterministic and side-effect free (required).
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Go interfaces may hold typed nil pointers:
//     var d *Dense = nil
//     var m Matrix = d
//     m != nil  // true
//     Nilable allows validators to detect this case without reflect.
//
// AI-Hints:
//   - Implement IsNil() on pointer-backed Matrix types (e.g., *Dense) as:
//     func (m *T) IsNil() bool { return m == nil }
//   - Keep IsNil pure and cheap; do not inspect internal buffers or perform deep checks.
type Nilable interface {
	// IsNil reports whether the receiver should be treated as nil.
	// It MUST be side-effect free and deterministic.
	IsNil() bool
}
