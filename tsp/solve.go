// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package tsp - unified dispatcher for TSP solvers.
//
// This file provides the canonical entry points to run TSP algorithms:
//
//   - SolveWithGraph: accept *core.Graph, build an adjacency matrix (optionally
//     with metric closure), derive stable vertex IDs, then delegate to SolveWithMatrix.
//   - SolveWithMatrix: accept a distance matrix + optional IDs and route to the
//     requested algorithm (Christofides / Held–Karp / TwoOptOnly / ThreeOptOnly / …),
//     applying strict validation and optional local-search post-passes.
//
// Design principles:
//   - Deterministic: seed routing to heuristics; no time-based randomness.
//   - Strict sentinels: only errors from types.go; no fmt.Errorf where a sentinel suffices.
//   - Hot-path discipline: no hidden allocations; preallocate slices where needed.
//   - Algorithmic clarity: doc strings with complexity and contracts.
//   - Stable cost: all returned costs are rounded to 1e−9 to prevent FP drift.
package tsp

import (
	"errors"
	"math"

	"github.com/katalvlaran/lvlath/matrix"
)

// solveMeta carries kernel-origin facts from solver dispatch to the canonical TSPResult.
// It prevents facades from inferring algorithmic state from Options alone.
//
// Implementation:
//   - Stage 1: Each dispatch branch initializes metadata from the selected algorithm.
//   - Stage 2: Kernels attach additional facts such as matching fallback or B&B counters.
//   - Stage 3: SolveMatrix publishes metadata into TSPResult.
//
// Behavior highlights:
//   - Private type; public callers consume TSPResult.
//   - Keeps SolveWithMatrix as a pure compatibility projection.
//   - Allows partial-result metadata without changing TSResult.
//
// Inputs:
//   - Produced by solvePreparedMatrix.
//
// Returns:
//   - Applied to TSPResult in publishTSPResult.
//
// Errors:
//   - None.
//
// Determinism:
//   - Metadata mirrors deterministic dispatch branches.
//
// Complexity:
//   - Time O(1), Space O(len(Warnings)) when copied.
//
// AI-Hints:
//   - Do not duplicate this metadata assembly in SolveMatrix.
//   - Do not infer MatchingFallback from Options.
type solveMeta struct {
	algorithm          Algorithm
	exact              bool
	optimal            bool
	timedOut           bool
	approximationRatio float64
	matchingFallback   bool
	iterations         int
	nodesExpanded      int
	warnings           []error
}

// newSolveMeta returns conservative metadata for the selected final algorithm.
// Implementation:
//   - Stage 1: Record the final algorithm after Auto selection.
//   - Stage 2: Mark exact algorithms.
//   - Stage 3: Mark optimal only for exact algorithms on successful completion.
//
// Complexity:
//   - Time O(1), Space O(1).
func newSolveMeta(algo Algorithm) solveMeta {
	meta := solveMeta{
		algorithm:          algo,
		approximationRatio: NoApproximationRatio,
	}

	if algo == ExactHeldKarp || algo == BranchAndBound {
		meta.exact = true
		meta.optimal = true
	}

	return meta
}

// applyApprox attaches Christofides metadata observed by tspApproxWithMeta.
// Implementation:
//   - Stage 1: Copy proven approximation ratio.
//   - Stage 2: Copy matching fallback flag.
//   - Stage 3: Append warnings in stable order.
//
// Complexity:
//   - Time O(len(approx.warnings)), Space O(len(approx.warnings)).
func (m *solveMeta) applyApprox(approx approxMeta) {
	m.approximationRatio = approx.provenRatio
	m.matchingFallback = approx.matchingFallback
	m.warnings = append(m.warnings, approx.warnings...)
}

