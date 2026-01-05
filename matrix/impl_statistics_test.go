// SPDX-License-Identifier: MIT

package matrix_test

import (
	"errors"
	"math"
	"testing"

	"github.com/katalvlaran/lvlath/matrix"
)

const epsTight = 1e-12

// ------------------------------
// CenterColumns / CenterRows
// ------------------------------

func TestCenterColumns_SmallAndFallback(t *testing.T) {
	t.Parallel()

	X := NewFilledDense(t, 2, 3, []float64{1, 2, 3, 10, 20, 30})
	Xh := hide{X}

	Yf, meansF, err := matrix.CenterColumns(X)
	if err != nil {
		t.Fatalf("fast: %v", err)
	}
	Ys, meansS, err := matrix.CenterColumns(Xh)
	if err != nil {
		t.Fatalf("slow: %v", err)
	}

	// Means should be [5.5, 11, 16.5].
	want := []float64{5.5, 11, 16.5}
	sliceClose(t, meansF, want, 0, 0)
	sliceClose(t, meansS, want, 0, 0)

	CompareClose(t, Yf, Ys, 0, 0)

	// Column averages of Y ≈ 0.
	var i, j int
	var sum, avg float64
	for j = 0; j < 3; j++ {
		sum = 0.0
		for i = 0; i < 2; i++ {
			sum += MustAt(t, Yf, i, j)
		}
		avg = sum / 2
		if math.Abs(avg) > epsTight {
			t.Fatalf("col %d not centered: avg=%g", j, avg)
		}
	}
}

func TestCenterRows_SmallAndFallback(t *testing.T) {
	t.Parallel()

	X := NewFilledDense(t, 2, 3, []float64{1, 2, 3, 10, 20, 30})
	Xh := hide{X}

	Yf, meansF, err := matrix.CenterRows(X)
	if err != nil {
		t.Fatalf("fast: %v", err)
	}
	Ys, meansS, err := matrix.CenterRows(Xh)
	if err != nil {
		t.Fatalf("slow: %v", err)
	}
	// Row means: [2, 20].
	sliceClose(t, meansF, []float64{2, 20}, 0, 0)
	sliceClose(t, meansS, []float64{2, 20}, 0, 0)

	CompareClose(t, Yf, Ys, 0, 0)

	// Row averages ≈ 0.
	var i, j int
	var sum, avg float64
	for i = 0; i < 2; i++ {
		sum = 0.0
		for j = 0; j < 3; j++ {
			sum += MustAt(t, Yf, i, j)
		}
		avg = sum / 3
		if math.Abs(avg) > epsTight {
			t.Fatalf("row %d not centered: avg=%g", i, avg)
		}
	}
}

func TestCenterColumns_ZeroSizeSafe(t *testing.T) {
	t.Parallel()

	X1, _ := matrix.NewDense(0, 3)
	Y1, m1, err := matrix.CenterColumns(X1)
	if err != nil {
		t.Fatalf("0x3: %v", err)
	}
	if Y1.Rows() != 0 || Y1.Cols() != 3 || len(m1) != 3 {
		t.Fatalf("shape mismatch 0x3")
	}

	X2, _ := matrix.NewDense(2, 0)
	Y2, m2, err := matrix.CenterColumns(X2)
	if err != nil {
		t.Fatalf("2x0: %v", err)
	}
	if Y2.Rows() != 2 || Y2.Cols() != 0 || len(m2) != 0 {
		t.Fatalf("shape mismatch 2x0")
	}
}

func TestCenterRows_ZeroSizeSafe(t *testing.T) {
	t.Parallel()

	X1, _ := matrix.NewDense(0, 3)
	Y1, m1, err := matrix.CenterRows(X1)
	if err != nil {
		t.Fatalf("0x3: %v", err)
	}
	if Y1.Rows() != 0 || Y1.Cols() != 3 || len(m1) != 0 {
		t.Fatalf("shape mismatch 0x3 (row means must be len=rows=0)")
	}

	X2, _ := matrix.NewDense(2, 0)
	Y2, m2, err := matrix.CenterRows(X2)
	if err != nil {
		t.Fatalf("2x0: %v", err)
	}
	if Y2.Rows() != 2 || Y2.Cols() != 0 || len(m2) != 2 {
		t.Fatalf("shape mismatch 2x0 (row means must be len=rows=2)")
	}
}

