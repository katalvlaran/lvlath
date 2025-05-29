// Package matrix provides self-contained, low-level utilities for constructing
// and manipulating graph representation matrices without external dependencies.
package matrix

import (
	"math"

	"github.com/katalvlaran/lvlath/core"
)

// BuildAdjacencyData constructs the index map and dense adjacency matrix from
// a list of vertex IDs and core.Edge values, applying the given MatrixOptions.
// It validates options, checks for unknown vertices, and supports directed,
// weighted, multi-edge, and loop configurations.
// Time: O(V+E); Memory: O(V²).
func BuildAdjacencyData(vertices []string, edges []*core.Edge, opts MatrixOptions) (map[string]int, [][]float64, error) {
	// Build index map
	n := len(vertices)
	idx := make(map[string]int, n)
	for i, v := range vertices {
		idx[v] = i
	}
	// Allocate zero-filled NxN matrix
	data := make([][]float64, n)
	for i := range data {
		data[i] = make([]float64, n)
	}
	// Populate matrix entries
	for _, e := range edges {
		i, ok := idx[e.From]
		if !ok {
			return nil, nil, ErrUnknownVertex
		}
		j, ok := idx[e.To]
		if !ok {
			return nil, nil, ErrUnknownVertex
		}
		w := 1.0
		if opts.Weighted {
			w = float64(e.Weight)
		}
		data[i][j] = w
		if !opts.Directed {
			data[j][i] = w
		}
	}

	return idx, data, nil
}

// BuildIncidenceData constructs the vertex-index mapping, filtered edge list,
// and raw incidence matrix from vertex IDs and core.Edge values. It applies
// MatrixOptions for loops and multi-edges, validates options, and checks
// for unknown vertices.
// Time: O(V+E); Memory: O(V·E).
func BuildIncidenceData(vertices []string, edges []*core.Edge, opts MatrixOptions) (map[string]int, []*core.Edge, [][]int, error) {
	// Build vertex index
	vCount := len(vertices)
	vIdx := make(map[string]int, vCount)
	for i, v := range vertices {
		vIdx[v] = i
	}
	// Filter edges based on options
	seen := make(map[[2]string]bool)
	var cols []*core.Edge
	for _, e := range edges {
		// Skip loops if not allowed
		if e.From == e.To && !opts.AllowLoops {
			continue
		}
		u, v := e.From, e.To
		// Collapse undirected multi-edges if not allowed
		if !opts.AllowMulti && !opts.Directed {
			if u > v {
				u, v = v, u
			}
			key := [2]string{u, v}
			if seen[key] {
				continue
			}
			seen[key] = true
		}
		cols = append(cols, e)
	}
	// Allocate V×E matrix
	eCount := len(cols)
	data := make([][]int, vCount)
	for i := range data {
		data[i] = make([]int, eCount)
	}
	// Populate incidence entries
	for j, e := range cols {
		iF, ok := vIdx[e.From]
		if !ok {
			return nil, nil, nil, ErrUnknownVertex
		}
		iT, ok := vIdx[e.To]
		if !ok {
			return nil, nil, nil, ErrUnknownVertex
		}
		if opts.Directed {
			data[iF][j] = -1
			data[iT][j] = +1
		} else {
			data[iF][j] = +1
			data[iT][j] = +1
		}
	}

	return vIdx, cols, data, nil
}

// TransposeData returns the transpose of the input r×c matrix, producing c×r.
// Time: O(r*c); Memory: O(r*c).
func TransposeData(data [][]float64) [][]float64 {
	r := len(data)
	if r == 0 {
		return [][]float64{}
	}
	c := len(data[0])
	t := make([][]float64, c)
	for i := range t {
		t[i] = make([]float64, r)
	}
	for i := 0; i < r; i++ {
		for j := 0; j < c; j++ {
			t[j][i] = data[i][j]
		}
	}

	return t
}

// MultiplyData multiplies A (r×n) by B (n×c), returning the r×c result.
// Returns ErrDimensionMismatch if inner dimensions differ or inputs are ragged.
// Time: O(r*n*c); Memory: O(r*c).
func MultiplyData(A, B [][]float64) ([][]float64, error) {
	r := len(A)
	if r == 0 {
		return [][]float64{}, nil
	}
	n := len(A[0])
	// Validate A is ragged-free
	for _, row := range A {
		if len(row) != n {
			return nil, ErrDimensionMismatch
		}
	}
	// Validate B dims
	if len(B) != n {
		return nil, ErrDimensionMismatch
	}
	c := len(B[0])
	for _, row := range B[1:] {
		if len(row) != c {
			return nil, ErrDimensionMismatch
		}
	}
	// Allocate result
	res := make([][]float64, r)
	for i := range res {
		res[i] = make([]float64, c)
	}
	// Multiply skipping zeros
	for i := 0; i < r; i++ {
		for k := 0; k < n; k++ {
			v := A[i][k]
			if v == 0 {
				continue
			}
			for j := 0; j < c; j++ {
				res[i][j] += v * B[k][j]
			}
		}
	}

	return res, nil
}