// trivialRing returns a canonical Hamiltonian cycle [start, start+1, …, n−1, 0, …, start]
// with closure; it allocates exactly n+1 integers and performs no matrix lookups.
//
// Contracts:
//   - 0 ≤ start < n; n ≥ 2.
//
// Complexity: O(n) time, O(n) space.
func trivialRing(n int, start int) ([]int, error) {
	if n < 2 {
		return nil, ErrDimensionMismatch
	}
	if start < 0 || start >= n {
		return nil, ErrStartOutOfRange
	}
	out := make([]int, n+1)

	var (
		i   int // loop iterator
		pos = 0 // independent index of the entry into the resulting slice.
	)

	// Fill from start to n-1.
	for i = start; i < n; i++ {
		out[pos] = i
		pos++
	}
	// Then wrap from 0 to start-1.
	for i = 0; i < start; i++ {
		out[pos] = i
		pos++
	}

	// Close the cycle by returning to start.
	out[n] = start

	return out, nil
}

// prepareSolverDistanceMatrix prepares direct matrix input for final TSP solver kernels.
// Implementation:
//   - Stage 1: Validate option policy before allocation.
//   - Stage 2: Validate pre-closure TSP distance semantics with complete=false.
//   - Stage 3: If metric closure is disabled, return the original matrix.
//   - Stage 4: If metric closure is enabled, copy values into a detached Dense matrix.
//   - Stage 5: Run matrix.APSPInPlace to compute Floyd-Warshall closure.
//   - Stage 6: Validate final TSP solver matrix with complete=true.
//
// Behavior highlights:
//   - Does not mutate caller-owned direct matrix inputs.
//   - Uses matrix.APSPInPlace as the single APSP kernel.
//   - Uses TSP validation only for TSP-specific laws: no negative weights and final completeness.
//
// Inputs:
//   - dist: direct distance matrix.
//   - opts: explicit solver policy.
//
// Returns:
//   - matrix.Matrix: original matrix or detached metric-closed matrix.
//   - bool: true when closure was applied.
//   - int: final matrix order.
//   - error: sentinel-classified failure.
//
// Errors:
//   - ErrInvalidOptions from option validation.
//   - ErrNilDistanceMatrix / ErrNonSquare / ErrDimensionMismatch from structure validation.
//   - ErrNaNInf, ErrNonZeroDiagonal, ErrNegativeWeight, ErrIncompleteGraph, ErrAsymmetry.
//   - ErrNegativeWeight joined with matrix.ErrNegativeCycle if APSP detects a negative cycle.
//
// Determinism:
//   - Copy is row-major.
//   - APSP uses matrix.FloydWarshall deterministic k→i→j order.
//   - Final validation uses deterministic row-major / upper-triangle scans.
//
// Complexity:
//   - Without closure: Time O(n^2), Space O(1).
//   - With closure: Time O(n^3), Space O(n^2).
//
// Notes:
//   - This is a policy-stage helper, not a second TSP algorithm.
//   - Graph inputs that already asked matrix.NewAdjacencyMatrix for metric closure should clear
//     RunMetricClosure before delegating to direct matrix solving to avoid double closure.
//
// AI-Hints:
//   - Do not replace this with “allow +Inf in validateAll”; that lets incomplete
//     final TSP instances reach kernels.
//   - Do not call matrix.InitDistancesInPlace here; direct matrix input already uses
//     +Inf as the missing-edge sentinel, not raw 0-as-no-edge adjacency.
func prepareSolverDistanceMatrix(dist matrix.Matrix, opts Options) (matrix.Matrix, bool, int, error) {
	if err := validateOptionsStandalone(opts); err != nil {
		return nil, false, 0, err
	}

	n, err := validateSolverDistanceMatrix(dist, mustEnforceSymmetry(opts), false, symTol)
	if err != nil {
		return nil, false, 0, err
	}

	if !opts.RunMetricClosure {
		n, err = validateSolverDistanceMatrix(dist, mustEnforceSymmetry(opts), true, symTol)
		if err != nil {
			return nil, false, 0, err
		}

		return dist, false, n, nil
	}

	closed, err := matrix.NewPreparedDense(n, n, matrix.WithAllowInfDistances())
	if err != nil {
		return nil, false, 0, errors.Join(ErrDimensionMismatch, err)
	}

	var (
		row, col int
		value    float64
		readErr  error
	)
	for row = 0; row < n; row++ {
		for col = 0; col < n; col++ {
			value, readErr = dist.At(row, col)
			if readErr != nil {
				return nil, false, 0, errors.Join(ErrDimensionMismatch, readErr)
			}

			if row == col {
				value = 0
			}

			if err = closed.Set(row, col, value); err != nil {
				return nil, false, 0, errors.Join(ErrNaNInf, err)
			}
		}
	}

	if err = matrix.APSPInPlace(closed); err != nil {
		if errors.Is(err, matrix.ErrNegativeCycle) {
			return nil, false, 0, errors.Join(ErrNegativeWeight, err)
		}
		if errors.Is(err, matrix.ErrNaNInf) {
			return nil, false, 0, errors.Join(ErrNaNInf, err)
		}

		return nil, false, 0, err
	}

	n, err = validateSolverDistanceMatrix(closed, mustEnforceSymmetry(opts), true, symTol)
	if err != nil {
		return nil, false, 0, err
	}

	return closed, true, n, nil
}

