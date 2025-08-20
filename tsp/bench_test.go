// Package tsp_test — comprehensive benchmarks for lvlath/tsp algorithms.
// Scope:
//   - Branch-and-Bound with SimpleBound and OneTreeBound (small n; exact).
//   - Auto pipeline on symmetric instances (typically Christofides path).
//   - 2-opt on TSP and ATSP (medium n; heuristic local search).
//   - Micro-benchmarks for MST (Prim O(n^2)), Eulerian (Hierholzer), TourCost.
//
// Policy:
//   - Deterministic geometry (rippled circles) and fixed seeds (seedDet).
//   - Pre-build all inputs outside timer; measure only algorithmic core.
//   - No flaky time limits; instances sized to be fast on CI.
//
// Notes:
//   - Reuses helpers from testutil_test.go: euclid, euclidAsym, startV, epsTiny,
//     seedDet, edgesCount, Repeat, mustFloatClose, etc.
package tsp_test

import (
	"math"
	"testing"

	"github.com/katalvlaran/lvlath/tsp"
)

// deepCopyAdj performs a deep copy of an adjacency list (each row cloned).
// Time: O(E). Memory: O(E). Needed when the callee mutates adjacency in-place.
func deepCopyAdj(adj [][]int) [][]int {
	// Allocate the outer slice with the same number of rows.
	var n = len(adj)          // number of vertices
	var cp = make([][]int, n) // output adjacency
	var u int                 // row index
	for u = 0; u < n; u++ {   // iterate rows
		// Copy row u entries into a fresh slice to avoid aliasing.
		cp[u] = append([]int(nil), adj[u]...) // deep copy of row
	}

	// Return the deep-copied adjacency.
	return cp
}

// ------------------------------------------------------------------------------------
// Branch-and-Bound (exact) — n=9, two bounds: SimpleBound and OneTreeBound.
// These sizes finish comfortably on CI while still exercising the machinery.
// ------------------------------------------------------------------------------------

// BenchmarkBB_SimpleBound_n9 measures the exact Branch-and-Bound with SimpleBound.
func BenchmarkBB_SimpleBound_n9(b *testing.B) {
	// Problem size tuned for fast exact solve on CI.
	const n = 9 // number of vertices for exact solve
	// Build a slightly rippled circle to avoid ties; symmetric metric.
	var pts = make([][2]float64, n) // buffer for coordinates
	var i int                       // loop iterator
	var th float64                  // angle
	var r float64                   // radius with ripple
	for i = 0; i < n; i++ {         // fill coordinates
		th = 2.0 * 3.141592653589793 * float64(i) / float64(n)  // uniform angle
		r = 1.0 + 0.02*float64((i*5)%7)                         // deterministic ripple
		pts[i] = [2]float64{r * math.Cos(th), r * math.Sin(th)} // position on circle
	}
	// Build symmetric distance matrix once (outside timer).
	var m = euclid(pts) // symmetric metric with zero diagonal

	// Configure BnB options with SimpleBound; no local search (exact path).
	var opt = tsp.DefaultOptions()  // start from defaults
	opt.Algo = tsp.BranchAndBound   // exact solver
	opt.Symmetric = true            // symmetric TSP
	opt.StartVertex = startV        // canonical start
	opt.Eps = epsTiny               // strict acceptance
	opt.BoundAlgo = tsp.SimpleBound // admissible LB (simple)
	opt.EnableLocalSearch = false   // avoid heuristic work in exact mode
	opt.Seed = seedDet              // seed unused here but kept for policy consistency

	// Report allocations and reset timer right before the loop.
	b.ReportAllocs() // enable allocation stats
	b.ResetTimer()   // reset benchmark timer

	// Run the benchmark loop.
	var it int                   // iteration counter
	for it = 0; it < b.N; it++ { // repeat per the harness
		// Solve the instance end-to-end via dispatcher.
		var _, err = tsp.SolveWithMatrix(m, nil, opt) // run exact BnB
		if err != nil {                               // exact solve should not fail on this instance
			b.Fatalf("BranchAndBound(SimpleBound) failed: %v", err)
		}
	}
}

