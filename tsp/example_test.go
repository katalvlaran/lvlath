// Package tsp_test provides runnable, deterministic examples that demonstrate
// how to solve (A)TSP with lvlath/tsp. Each example prints a tour and cost
// (or the optimal cost for BnB) with a stable // Output: block.
//
// Design goals:
//   - Deterministic: fixed seeds and synthetic metrics → identical output on CI.
//   - Minimal dependencies: a tiny local matrix.Matrix implementation is used
//     so that examples remain self-contained and easy to read.
//   - Readability: every step is documented; variables are predeclared; no
//     hidden allocations inside the hot loop.
//
// Contents:
//  1. Example_Christofides_TwoOpt_Metric5  (symmetric, n=5)
//  2. Example_TwoOptOnly_ATSP5             (asymmetric,  n=5)
//  3. Example_BranchAndBound_n6            (exact,       n=6)
package tsp_test

import (
	"fmt"
	"math"

	"github.com/katalvlaran/lvlath/matrix"
	"github.com/katalvlaran/lvlath/tsp"
)

// -----------------------------------------------------------------------------
// Local minimal dense matrix implementation (satisfies matrix.Matrix).
// We keep it tiny and explicit to make the examples self-contained.
// -----------------------------------------------------------------------------

// exDense is a simple dense matrix with [][]float64 storage and bound checks.
type exDense struct{ a [][]float64 }

// Ensure interface compliance at compile time.
var _ matrix.Matrix = exDense{}

// Rows returns the number of rows in the matrix.
func (m exDense) Rows() int { return len(m.a) }

// Cols returns the number of columns in the matrix (0 if no rows).
func (m exDense) Cols() int {
	if len(m.a) == 0 {
		return 0
	}
	return len(m.a[0])
}

// At returns the element at (i,j) with defensive bound checks.
func (m exDense) At(i, j int) (float64, error) {
	// Validate bounds explicitly; propagate the matrix package sentinel on error.
	if i < 0 || i >= m.Rows() || j < 0 || j >= m.Cols() {
		return 0, matrix.ErrIndexOutOfBounds
	}
	return m.a[i][j], nil
}

// Set writes the element at (i,j) with defensive bound checks.
func (m exDense) Set(i, j int, v float64) error {
	// Validate bounds explicitly; propagate the sentinel on error.
	if i < 0 || i >= m.Rows() || j < 0 || j >= m.Cols() {
		return matrix.ErrIndexOutOfBounds
	}
	m.a[i][j] = v
	return nil
}

// Clone produces a deep copy (rows are duplicated to avoid aliasing).
func (m exDense) Clone() matrix.Matrix {
	var n = m.Rows() // number of rows
	var cp = make([][]float64, n)
	var r int               // row index
	for r = 0; r < n; r++ { // copy each row
		cp[r] = append([]float64(nil), m.a[r]...)
	}
	return exDense{a: cp} // return the wrapped deep copy
}

// -----------------------------------------------------------------------------
// Metric generators (deterministic).
// -----------------------------------------------------------------------------

// makeRingMetric returns an n×n symmetric metric with circular graph distances:
// d(i,j) = min(|i-j|, n-|i-j|). This is a valid metric (triangle inequality holds).
// The optimal Hamiltonian cycle has total length n (use edges of length 1).
func makeRingMetric(n int) matrix.Matrix {
	// Allocate n×n matrix with zeros on the diagonal.
	var a = make([][]float64, n) // outer slice for rows
	var i, j int                 // loop iterators
	for i = 0; i < n; i++ {      // allocate each row
		a[i] = make([]float64, n) // fill row i
	}
	// Fill all pairwise distances using the circular metric.
	var diff int            // absolute index difference
	var dist int            // ring distance
	for i = 0; i < n; i++ { // iterate rows
		for j = 0; j < n; j++ { // iterate columns
			if i == j {
				a[i][j] = 0 // exact zero on the diagonal
				continue    // skip self
			}
			// Compute absolute difference on the index line.
			if i > j {
				diff = i - j
			} else {
				diff = j - i
			}
			// Map to the ring distance by taking the shorter arc.
			if diff < n-diff {
				dist = diff
			} else {
				dist = n - diff
			}
			// Store as float64 (solver expects float64 distances).
			a[i][j] = float64(dist)
		}
	}
	// Return as exDense which implements matrix.Matrix.
	return exDense{a: a}
}