// solvePreparedMatrix routes an already validated final distance matrix to one solver branch.
// It is the result-native dispatcher used by canonical facades after matrix preparation,
// option finalization, and optional ID validation have already completed.
// Implementation:
//   - Stage 1: Dispatch by opts.Algo.
//   - Stage 2: Run the chosen kernel.
//   - Stage 3: Apply local-search post-pass where the selected policy permits it.
//   - Stage 4: Canonicalize and validate the final tour.
//   - Stage 5: Publish a detached TSPResult with canonical metadata.
//
// Behavior highlights:
//   - Assumes prepareSolverDistanceMatrix and validateIDs already ran.
//   - Does not run metric closure.
//   - Does not mutate caller-owned matrix data.
//   - Does not expose TSResult on the internal dispatcher boundary.
//
// Inputs:
//   - dist: final complete solver matrix.
//   - ids: optional matrix-index ordered labels, already validated by SolveMatrix.
//   - opts: final options with RunMetricClosure=false.
//   - n: matrix order.
//   - metricClosureApplied: true when the facade or adapter applied APSP closure.
//
// Returns:
//   - *TSPResult: canonical successful or partial solver result.
//   - error: sentinel-classified failure.
//
// Errors:
//   - Kernel-specific sentinels from TSPApprox, TSPExact, TwoOpt, ThreeOpt, TSPBranchAndBound.
//   - ErrUnsupportedAlgorithm for unknown algorithm.
//
// Determinism:
//   - Fixed switch dispatch and kernel-level tie-breaks.
//   - Tour canonicalization is applied before return.
//
// Complexity:
//   - Depends on selected algorithm.
//
// AI-Hints:
//   - Do not call this from public code.
//   - Do not add validation here except final invariant checks; public facades own validation.
func solvePreparedMatrix(
	dist matrix.Matrix,
	ids []string,
	opts Options,
	n int,
	metricClosureApplied bool,
) (*TSPResult, error) {
	meta := newSolveMeta(opts.Algo)

	switch opts.Algo {
	case Christofides:
		// Christofides requires symmetric metric; validated in validateAll.
		// 1) Build a feasible tour via tspApprox.
		minimal, approx, err := tspApprox(dist, opts)
		if err != nil {
			return nil, err
		}
		meta.applyApprox(approx)

		result := publishTSPResult(minimal, ids, opts, meta, metricClosureApplied)

		if opts.EnableLocalSearch && n >= 4 {
			local, localErr := twoOptKernel(dist, result.Tour, opts)
			if localErr != nil && !errors.Is(localErr, ErrTimeLimit) {
				return nil, localErr
			}
			attachLocalSearchProgress(result, local)
			if errors.Is(localErr, ErrTimeLimit) {
				return result, ErrTimeLimit
			}

			if opts.BestImprovement {
				local, localErr = threeOptKernel(dist, result.Tour, opts, true)
				if localErr != nil && !errors.Is(localErr, ErrTimeLimit) {
					return nil, localErr
				}
				attachLocalSearchProgress(result, local)
				if errors.Is(localErr, ErrTimeLimit) {
					return result, ErrTimeLimit
				}

				local, localErr = twoOptKernel(dist, result.Tour, opts)
				if localErr != nil && !errors.Is(localErr, ErrTimeLimit) {
					return nil, localErr
				}
				attachLocalSearchProgress(result, local)
				if errors.Is(localErr, ErrTimeLimit) {
					return result, ErrTimeLimit
				}
			}
		}

		return result, nil

	case ExactHeldKarp:
		// Exact DP; no post-pass needed.
		result, err := TSPExact(dist, opts)
		if err != nil {
			return nil, err

		}
		// Stabilize cost for cross-platform consistency.
		result.Cost = round1e9(result.Cost)

		return publishTSPResult(result, ids, opts, meta, metricClosureApplied), nil

	case TwoOptOnly:
		// Build a canonical initial tour (deterministic), then run TwoOpt.
		meta.optimal = false
		base, err := trivialRing(n, opts.StartVertex)
		if err != nil {
			return nil, err
		}

		local, err := twoOptKernel(dist, base, opts)
		if err != nil && !errors.Is(err, ErrTimeLimit) {
			return nil, err
		}

		if !local.hasTour() {
			return nil, err
		}

		result := publishLocalSearchResult(local, ids, opts, TwoOptOnly, metricClosureApplied)
		if errors.Is(err, ErrTimeLimit) {
			return result, ErrTimeLimit
		}

		return result, nil

	case ThreeOptOnly:
		// Canonical initial tour; deterministic seed.
		meta.optimal = false
		base, err := trivialRing(n, opts.StartVertex)
		if err != nil {
			return nil, err
		}

		// Optional warm-up 2-opt pass (fast).
		if opts.EnableLocalSearch && n >= 4 {
			local, localErr := twoOptKernel(dist, base, opts)
			if localErr != nil && !errors.Is(localErr, ErrTimeLimit) {
				return nil, localErr
			}
			if local.hasTour() {
				base = local.tour
			}
			if errors.Is(localErr, ErrTimeLimit) {
				return publishLocalSearchResult(local, ids, opts, ThreeOptOnly, metricClosureApplied), ErrTimeLimit
			}
		}

		// 3-opt with user-selected policy (first/best) and optional shuffle.
		local, err := threeOptKernel(dist, base, opts, opts.BestImprovement)
		if err != nil && !errors.Is(err, ErrTimeLimit) {
			return nil, err
		}
		if !local.hasTour() {
			return nil, err
		}
		result := publishLocalSearchResult(local, ids, opts, ThreeOptOnly, metricClosureApplied)
		if errors.Is(err, ErrTimeLimit) {
			return result, ErrTimeLimit
		}

		// Optional final 2-opt polish (cheap).
		if opts.EnableLocalSearch && n >= 4 {
			local, err = twoOptKernel(dist, result.Tour, opts)
			if err != nil && !errors.Is(err, ErrTimeLimit) {
				return nil, err
			}

			attachLocalSearchProgress(result, local)
			if errors.Is(err, ErrTimeLimit) {
				return result, ErrTimeLimit
			}
		}

		return result, nil

	case BranchAndBound:
		result, err := runBranchAndBoundResult(dist, opts)
		if result == nil {
			return nil, err
		}

		return attachFacadeMetadata(result, ids, opts, metricClosureApplied), err

	default:
		return nil, ErrUnsupportedAlgorithm
	}
}

