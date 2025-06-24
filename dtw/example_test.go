package dtw_test

import (
	"fmt"
	"math"

	"github.com/katalvlaran/lvlath/dtw"
)

// //////////////////////////////////////////////////////////////////////////////
// ExampleDTW_medium
// //////////////////////////////////////////////////////////////////////////////
//
// Scenario:
//
//	Compare two moderately sized sequences with slight pacing differences.
//	  a = [1, 3, 4, 9, 8]
//	  b = [1, 4, 5, 9, 7]
//
// Options:
//   - Window = 1         (Sakoe-Chiba band → allow ±1 offset)
//   - SlopePenalty = 0.5 (penalize uneven stretches)
//   - ReturnPath = true  (retrieve alignment path)
//   - MemoryMode = FullMatrix (O(N·M) mem)
//
// Use case:
//
//	Time-series similarity when small local shifts are expected.
//
// Complexity: O(N·M) time, O(N·M) memory
//
// ExampleDTW_Medium demonstrates DTW with a small Sakoe-Chiba window.
// Playground: [![Playground - DTW](https://img.shields.io/badge/Go_Playground-DTW-blue?logo=go)](https://play.golang.org/p/dtw)
func ExampleDTW_Medium() {
	a := []float64{4.199, 4.170, 4.190, 4.080, 4.110, 4.092, 4.080, 4.101, 4.121, 4.071, 4.001}
	b := []float64{4.200, 4.171, 4.185, 4.087, 4.103, 4.098, 4.083, 4.110, 4.117, 4.076, 4.000}
	opts := dtw.DefaultOptions()
	//opts.Window = 1
	opts.SlopePenalty = 0.5
	opts.ReturnPath = true
	opts.MemoryMode = dtw.FullMatrix

	dist, path, err := dtw.DTW(a, b, opts)
	if err != nil {
		fmt.Println("error:", err)

		return
	}
	fmt.Printf("distance=%.0f\nlen(path)=%d\npath=%v\n", dist, len(path), path)
	// Output:
	// distance=0.50
	// path=[{0 0} {0 1} {1 1} {1 2} {2 2} {3 3} {3 4} {4 4}]

	// distance=0
	// len(path)=11
	// path=[{0 0} {1 1} {2 2} {3 3} {4 4} {5 5} {6 6} {7 7} {8 8} {9 9} {10 10}]
}

// //////////////////////////////////////////////////////////////////////////////
// ExampleDTW_medium2
// //////////////////////////////////////////////////////////////////////////////
//
// Scenario:
//
//	Compare two medium-length sequences with repetitions.
//	  a = [0, 0, 1, 2, 1, 0]
//	  b = [0, 1, 1, 1, 0]
//
// Options:
//   - No window constraint (Window = -1 → unlimited)
//   - No slope penalty (SlopePenalty = 0)
//   - ReturnPath = true
//   - MemoryMode = Rolling (O(min(N,M)) mem)
//
// Use case:
//
//	Speech pattern matching where insertions/deletions are free.
//
// Complexity: O(N·M) time, O(min(N,M)) memory
// ExampleDTW_MediumPlus demonstrates unlimited window and rolling memory.
// Memory: O(min(N,M)).
func ExampleDTW_MediumPlus() {
	a := []float64{0, 0, 1, 2, 1, 0}
	b := []float64{0, 1, 1, 1, 0}
	opts := dtw.DefaultOptions()
	opts.Window = -1
	opts.SlopePenalty = 0
	opts.ReturnPath = true
	opts.MemoryMode = dtw.TwoRows

	dist, path, err := dtw.DTW(a, b, opts)
	if err != nil {
		fmt.Println("error:", err)

		return
	}
	fmt.Printf("distance=%.0f\npath=%v\n", dist, path)
	// Output:
	// distance=1
	// path=[{0 0} {1 0} {2 1} {3 2} {4 3} {5 4}]
}

// //////////////////////////////////////////////////////////////////////////////
// ExampleDTW_medium_window
// //////////////////////////////////////////////////////////////////////////////
//
// Scenario:
//
//	Short sequences with strict alignment.
//	  a = [2, 3, 4]
//	  b = [2, 3, 4, 5]
//
// Options:
//   - Window = 0         (exact diagonal only)
//   - SlopePenalty = 0
//   - ReturnPath = false
//   - MemoryMode = FullMatrix
//
// Effect:
//
//	Strict window forces INF when lengths differ by >0.
//
// Complexity: O(N·M) time, O(N·M) memory
// ExampleDTW_WindowOnly shows strict diagonal alignment forcing infinite distance.
func ExampleDTW_WindowOnly() {
	a := []float64{2, 3, 4}
	b := []float64{2, 3, 4, 5}
	opts := dtw.DefaultOptions()
	opts.Window = 0
	opts.ReturnPath = false
	opts.MemoryMode = dtw.FullMatrix

	dist, _, _ := dtw.DTW(a, b, opts)
	if math.IsInf(dist, 1) {
		fmt.Println("distance=+Inf")
	}
	// Output:
	// distance=+Inf
}

// //////////////////////////////////////////////////////////////////////////////
// ExampleDTW_special
// //////////////////////////////////////////////////////////////////////////////
//
// Scenario:
//
//	Almost identical long-ish sequences with a one-point shift.
//	  a = [10, 11, 12, 13, 14, 15]
//	  b = [10, 11, 13, 14, 15]
//
// Options:
//   - Window = 1
//   - SlopePenalty = 1.0   (higher cost for insertion/deletion)
//   - ReturnPath = true
//   - MemoryMode = FullMatrix
//
// Use case:
//
//	Sensor data alignment where a single missing measurement incurs a penalty.
//
// Complexity: O(N·M) time, O(N·M) memory
// ExampleDTW_Special demonstrates a penalty for a missing element.
func ExampleDTW_Special() {
	a := []float64{10, 11, 12, 13, 14, 15}
	b := []float64{10, 11, 13, 14, 15}
	opts := dtw.DefaultOptions()
	opts.Window = 1
	opts.SlopePenalty = 1.0
	opts.ReturnPath = true
	opts.MemoryMode = dtw.FullMatrix

	dist, path, _ := dtw.DTW(a, b, opts)
	fmt.Printf("distance=%.0f\npath=%v\n", dist, path)
	// Output:
	// distance=1
	// path=[{0 0} {1 0} {2 1} {3 2} {4 3} {5 4}]
}
