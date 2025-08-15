// Package matrix_test provides unit tests for basic matrix operations
// covering nil guards, dimension mismatches, and happy paths.
package matrix_test

import (
	"testing"

	"github.com/katalvlaran/lvlath/matrix"
	"github.com/stretchr/testify/require"
)

func TestMethods_NilGuards(t *testing.T) {
	t.Parallel()

	// Test Add(nil, valid) returns ErrNilMatrix
	a, _ := matrix.NewDense(1, 1)                // prepare a valid 1×1 matrix
	_, err := matrix.Add(nil, a)                 // call Add with first operand nil
	require.ErrorIs(t, err, matrix.ErrNilMatrix) // expect ErrNilMatrix

	// Test Add(valid, nil) returns ErrNilMatrix
	_, err = matrix.Add(a, nil)                  // call Add with second operand nil
	require.ErrorIs(t, err, matrix.ErrNilMatrix) // expect ErrNilMatrix

	// Test Sub(nil, valid) returns ErrNilMatrix
	_, err = matrix.Sub(nil, a)                  // call Sub with first operand nil
	require.ErrorIs(t, err, matrix.ErrNilMatrix) // expect ErrNilMatrix

	// Test Sub(valid, nil) returns ErrNilMatrix
	_, err = matrix.Sub(a, nil)                  // call Sub with second operand nil
	require.ErrorIs(t, err, matrix.ErrNilMatrix) // expect ErrNilMatrix

	// Test Mul(nil, valid) returns ErrNilMatrix
	_, err = matrix.Mul(nil, a)                  // call Mul with first operand nil
	require.ErrorIs(t, err, matrix.ErrNilMatrix) // expect ErrNilMatrix

	// Test Mul(valid, nil) returns ErrNilMatrix
	_, err = matrix.Mul(a, nil)                  // call Mul with second operand nil
	require.ErrorIs(t, err, matrix.ErrNilMatrix) // expect ErrNilMatrix

	// Test Transpose(nil) returns ErrNilMatrix
	_, err = matrix.Transpose(nil)               // call Transpose with second operand nil
	require.ErrorIs(t, err, matrix.ErrNilMatrix) // expect ErrNilMatrix

	// Test Scale(nil, α) returns ErrNilMatrix
	_, err = matrix.Scale(nil, 2.0)              // call Scale with second operand nil
	require.ErrorIs(t, err, matrix.ErrNilMatrix) // expect ErrNilMatrix
}

func TestMethods_DimensionMismatch(t *testing.T) {
	t.Parallel()

	// Prepare a 3×4 and a 4×3 matrix for mismatch scenarios
	m1, _ := matrix.NewDense(3, 4) // 3 rows, 4 columns
	m2, _ := matrix.NewDense(4, 3) // 4 rows, 3 columns

	// Add should error on shape mismatch
	_, err := matrix.Add(m1, m2)                               // call Add on mismatched shapes
	require.ErrorIs(t, err, matrix.ErrMatrixDimensionMismatch) // expect ErrDimensionMismatch

	// Sub should error on shape mismatch
	_, err = matrix.Sub(m1, m2)                                // call Sub on mismatched shapes
	require.ErrorIs(t, err, matrix.ErrMatrixDimensionMismatch) // expect ErrDimensionMismatch

	// Mul should error when inner dims mismatch (4 != 4 is okay, but  m1.Cols()!=m2.Rows()? 4==4 → OK)
	// To force mismatch, use 3×4 × 2×2
	m3, _ := matrix.NewDense(2, 2)                             // 2×2 matrix
	_, err = matrix.Mul(m1, m3)                                // call Mul on inner-dimension mismatch
	require.ErrorIs(t, err, matrix.ErrMatrixDimensionMismatch) // expect ErrDimensionMismatch
}

