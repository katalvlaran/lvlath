// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

package dijkstra_test

import (
	"math"
	"testing"

	"github.com/katalvlaran/lvlath/dijkstra"
)

// AI-HINTS (file):
//   - These tests validate option governance only.
//   - External tests must exercise canonical option assembly through the test-only bridge.
//   - Use errors.Is protocol checks through mustErrorIs; never compare error strings.
//   - Keep graph construction out of this file.
//   - Do not reintroduce panic-based option validation.

const (
	testFirstMaxDistance       = 7.0
	testSecondMaxDistance      = 3.0
	testFirstInfEdgeThreshold  = 10.0
	testSecondInfEdgeThreshold = 4.0
)

// TestDefaultOptions verifies that the canonical default option snapshot matches
// the documented runtime policy.
//
// Implementation:
//   - Stage 1: Read the default option snapshot through the external test bridge.
//   - Stage 2: Assert path tracking is disabled by default.
//   - Stage 3: Assert MaxDistance defaults to +Inf.
//   - Stage 4: Assert InfEdgeThreshold defaults to +Inf.
//
// Behavior highlights:
//   - This test locks the public default-policy contract.
//   - The test does not depend on algorithm execution.
//
// Inputs:
//   - None.
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal test failure on any contract mismatch.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Default policy is part of the package contract and must remain stable.
//
// AI-Hints:
//   - Keep +Inf as the canonical default for both distance cutoff and wall threshold.
func TestDefaultOptions(t *testing.T) {
	config := dijkstra.DefaultOptionsSnapshot_TestOnly()

	mustEqualBool(t, config.TrackPaths, false, "TrackPaths default mismatch: got=%v want=%v", config.TrackPaths, false)

	if !math.IsInf(config.MaxDistance, 1) {
		t.Fatalf("MaxDistance default mismatch: got=%v want=+Inf", config.MaxDistance)
	}
	if !math.IsInf(config.InfEdgeThreshold, 1) {
		t.Fatalf("InfEdgeThreshold default mismatch: got=%v want=+Inf", config.InfEdgeThreshold)
	}
}

// TestApplyOptions_LastWriterWins verifies that canonical option assembly applies
// options in call order and that the last writer wins for each mutable field.
//
// Implementation:
//   - Stage 1: Gather an option snapshot through the external test bridge.
//   - Stage 2: Apply repeated MaxDistance mutations.
//   - Stage 3: Apply repeated InfEdgeThreshold mutations.
//   - Stage 4: Enable path tracking.
//   - Stage 5: Assert that the final state reflects the last write for each field.
//
// Behavior highlights:
//   - This test locks the last-writer-wins contract.
//   - The test validates the canonical assembly path, not ad-hoc field mutation.
//
// Inputs:
//   - None.
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal test failure on any contract mismatch.
//   - Fatal test failure if option assembly unexpectedly fails.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(n), Space O(1), where n is the number of options in the test.
//
// Notes:
//   - Call order is part of the option-governance contract.
//
// AI-Hints:
//   - Do not weaken this test by checking only one of the repeated fields.
func TestApplyOptions_LastWriterWins(t *testing.T) {
	config, err := dijkstra.GatherOptionsSnapshot_TestOnly(
		dijkstra.WithMaxDistance(testFirstMaxDistance),
		dijkstra.WithMaxDistance(testSecondMaxDistance),
		dijkstra.WithInfEdgeThreshold(testFirstInfEdgeThreshold),
		dijkstra.WithInfEdgeThreshold(testSecondInfEdgeThreshold),
		dijkstra.WithPathTracking(),
	)
	if err != nil {
		t.Fatalf("GatherOptionsSnapshot_TestOnly(...) error: %v", err)
	}

	mustEqualBool(t, config.TrackPaths, true, "TrackPaths mismatch: got=%v want=%v", config.TrackPaths, true)
	mustEqualFloat64(t, config.MaxDistance, testSecondMaxDistance, "MaxDistance mismatch: got=%v want=%v", config.MaxDistance, testSecondMaxDistance)
	mustEqualFloat64(t, config.InfEdgeThreshold, testSecondInfEdgeThreshold, "InfEdgeThreshold mismatch: got=%v want=%v", config.InfEdgeThreshold, testSecondInfEdgeThreshold)
}

