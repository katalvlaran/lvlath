// Package main demonstrates planning an efficient drone delivery route
// using the Christofides 1.5-approximation TSP solver in lvlath/tsp.
//
// Playground: https://go.dev/play/p/Sbje_qF16V7
//
// Scenario:
//
//	A delivery drone departs from a base and must visit several waypoints
//	(e.g. customer locations) before returning home.  We model each point
//	in 2D, compute the full distance matrix (Euclidean), and then use
//	tsp.TSPApprox to quickly compute a near-optimal round-trip.
//
// Waypoints (x,y):
//
//	Base  : (0, 0)
//	Point1: (2, 3)
//	Point2: (5, 2)
//	Point3: (6, 6)
//	Point4: (8, 3)
//	Point5: (7, 0)
//
// Use case:
//
//	Planning drone routes for last-mile logistics: fast approximation
//	for n≲20 waypoints, trading perfect optimality for speed.
//
// Complexity:
//   - Building matrix: O(n²)
//   - Christofides: O(n³) (for matching & Euler tour)
//   - Total: O(n³), with n = number of points.
//
// Expected output:
//   - Sequence of waypoint indices in visit order (starting+ending at 0).
//   - Total estimated distance of the cycle.
package main

//
//import (
//	"fmt"
//	"math"
//
//	"github.com/katalvlaran/lvlath/tsp"
//)
//
//func main10() {
//	// 1) Define the delivery points (including base at index 0)
//	points := [][2]float64{
//		{0, 0}, // Base
//		{2, 3},
//		{5, 2},
//		{6, 6},
//		{8, 3},
//		{7, 0},
//	}
//	n := len(points)
//
//	// 2) Build the full symmetric distance matrix (Euclidean distances)
//	dist := make([][]float64, n)
//	for i := range dist {
//		dist[i] = make([]float64, n)
//		for j := range dist {
//			if i == j {
//				dist[i][j] = 0
//			} else {
//				dx := points[i][0] - points[j][0]
//				dy := points[i][1] - points[j][1]
//				dist[i][j] = math.Hypot(dx, dy)
//			}
//		}
//	}
//
//	// 3) Compute the 1.5-approximate TSP tour
//	result, err := tsp.TSPApprox(dist)
//	if err != nil {
//		fmt.Println("Error computing TSP:", err)
//		return
//	}
//
//	// 4) Display the route and total cost
//	fmt.Printf("Drone delivery route (indices): %v\n", result.Tour)
//	fmt.Printf("Total approximate distance: %.2f units\n", result.Cost)
//	fmt.Println("\nVisit order with coordinates:")
//	for _, idx := range result.Tour {
//		p := points[idx]
//		label := "Waypoint"
//		if idx == 0 {
//			label = "Base"
//		}
//		fmt.Printf("  %s %d: (%.0f, %.0f)\n", label, idx, p[0], p[1])
//	}
//}
