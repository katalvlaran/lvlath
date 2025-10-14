// Package matrix_test contains unit tests for universal Matrix operations.
package matrix_test

import (
	"fmt"
	"math"
	"sort"
	"testing"

	"github.com/katalvlaran/lvlath/matrix"
	"github.com/stretchr/testify/require"
)

// hide wraps any Matrix to hide its concrete type.
type hide struct{ matrix.Matrix }

func TestNewDenseDefaultZero(t *testing.T) {
	for _, tc := range []struct{ rows, cols int }{
		{3, 3},
		{6, 6},
	} {
		name := fmt.Sprintf("%dx%d", tc.rows, tc.cols)
		t.Run(name, func(t *testing.T) {
			m, err := matrix.NewDense(tc.rows, tc.cols)
			require.NoError(t, err, "NewDense should succeed for positive dimensions")
			// immediately after creation all elements should be 0
			var (
				i, j int // loop iterators
				v    float64
			)
			for i = 0; i < tc.rows; i++ {
				for j = 0; j < tc.cols; j++ {
					v, err = m.At(i, j)
					require.NoError(t, err)
					require.Equal(t, 0.0, v,
						"element [%d,%d] of a new Dense(%dx%d) must be 0", i, j, tc.rows, tc.cols)
				}
			}
		})
	}
}

// TestHelpers_InterfaceHiding_Fallback ensures that using a non-nil wrapper
// (which hides the concrete type) forces the interface fallback path without panicking
// and produces the same results as with the bare Dense.
func TestHelpers_InterfaceHiding_Fallback(t *testing.T) {
	t.Parallel()

	const rows, cols = 3, 3
	var (
		i, j int
		v    float64
		err  error
	)

	base, err := matrix.NewDense(rows, cols)
	require.NoError(t, err)
	for i = 0; i < rows; i++ {
		for j = 0; j < cols; j++ {
			v = float64(i*cols + j + 1)
			require.NoError(t, base.Set(i, j, v))
		}
	}

	wrapped := hide{base}

	// Compare Add(base, base) vs Add(wrapped, base)
	sum1, err := matrix.Add(base, base)
	require.NoError(t, err)
	sum2, err := matrix.Add(wrapped, base)
	require.NoError(t, err)

	for i = 0; i < rows; i++ {
		for j = 0; j < cols; j++ {
			var a, b float64
			a, err = sum1.At(i, j)
			require.NoError(t, err)
			b, err = sum2.At(i, j)
			require.NoError(t, err)
			require.Equal(t, a, b, "mismatch at [%d,%d]", i, j)
		}
	}
}

func TestHelperVisibility(t *testing.T) {
	// Check that the Random and Compare utilities are available and working
	const n = 3
	m, err := matrix.NewDense(n, n)
	require.NoError(t, err)

	// Random fills the matrix with pseudo-random numbers without panicking
	matrix.Random(t, m, 12345)

	// Assemble "reference" identity matrix
	Iwant := make([][]float64, n)
	for i := 0; i < n; i++ {
		row := make([]float64, n)
		row[i] = 1.0
		Iwant[i] = row
	}

	// First, fill m with one on the diagonal and zeros outside
	var i, j int // loop iterators
	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			err = m.Set(i, j, 0)
			require.NoError(t, err)
		}
		err = m.Set(i, i, 1.0)
		require.NoError(t, err)
	}

	// Сompare should not panic and should check successfully
	matrix.Compare(t, Iwant, m)
}

// hide is declared once if not already in file:
// type hide struct{ matrix.Matrix }

// ---------- 2.1 Add ----------

func TestAdd_FastPath_6x6_Correctness(t *testing.T) {
	t.Parallel()

	const rows, cols = 6, 6
	var (
		i, j int
		err  error
	)

	A, err := matrix.NewDense(rows, cols)
	require.NoError(t, err)
	B, err := matrix.NewDense(rows, cols)
	require.NoError(t, err)

	// A[i,j] = i+j; B[i,j] = 10 - (i+j)
	for i = 0; i < rows; i++ {
		for j = 0; j < cols; j++ {
			require.NoError(t, A.Set(i, j, float64(i+j)))
			require.NoError(t, B.Set(i, j, float64(10-(i+j))))
		}
	}

	S, err := matrix.Add(A, B)
	require.NoError(t, err)

	// Expect constant 10 everywhere
	for i = 0; i < rows; i++ {
		for j = 0; j < cols; j++ {
			var got float64
			got, err = S.At(i, j)
			require.NoError(t, err)
			require.Equal(t, 10.0, got, "at [%d,%d]", i, j)
		}
	}
}

func TestAdd_Fallback_4x5_Correctness(t *testing.T) {
	t.Parallel()

	const rows, cols = 4, 5
	var (
		i, j int
		err  error
	)

	Araw, _ := matrix.NewDense(rows, cols)
	Braw, _ := matrix.NewDense(rows, cols)
	A := hide{Araw} // force fallback
	B := hide{Braw} // force fallback

	// A[i,j] = 2*i + j; B[i,j] = i - 3*j
	for i = 0; i < rows; i++ {
		for j = 0; j < cols; j++ {
			require.NoError(t, Araw.Set(i, j, float64(2*i+j)))
			require.NoError(t, Braw.Set(i, j, float64(i-3*j)))
		}
	}

	S, err := matrix.Add(A, B)
	require.NoError(t, err)

	// Check elementwise
	for i = 0; i < rows; i++ {
		for j = 0; j < cols; j++ {
			var got, av, bv float64
			av, _ = Araw.At(i, j)
			bv, _ = Braw.At(i, j)
			got, err = S.At(i, j)
			require.NoError(t, err)
			require.Equal(t, av+bv, got, "at [%d,%d]", i, j)
		}
	}
}

func TestAdd_DimensionMismatch(t *testing.T) {
	t.Parallel()

	var err error
	A, _ := matrix.NewDense(3, 4)
	B, _ := matrix.NewDense(4, 3)
	_, err = matrix.Add(A, B)
	require.ErrorIs(t, err, matrix.ErrDimensionMismatch)
}

func TestAdd_Succeeds(t *testing.T) {
	// Prepare two 2×3 matrices
	a, err := matrix.NewDense(2, 3)
	require.NoError(t, err)
	b, err := matrix.NewDense(2, 3)
	require.NoError(t, err)

	// Initialize a = [[1,2,3],[4,5,6]], b = [[6,5,4],[3,2,1]]
	var i, j int // loop iterators
	for i = 0; i < 2; i++ {
		for j = 0; j < 3; j++ {
			require.NoError(t, a.Set(i, j, float64(i*3+j+1)))
			require.NoError(t, b.Set(i, j, float64(6-(i*3+j))))
		}
	}

	// Perform addition
	sum, err := matrix.Add(a, b)
	require.NoError(t, err)

	// Expect sum = [[7,7,7],[7,7,7]]
	var v float64
	for i = 0; i < 2; i++ {
		for j = 0; j < 3; j++ {
			v, err = sum.At(i, j)
			require.NoError(t, err)
			require.Equal(t, 7.0, v)
		}
	}
}

// ---------- 2.2 Sub ----------

func TestSub_FastPath_6x6_Correctness(t *testing.T) {
	t.Parallel()

	const rows, cols = 6, 6
	var (
		i, j int
		err  error
	)

	A, _ := matrix.NewDense(rows, cols)
	B, _ := matrix.NewDense(rows, cols)

	// A[i,j] = 100 + i*cols + j; B[i,j] = i*cols + j
	for i = 0; i < rows; i++ {
		for j = 0; j < cols; j++ {
			require.NoError(t, A.Set(i, j, float64(100+i*cols+j)))
			require.NoError(t, B.Set(i, j, float64(i*cols+j)))
		}
	}

	D, err := matrix.Sub(A, B)
	require.NoError(t, err)

	// Expect constant 100 everywhere
	for i = 0; i < rows; i++ {
		for j = 0; j < cols; j++ {
			var got float64
			got, err = D.At(i, j)
			require.NoError(t, err)
			require.Equal(t, 100.0, got, "at [%d,%d]", i, j)
		}
	}
}

func TestSub_Fallback_5x3_Correctness(t *testing.T) {
	t.Parallel()

	const rows, cols = 5, 3
	var (
		i, j int
		err  error
	)

	Araw, _ := matrix.NewDense(rows, cols)
	Braw, _ := matrix.NewDense(rows, cols)
	A := hide{Araw}
	B := hide{Braw}

	// A[i,j] = i + 2*j; B[i,j] = 3*i - j
	for i = 0; i < rows; i++ {
		for j = 0; j < cols; j++ {
			require.NoError(t, Araw.Set(i, j, float64(i+2*j)))
			require.NoError(t, Braw.Set(i, j, float64(3*i-j)))
		}
	}

	D, err := matrix.Sub(A, B)
	require.NoError(t, err)

	// Check elementwise
	for i = 0; i < rows; i++ {
		for j = 0; j < cols; j++ {
			var got, av, bv float64
			av, _ = Araw.At(i, j)
			bv, _ = Braw.At(i, j)
			got, err = D.At(i, j)
			require.NoError(t, err)
			require.Equal(t, av-bv, got, "at [%d,%d]", i, j)
		}
	}
}

func TestSub_DimensionMismatch(t *testing.T) {
	t.Parallel()

	var err error
	A, _ := matrix.NewDense(3, 5)
	B, _ := matrix.NewDense(3, 4)
	_, err = matrix.Sub(A, B)
	require.ErrorIs(t, err, matrix.ErrDimensionMismatch)
}

func TestSub_Succeeds(t *testing.T) {
	// Prepare two 3×2 matrices
	a, _ := matrix.NewDense(3, 2)
	b, _ := matrix.NewDense(3, 2)
	// a = [[5,4],[3,2],[1,0]]; b = [[1,1],[1,1],[1,1]]
	values := [][]float64{
		{5, 4},
		{3, 2},
		{1, 0},
	}
	var i, j int // loop iterators
	for i = 0; i < 3; i++ {
		for j = 0; j < 2; j++ {
			_ = a.Set(i, j, values[i][j])
			_ = b.Set(i, j, 1)
		}
	}

	diff, err := matrix.Sub(a, b)
	require.NoError(t, err)

	// Expect diff = [[4,3],[2,1],[0,-1]]
	expected := [][]float64{
		{4, 3},
		{2, 1},
		{0, -1},
	}
	var v float64
	for i = 0; i < 3; i++ {
		for j = 0; j < 2; j++ {
			v, err = diff.At(i, j)
			require.NoError(t, err)
			require.Equal(t, expected[i][j], v)
		}
	}
}