// ------------------------------
// NormalizeRowsL1 / NormalizeRowsL2
// ------------------------------

func TestNormalizeRowsL1_Basics(t *testing.T) {
	t.Parallel()

	// Row 0 is truly degenerate (all zeros).
	X := NewFilledDense(t, 3, 3, []float64{
		0, 0, 0, // zero row
		1, -1, 3,
		-2, 0, 2,
	})

	Y, norms, err := matrix.NormalizeRowsL1(X)
	if err != nil {
		t.Fatalf("NormalizeRowsL1: %v", err)
	}

	// Row 0 stays zero; rows 1.. have L1≈1.
	var j int
	for j = 0; j < 3; j++ {
		if MustAt(t, Y, 0, j) != 0 {
			t.Fatalf("row0 changed")
		}
	}
	if norms[0] != 0 {
		t.Fatalf("row0 norm must be 0")
	}
	// Check ||row||_1 == 1 (within eps).
	var i int
	var s, v float64
	for i = 1; i < 3; i++ {
		s = 0.0
		for j = 0; j < 3; j++ {
			v = MustAt(t, Y, i, j)
			if v < 0 {
				v = -v
			}
			s += v
		}
		if math.Abs(s-1) > epsTight {
			t.Fatalf("row %d L1=%g, want 1", i, s)
		}
	}
}

func TestNormalizeRowsL2_Basics(t *testing.T) {
	t.Parallel()

	// Row 0 is truly degenerate (all zeros).
	X := NewFilledDense(t, 2, 3, []float64{
		0, 0, 0,
		3, 4, 0, // norm 5
	})

	Y, norms, err := matrix.NormalizeRowsL2(X)
	if err != nil {
		t.Fatalf("NormalizeRowsL2: %v", err)
	}

	// Row 0 remains zeros and its norm is 0.
	var j int
	for j = 0; j < 3; j++ {
		if MustAt(t, Y, 0, j) != 0 {
			t.Fatalf("row0 changed")
		}
	}
	if norms[0] != 0 {
		t.Fatalf("row0 norm want 0, got %g", norms[0])
	}

	// Row 1 becomes unit L2.
	sq := 0.0
	for j = 0; j < 3; j++ {
		v := MustAt(t, Y, 1, j)
		sq += v * v
	}
	if math.Abs(math.Sqrt(sq)-1) > epsTight {
		t.Fatalf("row1 L2 not ~1")
	}
}

func TestNormalizeRows_ZeroSizeSafe(t *testing.T) {
	t.Parallel()

	X1, _ := matrix.NewDense(0, 3)
	Y1, n1, err := matrix.NormalizeRowsL1(X1)
	if err != nil {
		t.Fatalf("L1 0x3: %v", err)
	}
	if Y1.Rows() != 0 || Y1.Cols() != 3 || len(n1) != 0 {
		t.Fatalf("L1 0x3 shape mismatch")
	}

	X2, _ := matrix.NewDense(2, 0)
	Y2, n2, err := matrix.NormalizeRowsL2(X2)
	if err != nil {
		t.Fatalf("L2 2x0: %v", err)
	}
	if Y2.Rows() != 2 || Y2.Cols() != 0 || len(n2) != 2 {
		t.Fatalf("L2 2x0 shape mismatch")
	}
}

// ------------------------------
// Covariance
// ------------------------------

func TestCovariance_Symmetric_DiagMatchesVariance(t *testing.T) {
	t.Parallel()

	X := NewFilledDense(t, 4, 3, []float64{
		1, 2, 3,
		2, 3, 4,
		3, 5, 7,
		-1, 0, 1,
	})

	Cov, means, err := matrix.Covariance(X)
	if err != nil {
		t.Fatalf("Covariance: %v", err)
	}
	// sanity: means length
	if len(means) != 3 {
		t.Fatalf("means len=%d want 3", len(means))
	}

	// symmetry
	var j, k int
	var ajk, akj float64
	for j = 0; j < 3; j++ {
		for k = 0; k < 3; k++ {
			ajk = MustAt(t, Cov, j, k)
			akj = MustAt(t, Cov, k, j)
			if ajk != akj {
				t.Fatalf("not symmetric at (%d,%d)", j, k)
			}
		}
	}

	// diagonal equals sample variance of centered columns
	r := 4
	// manual var for each column
	var i int
	var sum, xij, d, got float64
	for j = 0; j < 3; j++ {
		sum = 0.0
		for i = 0; i < r; i++ {
			xij = MustAt(t, X, i, j)
			d = xij - means[j]
			sum += d * d
		}
		wantVar := sum / float64(r-1)
		got = MustAt(t, Cov, j, j)
		if math.Abs(got-wantVar) > epsTight {
			t.Fatalf("var[%d]: got=%g want=%g", j, got, wantVar)
		}
	}
}

