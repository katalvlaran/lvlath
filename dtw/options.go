// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package dtw defines Dynamic Time Warping policy assembly for scalar sequence alignment.
// This file owns option defaults, explicit policy switches, and legacy option adaptation.
package dtw

import (
	"errors"
	"fmt"
	"math"
)

const (
	// DefaultWindow disables the Sakoe-Chiba band.
	DefaultWindow = -1

	// DefaultSlopePenalty keeps vertical and horizontal warping steps free.
	DefaultSlopePenalty = 0.0

	// DefaultValidateFinite rejects NaN and ±Inf input samples before the DP kernel runs.
	DefaultValidateFinite = true

	// DefaultReturnPath disables path reconstruction.
	DefaultReturnPath = false

	// DefaultReturnAccumulated disables publishing the accumulated-cost DP matrix.
	DefaultReturnAccumulated = false

	// DefaultReturnLocalCost disables publishing the local-cost matrix.
	DefaultReturnLocalCost = false

	// DefaultMemoryMode keeps canonical Align in rolling-row distance-only mode by default.
	DefaultMemoryMode = TwoRows

	// DefaultTieEpsilon is used for deterministic backtracking comparisons.
	DefaultTieEpsilon = 1e-9
)

// Option updates DTW runtime policy during canonical option assembly.
//
// Implementation:
//   - Stage 1: applyOptions starts from deterministic defaults.
//   - Stage 2: each Option mutates the private options state.
//   - Stage 3: finalizeOptions validates derived invariants before allocations.
//
// Behavior highlights:
//   - Options are explicit contract switches.
//   - nil Option values are rejected with ErrNilOption.
//   - Invalid values return sentinel-classified errors; no panic is used.
//
// Inputs:
//   - *options: private mutable assembly state.
//
// Returns:
//   - error: nil on success or a sentinel-classified validation error.
//
// Errors:
//   - ErrNilOption is produced by applyOptions, not by the Option itself.
//   - Individual options return ErrBadInput, ErrInvalidWindow, ErrInvalidPenalty,
//     or ErrNilCostFunc depending on the invalid value.
//
// Determinism:
//   - Options are applied in caller-provided order.
//   - Last writer wins unless a future option explicitly documents otherwise.
//
// Complexity:
//   - Time O(1) per option, Space O(1).
//
// Notes:
//   - Option callbacks must not allocate DP buffers or execute algorithmic work.
//
// AI-Hints:
//   - Do not introduce panic-based option validation.
//   - Do not hide mathematical mode changes behind implicit fallbacks.
type Option func(*options) error

// options stores finalized DTW runtime policy.
//
// Implementation:
//   - Stage 1: defaultOptionsInternal creates the baseline policy.
//   - Stage 2: Option callbacks update this structure.
//   - Stage 3: finalizeOptions validates all fields and derived invariants.
//
// Behavior highlights:
//   - The structure is private to prevent callers from bypassing validation.
//   - Field names intentionally mirror Result policy fields where relevant.
//
// Determinism:
//   - Once finalized, options is immutable by convention for one kernel call.
//
// Complexity:
//   - Space O(1).
//
// AI-Hints:
//   - Do not export this type; public mutation belongs to Option functions.
type options struct {
	// window is the Sakoe-Chiba band radius.
	// -1 disables the band; values >= 0 enforce |i-j| <= window.
	window int

	// slopePenalty is added to vertical and horizontal moves.
	// It must be finite and non-negative.
	slopePenalty float64

	// returnPath requests deterministic backtracking after the distance is known.
	// Canonical Align auto-selects FullMatrix for this mode.
	returnPath bool

	// returnAccumulated requests publishing the accumulated-cost matrix in Result.
	// It also forces FullMatrix storage because the full DP table must exist.
	returnAccumulated bool

	// returnLocalCost requests publishing the local scalar cost matrix in Result.
	// It does not by itself require FullMatrix accumulated storage.
	returnLocalCost bool

	// memoryMode controls accumulated-state storage.
	// TwoRows/NoMemory use rolling rows; FullMatrix stores the DP table.
	memoryMode MemoryMode

	// validateFinite controls input-sequence finite validation.
	// Local costs remain finite-validated even when this is false.
	validateFinite bool

	// tieEpsilon is used only by path reconstruction comparisons.
	// It does not alter the forward DP recurrence.
	tieEpsilon float64

	// costFunc computes the finite non-negative local cost for each admissible cell.
	costFunc CostFunc
}

