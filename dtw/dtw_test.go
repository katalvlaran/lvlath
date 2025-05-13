package dtw_test

import (
	"math"
	"testing"

	"github.com/katalvlaran/lvlath/dtw"
	"github.com/stretchr/testify/assert"
)

func TestDTW_EmptySequence(t *testing.T) {
	_, _, err := dtw.DTW([]float64{}, []float64{1, 2, 3}, nil)
	assert.ErrorIs(t, err, dtw.ErrEmptySequence, "empty first sequence should error")

	_, _, err = dtw.DTW([]float64{1, 2, 3}, []float64{}, nil)
	assert.ErrorIs(t, err, dtw.ErrEmptySequence, "empty second sequence should error")
}

func TestDTW_PathRequiresFullMatrix(t *testing.T) {
	opts := &dtw.DTWOptions{
		ReturnPath: true,
		MemoryMode: dtw.RollingArray,
	}
	_, _, err := dtw.DTW([]float64{1, 2}, []float64{1, 2}, opts)
	assert.ErrorIs(t, err, dtw.ErrPathNeedsFullMatrix, "ReturnPath with RollingArray must error")
}

func TestDTW_BasicDistance(t *testing.T) {
	a := []float64{0, 1, 2}
	b := []float64{0, 1, 2}
	// default options: no window, zero penalty, FullMatrix implied
	dist, path, err := dtw.DTW(a, b, nil)
	assert.NoError(t, err)
	assert.Equal(t, 0.0, dist, "identical sequences should have zero distance")
	// path is empty because ReturnPath defaults to false
	assert.Nil(t, path)
}

func TestDTW_SyntheticDistanceAndPath(t *testing.T) {
	a := []float64{1, 2, 3}
	b := []float64{1, 2, 2, 3}
	opts := &dtw.DTWOptions{
		ReturnPath:   true,
		MemoryMode:   dtw.FullMatrix,
		SlopePenalty: 0.0,
	}
	dist, path, err := dtw.DTW(a, b, opts)
	assert.NoError(t, err)
	// The optimal alignment is matching 1-1, 2-2, 3-3 => cost = 0
	assert.Equal(t, 0.0, dist, "perfect subsequence match yields zero cost")
	// Path should start at (0,0) and end at (2,3)
	assert.Len(t, path, 4, "path length should be n + offset")
	assert.Equal(t, [2]int{0, 0}, path[0], "first point")
	assert.Equal(t, [2]int{2, 3}, path[len(path)-1], "last point")
}

func TestDTW_WindowConstraint(t *testing.T) {
	// sequences differ in length; window=0 blocks off-diagonals
	a := []float64{1, 2, 3}
	b := []float64{1, 2, 3, 4}
	opts := &dtw.DTWOptions{
		Window:     0,
		MemoryMode: dtw.FullMatrix,
	}
	dist, _, err := dtw.DTW(a, b, opts)
	assert.NoError(t, err)
	assert.True(t, math.IsInf(dist, 1), "window=0 and len(b)=len(a)+1 should force infinite distance")
}

func TestDTW_SlopePenaltyAffectsDistance(t *testing.T) {
	a := []float64{1, 2, 3}
	b := []float64{1, 1, 2, 3}
	// Without penalty, extra element can be matched with zero cost.
	opts0 := &dtw.DTWOptions{
		MemoryMode: dtw.FullMatrix,
	}
	dist0, _, err := dtw.DTW(a, b, opts0)
	assert.NoError(t, err)
	assert.Equal(t, 0.0, dist0, "zero penalty allows perfect cost")

	// With penalty > 0, skipping step costs penalty twice (insert+delete).
	opts1 := &dtw.DTWOptions{
		MemoryMode:   dtw.FullMatrix,
		SlopePenalty: 1.0,
	}
	dist1, _, err := dtw.DTW(a, b, opts1)
	assert.NoError(t, err)
	assert.Equal(t, 2.0, dist1, "penalty of 1 adds cost for insertion/deletion")
}
