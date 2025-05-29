package core_test

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/katalvlaran/lvlath/core"
)

// MethodsSuite exercises all Graph method behaviors: vertex/edge management,
// constraints enforcement, queries, neighbors, cloning, loops, and multi-edges.
type MethodsSuite struct {
	suite.Suite
	g *core.Graph
}

// SetupTest initializes a default Graph before each test.
func (s *MethodsSuite) SetupTest() {
	// Default configuration: undirected, unweighted, no loops, no multi-edges
	s.g = core.NewGraph()
}

// TestAddRemoveVertex covers AddVertex, HasVertex, and RemoveVertex behavior.
func (s *MethodsSuite) TestAddRemoveVertex() {
	req := require.New(s.T())

	// Empty ID should be rejected
	err := s.g.AddVertex("")
	req.ErrorIs(err, core.ErrEmptyVertexID, "AddVertex(\"\") returns ErrEmptyVertexID")

	// Valid add
	req.NoError(s.g.AddVertex("A"), "AddVertex(\"A\") should succeed")
	req.True(s.g.HasVertex("A"), "HasVertex(\"A\") should report true after add")

	// Duplicate is no-op (count remains unchanged)
	count := len(s.g.Vertices())
	req.NoError(s.g.AddVertex("A"), "Adding existing vertex is no-op and should not error")
	req.Equal(count, len(s.g.Vertices()), "Vertex count remains the same after duplicate AddVertex")

	// Remove empty ID and nonexistent ID
	err = s.g.RemoveVertex("")
	req.ErrorIs(err, core.ErrEmptyVertexID, "RemoveVertex(\"\") returns ErrEmptyVertexID")
	err = s.g.RemoveVertex("X")
	req.ErrorIs(err, core.ErrVertexNotFound, "RemoveVertex(\"X\") returns ErrVertexNotFound")

	// Remove existing
	req.NoError(s.g.RemoveVertex("A"), "RemoveVertex(\"A\") should succeed")
	req.False(s.g.HasVertex("A"), "HasVertex(\"A\") should be false after removal")
}

// TestAddEdgeConstraints covers weight, loop, and multi-edge constraints enforcement.
func (s *MethodsSuite) TestAddEdgeConstraints() {
	req := require.New(s.T())

	// Unweighted graph rejects non-zero weight
	_, err := s.g.AddEdge("A", "B", 5)
	req.ErrorIs(err, core.ErrBadWeight, "AddEdge on unweighted graph with weight!=0 returns ErrBadWeight")

	// Enable weights and add weighted edge
	s.g = core.NewGraph(core.WithWeighted())
	_, err = s.g.AddEdge("A", "B", 7)
	req.NoError(err, "AddEdge on weighted graph accepts non-zero weight")
	req.True(s.g.HasEdge("A", "B"), "HasEdge(\"A\",\"B\") should be true after adding weighted edge")

	// Default disallows self-loop
	_, err = s.g.AddEdge("X", "X", 0)
	req.ErrorIs(err, core.ErrLoopNotAllowed, "AddEdge self-loop on graph without loops returns ErrLoopNotAllowed")

	// Enable loops and successfully add self-loop
	s.g = core.NewGraph(core.WithLoops())
	loopID, err := s.g.AddEdge("X", "X", 0)
	req.NoError(err, "AddEdge self-loop on loop-enabled graph should succeed")
	req.NotEmpty(loopID, "Self-loop edge ID should be non-empty and auto-generated")
	req.True(s.g.HasEdge("X", "X"), "HasEdge(\"X\",\"X\") should be true after adding loop")

	// Multi-edge disallowed by default: first edge OK, second same endpoints errors
	s.g = core.NewGraph()
	_, err = s.g.AddEdge("A", "B", 0)
	req.NoError(err, "First AddEdge on default graph succeeds")
	_, err = s.g.AddEdge("A", "B", 0)
	req.ErrorIs(err, core.ErrMultiEdgeNotAllowed, "Second parallel AddEdge on default graph errors ErrMultiEdgeNotAllowed")

	// Enable multi-edges and add parallel edges
	s.g = core.NewGraph(core.WithMultiEdges(), core.WithWeighted(), core.WithLoops())
	e1, err := s.g.AddEdge("A", "B", 1)
	req.NoError(err)
	e2, err := s.g.AddEdge("A", "B", 2)
	req.NoError(err)
	req.NotEqual(e1, e2, "Parallel edges produce distinct IDs when multi-edges enabled")
}

