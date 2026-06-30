package tsp

import "testing"

func BenchmarkBlossomDenseK64(b *testing.B) {
	problem := internalSeededMatchingProblem(64, 6400)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		match, _, _, err := solveMinimumWeightPerfectMatching(problem, blossomOptions{Eps: DefaultEps})
		if err != nil {
			b.Fatalf("blossom: %v", err)
		}
		if err = verifyPerfectMatching(match); err != nil {
			b.Fatalf("verify: %v", err)
		}
	}
}

func BenchmarkBlossomDenseK128(b *testing.B) {
	problem := internalSeededMatchingProblem(128, 12800)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		match, _, _, err := solveMinimumWeightPerfectMatching(problem, blossomOptions{Eps: DefaultEps})
		if err != nil {
			b.Fatalf("blossom: %v", err)
		}
		if err = verifyPerfectMatching(match); err != nil {
			b.Fatalf("verify: %v", err)
		}
	}
}
