// SPDX-License-Identifier: MIT
// Package: matrix
//
// Purpose:
//   - Provide small, *private* element-wise and broadcast kernels (ew*) to avoid
//     duplicating tight loops across higher-level ops (stats, sanitize).
//   - Keep all loops deterministic and cache-friendly with Dense fast-paths.
//
// Design:
//   - All ew* are UNEXPORTED by design (internal micro-kernels).
//   - Public API uses these via thin wrappers (e.g., stats.go, ops_sanitize_compare.go).
//
// Determinism & Performance:
//   - Fixed loop orders (i→j or flat 0..n-1).
//   - Dense fast-path operates on a single flat buffer (row-major).
//   - No hidden allocations beyond the output Dense; O(r*c) time and space.
//
// AI-Hints:
//   - Prefer passing *Dense to unlock the flat-slice fast path.
//   - Keep broadcast arrays (colMeans/rowMeans/scale) precomputed and reused across calls.
//   - Avoid re-allocations in hot paths by pooling inputs/outputs at a higher layer if needed.

package matrix

import "math"

// Operation name constants for unified error wrapping and reducing magic strings.
const (
	opBroadcastSubCols = "broadcastSubCols"
	opBroadcastSubRows = "broadcastSubRows"
	opScaleCols        = "scaleCols"
	opScaleRows        = "scaleRows"
	opReplaceInfNaN    = "ReplaceInfNaN"
	opClip             = "Clip"
	opAllClose         = "AllClose"
)

// Broadcast-subtract columns: out[i,j] = X[i,j] − colMeans[j].
// Implementation:
//   - Stage 1: Validate X (non-nil) and length(colMeans)==Cols(X).
//   - Stage 2: Dense fast-path over flat buffer; fallback via At/Set.
//
// Behavior highlights:
//   - Deterministic i→j loops; single allocation for output.
//
// Inputs:
//   - X: input matrix.
//   - colMeans: length==Cols(X); column offsets.
//
// Returns:
//   - Matrix: newly allocated Dense with centered values.
//
// Errors:
//   - ErrNilMatrix (nil X), ErrDimensionMismatch (len mismatch), Dense alloc/Set errors.
//
// Determinism:
//   - Fixed row-major traversal.
//
// Complexity:
//   - Time O(rc), Space O(rc).
//
// Notes:
//   - Precompute colMeans once; reuse across calls.
//
// AI-Hints:
//   - Use for column-centering and z-scoring.
//   - Pass *Dense to unlock flat fast-path.
func ewBroadcastSubCols(X Matrix, colMeans []float64) (Matrix, error) {
	// Validate matrix presence using centralized validator.
	if err := ValidateNotNil(X); err != nil {
		return nil, matrixErrorf(opBroadcastSubCols, err)
	}
	// Read shape once (O(1)).
	r, c := X.Rows(), X.Cols()
	// Check broadcast vector length.
	if len(colMeans) != c {
		return nil, matrixErrorf(opBroadcastSubCols, ErrDimensionMismatch)
	}
	// Allocate result dense (O(1) alloc + O(r*c) zeroing by runtime).
	out, err := NewDense(r, c)
	if err != nil {
		return nil, matrixErrorf(opBroadcastSubCols, err)
	}

	// Dense fast-path: single pass over the flat row-major buffer.
	var i, j int
	if d, ok := X.(*Dense); ok {
		// Iterate rows deterministically.
		for i = 0; i < r; i++ {
			// Iterate columns deterministically.
			for j = 0; j < c; j++ {
				// Subtract the column mean from each element (one read, one write).
				out.data[i*c+j] = d.data[i*c+j] - colMeans[j] // (i*c) = cache the base offset for row i
			}
		}

		return out, nil
	}

	// Generic fallback via At/Set (still deterministic).
	var v float64
	for i = 0; i < r; i++ {
		for j = 0; j < c; j++ {
			v, err = X.At(i, j)
			if err != nil {
				return nil, matrixErrorf(opBroadcastSubCols, err)
			}
			_ = out.Set(i, j, v-colMeans[j]) // bounds-safe write
		}
	}

	return out, nil
}