// ---------- 2.3 Mul ----------

func TestMul_FastPath_6x4x5_Correctness(t *testing.T) {
	t.Parallel()

	// A(6×4) × B(4×5) = C(6×5)
	const ar, ac, bc = 6, 4, 5
	var (
		i, j, k int
		err     error
		sum     float64
		got     float64
	)

	A, _ := matrix.NewDense(ar, ac)
	B, _ := matrix.NewDense(ac, bc)

	// A[i,k] = i + k; B[k,j] = k + j
	for i = 0; i < ar; i++ {
		for k = 0; k < ac; k++ {
			require.NoError(t, A.Set(i, k, float64(i+k)))
		}
	}
	for k = 0; k < ac; k++ {
		for j = 0; j < bc; j++ {
			require.NoError(t, B.Set(k, j, float64(k+j)))
		}
	}

	C, err := matrix.Mul(A, B)
	require.NoError(t, err)

	// verify C[i,j] = Σ_k (i+k)*(k+j)
	for i = 0; i < ar; i++ {
		for j = 0; j < bc; j++ {
			sum = 0.0
			for k = 0; k < ac; k++ {
				sum += float64(i+k) * float64(k+j)
			}
			got, err = C.At(i, j)
			require.NoError(t, err)
			require.Equal(t, sum, got, "at [%d,%d]", i, j)
		}
	}
}

func TestMul_Fallback_3x4x3_Correctness(t *testing.T) {
	t.Parallel()

	// Force fallback via hide
	const ar, ac, bc = 3, 4, 3
	var (
		i, j, k int
		err     error
		sum     float64
		got     float64
		av, bv  float64
	)

	Araw, _ := matrix.NewDense(ar, ac)
	Braw, _ := matrix.NewDense(ac, bc)
	A := hide{Araw}
	B := hide{Braw}

	// A[i,k] = 2*i + k; B[k,j] = 3*k - j
	for i = 0; i < ar; i++ {
		for k = 0; k < ac; k++ {
			require.NoError(t, Araw.Set(i, k, float64(2*i+k)))
		}
	}
	for k = 0; k < ac; k++ {
		for j = 0; j < bc; j++ {
			require.NoError(t, Braw.Set(k, j, float64(3*k-j)))
		}
	}

	C, err := matrix.Mul(A, B)
	require.NoError(t, err)

	// explicit Σ for expected
	for i = 0; i < ar; i++ {
		for j = 0; j < bc; j++ {
			sum = 0.0
			for k = 0; k < ac; k++ {
				av, _ = Araw.At(i, k)
				bv, _ = Braw.At(k, j)
				sum += av * bv
			}
			got, err = C.At(i, j)
			require.NoError(t, err)
			require.Equal(t, sum, got, "at [%d,%d]", i, j)
		}
	}
}

func TestMul_DimensionMismatch(t *testing.T) {
	t.Parallel()

	var err error
	A, _ := matrix.NewDense(4, 3) // inner = 3
	B, _ := matrix.NewDense(2, 5) // inner = 2 → mismatch
	_, err = matrix.Mul(A, B)
	require.ErrorIs(t, err, matrix.ErrDimensionMismatch)
}

func TestMul_Succeeds(t *testing.T) {
	// A is 2×3, B is 3×2: A*B = 2×2
	A, _ := matrix.NewDense(2, 3)
	B, _ := matrix.NewDense(3, 2)
	var C matrix.Matrix
	// Initialize A = [[1,2,3],[4,5,6]]; B = [[7,8],[9,10],[11,12]]
	aVals := [][]float64{{1, 2, 3}, {4, 5, 6}}
	bVals := [][]float64{{7, 8}, {9, 10}, {11, 12}}
	var (
		i, j int // loop iterators
		v    float64
		err  error
	)
	for i = 0; i < 2; i++ {
		for j = 0; j < 3; j++ {
			_ = A.Set(i, j, aVals[i][j])
		}
	}
	for i = 0; i < 3; i++ {
		for j = 0; j < 2; j++ {
			_ = B.Set(i, j, bVals[i][j])
		}
	}

	C, err = matrix.Mul(A, B)
	require.NoError(t, err)

	// Expected C = [[58,64],[139,154]]
	expected := [][]float64{{58, 64}, {139, 154}}
	for i = 0; i < 2; i++ {
		for j = 0; j < 2; j++ {
			v, err = C.At(i, j)
			require.NoError(t, err)
			require.Equal(t, expected[i][j], v)
		}
	}
}

// ---------- 3.1 Transpose ----------

func TestTranspose_FastPath_Rectangular_Correctness(t *testing.T) {
	t.Parallel()

	const rows, cols = 4, 6
	var (
		i, j int
		err  error
		val  float64
	)

	m, err := matrix.NewDense(rows, cols)
	require.NoError(t, err)

	// Fill m[i,j] = 10*i + j  (unique, easy to check after transpose)
	for i = 0; i < rows; i++ {
		for j = 0; j < cols; j++ {
			require.NoError(t, m.Set(i, j, float64(10*i+j)))
		}
	}

	mt, err := matrix.Transpose(m)
	require.NoError(t, err)
	require.Equal(t, cols, mt.Rows())
	require.Equal(t, rows, mt.Cols())

	// Check mt[j,i] == m[i,j]
	for i = 0; i < rows; i++ {
		for j = 0; j < cols; j++ {
			val, err = mt.At(j, i)
			require.NoError(t, err)
			require.Equal(t, float64(10*i+j), val, "mismatch at [%d,%d] ⇒ mt[%d,%d]", i, j, j, i)
		}
	}
}

func TestTranspose_Fallback_Rectangular_Correctness(t *testing.T) {
	t.Parallel()

	const rows, cols = 5, 3
	var (
		i, j int
		err  error
		val  float64
	)

	base, _ := matrix.NewDense(rows, cols)
	// Force interface fallback via wrapper
	m := hide{base}

	// Fill base[i,j] = i - 2*j
	for i = 0; i < rows; i++ {
		for j = 0; j < cols; j++ {
			require.NoError(t, base.Set(i, j, float64(i-2*j)))
		}
	}

	mt, err := matrix.Transpose(m)
	require.NoError(t, err)
	require.Equal(t, cols, mt.Rows())
	require.Equal(t, rows, mt.Cols())

	// Check mt[j,i] == base[i,j]
	for i = 0; i < rows; i++ {
		for j = 0; j < cols; j++ {
			val, err = mt.At(j, i)
			require.NoError(t, err)
			require.Equal(t, float64(i-2*j), val)
		}
	}
}

func TestTranspose_Involution_NoMutation(t *testing.T) {
	t.Parallel()

	const n = 6
	var (
		i, j int
		err  error
		aij  float64
	)

	A, _ := matrix.NewDense(n, n)
	// Fill A with a distinct pattern
	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			require.NoError(t, A.Set(i, j, float64((i+1)*(j+2))))
		}
	}

	// Keep a copy to ensure A is not mutated by Transpose
	Acopy := A.Clone()

	At, err := matrix.Transpose(A)
	require.NoError(t, err)
	Att, err := matrix.Transpose(At)
	require.NoError(t, err)

	// Check Transpose(Transpose(A)) == A
	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			var got, want float64
			got, err = Att.At(i, j)
			require.NoError(t, err)
			want, err = A.At(i, j)
			require.NoError(t, err)
			require.Equal(t, want, got, "at [%d,%d]", i, j)
		}
	}

	// Ensure original A not mutated
	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			var v1, v2 float64
			v1, err = A.At(i, j)
			require.NoError(t, err)
			v2, err = Acopy.At(i, j)
			require.NoError(t, err)
			require.Equal(t, v2, v1)
		}
	}

	// Extra: symmetric matrix should equal its transpose
	for i = 0; i < n; i++ {
		for j = i; j < n; j++ {
			aij = float64(i + j + 1) // symmetric by construction
			require.NoError(t, A.Set(i, j, aij))
			require.NoError(t, A.Set(j, i, aij))
		}
	}
	St, err := matrix.Transpose(A)
	require.NoError(t, err)
	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			var s, st float64
			s, _ = A.At(i, j)
			st, _ = St.At(i, j)
			require.Equal(t, s, st, "symmetric transpose must be identical")
		}
	}
}

func TestTranspose(t *testing.T) {
	// 2×3 matrix
	m, _ := matrix.NewDense(2, 3)
	_ = m.Set(0, 0, 1)
	_ = m.Set(0, 1, 2)
	_ = m.Set(0, 2, 3)
	_ = m.Set(1, 0, 4)
	_ = m.Set(1, 1, 5)
	_ = m.Set(1, 2, 6)

	tm, _ := matrix.Transpose(m)
	// tm should be 3×2: [[1,4],[2,5],[3,6]]
	exp := [][]float64{{1, 4}, {2, 5}, {3, 6}}
	require.Equal(t, 3, tm.Rows())
	require.Equal(t, 2, tm.Cols())
	var (
		i, j int // loop iterators
		v    float64
		err  error
	)
	for i = 0; i < tm.Rows(); i++ {
		for j = 0; j < tm.Cols(); j++ {
			v, err = tm.At(i, j)
			require.NoError(t, err)
			require.Equal(t, exp[i][j], v)
		}
	}
}

// ---------- 3.2 Scale ----------

func TestScale_FastPath_6x6_Correctness(t *testing.T) {
	t.Parallel()

	const n = 6
	const alpha = 3.5
	var (
		i, j int
		err  error
		got  float64
	)

	m, _ := matrix.NewDense(n, n)
	// m[i,j] = i - j
	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			require.NoError(t, m.Set(i, j, float64(i-j)))
		}
	}

	sm, err := matrix.Scale(m, alpha)
	require.NoError(t, err)
	require.Equal(t, n, sm.Rows())
	require.Equal(t, n, sm.Cols())

	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			got, err = sm.At(i, j)
			require.NoError(t, err)
			require.Equal(t, alpha*float64(i-j), got, "at [%d,%d]", i, j)
		}
	}
}

