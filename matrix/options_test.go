// SPDX-License-Identifier: MIT
package matrix_test

import (
	"math"
	"testing"

	"github.com/katalvlaran/lvlath/matrix"
)

// TestDefaultOptions verifies that NewMatrixOptions() equals documented defaults.
func TestDefaultOptions(t *testing.T) {
	o := matrix.NewMatrixOptionsSnapshot_TestOnly() // equal to o := defaultOptions()

	// numeric
	if o.Eps != matrix.DefaultEpsilon {
		t.Fatalf("eps default mismatch: got %v, want %v", o.Eps, matrix.DefaultEpsilon)
	}
	if o.ValidateNaNInf != matrix.DefaultValidateNaNInf {
		t.Fatalf("validateNaNInf default mismatch: got %v, want %v", o.ValidateNaNInf, matrix.DefaultValidateNaNInf)
	}

	// build policy
	if o.Directed != matrix.DefaultDirected {
		t.Fatalf("directed default mismatch: got %v, want %v", o.Directed, matrix.DefaultDirected)
	}
	if o.AllowMulti != matrix.DefaultAllowMulti {
		t.Fatalf("allowMulti default mismatch: got %v, want %v", o.AllowMulti, matrix.DefaultAllowMulti)
	}
	if o.AllowLoops != matrix.DefaultAllowLoops {
		t.Fatalf("allowLoops default mismatch: got %v, want %v", o.AllowLoops, matrix.DefaultAllowLoops)
	}
	if o.Weighted != matrix.DefaultWeighted {
		t.Fatalf("weighted default mismatch: got %v, want %v", o.Weighted, matrix.DefaultWeighted)
	}
	if o.MetricClose != matrix.DefaultMetricClosure {
		t.Fatalf("metricClose default mismatch: got %v, want %v", o.MetricClose, matrix.DefaultMetricClosure)
	}

	// export policy
	if o.EdgeThreshold != matrix.DefaultEdgeThreshold {
		t.Fatalf("edgeThreshold default mismatch: got %v, want %v", o.EdgeThreshold, matrix.DefaultEdgeThreshold)
	}
	if o.KeepWeights != matrix.DefaultKeepWeights {
		t.Fatalf("keepWeights default mismatch: got %v, want %v", o.KeepWeights, matrix.DefaultKeepWeights)
	}
	// undirected is derived in gatherOptions, not here.
}

// TestNewMatrixOptions_OrderAndIdempotence ensures each Option toggles exactly its intended field.
func TestNewMatrixOptions_OrderAndIdempotence(t *testing.T) {
	o1 := matrix.GatherOptionsSnapshot_TestOnly(matrix.WithDirected(), matrix.WithUndirected()) // last wins
	if o1.Directed != false {
		t.Fatalf("last-writer-wins failed: directed=%v, want false", o1.Directed)
	}
	o2 := matrix.GatherOptionsSnapshot_TestOnly(matrix.WithUndirected(), matrix.WithDirected())
	if o2.Directed != true {
		t.Fatalf("last-writer-wins failed: directed=%v, want true", o2.Directed)
	}

	o3 := matrix.GatherOptionsSnapshot_TestOnly(matrix.WithAllowMulti(), matrix.WithDisallowMulti())
	if o3.AllowMulti != false {
		t.Fatalf("allowMulti last-writer-wins failed: %v", o3.AllowMulti)
	}
	o4 := matrix.GatherOptionsSnapshot_TestOnly(matrix.WithDisallowMulti(), matrix.WithAllowMulti())
	if o4.AllowMulti != true {
		t.Fatalf("allowMulti last-writer-wins failed: %v", o4.AllowMulti)
	}

	o5 := matrix.GatherOptionsSnapshot_TestOnly(matrix.WithAllowLoops(), matrix.WithDisallowLoops())
	if o5.AllowLoops != false {
		t.Fatalf("allowLoops last-writer-wins failed: %v", o5.AllowLoops)
	}
	o6 := matrix.GatherOptionsSnapshot_TestOnly(matrix.WithDisallowLoops(), matrix.WithAllowLoops())
	if o6.AllowLoops != true {
		t.Fatalf("allowLoops last-writer-wins failed: %v", o6.AllowLoops)
	}

	o7 := matrix.GatherOptionsSnapshot_TestOnly(matrix.WithUnweighted(), matrix.WithWeighted())
	if o7.Weighted != true {
		t.Fatalf("weighted last-writer-wins failed: %v", o7.Weighted)
	}
	o8 := matrix.GatherOptionsSnapshot_TestOnly(matrix.WithWeighted(), matrix.WithUnweighted())
	if o8.Weighted != false {
		t.Fatalf("weighted last-writer-wins failed: %v", o8.Weighted)
	}

	o9 := matrix.GatherOptionsSnapshot_TestOnly(
		matrix.WithEpsilon(1e-6),
		matrix.WithNoValidateNaNInf(),
		matrix.WithDirected(),
		matrix.WithDisallowMulti(),
		matrix.WithAllowLoops(),
		matrix.WithWeighted(),
		matrix.WithMetricClosure(),
		matrix.WithEdgeThreshold(0.25),
		matrix.WithBinaryWeights(),
	)
	if got := o9.Eps; got != 1e-6 {
		t.Fatalf("eps: got %v, want 1e-6", got)
	}
	if got := o9.ValidateNaNInf; got {
		t.Fatalf("validateNaNInf: got %v, want false", got)
	}
	if got := o9.Directed; !got {
		t.Fatalf("directed: got %v, want true", got)
	}
	if got := o9.AllowMulti; got {
		t.Fatalf("allowMulti: got %v, want false", got)
	}
	if got := o9.AllowLoops; !got {
		t.Fatalf("allowLoops: got %v, want true", got)
	}
	if got := o9.Weighted; !got {
		t.Fatalf("weighted: got %v, want true", got)
	}
	if got := o9.MetricClose; !got {
		t.Fatalf("metricClose: got %v, want true", got)
	}
	if got := o9.EdgeThreshold; got != 0.25 {
		t.Fatalf("edgeThreshold: got %v, want 0.25", got)
	}
	if got := o9.KeepWeights; got {
		t.Fatalf("keepWeights: got %v, want false", got)
	}
}

