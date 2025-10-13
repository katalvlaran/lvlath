// SPDX-License-Identifier: MIT
// Package matrix: sentinel error set (unified, consistent).
// This file defines ONLY package-level sentinel errors used across the matrix
// package. All algorithms MUST return these sentinels and tests MUST check them
// via errors.Is. No algorithm should panic on user-triggered error conditions.
// Panics are reserved for programmer errors in private helpers (if any).

package matrix

import "errors"

// NOTE ON NAMING & PREFIXING
// --------------------------
// Every message is prefixed with "matrix: ..." for consistency and to allow
// easy grepping across logs. DO NOT %w wrap these sentinels when returning
// directly; if context is essential, wrap with fmt.Errorf("ctx: %w", ErrX)
// at the outer boundary — callers will still use errors.Is to match.
//
// ERROR PRIORITY (documented, enforced in tests):
// shape/index/NaN -> graph nil -> dimension mismatch -> structural violations
// -> export limitations (ErrMatrixNotImplemented).

var (
	// ErrBadShape is returned when requested shape is invalid (e.g., r<=0 or c<=0).
	// Algorithms must validate dense creation before allocation.
	ErrBadShape = errors.New("matrix: invalid shape")

	// ErrOutOfRange indicates that an index (row or column) is outside valid bounds.
	// Public indexers (At/Set) MUST return this, not panic.
	ErrOutOfRange = errors.New("matrix: index out of range")

	// ErrDimensionMismatch indicates incompatible dimensions between operands,
	// e.g., Add/Sub different shapes, or Mul where a.Cols != b.Rows.
	ErrDimensionMismatch = errors.New("matrix: dimension mismatch")

	// ErrNonSquare signals that a square matrix was required but the input wasn't.
	ErrNonSquare = errors.New("matrix: matrix is not square")

	// ErrAsymmetry signals that a matrix expected to be symmetric violated symmetry
	// within the configured numeric policy (epsilon).
	ErrAsymmetry = errors.New("matrix: matrix is not symmetric within eps")

	// ErrNonZeroDiagonal signals that a diagonal is required to be ~0 (within eps)
	// but a non-zero entry was observed (common in Laplacian-like checks).
	ErrNonZeroDiagonal = errors.New("matrix: diagonal not zero within eps")

	// ErrNaNInf signals a NaN or ±Inf value was encountered where finite values
	// are required by the numeric policy (ingestion, Set, etc.).
	ErrNaNInf = errors.New("matrix: NaN or Inf encountered")

	// ErrGraphNil indicates that a nil *core.Graph was passed into an adapter.
	ErrGraphNil = errors.New("matrix: graph is nil")

	// ErrUnknownVertex indicates that a referenced VertexID is not present
	// in the current vertex index (adjacency/incidence).
	ErrUnknownVertex = errors.New("matrix: unknown vertex id")

	// ErrNilMatrix indicates that a nil Matrix (receiver or argument) was used.
	ErrNilMatrix = errors.New("matrix: nil receiver")

	// ErrMatrixNotImplemented marks an intentionally unsupported operation
	// on the matrix surface (e.g., ToGraph on MetricClosure adjacency).
	ErrMatrixNotImplemented = errors.New("matrix: operation not implemented")

	// ErrMatrixEigenFailed indicates that an eigen/Jacobi routine failed to converge
	// under the given tolerance/iterations.
	ErrMatrixEigenFailed = errors.New("matrix: eigen decomposition failed")

	// ErrSingular is returned when a zero pivot is encountered during inversion/LU
	// in a non-pivoting scheme (intentional for determinism and simplicity).
	ErrSingular = errors.New("matrix: singular matrix")

	// ErrNonBinaryIncidence — non-±1 entry detected for unweighted incidence.
	ErrNonBinaryIncidence = errors.New("matrix: non-binary incidence")

	// ErrInvalidWeight — edge weight is NaN or ±Inf at ingestion stage.
	ErrInvalidWeight = errors.New("matrix: invalid edge weight")

	// ErrInvalidDimensions indicates that requested matrix dimensions are non-positive.
	ErrInvalidDimensions = errors.New("matrix: dimensions must be > 0")
)

// BACKWARD-COMPATIBILITY ALIASES (kept to avoid breaking current callers).
// These are maintained to let existing code compile while we migrate internals
// to the unified names above. They are semantically identical sentinels.

// ErrIndexOutOfBounds historically named the same condition as ErrOutOfRange.
// Keep it as an alias so errors.Is(err, ErrIndexOutOfBounds) remains true.
var ErrIndexOutOfBounds = ErrOutOfRange // Deprecated: use ErrOutOfRange.

// ErrNotSymmetric historically named symmetry violation.
// It aliases ErrAsymmetry to preserve errors.Is behavior during migration.
var ErrNotSymmetric = ErrAsymmetry // Deprecated: use ErrAsymmetry.

// ErrEigenFailed historically named the eigen failure sentinel.
// Alias to unified ErrMatrixEigenFailed for non-breaking migration.
var ErrEigenFailed = ErrMatrixEigenFailed // Deprecated: use ErrMatrixEigenFailed.
