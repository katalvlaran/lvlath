// SPDX-License-Identifier: MIT
// Package matrix_test contains test helpers
//
// Purpose:
//   • Provide small, deterministic test fixtures and utilities for builders/kernels.
//   • Keep all data finite and well-formed to avoid numeric-policy interference.

package matrix_test

import (
	"errors"
	"math"
	"math/rand"
	"testing"

	"github.com/katalvlaran/lvlath/matrix"
)

// Number of vertices used in table-driven graphs.
const V = 8

// Number of edges in an undirected complete graph K_V.
const EComplete = V * (V - 1) / 2

// hide WRAPS any Matrix to hide its concrete type from type assertions.
// Implementation:
//   - Stage 1: Embed matrix.Matrix to forward all methods.
//   - Stage 2: Use hide{X} in tests to force non-*Dense (fallback) paths.
//
// Behavior highlights:
//   - Prevents "*Dense" fast-path via type switch in code under test.
//
// Inputs:
//   - matrix.Matrix: any implementation.
//
// Returns:
//   - hide: wrapper that still satisfies Matrix but masks concrete type.
//
// Errors:
//   - None.
//
// Determinism:
//   - N/A (wrapper only).
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Useful to assert fast-path == fallback bitwise (or via AllClose).
//
// AI-Hints:
//   - Prefer wrapping ONLY the operand you want to de-opt; keep the other one *Dense to isolate path differences.
type hide struct{ matrix.Matrix }

// MustDense ALLOCATES an r×c *Dense or fails the test (fatal on error).
// Implementation:
//   - Stage 1: Call matrix.NewDense(r,c).
//   - Stage 2: t.Fatalf on error to abort the test early.
//
// Behavior highlights:
//   - Concise boilerplate reduction in tests.
//
// Inputs:
//   - r,c: matrix shape.
//
// Returns:
//   - *matrix.Dense allocated with zeroed data.
//
// Errors:
//   - Fatal test failure if allocation fails.
//
// Determinism:
//   - Deterministic zero-initialized buffer.
//
// Complexity:
//   - Time O(r*c) zeroing by runtime, Space O(r*c).
//
// Notes:
//   - Prefer MustDense when subsequent steps assume non-nil Dense.
//
// AI-Hints:
//   - When you need non-zero data, pair with RandomFill or manual Set.
func MustDense(t *testing.T, r, c int) *matrix.Dense {
	t.Helper()
	m, err := matrix.NewDense(r, c)
	if err != nil {
		t.Fatalf("NewDense(%d,%d): %v", r, c, err)
	}

	return m
}

// IdentityDense RETURNS an n×n identity Matrix (main diagonal = 1, else 0).
// Implementation:
//   - Stage 1: matrix.NewIdentity(n).
//   - Stage 2: t.Fatalf on error.
//
// Behavior highlights:
//   - Compact identity builder without exposing internal loops.
//
// Inputs:
//   - n: matrix size (n≥0).
//
// Returns:
//   - matrix.Matrix (likely *Dense) containing I_n.
//
// Errors:
//   - Fatal test failure if allocation fails.
//
// Determinism:
//   - Deterministic pattern (no RNG).
//
// Complexity:
//   - Time O(n^2) (initialization), Space O(n^2).
//
// Notes:
//   - Use in algebra/graph tests to assert neutral operations.
//
// AI-Hints:
//   - Great as a baseline for perturbations and property tests.
func IdentityDense(t *testing.T, n int) matrix.Matrix {
	t.Helper()
	m, err := matrix.NewIdentity(n)
	if err != nil {
		t.Fatalf("NewIdentity(%d): %v", n, err)
	}

	return m
}