// BenchmarkBB_OneTreeRoot_n9 measures BnB with the stronger OneTree bound.
func BenchmarkBB_OneTreeRoot_n9(b *testing.B) {
	// Use the same geometry as in the SimpleBound benchmark.
	const n = 9                     // identical size for apples-to-apples comparison
	var pts = make([][2]float64, n) // coordinate buffer
	var i int                       // loop iterator
	var th float64                  // angle
	var r float64                   // ripple radius
	for i = 0; i < n; i++ {         // fill coordinates
		th = 2.0 * 3.141592653589793 * float64(i) / float64(n)
		r = 1.0 + 0.02*float64((i*5)%7)
		pts[i] = [2]float64{r * math.Cos(th), r * math.Sin(th)}
	}
	// Build symmetric matrix once.
	var m = euclid(pts)

	// Configure OneTree bound; other options mirror the SimpleBound case.
	var opt = tsp.DefaultOptions()   // sane defaults
	opt.Algo = tsp.BranchAndBound    // exact solve
	opt.Symmetric = true             // symmetric TSP
	opt.StartVertex = startV         // canonical start
	opt.Eps = epsTiny                // strict epsilon
	opt.BoundAlgo = tsp.OneTreeBound // stronger admissible LB
	opt.EnableLocalSearch = false    // exact path only
	opt.Seed = seedDet               // fixed seed (not used here)

	// Benchmark loop.
	b.ReportAllocs()             // request allocation stats
	b.ResetTimer()               // reset timer
	var it int                   // iteration counter
	for it = 0; it < b.N; it++ { // harness-driven repetitions
		var _, err = tsp.SolveWithMatrix(m, nil, opt) // run BnB with OneTree
		if err != nil {
			b.Fatalf("BranchAndBound(OneTree) failed: %v", err)
		}
	}
}

// ------------------------------------------------------------------------------------
// 2-opt (heuristic) — medium sizes; deterministic shuffle order via Seed.
// We benchmark both TSP (symmetric) and ATSP (directed).
// ------------------------------------------------------------------------------------

// BenchmarkTwoOpt_Symmetric_n200 measures 2-opt on a symmetric 200-vertex instance.
func BenchmarkTwoOpt_Symmetric_n200(b *testing.B) {
	// Instance size balanced to exercise O(n^2) scanning but stay fast on CI.
	const n = 200 // number of vertices
	// Generate a gently rippled circle (deterministic).
	var pts = make([][2]float64, n) // coordinate buffer
	var i int                       // loop iterator
	var th float64                  // angle
	var r float64                   // radius with small ripple
	for i = 0; i < n; i++ {         // fill coordinates
		th = 2.0 * 3.141592653589793 * float64(i) / float64(n)  // uniform angle
		r = 1.0 + 0.015*float64((i*7)%11)                       // weak ripple to avoid ties
		pts[i] = [2]float64{r * math.Cos(th), r * math.Sin(th)} // position
	}
	// Build symmetric metric once (outside timer).
	var m = euclid(pts) // symmetric matrix

	// Configure 2-opt with deterministic neighborhood shuffle.
	var opt = tsp.DefaultOptions() // defaults
	opt.Algo = tsp.TwoOptOnly      // restrict to 2-opt
	opt.Symmetric = true           // TSP mode
	opt.StartVertex = startV       // canonical start
	opt.Eps = epsTiny              // strict improvement threshold
	opt.EnableLocalSearch = true   // enable local search
	opt.ShuffleNeighborhood = true // shuffle candidate order
	opt.Seed = seedDet             // fixed seed for determinism

	// Benchmark loop.
	b.ReportAllocs()             // allocation stats
	b.ResetTimer()               // reset timer
	var it int                   // loop counter
	for it = 0; it < b.N; it++ { // repeat
		var _, err = tsp.SolveWithMatrix(m, nil, opt) // run 2-opt
		if err != nil {                               // heuristic solve should not fail
			b.Fatalf("TwoOptOnly (symmetric) failed: %v", err)
		}
	}
}

