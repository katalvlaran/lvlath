// Package dtw defines configuration options and sentinel errors
// for the Dynamic Time Warping algorithm.
package dtw

import "errors" // we need sentinel error creation

// MemoryMode controls how much of the DP matrix DTW stores.
//
//   - NoMemory   - constant O(1) memory, compute distance only.
//   - TwoRows    - O(min(N,M)) memory, keep two rows, distance only.
//   - FullMatrix - O(N*M) memory, store entire matrix, supports backtracking.
type MemoryMode int

const (
	// NoMemory: minimal overhead, no path recovery, constant memory.
	NoMemory MemoryMode = iota

	// TwoRows: rolling two-row storage, no path recovery.
	TwoRows

	// FullMatrix: full DP matrix storage, enables optimal path recovery.
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
//	Window       - Sakoe-Chiba band size: maximum |i-j| allowed. Window <= 0 means no constraint.
//	SlopePenalty - penalty cost for insertion/deletion steps; controls stretch bias.
//	ReturnPath   - if true, DTW backtracks and returns the optimal warping path.
//	                Requires MemoryMode=FullMatrix.
//	MemoryMode   - choose NoMemory, TwoRows, or FullMatrix for DP storage.
type Options struct {
	Window       int
	SlopePenalty float64
	ReturnPath   bool
	MemoryMode   MemoryMode
}

// DefaultOptions returns an Options struct pre-populated with safe defaults.
//
//	Window:       -1       // no window constraint
//	SlopePenalty: 0.0      // free insertions/deletions
//	ReturnPath:   false    // only distance, no path
//	MemoryMode:   TwoRows  // minimal extra memory
func DefaultOptions() Options {
	return Options{
		Window:       -1,
		SlopePenalty: 0.0,
		ReturnPath:   false,
		MemoryMode:   TwoRows,
	}
}

// Validate checks that Options fields hold a valid combination.
// It returns ErrBadInput if Window < -1 or SlopePenalty < 0,
// and ErrPathNeedsMatrix if ReturnPath=true but MemoryMode!=FullMatrix.
func (o *Options) Validate() error {
	// Window must be -1 (disabled) or ≥ 0
	if o.Window < -1 {
		return ErrBadInput
	}
	// Penalty must be non-negative
	if o.SlopePenalty < 0 {
		return ErrBadInput
	}
	// If a path is requested, we need full-matrix storage
	if o.ReturnPath && o.MemoryMode != FullMatrix {
		return ErrPathNeedsMatrix
	}
	return nil
}