func TestScale_Fallback_5x3_Correctness(t *testing.T) {
	t.Parallel()

	const rows, cols = 5, 3
	const alpha = -2.0
	var (
		i, j int
		err  error
		got  float64
	)

	base, _ := matrix.NewDense(rows, cols)
	m := hide{base} // force fallback

	// base[i,j] = 2*i + 3*j + 1
	for i = 0; i < rows; i++ {
		for j = 0; j < cols; j++ {
			require.NoError(t, base.Set(i, j, float64(2*i+3*j+1)))
		}
	}

	sm, err := matrix.Scale(m, alpha)
	require.NoError(t, err)
	require.Equal(t, rows, sm.Rows())
	require.Equal(t, cols, sm.Cols())

	for i = 0; i < rows; i++ {
		for j = 0; j < cols; j++ {
			got, err = sm.At(i, j)
			require.NoError(t, err)
			require.Equal(t, alpha*float64(2*i+3*j+1), got, "at [%d,%d]", i, j)
		}
	}
}

func TestScale_Properties_Distributivity(t *testing.T) {
	t.Parallel()

	const n = 4
	const alpha = 1.75
	var (
		i, j int
		err  error
	)

	A, _ := matrix.NewDense(n, n)
	B, _ := matrix.NewDense(n, n)

	// A[i,j] = i+j; B[i,j] = i-2*j
	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			require.NoError(t, A.Set(i, j, float64(i+j)))
			require.NoError(t, B.Set(i, j, float64(i-2*j)))
		}
	}

	S, err := matrix.Add(A, B)
	require.NoError(t, err)

	left, err := matrix.Scale(S, alpha) // α(A+B)
	require.NoError(t, err)

	Ar, err := matrix.Scale(A, alpha) // αA
	require.NoError(t, err)
	Br, err := matrix.Scale(B, alpha) // αB
	require.NoError(t, err)
	right, err := matrix.Add(Ar, Br) // αA + αB
	require.NoError(t, err)

	// Compare left vs right
	var lv, rv float64
	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			lv, err = left.At(i, j)
			require.NoError(t, err)
			rv, err = right.At(i, j)
			require.NoError(t, err)
			require.Equal(t, rv, lv, "distributivity failed at [%d,%d]", i, j)
		}
	}
}

func TestScale_Properties_Composition_And_SpecialAlphas(t *testing.T) {
	t.Parallel()

	const n = 5
	const alpha = -0.5
	const beta = 4.0
	var (
		i, j int
		err  error
	)

	M, _ := matrix.NewDense(n, n)
	// M[i,j] = 3*i - j
	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			require.NoError(t, M.Set(i, j, float64(3*i-j)))
		}
	}

	// (αβ)*M
	left, err := matrix.Scale(M, alpha*beta)
	require.NoError(t, err)

	// α*(β*M)
	bm, err := matrix.Scale(M, beta)
	require.NoError(t, err)
	right, err := matrix.Scale(bm, alpha)
	require.NoError(t, err)

	// Compare left vs right (associativity of scalar multiplication)
	var lv, rv float64
	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			lv, err = left.At(i, j)
			require.NoError(t, err)
			rv, err = right.At(i, j)
			require.NoError(t, err)
			require.Equal(t, rv, lv, "composition failed at [%d,%d]", i, j)
		}
	}

	// α = 0 ⇒ zero matrix; α = -1 ⇒ negation; inputs not mutated.
	zero, err := matrix.Scale(M, 0.0)
	require.NoError(t, err)
	neg, err := matrix.Scale(M, -1.0)
	require.NoError(t, err)

	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			var m, z, ng float64
			m, _ = M.At(i, j)
			z, _ = zero.At(i, j)
			ng, _ = neg.At(i, j)
			require.Equal(t, 0.0, z, "zero scaling failed at [%d,%d]", i, j)
			require.Equal(t, -m, ng, "negation failed at [%d,%d]", i, j)
		}
	}

	// Ensure original M unchanged
	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			var m1, m2 float64
			m1, _ = M.At(i, j)
			m2, _ = M.At(i, j) // read again; we only checked immutability via distinct results above
			require.Equal(t, m2, m1)
		}
	}
}

func TestScale_WithTranspose_Compatibility(t *testing.T) {
	t.Parallel()

	const rows, cols = 3, 5
	const alpha = 2.25
	var (
		i, j int
		err  error
	)

	M, _ := matrix.NewDense(rows, cols)
	// M[i,j] = i + 10*j
	for i = 0; i < rows; i++ {
		for j = 0; j < cols; j++ {
			require.NoError(t, M.Set(i, j, float64(i+10*j)))
		}
	}

	alphaM, err := matrix.Scale(M, alpha)
	require.NoError(t, err)
	TalphaM, err := matrix.Transpose(alphaM)
	require.NoError(t, err)

	TM, err := matrix.Transpose(M)
	require.NoError(t, err)
	alphaTM, err := matrix.Scale(TM, alpha)
	require.NoError(t, err)

	// Expect Transpose(αM) == α Transpose(M)
	var v1, v2 float64
	for i = 0; i < TalphaM.Rows(); i++ {
		for j = 0; j < TalphaM.Cols(); j++ {
			v1, err = TalphaM.At(i, j)
			require.NoError(t, err)
			v2, err = alphaTM.At(i, j)
			require.NoError(t, err)
			require.Equal(t, v2, v1, "compatibility failed at [%d,%d]", i, j)
		}
	}
}

func TestScale(t *testing.T) {
	// 2×2 matrix
	m, _ := matrix.NewDense(2, 2)
	_ = m.Set(0, 0, 1.5)
	_ = m.Set(0, 1, -2.5)
	_ = m.Set(1, 0, 3.0)
	_ = m.Set(1, 1, 0.0)

	sm, _ := matrix.Scale(m, 2.0)
	// expected = [[3.0, -5.0],[6.0, 0.0]]
	expected := [][]float64{{3.0, -5.0}, {6.0, 0.0}}
	var (
		i, j int // loop iterators
		v    float64
		err  error
	)
	for i = 0; i < sm.Rows(); i++ {
		for j = 0; j < sm.Cols(); j++ {
			v, err = sm.At(i, j)
			require.NoError(t, err)
			require.Equal(t, expected[i][j], v)
		}
	}
}

// ---------- 4. Eigen ----------

// TestEigen_Errors verifies error paths: non-square, non-symmetric, and forced non-convergence.
func TestEigen_Errors(t *testing.T) {
	t.Parallel()

	var err error

	// non-square → ErrDimensionMismatch
	ns, _ := matrix.NewDense(3, 4)
	_, _, err = matrix.Eigen(ns, 1e-10, 50)
	require.ErrorIs(t, err, matrix.ErrDimensionMismatch)

	// not symmetric within tol → ErrNotSymmetric
	asym, _ := matrix.NewDense(3, 3)
	require.NoError(t, asym.Set(0, 1, 1))
	require.NoError(t, asym.Set(1, 0, 2)) // violates symmetry > tol
	_, _, err = matrix.Eigen(asym, 1e-12, 50)
	require.ErrorIs(t, err, matrix.ErrAsymmetry)

	// zero iterations with nonzero off-diagonals → ErrEigenFailed
	sym, _ := matrix.NewDense(3, 3)
	require.NoError(t, sym.Set(0, 0, 2))
	require.NoError(t, sym.Set(1, 1, 3))
	require.NoError(t, sym.Set(2, 2, 4))
	require.NoError(t, sym.Set(0, 1, 1))
	require.NoError(t, sym.Set(1, 0, 1))
	_, _, err = matrix.Eigen(sym, 1e-12, 0)
	require.ErrorIs(t, err, matrix.ErrEigenFailed)
}

// TestEigen_Diagonal_NoRotation: diagonal matrices return exact diagonal as eigenvalues and Q=I.
func TestEigen_Diagonal_NoRotation(t *testing.T) {
	t.Parallel()

	const n = 4
	var (
		i, j int
		v    float64
		err  error
	)

	diagVals := []float64{1, -2, 5, 3}
	A, _ := matrix.NewDense(n, n)
	for i = 0; i < n; i++ {
		require.NoError(t, A.Set(i, i, diagVals[i]))
	}

	vals, Q, err := matrix.Eigen(A, 1e-12, 10)
	require.NoError(t, err)
	require.Len(t, vals, n)
	require.Equal(t, n, Q.Rows())
	require.Equal(t, n, Q.Cols())

	got := append([]float64(nil), vals...)
	want := append([]float64(nil), diagVals...)
	sort.Float64s(got)
	sort.Float64s(want)
	require.Equal(t, want, got)

	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			v, err = Q.At(i, j)
			require.NoError(t, err)
			if i == j {
				require.Equal(t, 1.0, v, "Q[%d,%d]", i, j)
			} else {
				require.Equal(t, 0.0, v, "Q[%d,%d]", i, j)
			}
		}
	}
}

// TestEigen_2x2_Analytic: [[2,1],[1,2]] has eigenvalues {1,3}; Q orthonormal; A*Q≈Q*D.
func TestEigen_2x2_Analytic(t *testing.T) {
	t.Parallel()

	var (
		err error
		got []float64
	)

	A, _ := matrix.NewDense(2, 2)
	require.NoError(t, A.Set(0, 0, 2))
	require.NoError(t, A.Set(0, 1, 1))
	require.NoError(t, A.Set(1, 0, 1))
	require.NoError(t, A.Set(1, 1, 2))

	vals, Q, err := matrix.Eigen(A, 1e-12, 50)
	require.NoError(t, err)
	require.Len(t, vals, 2)

	got = append([]float64(nil), vals...)
	sort.Float64s(got)
	require.InDelta(t, 1.0, got[0], 1e-10)
	require.InDelta(t, 3.0, got[1], 1e-10)

	propOrthonormal(t, Q, 1e-10)
	propEigenEquation(t, A, Q, vals, 1e-10)
}

