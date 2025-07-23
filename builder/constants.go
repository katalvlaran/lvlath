// Package builder defines shared constants used by graph builders, ensuring
// consistent defaults and validation across all topology constructors.
package builder

//-----------------------------------------------------------------------------
// Builder Method Name Constants
//   used to prefix errors with the constructor name for context.
//-----------------------------------------------------------------------------

const (
	// MethodCycle is the canonical name for the Cycle constructor.
	MethodCycle = "Cycle"
	// MethodPath is the canonical name for the Path constructor.
	MethodPath = "Path"
	// MethodStar is the canonical name for the Star constructor.
	MethodStar = "Star"
	// MethodWheel is the canonical name for the Wheel constructor.
	MethodWheel = "Wheel"
	// MethodComplete is the canonical name for the Complete constructor.
	MethodComplete = "Complete"
	// MethodCompleteBipartite is the canonical name for the CompleteBipartite constructor.
	MethodCompleteBipartite = "CompleteBipartite"
	// MethodRandomSparse is the canonical name for the RandomSparse constructor.
	MethodRandomSparse = "RandomSparse"
	// MethodRandomRegular is the canonical name for the RandomRegular constructor.
	MethodRandomRegular = "RandomRegular"
	// MethodGrid is the canonical name for the Grid constructor.
	MethodGrid = "Grid"
	// MethodHexagram is the canonical name for the Hexagram constructor.
	MethodHexagram = "Hexagram"
	// MethodPlatonicSolid is the canonical name for the PlatonicSolid constructor.
	MethodPlatonicSolid = "PlatonicSolid"
)

//-----------------------------------------------------------------------------
// Vertex ID Defaults
//-----------------------------------------------------------------------------

// FirstVertexID is the identifier for the first vertex in sequential topologies
// (e.g., Path, Cycle) to avoid sprinkling literal "0" throughout the code.
const FirstVertexID = "0"

// CenterVertexID is the identifier for a central hub vertex in Star, Wheel,
// and stellated Platonic solids, ensuring tests and debugging remain consistent.
const CenterVertexID = "Center"

//-----------------------------------------------------------------------------
// Minimum Node Counts
//-----------------------------------------------------------------------------

// MinCycleNodes is the smallest meaningful size for a cycle (ring) topology.
// A cycle with fewer than 3 nodes cannot form a valid ring without loops or multi-edges.
// Complexity impact: Cycle builds O(n) edges; n >= MinCycleNodes.
const MinCycleNodes = 3

// MinPathNodes is the smallest meaningful size for a simple path.
// A path of fewer than 2 nodes has no edges.
// Complexity impact: Path adds n–1 edges; n >= MinPathNodes.
const MinPathNodes = 2

// MinStarNodes is the smallest meaningful size for a star topology.
// A star requires one center plus at least one leaf (2 nodes total).
// Complexity impact: Star adds n–1 edges; n >= MinStarNodes.
const MinStarNodes = 2

// MinWheelNodes is the smallest meaningful size for a wheel topology.
// A wheel is a cycle of at least 3 nodes plus one hub (4 nodes total).
// Complexity impact: Wheel builds O(n) cycle edges + O(n) hub edges; n >= MinWheelNodes.
const MinWheelNodes = 4

// MinGridDim is the smallest allowed dimension (rows or cols) for a 2D Grid.
// A grid of size 1×1 has no edges, but is considered valid.
const MinGridDim = 1

//-----------------------------------------------------------------------------
// Default Weights and Probability Bounds
//-----------------------------------------------------------------------------

// DefaultEdgeWeight is the default weight assigned to each edge when no
// custom WeightFn is provided.
const DefaultEdgeWeight int64 = 1

// MinProbability is the lower bound for the probability parameter p in
// RandomSparse (Erdős–Rényi) graph construction, inclusive.
const MinProbability = 0.0

// MaxProbability is the upper bound for the probability parameter p in
// RandomSparse construction, inclusive.
const MaxProbability = 1.0

// MaxPartition .
const MaxPartition = 1
