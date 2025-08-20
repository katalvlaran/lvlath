// Package tsp — tour utilities shared by exact/heuristic solvers.
//
// This file contains compact, allocation-conscious utilities that operate purely
// on tour structure (index sequences), without depending on distance matrices.
// Provided helpers:
//   - ValidatePermutation: verify a permutation over {0..n-1}.
//   - MakeTourFromPermutation: build a closed tour from a permutation, rotated to a start.
//   - ValidateTour: enforce Hamiltonian cycle invariants.
//   - RotateTourToStart: cyclic shift so the tour starts/ends at a given vertex.
//   - CanonicalizeOrientationInPlace: canonical direction w.r.t. neighbors of start.
//   - reverseArcInPlace: in-place segment reversal (2-opt core).
//   - IndexOfStart: locate start in [0..n-1] prefix.
//   - CopyTour: independent shallow copy of a tour slice.
//   - EqualToursModuloRotation: equality under rotation (fixed start, same direction).
//   - DebugString: compact printable representation for tests/debug.
//   - ShortcutEulerianToHamiltonian: skip revisits in an Eulerian sequence to form a tour.
//
// Design:
//   - No logging, no panics on user input — only sentinel errors from types.go.
//   - O(n) time for most helpers; in-place mutations avoid extra allocations.
//   - Deterministic behavior with clear pre/post-conditions.
package tsp

import "fmt"

// ValidatePermutation checks that perm is a permutation of {0..n-1} of length n.
// It does not allocate besides a single O(n) boolean marker slice.
//
// Complexity: O(n) time, O(n) space.
func ValidatePermutation(perm []int, n int) error {
	if len(perm) != n {
		return ErrDimensionMismatch
	}
	if n <= 0 {
		return ErrDimensionMismatch
	}
	seen := make([]bool, n)

	var (
		i int
		v int
	)
	for i = 0; i < n; i++ {
		v = perm[i]
		// Out-of-range element violates the dimension contract.
		if v < 0 || v >= n {
			return ErrDimensionMismatch
		}
		// Duplicate also violates the bijection/dimension contract.
		if seen[v] {
			return ErrDimensionMismatch
		}
		seen[v] = true
	}
	return nil
}

// MakeTourFromPermutation builds a closed Hamiltonian tour from a vertex permutation.
// Steps:
//  1. Validate that perm is a permutation of {0..n-1}.
//  2. Find the index of start in perm; rotate so perm[idx]==start becomes position 0.
//  3. Return a new slice of length n+1 with the closing start at position n.
//
// Contract:
//   - perm is a permutation (ValidatePermutation).
//   - start ∈ [0..n-1] and present in perm.
//   - Returned tour satisfies: len==n+1, tour[0]==tour[n]==start.
//
// Complexity: O(n) time, O(n) space.
func MakeTourFromPermutation(perm []int, n int, start int) ([]int, error) {
	if err := ValidatePermutation(perm, n); err != nil {
		return nil, err
	}
	if start < 0 || start >= n {
		return nil, ErrStartOutOfRange
	}

	// Locate start inside perm.
	var (
		i     int
		pivot = -1
	)
	for i = 0; i < n; i++ {
		if perm[i] == start {
			pivot = i
			break
		}
	}
	if pivot == -1 {
		// The permutation does not contain start — inconsistent input shape.
		return nil, ErrDimensionMismatch
	}

	// Rotate into a fresh [n+1] tour and close with start.
	tour := make([]int, n+1)
	for i = 0; i < n; i++ {
		tour[i] = perm[(pivot+i)%n]
	}
	tour[n] = start
	return tour, nil
}

// ValidateTour enforces Hamiltonian-cycle invariants (see types.go):
//
//	len(tour) == n+1, tour[0]==tour[n]==start,
//	each vertex v∈[0..n-1] appears exactly once in positions [0..n-1].
//
// Returns nil if valid.
//
// Complexity: O(n) time, O(n) space.
func ValidateTour(tour []int, n int, start int) error {
	if n <= 0 {
		return ErrDimensionMismatch
	}
	if len(tour) != n+1 {
		return ErrDimensionMismatch
	}
	if start < 0 || start >= n {
		return ErrStartOutOfRange
	}
	if tour[0] != start || tour[n] != start {
		return ErrDimensionMismatch
	}

	seen := make([]bool, n)

	var (
		i int
		v int
	)
	for i = 0; i < n; i++ {
		v = tour[i]
		if v < 0 || v >= n {
			return ErrDimensionMismatch
		}
		if seen[v] {
			return ErrDimensionMismatch
		}
		seen[v] = true
	}
	return nil
}

