// SPDX-License-Identifier: MIT
// Package matrix_test contains unit tests for matrix validators.
//
// Purpose:
//   - Validate structural preconditions (nil/shape/square).
//   - Validate numeric preconditions (finite-only) where explicitly required.
//
// Design:
//   - Tests avoid brittle coupling to AdjacencyMatrix field names.
//   - Optional index-metadata checks are performed only if an exported index-map field exists.
//
// Determinism & Performance:
//   - All tests use fixed inputs and deterministic loops.
//   - No randomized behavior in validator tests.
//
// AI-Hints:
//   - Keep validator tests focused on contracts (sentinel errors and invariants).
//   - Do not encode assumptions about unexported fields from another package (matrix_test).

package matrix_test

import (
	"math"
	"testing"

	"github.com/katalvlaran/lvlath/matrix"
)

// --- Basic structural validators ---------------------------------------------

func TestValidateNotNil(t *testing.T) {
	t.Parallel()

	AssertErrorIs(t, matrix.ValidateNotNil(nil), matrix.ErrNilMatrix)

	m := IdentityDense(t, 2)
	if err := matrix.ValidateNotNil(m); err != nil {
		t.Fatalf("ValidateNotNil: %v", err)
	}
}

func TestValidateSameShape(t *testing.T) {
	t.Parallel()

	a, _ := matrix.NewDense(3, 2)
	b, _ := matrix.NewDense(3, 2)
	c, _ := matrix.NewDense(2, 2)
	d, _ := matrix.NewDense(3, 1)

	if err := matrix.ValidateSameShape(a, b); err != nil {
		t.Fatalf("ValidateSameShape (equal): %v", err)
	}
	AssertErrorIs(t, matrix.ValidateSameShape(a, c), matrix.ErrDimensionMismatch)
	AssertErrorIs(t, matrix.ValidateSameShape(a, d), matrix.ErrDimensionMismatch)
}

func TestValidateSquare(t *testing.T) {
	t.Parallel()

	sq, _ := matrix.NewIdentity(4)
	if err := matrix.ValidateSquare(sq); err != nil {
		t.Fatalf("ValidateSquare (square): %v", err)
	}

	nsq, _ := matrix.NewDense(3, 4)
	AssertErrorIs(t, matrix.ValidateSquare(nsq), matrix.ErrDimensionMismatch)
	AssertErrorIs(t, matrix.ValidateSquare(nil), matrix.ErrNilMatrix)
}

func TestValidateVecLen(t *testing.T) {
	t.Parallel()

	x := make([]float64, 4)
	if err := matrix.ValidateVecLen(x, 4); err != nil {
		t.Fatalf("ValidateVecLen (ok): %v", err)
	}
	AssertErrorIs(t, matrix.ValidateVecLen(x, 5), matrix.ErrDimensionMismatch)
	AssertErrorIs(t, matrix.ValidateVecLen(nil, 3), matrix.ErrNilMatrix)
}

func TestValidateBinarySameShape(t *testing.T) {
	t.Parallel()

	a, _ := matrix.NewDense(2, 3)
	b, _ := matrix.NewDense(2, 3)
	c, _ := matrix.NewDense(2, 4)

	if err := matrix.ValidateBinarySameShape(a, b); err != nil {
		t.Fatalf("ValidateBinarySameShape (ok): %v", err)
	}
	AssertErrorIs(t, matrix.ValidateBinarySameShape(a, c), matrix.ErrDimensionMismatch)
	AssertErrorIs(t, matrix.ValidateBinarySameShape(a, nil), matrix.ErrNilMatrix)
}

func TestValidateSquareNonNil(t *testing.T) {
	t.Parallel()

	m := IdentityDense(t, 3)
	if err := matrix.ValidateSquareNonNil(m); err != nil {
		t.Fatalf("ValidateSquareNonNil (ok): %v", err)
	}
	AssertErrorIs(t, matrix.ValidateSquareNonNil(nil), matrix.ErrNilMatrix)

	nsq, _ := matrix.NewDense(2, 3)
	AssertErrorIs(t, matrix.ValidateSquareNonNil(nsq), matrix.ErrDimensionMismatch)
}

