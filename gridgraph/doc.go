// Package gridgraph treats a 2D grid of cells as a graph, enabling
// component analysis and minimal-cost “island” expansions.
//
// What:
//
//   - GridGraph wraps a rectangular [][]int grid with tunable LandThreshold.
//   - Identifies connected components (“islands”) of cells with value ≥ LandThreshold.
//   - Computes minimal conversions (0-1 BFS) to connect two island sets.
//   - Converts to a *core.Graph for arbitrary graph algorithms.
//
// Why:
//
//   - Game maps: contiguous land detection, optimal bridging.
//   - Resource planning: connect facilities with minimal upgrades.
//   - Topology analysis: count lakes, islands, and heterogeneous regions.
//
// Complexity:
//
//   - ConnectedComponents: O(W×H×d), Memory: O(W×H)    (d = number of neighbors, 4 or 8).
//   - ExpandIsland:          O(W×H×d), Memory: O(W×H).
//   - ToCoreGraph:           O(W×H×d + E), Memory: O(W×H + E).
//
// Options:
//
//   - GridOptions.LandThreshold: minimum value considered "land".
//   - GridOptions.Conn: Conn4 (4-neighbors) or Conn8 (8-neighbors).
//
// Errors:
//
//   - ErrEmptyGrid: input grid has no rows or no columns.
//   - ErrNonRectangular: rows have differing lengths.
//   - ErrComponentIndex: requested component index out of range.
//   - ErrNoPath: no conversion path exists between specified components.
//
// See: docs/GRID_GRAPH.md for full tutorial.
package gridgraph
