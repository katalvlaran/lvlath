// Package tsp_test provides focused unit tests for the Christofides approximation.
// Scope:
//  1. Valid tour and 1.5×MST sanity on a regular hexagon (symmetric metric).
//  2. Determinism: repeated runs produce identical tour/cost.
//  3. Dispatcher-level strict sentinel on asymmetry for Christofides.
//  4. Optional 2-opt polishing via SolveWithMatrix is never worse than raw Christofides.
package tsp_test

import (
	"errors"
	"math"
	"slices"
	"testing"

	"github.com/katalvlaran/lvlath/matrix"
	"github.com/katalvlaran/lvlath/tsp"
)

// mstWeight is a tiny helper that returns the MST total weight for a matrix.
// It uses the same exported routine as the implementation (Prim O(n^2)).
func mstWeight(t *testing.T, m matrix.Matrix) float64 {
	t.Helper()
	// The concrete param type must satisfy matrix.Matrix; we accept via interface.
	// Cast to tsp.MinimumSpanningTree signature implicitly via type inference.
	w, _, err := tsp.MinimumSpanningTree(m)
	if err != nil {
		t.Fatalf("MinimumSpanningTree failed: %v", err)
	}

	return w
}

//  1. Christofides on a regular hexagon - valid tour and cost ≤ 1.5×MST.
//     This is a robust sanity since for a convex regular polygon: perimeter ~ 6·s,
//     MST ~ 5·s, hence perimeter ≤ 1.5·MST holds with margin.
func TestTSPApprox_Hexagon_Valid_Le15xMST(t *testing.T) {
	// Regular hexagon on the unit circle.
	const n = 6
	pts := [][2]float64{
		{1, 0},
		{0.5, math.Sqrt(3) / 2},
		{-0.5, math.Sqrt(3) / 2},
		{-1, 0},
		{-0.5, -math.Sqrt(3) / 2},
		{0.5, -math.Sqrt(3) / 2},
	}
	m := euclid(pts) // symmetric metric with zero diagonal

	// Run pure Christofides (deterministic, no RNG).
	opt := tsp.DefaultOptions()
	opt.Symmetric = true
	opt.StartVertex = startV
	opt.EnableLocalSearch = false // keep Christofides pure; polishing is tested separately

	res, err := tsp.TSPApprox(m, opt)
	if err != nil {
		t.Fatalf("TSPApprox failed: %v", err)
	}
	// Validate the Hamiltonian cycle invariant.
	if err = tsp.ValidateTour(res.Tour, n, startV); err != nil {
		t.Fatalf("returned tour invalid: %v", err)
	}

	// Compare against 1.5×MST (robust sanity on this instance family).
	mst := mstWeight(t, m)
	limit := 1.5 * mst
	if round1e9(res.Cost) > round1e9(limit) {
		t.Fatalf("Christofides exceeded 1.5×MST: cost=%.12f mst=%.12f limit=%.12f",
			res.Cost, mst, limit)
	}
}

//  2. Determinism: Christofides has no RNG; repeated results must match exactly
//     (up to our canonical orientation which the implementation enforces).
func TestTSPApprox_Determinism_Repeat3(t *testing.T) {
	const n = 8
	// Slightly perturbed circle to avoid accidental symmetries.
	pts := make([][2]float64, n)
	var i int
	var th float64
	var r float64
	for i = 0; i < n; i++ {
		th = 2 * math.Pi * float64(i) / float64(n)
		r = 1.0 + 0.02*math.Sin(3*th)
		pts[i] = [2]float64{r * math.Cos(th), r * math.Sin(th)}
	}
	m := euclid(pts)

	opt := tsp.DefaultOptions()
	opt.Symmetric = true
	opt.StartVertex = startV
	opt.EnableLocalSearch = false // pure Christofides

	var baseOpen []int
	var baseCost float64

	Repeat(t, 3, func(t *testing.T) {
		res, err := tsp.TSPApprox(m, opt)
		if err != nil {
			t.Fatalf("TSPApprox failed: %v", err)
		}
		open := normalizeClosedToOpen(t, res.Tour)
		if baseOpen == nil {
			baseOpen = append([]int(nil), open...)
			baseCost = res.Cost
			return
		}
		if !slices.Equal(open, baseOpen) || round1e9(res.Cost) != round1e9(baseCost) {
			t.Fatalf("nondeterministic Christofides result.\nfirst: %v (%.12f)\n this: %v (%.12f)",
				baseOpen, baseCost, open, res.Cost)
		}
	})
}

//  3. Dispatcher-level strict sentinel on asymmetry:
//     Christofides is only for symmetric metrics. The dispatcher must reject
//     asymmetric matrices with ErrAsymmetry (strict sentinel).
func TestTSPApprox_Dispatcher_RejectsAsymmetry(t *testing.T) {
	// Make an asymmetric metric from a circle with directional bias.
	const n = 7
	pts := make([][2]float64, n)
	var i int
	var th float64
	for i = 0; i < n; i++ {
		th = 2 * math.Pi * float64(i) / float64(n)
		pts[i] = [2]float64{math.Cos(th), math.Sin(th)}
	}
	m := euclidAsym(pts, 0.2) // asymmetric metric

	// Ask the dispatcher to run the Christofides path (Symmetric==true).
	opt := tsp.DefaultOptions()
	opt.Symmetric = true // force TSP (Christofides path)
	opt.StartVertex = startV
	_, err := tsp.SolveWithMatrix(m, nil, opt)
	if !errors.Is(err, tsp.ErrAsymmetry) {
		t.Fatalf("want ErrAsymmetry from dispatcher, got %v", err)
	}
}

//  4. Optional 2-opt polishing is never worse than raw Christofides.
//     We compare pure TSPApprox cost vs SolveWithMatrix with EnableLocalSearch=true.
func TestTSPApprox_PolishTwoOpt_NotWorse(t *testing.T) {
	// Nontrivial symmetric instance (small ripple to create 2-opt opportunities).
	const n = 10
	pts := make([][2]float64, n)
	var i int
	var th float64
	var r float64
	for i = 0; i < n; i++ {
		th = 2 * math.Pi * float64(i) / float64(n)
		r = 1.0 + 0.03*math.Cos(4*th)
		pts[i] = [2]float64{r * math.Cos(th), r * math.Sin(th)}
	}
	m := euclid(pts)

	// Baseline: pure Christofides (no local search).
	base := tsp.DefaultOptions()
	base.Symmetric = true
	base.StartVertex = startV
	base.EnableLocalSearch = false
	resBase, err := tsp.TSPApprox(m, base)
	if err != nil {
		t.Fatalf("TSPApprox baseline failed: %v", err)
	}

	// Auto pipeline with local search: dispatcher should run Christofides then 2-opt.
	auto := tsp.DefaultOptions()
	auto.Symmetric = true
	auto.StartVertex = startV
	auto.EnableLocalSearch = true // enable polishing
	auto.BestImprovement = false  // fast 2-opt policy for this test
	auto.Eps = epsTiny

	resAuto, err := tsp.SolveWithMatrix(m, nil, auto)
	if err != nil {
		t.Fatalf("SolveWithMatrix (Christofides + 2-opt) failed: %v", err)
	}

	// Non-worsening guarantee after stable rounding.
	if round1e9(resAuto.Cost) > round1e9(resBase.Cost) {
		t.Fatalf("2-opt polishing made it worse: base=%.12f auto=%.12f", resBase.Cost, resAuto.Cost)
	}
}