// TestApplyOptions_NilOption verifies that canonical option assembly rejects a nil option
// through the explicit sentinel protocol.
//
// Implementation:
//   - Stage 1: Construct a nil functional option.
//   - Stage 2: Pass it through the canonical option assembly bridge.
//   - Stage 3: Assert ErrNilOption through errors.Is.
//
// Behavior highlights:
//   - This test locks explicit nil-option rejection.
//   - Panic-based handling is forbidden.
//
// Inputs:
//   - None.
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal test failure if the sentinel protocol check fails.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Nil option rejection is part of the public option-governance contract.
//
// AI-Hints:
//   - Never replace this with panic-expectation logic.
func TestApplyOptions_NilOption(t *testing.T) {
	var nilOption dijkstra.Option

	_, err := dijkstra.GatherOptionsSnapshot_TestOnly(nilOption)
	mustErrorIs(t, err, dijkstra.ErrNilOption)
}

// TestWithMaxDistance_RejectsNegative verifies that finite negative distance cutoffs
// are rejected by the option contract.
//
// Implementation:
//   - Stage 1: Attempt to assemble a configuration with a negative MaxDistance.
//   - Stage 2: Assert ErrBadMaxDistance through the sentinel protocol.
//
// Behavior highlights:
//   - Negative finite cutoffs are invalid.
//   - The option must fail explicitly instead of clamping or panicking.
//
// Inputs:
//   - None.
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal test failure if the sentinel protocol check fails.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Zero remains valid and is not part of this rejection case.
//
// AI-Hints:
//   - Keep finite negative rejection separate from NaN and -Inf coverage.
func TestWithMaxDistance_RejectsNegative(t *testing.T) {
	_, err := dijkstra.GatherOptionsSnapshot_TestOnly(dijkstra.WithMaxDistance(-1))
	mustErrorIs(t, err, dijkstra.ErrBadMaxDistance)
}

// TestWithMaxDistance_RejectsNaN verifies that NaN is rejected by the MaxDistance
// option contract.
//
// Implementation:
//   - Stage 1: Attempt to assemble a configuration with MaxDistance set to NaN.
//   - Stage 2: Assert ErrBadMaxDistance through the sentinel protocol.
//
// Behavior highlights:
//   - NaN is invalid configuration data.
//   - The option must fail explicitly instead of silently normalizing.
//
// Inputs:
//   - None.
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal test failure if the sentinel protocol check fails.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - This test covers a distinct invalid numeric class from finite negative values.
//
// AI-Hints:
//   - Do not remove NaN coverage; it protects the numeric-policy contract.
func TestWithMaxDistance_RejectsNaN(t *testing.T) {
	_, err := dijkstra.GatherOptionsSnapshot_TestOnly(dijkstra.WithMaxDistance(math.NaN()))
	mustErrorIs(t, err, dijkstra.ErrBadMaxDistance)
}

// TestWithMaxDistance_AcceptsPositiveInfinity verifies that +Inf is accepted as the
// canonical “no cutoff” distance policy.
//
// Implementation:
//   - Stage 1: Assemble a configuration with MaxDistance set to +Inf.
//   - Stage 2: Assert successful option assembly.
//   - Stage 3: Assert the finalized value remains +Inf.
//
// Behavior highlights:
//   - +Inf is valid configuration, not an error.
//   - The option stores the accepted value exactly.
//
// Inputs:
//   - None.
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal test failure if option assembly unexpectedly fails.
//   - Fatal test failure if the stored value is not +Inf.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - This is the canonical no-cutoff mode.
//
// AI-Hints:
//   - Do not weaken this by checking only “no error”; the stored +Inf matters too.
func TestWithMaxDistance_AcceptsPositiveInfinity(t *testing.T) {
	config, err := dijkstra.GatherOptionsSnapshot_TestOnly(dijkstra.WithMaxDistance(math.Inf(1)))
	if err != nil {
		t.Fatalf("GatherOptionsSnapshot_TestOnly(...) error: %v", err)
	}

	if !math.IsInf(config.MaxDistance, 1) {
		t.Fatalf("MaxDistance mismatch: got=%v want=+Inf", config.MaxDistance)
	}
}

