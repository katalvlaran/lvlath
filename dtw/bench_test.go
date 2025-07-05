package dtw_test

import (
	"testing"

	"github.com/katalvlaran/lvlath/dtw"
)

// benchmarkDTW is a helper that runs DTW on sequences of lengths n and m using opts.
// It resets the timer before entering the loop and fails on unexpected errors.
func benchmarkDTW(b *testing.B, n, m int, opts dtw.Options) {
	// Prepare two sequences a and b of specified lengths
	a := make([]float64, n)
	bSeq := make([]float64, m)
	for i := 0; i < n; i++ {
		a[i] = float64(i) // fill a with predictable increasing values
	}
	for j := 0; j < m; j++ {
		bSeq[j] = float64(j) // fill b with predictable increasing values
	}

	b.ResetTimer() // ignore setup time
	for i := 0; i < b.N; i++ {
		_, _, err := dtw.DTW(a, bSeq, &opts) // run DTW
		if err != nil {
			b.Fatalf("DTW failed: %v", err) // report and stop on error
		}
	}
}

// BenchmarkDTW_FullMatrixSmall benchmarks FullMatrix mode on small 100×100 sequences.
func BenchmarkDTW_FullMatrixSmall(b *testing.B) {
	// Default options, full matrix to enable backtracking
	opts := dtw.DefaultOptions()
	opts.MemoryMode = dtw.FullMatrix
	benchmarkDTW(b, 100, 100, opts)
}

// BenchmarkDTW_FullMatrixMedium benchmarks FullMatrix mode on medium 500×500 sequences.
func BenchmarkDTW_FullMatrixMedium(b *testing.B) {
	opts := dtw.DefaultOptions()
	opts.MemoryMode = dtw.FullMatrix
	benchmarkDTW(b, 500, 500, opts)
}

// BenchmarkDTW_TwoRowsSmall benchmarks TwoRows (rolling array) mode on small 100×100 sequences.
func BenchmarkDTW_TwoRowsSmall(b *testing.B) {
	opts := dtw.DefaultOptions()
	opts.MemoryMode = dtw.TwoRows
	benchmarkDTW(b, 100, 100, opts)
}

// BenchmarkDTW_TwoRowsMedium benchmarks TwoRows mode on medium 500×500 sequences.
func BenchmarkDTW_TwoRowsMedium(b *testing.B) {
	opts := dtw.DefaultOptions()
	opts.MemoryMode = dtw.TwoRows
	benchmarkDTW(b, 500, 500, opts)
}

// BenchmarkDTW_NoMemorySmall benchmarks NoMemory mode on small 100×100 sequences.
func BenchmarkDTW_NoMemorySmall(b *testing.B) {
	opts := dtw.DefaultOptions()
	opts.MemoryMode = dtw.NoMemory
	benchmarkDTW(b, 100, 100, opts)
}

// BenchmarkDTW_NoMemoryMedium benchmarks NoMemory mode on medium 500×500 sequences.
func BenchmarkDTW_NoMemoryMedium(b *testing.B) {
	opts := dtw.DefaultOptions()
	opts.MemoryMode = dtw.NoMemory
	benchmarkDTW(b, 500, 500, opts)
}

// BenchmarkDTW_WindowConstraint benchmarks FullMatrix with a strict window on mismatched lengths
func BenchmarkDTW_WindowConstraint(b *testing.B) {
	opts := dtw.DefaultOptions()
	opts.MemoryMode = dtw.FullMatrix
	opts.Window = 0 // only diagonal
	// Sequence lengths differ by 1 to force +Inf cost frequently
	benchmarkDTW(b, 100, 101, opts)
}
