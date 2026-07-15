package tsp

import (
	"fmt"
	"math"
	"testing"
)

func TestBlossomSolvesGreedyTrapK6(t *testing.T) {
	problem := matchingProblem{
		odd: []int{0, 1, 2, 3, 4, 5},
		n:   6,
		w: []float64{
			0, 1, 2, 2, 9, 9,
			1, 0, 2, 2, 9, 9,
			2, 2, 0, 9, 1, 9,
			2, 2, 9, 0, 9, 1,
			9, 9, 1, 9, 0, 2,
			9, 9, 9, 1, 2, 0,
		},
	}

	_, wantCost, err := exactMatchingOracleForTest(problem)
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}

	match, gotCost, stats, err := solveMinimumWeightPerfectMatching(problem, blossomOptions{Eps: DefaultEps})
	if err != nil {
		t.Fatalf("blossom: %v", err)
	}
	if err = verifyPerfectMatching(match); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if math.Abs(gotCost-wantCost) > DefaultEps {
		t.Fatalf("got %.12f want %.12f match=%v stats=%+v", gotCost, wantCost, match, stats)
	}
}

func TestBlossomK4Seed02Trace(t *testing.T) {
	problem := internalSeededMatchingProblem(4, 2)

	engine, err := newBlossomEngine(problem, blossomOptions{Eps: DefaultEps})
	if err != nil {
		t.Fatalf("newBlossomEngine: %v", err)
	}

	for engine.matchedPairs() < problem.n/2 {
		before := engine.matchedPairs()

		engine.resetForest()
		for vertex := 0; vertex < engine.problem.n; vertex++ {
			if engine.mate[vertex] == noVertex {
				engine.assignOuterRoot(engine.inBlossom[vertex])
			}
		}

		for step := 0; step < 64; step++ {
			event, ok, scanErr := engine.scanTightEdges()
			if scanErr != nil {
				t.Fatalf("scan step=%d before=%d mate=%v label=%v parent=%v root=%v err=%v",
					step, before, engine.mate, engine.label, engine.parent, engine.treeRoot, scanErr)
			}

			if ok {
				t.Logf("step=%d event=%+v mate=%v label=%v parent=%v root=%v",
					step, event, engine.mate, engine.label, engine.parent, engine.treeRoot)

				done, applyErr := engine.applyBlossomEvent(event)
				if applyErr != nil {
					t.Fatalf("apply step=%d event=%+v mate=%v mateEdge=%v label=%v labelEdge=%v parent=%v root=%v err=%v",
						step, event, engine.mate, engine.mateEdge, engine.label, engine.labelEdge, engine.parent, engine.treeRoot, applyErr)
				}
				if done {
					after := engine.matchedPairs()
					if after <= before {
						t.Fatalf("no progress: before=%d after=%d mate=%v", before, after, engine.mate)
					}
					break
				}

				continue
			}

			delta, deltaErr := engine.nextDelta()
			if deltaErr != nil {
				t.Fatalf("delta step=%d mate=%v label=%v parent=%v root=%v err=%v",
					step, engine.mate, engine.label, engine.parent, engine.treeRoot, deltaErr)
			}

			t.Logf("step=%d delta=%+v dual=%v", step, delta, engine.dual)

			if err = engine.applyDelta(delta); err != nil {
				t.Fatalf("applyDelta step=%d delta=%+v dual=%v err=%v", step, delta, engine.dual, err)
			}

			engine.rewindForestScan()
		}
	}

	match, err := engine.exportMatching()
	if err != nil {
		t.Fatalf("export: %v mate=%v", err, engine.mate)
	}
	t.Logf("match=%v", match)
}

func TestBlossomK4AllSeededMatchesOracle(t *testing.T) {
	k := 4
	for seed := int64(1); seed <= 25; seed++ {
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

func TestBlossomK6Seed23Diagnostic(t *testing.T) {
	problem := internalSeededMatchingProblem(6, 23)

	match, gotCost, stats, err := solveMinimumWeightPerfectMatching(problem, blossomOptions{Eps: DefaultEps})
	if err != nil {
		t.Fatalf("blossom diagnostic failed: %+v", err)
	}

	if err = verifyPerfectMatching(match); err != nil {
		t.Fatalf("verify: %v", err)
	}

	oracleMatch, wantCost, err := exactMatchingOracleForTest(problem)
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}

	if math.Abs(gotCost-wantCost) > DefaultEps {
		t.Fatalf("cost got %.12f want %.12f match=%v oracle=%v stats=%+v",
			gotCost, wantCost, match, oracleMatch, stats)
	}
}

func TestBlossomK8Seed05RejectsFirstInvalidLiftAndMatchesOracle(t *testing.T) {
	problem := internalSeededMatchingProblem(8, 5)

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
		t.Fatalf("cost got %.12f want %.12f match=%v", gotCost, wantCost, match)
	}
}

func TestBlossomNearTieK12Seed80DoesNotStall(t *testing.T) {
	problem := internalNearTieMatchingProblem(12, 80)

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

	tol := 1e-7 * math.Max(1, math.Abs(wantCost))
	if math.Abs(gotCost-wantCost) > tol {
		t.Fatalf("cost got %.12f want %.12f tol %.12g match=%v stats=%+v",
			gotCost, wantCost, tol, match, stats)
	}
}

func TestBlossomNearTieK12Seed86MatchesOracle(t *testing.T) {
	problem := internalNearTieMatchingProblem(12, 86)

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

	tol := 1e-7 * math.Max(1, math.Abs(wantCost))
	if math.Abs(gotCost-wantCost) > tol {
		t.Fatalf("cost got %.12f want %.12f tol %.12g match=%v stats=%+v",
			gotCost, wantCost, tol, match, stats)
	}
}