// TestRemoveEdge verifies RemoveEdge cleans both global map and adjacency.
func (s *MethodsSuite) TestRemoveEdge() {
	req := require.New(s.T())

	// Setup graph with two edges
	s.g = core.NewGraph(core.WithWeighted())
	e1, _ := s.g.AddEdge("A", "B", 1)
	_, _ = s.g.AddEdge("B", "C", 2)

	// Removing nonexistent edge returns ErrEdgeNotFound
	err := s.g.RemoveEdge("nope")
	req.ErrorIs(err, core.ErrEdgeNotFound, "RemoveEdge unknown ID returns ErrEdgeNotFound")

	// Remove first edge and verify adjacency
	req.NoError(s.g.RemoveEdge(e1), "RemoveEdge existing ID should succeed")
	req.False(s.g.HasEdge("A", "B"), "HasEdge(\"A\",\"B\") should be false after removal")
	req.False(s.g.HasEdge("B", "A"), "No mirror in undirected default graph")
	req.True(s.g.HasEdge("B", "C"), "Other edges remain unaffected")
}

// TestQueries covers HasEdge, Neighbors, Vertices, and Edges ordering and content.
func (s *MethodsSuite) TestQueries() {
	req := require.New(s.T())

	// Use weighted, loop-enabled graph for queries
	s.g = core.NewGraph(core.WithWeighted(), core.WithLoops())

	// Add a single undirected edge V1–V2
	req.NoError(s.g.AddVertex("V1"))
	_, err := s.g.AddEdge("V1", "V2", 0)
	req.NoError(err, "AddEdge(\"V1\",\"V2\") should succeed")
	// Mirror adjacency without second insertion
	req.True(s.g.HasEdge("V2", "V1"), "Undirected edge should mirror across endpoints")

	// Add a self-loop on V1
	_, err = s.g.AddEdge("V1", "V1", 1)
	req.NoError(err, "AddEdge self-loop should succeed")
	req.True(s.g.HasEdge("V1", "V1"), "HasEdge(\"V1\",\"V1\") should be true for loop")

	// HasEdge checks
	req.True(s.g.HasEdge("V1", "V2"))
	req.True(s.g.HasEdge("V1", "V1"))

	// Neighbors returns both edges sorted by ID
	nbs, err := s.g.Neighbors("V1")
	req.NoError(err)
	ids := make([]string, len(nbs))
	for i, e := range nbs {
		ids[i] = e.ID
	}
	sorted := append([]string(nil), ids...)
	sort.Strings(sorted)
	req.Equal(sorted, ids, "Neighbors should be sorted by edge ID")
	req.Len(ids, 2, "Neighbors for V1 include the undirected edge and the loop only once each")

	// Vertices returns sorted list
	vs := s.g.Vertices()
	sortedV := append([]string(nil), vs...)
	sort.Strings(sortedV)
	req.Equal(sortedV, vs, "Vertices() returns sorted vertex IDs")

	// Edges returns two edges: one undirected and one loop
	ees := s.g.Edges()
	req.Len(ees, 2, "Edges() should return two edges: V1–V2 and V1–V1 loop")
}

// TestCloneEmptyAndCloneMethods ensures CloneEmpty copies vertices only and Clone deep-copies edges.
func (s *MethodsSuite) TestCloneEmptyAndCloneMethods() {
	req := require.New(s.T())

	// Setup with multi-edges, weights, loops
	s.g = core.NewGraph(core.WithWeighted(), core.WithMultiEdges(), core.WithLoops())
	_, _ = s.g.AddEdge("A", "B", 1)
	_, _ = s.g.AddEdge("A", "B", 2)

	// CloneEmpty: no edges, vertices preserved
	ce := s.g.CloneEmpty()
	req.ElementsMatch(s.g.Vertices(), ce.Vertices(), "CloneEmpty preserves vertices")
	req.Empty(ce.Edges(), "CloneEmpty has no edges")

	// Clone: full deep copy
	c := s.g.Clone()
	req.ElementsMatch(s.g.Vertices(), c.Vertices(), "Clone copies vertices")
	req.Len(c.Edges(), len(s.g.Edges()), "Clone copies all edges")

	// Independence: mutating original does not affect clone
	eOrig := s.g.Edges()[0]
	eClone := c.Edges()[0]
	eOrig.Weight = 99
	req.NotEqual(eOrig.Weight, eClone.Weight, "Clone edge weights are independent")
}