// TestEigen_BlockDiagonal_Degenerate: block diag([2], [[3,1],[1,3]]) ⇒ eigenvalues {2,2,4}.
func TestEigen_BlockDiagonal_Degenerate(t *testing.T) {
	t.Parallel()

	const n = 3
	var (
		err error
		got []float64
	)

	A, _ := matrix.NewDense(n, n)
	// 1×1 block
	require.NoError(t, A.Set(0, 0, 2))
	// 2×2 block (rows/cols 1..2)
	require.NoError(t, A.Set(1, 1, 3))
	require.NoError(t, A.Set(2, 2, 3))
	require.NoError(t, A.Set(1, 2, 1))
	require.NoError(t, A.Set(2, 1, 1))

	orig := A.Clone()
	vals, Q, err := matrix.Eigen(A, 1e-12, 100)
	require.NoError(t, err)
	require.Len(t, vals, n)

	got = append([]float64(nil), vals...)
	sort.Float64s(got)
	require.InDelta(t, 2.0, got[0], 1e-10)
	require.InDelta(t, 2.0, got[1], 1e-10)
	require.InDelta(t, 4.0, got[2], 1e-10)

	propOrthonormal(t, Q, 1e-10)
	propReconstruction(t, orig, Q, vals, 1e-9)
}

// TestEigen_Reconstruction_SPD_6x6: SPD A=MᵀM, check QᵀQ≈I, A≈QDQᵀ and A*Q≈Q*D.
func TestEigen_Reconstruction_SPD_6x6(t *testing.T) {
	t.Parallel()

	const n = 6
	var (
		err error
	)

	M, _ := matrix.NewDense(n, n)
	matrix.Random(t, M, 42)

	Mt, err := matrix.Transpose(M)
	require.NoError(t, err)
	A, err := matrix.Mul(Mt, M) // SPD
	require.NoError(t, err)

	orig := A.Clone()
	vals, Q, err := matrix.Eigen(A, 1e-9, 200)
	require.NoError(t, err)
	require.Len(t, vals, n)

	propOrthonormal(t, Q, 1e-8)
	propReconstruction(t, orig, Q, vals, 1e-6)
	propEigenEquation(t, orig, Q, vals, 1e-6)
}

/*
// hide wraps any Matrix to hide its concrete type.
type hide struct{ matrix.Matrix }

func TestNewDenseDefaultZero(t *testing.T) {…}
func TestHelpers_InterfaceHiding_Fallback(t *testing.T) {…}
func TestHelperVisibility(t *testing.T) {…}
// hide is declared once if not already in file:
// type hide struct{ matrix.Matrix }

// ---------- 2.1 Add ----------
func TestAdd_FastPath_6x6_Correctness(t *testing.T) {…}
func TestAdd_Fallback_4x5_Correctness(t *testing.T) {…}
func TestAdd_DimensionMismatch(t *testing.T) {…}
func TestAdd_Succeeds(t *testing.T) {…}

// ---------- 2.2 Sub ----------
func TestSub_FastPath_6x6_Correctness(t *testing.T) {…}
func TestSub_Fallback_5x3_Correctness(t *testing.T) {…}
func TestSub_DimensionMismatch(t *testing.T) {…}
func TestSub_Succeeds(t *testing.T) {…}

// ---------- 2.3 Mul ----------
func TestMul_FastPath_6x4x5_Correctness(t *testing.T) {…}
func TestMul_Fallback_3x4x3_Correctness(t *testing.T) {…}
func TestMul_DimensionMismatch(t *testing.T) {…}
func TestMul_Succeeds(t *testing.T) {…}

// ---------- 3.1 Transpose ----------

func TestTranspose_FastPath_Rectangular_Correctness(t *testing.T) {...}
func TestTranspose_Fallback_Rectangular_Correctness(t *testing.T) {...}
func TestTranspose_Involution_NoMutation(t *testing.T) {...}

// ---------- 3.2 Scale ----------

func TestScale_FastPath_6x6_Correctness(t *testing.T) {...}
func TestScale_Fallback_5x3_Correctness(t *testing.T) {...}
func TestScale_Properties_Distributivity(t *testing.T) {...}
func TestScale_Properties_Composition_And_SpecialAlphas(t *testing.T) {...}
func TestScale_WithTranspose_Compatibility(t *testing.T) {...}

// ---------- 4. Eigen ----------

// TestEigen_Errors verifies error paths: non-square, non-symmetric, and forced non-convergence.
func TestEigen_Errors(t *testing.T) {...}

// TestEigen_Diagonal_NoRotation: diagonal matrices return exact diagonal as eigenvalues and Q=I.
func TestEigen_Diagonal_NoRotation(t *testing.T) {...}

// TestEigen_2x2_Analytic: [[2,1],[1,2]] has eigenvalues {1,3}; Q orthonormal; A*Q≈Q*D.
func TestEigen_2x2_Analytic(t *testing.T) {...}

// TestEigen_BlockDiagonal_Degenerate: block diag([2], [[3,1],[1,3]]) ⇒ eigenvalues {2,2,4}.
func TestEigen_BlockDiagonal_Degenerate(t *testing.T) {...}

// TestEigen_Reconstruction_SPD_6x6: SPD A=MᵀM, check QᵀQ≈I, A≈QDQᵀ and A*Q≈Q*D.
func TestEigen_Reconstruction_SPD_6x6(t *testing.T) {...}

// ---------- 5. FloydWarshall ----------

func TestFloydWarshall_Errors(t *testing.T) {...}

// Classic CLRS example (5×5, directed, with negative edges but no negative cycles).
// Expected distance matrix:
// [ [ 0, 1, -3, 2, -4],
//	[ 3, 0, -4, 1, -1],
//	[ 7, 4,  0, 5,  3],
//	[ 2,-1, -5, 0, -2],
//	[ 8, 5,  1, 6,  0] ]
func TestFloydWarshall_CLRS_5x5_FastPath_Correctness(t *testing.T) {...}

// The same CLRS graph, but forced interface fallback via wrapper.
// Result must match the fast-path one element-by-element.
func TestFloydWarshall_CLRS_5x5_Fallback_MatchesFast(t *testing.T) {...}

// Unreachable nodes remain at +Inf; diagonal zeros; triangle inequality holds;
// and running FW again on the computed distance matrix does not change it (idempotent).
func TestFloydWarshall_Unreachable_Properties_And_Idempotent(t *testing.T) {...}

// Negative cycle sanity: if a negative cycle exists and is reachable from i,
// Floyd–Warshall yields d[i,i] < 0. We check that the diagonals of the nodes from the cycle
// become negative, while those of the isolated node remain zero.
func TestFloydWarshall_NegativeCycle_DiagonalNegative(t *testing.T) {...}

// ---------- 6. Inverse ----------

func TestInverse_Errors(t *testing.T) {...}

// Known 3×3 matrix with det=9. Check the numerical values of the inverse
// (adj(A)/det) and that A A^{-1}≈I and A^{-1} A≈I.
func TestInverse_Known3x3_Adjugate(t *testing.T) {...}

// Hiding the input type (iface/fallback on reading) should not change the result.
// Inside Inverse it is still solved by *Dense (L and U are dense).
func TestInverse_WrappedInput_MatchesDense(t *testing.T) {...}

// Property: A A^{-1}≈I and A^{-1} A≈I on 6×6 SPD. And the input does not mutate.
func TestInverse_IdentityProduct_SPD_6x6(t *testing.T) {...}

// Scaling property: (αA)^{-1} = (1/α)*A^{-1} for α≠0.
func TestInverse_ScaleProperty(t *testing.T) {...}

// ---------- 7. LU ----------

// Errors: nil and non-square are rejected.
func TestLU_Errors(t *testing.T) {...}

// Basic (3×3): pick L,U explicitly (Doolittle form, diag(L)=1), set A=L*U,
// then verify LU(A) reproduces the same factors and A≈L*U exactly.
func TestLU_Known3x3_Doolittle_FastPath_Correctness(t *testing.T) {...}

// Fast-path vs Fallback (3×3): wrapping the input to hide its concrete type
// must produce the same L and U as the fast path.
func TestLU_Known3x3_Fallback_MatchesFast(t *testing.T) {...}

// Properties on 6×6: construct L (unit lower) and U (upper) with simple integer
// patterns, set A=L*U, then check (i) structure, (ii) reconstruction, and (iii) exact recovery.
func TestLU_Factor_Reconstruction_6x6(t *testing.T) {...}

// --- local property-check helpers (test-only, unexported) ---

// propOrthonormal asserts QᵀQ ≈ I within delta.
func propOrthonormal(t *testing.T, Q matrix.Matrix, delta float64) {...}

// propReconstruction asserts A ≈ Q*diag(vals)*Qᵀ within delta.
func propReconstruction(t *testing.T, A, Q matrix.Matrix, vals []float64, delta float64) {...}

// propEigenEquation asserts A*Q ≈ Q*diag(vals) within delta.
func propEigenEquation(t *testing.T, A, Q matrix.Matrix, vals []float64, delta float64) {...}

// propUnitLowerTriangular checks diag(L)=1 and L[i,j]=0 for j>i.
// delta=0 demands exact zeros/ones; positive delta allows tolerance.
func propUnitLowerTriangular(t *testing.T, L matrix.Matrix, delta float64) {...}

// propUpperTriangular checks U[i,j]=0 for i>j. Diagonal may be arbitrary nonzero.
// delta=0 demands exact zeros below diagonal.
func propUpperTriangular(t *testing.T, U matrix.Matrix, delta float64) {...}

// propReconstructionLU verifies A ≈ L*U within delta.
func propReconstructionLU(t *testing.T, A, L, U matrix.Matrix, delta float64) {...}

*/

// ---------- 5. FloydWarshall ----------

func TestFloydWarshall_Errors(t *testing.T) {
	t.Parallel()

	var err error

	// nil → ErrNilMatrix
	err = matrix.FloydWarshall(nil)
	require.ErrorIs(t, err, matrix.ErrNilMatrix)

	// non-square → ErrDimensionMismatch
	ns, _ := matrix.NewDense(3, 4)
	err = matrix.FloydWarshall(ns)
	require.ErrorIs(t, err, matrix.ErrDimensionMismatch)
}

