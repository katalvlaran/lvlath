package dtw_test

import (
	"math"
	"testing"

	"github.com/katalvlaran/lvlath/dtw"
	"github.com/stretchr/testify/assert"
)

// TestDTW_EmptyInput verifies that DTW returns ErrEmptyInput
// when either input sequence is empty.
func TestDTW_EmptyInput(t *testing.T) {
	opts := dtw.DefaultOptions()

	// Empty first sequence
	_, _, err := dtw.DTW([]float64{}, []float64{1, 2, 3}, &opts)
	assert.ErrorIs(t, err, dtw.ErrEmptyInput, "empty first sequence should error")

	// Empty second sequence
	_, _, err = dtw.DTW([]float64{1, 2, 3}, []float64{}, &opts)
	assert.ErrorIs(t, err, dtw.ErrEmptyInput, "empty second sequence should error")
}

// TestDTW_BadWindowOption ensures that Window < -1 triggers ErrBadInput.
func TestDTW_BadWindowOption(t *testing.T) {
	opts := dtw.DefaultOptions()
	opts.Window = -2

	_, _, err := dtw.DTW([]float64{1}, []float64{1}, &opts)
	assert.ErrorIs(t, err, dtw.ErrBadInput, "Window < -1 must error ErrBadInput")
}

// TestDTW_PathNeedsMatrix ensures ReturnPath=true with non-FullMatrix mode errors.
func TestDTW_PathNeedsMatrix(t *testing.T) {
	opts := dtw.DefaultOptions()
	opts.ReturnPath = true
	opts.MemoryMode = dtw.TwoRows

	_, _, err := dtw.DTW([]float64{1, 2}, []float64{1, 2}, &opts)
	assert.ErrorIs(t, err, dtw.ErrPathNeedsMatrix, "ReturnPath without FullMatrix must error ErrPathNeedsMatrix")
}

// TestDTW_BasicDistance verifies that identical sequences have zero distance
// and no path is returned by default.
func TestDTW_BasicDistance(t *testing.T) {
	a := []float64{0, 1, 2}
	b := []float64{0, 1, 2}
	opts := dtw.DefaultOptions()

	dist, path, err := dtw.DTW(a, b, &opts)
	assert.NoError(t, err, "identical sequences should not error")
	assert.Equal(t, 0.0, dist, "identical sequences must have zero distance")
	assert.Nil(t, path, "default ReturnPath=false should yield nil path")
}

// TestDTW_SyntheticDistanceAndPath checks a perfect subsequence match
// and that the path length equals n + (m-n).
func TestDTW_SyntheticDistanceAndPath(t *testing.T) {
	a := []float64{1, 2, 3}
	b := []float64{1, 2, 2, 3}
	opts := dtw.DefaultOptions()
	opts.ReturnPath = true
	opts.MemoryMode = dtw.FullMatrix

	dist, path, err := dtw.DTW(a, b, &opts)
	assert.NoError(t, err, "should not error on perfect match")
	assert.Equal(t, 0.0, dist, "perfect subsequence match yields zero cost")
	assert.Len(t, path, 4, "path length should be len(a)+(len(b)-len(a))")
	assert.Equal(t, dtw.Coord{I: 0, J: 0}, path[0], "first path point")
	assert.Equal(t, dtw.Coord{I: 2, J: 3}, path[len(path)-1], "last path point")
}

// TestDTW_WindowConstraint verifies that a strict window = 0
// with a length mismatch yields +Inf distance.
func TestDTW_WindowConstraint(t *testing.T) {
	a := []float64{1, 2, 3}
	b := []float64{1, 2, 3, 4}
	opts := dtw.DefaultOptions()
	opts.Window = 0
	opts.MemoryMode = dtw.FullMatrix

	dist, _, err := dtw.DTW(a, b, &opts)
	assert.NoError(t, err, "should not error with window constraint")
	assert.True(t, math.IsInf(dist, 1), "window=0 with length mismatch should yield +Inf")
}

// TestDTW_SlopePenaltyAffectsDistance ensures that a positive slope penalty
// increases the computed distance by exactly that penalty.
func TestDTW_SlopePenaltyAffectsDistance(t *testing.T) {
	a := []float64{1, 2, 3}
	b := []float64{1, 1, 2, 3}

	// No penalty
	opts := dtw.DefaultOptions()
	opts.MemoryMode = dtw.FullMatrix
	dist0, _, err := dtw.DTW(a, b, &opts)
	assert.NoError(t, err)
	assert.Equal(t, 0.0, dist0, "zero penalty allows perfect cost")

	// Penalty = 1.0
	opts.SlopePenalty = 1.0
	dist1, _, err := dtw.DTW(a, b, &opts)
	assert.NoError(t, err)
	assert.Equal(t, 1.0, dist1, "penalty=1.0 adds exactly one unit to distance")
}

// TestDTW_TwoRowsDistanceOnly confirms TwoRows mode matches FullMatrix distance
// and does not return a path.
func TestDTW_TwoRowsDistanceOnly(t *testing.T) {
	a := []float64{0, 1, 2, 3}
	b := []float64{0, 1, 1, 2, 3}

	// Reference with FullMatrix
	refOpts := dtw.DefaultOptions()
	refOpts.MemoryMode = dtw.FullMatrix
	refDist, _, _ := dtw.DTW(a, b, &refOpts)

	// TwoRows mode
	opts := dtw.DefaultOptions()
	opts.MemoryMode = dtw.TwoRows
	dist, path, err := dtw.DTW(a, b, &opts)
	assert.NoError(t, err)
	assert.Equal(t, refDist, dist, "TwoRows must match FullMatrix distance")
	assert.Nil(t, path, "TwoRows should not return a path")
}

// TestDTW_NoMemoryMode confirms NoMemory mode matches FullMatrix distance
// and does not return a path.
func TestDTW_NoMemoryMode(t *testing.T) {
	a := []float64{5, 6, 7}
	b := []float64{5, 7}

	refOpts := dtw.DefaultOptions()
	refOpts.MemoryMode = dtw.FullMatrix
	refDist, _, _ := dtw.DTW(a, b, &refOpts)

	opts := dtw.DefaultOptions()
	opts.MemoryMode = dtw.NoMemory
	dist, path, err := dtw.DTW(a, b, &opts)
	assert.NoError(t, err)
	assert.Equal(t, refDist, dist, "NoMemory must match FullMatrix distance")
	assert.Nil(t, path, "NoMemory should not return a path")
}

// TestDTW_NegativeWindowUnlimited verifies Window=-1 disables constraint.
func TestDTW_NegativeWindowUnlimited(t *testing.T) {
	a := []float64{1, 2, 3, 4}
	b := []float64{1, 2, 3}
	opts := dtw.DefaultOptions()
	opts.Window = -1
	opts.MemoryMode = dtw.FullMatrix

	dist, _, err := dtw.DTW(a, b, &opts)
	assert.NoError(t, err)
	assert.False(t, math.IsInf(dist, 1), "Window=-1 must allow alignment")
}

// TestDTW_BadInputCombination checks that contradictory options error out.
func TestDTW_BadInputCombination(t *testing.T) {
	opts := dtw.DefaultOptions()
	opts.Window = 0
	opts.MemoryMode = dtw.TwoRows
	opts.ReturnPath = true

	_, _, err := dtw.DTW([]float64{1}, []float64{1}, &opts)
	assert.ErrorIs(t, err, dtw.ErrPathNeedsMatrix, "invalid options must return ErrPathNeedsMatrix")
}