func TestCovariance_RowsLessThan2_Error(t *testing.T) {
	t.Parallel()

	X, _ := matrix.NewDense(1, 3)
	_, _, err := matrix.Covariance(X)
	if !errors.Is(err, matrix.ErrDimensionMismatch) {
		t.Fatalf("want ErrDimensionMismatch, got %v", err)
	}
}

func TestCovariance_FallbackMatchesFast(t *testing.T) {
	t.Parallel()

	X := RandFilledDense(t, 7, 5, 42)
	Cf, _, err := matrix.Covariance(X)
	if err != nil {
		t.Fatalf("fast: %v", err)
	}
	Cs, _, err := matrix.Covariance(hide{X})
	if err != nil {
		t.Fatalf("slow: %v", err)
	}
	CompareClose(t, Cf, Cs, epsTight, epsTight)
}

// ------------------------------
// Correlation
// ------------------------------

func TestCorrelation_Basics_DiagAndSymmetry(t *testing.T) {
	t.Parallel()

	// Two non-degenerate columns, one degenerate (constant).
	X := NewFilledDense(t, 5, 3, []float64{
		1, 2, 7,
		2, 3, 7,
		3, 4, 7,
		4, 5, 7,
		5, 6, 7,
	})

	Corr, means, stds, err := matrix.Correlation(X)
	if err != nil {
		t.Fatalf("Correlation: %v", err)
	}
	if len(means) != 3 || len(stds) != 3 {
		t.Fatalf("means/stds len mismatch")
	}

	// Symmetry and expected diag: 1,1,0 (third column degenerate)
	var j, k int
	for j = 0; j < 3; j++ {
		for k = 0; k < 3; k++ {
			if MustAt(t, Corr, j, k) != MustAt(t, Corr, k, j) {
				t.Fatalf("not symmetric (%d,%d)", j, k)
			}
		}
	}

	if math.Abs(MustAt(t, Corr, 0, 0)-1) > epsTight {
		t.Fatalf("diag[0] != 1")
	}
	if math.Abs(MustAt(t, Corr, 1, 1)-1) > epsTight {
		t.Fatalf("diag[1] != 1")
	}
	if math.Abs(MustAt(t, Corr, 2, 2)-0) > epsTight {
		t.Fatalf("diag[2] != 0 for degenerate")
	}
}

func TestCorrelation_ScaleInvariance_AndFallback(t *testing.T) {
	t.Parallel()

	X := RandFilledDense(t, 20, 6, 123)
	// Scale X by 7 without using Scale() to avoid external deps in the test.
	X7, _ := matrix.NewDense(X.Rows(), X.Cols())
	for i := 0; i < X.Rows(); i++ {
		for j := 0; j < X.Cols(); j++ {
			v, _ := X.At(i, j)
			_ = X7.Set(i, j, 7*v)
		}
	}

	C1, _, _, err := matrix.Correlation(X)
	if err != nil {
		t.Fatalf("Corr(X): %v", err)
	}
	C2, _, _, err := matrix.Correlation(X7)
	if err != nil {
		t.Fatalf("Corr(7X): %v", err)
	}
	CompareClose(t, C1, C2, epsTight, epsTight)

	// Fallback path equality
	Cs, _, _, err := matrix.Correlation(hide{X})
	if err != nil {
		t.Fatalf("Corr slow: %v", err)
	}
	CompareClose(t, C1, Cs, epsTight, epsTight)
}

func TestCorrelation_RowsLessThan2_Error(t *testing.T) {
	t.Parallel()

	X, _ := matrix.NewDense(1, 2)
	_, _, _, err := matrix.Correlation(X)
	if !errors.Is(err, matrix.ErrDimensionMismatch) {
		t.Fatalf("want ErrDimensionMismatch, got %v", err)
	}
}