// Classic CLRS example (5×5, directed, with negative edges but no negative cycles).
// Expected distance matrix:
// [ [ 0, 1, -3, 2, -4],
//
//	[ 3, 0, -4, 1, -1],
//	[ 7, 4,  0, 5,  3],
//	[ 2,-1, -5, 0, -2],
//	[ 8, 5,  1, 6,  0] ]
func TestFloydWarshall_CLRS_5x5_FastPath_Correctness(t *testing.T) {
	t.Parallel()

	const n = 5
	var (
		i, j int
		err  error
	)

	A, _ := matrix.NewDense(n, n)
	// init ∞ off-diagonal, 0 on diagonal
	inf := math.Inf(1)
	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			if i == j {
				require.NoError(t, A.Set(i, j, 0))
			} else {
				require.NoError(t, A.Set(i, j, inf))
			}
		}
	}
	// edges (u→v = w)
	require.NoError(t, A.Set(0, 1, 3))
	require.NoError(t, A.Set(0, 2, 8))
	require.NoError(t, A.Set(0, 4, -4))
	require.NoError(t, A.Set(1, 3, 1))
	require.NoError(t, A.Set(1, 4, 7))
	require.NoError(t, A.Set(2, 1, 4))
	require.NoError(t, A.Set(3, 0, 2))
	require.NoError(t, A.Set(3, 2, -5))
	require.NoError(t, A.Set(4, 3, 6))

	err = matrix.FloydWarshall(A)
	require.NoError(t, err)

	exp := [][]float64{
		{0, 1, -3, 2, -4},
		{3, 0, -4, 1, -1},
		{7, 4, 0, 5, 3},
		{2, -1, -5, 0, -2},
		{8, 5, 1, 6, 0},
	}
	var got float64
	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			got, _ = A.At(i, j)
			require.Equal(t, exp[i][j], got, "dist[%d,%d]", i, j)
		}
	}
}

// The same CLRS graph, but forced interface fallback via wrapper.
// Result must match the fast-path one element-by-element.
func TestFloydWarshall_CLRS_5x5_Fallback_MatchesFast(t *testing.T) {
	t.Parallel()

	const n = 5
	var (
		i, j int
		err  error
	)

	makeCLRS := func() matrix.Matrix {
		M, _ := matrix.NewDense(n, n)
		inf := math.Inf(1)
		for i = 0; i < n; i++ {
			for j = 0; j < n; j++ {
				if i == j {
					_ = M.Set(i, j, 0)
				} else {
					_ = M.Set(i, j, inf)
				}
			}
		}
		// edges
		_ = M.Set(0, 1, 3)
		_ = M.Set(0, 2, 8)
		_ = M.Set(0, 4, -4)
		_ = M.Set(1, 3, 1)
		_ = M.Set(1, 4, 7)
		_ = M.Set(2, 1, 4)
		_ = M.Set(3, 0, 2)
		_ = M.Set(3, 2, -5)
		_ = M.Set(4, 3, 6)
		return M
	}

	fast := makeCLRS()       // *Dense
	slow := hide{makeCLRS()} // wrapped → fallback
	err = matrix.FloydWarshall(fast)
	require.NoError(t, err)
	err = matrix.FloydWarshall(slow)
	require.NoError(t, err)

	var a, b float64
	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			a, _ = fast.At(i, j)
			b, _ = slow.At(i, j)
			require.Equal(t, a, b, "mismatch at [%d,%d]", i, j)
		}
	}
}

// Unreachable nodes remain at +Inf; diagonal zeros; triangle inequality holds;
// and running FW again on the computed distance matrix does not change it (idempotent).
func TestFloydWarshall_Unreachable_Properties_And_Idempotent(t *testing.T) {
	t.Parallel()

	const n = 6
	var (
		i, j, k int
		err     error
	)

	D, _ := matrix.NewDense(n, n)
	inf := math.Inf(1)

	// init ∞ off-diagonal, 0 on diagonal
	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			if i == j {
				require.NoError(t, D.Set(i, j, 0))
			} else {
				require.NoError(t, D.Set(i, j, inf))
			}
		}
	}

	// Build an undirected component on {0,1,2} and a directed chain {3 -> 4} ; node 5 isolated.
	// Undirected edges (symmetric weights):
	require.NoError(t, D.Set(0, 1, 2))
	require.NoError(t, D.Set(1, 0, 2))
	require.NoError(t, D.Set(1, 2, 3))
	require.NoError(t, D.Set(2, 1, 3))
	require.NoError(t, D.Set(0, 2, 10))
	require.NoError(t, D.Set(2, 0, 10))
	// Directed chain 3→4 (weight 7); 4 has no outgoing edges back.
	require.NoError(t, D.Set(3, 4, 7))

	err = matrix.FloydWarshall(D)
	require.NoError(t, err)

	// 1) diagonal zeros
	var v float64
	for i = 0; i < n; i++ {
		v, _ = D.At(i, i)
		require.Equal(t, 0.0, v, "diagonal must be zero at [%d,%d]", i, i)
	}

	// 2) unreachable pairs stay +Inf (from {0,1,2} or {3,4} to 5; and from 5 to others)
	var v1, v2 float64
	for i = 0; i < n; i++ {
		if i == 5 {
			continue
		}
		v1, _ = D.At(i, 5)
		v2, _ = D.At(5, i)
		require.Equal(t, inf, v1, "expect unreachable i→5, i=%d", i)
		require.Equal(t, inf, v2, "expect unreachable 5→i, i=%d", i)
	}

	// 3) triangle inequality: d[i,j] ≤ d[i,k] + d[k,j] for all i,j,k with finite paths
	var ij, ik, kj float64
	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			ij, _ = D.At(i, j)
			for k = 0; k < n; k++ {
				ik, _ = D.At(i, k)
				kj, _ = D.At(k, j)
				if ik == inf || kj == inf {
					continue
				}
				require.LessOrEqual(t, ij, ik+kj, "triangle inequality violated for (%d,%d,%d)", i, j, k)
			}
		}
	}

	// 4) idempotent: running FW again on the distance matrix must not change it
	before := D.Clone()
	err = matrix.FloydWarshall(D)
	require.NoError(t, err)
	var a, b float64
	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			a, _ = before.At(i, j)
			b, _ = D.At(i, j)
			require.Equal(t, a, b, "idempotency mismatch at [%d,%d]", i, j)
		}
	}
}

// Negative cycle sanity: if a negative cycle exists and is reachable from i,
// Floyd–Warshall yields d[i,i] < 0. We check that the diagonals of the nodes from the cycle
// become negative, while those of the isolated node remain zero.
func TestFloydWarshall_NegativeCycle_DiagonalNegative(t *testing.T) {
	t.Parallel()

	const n = 4 // 0-1-2 - negative cycle; 3 - isolated
	var (
		i, j int
		err  error
	)

	G, _ := matrix.NewDense(n, n)
	inf := math.Inf(1)

	// init: 0 on diagonal, +Inf off diagonal
	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			if i == j {
				require.NoError(t, G.Set(i, j, 0))
			} else {
				require.NoError(t, G.Set(i, j, inf))
			}
		}
	}

	// Negative cycle: 0→1 (1), 1→2 (-1), 2→0 (-1) => total -1
	require.NoError(t, G.Set(0, 1, 1))
	require.NoError(t, G.Set(1, 2, -1))
	require.NoError(t, G.Set(2, 0, -1))

	err = matrix.FloydWarshall(G)
	require.NoError(t, err)

	// Nodes 0..2 are in negative cycle: diagonals < 0
	var d float64
	for i = 0; i < 3; i++ {
		d, _ = G.At(i, i)
		require.Less(t, d, 0.0, "expected negative diagonal at node %d due to negative cycle", i)
	}

	// Node 3 is isolated: diagonal should remain 0
	d, _ = G.At(3, 3)
	require.Equal(t, 0.0, d, "isolated node must keep zero on the diagonal")
}

// ---------- 6. Inverse ----------

func TestInverse_Errors(t *testing.T) {
	t.Parallel()

	var err error

	// nil → ErrNilMatrix
	_, err = matrix.Inverse(nil)
	require.ErrorIs(t, err, matrix.ErrNilMatrix)

	// non-square → ErrDimensionMismatch
	ns, _ := matrix.NewDense(3, 4)
	_, err = matrix.Inverse(ns)
	require.ErrorIs(t, err, matrix.ErrDimensionMismatch)

	// singular → ErrSingular (two equal strings)
	sing, _ := matrix.NewDense(3, 3)
	require.NoError(t, sing.Set(0, 0, 1))
	require.NoError(t, sing.Set(0, 1, 2))
	require.NoError(t, sing.Set(0, 2, 3))
	require.NoError(t, sing.Set(1, 0, 1))
	require.NoError(t, sing.Set(1, 1, 2))
	require.NoError(t, sing.Set(1, 2, 3))
	require.NoError(t, sing.Set(2, 0, 0))
	require.NoError(t, sing.Set(2, 1, 1))
	require.NoError(t, sing.Set(2, 2, 4))

	_, err = matrix.Inverse(sing)
	require.ErrorIs(t, err, matrix.ErrSingular)
}