// TestLoopsAndDirection tests self-loop behavior in undirected vs directed graphs.
func (s *MethodsSuite) TestLoopsAndDirection() {
	req := require.New(s.T())

	// Undirected loop-enabled graph
	s.g = core.NewGraph(core.WithLoops())
	eid, err := s.g.AddEdge("X", "X", 0)
	req.NoError(err)
	nbs, _ := s.g.Neighbors("X")
	req.Len(nbs, 1, "Undirected self-loop appears exactly once in Neighbors")

	ees := s.g.Edges()
	// Only one entry per Edge.ID even though undirected mirror skips loops
	req.Len(ees, 1, "Edges() yields single entry for self-loop in undirected graph")
	req.Equal(eid, ees[0].ID, "Loop edge ID matches returned ID")

	// Directed graph with loops
	s.g = core.NewGraph(core.WithLoops(), core.WithDirected(true))
	eid2, err := s.g.AddEdge("Y", "Y", 0)
	req.NoError(err)
	nbs2, _ := s.g.Neighbors("Y")
	req.Len(nbs2, 1, "Directed self-loop appears once in Neighbors")
	req.True(nbs2[0].Directed, "Self-loop in directed graph must be marked Directed")
	req.Equal(eid2, nbs2[0].ID, "Returned ID matches neighbor ID")
}

// TestMultiEdges verifies parallel edges behave correctly when enabled.
func (s *MethodsSuite) TestMultiEdges() {
	req := require.New(s.T())

	// Enable multi-edges and weights
	s.g = core.NewGraph(core.WithMultiEdges(), core.WithWeighted())
	e1, err := s.g.AddEdge("A", "B", 1)
	req.NoError(err)
	e2, err := s.g.AddEdge("A", "B", 2)
	req.NoError(err)
	req.NotEqual(e1, e2, "Parallel edges produce distinct IDs")

	// Check weights on each ID
	ees := s.g.Edges()
	weights := make(map[string]int64, 2)
	for _, e := range ees {
		if e.From == "A" && e.To == "B" {
			weights[e.ID] = e.Weight
		}
	}
	req.ElementsMatch([]int64{1, 2}, []int64{weights[e1], weights[e2]}, "Weights match original values")
}

/*// TestMixedEdges checks per-edge directed override when mixed edges enabled.
func (s *MethodsSuite) TestMixedEdges() {
	req := require.New(s.T())

	// Enable mixed-edge support (Graph default undirected)
	s.g = core.NewGraph(core.WithMixedEdges())
	// Add a directed edge override
	_, err := s.g.AddEdge("A", "B", 0, core.WithEdgeDirected(true))
	req.NoError(err)
	nbsA, _ := s.g.Neighbors("A")
	req.Len(nbsA, 1, "Directed override edge appears in source neighbors")
	req.True(nbsA[0].Directed, "Edge should respect per-edge Directed flag")

	// Add an explicit undirected override
	eidU, err := s.g.AddEdge("B", "C", 0, core.WithEdgeDirected(false))
	req.NoError(err)
	nbsB, _ := s.g.Neighbors("B")
	// Find the B–C edge and ensure it's undirected
	found := false
	for _, e := range nbsB {
		if e.ID == eidU {
			req.False(e.Directed, "Edge override directed=false must be undirected")
			found = true
		}
	}
	req.True(found, fmt.Sprintf("Neighbor list for B should include edge %s", eidU))
}*/

// Entry point for the test suite
func TestMethodsSuite(t *testing.T) {
	suite.Run(t, new(MethodsSuite))
}
