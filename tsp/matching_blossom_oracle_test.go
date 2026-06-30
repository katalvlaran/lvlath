package tsp

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"testing"
)

func TestBlossomMatchesOracleForSeededEvenK2To16(t *testing.T) {
	for k := 2; k <= 16; k += 2 {
		for seed := int64(1); seed <= 100; seed++ {
			k := k
			seed := seed

			t.Run(fmt.Sprintf("k=%02d/seed=%03d", k, seed), func(t *testing.T) {
				problem := internalSeededMatchingProblem(k, seed)

				_, wantCost, err := exactMatchingOracleForTest(problem)
				if err != nil {
					t.Fatalf("oracle: %v", err)
				}

				match, gotCost, stats, err := solveMinimumWeightPerfectMatching(problem, blossomOptions{Eps: DefaultEps})
				if err != nil {
					t.Fatalf("blossom: %v stats=%+v", err, stats)
				}
				if err = verifyPerfectMatching(match); err != nil {
					t.Fatalf("verify blossom: %v match=%v stats=%+v", err, match, stats)
				}
				if math.Abs(gotCost-wantCost) > DefaultEps {
					t.Fatalf("cost got %.12f want %.12f match=%v stats=%+v", gotCost, wantCost, match, stats)
				}
			})
		}
	}
}

func TestBlossomRegressionSeeds(t *testing.T) {
	cases := []struct {
		name string
		k    int
		seed int64
	}{
		{name: "k06_seed23_forest_redirect", k: 6, seed: 23},
		{name: "k08_seed05_transactional_lift", k: 8, seed: 5},
		{name: "k08_seed02_nested_lift", k: 8, seed: 2},
		{name: "k08_seed06_nested_lift", k: 8, seed: 6},
		{name: "k08_seed11_nested_lift", k: 8, seed: 11},
		{name: "k10_seed14_refresh_candidate", k: 10, seed: 14},
		{name: "k10_seed18_refresh_candidate", k: 10, seed: 18},
		{name: "k12_seed06_shrink_nested", k: 12, seed: 6},
		{name: "k12_seed08_shrink_nested", k: 12, seed: 8},
		{name: "k12_seed09_augment_nested", k: 12, seed: 9},
		{name: "k12_seed19_augment_nested", k: 12, seed: 19},
		{name: "k12_seed20_augment_nested", k: 12, seed: 20},
		{name: "k14_seed01_augment_nested", k: 14, seed: 1},
		{name: "k14_seed03_augment_nested", k: 14, seed: 3},
		{name: "k14_seed09_augment_nested", k: 14, seed: 9},
		{name: "k14_seed14_augment_nested", k: 14, seed: 14},
		{name: "k14_seed23_augment_nested", k: 14, seed: 23},
		{name: "k14_seed25_augment_nested", k: 14, seed: 25},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			problem := internalSeededMatchingProblem(tc.k, tc.seed)

			_, wantCost, err := exactMatchingOracleForTest(problem)
			if err != nil {
				t.Fatalf("oracle: %v", err)
			}

			match, gotCost, stats, err := solveMinimumWeightPerfectMatching(problem, blossomOptions{Eps: DefaultEps})
			if err != nil {
				t.Fatalf("blossom: %v stats=%+v", err, stats)
			}
			if err = verifyPerfectMatching(match); err != nil {
				t.Fatalf("verify: %v match=%v stats=%+v", err, match, stats)
			}
			if math.Abs(gotCost-wantCost) > DefaultEps {
				t.Fatalf("cost got %.12f want %.12f match=%v stats=%+v", gotCost, wantCost, match, stats)
			}
		})
	}
}

func TestBlossomWideRangeWeightsMatchesOracleSmall(t *testing.T) {
	for _, seed := range []int64{7, 42, 1001, 9001, 17017, 424242} {
		seed := seed

		t.Run(fmt.Sprintf("seed=%d", seed), func(t *testing.T) {
			problem := internalWideRangeMatchingProblem(12, seed)

			_, wantCost, err := exactMatchingOracleForTest(problem)
			if err != nil {
				t.Fatalf("oracle: %v", err)
			}

			match, gotCost, stats, err := solveMinimumWeightPerfectMatching(problem, blossomOptions{Eps: DefaultEps})
			if err != nil {
				t.Fatalf("blossom: %v stats=%+v", err, stats)
			}
			if err = verifyPerfectMatching(match); err != nil {
				t.Fatalf("verify: %v match=%v stats=%+v", err, match, stats)
			}
			if math.Abs(gotCost-wantCost) > 10*DefaultEps {
				t.Fatalf("cost got %.12f want %.12f match=%v stats=%+v", gotCost, wantCost, match, stats)
			}
		})
	}
}

