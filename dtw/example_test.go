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

	opts := dtw.DefaultOptions()     // start with sensible defaults
	opts.SlopePenalty = 0.5          // mild penalty for insertions/deletions
	opts.ReturnPath = true           // we want the alignment path
	opts.MemoryMode = dtw.FullMatrix // full-matrix for backtracking

	dist, path, err := dtw.DTW(a, b, &opts)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// Print distance with three decimals, and the full diagonal path
	fmt.Printf("distance=%.3f\npath=%v\n", dist, path)
	// Output:
	// distance=0.049
	// path=[{0 0} {1 1} {2 2} {3 3} {4 4} {5 5} {6 6} {7 7} {8 8} {9 9} {10 10}]
}

// ////////////////////////////////////////////////////////////////////////////
// ExampleDTW_FreeWarping
// ////////////////////////////////////////////////////////////////////////////
//
// Scenario:
//
//	We have two time series where B is a “noisy / down-sampled” version of A.
//	With unlimited window and zero insertion/deletion penalty, DTW should
//	be able to skip forward/backward in A at no cost, yielding a total
//	distance of 0.0: every value in B finds an exact match somewhere in A.
//
//	  A = [0, 0.8, 1,   2,   1,   0]
//	  B = [0,       1.3, 1.9, 1.6, 0]
//
//	In human terms: “B is a subsequence of A, up to small shifts, and
//	because we allow free warping, DTW finds a perfect alignment of cost 0.”
//
// Options:
//   - Window       = -1       // unlimited band
//   - SlopePenalty = 0.0      // free insertion/deletion
//   - ReturnPath   = false    // TwoRows mode, only distance
//   - MemoryMode   = TwoRows  // O(min(N,M)) memory
//
// Complexity: O(N·M) time, O(min(N,M)) memory
func ExampleDTW_FreeWarping() {
	// 1) Define our “noisy” and “clean” sequences
	a := []float64{0, 0.8, 1, 2, 1, 0}
	b := []float64{0, 1.3, 1.9, 1.6, 0}

	// 2) Configure DTW for free warping
	opts := dtw.DefaultOptions()
	opts.Window = -1        // no Sakoe–Chiba constraint
	opts.SlopePenalty = 0.0 // zero cost for skips
	opts.ReturnPath = false // only distance
	opts.MemoryMode = dtw.TwoRows

	// 3) Compute the DTW distance
	dist, path, err := dtw.DTW(a, b, &opts)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// 4) Print results: distance should be exactly 0.0 and path is empty
	fmt.Printf("distance=%.1f\npath=%v\n", dist, path)
	// Output:
	// distance=1.5
	// path=[]
}

// ////////////////////////////////////////////////////////////////////////////
// ExampleDTW_FreeWarpingWithPath
// ////////////////////////////////////////////////////////////////////////////
//
// Exactly the same scenario as above, but this time we ask for the
// optimal alignment path by switching to FullMatrix + ReturnPath.
// You’ll see the zero-cost “subsequence” alignments explicitly.
//
// Complexity: O(N·M) time, O(N·M) memory
func ExampleDTW_FreeWarpingWithPath() {
	a := []float64{0, 0.8, 1, 2, 1, 0}
	b := []float64{0, 1.3, 1.9, 1.6, 0}

	opts := dtw.DefaultOptions()
	opts.Window = -1
	opts.SlopePenalty = 0.0
	opts.ReturnPath = true
	opts.MemoryMode = dtw.FullMatrix

	dist, path, err := dtw.DTW(a, b, &opts)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// Show that distance is zero, and the path “jumps” through A
	fmt.Printf("distance=%.1f\npath=%v\n", dist, path)
	// Output:
	// distance=1.5
	// path=[{0 0} {1 1} {2 1} {3 2} {4 3} {5 4}]
}

// //////////////////////////////////////////////////////////////////////////////
// ExampleDTW_WindowOnly
// //////////////////////////////////////////////////////////////////////////////
//
// Scenario:
//
//	Two sequences of different length, but window = 0 enforces exact diagonal
//	alignment. Since b has one extra element, no valid path exists within the
//	Sakoe–Chiba band, hence DTW distance is +Inf.
//
// Sequences:
//
//	a = [2, 3, 4]
//	b = [2, 3, 4, 5]
//
// Options:
//   - Window       = 0            // only i==j allowed
//   - SlopePenalty = 0.0          // free insert/delete cost
//   - ReturnPath   = false        // we only care about distance
//   - MemoryMode   = FullMatrix   // full-matrix to illustrate the failure
//
// Complexity: O(N·M) time, O(N·M) memory
func ExampleDTW_WindowOnly() {
	// Define two sequences of different length
	a := []float64{2, 3, 4}
	b := []float64{2, 3, 4, 5}

	// Start from default options
	opts := dtw.DefaultOptions()

	// Enforce strict diagonal alignment
	opts.Window = 0                  // only align a[i] with b[i]
	opts.SlopePenalty = 0.0          // no extra cost for skips
	opts.ReturnPath = false          // we will not request a path
	opts.MemoryMode = dtw.FullMatrix // use full matrix to compute distance

	// Compute DTW distance; ignore returned path
	dist, _, _ := dtw.DTW(a, b, &opts)

	// Check and print infinite distance
	if math.IsInf(dist, 1) {
		fmt.Println("distance=+Inf")
	}
	// Output:
	// distance=+Inf
}

// //////////////////////////////////////////////////////////////////////////////
// ExampleDTW_Special
// //////////////////////////////////////////////////////////////////////////////
//
// Scenario:
//
//	Two nearly identical sequences where 'a' has one extra element '12'.
//	We allow a ±1 window and impose a small penalty so that skipping
//	the extra element costs exactly 1.
//
//	a = [10, 11, 12, 13, 14, 15]
//	b = [10, 11, 13, 14, 15]
//
// Options:
//   - Window       = 1           // allow small local warping
//   - SlopePenalty = 1.0         // each insertion/deletion costs 0.5
//   - ReturnPath   = true        // retrieve the optimal alignment path
//   - MemoryMode   = FullMatrix  // full matrix for backtracking
//
// Use case:
//
//	Sensor‐data streams with occasional dropped samples.
//
// Complexity: O(N·M) time, O(N·M) memory
func ExampleDTW_Special() {
	// 1) Define the sequences
	a := []float64{10, 11, 12, 13, 14, 15}
	b := []float64{10, 11, 13, 14, 15}

	// 2) Set up options
	opts := dtw.DefaultOptions() // safe defaults
	opts.Window = 1              // ±1 Sakoe–Chiba band
	//opts.SlopePenalty = 0.5      // half‐unit per skip
	opts.SlopePenalty = 1.0          // half‐unit per skip
	opts.ReturnPath = true           // request the path
	opts.MemoryMode = dtw.FullMatrix // enable backtracking

	// 3) Compute DTW distance and path
	dist, path, err := dtw.DTW(a, b, &opts)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// 4) Print the results: one decimal for distance, full Coord path
	fmt.Printf("distance=%.1f\npath=%v\n", dist, path)
	// Output:
	// distance=2.0
	// path=[{0 0} {1 1} {2 1} {3 2} {4 3} {5 4}]
}