// defaultOptionsInternal returns the canonical DTW default policy.
//
// Implementation:
//   - Stage 1: set domain defaults.
//   - Stage 2: attach AbsoluteCost as the scalar local-cost function.
//   - Stage 3: return by value so callers receive an isolated assembly state.
//
// Behavior highlights:
//   - The default window is unconstrained.
//   - The default kernel is distance-only rolling rows.
//   - The default numeric policy rejects non-finite input samples.
//
// Returns:
//   - options: private mutable policy state for applyOptions.
//
// Errors:
//   - None.
//
// Determinism:
//   - Always returns the same policy values.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// AI-Hints:
//   - Keep defaults boring and explicit; do not add hidden preprocessing here.
func defaultOptionsInternal() options {
	return options{
		window:            DefaultWindow,
		slopePenalty:      DefaultSlopePenalty,
		returnPath:        DefaultReturnPath,
		returnAccumulated: DefaultReturnAccumulated,
		returnLocalCost:   DefaultReturnLocalCost,
		memoryMode:        DefaultMemoryMode,
		validateFinite:    DefaultValidateFinite,
		tieEpsilon:        DefaultTieEpsilon,
		costFunc:          AbsoluteCost,
	}
}

// applyOptions assembles canonical DTW options.
//
// Implementation:
//   - Stage 1: start from defaultOptionsInternal.
//   - Stage 2: apply caller-provided Option callbacks in order.
//   - Stage 3: finalize and validate the complete policy before any DP allocation.
//
// Behavior highlights:
//   - nil options are rejected.
//   - Last writer wins for ordinary fields.
//   - Derived invariants are centralized in finalizeOptions.
//
// Inputs:
//   - user: zero or more explicit DTW policy options.
//
// Returns:
//   - options: finalized private runtime policy.
//   - error: sentinel-classified option assembly failure.
//
// Errors:
//   - ErrNilOption when any option is nil.
//   - ErrBadInput / ErrInvalidWindow / ErrInvalidPenalty / ErrNilCostFunc from validation.
//
// Determinism:
//   - Option application order is exactly the input order.
//
// Complexity:
//   - Time O(k), Space O(1), where k=len(user).
//
// AI-Hints:
//   - Never allocate DP rows here; option assembly must remain cheap and fail-fast.
func applyOptions(user ...Option) (options, error) {
	cfg := defaultOptionsInternal()

	for idx, opt := range user {
		if opt == nil {
			return options{}, fmt.Errorf("dtw: option %d: %w", idx, ErrNilOption)
		}
		if err := opt(&cfg); err != nil {
			return options{}, err
		}
	}

	if err := finalizeOptions(&cfg); err != nil {
		return options{}, err
	}

	return cfg, nil
}