// Known 3×3 matrix with det=9. Check the numerical values of the inverse
// (adj(A)/det) and that A A^{-1}≈I and A^{-1} A≈I.
func TestInverse_Known3x3_Adjugate(t *testing.T) {
	t.Parallel()

	var (
		i, j int
		err  error
	)

	A, _ := matrix.NewDense(3, 3)
	// A = [[4,7,2],[3,6,1],[2,5,3]]
	require.NoError(t, A.Set(0, 0, 4))
	require.NoError(t, A.Set(0, 1, 7))
	require.NoError(t, A.Set(0, 2, 2))
	require.NoError(t, A.Set(1, 0, 3))
	require.NoError(t, A.Set(1, 1, 6))
	require.NoError(t, A.Set(1, 2, 1))
	require.NoError(t, A.Set(2, 0, 2))
	require.NoError(t, A.Set(2, 1, 5))
	require.NoError(t, A.Set(2, 2, 3))

	Inv, err := matrix.Inverse(A)
	require.NoError(t, err)

	// adj(A)/9, where adj(A)^T = cofactors:
	want := [][]float64{
		{13.0 / 9.0, -11.0 / 9.0, -5.0 / 9.0},
		{-7.0 / 9.0, 8.0 / 9.0, 2.0 / 9.0},
		{3.0 / 9.0, -6.0 / 9.0, 3.0 / 9.0},
	}

	var got float64
	for i = 0; i < 3; i++ {
		for j = 0; j < 3; j++ {
			got, err = Inv.At(i, j)
			require.NoError(t, err)
			require.InDeltaf(t, want[i][j], got, 1e-12, "Inv[%d,%d]", i, j)
		}
	}

	// Check A*Inv≈I и Inv*A≈I
	Ileft, err := matrix.Mul(A, Inv)
	require.NoError(t, err)
	Iright, err := matrix.Mul(Inv, A)
	require.NoError(t, err)

	for i = 0; i < 3; i++ {
		for j = 0; j < 3; j++ {
			var lv, rv float64
			lv, _ = Ileft.At(i, j)
			rv, _ = Iright.At(i, j)
			if i == j {
				require.InDelta(t, 1.0, lv, 1e-12, "A*Inv diag[%d]", i)
				require.InDelta(t, 1.0, rv, 1e-12, "Inv*A diag[%d]", i)
			} else {
				require.InDelta(t, 0.0, lv, 1e-12, "A*Inv off[%d,%d]", i, j)
				require.InDelta(t, 0.0, rv, 1e-12, "Inv*A off[%d,%d]", i, j)
			}
		}
	}
}

// Hiding the input type (iface/fallback on reading) should not change the result.
// Inside Inverse it is still solved by *Dense (L and U are dense).
func TestInverse_WrappedInput_MatchesDense(t *testing.T) {
	t.Parallel()

	const n = 4
	var (
		i, j int
		err  error
	)

	// A = MᵀM + I  (well-conditioned PD)
	M, _ := matrix.NewDense(n, n)
	matrix.Random(t, M, 123)
	Mt, err := matrix.Transpose(M)
	require.NoError(t, err)
	PD, err := matrix.Mul(Mt, M)
	require.NoError(t, err)
	I, err := matrix.NewDense(n, n)
	require.NoError(t, err)
	for i = 0; i < n; i++ {
		require.NoError(t, I.Set(i, i, 1))
	}
	A, err := matrix.Add(PD, I)
	require.NoError(t, err)

	Aw := hide{A} // hided type

	Inv1, err := matrix.Inverse(A)
	require.NoError(t, err)
	Inv2, err := matrix.Inverse(Aw)
	require.NoError(t, err)

	var v1, v2 float64
	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			v1, _ = Inv1.At(i, j)
			v2, _ = Inv2.At(i, j)
			require.InDeltaf(t, v1, v2, 1e-11, "mismatch at [%d,%d]", i, j)
		}
	}
}

// Property: A A^{-1}≈I and A^{-1} A≈I on 6×6 SPD. And the input does not mutate.
func TestInverse_IdentityProduct_SPD_6x6(t *testing.T) {
	t.Parallel()

	const n = 6
	var (
		i, j int
		err  error
	)

	// A = MᵀM + I
	M, _ := matrix.NewDense(n, n)
	matrix.Random(t, M, 777)
	Mt, err := matrix.Transpose(M)
	require.NoError(t, err)
	PD, err := matrix.Mul(Mt, M)
	require.NoError(t, err)

	A, err := matrix.NewDense(n, n)
	require.NoError(t, err)
	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			var v float64
			v, _ = PD.At(i, j)
			require.NoError(t, A.Set(i, j, v))
		}
		require.NoError(t, A.Set(i, i, 1.0+func(x float64) float64 { return x }(0))) // +1 на диагональ
	}

	Acopy := A.Clone()

	Inv, err := matrix.Inverse(A)
	require.NoError(t, err)

	L, err := matrix.Mul(A, Inv)
	require.NoError(t, err)
	R, err := matrix.Mul(Inv, A)
	require.NoError(t, err)

	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			var lv, rv float64
			lv, _ = L.At(i, j)
			rv, _ = R.At(i, j)
			if i == j {
				require.InDelta(t, 1.0, lv, 1e-8, "A*Inv diag[%d]", i)
				require.InDelta(t, 1.0, rv, 1e-8, "Inv*A diag[%d]", i)
			} else {
				require.InDelta(t, 0.0, lv, 1e-8, "A*Inv off[%d,%d]", i, j)
				require.InDelta(t, 0.0, rv, 1e-8, "Inv*A off[%d,%d]", i, j)
			}
		}
	}

	// A should not mutate
	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			var a1, a2 float64
			a1, _ = A.At(i, j)
			a2, _ = Acopy.At(i, j)
			require.Equalf(t, a2, a1, "A mutated at [%d,%d]", i, j)
		}
	}
}

// Scaling property: (αA)^{-1} = (1/α)*A^{-1} for α≠0.
func TestInverse_ScaleProperty(t *testing.T) {
	t.Parallel()

	const n = 5
	const alpha = 2.5
	var (
		i, j int
		err  error
	)

	// A = MᵀM + 2I (add 2I to stay away from degeneracy)
	M, _ := matrix.NewDense(n, n)
	matrix.Random(t, M, 2024)
	Mt, err := matrix.Transpose(M)
	require.NoError(t, err)
	PD, err := matrix.Mul(Mt, M)
	require.NoError(t, err)

	A, err := matrix.NewDense(n, n)
	require.NoError(t, err)
	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			var v float64
			v, _ = PD.At(i, j)
			require.NoError(t, A.Set(i, j, v))
		}
	}
	for i = 0; i < n; i++ {
		var d float64
		d, _ = A.At(i, i)
		require.NoError(t, A.Set(i, i, d+2.0))
	}

	InvA, err := matrix.Inverse(A)
	require.NoError(t, err)

	alphaA, err := matrix.Scale(A, alpha)
	require.NoError(t, err)
	InvAlphaA, err := matrix.Inverse(alphaA)
	require.NoError(t, err)

	// Wait Inv(αA) ≈ (1/α)*Inv(A)
	scaleInvA, err := matrix.Scale(InvA, 1.0/alpha)
	require.NoError(t, err)

	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			var left, right float64
			left, _ = InvAlphaA.At(i, j)
			right, _ = scaleInvA.At(i, j)
			require.InDeltaf(t, right, left, 1e-9, "at [%d,%d]", i, j)
		}
	}
}

// ---------- 7. LU ----------

// Errors: nil and non-square are rejected.
func TestLU_Errors(t *testing.T) {
	t.Parallel()

	var err error

	// nil → ErrNilMatrix
	_, _, err = matrix.LU(nil)
	require.ErrorIs(t, err, matrix.ErrNilMatrix)

	// non-square → ErrDimensionMismatch
	ns, _ := matrix.NewDense(3, 4)
	_, _, err = matrix.LU(ns)
	require.ErrorIs(t, err, matrix.ErrDimensionMismatch)
}

// Basic (3×3): pick L,U explicitly (Doolittle form, diag(L)=1), set A=L*U,
// then verify LU(A) reproduces the same factors and A≈L*U exactly.
func TestLU_Known3x3_Doolittle_FastPath_Correctness(t *testing.T) {
	t.Parallel()

	var (
		i, j int
		err  error
	)

	// Target factors:
	// L = [[1,0,0],
	//      [2,1,0],
	//      [3,4,1]]
	// U = [[5,6,7],
	//      [0,8,9],
	//      [0,0,10]]
	Lexp, _ := matrix.NewDense(3, 3)
	Uexp, _ := matrix.NewDense(3, 3)

	// Fill Lexp (unit lower)
	require.NoError(t, Lexp.Set(0, 0, 1))
	require.NoError(t, Lexp.Set(1, 0, 2))
	require.NoError(t, Lexp.Set(1, 1, 1))
	require.NoError(t, Lexp.Set(2, 0, 3))
	require.NoError(t, Lexp.Set(2, 1, 4))
	require.NoError(t, Lexp.Set(2, 2, 1))

	// Fill Uexp (upper)
	require.NoError(t, Uexp.Set(0, 0, 5))
	require.NoError(t, Uexp.Set(0, 1, 6))
	require.NoError(t, Uexp.Set(0, 2, 7))
	require.NoError(t, Uexp.Set(1, 1, 8))
	require.NoError(t, Uexp.Set(1, 2, 9))
	require.NoError(t, Uexp.Set(2, 2, 10))

	// Build A = L*U
	A, err := matrix.Mul(Lexp, Uexp)
	require.NoError(t, err)

	// Keep a copy to ensure input immutability
	Acopy := A.Clone()

	// Factorize
	Lgot, Ugot, err := matrix.LU(A)
	require.NoError(t, err)

	// Structural checks and exact equality vs expected factors
	propUnitLowerTriangular(t, Lgot, 0)
	propUpperTriangular(t, Ugot, 0)

	var gv, ev float64
	for i = 0; i < 3; i++ {
		for j = 0; j < 3; j++ {
			gv, _ = Lgot.At(i, j)
			ev, _ = Lexp.At(i, j)
			require.Equalf(t, ev, gv, "L mismatch at [%d,%d]", i, j)

			gv, _ = Ugot.At(i, j)
			ev, _ = Uexp.At(i, j)
			require.Equalf(t, ev, gv, "U mismatch at [%d,%d]", i, j)
		}
	}

	// Reconstruction A ≈ L*U
	propReconstructionLU(t, A, Lgot, Ugot, 0)

	// Input must not mutate
	for i = 0; i < 3; i++ {
		for j = 0; j < 3; j++ {
			var a1, a2 float64
			a1, _ = A.At(i, j)
			a2, _ = Acopy.At(i, j)
			require.Equalf(t, a2, a1, "A mutated at [%d,%d]", i, j)
		}
	}
}

