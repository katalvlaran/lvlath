// Package builder provides internal helper functions and constants
// used by GraphConstructor implementations to build common topologies.
//
// Design principles:
//   - Single Responsibility: each helper does one well-defined job.
//   - Error Context: wrap errors with builderErrorf for uniform reporting.
//   - Performance: avoid unnecessary allocations; reuse loop variables.
//   - Readability: explicit naming, minimal nesting, consistent style.
package builder

import (
	"fmt"
	"strconv"

	"github.com/katalvlaran/lvlath/core"
)

// builderErrorf wraps an inner error message with the given method context.
// It returns an error of the form "<Method>: <formatted message>".
//
// Parameters:
//   - method: canonical constructor name, e.g. MethodCycle.
//   - format: format string for the inner message.
//   - args:   values for the format placeholders.
//
// Complexity: O(len(format) + Σlen(args)), negligible for our use.
func builderErrorf(method, format string, args ...interface{}) error {
	// Build the inner message using fmt.Sprintf
	inner := fmt.Sprintf(format, args...)
	// Prefix with the method name and return a new error
	return fmt.Errorf("%s: %s", method, inner)
}

// addSequentialVertices inserts vertices with IDs "0".."n-1" into g.
// It is idempotent: re-adding existing vertices is a no-op in core.Graph.
//
// Parameters:
//   - g: target graph.
//   - n: number of vertices to add.
//
// Returns the first error encountered, wrapped with context.
//
// Complexity: O(n) time, O(1) extra space.
func addSequentialVertices(g *core.Graph, n int) error {
	var (
		i   int
		id  string
		err error
	)
	for i = 0; i < n; i++ {
		// convert index to string ID
		id = strconv.Itoa(i)
		// attempt to add the vertex to the graph
		if err = g.AddVertex(id); err != nil {
			// wrap and return error with context if AddVertex fails
			return fmt.Errorf("addSequentialVertices: AddVertex(%s): %w", id, err)
		}
	}

	// all vertices added successfully (or already existed)
	return nil
}

// addVerticesWithIDFn adds vertices idFn(0..n-1).
func addVerticesWithIDFn(g *core.Graph, n int, idFn IDFn) error {
	var (
		i   int
		vid string
		err error
	)
	for i = 0; i < n; i++ {
		vid = idFn(i)
		if err = g.AddVertex(vid); err != nil {
			return err
		}
	}
	return nil
}

// addCompleteEdges connects every unordered pair in ids with edges of weight w.
// For directed graphs, mirrors each edge in the opposite direction.
//
// Parameters:
//   - g:   target graph.
//   - ids: slice of vertex IDs.
//   - w:   weight to assign to every edge.
//
// Returns the first error encountered, wrapped with context.
//
// Complexity: O(m²) time where m = len(ids), O(1) extra space.
func addCompleteEdges(g *core.Graph, ids []string, w int64) error {
	var (
		i, j int
		u, v string
		err  error
	)
	// outer loop over vertex IDs
	for i = 0; i < len(ids); i++ {
		u = ids[i] // source vertex ID
		// inner loop over subsequent IDs to avoid duplicates
		for j = i + 1; j < len(ids); j++ {
			v = ids[j] // target vertex ID
			// add edge u -> v
			if _, err = g.AddEdge(u, v, w); err != nil {
				return fmt.Errorf("addCompleteEdges: AddEdge(%s->%s,w=%d): %w", u, v, w, err)
			}
			// if the graph is directed, also add edge v -> u
			if g.Directed() {
				if _, err = g.AddEdge(v, u, w); err != nil {
					return fmt.Errorf("addCompleteEdges: AddEdge(%s->%s,w=%d): %w", v, u, w, err)
				}
			}
		}
	}

	// all pairs connected successfully
	return nil
}

// makeIDs generates n vertex IDs by concatenating prefix and index.
// Example: makeIDs("L",3) → {"L0","L1","L2"}.
//
// Parameters:
//   - prefix: string prefix for each ID.
//   - n:      number of IDs to generate.
//
// Returns a slice of length n.
//
// Complexity: O(n) time and space.
func makeIDs(prefix string, n int) []string {
	ids := make([]string, n) // allocate slice once
	var i int
	for i = 0; i < n; i++ { // fill each element
		ids[i] = vertexID(prefix, i)
	}

	return ids
}

// vertexID returns a vertex identifier by concatenating prefix and index.
// Example: vertexID("R",2) → "R2".
//
// Parameters:
//   - prefix: string to prepend.
//   - i:      integer index.
//
// Complexity: O(len(prefix) + digits(i)), negligible.
func vertexID(prefix string, i int) string {
	// strconv.Itoa is preferred for simple integer-to-string conversion
	return prefix + strconv.Itoa(i)
}

// gridVertexID formats a 2D grid coordinate as "r,c".
// Example: gridVertexID(0,1) → "0,1".
//
// Parameters:
//   - r: row index.
//   - c: column index.
//
// Complexity: O(digits(r)+digits(c)), negligible.
func gridVertexID(r, c int) string {
	// strconv.Itoa is more efficient than fmt.Sprintf for simple int→string
	return strconv.Itoa(r) + "," + strconv.Itoa(c)
}