// TestGatherOptionsDerived checks derived export flag 'undirected := !directed'.
func TestGatherOptions_DerivedUndirected(t *testing.T) {
	got := matrix.GatherOptionsSnapshot_TestOnly(matrix.WithDirected())
	if got.Undirected != false {
		t.Fatalf("derived undirected expected false, got %v", got.Undirected)
	}
	got2 := matrix.GatherOptionsSnapshot_TestOnly(matrix.WithUndirected())
	if got2.Undirected != true {
		t.Fatalf("derived undirected expected true, got %v", got2.Undirected)
	}
}

func TestValidateNaNInfToggles(t *testing.T) {
	o1 := matrix.GatherOptionsSnapshot_TestOnly()
	if o1.ValidateNaNInf != true {
		t.Fatalf("default validateNaNInf expected true, got %v", o1.ValidateNaNInf)
	}
	o2 := matrix.GatherOptionsSnapshot_TestOnly(matrix.WithNoValidateNaNInf())
	if o2.ValidateNaNInf != false {
		t.Fatalf("WithNoValidateNaNInf expected false, got %v", o2.ValidateNaNInf)
	}
	o3 := matrix.GatherOptionsSnapshot_TestOnly(matrix.WithValidateNaNInf())
	if o3.ValidateNaNInf != true {
		t.Fatalf("WithValidateNaNInf expected true, got %v", o3.ValidateNaNInf)
	}
	o4 := matrix.GatherOptionsSnapshot_TestOnly(matrix.DisableValidateNaNInf())
	if o4.ValidateNaNInf != false {
		t.Fatalf("DisableValidateNaNInf expected false, got %v", o4.ValidateNaNInf)
	}
}

// TestDeprecatedAlias verifies DisableValidateNaNInf equals WithNoValidateNaNInf.
func TestDeprecatedAlias(t *testing.T) {
	a := matrix.GatherOptionsSnapshot_TestOnly(matrix.WithNoValidateNaNInf())
	b := matrix.GatherOptionsSnapshot_TestOnly(matrix.DisableValidateNaNInf())
	if a.ValidateNaNInf || b.ValidateNaNInf {
		t.Fatalf("alias mismatch: both should flip validateNaNInf=false")
	}
}

// TestPanics validates parameter guards in WithEpsilon and WithEdgeThreshold.
func TestPanics(t *testing.T) {
	// WithEpsilon invalids
	ExpectPanic(t, func() { _ = matrix.GatherOptionsSnapshot_TestOnly(matrix.WithEpsilon(math.NaN())) })
	ExpectPanic(t, func() { _ = matrix.GatherOptionsSnapshot_TestOnly(matrix.WithEpsilon(-1)) })
	ExpectPanic(t, func() { _ = matrix.GatherOptionsSnapshot_TestOnly(matrix.WithEpsilon(math.Inf(1))) })
	ExpectPanic(t, func() { _ = matrix.GatherOptionsSnapshot_TestOnly(matrix.WithEpsilon(math.Inf(-1))) })

	// WithEdgeThreshold invalids
	ExpectPanic(t, func() { _ = matrix.GatherOptionsSnapshot_TestOnly(matrix.WithEdgeThreshold(math.NaN())) })
	ExpectPanic(t, func() { _ = matrix.GatherOptionsSnapshot_TestOnly(matrix.WithEdgeThreshold(math.Inf(1))) })
}
