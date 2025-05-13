// Package gridgraph treats a 2D grid of cells as a graph, enabling
// component analysis and minimal‐cost “island” expansions.
//
// A GridGraph wraps a rectangular [][]int grid where each cell has a
// discrete value (e.g. 0 = water, ≥1 = land or resource ID).  You can:
//
//   - Choose connectivity: 4‐direction (N/E/S/W) or 8‐direction (incl. diagonals)
//   - Identify connected components (“islands”) of cells sharing the same non-zero value
//   - Convert to a *core.Graph for arbitrary graph algorithms (e.g. BFS, flow)
//   - Compute the minimal‐cell conversions to connect two islands (ExpandIsland)
//
// Typical use cases:
//
//	– Game maps: find contiguous land areas, build bridges at minimal cost
//	– Resource planning: connect facilities across terrain with least upgrades
//	– Topology analysis: count lakes, islands, or heterogeneous regions
//
// GridGraph is immutable once built.  All operations run in O(WH) time
// (W = width, H = height) and O(WH) memory.  Conversion to *core.Graph
// is O(WH + E) where E ≤ WH · neighbors.
//
//	go get github.com/katalvlaran/lvlath/gridgraph
package gridgraph
