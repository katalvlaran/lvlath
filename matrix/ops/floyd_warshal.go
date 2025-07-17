// Package ops provides advanced matrix operations for the lvlath/matrix package.
// floyd_warshall.go implements the Floyd–Warshall all-pairs shortest paths algorithm.
package ops

import (
	"fmt"

	"github.com/katalvlaran/lvlath/matrix"
)

// FloydWarshall computes the shortest path distances between all pairs of vertices
// in-place on the provided matrix m.  m must be square, with +Inf representing
// absent edges.  Returns ErrMatrixDimensionMismatch if m is not square.
// Complexity: O(n³) time, O(1) extra memory.
func FloydWarshall(m matrix.Matrix) error {
	// Stage 1: Validate input shape
	if m.Rows() != m.Cols() {
		return fmt.Errorf(
			"FloydWarshall: non-square matrix %dx%d: %w",
			m.Rows(), m.Cols(), matrix.ErrMatrixDimensionMismatch,
		)
	}
	n := m.Rows() // dimension of the matrix

	// Stage 2: Declare loop indices and temporaries
	var (
		i, j, k       int     // loop counters
		dik, dkj, dij float64 // path candidate values
		err           error   // error placeholder
	)

	// Stage 3: Core triple‐nested loop
	for k = 0; k < n; k++ { // for each intermediate vertex
		for i = 0; i < n; i++ { // for each source vertex
			for j = 0; j < n; j++ { // for each destination vertex
				// load current distance i→k
				dik, err = m.At(i, k)
				if err != nil {
					return fmt.Errorf("FloydWarshall: At(%d,%d): %w", i, k, err)
				}
				// load current distance k→j
				dkj, err = m.At(k, j)
				if err != nil {
					return fmt.Errorf("FloydWarshall: At(%d,%d): %w", k, j, err)
				}
				// load current distance i→j
				dij, err = m.At(i, j)
				if err != nil {
					return fmt.Errorf("FloydWarshall: At(%d,%d): %w", i, j, err)
				}
				// relax if path via k is shorter
				if dik+dkj < dij {
					if err = m.Set(i, j, dik+dkj); err != nil {
						return fmt.Errorf("FloydWarshall: Set(%d,%d): %w", i, j, err)
					}
				}
			}
		}
	}

	return nil
}
