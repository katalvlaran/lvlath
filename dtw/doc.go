// Package dtw computes Dynamic Time Warping (DTW) distances between
// numeric time series, with optional alignment path and memory optimizations.
//
// 🚀 What is DTW?
//
//	DTW finds the best match between two sequences by warping the time
//	axis to minimize cumulative distance.  It’s widely used in:
//	  • Speech recognition & audio alignment
//	  • Gesture / motion matching
//	  • Signature & handwriting verification
//	  • Time-series clustering & anomaly detection
//
// ✨ Key features:
//   - full-matrix mode: exact O(N·M) time & memory
//   - rolling mode: O(min(N,M)) memory (choose via MemoryMode)
//   - optional Sakoe–Chiba window (|i−j| ≤ w) for speed & constraint
//   - slope penalty to discourage excessive stretching
//   - on-demand alignment path (ReturnPath=true)
//
// ⚙️ Usage:
//
//	import "github.com/katalvlaran/lvlath/dtw"
//
//	opts := &dtw.DTWOptions{
//	  Window:       10,     // Sakoe–Chiba band ±10
//	  SlopePenalty: 0.5,    // penalty for 1×2 vs 2×1 steps
//	  ReturnPath:   true,   // also return warp path
//	  MemoryMode:   dtw.Rolling,
//	}
//
//	// compute
//	dist, path, err := dtw.DTW(a, b, opts)
//
// Performance:
//
//   - Time:   O(N·M)
//   - Memory: O(N·M) (FullMatrix) or O(min(N,M)) (Rolling)
//
// See examples in example_test.go and the tutorial in docs/TUTORIAL.md
// for detailed walkthrough and pseudocode.
package dtw
