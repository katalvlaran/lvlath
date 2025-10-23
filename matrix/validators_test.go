// SPDX-License-Identifier: Apache-2.0
// Package matrix_test contains unit tests for the matrix validators.
package matrix_test

import (
	"math"
	"testing"

	"github.com/katalvlaran/lvlath/matrix"
)

func TestValidateNotNil(t *testing.T) {
	t.Parallel()
	err := matrix.ValidateNotNil(nil)
	AssertErrorIs(t, err, matrix.ErrNilMatrix)

	m := IdentityDense(t, 2)
	if err = matrix.ValidateNotNil(m); err != nil {
		t.Fatalf("ValidateNotNil(%d): %v", m, err)
	}
}

func TestValidateSameShape(t *testing.T) {
	t.Parallel()
	a, _ := matrix.NewDense(3, 2)
	b, _ := matrix.NewDense(3, 2)
	c, _ := matrix.NewDense(2, 2)
	d, _ := matrix.NewDense(3, 1)

	if err := matrix.ValidateSameShape(a, b); err != nil {
		t.Fatalf("ValidateSameShape(%v,%v): %v", a, d, err)
	}
	AssertErrorIs(t, matrix.ValidateSameShape(a, c), matrix.ErrDimensionMismatch)
	AssertErrorIs(t, matrix.ValidateSameShape(a, d), matrix.ErrDimensionMismatch)
}

func TestValidateSquare(t *testing.T) {
	t.Parallel()
	sq, _ := matrix.NewIdentity(4)
	if err := matrix.ValidateSquare(sq); err != nil {
		t.Fatalf("ValidateSquare(%v): %v", sq, err)
	}

	nsq, _ := matrix.NewDense(3, 4)
	AssertErrorIs(t, matrix.ValidateSquare(nsq), matrix.ErrDimensionMismatch)
}

func TestValidateVecLen(t *testing.T) {
	t.Parallel()
	x := make([]float64, 4)
	if err := matrix.ValidateVecLen(x, 4); err != nil {
		t.Fatalf("ValidateVecLen(%v, %d): %v", x, 4, err)
	}
	AssertErrorIs(t, matrix.ValidateVecLen(x, 5), matrix.ErrDimensionMismatch)
	AssertErrorIs(t, matrix.ValidateVecLen(nil, 3), matrix.ErrNilMatrix)
}

func TestValidateBinarySameShape(t *testing.T) {
	t.Parallel()
	a, _ := matrix.NewDense(2, 3)
	b, _ := matrix.NewDense(2, 3)
	c, _ := matrix.NewDense(2, 4)
	d := matrix.Matrix(nil)

	if err := matrix.ValidateBinarySameShape(a, b); err != nil {
		t.Fatalf("ValidateBinarySameShape(%v, %v): %v", a, b, err)
	}
	AssertErrorIs(t, matrix.ValidateBinarySameShape(a, c), matrix.ErrDimensionMismatch)
	AssertErrorIs(t, matrix.ValidateBinarySameShape(a, d), matrix.ErrNilMatrix)
}

func TestValidateSquareNonNil(t *testing.T) {
	t.Parallel()
	m := IdentityDense(t, 3)
	if err := matrix.ValidateSquareNonNil(m); err != nil {
		t.Fatalf("ValidateSquareNonNil(%d): %v", m, err)
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
		t.Fatalf("ValidateMulCompatible(%v, %v): %v", a, b, err)
	}
	AssertErrorIs(t, matrix.ValidateMulCompatible(a, c), matrix.ErrDimensionMismatch)
	AssertErrorIs(t, matrix.ValidateMulCompatible(nil, b), matrix.ErrNilMatrix)
}

func TestValidateSymmetric(t *testing.T) {
	t.Parallel()
	m := IdentityDense(t, 3)
	if err := matrix.ValidateSymmetric(m, 1e-9); err != nil {
		t.Fatalf("ValidateSymmetric(%v, %v): %v", m, 1e-9, err)
	}

	asym, _ := matrix.NewDense(2, 2)
	_ = asym.Set(0, 1, 1.0)
	_ = asym.Set(1, 0, 2.0)
	AssertErrorIs(t, matrix.ValidateSymmetric(asym, 0.5), matrix.ErrAsymmetry)
	AssertErrorIs(t, matrix.ValidateSymmetric(asym, math.NaN()), matrix.ErrNaNInf)
}

func TestValidateGraphAdjacency(t *testing.T) {
	t.Parallel()
	g := &matrix.AdjacencyMatrix{Mat: IdentityDense(t, 2)}
	if err := matrix.ValidateGraphAdjacency(g); err != nil {
		t.Fatalf("ValidateGraphAdjacency(%v): %v", g, err)
	}

	gNil := (*matrix.AdjacencyMatrix)(nil)
	AssertErrorIs(t, matrix.ValidateGraphAdjacency(gNil), matrix.ErrNilMatrix)

	//vertIndx := map[string]int{"a":0}
	gBad := &matrix.AdjacencyMatrix{Mat: IdentityDense(t, 2), VertexIndex: map[string]int{"a": 0}}
	AssertErrorIs(t, matrix.ValidateGraphAdjacency(gBad), matrix.ErrDimensionMismatch)
}