// BenchmarkTwoOpt_ATSP_n150 measures 2-opt on an asymmetric 150-vertex instance.
func BenchmarkTwoOpt_ATSP_n150(b *testing.B) {
	// Instance size slightly smaller than TSP to keep runtime tight on CI.
	const n = 150 // number of vertices
	// Build unit-circle coordinates (no ripple needed once we bias asymmetry).
	var pts = make([][2]float64, n) // coordinates
	var i int                       // iterator
	var th float64                  // angle
	for i = 0; i < n; i++ {         // fill points
		th = 2.0 * 3.141592653589793 * float64(i) / float64(n) // angle
		pts[i] = [2]float64{math.Cos(th), math.Sin(th)}        // unit circle
	}
	// Build asymmetric metric via directional penalty (bias>0).
	var m = euclidAsym(pts, 0.12) // ATSP matrix

	// Configure 2-opt for ATSP with deterministic shuffle.
	var opt = tsp.DefaultOptions() // defaults
	opt.Algo = tsp.TwoOptOnly      // restrict to 2-opt
	opt.Symmetric = false          // ATSP mode
	opt.StartVertex = startV       // canonical start
	opt.Eps = epsTiny              // strict improvement threshold
	opt.EnableLocalSearch = true   // enable local search
	opt.ShuffleNeighborhood = true // shuffle candidate order
	opt.Seed = seedDet             // fixed seed for determinism

	// Benchmark loop.
	b.ReportAllocs()             // allocation stats
	b.ResetTimer()               // reset timer
	var it int                   // iteration counter
	for it = 0; it < b.N; it++ { // repeat
		var _, err = tsp.SolveWithMatrix(m, nil, opt) // run 2-opt ATSP
		if err != nil {
			b.Fatalf("TwoOptOnly (ATSP) failed: %v", err)
		}
	}
}

// ------------------------------------------------------------------------------------
// Auto pipeline (typically Christofides path under symmetric TSP).
// Medium size; includes MST + matching + Eulerian + shortcut.
// ------------------------------------------------------------------------------------

// BenchmarkAuto_SymmetricChristofides_n200 measures the Auto pipeline on 200 points.
func BenchmarkAuto_SymmetricChristofides_n200(b *testing.B) {
	// Size balanced to keep the run well below ~100ms per iteration on CI.
	const n = 200 // number of vertices
	// Build mildly perturbed circle to avoid degenerate ties.
	var pts = make([][2]float64, n) // coordinate buffer
	var i int                       // iterator
	var th float64                  // angle
	var r float64                   // radius with ripple
	for i = 0; i < n; i++ {         // fill coordinates
		th = 2.0 * 3.141592653589793 * float64(i) / float64(n)  // angle
		r = 1.0 + 0.02*float64((i*3)%8)                         // small ripple
		pts[i] = [2]float64{r * math.Cos(th), r * math.Sin(th)} // Cartesian
	}
	// Build symmetric matrix once.
	var m = euclid(pts) // symmetric metric

	// Configure Auto pipeline with local search enabled (default-leaning).
	var opt = tsp.DefaultOptions() // defaults
	opt.Symmetric = true           // symmetric TSP
	opt.StartVertex = startV       // canonical start
	opt.Eps = epsTiny              // strict tolerance
	opt.EnableLocalSearch = true   // enable local search refinements
	opt.Seed = seedDet             // fixed seed when RNG is used internally

	// Benchmark loop.
	b.ReportAllocs()             // allocation stats
	b.ResetTimer()               // reset timer
	var it int                   // iteration counter
	for it = 0; it < b.N; it++ { // repeat
		var _, err = tsp.SolveWithMatrix(m, nil, opt) // run Auto
		if err != nil {
			b.Fatalf("Auto (symmetric) failed: %v", err)
		}
	}
}

// ------------------------------------------------------------------------------------
// Micro-benchmarks: MST (Prim), Eulerian (Hierholzer), TourCost.
// These isolate hot primitives used across pipelines.
// ------------------------------------------------------------------------------------

