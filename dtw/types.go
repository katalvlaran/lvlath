// Package dtw defines options and modes for Dynamic Time Warping.
package dtw

// MemoryMode controls how DTW stores its DP matrix.
//
//   - FullMatrix   — keep the entire (n+1)x(m+1) matrix in memory.
//     Allows distance + full backtrace for the optimal warping path.
//     Memory: O(n·m).
//
//   - RollingArray — only keep two rows (current and previous).
//     Greatly reduces memory to O(min(n, m)), but cannot recover the path.
//     Use when you only need the distance.
type MemoryMode int

const (
	// FullMatrix mode: store all rows, support path recovery, uses O(N·M) memory.
	FullMatrix MemoryMode = iota

	// RollingArray mode: keep only two rows, no path recovery, uses O(min(N,M)) memory.
	RollingArray
)

// DTWOptions configures Dynamic Time Warping.
//
// Fields:
//   - Window       — maximum deviation |i-j| allowed (Sakoe–Chiba band).
//     A value of 0 (or negative) means no windowing constraint.
//   - SlopePenalty — penalty cost for insertion/deletion steps (controls locality bias).
//   - ReturnPath   — if true, DTW will backtrack and return the optimal warping path.
//     Requires MemoryMode=FullMatrix.
//   - MemoryMode   — choose FullMatrix or RollingArray storage.
//
// Example:
//
//	opts := &DTWOptions{
//	  Window:       10,           // only compare elements within ±10 steps
//	  SlopePenalty: 0.5,          // small penalty for non-diagonal moves
//	  ReturnPath:   true,         // we need the path, not just the distance
//	  MemoryMode:   FullMatrix,   // must be FullMatrix to support ReturnPath
//	}
//
//	dist, path, err := DTW(seqA, seqB, opts)
//	if err != nil {
//	  // handle ErrEmptySequence or ErrPathNeedsFullMatrix
//	}
//	fmt.Println("DTW distance:", dist)
//	fmt.Println("Warping path:", path)
type DTWOptions struct {
	Window       int
	SlopePenalty float64
	ReturnPath   bool
	MemoryMode   MemoryMode
}
