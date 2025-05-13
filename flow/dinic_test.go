package flow_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/flow"
)

// DinicSuite tests the Dinic implementation.
type DinicSuite struct {
	suite.Suite
}

func (s *DinicSuite) SetupTest() {
	// nothing for now
}

// TestSingleEdge: simple A→B with cap=7
func (s *DinicSuite) TestSingleEdge() {
	g := core.NewGraph(true, true)
	g.AddEdge("A", "B", 7)

	mf, res, err := flow.Dinic(g, "A", "B", nil)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 7.0, mf)
	// After flow, forward capacity is zero
	require.False(s.T(), res.HasEdge("A", "B"))
	// Reverse carries the flow
	require.True(s.T(), res.HasEdge("B", "A"))
}

// TestMultiPath: two disjoint paths sum up
func (s *DinicSuite) TestMultiPath() {
	g := core.NewGraph(true, true)
	g.AddEdge("A", "B", 5)
	g.AddEdge("A", "C", 4)
	g.AddEdge("C", "B", 3)

	mf, _, err := flow.Dinic(g, "A", "B", &flow.FlowOptions{Verbose: true})
	require.NoError(s.T(), err)
	// best flows: A→B=5, A→C→B=3 => total 8
	require.Equal(s.T(), 8.0, mf)
}

// TestNegativeCapacity yields EdgeError
func (s *DinicSuite) TestNegativeCapacity() {
	g := core.NewGraph(true, true)
	g.AddEdge("X", "Y", -2)

	_, _, err := flow.Dinic(g, "X", "Y", nil)
	var ee flow.EdgeError
	require.Error(s.T(), err)
	require.True(s.T(), errors.As(err, &ee))
	require.Equal(s.T(), "X", ee.From)
	require.Equal(s.T(), "Y", ee.To)
}

// TestSourceSinkNotFound covers missing endpoints
func (s *DinicSuite) TestSourceSinkNotFound() {
	g := core.NewGraph(true, true)
	g.AddVertex(&core.Vertex{ID: "A"})

	_, _, err1 := flow.Dinic(g, "X", "A", nil)
	require.True(s.T(), errors.Is(err1, flow.ErrSourceNotFound))

	_, _, err2 := flow.Dinic(g, "A", "Z", nil)
	require.True(s.T(), errors.Is(err2, flow.ErrSinkNotFound))
}

func TestDinicSuite(t *testing.T) {
	suite.Run(t, new(DinicSuite))
}
