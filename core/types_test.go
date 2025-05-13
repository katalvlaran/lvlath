package core_test

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/katalvlaran/lvlath/core"
)

type TypesSuite struct {
	suite.Suite
	g *core.Graph
}

func (s *TypesSuite) SetupTest() {
	// For tests that need a fresh weighted, undirected graph
	s.g = core.NewGraph(false, true)
}

func (s *TypesSuite) TestNewGraphFlags() {
	cases := []struct {
		directed, weighted bool
	}{
		{false, false},
		{false, true},
		{true, false},
		{true, true},
	}

	for _, tc := range cases {
		g := core.NewGraph(tc.directed, tc.weighted)
		require.Equal(s.T(), tc.directed, g.Directed(), "Directed() flag")
		require.Equal(s.T(), tc.weighted, g.Weighted(), "Weighted() flag")
	}
}

func (s *TypesSuite) TestCloneEmpty() {
	// Add vertices and edges to the original graph
	s.g.AddVertex(&core.Vertex{ID: "A"})
	s.g.AddEdge("A", "B", 2)
	s.g.AddEdge("B", "C", 3)

	clone := s.g.CloneEmpty()

	// Vertices should match
	origIDs := sortedIDs(s.g.Vertices())
	clonedIDs := sortedIDs(clone.Vertices())
	require.Equal(s.T(), origIDs, clonedIDs, "CloneEmpty should preserve vertices")

	// No edges in the clone
	require.Empty(s.T(), clone.Edges(), "CloneEmpty should not copy edges")
}

func (s *TypesSuite) TestCloneIndependence() {
	// Build a graph with one edge
	s.g.AddEdge("A", "B", 5)

	// Clone before mutation
	clone := s.g.Clone()

	// Mutate original
	origEdges := s.g.Edges()
	require.NotEmpty(s.T(), origEdges)
	origEdges[0].Weight = 42

	// Clone remains unaffected
	cloneEdges := clone.Edges()
	require.NotEmpty(s.T(), cloneEdges)
	require.Equal(s.T(), int64(5), cloneEdges[0].Weight, "Clone should remain independent of original")
}

func (s *TypesSuite) TestVerticesMapReadOnly() {
	// Setup a graph with one vertex
	g := core.NewGraph(false, false)
	g.AddVertex(&core.Vertex{ID: "X"})

	// Attempt to mutate via VerticesMap
	vm := g.VerticesMap()
	vm["Y"] = &core.Vertex{ID: "Y"}

	// The original graph should not include "Y"
	require.False(s.T(), g.HasVertex("Y"), "VerticesMap should be a safe, read-only copy")
}

func TestTypesSuite(t *testing.T) {
	suite.Run(t, new(TypesSuite))
}

// sortedIDs returns sorted IDs of the given vertices.
func sortedIDs(vs []*core.Vertex) []string {
	ids := make([]string, len(vs))
	for i, v := range vs {
		ids[i] = v.ID
	}
	sort.Strings(ids)
	return ids
}