// attachFacadeMetadata returns a detached result enriched with facade-level metadata.
// It is used when a lower-level kernel already publishes TSPResult directly, such as
// Branch-and-Bound partial-result paths, and the facade still needs to attach IDs and
// metric-closure facts.
//
// Implementation:
//   - Stage 1: Return nil for a nil result.
//   - Stage 2: Clone the kernel result to avoid mutating caller-visible state.
//   - Stage 3: Attach matrix-index ordered IDs and finalized facade policy fields.
//
// Behavior highlights:
//   - Preserves kernel-origin fields such as Optimal, TimedOut, and NodesExpanded.
//   - Does not recompute cost or tour.
//   - Does not clear warnings published by the kernel.
//
// Inputs:
//   - result: kernel-published result.
//   - ids: optional matrix-index ordered vertex IDs.
//   - opts: finalized solver options used by the facade.
//   - metricClosureApplied: true when APSP closure was applied before solving.
//
// Returns:
//   - *TSPResult: detached enriched result, or nil when result is nil.
//
// Errors:
//   - None. Inputs are assumed validated by the caller.
//
// Determinism:
//   - Preserves Tour, Warnings, and IDs order exactly.
//
// Complexity:
//   - Time O(len(Tour)+len(IDs)+len(Warnings)).
//   - Space O(len(Tour)+len(IDs)+len(Warnings)).
//
// Notes:
//   - This helper exists only for kernels that already return TSPResult.
//   - Minimal TSResult projection must not be used to enrich partial results.
//
// AI-Hints:
//   - Do not call Minimal here; partial-result metadata would be lost.
//   - Do not mutate result in place; Clone preserves result ownership boundaries.
func attachFacadeMetadata(
	result *TSPResult,
	ids []string,
	opts Options,
	metricClosureApplied bool,
) *TSPResult {
	if result == nil {
		return nil
	}

	enriched := result.Clone()
	enriched.Algorithm = opts.Algo
	enriched.MetricClosureApplied = metricClosureApplied
	enriched.Symmetric = opts.Symmetric

	if ids != nil {
		enriched.IDs = append([]string(nil), ids...)
	}

	return enriched
}

