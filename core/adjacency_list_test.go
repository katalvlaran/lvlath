package core_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/katalvlaran/lvlath/core"
)

type AdjacencySuite struct {
	suite.Suite
	g *core.Graph
}

func (s *AdjacencySuite) SetupTest() {
	// Undirected, unweighted by default; individual tests may override
	s.g = core.NewGraph(false, false)
}

func (s *AdjacencySuite) TestAddVertexAndHasVertex() {
	// Initially empty
	require := require.New(s.T())
	require.False(s.g.HasVertex("A"), "empty graph should not have A")

	// Add and check
	s.g.AddVertex(&core.Vertex{ID: "A"})
	require.True(s.g.HasVertex("A"), "graph should have A after AddVertex")

	// Idempotence: adding again does not change count
	before := len(s.g.Vertices())
	s.g.AddVertex(&core.Vertex{ID: "A"})
	require.Equal(before, len(s.g.Vertices()), "adding duplicate vertex should not increase count")
}

func (s *AdjacencySuite) TestRemoveVertex() {
	require := require.New(s.T())
	// Undirected: removing also drops mirror edges
	s.g.AddEdge("A", "B", 1)
	s.g.RemoveVertex("A")
	require.False(s.g.HasVertex("A"), "A should be removed")
	require.False(s.g.HasEdge("B", "A"), "mirror edge B→A should be removed")

	// Directed: only one direction
	dg := core.NewGraph(true, false)
	dg.AddEdge("X", "Y", 1)
	dg.RemoveVertex("Y")
	require.False(dg.HasVertex("Y"), "Y should be removed in directed graph")
	require.False(dg.HasEdge("X", "Y"), "edge X→Y should be removed in directed graph")
}

func (s *AdjacencySuite) TestAddEdgeHasEdgeAndMultiedges() {
	require := require.New(s.T())
	// Switch to weighted to test weight handling
	s.g = core.NewGraph(false, true)

	// Auto-add vertices
	s.g.AddEdge("A", "B", 5)
	require.True(s.g.HasVertex("A") && s.g.HasVertex("B"), "AddEdge should auto-add vertices")
	require.True(s.g.HasEdge("A", "B"), "expected edge A→B")
	require.True(s.g.HasEdge("B", "A"), "expected mirror edge B→A in undirected graph")

	// Add a second parallel edge
	s.g.AddEdge("A", "B", 7)
	edges := s.g.Edges()
	count := 0
	for _, e := range edges {
		if e.From.ID == "A" && e.To.ID == "B" {
			count++
		}
	}
	require.Equal(2, count, "expected 2 parallel A→B edges")
}

func (s *AdjacencySuite) TestRemoveEdge() {
	require := require.New(s.T())

	// Directed removal
	dg := core.NewGraph(true, false)
	dg.AddEdge("X", "Y", 2)
	dg.RemoveEdge("X", "Y")
	require.False(dg.HasEdge("X", "Y"), "directed RemoveEdge failed")

	// Undirected removal removes both
	ug := core.NewGraph(false, false)
	ug.AddEdge("U", "V", 3)
	ug.RemoveEdge("U", "V")
	require.False(ug.HasEdge("U", "V") || ug.HasEdge("V", "U"), "undirected RemoveEdge should remove both directions")
}

func (s *AdjacencySuite) TestNeighbors() {
	require := require.New(s.T())
	s.g.AddEdge("1", "2", 0)
	s.g.AddEdge("1", "2", 0) // parallel

	nb := s.g.Neighbors("1")
	require.Len(nb, 1, "Neighbors should return unique neighbors")
	require.Equal("2", nb[0].ID, "Neighbor ID should be '2'")

	// Nonexistent vertex
	require.Nil(s.g.Neighbors("X"), "Neighbors of missing vertex should be nil")
}

func (s *AdjacencySuite) TestVerticesAndEdges() {
	require := require.New(s.T())
	s.g.AddVertex(&core.Vertex{ID: "A"})
	s.g.AddVertex(&core.Vertex{ID: "B"})
	s.g.AddEdge("A", "B", 1)

	vs := s.g.Vertices()
	require.ElementsMatch([]string{"A", "B"}, sortedIDs(vs), "Vertices should list A and B")

	es := s.g.Edges()
	// Undirected => two edges: A→B and B→A
	require.Len(es, 2, "Edges length = 2 (A→B & B→A)")
}

func (s *AdjacencySuite) TestSelfLoop() {
	require := require.New(s.T())
	// Undirected self-loop: two edges
	s.g.AddEdge("Z", "Z", 10)
	require.True(s.g.HasEdge("Z", "Z"), "self-loop Z→Z should exist")
	edges := s.g.Edges()
	loopCount := 0
	for _, e := range edges {
		if e.From.ID == "Z" && e.To.ID == "Z" {
			loopCount++
		}
	}
	require.Equal(2, loopCount, "expected 2 self-loop edges (mirror)")

	// Directed self-loop: only one
	dg := core.NewGraph(true, false)
	dg.AddEdge("W", "W", 5)
	de := dg.Edges()
	count := 0
	for _, e := range de {
		if e.From.ID == "W" && e.To.ID == "W" {
			count++
		}
	}
	require.Equal(1, count, "expected 1 self-loop edge in directed graph")
}

func TestAdjacencySuite(t *testing.T) {
	suite.Run(t, new(AdjacencySuite))
}