func TestMethods_HappyPaths(t *testing.T) {
	t.Parallel()

	// Prepare two small 2×2 matrices for Add and Sub
	a, _ := matrix.NewDense(2, 2) // allocate 2×2 result
	_ = a.Set(0, 0, 1)            // a[0,0] = 1
	_ = a.Set(0, 1, 2)            // a[0,1] = 2
	_ = a.Set(1, 0, 3)            // a[1,0] = 3
	_ = a.Set(1, 1, 4)            // a[1,1] = 4

	b, _ := matrix.NewDense(2, 2) // allocate 2×2 result
	_ = b.Set(0, 0, 5)            // b[0,0] = 5
	_ = b.Set(0, 1, 6)            // b[0,1] = 6
	_ = b.Set(1, 0, 7)            // b[1,0] = 7
	_ = b.Set(1, 1, 8)            // b[1,1] = 8

	// Test Add: expected [[6,8],[10,12]]
	sum, err := matrix.Add(a, b) // perform element-wise addition
	require.NoError(t, err)      // expect no error

	var val float64
	val, _ = sum.At(0, 0)       // get sum[0,0]
	require.Equal(t, 6.0, val)  // check result
	val, _ = sum.At(0, 1)       // get sum[0,1]
	require.Equal(t, 8.0, val)  // check result
	val, _ = sum.At(1, 0)       // get sum[1,0]
	require.Equal(t, 10.0, val) // check result
	val, _ = sum.At(1, 1)       // get sum[1,1]
	require.Equal(t, 12.0, val) // check result

	// Test Sub: expected [[-4,-4],[-4,-4]]
	diff, err := matrix.Sub(a, b) // perform element-wise subtraction
	require.NoError(t, err)       // expect no error

	val, _ = diff.At(0, 0)      // get diff[0,0]
	require.Equal(t, -4.0, val) // check result
	val, _ = diff.At(0, 1)      // get diff[0,1]
	require.Equal(t, -4.0, val) // check result
	val, _ = diff.At(1, 0)      // get diff[1,0]
	require.Equal(t, -4.0, val) // check result
	val, _ = diff.At(1, 1)      // get diff[1,1]
	require.Equal(t, -4.0, val) // check result

	// Prepare m×n and n×p for Mul: 2×3 × 3×2 → 2×2
	m, _ := matrix.NewDense(2, 3) // allocate 2×3
	_ = m.Set(0, 0, 1)            // m[0,0] = 1
	_ = m.Set(0, 1, 2)            // m[0,1] = 2
	_ = m.Set(0, 2, 3)            // m[0,2] = 3
	_ = m.Set(1, 0, 4)            // m[1,0] = 4
	_ = m.Set(1, 1, 5)            // m[1,1] = 5
	_ = m.Set(1, 2, 6)            // m[1,2] = 6

	n, _ := matrix.NewDense(3, 2) // allocate 3×2
	_ = n.Set(0, 0, 7)            // n[0,0] = 7
	_ = n.Set(0, 1, 8)            // n[0,1] = 8
	_ = n.Set(1, 0, 9)            // n[1,0] = 9
	_ = n.Set(1, 1, 10)           // n[1,1] = 10
	_ = n.Set(2, 0, 11)           // n[2,0] = 11
	_ = n.Set(2, 1, 12)           // n[2,1] = 12

	// Expected product [[58,64],[139,154]]
	prod, err := matrix.Mul(m, n) // perform matrix multiplication
	require.NoError(t, err)       // expect no error

	val, _ = prod.At(0, 0)       // get prod[0,0]
	require.Equal(t, 58.0, val)  // check result
	val, _ = prod.At(0, 1)       // get prod[0,1]
	require.Equal(t, 64.0, val)  // check result
	val, _ = prod.At(1, 0)       // get prod[1,0]
	require.Equal(t, 139.0, val) // check result
	val, _ = prod.At(1, 1)       // get prod[1,1]
	require.Equal(t, 154.0, val) // check result
}

func TestMethods_TableDriven(t *testing.T) {
	t.Parallel()

	// Define test cases for various matrix shapes and values
	type tc struct {
		name         string
		aRows, aCols int
		bRows, bCols int
		alpha        float64
		wantAddErr   bool
		wantSubErr   bool
		wantMulErr   bool
	}

	tests := []tc{
		{
			// All is good: for +/− and × all conditions match
			name:  "Square",
			aRows: 3, aCols: 3,
			bRows: 3, bCols: 3,
			alpha:      2.0,
			wantAddErr: false, wantSubErr: false, wantMulErr: false,
		},
		{
			// Rectangular of the same shape: + and − ok; × is an error
			name:  "RectAdd",
			aRows: 2, aCols: 3,
			bRows: 2, bCols: 3,
			alpha:      1.5,
			wantAddErr: false, wantSubErr: false, wantMulErr: true,
		},
		{
			// Rectangular for ×: ok; + and − is an error
			name:  "RectMul",
			aRows: 2, aCols: 3,
			bRows: 3, bCols: 2,
			alpha:      2.5,
			wantAddErr: true, wantSubErr: true, wantMulErr: false,
		},
		{
			// All is bad: for +/− rows doesn't match, for × the internal dimension doesn't match
			name:  "BadMul",
			aRows: 2, aCols: 2,
			bRows: 3, bCols: 2,
			alpha:      0,
			wantAddErr: true, wantSubErr: true, wantMulErr: true,
		},
	}

	for _, c := range tests {
		c := c // capture
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			// Prepare matrices a and b of given sizes
			a, _ := matrix.NewDense(c.aRows, c.aCols)
			b, _ := matrix.NewDense(c.bRows, c.bCols)

			// Test Add
			_, err := matrix.Add(a, b)
			if c.wantAddErr {
				require.ErrorIs(t, err, matrix.ErrMatrixDimensionMismatch)
			} else {
				require.NoError(t, err)
			}

			// Test Sub
			_, err = matrix.Sub(a, b)
			if c.wantSubErr {
				require.ErrorIs(t, err, matrix.ErrMatrixDimensionMismatch)
			} else {
				require.NoError(t, err)
			}

			// Test Mul
			_, err = matrix.Mul(a, b)
			if c.wantMulErr {
				require.ErrorIs(t, err, matrix.ErrMatrixDimensionMismatch)
			} else {
				require.NoError(t, err)
			}

			// Test Scale
			res, err := matrix.Scale(a, c.alpha)
			require.NoError(t, err)
			require.NotNil(t, res)
			require.Equal(t, c.aRows, res.Rows())
			require.Equal(t, c.aCols, res.Cols())
		})
	}
}
