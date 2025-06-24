// Package dtw defines options and sentinel errors for Dynamic Time Warping.
package dtw

import "errors"

// MemoryMode controls how DTW stores its DP matrix.
//
//   - None       — no matrix stored; only compute and return distance. Memory: O(1).
//   - TwoRows    — keep two rows (current and previous); compute distance only. Memory: O(min(N, M)).
//   - FullMatrix — store entire (n+1)x(m+1) matrix; supports path recovery. Memory: O(N·M).
//
// Use FullMatrix when you need to retrieve the optimal warping path.
type MemoryMode int

const (
	// None mode: minimal overhead, no path, constant memory.
	None MemoryMode = iota

	// TwoRows mode: rolling array of two rows, no path recovery.
	TwoRows

	// FullMatrix mode: full DP matrix, supports backtracking for path.
	FullMatrix
)

// Sentinel errors for DTW input validation and path requirements.
var (
	// ErrEmptyInput indicates one or both input sequences are empty.
	ErrEmptyInput = errors.New("dtw: input sequences must be non-empty")

	// ErrPathNeedsMatrix indicates ReturnPath=true requires FullMatrix mode.
	ErrPathNeedsMatrix = errors.New("dtw: ReturnPath requires MemoryMode=FullMatrix")

	// ErrIncompletePath indicates path backtrace failed to reach (0,0).
	ErrIncompletePath = errors.New("dtw: path computation incomplete")

	// ErrBadInput indicates an invalid combination of options.
	ErrBadInput = errors.New("dtw: invalid options combination")
)

// Options configures the Dynamic Time Warping algorithm.
//
// Fields:
//
//	Window       — Sakoe-Chiba band size: maximum |i-j| allowed. Window <= 0 means no constraint.
//	SlopePenalty — penalty cost for insertion/deletion steps; controls stretch bias.
//	ReturnPath   — if true, DTW backtracks and returns the optimal warping path.
//	                Requires MemoryMode=FullMatrix.
//	MemoryMode   — choose None, TwoRows, or FullMatrix for DP storage.
type Options struct {
	Window       int
	SlopePenalty float64
	ReturnPath   bool
	MemoryMode   MemoryMode
}

// DefaultOptions returns an Options struct with sensible defaults:
//
//	Window:       -1 (no window constraint)
//	SlopePenalty: 0.0
//	ReturnPath:   false
//	MemoryMode:   TwoRows
func DefaultOptions() Options {
	return Options{
		Window:       -1,
		SlopePenalty: 0.0,
		ReturnPath:   false,
		MemoryMode:   TwoRows,
	}
}