// NewFilledDense BUILDS r×c *Dense from a row-major flat slice.
// Implementation:
//   - Stage 1: Validate len(vals)==r*c.
//   - Stage 2: Allocate Dense and Set(i,j, vals[i*c+j]).
//
// Behavior highlights:
//   - Deterministic fixture creation with explicit values.
//
// Inputs:
//   - r,c: shape; vals: row-major data of length r*c.
//
// Returns:
//   - *matrix.Dense with copied values.
//
// Errors:
//   - Fatal test failure if lengths mismatch or Set fails.
//
// Determinism:
//   - Deterministic fill order.
//
// Complexity:
//   - Time O(r*c), Space O(r*c).
//
// Notes:
//   - Prefer for small exact-equality tests.
//
// AI-Hints:
//   - Use with CompareExact for integer-like matrices.
func NewFilledDense(t *testing.T, r, c int, vals []float64) *matrix.Dense {
	t.Helper()
	if len(vals) != r*c {
		t.Fatalf("NewFilledDence: want %d values, got %d", r*c, len(vals))
	}
	d := MustDense(t, r, c)
	var i, j int // loop iterators
	for i = 0; i < r; i++ {
		for j = 0; j < c; j++ {
			MustSet(t, d, i, j, vals[i*c+j])
		}
	}

	return d
}

// RandomFill FILLS a Matrix with deterministic U(-1,1) values by seed.
// Implementation:
//   - Stage 1: rng := rand.New(rand.NewSource(seed)).
//   - Stage 2: For each cell, Set(i,j, rng.Float64()*2-1).
//
// Behavior highlights:
//   - Reproducible randomness for property tests.
//
// Inputs:
//   - m: target Matrix; seed: RNG seed.
//
// Returns:
//   - None (mutates m).
//
// Errors:
//   - Fatal test failure if Set returns error.
//
// Determinism:
//   - Deterministic for a fixed seed.
//
// Complexity:
//   - Time O(r*c), Space O(1) extra.
//
// Notes:
//   - Keeps values finite to avoid NaN/Inf policy interference.
//
// AI-Hints:
//   - Sweep multiple seeds in table-driven tests to increase coverage.
func RandomFill(t *testing.T, m matrix.Matrix, seed int64) {
	t.Helper()
	rng := rand.New(rand.NewSource(seed))
	r, c := m.Rows(), m.Cols()
	var (
		i, j int     // loop iterators
		v    float64 // random value
		err  error
	)
	for i = 0; i < r; i++ {
		for j = 0; j < c; j++ {
			v = rng.Float64()*2 - 1 // 0*2-1=-1 || 1*2-1=1
			if err = m.Set(i, j, v); err != nil {
				t.Fatalf("Set RandomFill(%d,%d): %v", i, j, err)
			}
		}
	}
}

// RandFilledDense RETURNS a new r×c Dense filled with deterministic U(-1,1).
// Implementation:
//   - Stage 1: Allocate Dense.
//   - Stage 2: Fill via seeded RNG, row-major.
//
// Behavior highlights:
//   - One-liner to allocate+fill.
//
// Inputs:
//   - r,c: shape; seed: RNG seed.
//
// Returns:
//   - matrix.Matrix (Dense) populated with random values.
//
// Errors:
//   - Fatal test failure if allocation/Set fails.
//
// Determinism:
//   - Deterministic per seed.
//
// Complexity:
//   - Time O(r*c), Space O(r*c).
//
// Notes:
//   - Prefer for medium-sized randomized tests.
//
// AI-Hints:
//   - Use identical seeds across fast vs fallback to isolate path differences.
func RandFilledDense(t *testing.T, r, c int, seed int64) matrix.Matrix {
	t.Helper()
	m, err := matrix.NewDense(r, c)
	if err != nil {
		t.Fatalf("NewDense(%d,%d): %v", r, c, err)
	}
	rng := rand.New(rand.NewSource(seed))
	var i, j int
	for i = 0; i < r; i++ {
		for j = 0; j < c; j++ {
			if err = m.Set(i, j, rng.Float64()*2-1); err != nil {
				t.Fatalf("Set(%d,%d): %v", i, j, err)
			}
		}
	}
	return m
}

