// Package main demonstrates aligning two noisy audio-like signals
// using Dynamic Time Warping (DTW) with lvlath/dtw.
//
// Playground: https://go.dev/play/p/O5bmqNKdxPm
//
// Scenario:
//
//	We have a clean “sine‐like” waveform `clean` and a noisy version `noisy`
//	where each sample is perturbed by random noise.  We use DTW to find
//	the best alignment path and measure similarity despite the noise.
//
// Signals (amplitude values):
//
//	clean = [0.0, 1.0, 0.0, -1.0, 0.0]
//	noisy = [0.1, 0.9, 0.2, -0.8, 0.1]
//
// We configure:
//   - Window = 1         → only align samples within ±1 index (Sakoe-Chiba band)
//   - SlopePenalty = 0.5 → penalize large insertions/deletions
//   - ReturnPath = true  → recover the alignment path
//   - MemoryMode = FullMatrix
//
// Use case:
//
//	Audio signal comparison & synchronization under noise.
//
// Complexity: O(N·M) time, O(N·M) memory (FullMatrix).
package main

//
//import (
//	"fmt"
//
//	"github.com/katalvlaran/lvlath/dtw"
//)
//
//func main6() {
//	// 1) Define two "audio" signals: clean vs. noisy
//	clean := []float64{0.0, 1.0, 0.0, -1.0, 0.0}
//	noisy := []float64{0.1, 0.9, 0.2, -0.8, 0.1}
//
//	// 2) DTW options: small window, moderate penalty, full matrix for path
//	opts := &dtw.DTWOptions{
//		Window:       1,
//		SlopePenalty: 0.5,
//		ReturnPath:   true,
//		MemoryMode:   dtw.FullMatrix,
//	}
//
//	// 3) Compute DTW distance and alignment path
//	dist, path, err := dtw.DTW(clean, noisy, opts)
//	if err != nil {
//		panic(fmt.Sprintf("DTW error: %v", err))
//	}
//
//	// 4) Interpret results
//	fmt.Printf("🎯 DTW distance between clean & noisy: %.3f\n", dist)
//	fmt.Println("🔗 Optimal alignment (clean_index → noisy_index):")
//	for _, p := range path {
//		fmt.Printf("  %d → %d\n", p[0], p[1])
//	}
//
//	// 5) Visual ASCII plot of alignment (optional)
//	fmt.Println("\nSignal alignment:")
//	for i, j := range path {
//		c := clean[i]
//		n := noisy[j[1]]
//		fmt.Printf(" [%2d] clean=%.1f  ↔  noisy=%.1f\n", i, c, n)
//	}
//}