// makeAsymEuclid builds an asymmetric matrix from 2D points with a small directional penalty.
// Base distance is Euclidean; for i>j we add 'bias' to break symmetry (ATSP).
func makeAsymEuclid(pts [][2]float64, bias float64) matrix.Matrix {
	var n = len(pts)             // number of points
	var a = make([][]float64, n) // allocate outer slice
	var i, j int                 // loop iterators
	for i = 0; i < n; i++ {      // allocate rows
		a[i] = make([]float64, n) // allocate row i
	}
	// Compute directed distances with bias on one orientation.
	var dx, dy, d float64 // deltas and Euclidean distance
	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			if i == j {
				a[i][j] = 0 // exact zeros on diagonal
				continue
			}
			dx = pts[i][0] - pts[j][0] // delta-X
			dy = pts[i][1] - pts[j][1] // delta-Y
			d = math.Hypot(dx, dy)     // Euclidean distance √(dx²+dy²)
			if i > j {
				d += bias // apply directional penalty for one orientation
			}
			a[i][j] = d // store directed distance
		}
	}
	return exDense{a: a} // wrap and return
}

// -----------------------------------------------------------------------------
// Tour normalization helpers for stable printing.
// -----------------------------------------------------------------------------

// rotateToStartZero rotates a tour so that it starts at vertex 0 (supports open or closed).
func rotateToStartZero(tour []int) []int {
	// If the tour is closed (n+1 with tour[0]==tour[n]) — strip the closing vertex later.
	var n = len(tour) // current length
	// Find index of vertex 0 (guaranteed to exist in valid tours).
	var i int
	for i = 0; i < n; i++ {
		if tour[i] == 0 {
			break // found the pivot
		}
	}
	// Rotate the slice so that 0 appears at position 0.
	var rotated = append(append([]int(nil), tour[i:]...), tour[:i]...)
	// If the tour is closed, ensure the last element equals the first (0).
	if len(rotated) >= 2 && rotated[0] == rotated[len(rotated)-1] {
		return rotated // already closed properly
	}
	return rotated
}

// toOpen normalizes a cycle to an open tour of length n by removing the closing vertex.
func toOpen(tour []int) []int {
	// If closed (n+1, last equals first) — drop the last element.
	if len(tour) >= 2 && tour[0] == tour[len(tour)-1] {
		return tour[:len(tour)-1] // return open form
	}
	return tour // already open
}

// -----------------------------------------------------------------------------
// 1) Symmetric TSP (n=5) — Christofides → 2-opt on a circular metric.
// -----------------------------------------------------------------------------

func Example_Christofides_TwoOpt_Metric5() {
	// Build a deterministic 5×5 ring metric: d(i,j)=min(|i-j|, 5-|i-j|).
	// The optimal Hamiltonian cycle uses 5 edges of length 1 → cost=5.
	var m = makeRingMetric(5) // symmetric metric

	// Configure the dispatcher for the symmetric TSP:
	//   - Default Auto path uses Christofides; EnableLocalSearch lets 2-opt polish.
	var opt = tsp.DefaultOptions() // start from sane defaults
	opt.Symmetric = true           // symmetric TSP mode
	opt.StartVertex = startV       // canonical start=0
	opt.Eps = epsTiny              // strict acceptance epsilon
	opt.EnableLocalSearch = true   // allow 2-opt post-improvement
	opt.Seed = seedDet             // fixed seed (RNG may be used internally)

	// Solve the instance end-to-end.
	var res, err = tsp.SolveWithMatrix(m, nil, opt) // run Auto (Christofides→Euler→2-opt)
	if err != nil {                                 // examples must never panic; print fatal errors
		fmt.Printf("solve failed: %v\n", err)
		return
	}

	// Normalize the returned tour (closed) to start at 0 for stable printing.
	var closed = rotateToStartZero(res.Tour) // rotate so that vertex 0 is first
	var open = toOpen(closed)                // strip the closing vertex for nicer formatting

	// Print the tour as a closed cycle 0->…->0 and the total cost.
	fmt.Printf("Christofides→TwoOpt (n=5, symmetric)\n") // header line
	fmt.Printf("Tour: ")
	var i int // loop iterator
	for i = 0; i < len(open); i++ {
		if i > 0 {
			fmt.Print("->") // ASCII arrow between vertices
		}
		fmt.Printf("%d", open[i]) // print vertex index
	}
	fmt.Printf("->0\n")                  // close the cycle visually
	fmt.Printf("Cost: %.3f\n", res.Cost) // print stabilized cost with 3 decimals

	// Output:
	// Christofides→TwoOpt (n=5, symmetric)
	// Tour: 0->1->2->3->4->0
	// Cost: 5.000
}