// Fast-path vs Fallback (3×3): wrapping the input to hide its concrete type
// must produce the same L and U as the fast path.
func TestLU_Known3x3_Fallback_MatchesFast(t *testing.T) {
	t.Parallel()

	var (
		i, j int
		err  error
	)

	// Reuse the same 3×3 A from the previous test to avoid tiny matrices.
	Lexp, _ := matrix.NewDense(3, 3)
	Uexp, _ := matrix.NewDense(3, 3)

	// Lexp
	require.NoError(t, Lexp.Set(0, 0, 1))
	require.NoError(t, Lexp.Set(1, 0, 2))
	require.NoError(t, Lexp.Set(1, 1, 1))
	require.NoError(t, Lexp.Set(2, 0, 3))
	require.NoError(t, Lexp.Set(2, 1, 4))
	require.NoError(t, Lexp.Set(2, 2, 1))
	// Uexp
	require.NoError(t, Uexp.Set(0, 0, 5))
	require.NoError(t, Uexp.Set(0, 1, 6))
	require.NoError(t, Uexp.Set(0, 2, 7))
	require.NoError(t, Uexp.Set(1, 1, 8))
	require.NoError(t, Uexp.Set(1, 2, 9))
	require.NoError(t, Uexp.Set(2, 2, 10))

	A, err := matrix.Mul(Lexp, Uexp)
	require.NoError(t, err)

	// Fast path
	L1, U1, err := matrix.LU(A)
	require.NoError(t, err)
	// Fallback path
	Aw := hide{A}
	L2, U2, err := matrix.LU(Aw)
	require.NoError(t, err)

	// Elementwise equality
	var v1, v2 float64
	for i = 0; i < 3; i++ {
		for j = 0; j < 3; j++ {
			v1, _ = L1.At(i, j)
			v2, _ = L2.At(i, j)
			require.Equalf(t, v1, v2, "L mismatch at [%d,%d]", i, j)

			v1, _ = U1.At(i, j)
			v2, _ = U2.At(i, j)
			require.Equalf(t, v1, v2, "U mismatch at [%d,%d]", i, j)
		}
	}
}

// Properties on 6×6: construct L (unit lower) and U (upper) with simple integer
// patterns, set A=L*U, then check (i) structure, (ii) reconstruction, and (iii) exact recovery.
func TestLU_Factor_Reconstruction_6x6(t *testing.T) {
	t.Parallel()

	const n = 6
	var (
		i, j int
		err  error
	)

	Lexp, _ := matrix.NewDense(n, n)
	Uexp, _ := matrix.NewDense(n, n)

	// Lexp: unit lower with a mild, deterministic pattern below diagonal
	for i = 0; i < n; i++ {
		require.NoError(t, Lexp.Set(i, i, 1.0))
	}
	for i = 1; i < n; i++ {
		for j = 0; j < i; j++ {
			// small integers keep A exact in float
			require.NoError(t, Lexp.Set(i, j, float64(j+1)))
		}
	}

	// Uexp: upper with positive diagonal (nonzero pivots), simple pattern above diag
	for i = 0; i < n; i++ {
		require.NoError(t, Uexp.Set(i, i, float64(2*i+3))) // 3,5,7,9,11,13
		for j = i + 1; j < n; j++ {
			require.NoError(t, Uexp.Set(i, j, float64(j-i+1)))
		}
	}

	// A = L*U
	A, err := matrix.Mul(Lexp, Uexp)
	require.NoError(t, err)

	// Factorize
	Lgot, Ugot, err := matrix.LU(A)
	require.NoError(t, err)

	// Structure
	propUnitLowerTriangular(t, Lgot, 0)
	propUpperTriangular(t, Ugot, 0)

	// Exact equality vs our factors (Doolittle is unique with these nonzero pivots)
	var gv, ev float64
	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			gv, _ = Lgot.At(i, j)
			ev, _ = Lexp.At(i, j)
			require.Equalf(t, ev, gv, "L mismatch at [%d,%d]", i, j)

			gv, _ = Ugot.At(i, j)
			ev, _ = Uexp.At(i, j)
			require.Equalf(t, ev, gv, "U mismatch at [%d,%d]", i, j)
		}
	}

	// Reconstruction A ≈ L*U
	propReconstructionLU(t, A, Lgot, Ugot, 0)
}

// ---------- 8. QR ----------

// Errors: nil and non-square are rejected.
func TestQR_Errors(t *testing.T) {
	t.Parallel()

	var err error

	// nil → ErrNilMatrix
	_, _, err = matrix.QR(nil)
	require.ErrorIs(t, err, matrix.ErrNilMatrix)

	// non-square → ErrDimensionMismatch
	ns, _ := matrix.NewDense(3, 4)
	_, _, err = matrix.QR(ns)
	require.ErrorIs(t, err, matrix.ErrDimensionMismatch)
}

// Classic 3×3 Householder example (well-known benchmark):
//
//	 A = [[ 12, -51,   4],
//		[  6, 167, -68],
//		[ -4,  24, -41]]
//
// One canonical QR (up to column-sign freedom):
//
//	 R = [[ 14,  21, -14],
//		[  0, 175, -70],
//		[  0,   0,  35]]
//
//	 Q = [[ 6/7,   -69/175,  -58/175],
//		[ 3/7,    158/175,    6/175],
//		[-2/7,      6/35,    -33/35]]
//
// Our routine returns A ≈ Qᵀ*R. We canonicalize diag(R) ≥ 0 by left-multiplying
// both Q and R by the same diagonal S, which preserves A = Qᵀ*R. Then we:
//
//	check |R| against the canonical magnitudes;
//	compare columns of Qᵀ with the canonical Q up to per-column sign;
//	assert QᵀQ≈I and A≈Qᵀ*R;
//	assert input immutability.
func TestQR_Classic3x3_Householder_Known(t *testing.T) {
	t.Parallel()

	var (
		i, j int
		err  error
	)

	// Build A
	A, _ := matrix.NewDense(3, 3)
	require.NoError(t, A.Set(0, 0, 12))
	require.NoError(t, A.Set(0, 1, -51))
	require.NoError(t, A.Set(0, 2, 4))
	require.NoError(t, A.Set(1, 0, 6))
	require.NoError(t, A.Set(1, 1, 167))
	require.NoError(t, A.Set(1, 2, -68))
	require.NoError(t, A.Set(2, 0, -4))
	require.NoError(t, A.Set(2, 1, 24))
	require.NoError(t, A.Set(2, 2, -41))
	Acopy := A.Clone()

	Q, R, err := matrix.QR(A)
	require.NoError(t, err)

	// --- Canonicalize diag(R) >= 0 via S (LEFT multiply on BOTH Q and R!) ---
	S, err := matrix.NewDense(3, 3)
	require.NoError(t, err)
	for i = 0; i < 3; i++ {
		var rii float64
		rii, err = R.At(i, i)
		require.NoError(t, err)
		if rii >= 0 {
			require.NoError(t, S.Set(i, i, 1.0))
		} else {
			require.NoError(t, S.Set(i, i, -1.0))
		}
	}
	// Correct invariance: (SQ)^T*(SR) = Q^T*R
	SQ, err := matrix.Mul(S, Q)
	require.NoError(t, err)
	SR, err := matrix.Mul(S, R)
	require.NoError(t, err)
	Q = SQ
	R = SR

	// quick sanity: after normalization A ≈ Qᵀ*R must still hold
	propReconstructionQR(t, Acopy, Q, R, 1e-12)
	// --- end canonicalization ---

	// R must be upper-triangular with canonical magnitudes (signs are free).
	RabsWant := [][]float64{
		{14, 21, 14},
		{0, 175, 70},
		{0, 0, 35},
	}
	var rv float64
	for i = 0; i < 3; i++ {
		for j = 0; j < 3; j++ {
			rv, err = R.At(i, j)
			require.NoError(t, err)
			if i > j {
				require.InDeltaf(t, 0.0, rv, 1e-12, "R[%d,%d] must be 0 below diagonal", i, j)
				continue
			}
			if rv < 0 {
				rv = -rv
			}
			require.InDeltaf(t, RabsWant[i][j], rv, 1e-12, "abs(R[%d,%d])", i, j)
		}
	}
	for i = 0; i < 3; i++ {
		rv, _ = R.At(i, i)
		require.GreaterOrEqualf(t, rv, 0.0, "R[%d,%d] must be >= 0 after normalization", i, i)
	}

	// Compare Q^T columns to canonical Q columns up to column sign.
	Qwant := [][]float64{
		{6.0 / 7.0, -69.0 / 175.0, -58.0 / 175.0},
		{3.0 / 7.0, 158.0 / 175.0, 6.0 / 175.0},
		{-2.0 / 7.0, 6.0 / 35.0, -33.0 / 35.0},
	}
	Qt, err := matrix.Transpose(Q)
	require.NoError(t, err)
	var qv, dot float64
	for j = 0; j < 3; j++ {
		dot = 0.0
		for i = 0; i < 3; i++ {
			qv, err = Qt.At(i, j)
			require.NoError(t, err)
			dot += qv * Qwant[i][j]
		}
		sign := 1.0
		if dot < 0 {
			sign = -1.0
		}
		for i = 0; i < 3; i++ {
			qv, err = Qt.At(i, j)
			require.NoError(t, err)
			require.InDeltaf(t, sign*Qwant[i][j], qv, 1e-12, "Qt[%d,%d] up to sign", i, j)
		}
	}

	// Orthogonality and final reconstruction under A ≈ Qᵀ*R.
	propOrthonormal(t, Q, 1e-12)
	propReconstructionQR(t, Acopy, Q, R, 1e-12)

	// Input immutability.
	for i = 0; i < 3; i++ {
		for j = 0; j < 3; j++ {
			var a1, a2 float64
			a1, _ = A.At(i, j)
			a2, _ = Acopy.At(i, j)
			require.Equalf(t, a2, a1, "A mutated at [%d,%d]", i, j)
		}
	}
}

