// Package builder_test contains unit tests for the WeightFn implementations
// in the builder package, covering both correct behavior and panic conditions.
package builder_test

import (
	"math/rand"
	"testing"

	"github.com/katalvlaran/lvlath/builder"
)

// TestWeightFnConstructors verifies that WeightFn constructors panic
// on invalid parameters according to their documented contracts.
func TestWeightFnConstructors(t *testing.T) {
	t.Parallel() // allow parallel execution of this test

	// table-driven tests for constructor panic conditions
	tests := []struct {
		name        string
		constructor func() builder.WeightFn
	}{
		{"ConstantWeightFn_negative", func() builder.WeightFn { return builder.ConstantWeightFn(-1) }},
		{"UniformWeightFn_minNegative", func() builder.WeightFn { return builder.UniformWeightFn(-1, 5) }},
		{"UniformWeightFn_maxLessThanMin", func() builder.WeightFn { return builder.UniformWeightFn(5, 4) }},
		{"NormalWeightFn_stddevNegative", func() builder.WeightFn { return builder.NormalWeightFn(0, -0.1) }},
		{"ExponentialWeightFn_zeroRate", func() builder.WeightFn { return builder.ExponentialWeightFn(0) }},
		{"ExponentialWeightFn_negativeRate", func() builder.WeightFn { return builder.ExponentialWeightFn(-1) }},
	}

	for _, tc := range tests {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel() // parallel subtest
			assertPanics(t, func() {
				tc.constructor()
			}, tc.name)
		})
	}
}

// TestWeightFnBehavior covers the runtime behavior of each WeightFn:
//   - DefaultWeightFn always returns DefaultEdgeWeight.
//   - ConstantWeightFn returns the fixed value.
//   - UniformWeightFn returns DefaultEdgeWeight on nil RNG, and uniform in [min,max].
//   - From1To100WeightFn returns values in [1,100].
//   - NormalWeightFn returns DefaultEdgeWeight on nil RNG and non-negative samples.
//   - ExponentialWeightFn returns DefaultEdgeWeight on nil RNG and non-negative samples.
func TestWeightFnBehavior(t *testing.T) {
	t.Parallel() // allow parallel execution

	const seed = 42
	rng := rand.New(rand.NewSource(seed)) // reproducible RNG

	// DefaultWeightFn: always DefaultEdgeWeight
	if w := builder.DefaultWeightFn(nil); w != builder.DefaultEdgeWeight {
		t.Errorf("DefaultWeightFn(nil): expected %d, got %d", builder.DefaultEdgeWeight, w)
	}
	if w := builder.DefaultWeightFn(rng); w != builder.DefaultEdgeWeight {
		t.Errorf("DefaultWeightFn(rng): expected %d, got %d", builder.DefaultEdgeWeight, w)
	}

	// ConstantWeightFn: always fixed value
	const constVal = 7
	wfnConst := builder.ConstantWeightFn(constVal)
	if w := wfnConst(nil); w != constVal {
		t.Errorf("ConstantWeightFn(nil): expected %d, got %d", constVal, w)
	}
	if w := wfnConst(rng); w != constVal {
		t.Errorf("ConstantWeightFn(rng): expected %d, got %d", constVal, w)
	}

	// UniformWeightFn: nil RNG -> default; equal min==max yields that value when RNG present
	min, max := int64(3), int64(3)
	wfnUni := builder.UniformWeightFn(min, max)
	if w := wfnUni(nil); w != builder.DefaultEdgeWeight {
		t.Errorf("UniformWeightFn(nil RNG): expected default %d, got %d", builder.DefaultEdgeWeight, w)
	}
	// with RNG and min==max, always min
	if w := wfnUni(rng); w != min {
		t.Errorf("UniformWeightFn(3,3): expected %d, got %d", min, w)
	}

	// From1To100WeightFn: always in [1,100]
	rng = rand.New(rand.NewSource(seed))
	w := builder.From1To100WeightFn(rng)
	if w < 1 || w > 100 {
		t.Errorf("From1To100WeightFn: expected in [1,100], got %d", w)
	}

	// NormalWeightFn: nil RNG -> default; RNG -> non-negative, clipped
	wfnNorm := builder.NormalWeightFn(10, 2)
	if w := wfnNorm(nil); w != builder.DefaultEdgeWeight {
		t.Errorf("NormalWeightFn(nil RNG): expected default %d, got %d", builder.DefaultEdgeWeight, w)
	}
	rng = rand.New(rand.NewSource(seed))
	w = wfnNorm(rng)
	if w < 0 {
		t.Errorf("NormalWeightFn: expected non-negative, got %d", w)
	}

	// ExponentialWeightFn: nil RNG -> default; RNG -> non-negative
	wfnExp := builder.ExponentialWeightFn(1.5)
	if w := wfnExp(nil); w != builder.DefaultEdgeWeight {
		t.Errorf("ExponentialWeightFn(nil RNG): expected default %d, got %d", builder.DefaultEdgeWeight, w)
	}
	rng = rand.New(rand.NewSource(seed))
	w = wfnExp(rng)
	if w < 0 {
		t.Errorf("ExponentialWeightFn: expected non-negative, got %d", w)
	}
}