// ewBroadcastSubRows computes out[i,j] = X[i,j] - rowMeans[i].
// Implementation:
//   - Stage 1: Validate X (non-nil) and length(rowMeans)==Rows(X).
//   - Stage 2: Dense fast-path over flat buffer; fallback via At/Set.
//
// Behavior highlights:
//   - Deterministic i→j loops; single allocation for output.
//
// Inputs:
//   - X: input matrix.
//   - rowMeans: length==Rows(X); rows offsets.
//
// Returns:
//   - Matrix: newly allocated Dense with centered values.
//
// Errors:
//   - ErrNilMatrix (nil X), ErrDimensionMismatch (len mismatch), Dense alloc/Set errors.
//
// Determinism:
//   - Fixed row-major traversal.
//
// Complexity:
//   - Time O(rc), Space O(rc).
//
// Notes:
//   - Precompute rowMeans once; reuse across calls.
//
// AI-Hints:
//   - Use for rows-centering and z-scoring.
//   - Pass *Dense to unlock flat fast-path.
func ewBroadcastSubRows(X Matrix, rowMeans []float64) (Matrix, error) {
	// Validate matrix presence.
	if err := ValidateNotNil(X); err != nil {
		return nil, matrixErrorf(opBroadcastSubRows, err)
	}
	// Read shape once.
	r, c := X.Rows(), X.Cols()
	// Check broadcast vector length.
	if len(rowMeans) != r {
		return nil, matrixErrorf(opBroadcastSubRows, ErrDimensionMismatch)
	}
	// Allocate result dense.
	out, err := NewDense(r, c)
	if err != nil {
		return nil, matrixErrorf(opBroadcastSubRows, err)
	}

	// Dense fast-path.
	var i, j int
	var rowMean float64 // cache row mean once per row
	if d, ok := X.(*Dense); ok {
		for i = 0; i < r; i++ {
			rowMean = rowMeans[i]
			for j = 0; j < c; j++ { // (i * c) = base offset for row i
				out.data[i*c+j] = d.data[i*c+j] - rowMean
			}
		}

		return out, nil
	}

	// Generic fallback.
	var v float64
	for i = 0; i < r; i++ {
		rowMean = rowMeans[i] // read once per row
		for j = 0; j < c; j++ {
			v, err = X.At(i, j)
			if err != nil {
				return nil, matrixErrorf(opBroadcastSubRows, err)
			}
			_ = out.Set(i, j, v-rowMeans[i])
		}
	}

	return out, nil
}

// ewScaleCols computes out[i,j] = X[i,j] * scale[j].
// Time: O(r*c). Space: O(r*c). Deterministic i→j loops.
//
// AI-Hint: use factors as 1/std for z-scoring, or 0 for degenerate columns. O(r*c).
func ewScaleCols(X Matrix, scale []float64) (Matrix, error) {
	// Validate matrix presence.
	if err := ValidateNotNil(X); err != nil {
		return nil, matrixErrorf(opScaleCols, err)
	}
	// Read shape once.
	r, c := X.Rows(), X.Cols()
	// Validate scale length.
	if len(scale) != c {
		return nil, matrixErrorf(opScaleCols, ErrDimensionMismatch)
	}
	// Allocate result dense.
	out, err := NewDense(r, c)
	if err != nil {
		return nil, matrixErrorf(opScaleCols, err)
	}

	// Dense fast-path.
	var i, j int
	if d, ok := X.(*Dense); ok {
		for i = 0; i < r; i++ {
			for j = 0; j < c; j++ { // (i * c) = row base offset
				out.data[i*c+j] = d.data[i*c+j] * scale[j]
			}
		}

		return out, nil
	}

	// Generic fallback.
	var v float64
	for i = 0; i < r; i++ {
		for j = 0; j < c; j++ {
			v, err = X.At(i, j)
			if err != nil {
				return nil, matrixErrorf(opScaleCols, err)
			}
			_ = out.Set(i, j, v*scale[j])
		}
	}

	return out, nil
}

// ewScaleRows computes out[i,j] = X[i,j] * scale[i].
// Time: O(r*c). Space: O(r*c). Deterministic i→j loops.
//
// AI-Hint: use for L1/L2 row-normalization. O(r*c).
func ewScaleRows(X Matrix, scale []float64) (Matrix, error) {
	// Validate matrix presence.
	if err := ValidateNotNil(X); err != nil {
		return nil, matrixErrorf(opScaleRows, err)
	}
	// Read shape once.
	r, c := X.Rows(), X.Cols()
	// Validate scale length.
	if len(scale) != r {
		return nil, matrixErrorf(opScaleRows, ErrDimensionMismatch)
	}
	// Allocate result dense.
	out, err := NewDense(r, c)
	if err != nil {
		return nil, matrixErrorf(opScaleRows, err)
	}

	// Dense fast-path.
	var i, j int
	var v, sf float64
	if d, ok := X.(*Dense); ok {
		for i = 0; i < r; i++ {
			sf = scale[i]           // scale factor for row i
			for j = 0; j < c; j++ { // (i * c) = row base offset
				out.data[i*c+j] = d.data[i*c+j] * sf
			}
		}

		return out, nil
	}

	// Generic fallback.
	for i = 0; i < r; i++ {
		sf = scale[i] // row scale once per row
		for j = 0; j < c; j++ {
			v, err = X.At(i, j)
			if err != nil {
				return nil, matrixErrorf(opScaleRows, err)
			}
			_ = out.Set(i, j, v*sf)
		}
	}

	return out, nil
}