// finalizeOptions validates complete DTW runtime policy.
//
// Implementation:
//   - Stage 1: validate cfg itself.
//   - Stage 2: validate domain options: window, penalty, memory mode.
//   - Stage 3: validate numeric/backtracking options.
//   - Stage 4: derive storage requirements for path and accumulated matrix modes.
//
// Behavior highlights:
//   - Canonical Align auto-selects FullMatrix when path or accumulated output is requested.
//   - Legacy Options are checked before this function when legacy-only restrictions apply.
//   - No hidden mathematical fallback is introduced.
//
// Inputs:
//   - cfg: private option state assembled by applyOptions or optionsFromLegacy.
//
// Returns:
//   - error: nil when cfg is usable by the kernel.
//
// Errors:
//   - ErrNilOptions when cfg is nil.
//   - ErrBadInput joined with ErrInvalidWindow for invalid window values.
//   - ErrBadInput joined with ErrInvalidPenalty for invalid slope penalty.
//   - ErrBadInput for invalid memory mode or invalid tie epsilon.
//   - ErrNilCostFunc for nil local-cost callback.
//
// Determinism:
//   - Pure validation and deterministic derived-field assignment.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// AI-Hints:
//   - Do not move ReturnPath memory upgrade into the kernel; policy belongs here.
func finalizeOptions(cfg *options) error {
	if cfg == nil {
		return ErrNilOptions
	}

	if cfg.window < -1 {
		return errors.Join(ErrBadInput, ErrInvalidWindow)
	}

	if math.IsNaN(cfg.slopePenalty) || math.IsInf(cfg.slopePenalty, 0) || cfg.slopePenalty < 0 {
		return errors.Join(ErrBadInput, ErrInvalidPenalty)
	}

	switch cfg.memoryMode {
	case NoMemory, TwoRows, FullMatrix:
	default:
		return ErrBadInput
	}

	if math.IsNaN(cfg.tieEpsilon) || math.IsInf(cfg.tieEpsilon, 0) || cfg.tieEpsilon < 0 {
		return ErrBadInput
	}

	if cfg.costFunc == nil {
		return ErrNilCostFunc
	}

	if cfg.returnPath || cfg.returnAccumulated {
		cfg.memoryMode = FullMatrix
	}

	return nil
}

// WithWindow sets the Sakoe-Chiba band radius.
//
// Implementation:
//   - Stage 1: validate the radius.
//   - Stage 2: store it in the private policy state.
//
// Behavior highlights:
//   - w == -1 disables the band.
//   - w >= 0 admits only cells satisfying |i-j| <= w.
//   - w == 0 is strict diagonal alignment.
//
// Inputs:
//   - w: Sakoe-Chiba radius.
//
// Returns:
//   - Option: policy callback for Align.
//
// Errors:
//   - ErrBadInput joined with ErrInvalidWindow when w < -1.
//
// Determinism:
//   - Does not depend on input data.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// AI-Hints:
//   - Do not document w <= 0 as unconstrained.
func WithWindow(w int) Option {
	return func(o *options) error {
		if w < -1 {
			return errors.Join(ErrBadInput, ErrInvalidWindow)
		}

		o.window = w
		return nil
	}
}

// WithSlopePenalty sets the penalty for vertical and horizontal warping steps.
//
// Implementation:
//   - Stage 1: reject NaN, Inf, and negative values.
//   - Stage 2: store the penalty in the private policy state.
//
// Behavior highlights:
//   - The penalty is not added to diagonal moves.
//   - Larger values bias the path toward diagonal alignment.
//
// Inputs:
//   - p: finite non-negative penalty.
//
// Returns:
//   - Option: policy callback for Align.
//
// Errors:
//   - ErrBadInput joined with ErrInvalidPenalty for NaN, Inf, or negative values.
//
// Determinism:
//   - Does not depend on input data.
//
// Complexity:
//   - Time O(1), Space O(1).
func WithSlopePenalty(p float64) Option {
	return func(o *options) error {
		if math.IsNaN(p) || math.IsInf(p, 0) || p < 0 {
			return errors.Join(ErrBadInput, ErrInvalidPenalty)
		}

		o.slopePenalty = p
		return nil
	}
}

// WithReturnPath controls deterministic path reconstruction.
//
// Implementation:
//   - Stage 1: store the caller request.
//   - Stage 2: finalizeOptions upgrades storage to FullMatrix when enabled.
//
// Behavior highlights:
//   - Path reconstruction uses the accumulated-cost matrix.
//   - No-path cases return ErrNoPath before backtracking starts.
//
// Inputs:
//   - on: true to request path reconstruction.
//
// Returns:
//   - Option: policy callback for Align.
//
// Errors:
//   - None directly.
//
// Determinism:
//   - Backtracking tie-break is diagonal, then vertical/up, then horizontal/left.
//
// Complexity:
//   - Option setup O(1); execution uses O(n*m) memory when enabled.
//
// AI-Hints:
//   - Do not attempt to backtrack from rolling rows.
func WithReturnPath(on bool) Option {
	return func(o *options) error {
		o.returnPath = on
		return nil
	}
}

