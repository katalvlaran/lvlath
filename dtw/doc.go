// Package dtw computes Dynamic Time Warping (DTW) distances between two numeric time series,
// optionally returning the optimal alignment path, with configurable memory-time trade-offs.
//
// What:
//
//	DTW measures similarity by warping the time axis to minimize cumulative distance between series.
//	Commonly used in:
//	  • Speech/audio alignment
//	  • Gesture and motion matching
//	  • Time-series clustering and anomaly detection
//	  • Signature/handwriting verification
//
// Why:
//   - Handles sequences of unequal lengths or variable speed.
//   - Provides interpretable alignment path for post-analysis.
//   - Flexible memory usage for large-scale data.
//
// Complexity:
//   - Time:   O(N·M)  (N = len(a), M = len(b))
//   - Memory:
//   - FullMatrix: O(N·M) — exact distance + path
//   - TwoRows:    O(min(N, M)) — distance only
//   - None:       O(1)        — minimal overhead
//
// Options:
//
//	Options struct with fields:
//	  • Window       int         // Sakoe-Chiba band: max |i-j| allowed (<=0 = no constraint)
//	  • SlopePenalty float64     // cost for insertion/deletion steps; controls stretch bias
//	  • ReturnPath   bool        // include optimal path; requires MemoryMode=FullMatrix
//	  • MemoryMode   MemoryMode  // select None, TwoRows, or FullMatrix storage
//	Use DefaultOptions() for sensible defaults.
//
// Errors:
//   - ErrEmptyInput      — input sequences must be non-empty
//   - ErrPathNeedsMatrix — ReturnPath=true requires FullMatrix mode
//   - ErrIncompletePath  — backtracking failed to reach origin
//   - ErrBadInput        — invalid combination of options
//
// See examples in example_test.go and the tutorial in docs/DTW.md
// for detailed walkthrough and pseudocode.
package dtw