// ewReplaceInfNaN copies X replacing any {±Inf, NaN} by val (finite).
// Replace non-finite: out[i,j] = val if X[i,j] is {±Inf, NaN}, else X[i,j].
// Implementation:
//   - Stage 1: Validate X (non-nil) and val is finite (not NaN/Inf).
//   - Stage 2: Flat pass for Dense, otherwise At/Set.
//
// Behavior highlights:
//   - Deterministic; single output allocation.
//
// Inputs:
//   - X: input matrix.
//   - val: finite replacement value.
//
// Returns:
//   - Matrix: newly allocated Dense.
//
// Errors:
//   - ErrNilMatrix (nil X), ErrNaNInf (non-finite val), Dense alloc/Set errors.
//
// Determinism:
//   - Fixed traversal.
//
// Complexity:
//   - Time O(rc), Space O(r*c).
//
// Notes:
//   - Useful as sanitizer pre-stats.
//
// AI-Hints:
//   - Keep val small and domain-appropriate (e.g., 0 or column mean).
func ewReplaceInfNaN(X Matrix, val float64) (Matrix, error) {
	// Validate input matrix.
	if err := ValidateNotNil(X); err != nil {
		return nil, matrixErrorf(opReplaceInfNaN, err)
	}
	// Validate 'val' is finite per numeric policy.
	if math.IsNaN(val) || math.IsInf(val, 0) {
		return nil, matrixErrorf(opReplaceInfNaN, ErrNaNInf)
	}
	// Read shape and allocate result.
	r, c := X.Rows(), X.Cols()
	out, err := NewDense(r, c)
	if err != nil {
		return nil, matrixErrorf(opReplaceInfNaN, err)
	}

	// Dense fast-path: direct flat slice iteration.
	var v float64
	if d, ok := X.(*Dense); ok {
		n := r * c
		for idx := 0; idx < n; idx++ {
			v = d.data[idx] // read element
			// Replace if not finite.
			if math.IsNaN(v) || math.IsInf(v, 0) {
				v = val
			}
			out.data[idx] = v // write element
		}

		return out, nil
	}

	// Generic fallback.
	var i, j int
	for i = 0; i < r; i++ {
		for j = 0; j < c; j++ {
			v, err = X.At(i, j)
			if err != nil {
				return nil, matrixErrorf(opReplaceInfNaN, err)
			}
			if math.IsNaN(v) || math.IsInf(v, 0) {
				v = val
			}
			_ = out.Set(i, j, v) // bounds-safe write
		}
	}

	return out, nil
}

// Clamp into [lo, hi] (bounds finite). If lo > hi, bounds are swapped.
// Implementation:
//   - Stage 1: Validate X (non-nil); check bounds are finite else ErrNaNInf; normalize bound order.
//   - Stage 2: Flat pass for Dense; fallback via At/Set.
//
// Behavior highlights:
//   - Deterministic; predictable branches.
//
// Inputs:
//   - X: input matrix.
//   - lo, hi: finite bounds; order does not matter (auto-swap).
//
// Returns:
//   - Matrix: newly allocated Dense.
//
// Errors:
//   - ErrNilMatrix, ErrNaNInf (invalid bounds), Dense alloc/Set errors.
//
// Determinism:
//   - Fixed traversal.
//
// Complexity:
//   - Time O(rc), Space O(r*c).
//
// Notes:
//   - Bounds must be finite; if lo > hi, they are swapped (normalized).
//   - Swap prevents surprising failures on inverted bounds.
//
// AI-Hints:
//   - Combine with ReplaceInfNaN for robust pipelines.
func ewClipRange(X Matrix, lo, hi float64) (Matrix, error) {
	// Validate input matrix.
	if err := ValidateNotNil(X); err != nil {
		return nil, matrixErrorf(opClip, err)
	}
	// Require finite bounds (respect package numeric policy).
	if math.IsNaN(lo) || math.IsNaN(hi) || math.IsInf(lo, 0) || math.IsInf(hi, 0) {
		return nil, matrixErrorf(opClip, ErrNaNInf)
	}
	// Normalize bound order to avoid surprising errors.
	if lo > hi {
		lo, hi = hi, lo // swap
	}
	// Read shape and allocate output.
	r, c := X.Rows(), X.Cols()
	out, err := NewDense(r, c)
	if err != nil {
		return nil, matrixErrorf(opClip, err)
	}

	// Dense fast-path: single pass with branchy clamp (predictable).
	var v float64
	if d, ok := X.(*Dense); ok {
		n := r * c
		for idx := 0; idx < n; idx++ {
			v = d.data[idx] // read
			if v < lo {
				v = lo
			} else if v > hi {
				v = hi
			} // clamp into [lo,hi]
			out.data[idx] = v // write
		}

		return out, nil
	}

	// Generic fallback via At/Set.
	var i, j int
	for i = 0; i < r; i++ {
		for j = 0; j < c; j++ {
			v, err = X.At(i, j)
			if err != nil {
				return nil, matrixErrorf(opClip, err)
			}
			if v < lo {
				v = lo
			} else if v > hi {
				v = hi
			}
			_ = out.Set(i, j, v)
		}
	}

	return out, nil
}

