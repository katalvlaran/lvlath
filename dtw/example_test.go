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
//   - Window = 1         (Sakoe–Chiba band → allow ±1 offset)
//   - SlopePenalty = 0.5 (penalize uneven stretches)
//   - ReturnPath = true  (retrieve alignment path)
//   - MemoryMode = FullMatrix (O(N·M) mem)
//
// Use case:
//
//	Time-series similarity when small local shifts are expected.
//
// Complexity: O(N·M) time, O(N·M) memory
func ExampleDTW_medium() {
	a := []float64{1, 3, 4, 9, 8}
	b := []float64{1, 4, 5, 9, 7}
	opts := &dtw.DTWOptions{
		Window:       1,
		SlopePenalty: 0.5,
		ReturnPath:   true,
		MemoryMode:   dtw.FullMatrix,
	}

	dist, path, err := dtw.DTW(a, b, opts)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("distance=%.2f\npath=%v\n", dist, path)
	// Output:
	// distance=1.00
	// path=[[0 0] [1 1] [2 2] [3 3] [4 4]]
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
func ExampleDTW_medium2() {
	a := []float64{0, 0, 1, 2, 1, 0}
	b := []float64{0, 1, 1, 1, 0}
	opts := &dtw.DTWOptions{
		Window:       -1, // no band
		SlopePenalty: 0,  // no penalty
		ReturnPath:   true,
		MemoryMode:   dtw.RollingArray, // reduced memory
	}

	dist, path, err := dtw.DTW(a, b, opts)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("distance=%.0f\npath=%v\n", dist, path)
	// Output:
	// distance=1
	// path=[[0 0] [1 0] [2 1] [3 2] [4 3] [5 4]]
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
func ExampleDTW_medium_window() {
	a := []float64{2, 3, 4}
	b := []float64{2, 3, 4, 5}
	opts := &dtw.DTWOptions{
		Window:     0,
		ReturnPath: false,
		MemoryMode: dtw.FullMatrix,
	}

	dist, _, _ := dtw.DTW(a, b, opts)
	if math.IsInf(dist, 1) {
		fmt.Println("distance=+Inf")
	}
	// Output: distance=+Inf
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
func ExampleDTW_special() {
	a := []float64{10, 11, 12, 13, 14, 15}
	b := []float64{10, 11, 13, 14, 15}
	opts := &dtw.DTWOptions{
		Window:       1,
		SlopePenalty: 1.0,
		ReturnPath:   true,
		MemoryMode:   dtw.FullMatrix,
	}

	dist, path, _ := dtw.DTW(a, b, opts)
	fmt.Printf("distance=%.0f\npath=%v\n", dist, path)
	// Output:
	// distance=1
	// path=[[0 0] [1 1] [2 1] [3 2] [4 3] [5 4]]
}
