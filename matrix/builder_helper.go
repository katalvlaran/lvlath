// SPDX-License-Identifier: MIT
// Package: matrix (test helpers)
//
// Purpose:
//   • Provide small, deterministic test fixtures and utilities for builders/kernels.
//   • Keep all data finite and well-formed to avoid numeric-policy interference.
//
// Note:
//   • Consider moving this file to internal/matrixtest to avoid leaking testing helpers into the main package surface.

package matrix

import (
	"math/rand"
	"testing"
)

// Canonical lengths for small test matrices.
const (
	LenM3 = 3 // 3×3
	LenM6 = 6 // 6×6
	LenM8 = 8 // 8×8
)

var (
	// DataM3T1: symmetric 3×3 pattern (e.g., simple undirected adjacency).
	DataM3T1 = [][]float64{
		{1.0, 0.0, 1.0},
		{0.0, 1.0, 0.0},
		{1.0, 0.0, 1.0},
	}
	// DataM3T2: 3×3 non-symmetric pattern (e.g., directed adjacency).
	DataM3T2 = [][]float64{
		{1.0, 1.0, 0.0},
		{1.0, 1.0, 1.0},
		{0.0, 1.0, 1.0},
	}
	// DataM6T1: 6×6 symmetric-like pattern for medium tests.
	DataM6T1 = [][]float64{
		{1.0, 0.0, 1.0, 1.0, 0.0, 0.0},
		{0.0, 1.0, 0.0, 0.0, 1.0, 0.0},
		{1.0, 0.0, 1.0, 0.0, 0.0, 1.0},
		{1.0, 0.0, 0.0, 1.0, 0.0, 1.0},
		{0.0, 1.0, 0.0, 0.0, 1.0, 0.0},
		{0.0, 0.0, 1.0, 1.0, 0.0, 1.0},
	}
	// DataM6T2: 6×6 diagonal & simple off-diagonal pattern.
	DataM6T2 = [][]float64{
		{1.0, 0.0, 0.0, 0.0, 0.0, 0.0},
		{0.0, 1.0, 0.0, 1.0, 0.0, 0.0},
		{0.0, 0.0, 1.0, 0.0, 1.0, 0.0},
		{0.0, 1.0, 0.0, 1.0, 0.0, 0.0},
		{0.0, 0.0, 1.0, 0.0, 1.0, 0.0},
		{0.0, 0.0, 0.0, 0.0, 0.0, 1.0},
	}
	// DataM8T1: 8×8 blocky pattern for larger cases.
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

// Random fills m with reproducible pseudorandoms in [-1, 1].
// Seed controls determinism; panics avoided via t.Fatalf on Set errors.
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

// Compare asserts that m matches the 2-D slice want exactly; fails the test on mismatch.
// Intended for small fixtures; use AllClose for tolerant comparisons.
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
