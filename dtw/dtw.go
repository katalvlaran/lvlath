// Package dtw provides a production-grade implementation of the
// Dynamic Time Warping (DTW) algorithm for time series alignment.
//
// DTW finds the minimal cumulative cost to align two sequences by
// stretching/compressing their time axes, subject to an optional
// Sakoe–Chiba window constraint and configurable insertion/deletion penalties.
package dtw

import (
	"math"
)

// Coord represents a single point (i,j) in the optimal warping path.
// I denotes the index in sequence A, J denotes the index in sequence B.
type Coord struct {
	I, J int
}

// DTW computes the DTW distance between sequences a and b,
// and optionally returns the alignment path if opts.ReturnPath=true.
//
// Preconditions:
//   - a and b must be non-empty (checked below).
//   - opts.Validate() must be called by the caller, or DTW will return ErrBadInput.
//
// Time complexity:    O(N*M) where N=len(a), M=len(b)
// Memory complexity:  O(1) for NoMemory,
//
//	O(min(N,M)) for TwoRows,
//	O(N*M) for FullMatrix (with backtrace support).
func DTW(a, b []float64, opts *Options) (dist float64, path []Coord, err error) {
	// 1) Validate input lengths
	n, m := len(a), len(b)
	if n == 0 || m == 0 {
		return 0, nil, ErrEmptyInput
	}
	// 2) Validate option combinations
	if err = opts.Validate(); err != nil {
		return 0, nil, err
	}

	// 3) Precompute constants and buffers
	penalty := opts.SlopePenalty    // non-negative insertion/deletion cost
	window := opts.Window           // Sakoe–Chiba band radius (-1 disabled)
	mode := opts.MemoryMode         // storage strategy
	needPath := opts.ReturnPath     // whether to reconstruct path
	infinity := math.Inf(1)         // positive infinity
	prevRow := make([]float64, m+1) // DP row for i-1
	currRow := make([]float64, m+1) // DP row for i

	// 4) If using FullMatrix, allocate dpMatrix[i][j] = cost up to (i,j)
	var dpMatrix [][]float64
	if mode == FullMatrix {
		// allocate all rows up front
		dpMatrix = make([][]float64, n+1)
		dpMatrix[0] = make([]float64, m+1)
		copy(dpMatrix[0], prevRow)
	}

	// 5) Initialize DP boundary for row 0: cost to align zero-length a with prefixes of b
	var j int
	for j = 1; j <= m; j++ {
		prevRow[j] = infinity // cannot align non-zero prefix with empty sequence
	}

	// 6) Main DP loop: fill rows 1..n
	var i int // loop iterator
	var localCost, matchCost, insertCost, deleteCost, bestPrev float64
	for i = 1; i <= n; i++ {
		// 6.1) Initialize boundary for column 0: cost to align prefixes of a with empty b
		currRow[0] = infinity

		// 6.2) Compute columns 1..m
		for j = 1; j <= m; j++ {
			// 6.2.1) Enforce Sakoe–Chiba window if enabled
			if window >= 0 && abs(i-j) > window {
				currRow[j] = infinity
				continue // skip cost computation outside band
			}

			// 6.2.2) Compute local cost = |a[i-1] - b[j-1]|
			localCost = math.Abs(a[i-1] - b[j-1])

			// 6.2.3) Recurrence relation: match (↖), insertion (↑), deletion (←)
			matchCost = prevRow[j-1]            // cost up to (i-1, j-1)
			insertCost = prevRow[j] + penalty   // insertion in b (advance i)
			deleteCost = currRow[j-1] + penalty // insertion in a (advance j)

			// 6.2.4) Choose minimum predecessor and add local cost
			bestPrev = min3(matchCost, insertCost, deleteCost)
			currRow[j] = localCost + bestPrev
		}

		// 6.3) If FullMatrix, store a copy of currRow for backtracking
		if mode == FullMatrix {
			rowCopy := make([]float64, m+1)
			copy(rowCopy, currRow)
			dpMatrix[i] = rowCopy
		}

		// 6.4) Rotate rows: current becomes previous for next iteration
		prevRow, currRow = currRow, prevRow
	}

	// 7) The final distance is at prevRow[m] after last rotation
	dist = prevRow[m]

	// 8) If requested, reconstruct the optimal path
	if needPath {
		path, err = backtrack(dpMatrix, a, b, opts)
	}

	return dist, path, err
}

// backtrack reconstructs the alignment path from dpMatrix.
// It walks backward from (N,M) to (0,0) following the minimal-cost moves.
func backtrack(dp [][]float64, a, b []float64, opts *Options) ([]Coord, error) {
	i, j := len(a), len(b)
	path := make([]Coord, 0, i+j)

	for i > 0 || j > 0 {
		// record the alignment (i-1,j-1) or boundary
		var x, y int
		if i > 0 && j > 0 {
			x, y = i-1, j-1
		} else if i > 0 {
			x, y = i-1, 0
		} else {
			x, y = 0, j-1
		}
		path = append(path, Coord{I: x, J: y})

		// compute the local cost at (i,j)
		moved := false
		var localCost float64
		if i > 0 && j > 0 {
			localCost = math.Abs(a[i-1] - b[j-1])
		}
		// “unwind” the localCost before comparing to predecessors
		curr := dp[i][j] - localCost

		// 1) match ↖
		if i > 0 && j > 0 && almostEqual(curr, dp[i-1][j-1]) {
			i, j = i-1, j-1
			moved = true
		}
		// 2) insertion (↑)
		if !moved && i > 0 && almostEqual(curr, dp[i-1][j]+opts.SlopePenalty) {
			i--
			moved = true
		}
		// 3) deletion  (←)
		if !moved && j > 0 && almostEqual(curr, dp[i][j-1]+opts.SlopePenalty) {
			j--
			moved = true
		}

		if !moved {
			return nil, ErrIncompletePath
		}
	}

	// reverse path so it's from (0,0)→(N,M)
	for l, r := 0, len(path)-1; l < r; l, r = l+1, r-1 {
		path[l], path[r] = path[r], path[l]
	}
	return path, nil
}

// min3 returns the minimum of three float64 values.
func min3(a, b, c float64) float64 {
	if a < b {
		if a < c {
			return a
		}

		return c
	}
	if b < c {
		return b
	}

	return c
}

// abs returns the absolute value of x.
func abs(x int) int {
	if x < 0 {
		return -x
	}

	return x
}

// almostEqual reports whether two floats are equal within a small epsilon.
func almostEqual(a, b float64) bool {
	const eps = 1e-9
	return math.Abs(a-b) <= eps
}