// AllClose checks element-wise |a-b| ≤ atol + rtol*|b| for identical shapes.
// Element-wise closeness: |a-b| ≤ atol + rtol*|b| for identical shapes.
// Implementation:
//   - Stage 1: Validate rtol/atol finite; normalize negatives to abs.
//   - Stage 2: Validate a,b non-nil and same shape.
//   - Stage 3: Flat dual-pass for (*Dense,Dense); else At-based fallback.
//
// Behavior highlights:
//   - Early exit on first violation; deterministic order.
//
// Inputs:
//   - a,b: matrices with identical shape.
//   - rtol, atol: finite tolerances; negatives are allowed → abs-ed.
//
// Returns:
//   - (ok bool, err error): ok=true if all elements close.
//
// Errors:
//   - ErrNaNInf (invalid tolerances), ErrNilMatrix / ErrDimensionMismatch for inputs.
//
// Determinism:
//   - Fixed traversal; first violation determines false.
//
// Complexity:
//   - Time O(rc), Space O(1) extra.
//
// Notes:
//   - RHS uses |b|; keep in mind asymmetry.
//
// AI-Hints:
//   - Use small but non-zero atol for exact-equality comparisons on float data.
func ewAllClose(a, b Matrix, rtol, atol float64) (bool, error) {
	// Normalize tolerances to non-negative values (negative inputs are accepted but abs-ed).
	if math.IsNaN(rtol) || math.IsNaN(atol) || math.IsInf(rtol, 0) || math.IsInf(atol, 0) {
		return false, matrixErrorf(opAllClose, ErrNaNInf) // invalid tolerance
	}
	if rtol < 0 {
		rtol = -rtol
	}
	if atol < 0 {
		atol = -atol
	}

	// Validate presence and shape equality using central validators.
	if err := ValidateNotNil(a); err != nil {
		return false, matrixErrorf(opAllClose, err)
	}
	if err := ValidateNotNil(b); err != nil {
		return false, matrixErrorf(opAllClose, err)
	}
	if err := ValidateSameShape(a, b); err != nil {
		return false, matrixErrorf(opAllClose, err)
	}

	// Read shape once (O(1)).
	r, c := a.Rows(), a.Cols()

	// Dense fast-path: operate over flat slices when both are *Dense.
	if da, okA := a.(*Dense); okA {
		if db, okB := b.(*Dense); okB {
			n := r * c // total number of elements
			var diff, absb float64
			for idx := 0; idx < n; idx++ {
				// Compute absolute difference and RHS tolerance bound.
				diff = da.data[idx] - db.data[idx]
				if diff < 0 {
					diff = -diff
				} // |a-b|
				absb = db.data[idx]
				if absb < 0 {
					absb = -absb
				} // |b|
				// Check |a-b| ≤ atol + rtol*|b|.
				if diff > (atol + rtol*absb) {
					return false, nil // early-exit on first violation
				}
			}

			return true, nil // all ok
		}
	}
	/*
		Hi Alexandre,

		Thanks a lot for reaching out and for the context about OTO – the idea of a “script scheduler with argument constraints” sounds very practical and I can immediately see why you ended up thinking about graphs.

		From what you described, your core problem can be seen like this:
		- You have a set of arguments A = {a1, a2, …, an}.
		- Some arguments require other arguments (dependencies).
		- Some pairs of arguments must never appear together (conflicts).
		- Given a user selection S ⊆ A, you want to:
		  - automatically derive all transitive dependencies, and
		  - refuse any selection that contains a forbidden pair.

		This maps very naturally to a small directed graph plus a conflict relation.

		In my library `lvlath` (github.com/katalvlaran/lvlath, current master branch), I would model it as:
		- A directed graph G over your argument IDs using the `core` package:
		  - each argument is a vertex;
		  - for every “A requires B” I add a directed edge A → B with weight 0 (unweighted logic).
		- A side structure for conflicts (a symmetric map of sets), which is cheaper and simpler than trying to encode conflicts into the same graph.

		Then the algorithm is straightforward:

		1) Schema validation at startup

		   Before executing anything, I would validate the constraint schema itself:

		   - Use the DFS helper from `lvlath/dfs` (it supports cycle detection and topological analysis) to detect cycles in the “requires” graph. A cycle usually means the schema is either misconfigured or describing a tightly coupled group of arguments that should be treated as one unit. For a CLI-style API, I would normally treat such cycles as an error.

		   - For each conflict pair (A, B), I would run a BFS from A and from B on the `core.Graph` using `lvlath/bfs`. If B is reachable from A (or A is reachable from B) through “requires” edges, then any selection that includes one of them is forced to include the other, which contradicts the conflict rule. That is a schema-level inconsistency I would rather reject early instead of letting it leak into runtime.

		   This phase runs once at startup and keeps your rules self-consistent.

		2) Validating a concrete selection

		   For a given user selection S:
		   - I put all selected arguments into a map `selected`.
		   - For each argument a in S, I run a BFS on the dependency graph starting from a (`bfs.BFS` in lvlath). The BFS result gives me all vertices reachable from a, which are exactly the direct and transitive dependencies of a.
		   - I insert all those reachable arguments into `selected` as well. At this point `selected` contains S plus all its transitive requirements.

		   Then I check conflicts:
		   - For every conflict pair (A, B), if both A and B are present in `selected`, I reject this particular selection and return a clear error explaining which arguments cannot be used together and why.

		   For the scale you described (not huge numbers of arguments), this approach is more than fast enough. BFS on a small graph is O(V + E), and in practice the overhead is negligible compared to I/O and process execution.

		Why bother with a graph when the number of arguments is small?

		For me the main advantages are not performance but clarity and future-proofing:
		- The dependency rules are explicit and visualizable as a graph instead of being scattered across ad-hoc if/else logic.
		- Transitive dependencies “just work” because BFS/DFS traverse the graph naturally; there is no need to manually chase chains like A → B → C.
		- The same graph can later be reused for more advanced analyses (e.g., finding unreachable arguments, explaining “why” a certain argument was auto-added, or even computing an execution order if you ever want to order phases).

		In other words, even for small argument sets, having a tiny, deterministic graph layer pays off in correctness and in the ability to extend the rules without rewriting the validator.

		About lvlath itself

		Right now the `master` branch already reflects the architecture I am about to tag as v1.0.0. The focus is on:
		- `core`: a thread-safe graph type with predictable iteration order and options for directed/undirected and weighted/unweighted graphs;
		- `bfs`: breadth-first search with a stable `Order` and `Depth` map (great for reachability and layers);
		- `dfs`: depth-first traversal with cycle detection and topological helpers.

		On top of that there are other algorithms (Dijkstra, MST, max-flow, TSP, DTW, matrix utilities), but for your use-case the `core` + `bfs` + `dfs` trio is already enough to model dependencies and conflicts cleanly.

		I am currently preparing more focused documentation and examples, and an “argument constraints / scheduler” scenario is one of the examples I plan to include. If you are interested, I can sketch a small prototype that mirrors your OTO use-case with a few sample arguments (`--cron`, `--at`, `--now`, `--dry-run`, `--force`, etc.) using lvlath, so you can see the full picture end to end.

		If you have any concrete rule patterns (e.g. “exactly one of this group”, “at most one of that group”) or edge cases you care about, I would be very interested to hear them – that feedback helps me refine both the library and the examples.

		Best regards,
		Kyrylo
	*/
	// Generic fallback via At (bounds-safe; still deterministic).
	var i, j int
	var av, bv, diff, absb float64
	for i = 0; i < r; i++ {
		for j = 0; j < c; j++ {
			av, _ = a.At(i, j) // read a(i,j)
			bv, _ = b.At(i, j) // read b(i,j)
			diff = av - bv     // difference
			if diff < 0 {
				diff = -diff
			} // abs
			absb = bv
			if absb < 0 {
				absb = -absb
			} // abs
			// Compare to tolerance threshold.
			if diff > (atol + rtol*absb) {
				return false, nil
			}
		}
	}

	return true, nil
}