func TestValidateMulCompatible(t *testing.T) {
	t.Parallel()

	a, _ := matrix.NewDense(2, 4)
	b, _ := matrix.NewDense(4, 3)
	c, _ := matrix.NewDense(3, 3)

	if err := matrix.ValidateMulCompatible(a, b); err != nil {
		t.Fatalf("ValidateMulCompatible (ok): %v", err)
	}
	AssertErrorIs(t, matrix.ValidateMulCompatible(a, c), matrix.ErrDimensionMismatch)
	AssertErrorIs(t, matrix.ValidateMulCompatible(nil, b), matrix.ErrNilMatrix)
	AssertErrorIs(t, matrix.ValidateMulCompatible(a, nil), matrix.ErrNilMatrix)
}

func TestValidateSymmetric(t *testing.T) {
	t.Parallel()

	m := IdentityDense(t, 3)
	if err := matrix.ValidateSymmetric(m, 1e-9); err != nil {
		t.Fatalf("ValidateSymmetric (identity): %v", err)
	}

	asym := MustDense(t, 2, 2)
	MustSet(t, asym, 0, 1, 1.0)
	MustSet(t, asym, 1, 0, 2.0)
	AssertErrorIs(t, matrix.ValidateSymmetric(asym, 0.5), matrix.ErrAsymmetry)

	// NaN/Inf tolerance must be rejected.
	AssertErrorIs(t, matrix.ValidateSymmetric(m, math.NaN()), matrix.ErrNaNInf) // NaN
}

// --- Value-level validator ----------------------------------------------------

func TestValidateAllFinite(t *testing.T) {
	t.Parallel()

	AssertErrorIs(t, matrix.ValidateAllFinite(nil), matrix.ErrNilMatrix)

	ok := NewFilledDense(t, 2, 2, []float64{1, 2, 3, 4})
	if err := matrix.ValidateAllFinite(ok); err != nil {
		t.Fatalf("ValidateAllFinite (finite): %v", err)
	}

	// Build a "dirty" matrix via raw ingest (Fill), not via Set().
	dirty, _ := matrix.NewPreparedDense(2, 2, matrix.WithNoValidateNaNInf())
	bad := []float64{1, math.NaN(), 3, math.Inf(1)}
	// Raw-ingest to avoid Set() numeric-policy on NaN/Inf.
	MustFillRowMajor(t, dirty, bad)

	AssertErrorIs(t, matrix.ValidateAllFinite(dirty), matrix.ErrNaNInf)
}

// --- Graph adjacency wrapper validator ---------------------------------------

func TestValidateGraphAdjacency_Structural(t *testing.T) {
	t.Parallel()

	// Nil wrapper.
	AssertErrorIs(t, matrix.ValidateGraphAdjacency((*matrix.AdjacencyMatrix)(nil)), matrix.ErrNilMatrix)

	// Nil Mat.
	gNilMat := &matrix.AdjacencyMatrix{Mat: nil}
	AssertErrorIs(t, matrix.ValidateGraphAdjacency(gNilMat), matrix.ErrNilMatrix)

	// Non-square Mat.
	nsq, _ := matrix.NewDense(2, 3)
	gNonSquare := &matrix.AdjacencyMatrix{Mat: nsq}
	AssertErrorIs(t, matrix.ValidateGraphAdjacency(gNonSquare), matrix.ErrDimensionMismatch)

	// Square ok.
	g := &matrix.AdjacencyMatrix{Mat: IdentityDense(t, 2)}
	if err := matrix.ValidateGraphAdjacency(g); err != nil {
		t.Fatalf("ValidateGraphAdjacency (square): %v", err)
	}
}