// TestWithMaxDistance_RejectsNegativeInfinity verifies that negative infinity is rejected
// explicitly by the MaxDistance option contract.
//
// Implementation:
//   - Stage 1: Attempt to assemble a configuration with MaxDistance set to -Inf.
//   - Stage 2: Assert ErrBadMaxDistance through the sentinel protocol.
//
// Behavior highlights:
//   - -Inf is invalid configuration data.
//   - This test covers a distinct explicit branch in the production code.
//
// Inputs:
//   - None.
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal test failure if the sentinel protocol check fails.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - This is a live validation branch and should remain covered.
//
// AI-Hints:
//   - Keep negative-infinity coverage separate from finite negative and NaN coverage.
func TestWithMaxDistance_RejectsNegativeInfinity(t *testing.T) {
	_, err := dijkstra.GatherOptionsSnapshot_TestOnly(dijkstra.WithMaxDistance(math.Inf(-1)))
	mustErrorIs(t, err, dijkstra.ErrBadMaxDistance)
}

// TestWithInfEdgeThreshold_RejectsZero verifies that a zero wall threshold is rejected
// because it would incorrectly classify zero-weight edges as impassable.
//
// Implementation:
//   - Stage 1: Attempt to assemble a configuration with InfEdgeThreshold set to zero.
//   - Stage 2: Assert ErrBadInfEdgeThreshold through the sentinel protocol.
//
// Behavior highlights:
//   - Zero is invalid for the edge-wall threshold.
//   - The option must fail explicitly instead of silently adjusting the value.
//
// Inputs:
//   - None.
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal test failure if the sentinel protocol check fails.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Positive finite thresholds remain valid and are not part of this rejection case.
//
// AI-Hints:
//   - Keep zero-threshold rejection explicit; it protects zero-weight edge semantics.
func TestWithInfEdgeThreshold_RejectsZero(t *testing.T) {
	_, err := dijkstra.GatherOptionsSnapshot_TestOnly(dijkstra.WithInfEdgeThreshold(0))
	mustErrorIs(t, err, dijkstra.ErrBadInfEdgeThreshold)
}

// TestWithInfEdgeThreshold_RejectsNaN verifies that NaN is rejected by the edge-wall
// threshold option contract.
//
// Implementation:
//   - Stage 1: Attempt to assemble a configuration with InfEdgeThreshold set to NaN.
//   - Stage 2: Assert ErrBadInfEdgeThreshold through the sentinel protocol.
//
// Behavior highlights:
//   - NaN is invalid configuration data.
//   - The option must fail explicitly instead of silently normalizing.
//
// Inputs:
//   - None.
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal test failure if the sentinel protocol check fails.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - This test covers a distinct invalid numeric class from zero and -Inf.
//
// AI-Hints:
//   - Do not remove NaN coverage; it protects the threshold numeric contract.
func TestWithInfEdgeThreshold_RejectsNaN(t *testing.T) {
	_, err := dijkstra.GatherOptionsSnapshot_TestOnly(dijkstra.WithInfEdgeThreshold(math.NaN()))
	mustErrorIs(t, err, dijkstra.ErrBadInfEdgeThreshold)
}