// finalizeSolverOptions resolves size-dependent solver policy after matrix validation.
// Implementation:
//   - Stage 1: Normalize MaxExactN zero value to DefaultMaxExactN.
//   - Stage 2: Resolve Auto into a concrete Algorithm.
//   - Stage 3: Re-run option validation on the concrete policy.
//
// Behavior highlights:
//   - Auto is explicit and opt-in.
//   - Exact algorithms still keep their resource guards.
//   - No hidden fallback happens when opts.Algo is not Auto.
//
// Inputs:
//   - n: validated matrix order.
//   - opts: user policy after metric-closure flag has been cleared for final solver input.
//
// Returns:
//   - Options: finalized solver policy.
//   - error: sentinel-classified invalid policy.
//
// Errors:
//   - ErrInvalidOptions for invalid MaxExactN.
//   - ErrUnsupportedAlgorithm / ErrATSPNotSupportedByAlgo from final validation.
//
// Determinism:
//   - Pure function of n and opts.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// AI-Hints:
//   - Do not change DefaultOptions to Auto without a compatibility decision.
//   - Do not silently choose heuristics unless opts.Algo == Auto.
func finalizeSolverOptions(n int, opts Options) (Options, error) {
	if opts.MaxExactN < 0 {
		return Options{}, ErrInvalidOptions
	}
	if opts.MaxExactN == 0 {
		opts.MaxExactN = DefaultMaxExactN
	}

	if opts.Algo == Auto {
		opts.Algo = chooseAlgorithm(n, opts)
	}

	if err := validateOptionsStandalone(opts); err != nil {
		return Options{}, err
	}

	return opts, nil
}

// chooseAlgorithm selects a concrete solver for explicit Auto mode.
// Implementation:
//   - Stage 1: Prefer exact Held-Karp when n fits MaxExactN.
//   - Stage 2: Prefer Christofides for symmetric larger instances.
//   - Stage 3: Use TwoOptOnly for asymmetric larger instances.
//
// Behavior highlights:
//   - Deterministic and documented.
//   - Does not override explicit non-Auto algorithms.
//
// Complexity:
//   - Time O(1), Space O(1).
func chooseAlgorithm(n int, opts Options) Algorithm {
	if n <= opts.MaxExactN {
		return ExactHeldKarp
	}
	if opts.Symmetric {
		return Christofides
	}

	return TwoOptOnly
}