// RotateTourToStart returns a fresh copy of the tour shifted so that
// out[0] == start and out[n] == start. The input may be either a closed tour
// (len==n+1) or a raw path (len==n, no closing vertex). In the raw-path case,
// the function appends the closing start.
//
// Pre-conditions:
//   - start must appear in tour at least once within the first n elements.
//
// Complexity: O(n) time, O(n) space.
func RotateTourToStart(tour []int, start int) ([]int, error) {
	if len(tour) == 0 {
		return nil, ErrDimensionMismatch
	}

	// Determine n (number of unique vertices).
	var (
		n        int
		isClosed bool
	)
	if tour[0] == tour[len(tour)-1] {
		n = len(tour) - 1
		isClosed = true
	} else {
		n = len(tour)
	}
	if start < 0 || start >= n {
		return nil, ErrStartOutOfRange
	}

	// Find start in the first n entries.
	var (
		i     int
		pivot = -1
	)
	for i = 0; i < n; i++ {
		if tour[i] == start {
			pivot = i
			break
		}
	}
	if pivot == -1 {
		return nil, ErrDimensionMismatch
	}

	// Build rotated copy and close it.
	out := make([]int, n+1)

	for i = 0; i < n; i++ {
		out[i] = tour[(pivot+i)%n]
	}
	out[n] = start

	_ = isClosed // documented above; not used in fast path.
	return out, nil
}

// CanonicalizeOrientationInPlace fixes the tour direction under a fixed start.
// If the right neighbor tour[1] is lexicographically “worse” than the left
// neighbor tour[n-1], the interior segment [1..n-1] is reversed in place.
// This yields a unique canonical orientation for the same cyclic order.
//
// Requirements:
//   - len(tour) == n+1 and tour[0]==tour[n] (already closed).
//   - The permutation part is assumed valid.
//
// Complexity: O(n) time, O(1) space.
func CanonicalizeOrientationInPlace(tour []int) error {
	if len(tour) < 3 {
		return ErrDimensionMismatch
	}
	var n = len(tour) - 1
	if tour[0] != tour[n] {
		return ErrDimensionMismatch
	}
	// Compare right vs left neighbor of start (indices 1 and n-1).
	if tour[1] > tour[n-1] {
		return reverseArcInPlace(tour, 1, n-1)
	}
	return nil
}

// reverseArcInPlace reverses the inclusive segment tour[i..k] in place,
// keeping the closing vertex intact. This is the primitive used by 2-opt.
//
// Contracts:
//   - The tour is closed: len(tour)==n+1 and tour[0]==tour[n].
//   - Indices satisfy: 1 ≤ i < k ≤ n-1.
//
// Complexity: O(k-i) time, O(1) space.
func reverseArcInPlace(tour []int, i, k int) error {
	var n = len(tour) - 1
	if n < 2 {
		return ErrDimensionMismatch
	}
	if tour[0] != tour[n] {
		return ErrDimensionMismatch
	}
	if i < 1 || k > n-1 || i >= k {
		return ErrDimensionMismatch
	}
	for i < k {
		tour[i], tour[k] = tour[k], tour[i]
		i++
		k--
	}
	return nil
}

// IndexOfStart returns the index of the first occurrence of start within the
// prefix [0..n-1] (ignores the closing vertex at n). Returns -1 if not found.
//
// Complexity: O(n) time.
func IndexOfStart(tour []int, start int) int {
	if len(tour) == 0 {
		return -1
	}
	var n int
	if tour[0] == tour[len(tour)-1] {
		n = len(tour) - 1
	} else {
		n = len(tour)
	}

	var i int
	for i = 0; i < n; i++ {
		if tour[i] == start {
			return i
		}
	}
	return -1
}