// -----------------------------------------------------------------------------
// 2) Asymmetric TSP (n=5) — TwoOptOnly on a biased Euclidean metric.
// -----------------------------------------------------------------------------

func Example_TwoOptOnly_ATSP5() {
	// Construct 5 points on a unit circle (deterministic angles).
	const n = 5 // number of vertices
	var pts = make([][2]float64, n)
	var i int      // loop iterator
	var th float64 // angle accumulator
	for i = 0; i < n; i++ {
		th = 2.0 * math.Pi * float64(i) / float64(n)    // equally spaced angle
		pts[i] = [2]float64{math.Cos(th), math.Sin(th)} // Cartesian coords
	}
	// Build an asymmetric matrix by adding a small directional penalty.
	var m = makeAsymEuclid(pts, 0.15) // ATSP metric

	// Configure a pure 2-opt run in ATSP mode with deterministic shuffle order.
	var opt = tsp.DefaultOptions() // defaults
	opt.Algo = tsp.TwoOptOnly      // restrict solver to Two-Opt neighborhood
	opt.Symmetric = false          // ATSP (directed) mode
	opt.StartVertex = startV       // canonical start
	opt.Eps = epsTiny              // strict acceptance epsilon
	opt.EnableLocalSearch = true   // enable local search
	opt.ShuffleNeighborhood = true // deterministic shuffle under fixed seed
	opt.Seed = seedDet             // fixed seed for reproducibility

	// Solve the instance.
	var res, err = tsp.SolveWithMatrix(m, nil, opt) // run ATSP 2-opt
	if err != nil {
		fmt.Printf("solve failed: %v\n", err)
		return
	}

	// Normalize to start at 0 and print the closed cycle and cost.
	var closed = rotateToStartZero(res.Tour) // rotate so that 0 is first
	var open = toOpen(closed)                // open cycle form for pretty print

	fmt.Printf("TwoOptOnly (n=5, ATSP)\n") // header
	fmt.Printf("Tour: ")
	for i = 0; i < len(open); i++ {
		if i > 0 {
			fmt.Print("->")
		}
		fmt.Printf("%d", open[i])
	}
	fmt.Printf("->0\n")                  // close visually
	fmt.Printf("Cost: %.3f\n", res.Cost) // print cost with 3 decimals

	// Output:
	// TwoOptOnly (n=5, ATSP)
	// Tour: 0->1->2->3->4->0
	// Cost: 5.878
}

// -----------------------------------------------------------------------------
// 3) Exact Branch-and-Bound (n=6) — ring metric with unique OPT=6.
// -----------------------------------------------------------------------------

func Example_BranchAndBound_n6() {
	// Build a 6×6 ring metric (symmetric). Any Hamiltonian cycle via unit edges
	// yields total cost 6, while any non-adjacent jump costs ≥2 → OPT=6.
	var m = makeRingMetric(6) // symmetric metric with integer distances

	// Configure the exact solver with a strong admissible bound.
	var opt = tsp.DefaultOptions()   // defaults
	opt.Algo = tsp.BranchAndBound    // exact Branch-and-Bound
	opt.Symmetric = true             // symmetric TSP
	opt.StartVertex = startV         // canonical start
	opt.Eps = epsTiny                // not relevant for exact, but consistent
	opt.BoundAlgo = tsp.OneTreeBound // stronger admissible lower bound
	opt.EnableLocalSearch = false    // no heuristics in exact mode
	opt.Seed = seedDet               // irrelevant here, but kept consistent

	// Solve exactly; res.Cost must equal 6.000 on this instance.
	var res, err = tsp.SolveWithMatrix(m, nil, opt) // exact solve
	if err != nil {
		fmt.Printf("solve failed: %v\n", err)
		return
	}

	// Print the optimal cost (no need to print the exact tour shape here).
	fmt.Printf("Branch&Bound (n=6) OPT: %.3f\n", res.Cost) // stable format

	// Output:
	// Branch&Bound (n=6) OPT: 6.000
}

