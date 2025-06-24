package dtw_test

import (
	"math"
	"testing"

	"github.com/katalvlaran/lvlath/dtw"
	"github.com/stretchr/testify/assert"
)

func TestDTW_EmptyInput(t *testing.T) {
	opts := dtw.DefaultOptions()
	_, _, err := dtw.DTW([]float64{}, []float64{1, 2, 3}, opts)
	assert.ErrorIs(t, err, dtw.ErrEmptyInput, "empty first sequence should error")

	_, _, err = dtw.DTW([]float64{1, 2, 3}, []float64{}, opts)
	assert.ErrorIs(t, err, dtw.ErrEmptyInput, "empty second sequence should error")
}

func TestDTW_PathNeedsMatrix(t *testing.T) {
	opts := dtw.DefaultOptions()
	opts.ReturnPath = true
	opts.MemoryMode = dtw.TwoRows

	_, _, err := dtw.DTW([]float64{1, 2}, []float64{1, 2}, opts)
	assert.ErrorIs(t, err, dtw.ErrPathNeedsMatrix, "ReturnPath without FullMatrix should error")
}

func TestDTW_BasicDistance(t *testing.T) {
	a := []float64{0, 1, 2}
	b := []float64{0, 1, 2}
	opts := dtw.DefaultOptions()
	dist, path, err := dtw.DTW(a, b, opts)
	assert.NoError(t, err)
	assert.Equal(t, 0.0, dist, "identical sequences must have zero distance")
	assert.Nil(t, path, "default ReturnPath=false should yield nil path")
}

func TestDTW_SyntheticDistanceAndPath(t *testing.T) {
	a := []float64{1, 2, 3}
	b := []float64{1, 2, 2, 3}
	opts := dtw.DefaultOptions()
	opts.ReturnPath = true
	opts.MemoryMode = dtw.FullMatrix
	opts.SlopePenalty = 0.0

	dist, path, err := dtw.DTW(a, b, opts)
	assert.NoError(t, err)
	assert.Equal(t, 0.0, dist, "perfect subsequence match yields zero cost")
	assert.Len(t, path, 4, "path length should equal n + offset")
	assert.Equal(t, dtw.Coord{I: 0, J: 0}, path[0], "first path point")
	assert.Equal(t, dtw.Coord{I: 2, J: 3}, path[len(path)-1], "last path point")
}

func TestDTW_WindowConstraint(t *testing.T) {
	a := []float64{1, 2, 3}
	b := []float64{1, 2, 3, 4}
	opts := dtw.DefaultOptions()
	opts.Window = 0
	opts.MemoryMode = dtw.FullMatrix

	dist, _, err := dtw.DTW(a, b, opts)
	assert.NoError(t, err)
	assert.True(t, math.IsInf(dist, 1), "window=0 with length mismatch should yield +Inf")
}

func TestDTW_SlopePenaltyAffectsDistance(t *testing.T) {
	a := []float64{1, 2, 3}
	b := []float64{1, 1, 2, 3}

	// Zero penalty
	opts0 := dtw.DefaultOptions()
	opts0.MemoryMode = dtw.FullMatrix
	dist0, _, err := dtw.DTW(a, b, opts0)
	assert.NoError(t, err)
	assert.Equal(t, 0.0, dist0, "zero penalty allows perfect cost")

	// Positive penalty
	opts1 := dtw.DefaultOptions()
	opts1.MemoryMode = dtw.FullMatrix
	opts1.SlopePenalty = 1.0
	dist1, _, err := dtw.DTW(a, b, opts1)
	assert.NoError(t, err)
	assert.Equal(t, 1.0, dist1, "penalty 1 adds cost per warp step")
}

func TestDTW_TwoRowsDistanceOnly(t *testing.T) {
	a := []float64{0, 1, 2, 3}
	b := []float64{0, 1, 1, 2, 3}
	// FullMatrix for reference
	reference, _, _ := dtw.DTW(a, b, dtw.DefaultOptions())

	opts := dtw.DefaultOptions()
	opts.MemoryMode = dtw.TwoRows
	distance, path, err := dtw.DTW(a, b, opts)
	assert.NoError(t, err)
	assert.Equal(t, reference, distance, "TwoRows must match FullMatrix distance")
	assert.Nil(t, path, "TwoRows should not return a path")
}

func TestDTW_NoneMode(t *testing.T) {
	a := []float64{5, 6, 7}
	b := []float64{5, 7}
	// Reference distance
	reference, _, _ := dtw.DTW(a, b, dtw.DefaultOptions())

	opts := dtw.DefaultOptions()
	opts.MemoryMode = dtw.None
	distance, path, err := dtw.DTW(a, b, opts)
	assert.NoError(t, err)
	assert.Equal(t, reference, distance, "None mode must match FullMatrix distance")
	assert.Nil(t, path, "None mode should not return a path")
}

func TestDTW_NegativeWindowUnlimited(t *testing.T) {
	a := []float64{1, 2, 3, 4}
	b := []float64{1, 2, 3}
	opts := dtw.DefaultOptions()
	opts.Window = -1 // unlimited
	opts.MemoryMode = dtw.FullMatrix
	distance, _, err := dtw.DTW(a, b, opts)
	assert.NoError(t, err)
	assert.NotEqual(t, math.Inf(1), distance, "negative window should allow alignment")
}

func TestDTW_BadInputCombination(t *testing.T) {
	opts := dtw.DefaultOptions()
	opts.Window = 0
	opts.MemoryMode = dtw.TwoRows
	opts.ReturnPath = true
	_, _, err := dtw.DTW([]float64{1}, []float64{1}, opts)
	assert.ErrorIs(t, err, dtw.ErrPathNeedsMatrix, "invalid options must error ErrBadInput or ErrPathNeedsMatrix")
}

func BenchmarkDTW_Small(b *testing.B) {
	a := make([]float64, 100)
	bArr := make([]float64, 100)
	opts := dtw.DefaultOptions()
	for i := range a {
		a[i], bArr[i] = float64(i), float64(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = dtw.DTW(a, bArr, opts)
	}
}

func BenchmarkDTW_Medium(b *testing.B) {
	N := 500
	a := make([]float64, N)
	bArr := make([]float64, N)
	for i := range a {
		a[i], bArr[i] = float64(i), float64(i)
	}
	opts := dtw.DefaultOptions()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = dtw.DTW(a, bArr, opts)
	}
}
