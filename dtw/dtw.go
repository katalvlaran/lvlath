package dtw

import (
	"math"
)

// Coord represents a point (i,j) in the optimal warping path.
type Coord struct{ I, J int }

// DTW computes the Dynamic Time Warping distance between sequences a and b.
// It optionally returns the optimal alignment path if opts.ReturnPath is true.
//
// Time Complexity:    O(N·M)      where N=len(a), M=len(b)
// Memory Complexity:  O(N·M)      for FullMatrix mode
//
//	O(min(N,M)) for TwoRows and None modes (distance only)
//
// Returns:
//
//	dist float64    - cumulative minimal cost
//	path []Coord    - nil unless ReturnPath=true (and MemoryMode=FullMatrix)
//	err  error      - sentinel errors for input or options
func DTW(a, b []float64, opts Options) (dist float64, path []Coord, err error) {
	n, m := len(a), len(b)
	// Validate input lengths
	if n == 0 || m == 0 {
		return 0, nil, ErrEmptyInput
	}
	// Validate path requirements
	if opts.ReturnPath && opts.MemoryMode != FullMatrix {
		return 0, nil, ErrPathNeedsMatrix
	}
	// Alias options
	penalty := opts.SlopePenalty
	window := opts.Window
	mode := opts.MemoryMode
	needPath := opts.ReturnPath

	inf := math.Inf(1)
	// Allocate DP storage
	var dpFull [][]float64
	prev := make([]float64, m+1)
	curr := make([]float64, m+1)
	if mode == FullMatrix {
		dpFull = make([][]float64, n+1)
		dpFull[0] = make([]float64, m+1)
		copy(dpFull[0], prev)
	}
	// Initialize first row
	for j := 1; j <= m; j++ {
		prev[j] = inf
	}

	// Fill DP rows
	for i := 1; i <= n; i++ {
		// First column
		curr[0] = inf
		for j := 1; j <= m; j++ {
			// Sakoe-Chiba window constraint
			if window >= 0 && abs(i-j) > window {
				curr[j] = inf
				continue
			}
			// Local cost
			c := math.Abs(a[i-1] - b[j-1])
			// Recurrence: insertion, deletion, match
			if window >= 0 {
				// Chebyshev mode
				insR := prev[j]
				delR := curr[j-1]
				matR := prev[j-1]
				insV := math.Max(insR, penalty)
				delV := math.Max(delR, penalty)
				matV := math.Max(matR, c)
				curr[j] = min3(insV, delV, matV)
			} else {
				// Sum-of-cost mode
				ins := prev[j] + penalty
				del := curr[j-1] + penalty
				match := prev[j-1]
				best := min3(ins, del, match)
				curr[j] = c + best
			}
		}
		// Save row if needed
		if mode == FullMatrix {
			dpFull[i] = make([]float64, m+1)
			copy(dpFull[i], curr)
		}
		// Rotate buffers
		prev, curr = curr, prev
	}
	// Distance in last filled row
	dist = prev[m]
	// Backtrack for path if requested
	if needPath {
		path, err = backtrack(dpFull, a, b, opts)
	}
	return dist, path, err
}

// backtrack reconstructs the optimal path from dpFull matrix.
func backtrack(dp [][]float64, a, b []float64, opts Options) ([]Coord, error) {
	i, j := len(a), len(b)
	path := make([]Coord, 0, i+j)
	//inf := math.Inf(1)
	for i > 0 || j > 0 {
		// Append current alignment
		path = append(path, Coord{I: i - 1, J: j - 1})
		// Determine move: match vs insertion vs deletion
		moved := false
		if opts.Window >= 0 { // Chebyshev backtrack
			// match
			if i > 0 && j > 0 {
				c := math.Abs(a[i-1] - b[j-1])
				if almostEqual(dp[i][j], math.Max(dp[i-1][j-1], c)) {
					i, j = i-1, j-1
					moved = true
				}
			}
			// insertion
			if !moved && i > 0 {
				if almostEqual(dp[i][j], math.Max(dp[i-1][j], opts.SlopePenalty)) {
					i--
					moved = true
				}
			}
			// deletion
			if !moved && j > 0 {
				if almostEqual(dp[i][j], math.Max(dp[i][j-1], opts.SlopePenalty)) {
					j--
					moved = true
				}
			}
		} else { // Sum-of-cost backtrack
			// match
			if i > 0 && j > 0 {
				c := math.Abs(a[i-1] - b[j-1])
				if almostEqual(dp[i][j], dp[i-1][j-1]+c) {
					i, j = i-1, j-1
					moved = true
				}
			}
			// insertion
			if !moved && i > 0 {
				if almostEqual(dp[i][j], dp[i-1][j]+opts.SlopePenalty) {
					i--
					moved = true
				}
			}
			// deletion
			if !moved && j > 0 {
				if almostEqual(dp[i][j], dp[i][j-1]+opts.SlopePenalty) {
					j--
					moved = true
				}
			}
		}

		if !moved {
			return nil, ErrIncompletePath
		}
	}
	// Reverse path to start at (0,0)
	for l, r := 0, len(path)-1; l < r; l, r = l+1, r-1 {
		path[l], path[r] = path[r], path[l]
	}

	return path, nil
}

// min3 returns the minimum of three floats.
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

// abs returns the absolute value of an int.
func abs(x int) int {
	if x < 0 {
		return -x
	}

	return x
}

// almostEqual compares floats within a small epsilon.
func almostEqual(a, b float64) bool {
	const eps = 1e-9
	return math.Abs(a-b) <= eps
}
