package dtw

import (
	"errors"
	"math"
)

// DTW — Dynamic Time Warping
//
// Description:
//
//	DTW measures similarity between two sequences that may vary
//	in time or speed by finding an optimal “warping path”.
//	It is widely used in speech recognition, time-series analysis,
//	gesture recognition, and many other domains.
//
// Algorithm Outline (Full-Matrix):
//  1. Let n = len(a), m = len(b). Allocate (n+1)x(m+1) DP matrix D.
//  2. Initialize:
//     D[0][0] = 0
//     D[i][0] = +∞ for i=1..n
//     D[0][j] = +∞ for j=1..m
//  3. For i = 1..n:
//     For j = 1..m (and |i-j| ≤ Window, if constrained):
//     cost = |a[i-1] - b[j-1]|
//     ins   = D[i-1][j]   + SlopePenalty
//     del   = D[i][j-1]   + SlopePenalty
//     match = D[i-1][j-1]
//     D[i][j] = cost + min(ins, del, match)
//  4. distance = D[n][m].
//  5. If ReturnPath && MemoryMode==FullMatrix, backtrack from (n,m) to (0,0)
//     following the predecessor with minimal D-value.
//
// Memory Modes:
//   - FullMatrix   — store full D, support ReturnPath. Memory: O(n·m).
//   - RollingArray — store only two rows (current & previous). Memory: O(min(n,m)).
//     ReturnPath is not supported.
//
// Complexity:
//
//	Time   = O(n·m)
//	Memory = O(n·m) (FullMatrix) or O(min(n,m)) (RollingArray)
//
// Errors:
//   - ErrEmptySequence         — if either input is empty.
//   - ErrPathNeedsFullMatrix   — if ReturnPath=true with RollingArray mode.
var (
	// ErrEmptySequence indicates one or both inputs are empty.
	ErrEmptySequence = errors.New("dtw: input sequences must be non-empty")

	// ErrPathNeedsFullMatrix indicates that path recovery requires FullMatrix mode.
	ErrPathNeedsFullMatrix = errors.New("dtw: ReturnPath requires MemoryMode=FullMatrix")
)

// DTW computes the Dynamic Time Warping distance between a and b.
// Returns (distance, path, error).
//
// If opts.ReturnPath is true, opts.MemoryMode must be FullMatrix.
//
// Example:
//
//	opts := &DTWOptions{ReturnPath: true, MemoryMode: FullMatrix}
//	dist, path, err := DTW(seqA, seqB, opts)
func DTW(a, b []float64, opts *DTWOptions) (distance float64, path [][2]int, err error) {
	n, m := len(a), len(b)
	if n == 0 || m == 0 {
		return 0, nil, ErrEmptySequence
	}

	// Apply options or defaults
	window := math.MaxInt32
	penalty := 0.0
	mem := FullMatrix
	wantPath := false
	if opts != nil {
		if opts.Window > 0 {
			window = opts.Window
		}
		penalty = opts.SlopePenalty
		mem = opts.MemoryMode
		wantPath = opts.ReturnPath
	}
	if wantPath && mem != FullMatrix {
		return 0, nil, ErrPathNeedsFullMatrix
	}

	// Prepare DP storage
	var dp [][]float64
	if mem == FullMatrix {
		dp = make([][]float64, n+1)
		for i := range dp {
			dp[i] = make([]float64, m+1)
		}
	} else {
		dp = make([][]float64, 2)
		dp[0] = make([]float64, m+1)
		dp[1] = make([]float64, m+1)
	}
	inf := math.Inf(1)

	// Initialize first row/col
	if mem == FullMatrix {
		for i := 1; i <= n; i++ {
			dp[i][0] = inf
		}
		for j := 1; j <= m; j++ {
			dp[0][j] = inf
		}
	} else {
		for j := 1; j <= m; j++ {
			dp[0][j] = inf
		}
	}

	// Fill DP
	for i := 1; i <= n; i++ {
		curr, prev := i%2, (i-1)%2
		if mem == RollingArray {
			dp[curr][0] = inf
		}
		for j := 1; j <= m; j++ {
			if window < math.MaxInt32 && abs(i-j) > window {
				if mem == FullMatrix {
					dp[i][j] = inf
				} else {
					dp[curr][j] = inf
				}
				continue
			}
			cost := math.Abs(a[i-1] - b[j-1])
			var ins, del, match float64
			if mem == FullMatrix {
				ins = dp[i-1][j] + penalty
				del = dp[i][j-1] + penalty
				match = dp[i-1][j-1]
			} else {
				ins = dp[prev][j] + penalty
				del = dp[curr][j-1] + penalty
				match = dp[prev][j-1]
			}
			best := min3(ins, del, match)
			if mem == FullMatrix {
				dp[i][j] = cost + best
			} else {
				dp[curr][j] = cost + best
			}
		}
	}

	// Extract final distance
	if mem == FullMatrix {
		distance = dp[n][m]
	} else {
		distance = dp[n%2][m]
	}

	// Backtrack path if requested
	if wantPath {
		i, j := n, m
		for i > 0 || j > 0 {
			path = append(path, [2]int{i - 1, j - 1})
			prevCost := dp[i][j] - math.Abs(a[i-1]-b[j-1])
			// choose predecessor
			if i > 0 && dp[i-1][j] == prevCost-penalty {
				i--
			} else if j > 0 && dp[i][j-1] == prevCost-penalty {
				j--
			} else {
				i--
				j--
			}
		}
		// reverse path in-place
		for l, r := 0, len(path)-1; l < r; l, r = l+1, r-1 {
			path[l], path[r] = path[r], path[l]
		}
	}

	return distance, path, nil
}

// abs returns the absolute value of an int.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
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
