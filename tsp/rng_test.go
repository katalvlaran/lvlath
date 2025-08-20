// Package tsp_test validates deterministic RNG behavior used by local-search
// neighborhoods (e.g., 2-opt) when shuffling is enabled.
package tsp_test

import (
	"math"
	"slices"
	"testing"

	"github.com/katalvlaran/lvlath/tsp"
)

// TestRNG_TSP_TwoOpt_Shuffle_SeedDeterminism checks that repeated runs with the
// same seed produce *identical* tours and costs on a symmetric metric instance.
func TestRNG_TSP_TwoOpt_Shuffle_SeedDeterminism(t *testing.T) {
	// Build a small but non-trivial symmetric instance: a gently rippled circle.
	// This shape creates multiple potential improving 2-opt moves so that the
	// neighborhood order matters (hence shuffling must be deterministic under a seed).
	const n = 10                    // number of vertices
	var pts = make([][2]float64, n) // coordinates buffer
	var i int                       // loop iterator
	var th float64                  // angle accumulator
	var r float64                   // radius (with ripple)
	for i = 0; i < n; i++ {         // fill points on a perturbed circle
		th = 2 * 3.141592653589793 * float64(i) / float64(n)    // angle on unit circle
		r = 1.0 + 0.025*float64(i%3)                            // tiny ripple to avoid symmetry ties
		pts[i] = [2]float64{r * math.Cos(th), r * math.Sin(th)} // Cartesian coordinates
	}
	var m = euclid(pts) // build symmetric metric matrix with zero diagonal

	// Configure the dispatcher to run 2-opt *only*, with shuffling enabled and
	// a fixed seed. This forces the path that exercises RNG in neighborhood order.
	var opt = tsp.DefaultOptions() // start from sane defaults
	opt.Algo = tsp.TwoOptOnly      // restrict to 2-opt to isolate RNG impact
	opt.Symmetric = true           // symmetric TSP mode
	opt.StartVertex = startV       // canonical start index (from test utils)
	opt.Eps = epsTiny              // strict acceptance epsilon
	opt.EnableLocalSearch = true   // enable local search
	opt.ShuffleNeighborhood = true // enable RNG-driven neighborhood order
	opt.Seed = seedDet             // fixed seed (0 => internal defaultRNGSeed)

	// Run three times and verify tours/costs are *identical* after normalization.
	var baseOpen []int                // baseline open tour (normalized)
	var baseCost float64              // baseline stabilized cost
	Repeat(t, 3, func(t *testing.T) { // repeat to lock determinism
		// Solve with the configured options.
		var res, err = tsp.SolveWithMatrix(m, nil, opt) // run dispatcher end-to-end
		if err != nil {                                 // solver should not fail here
			t.Fatalf("SolveWithMatrix failed: %v", err)
		}
		// Validate the returned tour shape/invariants to guard against regressions.
		if verr := tsp.ValidateTour(res.Tour, n, startV); verr != nil {
			t.Fatalf("returned tour invalid: %v", verr)
		}
		// Normalize the closed tour to an *open* cycle starting at 0 for comparison.
		var open = normalizeClosedToOpen(t, res.Tour) // use shared helper (rotation+strip)
		// Capture the first outcome and compare all subsequent runs against it.
		if baseOpen == nil { // first repetition: capture baseline
			baseOpen = append([]int(nil), open...) // deep copy for stability
			baseCost = res.Cost                    // capture stabilized cost (rounded in impl)
			return                                 // proceed to next repetition
		}
		// Compare structure: tours must be exactly identical (index-by-index).
		if !slices.Equal(open, baseOpen) {
			t.Fatalf("non-deterministic tour:\nfirst: %v\n this: %v", baseOpen, open)
		}
		// Compare cost: stabilized cost must also be identical.
		if round1e9(res.Cost) != round1e9(baseCost) {
			t.Fatalf("non-deterministic cost: first=%.12f this=%.12f", baseCost, res.Cost)
		}
	})
}

// TestRNG_ATSP_TwoOpt_Shuffle_SeedDeterminism mirrors the symmetric test above
// but on an asymmetric matrix (ATSP). We verify the same-seed determinism holds.
func TestRNG_ATSP_TwoOpt_Shuffle_SeedDeterminism(t *testing.T) {
	// Build a directed (asymmetric) instance by adding a small directional bias
	// to Euclidean distances on a circle. This exercises the ATSP path.
	const n = 9                     // number of vertices
	var pts = make([][2]float64, n) // coordinates buffer
	var i int                       // loop iterator
	var th float64                  // angle accumulator
	for i = 0; i < n; i++ {         // uniform angles on unit circle
		th = 2 * 3.141592653589793 * float64(i) / float64(n)
		pts[i] = [2]float64{math.Cos(th), math.Sin(th)} // unit circle coordinates
	}
	var m = euclidAsym(pts, 0.15) // asymmetric metric via directional penalty

	// Configure 2-opt with shuffling and a fixed seed in ATSP mode.
	var opt = tsp.DefaultOptions() // start from defaults
	opt.Algo = tsp.TwoOptOnly      // isolate the RNG to 2-opt neighborhood order
	opt.Symmetric = false          // ATSP mode (directed)
	opt.StartVertex = startV       // canonical start index
	opt.Eps = epsTiny              // strict acceptance epsilon
	opt.EnableLocalSearch = true   // enable local search
	opt.ShuffleNeighborhood = true // enable RNG-driven neighborhood order
	opt.Seed = seedDet             // fixed seed (0 => internal defaultRNGSeed)

	// Repeat three times and assert exact stability of tour and cost.
	var baseOpen []int                // baseline open tour (normalized)
	var baseCost float64              // baseline stabilized cost
	Repeat(t, 3, func(t *testing.T) { // repeat to lock determinism
		// Solve with the ATSP configuration.
		var res, err = tsp.SolveWithMatrix(m, nil, opt) // run dispatcher end-to-end
		if err != nil {                                 // solver should not fail on this instance
			t.Fatalf("SolveWithMatrix failed: %v", err)
		}
		// Validate the returned tour for ATSP invariants (directed cycle).
		if verr := tsp.ValidateTour(res.Tour, n, startV); verr != nil {
			t.Fatalf("returned tour invalid: %v", verr)
		}
		// Normalize the closed tour to an open cycle starting at 0 for comparison.
		var open = normalizeClosedToOpen(t, res.Tour) // rotation + strip
		// Capture the first outcome and compare all subsequent runs against it.
		if baseOpen == nil { // first repetition: capture baseline
			baseOpen = append([]int(nil), open...) // deep copy to avoid aliasing
			baseCost = res.Cost                    // capture stabilized cost
			return                                 // proceed to next repetition
		}
		// Structure must be identical across same-seed runs.
		if !slices.Equal(open, baseOpen) {
			t.Fatalf("non-deterministic tour (ATSP):\nfirst: %v\n this: %v", baseOpen, open)
		}
		// Cost must also be identical across same-seed runs.
		if round1e9(res.Cost) != round1e9(baseCost) {
			t.Fatalf("non-deterministic cost (ATSP): first=%.12f this=%.12f", baseCost, res.Cost)
		}
	})
}
