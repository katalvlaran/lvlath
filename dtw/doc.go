// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package dtw implements Dynamic Time Warping for deterministic sequence alignment.
//
// DTW compares ordered signals that describe the same process at different speeds.
// It is designed for cases where direct index-by-index comparison is too rigid:
// price patterns may stretch across more candles, voice commands may contain held
// phonemes, vibration events may drift in phase, and sensor signatures may be
// sampled with slightly different timing.
//
// Public facades:
//
//   - Align compares scalar sequences.
//     Use it when each time step is one number: price impulse, temperature,
//     normalized amplitude, trend score, or any other scalar signal.
//
//   - AlignCostMatrix compares a precomputed local-cost surface.
//     Use it when another model already computed frame-to-frame costs: acoustic
//     model scores, embedding distances, classifier losses, graph-derived costs,
//     or a domain-specific distance matrix.
//
//   - AlignMatrix compares multivariate sequences.
//     Rows are time steps, columns are feature dimensions. The local cost is
//     squared L2 row distance:
//     C_{ij}=|X_i-Y_j|_2^2
//
//     Use it for OHLC candle vectors, sensor feature vectors, gesture features,
//     or any time-indexed feature matrix.
//
//   - DTW is the legacy tuple-returning wrapper.
//     It exists for compatibility. New code should use Align, AlignCostMatrix,
//     or AlignMatrix because Result carries reachability, path state, options,
//     and optional matrix artifacts explicitly.
//
// Mathematical law:
//   - Scalar local cost defaults to |a[i]-b[j]|.
//   - WithSquaredCost changes scalar local cost to (a[i]-b[j])².
//   - AlignMatrix uses squared L2 row distance:
//     C(i,j)=||X_i-Y_j||².
//   - AlignCostMatrix accepts caller-provided finite non-negative local costs.
//   - Accumulated cost:
//     D(0,0)=0;
//     D(i,0)=+Inf for i>0;
//     D(0,j)=+Inf for j>0;
//     D(i,j)=cost(i,j)+min(
//     D(i-1,j-1),
//     D(i-1,j)+SlopePenalty,
//     D(i,j-1)+SlopePenalty,
//     ).
//   - +Inf means unreachable under the active window/domain policy.
//
// Result model:
//
//   - Result.Distance is the accumulated cost of the selected optimal alignment.
//   - Result.Reachable is false when no admissible path reaches the final cell.
//   - Result.Distance is +Inf when Result.Reachable is false.
//   - Result.Path is filled only when WithReturnPath(true) is used and the result
//     is reachable.
//   - Result.PathTracked tells whether path tracking was requested.
//   - Result.Window, Result.SlopePenalty, and Result.MemoryMode record the executed
//     policy.
//   - Result.Accumulated is present only with WithReturnAccumulated(true).
//   - Result.LocalCost is present only with WithReturnLocalCost(true).
//   - PathOrError separates three states: nil result, no path, and path not tracked.
//
// Reading a path:
//
//   - A path element {I,J} means “time step I from the first sequence aligns with
//     time step J from the second sequence”.
//   - Diagonal steps consume both sequences together.
//   - Vertical or horizontal steps represent stretching: one side advances while
//     the other side is held.
//   - Repeated I values often mean the second sequence stretched a phase.
//   - Repeated J values often mean the first sequence stretched a phase.
//   - The returned path is one deterministic optimal path, not all possible
//     optimal paths.
//
// Window policy:
//
//   - WithWindow(-1) disables the Sakoe-Chiba band.
//   - WithWindow(0) enforces strict diagonal alignment and forbids warping.
//   - WithWindow(w) for w>0 admits cells satisfying |i-j| <= w.
//   - A small positive window is usually safer for production systems than an
//     unconstrained path because it prevents unrealistic temporal jumps.
//   - Window < -1 is invalid and returns ErrInvalidWindow joined with ErrBadInput.
//
// SlopePenalty policy:
//
//   - WithSlopePenalty(p) adds p to vertical and horizontal steps.
//   - p must be finite and non-negative.
//   - p=0 allows stretching without additional structural penalty.
//   - Larger p biases the solution toward diagonal movement.
//   - Use a small positive p when repeated frames are realistic but should still
//     affect the score: held speech sounds, delayed candles, slow sensor rise,
//     or jitter around a physical event.
//
// Cost policy:
//
//   - Align defaults to AbsoluteCost: |a-b|.
//   - WithSquaredCost switches scalar Align to (a-b)^2.
//   - WithCostFunc installs a custom scalar local-cost function.
//   - Custom local costs must be finite and non-negative.
//   - AlignMatrix always uses squared L2 row distance in the current contract.
//   - AlignCostMatrix ignores CostFunc because the supplied matrix already is
//     the local-cost source.
//
// Numeric policy:
//
//   - Scalar input samples must be finite by default.
//   - WithValidateFinite(false) only skips scalar input finite checks; it does
//     not allow invalid local costs.
//   - AlignMatrix validates matrix inputs with matrix.ValidateAllFinite.
//   - AlignCostMatrix rejects NaN, +Inf, -Inf, and negative local costs.
//   - +Inf is reserved for unreachable accumulated DP states and no-path results.
//   - NaN is never a valid DTW distance, local cost, or path-comparison value.
//
// Memory policy:
//
//   - TwoRows stores only rolling DP rows and is the default distance-only mode.
//   - FullMatrix stores the accumulated DP matrix and enables path reconstruction.
//   - WithReturnPath(true) and WithReturnAccumulated(true) require full accumulated
//     storage.
//   - NoMemory is a deprecated compatibility alias for rolling-row distance mode.
//   - Exact DTW distance in this package is not an O(1)-memory algorithm.
//
// Matrix artifacts:
//
//   - Result.Accumulated is the DP surface D.
//     It is useful for debugging, heatmaps, path inspection, and explaining why
//     a path chose one predecessor over another.
//
//   - Result.LocalCost is the local cost surface C.
//     It is useful for model inspection, acoustic/embedding diagnostics, and
//     verifying that feature scaling is sensible.
//
//   - Returned matrices are detached artifacts.
//     Mutating Result.LocalCost or Result.Accumulated does not mutate the caller’s
//     input matrix.
//
// Error classification:
//
//   - Use errors.Is; never match error strings.
//   - ErrNilInput means a scalar input slice is nil.
//   - ErrEmptyInput means a scalar sequence or local-cost matrix domain is empty.
//   - ErrNilOption means a nil functional option was passed.
//   - ErrInvalidWindow means the window is below -1.
//   - ErrInvalidPenalty means slope penalty is NaN, Inf, or negative.
//   - ErrNaNInf means a finite-only input or local cost contained NaN or Inf.
//   - ErrNegativeCost means a local cost is negative.
//   - ErrNoPath means path tracking was requested but the final cell is unreachable.
//   - ErrPathNotTracked means the caller requested a path from a result that did
//     not track one.
//   - Matrix-backed adapters preserve relevant matrix sentinels, such as
//     matrix.ErrDimensionMismatch, by joining them with DTW-level errors.
//
// Determinism:
//
//   - DP loops are row-major: i ascending, then j ascending.
//   - Matrix local costs are materialized in row-major order.
//   - Backtracking tie-break is diagonal first, then vertical/up, then horizontal/left.
//   - Returned paths are ordered from start to end.
//   - No map iteration or randomized behavior participates in path selection.
//
// Operational recipes:
//
//   - Fast distance scan:
//     call Align or AlignMatrix without WithReturnPath and without matrix artifacts.
//     This keeps memory to rolling rows.
//
//   - Explain one match:
//     enable WithReturnPath(true), optionally WithReturnAccumulated(true), and inspect
//     the path anchors where one side repeats.
//
//   - Debug feature scaling:
//     use AlignMatrix with WithReturnLocalCost(true). If one feature dominates the
//     local-cost matrix, normalize outside DTW before alignment.
//
//   - Use external models:
//     pass their finite non-negative frame costs to AlignCostMatrix. Do not force
//     those costs through scalar slices.
//
// Common pitfalls:
//
//   - Window=0 is not “no constraint”; it is strict diagonal alignment.
//   - Unconstrained DTW can over-warp; production systems usually need a window.
//   - Very large slope penalties can turn DTW into near-diagonal comparison.
//   - DTW distance is not a metric in general; do not assume triangle inequality.
//   - AlignMatrix uses squared L2 distance; feature scale matters.
//   - Do not pass probabilities directly as costs unless the scale is intentional;
//     many models should convert probabilities to costs first, for example -log(p).
//
// Ownership:
//
//   - Input slices and input matrices are never mutated.
//   - Result.Path is detached and caller-owned.
//   - Result.Accumulated and Result.LocalCost are detached matrix.Dense artifacts.
//   - Result values are not internally synchronized; callers must synchronize their
//     own mutation of returned artifacts across goroutines.
//
// Partial-result law:
//
//   - When path tracking is requested and no admissible path exists, Align,
//     AlignCostMatrix, and AlignMatrix return Result{Distance:+Inf, Reachable:false}
//     together with ErrNoPath.
//
// Concurrency:
//
//   - Package functions are pure over caller-provided immutable inputs.
//   - Concurrent mutation of input slices or matrices during execution is unsupported.
//   - Result values are not internally synchronized; callers own synchronization if
//     they mutate returned Path or matrix artifacts across goroutines.
//
// Complexity summary:
//
//   - Scalar Align: O(n*m) time.
//   - AlignCostMatrix: O(n*m) materialization plus O(n*m) DP.
//   - AlignMatrix: O(n*m*d) local-cost construction plus O(n*m) DP.
//   - Distance-only rolling-row mode uses O(m) DP memory.
//   - Path or accumulated matrix output uses O(n*m) memory.
//
// AI-Hints:
//   - Do not treat Window=0 as unconstrained.
//   - Do not classify errors by string matching; use errors.Is.
//   - Do not replace Result with raw parallel return values on canonical APIs.
//   - Do not silently normalize matrix inputs inside AlignMatrix.
//   - Do not allow +Inf local costs; +Inf belongs to accumulated unreachable states.
//   - Do not attempt path reconstruction from rolling rows.
//   - Do not use AlignMatrix for heterogeneous feature scales without preprocessing.
//   - Do not document NoMemory as O(1) exact DTW.
package dtw
