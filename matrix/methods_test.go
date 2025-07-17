// matrix/methods_test.go
// SPDX-License-Identifier: Apache-2.0
// Package matrix_test contains unit tests for universal Matrix operations.
package matrix_test

import (
	"testing"

	matrix2 "github.com/katalvlaran/lvlath/matrix"
	"github.com/stretchr/testify/require"
)

func TestAdd_Succeeds(t *testing.T) {
	// Prepare two 2×3 matrices
	a, err := matrix2.NewDense(2, 3)
	require.NoError(t, err)
	b, err := matrix2.NewDense(2, 3)
	require.NoError(t, err)

	// Initialize a = [[1,2,3],[4,5,6]], b = [[6,5,4],[3,2,1]]
	for i := 0; i < 2; i++ {
		for j := 0; j < 3; j++ {
			require.NoError(t, a.Set(i, j, float64(i*3+j+1)))
			require.NoError(t, b.Set(i, j, float64(6-(i*3+j))))
		}
	}

	// Perform addition
	sum, err := matrix2.Add(a, b)
	require.NoError(t, err)

	// Expect sum = [[7,7,7],[7,7,7]]
	for i := 0; i < 2; i++ {
		for j := 0; j < 3; j++ {
			v, err := sum.At(i, j)
			require.NoError(t, err)
			require.Equal(t, 7.0, v)
		}
	}
}

func TestAdd_DimensionMismatch(t *testing.T) {
	a, _ := matrix2.NewDense(2, 2)
	b, _ := matrix2.NewDense(3, 2)
	_, err := matrix2.Add(a, b)
	require.ErrorIs(t, err, matrix2.ErrDimensionMismatch)
}

func TestSub_Succeeds(t *testing.T) {
	// Prepare two 3×2 matrices
	a, _ := matrix2.NewDense(3, 2)
	b, _ := matrix2.NewDense(3, 2)
	// a = [[5,4],[3,2],[1,0]]; b = [[1,1],[1,1],[1,1]]
	values := [][]float64{
		{5, 4},
		{3, 2},
		{1, 0},
	}
	for i := 0; i < 3; i++ {
		for j := 0; j < 2; j++ {
			_ = a.Set(i, j, values[i][j])
			_ = b.Set(i, j, 1)
		}
	}

	diff, err := matrix2.Sub(a, b)
	require.NoError(t, err)

	// Expect diff = [[4,3],[2,1],[0,-1]]
	expected := [][]float64{
		{4, 3},
		{2, 1},
		{0, -1},
	}
	for i := 0; i < 3; i++ {
		for j := 0; j < 2; j++ {
			v, err := diff.At(i, j)
			require.NoError(t, err)
			require.Equal(t, expected[i][j], v)
		}
	}
}

func TestSub_DimensionMismatch(t *testing.T) {
	a, _ := matrix2.NewDense(2, 3)
	b, _ := matrix2.NewDense(2, 2)
	_, err := matrix2.Sub(a, b)
	require.ErrorIs(t, err, matrix2.ErrDimensionMismatch)
}

func TestMul_Succeeds(t *testing.T) {
	// A is 2×3, B is 3×2: A*B = 2×2
	A, _ := matrix2.NewDense(2, 3)
	B, _ := matrix2.NewDense(3, 2)
	// Initialize A = [[1,2,3],[4,5,6]]; B = [[7,8],[9,10],[11,12]]
	aVals := [][]float64{{1, 2, 3}, {4, 5, 6}}
	bVals := [][]float64{{7, 8}, {9, 10}, {11, 12}}
	for i := 0; i < 2; i++ {
		for j := 0; j < 3; j++ {
			_ = A.Set(i, j, aVals[i][j])
		}
	}
	for i := 0; i < 3; i++ {
		for j := 0; j < 2; j++ {
			_ = B.Set(i, j, bVals[i][j])
		}
	}

	C, err := matrix2.Mul(A, B)
	require.NoError(t, err)

	// Expected C = [[58,64],[139,154]]
	expected := [][]float64{{58, 64}, {139, 154}}
	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			v, err := C.At(i, j)
			require.NoError(t, err)
			require.Equal(t, expected[i][j], v)
		}
	}
}

func TestMul_DimensionMismatch(t *testing.T) {
	A, _ := matrix2.NewDense(2, 3)
	B, _ := matrix2.NewDense(2, 2)
	_, err := matrix2.Mul(A, B)
	require.ErrorIs(t, err, matrix2.ErrDimensionMismatch)
}

func TestTranspose(t *testing.T) {
	// 2×3 matrix
	m, _ := matrix2.NewDense(2, 3)
	_ = m.Set(0, 0, 1)
	_ = m.Set(0, 1, 2)
	_ = m.Set(0, 2, 3)
	_ = m.Set(1, 0, 4)
	_ = m.Set(1, 1, 5)
	_ = m.Set(1, 2, 6)

	tm := matrix2.Transpose(m)
	// tm should be 3×2: [[1,4],[2,5],[3,6]]
	exp := [][]float64{{1, 4}, {2, 5}, {3, 6}}
	require.Equal(t, 3, tm.Rows())
	require.Equal(t, 2, tm.Cols())
	for i := 0; i < tm.Rows(); i++ {
		for j := 0; j < tm.Cols(); j++ {
			v, err := tm.At(i, j)
			require.NoError(t, err)
			require.Equal(t, exp[i][j], v)
		}
	}
}

func TestScale(t *testing.T) {
	// 2×2 matrix
	m, _ := matrix2.NewDense(2, 2)
	_ = m.Set(0, 0, 1.5)
	_ = m.Set(0, 1, -2.5)
	_ = m.Set(1, 0, 3.0)
	_ = m.Set(1, 1, 0.0)

	sm := matrix2.Scale(m, 2.0)
	// expected = [[3.0, -5.0],[6.0, 0.0]]
	expected := [][]float64{{3.0, -5.0}, {6.0, 0.0}}
	for i := 0; i < sm.Rows(); i++ {
		for j := 0; j < sm.Cols(); j++ {
			v, err := sm.At(i, j)
			require.NoError(t, err)
			require.Equal(t, expected[i][j], v)
		}
	}
}