func TestBlossomEqualWeightsDeterministic(t *testing.T) {
	for _, k := range []int{4, 8, 16, 32, 64} {
		k := k

		t.Run(fmt.Sprintf("k=%d", k), func(t *testing.T) {
			problem := internalConstantMatchingProblem(k, 1)

			first, firstCost, firstStats, err := solveMinimumWeightPerfectMatching(problem, blossomOptions{Eps: DefaultEps})
			if err != nil {
				t.Fatalf("first: %v stats=%+v", err, firstStats)
			}
			if err = verifyPerfectMatching(first); err != nil {
				t.Fatalf("first verify: %v", err)
			}

			for run := 0; run < 5; run++ {
				got, gotCost, stats, runErr := solveMinimumWeightPerfectMatching(problem, blossomOptions{Eps: DefaultEps})
				if runErr != nil {
					t.Fatalf("run %d: %v stats=%+v", run, runErr, stats)
				}
				if gotCost != firstCost {
					t.Fatalf("run %d cost got %.12f want %.12f", run, gotCost, firstCost)
				}
				for vertex := range got {
					if got[vertex] != first[vertex] {
						t.Fatalf("run %d match[%d]=%d want %d", run, vertex, got[vertex], first[vertex])
					}
				}
			}
		})
	}
}

func TestBlossomRejectsInvalidNumericProblems(t *testing.T) {
	cases := []struct {
		name string
		edit func(*matchingProblem)
		want error
	}{
		{
			name: "odd order",
			edit: func(problem *matchingProblem) {
				*problem = internalSeededMatchingProblem(5, 1)
			},
			want: ErrInvalidMatching,
		},
		{
			name: "nan",
			edit: func(problem *matchingProblem) {
				problem.w[1] = math.NaN()
			},
			want: ErrNaNInf,
		},
		{
			name: "inf",
			edit: func(problem *matchingProblem) {
				problem.w[1] = math.Inf(1)
			},
			want: ErrNaNInf,
		},
		{
			name: "negative",
			edit: func(problem *matchingProblem) {
				problem.w[1] = -1
			},
			want: ErrNegativeWeight,
		},
		{
			name: "asymmetric",
			edit: func(problem *matchingProblem) {
				problem.w[1] += 3
			},
			want: ErrAsymmetry,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			problem := internalSeededMatchingProblem(6, 1)
			tc.edit(&problem)

			_, _, _, err := solveMinimumWeightPerfectMatching(problem, blossomOptions{Eps: DefaultEps})
			if !errors.Is(err, tc.want) {
				t.Fatalf("err got %v want %v", err, tc.want)
			}
		})
	}
}

func TestBlossomRejectsInvalidOptionsAndOddProblem(t *testing.T) {
	problem := internalSeededMatchingProblem(4, 1)

	if _, _, _, err := solveMinimumWeightPerfectMatching(problem, blossomOptions{Eps: 0}); err != ErrInvalidOptions {
		t.Fatalf("invalid eps got %v", err)
	}

	oddProblem := matchingProblem{
		odd: []int{0, 1, 2},
		n:   3,
		w:   make([]float64, 9),
	}

	if _, _, _, err := solveMinimumWeightPerfectMatching(oddProblem, blossomOptions{Eps: DefaultEps}); err != ErrInvalidMatching {
		t.Fatalf("odd problem got %v", err)
	}
}

func TestBlossomMatchesOracleForSeededEvenK2To14(t *testing.T) {
	for k := 2; k <= 14; k += 2 {
		for seed := int64(1); seed <= 25; seed++ {
			k := k
			seed := seed

			t.Run(fmt.Sprintf("k=%02d/seed=%02d", k, seed), func(t *testing.T) {
				problem := internalSeededMatchingProblem(k, seed)

				_, wantCost, err := exactMatchingOracleForTest(problem)
				if err != nil {
					t.Fatalf("oracle: %v", err)
				}

				match, gotCost, _, err := solveMinimumWeightPerfectMatching(problem, blossomOptions{Eps: DefaultEps})
				if err != nil {
					t.Fatalf("blossom: %v", err)
				}
				if err = verifyPerfectMatching(match); err != nil {
					t.Fatalf("verify blossom: %v", err)
				}
				if math.Abs(gotCost-wantCost) > DefaultEps {
					t.Fatalf("cost got %.12f want %.12f", gotCost, wantCost)
				}
			})
		}
	}
}

func internalConstantMatchingProblem(k int, weight float64) matchingProblem {
	odd := make([]int, k)
	w := make([]float64, k*k)

	for vertex := 0; vertex < k; vertex++ {
		odd[vertex] = vertex
	}

	for row := 0; row < k; row++ {
		for col := 0; col < k; col++ {
			if row == col {
				continue
			}

			w[row*k+col] = weight
		}
	}

	return matchingProblem{
		odd: odd,
		w:   w,
		n:   k,
	}
}
func internalWideRangeMatchingProblem(k int, seed int64) matchingProblem {
	rng := rand.New(rand.NewSource(seed))

	odd := make([]int, k)
	w := make([]float64, k*k)

	for vertex := 0; vertex < k; vertex++ {
		odd[vertex] = vertex
	}

	for row := 0; row < k; row++ {
		for col := row + 1; col < k; col++ {
			exp := rng.Intn(6)
			scale := math.Pow10(exp)
			value := scale * (1 + rng.Float64()*1000)

			w[row*k+col] = value
			w[col*k+row] = value
		}
	}

	return matchingProblem{
		odd: odd,
		w:   w,
		n:   k,
	}
}
