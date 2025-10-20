// SPDX-License-Identifier: Apache-2.0
// Package matrix_test contains unit tests for the matrix validators.
package matrix_test

import (
	"math"
	"testing"

	"github.com/katalvlaran/lvlath/matrix"
	"github.com/stretchr/testify/require"
)

// identityDense returns a square *Dense matrix of size n×n.
func identityDense(t *testing.T, n int) matrix.Matrix {
	t.Helper()
	m, err := matrix.NewIdentity(n)
	require.NoError(t, err)
	for i := 0; i < n; i++ {
		require.NoError(t, m.Set(i, i, 1))
	}
	return m
}

func TestValidateNotNil(t *testing.T) {
	t.Parallel()
	err := matrix.ValidateNotNil(nil)
	require.ErrorIs(t, err, matrix.ErrNilMatrix)

	m := identityDense(t, 2)
	require.NoError(t, matrix.ValidateNotNil(m))
}

func TestValidateSameShape(t *testing.T) {
	t.Parallel()
	a, _ := matrix.NewDense(3, 2)
	b, _ := matrix.NewDense(3, 2)
	c, _ := matrix.NewDense(2, 2)
	d, _ := matrix.NewDense(3, 1)

	require.NoError(t, matrix.ValidateSameShape(a, b))
	require.ErrorIs(t, matrix.ValidateSameShape(a, c), matrix.ErrDimensionMismatch)
	require.ErrorIs(t, matrix.ValidateSameShape(a, d), matrix.ErrDimensionMismatch)
}

func TestValidateSquare(t *testing.T) {
	t.Parallel()
	sq, _ := matrix.NewIdentity(4)
	require.NoError(t, matrix.ValidateSquare(sq))

	nsq, _ := matrix.NewDense(3, 4)
	require.ErrorIs(t, matrix.ValidateSquare(nsq), matrix.ErrDimensionMismatch)
}

func TestValidateVecLen(t *testing.T) {
	t.Parallel()
	x := make([]float64, 4)
	require.NoError(t, matrix.ValidateVecLen(x, 4))
	require.ErrorIs(t, matrix.ValidateVecLen(x, 5), matrix.ErrDimensionMismatch)
	require.ErrorIs(t, matrix.ValidateVecLen(nil, 3), matrix.ErrNilMatrix)
}

func TestValidateBinarySameShape(t *testing.T) {
	t.Parallel()
	a, _ := matrix.NewDense(2, 3)
	b, _ := matrix.NewDense(2, 3)
	c, _ := matrix.NewDense(2, 4)
	d := matrix.Matrix(nil)

	require.NoError(t, matrix.ValidateBinarySameShape(a, b))
	require.ErrorIs(t, matrix.ValidateBinarySameShape(a, c), matrix.ErrDimensionMismatch)
	require.ErrorIs(t, matrix.ValidateBinarySameShape(a, d), matrix.ErrNilMatrix)
}

func TestValidateSquareNonNil(t *testing.T) {
	t.Parallel()
	m := identityDense(t, 3)
	require.NoError(t, matrix.ValidateSquareNonNil(m))
	require.ErrorIs(t, matrix.ValidateSquareNonNil(nil), matrix.ErrNilMatrix)

	nsq, _ := matrix.NewDense(2, 3)
	require.ErrorIs(t, matrix.ValidateSquareNonNil(nsq), matrix.ErrDimensionMismatch)
}

func TestValidateMulCompatible(t *testing.T) {
	t.Parallel()
	a, _ := matrix.NewDense(2, 4)
	b, _ := matrix.NewDense(4, 3)
	c, _ := matrix.NewDense(3, 3)

	require.NoError(t, matrix.ValidateMulCompatible(a, b))
	require.ErrorIs(t, matrix.ValidateMulCompatible(a, c), matrix.ErrDimensionMismatch)
	require.ErrorIs(t, matrix.ValidateMulCompatible(nil, b), matrix.ErrNilMatrix)
}

func TestValidateSymmetric(t *testing.T) {
	t.Parallel()
	m := identityDense(t, 3)
	require.NoError(t, matrix.ValidateSymmetric(m, 1e-9))

	asym, _ := matrix.NewDense(2, 2)
	_ = asym.Set(0, 1, 1.0)
	_ = asym.Set(1, 0, 2.0)
	require.ErrorIs(t, matrix.ValidateSymmetric(asym, 0.5), matrix.ErrAsymmetry)
	require.ErrorIs(t, matrix.ValidateSymmetric(asym, math.NaN()), matrix.ErrNaNInf)
}

func TestValidateGraphAdjacency(t *testing.T) {
	t.Parallel()
	g := &matrix.AdjacencyMatrix{Mat: identityDense(t, 2)}
	require.NoError(t, matrix.ValidateGraphAdjacency(g))

	gNil := (*matrix.AdjacencyMatrix)(nil)
	require.ErrorIs(t, matrix.ValidateGraphAdjacency(gNil), matrix.ErrNilMatrix)

	//vertIndx := map[string]int{"a":0}
	gBad := &matrix.AdjacencyMatrix{Mat: identityDense(t, 2), VertexIndex: map[string]int{"a": 0}}
	require.ErrorIs(t, matrix.ValidateGraphAdjacency(gBad), matrix.ErrDimensionMismatch)
}
