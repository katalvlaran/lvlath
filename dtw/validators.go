// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package dtw validates scalar sequence inputs and local costs before DTW consumes them.
// This file keeps numeric policy centralized and separate from the recurrence kernel.
package dtw

import (
	"fmt"
	"math"
)

// validateSequences checks scalar DTW input sequences.
//
// Implementation:
//   - Stage 1: reject nil slices.
//   - Stage 2: reject empty slices.
//   - Stage 3: optionally reject NaN and ±Inf samples.
//
// Behavior highlights:
//   - nil and empty are different failure classes.
//   - validateFinite=false only skips input-sample finite checks.
//   - Local costs remain finite-validated by validateLocalCost.
//
// Inputs:
//   - a: first scalar sequence.
//   - b: second scalar sequence.
//   - validateFinite: true to reject non-finite samples.
//
// Returns:
//   - error: nil when both sequences satisfy the configured input policy.
//
// Errors:
//   - ErrNilInput when a or b is nil.
//   - ErrEmptyInput when a or b has length zero.
//   - ErrNaNInf when validateFinite is true and any sample is NaN or ±Inf.
//
// Determinism:
//   - Scans a from low to high index, then b from low to high index.
//   - Reports the first invalid sample in deterministic order.
//
// Complexity:
//   - Time O(n+m), Space O(1).
//
// Notes:
//   - The function does not copy or mutate input slices.
//
// AI-Hints:
//   - Do not merge nil and empty inputs; they represent different caller mistakes.
//   - Do not skip local-cost validation even when validateFinite is false.
func validateSequences(a, b []float64, validateFinite bool) error {
	if a == nil || b == nil {
		return ErrNilInput
	}

	if len(a) == 0 || len(b) == 0 {
		return ErrEmptyInput
	}

	if !validateFinite {
		return nil
	}

	for idx, value := range a {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return fmt.Errorf("dtw: a[%d]=%v: %w", idx, value, ErrNaNInf)
		}
	}

	for idx, value := range b {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return fmt.Errorf("dtw: b[%d]=%v: %w", idx, value, ErrNaNInf)
		}
	}

	return nil
}

// validateLocalCost checks one local alignment cost.
//
// Implementation:
//   - Stage 1: reject NaN and ±Inf.
//   - Stage 2: reject negative costs.
//   - Stage 3: allow zero and positive finite costs.
//
// Behavior highlights:
//   - DTW requires local costs to be finite and non-negative.
//   - This validation is mandatory even when input finite validation is disabled.
//
// Inputs:
//   - cost: local scalar cost returned by CostFunc.
//
// Returns:
//   - error: nil when cost is finite and non-negative.
//
// Errors:
//   - ErrNaNInf when cost is NaN or ±Inf.
//   - ErrNegativeCost when cost < 0.
//
// Determinism:
//   - Pure numeric check.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// AI-Hints:
//   - Do not allow +Inf as a local cost in scalar DTW; +Inf is reserved for unreachable DP states.
func validateLocalCost(cost float64) error {
	if math.IsNaN(cost) || math.IsInf(cost, 0) {
		return ErrNaNInf
	}

	if cost < 0 {
		return ErrNegativeCost
	}

	return nil
}

// AbsoluteCost computes |ai-bj|.
//
// Implementation:
//   - Stage 1: subtract bj from ai.
//   - Stage 2: return the absolute value.
//
// Behavior highlights:
//   - This is the default scalar DTW local cost.
//   - It is symmetric for finite scalar inputs.
//
// Inputs:
//   - ai: sample from sequence A.
//   - bj: sample from sequence B.
//
// Returns:
//   - float64: finite non-negative local cost when inputs are finite.
//
// Errors:
//   - None directly.
//
// Determinism:
//   - Pure arithmetic.
//
// Complexity:
//   - Time O(1), Space O(1).
func AbsoluteCost(ai, bj float64) (float64, error) {
	return math.Abs(ai - bj), nil
}

// SquaredCost computes (ai-bj)^2.
//
// Implementation:
//   - Stage 1: compute the scalar difference.
//   - Stage 2: square the difference.
//
// Behavior highlights:
//   - SquaredCost penalizes large deviations more strongly than AbsoluteCost.
//   - It is symmetric for finite scalar inputs.
//
// Inputs:
//   - ai: sample from sequence A.
//   - bj: sample from sequence B.
//
// Returns:
//   - float64: finite non-negative local cost when inputs are finite.
//
// Errors:
//   - None directly.
//
// Determinism:
//   - Pure arithmetic.
//
// Complexity:
//   - Time O(1), Space O(1).
func SquaredCost(ai, bj float64) (float64, error) {
	diff := ai - bj
	return diff * diff, nil
}
