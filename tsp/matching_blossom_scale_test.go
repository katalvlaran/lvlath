package tsp

import "testing"

func TestBlossomLargeDenseScalesDeterministically(t *testing.T) {
	cases := []struct {
		name string
		k    int
		seed int64
	}{
		{name: "k64", k: 64, seed: 6400},
		{name: "k96", k: 96, seed: 9600},
		{name: "k128", k: 128, seed: 12800},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			problem := internalSeededMatchingProblem(tc.k, tc.seed)

			first, firstCost, firstStats, err := solveMinimumWeightPerfectMatching(problem, blossomOptions{Eps: DefaultEps})
			if err != nil {
				t.Fatalf("first run: %v stats=%+v", err, firstStats)
			}
			if err = verifyPerfectMatching(first); err != nil {
				t.Fatalf("first verify: %v", err)
			}

			second, secondCost, secondStats, err := solveMinimumWeightPerfectMatching(problem, blossomOptions{Eps: DefaultEps})
			if err != nil {
				t.Fatalf("second run: %v stats=%+v", err, secondStats)
			}
			if err = verifyPerfectMatching(second); err != nil {
				t.Fatalf("second verify: %v", err)
			}

			if firstCost != secondCost {
				t.Fatalf("cost mismatch got %.12f want %.12f firstStats=%+v secondStats=%+v",
					secondCost, firstCost, firstStats, secondStats)
			}

			for vertex := range first {
				if first[vertex] != second[vertex] {
					t.Fatalf("match[%d] got %d want %d firstStats=%+v secondStats=%+v",
						vertex, second[vertex], first[vertex], firstStats, secondStats)
				}
			}

			if firstStats.Augmentations != tc.k/2 {
				t.Fatalf("augmentations got %d want %d stats=%+v", firstStats.Augmentations, tc.k/2, firstStats)
			}
		})
	}
}

func TestBlossomLargeK64ProducesPerfectMatchingDeterministically(t *testing.T) {
	problem := internalSeededMatchingProblem(64, 6400)

	first, firstCost, firstStats, err := solveMinimumWeightPerfectMatching(problem, blossomOptions{Eps: DefaultEps})
	if err != nil {
		t.Fatalf("first Blossom run: %v stats=%+v", err, firstStats)
	}
	if err = verifyPerfectMatching(first); err != nil {
		t.Fatalf("first verify: %v match=%v stats=%+v", err, first, firstStats)
	}

	for run := 0; run < 5; run++ {
		got, gotCost, stats, runErr := solveMinimumWeightPerfectMatching(problem, blossomOptions{Eps: DefaultEps})
		if runErr != nil {
			t.Fatalf("run %d: %v stats=%+v", run, runErr, stats)
		}
		if err = verifyPerfectMatching(got); err != nil {
			t.Fatalf("run %d verify: %v match=%v stats=%+v", run, err, got, stats)
		}
		if gotCost != firstCost {
			t.Fatalf("run %d cost got %.12f want %.12f stats=%+v firstStats=%+v", run, gotCost, firstCost, stats, firstStats)
		}
		for vertex := range got {
			if got[vertex] != first[vertex] {
				t.Fatalf("run %d match[%d]=%d want %d stats=%+v firstStats=%+v",
					run, vertex, got[vertex], first[vertex], stats, firstStats)
			}
		}
	}
}

func TestBlossomLargeDenseK192K256Smoke(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large Blossom smoke in short mode")
	}

	for _, tc := range []struct {
		name string
		k    int
		seed int64
	}{
		{name: "k192", k: 192, seed: 19200},
		{name: "k256", k: 256, seed: 25600},
	} {
		t.Run(tc.name, func(t *testing.T) {
			problem := internalSeededMatchingProblem(tc.k, tc.seed)

			match, _, stats, err := solveMinimumWeightPerfectMatching(problem, blossomOptions{Eps: DefaultEps})
			if err != nil {
				t.Fatalf("blossom: %v stats=%+v", err, stats)
			}
			if err = verifyPerfectMatching(match); err != nil {
				t.Fatalf("verify: %v stats=%+v", err, stats)
			}
			if stats.Augmentations != tc.k/2 {
				t.Fatalf("augmentations got %d want %d stats=%+v", stats.Augmentations, tc.k/2, stats)
			}
		})
	}
}
