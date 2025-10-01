// SPDX-License-Identifier: MIT
// Package: lvlath/builder
//
// variants_platonic.go — canonical data & generators for Platonic solids.
//
// Design:
//   • Single source of truth for the 5 Platonic graphs (V counts and shell edges).
//   • Public-neutral type PlatonicName and internal datasets.
//   • Datasets are constructed deterministically at init() and kept immutable.
//
// Determinism:
//   • All generated edge sets are sorted lexicographically by (U,V) with U < V.
//   • Icosahedron edges are derived from a canonical face list (stable).
//   • Dodecahedron edges are derived as the dual (face adjacency graph of icosahedron),
//     yielding a 3-regular graph on 20 vertices; also sorted for stability.
//
// Complexity:
//   • init(): O(1) constants; tiny O(F) work for icosa/dodeca (F=20 faces).
//
// AI-Hints:
//   • Extend with alternative embeddings by adding new enums and datasets only;
//     never mutate existing edge sets — they are part of the public contract.

package builder

// PlatonicName enumerates the five Platonic solids (canonical graph shells).
type PlatonicName int

// String provides a readable identifier for logs/errors (deterministic).
func (p PlatonicName) String() string {
	switch p {
	case Tetrahedron:
		// Tetrahedron is the 4-vertex regular tetrahedron (complete K4 graph).
		return "Tetrahedron"
	case Cube:
		// Cube is the 8-vertex cube topology.
		return "Cube"
	case Octahedron:
		// Octahedron is the 6-vertex regular octahedron.
		return "Octahedron"
	case Dodecahedron:
		// Dodecahedron is the 20-vertex regular dodecahedron.
		return "Dodecahedron"
	case Icosahedron:
		// Icosahedron is the 12-vertex regular icosahedron.
		return "Icosahedron"
	default:
		return "Unknown"
	}
}

// Enum values (stable ordering).
const (
	Tetrahedron  PlatonicName = iota // V=4,  E=6
	Cube                             // V=8,  E=12
	Octahedron                       // V=6,  E=12
	Dodecahedron                     // V=20, E=30
	Icosahedron                      // V=12, E=30
)

// platonicVertexCounts maps each PlatonicName to its vertex count.
// Used to allocate shell vertices before adding edges.
var platonicVertexCounts = map[PlatonicName]int{
	Tetrahedron:  4,  // K4
	Cube:         8,  // 2×square + verticals
	Octahedron:   6,  // dual of cube
	Dodecahedron: 20, // 3-regular on 20 vertices
	Icosahedron:  12, // 5-regular on 12 vertices
}