// WithReturnAccumulated controls publication of the accumulated DP matrix.
//
// Implementation:
//   - Stage 1: store the caller request.
//   - Stage 2: finalizeOptions upgrades storage to FullMatrix when enabled.
//
// Behavior highlights:
//   - The returned Result.Accumulated matrix is detached.
//   - Internal FullMatrix storage may exist without publication when only path is requested.
//
// Inputs:
//   - on: true to publish Result.Accumulated.
//
// Returns:
//   - Option: policy callback for Align.
//
// Errors:
//   - None directly.
//
// Complexity:
//   - Option setup O(1); execution uses O(n*m) memory when enabled.
func WithReturnAccumulated(on bool) Option {
	return func(o *options) error {
		o.returnAccumulated = on
		return nil
	}
}

// WithReturnLocalCost controls publication of the local-cost matrix.
//
// Implementation:
//   - Stage 1: store the caller request.
//   - Stage 2: the kernel writes each admissible local cost into matrix.Dense.
//
// Behavior highlights:
//   - This mode does not require accumulated FullMatrix storage.
//   - Cells outside the window remain the matrix default value in this P0 scalar mode.
//
// Inputs:
//   - on: true to publish Result.LocalCost.
//
// Returns:
//   - Option: policy callback for Align.
//
// Errors:
//   - None directly.
//
// Complexity:
//   - Option setup O(1); execution uses O(n*m) extra memory when enabled.
func WithReturnLocalCost(on bool) Option {
	return func(o *options) error {
		o.returnLocalCost = on
		return nil
	}
}

// WithMemoryMode sets the accumulated-cost storage mode.
//
// Implementation:
//   - Stage 1: validate the memory mode enum.
//   - Stage 2: store the requested mode in the private policy state.
//
// Behavior highlights:
//   - NoMemory is accepted as a legacy alias for rolling-row distance mode.
//   - FullMatrix stores accumulated costs even if they are not published.
//   - WithReturnPath and WithReturnAccumulated override this to FullMatrix.
//
// Inputs:
//   - mode: NoMemory, TwoRows, or FullMatrix.
//
// Returns:
//   - Option: policy callback for Align.
//
// Errors:
//   - ErrBadInput for unknown modes.
//
// Determinism:
//   - Storage mode does not change the recurrence or distance value.
//
// Complexity:
//   - Time O(1), Space O(1).
func WithMemoryMode(mode MemoryMode) Option {
	return func(o *options) error {
		switch mode {
		case NoMemory, TwoRows, FullMatrix:
			o.memoryMode = mode
			return nil
		default:
			return ErrBadInput
		}
	}
}

// WithValidateFinite controls input finite-value validation.
//
// Implementation:
//   - Stage 1: store the validation flag.
//   - Stage 2: validateSequences applies the flag before DP execution.
//
// Behavior highlights:
//   - true rejects NaN and ±Inf in input sequences.
//   - false skips sequence finite checks but does not weaken local-cost validation.
//
// Inputs:
//   - on: true for strict finite input validation.
//
// Returns:
//   - Option: policy callback for Align.
//
// Errors:
//   - None directly.
//
// Complexity:
//   - Option setup O(1); validation cost is O(n+m) when enabled.
//
// AI-Hints:
//   - Keep enabled for production math; disabling it is only for controlled experiments.
func WithValidateFinite(on bool) Option {
	return func(o *options) error {
		o.validateFinite = on
		return nil
	}
}