// TestWithInfEdgeThreshold_AcceptsPositiveInfinity verifies that +Inf is accepted as the
// canonical “no finite edge is blocked by threshold” mode.
//
// Implementation:
//   - Stage 1: Assemble a configuration with InfEdgeThreshold set to +Inf.
//   - Stage 2: Assert successful option assembly.
//   - Stage 3: Assert the finalized value remains +Inf.
//
// Behavior highlights:
//   - +Inf is valid configuration, not an error.
//   - The option stores the accepted value exactly.
//
// Inputs:
//   - None.
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal test failure if option assembly unexpectedly fails.
//   - Fatal test failure if the stored value is not +Inf.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - This is the canonical no-wall-threshold mode.
//
// AI-Hints:
//   - Do not weaken this by checking only “no error”; the stored +Inf matters too.
func TestWithInfEdgeThreshold_AcceptsPositiveInfinity(t *testing.T) {
	config, err := dijkstra.GatherOptionsSnapshot_TestOnly(dijkstra.WithInfEdgeThreshold(math.Inf(1)))
	if err != nil {
		t.Fatalf("GatherOptionsSnapshot_TestOnly(...) error: %v", err)
	}

	if !math.IsInf(config.InfEdgeThreshold, 1) {
		t.Fatalf("InfEdgeThreshold mismatch: got=%v want=+Inf", config.InfEdgeThreshold)
	}
}

// TestWithInfEdgeThreshold_RejectsNegativeInfinity verifies that negative infinity is rejected
// explicitly by the edge-wall threshold option contract.
//
// Implementation:
//   - Stage 1: Attempt to assemble a configuration with InfEdgeThreshold set to -Inf.
//   - Stage 2: Assert ErrBadInfEdgeThreshold through the sentinel protocol.
//
// Behavior highlights:
//   - -Inf is invalid configuration data.
//   - This test covers a distinct explicit branch in the production code.
//
// Inputs:
//   - None.
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal test failure if the sentinel protocol check fails.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - This is a live validation branch and should remain covered.
//
// AI-Hints:
//   - Keep negative-infinity coverage separate from zero and NaN coverage.
func TestWithInfEdgeThreshold_RejectsNegativeInfinity(t *testing.T) {
	_, err := dijkstra.GatherOptionsSnapshot_TestOnly(dijkstra.WithInfEdgeThreshold(math.Inf(-1)))
	mustErrorIs(t, err, dijkstra.ErrBadInfEdgeThreshold)
}

// TestWithPathTracking verifies that the path-tracking option enables predecessor
// tracking without perturbing the remaining default policy.
//
// Implementation:
//   - Stage 1: Assemble a configuration with WithPathTracking.
//   - Stage 2: Assert TrackPaths becomes true.
//   - Stage 3: Assert remaining policy fields stay at their documented defaults.
//
// Behavior highlights:
//   - Path tracking is an explicit contract-level request.
//   - The option must not silently alter distance or threshold policy.
//
// Inputs:
//   - None.
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal test failure if option assembly unexpectedly fails.
//   - Fatal test failure on any policy mismatch.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - This test isolates WithPathTracking from algorithm execution.
//
// AI-Hints:
//   - Keep path-tracking enablement narrow; do not let it mutate unrelated fields.
func TestWithPathTracking(t *testing.T) {
	config, err := dijkstra.GatherOptionsSnapshot_TestOnly(dijkstra.WithPathTracking())
	if err != nil {
		t.Fatalf("GatherOptionsSnapshot_TestOnly(...) error: %v", err)
	}

	mustEqualBool(t, config.TrackPaths, true, "TrackPaths mismatch: got=%v want=%v", config.TrackPaths, true)

	if !math.IsInf(config.MaxDistance, 1) {
		t.Fatalf("MaxDistance mismatch: got=%v want=+Inf", config.MaxDistance)
	}
	if !math.IsInf(config.InfEdgeThreshold, 1) {
		t.Fatalf("InfEdgeThreshold mismatch: got=%v want=+Inf", config.InfEdgeThreshold)
	}
}