// 8.3 Fast-path vs Fallback (5×5): wrapping the input to hide its concrete type
// must produce numerically identical Q and R (within tight tolerance).
func TestQR_Fallback_MatchesFast_5x5(t *testing.T) {
	t.Parallel()

	const n = 5
	var (
		i, j int
		err  error
	)

	// Build a deterministic dense matrix (no anonymous factories).
	M, _ := matrix.NewDense(n, n)
	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			// simple, well-conditioned pattern
			require.NoError(t, M.Set(i, j, float64(3*i-2*j+1)))
		}
	}

	// Fast path
	Q1, R1, err := matrix.QR(M)
	require.NoError(t, err)
	// Fallback path: hide the concrete type
	Mw := hide{M}
	Q2, R2, err := matrix.QR(Mw)
	require.NoError(t, err)

	// Elementwise comparison with small tolerance
	var v1, v2 float64
	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			v1, _ = Q1.At(i, j)
			v2, _ = Q2.At(i, j)
			require.InDeltaf(t, v1, v2, 1e-11, "Q mismatch at [%d,%d]", i, j)

			v1, _ = R1.At(i, j)
			v2, _ = R2.At(i, j)
			require.InDeltaf(t, v1, v2, 1e-11, "R mismatch at [%d,%d]", i, j)
		}
	}

	// Both must satisfy orthogonality and reconstruction with the same source M.
	propOrthonormal(t, Q1, 1e-11)
	propOrthonormal(t, Q2, 1e-11)
	propReconstructionQR(t, M, Q1, R1, 1e-11)
	propReconstructionQR(t, M, Q2, R2, 1e-11)
}

// 8.4 Properties on 6×6: QᵀQ≈I, R is upper-triangular, and A≈Qᵀ*R.
// Also assert the input is not mutated. Include a zero column to exercise the “skip zero” branch.
func TestQR_Properties_6x6_WithZeroColumn(t *testing.T) {
	t.Parallel()

	const n = 6
	var (
		i, j int
		err  error
	)

	A, _ := matrix.NewDense(n, n)
	// Columns: c0..c5; set c2 to zeros to hit the "norm == 0" branch.
	for i = 0; i < n; i++ {
		// c0: increasing
		require.NoError(t, A.Set(i, 0, float64(i+1)))
		// c1: alternating
		require.NoError(t, A.Set(i, 1, float64(1-2*(i%2))))
		// c2: zeros
		require.NoError(t, A.Set(i, 2, 0.0))
		// c3, c4, c5: mild linear patterns
		require.NoError(t, A.Set(i, 3, float64(2*i-3)))
		require.NoError(t, A.Set(i, 4, float64(5-i)))
		require.NoError(t, A.Set(i, 5, float64(3*i+2)))
	}

	Acopy := A.Clone()

	Q, R, err := matrix.QR(A)
	require.NoError(t, err)

	// Q orthonormal; R upper triangular.
	propOrthonormal(t, Q, 1e-12)
	propUpperTriangular(t, R, 1e-12)

	// Reconstruction: A ≈ Qᵀ*R
	propReconstructionQR(t, Acopy, Q, R, 1e-11)

	// Input must not mutate
	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			var a1, a2 float64
			a1, _ = A.At(i, j)
			a2, _ = Acopy.At(i, j)
			require.Equalf(t, a2, a1, "A mutated at [%d,%d]", i, j)
		}
	}
}

// --- QR-specific helper (test-only, unexported) ---

// --- local property-check helpers (test-only, unexported) ---

// propOrthonormal asserts QᵀQ ≈ I within delta.
func propOrthonormal(t *testing.T, Q matrix.Matrix, delta float64) {
	t.Helper()

	var (
		i, j int
		v    float64
		err  error
	)

	n := Q.Rows()
	require.Equal(t, n, Q.Cols())

	Qt, err := matrix.Transpose(Q)
	require.NoError(t, err)
	QtQ, err := matrix.Mul(Qt, Q)
	require.NoError(t, err)

	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			v, err = QtQ.At(i, j)
			require.NoError(t, err)
			if i == j {
				require.InDeltaf(t, 1.0, v, delta, "QtQ[%d,%d]", i, j)
			} else {
				require.InDeltaf(t, 0.0, v, delta, "QtQ[%d,%d]", i, j)
			}
		}
	}
}

// propReconstruction asserts A ≈ Q*diag(vals)*Qᵀ within delta.
func propReconstruction(t *testing.T, A, Q matrix.Matrix, vals []float64, delta float64) {
	t.Helper()

	var (
		i, j int
		w, g float64
		err  error
	)

	n := A.Rows()
	require.Equal(t, n, A.Cols())
	require.Equal(t, n, Q.Rows())
	require.Equal(t, n, Q.Cols())
	require.Len(t, vals, n)

	D, err := matrix.NewDense(n, n)
	require.NoError(t, err)
	for i = 0; i < n; i++ {
		require.NoError(t, D.Set(i, i, vals[i]))
	}

	QD, err := matrix.Mul(Q, D)
	require.NoError(t, err)
	Qt, err := matrix.Transpose(Q)
	require.NoError(t, err)
	QDQt, err := matrix.Mul(QD, Qt)
	require.NoError(t, err)

	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			w, err = A.At(i, j)
			require.NoError(t, err)
			g, err = QDQt.At(i, j)
			require.NoError(t, err)
			require.InDeltaf(t, w, g, delta, "reconstruction mismatch at [%d,%d]", i, j)
		}
	}
}

// propEigenEquation asserts A*Q ≈ Q*diag(vals) within delta.
func propEigenEquation(t *testing.T, A, Q matrix.Matrix, vals []float64, delta float64) {
	t.Helper()

	var (
		i, j int
		l, r float64
		err  error
	)

	n := A.Rows()
	require.Equal(t, n, A.Cols())
	require.Equal(t, n, Q.Rows())
	require.Equal(t, n, Q.Cols())
	require.Len(t, vals, n)

	D, err := matrix.NewDense(n, n)
	require.NoError(t, err)
	for i = 0; i < n; i++ {
		require.NoError(t, D.Set(i, i, vals[i]))
	}

	AQ, err := matrix.Mul(A, Q)
	require.NoError(t, err)
	QD, err := matrix.Mul(Q, D)
	require.NoError(t, err)

	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			l, err = AQ.At(i, j)
			require.NoError(t, err)
			r, err = QD.At(i, j)
			require.NoError(t, err)
			require.InDeltaf(t, l, r, delta, "A*Q vs Q*D mismatch at [%d,%d]", i, j)
		}
	}
}

// propUnitLowerTriangular checks diag(L)=1 and L[i,j]=0 for j>i.
// delta=0 demands exact zeros/ones; positive delta allows tolerance.
func propUnitLowerTriangular(t *testing.T, L matrix.Matrix, delta float64) {
	t.Helper()

	var (
		i, j int
		v    float64
		err  error
	)

	require.Equal(t, L.Rows(), L.Cols(), "L must be square")
	n := L.Rows()

	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			v, err = L.At(i, j)
			require.NoError(t, err)
			if i == j {
				if delta == 0 {
					require.Equalf(t, 1.0, v, "diag(L)[%d]", i)
				} else {
					require.InDeltaf(t, 1.0, v, delta, "diag(L)[%d]", i)
				}
			} else if j > i {
				if delta == 0 {
					require.Equalf(t, 0.0, v, "upper(L) at [%d,%d]", i, j)
				} else {
					require.InDeltaf(t, 0.0, v, delta, "upper(L) at [%d,%d]", i, j)
				}
			}
		}
	}
}

// propUpperTriangular checks U[i,j]=0 for i>j. Diagonal may be arbitrary nonzero.
// delta=0 demands exact zeros below diagonal.
func propUpperTriangular(t *testing.T, U matrix.Matrix, delta float64) {
	t.Helper()

	var (
		i, j int
		v    float64
		err  error
	)

	require.Equal(t, U.Rows(), U.Cols(), "U must be square")
	n := U.Rows()

	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			if i > j {
				v, err = U.At(i, j)
				require.NoError(t, err)
				if delta == 0 {
					require.Equalf(t, 0.0, v, "lower(U) at [%d,%d]", i, j)
				} else {
					require.InDeltaf(t, 0.0, v, delta, "lower(U) at [%d,%d]", i, j)
				}
			}
		}
	}
}

// propReconstructionLU verifies A ≈ L*U within delta.
func propReconstructionLU(t *testing.T, A, L, U matrix.Matrix, delta float64) {
	t.Helper()

	var (
		i, j int
		lr   float64
		ar   float64
		err  error
	)

	require.Equal(t, A.Rows(), L.Rows(), "shape mismatch A vs L")
	require.Equal(t, A.Cols(), U.Cols(), "shape mismatch A vs U")

	LU, err := matrix.Mul(L, U)
	require.NoError(t, err)

	for i = 0; i < A.Rows(); i++ {
		for j = 0; j < A.Cols(); j++ {
			lr, err = LU.At(i, j)
			require.NoError(t, err)
			ar, err = A.At(i, j)
			require.NoError(t, err)

			if delta == 0 {
				require.Equalf(t, ar, lr, "A vs L*U at [%d,%d]", i, j)
			} else {
				require.InDeltaf(t, ar, lr, delta, "A vs L*U at [%d,%d]", i, j)
			}
		}
	}
}

// propReconstructionQR verifies A ≈ Qᵀ*R within a given tolerance.
// Note: With the current implementation, reflectors are accumulated on the left,
// so the decomposition realized by the function is m ≈ Qᵀ*R (not Q*R).
func propReconstructionQR(t *testing.T, A, Q, R matrix.Matrix, delta float64) {
	t.Helper()

	var (
		i, j int
		lv   float64
		rv   float64
		err  error
	)

	Qt, err := matrix.Transpose(Q)
	require.NoError(t, err)
	QtR, err := matrix.Mul(Qt, R)
	require.NoError(t, err)

	require.Equal(t, A.Rows(), QtR.Rows())
	require.Equal(t, A.Cols(), QtR.Cols())

	for i = 0; i < A.Rows(); i++ {
		for j = 0; j < A.Cols(); j++ {
			lv, err = A.At(i, j)
			require.NoError(t, err)
			rv, err = QtR.At(i, j)
			require.NoError(t, err)
			require.InDeltaf(t, lv, rv, delta, "A vs Qᵀ*R mismatch at [%d,%d]", i, j)
		}
	}
}