// MustSet WRITES v to m[i,j] or fails the test.
// Implementation:
//   - Stage 1: Call m.Set(i,j,v).
//   - Stage 2: t.Fatalf on error.
//
// Behavior highlights:
//   - Provides concise error text with indices.
//
// Inputs:
//   - m,i,j,v: target matrix, coordinates, value.
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal test failure on Set error.
//
// Determinism:
//   - N/A.
//
// Complexity:
//   - O(1) per call.
//
// Notes:
//   - Avoids boilerplate if err != nil {...} in tests.
//
// AI-Hints:
//   - Great with small builders like NewFilledDense.
func MustSet(t *testing.T, m matrix.Matrix, i, j int, v float64) {
	t.Helper()
	if err := m.Set(i, j, v); err != nil {
		t.Fatalf("Set(%d,%d,%v): %v", i, j, v, err)
	}
}

// MustAt READS m[i,j] or fails the test.
// Implementation:
//   - Stage 1: Call m.At(i,j).
//   - Stage 2: t.Fatalf on error, return value otherwise.
//
// Behavior highlights:
//   - Clear failure site on bounds/impl errors.
//
// Inputs:
//   - m,i,j.
//
// Returns:
//   - float64 value.
//
// Errors:
//   - Fatal test failure on At error.
//
// Determinism:
//   - N/A.
//
// Complexity:
//   - O(1) per call.
//
// Notes:
//   - Pair with CompareExact/Close.
//
// AI-Hints:
//   - Safe for fallback paths where At may allocate internally.
func MustAt(t *testing.T, m matrix.Matrix, i, j int) float64 {
	t.Helper()
	v, err := m.At(i, j)
	if err != nil {
		t.Fatalf("At(%d,%d): %v", i, j, err)
	}

	return v
}

// CompareExact ASSERTS strict equality between matrix and 2D literal.
// Implementation:
//   - Stage 1: Shape checks.
//   - Stage 2: Iterate and compare with == (no tolerances).
//
// Behavior highlights:
//   - Fails with exact mismatch location.
//
// Inputs:
//   - want: [][]float64 expected; m: Matrix under test.
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal test failure on size/value mismatch.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(r*c), Space O(1).
//
// Notes:
//   - Use only for integer-like or carefully crafted small matrices.
//
// AI-Hints:
//   - For floats use CompareClose instead.
func CompareExact(t *testing.T, want [][]float64, m matrix.Matrix) {
	t.Helper()
	r, c := m.Rows(), m.Cols()
	if len(want) != r {
		t.Fatalf("CompareExact: Rows = %d; want %d", r, len(want))
	}
	var i, j int // loop iterators
	var v float64
	for i = 0; i < r; i++ {
		if len(want[i]) != c {
			t.Fatalf("CompareExact: Cols[%d] = %d; want %d", i, c, len(want[i]))
		}
		for j = 0; j < c; j++ {
			if v = MustAt(t, m, i, j); v != want[i][j] {
				t.Fatalf("m[%d,%d]=%v; want %v", i, j, v, want[i][j])
			}

		}
	}
}

// CompareClose ASSERTS AllClose(a,b) under (rtol, atol).
// Implementation:
//   - Stage 1: matrix.AllClose(a,b, rtol, atol).
//   - Stage 2: t.Fatalf if false or if AllClose returns error.
//
// Behavior highlights:
//   - Encapsulates numeric tolerance logic used across tests.
//
// Inputs:
//   - a,b: matrices; rtol,atol: tolerances.
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal test failure on mismatch or AllClose error.
//
// Determinism:
//   - Deterministic for fixed inputs.
//
// Complexity:
//   - Time O(r*c), Space O(1).
//
// Notes:
//   - Consider constants e.g. RtolTiny=1e-12, AtolTiny=1e-12 for reuse.
//
// AI-Hints:
//   - Use (0,0) for pure equality when numbers are exact.
func CompareClose(t *testing.T, a, b matrix.Matrix, rtol, atol float64) {
	t.Helper()
	ok, err := matrix.AllClose(a, b, rtol, atol)
	if err != nil {
		t.Fatalf("AllClose err: %v", err)
	}
	if !ok {
		t.Fatalf("AllClose=false (rtol=%g, atol=%g)", rtol, atol)
	}
}