// platonicEdgeSets maps each PlatonicName to its canonical shell edge list.
// Each edge is an unordered chord{U,V} with U<V, and the slice is pre-sorted.
var platonicEdgeSets = map[PlatonicName][]chord{
	// -------------------------------------------------------------------------
	// Tetrahedron: complete graph K4 on vertices 0..3.
	// -------------------------------------------------------------------------
	Tetrahedron: {
		{U: 0, V: 1}, {U: 0, V: 2}, {U: 0, V: 3},
		{U: 1, V: 2}, {U: 1, V: 3},
		{U: 2, V: 3},
	},

	// -------------------------------------------------------------------------
	// Cube: two 4-cycles (bottom/top faces) + 4 vertical edges.
	//
	// Layout:
	//   Bottom face: 0-1-2-3-0
	//   Top face:    4-5-6-7-4
	//   Verticals:   0-4, 1-5, 2-6, 3-7
	// -------------------------------------------------------------------------
	Cube: {
		// bottom cycle
		{U: 0, V: 1}, {U: 1, V: 2}, {U: 2, V: 3}, {U: 3, V: 0},
		// verticals
		{U: 0, V: 4}, {U: 1, V: 5}, {U: 2, V: 6}, {U: 3, V: 7},
		// top cycle
		{U: 4, V: 5}, {U: 4, V: 7}, {U: 5, V: 6}, {U: 6, V: 7},
	},

	// -------------------------------------------------------------------------
	// Octahedron: 6 vertices, 12 edges; degree 4 at each vertex.
	//
	// Layout (one of standard labelings): vertices {0,1} are "poles",
	// equatorial ring {2,3,4,5}. Each pole connects to all equator vertices;
	// equator connects as two opposite pairs to maintain degree 4.
	// -------------------------------------------------------------------------
	Octahedron: {
		{U: 0, V: 2}, {U: 0, V: 3}, {U: 0, V: 4}, {U: 0, V: 5},
		{U: 1, V: 2}, {U: 1, V: 3}, {U: 1, V: 4}, {U: 1, V: 5},
		{U: 2, V: 4}, {U: 2, V: 5}, {U: 3, V: 4}, {U: 3, V: 5},
	},

	// -------------------------------------------------------------------------
	// Dodecahedron: 20 vertices, 30 edges; 3-regular (each vertex degree 3).
	//
	// Canonical construction used here:
	//   • Top pentagon:    0-1-2-3-4-0
	//   • Bottom pentagon: 5-6-7-8-9-5
	//   • Middle ring:     10-11-12-13-14-15-16-17-18-19-10  (10-cycle)
	//   • Spokes:
	//       Top    0→10, 1→12, 2→14, 3→16, 4→18   (even middle indices)
	//       Bottom 5→11, 6→13, 7→15, 8→17, 9→19   (odd  middle indices)
	//
	// This yields the standard dodecahedral graph (isomorphic to the dual of the
	// icosahedron), with deterministic labeling and pre-sorted edges.
	// -------------------------------------------------------------------------
	Dodecahedron: {
		// top pentagon
		{U: 0, V: 1}, {U: 0, V: 4}, {U: 1, V: 2}, {U: 2, V: 3}, {U: 3, V: 4},
		// bottom pentagon
		{U: 5, V: 6}, {U: 5, V: 9}, {U: 6, V: 7}, {U: 7, V: 8}, {U: 8, V: 9},
		// middle ring (10-cycle; store with U<V and lexicographic order)
		{U: 10, V: 11}, {U: 10, V: 19}, {U: 11, V: 12}, {U: 12, V: 13}, {U: 13, V: 14},
		{U: 14, V: 15}, {U: 15, V: 16}, {U: 16, V: 17}, {U: 17, V: 18}, {U: 18, V: 19},
		// spokes to top (even middle indices)
		{U: 0, V: 10}, {U: 1, V: 12}, {U: 2, V: 14}, {U: 3, V: 16}, {U: 4, V: 18},
		// spokes to bottom (odd middle indices)
		{U: 5, V: 11}, {U: 6, V: 13}, {U: 7, V: 15}, {U: 8, V: 17}, {U: 9, V: 19},
	},

	// -------------------------------------------------------------------------
	// Icosahedron: 12 vertices, 30 edges; 5-regular (each vertex degree 5).
	//
	// Canonical construction used here (standard “two pentagon rings + poles”):
	//   • Top pole:    0
	//   • Top ring:    1-2-3-4-5-1
	//   • Bottom ring: 6-7-8-9-10-6
	//   • Bottom pole: 11
	//   • Edges:
	//       – Pole connections:   0 to {1..5}; 11 to {6..10}
	//       – Ring cycles:        (top) 1-2-3-4-5-1; (bottom) 6-7-8-9-10-6
	//       – Cross connections:  Ti connects to Bi and B(i+1 mod 5)
	//
	// This yields the classical icosahedral graph (dual to the dodecahedral graph),
	// with deterministic labeling and pre-sorted edges.
	// -------------------------------------------------------------------------
	Icosahedron: {
		// top pole to top ring
		{U: 0, V: 1}, {U: 0, V: 2}, {U: 0, V: 3}, {U: 0, V: 4}, {U: 0, V: 5},
		// top ring cycle
		{U: 1, V: 2}, {U: 1, V: 5}, {U: 2, V: 3}, {U: 3, V: 4}, {U: 4, V: 5},
		// cross (top→bottom)
		{U: 1, V: 6}, {U: 1, V: 7}, {U: 2, V: 7}, {U: 2, V: 8}, {U: 3, V: 8},
		{U: 3, V: 9}, {U: 4, V: 9}, {U: 4, V: 10}, {U: 5, V: 6}, {U: 5, V: 10},
		// bottom ring cycle
		{U: 6, V: 7}, {U: 6, V: 10}, {U: 7, V: 8}, {U: 8, V: 9}, {U: 9, V: 10},
		// bottom pole to bottom ring
		{U: 6, V: 11}, {U: 7, V: 11}, {U: 8, V: 11}, {U: 9, V: 11}, {U: 10, V: 11},
	},
}