// CopyTour returns an independent copy of the input tour slice.
//
// Complexity: O(n) time, O(n) space.
func CopyTour(tour []int) []int {
	if tour == nil {
		return nil
	}
	out := make([]int, len(tour))
	copy(out, tour)
	return out
}

// EqualToursModuloRotation checks equality of two closed tours under rotation
// (fixed start value, same direction). Assumes both inputs are closed (len==n+1).
//
// Complexity: O(n) time.
func EqualToursModuloRotation(a, b []int) bool {
	if len(a) != len(b) || len(a) < 2 {
		return false
	}
	var (
		n  = len(a) - 1
		st = a[0]
	)
	if a[n] != st || b[n] != b[0] {
		return false
	}
	// Find st in b[0..n-1].
	var (
		j int
		p = -1
	)
	for j = 0; j < n; j++ {
		if b[j] == st {
			p = j
			break
		}
	}
	if p == -1 {
		return false
	}
	// Compare by rotation.
	var i int
	for i = 0; i < n; i++ {
		if a[i] != b[(p+i)%n] {
			return false
		}
	}
	return true
}

// DebugString returns a compact printable representation for tests/debug,
// e.g. "[0 3 1 2 | 0]" where the vertical bar marks the closure.
//
// Complexity: O(n) time, O(n) space for formatting.
func DebugString(tour []int) string {
	if len(tour) == 0 {
		return "[]"
	}
	var (
		n = len(tour) - 1
		s = "["
		i int
	)
	for i = 0; i < n; i++ {
		if i > 0 {
			s += " "
		}
		s += fmt.Sprintf("%d", tour[i])
	}
	s += " | "
	if n >= 0 {
		s += fmt.Sprintf("%d", tour[n])
	}
	s += "]"
	return s
}

// ShortcutEulerianToHamiltonian converts an Eulerian vertex sequence (with revisits)
// into a Hamiltonian cycle by skipping the first revisits and then closing the tour.
// This is the standard “shortcutting” step in Christofides:
//
//	Input:  euler — a vertex sequence of arbitrary length (often O(E)).
//	        n     — number of unique vertices (0..n-1).
//	        start — required starting vertex of the resulting tour.
//
// Algorithm:
//   - Maintain a visited[n] boolean array.
//   - Scan euler left-to-right; append a vertex v the first time it is seen.
//   - After the scan, ensure every vertex 0..n-1 was seen exactly once.
//   - Rotate the resulting n-length cycle so it starts at `start` and close it.
//
// Contracts:
//   - 0 ≤ v < n for every v ∈ euler; otherwise ErrDimensionMismatch.
//   - start ∈ [0..n-1].
//
// Returns:
//   - tour of length n+1 with tour[0]==tour[n]==start,
//   - ErrDimensionMismatch if euler misses some vertices or has out-of-range entries,
//   - ErrStartOutOfRange if start is invalid.
//
// Complexity: O(len(euler) + n) time, O(n) space.
func ShortcutEulerianToHamiltonian(euler []int, n int, start int) ([]int, error) {
	if n <= 0 {
		return nil, ErrDimensionMismatch
	}
	if start < 0 || start >= n {
		return nil, ErrStartOutOfRange
	}

	visited := make([]bool, n)
	cycle := make([]int, 0, n) // collect first occurrences

	var (
		idx int
		v   int
	)
	for idx = 0; idx < len(euler); idx++ {
		v = euler[idx]
		if v < 0 || v >= n {
			return nil, ErrDimensionMismatch
		}
		if !visited[v] {
			visited[v] = true
			cycle = append(cycle, v)
		}
	}

	// Ensure all vertices were seen exactly once.
	if len(cycle) != n {
		return nil, ErrDimensionMismatch
	}
	var i int
	for i = 0; i < n; i++ {
		if !visited[i] {
			return nil, ErrDimensionMismatch
		}
	}

	// Rotate to start and close.
	var p = -1
	for i = 0; i < n; i++ {
		if cycle[i] == start {
			p = i
			break
		}
	}
	if p == -1 {
		return nil, ErrDimensionMismatch
	}

	tour := make([]int, n+1)
	for i = 0; i < n; i++ {
		tour[i] = cycle[(p+i)%n]
	}
	tour[n] = start
	return tour, nil
}