// sliceClose ASSERTS |a[i]-b[i]| ≤ atol + rtol*|b[i]| element-wise.
// Implementation:
//   - Stage 1: Length check.
//   - Stage 2: Iterate with tolerance formula (as in AllClose).
//
// Behavior highlights:
//   - Aligns with matrix.AllClose policy for 1D slices.
//
// Inputs:
//   - a,b: slices; rtol,atol: tolerances.
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal test failure on mismatch.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(n), Space O(1).
//
// Notes:
//   - Useful for means/stds snapshots.
//
// AI-Hints:
//   - Keep tolerances consistent with CompareClose to avoid split-brain.
func sliceClose(t *testing.T, a, b []float64, rtol, atol float64) {
	t.Helper()
	if len(a) != len(b) {
		t.Fatalf("slice lengths: %d vs %d", len(a), len(b))
	}
	var diff, absb float64
	for i := range a {
		diff = a[i] - b[i]
		if diff < 0 {
			diff = -diff
		}
		absb = b[i]
		if absb < 0 {
			absb = -absb
		}
		if diff > (atol + rtol*absb) {
			t.Fatalf("sliceClose idx=%d: got=%g want=%g (rtol=%g atol=%g)", i, a[i], b[i], rtol, atol)
		}
	}
}

// AlmostEqualSlice CHECKS |a[i]-b[i]| ≤ eps for all i (boolean, not fatal).
// Implementation:
//   - Stage 1: Length check.
//   - Stage 2: abs-diff compare vs eps.
//
// Behavior highlights:
//   - Non-fatal predicate for conditional flows in tests.
//
// Inputs:
//   - a,b: slices; eps: absolute tolerance.
//
// Returns:
//   - bool: true if close.
//
// Errors:
//   - None.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(n), Space O(1).
//
// Notes:
//   - Keep for parity; prefer sliceClose for consistent failure messages.
//
// AI-Hints:
//   - Can drive table filters (skip flaky, etc.).
func AlmostEqualSlice(a, b []float64, eps float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if math.Abs(a[i]-b[i]) > eps {
			return false
		}
	}

	return true
}

// AssertErrorIs WRAPS errors.Is with consistent failure text.
// Implementation:
//   - Stage 1: if !errors.Is(err, target) → t.Fatalf.
//
// Behavior highlights:
//   - Reduces repeated boilerplate for sentinel checks.
//
// Inputs:
//   - err, target.
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal test failure if not matching.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - O(depth) for errors.Is chain.
//
// Notes:
//   - Prefer for ErrNilMatrix, ErrDimensionMismatch checks.
//
// AI-Hints:
//   - Combine with table-driven tests for coverage.
func AssertErrorIs(t *testing.T, err, target error) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Fatalf("want %v; got %v", target, err)
	}
}

// ExpectPanic ASSERTS that fn() panics (any value).
// Implementation:
//   - Stage 1: defer recover().
//   - Stage 2: t.Fatalf if recover()==nil.
//
// Behavior highlights:
//   - Clear intent when guarding parameter panics.
//
// Inputs:
//   - fn: closure expected to panic.
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal test failure if no panic.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - O(1).
//
// Notes:
//   - For typed panics, extend with predicate if/when needed.
//
// AI-Hints:
//   - Use in options guards (WithEpsilon, WithEdgeThreshold).
func ExpectPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatalf("expected panic, got nil")
		}
	}()
	fn()
}