// BenchmarkMST_Prim_n512 measures Prim’s O(n^2) MST on a 512-point symmetric metric.
func BenchmarkMST_Prim_n512(b *testing.B) {
	// Size selected to emphasize O(n^2) cost while staying swift on CI.
	const n = 512 // number of vertices
	// Coordinates: rippled circle.
	var pts = make([][2]float64, n) // coordinates
	var i int                       // iterator
	var th float64                  // angle
	var r float64                   // ripple radius
	for i = 0; i < n; i++ {         // fill points
		th = 2.0 * 3.141592653589793 * float64(i) / float64(n)  // angle
		r = 1.0 + 0.005*float64((i*11)%17)                      // tiny ripple
		pts[i] = [2]float64{r * math.Cos(th), r * math.Sin(th)} // position
	}
	// Build symmetric matrix once.
	var m = euclid(pts) // symmetric

	// Benchmark loop.
	b.ReportAllocs()             // allocation stats
	b.ResetTimer()               // reset timer
	var it int                   // iteration counter
	for it = 0; it < b.N; it++ { // repeat MST
		var _, _, err = tsp.MinimumSpanningTree(m) // run Prim O(n^2)
		if err != nil {
			b.Fatalf("MinimumSpanningTree failed: %v", err)
		}
	}
}

// BenchmarkEulerian_Hierholzer_n512 measures Hierholzer on a doubled MST.
func BenchmarkEulerian_Hierholzer_n512(b *testing.B) {
	// Use the same 512-point metric as MST bench to make results relatable.
	const n = 512 // number of vertices
	// Coordinates: rippled circle for determinism and non-triviality.
	var pts = make([][2]float64, n) // coordinates
	var i int                       // iterator
	var th float64                  // angle
	var r float64                   // ripple
	for i = 0; i < n; i++ {         // fill points
		th = 2.0 * 3.141592653589793 * float64(i) / float64(n)
		r = 1.0 + 0.005*float64((i*11)%17)
		pts[i] = [2]float64{r * math.Cos(th), r * math.Sin(th)}
	}
	// Build symmetric matrix and MST once outside timer.
	var m = euclid(pts)                             // symmetric
	var _, mstAdj, err = tsp.MinimumSpanningTree(m) // compute MST
	if err != nil {
		b.Fatalf("MinimumSpanningTree failed: %v", err)
	}
	// Double edges to get an Eulerian multigraph; keep a base copy.
	var baseMulti = doubleAdj(mstAdj) // multigraph ready

	// Benchmark loop.
	b.ReportAllocs()             // allocation stats
	b.ResetTimer()               // reset timer
	var it int                   // iteration counter
	for it = 0; it < b.N; it++ { // repeat Hierholzer
		// Make a working copy per run because Hierholzer may consume edges.
		var multi = deepCopyAdj(baseMulti)         // fresh copy for this iteration
		var _ = tsp.EulerianCircuit(multi, startV) // compute Eulerian circuit from startV
		// We intentionally ignore the returned walk; the purpose is timing traversal.
	}
}

// BenchmarkTourCost_n200 measures cost computation on a 200-vertex ring tour.
func BenchmarkTourCost_n200(b *testing.B) {
	// Use a 200-point symmetric metric (same as TwoOpt bench size).
	const n = 200 // number of vertices
	// Coordinates: mild ripple to keep distances generic.
	var pts = make([][2]float64, n) // coordinates
	var i int                       // iterator
	var th float64                  // angle
	var r float64                   // ripple
	for i = 0; i < n; i++ {         // fill points
		th = 2.0 * 3.141592653589793 * float64(i) / float64(n)
		r = 1.0 + 0.015*float64((i*7)%11)
		pts[i] = [2]float64{r * math.Cos(th), r * math.Sin(th)}
	}
	// Build symmetric matrix once.
	var m = euclid(pts) // symmetric
	// Construct a trivial perimeter tour 0→1→…→n−1→0.
	var tour = make([]int, n+1) // closed tour of length n+1
	for i = 0; i < n; i++ {     // fill tour vertices
		tour[i] = i // sequential order
	}
	tour[n] = 0 // close the cycle

	// Benchmark loop.
	b.ReportAllocs()             // allocation stats
	b.ResetTimer()               // reset timer
	var it int                   // iteration counter
	for it = 0; it < b.N; it++ { // repeat TourCost
		var _, err = tsp.TourCost(m, tour) // compute total cost
		if err != nil {
			b.Fatalf("TourCost failed: %v", err)
		}
	}
}
