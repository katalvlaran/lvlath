// SPDX-License-Identifier: MIT
package matrix_test

import (
	"math"
	"testing"

	"github.com/katalvlaran/lvlath/matrix"
)

// 1) TestDefaultOptions_Documented verifies that NewMatrixOptions() equals documented defaults.
func TestDefaultOptions_Documented(t *testing.T) {
	o := matrix.NewMatrixOptionsSnapshot_TestOnly()

	// numeric
	if o.Eps != matrix.DefaultEpsilon {
		t.Fatalf("eps default mismatch: got %v, want %v", o.Eps, matrix.DefaultEpsilon)
	}
	if o.ValidateNaNInf != matrix.DefaultValidateNaNInf {
		t.Fatalf("validateNaNInf default mismatch: got %v, want %v", o.ValidateNaNInf, matrix.DefaultValidateNaNInf)
	}
	if o.AllowInfDistances != matrix.DefaultAllowInfDistances {
		t.Fatalf("allowInfDistances default mismatch: got %v, want %v", o.AllowInfDistances, matrix.DefaultAllowInfDistances)
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
	if o.BinaryWeights != matrix.DefaultBinaryWeights {
		t.Fatalf("binaryWeights default mismatch: got %v, want %v", o.BinaryWeights, matrix.DefaultBinaryWeights)
	}
}

// 2) TestNewMatrixOptions_OrderAndIdempotence ensures each Option toggles exactly its intended field.
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

// 3) allowMulti last-writer-wins.
func TestNewMatrixOptions_LastWriterWins_AllowMulti(t *testing.T) {
	o1 := matrix.GatherOptionsSnapshot_TestOnly(matrix.WithAllowMulti(), matrix.WithDisallowMulti())
	if o1.AllowMulti {
		t.Fatalf("allowMulti last-writer-wins failed: got %v, want false", o1.AllowMulti)
	}

	o2 := matrix.GatherOptionsSnapshot_TestOnly(matrix.WithDisallowMulti(), matrix.WithAllowMulti())
	if !o2.AllowMulti {
		t.Fatalf("allowMulti last-writer-wins failed: got %v, want true", o2.AllowMulti)
	}
}

// 4) allowLoops last-writer-wins.
func TestNewMatrixOptions_LastWriterWins_AllowLoops(t *testing.T) {
	o1 := matrix.GatherOptionsSnapshot_TestOnly(matrix.WithAllowLoops(), matrix.WithDisallowLoops())
	if o1.AllowLoops {
		t.Fatalf("allowLoops last-writer-wins failed: got %v, want false", o1.AllowLoops)
	}

	o2 := matrix.GatherOptionsSnapshot_TestOnly(matrix.WithDisallowLoops(), matrix.WithAllowLoops())
	if !o2.AllowLoops {
		t.Fatalf("allowLoops last-writer-wins failed: got %v, want true", o2.AllowLoops)
	}
}

// 5) weighted last-writer-wins.
func TestNewMatrixOptions_LastWriterWins_Weighted(t *testing.T) {
	o1 := matrix.GatherOptionsSnapshot_TestOnly(matrix.WithUnweighted(), matrix.WithWeighted())
	if !o1.Weighted {
		t.Fatalf("weighted last-writer-wins failed: got %v, want true", o1.Weighted)
	}

	o2 := matrix.GatherOptionsSnapshot_TestOnly(matrix.WithWeighted(), matrix.WithUnweighted())
	if o2.Weighted {
		t.Fatalf("weighted last-writer-wins failed: got %v, want false", o2.Weighted)
	}
}

// 6) metricClose must imply allowInfDistances (distance-policy invariant).
func TestMetricClosure_EnablesAllowInfDistances(t *testing.T) {
	o := matrix.GatherOptionsSnapshot_TestOnly(matrix.WithMetricClosure())
	if !o.MetricClose {
		t.Fatalf("metricClose expected true, got %v", o.MetricClose)
	}
	if !o.AllowInfDistances {
		t.Fatalf("metricClose must imply allowInfDistances=true, got %v", o.AllowInfDistances)
	}
}

// 7) export weight mode must be internally consistent and last-writer-wins.
func TestExportWeightMode_LastWriterWins(t *testing.T) {
	o1 := matrix.GatherOptionsSnapshot_TestOnly(matrix.WithKeepWeights())
	if !o1.KeepWeights || o1.BinaryWeights {
		t.Fatalf("keepWeights mode mismatch: keep=%v binary=%v", o1.KeepWeights, o1.BinaryWeights)
	}

	o2 := matrix.GatherOptionsSnapshot_TestOnly(matrix.WithBinaryWeights())
	if o2.KeepWeights || !o2.BinaryWeights {
		t.Fatalf("binaryWeights mode mismatch: keep=%v binary=%v", o2.KeepWeights, o2.BinaryWeights)
	}

	o3 := matrix.GatherOptionsSnapshot_TestOnly(matrix.WithKeepWeights(), matrix.WithBinaryWeights())
	if o3.KeepWeights || !o3.BinaryWeights {
		t.Fatalf("last-writer-wins mismatch: keep=%v binary=%v", o3.KeepWeights, o3.BinaryWeights)
	}

	o4 := matrix.GatherOptionsSnapshot_TestOnly(matrix.WithBinaryWeights(), matrix.WithKeepWeights())
	if !o4.KeepWeights || o4.BinaryWeights {
		t.Fatalf("last-writer-wins mismatch: keep=%v binary=%v", o4.KeepWeights, o4.BinaryWeights)
	}
}

// 8) epsilon setter must store the value exactly and be idempotent.
func TestWithEpsilon_SetsValue(t *testing.T) {
	o := matrix.GatherOptionsSnapshot_TestOnly(matrix.WithEpsilon(1e-6), matrix.WithEpsilon(1e-6))
	if o.Eps != 1e-6 {
		t.Fatalf("eps mismatch: got %v, want %v", o.Eps, 1e-6)
	}
}

// 9) edgeThreshold setter must store the value exactly and be idempotent.
func TestWithEdgeThreshold_SetsValue(t *testing.T) {
	o := matrix.GatherOptionsSnapshot_TestOnly(matrix.WithEdgeThreshold(0.25), matrix.WithEdgeThreshold(0.25))
	if o.EdgeThreshold != 0.25 {
		t.Fatalf("edgeThreshold mismatch: got %v, want %v", o.EdgeThreshold, 0.25)
	}
}

// 10) validateNaNInf toggles + deprecated alias must match behavior.
func TestValidateNaNInfToggles_AndAlias(t *testing.T) {
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

// 11) allowInfDistances must be togglable and last-writer-wins.
func TestAllowInfDistances_ToggleAndOrder(t *testing.T) {
	o1 := matrix.GatherOptionsSnapshot_TestOnly()
	if o1.AllowInfDistances {
		t.Fatalf("default allowInfDistances expected false, got %v", o1.AllowInfDistances)
	}

	o2 := matrix.GatherOptionsSnapshot_TestOnly(matrix.WithAllowInfDistances())
	if !o2.AllowInfDistances {
		t.Fatalf("WithAllowInfDistances expected true, got %v", o2.AllowInfDistances)
	}

	o3 := matrix.GatherOptionsSnapshot_TestOnly(matrix.WithAllowInfDistances(), matrix.WithDisallowInfDistances())
	if o3.AllowInfDistances {
		t.Fatalf("last-writer-wins expected false, got %v", o3.AllowInfDistances)
	}
}

// 12) WithEpsilon must panic with stable message on invalid inputs.
func TestPanics_WithEpsilon_Message(t *testing.T) {
	ExpectPanicMessage(t, matrix.PanicEpsilonInvalid_TestOnly, func() { _ = matrix.WithEpsilon(math.NaN()) })
	ExpectPanicMessage(t, matrix.PanicEpsilonInvalid_TestOnly, func() { _ = matrix.WithEpsilon(-1) })
	ExpectPanicMessage(t, matrix.PanicEpsilonInvalid_TestOnly, func() { _ = matrix.WithEpsilon(math.Inf(1)) })
	ExpectPanicMessage(t, matrix.PanicEpsilonInvalid_TestOnly, func() { _ = matrix.WithEpsilon(math.Inf(-1)) })
}

// 13) WithEdgeThreshold must panic with stable message on non-finite inputs.
func TestPanics_WithEdgeThreshold_Message(t *testing.T) {
	ExpectPanicMessage(t, matrix.PanicEdgeThresholdInvalid_TestOnly, func() { _ = matrix.WithEdgeThreshold(math.NaN()) })
	ExpectPanicMessage(t, matrix.PanicEdgeThresholdInvalid_TestOnly, func() { _ = matrix.WithEdgeThreshold(math.Inf(1)) })
	ExpectPanicMessage(t, matrix.PanicEdgeThresholdInvalid_TestOnly, func() { _ = matrix.WithEdgeThreshold(math.Inf(-1)) })
}

// 14) TestDeprecatedAlias verifies DisableValidateNaNInf equals WithNoValidateNaNInf.
func TestDeprecatedAlias(t *testing.T) {
	a := matrix.GatherOptionsSnapshot_TestOnly(matrix.WithNoValidateNaNInf())
	b := matrix.GatherOptionsSnapshot_TestOnly(matrix.DisableValidateNaNInf())
	if a.ValidateNaNInf || b.ValidateNaNInf {
		t.Fatalf("alias mismatch: both should flip validateNaNInf=false")
	}
}

// 15) TestPanics validates parameter guards in WithEpsilon and WithEdgeThreshold.
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