/*
// Package main demonstrates a real-world logistics scenario using lvlath/core and lvlath/matrix
// to build a weighted graph of 10 locations, convert it to a distance matrix, and then solve
// the TSP with lvlath/tsp. We use TSPApprox (Christofides) to plan a near-optimal delivery route.
//
// Scenario:
//
//	A delivery company must dispatch a single vehicle from the “Hub” warehouse to  nine retail
//	outlets and return. We model the road network as an undirected, weighted graph where vertices
//	are locations and edges are the driving distances in kilometers. Converting to an adjacency
//	matrix and running TSPApprox (O(n³)) yields a practical route in milliseconds.
//
// Use case:
//
//	Daily route planning for last-mile deliveries across urban and suburban locations.
//
// Playground: [![Go Playground – TSP Logistics](https://img.shields.io/badge/Go_Playground-TSP_Logistics-blue?logo=go)](https://play.golang.org/p/your-snippet-id)
package tsp_test

import (
	"fmt"
	"log"

	"github.com/katalvlaran/lvlath/core"   // core graph types
	"github.com/katalvlaran/lvlath/matrix" // core graph types
	"github.com/katalvlaran/lvlath/tsp"    // TSP solvers
)

const (
	Hub        = "Hub"
	NorthMall  = "NorthMall"
	EastPlaza  = "EastPlaza"
	SouthPark  = "SouthPark"
	WestSide   = "WestSide"
	Uptown     = "Uptown"
	Downtown   = "Downtown"
	Airport    = "Airport"
	University = "University"
	Stadium    = "Stadium"
)

func ExampleTSP() {
	// 1) Build the weighted road network graph (undirected, weighted distances in km)
	g := core.NewGraph(core.WithWeighted())
	locations := []string{
		Hub, NorthMall, EastPlaza, SouthPark, WestSide,
		Uptown, Downtown, Airport, University, Stadium,
	}
	for _, loc := range locations {
		if err := g.AddVertex(loc); err != nil {
			log.Fatalf("add vertex %s: %v", loc, err)
		}
	}
	// Add pairwise roads (symmetric distances)
	roads := []struct {
		u, v string
		d    int64
	}{
		{Hub, NorthMall, 12}, {Hub, EastPlaza, 18}, {Hub, SouthPark, 20}, {Hub, WestSide, 15},
		{NorthMall, EastPlaza, 7}, {EastPlaza, SouthPark, 10}, {SouthPark, WestSide, 8}, {WestSide, NorthMall, 9},
		{NorthMall, Uptown, 6}, {Uptown, Downtown, 5}, {Downtown, EastPlaza, 11},
		{SouthPark, Airport, 14}, {Airport, University, 13}, {University, Stadium, 9}, {Stadium, Downtown, 12},
	}
	for _, r := range roads {
		if _, err := g.AddEdge(r.u, r.v, r.d); err != nil {
			log.Fatalf("add edge %s-%s: %v", r.u, r.v, err)
		}
	}

	// 2) Convert graph to adjacency matrix
	optsMat := matrix.NewMatrixOptions(matrix.WithWeighted(true))
	am, err := matrix.NewAdjacencyMatrix(g, optsMat)
	if err != nil {
		log.Fatalf("matrix conversion: %v", err)
	}
	// 'am.Index' maps location name → matrix index
	// 'am.Data' is the [][]float64 distance matrix

	// 3) Solve TSP via 1.5-approximation (Christofides)
	tspOpts := tsp.DefaultOptions()
	//res, err := tsp.TSPApprox(am.Data, tspOpts)
	res, err := tsp.TSPApprox(am.Mat, tspOpts)
	if err != nil {
		log.Fatalf("TSPApprox failed: %v", err)
	}

	// 4) Print route without extra indentation
	fmt.Println("Planned delivery route:")
	for i, idx := range res.Tour {
		fmt.Printf("%d: %s\n", i, locations[idx])
	}
	fmt.Printf("\nTotal distance: %.0f km\n", res.Cost)
	// Output:
	// Planned delivery route:
	// 0: Hub
	// 1: Airport
	// 2: NorthMall
	// 3: Uptown
	// 4: WestSide
	// 5: EastPlaza
	// 6: University
	// 7: SouthPark
	// 8: Downtown
	// 9: Stadium
	// 10: Hub
	//
	// Total distance: 20 km
}

*/
