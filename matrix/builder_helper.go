package matrix

import (
	"math/rand"
	"testing"
)

const (
	// ???
	LenM3 = 3
	LenM6 = 6
	LenM8 = 8
)

var (
	// ??
	DataM3T1 = [][]float64{
		{1.0, 0.0, 1.0},
		{0.0, 1.0, 0.0},
		{1.0, 0.0, 1.0},
	}
	// ??
	DataM3T2 = [][]float64{
		{1.0, 1.0, 0.0},
		{1.0, 1.0, 1.0},
		{0.0, 1.0, 1.0},
	}
	// ??
	DataM6T1 = [][]float64{
		{1.0, 0.0, 1.0, 1.0, 0.0, 0.0},
		{0.0, 1.0, 0.0, 0.0, 1.0, 0.0},
		{1.0, 0.0, 1.0, 0.0, 0.0, 1.0},
		{1.0, 0.0, 0.0, 1.0, 0.0, 1.0},
		{0.0, 1.0, 0.0, 0.0, 1.0, 0.0},
		{0.0, 0.0, 1.0, 1.0, 0.0, 1.0},
	}
	// ??
	DataM6T2 = [][]float64{
		{1.0, 0.0, 0.0, 0.0, 0.0, 0.0},
		{0.0, 1.0, 0.0, 1.0, 0.0, 0.0},
		{0.0, 0.0, 1.0, 0.0, 1.0, 0.0},
		{0.0, 1.0, 0.0, 1.0, 0.0, 0.0},
		{0.0, 0.0, 1.0, 0.0, 1.0, 0.0},
		{0.0, 0.0, 0.0, 0.0, 0.0, 1.0},
	}
	// ??
	DataM8T1 = [][]float64{
		{1.0, 0.0, 0.0, 0.0, 1.0, 0.0, 0.0, 0.0},
		{0.0, 1.0, 0.0, 1.0, 0.0, 1.0, 0.0, 0.0},
		{0.0, 0.0, 1.0, 0.0, 0.0, 0.0, 1.0, 0.0},
		{0.0, 1.0, 0.0, 1.0, 0.0, 0.0, 0.0, 1.0},
		{1.0, 0.0, 0.0, 0.0, 1.0, 0.0, 1.0, 0.0},
		{0.0, 1.0, 0.0, 0.0, 0.0, 1.0, 0.0, 0.0},
		{0.0, 0.0, 1.0, 0.0, 1.0, 0.0, 1.0, 0.0},
		{0.0, 0.0, 0.0, 1.0, 0.0, 0.0, 0.0, 1.0},
	}
)

// Random fills m with reproducible pseudorandom in [-1,1).
// ??
func Random(t *testing.T, m Matrix, seed int64) {
	t.Helper()
	rng := rand.New(rand.NewSource(seed))
	r, c := m.Rows(), m.Cols()
	var (
		i, j int // loop iterators
		v    float64
		err  error
	)
	for i = 0; i < r; i++ {
		for j = 0; j < c; j++ {
			v = rng.Float64()*2 - 1 // 0*2-1=-1 || 1*2-1=1
			if err = m.Set(i, j, v); err != nil {
				t.Fatalf("Set random(%d,%d): %v", i, j, err)
			}
		}
	}
}

// Compare asserts that m matches the 2-D slice want exactly.
// ??
func Compare(t *testing.T, want [][]float64, m Matrix) {
	t.Helper()
	r, c := m.Rows(), m.Cols()
	if len(want) != r {
		t.Fatalf("Rows = %d; want %d", r, len(want))
	}
	var (
		i, j int // loop iterators
		v    float64
		err  error
	)
	for i = 0; i < r; i++ {
		if len(want[i]) != c {
			t.Fatalf("Cols[%d] = %d; want %d", i, c, len(want[i]))
		}
		for j = 0; j < c; j++ {
			v, err = m.At(i, j)
			if err != nil {
				t.Fatalf("At(%d,%d): %v", i, j, err)
			}
			if v != want[i][j] {
				t.Errorf("At(%d,%d) = %v; want %v", i, j, v, want[i][j])
			}
		}
	}
}