// solveDegenerateIfAny handles mathematically valid degenerate TSP instances.
// A one-vertex distance matrix represents the Hamiltonian cycle [0,0] with zero
// total cost; empty matrices remain invalid because they contain no vertex to visit.
//
// Implementation:
//   - Stage 1: Delegate nil, typed-nil, and square shape checks to matrix.ValidateSquare.
//   - Stage 2: Return handled=false for normal n>=2 instances.
//   - Stage 3: Validate the single diagonal value and optional ID mapping.
//   - Stage 4: Publish an exact canonical TSPResult with detached slices.
//
// Behavior highlights:
//   - Does not run metric closure; n==1 closure is a no-op.
//   - Does not allocate algorithm DP/local-search buffers.
//   - Keeps Auto resolution deterministic through finalizeSolverOptions.
//
// Inputs:
//   - dist: direct distance matrix.
//   - ids: optional one-element matrix-index ID mapping.
//   - opts: caller policy already checked by validateOptionsStandalone.
//
// Returns:
//   - *TSPResult: degenerate exact result when handled.
//   - bool: true when n==1 or when shape validation produced a terminal error.
//   - error: nil on valid n==1 or sentinel-classified failure.
//
// Errors:
//   - ErrNilDistanceMatrix joined with matrix.ErrNilMatrix.
//   - ErrNonSquare joined with matrix.ErrNonSquare.
//   - ErrDimensionMismatch for empty matrix or ID length mismatch.
//   - ErrInvalidIDs for empty/duplicate IDs.
//   - ErrStartOutOfRange when StartVertex != 0.
//   - ErrNaNInf joined with matrix.ErrNaNInf for NaN or Inf diagonal.
//   - ErrNonZeroDiagonal when dist[0][0] is not structurally zero.
//
// Determinism:
//   - Fixed single-cell read and fixed `[0,0]` tour publication.
//
// Complexity:
//   - Time O(1), Space O(1) excluding detached result slices.
//
// Notes:
//   - This helper is a facade-level contract guard, not a general solver kernel.
//   - n==2 remains a normal solver instance and passes through regular kernels.
//
// AI-Hints:
//   - Do not route n==1 through Held-Karp or local search.
//   - Do not treat empty matrix as a zero-cost tour.
func solveDegenerateIfAny(dist matrix.Matrix, ids []string, opts Options) (*TSPResult, bool, error) {
	if err := matrix.ValidateSquare(dist); err != nil {
		if errors.Is(err, matrix.ErrNilMatrix) {
			return nil, true, errors.Join(ErrNilDistanceMatrix, err)
		}
		if errors.Is(err, matrix.ErrNonSquare) {
			return nil, true, errors.Join(ErrNonSquare, err)
		}

		return nil, true, errors.Join(ErrDimensionMismatch, err)
	}

	n := dist.Rows()
	if n == 0 {
		return nil, true, ErrDimensionMismatch
	}
	if n != 1 {
		return nil, false, nil
	}
	if opts.StartVertex != 0 {
		return nil, true, ErrStartOutOfRange
	}
	if ids != nil {
		if err := validateIDs(ids, n); err != nil {
			return nil, true, err
		}
	}

	value, err := dist.At(0, 0)
	if err != nil {
		return nil, true, errors.Join(ErrDimensionMismatch, err)
	}
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return nil, true, errors.Join(ErrNaNInf, matrix.ErrNaNInf)
	}
	if math.Abs(value) > symTol {
		return nil, true, ErrNonZeroDiagonal
	}

	finalOptions := opts
	finalOptions.RunMetricClosure = false

	finalOptions, err = finalizeSolverOptions(n, finalOptions)
	if err != nil {
		return nil, true, err
	}

	return &TSPResult{
		Tour:                 []int{0, 0},
		Cost:                 0,
		IDs:                  append([]string(nil), ids...),
		Algorithm:            finalOptions.Algo,
		Optimal:              true,
		Exact:                true,
		TimedOut:             false,
		MetricClosureApplied: false,
		Symmetric:            finalOptions.Symmetric,
		ApproximationRatio:   NoApproximationRatio,
		MatchingFallback:     false,
	}, true, nil
}