// DegreeFromData returns the sum of each row in data as a float64 slice.
// Time: O(r*c); Memory: O(r).
func DegreeFromData(data [][]float64) []float64 {
	r := len(data)
	deg := make([]float64, r)
	for i := range data {
		sum := 0.0
		for _, v := range data[i] {
			sum += v
		}
		deg[i] = sum
	}

	return deg
}

// findMaxOffDiagonal scans the upper-triangle of the symmetric matrix A
// and returns indices (p,q) of the largest |A[p][q]| and that value.
func findMaxOffDiagonal(A [][]float64) (p, q int, maxOff float64) {
	n := len(A)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			off := math.Abs(A[i][j])
			if off > maxOff {
				maxOff = off
				p, q = i, j
			}
		}
	}

	return
}

// computeJacobiCoefficients computes the cosine and sine of the rotation
// angle for the Jacobi rotation targeting A[p][q].
func computeJacobiCoefficients(A [][]float64, p, q int) (c, s float64) {
	app := A[p][p]
	aqq := A[q][q]
	apq := A[p][q]
	theta := 0.5 * math.Atan2(2*apq, aqq-app)
	c = math.Cos(theta)
	s = math.Sin(theta)

	return
}

// applyJacobi applies the Jacobi rotation defined by (c,s) at indices (p,q)
// to both the working matrix A and the eigenvector matrix V.
func applyJacobi(A, V [][]float64, p, q int, c, s float64) {
	n := len(A)
	// update diagonal
	app, aqq, apq := A[p][p], A[q][q], A[p][q]
	A[p][p] = c*c*app - 2*s*c*apq + s*s*aqq
	A[q][q] = s*s*app + 2*s*c*apq + c*c*aqq
	A[p][q], A[q][p] = 0, 0
	// update off-diagonals
	for i := 0; i < n; i++ {
		if i == p || i == q {
			continue
		}
		aip, aiq := A[i][p], A[i][q]
		A[i][p], A[p][i] = c*aip-s*aiq, c*aip-s*aiq
		A[i][q], A[q][i] = s*aip+c*aiq, s*aip+c*aiq
	}
	// update eigenvector matrix
	for i := 0; i < n; i++ {
		vip, viq := V[i][p], V[i][q]
		V[i][p], V[i][q] = c*vip-s*viq, s*vip+c*viq
	}
}

// EigenDecompose computes eigenvalues and eigenvectors of a symmetric matrix
// using the Jacobi rotation algorithm. It splits responsibilities into
// validation, finding pivots, computing rotation, and applying transformations.
// tol defines convergence threshold; maxIter caps the number of rotations.
// Returns ErrEigenFailed if convergence isn't reached within maxIter.
// Time: O(N³); Memory: O(N²).
func EigenDecompose(orig [][]float64, tol float64, maxIter int) ([]float64, [][]float64, error) {
	n := len(orig)
	if n == 0 || len(orig[0]) != n {
		return nil, nil, ErrDimensionMismatch
	}
	// Copy to avoid mutating input
	A := make([][]float64, n)
	for i := range A {
		A[i] = append([]float64(nil), orig[i]...)
	}
	// Initialize eigenvector matrix as identity
	V := make([][]float64, n)
	for i := range V {
		V[i] = make([]float64, n)
		V[i][i] = 1
	}
	// Jacobi rotation loop
	for iter := 0; iter < maxIter; iter++ {
		p, q, maxOff := findMaxOffDiagonal(A)
		if maxOff < tol {
			// Converged: extract eigenvalues and eigenvectors
			vals := make([]float64, n)
			for i := 0; i < n; i++ {
				vals[i] = A[i][i]
			}
			eve := make([][]float64, n)
			for j := 0; j < n; j++ {
				eve[j] = make([]float64, n)
				for i := 0; i < n; i++ {
					eve[j][i] = V[i][j]
				}
			}

			return vals, eve, nil
		}
		// Compute rotation
		c, s := computeJacobiCoefficients(A, p, q)
		applyJacobi(A, V, p, q, c, s)
	}

	return nil, nil, ErrEigenFailed
}