// WithCostFunc sets the scalar local-cost function.
//
// Implementation:
//   - Stage 1: reject nil callbacks.
//   - Stage 2: store the callback.
//   - Stage 3: the kernel validates every returned cost.
//
// Behavior highlights:
//   - The callback must be deterministic for deterministic results.
//   - Returned costs must be finite and non-negative.
//
// Inputs:
//   - fn: local-cost callback.
//
// Returns:
//   - Option: policy callback for Align.
//
// Errors:
//   - ErrNilCostFunc when fn is nil.
//   - ErrNaNInf or ErrNegativeCost can be produced later by the kernel.
//
// Determinism:
//   - Determinism depends on fn being deterministic.
//
// Complexity:
//   - Option setup O(1); callback execution participates in O(n*m) kernel time.
func WithCostFunc(fn CostFunc) Option {
	return func(o *options) error {
		if fn == nil {
			return ErrNilCostFunc
		}

		o.costFunc = fn
		return nil
	}
}

// WithAbsoluteCost selects |a-b| as the scalar local-cost function.
//
// Implementation:
//   - Stage 1: assign AbsoluteCost.
//   - Stage 2: the kernel validates produced costs as usual.
//
// Behavior highlights:
//   - This is the default scalar DTW cost.
//   - It is symmetric and non-negative for finite inputs.
//
// Returns:
//   - Option: policy callback for Align.
//
// Errors:
//   - None directly.
//
// Complexity:
//   - Time O(1), Space O(1).
func WithAbsoluteCost() Option {
	return func(o *options) error {
		o.costFunc = AbsoluteCost
		return nil
	}
}

// WithSquaredCost selects (a-b)^2 as the scalar local-cost function.
//
// Implementation:
//   - Stage 1: assign SquaredCost.
//   - Stage 2: the kernel validates produced costs as usual.
//
// Behavior highlights:
//   - Larger deviations are penalized quadratically.
//   - This is useful for signal comparison where large spikes should dominate.
//
// Returns:
//   - Option: policy callback for Align.
//
// Errors:
//   - None directly.
//
// Complexity:
//   - Time O(1), Space O(1).
func WithSquaredCost() Option {
	return func(o *options) error {
		o.costFunc = SquaredCost
		return nil
	}
}

// optionsFromLegacy converts legacy *Options into finalized runtime policy.
//
// Implementation:
//   - Stage 1: treat nil legacy options as DefaultOptions for compatibility.
//   - Stage 2: preserve legacy ReturnPath rule before canonical auto-upgrade.
//   - Stage 3: copy legacy fields into private options.
//   - Stage 4: finalize and validate the complete policy.
//
// Behavior highlights:
//   - nil *Options is accepted by DTW as defaults.
//   - (*Options)(nil).Validate still returns ErrNilOptions.
//   - Legacy ReturnPath requires MemoryMode == FullMatrix.
//
// Inputs:
//   - o: legacy options pointer.
//
// Returns:
//   - options: finalized private runtime policy.
//   - error: sentinel-classified validation failure.
//
// Errors:
//   - ErrPathNeedsMatrix for legacy ReturnPath without FullMatrix.
//   - ErrBadInput / ErrInvalidWindow / ErrInvalidPenalty from finalizeOptions.
//
// Determinism:
//   - Pure conversion; no data-dependent behavior.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// AI-Hints:
//   - Do not silently auto-upgrade legacy ReturnPath+TwoRows; that would change legacy behavior.
func optionsFromLegacy(o *Options) (options, error) {
	if o == nil {
		cfg := defaultOptionsInternal()
		return cfg, finalizeOptions(&cfg)
	}

	if o.ReturnPath && o.MemoryMode != FullMatrix {
		return options{}, ErrPathNeedsMatrix
	}

	cfg := defaultOptionsInternal()
	cfg.window = o.Window
	cfg.slopePenalty = o.SlopePenalty
	cfg.returnPath = o.ReturnPath
	cfg.memoryMode = o.MemoryMode

	if err := finalizeOptions(&cfg); err != nil {
		return options{}, err
	}

	return cfg, nil
}