// InDelta RETURNS whether |a-b| ≤ delta (boolean, non-fatal).
// Implementation:
//   - Stage 1: Compute diff and compare to ±delta.
//
// Behavior highlights:
//   - Lightweight predicate for coarse checks.
//
// Inputs:
//   - a,b: values; delta: absolute band.
//
// Returns:
//   - bool.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - O(1).
//
// Notes:
//   - Prefer CompareClose for matrices; keep InDelta for scalar asserts.
//
// AI-Hints:
//   - Useful for sanity checks on norms, traces, etc.
func InDelta(t *testing.T, a, b float64, delta float64) bool {
	t.Helper()
	diff := a - b
	if diff < -delta || diff > delta {
		return false
	}

	return true
}

// RowL1Norm RETURNS L1 norm of row i (Σ_j |m[i,j]|).
// Implementation:
//   - Stage 1: Iterate columns; abs accumulation.
//
// Behavior highlights:
//   - Convenience routine for normalization tests.
//
// Inputs:
//   - m: Matrix; i: row index.
//
// Returns:
//   - float64 L1 norm.
//
// Errors:
//   - Ignores At errors (safe in Dense tests). Convert to MustAt if needed.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(c), Space O(1).
//
// Notes:
//   - For exported helper semantics you can use MustAt to be strict.
//
// AI-Hints:
//   - Combine with NormalizeRowsL1 invariants.
func RowL1Norm(m matrix.Matrix, i int) float64 {
	var j int
	var s, v float64
	for j = 0; j < m.Cols(); j++ {
		v, _ = m.At(i, j)
		if v < 0 {
			v = -v
		}
		s += v
	}

	return s
}

// RowL2Norm RETURNS L2 norm of row i (sqrt(Σ_j m[i,j]^2)).
// Implementation:
//   - Stage 1: Iterate columns; sum of squares → sqrt.
//
// Behavior highlights:
//   - Convenience routine for L2 normalization tests.
//
// Inputs:
//   - m: Matrix; i: row index.
//
// Returns:
//   - float64 L2 norm.
//
// Errors:
//   - Ignores At errors; acceptable for local Dense tests.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(c), Space O(1).
//
// Notes:
//   - Keep consistent with NormalizeRowsL2 expectations.
//
// AI-Hints:
//   - Combine with InDelta for quick ≈1 checks.
func RowL2Norm(m matrix.Matrix, i int) float64 {
	var j int
	var s, v float64
	for j = 0; j < m.Cols(); j++ {
		v, _ = m.At(i, j)
		s += v * v
	}

	return math.Sqrt(s)
}

/* !! Preparations of needed matrix checkers and helpers for the future, (self)TEST ONLY !!

// IsSymmetricWithin checks |A[i,j]-A[j,i]| <= atol + rtol*|A[j,i]|
func IsSymmetricWithin(t *testing.T, A matrix.Matrix, rtol, atol float64)

// PSDProbe checks vᵀAv >= -eps for a few random v (cheap PSD sanity).
func PSDProbe(t *testing.T, A matrix.Matrix, trials int, seed int64, eps float64)

// Shape assertions
func MustDims(t *testing.T, m matrix.Matrix, r, c int)

// CompareWithMask compares only positions where mask[i*c+j] == true.
func CompareWithMask(t *testing.T, want [][]float64, got matrix.Matrix, mask []bool)

*/

// ---------- bench helpers () ----------

func mustDense(b *testing.B, r, c int) *matrix.Dense {
	d, err := matrix.NewZeros(r, c) // fast path alloc + zero
	if err != nil {
		b.Fatalf("NewZeros(%d,%d): %v", r, c, err)
	}
	return d
}

func fillDenseRand(b *testing.B, d *matrix.Dense, seed int64) {
	rng := rand.New(rand.NewSource(seed))
	rows, cols := d.Rows(), d.Cols()
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			_ = d.Set(i, j, rng.Float64()*2-1) // [-1,1]
		}
	}
}

func onesVec(n int) []float64 {
	v := make([]float64, n)
	for i := 0; i < n; i++ {
		v[i] = 1
	}
	return v
}
